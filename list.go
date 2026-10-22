package terminu

import (
	"cmp"
	"slices"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Column is one column of a List (layout/panel, ordinary lists).
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
	// of their widest value (layout/panel).
	Shrink   bool
	MinWidth int
	// Drop is when the column is taken away, whole and its header with it,
	// when the columns do not fit even with the shrinkable ones at their
	// MinWidth: 1 first, then 2 and on; 0 never (the name). As the panel
	// grows they come back in order, and the shrinkable ones grow back
	// (layout/panel).
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
}

// Row is one item of a List.
type Row struct {
	ID string // what marks are kept by; the row's place when empty
	// Cells are the row's text, one per column: a control character in one,
	// ESC too, is drawn by its name (Escape; layout/panel). The filter and
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
	// files); no title is drawn between them (layout/panel). The rows of one
	// group keep their order when the list is not sorted.
	SortGroup int
	// Group is the title of the group the row is in (layout/panel, groups).
	// A group's rows stay together, the groups in the order they first
	// appear, sorted within; the rows are drawn two cells further in, under
	// a title the cursor skips. A group with no row shown has no title.
	Group string
}

// SortKey is one column the list is sorted by.
type SortKey struct {
	Col  int
	Desc bool
}

// List is a panel body with one item per row: a header, a cursor, marks, a
// panel filter (layout/panel).
type List struct {
	Columns []Column
	// Marks keeps a column for marks in front of every row (Principle P3,
	// batches of marked items).
	Marks bool
	// Draggable keeps that column too, for the swap glyph of the row held in
	// a Drag mode (layout/panel): Hold, MoveHeld, Drop.
	Draggable bool
	// Anchor is the ID of a row a mode keeps picked while the cursor moves
	// on (a Compare mode's first pick, Rules K11): drawn whole in Yellow,
	// the mode's, with Base text, not bold and no arrow, as a selected range
	// (components/color, dialog/note). The cursor on it wins. "" is none.
	Anchor string
	// Filterable lets / open a panel filter on the list.
	Filterable bool
	// EmptyFact and EmptyHint are what an empty list says (layout/screen,
	// empty states); keys in the hint written in brackets.
	EmptyFact, EmptyHint string
	// Location, when set, is the list's location row: where the user is (a
	// directory, a remote path, the level drilled into), in Lavender on the
	// panel's first row, cut from the front. It is only shown: the cursor
	// does not stop on it and the scroll position does not count it
	// (layout/panel, location row). Whether a list has one is the app's.
	Location string
	// OnMove runs when the cursor comes to another row, by a key or not, the
	// first time included: before the frame is drawn, so a details panel
	// shows that row in the same frame (layout/panel, details). It does not
	// run while no row is under the cursor.
	OnMove func(Row) tea.Cmd

	rows   []Row
	groups bool   // some row has a Group
	at     string // the row OnMove last ran for, "\x00" + its ID
	sort   []SortKey
	view   []int // rows shown, in order: sorted, then filtered
	cursor int   // index into view
	top    int   // first row shown
	marked map[string]bool
	held   bool // the row under the cursor is held in a Drag mode

	filter *lineEditor
	typing bool
	scroll string
	height int // rows of items the last Draw had room for
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
// text (layout/panel). nil keeps the rows' order. Rows the keys cannot tell
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
// (New, Copy, Import), it selects it (Rules K3); a row the panel filter hides
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
// (layout/panel): it is drawn as the cursor in the mode's colour, Yellow,
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
// the top and the others move down one (Rules K12). It moves the rows
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
// to type on (layout/panel, panel filter).
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
			if Matches(strings.Join(l.rows[i].Cells, "\t"), q) {
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
	line := col < len(l.Columns) && l.Columns[col].Line
	if col < len(r.Styled) && r.Styled[col] != nil {
		spans := r.Styled[col]
		if line {
			spans = slices.Clone(spans)
			for i := range spans {
				spans[i].Text = expandTabs(spans[i].Text)
			}
		}
		return spansText(spans)
	}
	if line {
		return Escape(expandTabs(l.cell(row, col)))
	}
	return Escape(l.cell(row, col))
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

// Draw draws the list in w × h: the location row, the filter's typing row,
// the header, the rows.
func (l *List) Draw(w, h int, focused bool) []string {
	out := make([]string, 0, h)
	if l.Location != "" {
		// One cell from the frame on the left, the arrow's room on the right.
		room := w - 1 - Width(glyphHard)
		out = append(out, " "+Pad(Fg(Lavender).Render(TruncateLeft(Escape(l.Location), room)), w-1))
	}
	if l.filter != nil {
		out = append(out, l.filterRow(w, focused))
	}
	if len(l.rows) == 0 {
		// The location stays: an empty directory is still somewhere (G98).

		rest := EmptyState(w, h-len(out), l.EmptyFact, l.EmptyHint)
		l.scroll = ""
		return append(out, rest...)[:max(h, 0)]
	}
	arrow := Width(glyphHard)
	markW := 0
	if l.Marks || l.Draggable {
		markW = IconWidth() + 1
	}
	indent := ""
	if l.groups && markW == 0 {
		// Items two cells further in than their group's title; a mark column
		// goes in those two cells (layout/panel, groups).
		indent = "  "
	}
	textW := w - 1 - arrow - len(indent) // one cell of padding on the left, the arrow's on the right (layout/panel)
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
	if cl >= l.top+rows {
		l.top = cl - rows + 1
	}
	l.top = min(max(l.top, 0), max(len(lines)-rows, 0))
	for i := 0; i < rows; i++ {
		li := l.top + i
		if li >= len(lines) {
			out = append(out, Repeat(" ", w))
			continue
		}
		if lines[li].item < 0 {
			// A group title: Blue, upper case, not bold, one cell from the frame.
			out = append(out, " "+Pad(Fg(Blue).Render(Truncate(Escape(Upper(lines[li].title)), w-1-arrow)), w-1-arrow)+Repeat(" ", arrow))
			continue
		}
		vi := lines[li].item
		r := l.view[vi]
		mark := ""
		if markW > 0 {
			if l.marked[l.rowID(r)] {
				mark = glyphMarked + " "
			} else {
				mark = Repeat(" ", markW)
			}
		}
		text := l.rowText(r, widths)
		if vi == l.cursor {
			cursor := On(Subtext1)
			arrowStyle := Fg(Subtext1)
			if l.held {
				// The held row: the cursor in the mode's colour, the swap
				// glyph in its mark column (layout/panel, Rules K11).
				cursor, arrowStyle = On(Yellow), Fg(Yellow)
				mark = Pad(glyphSwap, markW)
			}
			if l.Typing() {
				// The focus is on the typing row: the list's cursor is dimmed once (P6).
				cursor = Style{Fg: DimColor(Base), Bg: DimColor(Subtext1), Bold: true}
				arrowStyle = Fg(DimColor(Subtext1))
			}
			// Base on the block, a swatch keeping its colour (components/color).
			out = append(out, onBlock(Pad(" "+indent+mark+text, w-arrow), cursor)+arrowStyle.Render(glyphHard))
			continue
		}
		if l.Anchor != "" && l.rowID(r) == l.Anchor {
			out = append(out, onBlock(Pad(" "+indent+mark+text, w-arrow), Style{Fg: Base, Bg: Yellow})+Repeat(" ", arrow))
			continue
		}
		if mark != "" && l.marked[l.rowID(r)] {
			mark = Fg(Green).Render(glyphMarked) + " "
		}
		out = append(out, " "+indent+mark+Pad(text, textW-markW)+Repeat(" ", arrow))
	}
	l.scroll = ""
	if len(lines) > rows {
		l.scroll = scrollText(true, l.cursor+1, 0, len(l.view))
	}
	return out
}

// listLine is one line of the list's rows: a row (its index in view) or a
// group's title (item -1).
type listLine struct {
	item  int
	title string
}

// lines are the rows shown, each group's title before its first row.
func (l *List) lines() []listLine {
	out := make([]listLine, 0, len(l.view))
	prev := ""
	for vi, r := range l.view {
		if g := l.rows[r].Group; g != "" && (vi == 0 || g != prev) {
			out = append(out, listLine{item: -1, title: g})
		}
		prev = l.rows[r].Group
		out = append(out, listLine{item: vi})
	}
	return out
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
// with no title has none (layout/panel).
func (l *List) headed() bool {
	return len(l.Columns) > 1 || len(l.Columns) == 1 && l.Columns[0].Title != ""
}

// widths are the columns' widths in w cells, two cells apart (layout/panel):
// a column as wide as its widest value or title, but the shrinkable ones,
// which share what the others leave, down to their MinWidth. While even that
// does not fit, columns are taken away whole in their Drop order (-1: not
// drawn).
func (l *List) widths(w int) []int {
	n := len(l.Columns)
	if n == 0 {
		return []int{w}
	}
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

// cutLast narrows the column the edge falls in to what is left of w, so it is
// cut its own way with "…", not chopped at the frame; the ones after it are
// not drawn (layout/panel, truncation).
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
	return strings.Join(parts, "  ")
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
	return strings.Join(parts, "  ")
}

// filterRow is the typing row: the typed words one cell from the frame, the
// count of matches over the total on the right; grey when the focus is on
// the list. Typing means the panel has the focus: the angle takes its frame's
// colour, Blue (components/layout/panel, panel filter).
func (l *List) filterRow(w int, focused bool) string {
	count := strconv.Itoa(len(l.view)) + "/" + strconv.Itoa(len(l.rows))
	return typingRow(l.filter, w, count, l.typing && focused, Blue)
}

// typingRow is a typing row that filters (a finder's, a panel filter's), w
// wide: nf-fa-angle_right one cell from the frame, a space, the typed words,
// the count of matches on the right in Overlay0 (components/input/finder).
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
// then a space (components/input/README, components/input/finder).
func typingAngle(c Color) string { return Fg(c).Render(glyphTyping) + " " }

// angleWidth is how wide typingAngle is.
func angleWidth() int { return Width(glyphTyping) + 1 }

// Scroll is "N of M" when the rows do not all fit.
func (l *List) Scroll() string { return l.scroll }

// MoveKeys are the list's movement keys; none when nothing is shown,
// nothing to move to (Rules M6).
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

// Actions are the list's own panel operations: the filter.
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
		l.cursor = min(l.cursor+half, max(n-1, 0))
	case "u", "PgUp":
		l.cursor = max(l.cursor-half, 0)
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

// typeKey is a key on the filter's typing row (components/input/finder).
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
