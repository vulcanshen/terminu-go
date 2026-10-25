package main

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"
)

// terminalScreen is [T]ERMINAL: a program in a PTY, placed both ways. [1]
// SHELL is a PTY in a panel (Enter goes in); [2] SESSIONS opens each session
// in a terminal popup, but its first row, which stands for [1] SHELL: the
// cursor on it points at that panel (layout/panel, the frame).
type terminalScreen struct {
	s        *showcase
	screen   *terminu.Screen
	shell    *terminu.Terminal
	shellP   *terminu.Panel
	list     *terminu.List
	sessions map[string]*terminu.Terminal
}

type session struct {
	name, what string
	command    []string
}

var sessions = []session{
	{"shell", "your shell", nil},
	{"keys", "cat -v: what each key sends to the program", []string{"cat", "-v"}},
	{"top", "a full-screen program", []string{"top"}},
	{"vi", "an editor that wants Esc", []string{"vi"}},
}

// panelRow is the sessions row that stands for [1] SHELL.
const panelRow = "panel"

// busy is the showcase standing still for a while, so that keys pile up
// behind it (record C2: two Escs while the app is busy).
type busy struct{}

func newTerminalScreen(s *showcase) *terminalScreen {
	ts := &terminalScreen{s: s, sessions: map[string]*terminu.Terminal{}}
	a := s.app
	ts.shell = ts.newShell()
	ts.shellP = &terminu.Panel{Label: "Shell", Body: ts.shell, Actions: []terminu.Action{
		{Key: "Enter", Label: "Type", Desc: "go into the shell; Alt-Esc comes back out", Scope: terminu.PanelScope, Hint: "type",
			Do: func() tea.Cmd {
				ts.shell.Enter()
				if !ts.shell.Running() {
					return ts.shell.Start()
				}
				return nil
			}},
		{Label: "Slow start", Desc: "a shell that takes 3 seconds to connect", Scope: terminu.PanelScope, Do: ts.slowStart},
	}}
	// Inside the shell ? is the shell's own: its keys are told here (Rules M6).
	ts.shellP.OtherKeys = []terminu.KeySection{{Title: "terminal", Rows: []terminu.KeyRow{
		{Key: "Alt-Esc", Desc: "leave the shell, asked first"},
		{Key: "PgUp", Desc: "the History mode: j|k, u|d, gg|G scroll back, Esc goes back"},
		{Key: "Alt-v", Desc: "the selection mode: the screen holds still, v selects, y copies"},
		{Key: "Ctrl-C", Desc: "to the shell, as every other key"},
	}}}
	ts.list = &terminu.List{Columns: []terminu.Column{{Title: "Session"}, {Title: "Status", Drop: 2}, {Title: "What", Shrink: true, Drop: 1}},
		Points: func(r terminu.Row) *terminu.Panel {
			if r.ID == panelRow {
				return ts.shellP
			}
			return nil
		}}
	ts.refresh()
	open := func() tea.Cmd {
		r, ok := ts.list.Current()
		if !ok {
			return nil
		}
		if r.ID == panelRow {
			return a.Focus(ts.shellP)
		}
		return a.OpenTerminal(ts.session(r.ID))
	}
	list := &terminu.Panel{Label: "Sessions", Body: ts.list, Actions: []terminu.Action{
		// A session clears the Space menu it is opened from (Rules T1).
		{Key: "Enter", Label: "Open", Desc: "the session in a terminal popup; the panel row goes to [1] SHELL", Scope: terminu.ItemScope, Hint: "open", Do: open, Clear: true},
		{Key: "Alt-t", Label: "term", Desc: "the same, from anywhere in the list", Scope: terminu.ItemScope, Hint: "term", Do: open, Clear: true},
	}}
	// A lowercase screen key in a lowercase name: the chip keeps the marked
	// letter as it is, [t]ERMINAL (Rules M5, panel: capitals).
	ts.screen = &terminu.Screen{Name: "terminal", Key: "t", Desc: "a terminal in a panel and in a popup",
		Panels: []*terminu.Panel{ts.shellP, list}, NarrowBelow: 60, Layout: func(w, h int) []terminu.Rect {
			left := w * 3 / 5
			return []terminu.Rect{{X: 0, Y: 0, W: left, H: h}, {X: left, Y: 0, W: w - left, H: h}}
		}}
	return ts
}

func (ts *terminalScreen) newShell() *terminu.Terminal {
	t := &terminu.Terminal{Title: "Shell", Interactive: true}
	// Alt-l gives every key to the shell (a family app running inside takes
	// Alt-v and PgUp itself) and takes them back (components/dialog/terminal).
	t.Keys = []terminu.TerminalKey{{Key: "Alt-l", Desc: "lock", Do: func() tea.Cmd {
		t.AllKeys = !t.AllKeys
		t.Keys[0].Desc = map[bool]string{true: "unlock", false: "lock"}[t.AllKeys]
		return nil
	}}}
	return t
}

// slowStart is a shell that takes a while to connect: until then only
// Ctrl-C and the exit key go through (Rules K10).
func (ts *terminalScreen) slowStart() tea.Cmd {
	ts.shell.Stop()
	ready := time.Now().Add(3 * time.Second)
	ts.shell = &terminu.Terminal{Title: "Shell", Subject: "slow-host", Interactive: true,
		Command:  []string{"sh", "-c", `echo "connecting to slow-host…"; sleep 3; echo connected; exec "${SHELL:-sh}"`},
		NotReady: func() bool { return time.Now().Before(ready) }}
	ts.shellP.Body = ts.shell
	ts.shell.Enter()
	return ts.shell.Start()
}

func (ts *terminalScreen) session(name string) *terminu.Terminal {
	if t := ts.sessions[name]; t != nil && !t.Exited() {
		return t
	}
	var s session
	for _, x := range sessions {
		if x.name == name {
			s = x
		}
	}
	a := ts.s.app
	t := &terminu.Terminal{Title: "Session", Subject: s.name, Command: s.command, Ends: true, Interactive: true}
	t.Keys = []terminu.TerminalKey{
		{Key: "Alt-t", Desc: "hide", Do: func() tea.Cmd { return t.Hide(a) }},
		{Key: "Alt-b", Desc: "busy", Do: func() tea.Cmd { return func() tea.Msg { return busy{} } }},
	}
	t.OnExit = func(err error) tea.Cmd {
		ts.refresh()
		text := s.name + " ended"
		if err != nil {
			text += ": " + err.Error()
		}
		return a.Toast(terminu.Info, text)
	}
	ts.sessions[name] = t
	defer ts.refresh()
	return t
}

func (ts *terminalScreen) refresh() {
	rows := []terminu.Row{{ID: panelRow, Cells: []string{panelRow, "in a panel", "the shell in [1] SHELL: the cursor here points at it"},
		Styled: [][]terminu.Span{nil, {{Text: "in a panel", Style: terminu.Fg(terminu.Overlay0)}}}}}
	for _, s := range sessions {
		// The status in its colour: a cell's colours come as spans.
		status, style := "not started", terminu.Fg(terminu.Overlay0)
		if t := ts.sessions[s.name]; t != nil {
			if t.Exited() {
				status = "ended"
			} else {
				status, style = "live", terminu.Fg(terminu.Green)
			}
		}
		rows = append(rows, terminu.Row{ID: s.name, Cells: []string{s.name, status, s.what},
			Styled: [][]terminu.Span{nil, {{Text: status, Style: style}}}})
	}
	ts.list.SetRows(rows)
}

func (ts *terminalScreen) onMsg(msg tea.Msg) tea.Cmd {
	if _, ok := msg.(busy); ok {
		time.Sleep(2 * time.Second) // keys pressed now wait in line
		return ts.s.app.Toast(terminu.Info, fmt.Sprintf("Was busy for %s", 2*time.Second))
	}
	return nil
}
