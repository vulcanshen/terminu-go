package terminu

import (
	"errors"
	"slices"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const glyphCompare = "\U000F018A" // nf-md-compare: the compare popup (tdp note)

// Compare is what a list's compare mode needs from the app
// (tdp panel.9; tdp note.8):
// the key, which rows can be compared and their text. The lib does the rest.
type Compare struct {
	// Key is the compare key (the app's choice, kbu's C): on a row it enters
	// the mode with that row as the anchor; in the mode it compares, as
	// Enter does.
	Key string
	// Can reports a row that can be compared: text, shown as text, or one
	// that holds many (Many). A row it says no to has no Compare action,
	// and in the mode the compare keys are off on it (tdp M2, M6). Nil:
	// every row.
	Can func(Row) bool
	// Name is the row as the title names it; its first cell when nil.
	Name func(Row) string
	// Text is the row's text, compared line by line.
	Text func(Row) (string, error)
	// Split does nothing.
	//
	// Deprecated: every comparison opens on UNIFIED and the tab chosen is
	// not kept (tdp note.8). The field stays so that code setting it still
	// builds; a later minor version takes it away.
	Split *bool
	// Many reports a row that holds many (a directory, a namespace, a
	// commit). Two such rows are not compared as text: the mode goes in to
	// the list of the items that differ between them, which Items gives, as
	// OpenDiff goes in (tdp panel.9), the location row <anchor> ↔ <the
	// other> as Name names them. A row that holds many and one that does
	// not are not compared: the compare keys are off (tdp M6). Nil: no row
	// holds many.
	Many func(Row) bool
	// Items gives the items that differ between two rows Many says yes to,
	// the anchor first, in the order they are listed, the ones the same on
	// both sides left out (DiffItem).
	Items func(anchor, other Row) ([]DiffItem, error)
}

func (c Compare) can(r Row) bool  { return c.Can == nil || c.Can(r) }
func (c Compare) many(r Row) bool { return c.Many != nil && c.Many(r) }

func (c Compare) name(r Row) string {
	if c.Name != nil {
		return c.Name(r)
	}
	if len(r.Cells) > 0 {
		return r.Cells[0]
	}
	return r.ID
}

// DiffMark is how an item differs, the anchor being the base (tdp
// panel.9): drawn before its name, in its colour.
type DiffMark byte

const (
	DiffAdded   DiffMark = '+' // only the other has it: Green
	DiffRemoved DiffMark = '-' // only the anchor has it: Red
	DiffChanged DiffMark = '~' // both have it, changed: Peach; a rename the app knows of too
)

func (m DiffMark) span() Span {
	switch m {
	case DiffAdded:
		return Span{"+", Fg(Green)}
	case DiffRemoved:
		return Span{"-", Fg(Red)}
	}
	return Span{"~", Fg(Peach)}
}

// DiffItem is an item that differs between two things that hold many (two
// directories, two namespaces, a commit and the one before it), or one
// item's two versions (tdp panel.9, note.8). The app names it and its two
// sides and gives what is in them; the lib draws the rest.
type DiffItem struct {
	// Mark is how the item differs. OpenDiff does not use it.
	Mark DiffMark
	// Name is the item as the list shows it after its mark, named as the
	// app names it (a path, internal/ui/ui.go); one the app knows was
	// renamed or moved is DiffChanged, named "<old> → <new>".
	Name string
	// Anchor and Other name the item's two sides as the app names them, so
	// that they say where, the older one the anchor (ui.go @ 1a2b3c4,
	// ui.go @ 5d6e7f8): its comparison is titled DIFF — <Anchor> ↔ <Other>,
	// the same however it is reached, and the list it goes into has
	// <Anchor> ↔ <Other> for its location row, in place of the panel's own
	// (tdp panel.9).
	Anchor, Other string
	// Text gives the item's text on each side, "" on a side it is not on:
	// opening it opens its comparison, UNIFIED and SPLIT (tdp note.8).
	Text func() (anchor, other string, err error)
	// Items, for an item that holds many itself, gives the items that
	// differ in it: opening it goes in to them, as Compare.Items.
	Items func() ([]DiffItem, error)
	// With neither the item is not text: it has no comparison, Enter off on
	// it (tdp M6).
}

func (d DiffItem) opens() bool { return d.Text != nil || d.Items != nil }

// splitMin is the narrowest terminal SPLIT is drawn on, as the finder's
// side by side (tdp note.8).
const splitMin = 96

// comparing is a compare mode under way.
type comparing struct {
	p    *Panel
	l    *List
	c    Compare
	mode *Mode
	name string // the anchor's
}

// CompareAction is l's Compare item action, for p's Actions: on a row c can
// compare it enters the compare mode (tdp K11), that row the anchor
// (List.Anchor). In the mode j|k, u|d and gg|G move; c.Key or Enter
// compares the row under the cursor with the anchor, off on the anchor, on
// a row c cannot compare and on a row that holds many against one that
// does not (tdp M6); the same text on both sides is an Info toast, not a
// popup. Two rows that hold many go in to the list of the items that
// differ, as OpenDiff does, the panel staying in the mode: its keys work
// there, on the item under the cursor, and Esc goes back up a level, the
// last back to the mode with the anchor kept. Esc leaves it, the anchor
// gone; Tab and the digits wait; an anchor no longer in the list ends it
// with a Warning toast, the lists gone into closed. The rows need their
// IDs. The mode does not go in the App Log; its toasts do, as every toast
// (tdp note.7).
func (a *App) CompareAction(p *Panel, l *List, c Compare) Action {
	return Action{Key: c.Key, Label: "Compare", Desc: "with another row, picked next", Scope: ItemScope,
		Hidden: func() bool { r, ok := l.Current(); return !ok || !c.can(r) },
		Do: func() tea.Cmd {
			if r, ok := l.Current(); ok && c.can(r) {
				a.startCompare(p, l, c, r)
			}
			return nil
		}}
}

func (a *App) startCompare(p *Panel, l *List, c Compare, anchor Row) {
	l.Anchor = anchor.ID
	cm := &comparing{p: p, l: l, c: c, name: c.name(anchor)}
	// in is the list of the items that differ gone into in the mode, nil
	// while in none: the mode's keys work on it.
	in := func() *diffLevel {
		if d := p.level(); d != nil && d.mode == cm.mode {
			return d
		}
		return nil
	}
	move := func(key string) tea.Cmd {
		if d := in(); d != nil {
			_, cmd := d.list.Key(key, tea.KeyMsg{})
			return cmd
		}
		_, cmd := l.Key(key, tea.KeyMsg{})
		return cmd
	}
	off := func() bool {
		if d := in(); d != nil {
			it, ok := d.current()
			return !ok || !it.opens()
		}
		r, ok := l.Current()
		an, found := l.rowByID(l.Anchor)
		return !ok || r.ID == l.Anchor || !c.can(r) || found && c.many(r) != c.many(an)
	}
	compare := ModeKey{Key: c.Key + "|Enter", Keys: []string{c.Key, "Enter"}, Desc: "compare", Disabled: off,
		Item: true, Label: "Compare", MenuDesc: "with " + cm.name,
		Do: func(string) tea.Cmd {
			if d := in(); d != nil {
				if it, ok := d.current(); ok {
					return a.openDiff(p, it)
				}
				return nil
			}
			return a.compareWith(cm)
		}}
	compare.menuDesc = func() string {
		if d := in(); d != nil {
			if it, _ := d.current(); it.Items != nil {
				return "the items that differ in it"
			}
			return "its two sides"
		}
		return "with " + cm.name
	}
	cm.mode = &Mode{Name: "Compare", Exit: "leave", Pause: true, Keys: []ModeKey{
		{Key: "j|k", Keys: []string{"j", "k", "↓", "↑"}, Desc: "down, up", NoFooter: true, Do: move},
		{Key: "u|d", Keys: []string{"u", "d", "PgUp", "PgDn"}, Desc: "half a page up, down", NoFooter: true, Do: move},
		{Key: "gg|G", Keys: []string{"gg", "G"}, Desc: "to the top, the bottom", NoFooter: true, Do: move},
		compare,
	}, OnExit: func() tea.Cmd { l.Anchor = ""; return nil }}
	p.EnterMode(cm.mode)
	a.comparings = append(a.comparings, cm)
}

// sideErr is an error reading one side, named as the error popup names it.
type sideErr struct {
	name string
	err  error
}

func (e sideErr) Error() string { return e.err.Error() }

// compareWith compares the anchor with the row under the cursor: their
// comparison, or the list of the items that differ when both hold many.
func (a *App) compareWith(cm *comparing) tea.Cmd {
	l, c := cm.l, cm.c
	other, ok := l.Current()
	anchor, found := l.rowByID(l.Anchor)
	if !ok || !found || other.ID == anchor.ID || !c.can(other) || c.many(anchor) != c.many(other) {
		return nil
	}
	item := DiffItem{Anchor: c.name(anchor), Other: c.name(other)}
	if c.many(anchor) {
		item.Items = func() ([]DiffItem, error) {
			if c.Items == nil {
				return nil, nil
			}
			return c.Items(anchor, other)
		}
	} else {
		item.Text = func() (string, string, error) {
			at, err := c.Text(anchor)
			if err != nil {
				return "", "", sideErr{item.Anchor, err}
			}
			bt, err := c.Text(other)
			if err != nil {
				return "", "", sideErr{item.Other, err}
			}
			return at, bt, nil
		}
	}
	return a.openDiff(cm.p, item)
}

// OpenDiff opens item as Enter does on it in a list of the items that
// differ (tdp note.8, panel.9), for an action of the app's own showing two
// versions of one item (a file in a commit against the commit before it, a
// file being edited against its last commit) or going into what changed in
// one (a commit against the one before it):
//   - with Text, its comparison, titled DIFF — <Anchor> ↔ <Other>, or the
//     Info toast "No differences between <Anchor> and <Other>" when the
//     two are the same;
//   - with Items, the list of the items that differ in it, in p in place
//     of p's body as panel.10 goes into an item, its location row
//     <Anchor> ↔ <Other> in place of p's own, or that toast when there is
//     none. Each item is its mark in its colour, then its name. j|k, u|d
//     and gg|G move, Enter opens the item under the cursor the same way
//     (off on one with neither Text nor Items), Esc goes back up a level,
//     the cursor on the item it came out of, the last back to p, its
//     cursor where it was; the Space menu and ? have the list's keys, not
//     p's actions, and Tab and the digits go to the other panels as ever,
//     the list kept. A filter on p is cleared before going in, as Esc
//     clears it (tdp panel.14).
//
// A comparison opens on UNIFIED; SPLIT is off narrower than 96 columns. An
// error from Text or Items is an error popup. Mark is not used. p is out of
// a mode: a compare mode goes in so itself (CompareAction).
func (a *App) OpenDiff(p *Panel, item DiffItem) tea.Cmd { return a.openDiff(p, item) }

func (a *App) openDiff(p *Panel, item DiffItem) tea.Cmd {
	subject := item.Name
	if subject == "" {
		subject = item.Anchor + " ↔ " + item.Other
	}
	fail := func(err error) tea.Cmd {
		if se := (sideErr{}); errors.As(err, &se) {
			return a.ErrorAbout("Compare", se.name, se.err.Error())
		}
		return a.ErrorAbout("Compare", subject, err.Error())
	}
	same := func() tea.Cmd { return a.Toast(Info, "No differences between "+item.Anchor+" and "+item.Other) }
	switch {
	case item.Items != nil:
		items, err := item.Items()
		if err != nil {
			return fail(err)
		}
		if len(items) == 0 {
			return same()
		}
		a.goInDiff(p, item, items)
	case item.Text != nil:
		at, bt, err := item.Text()
		if err != nil {
			return fail(err)
		}
		if at == bt {
			return same()
		}
		return a.diffViewer(item.Anchor, item.Other, at, bt)
	}
	return nil
}

// diffLevel is a list of the items that differ gone into in a panel, in
// place of its body (tdp panel.9): mode is the mode the panel was in, nil
// for none.
type diffLevel struct {
	list  *List
	items []DiffItem
	mode  *Mode
	acts  []Action
}

// current is the item under the list's cursor.
func (d *diffLevel) current() (DiffItem, bool) {
	r, ok := d.list.Current()
	if !ok {
		return DiffItem{}, false
	}
	i, _ := strconv.Atoi(r.ID)
	return d.items[i], true
}

// goInDiff goes into items, the items that differ in item, in p. It
// changes what p lists, so p's filter is cleared first, whether or not it
// hides the item (tdp panel.14): from OpenDiff and from a compare mode
// alike, the app having no way to clear it before.
func (a *App) goInDiff(p *Panel, item DiffItem, items []DiffItem) {
	if l, ok := p.body().(*List); ok && l.Filtering() {
		l.ClearFilter()
	}
	d := &diffLevel{items: items, mode: p.mode,
		list: &List{Columns: []Column{{Shrink: true, Path: true}}, Flags: true, Location: item.Anchor + " ↔ " + item.Other}}
	rows := make([]Row, len(items))
	for i, it := range items {
		rows[i] = Row{ID: strconv.Itoa(i), Cells: []string{it.Name}, Flag: it.Mark.span()}
	}
	d.list.SetRows(rows)
	open := func() tea.Cmd {
		if it, ok := d.current(); ok {
			return a.openDiff(p, it)
		}
		return nil
	}
	notText := func() bool { it, ok := d.current(); return !ok || it.Text == nil }
	notMany := func() bool { it, ok := d.current(); return !ok || it.Items == nil }
	d.acts = []Action{
		{Key: "Enter", Label: "Compare", Desc: "its two sides", Hint: "compare", Scope: ItemScope, Hidden: notText, Do: open},
		{Key: "Enter", Label: "Open", Desc: "the items that differ in it", Hint: "open", Scope: ItemScope, Hidden: notMany, Do: open},
	}
	p.levels = append(p.levels, d)
}

// diffViewer is the comparison of at, an's text, with bt, bn's (tdp
// note.8): on UNIFIED as it opens, the tab the user switches to kept while
// it is open only, SPLIT shown while there is room for it.
func (a *App) diffViewer(an, bn, at, bt string) tea.Cmd {
	split := false
	x, y := textLines(at), textLines(bt)
	ops := diffLines(x, y)
	v := &Viewer{Title: "Diff", Subject: an + " ↔ " + bn, Glyph: glyphCompare, Tabs: []string{"Unified", "Split"}, NoNumbers: true}
	v.tabOff = func(i int) bool { return i == 1 && a.w < splitMin }
	// layout lays the lines out for a terminal sw wide, as it opens and
	// whenever its size changes: SPLIT when it is chosen and there is room,
	// the line on top kept by the op it shows (lineOp: each line's op).
	var lineOp []int
	lastW, lastTab := -1, -1
	layout := func(sw int) {
		tab := 0
		if split && sw >= splitMin {
			tab = 1
		}
		w := popupWidth(sw, 0, false) - 4 // the viewer's text, numbers left out
		if w == lastW && tab == lastTab {
			return
		}
		keep := -1
		if v.top < len(lineOp) {
			keep = lineOp[v.top]
		}
		v.Tab = tab
		if tab == 1 {
			v.Lines, lineOp = diffSplit(ops, x, y, w)
		} else {
			v.Lines, lineOp = diffUnified(ops, x, y, w)
		}
		v.top, v.sub = 0, 0
		for i, o := range lineOp {
			if keep >= 0 && o >= keep {
				v.top = i
				break
			}
		}
		lastW, lastTab = w, tab
	}
	v.relayout = layout
	v.OnTab = func(tab int) tea.Cmd { split = tab == 1; layout(a.w); return nil }
	layout(a.w)
	return a.Viewer(v)
}

// checkCompares ends a compare mode whose anchor is no longer in its list,
// with a Warning toast, the lists gone into in it closed, and forgets the
// ones left.
func (a *App) checkCompares() tea.Cmd {
	var cmds []tea.Cmd
	kept := a.comparings[:0]
	for _, cm := range a.comparings {
		if cm.p.mode != cm.mode {
			continue
		}
		if _, ok := cm.l.rowByID(cm.l.Anchor); !ok {
			cm.p.LeaveMode()
			cm.l.Anchor = ""
			cmds = append(cmds, a.Toast(Warning, "Compare ended: "+cm.name+" is gone"))
			continue
		}
		kept = append(kept, cm)
	}
	a.comparings = kept
	return tea.Batch(cmds...)
}

func textLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

// diffOp is one line of a diff: ' ' the same on both sides, '-' only in the
// anchor's, '+' only in the other's; a and b are its line in each, -1 when
// it is not there.
type diffOp struct {
	op   byte
	a, b int
}

// diffLines is the shortest edit from x to y, line by line.
func diffLines(x, y []string) []diffOp {
	pre := 0
	for pre < len(x) && pre < len(y) && x[pre] == y[pre] {
		pre++
	}
	suf := 0
	for suf < len(x)-pre && suf < len(y)-pre && x[len(x)-1-suf] == y[len(y)-1-suf] {
		suf++
	}
	var ops []diffOp
	for i := range pre {
		ops = append(ops, diffOp{' ', i, i})
	}
	ops = append(ops, myers(x[pre:len(x)-suf], y[pre:len(y)-suf], pre, pre)...)
	for i := range suf {
		ops = append(ops, diffOp{' ', len(x) - suf + i, len(y) - suf + i})
	}
	return ops
}

// myers is the shortest edit script from a to b (Myers 1986), its lines
// counted on from ao and bo.
func myers(a, b []string, ao, bo int) []diffOp {
	n, m := len(a), len(b)
	if n == 0 && m == 0 {
		return nil
	}
	most := n + m
	off := most + 1
	v := make([]int, 2*most+3)
	var trace [][]int // trace[d]: v around the diagonals -d-1..d+1 before step d
	for d := 0; d <= most; d++ {
		trace = append(trace, slices.Clone(v[off-d-1:off+d+2]))
		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || k != d && v[off+k-1] < v[off+k+1] {
				x = v[off+k+1]
			} else {
				x = v[off+k-1] + 1
			}
			y := x - k
			for x < n && y < m && a[x] == b[y] {
				x++
				y++
			}
			v[off+k] = x
			if x >= n && y >= m {
				return backtrack(trace, n, m, ao, bo)
			}
		}
	}
	return nil
}

func backtrack(trace [][]int, x, y, ao, bo int) []diffOp {
	var rev []diffOp
	for d := len(trace) - 1; d >= 0; d-- {
		v := trace[d]
		at := func(k int) int { return v[k+d+1] }
		k := x - y
		pk := k - 1
		if k == -d || k != d && at(k-1) < at(k+1) {
			pk = k + 1
		}
		px := at(pk)
		py := px - pk
		for x > px && y > py {
			x, y = x-1, y-1
			rev = append(rev, diffOp{' ', ao + x, bo + y})
		}
		if d > 0 {
			if x == px {
				y--
				rev = append(rev, diffOp{'+', -1, bo + y})
			} else {
				x--
				rev = append(rev, diffOp{'-', ao + x, -1})
			}
		}
		x, y = px, py
	}
	slices.Reverse(rev)
	return rev
}

// hunks are the stretches of ops a compare shows: each change with 3 lines
// either side, the ones that meet joined (tdp note.8).
func hunks(ops []diffOp) [][2]int {
	var out [][2]int
	for i, o := range ops {
		if o.op == ' ' {
			continue
		}
		from, to := max(i-3, 0), min(i+4, len(ops))
		if n := len(out); n > 0 && from <= out[n-1][1] {
			out[n-1][1] = max(out[n-1][1], to)
			continue
		}
		out = append(out, [2]int{from, to})
	}
	return out
}

func numWidthOf(x, y []string) int { return len(itoa(max(len(x), len(y), 1))) }

// diffUnified is UNIFIED: both line numbers, then - in Red, + in Green or a
// space, the hunks w cells wide with an Overlay0 line between; at is the op
// each line shows (a line between hunks, the one before the next's first).
func diffUnified(ops []diffOp, x, y []string, w int) (out []string, at []int) {
	nw := numWidthOf(x, y)
	num := func(i int) string {
		if i < 0 {
			return Repeat(" ", nw)
		}
		return PadLeft(itoa(i+1), nw)
	}
	for hi, h := range hunks(ops) {
		if hi > 0 {
			out, at = append(out, Fg(Overlay0).Render(Repeat("─", w))), append(at, h[0]-1)
		}
		for j := h[0]; j < h[1]; j++ {
			o := ops[j]
			at = append(at, j)
			nums := Fg(Overlay0).Render(num(o.a)+" "+num(o.b)) + " "
			switch o.op {
			case '-':
				out = append(out, nums+Fg(Red).Render("-"+x[o.a]))
			case '+':
				out = append(out, nums+Fg(Green).Render("+"+y[o.b]))
			default:
				out = append(out, nums+" "+x[o.a])
			}
		}
	}
	return out, at
}

// diffSplit is SPLIT: the anchor's lines on the left, the other's on the
// right, each half with its numbers. A line too long folds within its half
// as a viewer's does (tdp note.2), and the two across from each other take
// as many rows as the longer, the shorter filled with blank rows, so they
// start on the same row (tdp note.8). A line with nothing across from it
// faces an empty half; at is the op each row shows, as diffUnified's.
func diffSplit(ops []diffOp, x, y []string, w int) (out []string, at []int) {
	nw := numWidthOf(x, y)
	half := max((w-2)/2, nw+2)
	// side is line i's rows in its half, its number on the first.
	side := func(i int, lines []string, st Style) []string {
		if i < 0 {
			return nil
		}
		var rows []string
		for r, part := range foldText(strings.TrimSuffix(lines[i], "\r"), half-nw-1) {
			lead := Repeat(" ", nw+1)
			if r == 0 {
				lead = Fg(Overlay0).Render(PadLeft(itoa(i+1), nw)) + " "
			}
			rows = append(rows, Pad(lead+st.Render(part), half))
		}
		return rows
	}
	pair := func(op int, l, r []string) {
		for k := range max(len(l), len(r)) {
			left, right := Repeat(" ", half), Repeat(" ", half)
			if k < len(l) {
				left = l[k]
			}
			if k < len(r) {
				right = r[k]
			}
			out, at = append(out, left+"  "+right), append(at, op)
		}
	}
	for hi, h := range hunks(ops) {
		if hi > 0 {
			out, at = append(out, Fg(Overlay0).Render(Repeat("─", w))), append(at, h[0]-1)
		}
		var dels, adds []int // ops
		flush := func() {
			for k := range max(len(dels), len(adds)) {
				l, r, op := -1, -1, 0
				if k < len(dels) {
					l, op = ops[dels[k]].a, dels[k]
				} else {
					op = adds[k]
				}
				if k < len(adds) {
					r = ops[adds[k]].b
				}
				pair(op, side(l, x, Fg(Red)), side(r, y, Fg(Green)))
			}
			dels, adds = nil, nil
		}
		for j := h[0]; j < h[1]; j++ {
			o := ops[j]
			switch o.op {
			case '-':
				if len(adds) > 0 {
					flush()
				}
				dels = append(dels, j)
			case '+':
				adds = append(adds, j)
			default:
				flush()
				pair(j, side(o.a, x, Style{}), side(o.b, y, Style{}))
			}
		}
		flush()
	}
	return out, at
}

// foldText is a line as a viewer draws it (tabs as spaces, a control
// character by name) cut into the rows it folds onto at w cells, as the
// viewer folds it (tdp note.2): one row at least.
func foldText(s string, w int) []string {
	s = readingLine(s, tabCells)
	chars := graphemes(stripSGR(s))
	cells, space := make([]int, len(chars)), make([]bool, len(chars))
	for k, c := range chars {
		cells[k], space[k] = Width(c), c == " "
	}
	starts := foldAt(cells, space, w)
	rows := make([]string, len(starts))
	if !strings.Contains(s, "\x1b") { // plain: by its characters
		starts = append(starts, len(chars))
		for r := range rows {
			rows[r] = strings.Join(chars[starts[r]:starts[r+1]], "")
		}
		return rows
	}
	from := 0 // in cells
	for r := range starts {
		to := Width(s)
		if r+1 < len(starts) {
			to = 0
			for _, c := range cells[:starts[r+1]] {
				to += c
			}
		}
		rows[r] = Cut(s, from, to)
		from = to
	}
	return rows
}
