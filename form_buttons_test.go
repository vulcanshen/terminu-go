package terminu

import (
	"errors"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// deliver runs cmd and hands what an action ends with back to the app, as
// the runtime would.
func (ta *testApp) deliver(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	switch msg := cmd().(type) {
	case tea.BatchMsg:
		for _, c := range msg {
			ta.deliver(c)
		}
	case formRan:
		ta.Update(msg)
		ta.settle()
	}
}

// buttonsForm is a form with three buttons in two rows, Test above Save As
// and the main Save (components/dialog/form, the action area).
type buttonsForm struct {
	f                 *Form
	name, host        string
	nameF, hostF      *FormField
	test, saveAs      *FormButton
	ran, saved, saves int
	res               FormResult
}

func newButtonsForm() *buttonsForm {
	b := &buttonsForm{}
	required := func(label string, v *string) func() string {
		return func() string {
			if *v == "" {
				return label + " is required"
			}
			return ""
		}
	}
	b.nameF = &FormField{Label: "Name", Value: func() string { return b.name }, Check: required("Name", &b.name)}
	b.hostF = &FormField{Label: "Host", Value: func() string { return b.host }, Check: required("Host", &b.host)}
	b.test = &FormButton{Label: "Test", Needs: []*FormField{b.hostF}, Run: func() FormResult { b.ran++; return b.res }}
	b.saveAs = &FormButton{Label: "Save As", Submit: func() (tea.Cmd, error) { b.saves++; return nil, nil }}
	b.f = &Form{Title: "New host", Fields: []*FormField{b.nameF, b.hostF}, Buttons: [][]*FormButton{{b.test}, {b.saveAs}},
		Submit: func() (tea.Cmd, error) { b.saved++; return nil, nil }}
	return b
}

func buttonAt(fv *formView) string {
	if fv.focus.kind != itemButton {
		return formStop(fv)
	}
	return fv.buttonLabel(fv.focus)
}

// Each row on the right, one cell between two buttons; the main submit at
// the bottom right.
func TestFormButtonsLayout(t *testing.T) {
	ta := newTestApp(t)
	b := newButtonsForm()
	ta.Form(b.f)
	ta.settle()
	_, plain := covEPopup(ta)
	test, save := covERow(plain, " Test "), covERow(plain, " Save ")
	if test < 0 || save != test+1 || !strings.HasSuffix(plain[test], " Test  ┃") || !strings.HasSuffix(plain[save], " Save As   Save  ┃") {
		t.Fatalf("Test above, Save As and Save below:\n%s", strings.Join(plain, "\n"))
	}
	if !strings.HasPrefix(plain[save+2], "┗") {
		t.Errorf("the last row of buttons is the body's last:\n%s", strings.Join(plain, "\n"))
	}
	// An empty last row puts the main submit on a row of its own.
	b = newButtonsForm()
	b.f.Buttons = [][]*FormButton{{b.test, b.saveAs}, {}}
	ta.Form(b.f)
	ta.settle()
	_, plain = covEPopup(ta)
	row := covERow(plain, " Save As ")
	if !strings.HasSuffix(plain[row], " Test   Save As  ┃") || !strings.HasSuffix(plain[row+1], " Save  ┃") {
		t.Errorf("an empty last row:\n%s", strings.Join(plain, "\n"))
	}
}

// The two areas: Tab between them, back to the item left, the first time to
// the main submit; j|k and gg|G inside the area (components/dialog/form).
func TestFormActionAreaMoves(t *testing.T) {
	ta := newTestApp(t)
	ta.Form(newButtonsForm().f)
	ta.settle()
	fv := ta.topView().(*formView)
	for _, c := range []struct{ keys, want string }{
		{"j", "Host"},
		{"Tab", "Save"},    // the main submit first
		{"j", "Test"},      // the action area wraps in reading order
		{"j", "Save As"},   // left to right, then the next row
		{"k k", "Save"},    // and back the other way
		{"gg", "Test"},     // the area's first
		{"G", "Save"},      // and last
		{"k Tab", "Host"},  // back to the field left
		{"Tab", "Save As"}, // and to the button left
		{"Shift-Tab", "Host"},
		{"j", "Name"}, // the field area wraps alone
	} {
		ta.press(strings.Fields(c.keys)...)
		if got := buttonAt(fv); got != c.want {
			t.Errorf("%s: %s, want %s", c.keys, got, c.want)
		}
	}
	ta.press("Tab") // back to Save As, the button left
	if h := covEBottom(ta); !strings.HasPrefix(h, "┗━ Enter:save as Tab:fields Esc:cancel ━") {
		t.Errorf("on Save As: %q", h)
	}
	ta.press("k")
	if h := covEBottom(ta); !strings.HasPrefix(h, "┗━ Enter:test Tab:fields Esc:cancel ━") {
		t.Errorf("on Test: %q", h)
	}
}

// An action checks only the fields it needs, runs with the loading icon
// after the title, and says it is done in a toast; the form stays.
func TestFormActionRun(t *testing.T) {
	ta := newTestApp(t)
	b := newButtonsForm()
	b.res = FormResult{Toast: "Connected to web-01"}
	ta.Form(b.f)
	ta.settle()
	fv := ta.topView().(*formView)
	ta.press("Tab", "j") // Test
	ta.deliver(ta.press("Enter"))
	if b.ran != 0 || fv.errs[b.hostF] == "" || fv.errs[b.nameF] != "" || buttonAt(fv) != "Host" {
		t.Fatalf("Test checks Host only, the focus to it: ran %d, errors %v, on %s", b.ran, fv.errs, buttonAt(fv))
	}
	if !strings.Contains(ta.screenText(), "Host is required") {
		t.Error("the error row shows the problem")
	}
	b.host = "web-01"
	ta.press("Tab")
	if buttonAt(fv) != "Test" {
		t.Fatalf("Tab back to Test: %s", buttonAt(fv))
	}
	cmd := ta.press("Enter")
	// One frame drawn once: the icon turns with the clock (layout/popup).
	text := ta.screenText()
	if !fv.isLoading() || !slices.ContainsFunc(loadingFrames[:], func(f string) bool { return strings.Contains(text, f) }) {
		t.Errorf("the loading icon after the title while it runs: %v\n%s", fv.isLoading(), text)
	}
	ta.deliver(cmd)
	if b.ran != 1 || fv.isLoading() || ta.topView() != fv || buttonAt(fv) != "Test" {
		t.Fatalf("it ran, the form stays, the focus on Test: ran %d, top %T, on %s", b.ran, ta.topView(), buttonAt(fv))
	}
	if !strings.Contains(ta.screenText(), "Connected to web-01") {
		t.Error("success says so in a toast")
	}
	if len(fv.errs) != 0 || strings.Contains(ta.screenText(), "Host is required") {
		t.Error("the errors stand until the next check: this one passed")
	}
	if fv.dirty {
		t.Error("an action that wrote nothing changed nothing")
	}
}

// A failure opens an error popup and Enter after it tries again; a result to
// read opens a note; a value written makes the form changed.
func TestFormActionResults(t *testing.T) {
	ta := newTestApp(t)
	b := newButtonsForm()
	b.host = "web-01"
	b.res = FormResult{Err: errors.New("connection refused")}
	ta.Form(b.f)
	ta.settle()
	fv := ta.topView().(*formView)
	ta.press("Tab", "j")
	ta.deliver(ta.press("Enter"))
	if n, ok := ta.topView().(*noteView); !ok || !strings.Contains(n.title, "TEST FAILED") {
		t.Fatalf("a failure opens an error popup: %T", ta.topView())
	}
	ta.press("Esc")
	if ta.topView() != fv || buttonAt(fv) != "Test" {
		t.Fatalf("closed, the focus on the button: %s", buttonAt(fv))
	}
	ta.deliver(ta.press("Enter"))
	if b.ran != 2 {
		t.Errorf("Enter again tries again: ran %d", b.ran)
	}
	ta.press("Esc")
	b.res = FormResult{Lines: []string{"ssh: connected", "uname: Linux"}}
	ta.deliver(ta.press("Enter"))
	if v, ok := ta.topView().(*viewerView); !ok || !strings.Contains(ta.screenText(), "uname: Linux") || v.v.Title != "Test" {
		t.Fatalf("a result to read opens a note: %T", ta.topView())
	}
	ta.press("Esc")
	b.res = FormResult{Toast: "Got the host key", Apply: func() { b.name = "web-01" }}
	ta.deliver(ta.press("Enter"))
	if b.name != "web-01" || !fv.dirty {
		t.Fatalf("Apply wrote the value, the form changed: %q, dirty %v", b.name, fv.dirty)
	}
	ta.press("Esc", "Esc") // the toast first (Rules K4), then the form
	if !strings.Contains(ta.screenText(), "Discard changes to New host?") {
		t.Error("Esc on a form an action changed asks first")
	}
}

// Another submit checks the whole form and closes it; the main one too.
func TestFormSaveAs(t *testing.T) {
	ta := newTestApp(t)
	b := newButtonsForm()
	b.host = "web-01"
	ta.Form(b.f)
	ta.settle()
	fv := ta.topView().(*formView)
	ta.press("Tab", "k", "Enter") // Save As
	if b.saves != 0 || fv.errs[b.nameF] == "" || buttonAt(fv) != "Name" {
		t.Fatalf("Save As checks the whole form: saves %d, on %s", b.saves, buttonAt(fv))
	}
	b.name = "web"
	ta.press("Tab", "Enter")
	if b.saves != 1 || b.saved != 0 || ta.top() != nil {
		t.Errorf("Save As submits and closes: saves %d, saved %d", b.saves, b.saved)
	}
}

// A button needs one of Submit and Run: a mistake in the program.
func TestFormButtonNeedsOne(t *testing.T) {
	for _, bad := range []*FormButton{
		{Label: "None"},
		{Label: "Both", Submit: func() (tea.Cmd, error) { return nil, nil }, Run: func() FormResult { return FormResult{} }},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: no panic", bad.Label)
				}
			}()
			newTestApp(t).Form(&Form{Title: "X", Buttons: [][]*FormButton{{bad}}})
		}()
	}
}

// On a cursor block the swatch keeps its colour, the hex turns Base
// (components/color, the cursor block; components/input/color-picker, value).
func TestSwatchOnCursor(t *testing.T) {
	c := Hex("#2A2A3C")
	swatch := sgrRGB(true, c.RGB)
	check := func(where, line string, block Style) {
		t.Helper()
		if !strings.Contains(line, swatch) || !strings.Contains(line, "███") {
			t.Errorf("%s: the swatch lost its colour: %q", where, line)
		}
		if !strings.Contains(line, "\x1b["+block.sgr()+"m #2A2A3C") {
			t.Errorf("%s: the hex is not Base on the block: %q", where, line)
		}
	}
	ta := newTestApp(t)
	ta.Form(&Form{Title: "Theme", Fields: []*FormField{{Label: "Background", Value: func() string { return ColorText(c) }}}})
	ta.settle()
	lines, layer := covFDraw(ta)
	check("form", covFRow(lines, "#2A2A3C"), On(layer))
	ta.press("Esc")
	// A value panel and a list.
	v := &Values{Fields: []*Field{{Label: "Background", Value: func() string { return ColorText(c) }}}}
	check("values", strings.Join(v.Draw(60, 3, true), "\n"), On(Subtext1))
	l := &List{}
	l.SetRows([]Row{{Cells: []string{ColorText(c)}}})
	check("list", l.Draw(30, 1, true)[0], On(Subtext1))
	// Other text on the block is Base, its own colour gone.
	if got := onBlock(Fg(Green).Render("ok"), On(Blue)); got != On(Blue).Render("ok") {
		t.Errorf("plain text on the block: %q", got)
	}
}
