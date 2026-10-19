package terminu

import (
	"bytes"
	"os"
)

// eraseRight is what bubbletea's renderer appends to a line it measures
// shorter than the terminal: erase to the end of the line.
var eraseRight = []byte("\x1b[K")

// screenOutput is the terminal the App draws on, minus every eraseRight.
//
// bubbletea v1 measures each line with x/ansi, where an icon is one cell.
// When the terminal gives icons two cells, a full line measures short and the
// renderer sends eraseRight after it, with the cursor on the last column.
// Terminals that follow xterm (Ghostty among them) then erase that last cell:
// the right frame. Every line the App draws is the full width (L4), so the
// erase never has anything left to clear and dropping it is safe.
//
// It embeds the file so bubbletea still finds the terminal (Fd) for raw mode
// and the window size.
type screenOutput struct{ *os.File }

func (o screenOutput) Write(p []byte) (int, error) {
	if clip := takeClipboard(); clip != nil {
		// What a copy put on the clipboard goes first, in the same write.
		_, err := o.File.Write(append(clip, bytes.ReplaceAll(p, eraseRight, nil)...))
		return len(p), err
	}
	if !bytes.Contains(p, eraseRight) {
		return o.File.Write(p)
	}
	// The renderer writes a whole frame at once, so eraseRight is never split
	// across two writes.
	_, err := o.File.Write(bytes.ReplaceAll(p, eraseRight, nil))
	return len(p), err
}
