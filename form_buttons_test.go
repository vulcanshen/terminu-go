package terminu

import (
	"context"
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
	ctx               context.Context // the last Run's
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
	b.test = &FormButton{Label: "Test", Needs: []*FormField{b.hostF}, Run: func(ctx context.Context) FormResult {
		b.ran++
		b.ctx = ctx
		return b.res
	}}
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
// TestFormButtonOpens: a button can open a popup over the form, a terminal
// popup for one; as it closes, Back reads what was done there into the
// form, which then counts as changed (components/dialog/form; sshu SS19).
func TestFormButtonOpens(t *testing.T) {
	ta := newTestApp(t)
	key := ""
	var term *Terminal
	f := &Form{Title: "New host", Name: "New host", Button: "Save",
		Fields: []*FormField{{Label: "Key", Value: func() string { return key }, Empty: "none"}},
		Buttons: [][]*FormButton{{{Label: "Copy key",
			Open: func() tea.Cmd {
				term = &Terminal{Title: "ssh-copy-id", in: &strings.Builder{}, started: true}
				ta.open(&termView{t: term, a: ta.App})
				return nil
			},
			Back: func() bool { key = "~/.ssh/id_ed25519.pub"; return true }}}},
	}
	ta.Form(f)
	ta.settle()
	fv := ta.topView().(*formView)
	ta.press("Tab", "h", "Enter") // the action area lands on Save; Copy key is before it
	ta.settle()
	if _, ok := ta.topView().(*termView); !ok {
		t.Fatalf("the terminal popup over the form: %T", ta.topView())
	}
	if key != "" {
		t.Error("Back runs only once it is gone")
	}
	ta.closeTop()
	ta.settle()
	if ta.topView() != fv || key != "~/.ssh/id_ed25519.pub" || !fv.dirty {
		t.Errorf("back on the form, the value read in, a change: %T %q %v", ta.topView(), key, fv.dirty)
	}
	if !strings.Contains(ta.screenText(), "~/.ssh/id_ed25519.pub") {
		t.Error("the form draws the value read back")
	}
}

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
		{Label: "Both", Submit: func() (tea.Cmd, error) { return nil, nil }, Run: func(context.Context) FormResult { return FormResult{} }},
		{Label: "Long and short", Submit: func() (tea.Cmd, error) { return nil, nil },
			SubmitLong: func(context.Context) (tea.Cmd, error) { return nil, nil }},
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

// While its work runs, Enter on the button does nothing and the hint leaves
// its Enter out, back once it ends (components/dialog/form).
func TestFormActionWhileRunning(t *testing.T) {
	ta := newTestApp(t)
	b := newButtonsForm()
	b.host = "web-01"
	b.res = FormResult{Toast: "Connected to web-01"}
	ta.Form(b.f)
	ta.settle()
	ta.press("Tab", "j") // Test
	cmd := ta.press("Enter")
	if h := covEBottom(ta); !strings.HasPrefix(h, "┗━ Tab:fields Esc:cancel ━") {
		t.Errorf("while it runs the hint has no Enter: %q", h)
	}
	if again := ta.press("Enter"); again != nil {
		ta.deliver(again)
	}
	ta.deliver(cmd)
	if b.ran != 1 {
		t.Errorf("Enter while it ran sent it again: ran %d", b.ran)
	}
	if h := covEBottom(ta); !strings.HasPrefix(h, "┗━ Enter:test Tab:fields Esc:cancel ━") {
		t.Errorf("once it ends Enter is back: %q", h)
	}
}

// Closed before its action ends, the form takes nothing back: ctx ends, and
// no toast, note, error popup or Apply (components/dialog/form).
func TestFormClosedDropsResult(t *testing.T) {
	for _, close := range [][]string{{"Esc"}, {"j", "Enter"}} { // Esc; Save As, another submit
		ta := newTestApp(t)
		b := newButtonsForm()
		b.name, b.host = "web", "web-01"
		applied := false
		b.res = FormResult{Err: errors.New("refused"), Apply: func() { applied = true }}
		ta.Form(b.f)
		ta.settle()
		ta.press("Tab", "j") // Test
		cmd := ta.press("Enter")
		ta.press(close...)
		if ta.top() != nil {
			t.Fatalf("%v did not close the form: %T", close, ta.topView())
		}
		ta.deliver(cmd)
		if b.ran != 1 || b.ctx.Err() == nil {
			t.Errorf("%v: its ctx did not end: ran %d", close, b.ran)
		}
		if ta.top() != nil || applied || strings.Contains(ta.screenText(), "refused") {
			t.Errorf("%v: the result showed after the form closed: top %T, applied %v", close, ta.topView(), applied)
		}
	}
}

// A submit that takes long runs off the update loop: the loading icon, no
// second press; failing it opens an error popup over the form, done it closes
// the form and runs its command.
func TestFormSubmitLong(t *testing.T) {
	ta := newTestApp(t)
	b := newButtonsForm()
	var fail error
	connects := 0
	b.f.Submit = nil
	b.f.Button = "Connect"
	b.f.SubmitLong = func(context.Context) (tea.Cmd, error) {
		connects++
		return func() tea.Msg { return nil }, fail
	}
	ta.Form(b.f)
	ta.settle()
	fv := ta.topView().(*formView)
	ta.deliver(ta.press("Tab", "Enter"))
	if connects != 0 || fv.errs[b.nameF] == "" {
		t.Fatalf("Connect checks the whole form first: connects %d", connects)
	}
	b.name, b.host = "web", "web-01"
	fail = errors.New("connection refused")
	ta.press("Tab")
	cmd := ta.press("Enter")
	if !fv.isLoading() || !strings.HasPrefix(covEBottom(ta), "┗━ Tab:fields Esc:cancel ━") {
		t.Errorf("it runs: loading %v, hint %q", fv.isLoading(), covEBottom(ta))
	}
	ta.deliver(cmd)
	if n, ok := ta.topView().(*noteView); !ok || !strings.Contains(n.title, "CONNECT FAILED") || buttonAt(fv) != "Connect" {
		t.Fatalf("a failure opens an error popup over the form: %T, on %s", ta.topView(), buttonAt(fv))
	}
	ta.press("Esc")
	fail = nil
	ta.deliver(ta.press("Enter"))
	if connects != 2 || ta.top() != nil {
		t.Errorf("done, the form closes: connects %d, top %T", connects, ta.topView())
	}
	// Done under a popup opened from the form, it closes that one too.
	b = newButtonsForm()
	b.name, b.host = "web", "web-01"
	b.nameF.Edit = func() Input { return TextInput{Title: "Name", Old: b.name} }
	b.f.Submit, b.f.SubmitLong = nil, func(context.Context) (tea.Cmd, error) { return nil, nil }
	ta.Form(b.f)
	ta.settle()
	cmd = ta.press("Tab", "Enter")
	ta.press("Tab", "Enter") // Name's input popup over the form
	if _, ok := ta.topView().(*lineInput); !ok {
		t.Fatalf("no input over the form: %T", ta.topView())
	}
	ta.deliver(cmd)
	ta.settle()
	if ta.top() != nil {
		t.Errorf("the form done closes with what is over it: %T", ta.topView())
	}
}

// A form with both Submit and SubmitLong is a mistake in the program.
func TestFormSubmitAndSubmitLong(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("no panic")
		}
	}()
	newTestApp(t).Form(&Form{Title: "X", Submit: func() (tea.Cmd, error) { return nil, nil },
		SubmitLong: func(context.Context) (tea.Cmd, error) { return nil, nil }})
}

// A button with a name: the verb in lower case in the hint, the name as it
// is; a failure titled with the name after ": " (components/dialog/form,
// hint; components/README, upper case).
func TestFormButtonSubject(t *testing.T) {
	ta := newTestApp(t)
	over := &FormButton{Label: "Overwrite", Subject: "KBU.svg", Submit: func() (tea.Cmd, error) { return nil, errors.New("read-only") }}
	ta.Form(&Form{Title: "Export", Fields: []*FormField{{Label: "Name", Value: func() string { return "kbu" }}},
		Button: "Save as", ButtonSubject: "KBU-2.svg", Buttons: [][]*FormButton{{over}}})
	ta.settle()
	ta.press("Tab")
	if _, plain := covEPopup(ta); covERow(plain, " Overwrite KBU.svg   Save as KBU-2.svg ") < 0 {
		t.Errorf("the buttons with their names:\n%s", strings.Join(plain, "\n"))
	}
	if h := covEBottom(ta); !strings.HasPrefix(h, "┗━ Enter:save as KBU-2.svg Tab:fields Esc:cancel ━") {
		t.Errorf("the main submit: %q", h)
	}
	ta.press("k")
	if h := covEBottom(ta); !strings.HasPrefix(h, "┗━ Enter:overwrite KBU.svg Tab:fields Esc:cancel ━") {
		t.Errorf("only the verb in lower case: %q", h)
	}
	ta.press("Enter")
	if n, ok := ta.topView().(*noteView); !ok || n.title != "OVERWRITE FAILED: KBU.svg" {
		t.Errorf("the failure keeps the name as it is: %T\n%s", ta.topView(), ta.screenText())
	}
}

// TestFormFocusWideArrow: with the powerline arrow two cells wide, the
// focused field still shows its whole value, no "…" (components/input, a value
// is cut only when too long; demou DM14).
func TestFormFocusWideArrow(t *testing.T) {
	for _, pw := range []int{1, 2} {
		withWidths(t, 1, pw)
		ta := newTestApp(t)
		ta.Form(testForm(nil))
		ta.settle()
		row := ""
		for _, l := range ta.Frame() {
			if strings.Contains(StripStyles(l), "Name") {
				row = StripStyles(l)
			}
		}
		if !strings.Contains(row, "required") || strings.Contains(row, "…") {
			t.Errorf("arrow %d cells: the focused field is %q", pw, row)
		}
	}
}
