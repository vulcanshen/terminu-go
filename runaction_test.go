package terminu

import (
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// runBody is a popup of the app's own whose keys run the app's actions, Esc
// and ? left to the lib.
type runBody struct {
	*testBody
	app *App
}

func (b *runBody) Key(key string, _ tea.KeyMsg) (bool, tea.Cmd) {
	if key == "Esc" || key == "?" {
		return false, nil
	}
	return true, b.app.RunAction(key)
}

// runApp is the test app with a panel operation that asks first, a disabled
// one and a mode with a key opening a popup that runs the app's actions.
func runApp(t *testing.T) (ta *testApp, m *Mode, exits *int) {
	t.Helper()
	ta = newTestApp(t)
	ta.pods.Actions = append(ta.pods.Actions,
		Action{Key: "n", Label: "New", Desc: "game", Scope: PanelScope,
			Ask: &Ask{Title: "New", Question: "New game?", Verb: "start"}, Do: ta.record("new")},
		Action{Key: "z", Label: "Zap", Desc: "the list", Scope: PanelScope,
			Disabled: func() bool { return true }, Do: ta.record("zap")},
		Action{Key: "f", Label: "Find", Desc: "a pod", Scope: PanelScope, Do: func() tea.Cmd {
			ta.ran = append(ta.ran, "find")
			return ta.Confirm(Ask{Title: "Find", Question: "Find it?", Verb: "find"}, nil)
		}})
	exits = new(int)
	m = &Mode{Name: "Play", Exit: "pause", OnExit: func() tea.Cmd { *exits++; return nil },
		Keys: []ModeKey{{Key: "s", Label: "Settings", Desc: "settings", Do: func(string) tea.Cmd { return ta.runPopup() }}}}
	return ta, m, exits
}

func (ta *testApp) runPopup() tea.Cmd {
	return ta.Popup(Popup{Title: "Settings", Body: &runBody{testBody: &testBody{w: 30, h: 5}, app: ta.App}})
}

func onRunPopup(ta *testApp) bool {
	v, ok := ta.topView().(*appPopup)
	if !ok {
		return false
	}
	_, ok = v.p.Body.(*runBody)
	return ok
}

// TestPopup74RunActionLeavesMode: a popup of the app's own opened in a mode
// of the app's own runs a panel operation that asks first as its menu does:
// the confirm opens over it, still in the mode; Esc on it changes nothing;
// Enter leaves the mode, its name, colour, Space menu and ? going with it, its
// OnExit not run, runs the action and, nothing new open, closes every popup
// (tdp popup-74, T1, F6).
func TestPopup74RunActionLeavesMode(t *testing.T) {
	ta, m, exits := runApp(t)
	opens := map[string]func(){
		"from the mode's key": func() { ta.press("s") },
		"from the mode's Space menu": func() {
			ta.press("Space")
			if got := menuItems(t, ta); got != "Settings,Pause" {
				t.Fatalf("the mode's menu lists %s", got)
			}
			ta.press("Enter")
		},
		"over the mode's ?": func() { ta.press("?"); ta.runPopup(); ta.settle() },
	}
	for _, how := range []string{"from the mode's key", "from the mode's Space menu", "over the mode's ?"} {
		ta.ran = nil
		ta.pods.EnterMode(m)
		opens[how]()
		n := openPopups(ta)
		if !onRunPopup(ta) || ta.pods.InMode() != m {
			t.Fatalf("%s: %T on top, in the mode %v", how, ta.topView(), ta.pods.InMode() != nil)
		}
		ta.press("n")
		if c, ok := ta.topView().(*confirmView); !ok || c.ask.Question != "New game?" || openPopups(ta) != n+1 {
			t.Fatalf("%s, n: %T on top, %d popups", how, ta.topView(), openPopups(ta))
		}
		if ta.pods.InMode() != m || len(ta.ran) != 0 {
			t.Errorf("%s: the confirm open, in the mode %v, ran %v", how, ta.pods.InMode() != nil, ta.ran)
		}
		ta.press("Esc")
		if !onRunPopup(ta) || openPopups(ta) != n || ta.pods.InMode() != m || len(ta.ran) != 0 || *exits != 0 {
			t.Fatalf("%s, Esc on the confirm: %T on top, %d popups, in the mode %v, ran %v, OnExit %d",
				how, ta.topView(), openPopups(ta), ta.pods.InMode() != nil, ta.ran, *exits)
		}
		ta.press("n", "Enter")
		if ta.top() != nil || ta.pods.InMode() != nil || !slices.Equal(ta.ran, []string{"new"}) || *exits != 0 {
			t.Fatalf("%s, Enter on the confirm: %T on top, in the mode %v, ran %v, OnExit %d",
				how, ta.topView(), ta.pods.InMode() != nil, ta.ran, *exits)
		}
		pods := covCCrop(ta.Frame(), 0, 40)
		if top := covCPlain(pods[2]); strings.Contains(top, "Play") || covCAt(pods[38], 0).St.Fg == Yellow {
			t.Errorf("%s: the mode's name or colour stayed: %q", how, top)
		}
		if f := covCFooter(ta); strings.Contains(f, "pause") {
			t.Errorf("%s: the footer is the mode's: %q", how, f)
		}
	}
}

// TestPopup74RunActionNothing: a disabled action, or a key with none, does
// nothing from the popup: the popup and the mode stay as they were.
func TestPopup74RunActionNothing(t *testing.T) {
	ta, m, exits := runApp(t)
	ta.pods.EnterMode(m)
	ta.press("s")
	for _, key := range []string{"z", "w", "y"} {
		if ta.press(key); !onRunPopup(ta) || openPopups(ta) != 1 || ta.pods.InMode() != m || len(ta.ran) != 0 || *exits != 0 {
			t.Errorf("%s: %T on top, %d popups, in the mode %v, ran %v, OnExit %d",
				key, ta.topView(), openPopups(ta), ta.pods.InMode() != nil, ta.ran, *exits)
		}
	}
	// No panel focused (nothing on the screen): no panics, nothing to run.
	if (&App{}).RunAction("n") != nil {
		t.Error("an app with no screen ran something")
	}
}

// TestPopup74RunActionQuit: q from the popup is the quit flow, asking first
// over it, and leaves no mode: Esc on the confirm is back on the popup, in
// the mode (tdp popup-74, K9).
func TestPopup74RunActionQuit(t *testing.T) {
	ta, m, exits := runApp(t)
	ta.pods.EnterMode(m)
	ta.press("s", "q")
	if !covAQuitConfirm(ta) || ta.pods.InMode() != m || openPopups(ta) != 2 {
		t.Fatalf("q: %T on top, in the mode %v, %d popups", ta.topView(), ta.pods.InMode() != nil, openPopups(ta))
	}
	ta.press("Esc")
	if !onRunPopup(ta) || ta.pods.InMode() != m || *exits != 0 {
		t.Errorf("Esc on the quit confirm: %T on top, in the mode %v, OnExit %d", ta.topView(), ta.pods.InMode() != nil, *exits)
	}
}

// TestPopup74RunActionFound: what RunAction finds is what the key runs on the
// focused panel out of a mode (tdp M4): its Enter action, its own operations
// before the global ones (a hidden one leaving its key to them, tdp M6), a
// screen's key; an action opening a popup keeps the one that called under
// it, as a menu stays (tdp T1).
func TestPopup74RunActionFound(t *testing.T) {
	ta, _, _ := runApp(t)
	hidden := true
	ta.pods.Actions = append(ta.pods.Actions, Action{Key: "S", Label: "Stop", Desc: "the job", Scope: PanelScope,
		Hidden: func() bool { return hidden }, Do: ta.record("stop")})
	for _, c := range []struct {
		key, ran string
		hidden   bool
	}{{"Enter", "logs", true}, {"x", "delete", true}, {"S", "start", true}, {"S", "stop", false}} {
		ta.ran, hidden = nil, c.hidden
		ta.runPopup()
		ta.settle()
		ta.press(c.key)
		if c.key == "x" { // it asks first
			ta.press("Enter")
		}
		if ta.top() != nil || !slices.Equal(ta.ran, []string{c.ran}) {
			t.Errorf("%s: %T on top, ran %v", c.key, ta.topView(), ta.ran)
		}
	}
	ta.ran = nil
	ta.runPopup()
	ta.settle()
	ta.press("f")
	if _, ok := ta.topView().(*confirmView); !ok || openPopups(ta) != 2 || !slices.Equal(ta.ran, []string{"find"}) {
		t.Errorf("an action opening a popup: %T on top, %d popups, ran %v", ta.topView(), openPopups(ta), ta.ran)
	}
	ta.press("Esc", "O")
	if ta.top() != nil || ta.Screen().Name != "Other" {
		t.Errorf("a screen's key: %T on top, on %s", ta.topView(), ta.Screen().Name)
	}
}
