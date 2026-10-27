package terminu

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// fakeSudo stands in for sudo: FAKE_MODE nopass needs no password, keep
// keeps a right one a while, zero keeps it for no time (timestamp_timeout=0),
// refuse takes the right one but is not for this user. The right password is
// "right"; three tries, as sudo's default. After a wrong one it waits a second
// for an end of its input before it asks again: sudo would read it at the
// next prompt. It logs what it is run with, each password it reads (try) and
// an end of its input (eof), never the password.
const fakeSudo = `#!/usr/bin/env bash
log() { echo "$*" >> "$FAKE_DIR/log"; }
log "sudo $*"
ok() { [ "$FAKE_MODE" = nopass ] || { [ "$FAKE_MODE" = keep ] && [ -e "$FAKE_DIR/kept" ]; }; }
ask() {
	n=0
	while [ $n -lt 3 ]; do
		printf "%s" "$1" >&2
		if ! read -r pw; then log eof; exit 1; fi
		log try
		if [ "$pw" = right ]; then
			if [ "$FAKE_MODE" = refuse ]; then echo "you are not in the sudoers file." >&2; exit 1; fi
			if [ "$FAKE_MODE" = keep ]; then touch "$FAKE_DIR/kept"; fi
			return 0
		fi
		echo "Sorry, try again." >&2
		read -r -t 1 more
		if [ $? -eq 1 ]; then log eof; exit 1; fi
		n=$((n + 1))
	done
	echo "sudo: 3 incorrect password attempts" >&2
	exit 1
}
case "$1" in
-n)
	shift
	ok || { echo "sudo: a password is required" >&2; exit 1; }
	if [ "$1" = -v ]; then exit 0; fi
	shift
	exec "$@"
	;;
-S)
	prompt=$3
	shift 3
	ok || ask "$prompt"
	if [ "$1" = -v ]; then exit 0; fi
	shift
	exec "$@"
	;;
esac
exit 2
`

// fakeDoas stands in for doas: -n works only with FAKE_MODE nopass.
const fakeDoas = `#!/bin/sh
echo "doas $*" >> "$FAKE_DIR/log"
if [ "$1" = -n ]; then
	[ "$FAKE_MODE" = nopass ] || { echo "doas: Authorization required" >&2; exit 1; }
	shift
fi
shift
exec "$@"
`

// elevateApp is the test app with sudo and doas stood in for, in mode.
type elevateApp struct {
	*testApp
	dir  string
	done []error
	ends int // Done's calls
	off  int // OnCancel's calls
}

func newElevateApp(t *testing.T, mode string) *elevateApp {
	t.Helper()
	ea := &elevateApp{testApp: newTestApp(t), dir: t.TempDir()}
	for name, script := range map[string]string{"sudo": fakeSudo, "doas": fakeDoas} {
		if err := os.WriteFile(filepath.Join(ea.dir, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("FAKE_DIR", ea.dir)
	t.Setenv("FAKE_MODE", mode)
	toolPath = func(name string) (string, error) { return filepath.Join(ea.dir, name), nil }
	t.Cleanup(func() { toolPath = exec.LookPath })
	return ea
}

// elevate is an Elevate of command, its ends kept.
func (ea *elevateApp) elevate(tool ElevateTool, command ...string) Elevate {
	return Elevate{Tool: tool, Command: command, Subject: "/etc/hosts", Verb: "write",
		Done:     func(err error) tea.Cmd { ea.done = append(ea.done, err); ea.ends++; return nil },
		OnCancel: func() tea.Cmd { ea.off++; return nil }}
}

// writes is a command that notes it ran, and what came on its standard input.
func (ea *elevateApp) writes() []string {
	return []string{"/bin/sh", "-c", `cat > "$FAKE_DIR/stdin"; echo ran >> "$FAKE_DIR/ran"`}
}

func (ea *elevateApp) file(name string) string {
	b, _ := os.ReadFile(filepath.Join(ea.dir, name))
	return string(b)
}

// typeIn types s a key at a time.
func (ea *elevateApp) typeIn(s string) {
	for _, r := range s {
		ea.press(string(r))
	}
}

// nowhere fails when the password is in the App Log or on the screen.
func (ea *elevateApp) nowhere(t *testing.T, password string) {
	t.Helper()
	for _, e := range ea.logs.entries {
		if strings.Contains(e.text, password) {
			t.Errorf("the password is in the App Log: %q", e.text)
		}
	}
	if strings.Contains(ea.screenText(), password) {
		t.Errorf("the password is on the screen:\n%s", ea.screenText())
	}
}

// TestElevateNoPassword: when none is needed it runs at once, nothing opened,
// once (tdp terminal.7, terminal-24).
func TestElevateNoPassword(t *testing.T) {
	ea := newElevateApp(t, "nopass")
	ea.deliver(ea.Elevate(ea.elevate(Sudo, ea.writes()...)))
	if ea.top() != nil || ea.ends != 1 || ea.done[0] != nil || ea.file("ran") != "ran\n" {
		t.Errorf("popup %v, Done %v, ran %q", ea.top() != nil, ea.done, ea.file("ran"))
	}
	if log := ea.file("log"); !strings.HasPrefix(log, "sudo -n -v\nsudo -n -- /bin/sh") {
		t.Errorf("asked sudo %q", log)
	}
}

// TestElevatePassword: a password needed is asked for in a password popup,
// the unlock look, PASSWORD; it runs off the update loop, the loading icon
// after the title, every key but Esc swallowed; a wrong one keeps the popup,
// the dots cleared and Wrong password on its error row, sudo stopped before
// it reads again; the right one runs the command once and closes it. The
// password is nowhere but sudo's standard input (tdp terminal.7,
// terminal-24, popup.4).
func TestElevatePassword(t *testing.T) {
	ea := newElevateApp(t, "keep")
	ea.deliver(ea.Elevate(ea.elevate(Sudo, ea.writes()...)))
	li, ok := ea.topView().(*lineInput)
	if !ok || !li.unlock || li.kind != kindPassword {
		t.Fatalf("no password popup: %T", ea.topView())
	}
	if text := ea.screenText(); !strings.Contains(text, "PASSWORD") || strings.Contains(text, "PASSWORD:") || !strings.Contains(text, "Enter:write") {
		t.Errorf("the popup:\n%s", text)
	}
	ea.press("Enter")
	if li.err != "Password can't be empty" || li.busy {
		t.Errorf("empty: %q, busy %v", li.err, li.busy)
	}
	ea.typeIn("wrong1")
	cmd := ea.press("Enter")
	if !li.isLoading() || !strings.ContainsAny(ea.Frame()[ea.top().y], strings.Join(loadingFrames[:], "")) {
		t.Errorf("checking: loading %v, title %q", li.isLoading(), StripStyles(ea.Frame()[ea.top().y]))
	}
	ea.press("x", "Backspace")
	if li.ed.value() != "wrong1" {
		t.Errorf("a key went in while it checks: %q", li.ed.value())
	}
	ea.deliver(cmd)
	if ea.topView() != li || li.err != "Wrong password" || !li.ed.empty() || li.busy || ea.ends != 0 {
		t.Errorf("wrong: top %T, error %q, value %q, busy %v, Done %d", ea.topView(), li.err, li.ed.value(), li.busy, ea.ends)
	}
	if !strings.Contains(ea.screenText(), "Wrong password") {
		t.Errorf("not on the error row:\n%s", ea.screenText())
	}
	if log := ea.file("log"); strings.Count(log, "try") != 1 || strings.Contains(log, "eof") || ea.file("ran") != "" {
		t.Errorf("sudo read %q; ran %q", log, ea.file("ran"))
	}
	ea.typeIn("right")
	ea.deliver(ea.press("Enter"))
	if ea.top() != nil || ea.ends != 1 || ea.done[0] != nil || ea.off != 0 {
		t.Errorf("right: popup %v, Done %v, OnCancel %d", ea.top() != nil, ea.done, ea.off)
	}
	if ea.file("ran") != "ran\n" || ea.file("stdin") != "" {
		t.Errorf("ran %q, its input %q", ea.file("ran"), ea.file("stdin"))
	}
	if log := ea.file("log"); !strings.Contains(log, "sudo -n -v\nsudo -n -- /bin/sh") {
		t.Errorf("kept a while, the command runs without the password: %q", log)
	}
	ea.nowhere(t, "right")
	ea.nowhere(t, "wrong1")
}

// TestElevateKeptForNoTime: sudo keeping the password for no time
// (timestamp_timeout=0) gets it again for the command, and the command reads
// none of it (tdp terminal.7).
func TestElevateKeptForNoTime(t *testing.T) {
	ea := newElevateApp(t, "zero")
	ea.deliver(ea.Elevate(ea.elevate(Sudo, ea.writes()...)))
	ea.typeIn("right")
	ea.deliver(ea.press("Enter"))
	if ea.top() != nil || ea.ends != 1 || ea.done[0] != nil {
		t.Errorf("popup %v, Done %v", ea.top() != nil, ea.done)
	}
	if ea.file("ran") != "ran\n" || ea.file("stdin") != "" || strings.Count(ea.file("log"), "try") != 2 {
		t.Errorf("ran %q, its input %q, sudo %q", ea.file("ran"), ea.file("stdin"), ea.file("log"))
	}
}

// TestElevateCommandFails: the command failing is not a wrong password: the
// popup closes and Done gets what it printed, for the app's error popup
// (tdp terminal.7, F5).
func TestElevateCommandFails(t *testing.T) {
	for _, mode := range []string{"keep", "zero", "nopass"} {
		ea := newElevateApp(t, mode)
		ea.deliver(ea.Elevate(ea.elevate(Sudo, "/bin/sh", "-c", "echo; echo cp: not permitted >&2; exit 3")))
		if mode != "nopass" {
			ea.typeIn("right")
			ea.deliver(ea.press("Enter"))
		}
		var ee *ElevateError
		if ea.top() != nil || ea.ends != 1 || !errors.As(ea.done[0], &ee) || ee.Error() != "cp: not permitted" || len(ee.Lines) != 1 {
			t.Errorf("%s: popup %v, Done %v", mode, ea.top() != nil, ea.done)
		}
	}
}

// TestElevateRefused: sudo taking the password but refusing the user is not
// a wrong password: the popup closes and Done gets why, once, with no
// OnCancel (tdp terminal.7).
func TestElevateRefused(t *testing.T) {
	ea := newElevateApp(t, "refuse")
	ea.deliver(ea.Elevate(ea.elevate(Sudo, ea.writes()...)))
	ea.typeIn("right")
	ea.deliver(ea.press("Enter"))
	var ee *ElevateError
	if ea.top() != nil || ea.ends != 1 || ea.off != 0 || !errors.As(ea.done[0], &ee) || ee.Error() != "you are not in the sudoers file." || ea.file("ran") != "" {
		t.Errorf("popup %v, Done %v, OnCancel %d, ran %q", ea.top() != nil, ea.done, ea.off, ea.file("ran"))
	}
}

// TestElevateCancel: Esc on the password popup cancels: the command does not
// run, Done does not come, nothing is logged; Esc while the password is
// checked stops sudo (tdp terminal.7, note.7).
func TestElevateCancel(t *testing.T) {
	ea := newElevateApp(t, "keep")
	ea.deliver(ea.Elevate(ea.elevate(Sudo, ea.writes()...)))
	ea.typeIn("right")
	ea.press("Esc")
	if ea.top() != nil || ea.off != 1 {
		t.Errorf("Esc: popup %v, OnCancel %d", ea.top() != nil, ea.off)
	}
	ea.deliver(ea.Elevate(ea.elevate(Sudo, ea.writes()...)))
	ea.typeIn("right")
	cmd := ea.press("Enter")
	ea.press("Esc")
	ea.deliver(cmd)
	if ea.top() != nil || ea.off != 2 || ea.ends != 0 || ea.file("ran") != "" || len(ea.logs.entries) != 0 {
		t.Errorf("Esc while checking: popup %v, OnCancel %d, Done %d, ran %q, logged %v", ea.top() != nil, ea.off, ea.ends, ea.file("ran"), ea.logs.entries)
	}
	if log := ea.file("log"); strings.Contains(log, "try") {
		t.Errorf("sudo read a password after Esc: %q", log)
	}
}

// TestElevateDoas: doas runs at once when it needs no password; otherwise it
// asks itself, in a terminal popup; leaving it cancels (tdp terminal.7).
func TestElevateDoas(t *testing.T) {
	ea := newElevateApp(t, "nopass")
	ea.deliver(ea.Elevate(ea.elevate(Doas, ea.writes()...)))
	if ea.top() != nil || ea.ends != 1 || ea.done[0] != nil || ea.file("ran") != "ran\n" {
		t.Errorf("no password: popup %v, Done %v, ran %q", ea.top() != nil, ea.done, ea.file("ran"))
	}
	ea = newElevateApp(t, "keep")
	ea.deliver(ea.Elevate(ea.elevate(Doas, "/bin/sh", "-c", "exit 0")))
	tv, ok := ea.topView().(*termView)
	if !ok {
		t.Fatalf("no terminal popup: %T", ea.topView())
	}
	if want := []string{filepath.Join(ea.dir, "doas"), "--", "/bin/sh", "-c", "exit 0"}; strings.Join(tv.t.Command, " ") != strings.Join(want, " ") ||
		!strings.Contains(ea.screenText(), "DOAS: /etc/hosts") {
		t.Errorf("runs %q:\n%s", tv.t.Command, ea.screenText())
	}
	tv.t.OnLeave()
	tv.t.OnExit(errors.New("killed"))
	if ea.off != 1 || ea.ends != 0 {
		t.Errorf("left: OnCancel %d, Done %d", ea.off, ea.ends)
	}
	tv.t.Stop()
}

// TestToolOutput: the prompts are counted by the lib's own prompt, split
// across writes too, and left out of the lines.
func TestToolOutput(t *testing.T) {
	again := 0
	o := &toolOutput{prompt: "terminu-ab", again: func() { again++ }}
	for _, s := range []string{"lecture\nterm", "inu-ab", "Sorry\n", "terminu-ab", "terminu-ab"} {
		o.Write([]byte(s))
	}
	if o.count() != 3 || again != 1 || strings.Join(o.lines(), "|") != "lecture|Sorry" {
		t.Errorf("prompts %d, again %d, lines %q", o.count(), again, o.lines())
	}
}
