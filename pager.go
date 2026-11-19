package terminu

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Pager is a panel body that shows lines with no cursor: a preview, a log.
// When Follow is set, new lines keep it at the bottom as long as it was
// there; scrolled up, it stays where it is. A Pager that takes lines while
// it is open (Append, ReadAgain) sets Follow (tdp note.9). Its lines are
// reading content, drawn
// as a terminal draws them (components, control characters): \n starts
// another line, a \r in one leaves its last state, SGR colours it, a tab is
// spaces, any other control character is its name in Red.
type Pager struct {
	Follow bool
	// EmptyFact and EmptyHint are what it says with no lines.
	EmptyFact, EmptyHint string

	lines  []string
	rows   []string // the lines as read (readingRows)
	top    int
	height int
	width  int // the text's, at the last Draw
	scroll string
	atEnd  bool
	diff   *pagerDiff // what SetDiff or SetDiffItems showed, laid out again as the width changes
}

// pagerDiff is a difference a Pager shows (SetDiff, SetDiffItems): lay
// lays it out for a text w cells wide; w and icon are what it was last laid
// out for.
type pagerDiff struct {
	lay     func(w int) []string
	w, icon int
}

// SetLines replaces the lines.
func (t *Pager) SetLines(lines []string) {
	t.diff = nil
	t.lines, t.rows = lines, readingRows(lines)
	t.clamp()
}

// SetDiff shows a difference as UNIFIED does (tdp note.8 "In a panel"):
// anchor's text against other's, only the parts that differ with 3 lines
// around, an Overlay0 line between them as wide as the panel, both line
// numbers in Overlay0, - in Red and + in Green; never a tool's own output.
// The same text on both sides shows the empty state (EmptyFact). Lines are
// cut at the panel's edge, as every Pager line is; the comparison folds
// them. The view stays where it was, as with SetLines; SetLines, Append and
// ReadAgain go back to plain lines. Enter is the panel's own action,
// App.OpenDiff opening the comparison of the same two (tdp note.8).
func (t *Pager) SetDiff(anchor, other string) {
	x, y := textLines(anchor), textLines(other)
	ops := diffLines(x, y)
	t.setDiff(func(w int) []string {
		lines, _ := diffUnified(ops, x, y, w)
		return lines
	})
}

// SetDiffItems shows the items that differ in what holds many (tdp note.8
// "In a panel": a folder that is new, a commit), never their bare names:
// each item's mark in its colour in the mark column, then its name, a long
// one cut from the front, as the list App.OpenDiff goes into draws them
// (tdp panel.9; a new folder's items all DiffAdded, + in Green). None is
// the empty state (EmptyFact). The view stays where it was, as with
// SetLines; SetLines, Append and ReadAgain go back to plain lines. Enter is
// the panel's own action, App.OpenDiff with Items giving the same items
// going into that list in the panel.
func (t *Pager) SetDiffItems(items []DiffItem) {
	t.setDiff(func(w int) []string {
		markW := IconWidth() + 1
		lines := make([]string, len(items))
		for i, it := range items {
			m := it.Mark.span()
			lines[i] = Pad(m.Style.Render(m.Text), markW) + FitLeft(Escape(it.Name), w-markW)
		}
		return lines
	})
}

func (t *Pager) setDiff(lay func(w int) []string) {
	t.diff = &pagerDiff{lay: lay, w: -1}
	t.layDiff(t.width)
	t.clamp()
}

// layDiff lays the difference out for a text w cells wide.
func (t *Pager) layDiff(w int) {
	d := t.diff
	if d == nil || d.w == w && d.icon == IconWidth() {
		return
	}
	lines := d.lay(max(w, 0))
	t.lines, t.rows, d.w, d.icon = lines, readingRows(lines), w, IconWidth()
}

// Append adds lines at the end.
func (t *Pager) Append(lines ...string) {
	t.diff = nil
	follow := t.Follow && (t.atEnd || t.top+t.height >= len(t.rows))
	t.lines = append(t.lines, lines...)
	t.rows = append(t.rows, readingRows(lines)...)
	if follow {
		t.top = max(len(t.rows)-t.height, 0)
	}
}

// ReadAgain puts lines in place of the Pager's, its source read again (a
// file it watches; tdp note.9): with Follow, at the new end when the end was
// in sight; otherwise where it was, at the new end when the lines are fewer
// than fill it. SetLines is new content, the view going nowhere.
func (t *Pager) ReadAgain(lines []string) {
	follow := t.Follow && (t.atEnd || t.top+t.height >= len(t.rows))
	t.SetLines(lines)
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
	t.height, t.width = h, w-2
	t.layDiff(w - 2)
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
