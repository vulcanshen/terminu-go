// Command showcase shows every component of terminu design and every way they
// combine, on fake data laid out like a real app. It is not released.
package main

import (
	"fmt"
	"io"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"
)

const version = "0.0.0-dev"

func main() {
	if len(os.Args) > 1 {
		os.Exit(command(os.Args[1:], os.Stdout, os.Stderr))
	}
	if err := newShowcase().app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "showcase:", err)
		os.Exit(1)
	}
}

// command runs the command args (tdp E1): what it prints on stdout, its
// errors and warnings on stderr, a line each after "showcase: ", worded as
// the TUI's toasts are; it gives the exit status, 0 done, 1 not done, 2 used
// wrong.
func command(args []string, stdout, stderr io.Writer) int {
	switch args[0] {
	case "version", "--version", "-v":
		fmt.Fprintln(stdout, "showcase "+version)
	case "help", "--help", "-h":
		fmt.Fprint(stdout, usage)
	case "iconwidth":
		for _, l := range terminu.Probe("showcase").Lines() {
			fmt.Fprintln(stdout, l)
		}
	case "settings":
		return settings(args[1:], stdout, stderr)
	case "update":
		// The showcase is never released: it is always a build from source,
		// which update says to update there (tdp E1).
		return terminu.SelfUpdate{App: "showcase", Version: "dev", Owner: "vulcanshen", Tap: "vulcanshen/tap",
			Source: "git pull && go install ./examples/showcase"}.Run(stdout, stderr)
	default:
		fmt.Fprintf(stderr, "showcase: Unknown command %q\n\n%s", args[0], usage)
		return 2
	}
	return 0
}

// settings prints the pretend settings the showcase uses, with no screen
// (tdp E3, E3-40): the two values that do not fit made to fit as at a start,
// and the old layout file it no longer uses, each a line on stderr in the
// words of its toast and App Log, the settings alone on stdout. "broken" is a
// settings file it cannot read: the path and why on stderr, no file touched,
// exit 1.
func settings(args []string, stdout, stderr io.Writer) int {
	say := func(lines ...string) {
		for _, l := range lines {
			fmt.Fprintln(stderr, "showcase: "+l)
		}
	}
	switch {
	case len(args) == 1 && args[0] == "broken":
		path, why := brokenSettings()
		say(terminu.CannotReadText(path, why), "Open showcase to move it aside and start fresh")
		return 1
	case len(args) > 0:
		fmt.Fprintf(stderr, "showcase: Unknown settings %q\n\n%s", args[0], usage)
		return 2
	}
	path, notices := fitSettings(false)
	for _, n := range notices {
		say(n.Lines()...)
	}
	say(terminu.NoLongerUsedLines(oldLayout())...)
	b, err := os.ReadFile(path)
	if err != nil {
		say(terminu.CannotReadText(path, err))
		return 1
	}
	fmt.Fprint(stdout, string(b))
	return 0
}

const usage = `showcase — every component of terminu design, on fake data

Usage:
  showcase                    run it
  showcase version            print the version (also --version, -v)
  showcase help               print this (also --help, -h)
  showcase update             bring it to its latest release (a build from source: says to update it there)
  showcase iconwidth          print how many cells an icon and a powerline symbol take
  showcase settings           print the pretend settings, made to fit; what it says on stderr
  showcase settings broken    the same with a settings file it can't read

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
	kinds    *kindsScreen
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
	// Its layouts still work 20 rows high, under the default 60 × 32 (tdp L1).
	a.MinWidth, a.MinHeight = 60, 20
	a.QuitDesc = "leave the showcase"
	a.Status = s.status
	a.Splash = splash()
	// A core key of the showcase's own (tdp K1): the same on every panel and
	// popup.
	a.CoreKeys = []terminu.CoreKey{{Key: "%", Desc: "the showcase's facts", Do: func() tea.Cmd {
		return a.Viewer(&terminu.Viewer{Title: "Facts", Glyph: glyphYaml, Lines: s.facts()})
	}}}
	a.Progress = func() (float64, bool) { return s.task, s.taskOn }
	a.Spinning = func() bool { return transferring() }
	s.app = a
	s.panels = newPanelsScreen(s)
	a.AddScreen(s.panels.screen)
	s.inputs = newInputsScreen(s)
	a.AddScreen(s.inputs.screen)
	s.dialogs = newDialogsScreen(s)
	a.AddScreen(s.dialogs.screen)
	s.terminal = newTerminalScreen(s)
	a.AddScreen(s.terminal.screen)
	s.kinds = newKindsScreen(a)
	a.AddScreen(s.kinds.screen)
	a.Global = []terminu.Action{
		{Key: "N", Label: "Narrow layout", Desc: "draw one panel at a time, as on a narrow terminal", Scope: terminu.GlobalScope,
			Do: func() tea.Cmd { a.ForceNarrow = !a.ForceNarrow; return nil }},
		// A digit: a global hotkey that only works in the global operation popup (tdp K1, M4).
		{Key: "1", Label: "About", Desc: "what the showcase is", Scope: terminu.GlobalScope,
			Do: func() tea.Cmd {
				return a.Toast(terminu.Info, "About showcase "+version+": every component of terminu design")
			}},
		{Key: "S", Label: "Start task", Desc: "a long job: the bar under the screen chips", Scope: terminu.GlobalScope,
			Do: s.startTask},
		// The app moves the focus where the user asked for it (App.Focus).
		{Key: "J", Label: "Jump to the log", Desc: "the Panels screen's log, from anywhere", Scope: terminu.GlobalScope,
			Do: func() tea.Cmd { return a.Focus(s.panels.log) }},
		{Label: "Hide secrets in preview", Desc: "the setting hide_secrets_in_preview, on or off", Scope: terminu.GlobalScope,
			Do: func() tea.Cmd {
				a.HideSecretsInPreview = !a.HideSecretsInPreview
				state := "off"
				if a.HideSecretsInPreview {
					state = "on"
				}
				return a.Toast(terminu.Info, "Turned hide_secrets_in_preview "+state)
			}},
	}
	a.OnMsg = s.onMsg
	a.Start = func() tea.Cmd {
		// What the app does by itself goes in the App Log (tdp note).
		a.Log(terminu.Info, "Started; pretend files in /tmp/terminu-showcase")
		// A pretend older showcase kept its layout in a file this one no
		// longer uses: left alone, and every start says so (tdp E3).
		a.NoLongerUsed(oldLayout())
		return a.Tick(time.Second, func(time.Time) tea.Msg { return logTick{} })
	}
	return s
}

// status is the statusbar (tdp screen): the layout with its hotkey, how
// wide an icon is, the log's errors and warnings, the running job. Every
// number keeps its width (tdp L2).
func (s *showcase) status(int) []terminu.Span {
	grey, here := terminu.Fg(terminu.Overlay0), terminu.Fg(terminu.Lavender)
	narrow := "off"
	if s.app.ForceNarrow {
		narrow = "on "
	}
	// Each group label: value, a space after the colon; two spaces between
	// groups, no other mark (components, spacing). Each value keeps its
	// width (tdp L2) but the last, which pushes nothing.
	spans := []terminu.Span{
		{Text: "[N]arrow: ", Style: grey}, {Text: narrow, Style: here},
		{Text: "  icon: ", Style: grey}, {Text: fmt.Sprint(terminu.IconWidth()), Style: here},
		{Text: "  powerline: ", Style: grey}, {Text: fmt.Sprint(terminu.PowerlineWidth()), Style: here},
		{Text: "  errors: ", Style: grey}, {Text: fmt.Sprintf("%-3d", s.panels.errors), Style: terminu.Fg(terminu.Red)},
		{Text: "  warnings: ", Style: grey}, {Text: fmt.Sprintf("%-3d", s.panels.warnings), Style: terminu.Fg(terminu.Peach)},
	}
	if s.taskOn {
		spans = append(spans, terminu.Span{Text: "  copying: ", Style: grey},
			terminu.Span{Text: fmt.Sprintf("%d%%", int(s.task*100)), Style: terminu.Fg(terminu.Green)})
	}
	return spans
}

// facts are what the showcase is now, for its own core key.
func (s *showcase) facts() []string {
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
	return s.app.Tick(200*time.Millisecond, func(time.Time) tea.Msg { return taskTick{} })
}

func (s *showcase) onMsg(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case logTick:
		s.panels.appendLog()
		return s.app.Tick(time.Second, func(time.Time) tea.Msg { return logTick{} })
	case indexTick:
		return s.inputs.indexStep()
	case grepDone:
		return s.inputs.grepDone(msg.(grepDone))
	case toastLater, loadTick, watchTick:
		return s.dialogs.onMsg(msg)
	case busy:
		return s.terminal.onMsg(msg)
	case taskTick:
		s.task += 0.04
		if s.task >= 1 {
			s.taskOn, s.task = false, 0
			return s.app.Toast(terminu.Success, "Copied 3 files")
		}
		return s.app.Tick(200*time.Millisecond, func(time.Time) tea.Msg { return taskTick{} })
	}
	return nil
}

// The showcase's own title glyphs (an input's glyph is the app's choice).
const (
	glyphPencil = "\U000F03EB" // nf-md-pencil
	glyphLock   = "\U000F033E" // nf-md-lock
	glyphTimer  = "\U000F13AB" // nf-md-timer_outline
)
