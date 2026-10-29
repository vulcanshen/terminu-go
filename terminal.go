package terminu

import (
	"io"
	"os"
	"slices"
	"strconv"
	"sync"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

// Terminal is a program running in a PTY (tdp terminal): a
// shell, an editor, a remote session. Put it in a panel as the panel's Body,
// or open it in a terminal popup (App.OpenTerminal). While the focus is in
// it every key goes to the program, but the exit key and the app's own keys
// (tdp K10).
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
	// Interactive marks a terminal the user types commands of their own in
	// (a shell, an ssh session, kubectl exec), not one the app opens to run
	// one program for one thing ($EDITOR, sudo asking for a password). It
	// has the two modes of tdp terminal, which every other PTY
	// leaves to its program, PgUp too:
	//   - History, on PgUp: j|k, u|d, gg|G, PgUp|PgDn scroll back through
	//     what went off the top, Esc goes back to the live screen, any other
	//     key leaves the mode and goes to the program. While the program
	//     fills the whole screen (vim, less, htop: the alternate screen),
	//     PgUp is the program's and the hint leaves PgUp:history out.
	//   - The selection, on Alt-v (the family's v, with Alt to get out of
	//     the program): the screen holds still, what the program prints
	//     meanwhile waiting; the cursor starts at the first line in sight
	//     and goes up into the history too; v starts selecting, y copies,
	//     Esc leaves. Copy takes the text; unset, it goes on the terminal's
	//     clipboard with a Success toast.
	Interactive bool
	// AllKeys, while set, hands every key to the program but the exit key
	// and the app's own Keys (the one that takes them back): the user gave
	// them all away (sshu's locked cell, a family app running inside). The
	// selection is the program's then and PgUp|PgDn go to it, the innermost
	// PTY scrolling its own history (tdp terminal).
	AllKeys bool
	// ExitKey is the key that takes the focus out (tdp K10), Alt-Esc when
	// unset; the app may change it while the program runs, and the hint and
	// the footer show it as it is. Alt-Esc always asks first; another asks
	// when ExitAsks.
	ExitKey  string
	ExitAsks bool
	// Copy takes what the selection copies (Interactive); unset, it goes on
	// the terminal's clipboard with a Success toast.
	Copy func(text string) tea.Cmd
	// NotReady, while it reports true, keeps the ordinary keys from the
	// program (it is still connecting); Ctrl-C and the exit key still work.
	NotReady func() bool
	// OnExit runs when the program ends.
	OnExit func(err error) tea.Cmd
	// OnLeave runs when the exit key is accepted (asked first or not), once
	// the focus is out and, with Ends, the program told to stop: where the
	// focus goes now is the app's (tdp K10), such as back to the list row
	// of the session left. The app's own keys that call Hide do not run it.
	OnLeave func() tea.Cmd
	// OnFirstOutput runs when the program first prints something after
	// Start: an ssh that has connected, to end the panel's loading and
	// NotReady (tdp K10, F7).
	OnFirstOutput func() tea.Cmd

	mu      sync.Mutex
	screen  *vt
	in      io.Writer // the PTY: keys go here
	proc    process
	out     chan struct{}
	started bool
	exited  bool
	spoke   bool // the program has printed something since Start
	err     error
	back    int  // lines scrolled back into the history
	inside  bool // a terminal in a panel: the focus is in it
	w, h    int
	// cursorColour is the selection's cursor: the popup's layer colour, or a
	// panel's cursor colour.
	cursorColour Color
	// mode is the mode it is in: "", History or the selection's Visual;
	// sel is the screen held still for the selection, as a viewer's lines.
	mode     string
	sel      *Viewer
	pendingG bool // g pressed in a mode, waiting for gg
}

// TerminalKey is one of the app's own keys in a PTY (tdp K10).
type TerminalKey struct {
	// Key is how it is written in the footer and the hint ("Alt-←|→|↑|↓");
	// Keys are the keys it takes, Key itself when empty, as a ModeKey's
	// (tdp M5).
	Key  string
	Keys []string
	Desc string
	Do   func() tea.Cmd
	// DoKey, when set, runs instead of Do, with the key pressed.
	DoKey func(key string) tea.Cmd
}

func (k TerminalKey) takes(key string) bool {
	return ModeKey{Key: k.Key, Keys: k.Keys}.takes(key)
}

// process is the running program, as the platform has it.
type process interface {
	resize(w, h int)
	stop()
}

type termOutput struct{ t *Terminal }
type termExited struct{ t *Terminal }

const exitKey = "Alt-Esc"

// selectKey puts an interactive terminal in the selection mode: the
// family's v, with Alt to get out of the program (tdp terminal).
const selectKey = "Alt-v"

// modes reports whether the terminal has its two modes now: interactive,
// the keys not all handed to the program.
func (t *Terminal) modes() bool { return t.Interactive && !t.AllKeys }

// fullScreen reports a program filling the whole screen: the alternate one.
func (t *Terminal) fullScreen() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.vt().altOn
}

// exit is the exit key now.
func (t *Terminal) exit() string { return cmpOr(t.ExitKey, exitKey) }

// The modes a terminal has (tdp K11; tdp terminal).
const (
	historyMode = "History"
	visualMode  = "Visual"
)

// Mode is the mode the terminal is in, "" for none.
func (t *Terminal) Mode() string { return t.mode }

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
	t.started, t.exited, t.err, t.spoke = true, false, nil, false
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

// Spoke reports whether the program has printed anything since Start.
func (t *Terminal) Spoke() bool { return t.spoke }

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

// hint is the exit key first, then the app's own keys
// (tdp terminal); in a mode, the mode's keys (tdp K11).
func (t *Terminal) hint() []KeyHint {
	switch {
	case t.mode == historyMode:
		return []KeyHint{{"?", "help"}, {"j|k", "scroll"}, {"Esc", "live"}}
	case t.mode == visualMode && t.sel.selecting:
		return []KeyHint{{"y", "copy"}, {"Esc", "unselect"}}
	case t.mode == visualMode:
		return []KeyHint{{"v", "select"}, {"Esc", "cancel"}}
	}
	h := []KeyHint{{t.exit(), t.exitVerb()}}
	for _, k := range t.Keys {
		h = append(h, KeyHint{k.Key, k.Desc})
	}
	if t.modes() {
		h = append(h, KeyHint{selectKey, "select"})
		if !t.fullScreen() {
			h = append(h, KeyHint{"PgUp", "history"})
		}
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
// in Text with Base on it whatever shape it asks for (tdp terminal).
func (t *Terminal) lines(w, h int, cursor bool) []string {
	t.resize(w, h)
	if t.mode == visualMode {
		return t.held(w, h)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.vt().render(t.back, cursor && !t.exited, Style{Fg: Base, Bg: Text})
}

// held is the screen held still for the selection, in w × h: the history
// and the screen as they were, scrolled to follow the cursor, the cursor and
// the selection drawn as a viewer's, a line wider than the panel folding
// (tdp note.2).
func (t *Terminal) held(w, h int) []string {
	v := t.sel
	v.sync()
	v.visible, v.textW = max(h, 1), w
	v.readLines()
	v.line = min(max(v.line, 0), max(len(v.text)-1, 0))
	v.follow(v.line)
	v.clampTop()
	vv := &viewerView{v: v}
	out := make([]string, 0, h)
	for i := v.top; i < len(v.text) && len(out) < h; i++ {
		rows := v.rowsOf(i, vv.lineView(i, t.cursorColour))
		if i == v.top {
			rows = rows[min(v.sub, len(rows)-1):]
		}
		for _, r := range rows {
			if len(out) < h {
				out = append(out, r)
			}
		}
	}
	return Pad2(out, w, h)
}

// selectMode holds the screen still and brings the selection's cursor out at
// the start of the first line in sight (tdp terminal).
func (t *Terminal) selectMode(a *App, layer Color) {
	t.mu.Lock()
	var lines []string
	vt := t.vt()
	history := vt.history
	if vt.altOn {
		history = nil // a program on the whole screen: only that screen, held still
	}
	for _, l := range history {
		lines = append(lines, renderCells(l, vt.w, -1, Style{}))
	}
	for _, l := range vt.cells {
		lines = append(lines, renderCells(l, vt.w, -1, Style{}))
	}
	first := len(history) - min(t.back, len(history))
	t.mu.Unlock()
	copyText := t.Copy
	if copyText == nil {
		copyText = func(text string) tea.Cmd {
			Clipboard(text)
			return a.Toast(Success, "Copied")
		}
	}
	t.sel = &Viewer{Lines: lines, NoNumbers: true, tabW: a.inputTabWidth(),
		Select: &Selection{Key: "v", Name: visualMode, Copy: copyText}}
	t.sel.sync()
	t.sel.inMode, t.sel.line, t.sel.col, t.sel.top = true, first, 0, first
	t.mode, t.back, t.cursorColour = visualMode, 0, layer
}

// leaveMode goes back to the live screen.
func (t *Terminal) leaveMode() {
	t.mode, t.sel, t.back, t.pendingG = "", nil, 0, false
}

// modeKey is a key in one of the terminal's modes; ok is false for a key the
// mode does not take, which leaves the mode and goes to the program (Rules
// K11).
func (t *Terminal) modeKey(a *App, key string) (cmd tea.Cmd, ok bool) {
	if t.pendingG {
		t.pendingG = false
		if key == "g" {
			key = "gg"
		}
	} else if key == "g" {
		t.pendingG = true
		return nil, true
	}
	if key == "?" {
		return a.open(t.modeRef()), true
	}
	if key == "Esc" && !(t.mode == visualMode && t.sel.selecting) { // selecting: a layer back, below
		t.leaveMode()
		return nil, true
	}
	if t.mode == historyMode {
		t.mu.Lock()
		defer t.mu.Unlock()
		half, most := max(t.vt().h/2, 1), len(t.vt().history)
		switch key {
		case "j":
			t.back--
		case "k":
			t.back++
		case "d", "PgDn":
			t.back -= half
		case "u", "PgUp":
			t.back += half
		case "gg":
			t.back = most
		case "G":
			t.back = 0
		default:
			return nil, false
		}
		t.back = min(max(t.back, 0), most)
		return nil, true
	}
	if key != "v" && key != "y" && key != "Esc" && !isMoveKey(key) {
		return nil, false
	}
	_, cmd = (&viewerView{v: t.sel}).key(a, key, tea.KeyMsg{})
	if !t.sel.inMode {
		t.leaveMode() // copied: back to the live screen
	}
	return cmd, true
}

// modeRef is a mode's key reference (tdp K11).
func (t *Terminal) modeRef() *noteView {
	if t.mode == visualMode {
		return keyReference(Upper(visualMode), (&viewerView{v: t.sel}).keyRef())
	}
	return keyReference(Upper(historyMode), []refSection{{title: historyMode, rows: []KeyRow{
		{Key: "j|k", Desc: "a line down, up"},
		{Key: "u|d", Desc: "half a page up, down"},
		{Key: "PgUp|PgDn", Desc: "half a page up, down"},
		{Key: "gg|G", Desc: "the oldest line, the live screen's last"},
		{Key: "Esc", Desc: "back to the live screen"},
	}}, {title: "the program", rows: []KeyRow{{Key: "any other", Desc: "leaves History and goes to the program"}}}})
}

// scroll is the position in the history, when scrolled back.
func (t *Terminal) scroll() string {
	if t.mode == visualMode {
		// The cursor's line, N of M, as a viewer's selection (tdp note).
		return scrollText(true, t.sel.line+1, 0, len(t.sel.Lines))
	}
	if t.back == 0 || t.screen == nil {
		return ""
	}
	total := len(t.screen.history) + t.screen.h
	first := len(t.screen.history) - t.back + 1
	return scrollText(false, first, first+t.screen.h-1, total)
}

// key is a key while the focus is in the terminal (tdp K10).
func (t *Terminal) key(a *App, key string, msg tea.KeyMsg) tea.Cmd {
	if t.mode != "" {
		if cmd, ok := t.modeKey(a, key); ok {
			return cmd
		}
		// Not the mode's: it leaves the mode and goes on as in the live
		// screen, q and Ctrl-C too (tdp K11).
		t.leaveMode()
	}
	if key == t.exit() && (key == exitKey || t.ExitAsks) {
		// Alt-Esc and two Escs are the same bytes: it always asks (tdp K10).
		verb := t.exitVerb()
		name := cmpOr(t.Subject, t.Title)
		q := "Leave " + name + "? It keeps running."
		if t.Ends {
			q = "Exit " + name + "? The program ends."
		}
		return a.Confirm(Ask{Title: Upper(verb[:1]) + verb[1:], Question: q, Verb: verb}, func() tea.Cmd { return t.leave(a) })
	}
	if key == t.exit() {
		return t.leave(a)
	}
	if t.modes() && key == selectKey {
		colour := Subtext1 // a panel's cursor
		if l := a.top(); l != nil {
			colour = LayerColor(a.layerOf(l))
		}
		t.selectMode(a, colour)
		return nil
	}
	for _, k := range t.Keys {
		if k.takes(key) {
			if k.DoKey != nil {
				return k.DoKey(key)
			}
			return run(k.Do)
		}
	}
	if t.modes() && key == "PgUp" && !t.fullScreen() {
		// Back into the history: the History mode (tdp terminal).
		t.mu.Lock()
		t.back = min(t.back+max(t.vt().h/2, 1), len(t.vt().history))
		t.mu.Unlock()
		t.mode = historyMode
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
// program stops; then the app's OnLeave.
func (t *Terminal) leave(a *App) tea.Cmd {
	if t.Ends {
		t.Stop()
	}
	hide := t.Hide(a)
	return tea.Batch(hide, run(t.OnLeave))
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

// Empty reports nothing on the screen yet: the program not started, or
// started and silent so far. The panel's LoadingText or Fail shows in its
// middle meanwhile (tdp panel.15). A Start that failed has its error there.
func (t *Terminal) Empty() bool { return !t.spoke && (t.started || !t.exited) }

// HasItem: a terminal has no items.
func (t *Terminal) HasItem() bool { return false }

// Scroll is the history position when scrolled back.
func (t *Terminal) Scroll() string { return t.scroll() }

// MoveKeys: none outside the terminal.
func (t *Terminal) MoveKeys() []KeyRow { return nil }

// --- the terminal popup ----------------------------------------------------------

// OpenTerminal opens a terminal popup with t in it, starting it if it has not
// started (tdp terminal).
func (a *App) OpenTerminal(t *Terminal) tea.Cmd {
	a.terminals = slices.DeleteFunc(a.terminals, func(o *Terminal) bool { return o == t || o.exited })
	a.terminals = append(a.terminals, t)
	return tea.Batch(a.open(&termView{t: t, a: a}), t.Start())
}

type termView struct {
	t *Terminal
	a *App
	labelled
}

func (tv *termView) modeName() string { return tv.t.modeName() }

// modeName is the mode the terminal is in, its layers joined with ">"
// (Visual>select).
func (t *Terminal) modeName() string {
	if t.mode == visualMode && t.sel != nil {
		return t.sel.modeName()
	}
	return t.mode
}

// topRows are the rows a terminal popup leaves above it: the screen chips
// and their divider, or the statusbar (tdp F7, the terminal exception).
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
	box := popupBox{color: layer, glyph: glyphTerminal, title: tv.t.title(), hint: tv.t.hint(), scroll: tv.t.scroll(), bare: true}
	if tv.t.mode != "" {
		box.color = Yellow // a mode marks its frame (tdp K11)
	}
	lab := tv.labelFor(tv.modeName())
	box.mode, box.closed, box.modeW = lab.name, 1-lab.shown, lab.room
	return box.draw(w, body)
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
		if t := m.t; !t.spoke {
			t.spoke = true
			if t.OnFirstOutput != nil {
				return tea.Batch(t.wait(), t.OnFirstOutput()), true
			}
		}
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
