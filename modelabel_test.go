package terminu

import (
	"strings"
	"testing"
	"time"
)

// TestModeLabelOpens: entering a mode its name opens from the middle on the
// top border in about 288ms by the clock, the junctions side by side first;
// the mode and the frame's colour change at once; a switch halfway turns it
// where it is, and leaving plays it back to the line (panel-146 to 148).
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
	step(144*time.Millisecond, "━┫ra┣━", "Drag")
	ta.Update2(keyMsg("Esc")) // halfway: it turns where it is
	if ta.pods.InMode() != nil {
		t.Fatal("Esc did not leave the mode at once")
	}
	step(0, "━┫ra┣━", "")
	step(72*time.Millisecond, "━┫r┣━", "")
	ta.Update2(keyMsg("D")) // turns again, from a quarter open
	step(72*time.Millisecond, "━┫ra┣━", "")
	step(144*time.Millisecond, "━┫Drag┣━", "")
	ta.Update(labelMsg{})
	if ta.labelsMoving() {
		t.Error("still moving once open")
	}
	ta.Update2(keyMsg("Esc"))
	step(288*time.Millisecond, "━━━━━━━━", "┫")
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
	clk.t = clk.t.Add(288 * time.Millisecond)
	if !strings.Contains(ta.screenText(), "┫Visual┣") {
		t.Errorf("the viewer's mode name does not open:\n%s", ta.screenText())
	}
}
