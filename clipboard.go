package terminu

import (
	"encoding/base64"
	"sync"
)

// The clipboard goes through the terminal (OSC 52): the text waits for the
// next frame the App draws and is sent just before it, in the same write, so
// it never lands in the middle of one.
var (
	clipMu   sync.Mutex
	clipNext []byte
)

// Clipboard puts text on the terminal's clipboard (OSC 52), as a viewer's
// selection mode copies by default (tdp note). It is sent with
// the next frame.
func Clipboard(text string) {
	clipMu.Lock()
	defer clipMu.Unlock()
	clipNext = []byte("\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(text)) + "\a")
}

// takeClipboard is what waits to go to the clipboard, nil when nothing does;
// it is then gone.
func takeClipboard() []byte {
	clipMu.Lock()
	defer clipMu.Unlock()
	b := clipNext
	clipNext = nil
	return b
}
