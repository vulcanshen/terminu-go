package terminu

import (
	"reflect"

	tea "github.com/charmbracelet/bubbletea"
)

// Popup is a popup of the app's own, for what none of the lib's popups holds
// (a grid to draw on). The app draws what is inside the frame and handles its
// keys; the lib draws the frame with the title and the hint in the layer
// colour, dims what is under it, opens and closes it with the animation
// (Rules F2, F8), and puts its own popups over it: a confirm, an input, an
// error popup, the key reference.
type Popup struct {
	// Title is what the popup is for, upper case on the frame; Subject is
	// the one thing it acts on, kept as it is after ": " ("DRAW: kamonu";
	// components/layout/popup). Glyph goes before the title.
	Title, Glyph, Subject string
	Body                  PopupBody
}

// PopupBody is the inside of a popup of the app's own. It must be comparable,
// a pointer most often: ClosePopup finds the popup by it.
//
// A body may also have QuitNote() string: what quitting now would lose
// ("Changes to kamonu are not saved."), "" for nothing. The quit flow's
// confirm puts it last, in Peach, and asks first even when App.Quit is nil
// (Rules K9: the quit flow is the app's).
//
// A body that scrolls may also have Scroll() string: its scroll position
// ("3 of 40", "1-20 of 120"), "" when everything fits. The lib draws it on
// the right of the bottom frame line in the layer colour and, when the popup
// opens, makes it wide enough for it and the whole hint
// (components/layout/popup, when the content does not fit).
type PopupBody interface {
	// Size is the inside on a sw × sh screen: w cells from frame line to
	// frame line, h rows between the blank rows above and below. It is asked
	// when the popup opens and when the terminal changes size; the lib makes
	// it wide enough for the title and the hint, and no bigger than the
	// screen allows (Rules F7).
	Size(sw, sh int) (w, h int)
	// Draw returns h lines of w cells; layer is the popup's colour. The cell
	// at either end is the blank margin, which only a full-row cursor covers
	// (components/layout/popup).
	Draw(w, h int, layer Color) []string
	// Key handles a key while the popup is on top and reports whether it
	// did. The lib does the keys it leaves: Esc closes the popup, ? opens its
	// key reference, q starts the quit flow (Rules K1). Ctrl-C never comes
	// here: it is always the quit flow (K9).
	Key(key string, msg tea.KeyMsg) (bool, tea.Cmd)
	// Hint is the hint on the bottom frame line now (Rules M5).
	Hint() []KeyHint
	// Keys is its key reference: only its own keys, Esc among them (Rules K6).
	Keys() []KeySection
}

// Popup opens p over what is on top. It panics when p.Body is nil or not
// comparable: that is a mistake in the program.
func (a *App) Popup(p Popup) tea.Cmd {
	if p.Body == nil || !reflect.TypeOf(p.Body).Comparable() {
		panic("terminu: popup " + p.Title + ": its Body must be a pointer or another comparable value")
	}
	return a.open(&appPopup{p: p})
}

// ClosePopup closes the popup b is the inside of, and every popup over it.
func (a *App) ClosePopup(b PopupBody) tea.Cmd {
	for _, l := range a.stack {
		if v, ok := l.view.(*appPopup); ok && v.p.Body == b && !l.closing {
			return a.closeView(v)
		}
	}
	return nil
}

// CloseAll closes every popup, the lib's and the app's: the top one
// animates, the rest go at once, as when what was done ends the whole
// stack (Rules T1; a long session started from a viewer). A confirm or an
// input closed this way is cancelled (OnCancel).
func (a *App) CloseAll() tea.Cmd { return a.closeAll() }

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
