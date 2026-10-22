package terminu

import (
	"encoding/base64"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vulcanshen/terminu-go/internal/testhook"
)

// The App's time is the real clock, or the one a test driving the App gives
// it (tdptest): that one moves only when the test moves it, so a popup's
// animation, a toast's stay and the loading icon play the same on every run.

// now is the App's time now.
func (a *App) now() time.Time {
	if a.testClock != nil {
		return a.testClock.Now()
	}
	return time.Now()
}

// Tick is tea.Tick on the App's clock. A timer of the app's own made with it
// (a debounce, a message that goes after a while) waits, under a test that
// drives the App, until the test moves the time on (tdptest); one made with
// tea.Tick runs on the real clock there too.
func (a *App) Tick(d time.Duration, fn func(time.Time) tea.Msg) tea.Cmd {
	if a.testClock == nil {
		return tea.Tick(d, fn)
	}
	return func() tea.Msg { return testhook.Timer{D: d, Fn: fn} }
}

// after is a lib timer: msg once d has passed on the App's clock.
func (a *App) after(d time.Duration, msg tea.Msg) tea.Cmd {
	return a.Tick(d, func(time.Time) tea.Msg { return msg })
}

func init() {
	testhook.UseClock = func(app any, c testhook.Clock) { app.(*App).testClock = c }
	testhook.Prepare = func(app any) error {
		a := app.(*App)
		if err := a.checkSplash(); err != nil {
			return err
		}
		if err := a.checkKeys(); err != nil {
			return err
		}
		tabCells = a.inputTabWidth()
		return nil
	}
	testhook.Animating = func(app any) bool {
		a := app.(*App)
		if a.anim || a.labelsMoving() {
			return true
		}
		if a.splashOn {
			_, done := a.shown(a.splashFrame, a.w, a.h)
			return !done
		}
		return false
	}
	testhook.Layers = func(app any) int {
		n := 0
		for _, l := range app.(*App).stack {
			if !l.closing {
				n++
			}
		}
		return n
	}
	testhook.Toast = func(app any) bool { t := app.(*App).toast; return t != nil && !t.closing }
	testhook.Typing = func(app any) bool { a := app.(*App); return a.typingOrMode() || a.ptyFocus() != nil }
	testhook.Screens = func(app any) any { return app.(*App).screens }
	testhook.MinSize = func(app any) (int, int) { return app.(*App).minSize() }
	testhook.Clipboard = func() (string, bool) {
		b := takeClipboard()
		if b == nil {
			return "", false
		}
		enc := strings.TrimSuffix(strings.TrimPrefix(string(b), "\x1b]52;c;"), "\a")
		text, _ := base64.StdEncoding.DecodeString(enc)
		return string(text), true
	}
	testhook.Stop = func(app any) {
		a := app.(*App)
		for _, t := range a.terminals {
			t.Stop()
		}
		for _, s := range a.screens {
			for _, p := range s.Panels {
				if t, ok := p.Body.(*Terminal); ok {
					t.Stop()
				}
			}
		}
	}
	testhook.Timers = func(app any) tea.Cmd { return app.(*App).sendTimers() }
	testhook.Once = func() tea.Cmd { return once(nil) }
}
