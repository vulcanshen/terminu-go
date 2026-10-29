// Package testhook is how tdptest reaches into the lib: terminu fills these
// in when it loads. Nothing outside this module can import it, so none of it
// is the lib's API.
package testhook

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Clock is a test's time: it moves only when the test moves it.
type Clock interface{ Now() time.Time }

// Timer is what a tick on a test clock gives back instead of waiting: the
// test fires Fn once its time reaches D from when it came back.
type Timer struct {
	D  time.Duration
	Fn func(time.Time) tea.Msg
}

// Every app below is a *terminu.App.
var (
	// UseClock puts the app on c: its ticks give back a Timer.
	UseClock func(app any, c Clock)
	// Prepare does what Run checks and sets before the program starts,
	// short of asking the terminal anything.
	Prepare func(app any) error
	// Animating reports a popup, toast or splash still playing.
	Animating func(app any) bool
	// Layers is how many popups are open, the toast and the closing ones
	// not counted.
	Layers func(app any) int
	// Toast reports a toast on the screen that Esc takes, not closing: not
	// the inspect toast (tdp toast.4).
	Toast func(app any) bool
	// Typing reports keys going to a typing row, a mode or a PTY: ? and q
	// are not the core keys there (tdp K8).
	Typing func(app any) bool
	// Screens is the app's screens, a []*terminu.Screen.
	Screens func(app any) any
	// MinSize is the app's least size (tdp L1).
	MinSize func(app any) (w, h int)
	// Clipboard is the text waiting to go to the clipboard, taken; ok false
	// when there is none.
	Clipboard func() (text string, ok bool)
	// Stop ends every program the app's terminals run.
	Stop func(app any)
	// Timers is the lib's own timers started outside a message (Driver.Do),
	// which Update sends at its end.
	Timers func(app any) tea.Cmd
	// Once is one of the lib's timers that run once, sent by the app and the
	// lib both: it comes back at once when its timer does.
	Once func() tea.Cmd
	// LineBreak reports a drawn row showing a line break by its name in Red
	// (\n, \r): in a name or a short text.
	LineBreak func(row string) bool
)
