// Command showcase shows every component of terminu design and every way they
// combine, on fake data laid out like a real app. It is not released.
package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"
)

const version = "0.0.0-dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "--version", "-v":
			fmt.Println("showcase " + version)
			return
		case "help", "--help", "-h":
			fmt.Print(usage)
			return
		default:
			fmt.Fprintf(os.Stderr, "showcase: unknown command %q\n\n%s", os.Args[1], usage)
			os.Exit(2)
		}
	}
	if err := newShowcase().app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "showcase:", err)
		os.Exit(1)
	}
}

const usage = `showcase — every component of terminu design, on fake data

Usage:
  showcase            run it
  showcase version    print the version (also --version, -v)
  showcase help       print this (also --help, -h)

Environment:
  SHOWCASE__ICON_WIDTH        1 or 2: how many cells a Nerd Font icon takes,
                              instead of asking the terminal
  SHOWCASE__POWERLINE_WIDTH   1 or 2: the same for a powerline symbol
  TERMINU__ICON_WIDTH         the same as SHOWCASE__ICON_WIDTH, for every terminu app
  TERMINU__POWERLINE_WIDTH    the same as SHOWCASE__POWERLINE_WIDTH, for every terminu app
`

// showcase is the app and its screens' state.
type showcase struct {
	app      *terminu.App
	panels   *panelsScreen
	inputs   *inputsScreen
	task     float64
	taskOn   bool
	started  time.Time
	logTimer bool
}

type logTick struct{}
type taskTick struct{}

func newShowcase() *showcase {
	s := &showcase{started: time.Now()}
	a := terminu.New("showcase")
	a.Quit = &terminu.Ask{Title: "Quit", Question: "Quit showcase?", Verb: "quit"}
	a.QuitDesc = "leave the showcase"
	a.Status = s.status
	a.Progress = func() (float64, bool) { return s.task, s.taskOn }
	s.app = a
	s.panels = newPanelsScreen(s)
	a.AddScreen(s.panels.screen)
	s.inputs = newInputsScreen(s)
	a.AddScreen(s.inputs.screen)
	a.AddScreen(placeholder("Dialogs", "D", "every kind of dialog, and how they stack"))
	a.AddScreen(placeholder("Terminal", "T", "a terminal in a panel and in a popup"))
	a.Global = []terminu.Action{
		{Label: "Narrow layout", Desc: "draw one panel at a time, as on a narrow terminal", Scope: terminu.GlobalScope,
			Do: func() tea.Cmd { a.ForceNarrow = !a.ForceNarrow; return nil }},
		{Key: "S", Label: "Start task", Desc: "a long job: the bar under the screen chips", Scope: terminu.GlobalScope,
			Do: s.startTask},
	}
	a.OnMsg = s.onMsg
	a.Start = func() tea.Cmd { return tea.Tick(time.Second, func(time.Time) tea.Msg { return logTick{} }) }
	return s
}

// status is the statusbar: how wide an icon is, the running job.
func (s *showcase) status(w int) string {
	text := terminu.Fg(terminu.Overlay0).Render("icon width ") + terminu.Fg(terminu.Lavender).Render(fmt.Sprint(terminu.IconWidth())) +
		terminu.Fg(terminu.Overlay0).Render("  powerline ") + terminu.Fg(terminu.Lavender).Render(fmt.Sprint(terminu.PowerlineWidth()))
	if s.taskOn {
		text += terminu.Fg(terminu.Overlay0).Render("  ·  ") + terminu.Fg(terminu.Green).Render(fmt.Sprintf("copying %d%%", int(s.task*100)))
	}
	return terminu.Truncate(text, w)
}

func (s *showcase) startTask() tea.Cmd {
	if s.taskOn {
		return s.app.Toast(terminu.Info, "A task is already running")
	}
	s.taskOn, s.task = true, 0
	return tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg { return taskTick{} })
}

func (s *showcase) onMsg(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case logTick:
		s.panels.appendLog()
		return tea.Tick(time.Second, func(time.Time) tea.Msg { return logTick{} })
	case indexTick:
		return s.inputs.indexStep()
	case taskTick:
		s.task += 0.04
		if s.task >= 1 {
			s.taskOn, s.task = false, 0
			return s.app.Toast(terminu.Info, "Copied 3 files")
		}
		return tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg { return taskTick{} })
	}
	return nil
}

// placeholder is a screen not built yet.
func placeholder(name, key, desc string) *terminu.Screen {
	p := &terminu.Pager{EmptyFact: name + " is not built yet", EmptyHint: "Press [P] to go back to Panels"}
	return &terminu.Screen{Name: name, Key: key, Desc: desc, Panels: []*terminu.Panel{{Label: name, Body: p}}}
}

// The showcase's own title glyphs (an input's glyph is the app's choice).
const (
	glyphPencil = "\U000F03EB" // nf-md-pencil
	glyphLock   = "\U000F033E" // nf-md-lock
	glyphTimer  = "\U000F13AB" // nf-md-timer_outline
)
