package terminu

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

// ElevateTool is a tool that raises privileges, the app's to choose by
// platform (tdp terminal.7).
type ElevateTool int

const (
	// Sudo reads the password on its standard input: the lib asks for it in
	// a password popup and hands it over.
	Sudo ElevateTool = iota
	// Doas cannot: it runs in a terminal popup and asks there itself.
	Doas
)

// Elevate is a command to run with raised privileges, such as a copy over
// /etc/hosts (tdp terminal.7). The app gives the command and the lib does
// the rest: it runs at once when no password is needed (sudo -n); otherwise
// sudo's password is asked for in a password popup, PASSWORD, the unlock
// look, and handed to sudo on its standard input and nowhere else. It goes in
// no file, not in the App Log, not on the screen and is sent nowhere; nothing
// gives it to the app. A wrong one keeps the popup open, the dots cleared and
// Wrong password on its error row; Esc cancels, and the command does not
// run. Doas asks for it itself, in a terminal popup that closes when it
// ends. On Windows the app raises privileges itself (gsudo, or the sudo
// Windows has, the system asking with UAC).
type Elevate struct {
	Tool ElevateTool
	// Command is what runs with raised privileges: the program and its
	// arguments ("cp", temp, "/etc/hosts").
	Command []string
	// Verb is what Enter on the password does, for the hint ("write"); run
	// when unset.
	Verb string
	// Subject is what it acts on ("/etc/hosts"), after the tool's name in
	// the title of doas's terminal popup.
	Subject string
	// Done gets how it ended: nil when the command ran and succeeded, an
	// *ElevateError when it or the tool failed, for the app's error popup
	// (tdp F5). Once the password is taken the command runs to its end, and
	// Done comes even when the popup was closed meanwhile.
	Done func(err error) tea.Cmd
	// OnCancel runs when the user cancels before the command runs: Esc on
	// the password popup, or leaving doas's terminal popup.
	OnCancel func() tea.Cmd
}

// ElevateError is a command run with raised privileges that failed, or the
// tool that did: Lines are what it printed, blank lines left out, and Err is
// how it ended.
type ElevateError struct {
	Lines []string
	Err   error
}

// Error is the first line it printed, or how it ended when it printed none.
func (e *ElevateError) Error() string {
	if len(e.Lines) > 0 {
		return e.Lines[0]
	}
	return e.Err.Error()
}

func (e *ElevateError) Unwrap() error { return e.Err }

// toolPath finds a tool on the PATH; the tests stand a script in for it.
var toolPath = exec.LookPath

// Elevate runs e's command with raised privileges (tdp terminal.7).
func (a *App) Elevate(e Elevate) tea.Cmd {
	ev := &elevation{e: e}
	if runtime.GOOS == "windows" {
		return ev.finish(&ElevateError{Err: errors.New("on Windows, raise privileges with gsudo or the sudo Windows has")})
	}
	return func() tea.Msg {
		name := "sudo"
		if e.Tool == Doas {
			name = "doas"
		}
		path, err := toolPath(name)
		if err != nil {
			return elevateRan{ev: ev, done: true, err: &ElevateError{Err: err}}
		}
		ev.path = path
		argv := append([]string{"-n", "--"}, e.Command...)
		if e.Tool == Doas {
			if _, err := tool(context.Background(), path, nil, false, argv...); err == nil {
				return elevateRan{ev: ev, done: true}
			}
			return elevateRan{ev: ev, ask: true}
		}
		// sudo -n -v says whether a password is needed without running the
		// command, so that it never runs twice.
		if _, err := tool(context.Background(), path, nil, false, "-n", "-v"); err != nil {
			return elevateRan{ev: ev, ask: true}
		}
		_, err = tool(context.Background(), path, nil, false, argv...)
		return elevateRan{ev: ev, done: true, err: err}
	}
}

// elevation is an Elevate under way.
type elevation struct {
	e    Elevate
	path string
	li   *lineInput // the password popup, once it is open
	mu   sync.Mutex
	// stop ends the check of a password under way; started is the command
	// running, which ends in Done whatever the popup does; cancelled is the
	// popup closed before that.
	stop      context.CancelFunc
	started   bool
	cancelled bool
}

// elevateRan is a step of an Elevate done, back on the update loop.
type elevateRan struct {
	ev      *elevation
	ask     bool // a password is needed
	wrong   bool // the password was wrong
	dropped bool // cancelled meanwhile: nothing to show
	done    bool // it ended: Done gets err
	err     error
}

func (m elevateRan) step(a *App) tea.Cmd {
	ev := m.ev
	switch {
	case m.ask && ev.e.Tool == Doas:
		return ev.terminal(a)
	case m.ask:
		return ev.ask(a)
	case m.dropped:
		return nil
	}
	li := ev.li
	open := li != nil && slices.ContainsFunc(a.stack, func(l *layer) bool { return l.view == li && !l.closing })
	if li != nil {
		li.busy = false
		li.ed.set("") // the dots cleared: the password is dropped
	}
	if m.wrong {
		if !open {
			return run(ev.e.OnCancel)
		}
		li.err = "Wrong password"
		return nil
	}
	var cmd tea.Cmd
	if open {
		for _, l := range a.stack {
			if l.view == li {
				l.answered = true
			}
		}
		cmd = a.closeView(li)
	}
	return tea.Batch(cmd, ev.finish(m.err))
}

// finish is Done with how it ended.
func (ev *elevation) finish(err error) tea.Cmd {
	if ev.e.Done == nil {
		return nil
	}
	return ev.e.Done(err)
}

// ask opens the password popup: the unlock look, titled PASSWORD
// (tdp terminal.7).
func (ev *elevation) ask(a *App) tea.Cmd {
	ev.li = &lineInput{kind: kindPassword, title: "Password", name: "Password", verb: cmpOr(ev.e.Verb, "run"),
		unlock: true, ed: newEditor(""), verify: ev.verify, onCancel: ev.cancel,
		check: func(v string) error {
			if v == "" {
				return ErrEmpty
			}
			return nil
		},
		commit: func(string, bool) error { return nil },
		done:   func(string, bool) tea.Cmd { return nil },
	}
	return a.open(ev.li)
}

// cancel is the popup closed with no answer: the check under way stops and
// the command does not run, unless it runs already.
func (ev *elevation) cancel() tea.Cmd {
	ev.mu.Lock()
	defer ev.mu.Unlock()
	if ev.started {
		return nil // Done says how it ended
	}
	ev.cancelled = true
	if ev.stop != nil {
		ev.stop()
	}
	return run(ev.e.OnCancel)
}

// verify hands password to sudo off the update loop: first alone (sudo -v),
// so that a wrong one is told apart from the command failing, then with the
// command. The password is dropped once it is handed over.
func (ev *elevation) verify(password string) tea.Cmd {
	ctx, stop := context.WithCancel(context.Background())
	ev.mu.Lock()
	ev.stop = stop
	ev.mu.Unlock()
	secret := []byte(password + "\n")
	return func() tea.Msg {
		defer clear(secret)
		defer stop()
		prompts, err := tool(ctx, ev.path, secret, true, "-v")
		ev.mu.Lock()
		switch {
		case ev.cancelled:
			ev.mu.Unlock()
			return elevateRan{ev: ev, dropped: true}
		case prompts >= 2:
			ev.mu.Unlock()
			return elevateRan{ev: ev, wrong: true}
		case err != nil:
			ev.mu.Unlock()
			return elevateRan{ev: ev, done: true, err: err} // sudo refused: not for this user
		}
		ev.started = true
		ev.mu.Unlock()
		wrong, err := ev.runCommand(secret)
		if wrong {
			ev.mu.Lock()
			ev.started = false
			ev.mu.Unlock()
			return elevateRan{ev: ev, wrong: true}
		}
		return elevateRan{ev: ev, done: true, err: err}
	}
}

// runCommand runs the command once the password is known right. When sudo
// keeps it a while, without it (sudo -n); when it keeps it for no time
// (timestamp_timeout=0), sudo asks again and gets it again. Either way a
// failure now is the command's.
func (ev *elevation) runCommand(secret []byte) (wrong bool, err error) {
	ctx := context.Background()
	if _, err := tool(ctx, ev.path, nil, false, "-n", "-v"); err == nil {
		_, err = tool(ctx, ev.path, nil, false, append([]string{"-n", "--"}, ev.e.Command...)...)
		return false, err
	}
	prompts, err := tool(ctx, ev.path, secret, false, append([]string{"--"}, ev.e.Command...)...)
	return prompts >= 2, err
}

// terminal runs doas in a terminal popup, where it asks for the password
// itself; the popup closes when it ends (tdp terminal.7).
func (ev *elevation) terminal(a *App) tea.Cmd {
	left := false
	t := &Terminal{Title: "doas", Subject: ev.e.Subject, Command: append([]string{ev.path, "--"}, ev.e.Command...), Ends: true}
	t.OnLeave = func() tea.Cmd {
		left = true
		return run(ev.e.OnCancel)
	}
	t.OnExit = func(err error) tea.Cmd {
		if left {
			return nil
		}
		if err != nil {
			return ev.finish(&ElevateError{Err: err})
		}
		return ev.finish(nil)
	}
	return a.OpenTerminal(t)
}

// tool runs the tool at path with args. With a secret it gets -S and a
// prompt of the lib's own (-p), and the secret on its standard input; prompts
// is how often it asked, counted by that prompt, not by anything it prints in
// the user's language. Asked a second time, the secret was wrong: the tool
// is stopped there, before it reads any more. hold keeps its standard input
// open meanwhile; otherwise it ends after the secret, for the command run
// after it. A failure is an *ElevateError.
func tool(ctx context.Context, path string, secret []byte, hold bool, args ...string) (prompts int, err error) {
	out := &toolOutput{}
	if secret != nil {
		b := make([]byte, 16)
		_, _ = rand.Read(b)
		out.prompt = "terminu-" + hex.EncodeToString(b)
		args = append([]string{"-S", "-p", out.prompt}, args...)
	}
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Stdout, cmd.Stderr = out, out
	var stdin io.WriteCloser
	if secret != nil {
		if stdin, err = cmd.StdinPipe(); err != nil {
			return 0, &ElevateError{Err: err}
		}
		out.again = func() {
			_ = cmd.Process.Kill()
			_ = stdin.Close() // a tool it may not stop gets nothing more to read
		}
	}
	if err := cmd.Start(); err != nil {
		return 0, &ElevateError{Err: err}
	}
	if stdin != nil {
		_, _ = stdin.Write(secret)
		if hold {
			defer context.AfterFunc(ctx, func() { _ = stdin.Close() })()
		} else {
			_ = stdin.Close()
		}
	}
	err = cmd.Wait()
	prompts = out.count()
	if err != nil {
		return prompts, &ElevateError{Lines: out.lines(), Err: err}
	}
	return prompts, nil
}

// toolOutput is what a tool prints, its prompts counted.
type toolOutput struct {
	mu      sync.Mutex
	prompt  string
	buf     bytes.Buffer
	prompts int
	again   func() // it asks a second time
}

func (o *toolOutput) Write(p []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.buf.Write(p)
	if o.prompt == "" {
		return len(p), nil
	}
	n := bytes.Count(o.buf.Bytes(), []byte(o.prompt))
	if n >= 2 && o.prompts < 2 && o.again != nil {
		o.again()
	}
	o.prompts = n
	return len(p), nil
}

func (o *toolOutput) count() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.prompts
}

// lines are what it printed, the prompts and blank lines left out.
func (o *toolOutput) lines() []string {
	o.mu.Lock()
	text := o.buf.String()
	o.mu.Unlock()
	if o.prompt != "" {
		text = strings.ReplaceAll(text, o.prompt, "\n")
	}
	var lines []string
	for l := range strings.SplitSeq(text, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}
