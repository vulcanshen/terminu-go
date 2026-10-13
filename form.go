package terminu

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// FormField is one field of a Form (components/dialog/form).
type FormField struct {
	Label string
	// Stacked puts the label on a row of its own and the value under it, as
	// wide as the inside: for long values (a URL, a path, a command).
	Stacked bool
	// Value is the value as shown (components/input/README, how a value is
	// shown); Empty is what an empty value means, in grey ("not set").
	Value func() string
	Empty string
	// Path cuts a value too long from the front: "…/.ssh/id_ed25519".
	Path bool
	// Edit makes the input popup Enter opens, when it opens: it starts from
	// the value then.
	Edit func() Input
	// Options make the field a radio drawn on the form, at most five
	// (Selected is the chosen one); with Multi, a group of checkboxes
	// (Checked says which).
	Options  []string
	Selected int
	Multi    bool
	Checked  []bool
	// OptionDisabled reports an option that cannot be chosen now: drawn in
	// the disabled colour and skipped (components/dialog/form).
	OptionDisabled func(i int) bool
	Disabled       func() bool
	// Check is the whole form's rule for this field (required, depends on
	// another field, a name taken), run when the form is submitted: what is
	// wrong, or "".
	Check func() string
}

func (f *FormField) disabled() bool { return f.Disabled != nil && f.Disabled() }

func (f *FormField) optionDisabled(i int) bool { return f.OptionDisabled != nil && f.OptionDisabled(i) }

// Form is several values filled in and submitted together
// (components/dialog/form): Enter on a field opens its input popup, options
// are chosen on the form, the button submits.
type Form struct {
	Title, Glyph string
	// Name is the form in the discard question: "Discard changes to New host?"
	Name   string
	Fields []*FormField
	// Button is the submit button's text: what the form does ("Save").
	Button string
	// Focus, when set, is the field the form opens on.
	Focus *FormField
	// Submit does what the form is for once every Check passes. An error
	// opens an error popup over the form, which stays; otherwise the form
	// closes and the command runs.
	Submit func() (tea.Cmd, error)
}

type formItemKind int

const (
	itemField formItemKind = iota
	itemError
	itemButton
)

// formItem is one place the focus stops at: a field, one of its options, the
// error row, the button.
type formItem struct {
	kind   formItemKind
	field  *FormField
	option int // -1 for the field itself
}

type formView struct {
	f     *Form
	focus formItem
	// back is the item of the field area the focus was last on: Tab on the
	// button or the error row returns there (components/dialog/form).
	back    formItem
	dirty   bool
	errs    map[*FormField]string
	errOn   *FormField // whose message the error row shows
	top     int        // the first row of the fields area shown
	visible int
}

// Form opens a form.
// It panics on a field with more than five options: those go in a select or
// a checkbox popup (components/dialog/form), a mistake in the program.
func (a *App) Form(f *Form) tea.Cmd {
	for _, fl := range f.Fields {
		if len(fl.Options) > 5 {
			panic(fmt.Sprintf("terminu: form field %q has %d options; more than five go in a select or a checkbox popup", fl.Label, len(fl.Options)))
		}
	}
	fv := &formView{f: f}
	fv.focus = fv.items()[0]
	if f.Focus != nil && !f.Focus.disabled() {
		fv.focus = fv.landing(f.Focus)
	}
	fv.back = fv.focus
	return a.open(fv)
}

// items lists the stops in order: each field (each option of one drawn on
// the form), the error row when there is an error, the button. Disabled
// fields are skipped.
func (fv *formView) items() []formItem {
	var out []formItem
	for _, f := range fv.f.Fields {
		if f.disabled() {
			continue
		}
		if len(f.Options) > 0 {
			for i := range f.Options {
				if !f.optionDisabled(i) {
					out = append(out, formItem{itemField, f, i})
				}
			}
			continue
		}
		out = append(out, formItem{itemField, f, -1})
	}
	if fv.errOn != nil {
		out = append(out, formItem{kind: itemError, option: -1})
	}
	return append(out, formItem{kind: itemButton, option: -1})
}

func (fv *formView) index(items []formItem) int {
	for i, it := range items {
		if it == fv.focus {
			return i
		}
	}
	return 0
}

// landing is where the focus goes in a field it is sent to (the form opened
// at it, a check failing on it, e back from the error row): a radio's chosen
// option or its first, a checkbox group's first option.
// TODO(tdp G74): the spec no longer says which option; this is v0.5.3's
// landing for Tab.
func (fv *formView) landing(f *FormField) formItem {
	switch {
	case len(f.Options) == 0:
		return formItem{itemField, f, -1}
	case !f.Multi && f.Selected >= 0 && f.Selected < len(f.Options):
		return formItem{itemField, f, f.Selected}
	}
	return formItem{itemField, f, 0}
}

func (fv *formView) labelWidth() int {
	w := 0
	for _, f := range fv.f.Fields {
		if !f.Stacked {
			w = max(w, Width(f.Label))
		}
	}
	return w
}

func (fv *formView) button() string { return cmpOr(fv.f.Button, "Save") }

func (fv *formView) hint() []KeyHint {
	var h []KeyHint
	// Tab names the area it goes to (components/dialog/form).
	switch it := fv.focus; it.kind {
	case itemError:
		return []KeyHint{{"Enter", "show"}, {"e", "edit"}, {"Tab", "fields"}, {"Esc", "cancel"}}
	case itemButton:
		h = append(h, KeyHint{"Enter", strings.ToLower(fv.button())}, KeyHint{"Tab", "fields"})
	default:
		switch {
		case len(it.field.Options) > 0 && it.field.Multi:
			h = append(h, KeyHint{"Enter|Space", "toggle"})
		case len(it.field.Options) > 0:
			h = append(h, KeyHint{"Enter", "choose"})
		default:
			h = append(h, KeyHint{"Enter", "edit"})
		}
		if fv.errs[it.field] != "" {
			h = append(h, KeyHint{"e", "error"})
		}
		h = append(h, KeyHint{"Tab", "actions"})
	}
	return append(h, KeyHint{"Esc", "cancel"})
}

// fieldRows is how many rows a field takes.
func fieldRows(f *FormField) int {
	n := max(len(f.Options), 1)
	if f.Stacked {
		n++
	}
	return n
}

func (fv *formView) measure(sw, sh int) (int, int) {
	rows := 0
	for _, f := range fv.f.Fields {
		rows += fieldRows(f)
	}
	// The fields, a blank row, the error row, the separator, the button row.
	return popupWidth(sw, 0, false), min(rows+4+4, max(sh-2, 3))
}

func (fv *formView) draw(w, h int, layer Color) []string {
	inner := w - 2
	cw := inner - 2 // the content, one cell from the frame on each side
	lw := fv.labelWidth()
	var rows []string
	fieldTop := map[*FormField]int{}
	for _, f := range fv.f.Fields {
		fieldTop[f] = len(rows)
		label := fv.label(f, layer)
		indent := 0
		if f.Stacked {
			rows = append(rows, " "+Fit(label, cw)+" ")
		} else {
			indent = lw + 2
			label = Pad(label, lw) + "  "
		}
		lead := func(first bool) string {
			if f.Stacked {
				return ""
			}
			if first {
				return label
			}
			return Repeat(" ", indent)
		}
		room := cw - indent
		switch {
		case len(f.Options) > 0:
			for i := range f.Options {
				if fv.focus == (formItem{itemField, f, i}) {
					rows = append(rows, " "+lead(i == 0)+fv.block(fv.option(f, i), room+1, layer))
					continue
				}
				rows = append(rows, " "+lead(i == 0)+Fit(fv.option(f, i), room)+" ")
			}
		default:
			value := fv.value(f)
			fit := Fit
			if f.Path {
				fit = FitLeft
				if f.Value != nil && f.Value() != "" {
					value = Fg(Text).Render(homeShort(f.Value()))
				}
			}
			if fv.focus == (formItem{itemField, f, -1}) {
				rows = append(rows, " "+lead(true)+fv.block(fit(ansi.Strip(value), room), room+1, layer))
				continue
			}
			rows = append(rows, " "+lead(true)+fit(value, room)+" ")
		}
	}
	// The fields area scrolls the least that keeps the focused field whole;
	// the error row, the separator and the button stay (components/dialog/form).
	area := max(h-8, 1)
	fv.visible = area
	if fv.focus.kind == itemField {
		top := fieldTop[fv.focus.field]
		bottom := top + fieldRows(fv.focus.field) - 1
		fv.top = scrollWindow(fv.top, bottom, area, len(rows))
		fv.top = min(fv.top, top)
	}
	fv.top = min(max(fv.top, 0), max(len(rows)-area, 0))
	body := make([]string, 0, h-4)
	for i := fv.top; i < fv.top+area; i++ {
		if i < len(rows) {
			body = append(body, rows[i])
		} else {
			body = append(body, Repeat(" ", inner))
		}
	}
	body = append(body, Repeat(" ", inner))
	body = append(body, fv.errorRow(cw, layer))
	body = append(body, " "+Fg(Overlay0).Render(Repeat("─", cw))+" ")
	button := " " + fv.button() + " "
	if fv.focus.kind == itemButton {
		button = On(layer).Render(button)
	} else {
		button = Style{Fg: Text, Bg: Surface1}.Render(button)
	}
	body = append(body, PadLeft(button, inner-1)+" ")
	scroll := ""
	// The position counts fields only; on the error row or the button there
	// is no field to count, so it is not written (components/dialog/form).
	if len(rows) > area && fv.focus.kind == itemField {
		for i, f := range fv.f.Fields {
			if f == fv.focus.field {
				scroll = scrollText(true, i+1, 0, len(fv.f.Fields))
			}
		}
	}
	return popupBox{color: layer, glyph: fv.f.Glyph, title: Upper(fv.f.Title), hint: fv.hint(), scroll: scroll}.draw(w, body)
}

// label is a field's label: the layer colour, Surface2 when disabled, Red
// when its value is wrong (components/dialog/form). It is not upper case.
func (fv *formView) label(f *FormField, layer Color) string {
	switch {
	case fv.errs[f] != "":
		return Fg(Red).Render(f.Label)
	case f.disabled():
		return Fg(Surface2).Render(f.Label)
	}
	return Fg(layer).Render(f.Label)
}

func (fv *formView) value(f *FormField) string {
	v := ""
	if f.Value != nil {
		v = f.Value()
	}
	if v == "" {
		return Fg(Overlay0).Render(f.Empty)
	}
	return Fg(Text).Render(v)
}

func (fv *formView) option(f *FormField, i int) string {
	if f.optionDisabled(i) {
		glyph := glyphRadioOff
		switch {
		case f.Multi && i < len(f.Checked) && f.Checked[i]:
			glyph = glyphMarked
		case f.Multi:
			glyph = glyphUnchecked
		case i == f.Selected:
			glyph = glyphRadioOn
		}
		return Fg(Surface2).Render(glyph + " " + f.Options[i])
	}
	if f.Multi {
		if i < len(f.Checked) && f.Checked[i] {
			return Fg(Green).Render(glyphMarked + " " + f.Options[i])
		}
		return Fg(Text).Render(glyphUnchecked + " " + f.Options[i])
	}
	if i == f.Selected {
		return Fg(Green).Render(glyphRadioOn + " " + f.Options[i])
	}
	return Fg(Text).Render(glyphRadioOff + " " + f.Options[i])
}

// block is the focus from the item's start to the frame: the layer colour,
// Base bold, the right-most cell an arrow (components/dialog/form).
func (fv *formView) block(s string, w int, layer Color) string {
	arrow := Width(glyphHard)
	return On(layer).Render(Fit(ansi.Strip(s), w-arrow)) + Fg(layer).Render(glyphHard)
}

func (fv *formView) errorRow(cw int, layer Color) string {
	if fv.errOn == nil {
		return Repeat(" ", cw+2)
	}
	msg := fv.errs[fv.errOn]
	if fv.focus.kind == itemError {
		return " " + fv.block(Truncate(msg, cw), cw+1, layer)
	}
	return " " + Fit(Fg(Red).Render(msg), cw) + " "
}

func (fv *formView) step(dir int) {
	items := fv.items()
	i := fv.index(items)
	fv.focus = items[(i+dir+len(items))%len(items)]
}

// tab moves between the field area and the action row, the error row going
// with the button: to the button from a field, back to the item left from
// the button or the error row (components/dialog/form, Rules K2).
func (fv *formView) tab() {
	items := fv.items()
	if fv.focus.kind == itemField {
		fv.focus = items[len(items)-1]
		return
	}
	fv.focus = items[0]
	for _, it := range items {
		if it == fv.back {
			fv.focus = it
		}
	}
}

func (fv *formView) key(a *App, key string, msg tea.KeyMsg) (bool, tea.Cmd) {
	defer func() {
		if fv.focus.kind == itemField {
			fv.back = fv.focus
		}
	}()
	items := fv.items()
	half := max(fv.visible/2, 1)
	switch key {
	case "j", "l", "↓", "→":
		fv.step(1)
	case "k", "h", "↑", "←":
		fv.step(-1)
	case "Tab", "Shift-Tab":
		fv.tab() // two areas: either way is the other one
	case "d", "PgDn":
		fv.focus = items[min(fv.index(items)+half, len(items)-1)]
	case "u", "PgUp":
		fv.focus = items[max(fv.index(items)-half, 0)]
	case "gg":
		fv.focus = items[0]
	case "G":
		fv.focus = items[len(items)-1]
	case "e":
		switch {
		case fv.focus.kind == itemField && fv.errs[fv.focus.field] != "":
			fv.errOn = fv.focus.field
			fv.focus = formItem{kind: itemError, option: -1}
		case fv.focus.kind == itemError:
			fv.focus = fv.landing(fv.errOn)
		}
	case "Enter":
		return true, fv.enter(a)
	case "Esc":
		if !fv.dirty {
			return true, a.closeTop()
		}
		name := cmpOr(fv.f.Name, fv.f.Title)
		return true, a.Confirm(Ask{Title: "Discard", Question: "Discard changes to " + name + "?", Verb: "discard"},
			func() tea.Cmd { return a.closeTop() })
	case "Space":
		// On a checkbox option Space ticks it too, as in a checkbox popup;
		// elsewhere it does nothing (Rules K5).
		if it := fv.focus; it.kind == itemField && it.field.Multi && it.option >= 0 {
			return true, fv.enter(a)
		}
		return true, nil
	case "?":
		return false, nil
	default:
		if msg.Type != tea.KeyRunes || msg.Alt {
			return true, nil
		}
		// Typing on a field one types in: a note says how (components/dialog/form).
		if it := fv.focus; it.kind == itemField && it.field.Edit != nil && it.field.Edit().typed() {
			label := it.field.Label
			return true, a.open(&noteView{
				glyph: fv.f.Glyph, title: Upper(label), name: label, known: true, enterCloses: true,
				lines: []string{Fg(Text).Render("Press [Enter] to edit " + label + ".")},
				hint:  []KeyHint{{"Enter|Esc", "close"}},
			})
		}
		return true, nil // q too: a form does not start the quit flow (Rules K9)
	}
	return true, nil
}

func (fv *formView) enter(a *App) tea.Cmd {
	it := fv.focus
	switch it.kind {
	case itemError:
		label := ""
		if fv.errOn != nil {
			label = fv.errOn.Label
		}
		return a.Error(label, fv.errs[fv.errOn])
	case itemButton:
		return fv.submit(a)
	}
	f := it.field
	switch {
	case len(f.Options) > 0 && f.Multi:
		for len(f.Checked) < len(f.Options) {
			f.Checked = append(f.Checked, false)
		}
		f.Checked[it.option] = !f.Checked[it.option]
		fv.dirty = true
	case len(f.Options) > 0:
		if f.Selected != it.option {
			f.Selected = it.option
			fv.dirty = true
		}
	case f.Edit != nil:
		return f.Edit().open(a, func() { fv.dirty = true })
	}
	return nil
}

// submit checks the whole form; with nothing wrong it does what the form is
// for (components/dialog/form, errors).
func (fv *formView) submit(a *App) tea.Cmd {
	fv.errs, fv.errOn = map[*FormField]string{}, nil
	for _, f := range fv.f.Fields {
		if f.Check == nil {
			continue
		}
		if msg := f.Check(); msg != "" {
			fv.errs[f] = msg
			if fv.errOn == nil {
				fv.errOn = f
			}
		}
	}
	if fv.errOn != nil {
		fv.focus = fv.landing(fv.errOn)
		return nil
	}
	if fv.f.Submit == nil {
		return a.closeTop()
	}
	cmd, err := fv.f.Submit()
	if err != nil {
		return a.Error(fv.button()+" failed", err.Error())
	}
	return tea.Batch(a.closeTop(), cmd)
}

func (fv *formView) surface() string { return fv.f.Title }

func (fv *formView) keyRef() []refSection {
	return []refSection{
		{rows: []KeyRow{
			{Key: "Enter", Desc: "edit the field, choose or toggle the option, press the button"},
			{Key: "Space", Desc: "toggle the checkbox option"},
			{Key: "e", Desc: "between a field that is wrong and the error row"},
			{Key: "Tab|Shift-Tab", Desc: "between the fields and the button, back to the item left"},
			{Key: "Esc", Desc: "cancel (asks first when something changed)"},
		}},
		{title: "move", rows: []KeyRow{
			{Key: "j|k", Desc: "next, previous"},
			{Key: "u|d", Desc: "half a page up, down"},
			{Key: "gg|G", Desc: "to the first, the button"},
		}},
	}
}

func (fv *formView) typing() bool { return false }
