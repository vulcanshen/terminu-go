package terminu

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Pager is a panel body that shows lines with no cursor: a preview, a log.
// When Follow is set, new lines keep it at the bottom as long as it was
// there (streaming content).
type Pager struct {
	Follow bool
	// EmptyFact and EmptyHint are what it says with no lines.
	EmptyFact, EmptyHint string

	lines  []string
	top    int
	height int
	scroll string
	atEnd  bool
}

// SetLines replaces the lines.
func (t *Pager) SetLines(lines []string) {
	t.lines = lines
	t.clamp()
}

// Append adds lines at the end.
func (t *Pager) Append(lines ...string) {
	follow := t.Follow && (t.atEnd || t.top+t.height >= len(t.lines))
	t.lines = append(t.lines, lines...)
	if follow {
		t.top = max(len(t.lines)-t.height, 0)
	}
}

// Lines are the lines.
func (t *Pager) Lines() []string { return t.lines }

func (t *Pager) clamp() {
	t.top = min(max(t.top, 0), max(len(t.lines)-t.height, 0))
}

// Empty reports whether there is nothing to show.
func (t *Pager) Empty() bool { return len(t.lines) == 0 }

// HasItem is false: a text has no items (Rules K3: Enter works on the whole panel).
func (t *Pager) HasItem() bool { return false }

// Draw draws the lines seen from the top, one cell from the frame.
func (t *Pager) Draw(w, h int, focused bool) []string {
	t.height = h
	if len(t.lines) == 0 {
		t.scroll = ""
		return EmptyState(w, h, t.EmptyFact, t.EmptyHint)
	}
	t.clamp()
	t.atEnd = t.top+h >= len(t.lines)
	out := make([]string, h)
	for i := range h {
		if t.top+i < len(t.lines) {
			out[i] = " " + Fit(t.lines[t.top+i], w-2) + " "
		} else {
			out[i] = Repeat(" ", w)
		}
	}
	t.scroll = ""
	if len(t.lines) > h {
		t.scroll = scrollText(false, t.top+1, min(t.top+h, len(t.lines)), len(t.lines))
	}
	return out
}

// Scroll is "N-M of T" when the lines do not all fit.
func (t *Pager) Scroll() string { return t.scroll }

// MoveKeys are the scrolling keys.
func (t *Pager) MoveKeys() []KeyRow {
	return []KeyRow{
		{Key: "j|k", Desc: "scroll down, up"},
		{Key: "u|d", Desc: "half a page up, down"},
		{Key: "gg|G", Desc: "to the top, the bottom"},
	}
}

// Key scrolls.
func (t *Pager) Key(key string, _ tea.KeyMsg) (bool, tea.Cmd) {
	half := max(t.height/2, 1)
	switch key {
	case "j", "↓":
		t.top++
	case "k", "↑":
		t.top--
	case "d", "PgDn":
		t.top += half
	case "u", "PgUp":
		t.top -= half
	case "gg", "Home":
		t.top = 0
	case "G", "End":
		t.top = len(t.lines)
	default:
		return false, nil
	}
	t.clamp()
	return true, nil
}
