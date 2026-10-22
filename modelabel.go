package terminu

import (
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// A mode's name opens from its middle on the frame and closes back to the
// line (components/layout/panel, the frame): the junctions side by side
// first, then apart, the name showing from its middle; about 288ms whatever
// its length, by the clock, as a popup opens (Rules F2). Only the name
// moves: the mode and the frame's colour change at once and keys work all
// the while. A switch halfway turns it where it is.

// modeLabelTime is how long a mode's name takes to open or to close.
const modeLabelTime = openFrames * frameTime

// labelAnim is a mode name opening (dir 1) or closing (dir -1), from where
// it was (from, 1 all open) since start.
type labelAnim struct {
	name  string
	from  float64
	dir   int
	start time.Time
}

func (l *labelAnim) at(now time.Time) float64 {
	p := l.from + float64(l.dir)*float64(now.Sub(l.start))/float64(modeLabelTime)
	return min(max(p, 0), 1)
}

func (l *labelAnim) moving(now time.Time) bool {
	p := l.at(now)
	return l.dir > 0 && p < 1 || l.dir < 0 && p > 0
}

// label is a mode's name as drawn: how far open, 1 all of it.
type label struct {
	name  string
	shown float64
}

// labelled is a frame that can carry a mode's name: what the App last
// worked out for it.
type labelled struct{ lab label }

func (x *labelled) setLabel(l label) { x.lab = l }

// labelFor is the name to draw for a frame in the mode current: as the App
// worked it out, or all open when it has not seen it yet.
func (x *labelled) labelFor(current string) label {
	if x.lab.name == "" && current != "" {
		return label{current, 1}
	}
	return x.lab
}

type labelMsg struct{}

// labelKeys are the frames on the screen that can carry a mode's name, with
// the mode each is in now.
func (a *App) labelKeys() map[any]string {
	out := map[any]string{}
	if s := a.Screen(); s != nil {
		for _, p := range s.Panels {
			out[p] = p.modeName()
		}
	}
	for _, l := range a.stack {
		if m, ok := l.view.(interface{ modeName() string }); ok && !l.closing {
			out[l.view] = m.modeName()
		}
	}
	return out
}

// syncLabels starts, turns and forgets the names' animations as the modes
// are now, with a tick while one moves.
func (a *App) syncLabels() tea.Cmd {
	now := a.now()
	keys := a.labelKeys()
	if a.labels == nil {
		a.labels = map[any]*labelAnim{}
	}
	for k, name := range keys {
		l := a.labels[k]
		switch {
		case name != "" && l == nil:
			a.labels[k] = &labelAnim{name: name, dir: 1, start: now}
		case name != "" && l.name != name:
			*l = labelAnim{name: name, dir: 1, start: now}
		case name != "" && l.dir < 0, name == "" && l != nil && l.dir > 0:
			*l = labelAnim{name: l.name, from: l.at(now), dir: -l.dir, start: now} // turned where it is
		}
	}
	moving := false
	for k, l := range a.labels {
		if _, ok := keys[k]; !ok || l.dir < 0 && !l.moving(now) {
			delete(a.labels, k)
			continue
		}
		moving = moving || l.moving(now)
	}
	if moving && !a.labelTick {
		a.labelTick = true
		return a.after(frameTime, labelMsg{})
	}
	return nil
}

// labelsMoving reports a mode's name still opening or closing.
func (a *App) labelsMoving() bool {
	now := a.now()
	for _, l := range a.labels {
		if l.moving(now) {
			return true
		}
	}
	return false
}

// labelOf is key's mode name as drawn now.
func (a *App) labelOf(key any) label {
	l := a.labels[key]
	if l == nil {
		return label{}
	}
	p := l.at(a.now())
	if l.dir < 0 && p <= 0 {
		return label{}
	}
	return label{l.name, p}
}

// endLabels plays every name's animation to its end: a test's settle.
func (a *App) endLabels() {
	now := a.now()
	for k, l := range a.labels {
		if l.dir < 0 {
			delete(a.labels, k)
			continue
		}
		l.from, l.start = 1, now
	}
}

// drawLabel is a mode's name on a frame, closed by closed (0 all open): the
// junctions where the shown part ends, line where the rest is, so it keeps
// its width all the while; the name in Yellow, the junctions the frame's.
func drawLabel(name string, closed float64, ls lineSet, frame Style) string {
	g := graphemes(name)
	r := len(g) - int(math.Round(closed*float64(len(g))))
	start := (len(g) - r) / 2 // an odd name a half cell to the left
	left, mid, right := strings.Join(g[:start], ""), strings.Join(g[start:start+r], ""), strings.Join(g[start+r:], "")
	out := frame.Render(Repeat(ls.h, Width(left)) + ls.jl)
	if mid != "" {
		out += Fg(Yellow).Bolded().Render(mid)
	}
	return out + frame.Render(ls.jr+Repeat(ls.h, Width(right))+ls.h)
}
