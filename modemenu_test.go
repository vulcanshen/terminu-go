package terminu

import (
	"strings"
	"testing"
)

func menuOnTop(t *testing.T, ta *testApp) *menuView {
	t.Helper()
	m, ok := ta.topView().(*menuView)
	if !ok || !m.space {
		t.Fatalf("Space did not open the mode's menu: %T", ta.topView())
	}
	return m
}

func cursorLabel(m *menuView) string { return m.rows[m.cursor].action.Label }

// TestModeSpaceMenu: Space in a panel's mode opens its Space menu, titled
// as the panel: ITEM OPERATION, the mode's keys under COMPARE MODE, the
// movement keys only shown in Blue, the cursor passing them; a row does as
// its key; Esc only closes the menu; [Esc] Leave leaves (menu-23 to 27,
// K11-1, screen-8).
func TestModeSpaceMenu(t *testing.T) {
	ta, _ := compareApp(t)
	ta.press("C")
	if f := covCFooter(ta); !strings.HasPrefix(f, "Space:menu ?:help C|Enter:compare") {
		t.Errorf("the mode's footer: %q", f)
	}
	ta.press("Space")
	m := menuOnTop(t, ta)
	text := ta.screenText()
	for _, want := range []string{"[1] PODS", "ITEM OPERATION", "[C]ompare", "with pod-00", "COMPARE MODE", "j|k", "down, up", "[Esc] Leave"} {
		if !strings.Contains(text, want) {
			t.Errorf("%q is not in the menu:\n%s", want, text)
		}
	}
	if strings.Contains(text, "PANEL OPERATION") || strings.Contains(text, "Global operation") || strings.Contains(text, "[m]ark") {
		t.Errorf("the panel's own actions are in the mode's menu:\n%s", text)
	}
	if got := cursorLabel(m); got != "Leave" {
		t.Errorf("the cursor is on %q: Compare is off on the anchor, the movement keys are passed", got)
	}
	if !strings.Contains(strings.Join(ta.Frame(), "\n"), renderKey("j|k", Fg(Blue), Overlay0)) {
		t.Error("a movement key is not Blue")
	}
	ta.press("Esc")
	if ta.top() != nil || ta.pods.InMode() == nil {
		t.Fatal("Esc in the menu leaves the mode or keeps the menu")
	}
	ta.press("j", "Space")
	if m := menuOnTop(t, ta); cursorLabel(m) != "Compare" {
		t.Fatalf("off the anchor the cursor is on %q", cursorLabel(m))
	}
	ta.press("Enter")
	if _, ok := ta.topView().(*viewerView); !ok || !strings.Contains(ta.screenText(), "DIFF — pod-00 ↔ pod-01") {
		t.Fatalf("[C]ompare from the menu did not compare: %T", ta.topView())
	}
	ta.press("Esc", "Space")
	m = menuOnTop(t, ta)
	ta.press("j")
	if got := cursorLabel(m); got != "Leave" {
		t.Fatalf("j went to %q", got)
	}
	ta.press("Enter")
	if ta.top() != nil || ta.pods.InMode() != nil || ta.list.Anchor != "" {
		t.Error("[Esc] Leave did not leave the mode")
	}
}

// TestModeSpaceMenuInPopups: in a viewer's selection mode [v] Select, then
// [y] Copy, titled as the viewer; in a reorder's Drag mode [Enter] Drop;
// in a terminal's History Space opens its menu, not the program's
// (K5-3, reorder-19).
func TestModeSpaceMenuInPopups(t *testing.T) {
	ta := newTestApp(t)
	ta.Viewer(&Viewer{Title: "Log", Lines: []string{"one two", "three"}})
	ta.settle()
	ta.press("v", "Space")
	m := menuOnTop(t, ta)
	if text := ta.screenText(); !strings.Contains(text, "[v] Select") || !strings.Contains(text, "VISUAL MODE") || !strings.Contains(m.cased, "LOG") {
		t.Errorf("the selection mode's menu (%q):\n%s", m.cased, text)
	}
	ta.press("Enter", "l", "Space")
	if !strings.Contains(ta.screenText(), "[y] Copy") {
		t.Errorf("selecting, the menu offers Copy:\n%s", ta.screenText())
	}
	ta.press("Enter")
	if v := ta.topView().(*viewerView).v; v.inMode || !strings.Contains(ta.screenText(), "Copied") || len(takeClipboard()) == 0 {
		t.Error("[y] Copy from the menu did not copy and leave")
	}
	ta.press("Esc", "Esc")

	var got []int
	ta.Reorder(testReorder(&got))
	ta.settle()
	rv := ta.topView().(*reorderView)
	ta.press("D", "j", "Space")
	if !strings.Contains(ta.screenText(), "[Enter] Drop") || !strings.Contains(ta.screenText(), "DRAG MODE") {
		t.Errorf("the Drag mode's menu:\n%s", ta.screenText())
	}
	ta.press("Enter")
	if rv.held || ta.topView() != rv {
		t.Error("[Enter] Drop from the menu did not drop it")
	}
	ta.press("Esc")
	ta.covBCloseAll()

	ta2, term, sent := termWithHistory(t)
	ta2.press("PgUp")
	before := sent.String()
	ta2.press("Space")
	menuOnTop(t, ta2)
	if sent.String() != before || !strings.Contains(ta2.screenText(), "[Esc] Live") {
		t.Errorf("Space in History: sent %q, menu:\n%s", sent.String()[len(before):], ta2.screenText())
	}
	ta2.press("Enter")
	if term.Mode() != "" {
		t.Errorf("[Esc] Live from the menu: still %q", term.Mode())
	}
}
