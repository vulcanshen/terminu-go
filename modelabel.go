package terminu

import (
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// A mode's name opens from its middle on the frame and closes back to the
// line (tdp panel.1): the junctions side by side
// first, then one more character each frame of 32ms, as a popup's (Rules
// F2), so every frame changes and the length goes with the name's; by the
// clock. Only the name moves: the mode and the frame's colour change at once
// and keys work all the while. A switch halfway turns it where it is.
//
// A mode of several layers joins them with ">" (Visual>select). A layer more
// grows from the name's tail a character a frame, the right end still, a
// layer less goes back the same way; leaving the whole mode from a layer
// closes the whole name from its middle. The title keeps the longest name's
// room all the while.

// labelTime is how long name takes to open or to close: a frame a character.
func labelTime(name string) time.Duration {
	return time.Duration(max(len(graphemes(name)), 1)) * frameTime
}

// labelAnim is a mode name opening (dir 1) or closing (dir -1), from where
// it was (from, 1 all open) since start: from its middle, or with tail from
// base, a layer growing or going at the tail of name.
type labelAnim struct {
	name  string
	base  string
	tail  bool
	from  float64
	dir   int
	start time.Time
}

// moves is how many characters the animation moves.
func (l *labelAnim) moves() int {
	if l.tail {
		return len(graphemes(l.name)) - len(graphemes(l.base))
	}
	return len(graphemes(l.name))
}

func (l *labelAnim) at(now time.Time) float64 {
	d := time.Duration(max(l.moves(), 1)) * frameTime // a frame a character
	p := l.from + float64(l.dir)*float64(now.Sub(l.start))/float64(d)
	return min(max(p, 0), 1)
}

// text is a tail animation's name as shown now: its base and as much of the
// layer as has grown.
func (l *labelAnim) text(now time.Time) string {
	if !l.tail {
		return l.name
	}
	g := graphemes(l.name)
	n := len(graphemes(l.base))
	k := n + int(math.Floor(l.at(now)*float64(len(g)-n)+1e-6))
	return strings.Join(g[:min(k, len(g))], "")
}

func (l *labelAnim) moving(now time.Time) bool {
	p := l.at(now)
	return l.dir > 0 && p < 1 || l.dir < 0 && p > 0
}

// label is a mode's name as drawn: how far open from its middle, 1 all of
// it, and the room the title leaves it, the longest name's while it moves.
type label struct {
	name  string
	shown float64
	room  int
}

// labelled is a frame that can carry a mode's name: what the App last
// worked out for it.
type labelled struct{ lab label }

func (x *labelled) setLabel(l label) { x.lab = l }

// labelFor is the name to draw for a frame in the mode current: as the App
// worked it out, or all open when it has not seen it yet.
func (x *labelled) labelFor(current string) label {
	if x.lab.name == "" && current != "" {
		return label{current, 1, Width(current)}
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
		case l == nil:
			if name != "" {
				a.labels[k] = &labelAnim{name: name, dir: 1, start: now}
			}
		case l.tail:
			switch name {
			case l.name, l.base:
				if want := map[bool]int{true: 1, false: -1}[name == l.name]; l.dir != want {
					*l = labelAnim{name: l.name, base: l.base, tail: true, from: l.at(now), dir: want, start: now} // turned
				}
			case "": // the whole mode left from a layer: the whole name closes from its middle
				*l = labelAnim{name: l.text(now), from: 1, dir: -1, start: now}
			default:
				*l = labelAnim{name: name, dir: 1, start: now}
			}
		case name == l.name && l.dir < 0, name == "" && l.dir > 0:
			*l = labelAnim{name: l.name, from: l.at(now), dir: -l.dir, start: now} // turned where it is
		case name == "" || name == l.name:
		case l.dir > 0 && strings.HasPrefix(name, l.name+">"): // a layer more
			*l = labelAnim{name: name, base: l.name, tail: true, dir: 1, start: now}
		case l.dir > 0 && strings.HasPrefix(l.name, name+">"): // a layer less
			*l = labelAnim{name: l.name, base: name, tail: true, from: 1, dir: -1, start: now}
		default:
			*l = labelAnim{name: name, dir: 1, start: now}
		}
	}
	moving := false
	for k, l := range a.labels {
		if _, ok := keys[k]; !ok || !l.tail && l.dir < 0 && !l.moving(now) {
			delete(a.labels, k)
			continue
		}
		if l.tail && !l.moving(now) { // grown or gone back: the name it ended as, all open
			*l = labelAnim{name: l.text(now), from: 1, dir: 1, start: now}
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
	now := a.now()
	if l.tail {
		return label{l.text(now), 1, Width(l.name)}
	}
	p := l.at(now)
	if l.dir < 0 && p <= 0 {
		return label{}
	}
	return label{l.name, p, Width(l.name)}
}

// endLabels plays every name's animation to its end: a test's settle.
func (a *App) endLabels() {
	now := a.now()
	for k, l := range a.labels {
		switch {
		case l.tail:
			name := l.name
			if l.dir < 0 {
				name = l.base
			}
			*l = labelAnim{name: name, from: 1, dir: 1, start: now}
		case l.dir < 0:
			delete(a.labels, k)
		default:
			l.from, l.start = 1, now
		}
	}
}

// drawLabel is a mode's name on a frame, closed by closed (0 all open): the
// junctions where the shown part ends, line where the rest is, so it keeps
// its width all the while; the name in Yellow, the junctions the frame's.
func drawLabel(name string, closed float64, ls lineSet, frame Style) string {
	g := graphemes(name)
	r := min(int(math.Floor((1-closed)*float64(len(g))+1e-6)), len(g)) // a character a frame
	start := (len(g) - r) / 2                                          // an odd name a half cell to the left
	left, mid, right := strings.Join(g[:start], ""), strings.Join(g[start:start+r], ""), strings.Join(g[start+r:], "")
	out := frame.Render(Repeat(ls.h, Width(left)) + ls.jl)
	if mid != "" {
		out += Fg(Yellow).Bolded().Render(mid)
	}
	return out + frame.Render(ls.jr+Repeat(ls.h, Width(right))+ls.h)
}
