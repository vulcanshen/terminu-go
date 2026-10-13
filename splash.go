package terminu

import (
	"errors"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Splash draws frame n (from 0) of the family's easter egg in w × h cells
// (Rules S): the app's docs/icon.svg, the family mark, cell for cell, with a
// reveal of the app's own (S5). done reports that the reveal is over; the
// last frame then stays until a key.
type Splash func(frame, w, h int) (lines []string, done bool)

type splashMsg struct{ id int }

func splashTick(id int) tea.Cmd {
	return tea.Tick(frameTime, func(time.Time) tea.Msg { return splashMsg{id} })
}

// takesV reports that p gave V to an action of its own: the splash does not
// open there (Rules S1).
func (p *Panel) takesV() bool {
	for _, act := range p.Actions {
		if act.Key == "V" {
			return true
		}
	}
	if b, ok := p.Body.(interface{ Actions() []Action }); ok {
		for _, act := range b.Actions() {
			if act.Key == "V" {
				return true
			}
		}
	}
	if p.mode != nil {
		for _, k := range p.mode.Keys {
			if k.takes("V") {
				return true
			}
		}
	}
	return false
}

// openSplash opens the splash: only from a panel, only on V (Rules S4).
func (a *App) openSplash() tea.Cmd {
	a.splashOn, a.splashFrame = true, 0
	a.splashID++
	return splashTick(a.splashID)
}

// stepSplash moves the reveal on by one frame until it is over.
func (a *App) stepSplash(msg splashMsg) tea.Cmd {
	if !a.splashOn || msg.id != a.splashID {
		return nil
	}
	if _, done := a.Splash(a.splashFrame, a.w, a.h); done {
		return nil
	}
	a.splashFrame++
	return splashTick(a.splashID)
}

// checkSplash refuses an app the splash cannot be opened in (Rules S1): none
// given, a global hotkey on V, or every panel giving V away.
func (a *App) checkSplash() error {
	if a.Splash == nil {
		return errors.New("terminu: no Splash (Rules S1: every terminu app has one)")
	}
	for _, act := range a.Global {
		if act.Key == "V" {
			return errors.New("terminu: global action " + act.Label + " takes V, kept for the splash (Rules S1)")
		}
	}
	for _, s := range a.screens {
		for _, p := range s.Panels {
			if !p.takesV() {
				return nil
			}
		}
	}
	return errors.New("terminu: every panel takes V; at least one must keep it for the splash (Rules S1)")
}
