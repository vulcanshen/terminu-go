//go:build unix

package terminu

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
)

// TestE5ProbePipedStdin: with input piped in and the screen a terminal
// (git log | app), the app still probes at startup (tdp E5): the replies are
// read from the controlling terminal and the pipe is left as it was. The
// test runs itself again in a PTY, stdin a pipe, and answers as a terminal
// where an icon takes two cells and a powerline symbol one.
func TestE5ProbePipedStdin(t *testing.T) {
	if os.Getenv("TERMINU_TEST_PROBE_CHILD") == "1" {
		ans, ok := askTerminal(true)
		rest, _ := io.ReadAll(os.Stdin)
		fmt.Fprintf(os.Stderr, "ok=%v icon=%d powerline=%d rest=%q", ok, ans.icon, ans.powerline, rest)
		return
	}
	ptm, tty, err := pty.Open()
	if err != nil {
		t.Skip("no PTY here:", err)
	}
	defer ptm.Close()
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pw.WriteString("commit 1597a28\n"); err != nil {
		t.Fatal(err)
	}
	pw.Close()
	var errOut bytes.Buffer
	cmd := exec.Command(os.Args[0], "-test.run=^TestE5ProbePipedStdin$")
	cmd.Env = append(os.Environ(), "TERMINU_TEST_PROBE_CHILD=1")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = pr, tty, &errOut
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Ctty: 1}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pr.Close()
	tty.Close()

	asked := make(chan string, 1)
	go func() { // the terminal: everything the app printed, until DA1's query
		var got []byte
		buf := make([]byte, 1024)
		for !bytes.Contains(got, []byte("\x1b[c")) {
			n, err := ptm.Read(buf)
			if err != nil {
				break
			}
			got = append(got, buf[:n]...)
		}
		asked <- string(got)
		io.Copy(io.Discard, ptm)
	}()
	select {
	case got := <-asked:
		if !strings.Contains(got, "\x1b[6n") {
			cmd.Process.Kill()
			cmd.Wait()
			t.Fatalf("with stdin a pipe, nothing was asked: %q; the child said %q", got, errOut.String())
		}
	case <-time.After(10 * time.Second):
		cmd.Process.Kill()
		cmd.Wait()
		t.Fatalf("with stdin a pipe, nothing was asked; the child said %q", errOut.String())
	}
	if _, err := ptm.WriteString("\x1b[1;3R\x1b[1;2R\x1b[?62;22c"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("%v: %s", err, errOut.String())
	}
	if want := `ok=true icon=2 powerline=1 rest="commit 1597a28\n"`; errOut.String() != want {
		t.Errorf("the child said %q, want %q", errOut.String(), want)
	}
}
