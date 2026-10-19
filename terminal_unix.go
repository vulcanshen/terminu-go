//go:build unix

package terminu

import (
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/creack/pty"
)

type unixProcess struct {
	cmd *exec.Cmd
	// mu keeps the update loop resizing the PTY apart from the reader
	// closing it when the program ends; pty is nil once closed.
	mu  sync.Mutex
	pty *os.File
}

func startPTY(argv []string, dir string, env []string, w, h int) (io.ReadWriter, process, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir, cmd.Env = dir, env
	f, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(w), Rows: uint16(h)})
	if err != nil {
		return nil, nil, err
	}
	return f, &unixProcess{cmd: cmd, pty: f}, nil
}

func (p *unixProcess) resize(w, h int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pty != nil {
		_ = pty.Setsize(p.pty, &pty.Winsize{Cols: uint16(w), Rows: uint16(h)})
	}
}

// stop hangs up, as closing a terminal window does.
func (p *unixProcess) stop() {
	if p.cmd.Process != nil {
		_ = p.cmd.Process.Signal(syscall.SIGHUP)
	}
}

func (p *unixProcess) wait() error {
	err := p.cmd.Wait()
	p.mu.Lock()
	p.pty.Close()
	p.pty = nil
	p.mu.Unlock()
	return err
}
