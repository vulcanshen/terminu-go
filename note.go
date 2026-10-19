package terminu

import (
	tea "github.com/charmbracelet/bubbletea"
)

// noteView is a read-only popup that scrolls (components/dialog/note): a key
// reference, a viewer, an error popup, the typing note on a form.
type noteView struct {
	glyph, title string
	lines        []string // drawn lines, already as wide as they will be
	// text, when set, is a paragraph in textStyle the lines are folded from
	// at the width it gets, each time it is measured: too long, it folds,
	// never cut (an error popup, the typing note; components/dialog/note).
	text      string
	textStyle Style
	// build, when set, makes the lines for a room of so many cells, each
	// time it is measured: a key reference whose "what each row is" section
	// folds its descriptions to the room there is.
	build func(room int) []string
	// onlyClose takes every key but Enter and Esc and does nothing with them,
	// ? and q too: the typing note, which comes while the user is still typing
	// (components/dialog/form, Rules K9).
	onlyClose    bool
	known        bool   // the width is known when it opens; a viewer's is not
	color        *Color // the frame's colour when it is not the layer's (Red)
	enterCloses  bool   // an error popup and the typing note close with Enter too
	helpCloses   bool   // a key reference closes with ? too
	hint         []KeyHint
	name         string
	extraKeys    []KeyRow
	top, visible int
}

func (n *noteView) measure(sw, sh int) (int, int) {
	w := popupWidth(sw, 0, false)
	if n.build != nil {
		n.lines = n.build(1 << 20) // as wide as they are, for the width
	}
	if n.text != "" {
		n.lines = nil
		for _, l := range Wrap(n.text, max(min(max(Width(n.text), 20), w-4), 1)) {
			n.lines = append(n.lines, n.textStyle.Render(l))
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
			body = append(body, padRow(n.lines[i], inner))
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
	default:
		return false, nil
	}
	n.top = min(max(n.top, 0), max(len(n.lines)-n.visible, 0))
	return true, nil
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
	secs := []refSection{{rows: rows}}
	if len(n.lines) > n.visible {
		// Only when it does not all fit (Rules M6).
		secs = append(secs, refSection{title: "move", rows: []KeyRow{
			{Key: "j|k", Desc: "scroll down, up"},
			{Key: "u|d", Desc: "half a page up, down"},
			{Key: "gg|G", Desc: "to the top, the bottom"},
		}})
	}
	return secs
}

// typing: the typing note takes what is typed on it, so the app's own core
// keys do not act there either.
func (n *noteView) typing() bool { return n.onlyClose }

// keyReference builds the key reference of a surface (components/dialog/note),
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
	// (components/dialog/note).
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
				key, desc := Pad(renderKey(r.Key, Fg(Blue), Overlay0), keyW), Fg(Text).Render(r.Desc)
				if r.Disabled {
					key, desc = Fg(Surface2).Render(Pad(r.Key, keyW)), Fg(Surface2).Render(r.Desc)
				}
				if s.names {
					for i, l := range Wrap(r.Desc, max(room-1-keyW-2, 1)) {
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
		glyph: glyphKeys, title: title + " KEYS", lines: build(1 << 20), build: build, known: true, helpCloses: true,
		hint: []KeyHint{{"?|Esc", "close"}}, name: title + " keys",
	}
}

// errorPopup is a popup's action failing (components/dialog/note, error
// popup): Red frame and text, the fire glyph, Enter or Esc to close.
func errorPopup(title, subject, message string) *noteView {
	red := Red
	return &noteView{
		glyph: glyphError, title: popupTitle(title, subject), text: message, textStyle: Fg(Red), known: true, color: &red, enterCloses: true,
		hint: []KeyHint{{"Enter|Esc", "close"}}, name: title,
	}
}
