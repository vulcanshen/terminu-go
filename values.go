package terminu

import (
	"slices"

	tea "github.com/charmbracelet/bubbletea"
)

// Field is one value of a Values panel (tdp panel.12).
type Field struct {
	Label string
	// Value is the value as shown (tdp input.10);
	// Empty is what an empty value means, shown in grey ("not set").
	// A control character in it, ESC too, is drawn by its name (Escape).
	Value func() string
	// Styled, when set, is how the value is shown in colours of the app's own
	// (a swatch, ColorSpans; a state in Green): each piece's text drawn as
	// Value's is. Value stays what the value is.
	Styled func() []Span
	Empty  string
	// Path cuts a value too long from the front: "…/.ssh/id_ed25519".
	Path bool
	// Options make the field a radio drawn in place: Selected is the chosen
	// one. With Multi it is a group of checkboxes: Checked says which.
	Options  []string
	Selected int
	Multi    bool
	Checked  []bool
	// OptionUnavailable reports a checkbox option that is not available:
	// drawn in Surface2, the tick too, but stopped on and flipped like any
	// other (tdp checkbox).
	OptionUnavailable func(i int) bool
	Disabled          func() bool
	// Err marks the field's value as wrong: its label turns Red.
	Err func() bool
	// Edit is what Enter does on a field that is not drawn in place: open its
	// input popup.
	Edit func() tea.Cmd
	// OnChange runs when a radio or checkbox changes in place; it takes
	// effect at once (there is no submit).
	OnChange func() tea.Cmd
	// Desc, when set, says what the row means, for the "what each row is"
	// section first in the panel's key reference: the label is a word or
	// two (tdp panel, note). Whether a panel has the
	// section is the app's: it has it when a field has a Desc.
	Desc string
}

func (f *Field) disabled() bool { return f.Disabled != nil && f.Disabled() }

// checkGlyph is a checkbox's glyph: ticked or not.
func checkGlyph(on bool) string {
	if on {
		return glyphMarked
	}
	return glyphUnchecked
}
func (f *Field) hasErr() bool { return f.Err != nil && f.Err() }

// items is how many places the cursor stops at in the field: one per option
// drawn in place, one otherwise.
func (f *Field) items() int {
	if f.hasAll() {
		return len(f.Options) + 1 // Select all first
	}
	if len(f.Options) > 0 {
		return len(f.Options)
	}
	return 1
}

// hasAll reports a checkbox group with a Select all row: two options or more.
func (f *Field) hasAll() bool { return f.Multi && len(f.Options) >= 2 }

func (f *Field) allOn() bool { return allChecked(len(f.Options), f.Checked, nil) }

// Values is a panel body with one value per row: a settings screen
// (tdp panel). It moves like a form; a value takes effect when it is set.
type Values struct {
	Fields []*Field
	// Header, when set, is the table header over the labels and the values.
	Header [2]string
	// OnMove runs when the cursor comes to another place, by a key or not,
	// the first time included: the field, and the option in it (-1 for a
	// field not drawn in place, -2 for a checkbox group's Select all). It runs before the frame is drawn, so a
	// details panel follows in the same frame (tdp panel.13).
	OnMove func(f *Field, option int) tea.Cmd
	// EmptyFact and EmptyHint are what it says with no fields, in Overlay0, as
	// a list does (tdp screen.8).
	EmptyFact, EmptyHint string

	at      valueItem // where OnMove last ran
	cursor  int       // item index across all fields
	top     int
	visible int
	scroll  string
	cover   int // rows at the bottom under the inspect toast (Uncover)
}

type valueItem struct {
	field  *Field
	option int // -1 for the field itself
}

func (v *Values) itemList() []valueItem {
	var out []valueItem
	for _, f := range v.Fields {
		if len(f.Options) > 0 {
			if f.hasAll() {
				out = append(out, valueItem{f, optAll})
			}
			for i := range f.Options {
				out = append(out, valueItem{f, i})
			}
		} else {
			out = append(out, valueItem{f, -1})
		}
	}
	return out
}

// moved runs OnMove when the cursor is at another place than last time.
func (v *Values) moved() tea.Cmd {
	items := v.itemList()
	if v.cursor < 0 || v.cursor >= len(items) {
		v.at = valueItem{}
		return nil
	}
	it := items[v.cursor]
	if it == v.at {
		return nil
	}
	v.at = it
	if v.OnMove == nil {
		return nil
	}
	return v.OnMove(it.field, it.option)
}

// HasItem: a value panel's rows are its items.
func (v *Values) HasItem() bool { return len(v.Fields) > 0 }

// Empty reports whether there are no fields.
func (v *Values) Empty() bool { return len(v.Fields) == 0 }

// Current is the field under the cursor.
func (v *Values) Current() *Field {
	items := v.itemList()
	if v.cursor < 0 || v.cursor >= len(items) {
		return nil
	}
	return items[v.cursor].field
}

func (v *Values) labelWidth() int {
	w := 0
	for _, f := range v.Fields {
		w = max(w, Width(Upper(f.Label)))
	}
	if v.Header[0] != "" {
		w = max(w, Width(Upper(v.Header[0])))
	}
	return w
}

// Uncover keeps the cursor's item above the rows at the bottom of the fields
// that the inspect toast covers (tdp toast.4); the lib calls it before each
// Draw.
func (v *Values) Uncover(rows int) { v.cover = rows }

// Draw draws the fields: labels in upper case in Blue, two cells, the values.
func (v *Values) Draw(w, h int, focused bool) []string {
	if len(v.Fields) == 0 {
		v.scroll = ""
		return EmptyState(w, h, v.EmptyFact, v.EmptyHint)
	}
	items := v.itemList()
	v.fixCursor(items, 1)
	lw := v.labelWidth()
	arrow := Width(glyphHard)
	valW := w - 1 - lw - 2 - arrow
	var rows []string
	var itemRow []int // the row each item starts on
	if v.Header[0] != "" {
		// Dim text: a Blue header would run into the Blue labels under it
		// (tdp panel).
		rows = append(rows, " "+Fg(Overlay0).Render(Pad(Upper(v.Header[0]), lw)+"  "+Fit(Upper(v.Header[1]), valW))+Repeat(" ", arrow))
	}
	headerRows := len(rows)
	idx := 0
	for _, f := range v.Fields {
		labelColor := Blue
		switch {
		case f.hasErr():
			labelColor = Red
		case f.disabled():
			labelColor = Surface2
		}
		label := Fg(labelColor).Render(Pad(Upper(f.Label), lw))
		blank := Repeat(" ", lw)
		n := f.items()
		for k := range n {
			lead := blank
			if k == 0 {
				lead = label
			}
			itemRow = append(itemRow, len(rows))
			value := v.valueText(f, k)
			ind := "" // an option under Select all, further in, outside the block
			if f.hasAll() && k > 0 {
				ind = optIndent
			}
			if idx == v.cursor && !f.disabled() {
				fit := Fit
				if f.Path {
					fit = FitLeft
				}
				// A swatch keeps its colour on the block (tdp color);
				// an option's block starts at its glyph (tdp form).
				rows = append(rows, " "+lead+"  "+ind+onBlock(fit(value, valW-Width(ind)), On(Subtext1))+Fg(Subtext1).Render(glyphHard))
			} else {
				fit := Fit
				if f.Path {
					fit = FitLeft
				}
				rows = append(rows, " "+lead+"  "+ind+fit(value, valW-Width(ind))+Repeat(" ", arrow))
			}
			idx++
		}
	}
	// Scroll the least that keeps the cursor's item in sight, the header fixed.
	room := h - headerRows
	v.visible = room
	body := rows[headerRows:]
	focusRow := 0
	if v.cursor < len(itemRow) {
		focusRow = itemRow[v.cursor] - headerRows
	}
	v.top = scrollWindow(v.top, focusRow, max(room-v.cover, 1), len(body)) // above the inspect toast (tdp toast.4)
	out := append([]string(nil), rows[:headerRows]...)
	for i := v.top; i < v.top+room; i++ {
		if i < len(body) {
			out = append(out, body[i])
		} else {
			out = append(out, Repeat(" ", w))
		}
	}
	v.scroll = ""
	if len(body) > room {
		// Counted in fields, as a form's: a radio's options are one
		// (tdp panel.12).
		v.scroll = scrollText(true, slices.Index(v.Fields, items[v.cursor].field)+1, 0, len(v.Fields))
	}
	return Pad2(out, w, h)
}

// valueText is item k of f as shown.
func (v *Values) valueText(f *Field, k int) string {
	if f.hasAll() {
		if k == 0 {
			return selectAllText(f.allOn())
		}
		return v.optionText(f, k-1) // the options under Select all, drawn further in
	}
	return v.optionText(f, k)
}

// optionText is option k of f, or f's value when it has no options.
func (v *Values) optionText(f *Field, k int) string {
	switch {
	case len(f.Options) > 0 && f.Multi:
		on := k < len(f.Checked) && f.Checked[k]
		if f.OptionUnavailable != nil && f.OptionUnavailable(k) {
			return Fg(Surface2).Render(checkGlyph(on) + " " + Escape(f.Options[k]))
		}
		if on {
			return Fg(Green).Render(glyphMarked + " " + Escape(f.Options[k]))
		}
		return Fg(Text).Render(glyphUnchecked + " " + Escape(f.Options[k]))
	case len(f.Options) > 0:
		if k == f.Selected {
			return Fg(Green).Render(glyphRadioOn + " " + Escape(f.Options[k]))
		}
		return Fg(Text).Render(glyphRadioOff + " " + Escape(f.Options[k]))
	}
	if f.Styled != nil {
		if sp := f.Styled(); spansPlain(sp) != "" {
			return Fg(Text).Render(spansText(sp))
		}
		return Fg(Overlay0).Render(Escape(f.Empty))
	}
	val := ""
	if f.Value != nil {
		val = f.Value()
	}
	if val == "" {
		return Fg(Overlay0).Render(Escape(f.Empty))
	}
	if f.Path {
		return valueText(homeShort(val), Fg(Text))
	}
	return valueText(val, Fg(Text))
}

// fixCursor moves the cursor off a disabled item, going by dir.
func (v *Values) fixCursor(items []valueItem, dir int) {
	if len(items) == 0 {
		v.cursor = 0
		return
	}
	v.cursor = min(max(v.cursor, 0), len(items)-1)
	for range items {
		if !items[v.cursor].field.disabled() {
			return
		}
		v.cursor = (v.cursor + dir + len(items)) % len(items)
	}
}

// Scroll is "N of M" when the items do not all fit.
func (v *Values) Scroll() string { return v.scroll }

// MoveKeys are a one-dimensional list's (tdp K12); none with no fields,
// nothing to move to (tdp M6).
func (v *Values) MoveKeys() []KeyRow {
	if len(v.Fields) == 0 {
		return nil
	}
	return oneWayKeys()
}

// tabMoveKeys are its keys in a panel with tabs: h|l go to the tabs
// (tdp K12).
func (v *Values) tabMoveKeys() []KeyRow {
	if len(v.Fields) == 0 {
		return nil
	}
	return []KeyRow{
		{Key: "j|k", Desc: "next, previous"},
		{Key: "u|d", Desc: "half a page up, down"},
		{Key: "gg|G", Desc: "to the first, the last"},
	}
}

// Key moves item by item and does Enter (tdp panel.12).
func (v *Values) Key(key string, _ tea.KeyMsg) (bool, tea.Cmd) {
	items := v.itemList()
	n := len(items)
	if n == 0 {
		return false, nil
	}
	half := max(v.visible/2, 1)
	switch key {
	case "j", "l", "↓", "→":
		v.cursor = (v.cursor + 1) % n
		v.fixCursor(items, 1)
	case "k", "h", "↑", "←":
		v.cursor = (v.cursor - 1 + n) % n
		v.fixCursor(items, -1)
	case "d", "PgDn":
		v.cursor = min(v.cursor+half, n-1)
		v.fixCursor(items, -1)
	case "u", "PgUp":
		v.cursor = max(v.cursor-half, 0)
		v.fixCursor(items, 1)
	case "gg":
		v.cursor = 0
		v.fixCursor(items, 1)
	case "G":
		v.cursor = n - 1
		v.fixCursor(items, -1)
	case "Enter":
		it := items[v.cursor]
		f := it.field
		if f.disabled() {
			return true, nil
		}
		switch {
		case len(f.Options) > 0 && f.Multi && it.option == optAll:
			f.Checked = checkAll(len(f.Options), f.Checked, !f.allOn(), nil)
			return true, run(f.OnChange)
		case len(f.Options) > 0 && f.Multi:
			for len(f.Checked) < len(f.Options) {
				f.Checked = append(f.Checked, false)
			}
			f.Checked[it.option] = !f.Checked[it.option]
			return true, run(f.OnChange)
		case len(f.Options) > 0:
			f.Selected = it.option
			return true, run(f.OnChange)
		case f.Edit != nil:
			return true, f.Edit()
		}
		return true, nil
	default:
		return false, nil
	}
	return true, nil
}

// Actions are the panel's own item operation: what Enter does on the item
// under the cursor, so the Space menu and ? have it as the hint does (Rules
// M3, K6); in the disabled colour on a field that cannot be used now (tdp M6).
func (v *Values) Actions() []Action {
	items := v.itemList()
	if v.cursor >= len(items) {
		return nil
	}
	f := items[v.cursor].field
	var label, desc string
	switch {
	case len(f.Options) > 0 && f.Multi && items[v.cursor].option == optAll:
		label, desc = "Toggle", "every option"
	case len(f.Options) > 0 && f.Multi:
		label, desc = "Toggle", "this option"
	case len(f.Options) > 0:
		label, desc = "Choose", "this option"
	case f.Edit != nil:
		label, desc = "Edit", "this value"
	default:
		return nil
	}
	return []Action{{Key: "Enter", Label: label, Desc: desc, Scope: ItemScope, Disabled: f.disabled,
		Do: func() tea.Cmd { _, cmd := v.Key("Enter", tea.KeyMsg{Type: tea.KeyEnter}); return cmd }}}
}

// EnterHint is what Enter does on the item under the cursor, for the panel's
// hint: edit a value, choose a radio option, toggle a checkbox.
func (v *Values) EnterHint() string {
	items := v.itemList()
	if v.cursor >= len(items) {
		return ""
	}
	it := items[v.cursor]
	switch {
	case it.field.disabled():
		return ""
	case len(it.field.Options) > 0 && it.field.Multi:
		return "toggle"
	case len(it.field.Options) > 0:
		return "choose"
	case it.field.Edit != nil:
		return "edit"
	}
	return ""
}
