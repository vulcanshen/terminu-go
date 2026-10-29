package terminu

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Body is what is inside a panel's frame. The lib has List, Values, Pager,
// FilePreview and Terminal; an app that draws its own content implements Body
// itself. A body with an Empty() bool method that reports true has the
// panel's LoadingText or Fail in its middle instead, whatever the kind of
// panel (tdp panel.15); the lib's all have it, and a panel with no Body is
// empty too. A body with an Uncover(rows int) method is told before each Draw
// how many rows at the bottom of its inside the inspect toast covers, 0 for
// none (tdp toast.4): it scrolls so that the item under its cursor shows above
// them, as List and Values do.
type Body interface {
	// Draw returns h lines of exactly w cells: the inside of the frame.
	Draw(w, h int, focused bool) []string
	// Key handles a key the panel did not take; it reports whether it did
	// something with it. Movement keys arrive here.
	Key(key string, msg tea.KeyMsg) (bool, tea.Cmd)
	// HasItem reports whether the cursor is on an item, so that item
	// operations apply (tdp M2).
	HasItem() bool
	// Scroll is the scroll position for the bottom border after the last
	// Draw ("3 of 40", "1-20 of 120"), "" when everything fits.
	Scroll() string
	// MoveKeys are the movement keys that work here, for the key
	// reference's move section (tdp K6).
	MoveKeys() []KeyRow
}

// KeyRow is one row of a key reference.
type KeyRow struct {
	Key, Desc string
	Disabled  bool
}

// KeySection is a titled section of a key reference ("terminal").
type KeySection struct {
	Title string
	Rows  []KeyRow
}

// Mode is a temporary state of a panel in which some keys mean something of
// their own and Esc leaves (tdp K11).
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
	// Inspect is the inspect toast of the item under the cursor in the mode,
	// as Panel.Inspect: while the panel is in the mode it is asked instead of
	// the panel's, an inner layer's before an outer one's.
	Inspect func() []string
}

// ModeKey is one of a mode's keys.
type ModeKey struct {
	// Key is how it is written in the footer and the key reference ("j|k");
	// Keys are the keys it takes, Key itself when empty.
	Key  string
	Keys []string
	Desc string
	// NoFooter keeps it out of the footer (movement keys): it stays in ?.
	NoFooter bool
	// Do runs it, with the key pressed.
	Do func(key string) tea.Cmd
	// Disabled reports that it cannot be pressed now: in the disabled colour
	// in the footer and the key reference, pressing it does nothing (tdp M6).
	Disabled func() bool
	// In the mode's Space menu (tdp menu.4): Item puts
	// it under ITEM OPERATION, a key acting on the item under the cursor or
	// the range selected; the others go under the mode's name, a NoFooter one
	// (a movement key) only shown. Label names its row, its hotkey marked
	// (Compare: [C]ompare), Desc capitalised when empty; MenuDesc describes it.
	Item     bool
	Label    string
	MenuDesc string
}

func (k ModeKey) off() bool { return k.Disabled != nil && k.Disabled() }

// first is the key the mode's Space menu presses for it.
func (k ModeKey) first() string {
	if len(k.Keys) > 0 {
		return k.Keys[0]
	}
	return strings.Split(k.Key, "|")[0]
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

// Failure is what a panel shows when its content could not be had (tdp panel.15):
// a fact in Red and a hint naming a key.
type Failure struct {
	Fact, Hint string
}

// Panel is a region of a screen with its own cursor and content (Principle
// "Panel"). It is numbered by its place in its screen, from [1].
type Panel struct {
	labelled // its mode's name as drawn (modelabel.go)
	// Label is the panel's name after its [N]. With tabs it is the first
	// tab's name: a panel has no other name (tdp panel.4).
	Label string
	// Name, when set, is the name of the one thing the panel shows, kept as it
	// is after the label, upper case, and ": " ([2] PODS: nginx-7d4f); with no
	// Label it is the whole label ([1] prod-web-01). The Space menu and the
	// key reference write it so too (components, upper case).
	Name string
	// Tabs are the panel's other tabs, after the label's own. Tab is the
	// current one, 0 being the label's; h/l switch tabs and call OnTab.
	Tabs  []string
	Tab   int
	OnTab func(tab int) tea.Cmd
	Body  Body
	// Actions are the panel's item and panel operations; a "Enter" key is
	// what Enter does here (tdp K3).
	Actions []Action
	// Details is the panel that shows the item under this panel's cursor
	// (tdp panel.13): the "Enter" action may send the
	// focus there (Action.ToDetails).
	Details *Panel
	// Stream marks streaming content, which is not dimmed when unfocused
	// (tdp T2).
	Stream bool
	// Loading puts the loading icon after the capsule chain; LoadingText is
	// what an empty panel says meanwhile ("loading pods…", tdp panel.15).
	Loading     bool
	LoadingText string
	// Fail, when set and the panel has nothing to show, is shown in its middle.
	Fail *Failure
	// OnEsc is the app's "one level up" for Esc on this panel (tdp K4); it
	// reports whether there was one.
	OnEsc func() tea.Cmd
	// HintPressableOnly leaves what cannot run now out of the bottom-border
	// hint (tdp M6: the app's choice); otherwise it is listed in the
	// disabled colour, as in the Space menu and ?.
	HintPressableOnly bool
	// OtherKeys are sections of its ? about another surface that has no key
	// reference of its own (tdp M6: a PTY, a grid), after the panel's own.
	OtherKeys []KeySection
	// Inspect, when set, is what the item under the panel's cursor has to show
	// in an inspect toast (tdp toast.4), nil for an item with nothing: each
	// string a row of its own, as Toast's. The lib asks it after every message
	// while the panel has the focus and shows the toast, with no timer and no
	// key of its own, every key going through it; it goes when the cursor
	// leaves the item, and an ordinary toast replaces it until that one is
	// gone. Which items have one and what it says are the app's.
	Inspect func() []string

	mode  *Mode
	under []*Mode // the layers mode is in, outermost first
	app   *App
}

// EnterMode puts the panel in m; Esc leaves it (tdp K11).
func (p *Panel) EnterMode(m *Mode) { p.mode, p.under = m, nil }

// EnterLayer goes a layer further into the mode the panel is in, to m (tdp
// K11): its name follows the mode's after ">" (Visual>select), its keys and
// Exit are the ones that work, and Esc leaves m alone, running its OnExit,
// back to the layer it was in. Out of a mode it is EnterMode.
func (p *Panel) EnterLayer(m *Mode) {
	if p.mode != nil {
		p.under = append(p.under, p.mode)
	}
	p.mode = m
}

// LeaveMode leaves the whole mode, every layer, without running an OnExit.
func (p *Panel) LeaveMode() { p.mode, p.under = nil, nil }

// InMode reports the mode the panel is in, its innermost layer, or nil.
func (p *Panel) InMode() *Mode { return p.mode }

// modeLayers is the names of the mode's layers, outermost first.
func (p *Panel) modeLayers() []string {
	var out []string
	for _, m := range p.under {
		out = append(out, m.Name)
	}
	return append(out, p.mode.Name)
}

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
// that ask to be in it (tdp panel); off marks the ones that cannot run now
// (tdp M6). None while typing in its filter.
func (p *Panel) hint() (out []KeyHint, off []bool) {
	if p.typing() {
		return nil, nil
	}
	// In its PTY the panel's keys go to the program: no hint, as on a filter's
	// typing row; the footer has the exit key (tdp panel).
	if t, ok := p.Body.(*Terminal); ok && t.inside {
		return nil, nil
	}
	add := func(key, desc string, disabled bool) {
		if disabled && p.HintPressableOnly {
			return
		}
		out = append(out, KeyHint{key, desc})
		off = append(off, disabled)
	}
	if a := p.enterAction(); a != nil && a.Hint != "" && (!a.disabled() || !p.HintPressableOnly) {
		add("Enter", a.Hint, a.disabled())
	} else if h, ok := p.Body.(interface{ EnterHint() string }); ok && h.EnterHint() != "" {
		add("Enter", h.EnterHint(), false)
	}
	// h|l switch tabs: written after Enter while they do, two tabs or more
	// and no mode (tdp panel.1).
	if len(p.Tabs) > 0 && p.mode == nil {
		add("h|l", "tab", false)
	}
	for _, a := range p.Actions {
		if a.Key == "Enter" || a.Hint == "" || a.hidden() || !p.scopeApplies(a) {
			continue
		}
		add(a.Key, a.Hint, a.disabled())
	}
	return out, off
}

// labelText is the label as the title writes it: upper case, the name after
// it as it is.
func (p *Panel) labelText() string {
	switch {
	case p.Name == "":
		return Upper(p.Label)
	case p.Label == "":
		return p.Name
	}
	return Upper(p.Label) + ": " + p.Name
}

// title is "[N] label", as the Space menu and the key reference write it.
func (p *Panel) title(n int) string { return "[" + itoa(n) + "] " + p.labelText() }

// capsules are the panel's [N] and its tabs (tdp panel.4):
// [N] is a part of its own, always filled; of the tabs only the current one
// is. On the first tab [N] and it meet with blank as wide as a join, no join
// drawn, so the text does not move when the tab changes: "[2] PODS", as a
// panel without tabs looks.
func (p *Panel) capsules(n int) []Capsule {
	num := "[" + itoa(n) + "]"
	if len(p.Tabs) == 0 {
		return []Capsule{{Text: num + " " + p.labelText(), Selected: true}}
	}
	if p.Tab == 0 {
		caps := []Capsule{{Text: num + Repeat(" ", PowerlineWidth()) + p.labelText(), Selected: true}}
		for _, t := range p.Tabs {
			caps = append(caps, Capsule{Text: Upper(t)})
		}
		return caps
	}
	caps := []Capsule{{Text: num, Selected: true}, {Text: p.labelText()}}
	for i, t := range p.Tabs {
		caps = append(caps, Capsule{Text: Upper(t), Selected: p.Tab == i+1})
	}
	return caps
}

// draw draws the panel, frame and all, in w × h cells; pointed, the focused
// list's cursor row stands for it (List.Points).
func (p *Panel) draw(n, w, h int, focused, pointed bool) []string {
	if w < 2 || h < 2 {
		return blankLines(w, h)
	}
	lines, color := lightLines, Surface2
	switch {
	case focused:
		lines, color = heavyLines, Blue
	case pointed: // only the frame's colour changes, its capsules' with it
		color = Lavender
	}
	if p.modeName() != "" {
		color = Yellow
	}
	lab := p.labelFor(p.modeName())
	mode := lab.name // still there while it closes
	b := box{width: w, lines: lines, color: color, hintKey: Fg(Blue)}
	// A line segment stays before the top-right corner (or the mode's name).
	b.head = drawChain(p.capsules(n), color, max(w-3-modeRoom(mode, lab.room), 1))
	if p.Loading {
		b.after = Fg(color).Render(loadingFrame()) // the frame's colour (tdp popup)
	}
	b.mode, b.modeClosed = mode, 1-lab.shown
	inner, ih := w-2, h-2
	var body []string
	switch {
	case p.Body == nil:
		body = blankLines(inner, ih)
	default:
		body = p.Body.Draw(inner, ih, focused)
	}
	empty := p.Body == nil // every kind of panel (tdp panel.15)
	if e, ok := p.Body.(interface{ Empty() bool }); ok {
		empty = e.Empty()
	}
	if empty {
		switch {
		case p.Loading && p.LoadingText != "":
			body = emptyState(inner, ih, loadingFrame()+" "+p.LoadingText, Fg(Overlay0), "", Style{})
		case p.Fail != nil:
			body = emptyState(inner, ih, p.Fail.Fact, Fg(Red), bracketKeys(p.Fail.Hint), Fg(Overlay0))
		}
	}
	b.body = body
	b.hint, b.hintOff = p.hint()
	if p.Body != nil {
		b.scroll = p.Body.Scroll()
	}
	_, pty := p.Body.(*Terminal)
	if !focused && !p.Stream && !pty { // a program's screen is live, like a stream (tdp T2)
		b.body = DimLines(b.body)
	}
	if !focused {
		// The hint dims on every panel without the focus, a stream's too: it
		// is not what streams (tdp T2).
		b.hintKey = Fg(DimColor(Blue))
		b.hintWords = DimColor(Overlay0)
	}
	return b.draw()
}

// modeName is the mode the panel is in now: its own, or its PTY's (Rules
// K11).
func (p *Panel) modeName() string {
	if p.mode != nil {
		return strings.Join(p.modeLayers(), ">")
	}
	if t, ok := p.Body.(*Terminal); ok {
		return t.modeName()
	}
	return ""
}

// modeRoom is how much of the top border a mode name takes, in room cells
// of name at least (the longest name while a layer grows).
func modeRoom(mode string, room int) int {
	if mode == "" {
		return 0
	}
	return max(Width(mode), room) + 3
}

func blankLines(w, h int) []string {
	out := make([]string, max(h, 0))
	for i := range out {
		out[i] = Repeat(" ", w)
	}
	return out
}

// emptyState is a sentence and a hint centred in w × h (tdp screen.8
// states), with a blank line between them. Too long, each folds at its spaces
// (components, wrapping), every line centred one cell from the frame: never
// cut.
func emptyState(w, h int, fact string, factStyle Style, hint string, hintStyle Style) []string {
	out := blankLines(w, h)
	room := max(w-2, 1)
	var lines []string
	for _, l := range Wrap(Escape(fact), room) {
		lines = append(lines, factStyle.Render(l))
	}
	if hint != "" {
		lines = append(lines, "")
		for _, l := range Wrap(hint, room) {
			lines = append(lines, hintStyle.Render(l))
		}
	}
	top := max((h-len(lines))/2, 0)
	for i, l := range lines {
		if top+i < h {
			out[top+i] = Center(l, w) // folded at w-2: a cell either side at least
		}
	}
	return out
}

// EmptyState draws an empty panel's two sentences, the fact and the hint
// naming keys, both in Overlay0 (tdp screen); keys written as [A] in the
// hint stay as they are.
func EmptyState(w, h int, fact, hint string) []string {
	return emptyState(w, h, fact, Fg(Overlay0), hint, Fg(Overlay0))
}

// bracketKeys leaves a sentence as it is: keys in sentences are already
// written in brackets by whoever wrote it (tdp M5).
func bracketKeys(s string) string { return strings.TrimSpace(s) }
