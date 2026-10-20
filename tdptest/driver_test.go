package tdptest

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	terminu "github.com/vulcanshen/terminu-go"
	"github.com/vulcanshen/terminu-go/internal/testhook"
)

// fakeT records what a driver reports, so a test can check it; Fatalf ends
// the call the way testing.T's does, by unwinding to catch.
type fakeT struct {
	*testing.T
	errs, fatals []string
}

type fatal struct{}

func (f *fakeT) Helper() {}
func (f *fakeT) Errorf(format string, args ...any) {
	f.errs = append(f.errs, fmt.Sprintf(format, args...))
}
func (f *fakeT) Fatalf(format string, args ...any) {
	f.fatals = append(f.fatals, fmt.Sprintf(format, args...))
	panic(fatal{})
}

// catch runs f and reports whether it stopped on Fatalf.
func catch(f func()) (stopped bool) {
	defer func() {
		if v := recover(); v != nil {
			if _, ok := v.(fatal); !ok {
				panic(v)
			}
			stopped = true
		}
	}()
	f()
	return false
}

type gotMsg struct{ what string }

// tiny is a small app: one list, an action that asks first, a global one
// that shows a toast; what came to OnMsg is in got.
type tiny struct {
	*terminu.App
	list *terminu.List
	got  []string
}

func newTiny() *tiny {
	ty := &tiny{App: terminu.New("tiny")}
	a := ty.App
	a.MinWidth, a.MinHeight = 30, 12
	a.Splash = func(frame, w, h int) ([]string, bool) { return []string{"tiny"}, true }
	ty.list = &terminu.List{Columns: []terminu.Column{{Title: "Name", Shrink: true}}}
	var rows []terminu.Row
	for i := range 20 {
		rows = append(rows, terminu.Row{ID: fmt.Sprint(i), Cells: []string{fmt.Sprintf("item-%02d", i)}})
	}
	ty.list.SetRows(rows)
	a.AddScreen(&terminu.Screen{Name: "Items", Panels: []*terminu.Panel{{Label: "Items", Body: ty.list, Actions: []terminu.Action{
		{Key: "x", Label: "Delete", Desc: "this item", Scope: terminu.ItemScope,
			Ask: &terminu.Ask{Title: "Delete", Question: "Delete it?", Verb: "delete"},
			Do:  func() tea.Cmd { return a.Toast(terminu.Success, "Deleted") }},
	}}}})
	a.Global = []terminu.Action{{Key: "S", Label: "Say", Desc: "a toast", Scope: terminu.GlobalScope,
		Do: func() tea.Cmd { return a.Toast(terminu.Info, "Hello") }}}
	a.OnMsg = func(msg tea.Msg) tea.Cmd {
		if m, ok := msg.(gotMsg); ok {
			ty.got = append(ty.got, m.what)
		}
		return nil
	}
	return ty
}

func say(what string) tea.Cmd { return func() tea.Msg { return gotMsg{what} } }

func TestStartRunsInitThenSize(t *testing.T) {
	ty := newTiny()
	sized := false
	ty.Start = func() tea.Cmd {
		sized = ty.App.Screen() != nil && func() bool { w, _ := ty.Size(); return w > 0 }()
		return ty.Toast(terminu.Info, "Welcome")
	}
	d := Start(t, ty.App, 80, 24)
	if sized {
		t.Error("Init ran after the size came: the program runs Init first")
	}
	if !d.Has("Welcome") {
		t.Error("what Start started is not on the screen when Start returns")
	}
	if testhook.Animating(ty.App) {
		t.Error("the toast is still opening")
	}
}

func TestAdvanceMovesTheTestClock(t *testing.T) {
	ty := newTiny()
	d := Start(t, ty.App, 80, 24)
	d.Press("S")
	before := d.Now()
	time.Sleep(10 * time.Millisecond)
	if !d.Has("Hello") {
		t.Fatal("no toast")
	}
	d.Advance(time.Second)
	if !d.Has("Hello") {
		t.Error("the toast went before its time")
	}
	d.Advance(10 * time.Second)
	if d.Has("Hello") {
		t.Error("the toast stays after its time")
	}
	if got := d.Now().Sub(before); got < 11*time.Second || got > 12*time.Second {
		t.Errorf("the clock moved %v, want 11s and the closing", got)
	}
}

func TestAppTickWaitsForTheTestClock(t *testing.T) {
	ty := newTiny()
	ty.Start = func() tea.Cmd {
		return tea.Batch(ty.Tick(time.Second, func(time.Time) tea.Msg { return gotMsg{"late"} }), say("now"))
	}
	d := Start(t, ty.App, 80, 24)
	if strings.Join(ty.got, ",") != "now" {
		t.Fatalf("got %v at the start", ty.got)
	}
	d.Advance(999 * time.Millisecond)
	if len(ty.got) != 1 {
		t.Error("App.Tick fired early")
	}
	d.Advance(time.Millisecond)
	if strings.Join(ty.got, ",") != "now,late" {
		t.Errorf("App.Tick did not fire on time: %v", ty.got)
	}
}

func TestAutoOffStepsTheOpening(t *testing.T) {
	ty := newTiny()
	d := Start(t, ty.App, 80, 24)
	d.Auto = false
	d.Press("Space")
	if !testhook.Animating(ty.App) {
		t.Fatal("with Auto off the Space menu should still be opening")
	}
	d.Press("Esc") // tdp F2: no keys while it opens
	frames := 0
	for testhook.Animating(ty.App) && d.Step() {
		frames++
	}
	if frames != 8 {
		t.Errorf("the opening took %d more frames, want 8", frames)
	}
	if testhook.Layers(ty.App) != 1 {
		t.Fatal("Esc closed the menu while it was opening")
	}
	d.Auto = true
	d.Press("Esc")
	if testhook.Layers(ty.App) != 0 {
		t.Error("Esc did not close the menu")
	}
}

func TestSequenceRunsInOrder(t *testing.T) {
	ty := newTiny()
	d := Start(t, ty.App, 80, 24)
	slow := func() tea.Msg { time.Sleep(20 * time.Millisecond); return gotMsg{"a"} }
	d.Do(func() tea.Cmd {
		return tea.Sequence(slow, ty.Tick(time.Second, func(time.Time) tea.Msg { return gotMsg{"b"} }), say("c"))
	})
	if strings.Join(ty.got, ",") != "a" {
		t.Fatalf("got %v: the sequence did not wait for its timer", ty.got)
	}
	d.Advance(time.Second)
	if strings.Join(ty.got, ",") != "a,b,c" {
		t.Errorf("got %v", ty.got)
	}
	d.Do(func() tea.Cmd { return tea.Batch(say("d"), say("e"), say("f")) })
	if strings.Join(ty.got[3:], ",") != "d,e,f" {
		t.Errorf("a batch's messages come in the order they were started: %v", ty.got[3:])
	}
}

func TestWaitForASlowCommand(t *testing.T) {
	ty := newTiny()
	d := Start(t, ty.App, 80, 24)
	d.Wait = 10 * time.Millisecond
	d.Do(func() tea.Cmd {
		return func() tea.Msg { time.Sleep(100 * time.Millisecond); return gotMsg{"slow"} }
	})
	if len(ty.got) != 0 {
		t.Fatal("the input waited longer than Wait")
	}
	d.WaitFor(func() bool { return len(ty.got) == 1 }, 2*time.Second)
	ft := &fakeT{T: t}
	d2 := Start(ft, newTiny().App, 80, 24)
	if !catch(func() { d2.WaitFor(func() bool { return false }, 20*time.Millisecond) }) {
		t.Error("WaitFor did not fail the test on its timeout")
	}
}

func TestQuitAndAfter(t *testing.T) {
	ty := newTiny()
	ty.Quit = &terminu.Ask{Title: "Quit", Question: "Quit tiny?", Verb: "quit"}
	d := Start(t, ty.App, 80, 24)
	d.Press("q")
	if d.Quit() || !d.Has("Quit tiny?") {
		t.Fatal("q quit before the confirm")
	}
	d.Press("Enter")
	if !d.Quit() {
		t.Fatal("the confirm did not quit")
	}
	ft := &fakeT{T: t}
	d2 := Start(ft, newTiny().App, 80, 24)
	d2.Press("q")
	if !d2.Quit() {
		t.Fatal("q did not quit an app with no Quit")
	}
	if !catch(func() { d2.Press("j") }) || !strings.Contains(ft.fatals[0], "after the app quit") {
		t.Errorf("a key after quitting: %v", ft.fatals)
	}
}

func TestL4ReportedWithWhere(t *testing.T) {
	ft := &fakeT{T: t}
	ty := newTiny()
	wide := false
	ty.Canvas = func(w, h int) []string {
		rows := make([]string, h)
		for i := range rows {
			rows[i] = strings.Repeat(" ", w)
		}
		if wide {
			rows[2] = "\a" + rows[2] // a bell: drawn as nothing, sent to the terminal
		}
		return rows
	}
	ty.CanvasKey = func(key string, msg tea.KeyMsg) (bool, tea.Cmd) { wide = key == "w"; return true, nil }
	d := Start(ft, ty.App, 40, 12)
	if len(ft.errs) != 0 {
		t.Fatalf("a frame that is right was reported: %v", ft.errs)
	}
	d.Note("Tricky[8]")
	d.Press("w")
	if len(ft.errs) != 1 || !strings.Contains(ft.errs[0], `row 3 sends '\a' to the terminal: "\a      `) ||
		!strings.Contains(ft.errs[0], "after Start(40×12) w") || !strings.Contains(ft.errs[0], "with Tricky[8]") {
		t.Errorf("reports: %q", ft.errs)
	}
	for range 5 {
		d.Press("w")
	}
	if len(ft.errs) != 4 {
		t.Errorf("reported %d times, want 3 and the note", len(ft.errs))
	}
	if p := rowsProblem([]string{"abc", "a\tb"}, 3, 2); p != "row 2 is 6 cells wide on a terminal 3 wide" {
		t.Errorf("a row too wide: %q", p)
	}
	if p := rowsProblem([]string{"a\tb"}, 6, 1); !strings.Contains(p, `sends '\t'`) {
		t.Errorf("a tab that reaches the terminal: %q", p)
	}
	if p := rowsProblem([]string{"\x1b[1;38;2;1:2:3mabc\x1b[0m"}, 3, 1); p != "" {
		t.Errorf("SGR is not a problem: %q", p)
	}
	if p := rowsProblem([]string{"abc"}, 3, 2); p != "1 rows on a terminal 2 rows high" {
		t.Errorf("rows: %q", p)
	}
}

func TestClipboardTaken(t *testing.T) {
	ty := newTiny()
	d := Start(t, ty.App, 80, 24)
	d.Do(func() tea.Cmd { terminu.Clipboard("copied text"); return nil })
	if got := d.Clipboard(); len(got) != 1 || got[0] != "copied text" {
		t.Errorf("clipboard: %q", got)
	}
}

func TestAPanickingCommandFails(t *testing.T) {
	ft := &fakeT{T: t}
	d := Start(ft, newTiny().App, 80, 24)
	if !catch(func() { d.Do(func() tea.Cmd { return func() tea.Msg { panic("boom") } }) }) ||
		!strings.Contains(ft.fatals[0], "a command panicked: boom") {
		t.Errorf("fatals: %q", ft.fatals)
	}
}

func TestStartRefusesWhatRunRefuses(t *testing.T) {
	ft := &fakeT{T: t}
	ty := newTiny()
	ty.Splash = nil
	if !catch(func() { Start(ft, ty.App, 80, 24) }) || !strings.Contains(ft.fatals[0], "no Splash") {
		t.Errorf("fatals: %q", ft.fatals)
	}
}

func TestKeyNames(t *testing.T) {
	names := []string{"q", "G", "?", "/", "3", "Ctrl-C", "Ctrl-W", "Alt-Esc", "Alt-t", "Shift-Tab"}
	for n := range keyTypes {
		names = append(names, n)
	}
	for _, n := range names {
		if _, ok := keyMsg(n); !ok {
			t.Errorf("%q is not taken", n)
		}
	}
	for _, n := range []string{"", "gg", "Home2", "Ctrl-I", "space", " "} {
		if _, ok := keyMsg(n); ok {
			t.Errorf("%q is taken as a key", n)
		}
	}
	var got []string
	for _, k := range burst("ab c\td") {
		got = append(got, fmt.Sprintf("%s:%d", terminu.KeyName(k), len(k.Runes)))
	}
	if strings.Join(got, " ") != "ab:2 Space:1 c:1 Tab:0 d:1" {
		t.Errorf("burst: %v", got)
	}
}

func TestTypeIsNotPress(t *testing.T) {
	ty := newTiny()
	d := Start(t, ty.App, 80, 24)
	var got string
	d.Do(func() tea.Cmd {
		return ty.Text(terminu.TextInput{Title: "Rename", Prompt: "New name", Verb: "rename",
			Done: func(v string) tea.Cmd { got = v; return nil }})
	})
	d.Type("Home")
	d.Press("Home")
	d.Type("x")
	d.Press("End", "Enter")
	if got != "xHome" {
		t.Errorf("Type typed and Press pressed: %q", got)
	}
}

func TestEachIconWidth(t *testing.T) {
	var seen []string
	EachIconWidth(t, func(t *testing.T) {
		seen = append(seen, fmt.Sprintf("%d/%d", terminu.IconWidth(), terminu.PowerlineWidth()))
		Start(t, newTiny().App, 60, 20).Press("Space")
	})
	if strings.Join(seen, " ") != "1/1 2/1 2/2" {
		t.Errorf("widths: %v", seen)
	}
	if terminu.IconWidth() != 1 || terminu.PowerlineWidth() != 1 {
		t.Error("the widths were not put back")
	}
}
