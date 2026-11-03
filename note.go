package terminu

import (
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// noteView is a read-only popup that scrolls (tdp note): a key
// reference, a viewer, an error popup, the typing note on a form.
type noteView struct {
	glyph, title string
	lines        []string // drawn lines, already as wide as they will be
	// text, when set, is the paragraphs in textStyle the lines are folded
	// from at the width it gets, each time it is measured, each on a row of
	// its own: too long, it folds, never cut (an error popup, the typing
	// note; tdp note).
	text      []string
	textStyle Style
	// build, when set, makes the lines for a room of so many cells, each
	// time it is measured: a key reference whose "what each row is" section
	// folds its descriptions to the room there is.
	build func(room int) []string
	// onlyClose takes every key but Enter and Esc and does nothing with them,
	// ? and q too: the typing note, which comes while the user is still typing
	// (tdp form, tdp K9).
	onlyClose    bool
	known        bool   // the width is known when it opens; a viewer's is not
	color        *Color // the frame's colour when it is not the layer's (Red)
	enterCloses  bool   // an error popup and the typing note close with Enter too
	helpCloses   bool   // a key reference closes with ? too
	hint         []KeyHint
	name         string
	extraKeys    []KeyRow
	top, visible int
	// search is a key reference's / search, as a viewer's (tdp note.5,
	// note.3): query is what was asked for, hits where it is in the lines,
	// found again each time they are built, current the one drawn as the
	// cursor block.
	search  bool
	query   string
	hits    []TextMatch
	current int
}

func (n *noteView) measure(sw, sh int) (int, int) {
	w := popupWidth(sw, 0, false)
	if n.build != nil {
		n.lines = n.build(1 << 20) // as wide as they are, for the width
	}
	if len(n.text) > 0 {
		widest := 0
		for _, p := range n.text {
			widest = max(widest, Width(p))
		}
		n.lines = nil
		for _, p := range n.text {
			for _, l := range Wrap(p, max(min(max(widest, 20), w-4), 1)) {
				n.lines = append(n.lines, n.textStyle.Render(l))
			}
		}
	}
	if n.known {
		content := 0
		for _, l := range n.lines {
			content = max(content, Width(l))
		}
		content = max(content+4, Width(titleCapsule(n.glyph, n.title, Base, 1<<20))+4, hintWidth(n.hint)+6)
		if t := len(n.lines); t+4 > max(sh-2, 3) {
			content = max(content, scrollRoom(n.hint, scrollText(false, t, t, t)))
		}
		w = min(content, w)
	}
	if n.build != nil {
		n.lines = n.build(max(w-4, 1)) // folded to the room it has
	}
	if n.query != "" {
		n.hits = findIn(noteChars(n.lines), n.query)
		n.current = min(n.current, max(len(n.hits)-1, 0))
	}
	h := min(len(n.lines)+4, max(sh-2, 3))
	return w, h
}

func (n *noteView) draw(w, h int, layer Color) []string {
	color := layer
	if n.color != nil {
		color = *n.color
	}
	inner := w - 2
	n.visible = max(h-4, 0)
	n.top = min(max(n.top, 0), max(len(n.lines)-n.visible, 0))
	body := make([]string, 0, n.visible)
	for i := n.top; i < n.top+n.visible; i++ {
		if i < len(n.lines) {
			body = append(body, padRow(n.hitsDrawn(i, layer), inner))
		} else {
			body = append(body, Repeat(" ", inner))
		}
	}
	scroll := ""
	if len(n.lines) > n.visible {
		scroll = scrollText(false, n.top+1, min(n.top+n.visible, len(n.lines)), len(n.lines))
	}
	return popupBox{color: color, glyph: n.glyph, title: n.title, hint: n.hint, scroll: scroll}.draw(w, body)
}

func (n *noteView) key(a *App, key string, _ tea.KeyMsg) (bool, tea.Cmd) {
	if n.onlyClose {
		if key == "Enter" || key == "Esc" {
			return true, a.closeTop()
		}
		return true, nil
	}
	half := max(n.visible/2, 1)
	switch key {
	case "j", "↓":
		n.top++
	case "k", "↑":
		n.top--
	case "d", "PgDn":
		n.top += half
	case "u", "PgUp":
		n.top -= half
	case "gg":
		n.top = 0
	case "G":
		n.top = len(n.lines)
	case "Enter":
		if n.enterCloses {
			return true, a.closeTop()
		}
		return true, nil
	case "?":
		if n.helpCloses {
			return true, a.closeTop()
		}
		return false, nil
	case "Space":
		return true, nil
	case "/", "n", "N":
		if !n.search {
			return false, nil
		}
		if key == "/" {
			return true, a.Text(TextInput{Title: "Search", Verb: "find", Search: true, Done: func(q string) tea.Cmd { return n.find(a, q) }})
		}
		if len(n.hits) > 0 {
			by := map[string]int{"n": 1, "N": -1}[key]
			n.current = ((n.current+by)%len(n.hits) + len(n.hits)) % len(n.hits)
			n.follow(n.hits[n.current].Line)
		}
		return true, nil
	default:
		return false, nil
	}
	n.top = min(max(n.top, 0), max(len(n.lines)-n.visible, 0))
	return true, nil
}

// find makes the places q is in the lines the hits, the first in sight; with
// none an Info toast says so (tdp note.3).
func (n *noteView) find(a *App, q string) tea.Cmd {
	n.query, n.current = q, 0
	n.hits = findIn(noteChars(n.lines), q)
	if len(n.hits) == 0 {
		return a.Toast(Info, "No line has "+q)
	}
	n.follow(n.hits[0].Line)
	return nil
}

// follow scrolls as little as it takes to show line.
func (n *noteView) follow(line int) {
	if line < n.top {
		n.top = line
	} else if line >= n.top+n.visible {
		n.top = line - n.visible + 1
	}
}

// noteChars is the plain characters of each line.
func noteChars(lines []string) [][]string {
	rows := make([][]string, len(lines))
	for i, l := range lines {
		rows[i] = graphemes(StripStyles(l))
	}
	return rows
}

// hitsDrawn is line i with the search's hits on it: each underlined in its
// own colours, the current one the cursor block (tdp note.2).
func (n *noteView) hitsDrawn(i int, layer Color) string {
	line := n.lines[i]
	if !slices.ContainsFunc(n.hits, func(m TextMatch) bool { return m.Line == i }) {
		return line
	}
	chars := graphemes(StripStyles(line))
	hit := make([]int, len(chars)) // 0 none, 1 a hit, 2 the current one
	for k, m := range n.hits {
		if m.Line != i {
			continue
		}
		for c := max(m.From, 0); c < min(m.To, len(chars)); c++ {
			hit[c] = 1
			if k == n.current {
				hit[c] = 2
			}
		}
	}
	var b strings.Builder
	cell := 0
	for k, c := range chars {
		w := Width(c)
		switch hit[k] {
		case 2:
			b.WriteString(On(layer).Render(c))
		case 1:
			b.WriteString(underlined(Cut(line, cell, cell+w)))
		default:
			b.WriteString(Cut(line, cell, cell+w))
		}
		cell += w
	}
	return b.String()
}

func (n *noteView) surface() string { return n.name }

func (n *noteView) keyRef() []refSection {
	rows := append([]KeyRow(nil), n.extraKeys...)
	switch {
	case n.enterCloses:
		rows = append(rows, KeyRow{Key: "Enter|Esc", Desc: "close"})
	default:
		rows = append(rows, KeyRow{Key: "Esc", Desc: "close"})
	}
	if n.search {
		rows = append([]KeyRow{{Key: "/", Desc: "search the text"}}, rows...)
	}
	var move []KeyRow
	if len(n.lines) > n.visible {
		// Only when it does not all fit (tdp M6).
		move = []KeyRow{
			{Key: "j|k", Desc: "scroll down, up"},
			{Key: "u|d", Desc: "half a page up, down"},
			{Key: "gg|G", Desc: "to the top, the bottom"},
		}
	}
	if len(n.hits) > 0 { // only with hits to go to (tdp M6)
		move = append(move, KeyRow{Key: "n|N", Desc: "the next, the previous hit"})
	}
	secs := []refSection{{rows: rows}}
	if len(move) > 0 {
		secs = append(secs, refSection{title: "move", rows: move})
	}
	return secs
}

// typing: the typing note takes what is typed on it, so the app's own core
// keys do not act there either.
func (n *noteView) typing() bool { return n.onlyClose }

// keyReference builds the key reference of a surface (tdp note),
// title its name cased as a title is (a name in it as it is, components'
// upper case), "KEYS" after it:
// a section title is a hexagon and upper case, one cell from the frame; every
// row is two cells in, its key padded to the widest key, two cells, then what
// it does.
func keyReference(title string, sections []refSection) *noteView {
	keyW := 0
	for _, s := range sections {
		for _, r := range s.rows {
			keyW = max(keyW, Width(r.Key))
		}
	}
	// The rows in room cells. A row of keys is one line, cut when too long;
	// a row of "what each row is" folds its description in the right column,
	// the name's column left blank under it: it is a sentence to read
	// (tdp note).
	build := func(room int) []string {
		var lines []string
		for _, s := range sections {
			if len(s.rows) == 0 {
				continue
			}
			if s.title != "" {
				lines = append(lines, Fg(Overlay0).Render(glyphHexagon+" "+Upper(s.title)))
			}
			for _, r := range s.rows {
				key, desc := Pad(renderKey(r.Key, Fg(Blue), Overlay0), keyW), Fg(Text).Render(Escape(r.Desc))
				if r.Disabled {
					key, desc = Fg(Surface2).Render(Pad(r.Key, keyW)), Fg(Surface2).Render(Escape(r.Desc))
				}
				if s.names {
					for i, l := range Wrap(Escape(r.Desc), max(room-1-keyW-2, 1)) {
						name := Repeat(" ", keyW)
						if i == 0 {
							name = Fg(Text).Render(Pad(r.Key, keyW))
						}
						lines = append(lines, " "+name+"  "+Fg(Text).Render(l))
					}
					continue
				}
				lines = append(lines, " "+key+"  "+desc)
			}
		}
		return lines
	}
	return &noteView{
		glyph: glyphKeys, title: title + " KEYS", lines: build(1 << 20), build: build, known: true, helpCloses: true, search: true,
		hint: []KeyHint{{"/", "search"}, {"?|Esc", "close"}}, name: title + " keys",
	}
}

// errorPopup is a popup's action failing (tdp note.6
// popup): Red frame and text, the fire glyph, Enter or Esc to close.
func errorPopup(title, subject string, message ...string) *noteView {
	red := Red
	text := make([]string, len(message))
	for i, m := range message {
		text[i] = Escape(m) // a short text: a \n from outside by its name (components, control characters)
	}
	return &noteView{
		glyph: glyphError, title: popupTitle(title, subject), text: text, textStyle: Fg(Red), known: true, color: &red, enterCloses: true,
		hint: []KeyHint{{"Enter|Esc", "close"}}, name: title,
	}
}
