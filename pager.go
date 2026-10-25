package terminu

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Pager is a panel body that shows lines with no cursor: a preview, a log.
// When Follow is set, new lines keep it at the bottom as long as it was
// there (streaming content). Its lines are reading content, drawn as a
// terminal draws them (components, control characters): \n starts another
// line, a \r in one leaves its last state, SGR colours it, a tab is spaces,
// any other control character is its name in Red.
type Pager struct {
	Follow bool
	// EmptyFact and EmptyHint are what it says with no lines.
	EmptyFact, EmptyHint string

	lines  []string
	rows   []string // the lines as read (readingRows)
	top    int
	height int
	scroll string
	atEnd  bool
}

// SetLines replaces the lines.
func (t *Pager) SetLines(lines []string) {
	t.lines, t.rows = lines, readingRows(lines)
	t.clamp()
}

// Append adds lines at the end.
func (t *Pager) Append(lines ...string) {
	follow := t.Follow && (t.atEnd || t.top+t.height >= len(t.rows))
	t.lines = append(t.lines, lines...)
	t.rows = append(t.rows, readingRows(lines)...)
	if follow {
		t.top = max(len(t.rows)-t.height, 0)
	}
}

// Lines are the lines.
func (t *Pager) Lines() []string { return t.lines }

func (t *Pager) clamp() {
	t.top = min(max(t.top, 0), max(len(t.rows)-t.height, 0))
}

// Empty reports whether there is nothing to show.
func (t *Pager) Empty() bool { return len(t.rows) == 0 }

// HasItem is false: a text has no items (tdp K3: Enter works on the whole panel).
func (t *Pager) HasItem() bool { return false }

// Draw draws the lines seen from the top, one cell from the frame.
func (t *Pager) Draw(w, h int, focused bool) []string {
	t.height = h
	if len(t.rows) == 0 {
		t.scroll = ""
		return EmptyState(w, h, t.EmptyFact, t.EmptyHint)
	}
	t.clamp()
	t.atEnd = t.top+h >= len(t.rows)
	out := make([]string, h)
	for i := range h {
		if t.top+i < len(t.rows) {
			out[i] = " " + Fit(readingLine(cutCR(t.rows[t.top+i]), tabCells), w-2) + " "
		} else {
			out[i] = Repeat(" ", w)
		}
	}
	t.scroll = ""
	if len(t.rows) > h {
		t.scroll = scrollText(false, t.top+1, min(t.top+h, len(t.rows)), len(t.rows))
	}
	return out
}

// Scroll is "N-M of T" when the lines do not all fit.
func (t *Pager) Scroll() string { return t.scroll }

// MoveKeys are the scrolling keys; none when every line fits, nothing to
// scroll (tdp M6).
func (t *Pager) MoveKeys() []KeyRow {
	if len(t.rows) <= t.height {
		return nil
	}
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
	case "gg":
		t.top = 0
	case "G":
		t.top = len(t.rows)
	default:
		return false, nil
	}
	t.clamp()
	return true, nil
}
