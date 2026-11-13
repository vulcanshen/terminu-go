package terminu

import (
	"maps"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestK11UserSetEnterAndMovement: in a mode of the app's own whose keys the
// user sets, Enter and the movement letters are keys of the mode like any:
// moving left set to a moves with a and ← alike, the h it left takes another
// action, Enter another, and PgUp does not follow an action put on u; the
// footer, ? and the Space menu write the keys as they are, the moved
// movement key kept out of the footer and only shown in the menu. With no
// pair starting with g in the mode, g alone is a key at once; with one there
// (gt, gg), g waits for the key after it (tdp K11-25, K11-20, K12-19).
func TestK11UserSetEnterAndMovement(t *testing.T) {
	ta := newTestApp(t)
	got := map[string]int{}
	do := func(name string) func(string) tea.Cmd {
		return func(string) tea.Cmd { got[name]++; return nil }
	}
	m := &Mode{Name: "Play", Exit: "pause", Keys: []ModeKey{
		{Key: "a|l", Keys: []string{"a", "l", "←", "→"}, Desc: "left, right", NoFooter: true, Do: do("move")},
		{Key: "Enter", Desc: "drop", Label: "Drop", Do: do("drop")},
		{Key: "h", Desc: "hold", Label: "Hold", Do: do("hold")},
		{Key: "g", Desc: "ghost", Label: "Ghost", Do: do("ghost")},
		{Key: "u", Desc: "spin", Label: "Spin", Do: do("spin")},
	}}
	ta.pods.EnterMode(m)
	before, _ := ta.list.Current()
	ta.press("a", "←", "l", "→", "Enter", "h", "g", "g", "u", "PgUp")
	if want := map[string]int{"move": 4, "drop": 1, "hold": 1, "ghost": 2, "spin": 1}; !maps.Equal(got, want) {
		t.Errorf("the mode's keys ran %v, want %v", got, want)
	}
	if now, _ := ta.list.Current(); now.ID != before.ID || ta.pods.InMode() != m {
		t.Errorf("something else took a key: the cursor %s then %s, in the mode %v", before.ID, now.ID, ta.pods.InMode() != nil)
	}
	if f := covCFooter(ta); !strings.HasPrefix(f, "Space:menu ?:help Enter:drop h:hold g:ghost u:spin Esc:pause") {
		t.Errorf("the footer: %q", f)
	}
	ta.press("?")
	keys := covAKeys(covARef(t, ta))
	for _, k := range []string{"a|l", "Enter", "h", "g", "u"} {
		if !keys[k] {
			t.Errorf("the mode's ? has no %s: %v", k, keys)
		}
	}
	ta.press("Esc", "Space")
	if items := menuItems(t, ta); items != "a|l,Drop,Hold,Ghost,Spin,Pause" {
		t.Errorf("the mode's menu lists %s", items)
	}
	ta.press("Esc")
	clear(got)
	m.Keys = append(m.Keys, ModeKey{Key: "gt", Desc: "turn", Do: do("turn")})
	ta.press("g", "t")
	if want := map[string]int{"turn": 1}; !maps.Equal(got, want) {
		t.Errorf("with gt in the mode, g t ran %v, want %v", got, want)
	}
	m.Keys = append(m.Keys, ModeKey{Key: "gg|G", Keys: []string{"gg", "G"}, Desc: "top, bottom", NoFooter: true, Do: do("top")})
	clear(got)
	ta.press("g", "g", "G")
	if want := map[string]int{"top": 2}; !maps.Equal(got, want) {
		t.Errorf("with gg in the mode, g g then G ran %v, want %v", got, want)
	}
}

// TestK12ModeMenuRefMovingKeys: in a mode's Space menu, a row whose key moves
// the cursor there (a mode's key on h or k: K12-19, K11-25) runs by Enter
// only; the menu's ? leaves its key out, listing the key once, for moving.
// The row keeps its marker, the key in the mode (tdp K12, K6, M3).
func TestK12ModeMenuRefMovingKeys(t *testing.T) {
	ta := newTestApp(t)
	held := 0
	ta.pods.EnterMode(&Mode{Name: "Play", Exit: "pause", Keys: []ModeKey{
		{Key: "h", Desc: "hold", Label: "Hold", Do: func(string) tea.Cmd { held++; return nil }},
		{Key: "r", Desc: "restart", Label: "Restart", Do: func(string) tea.Cmd { return nil }}}})
	ta.press("Space")
	if !strings.Contains(ta.screenText(), "[h]old") {
		t.Errorf("the row lost its marker:\n%s", ta.screenText())
	}
	ta.press("?")
	ref := covARef(t, ta)
	h, r := 0, false
	for _, row := range ref {
		f := strings.Fields(row)
		if len(f) == 0 {
			continue
		}
		if slices.Contains(strings.Split(f[0], "|"), "h") {
			h++
		}
		r = r || f[0] == "r"
	}
	if h != 1 || !r {
		t.Errorf("the menu's ? lists h %d times, r %v; want h once (moving) and r:\n%s", h, r, strings.Join(ref, "\n"))
	}
	ta.press("Esc", "h")
	if _, ok := ta.topView().(*menuView); !ok || held != 0 {
		t.Fatalf("h in the menu: %T on top, held %d", ta.topView(), held)
	}
	ta.press("Esc", "Space", "Enter") // the cursor on Hold, the first row
	if held != 1 {
		t.Errorf("Enter on the row: held %d", held)
	}
}
