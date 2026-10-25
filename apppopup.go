package terminu

import (
	"context"
	"reflect"
	"slices"

	tea "github.com/charmbracelet/bubbletea"
)

// Popup is a popup of the app's own, for what none of the seven kinds holds
// (a grid to draw on; tdp F1, popup.8). The app draws what is inside the
// frame and handles its keys; the lib draws the frame with the title and the
// hint in the layer colour, dims what is under it, opens and closes it with
// the animation (tdp F2, F8), and puts its own popups over it: a confirm, an
// input, an error popup, the key reference.
type Popup struct {
	// Title is what the popup is for, upper case on the frame; Subject is
	// the one thing it acts on, kept as it is after ": " ("DRAW: kamonu";
	// tdp popup.1). Glyph goes before the title.
	Title, Glyph, Subject string
	Body                  PopupBody
	// Actions, when set, says it has values to submit: it then works as a
	// form (tdp popup.8, form). The action area goes under the content;
	// Tab and Shift-Tab go between the two, back to where the focus was;
	// on the buttons the keys are the lib's. A body with Check keeps an
	// error row. Esc asks first once the body has Changed. q does nothing,
	// Ctrl-C is the quit flow (tdp K9).
	Actions *PopupActions
}

// PopupActions is the action area of a popup of the app's own with values to
// submit; each field works as a Form's of the same name (tdp form.7).
type PopupActions struct {
	// Button is the main submit's text ("Save"), at the bottom right;
	// ButtonSubject the name it acts on. Buttons are the others, row by row
	// from the top. A button's Needs has no fields to name here: a Run button
	// checks nothing.
	Button, ButtonSubject string
	Buttons               [][]*FormButton
	// Submit and SubmitLong do what it is for once the body's Check passes;
	// it then closes. With neither, the main button only closes it.
	Submit     func() (tea.Cmd, error)
	SubmitLong func(ctx context.Context) (tea.Cmd, error)
	// OnCancel runs when it closes without being submitted.
	OnCancel func() tea.Cmd
	// Name is what the discard question names: "Discard changes to
	// kamonu?"; the popup's Subject, else its Title, when unset.
	Name string
	// Content names the content in a button's hint, where Tab goes back
	// to: "grid" in "Tab:grid"; "content" when unset (tdp form.13).
	Content string
}

// PopupBody is the inside of a popup of the app's own. It must be comparable,
// a pointer most often: ClosePopup finds the popup by it.
//
// A body may also have QuitNote() string: what quitting now would lose
// ("Changes to kamonu are not saved."), "" for nothing. The quit flow's
// confirm puts it last, in Peach, and asks first even when App.Quit is nil
// (tdp K9: the quit flow is the app's).
//
// A body that scrolls may also have Scroll() string: its scroll position
// ("3 of 40", "1-20 of 120"), "" when everything fits. The lib draws it on
// the right of the bottom frame line in the layer colour and, when the popup
// opens, makes it wide enough for it and the whole hint
// (tdp popup.6).
//
// A body with Actions may also have:
//   - Changed() bool: something was changed, to be lost by Esc; once it says
//     so, Esc asks first (tdp form.15).
//   - Check() string: what is wrong with the values, "" for nothing. It runs
//     at each submit; what it says goes in the error row and stays until the
//     next check, the focus back on the content (tdp popup.7).
//   - Focus(on bool): called before each Draw, whether the focus is on the
//     content (true) or on a button (false): draw where the focus is only
//     while it is there (tdp L5).
type PopupBody interface {
	// Size is the inside on a sw × sh screen: w cells from frame line to
	// frame line, h rows between the blank rows above and below. It is asked
	// when the popup opens and when the terminal changes size; the lib makes
	// it wide enough for the title and the hint, and no bigger than the
	// screen allows (tdp F7).
	Size(sw, sh int) (w, h int)
	// Draw returns h lines of w cells; layer is the popup's colour. The cell
	// at either end is the blank margin, which only a full-row cursor covers
	// (tdp popup).
	Draw(w, h int, layer Color) []string
	// Key handles a key while the popup is on top and reports whether it
	// did. The lib does the keys it leaves: Esc closes the popup, ? opens its
	// key reference, q starts the quit flow (tdp K1). Ctrl-C never comes
	// here: it is always the quit flow (tdp K9). Space comes here too: on a
	// cell with several values it goes to the next one (tdp K5).
	//
	// With Actions, only the keys on the content come here, and not Tab,
	// Shift-Tab, Esc or q: those are the lib's, as on a form.
	Key(key string, msg tea.KeyMsg) (bool, tea.Cmd)
	// Hint is the hint on the bottom frame line now (tdp M5). With Actions,
	// only the content's own keys: the lib adds Tab:actions and Esc:cancel,
	// and writes the hint on the buttons.
	Hint() []KeyHint
	// Keys is its key reference: only its own keys, Esc among them (tdp K6).
	// With Actions, only the content's: the lib adds the buttons', Tab and
	// Esc.
	Keys() []KeySection
}

// Popup opens p over what is on top. It panics when p.Body is nil or not
// comparable: that is a mistake in the program.
func (a *App) Popup(p Popup) tea.Cmd {
	if p.Body == nil || !reflect.TypeOf(p.Body).Comparable() {
		panic("terminu: popup " + p.Title + ": its Body must be a pointer or another comparable value")
	}
	if x := p.Actions; x != nil {
		fv := newFormView(&Form{Title: p.Title, Glyph: p.Glyph, Subject: p.Subject, Name: cmpOr(x.Name, p.Subject),
			Button: x.Button, ButtonSubject: x.ButtonSubject, Buttons: x.Buttons,
			Submit: x.Submit, SubmitLong: x.SubmitLong, OnCancel: x.OnCancel}, p.Body)
		fv.content = x.Content
		return a.open(fv)
	}
	return a.open(&appPopup{p: p})
}

// bodyOf is the body of a popup of the app's own, nil for any other popup.
func bodyOf(v popupView) PopupBody {
	switch v := v.(type) {
	case *appPopup:
		return v.p.Body
	case *formView:
		return v.body
	}
	return nil
}

// ClosePopup closes the popup b is the inside of, and every popup over it.
func (a *App) ClosePopup(b PopupBody) tea.Cmd {
	for _, l := range a.stack {
		if bodyOf(l.view) == b && !l.closing {
			return a.closeView(l.view)
		}
	}
	return nil
}

// CloseAll closes every popup, the lib's and the app's, each playing its own
// closing at the same time (tdp F2), as when what was done ends the whole
// stack (tdp T1; a long session started from a viewer). A confirm or an
// input closed this way is cancelled (OnCancel).
func (a *App) CloseAll() tea.Cmd { return a.closeAll() }

// CloseTop closes the n popups on top, the lowest of them animating, as the
// last step of a flow ends the steps under it (tdp T1); a step that has
// been answered is not cancelled.
func (a *App) CloseTop(n int) tea.Cmd {
	var open []*layer
	for _, l := range a.stack {
		if !l.closing {
			open = append(open, l)
		}
	}
	if n <= 0 || len(open) == 0 {
		return nil
	}
	return a.closeView(open[len(open)-min(n, len(open))].view)
}

type appPopup struct{ p Popup }

func (v *appPopup) title() string { return popupTitle(v.p.Title, v.p.Subject) }

func (v *appPopup) measure(sw, sh int) (int, int) {
	w, h := v.p.Body.Size(sw, sh)
	w = max(w+2, Width(titleCapsule(v.p.Glyph, v.title(), Base, 1<<20))+4, hintWidth(v.p.Body.Hint())+6)
	if s := v.scroll(); s != "" {
		w = max(w, scrollRoom(v.p.Body.Hint(), s))
	}
	return min(w, max(min(sw-2, 120), 1)), min(max(h, 0)+4, max(sh-2, 3))
}

func (v *appPopup) draw(w, h int, layer Color) []string {
	inner, n := w-2, max(h-4, 0)
	rows := v.p.Body.Draw(inner, n, layer)
	body := make([]string, n)
	for i := range body {
		if i < len(rows) {
			body[i] = Fit(rows[i], inner)
		} else {
			body[i] = Repeat(" ", inner)
		}
	}
	return popupBox{color: layer, glyph: v.p.Glyph, title: v.title(), hint: v.p.Body.Hint(), scroll: v.scroll()}.draw(w, body)
}

// scroll is the body's scroll position, when it has one.
func (v *appPopup) scroll() string {
	if s, ok := v.p.Body.(interface{ Scroll() string }); ok {
		return s.Scroll()
	}
	return ""
}

func (v *appPopup) key(_ *App, key string, msg tea.KeyMsg) (bool, tea.Cmd) {
	return v.p.Body.Key(key, msg)
}

func (v *appPopup) surface() string { return v.p.Title }

func (v *appPopup) keyRef() []refSection {
	var out []refSection
	for _, s := range v.p.Body.Keys() {
		out = append(out, refSection{title: s.Title, rows: s.Rows})
	}
	return out
}

func (v *appPopup) typing() bool { return false }

// The rest is a popup of the app's own with Actions: a form whose field area
// is the body (tdp popup.8).

// tail is how many rows go under the content: a blank row and the error row
// when the values may be wrong, the separator, the buttons (tdp popup.7,
// form.7).
func (fv *formView) tail() int {
	n := 1 + len(fv.actionRows())
	if fv.checks() {
		n += 2
	}
	return n
}

// checks reports a body whose values may be wrong: the error row is kept for
// it (tdp F7).
func (fv *formView) checks() bool {
	_, ok := fv.body.(interface{ Check() string })
	return ok
}

// bodyScroll is the body's scroll position, when it has one.
func (fv *formView) bodyScroll() string {
	if s, ok := fv.body.(interface{ Scroll() string }); ok {
		return s.Scroll()
	}
	return ""
}

// bodyMeasure makes it wide enough for the content, the title, the rows of
// buttons and the hint wherever the focus is, with the scroll position on the
// content; then no bigger than the screen allows (tdp F7).
func (fv *formView) bodyMeasure(sw, sh int) (int, int) {
	w, h := fv.body.Size(sw, sh)
	w = max(w+2, Width(titleCapsule(fv.f.Glyph, popupTitle(fv.f.Title, fv.f.Subject), Base, 1<<20))+4)
	keep := fv.focus
	for _, it := range append(fv.fieldItems(), fv.buttonItems()...) {
		fv.focus = it
		w = max(w, hintWidth(fv.hint())+6)
		if s := fv.bodyScroll(); s != "" && it.kind == itemContent {
			w = max(w, scrollRoom(fv.hint(), s))
		}
	}
	fv.focus = keep
	for _, row := range fv.actionRows() {
		n := len(row) - 1
		for _, it := range row {
			n += Width(fv.buttonLabel(it)) + 2
		}
		w = max(w, n+4)
	}
	return min(w, max(min(sw-2, 120), 1)), min(max(h, 0)+fv.tail()+4, max(sh-2, 3))
}

func (fv *formView) bodyDraw(w, h int, layer Color) []string {
	inner, n := w-2, max(h-4-fv.tail(), 0)
	if f, ok := fv.body.(interface{ Focus(on bool) }); ok {
		f.Focus(fv.focus.kind == itemContent)
	}
	rows := fv.body.Draw(inner, n, layer)
	body := make([]string, 0, max(h-4, 0))
	for i := range n {
		if i < len(rows) {
			body = append(body, Fit(rows[i], inner))
		} else {
			body = append(body, Repeat(" ", inner))
		}
	}
	if fv.checks() {
		body = append(body, Repeat(" ", inner), fv.errorRow(inner-1-Width(glyphHard), layer))
	}
	body = append(body, fv.actionArea(inner, layer)...)
	body = body[:min(len(body), max(h-4, 0))]
	scroll := ""
	if fv.focus.kind == itemContent {
		scroll = fv.bodyScroll() // on a button there is no place in the content to tell (tdp form.7)
	}
	return popupBox{color: layer, glyph: fv.f.Glyph, title: popupTitle(fv.f.Title, fv.f.Subject), hint: fv.hint(), scroll: scroll,
		loading: fv.isLoading()}.draw(w, body)
}

// contentKey is a key on it: Tab, Shift-Tab, Esc and q are the lib's, as on
// a form (tdp popup.8, K9); the others on the content are the body's, and on
// the buttons the action area's.
func (fv *formView) contentKey(a *App, key string, msg tea.KeyMsg) (bool, tea.Cmd) {
	defer fv.remember()
	switch key {
	case "Tab", "Shift-Tab":
		fv.tab()
		return true, nil
	case "Esc":
		return true, fv.escape(a)
	case "q":
		return true, nil // no quit flow from it (tdp K9)
	}
	if fv.focus.kind == itemContent {
		if done, cmd := fv.body.Key(key, msg); done {
			return true, cmd
		}
		return key != "?", nil
	}
	items := fv.area()
	switch key {
	case "j", "l", "↓", "→":
		fv.step(1)
	case "k", "h", "↑", "←":
		fv.step(-1)
	case "gg":
		fv.focus = items[0]
	case "G":
		fv.focus = items[len(items)-1]
	case "Enter":
		return true, fv.press(a, fv.focus)
	case "?":
		return false, nil
	}
	return true, nil
}

// bodyKeyRef is the body's keys, in the disabled colour while the focus is on
// a button, then the lib's: Enter on a button, Tab, Esc, and moving on the
// buttons when there are several (tdp K6, M6).
func (fv *formView) bodyKeyRef() []refSection {
	on := fv.focus.kind == itemContent
	var out []refSection
	for _, s := range fv.body.Keys() {
		rows := slices.Clone(s.Rows)
		for i := range rows {
			rows[i].Disabled = rows[i].Disabled || !on
		}
		out = append(out, refSection{title: s.Title, rows: rows})
	}
	if len(out) == 0 || out[0].title != "" {
		out = slices.Insert(out, 0, refSection{})
	}
	out[0].rows = append(out[0].rows,
		KeyRow{Key: "Enter", Desc: "on a button: press it", Disabled: on},
		KeyRow{Key: "Tab|Shift-Tab", Desc: "between the " + fv.contentName() + " and the buttons, back to where it was"},
		KeyRow{Key: "Esc", Desc: "cancel (asks first when something changed)"})
	if len(fv.buttons) > 1 {
		i := slices.IndexFunc(out, func(s refSection) bool { return s.title == "move" })
		if i < 0 {
			out, i = append(out, refSection{title: "move"}), len(out)
		}
		out[i].rows = append(out[i].rows,
			KeyRow{Key: "h|k", Desc: "on the buttons: the one before", Disabled: on},
			KeyRow{Key: "j|l", Desc: "on the buttons: the one after", Disabled: on},
			KeyRow{Key: "gg|G", Desc: "on the buttons: the first, the last", Disabled: on})
	}
	return out
}
