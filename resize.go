package terminu

import (
	"cmp"
	"slices"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
)

// Lines is where the user moved the lines between a screen's panels (tdp
// screen.9), for the app to keep in its state (tdp E3), by the screen's
// Name. The lib changes it as the user moves a line or presses =, and then
// calls App.OnLines; nothing is written before the user moves a line. The
// app reads it back into App.Lines before Run.
type Lines map[string]ScreenLines

// ScreenLines is one screen's lines: the upright ones by the terminal's
// width, the level ones by its height, each used again when the terminal is
// that wide or that high. A line is known by where the screen's Layout has
// it by default, cells from the panel area's left or top ("52"; "52#2" for
// the second line at 52, further down or right), and maps to where it is
// now. A line back where Layout has it is not kept.
type ScreenLines struct {
	Widths  map[int]map[string]int `json:"widths,omitempty" yaml:"widths,omitempty"`
	Heights map[int]map[string]int `json:"heights,omitempty" yaml:"heights,omitempty"`
}

// The least size of a panel whose app gives none (Panel.MinWidth,
// MinHeight): its frame and a row in it.
const leastPanelW, leastPanelH = 10, 3

// The resize keys move the focused panel's lines (tdp screen.9): wider and
// narrower by 2 columns, taller and shorter by 1 row; = puts the screen's
// lines back as Layout has them.
func isResizeKey(key string) bool {
	return key == ">" || key == "<" || key == "+" || key == "-" || key == "="
}

// The sides of a place.
const (
	sideLeft = iota
	sideRight
	sideTop
	sideBottom
)

// split is a line between panels (tdp screen.9): an edge panels share, as
// far as they touch along it. Which panels touch it is the default
// layout's; where it is may be the user's.
type split struct {
	upright bool   // between panels side by side: it moves left and right
	id      string // its key in ScreenLines
	def     int    // where the default layout has it
	pos     int    // where it is now
	before  []int  // the places left of it, or above it
	after   []int  // the places right of it, or below it
}

// screenLayout is a screen's panels in a panel area of w × h: the app's
// Layout, its lines where the user left them.
type screenLayout struct {
	def, rects []Rect // the app's Layout and the places now; a place not drawn is empty
	shown      []bool
	lines      []*split
	edges      [][4]*split // each place's lines, by side; nil on the area's edge
	// stale says the kept upright (0) or level (1) lines did not fit this
	// terminal: the default ones are used, and what is kept for this size
	// goes (dropStale; a line moved before that starts it again).
	stale [2]bool
}

// defaultRects is the screen's Layout for w × h; with none, its shown places
// side by side in equal widths (tdp L5).
func (s *Screen) defaultRects(w, h int) []Rect {
	if s.Layout != nil {
		return s.Layout(w, h)
	}
	var rects []Rect
	shown := 0
	for i := range s.Panels {
		if !s.hidden[i] {
			shown++
		}
	}
	x, k := 0, 0
	for i := range s.Panels {
		if s.hidden[i] {
			rects = append(rects, Rect{})
			continue
		}
		pw := w / max(shown, 1)
		if k++; k == shown {
			pw = w - x
		}
		rects = append(rects, Rect{X: x, Y: 0, W: pw, H: h})
		x += pw
	}
	return rects
}

// findLines finds the lines between the shown places of def in w × h.
func findLines(def []Rect, shown []bool, w, h int) ([]*split, [][4]*split) {
	edges := make([][4]*split, len(def))
	var lines []*split
	type seg struct {
		from, to, place int
		after           bool
	}
	for _, upright := range []bool{true, false} {
		at := map[int][]seg{}
		size := w
		if !upright {
			size = h
		}
		for i, r := range def {
			if !shown[i] {
				continue
			}
			lo, hi, from, to := r.X, r.X+r.W, r.Y, r.Y+r.H
			if !upright {
				lo, hi, from, to = r.Y, r.Y+r.H, r.X, r.X+r.W
			}
			if hi > 0 && hi < size {
				at[hi] = append(at[hi], seg{from, to, i, false})
			}
			if lo > 0 && lo < size {
				at[lo] = append(at[lo], seg{from, to, i, true})
			}
		}
		places := make([]int, 0, len(at))
		for p := range at {
			places = append(places, p)
		}
		sort.Ints(places)
		for _, p := range places {
			segs := at[p]
			sort.SliceStable(segs, func(a, b int) bool { return segs[a].from < segs[b].from })
			n := 0
			for k := 0; k < len(segs); {
				l := &split{upright: upright, def: p, pos: p}
				end, j := segs[k].from, k
				for ; j < len(segs) && segs[j].from <= end; j++ { // touching along it: the same line
					end = max(end, segs[j].to)
					if segs[j].after {
						l.after = append(l.after, segs[j].place)
					} else {
						l.before = append(l.before, segs[j].place)
					}
				}
				k = j
				if len(l.before) == 0 || len(l.after) == 0 {
					continue // an edge with no panel across it
				}
				if n++; n == 1 {
					l.id = itoa(p)
				} else {
					l.id = itoa(p) + "#" + itoa(n)
				}
				lines = append(lines, l)
				lo, hi := sideLeft, sideRight
				if !upright {
					lo, hi = sideTop, sideBottom
				}
				for _, i := range l.before {
					edges[i][hi] = l
				}
				for _, i := range l.after {
					edges[i][lo] = l
				}
			}
		}
	}
	return lines, edges
}

// place puts each shown place between its lines.
func (lay *screenLayout) place() {
	lay.rects = make([]Rect, len(lay.def))
	for i, r := range lay.def {
		if !lay.shown[i] {
			continue
		}
		x0, x1, y0, y1 := r.X, r.X+r.W, r.Y, r.Y+r.H
		e := lay.edges[i]
		if e[sideLeft] != nil {
			x0 = e[sideLeft].pos
		}
		if e[sideRight] != nil {
			x1 = e[sideRight].pos
		}
		if e[sideTop] != nil {
			y0 = e[sideTop].pos
		}
		if e[sideBottom] != nil {
			y1 = e[sideBottom].pos
		}
		lay.rects[i] = Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
	}
}

// least is p's least size (tdp screen.9: the app's).
func (p *Panel) least() (int, int) {
	return cmp.Or(p.MinWidth, leastPanelW), cmp.Or(p.MinHeight, leastPanelH)
}

// leastAt is the least size of place i in one way: the panel's, or less
// when the default layout already has it smaller.
func (lay *screenLayout) leastAt(s *Screen, i int, upright bool) int {
	lw, lh := s.Panels[i].least()
	if upright {
		return min(lw, lay.def[i].W)
	}
	return min(lh, lay.def[i].H)
}

// fits reports whether every shown place is at least its least size in one
// way.
func (lay *screenLayout) fits(s *Screen, upright bool) bool {
	for i, r := range lay.rects {
		if !lay.shown[i] {
			continue
		}
		size := r.H
		if upright {
			size = r.W
		}
		if size < lay.leastAt(s, i, upright) {
			return false
		}
	}
	return true
}

// layoutOf lays the screen's panels out in w × h, its lines where the user
// left them for this terminal's width and height (tdp screen.9), narrow or
// not.
func (a *App) layoutOf(s *Screen, w, h int) *screenLayout {
	def := s.defaultRects(w, h)
	lay := &screenLayout{def: def, shown: make([]bool, len(def))}
	for i, r := range def {
		lay.shown[i] = i < len(s.Panels) && !s.hidden[i] && r.W > 0 && r.H > 0
	}
	lay.lines, lay.edges = findLines(def, lay.shown, w, h)
	kept := a.Lines[s.Name]
	for k, upright := range []bool{true, false} {
		at := kept.Heights[a.h]
		if upright {
			at = kept.Widths[a.w]
		}
		if len(at) == 0 {
			continue
		}
		for _, l := range lay.lines {
			if pos, ok := at[l.id]; ok && l.upright == upright {
				l.pos = pos
			}
		}
		lay.place()
		// Kept lines that no longer fit (the app's Layout or least sizes
		// changed): the default ones, no toast, no copy, no WARN, E3's one
		// exception (tdp screen.9); dropStale takes them away.
		if !lay.fits(s, upright) {
			lay.stale[k] = true
			for _, l := range lay.lines {
				if l.upright == upright {
					l.pos = l.def
				}
			}
		}
	}
	lay.place()
	return lay
}

// areaRows are the rows above the panel area and its height (base).
func (a *App) areaRows() (top, h int) {
	switch {
	case len(a.screens) > 1:
		top = 1
		if a.h > 3 {
			top = 2
		}
	case a.Status != nil:
		top = 1
	}
	return top, a.h - top - 1
}

// screenLayoutNow is the current screen laid out now, narrow or not.
func (a *App) screenLayoutNow(s *Screen) *screenLayout {
	_, h := a.areaRows()
	return a.layoutOf(s, a.w, h)
}

// panelRect is where p is on the screen now, in screen rows: ok false when
// it is not drawn.
func (a *App) panelRect(p *Panel) (Rect, bool) {
	s := a.Screen()
	if s == nil || a.Canvas != nil {
		return Rect{}, false
	}
	i := slices.Index(s.Panels, p)
	if i < 0 {
		return Rect{}, false
	}
	top, h := a.areaRows()
	if a.narrow(s) {
		if p != s.Focused() {
			return Rect{}, false
		}
		return Rect{Y: top, W: a.w, H: h}, true
	}
	lay := a.layoutOf(s, a.w, h)
	if i >= len(lay.rects) || !lay.shown[i] {
		return Rect{}, false
	}
	r := lay.rects[i]
	r.Y += top
	return r, true
}

// resizeMove is what a resize key does on place i: the line it moves, by
// how much, and the side of the panel that line is on; with no line to move,
// what the toast says and on which side.
type resizeMove struct {
	line  *split
	by    int
	side  int
	why   string
	stale [2]bool // the layout's (screenLayout.stale)
}

// planResize is what key does on the focused panel of s (tdp screen.9):
// the width moves the line on its right, the height the one under it; on
// the area's right or bottom edge the other one, so > always widens it and
// + always makes it taller. A move that would take a panel the line moves
// under its least size goes as far as it can, less than a whole step; at
// its least the toast says which panel stops it, the focused one first.
func (a *App) planResize(s *Screen, key string) resizeMove {
	i := s.at()
	upright := key == ">" || key == "<"
	grow := key == ">" || key == "+"
	way, near, far, step := "height", sideBottom, sideTop, 1
	if upright {
		way, near, far, step = "width", sideRight, sideLeft, 2
	}
	full := resizeMove{side: near, why: s.toastName(i) + " is already full " + way}
	if a.narrow(s) { // one panel drawn, across the whole area (tdp screen.5, screen.6)
		return full
	}
	lay := a.screenLayoutNow(s)
	if i >= len(lay.edges) || !lay.shown[i] {
		return full
	}
	m := resizeMove{side: near, line: lay.edges[i][near], stale: lay.stale}
	if m.line == nil {
		m.side, m.line = far, lay.edges[i][far]
	}
	if m.line == nil {
		return full
	}
	m.by = step
	if !grow {
		m.by = -step
	}
	if m.side == far {
		m.by = -m.by
	}
	shrink := m.line.after
	if m.by < 0 {
		shrink = m.line.before
	}
	var stops []int
	for _, j := range shrink {
		r := lay.rects[j]
		size := r.H
		if upright {
			size = r.W
		}
		room := max(size-lay.leastAt(s, j, upright), 0)
		if room < abs(m.by) {
			m.by = min(room, abs(m.by)) * sign(m.by)
		}
		if room == 0 {
			stops = append(stops, j)
		}
	}
	if m.by == 0 {
		stop := stops[0]
		if slices.Contains(stops, i) {
			stop = i // the focused panel first
		}
		m.line, m.why = nil, s.toastName(stop)+" is at its minimum "+way
	}
	return m
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}

// toastName is how a toast names place i: its [N], or a sign's label as its frame
// has it (tdp screen.9, panel.17).
func (s *Screen) toastName(i int) string {
	if !s.hasNum(i) {
		return s.Panels[i].labelText()
	}
	return "[" + itoa(s.num(i)) + "]"
}

// resize does key on the focused panel (tdp screen.9): it moves a line, or
// says on that panel's edge why none can move. A line moved does not go in
// the App Log, as a filter does not: only the toast does, as every toast
// (tdp screen.9, note.7).
func (a *App) resize(key string) tea.Cmd {
	s := a.Screen()
	if s == nil || s.Focused() == nil {
		return nil
	}
	m := a.planResize(s, key)
	if m.line == nil {
		return a.edgeToast(s.Focused(), m.side, m.why)
	}
	m.line.pos += m.by
	return a.keepLine(s, m.line, m.stale)
}

// keepLine keeps where line l is now for the screen, at the terminal's width
// or height, and gives the app its lines to write (tdp screen.9, E3).
func (a *App) keepLine(s *Screen, l *split, stale [2]bool) tea.Cmd {
	if a.Lines == nil {
		a.Lines = Lines{}
	}
	sl := a.Lines[s.Name]
	by, size, k := &sl.Heights, a.h, 1
	if l.upright {
		by, size, k = &sl.Widths, a.w, 0
	}
	if *by == nil {
		*by = map[int]map[string]int{}
	}
	at := (*by)[size]
	if at == nil || stale[k] {
		at = map[string]int{}
	}
	if l.pos == l.def {
		delete(at, l.id)
	} else {
		at[l.id] = l.pos
	}
	if len(at) == 0 {
		delete(*by, size)
	} else {
		(*by)[size] = at
	}
	a.setScreenLines(s, sl)
	return a.onLines()
}

func (a *App) setScreenLines(s *Screen, sl ScreenLines) {
	if len(sl.Widths) == 0 && len(sl.Heights) == 0 {
		delete(a.Lines, s.Name)
		return
	}
	a.Lines[s.Name] = sl
}

func (a *App) onLines() tea.Cmd {
	if a.OnLines == nil {
		return nil
	}
	return a.OnLines(a.Lines)
}

// linesMoved reports whether a line of the screen is not where Layout has it, at
// this terminal's size; lines tells whether it has any.
func (a *App) linesMoved(s *Screen) (moved, lines bool) {
	lay := a.screenLayoutNow(s)
	for _, l := range lay.lines {
		if l.pos != l.def {
			return true, true
		}
	}
	return false, len(lay.lines) > 0
}

// dropStale takes away the current screen's kept lines that no longer fit
// at this width or height, after every message, and gives the app its lines
// to write (tdp screen.9): nothing is said.
func (a *App) dropStale() tea.Cmd {
	s := a.Screen()
	if s == nil || a.Canvas != nil {
		return nil
	}
	lay := a.screenLayoutNow(s)
	if !lay.stale[0] && !lay.stale[1] {
		return nil
	}
	sl := a.Lines[s.Name]
	if lay.stale[0] {
		delete(sl.Widths, a.w)
	}
	if lay.stale[1] {
		delete(sl.Heights, a.h)
	}
	a.setScreenLines(s, sl)
	return a.onLines()
}

// defaultLayout puts every line of the screen back where Layout has it and
// forgets where they were at this width and this height (tdp screen.9).
func (a *App) defaultLayout() tea.Cmd {
	s := a.Screen()
	if s == nil {
		return nil
	}
	if moved, _ := a.linesMoved(s); !moved {
		return a.Toast(Info, "The layout is already the default")
	}
	sl := a.Lines[s.Name]
	delete(sl.Widths, a.w)
	delete(sl.Heights, a.h)
	a.setScreenLines(s, sl)
	return a.onLines()
}

// resizeActions are the Space menu rows of the resize keys, a panel
// operation of every panel (tdp screen.9): not there on a screen with no
// line, disabled when the key would only say why not.
func (a *App) resizeActions(p *Panel) []Action {
	s := a.Screen()
	if s == nil || p.Sign != 0 || !slices.Contains(s.Panels, p) {
		return nil
	}
	none := func() bool { _, lines := a.linesMoved(s); return !lines }
	row := func(key, label, desc string) Action {
		return Action{Key: key, Label: label, Desc: desc, Scope: PanelScope, Hidden: none,
			Disabled: func() bool { return a.planResize(s, key).line == nil },
			Do:       func() tea.Cmd { return a.resize(key) }}
	}
	return []Action{
		row(">", "Wider", "2 columns, kept for this width"),
		row("<", "Narrower", "2 columns, kept for this width"),
		row("+", "Taller", "1 row, kept for this height"),
		row("-", "Shorter", "1 row, kept for this height"),
	}
}

// defaultLayoutAction is [=] Default layout, a global operation: not there
// on a screen with no line, disabled while every line is where Layout has
// it (tdp screen.9).
func (a *App) defaultLayoutAction() (Action, bool) {
	s := a.Screen()
	if s == nil || a.Canvas != nil {
		return Action{}, false
	}
	return Action{Key: "=", Label: "Default layout", Desc: "this screen's lines back", Scope: GlobalScope,
		Hidden:   func() bool { _, lines := a.linesMoved(s); return !lines },
		Disabled: func() bool { moved, _ := a.linesMoved(s); return !moved },
		Do:       a.defaultLayout}, true
}
