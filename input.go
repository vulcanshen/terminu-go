package terminu

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
)

// TextInput is a one-line text input popup (components/input/text): a
// prompt, the value, an error row.
type TextInput struct {
	Title string // what the popup is for, in its title ("Rename")
	Glyph string // its title glyph: the app's choice
	// Prompt is one line saying what to type ("New name for assets").
	Prompt string
	// Verb is what Enter does, for the hint ("rename").
	Verb string
	// Old is the value now: offered in grey after the cursor, Tab takes it
	// (components/input/README, the grey suggestion).
	Old string
	// MaxLen, when set, is the most characters the value can have; the
	// popup's width is then known when it opens.
	MaxLen int
	// Check is the value's own rules: an error is written on the error row
	// and the value is not written back.
	Check func(string) error
	// Done gets the new value. It is not called when the value did not change.
	Done func(string) tea.Cmd
}

// PasswordInput is a masked input (components/input/password).
type PasswordInput struct {
	Title, Glyph, Prompt, Verb string
	// Set says a password exists now: eight grey dots stand for it.
	Set bool
	// Unlock is the unlock look: one ● per character, a cell apart, centred
	// (a PIN); otherwise one • per character from the left.
	Unlock bool
	MaxLen int
	Check  func(string) error
	// Done gets the new password; "" with clear true when it was removed.
	Done func(value string, clear bool) tea.Cmd
}

// NumberInput is a number (components/input/number).
type NumberInput struct {
	Title, Glyph, Prompt, Verb string
	// Name is the field's name in the range error ("Port").
	Name     string
	Old      string
	Min, Max float64
	Negative bool // "-" may be typed
	Decimal  bool // "." may be typed
	Check    func(string) error
	Done     func(string) tea.Cmd
}

type inputKind int

const (
	kindText inputKind = iota
	kindPassword
	kindNumber
)

// lineInput is the one-line input popup all three share
// (components/input/README, the one-line skeleton).
type lineInput struct {
	kind         inputKind
	title, glyph string
	prompt, verb string
	old          string // the suggestion; for a password, whether one is set
	hasOld       bool
	dropOld      bool // Backspace on an empty value threw the suggestion away
	touched      bool
	maxLen       int
	unlock       bool
	ed           *lineEditor
	err          string
	check        func(string) error
	done         func(string, bool) tea.Cmd
	allow        func(rune) bool // the characters a number takes
	knownW       int
}

// Input is an input popup to open on its own (App.Open) or from a form's
// field: TextInput, PasswordInput, NumberInput, Select, Checks, and the rest.
type Input interface {
	// open opens it; written, when set, runs when a value is written back.
	open(a *App, written func()) tea.Cmd
	// typed reports a popup one types in: a form answers typing on its
	// field with a note (components/dialog/form).
	typed() bool
}

// Open opens an input popup.
func (a *App) Open(in Input) tea.Cmd { return in.open(a, nil) }

// Text opens a text input popup.
func (a *App) Text(in TextInput) tea.Cmd { return in.open(a, nil) }

// Password opens a password input popup.
func (a *App) Password(in PasswordInput) tea.Cmd { return in.open(a, nil) }

// Number opens a number input popup.
func (a *App) Number(in NumberInput) tea.Cmd { return in.open(a, nil) }

func (in TextInput) typed() bool     { return true }
func (in PasswordInput) typed() bool { return true }
func (in NumberInput) typed() bool   { return true }

// then runs written before what a value's Done returns.
func then(written func(), cmd tea.Cmd) tea.Cmd {
	if written != nil {
		written()
	}
	return cmd
}

func (in TextInput) open(a *App, written func()) tea.Cmd {
	return a.open(&lineInput{
		kind: kindText, title: in.Title, glyph: in.Glyph, prompt: in.Prompt, verb: cmpOr(in.Verb, "save"),
		old: in.Old, hasOld: in.Old != "", maxLen: in.MaxLen, ed: newEditor(""), check: in.Check,
		done: func(v string, _ bool) tea.Cmd { return then(written, run2(in.Done, v)) },
	})
}

func (in PasswordInput) open(a *App, written func()) tea.Cmd {
	return a.open(&lineInput{
		kind: kindPassword, title: in.Title, glyph: in.Glyph, prompt: in.Prompt, verb: cmpOr(in.Verb, "save"),
		hasOld: in.Set, maxLen: in.MaxLen, unlock: in.Unlock, ed: newEditor(""), check: in.Check,
		done: func(v string, clear bool) tea.Cmd {
			if in.Done == nil {
				return then(written, nil)
			}
			return then(written, in.Done(v, clear))
		},
	})
}

func (in NumberInput) open(a *App, written func()) tea.Cmd {
	li := &lineInput{
		kind: kindNumber, title: in.Title, glyph: in.Glyph, prompt: in.Prompt, verb: cmpOr(in.Verb, "save"),
		old: in.Old, hasOld: in.Old != "", ed: newEditor(""),
		done: func(v string, _ bool) tea.Cmd { return then(written, run2(in.Done, v)) },
	}
	li.allow = func(r rune) bool {
		return r >= '0' && r <= '9' || r == '-' && in.Negative || r == '.' && in.Decimal
	}
	if in.Max > in.Min {
		li.knownW = max(len(formatNum(in.Min)), len(formatNum(in.Max)))
	}
	li.check = func(v string) error {
		n, err := strconv.ParseFloat(v, 64)
		inRange := err == nil && (in.Max <= in.Min || n >= in.Min && n <= in.Max)
		if !inRange {
			name := cmpOr(in.Name, in.Title)
			if in.Max > in.Min {
				return fmt.Errorf("%s must be %s-%s", name, formatNum(in.Min), formatNum(in.Max))
			}
			return fmt.Errorf("%s must be a number", name)
		}
		if in.Check != nil {
			return in.Check(v)
		}
		return nil
	}
	return a.open(li)
}

func formatNum(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

func run2(f func(string) tea.Cmd, v string) tea.Cmd {
	if f == nil {
		return nil
	}
	return f(v)
}

// showingOld reports whether the grey suggestion (or the grey dots) is shown.
func (li *lineInput) showingOld() bool { return li.hasOld && !li.dropOld && li.ed.empty() }

func (li *lineInput) hint() []KeyHint {
	h := []KeyHint{{"Enter", li.verb}}
	switch {
	case li.kind == kindPassword:
		h = append(h, KeyHint{"Backspace", "clear"})
	case li.showingOld():
		h = append(h, KeyHint{"Tab", "edit"}, KeyHint{"Backspace", "clear"})
	}
	return append(h, KeyHint{"Esc", "cancel"})
}

func (li *lineInput) measure(sw, sh int) (int, int) {
	most := max(min(sw-2, 120), 1)
	w := most
	content := li.knownW
	if li.maxLen > 0 {
		content = li.maxLen
		if li.unlock {
			content = li.maxLen*2 - 1
		}
	}
	if content > 0 {
		need := max(content+1, Width(li.prompt)) + 4
		need = max(need, Width(titleCapsule(li.glyph, Upper(li.title), Base, 1<<20))+4)
		// The whole hint, with the suggestion's keys: only a typing row here,
		// so the hint lists every key (Rules M3) and the box holds all of it
		// (layout/popup, width).
		full := []KeyHint{{"Enter", li.verb}, {"Tab", "edit"}, {"Backspace", "clear"}, {"Esc", "cancel"}}
		if li.kind == kindPassword || !li.hasOld {
			full = li.hint()
			if li.kind != kindPassword {
				full = []KeyHint{{"Enter", li.verb}, {"Esc", "cancel"}}
			}
		}
		need = max(need, hintWidth(full)+6)
		w = min(need, most)
	}
	return w, min(9, max(sh-2, 3))
}

func (li *lineInput) draw(w, h int, layer Color) []string {
	inner := w - 2
	cw := inner - 2
	var value string
	valueStyle, cursor := Fg(Lavender), Style{Fg: Base, Bg: Lavender}
	switch li.kind {
	case kindPassword:
		value = li.maskView(cw, valueStyle, cursor)
	default:
		after := ""
		if li.showingOld() {
			after = Fg(Overlay0).Render(li.old)
		}
		value = li.ed.view(cw, valueStyle, cursor, true, nil, after)
	}
	errRow := Fg(Red).Render(Truncate(li.err, cw))
	if li.unlock {
		errRow = Center(errRow, cw)
	}
	body := []string{
		padRow(Fg(Overlay0).Render(li.prompt), inner),
		Repeat(" ", inner),
		" " + value + " ",
		Repeat(" ", inner),
		padRow(errRow, inner),
	}
	return popupBox{color: layer, glyph: li.glyph, title: Upper(li.title), hint: li.hint()}.draw(w, Pad2(body, inner, h-4))
}

// maskView draws a password: dots for what was typed, eight grey ones for a
// password that is set (components/input/password).
func (li *lineInput) maskView(w int, style, cursor Style) string {
	dot := "•"
	if li.unlock {
		dot = "●"
	}
	if li.showingOld() {
		grey := strings.Repeat(dot, 8)
		if li.unlock {
			return Center(Fg(Overlay0).Render(strings.TrimSpace(strings.Repeat(dot+" ", 8))), w)
		}
		return Pad(cursor.Render(" ")+Fg(Overlay0).Render(grey), w)
	}
	if li.unlock {
		dots := strings.TrimSpace(strings.Repeat(dot+" ", len(li.ed.g)))
		return Center(style.Render(dots), w)
	}
	return li.ed.view(w, style, cursor, true, func(string) string { return dot }, "")
}

func (li *lineInput) key(a *App, key string, msg tea.KeyMsg) (bool, tea.Cmd) {
	switch key {
	case "Enter":
		return true, li.confirm(a)
	case "Esc":
		return true, a.closeTop()
	case "Tab":
		if li.kind != kindPassword && li.showingOld() {
			li.ed.set(li.old)
			li.touched = true
		}
		return true, nil
	case "Backspace":
		switch {
		case li.kind == kindPassword && !li.ed.empty():
			li.ed.set("")
		case li.ed.empty() && li.hasOld:
			li.dropOld, li.touched = true, true
		default:
			li.ed.key(key, msg)
		}
		return true, nil
	}
	if li.kind == kindPassword && (key == "←" || key == "→" || key == "Home" || key == "End" || key == "Delete") {
		return true, nil // a password only grows at its end
	}
	if li.allow != nil && (msg.Type == tea.KeyRunes || msg.Paste) {
		var kept []rune
		for _, r := range msg.Runes {
			if li.allow(r) {
				kept = append(kept, r)
			}
		}
		if len(kept) == 0 {
			return true, nil
		}
		li.ed.insert(string(kept), false)
		li.touched = true
		return true, nil
	}
	if li.kind == kindPassword && (msg.Type == tea.KeyRunes || msg.Paste || key == "Space") && li.maxLen > 0 && len(li.ed.g) >= li.maxLen {
		return true, nil
	}
	if li.ed.key(key, msg) {
		li.touched = true
		if li.maxLen > 0 && len(li.ed.g) > li.maxLen {
			li.ed.g = li.ed.g[:li.maxLen]
			li.ed.pos = min(li.ed.pos, li.maxLen)
		}
	}
	return true, nil // the typing row takes every key (Rules K8)
}

// confirm is Enter on the value (Rules K3): checked, written back and closed;
// a wrong value stays with its error; an unchanged one just closes.
func (li *lineInput) confirm(a *App) tea.Cmd {
	if li.kind == kindPassword {
		switch {
		case li.ed.empty() && li.showingOld():
			return a.closeTop() // nothing typed: the old password stays
		case li.ed.empty():
			return tea.Batch(a.closeTop(), li.done("", true))
		}
		v := li.ed.value()
		if li.check != nil {
			if err := li.check(v); err != nil {
				li.err = err.Error()
				return nil
			}
		}
		return tea.Batch(a.closeTop(), li.done(v, false))
	}
	if li.showingOld() {
		return a.closeTop() // nothing touched: no change
	}
	if li.ed.hasBreaks() {
		li.err = cmpOr(li.title, "The value") + " can't have line breaks or tabs"
		return nil
	}
	v := strings.TrimFunc(li.ed.value(), unicode.IsSpace)
	if li.check != nil {
		if err := li.check(v); err != nil {
			li.err = err.Error()
			return nil
		}
	}
	li.err = ""
	if v == li.old && li.hasOld && !li.dropOld {
		return a.closeTop()
	}
	if !li.hasOld && v == "" && !li.touched {
		return a.closeTop()
	}
	return tea.Batch(a.closeTop(), li.done(v, false))
}

func (li *lineInput) surface() string { return li.title }

// keyRef: the typing row takes ?, so a one-line input's key reference never
// opens; its hint lists every key (Rules M3).
func (li *lineInput) keyRef() []refSection { return nil }
func (li *lineInput) typing() bool         { return true }

// ErrEmpty is a ready "may not be empty" check.
var ErrEmpty = errors.New("can't be empty")
