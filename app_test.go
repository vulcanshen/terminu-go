package terminu

import (
	"fmt"
	"slices"
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

// anySize lets a test app draw itself on any terminal: the tests of other
// things use small ones (tdp L1: the app may lower its least size).
func anySize(a *App) *App {
	a.MinWidth, a.MinHeight = 1, 1
	return a
}

func newTestApp(t *testing.T) *testApp {
	t.Helper()
	ta := &testApp{App: anySize(New("test"))}
	ta.list = &List{Columns: []Column{{Title: "Name"}, {Title: "Age", Number: true, Drop: 1}}, Marks: true, Filterable: true}
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
	defer ta.endLabels() // the mode names all open or gone
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
		"Home": tea.KeyHome, "End": tea.KeyEnd, "PgUp": tea.KeyPgUp, "PgDn": tea.KeyPgDown, "Delete": tea.KeyDelete,
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
	ta.press("S", "x")
	if len(ta.ran) != 0 || ta.top() != nil {
		t.Errorf("hotkeys worked in the mode: ran %v, popup %T", ta.ran, ta.topView())
	}
	ta.press("2")
	if ta.Screen().focus != 0 || ta.toast == nil {
		t.Error("a digit in a paused mode should answer with a toast and stay")
	}
	ta.press("Esc") // the toast first (tdp K11)
	if ta.pods.InMode() == nil {
		t.Error("the first Esc left the mode while a toast was up")
	}
	ta.press("Esc")
	if ta.pods.InMode() != nil {
		t.Error("Esc did not leave the mode")
	}
}

func TestM5KeysJoinedWithBar(t *testing.T) {
	// Keys doing one thing are joined with |, drawn in the words colour,
	// Overlay0, not in the key colour (tdp M5).
	bar := Fg(Overlay0).Render("|")
	if h := renderHints([]KeyHint{{"Enter|Tab", "list"}}, 40, Fg(Blue), Overlay0); h != Fg(Blue).Render("Enter")+bar+Fg(Blue).Render("Tab")+Fg(Overlay0).Render(":list") {
		t.Errorf("hint: %q", h)
	}
	ta := newTestApp(t)
	ta.press("?")
	ta.settle()
	n := ta.topView().(*noteView)
	found := false
	for _, l := range n.lines {
		if strings.HasPrefix(StripStyles(l), " j|k") {
			found = strings.Contains(l, Fg(Blue).Render("j")+bar+Fg(Blue).Render("k"))
		}
	}
	if !found {
		t.Error("the key reference does not draw j|k with the bar in Overlay0")
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
	if f := ta.footerHints(); f[0].Key != "Enter|Tab" {
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
// width and the screen exactly its height (tdp L4, E5). Some are shorter
// than a popup's frame: a popup is cut off there, never a panic.
func TestL4EveryRowFullWidth(t *testing.T) {
	sizes := [][2]int{{80, 40}, {120, 40}, {100, 30}, {60, 20}, {30, 12}, {200, 50},
		{20, 6}, {40, 5}, {40, 4}, {20, 3}, {10, 2}, {1, 1}}
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
	for name, open := range dialogOpeners() {
		calls[name] = open
	}
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

func TestM6OtherSurfaceSection(t *testing.T) {
	ta := newTestApp(t)
	ta.pods.OtherKeys = []KeySection{{Title: "terminal", Rows: []KeyRow{{Key: "Alt-Esc", Desc: "leave the shell"}}}}
	ta.press("?")
	text := ta.screenText()
	app, other := strings.Index(text, "APP-WIDE"), strings.Index(text, "TERMINAL")
	if other < 0 || !strings.Contains(text, "Alt-Esc") || !strings.Contains(text, "leave the shell") {
		t.Fatalf("the section about another surface is missing:\n%s", text)
	}
	if app > other {
		t.Error("the section about another surface comes after the panel's own")
	}
}

func TestK1DigitGlobalNotInPanelKeyRef(t *testing.T) {
	ta := newTestApp(t)
	ta.Global = append(ta.Global, Action{Key: "1", Label: "About", Desc: "the app", Scope: GlobalScope, Do: ta.record("about")})
	ta.press("?")
	if strings.Contains(ta.screenText(), "About") {
		t.Error("a digit global hotkey is listed on a panel, where 1 jumps to a panel")
	}
	ta.press("Esc", "Space", "G", "Enter", "1")
	if len(ta.ran) == 0 || ta.ran[len(ta.ran)-1] != "about" {
		t.Errorf("1 in the global operation popup: ran %v", ta.ran)
	}
}

func TestM6HintListsDisabled(t *testing.T) {
	ta := newTestApp(t)
	ta.pods.Tabs = nil // room in the hint for m:mark, no h|l:tab
	off := false
	ta.pods.Actions[1].Disabled = func() bool { return off }
	hint := func() string { return ta.rowWith("Enter:logs") }
	off = true
	if row := hint(); !strings.Contains(StripStyles(row), "m:mark") || !strings.Contains(row, "38;2;88;91;112") {
		t.Errorf("a disabled action stays in the hint, in Surface2: %q", row)
	}
	ta.pods.HintPressableOnly = true
	if strings.Contains(StripStyles(hint()), "m:mark") {
		t.Error("HintPressableOnly lists a disabled action")
	}
	off = false
	if !strings.Contains(StripStyles(hint()), "m:mark") {
		t.Error("HintPressableOnly left out an action that can run")
	}
}

// TestE3TabsDrawnAsSpaces: a tab in the text of a panel or a popup is drawn as
// input_tab_width spaces, as in a viewer, and counted so: a row holding one
// is still exactly as wide as the terminal (tdp E3, L4).
// TestInitSendsLibTimers: the timers of a toast Start opens go out with
// Init, whatever Start returns, not at the first message.
func TestInitSendsLibTimers(t *testing.T) {
	ta := newTestApp(t)
	ta.Start = func() tea.Cmd { ta.Toast(Info, "Started"); return nil }
	if ta.Init() == nil || len(ta.timers) != 0 {
		t.Errorf("Init kept %d timers", len(ta.timers))
	}
}

func TestE3TabsDrawnAsSpaces(t *testing.T) {
	if got := Width("a\tb"); got != 2+tabCells {
		t.Errorf("a tab is %d cells", got-2)
	}
	for name, got := range map[string]string{
		"Fit": Fit("a\tb", 8), "Pad": Pad("a\tb", 8), "PadLeft": PadLeft("a\tb", 8), "Center": Center("a\tb", 8),
		"Cut": Cut("a\tb", 0, 8), "Truncate": Truncate("a\tb", 8),
	} {
		if strings.Contains(got, "\t") || Width(got) > 8 || !strings.Contains(got, "a    b") {
			t.Errorf("%s = %q", name, got)
		}
	}
	ta := newTestApp(t)
	ta.Screen().Panels[1].Body.(*Pager).SetLines([]string{"\tindented", "a\tb\tc", Fg(Red).Render("x\ty")})
	ta.list.SetRows([]Row{{ID: "1", Cells: []string{"a\tb", "1\t2"}}})
	frames := map[string]func(){
		"panels": func() {},
		"select": func() { ta.Select(Select{Title: "Pick", Options: []string{"one\ttwo", "three"}, Selected: -1}) },
		"menu":   func() { ta.Menu("Run", MenuSection{Actions: []Action{{Label: "Tab\tbed", Desc: "a\tb"}}}) },
	}
	for name, open := range frames {
		ta.covBCloseAll()
		open()
		ta.settle()
		for i, l := range ta.Frame() {
			if Width(l) != 120 || strings.Contains(l, "\t") {
				t.Errorf("%s: row %d is %d wide or holds a tab: %q", name, i, Width(l), l)
			}
		}
	}
}

// TestValueControlChars: a value on a form or a panel writes every control
// character as its red name, a tab and ESC too, a colour code included, as on
// a typing row; colours of the app's own come as spans (components, control
// characters; tdp input, tdp E3).
func TestValueControlChars(t *testing.T) {
	v := "a\tb\nc\x1b[31md"
	got := valueText(v, Fg(Text))
	if StripStyles(got) != `a\tb\nc\x1b[31md` {
		t.Errorf("valueText shows %q", StripStyles(got))
	}
	for _, name := range []string{`\t`, `\n`, `\x1b`} {
		if !strings.Contains(got, nameSGR+name) {
			t.Errorf("%s is not in Red: %q", name, got)
		}
	}
	if strings.Contains(got, "\x1b[31m") {
		t.Error("a colour code in a value reached the terminal")
	}
	f := &Field{Label: "Background", Value: func() string { return "#2A2A3C" }, Styled: func() []Span { return ColorSpans(Hex("#2A2A3C")) }}
	vals := &Values{Fields: []*Field{f}}
	if row := strings.Join(vals.Draw(40, 1, false), ""); !strings.Contains(StripStyles(row), "BACKGROUND  ███ #2A2A3C") ||
		!strings.Contains(row, "\x1b["+sgrRGB(true, Hex("#2A2A3C").RGB)) {
		t.Errorf("a styled value: %q", row)
	}
	ta := newTestApp(t)
	vals = &Values{Fields: []*Field{{Label: "Command", Value: func() string { return "make\tcheck" }}}}
	ta.Screen().Panels[1].Body = vals
	ta.press("2")
	if text := ta.screenText(); !strings.Contains(text, `make\tcheck`) {
		t.Errorf("a panel value: %q", text)
	}
	ta.Form(&Form{Title: "Run", Fields: []*FormField{{Label: "Command", Value: func() string { return "go\ttest" }}}, Button: "Run"})
	ta.settle()
	if text := ta.screenText(); !strings.Contains(text, `go\ttest`) {
		t.Errorf("a form value: %q", text)
	}
	// Text that is only read keeps its tab as spaces (tdp E3).
	if got := Pad("make\tcheck", 20); !strings.Contains(got, "make    check") {
		t.Errorf("text on the screen: %q", got)
	}
}

// TestSpinning: an app's own loading icon in a row turns with the lib's
// (LoadingFrame, App.Spinning).
func TestSpinning(t *testing.T) {
	ta := newTestApp(t)
	if !slices.Contains(loadingFrames[:], LoadingFrame()) {
		t.Errorf("not a loading frame: %q", LoadingFrame())
	}
	busy := true
	ta.Spinning = func() bool { return busy }
	ta.spin = false
	if _, cmd := ta.Update(struct{}{}); cmd == nil || !ta.spin {
		t.Error("the app's spinner keeps the ticks coming")
	}
	busy = false
	ta.spin = false
	ta.Update(struct{}{})
	if ta.spin {
		t.Error("nothing turning: no ticks")
	}
}

// TestAppMovesFocus: the app moves the focus or the screen where the user
// expects it (tdp K3); the focus stays on its panel when panels come and
// go, and a screen whose panels got fewer does not panic (sshu SS3).
func TestAppMovesFocus(t *testing.T) {
	ta := newTestApp(t)
	s := ta.Screen()
	logs, settings := s.Panels[1], s.Panels[2]
	ta.press("3")
	if s.Focused() != settings {
		t.Fatal("3 focuses [3]")
	}
	s.Panels = s.Panels[:2]
	if p := s.Focused(); p != logs {
		t.Errorf("[3] gone: the focus is on the panel now last, not out of range: %v", p)
	}
	ta.Frame()
	extra := &Panel{Label: "Extra", Body: &Pager{}}
	s.Panels = []*Panel{extra, s.Panels[0], s.Panels[1]}
	if s.Focused() != logs || !strings.Contains(ta.screenText(), "[3] LOG") {
		t.Errorf("a panel added in front: the focus stays on LOG, now [3]")
	}
	other := ta.screens[1]
	ta.Focus(other.Panels[0])
	if ta.Screen() != other || other.Focused() != other.Panels[0] {
		t.Error("Focus switches to the panel's screen")
	}
	ta.ShowScreen(s)
	if ta.Screen() != s || s.Focused() != logs {
		t.Error("ShowScreen goes back, the focus where it was")
	}
	ta.Focus(&Panel{Label: "Nowhere"})
	if ta.Screen() != s || s.Focused() != logs {
		t.Error("a panel on no screen changes nothing")
	}
}

// TestQuitFlowOfTheApp: the quit flow can be the app's own action, a menu of
// ways to leave; a second Ctrl-C during it still leaves at once (tdp K9).
func TestQuitFlowOfTheApp(t *testing.T) {
	ta := newTestApp(t)
	ta.QuitFlow = func() tea.Cmd {
		return ta.Menu("Quit", MenuSection{Actions: []Action{
			{Key: "s", Label: "Stay in the last directory", Desc: "the shell goes there", Do: func() tea.Cmd { return tea.Quit }},
		}})
	}
	ta.press("q")
	ta.settle()
	m, ok := ta.topView().(*menuView)
	if !ok || !ta.quitting {
		t.Fatalf("q runs the app's flow: %T, quitting %v", ta.topView(), ta.quitting)
	}
	ta.press("q")
	ta.settle()
	if len(ta.stack) != 1 {
		t.Errorf("q again during the flow starts no second one: %d layers", len(ta.stack))
	}
	ta.press("Esc")
	ta.settle()
	if ta.quitting || ta.top() != nil {
		t.Error("the flow ends with its popup")
	}
	ta.press("Ctrl-C")
	ta.settle()
	if ta.topView() == m || !ta.quitting {
		t.Error("Ctrl-C runs it too, a new one")
	}
	if cmd := ta.handleKey(keyMsg("Ctrl-C")); cmd == nil || fmt.Sprint(cmd()) != fmt.Sprint(tea.Quit()) {
		t.Error("a second Ctrl-C leaves at once")
	}
}

// TestL1LeastSize: under the app's least size, 80 × 40 unless it says less,
// only its name and what it needs are drawn, in the middle; no key acts but
// Ctrl-C twice; what it was showing comes back as it grows (tdp L1,
// tdp screen; tdp E5 down to 1×1).
func TestL1LeastSize(t *testing.T) {
	ta := newTestApp(t)
	ta.MinWidth, ta.MinHeight = 0, 0
	ta.resize(80, 40)
	if strings.Contains(ta.screenText(), "needs a terminal") {
		t.Error("80 × 40 is enough by default")
	}
	ta.press("Space")
	ta.settle()
	menu := ta.topView()
	ta.resize(79, 40)
	frame := ta.Frame()
	plain := StripStyles(strings.Join(frame, "\n"))
	if !strings.Contains(plain, "needs a terminal of at least 80×40") || strings.Contains(plain, "PODS") {
		t.Fatalf("under 80 wide, the name and the size only:\n%s", plain)
	}
	for i, l := range frame {
		if Width(l) != 79 {
			t.Errorf("row %d is %d wide", i, Width(l))
		}
		if p := StripStyles(l); strings.TrimSpace(p) == "test" {
			if i != 19 || !strings.Contains(l, Fg(Blue).Bolded().Render("test")) {
				t.Errorf("the name in the middle, Blue bold: row %d %q", i, l)
			}
			if left, right := len(p)-len(strings.TrimLeft(p, " ")), len(p)-len(strings.TrimRight(p, " ")); left-right > 1 || right-left > 1 {
				t.Errorf("not centred: %q", p)
			}
			if !strings.Contains(frame[i+1], Fg(Overlay0).Render("needs a terminal of at least 80×40")) {
				t.Errorf("the size under it in Overlay0: %q", frame[i+1])
			}
		}
	}
	// No key acts: q, Ctrl-C once, the menu keys; Ctrl-C twice leaves.
	for _, k := range []string{"q", "Esc", "j", "Ctrl-C", "x", "Ctrl-C"} {
		if cmd := ta.handleKey(keyMsg(k)); cmd != nil {
			t.Errorf("%s did something", k)
		}
	}
	if ta.quitting || ta.topView() != menu {
		t.Error("the keys reached the app")
	}
	ta.handleKey(keyMsg("Ctrl-C"))
	if cmd := ta.handleKey(keyMsg("Ctrl-C")); cmd == nil || fmt.Sprint(cmd()) != fmt.Sprint(tea.Quit()) {
		t.Error("Ctrl-C twice leaves at once")
	}
	// Grown again, all as it was.
	ta.resize(80, 40)
	if ta.topView() != menu || !strings.Contains(ta.screenText(), "PODS") {
		t.Error("what it showed comes back")
	}
	// Small sizes: the hint goes first, the name stays; down to 1×1.
	for _, c := range []struct {
		w, h int
		want string
	}{{80, 5, "needs a terminal"}, {80, 1, "test"}, {20, 10, "test"}, {3, 3, "tes"}, {1, 1, "t"}} {
		ta.resize(c.w, c.h)
		frame := ta.Frame()
		text := StripStyles(strings.Join(frame, "\n"))
		if len(frame) != c.h || !strings.Contains(text, c.want) {
			t.Errorf("%d×%d: %q", c.w, c.h, text)
		}
		if (c.h == 1 || c.w < 34) && strings.Contains(text, "needs") {
			t.Errorf("%d×%d: no room for the hint: %q", c.w, c.h, text)
		}
	}
	// The app may say less, never more.
	ta.MinWidth, ta.MinHeight = 60, 20
	ta.resize(59, 30)
	if !strings.Contains(ta.screenText(), "needs a terminal of at least 60×20") {
		t.Errorf("its own least size: %q", ta.screenText())
	}
	ta.resize(60, 20)
	if strings.Contains(ta.screenText(), "needs a terminal") {
		t.Error("60 × 20 is enough for it")
	}
	ta.MinWidth, ta.MinHeight = 100, 50
	ta.resize(80, 40)
	if strings.Contains(ta.screenText(), "needs a terminal") {
		t.Error("never more than 80 × 40")
	}
}

// TestOnResize: the app hears of a new terminal size after the lib has laid
// the screen out again (tdp L1; kbu KB20).
func TestOnResize(t *testing.T) {
	ta := newTestApp(t)
	var got [][2]int
	ta.OnResize = func(w, h int) tea.Cmd {
		if ta.w != w || ta.h != h {
			t.Errorf("the lib had not taken the size: %dx%d, heard %dx%d", ta.w, ta.h, w, h)
		}
		got = append(got, [2]int{w, h})
		return nil
	}
	ta.resize(100, 30)
	ta.resize(60, 20)
	if len(got) != 2 || got[0] != [2]int{100, 30} || got[1] != [2]int{60, 20} {
		t.Errorf("heard %v", got)
	}
}

// TestK9PopupAsksFirst: with a popup open the first q or Ctrl-C never leaves
// at once: an app with no confirm asks Quit test?, Esc back to the popup; a
// QuitFlow that would leave there asks first, Enter running it; one that asks
// something of its own asks only that; on a panel the flow is as it is
// (K9-6).
func TestK9PopupAsksFirst(t *testing.T) {
	quits := func(cmd tea.Cmd) bool {
		if cmd == nil {
			return false
		}
		_, ok := cmd().(tea.QuitMsg)
		return ok
	}
	ta := newTestApp(t)
	ta.Quit = nil
	if !quits(ta.press("q")) {
		t.Fatal("on a panel an app with no confirm leaves at once")
	}
	ta.press("Space")
	menu := ta.top()
	if quits(ta.press("q")) || !strings.Contains(ta.screenText(), "Quit test?") {
		t.Fatalf("q over a popup left or did not ask:\n%s", ta.screenText())
	}
	ta.press("Esc")
	if ta.top() != menu {
		t.Error("Esc did not go back to the popup")
	}
	ta.press("Ctrl-C")
	if !quits(ta.press("Ctrl-C")) {
		t.Error("the second Ctrl-C does not leave")
	}

	tb := newTestApp(t)
	tb.Quit = nil
	tb.QuitFlow = func() tea.Cmd { return tea.Quit } // nothing to ask: it would leave at once
	tb.press("Space")
	if quits(tb.press("q")) || !strings.Contains(tb.screenText(), "Quit test?") {
		t.Fatalf("a flow leaving at once over a popup did not ask:\n%s", tb.screenText())
	}
	if !quits(tb.press("Enter")) {
		t.Error("Enter does not run the flow")
	}

	tc := newTestApp(t)
	tc.Quit = nil
	tc.QuitFlow = func() tea.Cmd {
		return tc.Menu("Leave", MenuSection{Actions: []Action{{Label: "Here", Do: func() tea.Cmd { return tea.Quit }}}})
	}
	tc.press("Space", "q")
	if len(tc.stack) != 2 || strings.Contains(tc.screenText(), "Quit test?") {
		t.Errorf("a flow asking of its own was asked twice: %d layers\n%s", len(tc.stack), tc.screenText())
	}
}
