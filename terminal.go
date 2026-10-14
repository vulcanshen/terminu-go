package terminu

import (
	"io"
	"os"
	"strconv"
	"sync"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

// Terminal is a program running in a PTY (components/dialog/terminal): a
// shell, an editor, a remote session. Put it in a panel as the panel's Body,
// or open it in a terminal popup (App.OpenTerminal). While the focus is in
// it every key goes to the program, but the exit key and the app's own keys
// (Rules K10).
type Terminal struct {
	// Title is what runs, in capitals ("SHELL", "ALTERM"); Subject what it
	// runs on, as it is ("myhost"): the popup's title reads "ALTERM: myhost".
	Title, Subject string
	// Command is the program and its arguments: the user's shell ($SHELL)
	// when empty.
	Command []string
	Dir     string
	// Env is added to the app's environment. TERMINU__ICON_WIDTH and
	// TERMINU__POWERLINE_WIDTH are always set to the app's own (tdp E5).
	Env []string
	// Ends makes the exit key end the program ("Alt-Esc:exit"); otherwise
	// leaving keeps it running ("Alt-Esc:leave").
	Ends bool
	// Keys are the app's own keys in the PTY, shown after the exit key.
	Keys []TerminalKey
	// History lets PgUp and PgDn scroll back through what went off the top.
	History bool
	// NotReady, while it reports true, keeps the ordinary keys from the
	// program (it is still connecting); Ctrl-C and the exit key still work.
	NotReady func() bool
	// OnExit runs when the program ends.
	OnExit func(err error) tea.Cmd

	mu      sync.Mutex
	screen  *vt
	in      io.Writer // the PTY: keys go here
	proc    process
	out     chan struct{}
	started bool
	exited  bool
	err     error
	back    int  // lines scrolled back into the history
	inside  bool // a terminal in a panel: the focus is in it
	w, h    int
}

// TerminalKey is one of the app's own keys in a PTY (Rules K10).
type TerminalKey struct {
	Key, Desc string
	Do        func() tea.Cmd
}

// process is the running program, as the platform has it.
type process interface {
	resize(w, h int)
	stop()
}

type termOutput struct{ t *Terminal }
type termExited struct{ t *Terminal }

const exitKey = "Alt-Esc"

func (t *Terminal) vt() *vt {
	if t.screen == nil {
		t.screen = newVT(max(t.w, 80), max(t.h, 24))
	}
	return t.screen
}

// Start runs the program, sized as the terminal was last drawn (80 × 24
// before that). The command it returns keeps the screen up to date.
func (t *Terminal) Start() tea.Cmd {
	if t.started && !t.exited {
		return nil
	}
	t.mu.Lock()
	scr := t.vt()
	w, h := scr.w, scr.h
	t.mu.Unlock()
	env := append(os.Environ(),
		"TERM=xterm-256color", "COLORTERM=truecolor",
		FamilyIconWidth+"="+strconv.Itoa(IconWidth()),
		FamilyPowerlineWidth+"="+strconv.Itoa(PowerlineWidth()))
	env = append(env, t.Env...)
	argv := t.Command
	if len(argv) == 0 {
		argv = []string{cmpOr(os.Getenv("SHELL"), "/bin/sh")}
	}
	pty, proc, err := startPTY(argv, t.Dir, env, w, h)
	if err != nil {
		t.mu.Lock()
		t.screen.write([]byte("\r\n" + err.Error() + "\r\n"))
		t.mu.Unlock()
		t.exited, t.err = true, err
		return nil
	}
	t.in, t.proc = pty, proc
	t.started, t.exited, t.err = true, false, nil
	t.out = make(chan struct{}, 1)
	go t.read(pty, proc)
	return t.wait()
}

// read copies what the program prints into the screen, sends back what the
// screen answers, and tells the app something changed.
func (t *Terminal) read(r io.Reader, proc process) {
	buf := make([]byte, 32<<10)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			t.mu.Lock()
			t.screen.write(buf[:n])
			reply := t.screen.reply
			t.screen.reply = nil
			t.mu.Unlock()
			if len(reply) > 0 {
				_, _ = t.in.Write(reply)
			}
			select {
			case t.out <- struct{}{}:
			default:
			}
		}
		if err != nil {
			break
		}
	}
	if w, ok := proc.(interface{ wait() error }); ok {
		t.err = w.wait()
	}
	close(t.out)
}

func (t *Terminal) wait() tea.Cmd {
	ch := t.out
	return func() tea.Msg {
		if _, ok := <-ch; !ok {
			return termExited{t}
		}
		return termOutput{t}
	}
}

// Exited reports whether the program has ended.
func (t *Terminal) Exited() bool { return t.exited }

// Running reports whether the program has started and not ended.
func (t *Terminal) Running() bool { return t.started && !t.exited }

// Enter puts the focus in a terminal in a panel: keys go to the program
// until the exit key.
func (t *Terminal) Enter() { t.inside = true }

// Inside reports whether the focus is in the terminal.
func (t *Terminal) Inside() bool { return t.inside }

// Stop ends the program.
func (t *Terminal) Stop() {
	if t.proc != nil && !t.exited {
		t.proc.stop()
	}
}

func (t *Terminal) title() string {
	return popupTitle(cmpOr(t.Title, "Shell"), t.Subject)
}

func (t *Terminal) exitVerb() string {
	if t.Ends {
		return "exit"
	}
	return "leave"
}

// hint is the exit key first, then the app's own keys (components/dialog/terminal).
func (t *Terminal) hint() []KeyHint {
	h := []KeyHint{{exitKey, t.exitVerb()}}
	for _, k := range t.Keys {
		h = append(h, KeyHint{k.Key, k.Desc})
	}
	if t.History {
		h = append(h, KeyHint{"PgUp", "history"})
	}
	return h
}

// resize sizes the screen and the PTY to w × h.
func (t *Terminal) resize(w, h int) {
	if w == t.w && h == t.h || w <= 0 || h <= 0 {
		return
	}
	t.w, t.h = w, h
	t.mu.Lock()
	t.vt().resize(w, h)
	t.mu.Unlock()
	if t.proc != nil && !t.exited {
		t.proc.resize(w, h)
	}
}

// lines draws the screen in w × h; cursor draws the program's cursor, a block
// in Text with Base on it whatever shape it asks for (components/dialog/terminal).
func (t *Terminal) lines(w, h int, cursor bool) []string {
	t.resize(w, h)
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.vt().render(t.back, cursor && !t.exited, Style{Fg: Base, Bg: Text})
}

// scroll is the position in the history, when scrolled back.
func (t *Terminal) scroll() string {
	if t.back == 0 || t.screen == nil {
		return ""
	}
	total := len(t.screen.history) + t.screen.h
	first := len(t.screen.history) - t.back + 1
	return scrollText(false, first, first+t.screen.h-1, total)
}

// key is a key while the focus is in the terminal (Rules K10).
func (t *Terminal) key(a *App, key string, msg tea.KeyMsg) tea.Cmd {
	if key == exitKey {
		// Alt-Esc and two Escs are the same bytes: it always asks (Rules K10).
		verb := t.exitVerb()
		name := cmpOr(t.Subject, t.Title)
		q := "Leave " + name + "? It keeps running."
		if t.Ends {
			q = "Exit " + name + "? The program ends."
		}
		return a.Confirm(Ask{Title: Upper(verb[:1]) + verb[1:], Question: q, Verb: verb}, func() tea.Cmd { return t.leave(a) })
	}
	for _, k := range t.Keys {
		if k.Key == key {
			return run(k.Do)
		}
	}
	if t.History && (key == "PgUp" || key == "PgDn") {
		t.mu.Lock()
		half := max(t.vt().h/2, 1)
		if key == "PgUp" {
			t.back = min(t.back+half, len(t.screen.history))
		} else {
			t.back = max(t.back-half, 0)
		}
		t.mu.Unlock()
		return nil
	}
	t.back = 0
	if t.exited || t.in == nil {
		return nil
	}
	if t.NotReady != nil && t.NotReady() && key != "Ctrl-C" {
		return nil
	}
	t.mu.Lock()
	b := encodeKey(msg, t.vt().appCursor, t.vt().bracketed)
	t.mu.Unlock()
	if len(b) > 0 {
		_, _ = t.in.Write(b)
	}
	return nil
}

// leave is the exit key accepted: the focus comes out, and with Ends the
// program stops.
func (t *Terminal) leave(a *App) tea.Cmd {
	if t.Ends {
		t.Stop()
	}
	return t.Hide(a)
}

// Hide takes the focus out and leaves the program running: its popup closes,
// a terminal in a panel lets the panel's keys work again. For an app's own
// exit key, such as one that hides a terminal popup.
func (t *Terminal) Hide(a *App) tea.Cmd {
	t.inside = false
	if l := a.top(); l != nil {
		if tv, ok := l.view.(*termView); ok && tv.t == t {
			return a.closeTop()
		}
	}
	return nil
}

// encodeKey is the bytes a terminal sends for a key.
func encodeKey(msg tea.KeyMsg, appCursor, bracketed bool) []byte {
	if msg.Paste {
		s := string(msg.Runes)
		if bracketed {
			s = "\x1b[200~" + s + "\x1b[201~"
		}
		return []byte(s)
	}
	var s string
	cursor := func(c byte) string {
		if appCursor {
			return "\x1bO" + string(c)
		}
		return "\x1b[" + string(c)
	}
	switch msg.Type {
	case tea.KeyRunes:
		s = string(msg.Runes)
	case tea.KeySpace:
		s = " "
	case tea.KeyUp:
		s = cursor('A')
	case tea.KeyDown:
		s = cursor('B')
	case tea.KeyRight:
		s = cursor('C')
	case tea.KeyLeft:
		s = cursor('D')
	case tea.KeyHome:
		s = cursor('H')
	case tea.KeyEnd:
		s = cursor('F')
	case tea.KeyShiftTab:
		s = "\x1b[Z"
	case tea.KeyPgUp:
		s = "\x1b[5~"
	case tea.KeyPgDown:
		s = "\x1b[6~"
	case tea.KeyDelete:
		s = "\x1b[3~"
	case tea.KeyInsert:
		s = "\x1b[2~"
	case tea.KeyCtrlUp:
		s = "\x1b[1;5A"
	case tea.KeyCtrlDown:
		s = "\x1b[1;5B"
	case tea.KeyCtrlRight:
		s = "\x1b[1;5C"
	case tea.KeyCtrlLeft:
		s = "\x1b[1;5D"
	case tea.KeyShiftUp:
		s = "\x1b[1;2A"
	case tea.KeyShiftDown:
		s = "\x1b[1;2B"
	case tea.KeyShiftRight:
		s = "\x1b[1;2C"
	case tea.KeyShiftLeft:
		s = "\x1b[1;2D"
	default:
		// Bubble Tea numbers F1, F2, … downwards.
		f := int(tea.KeyF1 - msg.Type)
		switch {
		case f >= 0 && f < 4:
			s = "\x1bO" + string(rune('P'+f))
		case f >= 4 && f < 12:
			codes := []int{15, 17, 18, 19, 20, 21, 23, 24}
			s = "\x1b[" + strconv.Itoa(codes[f-4]) + "~"
		case msg.Type >= 0 && msg.Type <= 127:
			// Control keys are their own byte: Enter, Tab, Esc, Backspace, Ctrl-X.
			s = string([]byte{byte(msg.Type)})
		}
	}
	if msg.Alt && s != "" {
		s = "\x1b" + s
	}
	if !utf8.ValidString(s) && msg.Type == tea.KeyRunes {
		return nil
	}
	return []byte(s)
}

// --- a terminal in a panel ----------------------------------------------------

// Draw draws the program's screen; its cursor shows while the focus is in it.
func (t *Terminal) Draw(w, h int, focused bool) []string {
	return t.lines(w, h, focused && t.inside)
}

// Key does nothing: while the focus is in the terminal the app sends it
// every key; outside it the panel's own keys apply.
func (t *Terminal) Key(string, tea.KeyMsg) (bool, tea.Cmd) { return false, nil }

// HasItem: a terminal has no items.
func (t *Terminal) HasItem() bool { return false }

// Scroll is the history position when scrolled back.
func (t *Terminal) Scroll() string { return t.scroll() }

// MoveKeys: none outside the terminal.
func (t *Terminal) MoveKeys() []KeyRow { return nil }

// --- the terminal popup ----------------------------------------------------------

// OpenTerminal opens a terminal popup with t in it, starting it if it has not
// started (components/dialog/terminal).
func (a *App) OpenTerminal(t *Terminal) tea.Cmd {
	return tea.Batch(a.open(&termView{t: t, a: a}), t.Start())
}

type termView struct {
	t *Terminal
	a *App
}

// topRows are the rows a terminal popup leaves above it: the screen chips
// and their divider, or the statusbar (Rules F7, the terminal exception).
func (a *App) topRows() int {
	switch {
	case len(a.screens) > 1 && a.h > 3:
		return 2
	case len(a.screens) > 1 || a.Status != nil:
		return 1
	}
	return 0
}

func (tv *termView) measure(sw, sh int) (int, int) {
	return max(sw-2, 1), max(sh-tv.a.topRows(), 3)
}

// at puts the popup under the top rows, down to the last row: it covers the
// footer.
func (tv *termView) at(sw, sh int) (int, int) { return 1, tv.a.topRows() }

func (tv *termView) draw(w, h int, layer Color) []string {
	body := tv.t.lines(w-2, h-2, true)
	return popupBox{color: layer, glyph: glyphTerminal, title: tv.t.title(), hint: tv.t.hint(), scroll: tv.t.scroll(), bare: true}.draw(w, body)
}

func (tv *termView) key(a *App, key string, msg tea.KeyMsg) (bool, tea.Cmd) {
	return true, tv.t.key(a, key, msg)
}

func (tv *termView) surface() string      { return tv.t.title() }
func (tv *termView) keyRef() []refSection { return nil }
func (tv *termView) typing() bool         { return true }

// ptyFocus is the terminal the focus is in, if any: a terminal popup on top,
// or a terminal in the focused panel the user went into.
func (a *App) ptyFocus() *Terminal {
	if l := a.top(); l != nil {
		if tv, ok := l.view.(*termView); ok {
			return tv.t
		}
		return nil
	}
	if s := a.Screen(); s != nil {
		if p := s.Focused(); p != nil {
			if t, ok := p.Body.(*Terminal); ok && t.inside {
				return t
			}
		}
	}
	return nil
}

// terminalMsg handles a terminal's news: new output, or the program ended.
func (a *App) terminalMsg(msg tea.Msg) (tea.Cmd, bool) {
	switch m := msg.(type) {
	case termOutput:
		return m.t.wait(), true
	case termExited:
		t := m.t
		t.exited, t.inside = true, false
		var cmds []tea.Cmd
		for _, l := range a.stack {
			if tv, ok := l.view.(*termView); ok && tv.t == t && !l.closing {
				if l == a.top() {
					cmds = append(cmds, a.closeTop())
				}
			}
		}
		if t.OnExit != nil {
			cmds = append(cmds, t.OnExit(t.err))
		}
		return tea.Batch(cmds...), true
	}
	return nil, false
}
