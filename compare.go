package terminu

import (
	"slices"
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
	// Can reports a row that can be compared: text, or shown as text. A row
	// it says no to has no Compare action, and in the mode the compare keys
	// are off on it (tdp M2, M6). Nil: every row.
	Can func(Row) bool
	// Name is the row as the title names it; its first cell when nil.
	Name func(Row) string
	// Text is the row's text, compared line by line.
	Text func(Row) (string, error)
	// Split is which the compare shows, UNIFIED or SPLIT: the lib reads it
	// and sets it as the user switches, for the app to keep with its state
	// (tdp E3). Nil: kept while the app runs.
	Split *bool
}

func (c Compare) can(r Row) bool { return c.Can == nil || c.Can(r) }

func (c Compare) name(r Row) string {
	if c.Name != nil {
		return c.Name(r)
	}
	if len(r.Cells) > 0 {
		return r.Cells[0]
	}
	return r.ID
}

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
// compares the row under the cursor with the anchor, off on the anchor and
// on a row c cannot compare (tdp M6); the same text on both sides is an
// Info toast, not a popup. Esc leaves it, the anchor gone; Tab and the
// digits wait; an anchor no longer in the list ends it with a Warning toast.
// The rows need their IDs.
func (a *App) CompareAction(p *Panel, l *List, c Compare) Action {
	if c.Split == nil {
		c.Split = new(bool)
	}
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
	move := func(key string) tea.Cmd { _, cmd := l.Key(key, tea.KeyMsg{}); return cmd }
	off := func() bool { r, ok := l.Current(); return !ok || r.ID == l.Anchor || !c.can(r) }
	cm.mode = &Mode{Name: "Compare", Exit: "leave", Pause: true, Keys: []ModeKey{
		{Key: "j|k", Keys: []string{"j", "k", "↓", "↑"}, Desc: "down, up", NoFooter: true, Do: move},
		{Key: "u|d", Keys: []string{"u", "d"}, Desc: "half a page up, down", NoFooter: true, Do: move},
		{Key: "gg|G", Keys: []string{"gg", "G"}, Desc: "to the top, the bottom", NoFooter: true, Do: move},
		{Key: c.Key + "|Enter", Keys: []string{c.Key, "Enter"}, Desc: "compare", Disabled: off,
			Item: true, Label: "Compare", MenuDesc: "with " + cm.name,
			Do: func(string) tea.Cmd { return a.compareWith(cm) }},
	}, OnExit: func() tea.Cmd { l.Anchor = ""; return nil }}
	p.EnterMode(cm.mode)
	a.comparings = append(a.comparings, cm)
}

// compareWith opens the compare of the anchor with the row under the cursor.
func (a *App) compareWith(cm *comparing) tea.Cmd {
	l, c := cm.l, cm.c
	other, ok := l.Current()
	anchor, found := l.rowByID(l.Anchor)
	if !ok || !found || other.ID == anchor.ID || !c.can(other) {
		return nil
	}
	an, bn := c.name(anchor), c.name(other)
	at, err := c.Text(anchor)
	if err != nil {
		return a.ErrorAbout("Compare", an, err.Error())
	}
	bt, err := c.Text(other)
	if err != nil {
		return a.ErrorAbout("Compare", bn, err.Error())
	}
	if at == bt {
		return a.Toast(Info, "No differences between "+an+" and "+bn)
	}
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
		if *c.Split && sw >= splitMin {
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
		v.Tab, v.NoWrap = tab, tab == 1
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
	v.OnTab = func(tab int) tea.Cmd { *c.Split = tab == 1; layout(a.w); return nil }
	layout(a.w)
	return a.Viewer(v)
}

// checkCompares ends a compare mode whose anchor is no longer in its list,
// with a Warning toast, and forgets the ones left.
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
// right, each half with its numbers, a line too long cut with "…"; a line
// with nothing across from it faces an empty half; at is as diffUnified's.
func diffSplit(ops []diffOp, x, y []string, w int) (out []string, at []int) {
	nw := numWidthOf(x, y)
	half := max((w-2)/2, nw+2)
	side := func(i int, lines []string, st Style) string {
		if i < 0 {
			return Repeat(" ", half)
		}
		return Pad(Fg(Overlay0).Render(PadLeft(itoa(i+1), nw))+" "+st.Render(Truncate(lines[i], half-nw-1)), half)
	}
	for hi, h := range hunks(ops) {
		if hi > 0 {
			out, at = append(out, Fg(Overlay0).Render(Repeat("─", w))), append(at, h[0]-1)
		}
		var dels, adds []int // ops
		flush := func() {
			for k := range max(len(dels), len(adds)) {
				l, r := -1, -1
				if k < len(dels) {
					l = ops[dels[k]].a
					at = append(at, dels[k])
				} else {
					at = append(at, adds[k])
				}
				if k < len(adds) {
					r = ops[adds[k]].b
				}
				out = append(out, side(l, x, Fg(Red))+"  "+side(r, y, Fg(Green)))
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
				at = append(at, j)
				out = append(out, side(o.a, x, Style{})+"  "+side(o.b, y, Style{}))
			}
		}
		flush()
	}
	return out, at
}
