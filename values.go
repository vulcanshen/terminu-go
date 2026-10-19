package terminu

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Field is one value of a Values panel (layout/panel, one value per row).
type Field struct {
	Label string
	// Value is the value as shown (components/input/README, how a value is
	// shown); Empty is what an empty value means, shown in grey ("not set").
	Value func() string
	Empty string
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
	// other (components/input/checkbox).
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
	if len(f.Options) > 0 {
		return len(f.Options)
	}
	return 1
}

// Values is a panel body with one value per row: a settings screen
// (layout/panel). It moves like a form; a value takes effect when it is set.
type Values struct {
	Fields []*Field
	// Header, when set, is the table header over the labels and the values.
	Header [2]string
	// OnMove runs when the cursor comes to another place, by a key or not,
	// the first time included: the field, and the option in it (-1 for a
	// field not drawn in place). It runs before the frame is drawn, so a
	// details panel follows in the same frame (layout/panel, details).
	OnMove func(f *Field, option int) tea.Cmd
	// EmptyFact and EmptyHint are what it says with no fields, in Overlay0, as
	// a list does (layout/screen, the empty state).
	EmptyFact, EmptyHint string

	at      valueItem // where OnMove last ran
	cursor  int       // item index across all fields
	top     int
	visible int
	scroll  string
}

type valueItem struct {
	field  *Field
	option int // -1 for the field itself
}

func (v *Values) itemList() []valueItem {
	var out []valueItem
	for _, f := range v.Fields {
		if len(f.Options) > 0 {
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
		// (components/layout/panel).
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
			if idx == v.cursor && !f.disabled() {
				fit := Fit
				if f.Path {
					fit = FitLeft
				}
				// A swatch keeps its colour on the block (components/color).
				rows = append(rows, " "+lead+"  "+onBlock(fit(value, valW), On(Subtext1))+Fg(Subtext1).Render(glyphHard))
			} else {
				fit := Fit
				if f.Path {
					fit = FitLeft
				}
				rows = append(rows, " "+lead+"  "+fit(value, valW)+Repeat(" ", arrow))
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
	v.top = scrollWindow(v.top, focusRow, max(room, 1), len(body))
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
		v.scroll = scrollText(true, v.cursor+1, 0, len(items))
	}
	return Pad2(out, w, h)
}

// valueText is item k of f as shown.
func (v *Values) valueText(f *Field, k int) string {
	switch {
	case len(f.Options) > 0 && f.Multi:
		on := k < len(f.Checked) && f.Checked[k]
		if f.OptionUnavailable != nil && f.OptionUnavailable(k) {
			return Fg(Surface2).Render(checkGlyph(on) + " " + f.Options[k])
		}
		if on {
			return Fg(Green).Render(glyphMarked + " " + f.Options[k])
		}
		return Fg(Text).Render(glyphUnchecked + " " + f.Options[k])
	case len(f.Options) > 0:
		if k == f.Selected {
			return Fg(Green).Render(glyphRadioOn + " " + f.Options[k])
		}
		return Fg(Text).Render(glyphRadioOff + " " + f.Options[k])
	}
	val := ""
	if f.Value != nil {
		val = f.Value()
	}
	if val == "" {
		return Fg(Overlay0).Render(f.Empty)
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

// MoveKeys are a one-dimensional list's (Rules K12).
func (v *Values) MoveKeys() []KeyRow {
	return []KeyRow{
		{Key: "j|k", Desc: "next, previous"},
		{Key: "u|d", Desc: "half a page up, down"},
		{Key: "gg|G", Desc: "to the first, the last"},
	}
}

// Key moves item by item and does Enter (layout/panel, one value per row).
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
// M3, K6); in the disabled colour on a field that cannot be used now (M6).
func (v *Values) Actions() []Action {
	items := v.itemList()
	if v.cursor >= len(items) {
		return nil
	}
	f := items[v.cursor].field
	var label, desc string
	switch {
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
