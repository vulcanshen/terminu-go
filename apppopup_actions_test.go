package terminu

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// gridBody is a popup of the app's own with values to submit: a row of cells,
// each with three values Enter and Space go through.
type gridBody struct {
	cells   [4]int
	at      int
	keys    []string
	focused []bool // what each Draw was told
	changed bool
	wrong   string // what Check says
	checked int
}

func (b *gridBody) Size(int, int) (int, int) { return 30, 3 }

func (b *gridBody) Draw(w, h int, layer Color) []string {
	out := make([]string, h)
	for i := range out {
		out[i] = " " + Fit("cells", w-2) + " "
	}
	return out
}

func (b *gridBody) Focus(on bool) { b.focused = append(b.focused, on) }

func (b *gridBody) Key(key string, _ tea.KeyMsg) (bool, tea.Cmd) {
	b.keys = append(b.keys, key)
	switch key {
	case "Enter", "Space":
		b.cells[b.at] = (b.cells[b.at] + 1) % 3
		b.changed = true
	case "l":
		b.at = min(b.at+1, len(b.cells)-1)
	default:
		return false, nil
	}
	return true, nil
}

func (b *gridBody) Hint() []KeyHint { return []KeyHint{{"Enter|Space", "cycle"}} }

func (b *gridBody) Keys() []KeySection {
	return []KeySection{
		{Rows: []KeyRow{{Key: "Enter|Space", Desc: "the cell's next value"}}},
		{Title: "move", Rows: []KeyRow{{Key: "h|l", Desc: "left, right"}}},
	}
}

func (b *gridBody) Changed() bool { return b.changed }

// checkedBody is a gridBody whose values may be wrong.
type checkedBody struct{ gridBody }

func (b *checkedBody) Check() string { b.checked++; return b.wrong }

// tdp popup.8, form.7: with values to submit, the action area goes under the
// content, the separator over it, the main submit at the bottom right; Tab
// goes between the content and the buttons, the body told where the focus is
// and its hint in front of the lib's.
func TestAppPopupActions(t *testing.T) {
	ta := newTestApp(t)
	b := &gridBody{}
	saved, savedAs := 0, 0
	ta.Popup(Popup{Title: "Draw", Subject: "kamonu", Body: b, Actions: &PopupActions{
		Button: "Save", Content: "grid",
		Buttons: [][]*FormButton{{{Label: "Save As", Submit: func() (tea.Cmd, error) { savedAs++; return nil, nil }}}},
		Submit:  func() (tea.Cmd, error) { saved++; return nil, nil },
	}})
	ta.settle()
	lines, _ := covFDraw(ta)
	text := covFText(lines)
	l := ta.top()
	// The inside: a blank row, the three rows of cells, the separator, the
	// buttons, a blank row; no error row with no Check.
	if l.h != 3+2+4 || !strings.HasPrefix(text[2], "┃ cells") || !strings.HasPrefix(text[5], "┃ ──") {
		t.Fatalf("%d rows:\n%s", l.h, strings.Join(text, "\n"))
	}
	if !strings.HasSuffix(text[6], "Save As   Save  ┃") {
		t.Errorf("the buttons are not on the right, Save last: %q", text[6])
	}
	if h := covFHint(ta); !strings.Contains(h, " Enter|Space:cycle Tab:actions Esc:cancel ") {
		t.Errorf("hint on the content %q", h)
	}
	if !slices.Equal(b.focused[len(b.focused)-1:], []bool{true}) {
		t.Errorf("Draw was told %v", b.focused)
	}
	ta.press("Tab")
	covFDraw(ta)
	if h := covFHint(ta); !strings.Contains(h, " Enter:save Tab:grid Esc:cancel ") {
		t.Errorf("hint on Save %q", h)
	}
	if b.focused[len(b.focused)-1] {
		t.Error("Draw was not told the focus is on a button")
	}
	if len(b.keys) != 0 {
		t.Errorf("Tab reached the body: %v", b.keys)
	}
	// On the buttons the keys are the action area's: round, as on a form.
	ta.press("l")
	if h := covFHint(ta); !strings.Contains(h, " Enter:save as Tab:grid ") || len(b.keys) != 0 {
		t.Errorf("l from Save: hint %q, the body got %v", covFHint(ta), b.keys)
	}
	ta.press("Shift-Tab", "Tab")
	if h := covFHint(ta); !strings.Contains(h, " Enter:save as ") {
		t.Errorf("Tab back to the button left: %q", h)
	}
	ta.press("Enter")
	if savedAs != 1 || saved != 0 || ta.top() != nil {
		t.Errorf("Save As: %d, Save %d; it closes", savedAs, saved)
	}
}

// tdp K5 (K5-6), K9 (K9-7), F3 (F3-8): Space goes to the body, to the next
// value of the cell; q does nothing and never reaches the body, Ctrl-C is the
// quit flow; Esc closes it unchanged, and once changed asks first.
func TestAppPopupActionsKeys(t *testing.T) {
	ta := newTestApp(t)
	b := &gridBody{}
	cancelled := 0
	ta.Popup(Popup{Title: "Draw", Subject: "kamonu", Body: b, Actions: &PopupActions{OnCancel: func() tea.Cmd { cancelled++; return nil }}})
	ta.settle()
	ta.press("Space", "Space")
	if b.cells[0] != 2 || !slices.Equal(b.keys, []string{"Space", "Space"}) {
		t.Errorf("Space: the cell is %d, the body got %v", b.cells[0], b.keys)
	}
	b.keys = nil
	ta.press("q")
	if len(b.keys) != 0 || ta.quitting || ta.top() == nil {
		t.Errorf("q: the body got %v, quitting %v", b.keys, ta.quitting)
	}
	ta.press("Ctrl-C")
	if !ta.quitting {
		t.Error("Ctrl-C does not start the quit flow")
	}
	ta.press("Esc")
	ta.press("Esc")
	c, ok := ta.topView().(*confirmView)
	if !ok || c.ask.Question != "Discard changes to kamonu?" || len(b.keys) != 0 {
		t.Fatalf("Esc once changed: %T, the body got %v", ta.topView(), b.keys)
	}
	ta.press("Esc")
	if _, ok := ta.topView().(*formView); !ok || b.cells[0] != 2 {
		t.Fatal("Esc on the confirm goes back, the values kept")
	}
	ta.press("Esc", "Enter")
	ta.settle()
	if ta.top() != nil || cancelled != 1 {
		t.Errorf("discarded: %d layers, OnCancel %d times", len(ta.stack), cancelled)
	}
	// Unchanged, Esc closes it at once.
	b2 := &gridBody{}
	ta.Popup(Popup{Title: "Draw", Body: b2, Actions: &PopupActions{}})
	ta.settle()
	ta.press("Esc")
	if ta.top() != nil {
		t.Error("Esc unchanged does not close it")
	}
}

// tdp popup.7: a body with Check keeps an error row; a submit checks it, a
// problem goes in the error row in Red and the focus back on the content, the
// submit not done; the message stays until the next check.
func TestAppPopupActionsCheck(t *testing.T) {
	ta := newTestApp(t)
	b := &checkedBody{}
	b.wrong = "Draw at least one cell"
	saved := 0
	ta.Popup(Popup{Title: "Draw", Body: b, Actions: &PopupActions{Submit: func() (tea.Cmd, error) { saved++; return nil, nil }}})
	ta.settle()
	if l := ta.top(); l.h != 3+2+2+4 {
		t.Errorf("height %d: no blank row and error row kept", l.h)
	}
	ta.press("Tab", "Enter")
	lines, _ := covFDraw(ta)
	text := covFText(lines)
	if saved != 0 || b.checked != 1 || ta.top() == nil {
		t.Fatalf("a wrong value: saved %d, checked %d", saved, b.checked)
	}
	if !strings.HasPrefix(text[6], "┃ Draw at least one cell") || !strings.Contains(lines[6], Fg(Red).Render(b.wrong)) {
		t.Errorf("error row %q", text[6])
	}
	if h := covFHint(ta); !strings.Contains(h, "Tab:actions") || !b.focused[len(b.focused)-1] {
		t.Errorf("the focus is not back on the content: %q", h)
	}
	b.wrong = ""
	ta.press("Space")
	if text := covFText(covFScreen(ta)); !strings.Contains(strings.Join(text, "\n"), "Draw at least one cell") {
		t.Error("the error went before the next check")
	}
	ta.press("Tab", "Enter")
	if saved != 1 || ta.top() != nil {
		t.Errorf("right: saved %d", saved)
	}
}

// tdp popup.8 (popup-72), K6, M6: ? lists the body's keys, then the lib's;
// on a button the body's are in the disabled colour, Enter on a button not.
// The buttons' moving keys only with more than one button.
func TestAppPopupActionsKeyRef(t *testing.T) {
	ta := newTestApp(t)
	b := &gridBody{}
	ta.Popup(Popup{Title: "Draw", Body: b, Actions: &PopupActions{Content: "grid"}})
	ta.settle()
	ref := ta.topView().keyRef()
	rows := map[string]KeyRow{}
	for _, s := range ref {
		for _, r := range s.rows {
			rows[r.Key] = r
		}
	}
	for key, disabled := range map[string]bool{"Enter|Space": false, "Enter": true, "Tab|Shift-Tab": false, "Esc": false, "h|l": false} {
		if r, ok := rows[key]; !ok || r.Disabled != disabled {
			t.Errorf("%s: listed %v, disabled %v, want %v", key, ok, r.Disabled, disabled)
		}
	}
	if _, ok := rows["gg|G"]; ok {
		t.Error("moving on one button is listed")
	}
	if !strings.Contains(rows["Tab|Shift-Tab"].Desc, "the grid and the buttons") {
		t.Errorf("Tab %q", rows["Tab|Shift-Tab"].Desc)
	}
	ta.press("Tab")
	for _, s := range ta.topView().keyRef() {
		for _, r := range s.rows {
			if want := r.Key == "Enter|Space" || r.Key == "h|l"; r.Disabled != want {
				t.Errorf("on the button, %s disabled %v", r.Key, r.Disabled)
			}
		}
	}
	ta.press("?")
	if s := strings.Join(covFScreen(ta), "\n"); !strings.Contains(s, "DRAW KEYS") || !strings.Contains(s, "on a button: press it") {
		t.Errorf("the key reference:\n%s", s)
	}
}

// tdp form.7 on a popup of the app's own: a button whose work still runs puts
// the loading icon after the title and has no Enter in the hint; Enter on it
// sends nothing again until it ends.
func TestAppPopupActionsRunning(t *testing.T) {
	ta := newTestApp(t)
	ran := 0
	ta.Popup(Popup{Title: "Draw", Body: &gridBody{}, Actions: &PopupActions{Content: "grid", Buttons: [][]*FormButton{{{Label: "Test",
		Run: func(context.Context) FormResult { ran++; return FormResult{Toast: "Tested"} }}}}}})
	ta.settle()
	ta.press("Tab", "h") // Test
	cmd := ta.press("Enter")
	if h := covEBottom(ta); !strings.HasPrefix(h, "┗━ Tab:grid Esc:cancel ━") || !ta.topView().(*formView).isLoading() {
		t.Errorf("while it runs: hint %q, loading %v", h, ta.topView().(*formView).isLoading())
	}
	if again := ta.press("Enter"); again != nil {
		ta.deliver(again)
	}
	ta.deliver(cmd)
	if ran != 1 {
		t.Errorf("ran %d times", ran)
	}
	if h := covEBottom(ta); !strings.HasPrefix(h, "┗━ Enter:test Tab:grid Esc:cancel ━") {
		t.Errorf("once it ends: %q", h)
	}
}

// longBody is a gridBody that scrolls and has something to lose on quit.
type longBody struct{ gridBody }

func (b *longBody) Scroll() string   { return "2 of 9" }
func (b *longBody) QuitNote() string { return "Changes to kamonu are not saved." }

// The body's scroll position is written on the content only, as a form's on
// its fields (tdp form.7); ? on the content opens the key reference; the
// quit confirm says what it would lose; ClosePopup closes it.
func TestAppPopupActionsScrollAndClose(t *testing.T) {
	ta := newTestApp(t)
	b := &longBody{}
	ta.Popup(Popup{Title: "Draw", Body: b, Actions: &PopupActions{}})
	ta.settle()
	if h := covEBottom(ta); !strings.Contains(h, " 2 of 9 ") {
		t.Errorf("on the content: %q", h)
	}
	ta.press("Tab")
	if h := covEBottom(ta); strings.Contains(h, "of 9") {
		t.Errorf("on the button: %q", h)
	}
	ta.press("Tab", "?")
	if s := strings.Join(covFScreen(ta), "\n"); !strings.Contains(s, "DRAW KEYS") {
		t.Errorf("? on the content:\n%s", s)
	}
	ta.press("Esc", "Ctrl-C")
	if c, ok := ta.topView().(*confirmView); !ok || !slices.Contains(c.ask.Details, b.QuitNote()) {
		t.Errorf("the quit confirm: %T", ta.topView())
	}
	ta.press("Esc")
	ta.ClosePopup(b)
	ta.settle()
	if ta.top() != nil {
		t.Errorf("ClosePopup leaves %d layers", len(ta.stack))
	}
}

// PopupActions.ErrorSubject names the thing in the title of a failure's error
// popup, SAVE FAILED: kamonu, and leaves the button as it is; unset, the
// title has the button's own subject, here none (tdp note.6, popup.1).
func TestAppPopupActionsErrorSubject(t *testing.T) {
	for _, c := range []struct{ subject, title string }{{"kamonu", "SAVE FAILED: kamonu"}, {"", "SAVE FAILED"}} {
		ta := newTestApp(t)
		ta.Popup(Popup{Title: "Draw", Subject: "kamonu", Body: &gridBody{}, Actions: &PopupActions{Button: "Save", ErrorSubject: c.subject,
			Submit: func() (tea.Cmd, error) { return nil, errors.New("disk full") }}})
		ta.settle()
		if row := ta.rowWith("Save"); strings.Contains(row, "Save kamonu") {
			t.Errorf("%q: the button took the subject: %q", c.subject, row)
		}
		ta.press("Tab", "Enter")
		ta.settle()
		n, ok := ta.topView().(*noteView)
		if !ok || n.title != c.title {
			t.Errorf("%q: the error popup %T is titled %q, want %q", c.subject, ta.topView(), n.title, c.title)
		}
	}
}
