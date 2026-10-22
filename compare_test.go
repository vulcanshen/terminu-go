package terminu

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestDiffLines: the shortest edit, line by line, the anchor's side first.
func TestDiffLines(t *testing.T) {
	ops := diffLines([]string{"a", "b", "c", "d"}, []string{"a", "c", "d", "e"})
	var got []string
	for _, o := range ops {
		got = append(got, fmt.Sprintf("%c%d,%d", o.op, o.a, o.b))
	}
	if want := " 0,0 -1,-1  2,1  3,2 +-1,3"; strings.Join(got, " ") != want {
		t.Errorf("ops %q, want %q", strings.Join(got, " "), want)
	}
	if len(diffLines(nil, []string{"x"})) != 1 || len(diffLines([]string{"x"}, nil)) != 1 {
		t.Error("one side empty")
	}
}

// compareApp is the test app with a Compare action on its pods: rows of the
// same age have the same text.
func compareApp(t *testing.T) (*testApp, *bool) {
	ta := newTestApp(t)
	split := new(bool)
	ta.pods.Actions = append(ta.pods.Actions, ta.CompareAction(ta.pods, ta.list, Compare{Key: "C", Split: split,
		Can: func(r Row) bool { return r.ID != "2" },
		Text: func(r Row) (string, error) {
			var b strings.Builder
			for i := range 20 {
				fmt.Fprintf(&b, "line %d\n", i)
			}
			return "age: " + r.Cells[1] + "\n" + b.String() + "kind: pod\n", nil
		}}))
	return ta, split
}

// TestCompareMode: the compare key enters the mode, its row the anchor;
// the compare keys are off on the anchor and on a row that cannot be
// compared; Enter compares; the same text is an Info toast; Esc leaves, the
// anchor gone (components/layout/panel, compare mode: panel-138 to 144).
func TestCompareMode(t *testing.T) {
	ta, _ := compareApp(t)
	ta.press("C")
	if m := ta.pods.InMode(); m == nil || m.Name != "Compare" || ta.list.Anchor != "0" {
		t.Fatalf("not in the compare mode: %v, anchor %q", m, ta.list.Anchor)
	}
	frame := strings.Join(ta.Frame(), "\n")
	if !strings.Contains(covCFooter(ta), "C|Enter:compare") || !strings.Contains(frame, Fg(Surface2).Render(":compare")) {
		t.Errorf("on the anchor the compare keys are off: %q", covCFooter(ta))
	}
	ta.press("Enter")
	if ta.top() != nil {
		t.Error("Enter on the anchor compared")
	}
	ta.press("j", "j") // pod-02 cannot be compared
	if ta.press("C"); ta.top() != nil {
		t.Error("compared a row that cannot be")
	}
	ta.press("Tab")
	if ta.Screen().Focused() != ta.pods || !strings.Contains(ta.screenText(), "Press [Esc] to leave Compare first") {
		t.Error("Tab left the mode's panel")
	}
	ta.toast = nil
	ta.press("k", "Enter")
	v, ok := ta.topView().(*viewerView)
	if !ok {
		t.Fatalf("no compare popup: %T", ta.topView())
	}
	if text := ta.screenText(); !strings.Contains(text, "DIFF — pod-00 ↔ pod-01") || !strings.Contains(text, "UNIFIED") {
		t.Errorf("the popup's title:\n%s", text)
	}
	if !slicesContain(v.v.Lines, Fg(Red).Render("-age: 0")) || !slicesContain(v.v.Lines, Fg(Green).Render("+age: 1")) {
		t.Errorf("UNIFIED: %q", v.v.Lines)
	}
	ta.press("Esc")
	if ta.pods.InMode() == nil || ta.list.Anchor != "0" {
		t.Error("closing the compare left the mode")
	}
	ta.press("G", "k", "k", "k", "k") // pod-35, the same age as pod-00
	if r, _ := ta.list.Current(); r.ID != "35" {
		t.Fatalf("at %s", r.ID)
	}
	ta.press("Enter")
	if ta.top() != nil || !strings.Contains(ta.screenText(), "No differences between pod-00 and pod-35") {
		t.Errorf("the same text:\n%s", ta.screenText())
	}
	ta.toast = nil
	if ta.press("g", "g"); ta.list.cursor != 0 {
		t.Errorf("gg in the mode: at %d", ta.list.cursor)
	}
	ta.press("Esc")
	if ta.pods.InMode() != nil || ta.list.Anchor != "" {
		t.Error("Esc did not leave the mode, or the anchor stayed")
	}
}

func slicesContain(lines []string, s string) bool {
	for _, l := range lines {
		if strings.Contains(l, s) {
			return true
		}
	}
	return false
}

// TestCompareHunks: only the stretches that changed, 3 lines either side,
// an Overlay0 line between them (note-81).
func TestCompareHunks(t *testing.T) {
	x := strings.Split("a b c d e f g h i j k l m n o", " ")
	y := append([]string(nil), x...)
	y[1], y[13] = "B", "N"
	lines := diffUnified(diffLines(x, y), x, y, 30)
	plain := make([]string, len(lines))
	for i, l := range lines {
		plain[i] = StripStyles(l)
	}
	if len(lines) != 6+1+6 || !strings.HasPrefix(plain[6], "───") || !strings.Contains(lines[6], Fg(Overlay0).Render(strings.Repeat("─", 30))) {
		t.Errorf("hunks:\n%s", strings.Join(plain, "\n"))
	}
	if plain[0] != " 1  1  a" || plain[1] != " 2    -b" || plain[2] != "    2 +B" {
		t.Errorf("numbers and signs: %q", plain[:3])
	}
}

// TestCompareSplit: SPLIT puts the anchor on the left, each half numbered,
// cut with …; it is remembered; under 96 columns it is off and UNIFIED is
// shown (note-83 to 85).
func TestCompareSplit(t *testing.T) {
	ta, split := compareApp(t)
	ta.press("C", "j", "Enter", "l")
	v := ta.topView().(*viewerView).v
	if v.Tab != 1 || !*split || !v.NoWrap {
		t.Fatalf("l did not switch to SPLIT: tab %d, split %v", v.Tab, *split)
	}
	if !slicesContain(v.Lines, Fg(Red).Render("age: 0")) || !slicesContain(v.Lines, Fg(Green).Render("age: 1")) {
		t.Errorf("SPLIT: %q", v.Lines)
	}
	row := StripStyles(v.Lines[0])
	if !strings.HasPrefix(row, " 1 age: 0") || !strings.Contains(row, " 1 age: 1") {
		t.Errorf("each half numbered: %q", row)
	}
	if got := diffSplit([]diffOp{{'-', 0, -1}}, []string{strings.Repeat("x", 40)}, nil, 30); !strings.Contains(StripStyles(got[0]), "…") || Width(got[0]) > 30 {
		t.Errorf("a long line is cut: %q", got)
	}
	ta.press("Esc", "Enter")
	if v := ta.topView().(*viewerView).v; v.Tab != 1 {
		t.Error("SPLIT is not remembered")
	}
	ta.press("Esc")
	ta.resize(90, 30)
	ta.press("Enter")
	v = ta.topView().(*viewerView).v
	if ta.press("l"); v.Tab != 0 || !*split {
		t.Errorf("under 96 columns: tab %d, remembered %v", v.Tab, *split)
	}
	if !strings.Contains(strings.Join(ta.Frame(), "\n"), Style{Fg: Surface2, Bg: Mantle}.Render("SPLIT")) {
		t.Error("SPLIT is not drawn off")
	}
}

// TestCompareAnchorGone: the anchor no longer in the list ends the mode with
// a Warning toast (panel-145).
func TestCompareAnchorGone(t *testing.T) {
	ta, _ := compareApp(t)
	ta.press("C")
	rows := ta.list.Rows()
	ta.list.SetRows(rows[1:])
	ta.press("j")
	if ta.pods.InMode() != nil || ta.list.Anchor != "" || !strings.Contains(ta.screenText(), "Compare ended: pod-00 is gone") {
		t.Errorf("the anchor gone:\n%s", ta.screenText())
	}
}

// TestModeKeyDisabled: a mode key that cannot be pressed now does nothing
// and is in the disabled colour in its key reference (Rules M6).
func TestModeKeyDisabled(t *testing.T) {
	ta := newTestApp(t)
	ran := false
	ta.pods.EnterMode(&Mode{Name: "Pick", Keys: []ModeKey{{Key: "p", Desc: "pick",
		Disabled: func() bool { return true }, Do: func(string) tea.Cmd { ran = true; return nil }}}})
	if ta.press("p"); ran {
		t.Error("a disabled mode key ran")
	}
	ta.press("?")
	if !strings.Contains(strings.Join(ta.Frame(), "\n"), Fg(Surface2).Render("pick")) {
		t.Error("the key reference does not show it off")
	}
}
