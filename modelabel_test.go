package terminu

import (
	"strings"
	"testing"
	"time"
)

// TestModeLabelOpens: entering a mode its name opens from the middle on the
// top border, the junctions side by side first, then a character every 32ms
// by the clock; the mode and the frame's colour change at once; a switch
// halfway turns it where it is, and leaving plays it back to the line
// (panel-146 to 148).
func TestModeLabelOpens(t *testing.T) {
	ta, clk := logApp(t)
	top := func() string { return covCPlain(Cut(ta.Frame()[2], 0, 40)) }
	head := strings.Split(top(), "━")[0]
	ta.Update2(keyMsg("D"))
	if ta.pods.InMode() == nil || covCAt(Cut(ta.Frame()[2], 0, 40), 0).St.Fg != Yellow {
		t.Fatal("the mode and its frame's colour do not change at once")
	}
	step := func(d time.Duration, want, not string) {
		t.Helper()
		clk.t = clk.t.Add(d)
		got := top()
		if !strings.Contains(got, want) || not != "" && strings.Contains(got, not) || !strings.HasPrefix(got, head) {
			t.Errorf("after %v: %q, want %q", d, got, want)
		}
	}
	step(0, "━┫┣━", "Drag")
	if !ta.labelsMoving() {
		t.Error("the name is not counted as moving")
	}
	step(32*time.Millisecond, "━┫r┣━", "Drag")
	step(32*time.Millisecond, "━┫ra┣━", "Drag")
	ta.Update2(keyMsg("Esc")) // halfway: it turns where it is
	if ta.pods.InMode() != nil {
		t.Fatal("Esc did not leave the mode at once")
	}
	step(0, "━┫ra┣━", "")
	step(32*time.Millisecond, "━┫r┣━", "")
	ta.Update2(keyMsg("D")) // turns again, from a quarter open
	step(32*time.Millisecond, "━┫ra┣━", "")
	step(32*time.Millisecond, "━┫Dra┣━", "")
	step(32*time.Millisecond, "━┫Drag┣━", "")
	ta.Update(labelMsg{})
	if ta.labelsMoving() {
		t.Error("still moving once open")
	}
	ta.Update2(keyMsg("Esc"))
	step(128*time.Millisecond, "━━━━━━━━", "┫")
	ta.Update(labelMsg{})
	if len(ta.labels) != 0 {
		t.Error("a closed name is kept")
	}
	// In a popup the same: a viewer's selection mode.
	ta.Viewer(&Viewer{Title: "Log", Lines: []string{"one"}})
	ta.settle()
	ta.Update2(keyMsg("v"))
	if !strings.Contains(ta.screenText(), "┫┣") {
		t.Errorf("the viewer's mode name does not open from its middle:\n%s", ta.screenText())
	}
	clk.t = clk.t.Add(6 * 32 * time.Millisecond) // Visual: six characters, six frames after the first
	if !strings.Contains(ta.screenText(), "┫Visual┣") {
		t.Errorf("the viewer's mode name does not open:\n%s", ta.screenText())
	}
}

// TestModeLabelFrames: every 32ms frame shows one more character, so a
// longer name takes longer: Compare's 8 frames all differ (panel-146).
func TestModeLabelFrames(t *testing.T) {
	seen := map[string]bool{}
	for f := range 8 {
		l := &labelAnim{name: "Compare", dir: 1}
		got := StripStyles(drawLabel(l.name, 1-l.at(l.start.Add(time.Duration(f)*frameTime)), heavyLines, Fg(Blue)))
		if seen[got] {
			t.Errorf("frame %d is the one before it: %q", f, got)
		}
		late := StripStyles(drawLabel(l.name, 1-l.at(l.start.Add(time.Duration(f)*frameTime+31*time.Millisecond)), heavyLines, Fg(Blue)))
		if f < 7 && late != got {
			t.Errorf("frame %d changes within its 32ms: %q, then %q", f, got, late)
		}
		seen[got] = true
		if f == 7 && !strings.Contains(got, "┫Compare┣") {
			t.Errorf("frame 8 is not all open: %q", got)
		}
	}
}

// TestModeLabelLayers: a layer more grows from the name's tail a character
// every 32ms, the right end and the title still; a layer less goes back the
// same way; a switch halfway turns it; y leaving the whole mode closes the
// whole name from its middle (panel-149, panel-150; note-34).
func TestModeLabelLayers(t *testing.T) {
	ta, clk := logApp(t)
	ta.Viewer(&Viewer{Title: "Log", Lines: []string{"one two three"}})
	ta.settle()
	label := func() (string, int) {
		t.Helper()
		for _, l := range strings.Split(ta.screenText(), "\n") {
			if i, j := strings.Index(l, "┫"), strings.Index(l, "┣"); i >= 0 && j > i {
				return l[i+len("┫") : j], Width(l[:j])
			}
		}
		return "", -1
	}
	title := func() int { return colOf(strings.Split(ta.screenText(), "\n"), "LOG") }
	step := func(d time.Duration, want string) {
		t.Helper()
		clk.t = clk.t.Add(d)
		if got, _ := label(); got != want {
			t.Errorf("after %v: %q, want %q", d, got, want)
		}
	}
	ta.Update2(keyMsg("v"))
	step(6*frameTime, "Visual")
	ta.Update(labelMsg{})
	_, right := label()
	head := title()
	ta.Update2(keyMsg("v")) // a layer more
	step(0, "Visual")
	step(frameTime, "Visual>")
	step(frameTime, "Visual>s")
	if _, r := label(); r != right || title() != head {
		t.Errorf("the right end moved %d to %d, or the title %d to %d", right, r, head, title())
	}
	ta.Update2(keyMsg("Esc")) // halfway: turns where it is
	step(frameTime, "Visual>")
	ta.Update2(keyMsg("v")) // and turns again
	step(frameTime, "Visual>s")
	step(6*frameTime, "Visual>select")
	ta.Update(labelMsg{})
	if text := ta.screenText(); !strings.Contains(text, "y:copy Esc:unselect") {
		t.Errorf("selecting, the hint:\n%s", text)
	}
	ta.Update2(keyMsg("Esc")) // a layer less: from the tail
	step(frameTime, "Visual>selec")
	step(6*frameTime, "Visual")
	ta.Update(labelMsg{})
	if _, r := label(); r != right {
		t.Errorf("back to Visual the right end is at %d, was %d", r, right)
	}
	ta.Update2(keyMsg("v"))
	clk.t = clk.t.Add(7 * frameTime)
	ta.Update(labelMsg{})
	ta.Update2(keyMsg("y")) // out of the whole mode from a layer: closes from the middle
	step(0, "Visual>select")
	clk.t = clk.t.Add(2 * frameTime) // two characters off, one each end: from the tail it would be Visual>sele
	if got, _ := label(); got != "isual>selec" {
		t.Errorf("leaving with y does not close from the middle: %q", got)
	}
	takeClipboard()
}

// TestModeMenuLayers: the selection mode's Space menu once selecting names
// the whole mode, VISUAL>SELECT MODE, and leaves with [Esc] Unselect.
func TestModeMenuLayers(t *testing.T) {
	ta := newTestApp(t)
	ta.Viewer(&Viewer{Title: "Log", Lines: []string{"one two"}})
	ta.settle()
	ta.press("v", "v", "Space")
	if text := ta.screenText(); !strings.Contains(text, "VISUAL>SELECT MODE") || !strings.Contains(text, "[Esc] Unselect") {
		t.Errorf("the menu once selecting:\n%s", text)
	}
}

// TestModeLabelLayersRoom: while a layer grows the title keeps the longest
// name's room, so a long title is cut the same all the while; y halfway
// through closes from the middle what is shown (panel-150).
func TestModeLabelLayersRoom(t *testing.T) {
	ta, clk := logApp(t)
	ta.Viewer(&Viewer{Title: strings.Repeat("long title ", 20), Lines: []string{"one two three"}})
	ta.settle()
	title := func() string {
		for _, l := range strings.Split(ta.screenText(), "\n") {
			if i := strings.Index(l, glyphRoundRight); i >= 0 && strings.Contains(l, "LONG") {
				return l[:i]
			}
		}
		return ""
	}
	ta.Update2(keyMsg("v"))
	clk.t = clk.t.Add(6 * frameTime)
	ta.Update(labelMsg{})
	ta.Update2(keyMsg("v"))
	first := title()
	for range 7 {
		clk.t = clk.t.Add(frameTime)
		if got := title(); got != first {
			t.Fatalf("the title moved while the layer grew:\n%q\n%q", first, got)
		}
	}
	ta.Update(labelMsg{})
	ta.Update2(keyMsg("Esc"))
	clk.t = clk.t.Add(7 * frameTime)
	ta.Update(labelMsg{})
	ta.Update2(keyMsg("v"))
	clk.t = clk.t.Add(2 * frameTime) // Visual>s, growing
	ta.Update2(keyMsg("y"))
	clk.t = clk.t.Add(2 * frameTime)
	if !strings.Contains(ta.screenText(), "┫isual>┣") {
		t.Errorf("y while growing does not close from the middle:\n%s", ta.screenText())
	}
	takeClipboard()
}

// TestTerminalSelectEscLayer: on a PTY too Esc goes a layer back: selecting,
// back to Visual; then out (K11-3).
func TestTerminalSelectEscLayer(t *testing.T) {
	ta, term, _ := termWithHistory(t)
	ta.Update(altV)
	ta.settle()
	ta.press("v")
	if term.modeName() != "Visual>select" {
		t.Fatalf("selecting on a PTY: %q", term.modeName())
	}
	ta.press("Esc")
	if term.Mode() != "Visual" || term.sel.selecting {
		t.Errorf("Esc while selecting: %q, selecting %v", term.Mode(), term.sel.selecting)
	}
	ta.press("Esc")
	if term.Mode() != "" {
		t.Errorf("Esc in Visual: still %q", term.Mode())
	}
}
