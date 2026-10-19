package terminu

import (
	"context"
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
	// OptionUnavailable reports a checkbox option that is not available:
	// drawn in Surface2, the tick too, but stopped on and flipped like any
	// other (components/input/checkbox).
	OptionUnavailable func(i int) bool
	Disabled          func() bool
	// Check is the whole form's rule for this field (required, depends on
	// another field, a name taken), run when the form is submitted: what is
	// wrong, or "".
	Check func() string
}

func (f *FormField) disabled() bool { return f.Disabled != nil && f.Disabled() }

func (f *FormField) optionDisabled(i int) bool { return f.OptionDisabled != nil && f.OptionDisabled(i) }

func (f *FormField) optionUnavailable(i int) bool {
	return f.OptionUnavailable != nil && f.OptionUnavailable(i)
}

// Form is several values filled in and submitted together
// (components/dialog/form): Enter on a field opens its input popup, options
// are chosen on the form, the button submits.
type Form struct {
	Title, Glyph string
	// Subject is the one thing it acts on, kept as it is after ": " in the
	// title: "EDIT: filu" (components/layout/popup).
	Subject string
	// Name is the form in the discard question: "Discard changes to New host?"
	Name   string
	Fields []*FormField
	// Button is the main submit button's text: what the form does ("Save").
	// ButtonSubject is the name it acts on, kept as it is after the verb on
	// the button and in the hint: "Overwrite KBU.svg", "Enter:overwrite
	// KBU.svg" (components/dialog/form, hint).
	Button, ButtonSubject string
	// Buttons are the action area's other buttons, row by row from the top,
	// each row on the right. The main submit goes at the end of the last
	// row, the bottom right; an empty last row puts it on a row of its own.
	Buttons [][]*FormButton
	// Focus, when set, is the field the form opens on.
	Focus *FormField
	// Submit does what the form is for once every Check passes. An error
	// opens an error popup over the form, which stays; otherwise the form
	// closes and the command runs.
	Submit func() (tea.Cmd, error)
	// SubmitLong is a Submit that takes long ("Connect"), in its place: it
	// runs off the update loop, the loading icon after the title and Enter
	// on its button doing nothing until it ends. ctx ends when the form
	// closes first; what it returns then is dropped (components/dialog/form).
	SubmitLong func(ctx context.Context) (tea.Cmd, error)
}

// FormButton is a button of a form's action area besides the main submit
// (components/dialog/form): another submit, or an action that keeps the
// form open. It has one of Submit, SubmitLong, Run and Open.
type FormButton struct {
	// Label is what it does ("Overwrite"); Subject the name it acts on, kept
	// as it is, as the form's ButtonSubject.
	Label, Subject string
	// Submit makes it another submit ("Save As"): it checks the whole form
	// as the main one does, and the form closes once it is done. SubmitLong
	// is one that takes long, as the form's.
	Submit     func() (tea.Cmd, error)
	SubmitLong func(ctx context.Context) (tea.Cmd, error)
	// Run makes it an action that keeps the form open ("Test"). It checks
	// Needs first, the fields it needs (their Check); then it runs off the
	// update loop, the loading icon after the title and Enter on it doing
	// nothing until it ends. ctx ends when the form closes first; its result
	// is then dropped, Apply too.
	Needs []*FormField
	Run   func(ctx context.Context) FormResult
	// Open makes it an action that opens a popup over the form, on the update
	// loop, the form staying under it: a terminal popup (OpenTerminal) that
	// copies a key to the host, say. Back, when set, runs as the popup it
	// opened closes and the form is on top again: where the app reads what
	// was done there into the fields; true counts as a change to the form.
	Open func() tea.Cmd
	Back func() (changed bool)
}

// FormResult is what came of a form's action (components/dialog/form): Err
// opens an error popup, the focus on the button so that Enter tries again;
// else Lines open a note to read; else Toast says it in one sentence.
type FormResult struct {
	Toast string
	Lines []string
	Err   error
	// Apply, when set, runs on the update loop as it ends, before the result
	// shows: where the action writes a value into a field, which then counts
	// as changed.
	Apply func()
}

type formItemKind int

const (
	itemField formItemKind = iota
	itemError
	itemButton
)

// formItem is one place the focus stops at: a field, one of its options, the
// error row, a button (option: its place in the action area).
type formItem struct {
	kind   formItemKind
	field  *FormField
	option int // -1 for the field itself
}

type formView struct {
	f     *Form
	focus formItem
	// back and backAction are the items of the field area and the action
	// area the focus was last on: Tab returns there (components/dialog/form).
	back, backAction formItem
	// buttons are the action area's in reading order, nil the main submit.
	buttons []*FormButton
	// running are the buttons whose work runs off the update loop, by their
	// place, with what ends its ctx; gone is set once the form closes.
	running map[int]context.CancelFunc
	gone    bool
	dirty   bool
	errs    map[*FormField]string
	errOn   *FormField // whose message the error row shows
	top     int        // the first row of the fields area shown
	visible int
}

// Form opens a form.
// It panics on a field with more than five options: those go in a select or
// a checkbox popup (components/dialog/form); on a form with both Submit and
// SubmitLong; and on a button with not one of Submit, SubmitLong, Run and Open:
// mistakes in the program.
func (a *App) Form(f *Form) tea.Cmd {
	for _, fl := range f.Fields {
		if len(fl.Options) > 5 {
			panic(fmt.Sprintf("terminu: form field %q has %d options; more than five go in a select or a checkbox popup", fl.Label, len(fl.Options)))
		}
	}
	if f.Submit != nil && f.SubmitLong != nil {
		panic(fmt.Sprintf("terminu: form %q has both Submit and SubmitLong", f.Title))
	}
	fv := &formView{f: f, running: map[int]context.CancelFunc{}}
	for _, row := range f.Buttons {
		for _, b := range row {
			n := 0
			for _, set := range []bool{b.Submit != nil, b.SubmitLong != nil, b.Run != nil, b.Open != nil} {
				if set {
					n++
				}
			}
			if n != 1 {
				panic(fmt.Sprintf("terminu: form button %q needs one of Submit, SubmitLong, Run and Open", b.Label))
			}
			fv.buttons = append(fv.buttons, b)
		}
	}
	fv.buttons = append(fv.buttons, nil)
	// The first time Tab goes to the action area it lands on the main submit.
	fv.backAction = fv.main()
	fv.focus = fv.main()
	if items := fv.fieldItems(); len(items) > 0 {
		fv.focus = items[0]
	}
	if f.Focus != nil && !f.Focus.disabled() {
		fv.focus = fv.landing(f.Focus)
	}
	fv.back = fv.focus
	return a.open(fv)
}

// main is the main submit, the last button: the bottom right.
func (fv *formView) main() formItem {
	return formItem{kind: itemButton, option: len(fv.buttons) - 1}
}

// actionRows are the action area's rows of buttons, top to bottom.
func (fv *formView) actionRows() [][]formItem {
	var rows [][]formItem
	i := 0
	for _, row := range fv.f.Buttons {
		var r []formItem
		for range row {
			r = append(r, formItem{kind: itemButton, option: i})
			i++
		}
		rows = append(rows, r)
	}
	if len(rows) == 0 {
		rows = append(rows, nil)
	}
	rows[len(rows)-1] = append(rows[len(rows)-1], fv.main())
	return rows
}

// buttonParts are a button's verb and the name it acts on, if any.
func (fv *formView) buttonParts(it formItem) (label, subject string) {
	if b := fv.buttons[it.option]; b != nil {
		return b.Label, b.Subject
	}
	return fv.button(), fv.f.ButtonSubject
}

// buttonLabel is the text on a button: "Overwrite KBU.svg".
func (fv *formView) buttonLabel(it formItem) string {
	label, subject := fv.buttonParts(it)
	if subject == "" {
		return label
	}
	return label + " " + subject
}

// buttonHint is what a button's Enter says in the hint: the verb in lower
// case, the name as it is, "overwrite KBU.svg" (components/dialog/form, hint;
// components/README, upper case).
func (fv *formView) buttonHint(it formItem) string {
	label, subject := fv.buttonParts(it)
	if subject == "" {
		return strings.ToLower(label)
	}
	return strings.ToLower(label) + " " + subject
}

// failed is the error popup of a button's work failing: its verb in the
// title, the name after ": " as it is (components/layout/popup, title).
func (fv *formView) failed(a *App, it formItem, err error) tea.Cmd {
	label, subject := fv.buttonParts(it)
	n := errorPopup(label+" failed", "", err.Error())
	n.title = popupTitle(label+" failed", subject)
	return a.open(n)
}

// area is the stops of the area the focus is in (components/dialog/form):
// the field area, or the buttons in reading order.
func (fv *formView) area() []formItem {
	if fv.focus.kind != itemButton {
		if items := fv.fieldItems(); len(items) > 0 {
			return items
		}
	}
	var out []formItem
	for i := range fv.buttons {
		out = append(out, formItem{kind: itemButton, option: i})
	}
	return out
}

// fieldItems lists the field area's stops in order: each field (each option
// of one drawn on the form), then the error row when there is an error.
// Disabled fields are skipped.
func (fv *formView) fieldItems() []formItem {
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
	return out
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
// at it, a check failing on it, e back from the error row): the first option
// with a value — a radio's chosen one, a checkbox group's first ticked one —
// or else the first (components/dialog/form). Disabled options are skipped.
func (fv *formView) landing(f *FormField) formItem {
	if len(f.Options) == 0 {
		return formItem{itemField, f, -1}
	}
	first := -1
	for i := range f.Options {
		if f.optionDisabled(i) {
			continue
		}
		if first < 0 {
			first = i
		}
		if f.Multi && i < len(f.Checked) && f.Checked[i] || !f.Multi && i == f.Selected {
			return formItem{itemField, f, i}
		}
	}
	return formItem{itemField, f, max(first, 0)}
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
		// The error row is in the field area: Tab goes to the actions.
		return []KeyHint{{"Enter", "show"}, {"e", "edit"}, {"Tab", "actions"}, {"Esc", "cancel"}}
	case itemButton:
		// A button whose work still runs has no Enter until it ends.
		if fv.running[it.option] == nil {
			h = append(h, KeyHint{"Enter", fv.buttonHint(it)})
		}
		h = append(h, KeyHint{"Tab", "fields"})
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
	// The fields, a blank row, the error row, the separator, the button rows.
	return popupWidth(sw, 0, false), min(rows+3+len(fv.actionRows())+4, max(sh-2, 3))
}

func (fv *formView) draw(w, h int, layer Color) []string {
	inner := w - 2
	// The content, one cell from the frame on the left; on the right as much
	// room as the arrow at the end of the focus block, on every row, so the
	// arrow never covers a value (components/dialog/form, layout/popup).
	arrow := Width(glyphHard)
	pad := Repeat(" ", arrow)
	cw := inner - 1 - arrow
	lw := fv.labelWidth()
	var rows []string
	fieldTop := map[*FormField]int{}
	for _, f := range fv.f.Fields {
		fieldTop[f] = len(rows)
		label := fv.label(f, layer)
		indent := 0
		if f.Stacked {
			rows = append(rows, " "+Fit(label, cw)+pad)
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
					rows = append(rows, " "+lead(i == 0)+fv.block(fv.option(f, i), room+arrow, layer))
					continue
				}
				rows = append(rows, " "+lead(i == 0)+Fit(fv.option(f, i), room)+pad)
			}
		default:
			value := fv.value(f)
			fit := Fit
			if f.Path {
				fit = FitLeft
				if f.Value != nil && f.Value() != "" {
					value = valueText(homeShort(f.Value()), Fg(Text))
				}
			}
			if fv.focus == (formItem{itemField, f, -1}) {
				// Fitted to the block, the arrow taken off: a value that fits gets no ….
				rows = append(rows, " "+lead(true)+fv.block(fit(value, room), room+arrow, layer))
				continue
			}
			rows = append(rows, " "+lead(true)+fit(value, room)+pad)
		}
	}
	// The fields area scrolls the least that keeps the focused field whole;
	// the error row, the separator and the buttons stay (components/dialog/form).
	actions := fv.actionRows()
	area := max(h-7-len(actions), 1)
	fv.visible = area
	if fv.focus.kind == itemField {
		top := fieldTop[fv.focus.field]
		bottom := top + fieldRows(fv.focus.field) - 1
		fv.top = scrollWindow(fv.top, bottom, area, len(rows))
		fv.top = min(fv.top, top)
	}
	fv.top = min(max(fv.top, 0), max(len(rows)-area, 0))
	body := make([]string, 0, max(h-4, 0))
	for i := fv.top; i < fv.top+area; i++ {
		if i < len(rows) {
			body = append(body, rows[i])
		} else {
			body = append(body, Repeat(" ", inner))
		}
	}
	body = append(body, Repeat(" ", inner))
	body = append(body, fv.errorRow(cw, layer))
	body = append(body, " "+Fg(Overlay0).Render(Repeat("─", inner-2))+" ")
	// Each row of buttons on the right, one cell between two buttons.
	for _, row := range actions {
		var parts []string
		for _, it := range row {
			button := " " + fv.buttonLabel(it) + " "
			if fv.focus == it {
				button = On(layer).Render(button)
			} else {
				button = Style{Fg: Text, Bg: Surface1}.Render(button)
			}
			parts = append(parts, button)
		}
		body = append(body, PadLeft(strings.Join(parts, " "), inner-1)+" ")
	}
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
	return popupBox{color: layer, glyph: fv.f.Glyph, title: popupTitle(fv.f.Title, fv.f.Subject), hint: fv.hint(), scroll: scroll,
		loading: fv.isLoading()}.draw(w, body)
}

// isLoading reports an action running: the loading icon goes after the title
// (components/dialog/form, the action area).
func (fv *formView) isLoading() bool { return len(fv.running) > 0 }

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
	return valueText(v, Fg(Text))
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
		on := i < len(f.Checked) && f.Checked[i]
		if f.optionUnavailable(i) {
			return Fg(Surface2).Render(checkGlyph(on) + " " + f.Options[i])
		}
		if on {
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
// Base bold, a swatch keeping its colour, the right-most cell an arrow
// (components/dialog/form, components/color).
func (fv *formView) block(s string, w int, layer Color) string {
	arrow := Width(glyphHard)
	return onBlock(Fit(s, w-arrow), On(layer)) + Fg(layer).Render(glyphHard)
}

func (fv *formView) errorRow(cw int, layer Color) string {
	arrow := Width(glyphHard)
	if fv.errOn == nil {
		return Repeat(" ", 1+cw+arrow)
	}
	msg := fv.errs[fv.errOn]
	if fv.focus.kind == itemError {
		return " " + fv.block(Truncate(msg, cw), cw+arrow, layer)
	}
	return " " + Fit(Fg(Red).Render(msg), cw) + Repeat(" ", arrow)
}

// step moves one item within the focus's area, wrapping at its ends (Rules
// K12): j|k never leave the area.
func (fv *formView) step(dir int) {
	items := fv.area()
	i := fv.index(items)
	fv.focus = items[(i+dir+len(items))%len(items)]
}

// tab moves between the field area, the error row included, and the action
// area, back to the item left in each; the first time to the main submit
// (components/dialog/form, Rules K2).
func (fv *formView) tab() {
	if fv.focus.kind != itemButton {
		fv.focus = fv.backAction
		return
	}
	items := fv.fieldItems()
	if len(items) == 0 {
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
		if fv.focus.kind == itemButton {
			fv.backAction = fv.focus
		} else {
			fv.back = fv.focus
		}
	}()
	items := fv.area()
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
				glyph: fv.f.Glyph, title: Upper(label), name: label, known: true, enterCloses: true, onlyClose: true,
				text: "Press [Enter] to edit " + label + ".", textStyle: Fg(Text),
				hint: []KeyHint{{"Enter|Esc", "close"}},
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
		return fv.press(a, it)
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

// press presses a button: a submit, or an action that keeps the form open
// (components/dialog/form, the action area).
func (fv *formView) press(a *App, it formItem) tea.Cmd {
	if fv.running[it.option] != nil {
		return nil // it still runs: the same thing is not sent twice
	}
	b := fv.buttons[it.option]
	submit, long := fv.f.Submit, fv.f.SubmitLong
	if b != nil {
		submit, long = b.Submit, b.SubmitLong
	}
	switch {
	case b != nil && b.Open != nil:
		n := len(a.stack)
		cmd := b.Open()
		if len(a.stack) > n && b.Back != nil {
			// What it opened is over the form: once that is gone, the app
			// reads back what was done there.
			back := b.Back
			a.stack[len(a.stack)-1].below = func() tea.Cmd {
				if back() {
					fv.dirty = true
				}
				return nil
			}
		}
		return cmd
	case b != nil && b.Run != nil:
		if !fv.check(b.Needs) {
			return nil
		}
		run := b.Run
		return fv.start(it, func(ctx context.Context) formRan { return formRan{res: run(ctx)} })
	case long != nil:
		if !fv.check(fv.f.Fields) {
			return nil
		}
		return fv.start(it, func(ctx context.Context) formRan {
			cmd, err := long(ctx)
			return formRan{submit: true, cmd: cmd, err: err}
		})
	}
	return fv.submit(a, it, submit)
}

// start runs a button's work off the update loop: the loading icon after the
// title until it ends; the form closing ends ctx (components/dialog/form).
func (fv *formView) start(it formItem, work func(ctx context.Context) formRan) tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	fv.running[it.option] = cancel
	return func() tea.Msg {
		m := work(ctx)
		m.fv, m.at = fv, it
		return m
	}
}

// closed is the form gone: what still runs is cancelled, and what it ends
// with has nowhere to go (components/dialog/form).
func (fv *formView) closed() {
	fv.gone = true
	for _, cancel := range fv.running {
		cancel()
	}
}

// formRan is a button's work ended, back on the update loop: an action's
// result, or a long submit's command and error.
type formRan struct {
	fv     *formView
	at     formItem
	res    FormResult
	submit bool
	cmd    tea.Cmd
	err    error
}

// done shows what came of it: an error popup with the focus on its button, so
// that Enter tries again; a long submit done closes the form; an action's note
// to read or toast. Once the form is gone it shows nothing.
func (m formRan) done(a *App) tea.Cmd {
	fv := m.fv
	if cancel := fv.running[m.at.option]; cancel != nil {
		cancel()
		delete(fv.running, m.at.option)
	}
	if fv.gone {
		return nil
	}
	switch {
	case m.submit && m.err != nil:
		fv.focus = m.at
		return fv.failed(a, m.at, m.err)
	case m.submit:
		return tea.Batch(a.closeView(fv), m.cmd)
	}
	if m.res.Apply != nil {
		m.res.Apply()
		fv.dirty = true // a value written into a field counts as a change
	}
	switch {
	case m.res.Err != nil:
		fv.focus = m.at
		return fv.failed(a, m.at, m.res.Err)
	case len(m.res.Lines) > 0:
		label, subject := fv.buttonParts(m.at)
		return a.Viewer(&Viewer{Title: label, Subject: subject, Lines: m.res.Lines})
	case m.res.Toast != "":
		return a.Toast(Info, m.res.Toast)
	}
	return nil
}

// check runs the Check of fields, the whole form's rules for them; the errors
// stand until the next check (components/dialog/form, errors). On a problem
// the focus goes to the first field with one.
func (fv *formView) check(fields []*FormField) bool {
	fv.errs, fv.errOn = map[*FormField]string{}, nil
	for _, f := range fields {
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
		return false
	}
	return true
}

// submit checks the whole form; with nothing wrong it does what the button is
// for and closes (components/dialog/form, errors).
func (fv *formView) submit(a *App, it formItem, do func() (tea.Cmd, error)) tea.Cmd {
	if !fv.check(fv.f.Fields) {
		return nil
	}
	if do == nil {
		return a.closeTop()
	}
	cmd, err := do()
	if err != nil {
		return fv.failed(a, it, err)
	}
	// Only the form closes: a popup the submit opened stays over it.
	return tea.Batch(a.closeOnly(fv), cmd)
}

func (fv *formView) surface() string { return fv.f.Title }

// keyRef lists Space only when the form has checkbox options and e only when
// a field is wrong; either is in the disabled colour when it does nothing
// where the focus is (Rules M6).
func (fv *formView) keyRef() []refSection {
	rows := []KeyRow{{Key: "Enter", Desc: "edit the field, choose or toggle the option, press the button"}}
	if slices.ContainsFunc(fv.f.Fields, func(f *FormField) bool { return f.Multi && len(f.Options) > 0 }) {
		it := fv.focus
		rows = append(rows, KeyRow{Key: "Space", Desc: "toggle the checkbox option",
			Disabled: it.kind != itemField || !it.field.Multi || it.option < 0})
	}
	if len(fv.errs) > 0 {
		it := fv.focus
		rows = append(rows, KeyRow{Key: "e", Desc: "between a field that is wrong and the error row",
			Disabled: it.kind != itemError && (it.kind != itemField || fv.errs[it.field] == "")})
	}
	rows = append(rows,
		KeyRow{Key: "Tab|Shift-Tab", Desc: "between the fields and the buttons, back to the item left"},
		KeyRow{Key: "Esc", Desc: "cancel (asks first when something changed)"})
	return []refSection{
		{rows: rows},
		{title: "move", rows: []KeyRow{
			{Key: "h|k", Desc: "previous in this area"},
			{Key: "j|l", Desc: "next in this area"},
			{Key: "u|d", Desc: "half a page up, down"},
			{Key: "gg|G", Desc: "to the first, the last in this area"},
		}},
	}
}

func (fv *formView) typing() bool { return false }
