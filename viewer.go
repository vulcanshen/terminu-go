package terminu

import (
	"slices"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	tea "github.com/charmbracelet/bubbletea"
)

// Viewer is a note to read (components/dialog/note): a YAML view, a log, a
// file. Its lines are numbered on the left and a long one folds onto the rows
// under it. It scrolls, may have keys of its own and a selection mode, and
// closes with Esc only. Keep the pointer: lines may keep arriving while
// Loading is true.
type Viewer struct {
	// Title is what it shows, in capitals ("YAML"); Subject is the thing
	// itself, as it is ("pod/nginx"): the title reads "YAML — pod/nginx".
	Title, Subject string
	Glyph          string
	Lines          []string
	Loading        bool
	// Lang is the language its lines are coloured in, as the file preview
	// colours a text file (components/dialog/note): a chroma lexer's name or
	// alias ("yaml", "json"), a file name to judge by ("deploy.yaml"), or a
	// content type ("application/json"). Unset or not known: plain text, in
	// Text (an App Log).
	Lang string
	// Tabs, when set, follow the title (components/layout/popup); h/l switch
	// them and OnTab fills Lines for the new one.
	Tabs  []string
	Tab   int
	OnTab func(tab int) tea.Cmd
	// Keys are the viewer's own hotkeys, first in its hint.
	Keys []ViewerKey
	// Select, when set, gives it a selection mode (Rules K11).
	Select *Selection

	// top is the line the first row in sight belongs to and sub that row
	// among the rows the line folds onto; visible the rows of text; textW
	// the cells of text a row holds, after the line numbers; numW the line
	// numbers' column, as wide as it has been since it opened.
	top, sub, visible, textW, numW int
	// colours caches the lines as drawn for the lines and Lang they are of.
	colours viewerColours
	// The selection mode (components/dialog/note): the first press of its
	// key brings the cursor out (inMode), the second starts the selection
	// there. The cursor and where the selection started are in characters
	// of the plain lines.
	inMode, selecting bool
	line, col         int
	fromL, fromC      int
	plain             [][]string
}

// ViewerKey is one of a viewer's own hotkeys.
type ViewerKey struct {
	Key, Desc string
	Do        func() tea.Cmd
}

// Selection is a viewer's selection mode: Key brings the cursor out at the
// start of the first line in sight, Key again starts the selection where it
// is, the cursor moves through the text (h|j|k|l, w|b|e, 0|$), CopyKey hands
// the selected text to Copy and leaves; Esc leaves.
type Selection struct {
	Key     string // "v"
	Name    string // the mode's name on the frame: "Visual"
	CopyKey string // "y" when unset
	Copy    func(text string) tea.Cmd
}

func (s *Selection) copyKey() string { return cmpOr(s.CopyKey, "y") }

// Viewer opens a viewer.
func (a *App) Viewer(v *Viewer) tea.Cmd {
	v.numW = 0
	return a.open(&viewerView{v: v})
}

// ScrollTo puts line i at the top as far as the lines go; in the selection
// mode the cursor goes to its start.
func (v *Viewer) ScrollTo(i int) {
	v.top, v.sub = i, 0
	if v.inMode {
		v.line, v.col = max(i, 0), 0
	}
}

// Top is the first line in sight.
func (v *Viewer) Top() int { return v.top }

type viewerView struct{ v *Viewer }

func (vv *viewerView) title() string {
	t := Upper(vv.v.Title)
	if vv.v.Subject != "" {
		t += " — " + vv.v.Subject
	}
	return t
}

func (vv *viewerView) hint() []KeyHint {
	v := vv.v
	switch {
	case v.selecting:
		return []KeyHint{{v.Select.copyKey(), "copy"}, {"Esc", "cancel"}}
	case v.inMode:
		return []KeyHint{{v.Select.Key, "select"}, {"Esc", "cancel"}}
	}
	var h []KeyHint
	for _, k := range v.Keys {
		h = append(h, KeyHint{k.Key, k.Desc})
	}
	if v.Select != nil {
		h = append(h, KeyHint{v.Select.Key, strings.ToLower(cmpOr(v.Select.Name, "select"))})
	}
	return append(h, KeyHint{"Esc", "close"})
}

func (vv *viewerView) measure(sw, sh int) (int, int) {
	v := vv.v
	w := popupWidth(sw, 0, false)
	v.readLines()
	v.textW = v.textWidth(w - 2)
	most := max(sh-2, 3)
	rows := 0 // the rows its lines fold onto, counted as far as the most
	for i := range v.Lines {
		if rows += len(v.wrap(i)); rows+4 >= most {
			break
		}
	}
	h := min(rows+4, most)
	v.visible = max(h-4, 1)
	return w, h
}

// numWidth is the line numbers' column (components/dialog/note): at least
// three digits, and it only grows. A viewer's lines are known when it opens,
// so the column keeps its width; lines that stream in widen it past 999; a
// tab with more lines widens it, and going back to one with fewer keeps it.
func (v *Viewer) numWidth() int {
	v.numW = max(v.numW, len(strconv.Itoa(len(v.Lines))), 3)
	return v.numW
}

// textWidth is how many cells of text a row holds in inner cells: one cell
// of padding either side, the line numbers and the space after them.
func (v *Viewer) textWidth(inner int) int { return max(inner-3-v.numWidth(), 1) }

// wrap is where line i's rows start, in cells: a line too long for a row
// folds onto the rows under it, a character never cut in half.
func (v *Viewer) wrap(i int) []int {
	starts := []int{0}
	used, cell := 0, 0
	for _, c := range v.plain[i] {
		cw := Width(c)
		if used+cw > v.textW && used > 0 {
			starts = append(starts, cell)
			used = 0
		}
		used += cw
		cell += cw
	}
	return starts
}

// maxTop is the last row that can be at the top, as a line and a row of it:
// from there the rows to the end fill the view. It scrolls a row at a time
// (components/dialog/note), so a line taller than the view is read whole.
func (v *Viewer) maxTop() (line, sub int) {
	rows := 0
	for t := len(v.Lines) - 1; t >= 0; t-- {
		n := len(v.wrap(t))
		if rows+n >= v.visible {
			return t, rows + n - v.visible
		}
		rows += n
	}
	return 0, 0
}

// clampTop keeps the first row in sight between the first row and maxTop.
func (v *Viewer) clampTop() {
	if v.top < 0 || len(v.Lines) == 0 {
		v.top, v.sub = 0, 0
		return
	}
	if t, s := v.maxTop(); v.top > t || v.top == t && v.sub > s {
		v.top, v.sub = t, s
	}
	v.sub = min(max(v.sub, 0), len(v.wrap(v.top))-1)
}

// scrollRows scrolls n rows, down when n is positive: a folded line is
// passed a row at a time, as less does (components/dialog/note).
func (v *Viewer) scrollRows(n int) {
	v.clampTop()
	for ; n > 0; n-- {
		if v.sub+1 < len(v.wrap(v.top)) {
			v.sub++
		} else if v.top+1 < len(v.Lines) {
			v.top, v.sub = v.top+1, 0
		}
	}
	for ; n < 0; n++ {
		if v.sub > 0 {
			v.sub--
		} else if v.top > 0 {
			v.top--
			v.sub = len(v.wrap(v.top)) - 1
		}
	}
	v.clampTop()
}

// follow scrolls as little as it takes to show line whole (the selection
// mode's cursor; components/dialog/note).
func (v *Viewer) follow(line int) {
	if line < v.top || line == v.top && v.sub > 0 {
		v.top, v.sub = line, 0
		return
	}
	t, rows := line, 0
	for t >= 0 {
		if rows += len(v.wrap(t)); rows > v.visible {
			break
		}
		t--
	}
	if top := max(v.top, min(t+1, line)); top != v.top {
		v.top, v.sub = top, 0
	}
}

func (vv *viewerView) isLoading() bool { return vv.v.Loading }

func (vv *viewerView) draw(w, h int, layer Color) []string {
	v := vv.v
	inner := w - 2
	v.visible = max(h-4, 1)
	v.readLines()
	v.textW = v.textWidth(inner)
	numW := v.numWidth()
	if v.inMode {
		v.line = min(max(v.line, 0), max(len(v.Lines)-1, 0))
		v.follow(v.line) // the view follows the cursor
	}
	v.clampTop()
	// Each line's first row has its number, right-aligned, a space after it;
	// the rows it folds onto leave the column blank. Numbers in Overlay0, the
	// cursor's line in Text (components/dialog/note).
	body := make([]string, 0, v.visible)
	last, whole := v.top-1, true // the line of the last row in sight, and whether all of it is
	for i := v.top; i < len(v.Lines) && len(body) < v.visible; i++ {
		styled := vv.lineView(i, layer)
		starts := v.wrap(i)
		first := 0
		if i == v.top {
			first = v.sub // the rows of the top line scrolled past
		}
		for r := first; r < len(starts); r++ {
			from := starts[r]
			if len(body) == v.visible {
				whole = false
				break
			}
			num := Repeat(" ", numW)
			if r == 0 {
				style := Fg(Overlay0)
				if v.inMode && i == v.line {
					style = Fg(Text)
				}
				num = style.Render(PadLeft(strconv.Itoa(i+1), numW))
			}
			to := from + v.textW // the last row: the cursor's cell on an empty line too
			if r+1 < len(starts) {
				to = starts[r+1]
			}
			body = append(body, " "+num+" "+Fit(Cut(styled, from, to), v.textW)+" ")
		}
		last = i
	}
	for len(body) < v.visible {
		body = append(body, Repeat(" ", inner))
	}
	// The scroll position, in lines, the numbers on the left: the lines the
	// first and the last row in sight belong to, N-M of T; in the selection
	// mode the cursor's line, N of M (components/dialog/note).
	scroll := ""
	if v.top > 0 || v.sub > 0 || last < len(v.Lines)-1 || !whole {
		if v.inMode {
			scroll = scrollText(true, v.line+1, 0, len(v.Lines))
		} else {
			scroll = scrollText(false, v.top+1, last+1, len(v.Lines))
		}
	}
	box := popupBox{color: layer, glyph: v.Glyph, title: vv.title(), hint: vv.hint(), scroll: scroll, loading: v.Loading}
	for i, t := range v.Tabs {
		box.tabs = append(box.tabs, Capsule{Text: Upper(t), Selected: i == v.Tab})
	}
	if v.inMode {
		// A mode marks its frame: the mode colour and its name (Rules K11).
		box.color, box.mode = Yellow, cmpOr(v.Select.Name, "Select")
	}
	return box.draw(w, body)
}

// viewerColours is the lines as drawn, for the lines and Lang they are of.
type viewerColours struct {
	lang       string
	lines, out []string
}

// coloured is the lines as drawn (components/dialog/note): in the colours
// Lang reads them in, the file preview's; with no Lang, or one not known,
// plain text in Text, where a line the app styled itself (a log's levels)
// keeps its styles.
func (v *Viewer) coloured() []string {
	c := &v.colours
	if c.lang == v.Lang && slices.Equal(c.lines, v.Lines) && len(c.out) == len(v.Lines) {
		return c.out
	}
	c.lang, c.lines = v.Lang, slices.Clone(v.Lines)
	c.out = make([]string, len(v.Lines))
	if lexer := lexerFor(v.Lang); lexer != nil {
		plain := make([]string, len(v.Lines))
		for i, l := range v.Lines {
			plain[i] = StripStyles(l)
		}
		for i, l := range colourise(lexer, plain) {
			if StripStyles(l) != plain[i] { // the lexer changed the text: not coloured
				l = Fg(Text).Render(plain[i])
			}
			c.out[i] = l
		}
		return c.out
	}
	for i, l := range v.Lines {
		c.out[i] = onText(l)
	}
	return c.out
}

// onText is l with what it leaves unstyled in Text: its start, and after
// every reset in it.
func onText(l string) string {
	fg := "\x1b[" + sgrRGB(true, Text.RGB) + "m"
	l = strings.ReplaceAll(l, "\x1b[0m", "\x1b[0m"+fg)
	l = strings.ReplaceAll(l, "\x1b[m", "\x1b[m"+fg)
	return fg + l + "\x1b[0m"
}

// lexerFor is the lexer a viewer's Lang names: a lexer's name or alias, a
// file name, or a content type; nil when there is none.
func lexerFor(lang string) chroma.Lexer {
	if lang == "" {
		return nil
	}
	if l := lexers.Get(lang); l != nil {
		return l
	}
	return lexers.MatchMimeType(strings.TrimSpace(strings.SplitN(lang, ";", 2)[0]))
}

// readLines makes the plain characters of each line, which the cursor
// moves through.
func (v *Viewer) readLines() {
	v.plain = v.plain[:0]
	for _, l := range v.Lines {
		v.plain = append(v.plain, graphemes(StripStyles(l)))
	}
}

// selection is where the selection starts and ends, the start first.
func (v *Viewer) selection() (pos, pos) {
	start, end := pos{v.fromL, v.fromC}, pos{v.line, v.col}
	if end.line < start.line || end.line == start.line && end.col < start.col {
		start, end = end, start
	}
	return start, end
}

// lineView is line i as drawn, in the selection mode with the cursor, a
// block of the layer colour, and the part selected in the mode's colour,
// Yellow, its syntax colours not shown (components/dialog/note,
// components/color).
func (vv *viewerView) lineView(i int, layer Color) string {
	v := vv.v
	line := v.coloured()[i]
	chars := v.plain[i]
	from, to := -1, -1 // the characters selected on this line
	if v.selecting {
		start, end := v.selection()
		if i >= start.line && i <= end.line {
			from, to = 0, len(chars)
			if i == start.line {
				from = start.col
			}
			if i == end.line {
				to = min(end.col+1, len(chars)) // the character under the cursor is in
			}
		}
	}
	cx := -1
	if v.inMode && i == v.line {
		cx = v.col
	}
	sel, cur := Style{Fg: Base, Bg: Yellow}, On(layer)
	var b strings.Builder
	cell, plainFrom := 0, -1 // plainFrom: where the run of untouched characters began, in cells
	flush := func() {
		if plainFrom >= 0 {
			b.WriteString(Cut(line, plainFrom, cell))
			plainFrom = -1
		}
	}
	for k, c := range chars {
		switch {
		case k == cx:
			flush()
			b.WriteString(cur.Render(c))
		case k >= from && k < to:
			flush()
			b.WriteString(sel.Render(c))
		default:
			if plainFrom < 0 {
				plainFrom = cell
			}
		}
		cell += Width(c)
	}
	flush()
	switch {
	case cx >= len(chars):
		b.WriteString(cur.Render(" ")) // on an empty line the cursor still shows
	case len(chars) == 0 && from >= 0:
		b.WriteString(sel.Render(" "))
	}
	return b.String()
}

// selected is the text from where the selection started to the cursor, the
// character under the cursor in.
func (v *Viewer) selected() string {
	start, end := v.selection()
	var out []string
	for i := start.line; i <= end.line && i < len(v.plain); i++ {
		chars := v.plain[i]
		from, to := 0, len(chars)
		if i == start.line {
			from = min(start.col, len(chars))
		}
		if i == end.line {
			to = min(end.col+1, len(chars))
		}
		out = append(out, strings.Join(chars[from:max(to, from)], ""))
	}
	return strings.Join(out, "\n")
}

func (v *Viewer) classAt(p pos) int {
	if p.line >= len(v.plain) || p.col >= len(v.plain[p.line]) {
		return 0
	}
	return charClass(v.plain[p.line][p.col])
}

func (v *Viewer) next(p pos) (pos, bool) {
	if p.col+1 < len(v.plain[p.line]) {
		return pos{p.line, p.col + 1}, true
	}
	if p.line+1 < len(v.plain) {
		return pos{p.line + 1, 0}, true
	}
	return p, false
}

func (v *Viewer) prev(p pos) (pos, bool) {
	if p.col > 0 {
		return pos{p.line, p.col - 1}, true
	}
	if p.line > 0 {
		return pos{p.line - 1, max(len(v.plain[p.line-1])-1, 0)}, true
	}
	return p, false
}

// move moves the cursor through the text as vim does, without wrapping at
// the ends (Rules K12): h|j|k|l, w|b|e, 0|$, u|d, gg|G. It reports whether
// key moved.
func (vv *viewerView) move(key string) bool {
	v := vv.v
	if len(v.plain) != len(v.Lines) {
		v.readLines()
	}
	if len(v.plain) == 0 {
		return isMoveKey(key)
	}
	half := max(v.visible/2, 1)
	lineLen := func() int { return len(v.plain[v.line]) }
	p := pos{v.line, v.col}
	switch key {
	case "h", "←":
		v.col = max(v.col-1, 0)
	case "l", "→":
		v.col = min(v.col+1, max(lineLen()-1, 0))
	case "j", "↓":
		v.line = min(v.line+1, len(v.plain)-1)
	case "k", "↑":
		v.line = max(v.line-1, 0)
	case "d", "PgDn":
		v.line = min(v.line+half, len(v.plain)-1)
	case "u", "PgUp":
		v.line = max(v.line-half, 0)
	case "gg":
		v.line, v.col = 0, 0
	case "G":
		v.line = len(v.plain) - 1
	case "0":
		v.col = 0
	case "$":
		v.col = max(lineLen()-1, 0)
	case "w":
		c, ok := v.classAt(p), true
		for ok && c != 0 && v.classAt(p) == c {
			p, ok = v.next(p)
		}
		for ok && v.classAt(p) == 0 {
			p, ok = v.next(p)
		}
		v.line, v.col = p.line, p.col
	case "b":
		p, ok := v.prev(p)
		for ok && v.classAt(p) == 0 {
			p, ok = v.prev(p)
		}
		for c := v.classAt(p); ok; {
			q, more := v.prev(p)
			if !more || v.classAt(q) != c || q.line != p.line {
				break
			}
			p = q
		}
		v.line, v.col = p.line, p.col
	case "e":
		p, ok := v.next(p)
		for ok && v.classAt(p) == 0 {
			p, ok = v.next(p)
		}
		for c := v.classAt(p); ok; {
			q, more := v.next(p)
			if !more || v.classAt(q) != c || q.line != p.line {
				break
			}
			p = q
		}
		v.line, v.col = p.line, p.col
	default:
		return false
	}
	v.col = min(v.col, max(lineLen()-1, 0))
	return true
}

func isMoveKey(key string) bool {
	switch key {
	case "h", "j", "k", "l", "←", "→", "↑", "↓", "w", "b", "e", "0", "$", "u", "d", "PgUp", "PgDn", "gg", "G":
		return true
	}
	return false
}

func (vv *viewerView) key(a *App, key string, msg tea.KeyMsg) (bool, tea.Cmd) {
	v := vv.v
	if v.inMode {
		// Other keys do nothing in the mode (Rules K11); h|l move the cursor,
		// not the tabs (Rules K12).
		switch key {
		case "?", "q", "Ctrl-C":
			return false, nil // the mode's key reference, the quit flow (Rules K11)
		case "Esc":
			v.inMode, v.selecting = false, false // at any step: the whole mode, no copy
		case v.Select.Key:
			if !v.selecting { // pressed again once selecting, it does nothing
				v.selecting = true
				v.fromL, v.fromC = v.line, v.col
			}
		case v.Select.copyKey():
			if v.selecting {
				v.inMode, v.selecting = false, false
				if v.Select.Copy != nil {
					return true, v.Select.Copy(v.selected())
				}
			}
		default:
			if vv.move(key) {
				v.follow(v.line)
			}
		}
		return true, nil
	}
	// No cursor: the keys scroll, h|l switch the tabs (components/dialog/note).
	half := max(v.visible/2, 1)
	switch key {
	case "j", "↓":
		v.scrollRows(1)
	case "k", "↑":
		v.scrollRows(-1)
	case "d", "PgDn":
		v.scrollRows(half)
	case "u", "PgUp":
		v.scrollRows(-half)
	case "gg":
		v.top, v.sub = 0, 0
	case "G":
		v.top, v.sub = len(v.Lines), 0
	case "h", "←", "l", "→":
		if n := len(v.Tabs); n > 0 {
			dir := 1
			if key == "h" || key == "←" {
				dir = -1
			}
			v.Tab = (v.Tab + dir + n) % n
			v.top, v.sub = 0, 0
			return true, run2tab(v.OnTab, v.Tab)
		}
	case "Enter", "Space":
		// A note closes with Esc only (components/dialog/note).
	default:
		if v.Select != nil && key == v.Select.Key {
			// The cursor comes out at the start of the first line in sight;
			// nothing is selected yet.
			v.inMode = true
			v.line, v.col = min(max(v.top, 0), max(len(v.Lines)-1, 0)), 0
			return true, nil
		}
		for _, k := range v.Keys {
			if k.Key == key {
				if k.Do == nil {
					return true, nil
				}
				return true, k.Do()
			}
		}
		return false, nil
	}
	v.clampTop()
	return true, nil
}

func run2tab(f func(int) tea.Cmd, i int) tea.Cmd {
	if f == nil {
		return nil
	}
	return f(i)
}

func (vv *viewerView) surface() string { return vv.title() }

func (vv *viewerView) keyRef() []refSection {
	v := vv.v
	move := []KeyRow{
		{Key: "h|j|k|l", Desc: "left, down, up, right"},
		{Key: "w|b|e", Desc: "next word, word before, end of the word"},
		{Key: "0|$", Desc: "start, end of the line"},
		{Key: "u|d", Desc: "half a page up, down"},
		{Key: "gg|G", Desc: "first, last line"},
	}
	if v.inMode {
		first := KeyRow{Key: v.Select.Key, Desc: "start selecting at the cursor"}
		if v.selecting {
			first = KeyRow{Key: v.Select.copyKey(), Desc: "copy the selection and leave"}
		}
		return []refSection{
			{rows: []KeyRow{first, {Key: "Esc", Desc: "leave " + cmpOr(v.Select.Name, "Select")}}},
			{title: "move", rows: move},
		}
	}
	var rows []KeyRow
	for _, k := range v.Keys {
		rows = append(rows, KeyRow{Key: k.Key, Desc: k.Desc})
	}
	if v.Select != nil {
		rows = append(rows, KeyRow{Key: v.Select.Key, Desc: "select text (" + cmpOr(v.Select.Name, "Select") + ")"})
	}
	rows = append(rows, KeyRow{Key: "Esc", Desc: "close"})
	move = []KeyRow{
		{Key: "j|k", Desc: "scroll down, up"},
		{Key: "u|d", Desc: "half a page up, down"},
		{Key: "gg|G", Desc: "to the top, the bottom"},
	}
	if len(v.Tabs) > 0 {
		move = append([]KeyRow{{Key: "h|l", Desc: "the tab before, after"}}, move...)
	}
	return []refSection{{rows: rows}, {title: "move", rows: move}}
}

func (vv *viewerView) typing() bool { return false }
