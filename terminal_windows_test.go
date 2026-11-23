//go:build windows

package terminu

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/sys/windows"
)

// helperEnv makes the test binary the program a ConPTY test runs, in the way
// it names (TestConPTYHelper).
const helperEnv = "TERMINU__CONPTY_HELPER"

// TestConPTYHelper is not a test: it is the program the ConPTY tests run.
//   - "echo" says hello with the widths it was given and the folder it runs
//     in, then answers each line: "size" with the console's size, "quit" by
//     ending, any other with "got <line>".
//   - "exit3" ends at once, with exit status 3.
//   - "wait" says so and waits a minute.
func TestConPTYHelper(t *testing.T) {
	switch os.Getenv(helperEnv) {
	case "":
		t.Skip("the program the ConPTY tests run")
	case "echo":
		wd, _ := os.Getwd()
		fmt.Printf("hello %s%s %s\n", os.Getenv(FamilyIconWidth), os.Getenv(FamilyPowerlineWidth), filepath.Base(wd))
		in := bufio.NewScanner(os.Stdin)
		for in.Scan() {
			switch line := strings.TrimSpace(in.Text()); line {
			case "size":
				var info windows.ConsoleScreenBufferInfo
				_ = windows.GetConsoleScreenBufferInfo(windows.Handle(os.Stdout.Fd()), &info)
				w := info.Window
				fmt.Printf("size %dx%d\n", w.Right-w.Left+1, w.Bottom-w.Top+1)
			case "quit":
				os.Exit(0)
			default:
				fmt.Printf("got %s\n", line)
			}
		}
		os.Exit(0)
	case "exit3":
		os.Exit(3)
	case "wait":
		fmt.Println("waiting")
		time.Sleep(time.Minute)
		os.Exit(0)
	}
}

// helper is a terminal running the helper program in the way given.
func helper(t *testing.T, way string) *Terminal {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return &Terminal{Command: []string{exe, "-test.run=^TestConPTYHelper$"}, Env: []string{helperEnv + "=" + way}}
}

// pump feeds a terminal's news to the app as its update loop would: the
// commands run, their messages handled one at a time.
type pump struct {
	t    *testing.T
	ta   *testApp
	term *Terminal
	msgs chan tea.Msg
}

func newPump(t *testing.T, ta *testApp, term *Terminal) *pump {
	return &pump{t: t, ta: ta, term: term, msgs: make(chan tea.Msg, 16)}
}

func (p *pump) run(c tea.Cmd) {
	if c != nil {
		go func() { p.msgs <- c() }()
	}
}

// until feeds the app until done reports true.
func (p *pump) until(done func() bool) {
	p.t.Helper()
	deadline := time.After(15 * time.Second)
	for !done() {
		select {
		case msg := <-p.msgs:
			switch m := msg.(type) {
			case tea.BatchMsg:
				for _, c := range m {
					p.run(c)
				}
			case termOutput, termExited:
				p.run(p.ta.Update2(m))
			}
		case <-deadline:
			p.t.Fatalf("timed out; the program's screen:\n%s", screenOf(p.term))
		}
	}
}

func screenOf(term *Terminal) string {
	term.mu.Lock()
	defer term.mu.Unlock()
	return strings.Join(term.vt().text(), "\n")
}

func shows(term *Terminal, s string) func() bool {
	return func() bool { return strings.Contains(screenOf(term), s) }
}

// TestConPTYRuns: on Windows a terminal runs its program in a ConPTY: in the
// folder given, with the app's widths (tdp E5), what it prints on the
// screen, what is sent going to it, and it ends as it ends (tdp terminal).
func TestConPTYRuns(t *testing.T) {
	withWidths(t, 2, 1)
	ta := newTestApp(t)
	term := helper(t, "echo")
	term.Dir = filepath.Join(t.TempDir(), "conpty-dir")
	if err := os.Mkdir(term.Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	var ended []error
	term.OnExit = func(err error) tea.Cmd { ended = append(ended, err); return nil }
	p := newPump(t, ta, term)
	p.run(term.Start())
	if term.Exited() {
		t.Fatalf("it did not start: %v", term.err)
	}
	p.until(shows(term, "hello 21 conpty-dir"))
	term.Send([]byte("abc\r"))
	p.until(shows(term, "got abc"))
	term.Send([]byte("quit\r"))
	p.until(term.Exited)
	if len(ended) != 1 || ended[0] != nil {
		t.Errorf("OnExit got %v, want one nil", ended)
	}
	term.Send([]byte("late\r")) // ended: nothing
}

// TestConPTYResize: the program sees the size the terminal is drawn at, and
// the new one after a resize.
func TestConPTYResize(t *testing.T) {
	ta := newTestApp(t)
	term := helper(t, "echo")
	defer term.Stop()
	p := newPump(t, ta, term)
	p.run(term.Start())
	p.until(shows(term, "hello"))
	term.Send([]byte("size\r"))
	p.until(shows(term, "size 80x24"))
	term.resize(100, 30)
	// The new size and the line reach the console by different ways: ask
	// until it has the new one.
	for i := 0; !shows(term, "size 100x30")(); i++ {
		if i == 100 {
			t.Fatalf("the program never saw 100x30; its screen:\n%s", screenOf(term))
		}
		term.Send([]byte("size\r"))
		time.Sleep(100 * time.Millisecond)
	}
}

// TestConPTYProgramPath: a program is found as os/exec finds it: a path
// without its extension gets one from PATHEXT, and a relative one is taken
// from the terminal's folder.
func TestConPTYProgramPath(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if got, err := programPath(strings.TrimSuffix(exe, ".exe"), ""); err != nil || !strings.EqualFold(got, exe) {
		t.Errorf("without .exe: %q, %v; want %q", got, err, exe)
	}
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	tool := filepath.Join(dir, "bin", "tool.exe")
	if err := os.WriteFile(tool, nil, 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := programPath(`bin\tool`, dir); err != nil || !strings.EqualFold(got, tool) {
		t.Errorf(`bin\tool from %s: %q, %v; want %q`, dir, got, err, tool)
	}
}

// TestConPTYExitStatus: a program that fails ends the terminal with its exit
// status, as on unix.
func TestConPTYExitStatus(t *testing.T) {
	ta := newTestApp(t)
	term := helper(t, "exit3")
	var got error
	term.OnExit = func(err error) tea.Cmd { got = err; return nil }
	p := newPump(t, ta, term)
	p.run(term.Start())
	p.until(term.Exited)
	var ee *exec.ExitError
	if !errors.As(got, &ee) || ee.ExitCode() != 3 {
		t.Errorf("OnExit got %v, want exit status 3", got)
	}
}

// TestConPTYStop: Stop closes the console as closing its window does, and
// the program in it ends; Stop itself does not wait for that.
func TestConPTYStop(t *testing.T) {
	ta := newTestApp(t)
	term := helper(t, "wait")
	p := newPump(t, ta, term)
	p.run(term.Start())
	p.until(shows(term, "waiting"))
	began := time.Now()
	term.Stop()
	if d := time.Since(began); d > 100*time.Millisecond {
		t.Errorf("Stop took %v", d)
	}
	p.until(term.Exited)
}

// TestConPTYDefaultShell: with no Command a terminal runs the user's shell,
// on Windows the command interpreter (%ComSpec%).
func TestConPTYDefaultShell(t *testing.T) {
	withWidths(t, 2, 1)
	ta := newTestApp(t)
	term := &Terminal{}
	var got error
	term.OnExit = func(err error) tea.Cmd { got = err; return nil }
	p := newPump(t, ta, term)
	p.run(term.Start())
	term.Send([]byte("echo conpty%TERMINU__ICON_WIDTH%\r"))
	p.until(shows(term, "conpty2"))
	term.Send([]byte("exit 4\r"))
	p.until(term.Exited)
	var ee *exec.ExitError
	if !errors.As(got, &ee) || ee.ExitCode() != 4 {
		t.Errorf("OnExit got %v, want exit status 4", got)
	}
}

// TestConPTYPopup: a terminal popup on Windows: the program's screen in the
// popup, the keys pressed going to it, and the popup closing when it ends
// (tdp terminal).
func TestConPTYPopup(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(100, 30)
	term := helper(t, "echo")
	term.Title, term.Ends = "Edit", true
	p := newPump(t, ta, term)
	p.run(ta.OpenTerminal(term))
	frameShows := func(s string) func() bool {
		return func() bool { ta.settle(); return strings.Contains(ta.screenText(), s) }
	}
	p.until(frameShows("hello"))
	if _, ok := ta.topView().(*termView); !ok {
		t.Fatal("no terminal popup")
	}
	ta.press("x", "y", "Enter")
	p.until(frameShows("got xy"))
	ta.press("q", "u", "i", "t", "Enter")
	p.until(term.Exited)
	ta.settle()
	if _, ok := ta.topView().(*termView); ok {
		t.Error("the popup stays after its program ended")
	}
}
