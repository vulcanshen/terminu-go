//go:build unix

package terminu

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
)

// hangupDir, set, makes TestHangupEndsRun the app it runs: a copy of this
// test binary in a PTY of its own.
const hangupDir = "TERMINU_TEST_HANGUP"

// TestHangupEndsRun: closing the terminal (SIGHUP) ends Run as leaving does,
// returning nil, so what main does after Run still runs and the app ends
// well; the program in the app's Terminal is hung up first, as a closed
// terminal window does, so main does not wait on it for ever (tgo #4).
func TestHangupEndsRun(t *testing.T) {
	if dir := os.Getenv(hangupDir); dir != "" {
		hangupApp(dir)
		return
	}
	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestHangupEndsRun$")
	cmd.Env = append(os.Environ(), hangupDir+"="+dir, FamilyIconWidth+"=1", FamilyPowerlineWidth+"=1")
	// Its stderr is a file, for its goroutines when it does not end.
	errf, err := os.Create(filepath.Join(dir, "stderr"))
	if err != nil {
		t.Fatal(err)
	}
	defer errf.Close()
	cmd.Stderr = errf
	// Nothing reads the PTY: a read waiting on it would hold the close below
	// off until it came back (Go closes a file once its reads are done), and
	// the app would never be hung up.
	f, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: 80, Rows: 24})
	if err != nil {
		t.Fatal(err)
	}
	if !waitForFile(filepath.Join(dir, "ready"), 10*time.Second) {
		cmd.Process.Kill()
		after, _ := os.ReadFile(filepath.Join(dir, "after"))
		t.Fatalf("the program in the app's Terminal never started; after Run: %q", after)
	}
	f.Close() // the terminal window closes
	ended := make(chan error, 1)
	go func() { ended <- cmd.Wait() }()
	select {
	case err := <-ended:
		if err != nil {
			t.Errorf("the app did not end well: %v", err)
		}
	case <-time.After(10 * time.Second):
		cmd.Process.Signal(syscall.SIGQUIT) // it dies telling where it is stuck
		<-ended
		dump, _ := os.ReadFile(filepath.Join(dir, "stderr"))
		t.Fatalf("the app never ended\n%s", dump)
	}
	after, _ := os.ReadFile(filepath.Join(dir, "after"))
	if want := "Run returned <nil>; the shell was hung up: true"; string(after) != want {
		t.Errorf("after Run: %q, want %q", after, want)
	}
}

// hangupApp is the app: a Terminal running a shell that notes when it is
// ready and when it is hung up; after Run it waits a while to be told the
// shell was hung up, as main waiting on a session would, and notes it.
func hangupApp(dir string) {
	sh := &Terminal{Command: []string{"/bin/sh", "-c",
		`trap 'echo > "$0/hup"; exit 0' HUP; echo > "$0/ready"; while :; do sleep 0.1; done`, dir}}
	a := New("hangup")
	a.Splash = func(int, int, int) ([]string, bool) { return []string{"hangup"}, true }
	a.AddScreen(&Screen{Name: "Shell", Panels: []*Panel{{Label: "Shell", Body: sh}}})
	a.Start = sh.Start
	err := a.Run()
	hup := waitForFile(filepath.Join(dir, "hup"), 3*time.Second)
	os.WriteFile(filepath.Join(dir, "after"), fmt.Appendf(nil, "Run returned %v; the shell was hung up: %v", err, hup), 0o600)
}

// waitForFile reports whether path comes to be there within d.
func waitForFile(path string, d time.Duration) bool {
	for end := time.Now().Add(d); time.Now().Before(end); time.Sleep(20 * time.Millisecond) {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

// TestTerminalEndedIsForgotten: a program that ends leaves the ones a hangup
// stops (tgo #4).
func TestTerminalEndedIsForgotten(t *testing.T) {
	tm := &Terminal{Command: []string{"/bin/sh", "-c", "read line"}}
	if tm.Start() == nil {
		t.Fatal("it did not start")
	}
	running.Lock()
	in := running.procs[tm.proc]
	running.Unlock()
	if !in {
		t.Error("a program started is not among the running")
	}
	io.WriteString(tm.in, "\n") // it ends
	for range tm.out {
	}
	running.Lock()
	defer running.Unlock()
	if running.procs[tm.proc] {
		t.Error("a program that ended is still among the running")
	}
}
