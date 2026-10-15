package terminu

import (
	"strconv"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rivo/uniseg"
)

// Textarea is multi-line text (components/input/textarea): it opens writing,
// with the value there and the cursor at its end; Esc moves to the moving
// state, where Enter saves.
type Textarea struct {
	Title, Glyph string
	// Subject is the one thing it acts on, kept as it is after ": " in the
	// title: "EDIT: filu" (components/layout/popup).
	Subject string
	// Name is the text's name in its errors when the title is not ("Notes"
	// on an "Edit"); left unset, the title names it (components/input/README).
	Name string
	Old  string
	// Done gets the new text. It is not called when it did not change.
	Done func(string) tea.Cmd
}

func (t Textarea) typed() bool { return true }

// Textarea opens a textarea.
func (a *App) Textarea(t Textarea) tea.Cmd { return t.open(a, nil) }

func (t Textarea) open(a *App, written func()) tea.Cmd {
	tv := &textareaView{t: t, writing: true, written: written, tabW: a.inputTabWidth()}
	for _, l := range splitLines(t.Old) {
		tv.lines = append(tv.lines, graphemes(l))
	}
	tv.line = len(tv.lines) - 1
	tv.col = len(tv.lines[tv.line])
	return a.open(tv)
}

// splitLines cuts text into its lines: "\n", "\r\n" and a lone "\r" are
// each a line break (components/input/textarea: many terminals paste a line
// break as "\r").
func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.Split(strings.ReplaceAll(s, "\r", "\n"), "\n")
}

// graphemes is a line's characters. A pasted control character stays, one
// character on its own, drawn as its red name (components/input/textarea, as
// the one-line inputs).
func graphemes(s string) []string {
	var out []string
	g := uniseg.NewGraphemes(s)
	for g.Next() {
		out = append(out, g.Str())
	}
	return out
}

type textareaView struct {
	t         Textarea
	lines     [][]string
	line, col int
	writing   bool
	top       int // the first visual row shown
	visible   int
	textW     int
	written   func()
	tabW      int    // input_tab_width: the spaces Tab puts in, the cells a pasted tab takes
	err       string // the error row: the value holds a control character
}

// cellW is how wide a character is drawn: a pasted tab takes input_tab_width
// cells, not lined up to a tab stop; another control character its name
// (components/input/textarea).
func (tv *textareaView) cellW(c string) int {
	if c == "\t" {
		return tv.tabW
	}
	return cellWidth(c)
}

// hasControl reports whether the text holds a control character other than
// a line break or a tab: it is not saved (components/input/textarea).
func (tv *textareaView) hasControl() bool {
	for _, l := range tv.lines {
		for _, c := range l {
			if _, ok := ctrlName(c); ok && c != "\t" {
				return true
			}
		}
	}
	return false
}

func (tv *textareaView) value() string {
	parts := make([]string, len(tv.lines))
	for i, l := range tv.lines {
		parts[i] = strings.Join(l, "")
	}
	return strings.Join(parts, "\n")
}

// vrow is one row on screen: a piece of a line (a line too long folds onto
// the rows under it; components/input/textarea).
type vrow struct{ line, start, end int }

func (tv *textareaView) rows(w int) []vrow {
	var out []vrow
	for li, l := range tv.lines {
		start, used := 0, 0
		for i, c := range l {
			cw := tv.cellW(c)
			if used+cw > w && i > start {
				out = append(out, vrow{li, start, i})
				start, used = i, 0
			}
			used += cw
		}
		// The cursor at the end of a full row goes on a row of its own.
		if li == tv.line && tv.col == len(l) && used >= w && len(l) > start {
			out = append(out, vrow{li, start, len(l)})
			start = len(l)
		}
		out = append(out, vrow{li, start, len(l)})
	}
	return out
}

func (tv *textareaView) cursorRow(rows []vrow) int {
	for i, r := range rows {
		if r.line == tv.line && tv.col >= r.start && (tv.col < r.end || tv.col == r.end && (i+1 == len(rows) || rows[i+1].line != r.line)) {
			return i
		}
	}
	return 0
}

func (tv *textareaView) hint() []KeyHint {
	if tv.writing {
		return []KeyHint{{"Esc", "move"}}
	}
	return []KeyHint{{"Enter", "save"}, {"i", "write"}, {"Esc", "cancel"}}
}

// numWidth is the line numbers' column: at least two digits, so it widens
// only at line 100, 1000… (components/input/textarea).
func (tv *textareaView) numWidth() int { return max(len(strconv.Itoa(len(tv.lines))), 2) }

func (tv *textareaView) measure(sw, sh int) (int, int) {
	w := popupWidth(sw, 0, false)
	tv.textW = max(w-4-tv.numWidth()-1, 1)
	// As tall as it can be; the text scrolls in it (components/input/textarea).
	return w, max(sh-2, 3)
}

func (tv *textareaView) draw(w, h int, layer Color) []string {
	inner := w - 2
	numW := tv.numWidth()
	tv.textW = max(inner-2-numW-1, 1)
	rows := tv.rows(tv.textW)
	// The text, a blank row and the error row, which stays put (layout/popup).
	n := max(h-6, 1)
	tv.visible = n
	cur := tv.cursorRow(rows)
	tv.top = scrollWindow(tv.top, cur, n, len(rows))
	body := make([]string, 0, n)
	// The value is Lavender; the cursor is a Lavender block while writing and
	// a block of the layer colour while moving (components/input/textarea).
	style, cursor := Fg(Lavender), Style{Fg: Base, Bg: Lavender}
	if !tv.writing {
		cursor = On(layer)
	}
	for i := tv.top; i < tv.top+n; i++ {
		if i >= len(rows) {
			body = append(body, Repeat(" ", inner))
			continue
		}
		r := rows[i]
		num := Repeat(" ", numW)
		if i == 0 || rows[i-1].line != r.line {
			numStyle := Fg(Overlay0)
			if r.line == tv.line {
				numStyle = Fg(Text)
			}
			num = numStyle.Render(PadLeft(strconv.Itoa(r.line+1), numW))
		}
		var b strings.Builder
		l := tv.lines[r.line]
		for k := r.start; k < r.end; k++ {
			c := l[k]
			switch {
			case c == "\t":
				text := Repeat(" ", tv.tabW)
				if i == cur && k == tv.col {
					b.WriteString(cursor.Render(text))
				} else {
					b.WriteString(style.Render(text))
				}
			case i == cur && k == tv.col:
				b.WriteString(cursorOn(c, cursor))
			default:
				b.WriteString(cellText(c, style))
			}
		}
		if i == cur && tv.col == r.end && tv.col == len(l) {
			b.WriteString(cursor.Render(" "))
		}
		body = append(body, " "+num+" "+Fit(b.String(), tv.textW)+" ")
	}
	body = append(body, Repeat(" ", inner), padRow(Fg(Red).Render(Truncate(tv.err, inner-2)), inner))
	scroll := ""
	if len(rows) > n {
		scroll = scrollText(true, tv.line+1, 0, len(tv.lines))
	}
	return popupBox{color: layer, glyph: tv.t.Glyph, title: popupTitle(tv.t.Title, tv.t.Subject), hint: tv.hint(), scroll: scroll}.draw(w, body)
}

func (tv *textareaView) insert(s string) {
	parts := splitLines(s)
	l := tv.lines[tv.line]
	rest := append([]string(nil), l[tv.col:]...)
	head := append(l[:tv.col:tv.col], graphemes(parts[0])...)
	if len(parts) == 1 {
		tv.lines[tv.line] = append(head, rest...)
		tv.col = len(head)
		return
	}
	newLines := [][]string{head}
	for _, p := range parts[1 : len(parts)-1] {
		newLines = append(newLines, graphemes(p))
	}
	last := graphemes(parts[len(parts)-1])
	tv.col = len(last)
	newLines = append(newLines, append(last, rest...))
	tv.lines = append(tv.lines[:tv.line], append(newLines, tv.lines[tv.line+1:]...)...)
	tv.line += len(parts) - 1
}

// vertical moves the cursor a row on screen up or down, keeping its place
// across the row as well as it can.
func (tv *textareaView) vertical(dir int) {
	rows := tv.rows(tv.textW)
	cur := tv.cursorRow(rows)
	next := cur + dir
	if next < 0 || next >= len(rows) {
		return
	}
	x := 0
	for k := rows[cur].start; k < tv.col && k < len(tv.lines[tv.line]); k++ {
		x += tv.cellW(tv.lines[tv.line][k])
	}
	r := rows[next]
	tv.line, tv.col = r.line, r.start
	used := 0
	for tv.col < r.end && used+tv.cellW(tv.lines[r.line][tv.col]) <= x {
		used += tv.cellW(tv.lines[r.line][tv.col])
		tv.col++
	}
	if !tv.writing {
		tv.clampMove()
	}
}

// clampMove keeps the moving cursor on a character, as vim does.
func (tv *textareaView) clampMove() {
	tv.col = min(tv.col, max(len(tv.lines[tv.line])-1, 0))
}

func (tv *textareaView) key(a *App, key string, msg tea.KeyMsg) (bool, tea.Cmd) {
	if tv.writing {
		return true, tv.writeKey(key, msg)
	}
	return tv.moveKey(a, key)
}

func (tv *textareaView) writeKey(key string, msg tea.KeyMsg) tea.Cmd {
	if msg.Paste {
		tv.insert(string(msg.Runes))
		return nil
	}
	l := tv.lines[tv.line]
	switch key {
	case "Esc":
		tv.writing = false
		tv.clampMove()
	case "Enter":
		tv.insert("\n")
	case "Tab":
		tv.insert(Repeat(" ", tv.tabW)) // spaces, input_tab_width of them (Rules K8)
	case "Space":
		tv.insert(" ")
	case "←":
		if tv.col > 0 {
			tv.col--
		} else if tv.line > 0 {
			tv.line--
			tv.col = len(tv.lines[tv.line])
		}
	case "→":
		if tv.col < len(l) {
			tv.col++
		} else if tv.line+1 < len(tv.lines) {
			tv.line++
			tv.col = 0
		}
	case "↑":
		tv.vertical(-1)
	case "↓":
		tv.vertical(1)
	case "Home": // the start and the end of the whole text; 0|$ in the moving state
		tv.line, tv.col = 0, 0
	case "End":
		tv.line = len(tv.lines) - 1
		tv.col = len(tv.lines[tv.line])
	case "Backspace":
		switch {
		case tv.col > 0:
			tv.lines[tv.line] = append(l[:tv.col-1], l[tv.col:]...)
			tv.col--
		case tv.line > 0:
			// At the start of a line it joins the line above.
			prev := tv.lines[tv.line-1]
			tv.col = len(prev)
			tv.lines[tv.line-1] = append(prev, l...)
			tv.lines = append(tv.lines[:tv.line], tv.lines[tv.line+1:]...)
			tv.line--
		}
	case "Delete":
		switch {
		case tv.col < len(l):
			tv.lines[tv.line] = append(l[:tv.col], l[tv.col+1:]...)
		case tv.line+1 < len(tv.lines):
			tv.lines[tv.line] = append(l, tv.lines[tv.line+1]...)
			tv.lines = append(tv.lines[:tv.line+1], tv.lines[tv.line+2:]...)
		}
	default:
		if msg.Type == tea.KeyRunes && !msg.Alt {
			tv.insert(string(msg.Runes))
		}
	}
	return nil
}

// charClass sorts characters for w/b/e as vim does: blanks, word characters,
// the rest.
func charClass(c string) int {
	r := []rune(c)[0]
	switch {
	case unicode.IsSpace(r):
		return 0
	case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_':
		return 1
	}
	return 2
}

// pos is a place in the text; a line's end counts as a blank between lines.
type pos struct{ line, col int }

func (tv *textareaView) classAt(p pos) int {
	l := tv.lines[p.line]
	if p.col >= len(l) {
		return 0
	}
	return charClass(l[p.col])
}

func (tv *textareaView) nextPos(p pos) (pos, bool) {
	if p.col < len(tv.lines[p.line]) {
		return pos{p.line, p.col + 1}, true
	}
	if p.line+1 < len(tv.lines) {
		return pos{p.line + 1, 0}, true
	}
	return p, false
}

func (tv *textareaView) prevPos(p pos) (pos, bool) {
	if p.col > 0 {
		return pos{p.line, p.col - 1}, true
	}
	if p.line > 0 {
		return pos{p.line - 1, len(tv.lines[p.line-1])}, true
	}
	return p, false
}

// wordStart is w: the start of the next word.
func (tv *textareaView) wordStart() {
	p := pos{tv.line, tv.col}
	c := tv.classAt(p)
	ok := true
	for ok && tv.classAt(p) == c && c != 0 {
		p, ok = tv.nextPos(p)
	}
	for ok && tv.classAt(p) == 0 {
		p, ok = tv.nextPos(p)
	}
	tv.line, tv.col = p.line, p.col
}

// wordBack is b: the start of this word, or of the one before.
func (tv *textareaView) wordBack() {
	p, ok := tv.prevPos(pos{tv.line, tv.col})
	for ok && tv.classAt(p) == 0 {
		p, ok = tv.prevPos(p)
	}
	c := tv.classAt(p)
	for ok {
		q, more := tv.prevPos(p)
		if !more || tv.classAt(q) != c {
			break
		}
		p = q
	}
	tv.line, tv.col = p.line, p.col
}

// wordEnd is e: the end of this word, or of the next.
func (tv *textareaView) wordEnd() {
	p, ok := tv.nextPos(pos{tv.line, tv.col})
	for ok && tv.classAt(p) == 0 {
		p, ok = tv.nextPos(p)
	}
	c := tv.classAt(p)
	for ok {
		q, more := tv.nextPos(p)
		if !more || tv.classAt(q) != c {
			break
		}
		p = q
	}
	tv.line, tv.col = p.line, p.col
}

func (tv *textareaView) moveKey(a *App, key string) (bool, tea.Cmd) {
	half := max(tv.visible/2, 1)
	switch key {
	case "h", "←":
		tv.col = max(tv.col-1, 0)
	case "l", "→":
		tv.col++
	case "j", "↓":
		tv.vertical(1)
	case "k", "↑":
		tv.vertical(-1)
	case "w":
		tv.wordStart()
	case "b":
		tv.wordBack()
	case "e":
		tv.wordEnd()
	case "0":
		tv.col = 0
	case "$":
		tv.col = len(tv.lines[tv.line])
	case "d", "PgDn":
		for range half {
			tv.vertical(1)
		}
	case "u", "PgUp":
		for range half {
			tv.vertical(-1)
		}
	case "gg":
		tv.line, tv.col = 0, 0
	case "G":
		tv.line, tv.col = len(tv.lines)-1, 0
	case "i":
		tv.writing = true
		return true, nil
	case "a":
		tv.writing = true
		tv.col = min(tv.col+1, len(tv.lines[tv.line]))
		return true, nil
	case "o":
		tv.writing = true
		tv.col = len(tv.lines[tv.line])
		tv.insert("\n")
		return true, nil
	case "Enter":
		// A control character is not saved: the error row names the value,
		// as on a one-line input (components/input/textarea).
		if tv.hasControl() {
			tv.err = cmpOr(tv.t.Name, cmpOr(tv.t.Title, "The text")) + " can't have control characters"
			return true, nil
		}
		tv.err = ""
		v := tv.value()
		if v == tv.t.Old || tv.t.Done == nil {
			return true, a.closeTop()
		}
		return true, tea.Batch(a.closeTop(), then(tv.written, tv.t.Done(v)))
	case "Esc":
		if tv.value() == tv.t.Old {
			return true, a.closeTop()
		}
		return true, a.Confirm(Ask{Title: "Discard", Question: "Discard changes to " + tv.t.Title + "?", Verb: "discard"},
			func() tea.Cmd { return a.closeTop() })
	case "Space":
		return true, nil
	default:
		return false, nil
	}
	tv.clampMove()
	return true, nil
}

func (tv *textareaView) surface() string { return tv.t.Title }

func (tv *textareaView) keyRef() []refSection {
	return []refSection{
		{rows: []KeyRow{
			{Key: "Enter", Desc: "save"},
			{Key: "i", Desc: "write before the cursor"},
			{Key: "a", Desc: "write after the cursor"},
			{Key: "o", Desc: "write on a new line below"},
			{Key: "Esc", Desc: "cancel (asks first when something changed)"},
		}},
		{title: "move", rows: []KeyRow{
			{Key: "h|j|k|l", Desc: "left, down, up, right"},
			{Key: "w|b|e", Desc: "next word, word before, end of the word"},
			{Key: "0|$", Desc: "start, end of the line"},
			{Key: "u|d", Desc: "half a page up, down"},
			{Key: "gg|G", Desc: "first, last line"},
		}},
	}
}

func (tv *textareaView) typing() bool { return tv.writing }
