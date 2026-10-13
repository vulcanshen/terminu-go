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
	dialogs  *dialogsScreen
	terminal *terminalScreen
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
	a.Splash = splash()
	// A core key of the showcase's own (Rules K1): the same on every panel and
	// popup.
	a.CoreKeys = []terminu.CoreKey{{Key: "!", Desc: "the app log", Do: func() tea.Cmd {
		return a.Viewer(&terminu.Viewer{Title: "App log", Glyph: glyphYaml, Lines: s.appLog()})
	}}}
	a.Progress = func() (float64, bool) { return s.task, s.taskOn }
	s.app = a
	s.panels = newPanelsScreen(s)
	a.AddScreen(s.panels.screen)
	s.inputs = newInputsScreen(s)
	a.AddScreen(s.inputs.screen)
	s.dialogs = newDialogsScreen(s)
	a.AddScreen(s.dialogs.screen)
	s.terminal = newTerminalScreen(s)
	a.AddScreen(s.terminal.screen)
	a.Global = []terminu.Action{
		{Key: "N", Label: "Narrow layout", Desc: "draw one panel at a time, as on a narrow terminal", Scope: terminu.GlobalScope,
			Do: func() tea.Cmd { a.ForceNarrow = !a.ForceNarrow; return nil }},
		// A digit: a global hotkey that only works in the global operation popup (Rules K1, M4).
		{Key: "1", Label: "About", Desc: "what the showcase is", Scope: terminu.GlobalScope,
			Do: func() tea.Cmd {
				return a.Toast(terminu.Info, "showcase "+version+": every component of terminu design")
			}},
		{Key: "S", Label: "Start task", Desc: "a long job: the bar under the screen chips", Scope: terminu.GlobalScope,
			Do: s.startTask},
		{Label: "Hide secrets in preview", Desc: "the setting hide_secrets_in_preview, on or off", Scope: terminu.GlobalScope,
			Do: func() tea.Cmd {
				a.HideSecretsInPreview = !a.HideSecretsInPreview
				state := "off"
				if a.HideSecretsInPreview {
					state = "on"
				}
				return a.Toast(terminu.Info, "hide_secrets_in_preview is "+state)
			}},
	}
	a.OnMsg = s.onMsg
	a.Start = func() tea.Cmd { return tea.Tick(time.Second, func(time.Time) tea.Msg { return logTick{} }) }
	return s
}

// status is the statusbar (layout/screen): the layout with its hotkey, how
// wide an icon is, the log's errors and warnings, the running job. Every
// number keeps its width (Rules L2).
func (s *showcase) status(w int) string {
	grey := terminu.Fg(terminu.Overlay0)
	narrow := "off"
	if s.app.ForceNarrow {
		narrow = "on "
	}
	text := grey.Render("[N]arrow: ") + terminu.Fg(terminu.Lavender).Render(narrow) +
		grey.Render("  icon ") + terminu.Fg(terminu.Lavender).Render(fmt.Sprint(terminu.IconWidth())) +
		grey.Render("  powerline ") + terminu.Fg(terminu.Lavender).Render(fmt.Sprint(terminu.PowerlineWidth())) +
		terminu.Fg(terminu.Red).Render(fmt.Sprintf("%3d", s.panels.errors)) + grey.Render(" err") +
		terminu.Fg(terminu.Peach).Render(fmt.Sprintf("%3d", s.panels.warnings)) + grey.Render(" warn")
	if s.taskOn {
		text += grey.Render("  ·  ") + terminu.Fg(terminu.Green).Render(fmt.Sprintf("copying %3d%%", int(s.task*100)))
	}
	return terminu.Truncate(text, w)
}

// appLog is what the showcase has done, for its own core key.
func (s *showcase) appLog() []string {
	return []string{
		"started " + s.started.Format("15:04:05"),
		fmt.Sprintf("icon width %d, powerline width %d", terminu.IconWidth(), terminu.PowerlineWidth()),
		fmt.Sprintf("%d log lines, %d errors, %d warnings", s.panels.logN, s.panels.errors, s.panels.warnings),
		"pretend files in /tmp/terminu-showcase",
	}
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
	case toastLater, loadTick:
		return s.dialogs.onMsg(msg)
	case busy:
		return s.terminal.onMsg(msg)
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

// The showcase's own title glyphs (an input's glyph is the app's choice).
const (
	glyphPencil = "\U000F03EB" // nf-md-pencil
	glyphLock   = "\U000F033E" // nf-md-lock
	glyphTimer  = "\U000F13AB" // nf-md-timer_outline
)
