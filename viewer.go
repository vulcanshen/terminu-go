package terminu

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Viewer is a note to read (components/dialog/note): a YAML view, a log, a
// file. It scrolls, may have keys of its own and a selection mode, and
// closes with Esc only. Keep the pointer: lines may keep arriving while
// Loading is true.
type Viewer struct {
	// Title is what it shows, in capitals ("YAML"); Subject is the thing
	// itself, as it is ("pod/nginx"): the title reads "YAML — pod/nginx".
	Title, Subject string
	Glyph          string
	Lines          []string
	Loading        bool
	// Tabs, when set, follow the title (components/layout/popup); h/l switch
	// them and OnTab fills Lines for the new one.
	Tabs  []string
	Tab   int
	OnTab func(tab int) tea.Cmd
	// Keys are the viewer's own hotkeys, first in its hint.
	Keys []ViewerKey
	// Select, when set, gives it a selection mode (Rules K11).
	Select *Selection

	top, visible int
	// The selection mode: the cursor and where the selection started, in
	// characters of the plain lines.
	selecting    bool
	line, col    int
	fromL, fromC int
	plain        [][]string
}

// ViewerKey is one of a viewer's own hotkeys.
type ViewerKey struct {
	Key, Desc string
	Do        func() tea.Cmd
}

// Selection is a viewer's selection mode: Key starts it, the cursor moves
// through the text (h/j/k/l, w/b/e, 0/$), CopyKey hands the selected text to
// Copy and leaves; Esc leaves.
type Selection struct {
	Key     string // "v"
	Name    string // the mode's name on the frame: "Visual"
	CopyKey string // "y" when unset
	Copy    func(text string) tea.Cmd
}

func (s *Selection) copyKey() string { return cmpOr(s.CopyKey, "y") }

// Viewer opens a viewer.
func (a *App) Viewer(v *Viewer) tea.Cmd { return a.open(&viewerView{v: v}) }

// ScrollTo puts line i at the top, as far as the lines go.
func (v *Viewer) ScrollTo(i int) { v.top = i }

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
	if v.selecting {
		return []KeyHint{{v.Select.copyKey(), "copy"}, {"Esc", "cancel"}}
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
	return popupWidth(sw, 0, false), min(len(vv.v.Lines)+4, max(sh-2, 3))
}

func (vv *viewerView) isLoading() bool { return vv.v.Loading }

func (vv *viewerView) draw(w, h int, layer Color) []string {
	v := vv.v
	inner := w - 2
	v.visible = max(h-4, 1)
	v.plain = v.plain[:0]
	for _, l := range v.Lines {
		v.plain = append(v.plain, graphemes(StripStyles(l)))
	}
	if v.selecting {
		v.top = scrollWindow(v.top, v.line, v.visible, len(v.Lines))
	}
	v.top = min(max(v.top, 0), max(len(v.Lines)-v.visible, 0))
	body := make([]string, 0, v.visible)
	for i := v.top; i < v.top+v.visible; i++ {
		if i >= len(v.Lines) {
			body = append(body, Repeat(" ", inner))
			continue
		}
		body = append(body, padRow(vv.lineWithSelection(i), inner))
	}
	scroll := ""
	if len(v.Lines) > v.visible {
		scroll = scrollText(false, v.top+1, min(v.top+v.visible, len(v.Lines)), len(v.Lines))
	}
	box := popupBox{color: layer, glyph: v.Glyph, title: vv.title(), hint: vv.hint(), scroll: scroll, loading: v.Loading}
	for i, t := range v.Tabs {
		box.tabs = append(box.tabs, Capsule{Text: Upper(t), Selected: i == v.Tab})
	}
	if v.selecting {
		// A mode marks its frame: the mode colour and its name (Rules K11).
		box.color, box.mode = Yellow, cmpOr(v.Select.Name, "Select")
	}
	return box.draw(w, body)
}

// lineWithSelection is line i with the part selected drawn in the mode's
// colour (components/color: what a mode selects is Yellow). TODO(tdp G39):
// as vim's v: from the start to the cursor, the character under it in.
func (vv *viewerView) lineWithSelection(i int) string {
	v := vv.v
	line := v.Lines[i]
	if !v.selecting {
		return line
	}
	start, end := pos{v.fromL, v.fromC}, pos{v.line, v.col}
	if end.line < start.line || end.line == start.line && end.col < start.col {
		start, end = end, start
	}
	if i < start.line || i > end.line {
		return line
	}
	chars := v.plain[i]
	from, to := 0, len(chars)
	if i == start.line {
		from = start.col
	}
	if i == end.line {
		to = min(end.col+1, len(chars)) // the character under the cursor is in
	}
	cells := func(n int) int {
		w := 0
		for _, c := range chars[:min(n, len(chars))] {
			w += Width(c)
		}
		return w
	}
	picked := strings.Join(chars[min(from, len(chars)):max(to, min(from, len(chars)))], "")
	if picked == "" {
		picked = " " // an empty line in the selection still shows
	}
	total := Width(line)
	return Cut(line, 0, cells(from)) + Style{Fg: Base, Bg: Yellow}.Render(picked) + Cut(line, cells(to), total)
}

// selected is the text from where the selection started to the cursor, the
// character under the cursor in.
func (v *Viewer) selected() string {
	start, end := pos{v.fromL, v.fromC}, pos{v.line, v.col}
	if end.line < start.line || end.line == start.line && end.col < start.col {
		start, end = end, start
	}
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

// selectKey is a key in the selection mode: the cursor moves through the
// text as in vim, without wrapping at the ends (Rules K12); other keys do
// nothing (Rules K11).
func (vv *viewerView) selectKey(key string) tea.Cmd {
	v := vv.v
	if len(v.plain) == 0 {
		v.selecting = false
		return nil
	}
	half := max(v.visible/2, 1)
	lineLen := func() int { return len(v.plain[v.line]) }
	p := pos{v.line, v.col}
	switch key {
	case "Esc":
		v.selecting = false
		return nil
	case v.Select.copyKey():
		v.selecting = false
		if v.Select.Copy == nil {
			return nil
		}
		return v.Select.Copy(v.selected())
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
	case "gg", "Home":
		v.line, v.col = 0, 0
	case "G", "End":
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
	}
	v.col = min(v.col, max(lineLen()-1, 0))
	return nil
}

func (vv *viewerView) key(a *App, key string, msg tea.KeyMsg) (bool, tea.Cmd) {
	v := vv.v
	if v.selecting {
		switch key {
		case "?", "q", "Ctrl-C":
			return false, nil // the mode's key reference, the quit flow (Rules K11)
		}
		return true, vv.selectKey(key)
	}
	half := max(v.visible/2, 1)
	switch key {
	case "j", "↓":
		v.top++
	case "k", "↑":
		v.top--
	case "d", "PgDn":
		v.top += half
	case "u", "PgUp":
		v.top -= half
	case "gg", "Home":
		v.top = 0
	case "G", "End":
		v.top = len(v.Lines)
	case "h", "←", "l", "→":
		if n := len(v.Tabs); n > 0 {
			dir := 1
			if key == "h" || key == "←" {
				dir = -1
			}
			v.Tab = (v.Tab + dir + n) % n
			v.top = 0
			return true, run2tab(v.OnTab, v.Tab)
		}
	case "Enter", "Space":
		// A note closes with Esc only (components/dialog/note).
	default:
		if v.Select != nil && key == v.Select.Key {
			v.selecting = true
			v.line, v.col = min(v.top, max(len(v.Lines)-1, 0)), 0
			v.fromL, v.fromC = v.line, v.col
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
	v.top = min(max(v.top, 0), max(len(v.Lines)-v.visible, 0))
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
	if v.selecting {
		return []refSection{
			{rows: []KeyRow{{Key: v.Select.copyKey(), Desc: "copy the selection and leave"}, {Key: "Esc", Desc: "leave " + cmpOr(v.Select.Name, "Select")}}},
			{title: "move", rows: []KeyRow{
				{Key: "h/j/k/l", Desc: "left, down, up, right"},
				{Key: "w/b/e", Desc: "next word, word before, end of the word"},
				{Key: "0/$", Desc: "start, end of the line"},
				{Key: "u/d", Desc: "half a page up, down"},
				{Key: "gg/G", Desc: "first, last line"},
			}},
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
	move := []KeyRow{
		{Key: "j/k", Desc: "scroll down, up"},
		{Key: "u/d", Desc: "half a page up, down"},
		{Key: "gg/G", Desc: "to the top, the bottom"},
	}
	if len(v.Tabs) > 0 {
		move = append([]KeyRow{{Key: "h/l", Desc: "the tab before, after"}}, move...)
	}
	return []refSection{{rows: rows}, {title: "move", rows: move}}
}

func (vv *viewerView) typing() bool { return false }
