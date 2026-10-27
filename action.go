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
// the hint, so they cannot drift apart (tdp M3).
type Action struct {
	// Key is the hotkey as KeyName writes it ("r", "D", "Alt-t", "go",
	// "Enter", "/"); "" for none. It may not take a core key or a movement
	// letter (tdp K1, K12); "Enter" is the panel's Enter action.
	Key string
	// Label is the action's name ("Rename"); the hotkey marker is added
	// where tdp M5 puts it.
	Label string
	// Desc is one line: what it does to what.
	Desc  string
	Scope Scope
	// Hint is the word after the key in the panel's bottom-border hint
	// ("logs" for Enter:logs); "" keeps the action out of the hint.
	Hint string
	// Ask, when set, asks first (tdp F6): every time.
	Ask *Ask
	// AskFor, when set, is asked when the action runs, after Hidden and
	// Disabled, for the question it asks then (the name of the item under the
	// cursor); nil asks nothing (a step that turns out to need no asking). It
	// wins over Ask. Run from a menu, the confirm opens over the menu, so Esc
	// on it goes back to the menu (tdp T1).
	AskFor func() *Ask
	// Hidden reports that its object does not exist now: the row is not
	// shown at all (tdp M6).
	Hidden func() bool
	// Disabled reports that it cannot run now: the row is shown in the
	// disabled colour, the cursor skips it and its hotkey does nothing.
	Disabled func() bool
	// Run from a menu, an action that opens a popup keeps the menu under it,
	// to come back to (tdp T1, F4); one that opens nothing has done the
	// menu's work, and the menu closes. A confirm before it is only a door:
	// Esc on it goes back to the menu, and once it is accepted what the action
	// does decides.
	//
	// Clear makes what it opens clear the menu, and the popups under it, as
	// it opens (tdp T1: the item is gone, a shell, another screen, a drill-down).
	Clear bool
	// Stay keeps the menu it runs from open even when it opens nothing.
	Stay bool
	// Detail marks a row whose Enter opens its full text over the menu (a
	// worklist, tdp F1): the menu stays and its hint is Enter:detail
	// (tdp menu).
	Detail bool
	// EnterHint is the word after Enter in a menu's hint while the cursor is
	// on this row ("cancel" on a work list's transfer, which asks first with
	// the row's full text); "" is run, or detail with Detail
	// (tdp menu.5).
	EnterHint string
	// Live, when set, gives the row's label and description each time the
	// menu is drawn: a work list's progress and state change while it is open
	// (tdp menu). The row keeps the width it had when the menu
	// opened.
	Live func() (label, desc string)
	// ToDetails, on a panel's "Enter" action only, says whether Enter on the
	// item under the cursor also sends the focus to the panel's Details: the
	// one place Enter changes panels, whether and on which items being the
	// app's call (tdp K3). Going in moves the focus, so it does not go in the
	// App Log (tdp note.7).
	ToDetails func() bool
	// Do runs it.
	Do func() tea.Cmd
	// show is a mode's movement key in its Space menu: shown, the cursor
	// passing it, never run (tdp menu.4).
	show bool
}

func (a Action) hidden() bool   { return a.Hidden != nil && a.Hidden() }
func (a Action) disabled() bool { return a.Disabled != nil && a.Disabled() }

// Ask is a confirm popup (tdp confirm): Enter accepts, Esc
// cancels.
type Ask struct {
	Title string // the action's name, after the shield glyph: "Delete"
	// Subject is the one thing it acts on, kept as it is after ": " in the
	// title: "EDIT: filu" (tdp popup).
	Subject  string
	Question string   // the last line: "Delete web-a8k2?"
	Verb     string   // Enter:<verb> in the hint: "delete"
	Details  []string // what to read before answering, above the question
	Warn     []int    // which lines of Details are warnings (Peach)
	// OnCancel runs when it closes with no answer: Esc, or the popup it is
	// over closing it, so whoever waits for the answer (a web page's prompt,
	// ssh's askpass) hears it was cancelled (tdp F6).
	OnCancel func() tea.Cmd
}
