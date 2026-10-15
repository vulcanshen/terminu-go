package terminu

import (
	"slices"

	tea "github.com/charmbracelet/bubbletea"
)

// Reorder puts a set of items in an order by hand (components/input/reorder):
// the panel's Drag mode in a popup. D picks up the item under the cursor,
// j|k move it, Enter drops it; Enter with nothing held writes the order back.
type Reorder struct {
	Title, Glyph string
	// Subject is the one thing it acts on, kept as it is after ": " in the
	// title: "EDIT: filu" (components/layout/popup).
	Subject string
	Options []string
	// Done gets the new order: the options' indexes as they now stand
	// ([2 0 1]: the third first). It is not called when the order did not
	// change.
	Done func(order []int) tea.Cmd
}

func (r Reorder) typed() bool { return false }

// Reorder opens a reorder popup.
func (a *App) Reorder(r Reorder) tea.Cmd { return r.open(a, nil) }

type reorderView struct {
	r       Reorder
	order   []int // the options' indexes as they now stand
	cursor  int   // index into order
	top     int
	visible int
	// held is set in the Drag mode; from and before are where the held item
	// was picked up and the order then, for Esc to put it back.
	held    bool
	from    int
	before  []int
	changed bool // an item was dropped somewhere else, never cleared
	written func()
}

func (r Reorder) open(a *App, written func()) tea.Cmd {
	rv := &reorderView{r: r, written: written}
	for i := range r.Options {
		rv.order = append(rv.order, i)
	}
	return a.open(rv)
}

// The hints: nothing held, holding one (components/input/reorder).
var (
	reorderHint     = []KeyHint{{"Enter", "apply"}, {"D", "drag"}, {"Esc", "cancel"}}
	reorderHeldHint = []KeyHint{{"Enter", "drop"}, {"Esc", "cancel"}}
)

func (rv *reorderView) hint() []KeyHint {
	if rv.held {
		return reorderHeldHint
	}
	return reorderHint
}

// lead is the cell at the start of every row for the held item's glyph and
// the space after it.
func (rv *reorderView) lead() int { return IconWidth() + 1 }

func (rv *reorderView) measure(sw, sh int) (int, int) {
	widest := 0
	for _, o := range rv.r.Options {
		widest = max(widest, Width(o))
	}
	// The left cell, the glyph's two, the item, the arrow, the frame.
	need := 1 + rv.lead() + widest + Width(glyphHard) + 2
	need = max(need, Width(titleCapsule(rv.r.Glyph, popupTitle(rv.r.Title, rv.r.Subject), Base, 1<<20))+4,
		hintWidth(reorderHint)+6, hintWidth(reorderHeldHint)+6)
	h := min(len(rv.r.Options)+4, max(sh-2, 3))
	if len(rv.r.Options) > h-4 {
		n := len(rv.r.Options)
		need = max(need, scrollRoom(reorderHint, scrollText(true, n, 0, n)))
	}
	return min(need, max(min(sw-2, 120), 1)), h
}

func (rv *reorderView) draw(w, h int, layer Color) []string {
	inner := w - 2
	n := h - 4
	rv.visible = n
	rv.top = scrollWindow(rv.top, rv.cursor, n, len(rv.order))
	arrow := Width(glyphHard)
	body := make([]string, 0, n)
	for i := rv.top; i < rv.top+n && i < len(rv.order); i++ {
		item := rv.r.Options[rv.order[i]]
		switch {
		case i == rv.cursor && rv.held:
			// The held item: the cursor in the mode's colour, the swap glyph
			// at its start (Rules K11).
			body = append(body, cursorRow(Pad(glyphSwap, rv.lead())+item, inner, On(Yellow)))
		case i == rv.cursor:
			body = append(body, cursorRow(Repeat(" ", rv.lead())+item, inner, On(layer)))
		default:
			body = append(body, " "+Repeat(" ", rv.lead())+Fit(Fg(Text).Render(item), inner-1-arrow-rv.lead())+Repeat(" ", arrow))
		}
	}
	for len(body) < n {
		body = append(body, Repeat(" ", inner))
	}
	scroll := ""
	if len(rv.order) > n {
		scroll = scrollText(true, rv.cursor+1, 0, len(rv.order))
	}
	box := popupBox{color: layer, glyph: rv.r.Glyph, title: popupTitle(rv.r.Title, rv.r.Subject), hint: rv.hint(), scroll: scroll}
	if rv.held {
		// The Drag mode marks its frame: the mode colour and its name (Rules K11).
		box.color, box.mode = Yellow, "Drag"
	}
	return box.draw(w, body)
}

func (rv *reorderView) key(a *App, key string, _ tea.KeyMsg) (bool, tea.Cmd) {
	if rv.held {
		return rv.dragKey(key)
	}
	n := len(rv.order)
	if n == 0 {
		switch key {
		case "Enter":
			return true, a.closeTop()
		case "Space", "D":
			return true, nil
		}
		return false, nil
	}
	half := max(rv.visible/2, 1)
	switch key {
	case "j", "↓":
		rv.cursor = (rv.cursor + 1) % n
	case "k", "↑":
		rv.cursor = (rv.cursor + n - 1) % n
	case "d", "PgDn":
		rv.cursor = min(rv.cursor+half, n-1)
	case "u", "PgUp":
		rv.cursor = max(rv.cursor-half, 0)
	case "gg":
		rv.cursor = 0
	case "G":
		rv.cursor = n - 1
	case "D":
		rv.held, rv.from, rv.before = true, rv.cursor, slices.Clone(rv.order)
	case "Enter":
		if !rv.moved() || rv.r.Done == nil {
			return true, a.closeTop()
		}
		return true, tea.Batch(a.closeTop(), then(rv.written, rv.r.Done(slices.Clone(rv.order))))
	case "Esc":
		if !rv.changed {
			return true, a.closeTop()
		}
		return true, a.Confirm(Ask{Title: "Discard", Question: "Discard changes to " + rv.r.Title + "?", Verb: "discard"},
			func() tea.Cmd { return a.closeTop() })
	case "Space":
		// Space does nothing here (Rules K5): D picks up.
	default:
		return false, nil
	}
	return true, nil
}

// dragKey is a key in the Drag mode: only its own keys and the core keys of
// Rules K11 work.
func (rv *reorderView) dragKey(key string) (bool, tea.Cmd) {
	switch key {
	case "?", "q", "Ctrl-C":
		return false, nil // the mode's key reference, the quit flow (Rules K11)
	case "j", "↓":
		rv.cursor = moveWrap(rv.order, rv.cursor, true)
	case "k", "↑":
		rv.cursor = moveWrap(rv.order, rv.cursor, false)
	case "Enter":
		rv.held = false
		if rv.cursor != rv.from {
			rv.changed = true
		}
	case "Esc":
		// Back to where it was picked up, one layer at a time (Rules K4).
		rv.held, rv.order, rv.cursor = false, rv.before, rv.from
	}
	return true, nil
}

// moved reports an order other than the one it opened with.
func (rv *reorderView) moved() bool {
	for i, o := range rv.order {
		if o != i {
			return true
		}
	}
	return false
}

// moveWrap moves s[i] one place down or up and returns where it went. Past
// the ends it wraps: the last moved down goes first and the others move down
// one; the first moved up goes last (Rules K12).
func moveWrap[T any](s []T, i int, down bool) int {
	n := len(s)
	switch {
	case n < 2:
		return i
	case down && i == n-1:
		v := s[i]
		copy(s[1:], s[:n-1])
		s[0] = v
		return 0
	case !down && i == 0:
		v := s[0]
		copy(s, s[1:])
		s[n-1] = v
		return n - 1
	case down:
		s[i], s[i+1] = s[i+1], s[i]
		return i + 1
	default:
		s[i], s[i-1] = s[i-1], s[i]
		return i - 1
	}
}

func (rv *reorderView) surface() string {
	if rv.held {
		return "Drag"
	}
	return rv.r.Title
}

func (rv *reorderView) keyRef() []refSection {
	if rv.held {
		// Only the mode's keys (Rules K11).
		return []refSection{{title: "Drag", rows: []KeyRow{
			{Key: "j|k", Desc: "move it down, up"},
			{Key: "Enter", Desc: "drop it"},
			{Key: "Esc", Desc: "put it back"},
		}}}
	}
	return []refSection{
		{rows: []KeyRow{
			{Key: "Enter", Desc: "apply the order"},
			{Key: "D", Desc: "pick up the item to move it (Drag)"},
			{Key: "Esc", Desc: "cancel (asks first when the order changed)"},
		}},
		{title: "move", rows: []KeyRow{
			{Key: "j|k", Desc: "next, previous"},
			{Key: "u|d", Desc: "half a page up, down"},
			{Key: "gg|G", Desc: "to the first, the last"},
		}},
	}
}

func (rv *reorderView) typing() bool { return false }

// inMode reports the Drag mode, where the app's own core keys do not work
// (Rules K11).
func (rv *reorderView) inMode() bool { return rv.held }
