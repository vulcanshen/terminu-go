//go:build unix

package terminu

import (
	"os"
	"time"

	"github.com/charmbracelet/x/term"
	"golang.org/x/sys/unix"
)

// probeTimeout bounds the wait for a terminal that answers nothing at all,
// not even DA1.
const probeTimeout = 300 * time.Millisecond

// askTerminal sends the probe and reads the replies until DA1's, in raw mode.
// It is false when stdin or stdout is not a terminal or nothing came back.
func askTerminal(widths bool) (answers, bool) {
	in, out := os.Stdin, os.Stdout
	if !term.IsTerminal(in.Fd()) || !term.IsTerminal(out.Fd()) {
		return answers{}, false
	}
	state, err := term.MakeRaw(in.Fd())
	if err != nil {
		return answers{}, false
	}
	defer term.Restore(in.Fd(), state)
	if _, err := out.WriteString(probeQueries(widths)); err != nil {
		return answers{}, false
	}
	defer out.WriteString("\r\x1b[2K") // wipe the probe before Bubble Tea takes the screen

	fd := int(in.Fd())
	deadline := time.Now().Add(probeTimeout)
	var buf []byte
	chunk := make([]byte, 256)
	for {
		ms := int(time.Until(deadline).Milliseconds())
		if ms <= 0 {
			break
		}
		n, err := unix.Poll([]unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}, ms)
		if err == unix.EINTR {
			continue
		}
		if err != nil || n == 0 {
			break
		}
		k, err := unix.Read(fd, chunk)
		if err != nil || k <= 0 {
			break
		}
		buf = append(buf, chunk[:k]...)
		if ans, done := parseAnswers(string(buf)); done {
			return ans, true
		}
	}
	ans, _ := parseAnswers(string(buf))
	return ans, len(buf) > 0
}
