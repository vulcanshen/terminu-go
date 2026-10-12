//go:build !unix

package terminu

import (
	"errors"
	"io"
)

// startPTY: a PTY needs unix here.
func startPTY([]string, string, []string, int, int) (io.ReadWriter, process, error) {
	return nil, nil, errors.New("a terminal needs a unix system")
}
