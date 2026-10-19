package terminu

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
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
	// Suggest, when there is no Old, is a value the app proposes for a new
	// thing ("assets.zip"), offered in grey as Old is; Enter with nothing
	// touched takes it, checked and written back as any value
	// (components/input/README, a suggested value).
	Suggest string
	// MaxLen, when set, is the most characters the value can have; the
	// popup's width is then known when it opens.
	MaxLen int
	// Check is the value's own rules: an error is written on the error row
	// and the value is not written back.
	Check func(string) error
	// Search marks a value that is only searched for, never used: control
	// characters in it are drawn, not held back on Enter, as on a finder's
	// typing row (components/input/README, pasted control characters).
	Search bool
	// Done gets the new value. It is not called when the value did not change.
	Done func(string) tea.Cmd
	// Stay keeps it open under what Done opens: a step of a flow (current
	// PIN, new PIN, confirm PIN), each step a popup of its own, Esc on the
	// next one coming back here (Rules F1, F4). The app ends the flow with
	// CloseTop when its last step is done (T1).
	Stay bool
	// Commit, when set, writes a changed value back when Enter takes it, on
	// the update loop, before Done: an error keeps the popup open with the
	// value in it and opens an error popup over it, "RENAME FAILED"
	// (components/layout/popup, the action failing; Rules F4).
	Commit func(string) error
	// OnCancel runs when it closes with no answer: Esc, or the popup it is
	// over closing it, so whoever waits for the answer (a web page's prompt,
	// ssh's askpass) hears it was cancelled (Rules F6).
	OnCancel func() tea.Cmd
	// Unchanged, when set, runs when Enter takes the value it opened with, so
	// nothing is written back and Done is not called: a step of a flow ends
	// the flow (CloseTop), a page waiting for an answer gets the value it
	// offered (Rules K3, F6).
	Unchanged func() tea.Cmd
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
	// (a PIN), and no prompt row: the title says what to type (UNLOCK,
	// CURRENT PIN); otherwise one • per character from the left.
	Unlock bool
	MaxLen int
	Check  func(string) error
	// PausedUntil, when set, is asked when the popup opens and after each
	// Enter: until the time it gives, the popup takes no value, the error row
	// counting down each second ("try again in 27 s") and every key but Esc
	// swallowed (a PIN wrong too many times; components/layout/popup, the
	// error row).
	PausedUntil func() time.Time
	// Done gets the new password; "" with clear true when it was removed.
	Done func(value string, clear bool) tea.Cmd
	// Stay keeps it open under what Done opens: a step of a flow (current
	// PIN, new PIN, confirm PIN), each step a popup of its own, Esc on the
	// next one coming back here (Rules F1, F4). The app ends the flow with
	// CloseTop when its last step is done (T1).
	Stay bool
	// Commit, when set, writes a changed value back when Enter takes it, on
	// the update loop, before Done: an error keeps the popup open with the
	// value in it and opens an error popup over it, "RENAME FAILED"
	// (components/layout/popup, the action failing; Rules F4).
	Commit func(value string, clear bool) error
	// OnCancel runs when it closes with no answer: Esc, or the popup it is
	// over closing it, so whoever waits for the answer (a web page's prompt,
	// ssh's askpass) hears it was cancelled (Rules F6).
	OnCancel func() tea.Cmd
	// Unchanged, when set, runs when Enter takes the value it opened with, so
	// nothing is written back and Done is not called: a step of a flow ends
	// the flow (CloseTop), a page waiting for an answer gets the value it
	// offered (Rules K3, F6).
	Unchanged func() tea.Cmd
}

// NumberInput is a number (components/input/number).
type NumberInput struct {
	Title, Glyph, Prompt, Verb string
	// Subject is the one thing it acts on, kept as it is after ": " in the
	// title: "EDIT: filu" (components/layout/popup).
	Subject string
	// Name is the value's name in its errors ("Port must be 1-65535"); left
	// unset, the title names it.
	Name string
	Old  string
	// Suggest is a value the app proposes when there is no Old, as on a
	// TextInput.
	Suggest  string
	Min, Max float64
	Negative bool // "-" may be typed
	Decimal  bool // "." may be typed
	// Optional lets the value be empty: an empty one is not checked against
	// the range and is written back empty; what it means the form or panel
	// says in grey (components/input/number). Otherwise empty is an error.
	Optional bool
	// Unit is what the number counts, the whole word ("seconds"), in grey on
	// a row of its own under the value (components/input/number).
	Unit  string
	Check func(string) error
	Done  func(string) tea.Cmd
	// Stay keeps it open under what Done opens: a step of a flow (current
	// PIN, new PIN, confirm PIN), each step a popup of its own, Esc on the
	// next one coming back here (Rules F1, F4). The app ends the flow with
	// CloseTop when its last step is done (T1).
	Stay bool
	// Commit, when set, writes a changed value back when Enter takes it, on
	// the update loop, before Done: an error keeps the popup open with the
	// value in it and opens an error popup over it, "RENAME FAILED"
	// (components/layout/popup, the action failing; Rules F4).
	Commit func(string) error
	// OnCancel runs when it closes with no answer: Esc, or the popup it is
	// over closing it, so whoever waits for the answer (a web page's prompt,
	// ssh's askpass) hears it was cancelled (Rules F6).
	OnCancel func() tea.Cmd
	// Unchanged, when set, runs when Enter takes the value it opened with, so
	// nothing is written back and Done is not called: a step of a flow ends
	// the flow (CloseTop), a page waiting for an answer gets the value it
	// offered (Rules K3, F6).
	Unchanged func() tea.Cmd
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
	old          string // the value it opened with
	hasOld       bool   // there is one: offered in grey; for a password, one is set
	suggest      string // the app's proposal when there is no old value, in grey
	dropOld      bool   // Backspace on an empty value threw the grey value away
	unit         string // a number's unit, on the row under the value
	pausedUntil  func() time.Time
	pause        time.Time // until when it takes no value
	search       bool      // only searched for: control characters are not held back
	onCancel     func() tea.Cmd
	unchanged    func() tea.Cmd                   // Enter on the value it opened with
	commit       func(v string, clear bool) error // nil without Commit
	stay         bool                             // a step of a flow: Enter leaves it open under the next
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
		old: in.Old, hasOld: in.Old != "", suggest: proposal(in.Old, in.Suggest), maxLen: in.MaxLen, ed: newEditor(""), check: in.Check, search: in.Search, onCancel: in.OnCancel, unchanged: in.Unchanged, stay: in.Stay,
		commit: func(v string, _ bool) error { return runCommit(in.Commit, v) },
		done:   func(v string, _ bool) tea.Cmd { return then(written, run2(in.Done, v)) },
	})
}

func (in PasswordInput) open(a *App, written func()) tea.Cmd {
	li := &lineInput{
		kind: kindPassword, title: in.Title, name: cmpOr(in.Name, in.Title), subject: in.Subject, glyph: in.Glyph, prompt: in.Prompt, verb: cmpOr(in.Verb, "save"),
		hasOld: in.Set, maxLen: in.MaxLen, unlock: in.Unlock, ed: newEditor(""), check: in.Check, onCancel: in.OnCancel, unchanged: in.Unchanged, stay: in.Stay, pausedUntil: in.PausedUntil,
		commit: func(v string, clear bool) error {
			if in.Commit == nil {
				return nil
			}
			return in.Commit(v, clear)
		},
		done: func(v string, clear bool) tea.Cmd {
			if in.Done == nil {
				return then(written, nil)
			}
			return then(written, in.Done(v, clear))
		},
	}
	return tea.Batch(a.open(li), li.paused())
}

func (in NumberInput) open(a *App, written func()) tea.Cmd {
	li := &lineInput{
		kind: kindNumber, title: in.Title, name: cmpOr(in.Name, in.Title), subject: in.Subject, glyph: in.Glyph, prompt: in.Prompt, verb: cmpOr(in.Verb, "save"),
		old: in.Old, hasOld: in.Old != "", suggest: proposal(in.Old, in.Suggest), unit: in.Unit, ed: newEditor(""), onCancel: in.OnCancel, unchanged: in.Unchanged, stay: in.Stay,
		commit: func(v string, _ bool) error { return runCommit(in.Commit, v) },
		done:   func(v string, _ bool) tea.Cmd { return then(written, run2(in.Done, v)) },
	}
	li.allow = func(r rune) bool {
		return r >= '0' && r <= '9' || r == '-' && in.Negative || r == '.' && in.Decimal
	}
	if in.Max > in.Min {
		li.knownW = max(len(formatNum(in.Min)), len(formatNum(in.Max)))
	}
	li.check = func(v string) error {
		if v == "" {
			// Empty is a value of its own: not a number out of range
			// (components/input/number).
			if !in.Optional {
				return ErrEmpty
			}
			if in.Check != nil {
				return in.Check(v)
			}
			return nil
		}
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

// proposal is the app's proposed value, offered only when there is no old
// one: the two never show at once (components/input/README).
func proposal(old, suggest string) string {
	if old != "" {
		return ""
	}
	return suggest
}

func run2(f func(string) tea.Cmd, v string) tea.Cmd {
	if f == nil {
		return nil
	}
	return f(v)
}

// showingOld reports whether the grey suggestion (or the grey dots) is shown:
// the old value or the app's proposal.
func (li *lineInput) showingOld() bool {
	return (li.hasOld || li.suggest != "") && !li.dropOld && li.ed.empty()
}

// grey is the value the grey suggestion offers.
func (li *lineInput) grey() string { return cmpOr(li.old, li.suggest) }

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
		if li.unit != "" {
			content = max(content, Width(li.unit)) // the unit row lines up with the value
		}
		prompt := Width(li.prompt)
		if li.unlock {
			prompt = 0 // no prompt row (components/input/password)
		}
		need := max(content+extra, prompt) + 4
		need = max(need, Width(titleCapsule(li.glyph, popupTitle(li.title, li.subject), Base, 1<<20))+4)
		// The whole hint, with the suggestion's keys: only a typing row here,
		// so the hint lists every key (Rules M3) and the box holds all of it
		// (layout/popup, width).
		full := []KeyHint{{"Enter", li.verb}, {"Tab", "edit"}, {"Backspace", "clear"}, {"Esc", "cancel"}}
		if li.kind == kindPassword || !li.hasOld && li.suggest == "" {
			full = li.hint()
			if li.kind != kindPassword {
				full = []KeyHint{{"Enter", li.verb}, {"Esc", "cancel"}}
			}
		}
		need = max(need, hintWidth(full)+6)
		w = min(need, most)
	}
	frame := 4 // the borders and a blank row inside each
	if li.unlock {
		frame = 2 // no padding (components/input/password)
	}
	tall := max(sh-2, 3)
	return w, min(len(li.rows(w-2, "", tall-frame))+frame, tall)
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
			suggest = li.grey()
		}
		value = typingAngle(layer) + li.ed.view(cw-angleWidth(), valueStyle, cursor, true, nil, suggest)
	}
	body := li.rows(inner, " "+value+" ", h-4)
	box := popupBox{color: layer, glyph: li.glyph, title: popupTitle(li.title, li.subject), hint: li.hint()}
	if li.unlock {
		// No padding below the error row: two blank rows, the typing row in
		// the middle, a blank row, the error row; 7 rows with the frame
		// (components/input/password). Too short, the blank rows go first
		// (fitRows).
		box.bare = true
		return box.draw(w, body)
	}
	// The typing row comes after the prompt and a blank row; a unit, a blank
	// row and the error row after it.
	typing, errAt := len(body)-3-btoi(li.unit != ""), len(body)-1
	// Too short for all of it, the blank rows of the frame give way first
	// (fitRows); then the rows above the error row scroll to keep the typing
	// row in sight and the error row stays (components/layout/popup); with
	// one row, the typing row.
	if h-4 < len(body) {
		switch n := max(h-2, 0); {
		case n == 0:
			body = nil
		case n == 1:
			body = body[typing : typing+1]
		case n < len(body):
			top := scrollWindow(0, typing, n-1, errAt)
			body = append(body[top:top+n-1:top+n-1], body[errAt])
		}
	}
	return box.draw(w, Pad2(body, inner, max(h-4, len(body))))
}

// rows are the body of a one-line input inner cells wide, value being its
// typing row: the prompt, a blank row, the typing row, a number's unit, a
// blank row, the error row (components/input/README); the unlock look has two
// blank rows instead of the prompt and padding (components/input/password).
func (li *lineInput) rows(inner int, value string, most int) []string {
	cw := inner - 2
	errRow := Fg(Red).Render(Truncate(li.err, cw))
	blank := Repeat(" ", inner)
	if li.unlock {
		return []string{blank, blank, value, blank, padRow(Center(errRow, cw), inner)}
	}
	var body []string
	for _, l := range li.promptRows(cw, most) {
		body = append(body, padRow(Fg(Overlay0).Render(l), inner))
	}
	body = append(body, blank, value)
	if li.unit != "" {
		// Under the value's first cell, not moving as it is typed.
		body = append(body, padRow(Repeat(" ", angleWidth())+Fg(Overlay0).Render(li.unit), inner))
	}
	return append(body, blank, padRow(errRow, inner))
}

// promptRows is the prompt folded to cw cells (components, wrapping), several
// sentences a web page's prompt gives, its own line breaks too: as many rows
// as the popup has room for beside its other rows, at least one; the last one
// kept ends in "…" when not all fit (components/input/README).
func (li *lineInput) promptRows(cw, most int) []string {
	rows := Wrap(li.prompt, max(cw, 1))
	if len(rows) == 0 {
		rows = []string{""}
	}
	others := 4 // a blank row, the typing row, a blank row, the error row
	if li.unit != "" {
		others++
	}
	if n := max(most-others, 1); len(rows) > n {
		rows = rows[:n]
		last := Truncate(rows[n-1]+" …", cw)
		if !strings.HasSuffix(last, "…") {
			last = Truncate(last+"…", cw)
		}
		rows[n-1] = last
	}
	return rows
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
	if key != "Esc" && time.Now().Before(li.pause) {
		return true, nil // paused: every key but Esc swallowed
	}
	switch key {
	case "Enter":
		return true, li.confirm(a)
	case "Esc":
		return true, a.closeTop()
	case "Tab":
		if li.kind != kindPassword && li.showingOld() {
			li.ed.set(li.grey())
		}
		return true, nil
	case "Backspace":
		switch {
		case li.kind == kindPassword && !li.ed.empty():
			li.ed.set("")
		case li.ed.empty() && (li.hasOld || li.suggest != ""):
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
// a wrong value stays with its error. The value it opened with just closes,
// but only when it passes: one that opened empty and may not be, Enter with
// nothing typed is an error (components/input/README).
func (li *lineInput) confirm(a *App) tea.Cmd {
	if li.kind == kindPassword {
		if li.ed.empty() && li.showingOld() {
			return li.same(a) // nothing typed: the old password stays
		}
		if li.ed.hasControl() {
			li.err = cmpOr(li.name, "The value") + " can't have control characters"
			return nil
		}
		v := li.ed.value()
		if !li.checked(v) {
			return li.paused()
		}
		switch {
		case v == "" && li.hasOld:
			return li.write(a, "", true) // Backspace threw the old one away
		case v == "":
			return li.same(a) // empty it opened, empty it closes
		}
		return li.write(a, v, false)
	}
	if li.showingOld() && li.suggest == "" {
		return li.same(a) // nothing touched: no change
	}
	if li.ed.hasControl() && !li.search {
		li.err = cmpOr(li.name, "The value") + " can't have control characters"
		return nil
	}
	v := strings.TrimFunc(li.ed.value(), unicode.IsSpace)
	if li.showingOld() {
		v = li.suggest // nothing touched: the app's proposal is taken
	}
	if !li.checked(v) {
		return nil
	}
	if v == li.old {
		return li.same(a) // the value it opened with: nothing to write back (Rules K3)
	}
	return li.write(a, v, false)
}

// same is Enter on the value it opened with: closed, nothing written back,
// the app told when it wants to be (Unchanged).
func (li *lineInput) same(a *App) tea.Cmd { return tea.Batch(a.answer(), run(li.unchanged)) }

// checked runs the value's check: its error goes on the error row, or the
// row is cleared.
func (li *lineInput) checked(v string) bool {
	if li.check != nil {
		if err := li.check(v); err != nil {
			li.err = checkMessage(li.name, err)
			return false
		}
	}
	li.err = ""
	return true
}

// pauseTick counts a paused input down, a second at a time.
type pauseTick struct{ li *lineInput }

// paused starts the pause the app asks for, if it asks for one now: the
// error row counts down (components/layout/popup, the error row).
func (li *lineInput) paused() tea.Cmd {
	if li.pausedUntil == nil {
		return nil
	}
	li.pause = li.pausedUntil()
	return li.countDown()
}

// countDown writes what is left of the pause on the error row and ticks
// again in a second; once it is over it clears the row.
func (li *lineInput) countDown() tea.Cmd {
	left := time.Until(li.pause)
	if left <= 0 {
		if strings.HasPrefix(li.err, "try again in ") {
			li.err = ""
		}
		return nil
	}
	li.err = fmt.Sprintf("try again in %d s", int(math.Ceil(left.Seconds())))
	next := left - left.Truncate(time.Second)
	if next == 0 {
		next = time.Second
	}
	return tea.Tick(next, func(time.Time) tea.Msg { return pauseTick{li} })
}

// write writes a value Enter took back: Commit first, then Done.
func (li *lineInput) write(a *App, v string, clear bool) tea.Cmd {
	if err := li.commit(v, clear); err != nil {
		return a.commitFailed(li.title, li.subject, err)
	}
	if li.stay {
		// A step of a flow: answered, and open under the next step.
		if l := a.top(); l != nil {
			l.answered = true
		}
		return li.done(v, clear)
	}
	return tea.Batch(a.answer(), li.done(v, clear))
}

func (li *lineInput) surface() string { return li.title }

func (li *lineInput) cancel() tea.Cmd { return run(li.onCancel) }

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
