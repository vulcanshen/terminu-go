package terminu

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rivo/uniseg"
)

// lineEditor is one typing row (components/input/README, editing keys): a
// character is what looks like one (a CJK character, an emoji, a letter with
// its combining marks); a pasted control character is one character too.
type lineEditor struct {
	g      []string // grapheme clusters; a pasted control character is one on its own
	pos    int      // the cursor: before g[pos]
	scroll int      // the first cluster shown when the value is wider than the row
}

func newEditor(s string) *lineEditor {
	e := &lineEditor{}
	e.set(s)
	return e
}

// set replaces the value, through the same filter as a paste, and puts the
// cursor at its end.
func (e *lineEditor) set(s string) {
	e.g, e.pos, e.scroll = nil, 0, 0
	e.insert(s, true)
}

func (e *lineEditor) value() string { return strings.Join(e.g, "") }
func (e *lineEditor) empty() bool   { return len(e.g) == 0 }

// insert puts s at the cursor. Pasted text keeps every control character
// (C0, DEL, C1), each one character, "\r\n" two; typed text has none of those
// to begin with.
func (e *lineEditor) insert(s string, pasted bool) {
	var add []string
	gr := uniseg.NewGraphemes(s)
	for gr.Next() {
		c := gr.Str()
		if c == "\r\n" {
			if pasted {
				add = append(add, "\r", "\n")
			}
			continue
		}
		if _, ok := ctrlName(c); ok {
			if pasted {
				add = append(add, c)
			}
			continue
		}
		add = append(add, c)
	}
	// A combining mark that comes on its own key joins the character before
	// it: the two look like one.
	if len(add) > 0 && e.pos > 0 && uniseg.GraphemeClusterCount(e.g[e.pos-1]+add[0]) == 1 {
		e.g[e.pos-1] += add[0]
		add = add[1:]
	}
	e.g = append(e.g[:e.pos], append(add, e.g[e.pos:]...)...)
	e.pos += len(add)
}

// key applies one editing key and reports whether it was one: characters and
// Space are typed, ←/→ move a character, Home/End to an end, Backspace and
// Delete remove one. Ctrl- and Alt- combinations type nothing.
func (e *lineEditor) key(name string, msg tea.KeyMsg) bool {
	if msg.Paste {
		e.insert(string(msg.Runes), true)
		return true
	}
	switch name {
	case "←":
		e.pos = max(e.pos-1, 0)
	case "→":
		e.pos = min(e.pos+1, len(e.g))
	case "Home":
		e.pos = 0
	case "End":
		e.pos = len(e.g)
	case "Backspace":
		if e.pos > 0 {
			e.g = append(e.g[:e.pos-1], e.g[e.pos:]...)
			e.pos--
		}
	case "Delete":
		if e.pos < len(e.g) {
			e.g = append(e.g[:e.pos], e.g[e.pos+1:]...)
		}
	case "Space":
		e.insert(" ", false)
	default:
		if msg.Type == tea.KeyRunes && !msg.Alt {
			e.insert(string(msg.Runes), false)
			return true
		}
		return false
	}
	return true
}

// ctrlName is how a control character (C0, DEL, C1) is written: \r, \n and
// \t by name, the rest as \xNN (components/input/README, pasted control
// characters). ok is false for anything else.
func ctrlName(c string) (name string, ok bool) {
	switch c {
	case "\r":
		return `\r`, true
	case "\n":
		return `\n`, true
	case "\t":
		return `\t`, true
	}
	r := []rune(c)
	if len(r) != 1 || !(r[0] < 0x20 || r[0] >= 0x7f && r[0] <= 0x9f) {
		return "", false
	}
	return fmt.Sprintf(`\x%02x`, r[0]), true
}

// cellText is how a cluster is drawn: a pasted control character as its red
// name, never cut apart; grey on a grey row.
func cellText(c string, plain Style) string {
	if name, ok := ctrlName(c); ok {
		marker := Red
		if plain.Fg == Overlay0 {
			marker = Overlay0
		}
		return Fg(marker).Render(name)
	}
	return plain.Render(c)
}

// greyText is s as the user did not type it (the old value offered): through
// the same filter as a paste, all of it grey (components/input/README).
func greyText(s string) string {
	var b, run strings.Builder
	flush := func() {
		if run.Len() > 0 {
			b.WriteString(Fg(Overlay0).Render(run.String()))
			run.Reset()
		}
	}
	for _, c := range newEditor(s).g {
		if _, ok := ctrlName(c); ok {
			flush()
			b.WriteString(cellText(c, Fg(Overlay0)))
			continue
		}
		run.WriteString(c)
	}
	flush()
	return b.String()
}

func cellWidth(c string) int {
	if name, ok := ctrlName(c); ok {
		return len(name)
	}
	return Width(c)
}

// view draws the row in w cells: the value in style, the cursor (when shown) a
// block of cursorStyle, and the part near the cursor when the value is wider
// than the row, a "…" on the side that is cut (components/input/README).
// mask, when set, draws each character as something else (password).
// suggest, when the value is empty, is offered in grey from the cursor's cell
// on: the cursor sits on its first character (the grey suggestion).
func (e *lineEditor) view(w int, style, cursorStyle Style, showCursor bool, mask func(string) string, suggest string) string {
	if w <= 0 {
		return ""
	}
	if len(e.g) == 0 && suggest != "" {
		e.scroll = 0
		return Pad(suggestion(suggest, w, cursorStyle, showCursor), w)
	}
	cells := make([]string, len(e.g))
	widths := make([]int, len(e.g))
	for i, c := range e.g {
		if mask != nil {
			cells[i] = mask(c)
		} else {
			cells[i] = c
		}
		widths[i] = cellWidth(cells[i])
		if mask != nil {
			widths[i] = Width(cells[i])
		}
	}
	draw := func(i int) string {
		if mask != nil {
			return style.Render(cells[i])
		}
		return cellText(cells[i], style)
	}
	cursorW := 0
	if showCursor && e.pos == len(e.g) {
		cursorW = 1
	}
	total := cursorW
	for _, x := range widths {
		total += x
	}
	if total <= w {
		e.scroll = 0
		var b strings.Builder
		for i := range cells {
			if showCursor && i == e.pos {
				b.WriteString(cursorOn(cells[i], cursorStyle))
				continue
			}
			b.WriteString(draw(i))
		}
		if cursorW == 1 {
			b.WriteString(cursorStyle.Render(" "))
		}
		return Pad(b.String(), w)
	}
	// Scroll just enough to keep the cursor in sight, an ellipsis on each cut side.
	if e.pos < e.scroll {
		e.scroll = e.pos
	}
	fits := func(start int) bool {
		used := 0
		if start > 0 {
			used++
		}
		for i := start; i < e.pos && i < len(cells); i++ {
			used += widths[i]
		}
		if e.pos < len(cells) {
			used += widths[e.pos]
		} else {
			used++
		}
		if e.pos+1 < len(cells) {
			used++ // room for the trailing ellipsis
		}
		return used <= w
	}
	for e.scroll < e.pos && !fits(e.scroll) {
		e.scroll++
	}
	var b strings.Builder
	used := 0
	if e.scroll > 0 {
		b.WriteString(style.Render("…"))
		used++
	}
	i := e.scroll
	for ; i < len(cells); i++ {
		need := widths[i]
		rest := i+1 < len(cells) || (showCursor && e.pos == len(e.g))
		if used+need > w-boolInt(rest && i != e.pos) && i > e.pos {
			break
		}
		if used+need > w {
			break
		}
		if showCursor && i == e.pos {
			b.WriteString(cursorOn(cells[i], cursorStyle))
		} else {
			b.WriteString(draw(i))
		}
		used += need
	}
	if i == len(cells) && showCursor && e.pos == len(e.g) && used < w {
		b.WriteString(cursorStyle.Render(" "))
		used++
	}
	if i < len(cells) && used < w {
		b.WriteString(style.Render("…"))
	}
	return Pad(b.String(), w)
}

// suggestion is s offered in grey (components/input/README, the grey
// suggestion): from the cursor's cell on, the cursor on its first character
// like any cursor, the rest Overlay0, cut with "…" when wider than w.
func suggestion(s string, w int, cursorStyle Style, showCursor bool) string {
	g := newEditor(s).g
	if !showCursor || len(g) == 0 {
		return Truncate(greyText(s), w)
	}
	first := cursorOn(g[0], cursorStyle)
	return first + Truncate(greyText(strings.Join(g[1:], "")), w-Width(first))
}

// cursorOn draws the character under the cursor as a block.
func cursorOn(c string, cursorStyle Style) string {
	if name, ok := ctrlName(c); ok { // TODO(tdp G81): a control character's name stays red on the cursor

		return Style{Fg: Red, Bg: cursorStyle.Bg}.Render(name)
	}
	return cursorStyle.Render(c)
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// hasControl reports whether the value has a pasted control character, which
// a one-line value that is used must not have (components/input/README).
func (e *lineEditor) hasControl() bool {
	for _, c := range e.g {
		if _, ok := ctrlName(c); ok {
			return true
		}
	}
	return false
}
