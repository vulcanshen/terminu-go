// Package tdptest drives a terminu App in a Go test the way a user does —
// keys, a paste, the terminal changing size, time going by — and checks,
// after every message, what tdp asks of every screen: each row exactly as
// wide as the terminal (tdp L4).
//
// The App runs on the test's clock: its animations, a toast's stay and the
// loading icon move only when the test moves the time (Advance, Step), and
// by default each input plays the animations to their end (Driver.Auto).
// The app's own timers follow the test's clock when they are made with
// App.Tick; one made with tea.Tick runs on the real clock and comes back
// whenever it is due. The App's own commands run as they do in the
// program; one that takes long comes back with a later input, or WaitFor.
//
// The lib's widths are the whole process's: a test that drives an App does
// not run in parallel with another (EachIconWidth sets them).
package tdptest

import (
	"fmt"
	"reflect"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"

	terminu "github.com/vulcanshen/terminu-go"
	"github.com/vulcanshen/terminu-go/internal/testhook"
)

// Driver runs one App in a test.
type Driver struct {
	// Auto plays the animations to their end after each input, as a user
	// who waits a moment sees them; on from Start. Off, the time stands
	// still until Step or Advance moves it: a popup half open, a key
	// pressed while it opens (tdp F2).
	Auto bool
	// Wait is how long an input waits for the commands it started, real
	// I/O included, before the test goes on: 200ms from Start. A command
	// still running then comes back with a later input, or WaitFor.
	Wait time.Duration

	t      testing.TB
	app    *terminu.App
	w, h   int
	mu     sync.Mutex // guards now: the App reads it from any goroutine (App.Log)
	now    time.Time
	timers []timer
	seq    int
	out    map[int]bool // the commands running
	fresh  []int        // the ones started since the last wait
	ready  []back
	back   chan back
	stop   chan struct{}
	quit   bool
	trail  []string
	notes  []string
	copied []string
	broke  int
}

// back is what a command gave back; rest is the sequence it was in, still
// to run.
type back struct {
	seq  int
	msg  tea.Msg
	rest []tea.Cmd
}

type timer struct {
	at   time.Time
	seq  int
	fn   func(time.Time) tea.Msg
	rest []tea.Cmd
}

// panicked is a command that panicked, for the test to report.
type panicked struct {
	value any
	stack []byte
}

// spinTurn is the loading icon's eight frames: the clock starts on a whole
// turn, so the icon's frames are the same on every run.
const spinTurn = 8 * 90

// Start runs app on a w × h terminal as the program does: Init, a frame
// before the size is known, then the size; what Init started (the app's
// Start, a first toast) has run and the animations have played when it
// returns. It fails the test when Run would refuse the app (no Splash, a
// key the app may not take). When the test ends, the programs in the app's
// terminals are stopped.
func Start(t testing.TB, app *terminu.App, w, h int) *Driver {
	t.Helper()
	d := &Driver{
		Auto: true, Wait: 200 * time.Millisecond,
		t: t, app: app, w: w, h: h,
		now: time.UnixMilli(time.Now().UnixMilli() / spinTurn * spinTurn),
		out: map[int]bool{}, back: make(chan back), stop: make(chan struct{}),
	}
	if err := testhook.Prepare(app); err != nil {
		t.Fatalf("tdptest: Run would refuse the app: %v", err)
	}
	testhook.UseClock(app, d)
	t.Cleanup(func() {
		close(d.stop)
		testhook.Stop(app)
	})
	d.note(fmt.Sprintf("Start(%d×%d)", w, h))
	cmd := app.Init()
	app.View()
	d.launch(cmd, nil)
	d.update(tea.WindowSizeMsg{Width: w, Height: h})
	d.settle()
	return d
}

// Note adds what a failure is to name besides the keys pressed: which of
// Tricky the app was given, say.
func (d *Driver) Note(text string) { d.notes = append(d.notes, text) }

// Now is the test's time, the App's clock.
func (d *Driver) Now() time.Time {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.now
}

// setNow moves the test's time to at.
func (d *Driver) setNow(at time.Time) {
	d.mu.Lock()
	d.now = at
	d.mu.Unlock()
}

// Press presses keys one after another, each named as tdp writes it (tdp
// M5, terminu.KeyName): "j", "Enter", "Esc", "Space", "Ctrl-C", "Alt-Esc",
// "Shift-Tab", "↓". Text goes in with Type, Burst or Paste: Press("Home")
// is the Home key.
func (d *Driver) Press(keys ...string) {
	d.t.Helper()
	for _, k := range keys {
		msg, ok := keyMsg(k)
		if !ok {
			d.t.Fatalf("tdptest: Press(%q): not a key as tdp names it; text goes in with Type", k)
		}
		d.input(k, msg)
	}
}

// Type types text a key at a time, as a terminal sends it: a control
// character is its own key (\t Tab, \r Enter, \x1b Esc, \x7f Backspace).
func (d *Driver) Type(text string) {
	d.t.Helper()
	d.begin(fmt.Sprintf("Type(%q)", text))
	for _, r := range text {
		d.update(typed(r))
		d.settle()
	}
}

// Burst sends text the way it comes when it arrives all at once (a key
// held down, a fast typist, a barcode reader): the characters together in
// one key, as Bubble Tea gives them.
func (d *Driver) Burst(text string) {
	d.t.Helper()
	d.begin(fmt.Sprintf("Burst(%q)", text))
	for _, k := range burst(text) {
		d.update(k)
	}
	d.settle()
}

// Paste pastes text: one key, marked as a paste.
func (d *Driver) Paste(text string) {
	d.t.Helper()
	d.input(fmt.Sprintf("Paste(%q)", text), tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text), Paste: true})
}

// Resize changes the terminal's size.
func (d *Driver) Resize(w, h int) {
	d.t.Helper()
	d.w, d.h = w, h
	d.input(fmt.Sprintf("Resize(%d×%d)", w, h), tea.WindowSizeMsg{Width: w, Height: h})
}

// Send sends a message of the app's own, as a command of it would.
func (d *Driver) Send(msg tea.Msg) {
	d.t.Helper()
	d.begin(fmt.Sprintf("Send(%T)", msg))
	d.deliver(back{msg: msg})
	d.settle()
}

// Do runs f on the update loop, where the App may be changed (app.Focus,
// app.ShowScreen, app.Toast), and runs the command it gives back.
func (d *Driver) Do(f func() tea.Cmd) {
	d.t.Helper()
	d.begin("Do")
	cmd := f()
	d.check()
	d.launch(cmd, nil)
	d.settle()
}

// Advance moves the time on by dur: the timers due in it fire in order (a
// toast going, a debounce, the loading icon turning), then the animations
// play to their end when Auto is on.
func (d *Driver) Advance(dur time.Duration) {
	d.t.Helper()
	d.begin(fmt.Sprintf("Advance(%v)", dur))
	end := d.Now().Add(dur)
	d.pump()
	for len(d.timers) > 0 && !d.timers[0].at.After(end) {
		d.fire()
	}
	d.setNow(end)
	d.settle()
}

// Step moves the time on to the next timer and fires it alone — one frame
// of an animation, say — whatever Auto says. It reports false when no
// timer is waiting.
func (d *Driver) Step() bool {
	d.t.Helper()
	d.note("Step")
	d.pump()
	if len(d.timers) == 0 {
		return false
	}
	d.fire()
	return true
}

// Settle plays the animations to their end now, as an input does with Auto.
func (d *Driver) Settle() {
	d.t.Helper()
	auto := d.Auto
	d.Auto = true
	d.settle()
	d.Auto = auto
}

// WaitFor waits, on the real clock, up to timeout for cond to hold, taking
// what the App's commands give back as it comes: a file read, a PTY's
// output. It fails the test when cond does not come to hold.
func (d *Driver) WaitFor(cond func() bool, timeout time.Duration) {
	d.t.Helper()
	d.settle()
	limit := time.NewTimer(timeout)
	defer limit.Stop()
	for !cond() {
		select {
		case b := <-d.back:
			d.took(b)
			d.pump()
			d.settle()
		case <-limit.C:
			d.t.Fatalf("tdptest: waited %v and it never came%s", timeout, d.where())
		}
	}
}

// Quit reports whether the App has asked to quit (tea.Quit). Nothing more
// goes to it after that.
func (d *Driver) Quit() bool { return d.quit }

// Screen is the screen as it is now, a row a line, without the styles.
func (d *Driver) Screen() []string {
	lines := d.app.Frame()
	for i, l := range lines {
		lines[i] = terminu.StripStyles(l)
	}
	return lines
}

// Styled is the screen as it is now with its styles: the SGR sequences the
// lib writes, as the terminal gets them.
func (d *Driver) Styled() []string { return d.app.Frame() }

// Has reports whether text is on the screen now, within a row.
func (d *Driver) Has(text string) bool {
	return slices.ContainsFunc(d.Screen(), func(l string) bool { return strings.Contains(l, text) })
}

// Clipboard is what went to the terminal's clipboard so far, oldest first
// (a viewer's selection, terminu.Clipboard): taken here, never sent.
func (d *Driver) Clipboard() []string { return d.copied }

// --- running the App ---------------------------------------------------------

func (d *Driver) input(what string, msg tea.Msg) {
	d.t.Helper()
	d.begin(what)
	d.update(msg)
	d.settle()
}

func (d *Driver) begin(what string) {
	d.t.Helper()
	if d.quit {
		d.t.Fatalf("tdptest: %s after the app quit%s", what, d.where())
	}
	d.note(what)
}

func (d *Driver) note(what string) {
	d.trail = append(d.trail, what)
	if len(d.trail) > 24 {
		d.trail = d.trail[len(d.trail)-24:]
	}
}

func (d *Driver) update(msg tea.Msg) {
	if d.quit {
		return
	}
	_, cmd := d.app.Update(msg)
	d.check()
	d.launch(cmd, nil)
}

// launch runs cmd as the program does, on its own; rest is the sequence it
// is in, run once it is back.
func (d *Driver) launch(cmd tea.Cmd, rest []tea.Cmd) {
	if cmd == nil {
		if len(rest) > 0 {
			d.launch(rest[0], rest[1:])
		}
		return
	}
	d.seq++
	n := d.seq
	if instant()[reflect.ValueOf(cmd).Pointer()] {
		d.ready = append(d.ready, back{seq: n, msg: cmd(), rest: rest})
		return
	}
	d.out[n] = true
	d.fresh = append(d.fresh, n)
	go func() {
		var msg tea.Msg
		func() {
			defer func() {
				if v := recover(); v != nil {
					msg = panicked{v, debug.Stack()}
				}
			}()
			msg = cmd()
		}()
		select {
		case d.back <- back{seq: n, msg: msg, rest: rest}:
		case <-d.stop:
		}
	}()
}

// instant are the commands known to come back at once, run where they are
// started rather than on their own: a timer on the test clock, the ones
// tea.Batch and tea.Sequence make, tea.Quit.
var instant = sync.OnceValue(func() map[uintptr]bool {
	nop := func() tea.Msg { return nil }
	a := terminu.New("")
	testhook.UseClock(a, zeroClock{})
	m := map[uintptr]bool{}
	for _, c := range []tea.Cmd{a.Tick(0, nil), tea.Batch(nop, nop), tea.Sequence(nop, nop), tea.Quit} {
		m[reflect.ValueOf(c).Pointer()] = true
	}
	return m
})

type zeroClock struct{}

func (zeroClock) Now() time.Time { return time.Time{} }

func (d *Driver) took(b back) {
	delete(d.out, b.seq)
	d.ready = append(d.ready, b)
}

// pump hands the App what its commands give back, in the order they were
// started, until nothing more comes back in time.
func (d *Driver) pump() {
	for {
		d.gather()
		if len(d.ready) == 0 {
			return
		}
		ready := d.ready
		d.ready = nil
		slices.SortFunc(ready, func(x, y back) int { return x.seq - y.seq })
		for _, b := range ready {
			d.deliver(b)
		}
	}
}

// gather waits up to Wait for the commands started since the last time,
// then takes whatever else has come back.
func (d *Driver) gather() {
	fresh := d.fresh
	d.fresh = nil
	limit := time.NewTimer(d.Wait)
	defer limit.Stop()
	for slices.ContainsFunc(fresh, func(n int) bool { return d.out[n] }) {
		select {
		case b := <-d.back:
			d.took(b)
		case <-limit.C:
			fresh = nil
		}
	}
	for {
		select {
		case b := <-d.back:
			d.took(b)
		default:
			return
		}
	}
}

var cmdType = reflect.TypeOf(tea.Cmd(nil))

func (d *Driver) deliver(b back) {
	d.t.Helper()
	switch m := b.msg.(type) {
	case nil:
	case panicked:
		d.t.Fatalf("tdptest: a command panicked: %v\n%s%s", m.value, m.stack, d.where())
	case tea.QuitMsg:
		d.quit = true
	case tea.BatchMsg:
		for _, c := range m {
			d.launch(c, nil)
		}
	case testhook.Timer:
		d.seq++
		t := timer{at: d.Now().Add(m.D), seq: d.seq, fn: m.Fn, rest: b.rest}
		i, _ := slices.BinarySearchFunc(d.timers, t, func(x, y timer) int {
			if c := x.at.Compare(y.at); c != 0 {
				return c
			}
			return x.seq - y.seq
		})
		d.timers = slices.Insert(d.timers, i, t)
		return // the rest of its sequence waits for it
	default:
		// tea.Sequence's message is a list of commands, run one after another.
		if v := reflect.ValueOf(m); v.Kind() == reflect.Slice && v.Type().Elem() == cmdType {
			cmds := make([]tea.Cmd, v.Len())
			for i := range cmds {
				cmds[i] = v.Index(i).Interface().(tea.Cmd)
			}
			d.launch(nil, cmds)
		} else {
			d.update(m)
		}
	}
	if len(b.rest) > 0 {
		d.launch(nil, b.rest)
	}
}

// fire moves the time to the first timer and fires it.
func (d *Driver) fire() {
	t := d.timers[0]
	d.timers = d.timers[1:]
	if t.at.After(d.Now()) {
		d.setNow(t.at)
	}
	d.deliver(back{msg: t.fn(d.Now()), rest: t.rest})
	d.pump()
}

// settle takes what came back and, with Auto, plays the animations to
// their end.
func (d *Driver) settle() {
	d.t.Helper()
	d.pump()
	if !d.Auto {
		return
	}
	for n := 0; testhook.Animating(d.app); n++ {
		if n == 100000 || len(d.timers) == 0 {
			d.t.Fatalf("tdptest: an animation does not end%s", d.where())
		}
		d.fire()
	}
}

// --- checks ------------------------------------------------------------------

// check draws the frame as the program would after a message and checks it
// (tdp L4); what went to the clipboard is taken.
func (d *Driver) check() {
	d.t.Helper()
	lines := d.app.Frame()
	if text, ok := testhook.Clipboard(); ok {
		d.copied = append(d.copied, text)
	}
	problem := rowsProblem(lines, d.w, d.h)
	if problem == "" {
		return
	}
	d.broke++
	switch {
	case d.broke <= 3:
		d.t.Errorf("tdptest: tdp L4: %s%s", problem, d.where())
	case d.broke == 4:
		d.t.Errorf("tdptest: tdp L4 broken again; no more reported for this app")
	}
}

// rowsProblem is what is wrong with a frame for a w × h terminal: rows not
// h, a row not w cells, a control character that would reach the terminal
// (its width cannot be known); "" when nothing is.
func rowsProblem(lines []string, w, h int) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	if len(lines) != h {
		return fmt.Sprintf("%d rows on a terminal %d rows high", len(lines), h)
	}
	for i, l := range lines {
		if n := terminu.Width(l); n != w {
			return fmt.Sprintf("row %d is %d cells wide on a terminal %d wide", i+1, n, w)
		}
		if r, ok := control(l); ok {
			return fmt.Sprintf("row %d sends %q to the terminal: %q", i+1, r, withoutSGR(l))
		}
	}
	return ""
}

// control is the first control character in a row, SGR sequences aside.
func control(row string) (rune, bool) {
	for i := 0; i < len(row); {
		if n := sgrLen(row[i:]); n > 0 {
			i += n
			continue
		}
		r, size := utf8.DecodeRuneInString(row[i:])
		if r < 0x20 || r == 0x7f || r >= 0x80 && r < 0xa0 {
			return r, true
		}
		i += size
	}
	return 0, false
}

// withoutSGR is row without its SGR sequences, the control characters in it
// kept.
func withoutSGR(row string) string {
	var b strings.Builder
	for i := 0; i < len(row); i++ {
		if n := sgrLen(row[i:]); n > 0 {
			i += n - 1
			continue
		}
		b.WriteByte(row[i])
	}
	return b.String()
}

// sgrLen is the length of the SGR sequence s starts with, ESC [ digits ; :
// and m; 0 when it starts with none.
func sgrLen(s string) int {
	if len(s) < 3 || s[0] != 0x1b || s[1] != '[' {
		return 0
	}
	for j := 2; j < len(s); j++ {
		switch c := s[j]; {
		case c == 'm':
			return j + 1
		case c >= '0' && c <= '9' || c == ';' || c == ':':
		default:
			return 0
		}
	}
	return 0
}

// where is the state for a failure: the size, the widths, what was done,
// the screen.
func (d *Driver) where() string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n  on %d×%d, icon %d, powerline %d, after %s\n",
		d.w, d.h, terminu.IconWidth(), terminu.PowerlineWidth(), strings.Join(d.trail, " "))
	for _, n := range d.notes {
		fmt.Fprintf(&b, "  with %s\n", n)
	}
	b.WriteString("  screen:\n")
	for i, l := range d.Screen() {
		fmt.Fprintf(&b, "  %3d|%s|\n", i+1, l)
	}
	return b.String()
}

// EachIconWidth runs f once for each width a Nerd Font icon and a powerline
// symbol take on a terminal: one cell each, an icon two, both two (tdp E5).
// The widths are put back after.
func EachIconWidth(t *testing.T, f func(t *testing.T)) {
	t.Helper()
	icon, pl := terminu.IconWidth(), terminu.PowerlineWidth()
	defer terminu.SetIconWidth(icon, pl)
	for _, w := range [][2]int{{1, 1}, {2, 1}, {2, 2}} {
		t.Run(fmt.Sprintf("icon%d_powerline%d", w[0], w[1]), func(t *testing.T) {
			terminu.SetIconWidth(w[0], w[1])
			f(t)
		})
	}
}
