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
	// Subject is the one thing it acts on, kept as it is after ": " in the
	// title: "EDIT: filu" (components/layout/popup).
	Subject string
	Glyph   string // its title glyph: the app's choice
	// Prompt is one line saying what to type ("New name for assets").
	Prompt string
	// Verb is what Enter does, for the hint ("rename").
	Verb string
	// Name is the value's name in its errors when the title says what the
	// popup does rather than what the value is called: "Name can't have
	// control characters" on a "Save as" (components/input/README). Left
	// unset, the title names it.
	Name string
	// Old is the value now: offered in grey from the cursor on, Tab takes it
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
	// Subject is the one thing it acts on, kept as it is after ": " in the
	// title: "EDIT: filu" (components/layout/popup).
	Subject string
	// Name is the value's name in its errors when the title is not
	// ("Password" on a "Log in"); left unset, the title names it.
	Name string
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
	// Subject is the one thing it acts on, kept as it is after ": " in the
	// title: "EDIT: filu" (components/layout/popup).
	Subject string
	// Name is the value's name in its errors ("Port must be 1-65535"); left
	// unset, the title names it.
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
	subject      string
	name         string // the value's name in its errors: the app's Name, else the title
	prompt, verb string
	old          string // the suggestion; for a password, whether one is set
	hasOld       bool
	dropOld      bool // Backspace on an empty value threw the suggestion away
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
		kind: kindText, title: in.Title, name: cmpOr(in.Name, in.Title), subject: in.Subject, glyph: in.Glyph, prompt: in.Prompt, verb: cmpOr(in.Verb, "save"),
		old: in.Old, hasOld: in.Old != "", maxLen: in.MaxLen, ed: newEditor(""), check: in.Check,
		done: func(v string, _ bool) tea.Cmd { return then(written, run2(in.Done, v)) },
	})
}

func (in PasswordInput) open(a *App, written func()) tea.Cmd {
	return a.open(&lineInput{
		kind: kindPassword, title: in.Title, name: cmpOr(in.Name, in.Title), subject: in.Subject, glyph: in.Glyph, prompt: in.Prompt, verb: cmpOr(in.Verb, "save"),
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
		kind: kindNumber, title: in.Title, name: cmpOr(in.Name, in.Title), subject: in.Subject, glyph: in.Glyph, prompt: in.Prompt, verb: cmpOr(in.Verb, "save"),
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
		// The angle before the value, and the cursor after its last character.
		// The unlock look has no angle, and its cursor sits on the last ● once
		// all of it is typed (components/input/password).
		extra := angleWidth() + 1
		if li.unlock {
			extra = 0
		}
		if li.kind == kindPassword && li.hasOld {
			content = max(content, oldDotsWidth(li.unlock)) // the eight grey dots
		}
		need := max(content+extra, Width(li.prompt)) + 4
		need = max(need, Width(titleCapsule(li.glyph, popupTitle(li.title, li.subject), Base, 1<<20))+4)
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
	// The typing row starts with the angle, in the layer colour: a one-line
	// input is always typing (components/input/README). The unlock look has
	// none (components/input/password).
	var value string
	valueStyle, cursor := Fg(Lavender), Style{Fg: Base, Bg: Lavender}
	switch {
	case li.unlock:
		value = li.maskView(cw, valueStyle, cursor)
	case li.kind == kindPassword:
		value = typingAngle(layer) + li.maskView(cw-angleWidth(), valueStyle, cursor)
	default:
		suggest := ""
		if li.showingOld() {
			suggest = li.old
		}
		value = typingAngle(layer) + li.ed.view(cw-angleWidth(), valueStyle, cursor, true, nil, suggest)
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
	// Too short for all of it, the blank rows of the frame give way first
	// (fitRows); then the rows above the error row scroll to keep the typing
	// row in sight and the error row stays (components/layout/popup); with
	// one row, the typing row.
	if h-4 < len(body) {
		switch n := max(h-2, 0); {
		case n == 0:
			body = nil
		case n == 1:
			body = body[2:3]
		case n < len(body):
			top := scrollWindow(0, 2, n-1, 4)
			body = append(body[top:top+n-1:top+n-1], body[4])
		}
	}
	return popupBox{color: layer, glyph: li.glyph, title: popupTitle(li.title, li.subject), hint: li.hint()}.draw(w, Pad2(body, inner, max(h-4, len(body))))
}

// maskView draws a password: dots for what was typed, eight grey ones for a
// password that is set (components/input/password).
func (li *lineInput) maskView(w int, style, cursor Style) string {
	dot := "•"
	if li.unlock {
		dot = "●"
	}
	if li.showingOld() {
		// Eight dots from the cursor's cell on, the cursor on the first, as the
		// grey suggestion; the unlock look's a cell apart and centred.
		rest := strings.Repeat(dot, 7)
		if li.unlock {
			rest = strings.Repeat(" "+dot, 7)
			return Center(cursor.Render(dot)+Fg(Overlay0).Render(rest), w)
		}
		return Pad(cursor.Render(dot)+Fg(Overlay0).Render(rest), w)
	}
	if li.unlock {
		return Center(li.unlockDots(w, style, cursor), w)
	}
	return li.ed.view(w, style, cursor, true, func(string) string { return dot }, "")
}

// oldDotsWidth is how wide the eight grey dots of a password that is set are
// drawn: 8, or 15 a cell apart in the unlock look (components/input/password).
func oldDotsWidth(unlock bool) int {
	if unlock {
		return 15
	}
	return 8
}

// unlockDots is the unlock look's value, at most w cells: a ● a cell apart
// for each character, the cursor where the next one goes; once all of them
// are typed, the cursor is on the last ● (components/input/password). Too
// wide, it keeps the cursor's end.
func (li *lineInput) unlockDots(w int, style, cursor Style) string {
	n := len(li.ed.g)
	full := li.maxLen > 0 && n >= li.maxLen
	plain := strings.Repeat("● ", n) + " " // what is drawn, one cell a character
	if full {
		plain = strings.TrimSuffix(plain, "  ")
	}
	cells := []rune(plain)
	if len(cells) > w {
		cells = append([]rune{'…'}, cells[len(cells)-max(w-1, 1):]...)
	}
	last := len(cells) - 1
	return style.Render(string(cells[:last])) + cursor.Render(string(cells[last]))
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
		}
		return true, nil
	case "Backspace":
		switch {
		case li.kind == kindPassword && !li.ed.empty():
			li.ed.set("")
		case li.ed.empty() && li.hasOld:
			li.dropOld = true
		default:
			li.ed.key(key, msg)
		}
		return true, nil
	}
	if li.kind == kindPassword && (key == "←" || key == "→" || key == "Home" || key == "End" || key == "Delete") {
		return true, nil // a password only grows at its end
	}
	if li.allow != nil && (msg.Type == tea.KeyRunes || msg.Paste || key == "Space") {
		if msg.Alt && !msg.Paste {
			return true, nil // Alt- combinations type nothing
		}
		if key == "Space" {
			msg.Runes = []rune{' '}
		}
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
		return true, nil
	}
	if li.kind == kindPassword && (msg.Type == tea.KeyRunes || msg.Paste || key == "Space") && li.maxLen > 0 && len(li.ed.g) >= li.maxLen {
		return true, nil
	}
	if li.ed.key(key, msg) {
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
		if li.ed.hasControl() {
			li.err = cmpOr(li.name, "The value") + " can't have control characters"
			return nil
		}
		v := li.ed.value()
		if li.check != nil {
			if err := li.check(v); err != nil {
				li.err = checkMessage(li.name, err)
				return nil
			}
		}
		return tea.Batch(a.closeTop(), li.done(v, false))
	}
	if li.showingOld() {
		return a.closeTop() // nothing touched: no change
	}
	if li.ed.hasControl() {
		li.err = cmpOr(li.name, "The value") + " can't have control characters"
		return nil
	}
	v := strings.TrimFunc(li.ed.value(), unicode.IsSpace)
	if v == li.old {
		return a.closeTop() // the value it opened with: nothing to write back (Rules K3)
	}
	if li.check != nil {
		if err := li.check(v); err != nil {
			li.err = checkMessage(li.name, err)
			return nil
		}
	}
	li.err = ""
	return tea.Batch(a.closeTop(), li.done(v, false))
}

func (li *lineInput) surface() string { return li.title }

// keyRef: the typing row takes ?, so a one-line input's key reference never
// opens; its hint lists every key (Rules M3).
func (li *lineInput) keyRef() []refSection { return nil }
func (li *lineInput) typing() bool         { return true }

// ErrEmpty is a ready "may not be empty" check. The input names the value
// before it: "Name can't be empty" (components/input/README).
var ErrEmpty = errors.New("can't be empty")

// checkMessage is a check's error for the error row; ErrEmpty, which cannot
// know the value, gets its name in front, as the control character error has.
func checkMessage(name string, err error) string {
	if err == ErrEmpty {
		return cmpOr(name, "The value") + " " + err.Error()
	}
	return err.Error()
}
