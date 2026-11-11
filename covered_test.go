package terminu

import "testing"

// TestCovered: Covered is true while a popup is over the screen, one that is
// closing until it is gone, and while the splash is; a toast is not a popup
// and leaves it false (tgo #3; tdp F1).
func TestCovered(t *testing.T) {
	ta := newTestApp(t)
	ta.Splash = func(int, int, int) ([]string, bool) { return []string{"test"}, true }
	if ta.Covered() {
		t.Fatal("covered with nothing open")
	}
	ta.Update(ta.Toast(Info, "Hello")())
	ta.settle()
	if ta.Covered() {
		t.Error("a toast covers the screen")
	}
	ta.press("Esc") // the toast goes first (tdp K4)
	for _, key := range []string{"?", "Space", "q"} {
		ta.press(key)
		if !ta.Covered() {
			t.Errorf("%s: its popup does not cover the screen", key)
		}
		ta.Update2(keyMsg("Esc"))
		ta.Update(animMsg{})
		if !ta.Covered() {
			t.Errorf("%s: its popup no longer covers the screen as it closes", key)
		}
		ta.settle()
		if ta.Covered() {
			t.Errorf("%s: still covered once its popup is gone", key)
		}
	}
	ta.press("V")
	if !ta.Covered() {
		t.Error("the splash does not cover the screen")
	}
	ta.press("j") // any key closes it
	if ta.Covered() {
		t.Error("still covered once the splash is gone")
	}
}
