package terminu

import (
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// spaceMode is a mode of the app's own whose drop the user put on Space, the
// arrow key under it left as the mode's default.
func spaceMode(did *[]string, off *bool) *Mode {
	do := func(what string) func(string) tea.Cmd {
		return func(string) tea.Cmd { *did = append(*did, what); return nil }
	}
	return &Mode{Name: "Play", Exit: "pause", Keys: []ModeKey{
		{Key: "r", Desc: "restart", Do: do("restart")},
		{Key: "h|l", Keys: []string{"h", "l", "←", "→"}, Desc: "left, right", NoFooter: true, Do: do("move")},
		{Key: "Space", Keys: []string{"Space", "↓"}, Desc: "drop", NoFooter: true, Do: do("drop"),
			Disabled: func() bool { return *off }},
	}}
}

// TestK11UserSetSpace: in a mode of the app's own, an action the user put
// on Space is done by Space and the mode has no Space menu; the footer writes
// Space:<action> first, where Space:menu was, and keeps it when narrow; ?
// lists it among the mode's keys, not among the core keys; disabled, Space
// does nothing; Esc still leaves, and Space then opens the panel's Space
// menu (tdp K11-27, K11-1, K11-15, K11-20, screen.2, K5).
func TestK11UserSetSpace(t *testing.T) {
	ta := newTestApp(t)
	var did []string
	off := false
	ta.pods.EnterMode(spaceMode(&did, &off))
	ta.press("Space")
	if ta.top() != nil || strings.Join(did, ",") != "drop" {
		t.Fatalf("Space: %T on top, did %v", ta.topView(), did)
	}
	if f := covCFooter(ta); f != "Space:drop ?:help r:restart Esc:pause" {
		t.Errorf("the footer: %q", f)
	}
	if f := StripStyles(ta.footer(30)); f != "Space:drop ?:help Esc:pause" {
		t.Errorf("a narrow footer: %q", f)
	}

	ta.press("?")
	_, plain := covEPopup(ta)
	keys := covEKeys(plain)
	if !slices.Contains(keys["PLAY"], "Space|↓") || slices.Contains(keys["CORE KEYS"], "Space") {
		t.Errorf("the key reference: %v", keys)
	}
	if !strings.Contains(strings.Join(plain, "\n"), "drop") {
		t.Errorf("the key reference does not say what Space does:\n%s", strings.Join(plain, "\n"))
	}
	ta.press("Esc")
	ta.pods.InMode().Keys[2].Keys = nil // Space alone, the arrow key given up
	ta.press("?")
	_, plain = covEPopup(ta)
	if keys := covEKeys(plain); !slices.Contains(keys["PLAY"], "Space") || slices.Contains(keys["CORE KEYS"], "Space") {
		t.Errorf("the key reference, Space alone: %v", keys)
	}
	ta.press("Esc")

	off = true
	ta.press("Space")
	if ta.top() != nil || len(did) != 1 {
		t.Fatalf("Space disabled: %T on top, did %v", ta.topView(), did)
	}
	off = false

	ta.press("Esc")
	if ta.pods.InMode() != nil {
		t.Fatal("Esc does not leave the mode")
	}
	ta.press("Space")
	menuOnTop(t, ta)
	if text := ta.screenText(); strings.Contains(text, "PLAY MODE") || len(did) != 1 {
		t.Errorf("Space out of the mode: did %v\n%s", did, text)
	}
}

// TestK11SpaceStaysTheMenu: Space opens the mode's Space menu where the user
// put no action on it, a ModeKey taking Space with no Do being the core key;
// in a layer, the innermost layer's keys say what Space does (tdp K11).
func TestK11SpaceStaysTheMenu(t *testing.T) {
	ta := newTestApp(t)
	ta.pods.EnterMode(&Mode{Name: "Play", Exit: "pause", Keys: []ModeKey{
		{Key: "r", Desc: "restart", Do: func(string) tea.Cmd { return nil }},
		{Key: "Space", Desc: "menu"}}})
	ta.press("Space")
	if got := menuItems(t, ta); !strings.HasPrefix(got, "Restart,") || !strings.Contains(ta.screenText(), "PLAY MODE") {
		t.Errorf("a Space with no Do: not the mode's Space menu, %s", got)
	}
	ta.press("Esc", "?")
	_, plain := covEPopup(ta)
	if keys := covEKeys(plain); !slices.Contains(keys["CORE KEYS"], "Space") || slices.Contains(keys["PLAY"], "Space") {
		t.Errorf("a Space with no Do is not listed once, among the core keys: %v", keys)
	}
	ta.press("Esc", "Esc")

	var did []string
	off := false
	ta.pods.EnterMode(spaceMode(&did, &off))
	ta.pods.EnterLayer(&Mode{Name: "Aim", Exit: "back", Keys: []ModeKey{
		{Key: "a", Desc: "aim", Do: func(string) tea.Cmd { return nil }}}})
	if f := covCFooter(ta); f != "Space:menu ?:help a:aim Esc:back" {
		t.Errorf("the layer's footer: %q", f)
	}
	ta.press("Space")
	if text := ta.screenText(); !strings.Contains(text, "AIM MODE") || len(did) != 0 {
		t.Fatalf("Space in a layer with no action on it: did %v\n%s", did, text)
	}
	ta.press("Esc", "Esc")
	if m := ta.pods.InMode(); m == nil || m.Name != "Play" {
		t.Fatalf("Esc does not go back to the outer layer: %v", m)
	}
	ta.press("Space")
	if ta.top() != nil || strings.Join(did, ",") != "drop" {
		t.Errorf("Space back in the outer layer: %T on top, did %v", ta.topView(), did)
	}
}
