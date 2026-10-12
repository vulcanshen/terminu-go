package terminu

import (
	"slices"

	tea "github.com/charmbracelet/bubbletea"
)

// confirmView is a confirm popup (components/dialog/confirm): what to read in
// Overlay0 (a warning in Peach), a blank line, the question in Text last.
// Enter accepts, Esc cancels; Space does nothing.
type confirmView struct {
	ask     Ask
	accept  func() tea.Cmd
	keys    []ModeKey // its own hotkeys (y/n), listed in its ?
	content []string  // wrapped, coloured
	quest   []string
	top     int
	visible int
}

func newConfirm(ask Ask, accept func() tea.Cmd) *confirmView {
	return &confirmView{ask: ask, accept: accept}
}

// askKey runs one of the confirm's own hotkeys: as Enter, or as Esc and then
// its Do.
func (c *confirmView) askKey(a *App, k AskKey) tea.Cmd {
	if k.Accept {
		return a.acceptConfirm(c)
	}
	return tea.Batch(a.closeTop(), run(k.Do))
}

func (c *confirmView) hint() []KeyHint {
	return []KeyHint{{"Enter", c.ask.Verb}, {"Esc", "cancel"}}
}

func (c *confirmView) wrapTo(w int) {
	c.content, c.quest = nil, nil
	for i, d := range c.ask.Details {
		style := Fg(Overlay0)
		if slices.Contains(c.ask.Warn, i) {
			style = Fg(Peach)
		}
		if d == "" {
			c.content = append(c.content, "")
			continue
		}
		for _, l := range Wrap(d, w) {
			c.content = append(c.content, style.Render(l))
		}
	}
	for _, l := range Wrap(c.ask.Question, w) {
		c.quest = append(c.quest, Fg(Text).Render(l))
	}
}

func (c *confirmView) measure(sw, sh int) (int, int) {
	most := max(min(sw-2, 120), 1)
	content := Width(c.ask.Question)
	for _, d := range c.ask.Details {
		content = max(content, Width(d))
	}
	w := min(max(content+4, Width(titleCapsule(glyphConfirm, Upper(c.ask.Title), Base, 1<<20))+4, hintWidth(c.hint())+6), most)
	c.wrapTo(w - 4)
	rows := len(c.quest)
	if len(c.content) > 0 {
		rows += len(c.content) + 1
	}
	return w, min(rows+4, max(sh-2, 3))
}

func (c *confirmView) draw(w, h int, layer Color) []string {
	inner := w - 2
	c.wrapTo(w - 4)
	n := h - 4
	var body []string
	if len(c.content) > 0 {
		// Only the content scrolls; the question stays last (components/dialog/confirm).
		c.visible = max(n-len(c.quest)-1, 0)
		c.top = min(max(c.top, 0), max(len(c.content)-c.visible, 0))
		for i := c.top; i < c.top+c.visible; i++ {
			if i < len(c.content) {
				body = append(body, padRow(c.content[i], inner))
			} else {
				body = append(body, Repeat(" ", inner))
			}
		}
		body = append(body, Repeat(" ", inner))
	}
	for _, q := range c.quest {
		if len(body) < n {
			body = append(body, padRow(q, inner))
		}
	}
	for len(body) < n {
		body = append(body, Repeat(" ", inner))
	}
	scroll := ""
	if len(c.content) > c.visible && c.visible > 0 {
		scroll = scrollText(false, c.top+1, min(c.top+c.visible, len(c.content)), len(c.content))
	}
	return popupBox{color: layer, glyph: glyphConfirm, title: Upper(c.ask.Title), hint: c.hint(), scroll: scroll}.draw(w, body)
}

func (c *confirmView) key(a *App, key string, _ tea.KeyMsg) (bool, tea.Cmd) {
	switch key {
	case "Enter":
		return true, a.acceptConfirm(c)
	case "Space":
		return true, nil
	case "j", "↓":
		c.top++
	case "k", "↑":
		c.top--
	default:
		for _, k := range c.keys {
			if k.takes(key) {
				return true, k.Do(key)
			}
		}
		for _, k := range c.ask.Keys {
			if k.Key == key {
				return true, c.askKey(a, k)
			}
		}
		return false, nil
	}
	return true, nil
}

func (c *confirmView) surface() string { return c.ask.Title }

func (c *confirmView) keyRef() []refSection {
	rows := []KeyRow{{Key: "Enter", Desc: c.ask.Verb}}
	for _, k := range c.keys {
		rows = append(rows, KeyRow{Key: k.Key, Desc: k.Desc})
	}
	for _, k := range c.ask.Keys {
		rows = append(rows, KeyRow{Key: k.Key, Desc: k.Desc})
	}
	rows = append(rows, KeyRow{Key: "Esc", Desc: "cancel"})
	secs := []refSection{{rows: rows}}
	if len(c.ask.Details) > 0 {
		secs = append(secs, refSection{title: "move", rows: []KeyRow{{Key: "j/k", Desc: "scroll down, up"}}})
	}
	return secs
}

func (c *confirmView) typing() bool { return false }
