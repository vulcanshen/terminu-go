package terminu

import (
	"slices"
	"testing"
)

// pointedApp is the test app with its list's rows standing for panels: pod-00
// for [2] LOG, pod-01 for [3] SETTINGS, pod-02 for the other screen's panel,
// the rest for none.
func pointedApp(t *testing.T) *testApp {
	ta := newTestApp(t)
	panels := ta.screens[0].Panels
	other := ta.screens[1].Panels[0]
	ta.list.Points = func(r Row) *Panel {
		return map[string]*Panel{"0": panels[1], "1": panels[2], "2": other}[r.ID]
	}
	return ta
}

// frameAt is the colour and glyph of the top-left corner of the panel x
// columns in, and the colour of its [N] capsule.
func frameAt(ta *testApp, x int, n string) (corner covCCell, capsule Color) {
	top := ta.Frame()[2]
	corner = covCAt(top, x)
	for _, c := range covCCells(top) {
		if c.Col > x && c.Ch == n {
			return corner, c.St.Bg
		}
	}
	return corner, Color{}
}

// TestPointedPanel: the panel the focused list's cursor row stands for has a
// Lavender frame, still thin and square, its capsules with it; the others
// keep theirs; its content is dimmed as any panel's without the focus
// (panel-151).
func TestPointedPanel(t *testing.T) {
	ta := pointedApp(t)
	if c, cp := frameAt(ta, 40, "2"); c.Ch != lightLines.tl || c.St.Fg != Lavender || cp != Lavender {
		t.Errorf("[2] LOG pointed at: %q in %v, its capsule on %v", c.Ch, c.St.Fg, cp)
	}
	if c, cp := frameAt(ta, 80, "3"); c.St.Fg != Surface2 || cp != Surface2 {
		t.Errorf("[3] SETTINGS not pointed at: %v, its capsule on %v", c.St.Fg, cp)
	}
	if c, _ := frameAt(ta, 0, "1"); c.Ch != heavyLines.tl || c.St.Fg != Blue {
		t.Errorf("the list keeps the focus: %q in %v", c.Ch, c.St.Fg)
	}
	ta.press("j")
	settings := ta.screens[0].Panels[2]
	pointed, plain := settings.draw(3, 40, 10, false, true), settings.draw(3, 40, 10, false, false)
	inside := func(line string) []covCCell { // the cells inside the frame
		var out []covCCell
		for _, c := range covCCells(line) {
			if c.Col > 0 && c.Col < 39 {
				out = append(out, c)
			}
		}
		return out
	}
	for i := 1; i < 9; i++ {
		if !slices.Equal(inside(pointed[i]), inside(plain[i])) {
			t.Errorf("row %d of a pointed panel is not as without the focus:\n%q\n%q", i, pointed[i], plain[i])
		}
	}
}

// TestPointedFollowsCursor: the cursor on another row points at that row's
// panel; a row whose panel is not on the screen, or that stands for none,
// points at nothing (panel-152).
func TestPointedFollowsCursor(t *testing.T) {
	ta := pointedApp(t)
	ta.press("j")
	if c, _ := frameAt(ta, 80, "3"); c.St.Fg != Lavender {
		t.Errorf("pod-01: [3] SETTINGS in %v", c.St.Fg)
	}
	if c, _ := frameAt(ta, 40, "2"); c.St.Fg != Surface2 {
		t.Errorf("pod-01: [2] LOG still in %v", c.St.Fg)
	}
	for _, key := range []string{"j", "j"} { // pod-02, its panel on the other screen; pod-03, none
		ta.press(key)
		if p := ta.screens[0].pointed(3); p != nil {
			t.Errorf("after %s, %q is pointed at", key, p.Label)
		}
		for _, x := range []int{40, 80} {
			if c, _ := frameAt(ta, x, "x"); c.St.Fg != Surface2 {
				t.Errorf("after %s, the panel at %d in %v", key, x, c.St.Fg)
			}
		}
	}
	// A panel the layout has no room for is not on the screen either.
	s := ta.screens[0]
	s.Layout = func(w, h int) []Rect { return []Rect{{X: 0, Y: 0, W: w / 2, H: h}, {X: w / 2, Y: 0, W: w - w/2, H: h}} }
	ta.press("k", "k")
	if p := s.pointed(2); p != nil {
		t.Errorf("pod-01 points at %q, not shown", p.Label)
	}
	ta.press("k")
	if c, _ := frameAt(ta, 60, "2"); c.St.Fg != Lavender {
		t.Errorf("pod-00 with two shown: [2] LOG in %v", c.St.Fg)
	}
	// With no row under the cursor nothing is pointed at.
	ta.list.Points = func(Row) *Panel { return s.Panels[1] }
	ta.list.SetRows(nil)
	if p := s.pointed(2); p != nil {
		t.Errorf("an empty list points at %q", p.Label)
	}
}

// TestPointedOnlyFromFocus: the list points only while it has the focus: on
// the pointed panel it is drawn focused, on another nothing is pointed at;
// under a popup the pointed panel is dimmed with the rest (panel-153).
func TestPointedOnlyFromFocus(t *testing.T) {
	ta := pointedApp(t)
	ta.press("Tab")
	if c, _ := frameAt(ta, 40, "2"); c.Ch != heavyLines.tl || c.St.Fg != Blue {
		t.Errorf("focus on [2] LOG: %q in %v", c.Ch, c.St.Fg)
	}
	ta.press("Tab")
	for _, x := range []int{0, 40} {
		if c, _ := frameAt(ta, x, "x"); c.St.Fg != Surface2 {
			t.Errorf("focus on [3] SETTINGS: the panel at %d in %v", x, c.St.Fg)
		}
	}
	ta.press("Tab", "Space")
	if c, _ := frameAt(ta, 40, "2"); c.St.Fg != DimColor(Lavender) {
		t.Errorf("under the Space menu [2] LOG is in %v, want %v", c.St.Fg, DimColor(Lavender))
	}
}
