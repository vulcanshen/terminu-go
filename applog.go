package terminu

import (
	"strconv"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// The App Log (tdp note, App Log; tdp F5): what the app did
// and what went wrong, newest last, kept in memory only. ! on a panel opens
// it; every toast and error popup goes in by itself, and the screen switches
// the lib runs; the app logs its operations and what it does in the
// background.

// logMost is how many entries the App Log keeps; the oldest go first.
const logMost = 1000

type logEntry struct {
	at   time.Time
	kind ToastKind
	text string
}

// appLog is the App Log's entries and what of them is unseen: the errors
// and warnings logged since it was last open.
type appLog struct {
	mu          sync.Mutex
	entries     []logEntry
	logged      int // entries ever logged: what the open view has caught up to
	errs, warns int
	open        bool // on the screen: what comes in is seen
	view        *Viewer
	viewLogged  int
}

// Log writes to the App Log: in kind's level (Info INFO, Success OK,
// Warning WARN, Error ERR), at the App's time. Every toast and error popup is
// logged by itself, and so is a switch of screens the user runs ("Switched to
// Pods"). The app logs the operations of its own the user runs and their
// results: every item, panel and global operation (tdp P3), from a menu, a
// hotkey or Enter, one that only looks too (opening a viewer); not moving the
// cursor or the focus, and what is done in a popup only as the result of the
// operation it belongs to (tdp note.7). It also logs what it does in the
// background (moving its files, defaults it made at the start, the details
// of what it could not reach). Each entry starts with a capital letter as a
// toast does (tdp toast.2). It may be called from any goroutine; the screen
// shows it with the next frame.
func (a *App) Log(kind ToastKind, text string) {
	at := a.now()
	l := &a.logs
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, logEntry{at, kind, text})
	l.logged++
	if n := len(l.entries) - logMost; n > 0 {
		l.entries = append(l.entries[:0], l.entries[n:]...)
	}
	if !l.open {
		switch kind {
		case Error:
			l.errs++
		case Warning:
			l.warns++
		}
	}
}

// level is a kind's level in the App Log, four cells, in its colour.
func (k ToastKind) level() string {
	switch k {
	case Success:
		return Fg(Green).Render("OK  ")
	case Warning:
		return Fg(Peach).Render("WARN")
	case Error:
		return Fg(Red).Render("ERR ")
	}
	return Fg(Text).Render("INFO")
}

// line is an entry as the App Log shows it: its time HH:MM:SS in Overlay0,
// its level, what happened.
func (e logEntry) line() string {
	return Fg(Overlay0).Render(e.at.Format("15:04:05")) + " " + e.kind.level() + " " + e.text
}

// openLog opens the App Log at its end: what it has is seen now.
func (a *App) openLog() tea.Cmd {
	v := &Viewer{Title: "App Log", Glyph: glyphLog}
	l := &a.logs
	l.mu.Lock()
	v.Lines = l.lines()
	l.errs, l.warns, l.open, l.view, l.viewLogged = 0, 0, true, v, l.logged
	l.mu.Unlock()
	v.top = 1 << 30 // the newest, at the bottom
	return a.Viewer(v)
}

// lines are the entries as the App Log shows them.
func (l *appLog) lines() []string {
	lines := make([]string, len(l.entries))
	for i, e := range l.entries {
		lines[i] = e.line()
	}
	return lines
}

// syncLog brings the open App Log up to what was logged, following its end
// when it was there, as streaming content does; once it has closed, what
// comes in is unseen again.
func (a *App) syncLog() {
	l := &a.logs
	l.mu.Lock()
	defer l.mu.Unlock()
	v := l.view
	if v == nil {
		return
	}
	if !a.logOnScreen(v) {
		l.view, l.open = nil, false
		return
	}
	if l.viewLogged == l.logged {
		return
	}
	line, sub := v.maxTop()
	atEnd := v.textW == 0 || v.top > line || v.top == line && v.sub >= sub
	v.Lines, l.viewLogged = l.lines(), l.logged
	if atEnd {
		v.top = 1 << 30
	}
}

// logOnScreen reports whether v is in the stack, not closing.
func (a *App) logOnScreen(v *Viewer) bool {
	for _, layer := range a.stack {
		if vv, ok := layer.view.(*viewerView); ok && vv.v == v && !layer.closing {
			return true
		}
	}
	return false
}

// logHint is the footer's ! pair: !:log, or how many errors (Red), else
// warnings (Peach), came in since the App Log was last open (tdp
// screen.2).
func (a *App) logHint() KeyHint {
	l := &a.logs
	l.mu.Lock()
	defer l.mu.Unlock()
	count := func(n int, one string) string {
		s := strconv.Itoa(n) + " " + one
		if n != 1 {
			s += "s"
		}
		return s
	}
	switch {
	case l.errs > 0:
		return KeyHint{"!", Fg(Red).Render(count(l.errs, "error"))}
	case l.warns > 0:
		return KeyHint{"!", Fg(Peach).Render(count(l.warns, "warning"))}
	}
	return KeyHint{"!", "log"}
}

// errorText is an error popup as the App Log writes it.
func errorText(title, subject, message string) string {
	return strings.TrimSpace(title+" "+subject) + ": " + message
}
