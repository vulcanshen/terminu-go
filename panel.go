package terminu

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Body is what is inside a panel's frame. The lib has List, Values and Text;
// an app that draws its own content implements Body itself.
type Body interface {
	// Draw returns h lines of exactly w cells: the inside of the frame.
	Draw(w, h int, focused bool) []string
	// Key handles a key the panel did not take; it reports whether it did
	// something with it. Movement keys arrive here.
	Key(key string, msg tea.KeyMsg) (bool, tea.Cmd)
	// HasItem reports whether the cursor is on an item, so that item
	// operations apply (Rules M2).
	HasItem() bool
	// Scroll is the scroll position for the bottom border after the last
	// Draw ("3 of 40", "1-20 of 120"), "" when everything fits.
	Scroll() string
	// MoveKeys are the movement keys that work here, for the key
	// reference's move section (Rules K6).
	MoveKeys() []KeyRow
}

// KeyRow is one row of a key reference.
type KeyRow struct {
	Key, Desc string
	Disabled  bool
}

// Mode is a temporary state of a panel in which some keys mean something of
// their own and Esc leaves (Rules K11).
type Mode struct {
	Name string // shown at the right of the top border: one word
	// Keys are the mode's own keys: the only ones that work besides the
	// core keys of K11. They are listed in its key reference and the footer.
	Keys []ModeKey
	// Exit is the verb Esc does in the footer ("cancel" for Esc:cancel).
	Exit string
	// Pause suspends Tab and the digits in the mode; pressing them then says
	// to leave the mode first.
	Pause bool
	// OnExit runs when Esc leaves the mode, to go back to the state before it.
	OnExit func() tea.Cmd
}

// ModeKey is one of a mode's keys.
type ModeKey struct {
	// Key is how it is written in the footer and the key reference ("j/k");
	// Keys are the keys it takes, Key itself when empty.
	Key  string
	Keys []string
	Desc string
	// NoFooter keeps it out of the footer (movement keys): it stays in ?.
	NoFooter bool
	// Do runs it, with the key pressed.
	Do func(key string) tea.Cmd
}

func (k ModeKey) takes(key string) bool {
	if len(k.Keys) == 0 {
		return k.Key == key
	}
	for _, x := range k.Keys {
		if x == key {
			return true
		}
	}
	return false
}

// Failure is what a panel shows when its content could not be had (layout/panel,
// loading and errors): a fact in Red and a hint naming a key.
type Failure struct {
	Fact, Hint string
}

// Panel is a region of a screen with its own cursor and content (Principle
// "Panel"). It is numbered by its place in its screen, from [1].
type Panel struct {
	// Label is the panel's name in its [N] capsule.
	Label string
	// Tabs are the panel's other tabs, after the label's own (for now the
	// label is the first tab, G23). Tab is the current one, 0 being the
	// label's; h/l switch tabs and call OnTab.
	Tabs  []string
	Tab   int
	OnTab func(tab int) tea.Cmd
	Body  Body
	// Actions are the panel's item and panel operations; a "Enter" key is
	// what Enter does here (Rules K3).
	Actions []Action
	// Stream marks streaming content, which is not dimmed when unfocused
	// (Rules T2).
	Stream bool
	// Loading puts the loading icon after the capsule chain; LoadingText is
	// what an empty panel says meanwhile ("loading pods…").
	Loading     bool
	LoadingText string
	// Fail, when set and the panel has nothing to show, is shown in its middle.
	Fail *Failure
	// OnEsc is the app's "one level up" for Esc on this panel (Rules K4); it
	// reports whether there was one.
	OnEsc func() tea.Cmd

	mode *Mode
	app  *App
}

// EnterMode puts the panel in m; Esc leaves it (Rules K11).
func (p *Panel) EnterMode(m *Mode) { p.mode = m }

// LeaveMode leaves the mode without running its OnExit.
func (p *Panel) LeaveMode() { p.mode = nil }

// InMode reports the mode the panel is in, or nil.
func (p *Panel) InMode() *Mode { return p.mode }

// typer is a body with a typing row (the panel filter).
type typer interface {
	Typing() bool
}

func (p *Panel) typing() bool {
	t, ok := p.Body.(typer)
	return ok && t.Typing()
}

// enterAction is the action Enter runs here, if any.
func (p *Panel) enterAction() *Action {
	for i := range p.Actions {
		if p.Actions[i].Key == "Enter" && !p.Actions[i].hidden() && p.scopeApplies(p.Actions[i]) {
			return &p.Actions[i]
		}
	}
	return nil
}

func (p *Panel) scopeApplies(a Action) bool {
	return a.Scope != ItemScope || (p.Body != nil && p.Body.HasItem())
}

// hint is the panel's bottom-border hint: what Enter does, then the actions
// that ask to be in it (layout/panel). None while typing in its filter.
func (p *Panel) hint() []KeyHint {
	if p.typing() {
		return nil
	}
	var out []KeyHint
	if a := p.enterAction(); a != nil && a.Hint != "" && !a.disabled() {
		out = append(out, KeyHint{"Enter", a.Hint})
	} else if h, ok := p.Body.(interface{ EnterHint() string }); ok && h.EnterHint() != "" {
		out = append(out, KeyHint{"Enter", h.EnterHint()})
	}
	for _, a := range p.Actions {
		if a.Key == "Enter" || a.Hint == "" || a.hidden() || a.disabled() || !p.scopeApplies(a) {
			continue
		}
		out = append(out, KeyHint{a.Key, a.Hint})
	}
	return out
}

// capsules are the panel's [N] label and tabs (layout/panel, capsule chain).
func (p *Panel) capsules(n int) []Capsule {
	caps := []Capsule{{Text: "[" + itoa(n) + "] " + Upper(p.Label), Selected: true}}
	for i, t := range p.Tabs {
		caps = append(caps, Capsule{Text: Upper(t), Selected: p.Tab == i+1})
	}
	return caps
}

// draw draws the panel, frame and all, in w × h cells.
func (p *Panel) draw(n, w, h int, focused bool) []string {
	if w < 2 || h < 2 {
		return blankLines(w, h)
	}
	lines, color := lightLines, Surface2
	if focused {
		lines, color = heavyLines, Blue
	}
	mode := ""
	if p.mode != nil {
		color, mode = Yellow, p.mode.Name
	}
	b := box{width: w, lines: lines, color: color, hintKey: Fg(Blue)}
	b.head = drawChain(p.capsules(n), color, max(w-2-modeRoom(mode), 1))
	if p.Loading {
		b.after = Fg(Text).Render(loadingFrame())
	}
	b.mode = mode
	inner, ih := w-2, h-2
	var body []string
	switch {
	case p.Body == nil:
		body = blankLines(inner, ih)
	default:
		body = p.Body.Draw(inner, ih, focused)
	}
	if empty, ok := p.Body.(interface{ Empty() bool }); ok && empty.Empty() {
		switch {
		case p.Loading && p.LoadingText != "":
			body = emptyState(inner, ih, Fg(Overlay0).Render(loadingFrame()+" "+p.LoadingText), "")
		case p.Fail != nil:
			body = emptyState(inner, ih, Fg(Red).Render(p.Fail.Fact), Fg(Overlay0).Render(bracketKeys(p.Fail.Hint)))
		}
	}
	b.body = body
	b.hint = p.hint()
	if p.Body != nil {
		b.scroll = p.Body.Scroll()
	}
	if !focused && !p.Stream {
		b.body = DimLines(b.body)
		b.hintKey = Fg(DimColor(Blue))
		b.hintWords = DimColor(Overlay0)
	}
	return b.draw()
}

// modeRoom is how much of the top border a mode name takes.
func modeRoom(mode string) int {
	if mode == "" {
		return 0
	}
	return Width(mode) + 3
}

func blankLines(w, h int) []string {
	out := make([]string, max(h, 0))
	for i := range out {
		out[i] = Repeat(" ", w)
	}
	return out
}

// emptyState is a sentence and a hint centred in w × h (layout/screen, empty
// states), with a blank line between them.
func emptyState(w, h int, fact, hint string) []string {
	out := blankLines(w, h)
	lines := []string{fact}
	if hint != "" {
		lines = append(lines, "", hint)
	}
	top := max((h-len(lines))/2, 0)
	for i, l := range lines {
		if top+i < h {
			out[top+i] = Center(Truncate(l, w), w)
		}
	}
	return out
}

// EmptyState draws an empty panel's two sentences, the fact and the hint
// naming keys, both in Overlay0 (layout/screen); keys written as [A] in the
// hint stay as they are.
func EmptyState(w, h int, fact, hint string) []string {
	return emptyState(w, h, Fg(Overlay0).Render(fact), Fg(Overlay0).Render(hint))
}

// bracketKeys leaves a sentence as it is: keys in sentences are already
// written in brackets by whoever wrote it (Rules M5).
func bracketKeys(s string) string { return strings.TrimSpace(s) }
