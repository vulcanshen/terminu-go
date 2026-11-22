package terminu

import (
	"cmp"
	"slices"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Column is one column of a List (tdp panel.5).
type Column struct {
	// Title is the column's header. A list of one column whose Title is ""
	// has no header.
	Title string
	// Number marks a number column: right-aligned and sorted by value (9
	// before 10). Text columns are left-aligned and sorted as text.
	Number bool
	// Flex did let the column give up room first when the panel is narrow. It
	// does nothing since tdp v0.8.0: mark a long column Shrink.
	Flex bool
	// Shrink marks a column of long values (a name, a URL, a line of text):
	// the shrinkable columns share the width the others leave, cut each its
	// own way when too long, down to MinWidth (12 cells when unset). The
	// other columns, short values (a state, a number, a date), keep the width
	// of their widest value (tdp panel).
	Shrink   bool
	MinWidth int
	// Drop is when the column is taken away, whole and its header with it,
	// when the columns do not fit even with the shrinkable ones at their
	// MinWidth: 1 first, then 2 and on; 0 never (the name). As the panel
	// grows they come back in order, and the shrinkable ones grow back
	// (tdp panel).
	Drop int
	// Path cuts the column's values from the front ("…/.ssh/id_ed25519").
	Path bool
	// Middle cuts the column's values in the middle, for a value whose two
	// ends both matter ("demo-app…66-gm72n").
	Middle bool
	// Line marks a column of lines of text, for reading (a line of source
	// code, of a log): a Tab in it is drawn as input_tab_width spaces, its
	// layout; any other control character is still its name in Red
	// (components, control characters).
	Line bool
	// Fill marks the column that takes a row's room left over, with
	// SpreadFill and SpreadGaps (List.Spread): a long one, a commit's
	// description (tdp panel.5).
	Fill bool
}

// Row is one item of a List.
type Row struct {
	ID string // what marks are kept by; the row's place when empty
	// Cells are the row's text, one per column: a control character in one,
	// ESC too, is drawn by its name (Escape; tdp panel). The filter and
	// sorting go by them.
	Cells []string
	// Styled, when set for a cell, is how it is drawn in colours of the app's
	// own: each piece's text drawn as a cell's is. The cell's text stays in
	// Cells.
	Styled [][]Span
	// Sort, when set, are the values sorted by instead of the cells (an age
	// shown as "3d" sorted by seconds).
	Sort []string
	// SortGroup keeps rows apart when the list is sorted: the lower group
	// first, sorting only within each (a file manager's directories before its
	// files); no title is drawn between them (tdp panel). The rows of one
	// group keep their order when the list is not sorted.
	SortGroup int
	// Group is the title of the group the row is in (tdp panel.11).
	// A group's rows stay together, the groups in the order they first
	// appear, sorted within; the rows are drawn two cells further in, under
	// a title the cursor skips. A group with no row shown has no title.
	// With groups, every row is in one: a row that fits no group goes in a
	// group of its own ("Other").
	Group string
	// Below are the rows the item takes under its first, as many as it needs,
	// on the items that need them (tdp panel.5, an item of several rows: a
	// commit's branches and tags under it). Each is a row of cells under the
	// list's columns: a cell left "" leaves its column blank, so what the
	// row holds starts where the app puts it, and its last cell with text
	// runs on to the row's end. The first row keeps the columns, the header
	// and the tick; the cursor is drawn on it, the rows below lit with it,
	// Surface1 across, their text in its own colours. Moving, N of M and
	// scrolling count items, the item under the cursor always in sight whole;
	// the filter matches an item by the text of any of its rows.
	Below []BelowRow
	// Flag is drawn in the mark column of a list with Flags: one sign, its
	// glyph and colour the app's (locku's Green ● before the active
	// profile); none leaves the column blank (tdp panel.7).
	Flag Span
	// CursorColor, when set, is the cursor block's colour while the cursor
	// is on the row, the arrow on its right too, its text Base bold as ever:
	// the colour the row's content has of its own (a ranking's places), and
	// never another (tdp color.2). Unset it is Subtext1, as it is again while
	// the panel is in a mode, the mode's Yellow marking what it marks.
	CursorColor Color
}

// SortKey is one column the list is sorted by.
type SortKey struct {
	Col  int
	Desc bool
}

// List is a panel body with one item per row: a header, a cursor, marks, a
// panel filter (tdp panel).
type List struct {
	Columns []Column
	// Spread is where a row's room goes when every column already shows its
	// widest value and none is taken away, the app's pick for each list (tdp
	// panel.5, when there is room left over): it shows no more text, so it
	// only says where the blank is. SpreadEnd, the default, leaves it at the
	// row's end. When the room is short, every Spread is the same: the
	// shrinkable columns narrow, then columns go. The header, the rows and an
	// item's rows below follow the same widths.
	Spread Spread
	// Marks keeps a column for marks in front of every row (Principle P3,
	// batches of marked items).
	Marks bool
	// Draggable keeps that column too, for the swap glyph of the row held in
	// a Drag mode (tdp panel): Hold, MoveHeld, Drop.
	Draggable bool
	// Flags keeps that column in a list that cannot be marked for the rows'
	// flags (Row.Flag), one a row (tdp panel.7). A list that can be marked
	// keeps it for the tick: Marks and Flags together is a mistake, and
	// AddScreen panics.
	Flags bool
	// Anchor is the ID of a row a mode keeps picked while the cursor moves
	// on (a Diff mode's first pick, tdp K11): drawn whole in Yellow,
	// the mode's, with Base text, not bold and no arrow, as a selected range
	// (tdp color, note). The cursor on it wins. "" is none.
	Anchor string
	// Filterable lets / open a panel filter on the list.
	Filterable bool
	// EmptyFact and EmptyHint are what an empty list says (tdp screen.8);
	// keys in the hint written in brackets.
	EmptyFact, EmptyHint string
	// Location, when set, is the list's location row: where the user is (a
	// directory, a remote path, the level drilled into), in Lavender on the
	// panel's first row, cut from the front. It is only shown: the cursor
	// does not stop on it and the scroll position does not count it
	// (tdp panel.10). Whether a list has one is the app's. In a panel the
	// panel draws it, as Panel.Location: it stays over Loading and Fail and
	// dims with the focus, a streaming panel's too.
	Location string
	// LocationGlyph, when set, starts the location row: one glyph for the
	// kind of place, one space after it, in the row's colour, dimming with
	// it. A row too long is cut in Location only: the glyph stays and the
	// … comes right after it (tdp panel.10). Whether there is one, and
	// which, is the app's; it goes with this Location.
	LocationGlyph string
	// OnMove runs when the cursor comes to another row, by a key or not, the
	// first time included: before the frame is drawn, so a details panel
	// shows that row in the same frame (tdp panel.13). It does not
	// run while no row is under the cursor.
	OnMove func(Row) tea.Cmd
	// Points, when set, is the panel on the screen a row stands for, nil for
	// none (tdp panel.1; whether a list has one
	// is the app's). While the list has the focus, the panel its cursor's row
	// stands for is pointed at: its frame Lavender, still thin and square, not
	// focused, its content dimmed as any panel without the focus. A panel not
	// on the screen is not pointed at.
	Points func(Row) *Panel

	rows   []Row
	groups bool   // some row has a Group
	at     string // the row OnMove last ran for, "\x00" + its ID
	sort   []SortKey
	view   []int // rows shown, in order: sorted, then filtered
	cursor int   // index into view
	top    int   // first row shown
	cover  int   // rows at the bottom under the inspect toast (Uncover)
	// noteTop and noteBottom are the cursor's item's first and last rows at
	// the last draw, -1 when not drawn; noteCol is where its text starts
	// (NoteAt).
	noteTop, noteBottom, noteCol int
	marked                       map[string]bool
	held                         bool // the row under the cursor is held in a Drag mode
	inMode                       bool // the panel it is on is in a mode: the cursor is Subtext1

	filter *lineEditor
	typing bool
	scroll string
	height int // rows of items the last Draw had room for
	// gapMore is how much wider than two cells each gap between columns is,
	// as the last widths worked it out (SpreadGaps).
	gapMore int
}

// SetRows replaces the list's rows, keeping the cursor on the same row when
// it is still there.
func (l *List) SetRows(rows []Row) {
	cur, ok := l.Current()
	l.rows = rows
	l.rebuild()
	if ok {
		l.Select(cur.ID)
	}
}

// Rows are all the rows, as set.
func (l *List) Rows() []Row { return l.rows }

// SortBy sorts the list by keys, first key first: numbers by value, text as
// text (tdp panel). nil keeps the rows' order. Rows the keys cannot tell
// apart go by their IDs, so the same rows come out in the same order however
// they were set: a list must keep one order every time it is opened or
// fetched again, and data with no order of its own (a kubeconfig's contexts)
// is sorted by name.
func (l *List) SortBy(keys ...SortKey) {
	cur, ok := l.Current()
	l.sort = keys
	l.rebuild()
	if ok {
		l.Select(cur.ID)
	}
}

// Sorting is what the list is sorted by.
func (l *List) Sorting() []SortKey { return l.sort }

// Current is the row under the cursor.
func (l *List) Current() (Row, bool) {
	if l.cursor < 0 || l.cursor >= len(l.view) {
		return Row{}, false
	}
	return l.rows[l.view[l.cursor]], true
}

// Select puts the cursor on the row with this ID, the next frame scrolling
// it into sight, and reports whether it is shown. After the app adds an item
// (New, Copy, Import), it selects it (tdp K3); a row the panel filter hides
// is not shown, so the app clears the filter first if it wants it seen.
func (l *List) Select(id string) bool {
	for i, r := range l.view {
		if l.rowID(r) == id {
			l.cursor = i
			return true
		}
	}
	return false
}

func (l *List) rowID(i int) string {
	if l.rows[i].ID != "" {
		return l.rows[i].ID
	}
	return strconv.Itoa(i)
}

// ToggleMark marks or unmarks the row under the cursor.
func (l *List) ToggleMark() {
	r, ok := l.Current()
	if !ok {
		return
	}
	if l.marked == nil {
		l.marked = map[string]bool{}
	}
	id := l.rowID(l.view[l.cursor])
	_ = r
	if l.marked[id] {
		delete(l.marked, id)
	} else {
		l.marked[id] = true
	}
}

// Marked are the IDs of the marked rows.
func (l *List) Marked() []string {
	var out []string
	for i := range l.rows {
		if l.marked[l.rowID(i)] {
			out = append(out, l.rowID(i))
		}
	}
	return out
}

// ClearMarks unmarks every row.
func (l *List) ClearMarks() { l.marked = nil }

// SetMarked marks the rows with these IDs and unmarks the others, the rows
// the panel filter hides too.
func (l *List) SetMarked(ids []string) {
	l.marked = map[string]bool{}
	for _, id := range ids {
		l.marked[id] = true
	}
}

// Hold picks up the row under the cursor in the panel's Drag mode
// (tdp panel): it is drawn as the cursor in the mode's colour, Yellow,
// with the swap glyph in its mark column, until Drop. It reports whether
// there was a row. The Drag mode and its keys are the app's (Panel.EnterMode).
func (l *List) Hold() bool {
	_, ok := l.Current()
	l.held = ok
	return ok
}

// Holding reports a row held.
func (l *List) Holding() bool { return l.held }

// Drop puts the held row down where it is; its own mark shows again.
func (l *List) Drop() { l.held = false }

// MoveHeld swaps the held row with the next or the previous row shown, the
// cursor going with it. Past the ends it wraps: the last moved down goes to
// the top and the others move down one (tdp K12). It moves the rows
// themselves: a sorted list shows no move, so the app leaves the sort first.
func (l *List) MoveHeld(down bool) {
	if !l.held {
		return
	}
	shown := make([]Row, len(l.view))
	for i, r := range l.view {
		shown[i] = l.rows[r]
	}
	l.cursor = moveWrap(shown, l.cursor, down)
	for i, r := range l.view {
		l.rows[r] = shown[i]
	}
}

// Empty reports whether the list has no rows at all.
func (l *List) Empty() bool { return len(l.rows) == 0 }

// HasItem reports whether the cursor is on a row.
func (l *List) HasItem() bool { _, ok := l.Current(); return ok }

// Typing reports whether the focus is on the filter's typing row.
func (l *List) Typing() bool { return l.filter != nil && l.typing }

// Filtering reports whether a filter is set.
func (l *List) Filtering() bool { return l.filter != nil }

// OpenFilter puts a typing row at the top of the list, or goes back to it
// to type on (tdp panel.14).
func (l *List) OpenFilter() {
	if l.filter == nil {
		l.filter = newEditor("")
	}
	l.typing = true
}

// ClearFilter takes the filter away.
func (l *List) ClearFilter() {
	l.filter, l.typing = nil, false
	cur, ok := l.Current()
	l.rebuild()
	if ok {
		l.Select(cur.ID)
	}
}

// Focus is called when the panel gets the focus: a list with a filter set
// goes straight back to its typing row.
func (l *List) Focus() {
	if l.filter != nil {
		l.typing = true
	}
}

func (l *List) rebuild() {
	idx := make([]int, len(l.rows))
	for i := range idx {
		idx[i] = i
	}
	sortGroups := slices.ContainsFunc(l.rows, func(r Row) bool { return r.SortGroup != 0 })
	if len(l.sort) > 0 || sortGroups {
		slices.SortStableFunc(idx, func(a, b int) int {
			if c := cmp.Compare(l.rows[a].SortGroup, l.rows[b].SortGroup); c != 0 {
				return c
			}
			for _, k := range l.sort {
				c := l.compare(k.Col, a, b)
				if k.Desc {
					c = -c
				}
				if c != 0 {
					return c
				}
			}
			if len(l.sort) > 0 {
				return cmp.Compare(l.rows[a].ID, l.rows[b].ID)
			}
			return 0
		})
	}
	l.groups = slices.ContainsFunc(l.rows, func(r Row) bool { return r.Group != "" })
	if l.groups {
		rank := map[string]int{}
		for _, r := range l.rows {
			if _, ok := rank[r.Group]; !ok {
				rank[r.Group] = len(rank)
			}
		}
		slices.SortStableFunc(idx, func(a, b int) int { return cmp.Compare(rank[l.rows[a].Group], rank[l.rows[b].Group]) })
	}
	if l.filter != nil && !l.filter.empty() {
		q := l.filter.value()
		kept := idx[:0]
		for _, i := range idx {
			if l.matches(i, q) {
				kept = append(kept, i)
			}
		}
		idx = kept
	}
	l.view = idx
	l.cursor = min(max(l.cursor, 0), max(len(idx)-1, 0))
}

func (l *List) sortValue(row, col int) string {
	if col < len(l.rows[row].Sort) {
		return l.rows[row].Sort[col]
	}
	return l.cell(row, col)
}

// rowByID is the row whose ID is id.
func (l *List) rowByID(id string) (Row, bool) {
	for _, r := range l.rows {
		if r.ID == id {
			return r, true
		}
	}
	return Row{}, false
}

// BelowRow is a row an item takes under its first (Row.Below): its cells
// under the list's columns, the text the filter matches; Styled, when set for
// a cell, draws it in colours of the app's own, as a Row's does.
type BelowRow struct {
	Cells  []string
	Styled [][]Span
}

func (l *List) cell(row, col int) string {
	if col < len(l.rows[row].Cells) {
		return l.rows[row].Cells[col]
	}
	return ""
}

// shown is a cell as it is drawn: its spans, or its text escaped; in a Line
// column its tabs are spaces first.
func (l *List) shown(row, col int) string {
	r := &l.rows[row]
	return l.shownOf(r.Cells, r.Styled, col)
}

// shownOf is the cell col of a row's cells and spans as it is drawn.
func (l *List) shownOf(cells []string, styled [][]Span, col int) string {
	line := col < len(l.Columns) && l.Columns[col].Line
	if col < len(styled) && styled[col] != nil {
		spans := styled[col]
		if line {
			spans = slices.Clone(spans)
			for i := range spans {
				spans[i].Text = expandTabs(spans[i].Text)
			}
		}
		return spansText(spans)
	}
	text := ""
	if col < len(cells) {
		text = cells[col]
	}
	if line {
		return Escape(expandTabs(text))
	}
	return Escape(text)
}

// belowText is row's below row k as drawn in room cells, the columns ws
// wide: its cells under the columns, its last cell with text running on to
// the row's end (tdp panel.5).
func (l *List) belowText(row, k int, ws []int, room int) string {
	b := l.rows[row].Below[k]
	last := -1
	for c := range ws {
		if ws[c] >= 0 && (c < len(b.Cells) && b.Cells[c] != "" || c < len(b.Styled) && b.Styled[c] != nil) {
			last = c
		}
	}
	var parts []string
	used := 0
	for c := range ws {
		if ws[c] < 0 {
			continue
		}
		v := l.shownOf(b.Cells, b.Styled, c)
		if c == last {
			if c < len(l.Columns) && l.Columns[c].Path {
				parts = append(parts, FitLeft(v, max(room-used, 0)))
			} else {
				parts = append(parts, Fit(v, max(room-used, 0)))
			}
			break
		}
		parts = append(parts, Fit(v, ws[c]))
		used += ws[c] + Width(l.gap())
	}
	return strings.Join(parts, l.gap())
}

// matches reports whether row's text, any of its rows', matches q (tdp
// panel.14, panel.5).
func (l *List) matches(row int, q string) bool {
	r := &l.rows[row]
	if Matches(strings.Join(r.Cells, "\t"), q) {
		return true
	}
	for _, b := range r.Below {
		if Matches(strings.Join(b.Cells, "\t"), q) {
			return true
		}
	}
	return false
}

// compare orders two rows by one column: numbers by value, text as text.
func (l *List) compare(col, a, b int) int {
	x, y := l.sortValue(a, col), l.sortValue(b, col)
	if col < len(l.Columns) && l.Columns[col].Number {
		nx, ex := leadingNumber(x)
		ny, ey := leadingNumber(y)
		if ex == nil && ey == nil {
			return cmp.Compare(nx, ny)
		}
	}
	return strings.Compare(strings.ToLower(x), strings.ToLower(y))
}

// leadingNumber reads the number a value starts with ("12", "3.5", "9d").
func leadingNumber(s string) (float64, error) {
	s = strings.TrimSpace(s)
	end := 0
	for end < len(s) && (s[end] >= '0' && s[end] <= '9' || s[end] == '.' || (end == 0 && s[end] == '-')) {
		end++
	}
	return strconv.ParseFloat(s[:end], 64)
}

// locationRow is a panel's location row (tdp panel.10) in w cells: one
// cell from the frame on the left, the arrow's room on the right, in
// Lavender, cut from the front; a glyph, when there is one, first with a
// space after it, staying when the text is cut.
func locationRow(glyph, loc string, w int) string {
	room := w - 1 - Width(glyphHard)
	if glyph != "" {
		glyph = Escape(glyph) + " "
	}
	return " " + Pad(Fg(Lavender).Render(glyph+TruncateLeft(Escape(loc), room-Width(glyph))), w-1)
}

// cursorColour is the colour the cursor of row r is drawn in, where the fade
// of the item's other rows starts from (tdp panel.5): Subtext1 or the row's
// own (tdp color.2), Yellow while held (Drag), dimmed once while typing in
// the filter (tdp P6).
func (l *List) cursorColour(r int) Color {
	colour := Subtext1
	if c := l.rows[r].CursorColor; c.Set && !l.inMode {
		colour = c
	}
	if l.held {
		colour = Yellow
	}
	if l.Typing() {
		colour = DimColor(colour)
	}
	return colour
}

// NoteAt is where the item under the cursor was at the last Draw: its
// first and last rows and the cell its text starts at, for its note's head
// (Panel.Note, tdp panel.18).
func (l *List) NoteAt() (top, bottom, col int, ok bool) {
	return l.noteTop, l.noteBottom, l.noteCol, l.noteTop >= 0
}

// Uncover keeps the cursor's row above the rows at the bottom of the list
// that the inspect toast covers (tdp toast.4); the lib calls it before each
// Draw.
func (l *List) Uncover(rows int) { l.cover = rows }

// Draw draws the list in w × h: the location row, the filter's typing row,
// the header, the rows.
func (l *List) Draw(w, h int, focused bool) []string { return finishFades(l.draw(w, h, focused, true)) }

// draw draws the list, its location row only when own: a panel draws a
// list's location row itself (Panel.Location).
func (l *List) draw(w, h int, focused, own bool) []string {
	out := make([]string, 0, h)
	l.noteTop, l.noteBottom = -1, -1
	if own && l.Location != "" {
		out = append(out, locationRow(l.LocationGlyph, l.Location, w))
	}
	if l.filter != nil {
		out = append(out, l.filterRow(w, focused))
	}
	if len(l.rows) == 0 {
		// The location stays: an empty directory is still somewhere.

		rest := EmptyState(w, h-len(out), l.EmptyFact, l.EmptyHint)
		l.scroll = ""
		return append(out, rest...)[:max(h, 0)]
	}
	arrow := Width(glyphHard)
	markW := 0
	if l.Marks || l.Draggable || l.Flags {
		markW = IconWidth() + 1
	}
	indent := ""
	if l.groups && markW == 0 {
		// Items two cells further in than their group's title; a mark column
		// goes in those two cells (tdp panel.11).
		indent = "  "
	}
	textW := w - 1 - arrow - len(indent) // one cell of padding on the left, the arrow's on the right (tdp panel)
	l.noteCol = 1 + len(indent) + markW
	widths := l.widths(textW - markW)
	if l.headed() {
		// The header goes in with the items, so the columns line up.
		out = append(out, " "+indent+Repeat(" ", markW)+Pad(Fg(Blue).Render(l.header(widths)), textW-markW)+Repeat(" ", arrow))
	}
	rows := h - len(out)
	l.height = rows
	if rows <= 0 {
		l.scroll = ""
		return out[:h]
	}
	lines := l.lines()
	cl := slices.IndexFunc(lines, func(ln listLine) bool { return ln.item == l.cursor })
	// Scroll the least that keeps the cursor in sight, with its group's
	// title when it is on the group's first row.
	want := cl
	if cl > 0 && lines[cl-1].item < 0 {
		want = cl - 1
	}
	if want < l.top {
		l.top = want
	}
	seen := max(rows-l.cover, 1) // the rows above the inspect toast (tdp toast.4)
	end := cl                    // the cursor's item's last row: all of it in sight (tdp panel.5)
	for end+1 < len(lines) && lines[end+1].item == l.cursor && lines[end+1].below > 0 {
		end++
	}
	if end >= l.top+seen {
		l.top = end - seen + 1
	}
	l.top = min(l.top, max(want, 0)) // one taller than the view: from its first row
	l.top = min(max(l.top, 0), max(len(lines)-seen, 0))
	for i := 0; i < rows; i++ {
		li := l.top + i
		if li >= len(lines) {
			out = append(out, Repeat(" ", w))
			continue
		}
		if lines[li].item == l.cursor && l.cursor >= 0 { // where its note's head goes (tdp panel.18)
			if l.noteTop < 0 {
				l.noteTop = len(out)
			}
			l.noteBottom = len(out)
		}
		if lines[li].item < 0 {
			// A group title: Blue, upper case, not bold, one cell from the frame.
			out = append(out, " "+Pad(Fg(Blue).Render(Truncate(Escape(Upper(lines[li].title)), w-1-arrow)), w-1-arrow)+Repeat(" ", arrow))
			continue
		}
		vi := lines[li].item
		r := l.view[vi]
		if k := lines[li].below; k > 0 {
			// A row under the item's first: under the columns, the mark
			// column blank, its text in its own colours. The cursor's
			// item's, and an anchor's, fade out across the inside from the
			// first row's block colour dimmed twice, the cursor's colour
			// or Yellow, into the terminal's own background (tdp panel.5,
			// panel.9).
			row := " " + Pad(indent+Repeat(" ", markW)+l.belowText(r, k-1, widths, textW-markW), w-1-arrow)
			mark, marked := l.cursorColour(r), vi == l.cursor
			if l.Anchor != "" && l.rowID(r) == l.Anchor {
				mark, marked = Yellow, true // the cursor on it or not
			}
			if !marked {
				out = append(out, row+Repeat(" ", arrow))
				continue
			}
			out = append(out, fadeRow(row, DimRGB(DimRGB(mark.RGB)))+Repeat(" ", arrow))
			continue
		}
		// mark is the row's sign on a cursor block; shown, the one drawn
		// elsewhere: the tick in Green, a flag in its colour.
		mark, shown := "", ""
		if markW > 0 {
			mark = Repeat(" ", markW)
			shown = mark
			switch {
			case l.marked[l.rowID(r)]:
				mark = glyphMarked + " "
				shown = Fg(Green).Render(glyphMarked) + " "
			case l.Flags && !l.Marks && l.rows[r].Flag.Text != "":
				f := l.rows[r].Flag
				flag := Truncate(Escape(f.Text), markW-1)
				mark = Pad(flag, markW)
				shown = Pad(f.Style.Render(flag), markW)
			}
		}
		text := l.rowText(r, widths)
		if vi == l.cursor {
			colour := Subtext1
			if c := l.rows[r].CursorColor; c.Set && !l.inMode {
				colour = c // the row's own colour (tdp color.2)
			}
			cursor := On(colour)
			arrowStyle := Fg(colour)
			if l.held {
				// The held row: the cursor in the mode's colour, the swap
				// glyph in its mark column (tdp panel, tdp K11).
				cursor, arrowStyle = On(Yellow), Fg(Yellow)
				mark = Pad(glyphSwap, markW)
			}
			if l.Typing() {
				// The focus is on the typing row: the list's cursor is dimmed once (tdp P6).
				cursor = Style{Fg: DimColor(Base), Bg: DimColor(colour), Bold: true}
				arrowStyle = Fg(DimColor(colour))
			}
			// Base on the block, a swatch keeping its colour (tdp color).
			out = append(out, onBlock(Pad(" "+indent+mark+text, w-arrow), cursor)+arrowStyle.Render(glyphHard))
			continue
		}
		if l.Anchor != "" && l.rowID(r) == l.Anchor {
			out = append(out, onBlock(Pad(" "+indent+mark+text, w-arrow), Style{Fg: Base, Bg: Yellow})+Repeat(" ", arrow))
			continue
		}
		out = append(out, " "+indent+shown+Pad(text, textW-markW)+Repeat(" ", arrow))
	}
	l.scroll = ""
	if len(lines) > rows {
		l.scroll = scrollText(true, l.cursor+1, 0, len(l.view))
	}
	return out
}

// fadeRow is row on a background fading out from start across its cells
// into the terminal's own, its text keeping its colours (tdp panel.5): each
// cell carries a fade's mark (fadeSGR), so a dimmed panel dims the start and
// the fade still ends in the terminal's background.
func fadeRow(row string, start RGB) string {
	n := Width(row)
	var b strings.Builder
	col := 0
	walk(row, func(seq string, cells int) {
		if cells > 0 {
			b.WriteString(fadeSGR(start, col, n))
			col += cells
		}
		b.WriteString(seq)
	})
	return b.String() + "\x1b[0m"
}

// listLine is one line of the list's rows: a row (its index in view) or a
// group's title (item -1); below is which of the item's rows under its first
// it is, 0 for the first.
type listLine struct {
	item  int
	title string
	below int
}

// lines are the rows shown, each group's title before its first row, each
// item's rows under its first after it.
func (l *List) lines() []listLine {
	out := make([]listLine, 0, len(l.view))
	prev := ""
	for vi, r := range l.view {
		if g := l.rows[r].Group; g != "" && (vi == 0 || g != prev) {
			out = append(out, listLine{item: -1, title: g})
		}
		prev = l.rows[r].Group
		out = append(out, listLine{item: vi})
		for k := range l.rows[r].Below {
			out = append(out, listLine{item: vi, below: k + 1})
		}
	}
	return out
}

// pageItems is how many items from the cursor, down when dir is 1, fit in
// rows rows, at least one: a page moves by items, landing on one's first
// row (tdp panel.5).
func (l *List) pageItems(rows, dir int) int {
	n, used := 0, 0
	for i := l.cursor + dir; i >= 0 && i < len(l.view); i += dir {
		if used += 1 + len(l.rows[l.view[i]].Below); used > rows {
			break
		}
		n++
	}
	return max(n, 1)
}

// moved runs OnMove when the row under the cursor is another than last time.
func (l *List) moved() tea.Cmd {
	r, ok := l.Current()
	if !ok {
		l.at = ""
		return nil
	}
	at := "\x00" + l.rowID(l.view[l.cursor])
	if at == l.at {
		return nil
	}
	l.at = at
	if l.OnMove == nil {
		return nil
	}
	return l.OnMove(r)
}

// headed reports whether the list draws a header: one of a single column
// with no title has none (tdp panel).
func (l *List) headed() bool {
	return len(l.Columns) > 1 || len(l.Columns) == 1 && l.Columns[0].Title != ""
}

// widths are the columns' widths in w cells, two cells apart (tdp panel):
// a column as wide as its widest value or title, but the shrinkable ones,
// which share what the others leave, down to their MinWidth. While even that
// does not fit, columns are taken away whole in their Drop order (-1: not
// drawn).
func (l *List) widths(w int) []int {
	n := len(l.Columns)
	if n == 0 {
		return []int{w}
	}
	l.gapMore = 0
	ws := make([]int, n)
	for c := range n {
		ws[c] = Width(l.title(c))
		for r := range l.rows {
			ws[c] = max(ws[c], Width(l.shown(r, c)))
		}
	}
	least := func(c int) int {
		if !l.Columns[c].Shrink {
			return ws[c]
		}
		return min(ws[c], cmpInt(l.Columns[c].MinWidth, 12))
	}
	// total is the shown columns at their least, and how many are shown.
	total := func() (t, shown int) {
		for c, x := range ws {
			if x >= 0 {
				t, shown = t+least(c), shown+1
			}
		}
		return t + 2*(shown-1), shown
	}
	for t, shown := total(); t > w && shown > 1; t, shown = total() {
		pick := -1
		for c := range n {
			if d := l.Columns[c].Drop; ws[c] >= 0 && d > 0 && (pick < 0 || d < l.Columns[pick].Drop) {
				pick = c
			}
		}
		if pick < 0 {
			break
		}
		ws[pick] = -1
	}
	// The shrinkable columns share what the others leave: each its least,
	// then the rest in proportion to what more it would take.
	room := w
	var shrink []int
	want := 0
	for c, x := range ws {
		switch {
		case x < 0:
		case l.Columns[c].Shrink:
			shrink = append(shrink, c)
			room -= 2
			want += x
		default:
			room -= x + 2
		}
	}
	room += 2
	if want <= room || len(shrink) == 0 {
		if !slices.Contains(ws, -1) {
			l.spread(ws, w) // every column at its widest: the room left over
		}
		return cutLast(ws, w)
	}
	base, more := 0, 0
	for _, c := range shrink {
		base, more = base+least(c), more+ws[c]-least(c)
	}
	extra := max(room-base, 0)
	left := room
	for _, c := range shrink {
		nw := least(c)
		if more > 0 {
			nw += (ws[c] - least(c)) * extra / more
		}
		if room < base {
			nw = max(ws[c]*max(room, 0)/want, 1) // narrower than their least: the last resort
		}
		ws[c] = nw
		left -= nw
	}
	for _, c := range shrink { // what rounding down left, a cell each
		if left <= 0 {
			break
		}
		ws[c]++
		left--
	}
	return cutLast(ws, w)
}

// Spread is where a list's room left over goes (List.Spread; tdp
// panel.5).
type Spread int

const (
	// SpreadEnd leaves the room at the row's end, the columns keeping their
	// widths: the default.
	SpreadEnd Spread = iota
	// SpreadFill gives it all to the column marked Fill (a description, so
	// the columns after it sit at the right edge); with none, as End.
	SpreadFill
	// SpreadEven widens the narrowest columns first, until the columns are
	// as even as the room lets them be: one already wider keeps its width,
	// nothing is cut, a number stays right-aligned in its wider column.
	SpreadEven
	// SpreadGaps widens every gap between columns alike, by spreadGapMore
	// cells at most; what is left after goes to the Fill column, or with
	// none to the row's end.
	SpreadGaps
)

// spreadGapMore is how much a gap between columns may widen with
// SpreadGaps: from its two cells to six.
const spreadGapMore = 4

// spread places the room left over in w when every shown column is at its
// widest, as l.Spread says: in the columns' widths ws, or in the gaps
// (l.gapMore).
func (l *List) spread(ws []int, w int) {
	var shown []int
	used := -2
	for c, x := range ws {
		if x >= 0 {
			shown, used = append(shown, c), used+x+2
		}
	}
	left := w - used
	if left <= 0 || len(shown) == 0 {
		return
	}
	fill := -1
	for _, c := range shown {
		if l.Columns[c].Fill {
			fill = c
			break
		}
	}
	switch l.Spread {
	case SpreadFill:
		if fill >= 0 {
			ws[fill] += left
		}
	case SpreadEven:
		for left > 0 {
			low, next, at := -1, -1, 0
			for _, c := range shown {
				if low < 0 || ws[c] < low {
					low = ws[c]
				}
			}
			for _, c := range shown {
				switch {
				case ws[c] == low:
					at++
				case next < 0 || ws[c] < next:
					next = ws[c]
				}
			}
			step := left / at
			if next >= 0 {
				step = min(step, next-low)
			}
			if step == 0 { // fewer cells than columns at the level: one each, from the left
				for _, c := range shown {
					if ws[c] == low && left > 0 {
						ws[c]++
						left--
					}
				}
				break
			}
			for _, c := range shown {
				if ws[c] == low {
					ws[c] += step
				}
			}
			left -= step * at
		}
	case SpreadGaps:
		if len(shown) > 1 {
			l.gapMore = min(spreadGapMore, left/(len(shown)-1))
			left -= l.gapMore * (len(shown) - 1)
		}
		if fill >= 0 {
			ws[fill] += left
		}
	}
}

// gap is what is between two columns: two cells, more with SpreadGaps.
func (l *List) gap() string { return Repeat(" ", 2+l.gapMore) }

// cutLast narrows the column the edge falls in to what is left of w, so it is
// cut its own way with "…", not chopped at the frame; the ones after it are
// not drawn (tdp panel.5).
func cutLast(ws []int, w int) []int {
	used := 0
	for c, x := range ws {
		if x < 0 {
			continue
		}
		gap := 0
		if used > 0 {
			gap = 2
		}
		if used+gap+x <= w {
			used += gap + x
			continue
		}
		ws[c] = w - used - gap
		if ws[c] < 1 {
			ws[c] = -1
		}
		for k := c + 1; k < len(ws); k++ {
			ws[k] = -1
		}
		break
	}
	return ws
}

// cmpInt is n, or def when n is 0.
func cmpInt(n, def int) int {
	if n == 0 {
		return def
	}
	return n
}

// title is a column's header: upper case, with its sort order and arrow.
func (l *List) title(c int) string {
	t := Upper(l.Columns[c].Title)
	for i, k := range l.sort {
		if k.Col != c {
			continue
		}
		if len(l.sort) > 1 {
			t += " (" + strconv.Itoa(i+1) + ")"
		}
		if k.Desc {
			t += " " + glyphSortDesc
		} else {
			t += " " + glyphSortAsc
		}
	}
	return t
}

func (l *List) header(ws []int) string {
	var parts []string
	for c := range ws {
		if ws[c] < 0 {
			continue // taken away
		}
		t := Truncate(l.title(c), ws[c])
		if l.Columns[c].Number {
			parts = append(parts, PadLeft(t, ws[c]))
		} else {
			parts = append(parts, Pad(t, ws[c]))
		}
	}
	return strings.Join(parts, l.gap())
}

func (l *List) rowText(r int, ws []int) string {
	if len(l.Columns) == 0 {
		return Truncate(l.shown(r, 0), ws[0])
	}
	var parts []string
	for c := range ws {
		if ws[c] < 0 {
			continue // taken away
		}
		v := l.shown(r, c)
		col := l.Columns[c]
		switch {
		case col.Path:
			v = FitLeft(v, ws[c])
		case col.Middle:
			v = Pad(TruncateMiddle(v, ws[c]), ws[c])
		case col.Number:
			v = PadLeft(Truncate(v, ws[c]), ws[c])
		default:
			v = Fit(v, ws[c])
		}
		parts = append(parts, v)
	}
	return strings.Join(parts, l.gap())
}

// filterRow is the typing row: the typed words one cell from the frame, the
// count of matches over the total on the right; grey when the focus is on
// the list. Typing means the panel has the focus: the angle takes its frame's
// colour, Blue (tdp panel.14).
func (l *List) filterRow(w int, focused bool) string {
	count := strconv.Itoa(len(l.view)) + "/" + strconv.Itoa(len(l.rows))
	return typingRow(l.filter, w, count, l.typing && focused, Blue)
}

// typingRow is a typing row that filters (a finder's, a panel filter's), w
// wide: nf-fa-angle_right one cell from the frame, a space, the typed words,
// the count of matches on the right in Overlay0 (tdp finder).
// Lit while it has the focus: the angle in lit (a popup's layer colour, a
// panel's frame colour), the words Lavender with a cursor; the whole row is
// Overlay0 otherwise.
func typingRow(ed *lineEditor, w int, count string, focus bool, lit Color) string {
	room := max(w-2-angleWidth()-Width(count)-2, 1)
	if focus {
		return " " + typingAngle(lit) + ed.view(room, Fg(Lavender), Style{Fg: Base, Bg: Lavender}, true, nil, "") +
			"  " + Fg(Overlay0).Render(count) + " "
	}
	return " " + typingAngle(Overlay0) + ed.view(room, Fg(Overlay0), Fg(Overlay0), false, nil, "") +
		"  " + Fg(Overlay0).Render(count) + " "
}

// typingAngle is what every typing row starts with: nf-fa-angle_right in c,
// then a space (tdp input, finder).
func typingAngle(c Color) string { return Fg(c).Render(glyphTyping) + " " }

// angleWidth is how wide typingAngle is.
func angleWidth() int { return Width(glyphTyping) + 1 }

// Scroll is "N of M" when the rows do not all fit.
func (l *List) Scroll() string { return l.scroll }

// MoveKeys are the list's movement keys; none when nothing is shown,
// nothing to move to (tdp M6).
func (l *List) MoveKeys() []KeyRow {
	if len(l.view) == 0 {
		return nil
	}
	return []KeyRow{
		{Key: "j|k", Desc: "down, up"},
		{Key: "u|d", Desc: "half a page up, down"},
		{Key: "gg|G", Desc: "to the top, the bottom"},
	}
}

// Actions are the list's own panel operations: the filter. Filtering only
// changes what is in view, so it does not go in the App Log (tdp note.7).
func (l *List) Actions() []Action {
	if !l.Filterable {
		return nil
	}
	return []Action{{Key: "/", Label: "Filter", Desc: "narrow the list as you type", Scope: PanelScope,
		Do: func() tea.Cmd { l.OpenFilter(); return nil }}}
}

// Key moves the cursor and types in the filter.
func (l *List) Key(key string, msg tea.KeyMsg) (bool, tea.Cmd) {
	if l.Typing() {
		return l.typeKey(key, msg), nil
	}
	n := len(l.view)
	half := max(l.height/2, 1)
	switch key {
	case "j", "↓", "l", "→":
		if n > 0 {
			l.cursor = (l.cursor + 1) % n
		}
	case "k", "↑", "h", "←":
		if n > 0 {
			l.cursor = (l.cursor - 1 + n) % n
		}
	case "d", "PgDn":
		l.cursor = min(l.cursor+l.pageItems(half, 1), max(n-1, 0))
	case "u", "PgUp":
		l.cursor = max(l.cursor-l.pageItems(half, -1), 0)
	case "gg":
		l.cursor = 0
	case "G":
		l.cursor = max(n-1, 0)
	case "/":
		if l.Filterable {
			l.OpenFilter()
			return true, nil
		}
		return false, nil
	default:
		return false, nil
	}
	return true, nil
}

// typeKey is a key on the filter's typing row (tdp finder).
func (l *List) typeKey(key string, msg tea.KeyMsg) bool {
	n := len(l.view)
	switch key {
	case "Enter":
		if n > 0 {
			l.typing = false
		}
	case "Tab":
		l.typing = false
	case "↓":
		if n > 0 {
			l.cursor = (l.cursor + 1) % n
		}
	case "↑":
		if n > 0 {
			l.cursor = (l.cursor - 1 + n) % n
		}
	default:
		if !l.filter.key(key, msg) {
			return false
		}
		l.rebuild()
		l.cursor, l.top = 0, 0
	}
	return true
}
