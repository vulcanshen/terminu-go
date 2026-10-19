package terminu

import tea "github.com/charmbracelet/bubbletea"

// Scope is what an action works on (Principle P3).
type Scope int

const (
	ItemScope   Scope = iota // the one item under the cursor
	PanelScope               // the focused panel, or its tab, as a whole
	GlobalScope              // the whole app
)

// Action is something the user can do. One registration gives the Space menu
// row, the global operation popup row, the key reference row, the hotkey and
// the hint, so they cannot drift apart (Rules M3).
type Action struct {
	// Key is the hotkey as KeyName writes it ("r", "D", "Alt-t", "go",
	// "Enter", "/"); "" for none. It may not take a core key or a movement
	// letter (Rules K1, K12); "Enter" is the panel's Enter action.
	Key string
	// Label is the action's name ("Rename"); the hotkey marker is added
	// where Rules M5 puts it.
	Label string
	// Desc is one line: what it does to what.
	Desc  string
	Scope Scope
	// Hint is the word after the key in the panel's bottom-border hint
	// ("logs" for Enter:logs); "" keeps the action out of the hint.
	Hint string
	// Ask, when set, asks first (Rules F6): every time.
	Ask *Ask
	// Hidden reports that its object does not exist now: the row is not
	// shown at all (Rules M6).
	Hidden func() bool
	// Disabled reports that it cannot run now: the row is shown in the
	// disabled colour, the cursor skips it and its hotkey does nothing.
	Disabled func() bool
	// Stay keeps the menu it runs from open (a step of its own over it).
	Stay bool
	// Detail marks a row whose Enter opens its full text over the menu (a
	// worklist, Rules F1): the menu stays and its hint is Enter:detail
	// (components/dialog/menu).
	Detail bool
	// ToDetails, on a panel's "Enter" action only, says whether Enter on the
	// item under the cursor also sends the focus to the panel's Details: the
	// one place Enter changes panels, whether and on which items being the
	// app's call (Rules K3).
	ToDetails func() bool
	// Do runs it.
	Do func() tea.Cmd
}

func (a Action) hidden() bool   { return a.Hidden != nil && a.Hidden() }
func (a Action) disabled() bool { return a.Disabled != nil && a.Disabled() }

// Ask is a confirm popup (components/dialog/confirm): Enter accepts, Esc
// cancels.
type Ask struct {
	Title string // the action's name, after the shield glyph: "Delete"
	// Subject is the one thing it acts on, kept as it is after ": " in the
	// title: "EDIT: filu" (components/layout/popup).
	Subject  string
	Question string   // the last line: "Delete web-a8k2?"
	Verb     string   // Enter:<verb> in the hint: "delete"
	Details  []string // what to read before answering, above the question
	Warn     []int    // which lines of Details are warnings (Peach)
	// OnCancel runs when it closes with no answer: Esc, or the popup it is
	// over closing it, so whoever waits for the answer (a web page's prompt,
	// ssh's askpass) hears it was cancelled (Rules F6).
	OnCancel func() tea.Cmd
}
