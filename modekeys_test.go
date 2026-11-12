package terminu

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestK12FreeKeyInAMode: in a mode a movement key with nothing to move may do
// an action of the mode's own, its arrow key the same, shown as the mode's
// other keys are; outside the mode the key moves as ever (tdp K12-19).
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
