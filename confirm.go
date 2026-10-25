package terminu

import (
	"slices"

	tea "github.com/charmbracelet/bubbletea"
)

// confirmView is a confirm popup (tdp confirm): what to read in
// Overlay0 (a warning in Peach), a blank line, the question in Text last.
// Enter accepts, Esc cancels, and nothing else answers it (no y, n: Rules
// F6); Space does nothing.
type confirmView struct {
	ask    Ask
	accept func() tea.Cmd
	// quit, when set, is Esc: it leaves the app instead of cancelling, the
	// one confirm that does (tdp E3, MoveAside; tdp confirm).
	quit    func() tea.Cmd
	content []string // wrapped, coloured
	quest   []string
	top     int
	visible int
}

func newConfirm(ask Ask, accept func() tea.Cmd) *confirmView {
	return &confirmView{ask: ask, accept: accept}
}

func (c *confirmView) hint() []KeyHint {
	return []KeyHint{{"Enter", c.ask.Verb}, {"Esc", c.escVerb()}}
}

// escVerb is what Esc does: cancel, or quit (MoveAside).
func (c *confirmView) escVerb() string {
	if c.quit != nil {
		return "quit"
	}
	return "cancel"
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
		for _, l := range Wrap(Escape(d), w) {
			c.content = append(c.content, style.Render(l))
		}
	}
	for _, l := range Wrap(Escape(c.ask.Question), w) {
		c.quest = append(c.quest, Fg(Text).Render(l))
	}
}

func (c *confirmView) measure(sw, sh int) (int, int) {
	most := max(min(sw-2, 120), 1)
	content := Width(Escape(c.ask.Question))
	for _, d := range c.ask.Details {
		content = max(content, Width(Escape(d)))
	}
	w := min(max(content+4, Width(titleCapsule(glyphConfirm, popupTitle(c.ask.Title, c.ask.Subject), Base, 1<<20))+4, hintWidth(c.hint())+6), most)
	rows := func() int {
		c.wrapTo(w - 4)
		n := len(c.quest)
		if len(c.content) > 0 {
			n += len(c.content) + 1
		}
		return n
	}
	h := min(rows()+4, max(sh-2, 3))
	// Only the details scroll: room for their position at the widest.
	if t := len(c.content); t > 0 && rows()+4 > h {
		w = min(max(w, scrollRoom(c.hint(), scrollText(false, t, t, t))), most)
		h = min(rows()+4, max(sh-2, 3))
	}
	return w, h
}

func (c *confirmView) draw(w, h int, layer Color) []string {
	inner := w - 2
	c.wrapTo(w - 4)
	n := h - 4
	var body []string
	if len(c.content) > 0 {
		// Only the content scrolls; the question stays last (tdp confirm).
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
	return popupBox{color: layer, glyph: glyphConfirm, title: popupTitle(c.ask.Title, c.ask.Subject), hint: c.hint(), scroll: scroll}.draw(w, body)
}

func (c *confirmView) key(a *App, key string, _ tea.KeyMsg) (bool, tea.Cmd) {
	switch key {
	case "Enter":
		return true, a.acceptConfirm(c)
	case "Esc":
		if c.quit != nil {
			return true, c.quit()
		}
		return false, nil
	case "Space":
		return true, nil
	case "j", "↓":
		c.top++
	case "k", "↑":
		c.top--
	default:
		return false, nil
	}
	return true, nil
}

func (c *confirmView) surface() string { return c.ask.Title }

func (c *confirmView) cancel() tea.Cmd { return run(c.ask.OnCancel) }

func (c *confirmView) keyRef() []refSection {
	secs := []refSection{{rows: []KeyRow{{Key: "Enter", Desc: c.ask.Verb}, {Key: "Esc", Desc: c.escVerb()}}}}
	if len(c.content) > c.visible {
		// Only when the details do not all fit (tdp M6).
		secs = append(secs, refSection{title: "move", rows: []KeyRow{{Key: "j|k", Desc: "scroll down, up"}}})
	}
	return secs
}

func (c *confirmView) typing() bool { return false }
