package terminu

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// testBody is a popup of the app's own, shaped like a grid to draw on with
// buttons under it: Tab goes between the two.
type testBody struct {
	w, h    int
	drawn   [2]int // the size Draw got last
	keys    []string
	actions bool
	takeEsc bool
	takeQ   bool
	sloppy  bool // rows too wide and too many
	note    string
}

func (b *testBody) QuitNote() string { return b.note }

func (b *testBody) Size(sw, sh int) (int, int) { return b.w, b.h }

func (b *testBody) Draw(w, h int, layer Color) []string {
	b.drawn = [2]int{w, h}
	if b.sloppy {
		return slices.Repeat([]string{Repeat("\u4e2d", w+10)}, h+3)
	}
	var out []string
	for i := range h {
		out = append(out, " "+Fit(fmt.Sprintf("row %d", i), w-2)+" ")
	}
	return out
}

func (b *testBody) Key(key string, _ tea.KeyMsg) (bool, tea.Cmd) {
	b.keys = append(b.keys, key)
	switch key {
	case "Tab":
		b.actions = !b.actions
		return true, nil
	case "Esc":
		return b.takeEsc, nil
	case "q":
		return b.takeQ, nil
	}
	return false, nil
}

func (b *testBody) Hint() []KeyHint {
	if b.actions {
		return []KeyHint{{"Enter", "save"}, {"Tab", "grid"}, {"Esc", "cancel"}}
	}
	return []KeyHint{{"Enter|Space", "cycle"}, {"Tab", "actions"}, {"Esc", "cancel"}}
}

func (b *testBody) Keys() []KeySection {
	return []KeySection{
		{Rows: []KeyRow{{Key: "Enter|Space", Desc: "cycle the cell"}, {Key: "Esc", Desc: "cancel"}}},
		{Title: "move", Rows: []KeyRow{{Key: "h|j|k|l", Desc: "left, down, up, right"}}},
	}
}

func testPopup(b *testBody) Popup {
	return Popup{Title: "Draw", Glyph: "\U000F02C1", Subject: "kamonu", Body: b}
}

// The lib draws the frame, the title with its subject and the hint around
// what the app draws; the app gets the inside from frame line to frame line
// (components/layout/popup).
func TestAppPopupFrame(t *testing.T) {
	ta := newTestApp(t)
	b := &testBody{w: 52, h: 21}
	ta.Popup(testPopup(b))
	ta.settle()
	l := ta.top()
	if l.w != 54 || l.h != 25 {
		t.Fatalf("%d×%d, want 54×25 (the inside, the frame, the blank rows)", l.w, l.h)
	}
	lines, layer := covFDraw(ta)
	text := covFText(lines)
	if b.drawn != [2]int{52, 21} {
		t.Errorf("Draw got %v, want the inside 52×21", b.drawn)
	}
	if !strings.HasPrefix(text[0], "┏━") || !strings.Contains(text[0], "DRAW: kamonu") {
		t.Errorf("title %q", text[0])
	}
	if !strings.Contains(lines[0], Fg(layer).Render("┏")) {
		t.Errorf("the frame is not in the layer colour: %q", lines[0])
	}
	if text[1] != "┃"+Repeat(" ", 52)+"┃" || !strings.HasPrefix(text[2], "┃ row 0 ") || !strings.HasPrefix(text[22], "┃ row 20 ") {
		t.Errorf("blank row, then the app's rows:\n%s", strings.Join(text[:4], "\n"))
	}
	if h := covFHint(ta); !strings.Contains(h, " Enter|Space:cycle Tab:actions Esc:cancel ") {
		t.Errorf("hint %q", h)
	}
	// The hint follows the app's state.
	ta.press("Tab")
	if h := covFHint(ta); !strings.Contains(h, " Enter:save Tab:grid Esc:cancel ") {
		t.Errorf("hint after Tab %q", h)
	}
}

// Rules F7: wide enough for the title and the hint, no bigger than the
// screen allows; Draw gets what there is room for.
func TestAppPopupSize(t *testing.T) {
	ta := newTestApp(t)
	b := &testBody{w: 4, h: 1}
	ta.Popup(Popup{Title: "A title much longer than the inside and the hint", Body: b})
	ta.settle()
	if l := ta.top(); l.w != Width(titleCapsule("", "A TITLE MUCH LONGER THAN THE INSIDE AND THE HINT", Base, 1<<20))+4 || l.h != 5 {
		t.Errorf("%d×%d: not as wide as the title", l.w, l.h)
	}
	ta.press("Esc")
	ta.Popup(Popup{Title: "Draw", Body: b})
	ta.settle()
	if l := ta.top(); l.w != hintWidth(b.Hint())+6 {
		t.Errorf("width %d: not as wide as the hint", l.w)
	}
	ta.press("Esc")
	big := &testBody{w: 300, h: 100}
	ta.resize(40, 12)
	ta.Popup(Popup{Title: "Draw", Body: big})
	ta.settle()
	covFDraw(ta)
	if l := ta.top(); l.w != 38 || l.h != 10 || big.drawn != [2]int{36, 6} {
		t.Errorf("%d×%d, Draw %v: want 38×10 and 36×6 on a 40×12 screen", l.w, l.h, big.drawn)
	}
	// A new terminal size asks Size again.
	big.w = 20
	ta.resize(60, 20)
	if l := ta.top(); l.w != hintWidth(big.Hint())+6 || l.h != 18 {
		t.Errorf("%d×%d after the resize", l.w, l.h)
	}
}

// The app's keys first; Esc, ? and q it leaves are the lib's (Rules K1, K4,
// K6, K9). Ctrl-C never reaches it.
func TestAppPopupKeys(t *testing.T) {
	ta := newTestApp(t)
	b := &testBody{w: 30, h: 5}
	ta.Popup(testPopup(b))
	// Not while it opens (Rules F2).
	ta.Update(keyMsg("j"))
	if len(b.keys) != 0 {
		t.Fatalf("a key while opening: %v", b.keys)
	}
	ta.settle()
	ta.press("j", "1", "Space")
	if !slices.Equal(b.keys, []string{"j", "1", "Space"}) || ta.Screen().focus != 0 {
		t.Errorf("keys %v, panel focus %d: every key goes to the popup first", b.keys, ta.Screen().focus)
	}
	ta.press("?")
	ref := strings.Join(covFScreen(ta), "\n")
	for _, s := range []string{"DRAW KEYS", "cycle the cell", "MOVE", "left, down, up, right"} {
		if !strings.Contains(ref, s) {
			t.Errorf("the key reference has no %q:\n%s", s, ref)
		}
	}
	ta.press("Esc")
	if _, ok := ta.topView().(*appPopup); !ok {
		t.Fatal("Esc on the key reference closes only it")
	}
	b.takeQ = true
	ta.press("q")
	if _, ok := ta.topView().(*appPopup); !ok || ta.quitting {
		t.Error("q the app takes does nothing")
	}
	b.takeQ = false
	ta.press("q")
	if !ta.quitting {
		t.Error("q the app leaves starts the quit flow")
	}
	ta.press("Esc")
	b.keys = nil
	ta.press("Ctrl-C")
	if len(b.keys) != 0 || !ta.quitting {
		t.Errorf("Ctrl-C reached the app (%v) or did not start the quit flow", b.keys)
	}
	ta.press("Esc")
	b.takeEsc = true
	ta.press("Esc")
	if _, ok := ta.topView().(*appPopup); !ok {
		t.Fatal("Esc the app takes does not close it")
	}
	b.takeEsc = false
	ta.press("Esc")
	if ta.top() != nil {
		t.Error("Esc the app leaves closes it")
	}
}

// The lib's popups go over it, one layer at a time; ClosePopup closes it and
// what is over it (Rules K4).
func TestAppPopupStack(t *testing.T) {
	ta := newTestApp(t)
	b := &testBody{w: 30, h: 5}
	ta.Popup(testPopup(b))
	ta.settle()
	ta.Confirm(Ask{Title: "Discard", Question: "Discard changes to kamonu?", Verb: "discard"}, func() tea.Cmd { return ta.ClosePopup(b) })
	ta.settle()
	ta.press("Esc")
	if _, ok := ta.topView().(*appPopup); !ok || len(b.keys) != 0 {
		t.Fatal("Esc closes the confirm over it, and nothing reaches the app")
	}
	ta.Confirm(Ask{Title: "Discard", Question: "Discard changes to kamonu?", Verb: "discard"}, func() tea.Cmd { return ta.ClosePopup(b) })
	ta.settle()
	ta.press("Enter")
	if ta.top() != nil {
		t.Errorf("the confirm's accept closes the popup: %d layers", len(ta.stack))
	}
	ta.Popup(testPopup(b))
	ta.settle()
	ta.Text(TextInput{Title: "Save as"})
	ta.settle()
	if cmd := ta.ClosePopup(b); cmd == nil {
		t.Fatal("ClosePopup did nothing")
	}
	ta.settle()
	if ta.top() != nil {
		t.Errorf("ClosePopup leaves %d layers", len(ta.stack))
	}
	if ta.ClosePopup(b) != nil {
		t.Error("ClosePopup on a popup that is not open")
	}
}

type sliceBody []string

func (sliceBody) Size(int, int) (int, int)               { return 1, 1 }
func (sliceBody) Draw(int, int, Color) []string          { return nil }
func (sliceBody) Key(string, tea.KeyMsg) (bool, tea.Cmd) { return false, nil }
func (sliceBody) Hint() []KeyHint                        { return nil }
func (sliceBody) Keys() []KeySection                     { return nil }

// ClosePopup finds the popup by its Body: one that cannot be compared is a
// mistake in the program.
func TestAppPopupComparable(t *testing.T) {
	for name, body := range map[string]PopupBody{"nil": nil, "slice": sliceBody{"x"}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: no panic", name)
				}
			}()
			newTestApp(t).Popup(Popup{Title: "Draw", Body: body})
		}()
	}
}

// A form's submit that opens the next popup: only the form closes, and the
// new popup is the first layer from its first frame (Rules F8).
func TestAppPopupFromFormSubmit(t *testing.T) {
	ta := newTestApp(t)
	b := &testBody{w: 30, h: 5}
	ta.Form(&Form{Title: "New kamon", Button: "Create", Fields: []*FormField{{Label: "Name", Value: func() string { return "kbu" }}},
		Submit: func() (tea.Cmd, error) { return ta.Popup(testPopup(b)), nil }})
	ta.settle()
	ta.press("Tab")
	ta.Update(keyMsg("Enter"))
	for range openFrames - 1 {
		ta.Update(animMsg{})
	}
	if len(ta.stack) != 2 || !ta.stack[0].closing || ta.stack[1].view.(*appPopup).p.Body != b {
		t.Fatalf("the form closing under the new popup: %d layers", len(ta.stack))
	}
	if row := ta.rowWith("DRAW: kamonu"); !strings.Contains(row, Fg(LayerColor(1)).Render("┏")) {
		t.Errorf("the new popup is not in the first layer's colour: %q", row)
	}
	ta.settle()
	if len(ta.stack) != 1 || ta.top().view.(*appPopup).p.Body != b {
		t.Errorf("after the animation only the new popup is left: %d layers", len(ta.stack))
	}
}

// What a popup of the app's own would lose goes last in the quit confirm, in
// Peach (Rules K9: the quit flow is the app's).
func TestAppPopupQuitNote(t *testing.T) {
	ta := newTestApp(t)
	ta.Quit.Details = []string{"3 jobs are running."}
	b := &testBody{w: 30, h: 5}
	ta.Popup(testPopup(b))
	ta.settle()
	ta.press("Ctrl-C")
	if c, ok := ta.topView().(*confirmView); !ok || len(c.ask.Details) != 1 {
		t.Fatalf("no note, the app's quit confirm as it is: %T", ta.topView())
	}
	ta.press("Esc")
	b.note = "Changes to kamonu are not saved."
	ta.press("Ctrl-C")
	c := ta.topView().(*confirmView)
	if !slices.Equal(c.ask.Details, []string{"3 jobs are running.", "", b.note}) || !slices.Equal(c.ask.Warn, []int{2}) {
		t.Errorf("details %q, warn %v", c.ask.Details, c.ask.Warn)
	}
	if row := ta.rowWith(b.note); !strings.Contains(row, Fg(Peach).Render(b.note)) {
		t.Errorf("the note is not in Peach: %q", row)
	}
	if len(ta.Quit.Details) != 1 {
		t.Errorf("App.Quit changed: %q", ta.Quit.Details)
	}
	ta.press("Esc")
	// With no quit confirm of the app's, a note still asks first.
	ta.Quit = nil
	ta.press("Ctrl-C")
	if c, ok := ta.topView().(*confirmView); !ok || c.ask.Question != "Quit test?" || !ta.quitting {
		t.Fatalf("a note with Quit nil: %T", ta.topView())
	}
	if ta.press("Ctrl-C") == nil {
		t.Error("the second Ctrl-C does not leave")
	}
}

// The quit confirm closed with the popup under it ends the quit flow: the
// next Ctrl-C asks again (Rules K9).
func TestK9QuitConfirmClosedUnder(t *testing.T) {
	ta := newTestApp(t)
	b := &testBody{w: 30, h: 5}
	ta.Popup(testPopup(b))
	ta.settle()
	ta.press("Ctrl-C")
	ta.ClosePopup(b)
	ta.settle()
	if ta.top() != nil || ta.quitting {
		t.Fatalf("%d layers, quitting %v", len(ta.stack), ta.quitting)
	}
	ta.press("Ctrl-C")
	if _, ok := ta.topView().(*confirmView); !ok {
		t.Error("Ctrl-C does not ask again")
	}
}

// scrollBody is a test body that scrolls: it says where it is.
type scrollBody struct {
	testBody
	at string
}

func (b *scrollBody) Scroll() string { return b.at }

// TestAppPopupScroll: an app popup whose body scrolls has its scroll position
// on the right of the bottom frame line, in the layer colour, and room for it
// with the whole hint (components/layout/popup; kamonu KM9).
func TestAppPopupScroll(t *testing.T) {
	ta := newTestApp(t)
	b := &scrollBody{testBody: testBody{w: 10, h: 3}, at: "3 of 40"}
	ta.Popup(Popup{Title: "Draw", Body: b})
	ta.settle()
	l := ta.top()
	bottom := StripStyles(ta.Frame()[l.y+l.h-1])
	if !strings.Contains(bottom, "Esc:cancel") || !strings.Contains(bottom, " 3 of 40 ━┛") {
		t.Errorf("bottom frame line %q", bottom)
	}
	b.at = ""
	if bottom := StripStyles(ta.Frame()[l.y+l.h-1]); strings.Contains(bottom, "of 40") {
		t.Errorf("everything fits, still a scroll position: %q", bottom)
	}
}

// TestCloseAll: the app can close the whole stack, the lib's popups among it
// (Rules T1; kbu KB10).
func TestCloseAll(t *testing.T) {
	ta := newTestApp(t)
	ta.Viewer(testViewer())
	ta.Popup(Popup{Title: "Draw", Body: &testBody{w: 10, h: 3}})
	ta.Text(TextInput{Title: "Name", OnCancel: ta.record("cancel")})
	ta.settle()
	ta.CloseAll()
	ta.settle()
	if ta.top() != nil || len(ta.stack) != 0 || !slices.Equal(ta.ran, []string{"cancel"}) {
		t.Errorf("%d popups left, ran %v", len(ta.stack), ta.ran)
	}
}

// TestCanvas: a screen the app draws itself, whole, has the lib's popups and
// toasts over it and no footer; its keys are the app's, the screens under it
// get none, and only Ctrl-C is left to the quit flow (Rules E8, K9; locku
// LK1).
func TestCanvas(t *testing.T) {
	ta := newTestApp(t)
	ta.Quit = &Ask{Question: "Quit test?", Verb: "quit"}
	var keys []string
	type tick struct{}
	ta.Canvas = func(w, h int) []string { return []string{"", Center("LOCKED", w)} }
	ta.CanvasKey = func(key string, _ tea.KeyMsg) (bool, tea.Cmd) {
		switch key {
		case "Ctrl-C", "q", "x":
			return false, nil // left to the lib
		case "Enter":
			keys = append(keys, key)
			return true, ta.Password(PasswordInput{Title: "Unlock", Unlock: true, MaxLen: 4, Done: func(v string, _ bool) tea.Cmd {
				keys = append(keys, "pin "+v)
				return nil
			}})
		}
		keys = append(keys, key)
		return true, func() tea.Msg { return tick{} }
	}
	ta.resize(40, 10)
	lines := ta.Frame()
	if len(lines) != 10 || !strings.Contains(StripStyles(lines[1]), "LOCKED") || strings.Contains(ta.screenText(), "quit") {
		t.Fatalf("the canvas:\n%s", ta.screenText())
	}
	for i, l := range lines {
		if Width(l) != 40 {
			t.Errorf("row %d is %d wide", i, Width(l))
		}
	}
	if !batchHas(ta.press("t"), tick{}) {
		t.Error("the command of a key the canvas took did not come back")
	}
	ta.press("q", "x", "?", "Space")
	if ta.quitting || ta.top() != nil || len(ta.ran) != 0 {
		t.Fatalf("a key the canvas left did something: quitting %v, %T open, ran %v", ta.quitting, ta.topView(), ta.ran)
	}
	ta.press("Enter")
	ta.settle()
	if _, ok := ta.topView().(*lineInput); !ok {
		t.Fatalf("no PIN popup over the canvas: %T", ta.topView())
	}
	ta.press("1", "2", "3", "4", "Enter")
	ta.settle()
	if !slices.Equal(keys, []string{"t", "?", "Space", "Enter", "pin 1234"}) || ta.top() != nil {
		t.Errorf("the canvas and its PIN popup got %v", keys)
	}
	ta.Toast(Info, "Unlocked")
	ta.settle()
	if !strings.Contains(ta.screenText(), "Unlocked") {
		t.Errorf("no toast over the canvas:\n%s", ta.screenText())
	}
	ta.press("Ctrl-C")
	if !ta.quitting {
		t.Error("Ctrl-C the canvas left: no quit flow")
	}
}

// batchHas reports whether cmd, or a command it batches, gives msg.
func batchHas(cmd tea.Cmd, msg tea.Msg) bool {
	if cmd == nil {
		return false
	}
	switch got := cmd().(type) {
	case tea.BatchMsg:
		for _, c := range got {
			if batchHas(c, msg) {
				return true
			}
		}
		return false
	default:
		return got == msg
	}
}
