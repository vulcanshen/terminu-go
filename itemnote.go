package terminu

import (
	tea "github.com/charmbracelet/bubbletea"
)

const glyphNote = "\U000F02FD" // nf-md-information_outline: an item's note (tdp panel.18)

// NoteSpot is a body of the app's own that has items with notes (Panel.Note,
// tdp panel.18): it tells where the item under its cursor was at its last
// Draw, its first and last rows and the cell its text starts at, inside the
// body; ok false when it is not in sight. List is one.
type NoteSpot interface {
	NoteAt() (top, bottom, col int, ok bool)
}

// noteSpot is where an item with a note is on the screen, for its head and
// for the note it opens (tdp panel.18): the item's rows, the cell under its
// text, the inside of its panel, the panel's height.
type noteSpot struct {
	p                   *Panel
	top, bottom, x      int
	inTop, inBottom     int
	inLeft, inRight, ph int
	up                  bool // no room below: the head and the note above the item
	headY               int
}

// noteOf is the note of the item under p's cursor: its mode's, the
// innermost layer with a Note first, or the panel's; none in a list of the
// items that differ or a diff the lib shows in it.
func (p *Panel) noteOf() []string {
	if p.level() != nil || p.over != nil {
		return nil
	}
	if p.mode != nil {
		layers := append(append([]*Mode{}, p.under...), p.mode)
		for i := len(layers) - 1; i >= 0; i-- {
			if layers[i].Note != nil {
				return layers[i].Note()
			}
		}
		return nil
	}
	if p.Note != nil {
		return p.Note()
	}
	return nil
}

// noteAction is the i action the lib puts among a panel's with a Note (tdp
// panel.18, K13).
func (p *Panel) noteAction() Action {
	return Action{Key: "i", Label: "Note", Desc: "this item's note", Scope: ItemScope,
		Disabled: func() bool { return len(p.noteOf()) == 0 },
		Do: func() tea.Cmd {
			if p.app == nil {
				return nil
			}
			return p.app.openNote(p)
		}}
}

// noteKey is the i key the lib puts among a mode's keys when the mode has a
// Note (tdp panel.18, K13): an item key, in its Space menu and ?.
func (p *Panel) noteKey(m *Mode) {
	if m == nil || m.Note == nil {
		return
	}
	for _, k := range m.Keys {
		if k.takes("i") {
			return
		}
	}
	m.Keys = append(m.Keys, ModeKey{Key: "i", Desc: "note", Item: true, Label: "Note", MenuDesc: "this item's note", NoFooter: true,
		Disabled: func() bool { return len(p.noteOf()) == 0 },
		Do: func(string) tea.Cmd {
			if p.app == nil {
				return nil
			}
			return p.app.openNote(p)
		}})
}

// noteHead is the head of a note: a small box with the glyph in Text and i,
// the box and i in Blue as a key that can be pressed now (tdp panel.18);
// its top border under the item, its bottom border above it (up).
func noteHead(up bool) string {
	l, r := "┌", "┐"
	if up {
		l, r = "└", "┘"
	}
	return Fg(Blue).Render(l+" ") + Fg(Text).Render(glyphNote) + Fg(Blue).Render(" i "+r)
}

// drawNoteHead draws the head of the note of the item under the cursor of
// p, the focused panel at r in area, the area starting on screen row top
// (tdp panel.18): right under the item, over the row below; above it when
// the item is on the panel's last row. None while a popup is open, while
// typing in its filter, or in its PTY. It keeps where the item is for i.
func (a *App) drawNoteHead(area []string, p *Panel, r Rect, top, w int) {
	a.note = nil
	if p == nil || a.top() != nil || p.typing() || len(p.noteOf()) == 0 {
		return
	}
	if t, ok := p.body().(*Terminal); ok && t.inside {
		return
	}
	ns, ok := p.body().(NoteSpot)
	if !ok {
		return
	}
	it, ib, col, ok := ns.NoteAt()
	if !ok {
		return
	}
	s := &noteSpot{p: p, inTop: top + r.Y + 1, inBottom: top + r.Y + r.H - 2, inLeft: r.X + 1, inRight: r.X + r.W - 2, ph: r.H}
	s.top, s.bottom = top+r.Y+p.bodyY+it, top+r.Y+p.bodyY+ib
	hw := 6 + IconWidth()
	s.x = min(max(r.X+1+col, s.inLeft), max(s.inRight-hw+1, s.inLeft))
	switch {
	case s.bottom+1 <= s.inBottom:
		s.headY = s.bottom + 1
	case s.top-1 >= s.inTop:
		s.headY, s.up = s.top-1, true
	default:
		return
	}
	Composite(area, []string{noteHead(s.up)}, s.x, s.headY-top, w)
	a.note = s
}

// openNote opens the note of the item under p's cursor (tdp panel.18): a
// viewer titled NOTE anchored where its head is, under the item or above it,
// at most half the panel high, the box unfolding from the head; Esc closes
// it back to its head.
func (a *App) openNote(p *Panel) tea.Cmd {
	a.Frame() // where the item is now: its head drawn afresh
	s := a.note
	lines := p.noteOf()
	if s == nil || s.p != p || len(lines) == 0 {
		return nil
	}
	v := &Viewer{Title: "Note", Glyph: glyphNote, Lines: lines, NoNumbers: true}
	spot := *s
	v.note = &spot
	cmd := a.Viewer(v)
	if n := len(a.stack); n > 0 {
		a.stack[n-1].unfold = 1
		if spot.up {
			a.stack[n-1].unfold = -1
		}
	}
	return cmd
}

// noteSize is a note's width and height limits on a screen sw wide and its
// place once measured w × h: down under its item where it fits, else up
// above it (tdp panel.18).
func (s *noteSpot) most() (w, h int) {
	below, above := s.inBottom-s.bottom, s.top-s.inTop
	room := max(below, above)
	return max(s.inRight-s.x+1, 8), max(min(s.ph/2, room), 5)
}

// place is where a note w × h goes: from the head down, or up when it does
// not fit below and fits better above.
func (s *noteSpot) place(sh, h int) (int, bool) {
	below, above := s.inBottom-s.bottom, s.top-s.inTop
	if h <= below || below >= above && !s.up {
		return min(s.bottom+1, max(sh-h, 0)), false
	}
	return max(s.top-h, 0), true
}
