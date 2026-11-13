package terminu

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestK12FreeKeyInAMode: in a mode a movement key with nothing to move may do
// an action of the mode's own, its arrow key the same, shown in ? and the
// footer, and in the Space menu of a mode that lists it (K11); outside the
// mode the key moves as ever (tdp K12-19).
func TestK12FreeKeyInAMode(t *testing.T) {
	ta := newTestApp(t)
	turned := 0
	ta.pods.EnterMode(&Mode{Name: "Play", Exit: "stop", Keys: []ModeKey{{Key: "k", Keys: []string{"k", "↑"},
		Desc: "turn the piece", Label: "Turn", Do: func(string) tea.Cmd { turned++; return nil }}}})
	ta.press("j") // nothing else in the mode moves
	before, _ := ta.list.Current()
	ta.press("k", "↑")
	if now, _ := ta.list.Current(); turned != 2 || now.ID != before.ID {
		t.Errorf("k and ↑ in the mode: turned %d times, the cursor %s then %s", turned, before.ID, now.ID)
	}
	if foot := StripStyles(ta.footer(200)); !strings.Contains(foot, "k:turn the piece") {
		t.Errorf("the footer: %q", foot)
	}
	ta.press("?")
	if keys := covAKeys(covARef(t, ta)); !keys["k"] {
		t.Errorf("the mode's key reference has no k: %v", keys)
	}
	ta.press("Esc", "Space")
	if !strings.Contains(ta.screenText(), "Turn") {
		t.Errorf("the mode's Space menu has no Turn:\n%s", ta.screenText())
	}
	ta.press("Esc", "Esc", "j") // the menu, the mode; then down a row
	if now, _ := ta.list.Current(); now.ID == before.ID {
		t.Fatal("out of the mode j does not move")
	}
	ta.press("k")
	if now, _ := ta.list.Current(); turned != 2 || now.ID != before.ID {
		t.Errorf("out of the mode k turned (%d) or did not move up: %s", turned, now.ID)
	}
}

// TestK11ModeKeysTheUserSets: a mode's key the user has set works, and the
// footer, the key reference and the mode's Space menu show the key in force
// (tdp K11-14; refusing a key already in use, with a Warning, is the app's).
func TestK11ModeKeysTheUserSets(t *testing.T) {
	ta := newTestApp(t)
	dropped := 0
	m := &Mode{Name: "Play", Exit: "stop", Keys: []ModeKey{{Key: "Enter", Desc: "drop the piece", Label: "Drop",
		Do: func(string) tea.Cmd { dropped++; return nil }}}}
	ta.pods.EnterMode(m)
	m.Keys[0].Key = "x" // the user set it
	ta.press("Enter", "x")
	if dropped != 1 {
		t.Errorf("Enter then x dropped %d times, want once (x)", dropped)
	}
	if foot := StripStyles(ta.footer(200)); !strings.Contains(foot, "x:drop the piece") || strings.Contains(foot, "Enter:drop") {
		t.Errorf("the footer: %q", foot)
	}
	ta.press("?")
	if keys := covAKeys(covARef(t, ta)); !keys["x"] || keys["Enter"] {
		t.Errorf("the mode's key reference: %v", keys)
	}
	ta.press("Esc", "Space")
	if text := ta.screenText(); !strings.Contains(text, "Drop") {
		t.Errorf("the mode's Space menu has no Drop:\n%s", text)
	}
	ta.press("x") // the menu presses the key in force
	if dropped != 2 {
		t.Errorf("x in the mode's Space menu: dropped %d times", dropped)
	}
}

// menuItems is the labels of the rows of the menu on top.
func menuItems(t *testing.T, ta *testApp) string {
	t.Helper()
	var rows []string
	for _, r := range menuOnTop(t, ta).rows {
		if r.kind == menuItem {
			rows = append(rows, r.action.Label)
		}
	}
	return strings.Join(rows, ",")
}

// TestK11OwnModeSpaceMenu: what the Space menu of a mode of the app's own
// lists is the app's (NoMenu, NoLeaveRow); Space still opens it, titled as any
// mode's, and every key stays in ? and the footer; a menu left with nothing
// lists leaving all the same (tdp K11-15, menu-26).
func TestK11OwnModeSpaceMenu(t *testing.T) {
	ta := newTestApp(t)
	var did []string
	do := func(what string) func(string) tea.Cmd {
		return func(string) tea.Cmd { did = append(did, what); return nil }
	}
	ta.pods.EnterMode(&Mode{Name: "Play", Exit: "pause", NoLeaveRow: true, Keys: []ModeKey{
		{Key: "k", Keys: []string{"k", "↑"}, Desc: "turn", NoMenu: true, Do: do("turn")},
		{Key: "h|l", Keys: []string{"h", "l", "←", "→"}, Desc: "left, right", NoFooter: true, NoMenu: true, Do: do("move")},
		{Key: "r", Desc: "restart", Do: do("restart")},
		{Key: "s", Desc: "save", Do: do("save")},
	}})
	ta.press("Space")
	if got := menuItems(t, ta); got != "Restart,Save" {
		t.Errorf("the menu lists %s, want Restart,Save", got)
	}
	if text := ta.screenText(); !strings.Contains(text, "[1] PODS") || !strings.Contains(text, "PLAY MODE") {
		t.Errorf("the menu is not titled as a mode's:\n%s", text)
	}
	ta.press("Enter", "k")
	if strings.Join(did, ",") != "restart,turn" || ta.pods.InMode() == nil {
		t.Errorf("Restart from the menu, then k: did %v, in the mode %v", did, ta.pods.InMode() != nil)
	}
	if foot := StripStyles(ta.footer(200)); !strings.Contains(foot, "k:turn") || !strings.Contains(foot, "Esc:pause") {
		t.Errorf("the footer: %q", foot)
	}
	ta.press("?")
	if keys := covAKeys(covARef(t, ta)); !keys["k"] || !keys["h|l"] || !keys["r"] || !keys["Esc"] {
		t.Errorf("the mode's key reference: %v", keys)
	}
	ta.press("Esc", "Esc")
	if ta.top() != nil || ta.pods.InMode() != nil {
		t.Fatal("Esc does not leave the mode")
	}

	ta.pods.EnterMode(&Mode{Name: "Play", Exit: "pause", NoLeaveRow: true, Keys: []ModeKey{
		{Key: "k", Desc: "turn", NoMenu: true, Do: do("turn")}}})
	ta.press("Space")
	if got := menuItems(t, ta); got != "Pause" {
		t.Errorf("a menu with nothing listed lists %s, want Pause", got)
	}
	ta.press("Enter")
	if ta.top() != nil || ta.pods.InMode() != nil {
		t.Error("[Esc] Pause from the menu does not leave the mode")
	}
}

// openPopups counts the popups open, those closing left out.
func openPopups(ta *testApp) int {
	n := 0
	for _, l := range ta.stack {
		if !l.closing {
			n++
		}
	}
	return n
}

// TestK11LeaveOwnModePopup: leaving a mode of the app's own may open one
// popup (its OnExit): the mode is left by then, Esc on the popup closes it
// and stays outside, and an action of the popup's may go back into the mode;
// the Space menu's row leaving the mode does the same (tdp K11-16, K11-3, K4).
func TestK11LeaveOwnModePopup(t *testing.T) {
	ta := newTestApp(t)
	var m *Mode
	inModeAtExit := false
	m = &Mode{Name: "Play", Exit: "pause", Keys: []ModeKey{{Key: "r", Desc: "restart", Do: func(string) tea.Cmd { return nil }}},
		OnExit: func() tea.Cmd {
			inModeAtExit = ta.pods.InMode() != nil
			return ta.Confirm(Ask{Title: "Paused", Question: "Go on?", Verb: "go on"},
				func() tea.Cmd { ta.pods.EnterMode(m); return nil })
		}}
	ta.pods.EnterMode(m)
	ta.press("Esc")
	if _, ok := ta.topView().(*confirmView); !ok || openPopups(ta) != 1 {
		t.Fatalf("Esc leaving the mode: %T on top, %d popups", ta.topView(), openPopups(ta))
	}
	if inModeAtExit || ta.pods.InMode() != nil {
		t.Error("the mode is not left when its popup opens")
	}
	ta.press("Esc")
	if ta.top() != nil || ta.pods.InMode() != nil {
		t.Fatalf("Esc on the popup: %T on top, in the mode %v", ta.topView(), ta.pods.InMode() != nil)
	}
	ta.press("Esc")
	if ta.top() != nil || ta.pods.InMode() != nil {
		t.Error("Esc again opened something or went back into the mode")
	}
	ta.pods.EnterMode(m)
	ta.press("Esc", "Enter")
	if ta.top() != nil || ta.pods.InMode() != m {
		t.Error("the popup's Enter does not go back into the mode")
	}

	ta.press("Space")
	if got := menuItems(t, ta); got != "Restart,Pause" {
		t.Fatalf("the menu lists %s", got)
	}
	ta.press("j", "Enter")
	if _, ok := ta.topView().(*confirmView); !ok || openPopups(ta) != 1 || ta.pods.InMode() != nil {
		t.Fatalf("[Esc] Pause from the menu: %T on top, %d popups, in the mode %v", ta.topView(), openPopups(ta), ta.pods.InMode() != nil)
	}
	ta.press("Esc")
	if ta.top() != nil || ta.pods.InMode() != nil {
		t.Error("Esc on the popup opened from the menu does not close it outside the mode")
	}
}

// TestK11OwnModePopupGoesBack: a popup a key of a mode of the app's own
// opens, by the key or from the mode's Space menu, is drawn and answered as
// its component (a confirm: Enter accepts, Esc cancels, y does nothing);
// where it goes back to is the app's, and the menu's row closes the menu
// before the key runs: cancelled or accepted, it goes back to the mode, not
// to the menu (tdp K11-26, F4-2, T1-2).
func TestK11OwnModePopupGoesBack(t *testing.T) {
	ta := newTestApp(t)
	restarts := 0
	ta.pods.EnterMode(&Mode{Name: "Play", Exit: "pause", Keys: []ModeKey{{Key: "r", Desc: "restart",
		Do: func(string) tea.Cmd {
			return ta.Confirm(Ask{Title: "Restart", Question: "Start over?", Verb: "restart"},
				func() tea.Cmd { restarts++; return nil })
		}}}})
	for _, from := range [][]string{{"r"}, {"Space", "Enter"}} {
		ta.press(from...)
		if _, ok := ta.topView().(*confirmView); !ok || openPopups(ta) != 1 || ta.pods.InMode() == nil {
			t.Fatalf("%v: %T on top, %d popups, in the mode %v", from, ta.topView(), openPopups(ta), ta.pods.InMode() != nil)
		}
		if !strings.Contains(ta.screenText(), "Enter:restart Esc:cancel") {
			t.Errorf("%v: the confirm's hint:\n%s", from, ta.screenText())
		}
		ta.press("y")
		if _, ok := ta.topView().(*confirmView); !ok || restarts != 0 {
			t.Fatalf("%v: y answered the confirm", from)
		}
		ta.press("Esc")
		if ta.top() != nil || ta.pods.InMode() == nil || restarts != 0 {
			t.Fatalf("%v, Esc: %T on top, in the mode %v, %d restarts", from, ta.topView(), ta.pods.InMode() != nil, restarts)
		}
		ta.press(from...)
		ta.press("Enter")
		if ta.top() != nil || ta.pods.InMode() == nil || restarts != 1 {
			t.Fatalf("%v, Enter: %T on top, in the mode %v, %d restarts", from, ta.topView(), ta.pods.InMode() != nil, restarts)
		}
		restarts = 0
	}
}

// TestScreen36CoreKeyInModeFooter: the footer of a mode of the app's own may
// write a core key among its keys, a ModeKey with no Do (q:quit); the key does
// what it does in any mode, q the quit flow, its row in the Space menu too,
// and ? lists it once, among the core keys (tdp screen-36, screen-8, K11).
func TestScreen36CoreKeyInModeFooter(t *testing.T) {
	ta := newTestApp(t)
	ta.pods.EnterMode(&Mode{Name: "Play", Exit: "pause", Keys: []ModeKey{
		{Key: "r", Desc: "restart", Do: func(string) tea.Cmd { return nil }},
		{Key: "q", Desc: "quit"}}})
	if f := covCFooter(ta); !strings.HasPrefix(f, "Space:menu ?:help r:restart q:quit Esc:pause") {
		t.Errorf("the footer: %q", f)
	}
	ta.press("q")
	if !covAQuitConfirm(ta) {
		t.Fatal("q in the mode does not start the quit flow")
	}
	ta.press("Esc")
	if ta.top() != nil || ta.pods.InMode() == nil {
		t.Fatal("closing the quit confirm left the mode")
	}
	ta.press("?")
	n := 0
	for _, r := range covARef(t, ta) {
		if f := strings.Fields(r); len(f) > 0 && (f[0] == "q" || strings.HasPrefix(f[0], "q|")) {
			n++
		}
	}
	if n != 1 {
		t.Errorf("? lists q %d times, want once:\n%s", n, strings.Join(covARef(t, ta), "\n"))
	}
	ta.press("Esc", "Space")
	if got := menuItems(t, ta); got != "Restart,Quit,Pause" {
		t.Fatalf("the menu lists %s", got)
	}
	ta.press("j", "Enter")
	if !covAQuitConfirm(ta) {
		t.Errorf("Quit from the mode's Space menu: %T on top", ta.topView())
	}
	// a key of the mode's own taking a core key too is listed all the same
	if (ModeKey{Key: "p", Keys: []string{"Esc", "p"}}).coreListed() || !(ModeKey{Key: "q|Ctrl-C"}).coreListed() {
		t.Error("coreListed: Esc|p is the mode's own, q|Ctrl-C a core key")
	}
}

// TestK11LeaveModeClosesItsPopups: however a mode of the app's own is left
// (here LeaveMode, as a game over does), its Space menu and its ? close with
// it, and what is open over them; a popup that is not the mode's stays, and
// one the app opens then is the only one (tdp K11-21, K11-16).
func TestK11LeaveModeClosesItsPopups(t *testing.T) {
	ta := newTestApp(t)
	m := &Mode{Name: "Play", Exit: "pause", Keys: []ModeKey{{Key: "r", Desc: "restart", Do: func(string) tea.Cmd { return nil }}}}
	for _, keys := range [][]string{{"Space"}, {"?"}, {"Space", "?"}} {
		ta.pods.EnterMode(m)
		ta.press(keys...)
		if openPopups(ta) != len(keys) {
			t.Fatalf("%v: %d popups open", keys, openPopups(ta))
		}
		ta.pods.LeaveMode()
		if ta.top() != nil || ta.pods.InMode() != nil {
			t.Errorf("%v then LeaveMode: %T still on top", keys, ta.topView())
		}
		ta.Confirm(Ask{Title: "Game over", Question: "Again?", Verb: "play"}, nil)
		ta.settle() // as when the app opens it in an update
		if _, ok := ta.topView().(*confirmView); !ok || openPopups(ta) != 1 {
			t.Errorf("%v: the popup the app opens then: %T, %d popups", keys, ta.topView(), openPopups(ta))
		}
		ta.press("Esc")
	}
	ta.pods.EnterMode(m)
	ta.Confirm(Ask{Title: "Hold", Question: "Hold on?", Verb: "hold"}, nil)
	ta.pods.LeaveMode()
	if _, ok := ta.topView().(*confirmView); !ok {
		t.Errorf("a popup not the mode's closed with it: %T", ta.topView())
	}
}
