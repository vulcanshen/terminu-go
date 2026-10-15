package terminu

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

// Tests for docs/coverage.md: Principle, Rules K, Rules E.

// covAQuits reports whether cmd is tea.Quit itself: leaving the app.
func covAQuits(cmd tea.Cmd) bool {
	return cmd != nil && reflect.ValueOf(cmd).Pointer() == reflect.ValueOf(tea.Quit).Pointer()
}

// covAQuitConfirm reports whether the quit confirm is the top popup.
func covAQuitConfirm(ta *testApp) bool {
	l := ta.top()
	return l != nil && ta.isQuitConfirm(l)
}

// covARegister registers one panel action with key and reports the panic,
// if AddScreen refused it.
func covARegister(key string, scope Scope) (refused any) {
	defer func() { refused = recover() }()
	New("test").AddScreen(&Screen{Name: "S", Panels: []*Panel{{Label: "P", Body: &Pager{},
		Actions: []Action{{Key: key, Label: "Act", Scope: scope}}}}})
	return nil
}

// covARef is the key reference on top, row by row, with single spaces.
func covARef(t *testing.T, ta *testApp) []string {
	t.Helper()
	n, ok := ta.topView().(*noteView)
	if !ok || !n.helpCloses {
		t.Fatalf("no key reference on top: %T", ta.topView())
	}
	var rows []string
	for _, l := range n.lines {
		rows = append(rows, strings.Join(strings.Fields(StripStyles(l)), " "))
	}
	return rows
}

// covAKeys are the keys a key reference lists: the first word of each row.
func covAKeys(rows []string) map[string]bool {
	keys := map[string]bool{}
	for _, r := range rows {
		if f := strings.Fields(r); len(f) > 0 {
			keys[f[0]] = true
		}
	}
	return keys
}

// covASurface is one place the focus can be, and how to get there.
type covASurface struct {
	name string
	open func(ta *testApp)
}

// covAPopups opens every kind of popup that is not the input state.
func covAPopups() []covASurface {
	return []covASurface{
		{"Space menu", func(ta *testApp) { ta.press("Space") }},
		{"global operation popup", func(ta *testApp) { ta.press("Space", "G", "Enter") }},
		{"menu", func(ta *testApp) {
			ta.Menu("Sort by", MenuSection{Actions: []Action{{Label: "Name"}, {Label: "Age"}}})
		}},
		{"confirm", func(ta *testApp) { ta.Confirm(Ask{Title: "Delete", Question: "Delete it?", Verb: "delete"}, nil) }},
		{"key reference", func(ta *testApp) { ta.press("?") }},
		{"viewer", func(ta *testApp) { ta.Viewer(testViewer()) }},
		{"error popup", func(ta *testApp) { ta.Error("Save failed", "The disk is full.") }},
		{"select", func(ta *testApp) { ta.Select(Select{Title: "Profile", Options: []string{"clock", "matrix", "pipes"}}) }},
		{"checkbox popup", func(ta *testApp) {
			ta.Checks(Checks{Title: "Namespaces", Options: []string{"default", "kube-system"}})
		}},
		{"finder's list", func(ta *testApp) { ta.Find(testFinder()); ta.settle(); ta.press("Tab") }},
		{"form", func(ta *testApp) { ta.Form(testForm(nil)); ta.settle(); ta.press("Tab") }},
		{"date picker", func(ta *testApp) { ta.PickDate(DatePicker{Title: "Due date"}) }},
		{"color picker", func(ta *testApp) { ta.PickColor(ColorPicker{Title: "Background", Value: Hex("#2A2A3C")}) }},
		{"reorder", func(ta *testApp) { ta.Reorder(Reorder{Title: "Order", Options: []string{"Usage", "Cost"}}) }},
		{"textarea, moving", func(ta *testApp) {
			ta.Textarea(Textarea{Title: "Comment", Old: "one"})
			ta.settle()
			ta.press("Esc")
		}},
	}
}

// covAOpen is a fresh test app with s open.
func covAOpen(t *testing.T, s covASurface) *testApp {
	t.Helper()
	ta := newTestApp(t)
	s.open(ta)
	ta.settle()
	return ta
}

// --- Principle ---------------------------------------------------------------

func TestP3ScopeRequired(t *testing.T) {
	ta := newTestApp(t)
	ta.pods.Actions = append(ta.pods.Actions, Action{Label: "Delete marked", Desc: "every marked pod", Scope: PanelScope})
	settings := ta.Screen().Panels[2]
	settings.Actions = append(settings.Actions, Action{Key: "R", Label: "Reset", Desc: "every setting", Scope: PanelScope})
	sections := func() map[string][]string {
		out := map[string][]string{}
		title := ""
		for _, r := range ta.topView().(*menuView).rows {
			switch r.kind {
			case menuTitle:
				title = r.label
			case menuSeparator:
				title = "untitled"
			case menuItem:
				out[title] = append(out[title], r.action.Label)
			}
		}
		return out
	}
	ta.press("Space")
	got := sections()
	want := map[string][]string{
		"item operation":  {"Logs", "mark", "Delete", "Drag"},
		"panel operation": {"Refresh", "Delete marked", "Filter"}, // a batch of marks is the panel's
		"untitled":        {"Global operation"},
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("[1] PODS Space menu by section:\n got %v\nwant %v", got, want)
	}
	ta.press("Esc", "3", "Space")
	if got := sections(); !slices.Equal(got["panel operation"], []string{"Reset"}) {
		t.Errorf("an action about [3] only is in [3]'s menu: %v", got)
	}
	ta.press("G", "Enter")
	var global []string
	for _, r := range ta.topView().(*menuView).rows {
		if r.kind == menuItem {
			global = append(global, r.action.Label)
		}
	}
	if want := []string{"Panels", "Other", "Start", "Quit"}; !slices.Equal(global, want) {
		t.Errorf("the global operation popup holds switching screens, the app's global ones and quitting: %v, want %v", global, want)
	}
}

// --- K1 ----------------------------------------------------------------------

func TestK1CoreKeysEverySurface(t *testing.T) {
	// The panels: Space, ?, Esc, Tab, q and the digits mean the same on each.
	ta := newTestApp(t)
	for i := range 3 {
		ta.press(fmt.Sprint(i + 1))
		if ta.Screen().focus != i {
			t.Fatalf("%d did not go to panel [%d]", i+1, i+1)
		}
		ta.press("Space")
		if m, ok := ta.topView().(*menuView); !ok || !m.space {
			t.Errorf("panel [%d]: Space did not open the Space menu", i+1)
		}
		ta.press("Space", "?")
		if n, ok := ta.topView().(*noteView); !ok || !n.helpCloses {
			t.Errorf("panel [%d]: ? did not open the key reference", i+1)
		}
		ta.press("?", "Esc")
		if ta.top() != nil || ta.quitting {
			t.Errorf("panel [%d]: Esc with nothing to close did something", i+1)
		}
		ta.press("q")
		if !ta.quitting {
			t.Errorf("panel [%d]: q did not start the quit flow", i+1)
		}
		ta.press("Esc", "Tab")
		if ta.Screen().focus != (i+1)%3 {
			t.Errorf("panel [%d]: Tab went to [%d]", i+1, ta.Screen().focus+1)
		}
	}
	// The popups: the digits are free there, q quits (not on a form), ? opens
	// and closes the popup's own key reference, Esc closes one layer.
	for _, s := range covAPopups() {
		ta := covAOpen(t, s)
		base, depth := ta.top(), len(ta.stack)
		ta.press("2")
		if ta.top() != base || ta.Screen().focus != 0 {
			t.Errorf("%s: 2 went to a panel", s.name)
		}
		ta.press("q")
		switch {
		case s.name == "form":
			if ta.quitting || ta.top() != base {
				t.Errorf("form: q started the quit flow (Rules K9)")
			}
		case !ta.quitting || !covAQuitConfirm(ta):
			t.Errorf("%s: q did not start the quit flow", s.name)
		default:
			ta.press("Esc")
		}
		if ta.top() != base {
			t.Fatalf("%s: Esc on the quit confirm did not come back", s.name)
		}
		ta.press("?")
		if s.name == "key reference" {
			if ta.top() != nil {
				t.Error("? on the key reference did not close it")
			}
			continue
		}
		if n, ok := ta.topView().(*noteView); !ok || !n.helpCloses {
			t.Errorf("%s: ? did not open its key reference: %T", s.name, ta.topView())
		}
		ta.press("?")
		if ta.top() != base {
			t.Errorf("%s: ? again did not close the key reference", s.name)
		}
		ta.press("Esc")
		if len(ta.stack) != depth-1 {
			t.Errorf("%s: Esc closed %d layers", s.name, depth-len(ta.stack))
		}
	}
}

func TestK1CustomCoreKeyEverywhere(t *testing.T) {
	ta := newTestApp(t)
	ta.CoreKeys = []CoreKey{{Key: "!", Desc: "the app log", Do: ta.record("log")}}
	// The same on a panel, over the Space menu, a confirm and a key reference.
	for _, open := range [][]string{nil, {"Space"}, {"x"}, {"?"}} {
		ta.ran = nil
		ta.press(open...)
		ta.press("!")
		if len(ta.ran) != 1 || ta.ran[0] != "log" {
			t.Errorf("! after %v ran %v", open, ta.ran)
		}
		for len(ta.stack) > 0 {
			ta.press("Esc")
		}
	}
	// A typing row takes it as a character (Rules K8).
	ta.ran = nil
	ta.press("/", "!")
	if len(ta.ran) > 0 || ta.list.filter.value() != "!" {
		t.Errorf("! on a typing row: ran %v, typed %q", ta.ran, ta.list.filter.value())
	}
	ta.press("Esc", "Esc")
	// A panel's ? lists it with the core keys.
	ta.press("?")
	if !strings.Contains(ta.screenText(), "the app log") {
		t.Error("the app's core key is not in the key reference")
	}
	ta.press("Esc")
	// No hotkey may take it.
	ta.pods.Actions = append(ta.pods.Actions, Action{Key: "!", Label: "Bang", Desc: "it", Scope: ItemScope})
	if ta.checkKeys() == nil {
		t.Error("a hotkey took the app's core key")
	}
}

func TestK1DigitsPerScreen(t *testing.T) {
	ta := newTestApp(t)
	alpha, beta := &Panel{Label: "Alpha", Body: &Pager{}}, &Panel{Label: "Beta", Body: &Pager{}}
	ta.AddScreen(&Screen{Name: "Two", Key: "W", Panels: []*Panel{alpha, beta}})
	ta.press("W")
	if text := ta.screenText(); !strings.Contains(text, "[1] ALPHA") || !strings.Contains(text, "[2] BETA") {
		t.Errorf("each screen numbers its panels from [1]:\n%s", text)
	}
	ta.press("2")
	if ta.cur != 2 || ta.Screen().Focused() != beta {
		t.Error("2 did not go to this screen's [2]")
	}
	ta.press("3")
	if ta.Screen().Focused() != beta {
		t.Error("3, with no [3] on this screen, did something")
	}
	if ta.screens[0].focus != 0 {
		t.Error("a digit on one screen moved the focus on another")
	}
	ta.press("P", "3")
	if ta.cur != 0 || ta.Screen().focus != 2 {
		t.Error("back on PANELS, 3 is its [3]")
	}
}

func TestK1DigitHotkeyRejectedOnPanel(t *testing.T) {
	for d := '1'; d <= '9'; d++ {
		for _, scope := range []Scope{ItemScope, PanelScope} {
			if covARegister(string(d), scope) == nil {
				t.Errorf("a panel took %c as a hotkey (scope %d)", d, scope)
			}
		}
	}
}

func TestM4DigitGlobalOnlyInPopup(t *testing.T) {
	ta := newTestApp(t)
	ta.Global = append(ta.Global, Action{Key: "1", Label: "About", Desc: "the app", Scope: GlobalScope, Do: ta.record("about")})
	ta.press("2", "1")
	if ta.Screen().focus != 0 || len(ta.ran) != 0 {
		t.Errorf("on a panel 1 goes to [1]: focus [%d], ran %v", ta.Screen().focus+1, ta.ran)
	}
	ta.press("Space", "1")
	if len(ta.ran) != 0 || ta.top() == nil {
		t.Errorf("1 in the Space menu ran %v", ta.ran)
	}
	ta.press("G", "Enter")
	if !strings.Contains(ta.screenText(), "[1] About") {
		t.Errorf("the global operation popup shows [1] About:\n%s", ta.screenText())
	}
	ta.press("1")
	if fmt.Sprint(ta.ran) != "[about]" || ta.top() != nil {
		t.Errorf("1 in the global operation popup: ran %v, top %v", ta.ran, ta.top())
	}
}

// --- K2 ----------------------------------------------------------------------

func TestK2TabCyclesPanels(t *testing.T) {
	ta := newTestApp(t)
	var seen []int
	for range 4 {
		seen = append(seen, ta.Screen().focus+1)
		if n := strings.Count(ta.screenText(), "┏"); n != 1 {
			t.Errorf("%d heavy frames: the focus is on one panel, never on the chips or the footer", n)
		}
		ta.press("Tab")
	}
	if fmt.Sprint(seen) != "[1 2 3 1]" {
		t.Errorf("Tab went %v, want [1 2 3 1]", seen)
	}
	if ta.cur != 0 {
		t.Error("Tab crossed to another screen")
	}
	ta.press("O", "Tab")
	if ta.cur != 1 || ta.Screen().focus != 0 {
		t.Error("Tab on a screen of one panel left it")
	}
}

func TestK2TabStaysInPopup(t *testing.T) {
	surfaces := append(covAPopups(), covASurface{"text input", func(ta *testApp) {
		ta.Text(TextInput{Title: "Rename", Old: "assets"})
	}})
	for _, s := range surfaces {
		ta := covAOpen(t, s)
		base, depth := ta.top(), len(ta.stack)
		ta.press("Tab", "Tab", "Tab")
		if ta.top() != base || len(ta.stack) != depth || ta.Screen().focus != 0 || ta.cur != 0 {
			t.Errorf("%s: Tab left the popup (top %T, focus [%d], screen %d)", s.name, ta.topView(), ta.Screen().focus+1, ta.cur)
		}
	}
}

func TestFinderTypingTab(t *testing.T) {
	ta := newTestApp(t)
	ta.Find(testFinder())
	ta.settle()
	fv := ta.topView().(*finderView)
	ta.press("r", "e", "Tab")
	if !fv.onList || fv.ed.value() != "re" {
		t.Errorf("Tab on a finder's typing row goes to the list: onList %v, typed %q", fv.onList, fv.ed.value())
	}
	ta.press("Esc")
	ta.Select(Select{Title: "Profile", Options: []string{"clock", "matrix", "pipes"}})
	ta.settle()
	sv := ta.topView().(*finderView)
	ta.press("/", "p", "Tab")
	if !sv.onList || sv.ed.value() != "p" {
		t.Errorf("Tab on a select's typing row goes to the list: onList %v, typed %q", sv.onList, sv.ed.value())
	}
}

func TestPanelFilterTab(t *testing.T) {
	ta := newTestApp(t)
	ta.press("/", "1", "Tab")
	if ta.list.Typing() || !ta.list.Filtering() || ta.list.filter.value() != "1" || ta.Screen().focus != 0 {
		t.Fatal("Tab on the typing row goes to the list, the filter kept")
	}
	ta.press("Tab")
	if ta.Screen().focus != 1 {
		t.Error("Tab on the filtered list goes to the next panel")
	}
	ta.press("Tab", "Tab")
	if ta.Screen().focus != 0 || !ta.list.Typing() {
		t.Error("coming back, the focus is on the typing row")
	}
	ta.press("Tab")
	if ta.Screen().focus != 0 || ta.list.Typing() {
		t.Error("Tab stops twice on a filtered panel: the typing row, then the list")
	}
}

func TestInputSuggestionTab(t *testing.T) {
	ta := newTestApp(t)
	got := ""
	ta.Text(TextInput{Title: "Rename", Old: "assets", Done: func(v string) tea.Cmd { got = v; return nil }})
	ta.settle()
	li := ta.topView().(*lineInput)
	if !strings.Contains(ta.screenText(), "Tab:edit") {
		t.Error("the hint does not offer Tab for the grey suggestion")
	}
	ta.press("Tab")
	if li.ed.value() != "assets" {
		t.Fatalf("Tab did not take the suggestion: %q", li.ed.value())
	}
	ta.press("-", "2", "Enter")
	if got != "assets-2" {
		t.Errorf("wrote %q", got)
	}
	port := ""
	ta.Number(NumberInput{Title: "Port", Old: "22", Min: 1, Max: 65535, Done: func(v string) tea.Cmd { port = v; return nil }})
	ta.settle()
	ta.press("Tab", "0", "Enter")
	if port != "220" {
		t.Errorf("a number's suggestion: wrote %q", port)
	}
}

func TestInputNoSuggestionTabNoop(t *testing.T) {
	ta := newTestApp(t)
	ta.Text(TextInput{Title: "Nickname"})
	ta.settle()
	li := ta.topView().(*lineInput)
	before := ta.screenText()
	ta.press("Tab")
	if ta.screenText() != before || li.ed.value() != "" || ta.topView() != li {
		t.Errorf("Tab with no suggestion did something: %q", li.ed.value())
	}
	ta.press("a", "Tab")
	if li.ed.value() != "a" {
		t.Errorf("Tab after typing did something: %q", li.ed.value())
	}
	ta.press("Esc")
	ta.Text(TextInput{Title: "Rename", Old: "assets"})
	ta.settle()
	li = ta.topView().(*lineInput)
	ta.press("Backspace", "Tab")
	if li.ed.value() != "" {
		t.Errorf("the suggestion thrown away, Tab still took it: %q", li.ed.value())
	}
}

// --- K3 ----------------------------------------------------------------------

func TestK3EnterRunsRegistered(t *testing.T) {
	ta := newTestApp(t)
	if !strings.Contains(ta.screenText(), "Enter:logs") {
		t.Error("the hint does not say what Enter does here")
	}
	ta.press("Enter", "j", "j", "Enter")
	if fmt.Sprint(ta.ran) != "[logs logs]" {
		t.Errorf("Enter on two pods ran %v, the same action each time", ta.ran)
	}
}

func TestK3EnterOnContentPanel(t *testing.T) {
	ta := newTestApp(t)
	logp := ta.Screen().Panels[1]
	logp.Actions = []Action{{Key: "Enter", Label: "View", Desc: "the whole log", Scope: PanelScope, Hint: "view", Do: ta.record("view")}}
	ta.press("2")
	if !strings.Contains(ta.screenText(), "Enter:view") {
		t.Error("the log's hint does not say what Enter does")
	}
	ta.press("Enter")
	if fmt.Sprint(ta.ran) != "[view]" {
		t.Errorf("Enter on a panel without items ran %v", ta.ran)
	}
}

func TestK3EnterKeepsFocus(t *testing.T) {
	ta := newTestApp(t)
	ta.press("j", "j", "Enter")
	if r, _ := ta.list.Current(); ta.Screen().focus != 0 || r.ID != "2" {
		t.Errorf("Enter moved the focus: panel [%d], row %s", ta.Screen().focus+1, r.ID)
	}
	ta.press("3", "Enter")
	if ta.values.cursor != 0 || ta.Screen().focus != 2 {
		t.Errorf("Enter on a value moved the focus: %d", ta.values.cursor)
	}
	ta.press("j", "j", "Enter") // Layout: column
	if f := ta.values.Fields[1]; f.Selected != 1 || ta.values.cursor != 2 {
		t.Errorf("choosing a radio in place moved the focus: selected %d, cursor %d", f.Selected, ta.values.cursor)
	}
	r := &formResult{}
	ta.Form(testForm(r))
	ta.settle()
	fv := ta.topView().(*formView)
	ta.press("j", "j", "j", "j", "j", "Enter") // Agent: a checkbox
	if formStop(fv) != "Agent:0" {
		t.Errorf("toggling a checkbox moved the focus to %s", formStop(fv))
	}
	ta.press("gg", "Enter", "w", "Enter")
	if r.name != "w" || ta.topView() != fv || formStop(fv) != "Name" {
		t.Errorf("after the input popup the focus is back on its field: %s, name %q", formStop(fv), r.name)
	}
}

func TestK3InputConfirm(t *testing.T) {
	ta := newTestApp(t)
	got := ""
	ta.Text(TextInput{Title: "Name", Done: func(v string) tea.Cmd { got = v; return nil }})
	ta.settle()
	ta.press("w", "e", "b", "Enter")
	if got != "web" || ta.top() != nil {
		t.Errorf("Enter wrote %q and left the popup %v", got, ta.top())
	}
	ta.Number(NumberInput{Title: "Port", Min: 1, Max: 65535, Done: func(v string) tea.Cmd { got = v; return nil }})
	ta.settle()
	ta.press("8", "0", "Enter")
	if got != "80" || ta.top() != nil {
		t.Errorf("a number: wrote %q, popup %v", got, ta.top())
	}
}

func TestK3InputInvalidStays(t *testing.T) {
	ta := newTestApp(t)
	got, calls := "", 0
	ta.Number(NumberInput{Title: "Port", Min: 1, Max: 65535, Done: func(v string) tea.Cmd { got = v; calls++; return nil }})
	ta.settle()
	li := ta.topView().(*lineInput)
	ta.press("0", "Enter")
	if calls != 0 || ta.topView() != li {
		t.Fatalf("a wrong value was written (%q) or closed the popup", got)
	}
	row := ta.rowWith("Port must be 1-65535")
	if row == "" || !strings.Contains(row, "38;2;243;139;168") {
		t.Errorf("the error row does not say what is wrong in Red:\n%s", ta.screenText())
	}
	ta.press("Backspace", "8", "0", "Enter")
	if got != "80" || ta.top() != nil {
		t.Errorf("fixed, it is written: %q", got)
	}
}

func TestK3InputUnchangedNoWrite(t *testing.T) {
	ta := newTestApp(t)
	calls := 0
	done := func(string) tea.Cmd { calls++; return nil }
	for _, keys := range [][]string{{"Enter"}, {"Tab", "Enter"}, {"a", "s", "s", "e", "t", "s", "Enter"}} {
		ta.Text(TextInput{Title: "Rename", Old: "assets", Done: done})
		ta.settle()
		ta.press(keys...)
		if calls != 0 || ta.top() != nil {
			t.Errorf("%v: the same value was written (%d) or the popup stayed", keys, calls)
		}
	}
	ta.Number(NumberInput{Title: "Port", Old: "22", Min: 1, Max: 65535, Done: done})
	ta.settle()
	ta.press("2", "2", "Enter")
	if calls != 0 || ta.top() != nil {
		t.Error("a number: the same value was written")
	}
	day := time.Date(2026, 10, 15, 0, 0, 0, 0, time.Local)
	ta.PickDate(DatePicker{Title: "Due", Value: day, Done: func(time.Time) tea.Cmd { calls++; return nil }})
	ta.settle()
	ta.press("Enter")
	if calls != 0 || ta.top() != nil {
		t.Error("a date: the same value was written")
	}
}

func TestDatetimeEnterMonth(t *testing.T) {
	ta := newTestApp(t)
	calls := 0
	ta.PickDate(DatePicker{Title: "Due", Value: time.Date(2026, 10, 15, 0, 0, 0, 0, time.Local),
		Done: func(time.Time) tea.Cmd { calls++; return nil }})
	ta.settle()
	dv := ta.topView().(*dateView)
	ta.press("Tab")
	if dv.focus != onMonth {
		t.Fatalf("Tab after the days goes to the month: %v", dv.focus)
	}
	ta.press("Enter")
	if dv.focus != onDays || calls != 0 || ta.topView() != dv {
		t.Errorf("Enter on the month goes to the days without choosing: focus %v, written %d", dv.focus, calls)
	}
	ta.press("Tab", "Tab", "Enter")
	if dv.focus != onDays || calls != 0 {
		t.Errorf("Enter on the year goes to the days: focus %v, written %d", dv.focus, calls)
	}
}

func TestMenuEnterRuns(t *testing.T) {
	ta := newTestApp(t)
	ta.Menu("Sort by", MenuSection{Actions: []Action{{Label: "Name", Do: ta.record("name")}, {Label: "Age", Do: ta.record("age")}}})
	ta.settle()
	ta.press("j", "Enter")
	if fmt.Sprint(ta.ran) != "[age]" || ta.top() != nil {
		t.Errorf("Enter ran %v; the menu %v", ta.ran, ta.top())
	}
	ta.press("Space", "j", "Enter") // mark, the second row
	if len(ta.list.Marked()) != 1 || ta.top() != nil {
		t.Errorf("Enter in the Space menu: marked %v", ta.list.Marked())
	}
}

func TestConfirmEnterAccepts(t *testing.T) {
	ta := newTestApp(t)
	accepted := 0
	ta.Confirm(Ask{Title: "Overwrite", Question: "Overwrite it?", Verb: "overwrite"}, func() tea.Cmd { accepted++; return nil })
	ta.settle()
	if !strings.Contains(ta.screenText(), "Enter:overwrite") {
		t.Error("the hint does not say what Enter does")
	}
	ta.press("Enter")
	if accepted != 1 || ta.top() != nil {
		t.Errorf("Enter: accepted %d, popup %v", accepted, ta.top())
	}
	ta.press("q")
	if !covAQuits(ta.Update2(keyMsg("Enter"))) {
		t.Error("Enter on the quit confirm did not leave")
	}
}

// --- K4 ----------------------------------------------------------------------

func TestK4EscNeverQuits(t *testing.T) {
	states := []covASurface{
		{"a panel", func(*testApp) {}},
		{"the panel filter", func(ta *testApp) { ta.press("/", "x") }},
		{"a mode", func(ta *testApp) { ta.press("D") }},
		{"three popups", func(ta *testApp) { ta.press("Space", "G", "Enter", "?") }},
		{"a toast", func(ta *testApp) { ta.Toast(Info, "Copied") }},
		{"a changed form", func(ta *testApp) { ta.Form(testForm(nil)); ta.settle(); ta.press("Enter", "w", "Enter") }},
		{"a changed textarea", func(ta *testApp) { ta.Textarea(Textarea{Title: "Comment"}); ta.settle(); ta.press("x") }},
		{"a text input", func(ta *testApp) { ta.Text(TextInput{Title: "Rename"}) }},
	}
	for _, s := range states {
		ta := newTestApp(t)
		ta.Quit = nil // a quit would leave at once, with no confirm to hide it
		s.open(ta)
		ta.settle()
		for i := range 8 {
			if covAQuits(ta.press("Esc")) {
				t.Errorf("%s: Esc %d left the app", s.name, i+1)
			}
		}
	}
}

func TestK4EscOnlyTop(t *testing.T) {
	ta := newTestApp(t)
	ta.press("Space", "G", "Enter", "j")
	global := ta.topView().(*menuView)
	cursor := global.cursor
	ta.press("q")
	if len(ta.stack) != 3 || !covAQuitConfirm(ta) {
		t.Fatalf("Space menu → global operation popup → quit confirm: %d layers", len(ta.stack))
	}
	under := []popupView{ta.stack[0].view, ta.stack[1].view}
	ta.press("Esc")
	if len(ta.stack) != 2 || ta.stack[0].view != under[0] || ta.stack[1].view != under[1] {
		t.Fatalf("Esc closed more than the top: %d layers", len(ta.stack))
	}
	if global.cursor != cursor || ta.quitting {
		t.Errorf("the popup under it is not as it was: cursor %d → %d, quitting %v", cursor, global.cursor, ta.quitting)
	}
}

// --- K5 ----------------------------------------------------------------------

func TestK5SpaceNoopOnPopups(t *testing.T) {
	for _, s := range covAPopups() {
		if s.name == "Space menu" || s.name == "checkbox popup" {
			continue // the Space menu closes (TestK5SpaceToggles); the checkbox list checks (TestCheckboxPopupSpace)
		}
		ta := covAOpen(t, s)
		base, depth, before := ta.top(), len(ta.stack), ta.screenText()
		ta.press("Space")
		if ta.top() != base || len(ta.stack) != depth || ta.screenText() != before {
			t.Errorf("%s: Space did something", s.name)
		}
	}
	ta := newTestApp(t)
	ta.press("Space", "G", "Enter", "Space")
	if len(ta.stack) != 2 {
		t.Error("Space closed the global operation popup: only the Space menu is its own")
	}
}

func TestCheckboxPopupSpace(t *testing.T) {
	ta := newTestApp(t)
	var got []bool
	ta.Checks(Checks{Title: "Namespaces", Options: []string{"default", "kube-system"}, Checked: []bool{false, false},
		Done: func(v []bool) tea.Cmd { got = v; return nil }})
	ta.settle()
	fv := ta.topView().(*finderView)
	ta.press("Space")
	if !fv.checked[0] || got != nil || ta.topView() != fv {
		t.Fatal("Space on the list checks the row on screen, writes nothing and keeps the popup")
	}
	if !strings.Contains(ta.screenText(), glyphMarked+" default") {
		t.Errorf("the row does not show its check:\n%s", ta.screenText())
	}
	ta.press("Space")
	if fv.checked[0] {
		t.Error("Space again did not uncheck it")
	}
	ta.press("/", "Space")
	if fv.ed.value() != " " || fv.checked[0] {
		t.Errorf("on the typing row Space is a character: %q", fv.ed.value())
	}
}

func TestK5NoMenuOnPopup(t *testing.T) {
	for _, s := range covAPopups() {
		if s.name == "Space menu" || s.name == "checkbox popup" {
			continue
		}
		ta := covAOpen(t, s)
		depth := len(ta.stack)
		ta.press("Space")
		if m, ok := ta.topView().(*menuView); len(ta.stack) != depth || ok && m.space {
			t.Errorf("%s: Space put a menu on the popup", s.name)
		}
	}
	// A popup's own keys are in its hint and its ?.
	ta := newTestApp(t)
	v := testViewer()
	v.Keys = []ViewerKey{{Key: "c", Desc: "copy"}}
	ta.Viewer(v)
	ta.settle()
	if !strings.Contains(ta.screenText(), "c:copy") {
		t.Error("the viewer's own key is not in its hint")
	}
	ta.press("?")
	if !slices.Contains(covARef(t, ta), "c copy") {
		t.Errorf("the viewer's own key is not in its ?: %v", covARef(t, ta))
	}
}

// --- K6 ----------------------------------------------------------------------

func TestK6HelpEverySurface(t *testing.T) {
	check := func(name string, ta *testApp) {
		t.Helper()
		base := ta.top()
		ta.press("?")
		n, ok := ta.topView().(*noteView)
		if !ok || !strings.HasSuffix(n.title, "KEYS") {
			t.Errorf("%s: ? did not open a key reference: %T", name, ta.topView())
			return
		}
		ta.press("?")
		if ta.top() != base {
			t.Errorf("%s: ? again did not close it", name)
		}
		ta.press("?", "Esc")
		if ta.top() != base {
			t.Errorf("%s: Esc did not close it", name)
		}
	}
	ta := newTestApp(t)
	for i := range 3 {
		ta.press(fmt.Sprint(i + 1))
		check(fmt.Sprintf("panel [%d]", i+1), ta)
	}
	ta.press("O")
	check("the other screen", ta)
	ta.press("P", "1", "D")
	check("a mode", ta)
	for _, s := range covAPopups() {
		if s.name != "key reference" {
			check(s.name, covAOpen(t, s))
		}
	}
	ta = newTestApp(t)
	v := testViewer()
	v.Select = &Selection{Key: "v", Name: "Visual"}
	ta.Viewer(v)
	ta.settle()
	ta.press("v")
	check("a viewer's selection mode", ta)
}

func TestK6PanelKeys(t *testing.T) {
	ta := newTestApp(t)
	ta.Screen().Panels[2].Actions = []Action{{Key: "R", Label: "Reset", Desc: "every setting", Scope: PanelScope}}
	ta.press("?")
	keys := covAKeys(covARef(t, ta))
	for _, k := range []string{"Enter", "m", "x", "D", "/", "j|k", "u|d", "gg|G", "h|l", "Space", "?", "Tab", "1–3", "q|Ctrl-C", "O", "S"} {
		if !keys[k] {
			t.Errorf("[1] PODS ? does not list %s", k)
		}
	}
	for _, k := range []string{"R", "Esc"} {
		if keys[k] {
			t.Errorf("[1] PODS ? lists %s, which does nothing here", k)
		}
	}
	ta.press("Esc", "/", "1", "Tab", "?")
	if !covAKeys(covARef(t, ta))["Esc"] {
		t.Error("with a filter set, Esc (clear it) is not listed")
	}
	ta.press("Esc", "Esc", "2", "?")
	keys = covAKeys(covARef(t, ta))
	if keys["m"] || keys["h|l"] || !keys["j|k"] {
		t.Errorf("[2] LOG ? lists another panel's keys or misses its own: %v", keys)
	}
}

func TestK6PopupKeysOnly(t *testing.T) {
	cases := []struct {
		name      string
		open      []string
		want, not []string
	}{
		{"Space menu", []string{"Space"}, []string{"Enter", "m", "x", "D", "/", "j|k", "Space|Esc"},
			[]string{"Tab", "1–3", "q|Ctrl-C", "S", "O", "?", "h|l"}},
		{"global operation popup", []string{"Space", "G", "Enter"}, []string{"Enter", "O", "S", "q", "Esc", "j|k"},
			[]string{"m", "x", "Tab", "1–3", "Space|Esc", "?"}},
		{"confirm", []string{"x"}, []string{"Enter", "Esc"}, []string{"Tab", "q|Ctrl-C", "m", "S", "?"}},
	}
	for _, c := range cases {
		ta := newTestApp(t)
		ta.press(c.open...)
		ta.press("?")
		keys := covAKeys(covARef(t, ta))
		for _, k := range c.want {
			if !keys[k] {
				t.Errorf("%s: ? does not list %s", c.name, k)
			}
		}
		for _, k := range c.not {
			if keys[k] {
				t.Errorf("%s: ? lists %s, not the popup's own", c.name, k)
			}
		}
	}
}

// --- K7 ----------------------------------------------------------------------

type covAAlias struct{ main, alt []string }

// covAMoveSurface is a surface one moves on with the keys of Rules K12.
type covAMoveSurface struct {
	covASurface
	aliases []covAAlias
	skip    map[string]string // by the main key: a known deviation
}

// covAMoveSurfaces are the lists, menus, viewers and pickers, and the
// textarea's moving state: everything that moves but is not a typing row.
func covAMoveSurfaces() []covAMoveSurface {
	k := func(keys ...string) []string { return keys }
	moves := []covAAlias{
		{k("j"), k("↓")}, {k("k"), k("↑")}, {k("l"), k("→")}, {k("h"), k("←")},
		{k("d"), k("PgDn")}, {k("G", "u"), k("G", "PgUp")},
	}
	text := []covAAlias{
		{k("j"), k("↓")}, {k("k"), k("↑")}, {k("l"), k("→")}, {k("h"), k("←")},
	}
	rows := make([]Row, 40)
	for i := range rows {
		rows[i] = Row{ID: fmt.Sprint(i), Cells: []string{fmt.Sprintf("row-%02d", i)}}
	}
	lines := make([]string, 40)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d of the log", i)
	}
	return []covAMoveSurface{
		{covASurface{"list without tabs", func(ta *testApp) {
			l := &List{Columns: []Column{{Title: "Name"}}}
			l.SetRows(rows)
			ta.AddScreen(&Screen{Name: "Flat", Key: "F", Panels: []*Panel{{Label: "Rows", Body: l}}})
			ta.press("F")
		}}, moves, map[string]string{}},
		{covASurface{"list with tabs", func(*testApp) {}}, moves, nil},
		{covASurface{"values", func(ta *testApp) { ta.press("3") }}, moves, nil},
		{covASurface{"pager", func(ta *testApp) { ta.Screen().Panels[1].Body.(*Pager).SetLines(lines); ta.press("2") }}, moves, nil},
		{covAPopups()[0], moves, nil}, // the Space menu
		{covASurface{"select", func(ta *testApp) { ta.Select(Select{Title: "Pick", Options: lines}) }}, moves, nil},
		{covAPopups()[9], moves, nil}, // the finder's list
		{covAPopups()[8], moves, nil}, // the checkbox popup
		{covASurface{"form", func(ta *testApp) { ta.Form(testForm(nil)) }}, moves, nil},
		{covAPopups()[4], moves, nil}, // the key reference
		{covASurface{"viewer", func(ta *testApp) { v := testViewer(); v.Tabs = []string{"Events"}; ta.Viewer(v) }}, moves, nil},
		{covASurface{"viewer's selection mode", func(ta *testApp) {
			v := &Viewer{Title: "YAML", Lines: lines, Select: &Selection{Key: "v", Name: "Visual"}}
			ta.Viewer(v)
			ta.settle()
			ta.press("v", "j", "w")
		}}, moves, nil},
		{covAPopups()[11], moves, nil}, // the date picker
		{covAPopups()[12], moves, nil}, // the color picker
		{covASurface{"textarea, moving", func(ta *testApp) {
			ta.Textarea(Textarea{Title: "Comment", Old: "one two\nthree four five\nsix"})
			ta.settle()
			ta.press("Esc", "k", "b")
		}}, text, nil},
	}
}

func TestK7AliasEverywhere(t *testing.T) {
	for _, s := range covAMoveSurfaces() {
		for _, a := range s.aliases {
			t.Run(s.name+"/"+strings.Join(a.main, "")+"="+strings.Join(a.alt, ""), func(t *testing.T) {
				if why := s.skip[strings.Join(a.main, "")]; why != "" {
					t.Skip(why)
				}
				after := func(keys []string) string {
					ta := newTestApp(t)
					ta.resize(80, 16)
					s.open(ta)
					ta.settle()
					ta.press(keys...)
					return ta.screenText()
				}
				if m, alt := after(a.main), after(a.alt); m != alt {
					t.Errorf("%v and %v differ:\n%s\n---\n%s", a.main, a.alt, m, alt)
				}
			})
		}
	}
}

// Home and End move only on a typing row; everywhere else gg|G and 0|$ do
// it, and Home and End do nothing (Rules K12).
func TestK12HomeEndOnlyTyping(t *testing.T) {
	for _, s := range covAMoveSurfaces() {
		for _, keys := range [][]string{{"End"}, {"G", "Home"}, {"j", "Home"}} {
			t.Run(s.name+"/"+strings.Join(keys, ""), func(t *testing.T) {
				ta := newTestApp(t)
				ta.resize(80, 16)
				s.open(ta)
				ta.settle()
				ta.press(keys[:len(keys)-1]...)
				before := ta.screenText()
				ta.press(keys[len(keys)-1])
				if after := ta.screenText(); after != before {
					t.Errorf("%s did something:\n%s\n---\n%s", keys[len(keys)-1], before, after)
				}
			})
		}
	}

	// On a typing row they go to an end.
	ta := newTestApp(t)
	ta.Text(TextInput{Title: "Name"})
	ta.settle()
	ta.press("a", "b", "c", "Home", "x", "End", "y")
	if v := ta.topView().(*lineInput).ed.value(); v != "xabcy" {
		t.Errorf("Home and End on a typing row: %q", v)
	}
	ta.Textarea(Textarea{Title: "Comment", Old: "one two"})
	ta.settle()
	ta.press("Home", "x", "End", "y")
	if v := ta.topView().(*textareaView).value(); v != "xone twoy" {
		t.Errorf("Home and End while writing in a textarea: %q", v)
	}
}

// --- K8, K9 ------------------------------------------------------------------

func TestK8InputState(t *testing.T) {
	ta := newTestApp(t)
	got := ""
	ta.Text(TextInput{Title: "Rename", Done: func(v string) tea.Cmd { got = v; return nil }})
	ta.settle()
	li := ta.topView().(*lineInput)
	ta.press("q", "?", "Space", "x", "S", "D", "j")
	if v := li.ed.value(); v != "q? xSDj" {
		t.Errorf("the keys are characters on the typing row: %q", v)
	}
	if len(ta.stack) != 1 || ta.quitting || len(ta.ran) != 0 {
		t.Errorf("a key worked as a key in the input state: %d layers, ran %v", len(ta.stack), ta.ran)
	}
	ta.press("Esc")
	if ta.top() != nil || got != "" {
		t.Fatal("Esc did not cancel the input")
	}
	ta.press("?")
	if _, ok := ta.topView().(*noteView); !ok {
		t.Error("? is not a key again once out of the input")
	}
	ta.press("Esc", "/", "q", "?", "Space", "m", "D")
	if v := ta.list.filter.value(); v != "q? mD" {
		t.Errorf("the keys are characters on the panel filter: %q", v)
	}
	if ta.top() != nil || ta.quitting || ta.pods.InMode() != nil || len(ta.list.Marked()) != 0 {
		t.Error("a key worked as a key on the panel filter")
	}
	ta.press("Tab", "?")
	if _, ok := ta.topView().(*noteView); !ok {
		t.Error("on the list again, ? is not the key reference")
	}
	ta.press("Esc")
	ta.Select(Select{Title: "Profile", Options: []string{"clock", "matrix"}})
	ta.settle()
	ta.press("q")
	if !ta.quitting {
		t.Error("a select's list is not the input state: q starts the quit flow")
	}
	ta.press("Esc", "Esc")
	ta.Form(testForm(nil))
	ta.settle()
	ta.press("Tab", "Tab", "?")
	if _, ok := ta.topView().(*noteView); !ok {
		t.Error("a form is not the input state: ? is its key reference")
	}
}

func TestK8CtrlCInInput(t *testing.T) {
	states := []covASurface{
		{"text input", func(ta *testApp) { ta.Text(TextInput{Title: "Rename"}) }},
		{"textarea, writing", func(ta *testApp) { ta.Textarea(Textarea{Title: "Comment"}) }},
		{"finder's typing row", func(ta *testApp) { ta.Find(testFinder()) }},
		{"panel filter", func(ta *testApp) { ta.press("/") }},
	}
	for _, s := range states {
		ta := covAOpen(t, s)
		depth := len(ta.stack)
		ta.press("Ctrl-C")
		if !ta.quitting || len(ta.stack) != depth+1 || !covAQuitConfirm(ta) {
			t.Errorf("%s: Ctrl-C did not start the quit flow", s.name)
			continue
		}
		if !covAQuits(ta.Update2(keyMsg("Ctrl-C"))) {
			t.Errorf("%s: the second Ctrl-C did not leave", s.name)
		}
	}
}

func TestK9QNoopOnForm(t *testing.T) {
	ta := newTestApp(t)
	ta.Form(testForm(nil))
	ta.settle()
	fv := ta.topView().(*formView)
	for _, at := range [][]string{{"j", "j"}, {"G"}} { // an option, the button
		ta.press("gg")
		ta.press(at...)
		ta.press("q")
		if ta.quitting || ta.topView() != fv {
			t.Errorf("q on %s did something: %T", formStop(fv), ta.topView())
		}
	}
	ta.press("gg", "q")
	if n, ok := ta.topView().(*noteView); ta.quitting || !ok || n.title != "NAME" {
		t.Errorf("q on a field one types in opens the typing note, not the quit flow: %T", ta.topView())
	}
	ta.press("Esc", "Ctrl-C")
	if !ta.quitting || !covAQuitConfirm(ta) {
		t.Error("Ctrl-C on a form starts the quit flow")
	}
}

func TestK9SecondCtrlCQuits(t *testing.T) {
	ta := newTestApp(t)
	if covAQuits(ta.press("Ctrl-C")) || !covAQuitConfirm(ta) {
		t.Fatal("the app asks first: the first Ctrl-C opens the quit confirm")
	}
	ta.press("Esc")
	if ta.quitting {
		t.Fatal("Esc on the quit confirm did not end the quit flow")
	}
	if covAQuits(ta.press("Ctrl-C")) {
		t.Error("after a cancel, Ctrl-C starts the flow again rather than leaving")
	}
	ta.press("?") // a key reference over the quit confirm: still in the flow
	if !covAQuits(ta.Update2(keyMsg("Ctrl-C"))) {
		t.Error("Ctrl-C during the quit flow did not leave at once")
	}
	ta = newTestApp(t)
	ta.press("Space", "G", "Enter", "G", "Enter") // the Quit row
	if !ta.quitting || !covAQuits(ta.Update2(keyMsg("Ctrl-C"))) {
		t.Error("Ctrl-C during the flow started from the global operation popup did not leave")
	}
	ta = newTestApp(t)
	ta.Quit = nil // the app's choice: leave at once
	if !covAQuits(ta.press("Ctrl-C")) {
		t.Error("an app that leaves at once did not")
	}
}

func TestK9QuitInGlobalPopup(t *testing.T) {
	ta := newTestApp(t)
	ta.press("Space", "G", "Enter")
	m := ta.topView().(*menuView)
	row := -1
	for i, r := range m.rows {
		if r.kind == menuItem && r.action.Label == "Quit" {
			row = i
		}
	}
	if row < 0 || !strings.Contains(ta.screenText(), "leave test") {
		t.Fatalf("no Quit row in the global operation popup:\n%s", ta.screenText())
	}
	for m.cursor != row {
		ta.press("j")
	}
	ta.press("Enter")
	if c, ok := ta.topView().(*confirmView); !ok || c.ask.Question != "Quit test?" || !ta.quitting {
		t.Fatalf("the Quit row did not start the app's quit flow: %T", ta.topView())
	}
	if !covAQuits(ta.Update2(keyMsg("Enter"))) {
		t.Error("Enter on the quit confirm did not leave")
	}
}

// --- K10 ---------------------------------------------------------------------

func TestK10ReservedKeysShown(t *testing.T) {
	ta := newTestApp(t)
	sent := &bytes.Buffer{}
	hidden := 0
	term := &Terminal{Title: "Alterm", in: sent, started: true}
	term.Keys = []TerminalKey{{Key: "Alt-t", Desc: "hide", Do: func() tea.Cmd { hidden++; return term.Hide(ta.App) }}}
	ta.open(&termView{t: term, a: ta.App})
	ta.settle()
	if last := StripStyles(ta.Frame()[ta.h-1]); !strings.Contains(last, "Alt-Esc:leave Alt-t:hide") {
		t.Errorf("the terminal popup's hint does not show the kept key after the exit key: %q", last)
	}
	ta.press("Alt-t")
	if hidden != 1 || sent.Len() != 0 {
		t.Errorf("the kept key is the app's, not the program's: ran %d, sent %q", hidden, sent.String())
	}
	if ta.top() != nil {
		t.Errorf("the app's Alt-t (no confirm, its choice) did not hide the popup: %T", ta.topView())
	}
	ta2, term2, _ := ptyApp(t)
	term2.Keys = []TerminalKey{{Key: "Alt-z", Desc: "zoom"}}
	if foot := StripStyles(ta2.footer(120)); !strings.Contains(foot, "Alt-z:zoom") {
		t.Errorf("in a panel's PTY the footer does not show the kept key: %q", foot)
	}
}

// --- K11 ---------------------------------------------------------------------

func TestK11ModeCoreKeys(t *testing.T) {
	ta := newTestApp(t)
	exited := 0
	ta.pods.EnterMode(&Mode{Name: "Drag", Exit: "cancel",
		Keys:   []ModeKey{{Key: "j|k", Keys: []string{"j", "k"}, Desc: "move the pod", NoFooter: true, Do: func(string) tea.Cmd { return nil }}},
		OnExit: func() tea.Cmd { exited++; return nil }})
	ta.press("Space")
	if ta.top() != nil {
		t.Errorf("Space in a mode opened %T", ta.topView())
	}
	ta.press("?")
	if n, ok := ta.topView().(*noteView); !ok || n.title != "DRAG KEYS" {
		t.Fatalf("? in a mode is the mode's key reference: %T", ta.topView())
	}
	keys := covAKeys(covARef(t, ta))
	if !keys["j|k"] || !keys["Esc"] || keys["Space"] || keys["m"] || keys["S"] {
		t.Errorf("the mode's key reference lists what works in the mode only: %v", keys)
	}
	ta.press("?")
	if ta.top() != nil || ta.pods.InMode() == nil {
		t.Error("? again closes it and stays in the mode")
	}
	ta.press("q")
	if !covAQuitConfirm(ta) {
		t.Error("q in a mode does not start the quit flow")
	}
	ta.press("Esc", "Ctrl-C")
	if !covAQuitConfirm(ta) {
		t.Error("Ctrl-C in a mode does not start the quit flow")
	}
	ta.press("Esc")
	if ta.pods.InMode() == nil {
		t.Fatal("closing the quit confirm left the mode")
	}
	ta.press("Esc")
	if ta.pods.InMode() != nil || exited != 1 {
		t.Errorf("Esc leaves the mode, back to before it: in mode %v, OnExit ran %d", ta.pods.InMode() != nil, exited)
	}
}

func TestK11TabPausedToast(t *testing.T) {
	ta := newTestApp(t)
	ta.press("D", "Tab")
	if ta.Screen().focus != 0 || ta.toast == nil || !strings.Contains(ta.screenText(), "Press [Esc] to leave Drag first") {
		t.Fatalf("Tab in a paused mode answers with a toast and stays:\n%s", ta.screenText())
	}
	ta.press("Esc")
	if ta.pods.InMode() == nil || ta.toast != nil {
		t.Fatal("the first Esc takes the toast, not the mode")
	}
	ta.press("Esc", "Tab")
	if ta.pods.InMode() != nil || ta.Screen().focus != 1 {
		t.Error("out of the mode, Tab goes to the next panel again")
	}
	ta.press("1")
	ta.pods.EnterMode(&Mode{Name: "Look"}) // not paused: Tab works
	ta.press("Tab")
	if ta.Screen().focus != 1 || ta.toast != nil {
		t.Error("in a mode that does not pause Tab, Tab goes to the next panel")
	}
}

func TestK11DigitsPausedToast(t *testing.T) {
	ta := newTestApp(t)
	ta.press("D")
	for _, d := range []string{"2", "3", "9"} {
		ta.press(d)
		if ta.Screen().focus != 0 || !strings.Contains(ta.screenText(), "Press [Esc] to leave Drag first") {
			t.Errorf("%s in a paused mode answers with a toast and stays", d)
		}
		ta.press("Esc")
		if ta.pods.InMode() == nil {
			t.Fatalf("%s: the first Esc left the mode, not the toast", d)
		}
	}
	ta.pods.LeaveMode()
	ta.pods.EnterMode(&Mode{Name: "Look"})
	ta.press("3")
	if ta.Screen().focus != 2 || ta.toast != nil {
		t.Error("in a mode that does not pause the digits, 3 goes to [3]")
	}
}

func TestK11ModeLabel(t *testing.T) {
	ta := newTestApp(t)
	yellow, blue := "38;2;249;226;175", "38;2;137;180;250"
	ta.press("D")
	top := ta.rowWith("PODS")
	if !strings.Contains(StripStyles(top), "┫Drag┣━┓") {
		t.Errorf("the mode name is not at the right of the top border: %q", StripStyles(top))
	}
	if !strings.Contains(top, "1;"+yellow+"mDrag") || !strings.Contains(top, yellow+"m┏") {
		t.Errorf("the name is not bold in the mode colour, or the frame did not turn it: %q", top)
	}
	ta.press("Esc")
	top = ta.rowWith("PODS")
	if strings.Contains(StripStyles(top), "Drag") || !strings.Contains(top, blue+"m┏") {
		t.Errorf("leaving the mode does not put the frame back: %q", top)
	}
	ta.pods.EnterMode(&Mode{Name: "Look"})
	ta.press("Tab")
	if top := StripStyles(ta.rowWith("PODS")); !strings.Contains(top, "┤Look├─┐") || !strings.HasPrefix(top, "┌") {
		t.Errorf("a panel in a mode without the focus keeps its light line and the name: %q", top)
	}
}

// --- K12 ---------------------------------------------------------------------

func TestK12HotkeyCannotTakeMovement(t *testing.T) {
	for _, key := range []string{"j", "k", "u", "d", "g", "G", "h", "l"} {
		for _, scope := range []Scope{ItemScope, PanelScope} {
			if covARegister(key, scope) == nil {
				t.Errorf("a panel took %q, kept for moving (scope %d)", key, scope)
			}
		}
	}
	for _, key := range []string{"go", "J", "K", "U", "H", "L", "x"} {
		if why := covARegister(key, ItemScope); why != nil {
			t.Errorf("%q refused: %v", key, why)
		}
	}
}

func TestK12OneDimensional(t *testing.T) {
	ta := newTestApp(t)
	ta.press("3", "l")
	if ta.values.cursor != 1 {
		t.Errorf("l on values without tabs: %d, want the next", ta.values.cursor)
	}
	ta.press("h")
	if ta.values.cursor != 0 {
		t.Errorf("h on values without tabs: %d, want the previous", ta.values.cursor)
	}
	vals := &Values{Fields: []*Field{{Label: "A", Value: func() string { return "a" }}, {Label: "B", Value: func() string { return "b" }}}}
	prefs := &Panel{Label: "Prefs", Tabs: []string{"Advanced"}, Body: vals}
	ta.AddScreen(&Screen{Name: "Prefs", Key: "W", Panels: []*Panel{prefs}})
	ta.press("W", "l")
	if prefs.Tab != 1 || vals.cursor != 0 {
		t.Errorf("in a panel with tabs l is the tabs': tab %d, cursor %d", prefs.Tab, vals.cursor)
	}
	ta.press("j")
	if vals.cursor != 1 {
		t.Error("in a panel with tabs j still moves in the list")
	}
	ta.Menu("Sort by", MenuSection{Actions: []Action{{Label: "Name"}, {Label: "Age"}, {Label: "Size"}}})
	ta.settle()
	m := ta.topView().(*menuView)
	ta.press("l")
	if m.cursor != 1 {
		t.Errorf("l in a menu: row %d", m.cursor)
	}
	ta.press("h")
	if m.cursor != 0 {
		t.Errorf("h in a menu: row %d", m.cursor)
	}
	ta.press("Esc")
	ta.Form(testForm(nil))
	ta.settle()
	fv := ta.topView().(*formView)
	ta.press("l")
	if formStop(fv) != "Port" {
		t.Errorf("l on a form: %s", formStop(fv))
	}
	ta.press("h")
	if formStop(fv) != "Name" {
		t.Errorf("h on a form: %s", formStop(fv))
	}
}

func TestK12HalfPageAndEnds(t *testing.T) {
	ta := newTestApp(t)
	ta.Frame()
	half := ta.list.height / 2
	cur := func() string { r, _ := ta.list.Current(); return r.ID }
	ta.press("d")
	if cur() != fmt.Sprint(half) {
		t.Errorf("d went to %s, want %d (half of %d rows)", cur(), half, ta.list.height)
	}
	ta.press("d", "u")
	if cur() != fmt.Sprint(half) {
		t.Errorf("d u went to %s", cur())
	}
	ta.press("G")
	if cur() != "39" || !strings.Contains(ta.screenText(), " 40 of 40 ") {
		t.Errorf("G went to %s", cur())
	}
	ta.press("gg")
	if cur() != "0" || !strings.Contains(ta.screenText(), " 1 of 40 ") {
		t.Errorf("gg went to %s", cur())
	}
}

func TestK12SingleGNoop(t *testing.T) {
	ta := newTestApp(t)
	gone := 0
	ta.pods.Actions = append(ta.pods.Actions, Action{Key: "go", Label: "goto", Desc: "a pod by name", Scope: PanelScope,
		Do: func() tea.Cmd { gone++; return nil }})
	ta.press("j", "j")
	before := ta.screenText()
	ta.press("g")
	if ta.screenText() != before || ta.top() != nil || len(ta.ran) != 0 {
		t.Error("a single g did something")
	}
	ta.press("o")
	if gone != 1 {
		t.Error("go, a g combination, is a hotkey")
	}
	ta.press("g", "g")
	if r, _ := ta.list.Current(); r.ID != "0" {
		t.Errorf("gg went to %s", r.ID)
	}
	ta.Menu("Sort by", MenuSection{Actions: []Action{{Label: "Name"}, {Label: "Age"}}})
	ta.settle()
	ta.press("j")
	before = ta.screenText()
	ta.press("g")
	if ta.screenText() != before {
		t.Error("a single g did something in a menu")
	}
	t.Run("the key after a lone g", func(t *testing.T) {
		ta := newTestApp(t)
		ta.press("g", "Space")
		if m, ok := ta.topView().(*menuView); !ok || !m.space {
			t.Error("g then Space: no Space menu")
		}
		ta = newTestApp(t)
		ta.press("Space", "g", "Esc")
		if ta.top() != nil {
			t.Error("g then Esc: the popup stayed")
		}
		ta = newTestApp(t)
		ta.press("g", "?")
		if _, ok := ta.topView().(*noteView); !ok {
			t.Error("g then ?: no key reference")
		}
		// A pair nobody registered: the g is forgotten and the key acts.
		ta = newTestApp(t)
		ta.press("g", "j")
		if r, _ := ta.list.Current(); r.ID != "1" {
			t.Errorf("g then j went to %s, not one row down", r.ID)
		}
		ta.Menu("Sort by", MenuSection{Actions: []Action{{Label: "Name"}, {Label: "Age"}}})
		ta.settle()
		m := ta.topView().(*menuView)
		was := m.cursor
		ta.press("g", "j")
		if m.cursor == was {
			t.Error("g then j in a menu: the cursor did not move")
		}
	})
}

func TestK12NoCtrlUD(t *testing.T) {
	ctrl := map[string]tea.KeyMsg{"Ctrl-D": {Type: tea.KeyCtrlD}, "Ctrl-U": {Type: tea.KeyCtrlU}}
	for _, s := range []covASurface{
		{"list", func(ta *testApp) { ta.press("j", "j") }},
		{"values", func(ta *testApp) { ta.press("3") }},
		{"menu", func(ta *testApp) { ta.press("Space") }},
		{"viewer", func(ta *testApp) { ta.Viewer(testViewer()); ta.settle(); ta.press("d") }},
	} {
		for name, msg := range ctrl {
			ta := covAOpen(t, s)
			before := ta.screenText()
			ta.Update(msg)
			ta.settle()
			if ta.screenText() != before {
				t.Errorf("%s: %s did something", s.name, name)
			}
		}
	}
}

func TestK12SpatialNoWrap(t *testing.T) {
	ta := newTestApp(t)
	day := func(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.Local) }
	ta.PickDate(DatePicker{Title: "Due", Value: day(2026, 10, 1)})
	ta.settle()
	dv := ta.topView().(*dateView)
	ta.press("h")
	if !dv.day.Equal(day(2026, 9, 30)) {
		t.Errorf("h on the first of the month: %v, want the day before", dv.day)
	}
	ta.press("Esc")
	ta.PickDate(DatePicker{Title: "Due", Value: day(2026, 12, 15)})
	ta.settle()
	dv = ta.topView().(*dateView)
	ta.press("Tab", "l")
	if !dv.day.Equal(day(2027, 1, 15)) {
		t.Errorf("l on December: %v, want January of the next year", dv.day)
	}
	ta.press("Esc")
	v := &Viewer{Title: "YAML", Lines: []string{"ab", "cd"}, Select: &Selection{Key: "v", Name: "Visual"}}
	ta.Viewer(v)
	ta.settle()
	ta.press("v", "k", "h")
	if v.line != 0 || v.col != 0 {
		t.Errorf("k and h at the start of the text: %d,%d", v.line, v.col)
	}
	ta.press("G", "$", "j", "l")
	if v.line != 1 || v.col != 1 {
		t.Errorf("j and l at the end of the text: %d,%d", v.line, v.col)
	}
	ta.press("Esc", "Esc")
	ta.Textarea(Textarea{Title: "Comment", Old: "ab\ncd"})
	ta.settle()
	tv := ta.topView().(*textareaView)
	ta.press("Esc", "j", "l")
	if tv.line != 1 || tv.col != 1 {
		t.Errorf("j and l at the end of the text: %d,%d", tv.line, tv.col)
	}
	ta.press("gg", "k", "h")
	if tv.line != 0 || tv.col != 0 {
		t.Errorf("k and h at the start of the text: %d,%d", tv.line, tv.col)
	}
}

func TestK12TextMotions(t *testing.T) {
	ta := newTestApp(t)
	ta.Textarea(Textarea{Title: "Comment", Old: "one two three"})
	ta.settle()
	tv := ta.topView().(*textareaView)
	var cols []int
	for _, k := range []string{"Esc", "0", "w", "w", "b", "e", "$"} {
		ta.press(k)
		cols = append(cols, tv.col)
	}
	if fmt.Sprint(cols) != "[12 0 4 8 4 6 12]" {
		t.Errorf("textarea moving: Esc 0 w w b e $ went %v", cols)
	}
	ta.press("Esc")
	var tabs []int
	v := &Viewer{Title: "DevTools", Tabs: []string{"Console", "Network"}, Lines: []string{"kind: Pod"},
		Select: &Selection{Key: "v", Name: "Visual"}, OnTab: func(i int) tea.Cmd { tabs = append(tabs, i); return nil }}
	ta.Viewer(v)
	ta.settle()
	ta.press("v", "l", "l")
	if v.col != 2 || len(tabs) != 0 {
		t.Errorf("in the selection mode l moves the cursor, not the tabs: col %d, tabs %v", v.col, tabs)
	}
	ta.press("h", "0", "w")
	if v.col != 4 || len(tabs) != 0 {
		t.Errorf("0 w: col %d", v.col)
	}
	ta.press("Esc", "l")
	if fmt.Sprint(tabs) != "[1]" {
		t.Errorf("out of the mode, l is the tabs': %v", tabs)
	}
}

// --- Rules E -----------------------------------------------------------------

func TestE1VersionHelp(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the showcase")
	}
	gocmd, err := exec.LookPath("go")
	if err != nil {
		t.Skip("no go command to build the showcase with")
	}
	bin := filepath.Join(t.TempDir(), "showcase")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if out, err := exec.Command(gocmd, "build", "-o", bin, "./examples/showcase").CombinedOutput(); err != nil {
		t.Fatalf("building the showcase: %v\n%s", err, out)
	}
	version := regexp.MustCompile(`^showcase \S+\n$`)
	first := ""
	for _, arg := range []string{"version", "--version", "-v"} {
		out, err := exec.Command(bin, arg).Output()
		if err != nil || !version.Match(out) {
			t.Errorf("showcase %s: %q, %v", arg, out, err)
		}
		if first == "" {
			first = string(out)
		} else if string(out) != first {
			t.Errorf("showcase %s printed %q, not what version printed", arg, out)
		}
	}
	for _, arg := range []string{"help", "-h", "--help"} {
		out, err := exec.Command(bin, arg).Output()
		if s := string(out); err != nil || !strings.Contains(s, "Usage:") || !strings.Contains(s, "showcase version") {
			t.Errorf("showcase %s does not print the usage: %q, %v", arg, s, err)
		}
	}
}

// covAPrivateUse reports a character of Unicode's private use areas: the
// Nerd Font icons and powerline symbols.
func covAPrivateUse(r rune) bool {
	return r >= 0xE000 && r <= 0xF8FF || r >= 0xF0000 && r <= 0xFFFFD || r >= 0x100000 && r <= 0x10FFFD
}

func TestE4NoLiteralPUA(t *testing.T) {
	files := 0
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != "." && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		files++
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(b), "\n") {
			for _, r := range line {
				if covAPrivateUse(r) {
					t.Errorf("%s:%d: a literal private-use character; write it as a code point (\\U%08X)", path, i+1, r)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if files < 10 {
		t.Fatalf("only %d Go files found", files)
	}
}

func TestE5ProbeCPR(t *testing.T) {
	q := probeQueries(true)
	if want := "\r" + iconProbe + "\x1b[6n\r" + powerlineProbe + "\x1b[6n"; !strings.HasPrefix(q, want) {
		t.Errorf("the probe prints an icon, asks where the cursor is, then a powerline symbol and asks again, each from column 1: %q", q)
	}
	if strings.Contains(probeQueries(false), "\x1b[6n") {
		t.Error("the cursor is asked for when both widths come from variables")
	}
	icon, _ := utf8.DecodeRuneInString(iconProbe)
	pl, _ := utf8.DecodeRuneInString(powerlineProbe)
	if !covAPrivateUse(icon) || icon >= 0xE0A0 && icon <= 0xE0D7 {
		t.Errorf("the icon probed, U+%04X, is not an icon", icon)
	}
	if pl < 0xE0A0 || pl > 0xE0D7 {
		t.Errorf("the powerline symbol probed, U+%04X, is not one", pl)
	}
	// Column 3 after printing one from column 1: it took two cells.
	for _, c := range []struct {
		reply         string
		icon, powerln int
	}{
		{"\x1b[12;3R\x1b[12;2R\x1b[?62;22c", 2, 1},
		{"\x1b[12;2R\x1b[12;3R\x1b[?62;22c", 1, 2},
		{"\x1b[12;3R\x1b[12;3R\x1b[?62;22c", 2, 2},
	} {
		ans, done := parseAnswers(c.reply)
		if !done || ans.icon != c.icon || ans.powerline != c.powerln {
			t.Errorf("%q: icon %d, powerline %d, want %d, %d", c.reply, ans.icon, ans.powerline, c.icon, c.powerln)
		}
	}
}
