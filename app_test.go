package terminu

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// testApp is a small app with two screens; the first has a list with tabs, a
// pager and a values panel.
type testApp struct {
	*App
	list   *List
	pods   *Panel
	values *Values
	ran    []string
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()
	ta := &testApp{App: New("test")}
	ta.list = &List{Columns: []Column{{Title: "Name", Flex: true}, {Title: "Age", Number: true}}, Marks: true, Filterable: true}
	var rows []Row
	for i := range 40 {
		rows = append(rows, Row{ID: fmt.Sprint(i), Cells: []string{fmt.Sprintf("pod-%02d", i), fmt.Sprint(i % 7)}})
	}
	ta.list.SetRows(rows)
	ta.pods = &Panel{Label: "Pods", Tabs: []string{"Events"}, Body: ta.list, Actions: []Action{
		{Key: "Enter", Label: "Logs", Desc: "follow its log", Scope: ItemScope, Hint: "logs", Do: ta.record("logs")},
		{Key: "m", Label: "mark", Desc: "this pod", Scope: ItemScope, Hint: "mark", Do: func() tea.Cmd { ta.list.ToggleMark(); return nil }},
		{Key: "x", Label: "Delete", Desc: "this pod", Scope: ItemScope,
			Ask: &Ask{Title: "Delete", Question: "Delete it?", Verb: "delete", Details: []string{"It is gone for good."}, Warn: []int{0}},
			Do:  ta.record("delete")},
		{Key: "D", Label: "Drag", Desc: "this pod", Scope: ItemScope, Do: func() tea.Cmd {
			ta.pods.EnterMode(&Mode{Name: "Drag", Exit: "cancel", Pause: true,
				Keys: []ModeKey{{Key: "Enter", Desc: "drop", Do: func(string) tea.Cmd { ta.pods.LeaveMode(); return nil }}}})
			return nil
		}},
		{Label: "Refresh", Desc: "the list", Scope: PanelScope, Do: ta.record("refresh")},
	}}
	log := &Panel{Label: "Log", Body: &Pager{}, Stream: true}
	log.Body.(*Pager).SetLines([]string{"one", "two", "three"})
	name := "web"
	ta.values = &Values{Fields: []*Field{
		{Label: "Name", Value: func() string { return name }, Edit: ta.record("edit name")},
		{Label: "Layout", Options: []string{"row", "column"}},
		{Label: "Remote", Value: func() string { return "x" }, Disabled: func() bool { return true }},
	}}
	settings := &Panel{Label: "Settings", Body: ta.values}
	ta.AddScreen(&Screen{Name: "Panels", Key: "P", Panels: []*Panel{ta.pods, log, settings}})
	ta.AddScreen(&Screen{Name: "Other", Key: "O", Panels: []*Panel{{Label: "Only", Body: &Pager{EmptyFact: "nothing"}}}})
	ta.Quit = &Ask{Title: "Quit", Question: "Quit test?", Verb: "quit"}
	ta.Global = []Action{{Key: "S", Label: "Start", Desc: "a job", Scope: GlobalScope, Do: ta.record("start")}}
	ta.resize(120, 40)
	return ta
}

func (ta *testApp) record(what string) func() tea.Cmd {
	return func() tea.Cmd { ta.ran = append(ta.ran, what); return nil }
}

func (ta *testApp) resize(w, h int) { ta.Update(tea.WindowSizeMsg{Width: w, Height: h}) }

// settle plays every animation to its end.
func (ta *testApp) settle() {
	for range 40 {
		if !ta.anim {
			return
		}
		ta.Update(animMsg{})
	}
}

// press sends keys by their KeyName and lets every animation finish.
func (ta *testApp) press(keys ...string) tea.Cmd {
	var last tea.Cmd
	for _, k := range keys {
		last = ta.Update2(keyMsg(k))
		ta.settle()
	}
	return last
}

// Update2 is Update returning only the command.
func (ta *testApp) Update2(msg tea.Msg) tea.Cmd {
	_, cmd := ta.Update(msg)
	return cmd
}

func keyMsg(name string) tea.KeyMsg {
	special := map[string]tea.KeyType{
		"Enter": tea.KeyEnter, "Esc": tea.KeyEsc, "Tab": tea.KeyTab, "Shift-Tab": tea.KeyShiftTab,
		"Space": tea.KeySpace, "Backspace": tea.KeyBackspace, "Ctrl-C": tea.KeyCtrlC,
		"↑": tea.KeyUp, "↓": tea.KeyDown, "←": tea.KeyLeft, "→": tea.KeyRight,
	}
	if t, ok := special[name]; ok {
		if t == tea.KeySpace {
			return tea.KeyMsg{Type: t, Runes: []rune{' '}}
		}
		return tea.KeyMsg{Type: t}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(name)}
}

func (ta *testApp) topView() popupView {
	if l := ta.top(); l != nil {
		return l.view
	}
	return nil
}

func (ta *testApp) screenText() string { return StripStyles(strings.Join(ta.Frame(), "\n")) }

func TestK5SpaceToggles(t *testing.T) {
	ta := newTestApp(t)
	ta.press("Space")
	m, ok := ta.topView().(*menuView)
	if !ok || !m.space {
		t.Fatalf("Space did not open the Space menu: %T", ta.topView())
	}
	ta.press("Space")
	if ta.top() != nil {
		t.Error("Space did not close the Space menu")
	}
}

func TestM2SectionsOrder(t *testing.T) {
	ta := newTestApp(t)
	ta.press("Space")
	m := ta.topView().(*menuView)
	var order []string
	for _, r := range m.rows {
		switch r.kind {
		case menuTitle:
			order = append(order, r.label)
		case menuItem:
			if r.action.Label == "Global operation" {
				order = append(order, "global")
			}
		}
	}
	if got := strings.Join(order, ","); got != "item operation,panel operation,global" {
		t.Errorf("sections = %s", got)
	}
}

func TestK4EscOneLayer(t *testing.T) {
	ta := newTestApp(t)
	ta.press("Space")
	ta.press("G", "Enter") // the Global operation row
	if len(ta.stack) != 2 {
		t.Fatalf("want two popups, have %d", len(ta.stack))
	}
	ta.press("Esc")
	if len(ta.stack) != 1 {
		t.Fatalf("Esc closed %d layers", 2-len(ta.stack))
	}
	ta.press("Esc")
	if len(ta.stack) != 0 {
		t.Fatal("second Esc did not close the Space menu")
	}
	ta.press("Esc", "Esc")
	if cmd := ta.press("Esc"); cmd != nil {
		if _, quit := cmd().(tea.QuitMsg); quit {
			t.Error("Esc quit the app")
		}
	}
}

func TestK1DigitsJumpPanels(t *testing.T) {
	ta := newTestApp(t)
	ta.press("3")
	if ta.Screen().focus != 2 {
		t.Errorf("3 focused panel %d", ta.Screen().focus+1)
	}
	ta.press("9")
	if ta.Screen().focus != 2 {
		t.Error("a digit with no panel moved the focus")
	}
	ta.press("Tab")
	if ta.Screen().focus != 0 {
		t.Error("Tab did not wrap to the first panel")
	}
}

func TestK1HotkeyCannotTakeCoreKey(t *testing.T) {
	for _, key := range []string{"q", "?", "Space", "Tab", "Esc", "3", "j", "g", "G", "gg"} {
		if checkPanelKey(key) == nil {
			t.Errorf("%q was accepted as a panel hotkey", key)
		}
	}
	for _, key := range []string{"m", "D", "Alt-t", "go", "/", "Enter"} {
		if err := checkPanelKey(key); err != nil {
			t.Errorf("%q refused: %v", key, err)
		}
	}
}

func TestK12StepWraps(t *testing.T) {
	ta := newTestApp(t)
	ta.press("k")
	if r, _ := ta.list.Current(); r.ID != "39" {
		t.Errorf("k on the first row went to %s, want the last", r.ID)
	}
	ta.press("j")
	if r, _ := ta.list.Current(); r.ID != "0" {
		t.Errorf("j on the last row went to %s, want the first", r.ID)
	}
}

func TestK12HalfPageStops(t *testing.T) {
	ta := newTestApp(t)
	ta.Frame()
	ta.press("G", "d")
	if r, _ := ta.list.Current(); r.ID != "39" {
		t.Errorf("d at the end moved to %s", r.ID)
	}
	ta.press("gg", "u")
	if r, _ := ta.list.Current(); r.ID != "0" {
		t.Errorf("u at the top moved to %s", r.ID)
	}
}

func TestK12HLSwitchTabs(t *testing.T) {
	ta := newTestApp(t)
	ta.press("l")
	if ta.pods.Tab != 1 {
		t.Errorf("l: tab %d", ta.pods.Tab)
	}
	ta.press("l")
	if ta.pods.Tab != 0 {
		t.Errorf("l on the last tab did not wrap: %d", ta.pods.Tab)
	}
}

func TestF6ConfirmEveryTime(t *testing.T) {
	ta := newTestApp(t)
	ta.press("x")
	c, ok := ta.topView().(*confirmView)
	if !ok {
		t.Fatalf("x did not ask: %T", ta.topView())
	}
	if len(ta.ran) != 0 {
		t.Fatal("ran before the answer")
	}
	ta.press("Space")
	if ta.topView() != c {
		t.Error("Space did something on a confirm")
	}
	ta.press("Esc")
	if len(ta.ran) != 0 || ta.top() != nil {
		t.Fatal("Esc did not cancel")
	}
	ta.press("x", "Enter")
	if strings.Join(ta.ran, ",") != "delete" || ta.top() != nil {
		t.Errorf("Enter: ran %v, stack %d", ta.ran, len(ta.stack))
	}
}

func TestK9QuitSameFlow(t *testing.T) {
	ta := newTestApp(t)
	ta.press("q")
	if _, ok := ta.topView().(*confirmView); !ok {
		t.Fatal("q did not start the quit flow")
	}
	cmd := ta.Update2(keyMsg("Ctrl-C"))
	if cmd == nil {
		t.Fatal("the second Ctrl-C did nothing")
	}
	if _, quit := cmd().(tea.QuitMsg); !quit {
		t.Error("the second Ctrl-C did not quit")
	}
}

func TestK11HotkeysSilentInMode(t *testing.T) {
	ta := newTestApp(t)
	ta.press("D")
	if ta.pods.InMode() == nil {
		t.Fatal("D did not enter the mode")
	}
	ta.press("S", "x", "Space")
	if len(ta.ran) != 0 || ta.top() != nil {
		t.Errorf("hotkeys worked in the mode: ran %v, popup %T", ta.ran, ta.topView())
	}
	ta.press("2")
	if ta.Screen().focus != 0 || ta.toast == nil {
		t.Error("a digit in a paused mode should answer with a toast and stay")
	}
	ta.press("Esc") // the toast first (K11)
	if ta.pods.InMode() == nil {
		t.Error("the first Esc left the mode while a toast was up")
	}
	ta.press("Esc")
	if ta.pods.InMode() != nil {
		t.Error("Esc did not leave the mode")
	}
}

func TestM4PanelKeyWins(t *testing.T) {
	ta := newTestApp(t)
	ta.press("D")
	if ta.cur != 0 || ta.pods.InMode() == nil {
		t.Error("the panel's D should win over a global D")
	}
}

func TestM4GlobalHotkeyOnPanel(t *testing.T) {
	ta := newTestApp(t)
	ta.press("S")
	if strings.Join(ta.ran, ",") != "start" {
		t.Errorf("S on a panel: %v", ta.ran)
	}
	ta.press("O")
	if ta.cur != 1 {
		t.Error("a screen's hotkey did not switch screens")
	}
}

func TestPanelFilter(t *testing.T) {
	ta := newTestApp(t)
	ta.press("/", "1", "5")
	if !ta.list.Typing() {
		t.Fatal("/ did not open the typing row")
	}
	if got := len(ta.list.view); got != 1 {
		t.Errorf("15 matched %d rows", got)
	}
	if f := ta.footerHints(); f[0].Key != "Enter/Tab" {
		t.Errorf("footer while typing: %v", f)
	}
	ta.press("Enter")
	if ta.list.Typing() {
		t.Error("Enter stayed on the typing row")
	}
	ta.press("Esc")
	if ta.list.Filtering() {
		t.Error("Esc on the list did not clear the filter")
	}
}

func TestToastPosition(t *testing.T) {
	ta := newTestApp(t)
	ta.Update(nil)
	ta.Toast(Info, "Copied 3 paths")
	ta.settle()
	if got, want := ta.toast.y+ta.toast.h-1, ta.h-3; got != want {
		t.Errorf("toast's bottom border on row %d, want %d", got, want)
	}
}

func TestF2AnimationFrames(t *testing.T) {
	ta := newTestApp(t)
	ta.press("Space")
	ta.press("Esc")
	ta.Update(keyMsg("Space"))
	l := ta.top()
	frames := 1
	for ta.anim {
		ta.Update(animMsg{})
		frames++
	}
	if frames != openFrames || l.frame != openFrames {
		t.Errorf("opened in %d frames, want %d", frames, openFrames)
	}
	ta.Update(keyMsg("j")) // keys go to the popup once it is open
}

func TestF2NoKeysWhileOpening(t *testing.T) {
	ta := newTestApp(t)
	ta.Update(keyMsg("Space"))
	ta.Update(keyMsg("Esc"))
	if l := ta.top(); l == nil || l.closing {
		t.Error("a popup still opening took Esc")
	}
}

// TestL4EveryRowFullWidth draws every screen and popup at many sizes, with an
// icon one cell and two, and checks that every row is exactly the terminal's
// width and the screen exactly its height (Rules L4, E5).
func TestL4EveryRowFullWidth(t *testing.T) {
	sizes := [][2]int{{80, 40}, {120, 40}, {100, 30}, {60, 20}, {30, 12}, {200, 50}}
	opens := map[string][]string{
		"none":     nil,
		"space":    {"Space"},
		"global":   {"Space", "G", "Enter"},
		"keys":     {"?"},
		"confirm":  {"x"},
		"quit":     {"q"},
		"filter":   {"/", "p"},
		"mode":     {"D"},
		"panel 3":  {"3"},
		"screen 2": {"O"},
	}
	calls := inputOpeners(t)
	for name := range calls {
		opens[name] = nil
	}
	for _, widths := range [][2]int{{1, 1}, {2, 1}, {2, 2}} {
		for name, keys := range opens {
			for _, sz := range sizes {
				withWidths(t, widths[0], widths[1])
				ta := newTestApp(t)
				ta.resize(sz[0], sz[1])
				ta.press(keys...)
				if open := calls[name]; open != nil {
					open(ta)
					ta.settle()
				}
				ta.Toast(Warning, "a toast that is long enough to wrap on a narrow screen and more")
				ta.settle()
				lines := ta.Frame()
				label := fmt.Sprintf("icon %d/%d %s %dx%d", widths[0], widths[1], name, sz[0], sz[1])
				if len(lines) != sz[1] {
					t.Errorf("%s: %d lines", label, len(lines))
				}
				for i, l := range lines {
					if w := Width(l); w != sz[0] {
						t.Errorf("%s: line %d is %d wide", label, i, w)
					}
				}
			}
		}
	}
}

func TestPanelTabChain(t *testing.T) {
	p := &Panel{Label: "Pods", Tabs: []string{"Events", "Yaml"}}
	got := func() []Capsule { return p.capsules(2) }
	if c := got(); len(c) != 3 || c[0] != (Capsule{Text: "[2] PODS", Selected: true}) || c[1].Selected || c[2].Selected {
		t.Errorf("on the first tab [N] and it are one filled part: %+v", c)
	}
	p.Tab = 1
	if c := got(); len(c) != 4 || c[0] != (Capsule{Text: "[2]", Selected: true}) || c[1] != (Capsule{Text: "PODS"}) ||
		c[2] != (Capsule{Text: "EVENTS", Selected: true}) || c[3].Selected {
		t.Errorf("on another tab [N] is a part of its own and only that tab is filled: %+v", c)
	}
	if c := (&Panel{Label: "Log"}).capsules(3); len(c) != 1 || c[0].Text != "[3] LOG" {
		t.Errorf("without tabs: %+v", c)
	}
}
