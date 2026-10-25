package tdptest

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vulcanshen/terminu-go"
	"github.com/vulcanshen/terminu-go/internal/testhook"
)

// dropMsg runs do in the app's OnMsg and returns what do returns: an app
// that keeps only some of the commands the lib gave it.
type dropMsg struct{ do func() tea.Cmd }

// TestToastCommandDropped: the lib sends its own timers, the frame tick and
// a toast's time, whatever the app returns: two toasts with only one's
// command returned, a popup and a toast with only the toast's, all dropped.
// The animations still play, the toast still goes in its time, and the
// popups after it still open (filu, 2026-10-21).
func TestToastCommandDropped(t *testing.T) {
	for _, c := range []struct {
		name string
		do   func(a *terminu.App) tea.Cmd
		pop  bool // a popup is left open
	}{
		{"two toasts, the second's returned", func(a *terminu.App) tea.Cmd {
			a.Toast(terminu.Info, "First")
			return a.Toast(terminu.Info, "Second")
		}, false},
		{"two toasts, the first's returned", func(a *terminu.App) tea.Cmd {
			cmd := a.Toast(terminu.Info, "First")
			a.Toast(terminu.Info, "Second")
			return cmd
		}, false},
		{"a popup and a toast, the toast's returned", func(a *terminu.App) tea.Cmd {
			a.Error("Open", "Not found")
			return a.Toast(terminu.Info, "Second")
		}, true},
		{"all dropped", func(a *terminu.App) tea.Cmd {
			a.Error("Open", "Not found")
			a.Toast(terminu.Info, "Second")
			return nil
		}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			ty := newTiny()
			a := ty.App
			a.OnMsg = func(msg tea.Msg) tea.Cmd {
				if m, ok := msg.(dropMsg); ok {
					return m.do()
				}
				return nil
			}
			d := Start(t, a, 80, 24)
			d.Send(dropMsg{func() tea.Cmd { return c.do(a) }})
			if testhook.Animating(a) || !d.Has("Second") {
				t.Fatalf("the toast did not finish opening:\n%s", d.where())
			}
			if c.pop && (testhook.Layers(a) != 1 || !d.Has("OPEN")) {
				t.Errorf("the popup did not open: %d layers", testhook.Layers(a))
			}
			d.Advance(2200 * time.Millisecond)
			if testhook.Toast(a) || d.Has("Second") {
				t.Errorf("the toast did not go in its time")
			}
			if c.pop {
				d.Press("Esc")
			}
			d.Press("Space")
			if testhook.Animating(a) || testhook.Layers(a) != 1 {
				t.Errorf("the Space menu did not open after it: %d layers", testhook.Layers(a))
			}
		})
	}
}

// TestStartSendsLibTimers: a toast the app's Start opens and drops still
// plays and goes: Init sends the lib's own timers too.
func TestStartSendsLibTimers(t *testing.T) {
	ty := newTiny()
	ty.App.Start = func() tea.Cmd { ty.App.Toast(terminu.Info, "Started"); return nil }
	d := Start(t, ty.App, 80, 24)
	if testhook.Animating(ty.App) || !d.Has("Started") {
		t.Fatal("the toast did not open")
	}
	d.Advance(2200 * time.Millisecond)
	if d.Has("Started") {
		t.Error("the toast did not go")
	}
}

// TestOpeningKeepsItsTime: a frame tick the app returns and the lib sends
// too runs once: the opening takes its 8 frames of 32ms, not half the time
// (tdp F2).
func TestOpeningKeepsItsTime(t *testing.T) {
	ty := newTiny()
	d := Start(t, ty.App, 80, 24)
	d.Auto = false
	d.Do(func() tea.Cmd { return ty.App.Toast(terminu.Info, "Hello") })
	d.Advance(7 * 32 * time.Millisecond)
	if !testhook.Animating(ty.App) {
		t.Fatal("the opening ended early: a tick ran twice")
	}
	d.Advance(32 * time.Millisecond)
	if testhook.Animating(ty.App) {
		t.Error("the opening did not end in its time")
	}
}

// TestDoSendsLibTimers: a toast an app's Do opens and drops still plays
// and goes: Do sends the lib's own timers too.
func TestDoSendsLibTimers(t *testing.T) {
	ty := newTiny()
	d := Start(t, ty.App, 80, 24)
	d.Do(func() tea.Cmd { ty.App.Toast(terminu.Info, "Dropped"); return nil })
	if testhook.Animating(ty.App) || !d.Has("Dropped") {
		t.Fatal("the toast did not open")
	}
	d.Advance(2200 * time.Millisecond)
	if d.Has("Dropped") {
		t.Error("the toast did not go")
	}
}

// TestModeLabelPlays: a mode name opening is an animation to the driver,
// played by its ticks on the clock: with Auto off it steps, and it ends open.
func TestModeLabelPlays(t *testing.T) {
	ty := newTiny()
	p := ty.App.Screen().Panels[0]
	p.Actions = append(p.Actions, terminu.Action{Key: "M", Label: "Mark mode", Desc: "a mode", Scope: terminu.PanelScope,
		Do: func() tea.Cmd { p.EnterMode(&terminu.Mode{Name: "Mark"}); return nil }})
	d := Start(t, ty.App, 80, 24)
	d.Auto = false
	d.Press("M")
	if !testhook.Animating(ty.App) || d.Has("Mark") {
		t.Fatal("the mode name is not opening")
	}
	steps := 0
	for testhook.Animating(ty.App) && d.Step() {
		steps++
	}
	if steps < 2 || !d.Has("┫Mark┣") {
		t.Errorf("%d steps, then:\n%s", steps, strings.Join(d.Screen(), "\n"))
	}
}
