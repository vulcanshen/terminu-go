package terminu

import (
	"errors"
	"slices"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const glyphCompare = "\U000F018A" // nf-md-compare: the Diff popup (tdp note.8)

// DiffMode is what a list's Diff mode needs from the app (tdp panel.9,
// note.8): the key, which rows can be compared and their text. The lib does
// the rest.
type DiffMode struct {
	// Key is the diff key, D in every app (tdp K13, [D]iff): on a row it enters
	// the mode with that row as the anchor; in the mode it opens the Diff,
	// as Enter does.
	Key string
	// Can reports a row that can be compared: text, shown as text, or one
	// that holds many (Many). A row it says no to has no Diff action, and
	// in the mode the diff keys are off on it (tdp M2, M6). Nil: every row.
	Can func(Row) bool
	// Name is the row as the comparison's first row names it; its first
	// cell when nil.
	Name func(Row) string
	// Text is the row's text, compared line by line.
	Text func(Row) (string, error)
	// Split does nothing.
	//
	// Deprecated: every comparison opens on UNIFIED and the tab chosen is
	// not kept (tdp note.8). The field stays so that code setting it still
	// builds; a later minor version takes it away.
	Split *bool
	// Many reports a row that holds many (a directory, a namespace, a
	// commit). Two such rows are not compared as text: the mode goes in to
	// the list of the items that differ between them, which Items gives, as
	// OpenDiff goes in (tdp panel.9), the location row <anchor> ↔ <the
	// other> as Name names them. A row that holds many and one that does
	// not are not compared: the diff keys are off (tdp M6). Nil: no row
	// holds many.
	Many func(Row) bool
	// Items gives the items that differ between two rows Many says yes to,
	// the anchor first, in the order they are listed, the ones the same on
	// both sides left out (DiffItem).
	Items func(anchor, other Row) ([]DiffItem, error)
	// InDetails shows the diff in the list's details panel (Panel.Details,
	// tdp panel.13) in place of opening it or going into the list (tdp
	// panel.9 "With a details panel"; which is the app's). The mode spans
	// the list and its details, both frames in the mode's colour with Diff
	// on them, the heavy line on the one with the focus, which entering
	// leaves on the list. The details follow the cursor, no key pressed: the
	// diff of the anchor and the row under the cursor, as a panel shows one
	// (Pager.SetDiff: UNIFIED; for two rows that hold many the items that
	// differ with their marks, the location row <anchor> ↔ <the other>); on
	// the anchor, or a row the same as it, an empty state saying so, no
	// toast; on a row it cannot diff with the anchor (Can says no, or one
	// holds many and the other not) the empty state "Can't diff <row> with
	// <anchor>"; a side that cannot be read is the details' error state, a
	// Red "<side>: <error>", no error popup (tdp panel.15). Key or Enter on the list sends the focus to the details; there
	// Key or Enter opens an item's diff or goes into it, as in the list.
	// Esc goes back the way the user came: the diff to the details, a level
	// up, the details to the list, the mode kept with its anchor, and on the
	// list out of the mode, the details showing the app's own content again.
	// Tab and the two panels' digits move between them; the other panels
	// wait (tdp K11). The panel's Details must be another panel of the
	// screen (AddScreen panics otherwise).
	InDetails bool
	// Noun is what a row is, as the details' empty state on the anchor names
	// it: "Move to another <Noun> to diff with <anchor>" (InDetails); "item"
	// when empty.
	Noun string
}

// Compare is DiffMode's old name.
//
// Deprecated: the mode is the Diff mode (tdp panel.9): use DiffMode. The
// name stays so that code using it still builds; a later minor version
// takes it away.
type Compare = DiffMode

func (c DiffMode) can(r Row) bool  { return c.Can == nil || c.Can(r) }
func (c DiffMode) many(r Row) bool { return c.Many != nil && c.Many(r) }

func (c DiffMode) name(r Row) string {
	if c.Name != nil {
		return c.Name(r)
	}
	if len(r.Cells) > 0 {
		return r.Cells[0]
	}
	return r.ID
}

// DiffMark is how an item differs, the anchor being the base (tdp
// panel.9): drawn before its name, in its colour.
type DiffMark byte

const (
	DiffAdded   DiffMark = '+' // only the other has it: Green
	DiffRemoved DiffMark = '-' // only the anchor has it: Red
	DiffChanged DiffMark = '~' // both have it, changed: Peach; a rename the app knows of too
)

func (m DiffMark) span() Span {
	switch m {
	case DiffAdded:
		return Span{"+", Fg(Green)}
	case DiffRemoved:
		return Span{"-", Fg(Red)}
	}
	return Span{"~", Fg(Peach)}
}

// DiffItem is an item that differs between two things that hold many (two
// directories, two namespaces, a commit and the one before it), or one
// item's two versions (tdp panel.9, note.8). The app names it and its two
// sides and gives what is in them; the lib draws the rest.
type DiffItem struct {
	// Mark is how the item differs. OpenDiff does not use it.
	Mark DiffMark
	// Name is the item as the list shows it after its mark, named as the
	// app names it (a path, internal/ui/ui.go); one the app knows was
	// renamed or moved is DiffChanged, named "<old> → <new>".
	Name string
	// Anchor and Other name the item's two sides as the app names them, so
	// that they say where, the older one the anchor (ui.go @ 1a2b3c4,
	// ui.go @ 5d6e7f8): its comparison, titled DIFF, has <Anchor> ↔ <Other>
	// for its first row, the same however it is reached, and the list it
	// goes into has <Anchor> ↔ <Other> for its location row, in place of
	// the panel's own (tdp panel.9).
	Anchor, Other string
	// Text gives the item's text on each side, "" on a side it is not on:
	// opening it opens its comparison, UNIFIED and SPLIT (tdp note.8).
	Text func() (anchor, other string, err error)
	// Items, for an item that holds many itself, gives the items that
	// differ in it: opening it goes in to them, as Compare.Items.
	Items func() ([]DiffItem, error)
	// With neither the item is not text: it has no comparison, Enter off on
	// it (tdp M6).
}

func (d DiffItem) opens() bool { return d.Text != nil || d.Items != nil }

// splitMin is the narrowest terminal SPLIT is drawn on, as the finder's
// side by side (tdp note.8).
const splitMin = 96

// comparing is a Diff mode under way.
type comparing struct {
	p    *Panel
	l    *List
	c    DiffMode
	mode *Mode
	name string // the anchor's
	// det is the list's details the diff is shown in (InDetails), nil for
	// none; detMode is the mode's layer there, at the row the details were
	// filled for ("\x00" + its ID, "" for none).
	det     *Panel
	detMode *Mode
	at      string
}

// DiffModeAction is l's Diff item action, for p's Actions: on a row c can
// compare it enters the Diff mode (tdp K11, panel.9), that row the anchor
// (List.Anchor). In the mode j|k, u|d and gg|G move; c.Key or Enter opens
// the Diff of the row under the cursor with the anchor, off on the anchor, on
// a row c cannot compare and on a row that holds many against one that
// does not (tdp M6); the same text on both sides is an Info toast, not a
// popup. Two rows that hold many go in to the list of the items that
// differ, as OpenDiff does, the panel staying in the mode: its keys work
// there, on the item under the cursor, and Esc goes back up a level, the
// last back to the mode with the anchor kept. Esc leaves it, the anchor
// gone; Tab and the digits wait; an anchor no longer in the list ends it
// with a Warning toast, the lists gone into closed. The rows need their
// IDs. The mode does not go in the App Log; its toasts do, as every toast
// (tdp note.7). With c.InDetails the diff is shown in p's Details instead
// (DiffMode.InDetails).
func (a *App) DiffModeAction(p *Panel, l *List, c DiffMode) Action {
	return Action{Key: c.Key, Label: "Diff", Desc: "with another row, picked next", Scope: ItemScope, inDetails: c.InDetails,
		Hidden: func() bool { r, ok := l.Current(); return !ok || !c.can(r) },
		Do: func() tea.Cmd {
			if r, ok := l.Current(); ok && c.can(r) {
				a.startCompare(p, l, c, r)
			}
			return nil
		}}
}

// CompareAction is DiffModeAction's old name.
//
// Deprecated: the mode is the Diff mode (tdp panel.9): use DiffModeAction.
// The name stays so that code using it still builds; a later minor version
// takes it away.
func (a *App) CompareAction(p *Panel, l *List, c Compare) Action { return a.DiffModeAction(p, l, c) }

func (a *App) startCompare(p *Panel, l *List, c DiffMode, anchor Row) {
	l.Anchor = anchor.ID
	cm := &comparing{p: p, l: l, c: c, name: c.name(anchor)}
	// in is the list of the items that differ gone into in the mode, nil
	// while in none: the mode's keys work on it.
	in := func() *diffLevel {
		if d := p.level(); d != nil && d.mode == cm.mode {
			return d
		}
		return nil
	}
	move := func(key string) tea.Cmd {
		if d := in(); d != nil {
			_, cmd := d.list.Key(key, tea.KeyMsg{})
			return cmd
		}
		_, cmd := l.Key(key, tea.KeyMsg{})
		return cmd
	}
	off := func() bool {
		if d := in(); d != nil {
			it, ok := d.current()
			return !ok || !it.opens()
		}
		r, ok := l.Current()
		an, found := l.rowByID(l.Anchor)
		return !ok || r.ID == l.Anchor || !c.can(r) || found && c.many(r) != c.many(an)
	}
	if c.InDetails && p.Details != nil && p.Details != p {
		cm.det = p.Details
		off = func() bool { // on the list: off where the details have nothing to go to
			o := cm.det.over
			return o == nil || o.mode != cm.detMode || !o.shows
		}
	}
	compare := ModeKey{Key: c.Key + "|Enter", Keys: []string{c.Key, "Enter"}, Desc: "diff", Disabled: off,
		Item: true, Label: "Diff", MenuDesc: "with " + cm.name,
		Do: func(string) tea.Cmd {
			if cm.det != nil { // the focus to the details (tdp panel.9)
				if s := a.Screen(); s != nil {
					if i := slices.Index(s.Panels, cm.det); i >= 0 {
						return a.focusPanel(s, i)
					}
				}
				return nil
			}
			if d := in(); d != nil {
				if it, ok := d.current(); ok {
					return a.openDiff(p, it)
				}
				return nil
			}
			return a.compareWith(cm)
		}}
	compare.menuDesc = func() string {
		if d := in(); d != nil {
			if it, _ := d.current(); it.Items != nil {
				return "the items that differ in it"
			}
			return "its two sides"
		}
		return "with " + cm.name
	}
	cm.mode = &Mode{Name: "Diff", Exit: "leave", Pause: true, Keys: []ModeKey{
		{Key: "j|k", Keys: []string{"j", "k", "↓", "↑"}, Desc: "down, up", NoFooter: true, Do: move},
		{Key: "u|d", Keys: []string{"u", "d", "PgUp", "PgDn"}, Desc: "half a page up, down", NoFooter: true, Do: move},
		{Key: "gg|G", Keys: []string{"gg", "G"}, Desc: "to the top, the bottom", NoFooter: true, Do: move},
		compare,
	}, OnExit: func() tea.Cmd { l.Anchor = ""; cm.leaveDetails(); return nil }}
	p.EnterMode(cm.mode)
	if cm.det != nil {
		cm.mode.with = cm.det
		cm.detMode = a.detailsMode(cm)
		cm.det.LeaveDiff() // what the app went into there is its own row's
		cm.det.EnterMode(cm.detMode)
		a.fillDetails(cm)
	}
	a.comparings = append(a.comparings, cm)
}

// detailsMode is the Diff mode's layer in the list's details (tdp panel.9
// "With a details panel"): its keys move there and open the item under its
// cursor, Esc going back to the list.
func (a *App) detailsMode(cm *comparing) *Mode {
	det, c := cm.det, cm.c
	// item is what Key or Enter opens there: the item under the cursor of
	// the list gone into or of the items that differ, or the two texts.
	item := func() (DiffItem, bool) {
		if d := det.level(); d != nil && d.mode == cm.detMode {
			return d.current()
		}
		o := det.over
		switch {
		case o == nil || o.mode != cm.detMode || !o.shows:
			return DiffItem{}, false
		case o.items != nil:
			return itemAt(o.body.(*List), o.items)
		}
		return o.pair, true
	}
	move := func(key string) tea.Cmd { _, cmd := det.body().Key(key, tea.KeyMsg{}); return cmd }
	still := func() bool { return len(det.body().MoveKeys()) == 0 } // nothing to move (tdp M6)
	open := ModeKey{Key: c.Key + "|Enter", Keys: []string{c.Key, "Enter"}, Desc: "diff", Item: true, Label: "Diff",
		Disabled: func() bool { it, ok := item(); return !ok || !it.opens() },
		Do: func(string) tea.Cmd {
			if it, ok := item(); ok {
				return a.openDiff(det, it)
			}
			return nil
		}}
	open.menuDesc = func() string {
		if it, _ := item(); it.Items != nil {
			return "the items that differ in it"
		}
		return "its two sides"
	}
	back := func() tea.Cmd {
		if s := a.Screen(); s != nil {
			if i := slices.Index(s.Panels, cm.p); i >= 0 {
				return a.focusPanel(s, i)
			}
		}
		return nil
	}
	return &Mode{Name: "Diff", Exit: "back", Pause: true, with: cm.p, back: back, backDesc: "back to the list, still in Diff", Keys: []ModeKey{
		{Key: "j|k", Keys: []string{"j", "k", "↓", "↑"}, Desc: "down, up", NoFooter: true, Disabled: still, Do: move},
		{Key: "u|d", Keys: []string{"u", "d", "PgUp", "PgDn"}, Desc: "half a page up, down", NoFooter: true, Disabled: still, Do: move},
		{Key: "gg|G", Keys: []string{"gg", "G"}, Desc: "to the top, the bottom", NoFooter: true, Disabled: still, Do: move},
		open,
	}}
}

// diffOver is a Diff mode's diff shown in a list's details, in place of its
// body: body is what it shows, acts the Enter its hint writes. shows is
// false for an empty state; items are body's when it lists the items that
// differ; pair is the two rows, for Enter on their texts.
type diffOver struct {
	mode  *Mode
	body  Body
	acts  []Action
	shows bool
	items []DiffItem
	pair  DiffItem
}

// fillDetails shows in cm's details the diff of the anchor and the row under
// the cursor, when the cursor is on another row than last time (tdp panel.9
// "With a details panel"): what was gone into there goes with the old row.
func (a *App) fillDetails(cm *comparing) {
	l, c, det := cm.l, cm.c, cm.det
	r, ok := l.Current()
	at := ""
	if ok {
		at = "\x00" + r.ID
	}
	if o := det.over; o != nil && o.mode == cm.detMode && at == cm.at {
		return
	}
	cm.at = at
	det.levels = slices.DeleteFunc(det.levels, func(d *diffLevel) bool { return d.mode == cm.detMode })
	o := &diffOver{mode: cm.detMode}
	det.over = o
	anchor, found := l.rowByID(l.Anchor)
	state := func(fact string, fail *Failure) { o.body = diffState{fact: fact, fail: fail} }
	switch {
	case !ok || !found:
		state("", nil)
		return
	case r.ID == anchor.ID:
		state("Move to another "+cmpOr(c.Noun, "item")+" to diff with "+cm.name, nil)
		return
	case !c.can(r) || c.many(r) != c.many(anchor):
		state("Can't diff "+c.name(r)+" with "+cm.name, nil) // an empty state saying so (tdp panel.9)
		return
	}
	o.pair = cm.pairItem(anchor, r)
	fail := func(err error) {
		side := o.pair.Anchor + " ↔ " + o.pair.Other
		if se := (sideErr{}); errors.As(err, &se) {
			side = se.name
		}
		// The details' error state (tdp panel.9, panel.15): no error popup,
		// which would open each time the cursor came by.
		state("", &Failure{Fact: side + ": " + err.Error()})
	}
	same := "No differences between " + o.pair.Anchor + " and " + o.pair.Other
	if o.pair.Items != nil {
		items, err := o.pair.Items()
		switch {
		case err != nil:
			fail(err)
		case len(items) == 0:
			state(same, nil)
		default:
			var list *List
			list, o.acts = diffList(o.pair, items, func(it DiffItem) tea.Cmd { return a.openDiff(det, it) })
			o.body, o.items, o.shows = list, items, true
		}
		return
	}
	x, y, err := o.pair.Text()
	switch {
	case err != nil:
		fail(err)
	case x == y:
		state(same, nil)
	default:
		pg := &Pager{}
		pg.SetDiff(x, y)
		o.body, o.shows = pg, true
		pair := o.pair
		o.acts = []Action{{Key: "Enter", Label: "Diff", Desc: "its two sides", Hint: "diff", Scope: PanelScope,
			Do: func() tea.Cmd { return a.openDiff(det, pair) }}}
	}
}

// leaveDetails takes the Diff mode out of its list's details: they show
// the app's own content again, following the cursor's own row.
func (cm *comparing) leaveDetails() {
	det := cm.det
	if det == nil {
		return
	}
	if det.mode == cm.detMode {
		det.LeaveMode()
	}
	if det.over != nil && det.over.mode == cm.detMode {
		det.over = nil
	}
}

// diffState is what a Diff mode's details show with no diff: a sentence
// in Overlay0, or a failure, in Red (tdp panel.9, panel.15).
type diffState struct {
	fact string
	fail *Failure
}

func (s diffState) Draw(w, h int, _ bool) []string         { return EmptyState(w, h, s.fact, "") }
func (s diffState) Key(string, tea.KeyMsg) (bool, tea.Cmd) { return false, nil }
func (s diffState) HasItem() bool                          { return false }
func (s diffState) Scroll() string                         { return "" }
func (s diffState) MoveKeys() []KeyRow                     { return nil }
func (s diffState) Empty() bool                            { return true }
func (s diffState) Failure() *Failure                      { return s.fail }

// sideErr is an error reading one side, named as the error popup names it.
type sideErr struct {
	name string
	err  error
}

func (e sideErr) Error() string { return e.err.Error() }

// compareWith compares the anchor with the row under the cursor: their
// comparison, or the list of the items that differ when both hold many.
func (a *App) compareWith(cm *comparing) tea.Cmd {
	l, c := cm.l, cm.c
	other, ok := l.Current()
	anchor, found := l.rowByID(l.Anchor)
	if !ok || !found || other.ID == anchor.ID || !c.can(other) || c.many(anchor) != c.many(other) {
		return nil
	}
	return a.openDiff(cm.p, cm.pairItem(anchor, other))
}

// pairItem is the anchor and another row as one item: the items that
// differ when they hold many, else their texts.
func (cm *comparing) pairItem(anchor, other Row) DiffItem {
	c := cm.c
	item := DiffItem{Anchor: c.name(anchor), Other: c.name(other)}
	if c.many(anchor) {
		item.Items = func() ([]DiffItem, error) {
			if c.Items == nil {
				return nil, nil
			}
			return c.Items(anchor, other)
		}
	} else {
		item.Text = func() (string, string, error) {
			at, err := c.Text(anchor)
			if err != nil {
				return "", "", sideErr{item.Anchor, err}
			}
			bt, err := c.Text(other)
			if err != nil {
				return "", "", sideErr{item.Other, err}
			}
			return at, bt, nil
		}
	}
	return item
}

// OpenDiff opens item as Enter does on it in a list of the items that
// differ (tdp note.8, panel.9), for an action of the app's own showing two
// versions of one item (a file in a commit against the commit before it, a
// file being edited against its last commit) or going into what changed in
// one (a commit against the one before it):
//   - with Text, its comparison, titled DIFF, its first row <Anchor> ↔
//     <Other>, folded and never cut (tdp note.8), or the Info toast "No
//     differences between <Anchor> and <Other>" when the two are the same;
//   - with Items, the list of the items that differ in it, in p in place
//     of p's body as panel.10 goes into an item, its location row
//     <Anchor> ↔ <Other> in place of p's own, or that toast when there is
//     none. Each item is its mark in its colour, then its name. j|k, u|d
//     and gg|G move, Enter opens the item under the cursor the same way
//     (off on one with neither Text nor Items), Esc goes back up a level,
//     the cursor on the item it came out of, the last back to p, its
//     cursor where it was; the Space menu and ? have the list's keys, not
//     p's actions, and Tab and the digits go to the other panels as ever,
//     the list kept. A filter on p is cleared before going in, as Esc
//     clears it (tdp panel.14).
//
// A comparison is as wide as the terminal but 2 (tdp F7) and opens on
// UNIFIED; SPLIT is off narrower than 96 columns. An
// error from Text or Items is an error popup. Mark is not used. p is out of
// a mode: a Diff mode goes in so itself (DiffModeAction).
func (a *App) OpenDiff(p *Panel, item DiffItem) tea.Cmd { return a.openDiff(p, item) }

func (a *App) openDiff(p *Panel, item DiffItem) tea.Cmd {
	subject := item.Name
	if subject == "" {
		subject = item.Anchor + " ↔ " + item.Other
	}
	fail := func(err error) tea.Cmd {
		if se := (sideErr{}); errors.As(err, &se) {
			return a.ErrorAbout("Diff", se.name, se.err.Error())
		}
		return a.ErrorAbout("Diff", subject, err.Error())
	}
	same := func() tea.Cmd { return a.Toast(Info, "No differences between "+item.Anchor+" and "+item.Other) }
	switch {
	case item.Items != nil:
		items, err := item.Items()
		if err != nil {
			return fail(err)
		}
		if len(items) == 0 {
			return same()
		}
		a.goInDiff(p, item, items)
	case item.Text != nil:
		at, bt, err := item.Text()
		if err != nil {
			return fail(err)
		}
		if at == bt {
			return same()
		}
		return a.diffViewer(item.Anchor, item.Other, at, bt)
	}
	return nil
}

// diffLevel is a list of the items that differ gone into in a panel, in
// place of its body (tdp panel.9): mode is the mode the panel was in, nil
// for none.
type diffLevel struct {
	list  *List
	items []DiffItem
	mode  *Mode
	acts  []Action
}

// current is the item under the list's cursor.
func (d *diffLevel) current() (DiffItem, bool) { return itemAt(d.list, d.items) }

// itemAt is the item of items under l's cursor, l listing them (diffList).
func itemAt(l *List, items []DiffItem) (DiffItem, bool) {
	r, ok := l.Current()
	if !ok {
		return DiffItem{}, false
	}
	i, _ := strconv.Atoi(r.ID)
	return items[i], true
}

// goInDiff goes into items, the items that differ in item, in p. It
// changes what p lists, so p's filter is cleared first, whether or not it
// hides the item (tdp panel.14): from OpenDiff and from a Diff mode
// alike, the app having no way to clear it before.
func (a *App) goInDiff(p *Panel, item DiffItem, items []DiffItem) {
	if l, ok := p.body().(*List); ok && l.Filtering() {
		l.ClearFilter()
	}
	d := &diffLevel{items: items, mode: p.mode}
	d.list, d.acts = diffList(item, items, func(it DiffItem) tea.Cmd { return a.openDiff(p, it) })
	p.levels = append(p.levels, d)
}

// diffList is the list of items, the items that differ in item (tdp
// panel.9): each its mark in its colour, then its name, the location row
// <Anchor> ↔ <Other>; acts are Enter on the item under its cursor, open.
func diffList(item DiffItem, items []DiffItem, open func(DiffItem) tea.Cmd) (*List, []Action) {
	l := &List{Columns: []Column{{Shrink: true, Path: true}}, Flags: true, Location: item.Anchor + " ↔ " + item.Other}
	rows := make([]Row, len(items))
	for i, it := range items {
		rows[i] = Row{ID: strconv.Itoa(i), Cells: []string{it.Name}, Flag: it.Mark.span()}
	}
	l.SetRows(rows)
	do := func() tea.Cmd {
		if it, ok := itemAt(l, items); ok {
			return open(it)
		}
		return nil
	}
	notText := func() bool { it, ok := itemAt(l, items); return !ok || it.Text == nil }
	notMany := func() bool { it, ok := itemAt(l, items); return !ok || it.Items == nil }
	return l, []Action{
		{Key: "Enter", Label: "Diff", Desc: "its two sides", Hint: "diff", Scope: ItemScope, Hidden: notText, Do: do},
		{Key: "Enter", Label: "Open", Desc: "the items that differ in it", Hint: "open", Scope: ItemScope, Hidden: notMany, Do: do},
	}
}

// diffViewer is the comparison of at, an's text, with bt, bn's (tdp
// note.8): on UNIFIED as it opens, the tab the user switches to kept while
// it is open only, SPLIT shown while there is room for it.
func (a *App) diffViewer(an, bn, at, bt string) tea.Cmd {
	split := false
	x, y := textLines(at), textLines(bt)
	ops := diffLines(x, y)
	// Titled DIFF, its tabs after it; its first row the two sides, folded
	// and never cut; as wide as the terminal but 2 (tdp note.8, F7).
	v := &Viewer{Title: "Diff", Glyph: glyphCompare, Tabs: []string{"Unified", "Split"}, NoNumbers: true, wide: true,
		head: func(w int) []string { return diffHead(an, bn, w) }}
	v.tabOff = func(i int) bool { return i == 1 && a.w < splitMin }
	// layout lays the lines out for a terminal sw wide, as it opens and
	// whenever its size changes: SPLIT when it is chosen and there is room,
	// the line on top kept by the op it shows (lineOp: each line's op).
	var lineOp []int
	lastW, lastTab := -1, -1
	layout := func(sw int) {
		tab := 0
		if split && sw >= splitMin {
			tab = 1
		}
		w := v.width(sw) - 4 // the viewer's text, numbers left out
		if w == lastW && tab == lastTab {
			return
		}
		keep := -1
		if v.top < len(lineOp) {
			keep = lineOp[v.top]
		}
		v.Tab = tab
		if tab == 1 {
			v.Lines, lineOp = diffSplit(ops, x, y, w)
		} else {
			v.Lines, lineOp = diffUnified(ops, x, y, w)
		}
		v.top, v.sub = 0, 0
		for i, o := range lineOp {
			if keep >= 0 && o >= keep {
				v.top = i
				break
			}
		}
		lastW, lastTab = w, tab
	}
	v.relayout = layout
	v.OnTab = func(tab int) tea.Cmd { split = tab == 1; layout(a.w); return nil }
	layout(a.w)
	return a.Viewer(v)
}

// diffHead is the comparison's first row, <an> ↔ <bn>, for w cells (tdp
// note.8): folded when it does not fit, never cut, before ↔ into two rows,
// the second starting with ↔, and a side still too wide on into the rows
// under it, after a / where it can be.
func diffHead(an, bn string, w int) []string {
	if one := an + " ↔ " + bn; Width(one) <= w {
		return []string{one}
	}
	return append(foldPath(an, w), foldPath("↔ "+bn, w)...)
}

// foldPath folds s into rows of w cells: each after the last / that fits,
// or where the row is full when none does.
func foldPath(s string, w int) []string {
	var rows []string
	for Width(s) > w {
		g := graphemes(s)
		fit, used := 0, 0
		for _, c := range g {
			if used+Width(c) > w {
				break
			}
			used += Width(c)
			fit++
		}
		for k := fit - 1; k > 0; k-- {
			if g[k] == "/" {
				fit = k + 1
				break
			}
		}
		fit = max(fit, 1) // a character wider than the row: a row of its own
		rows = append(rows, strings.Join(g[:fit], ""))
		s = strings.Join(g[fit:], "")
	}
	return append(rows, s)
}

// checkCompares ends a Diff mode whose anchor is no longer in its list,
// with a Warning toast, the lists gone into in it closed, and forgets the
// ones left, their details too; a mode shown in its list's details shows
// there the row now under the cursor.
func (a *App) checkCompares() tea.Cmd {
	var cmds []tea.Cmd
	kept := a.comparings[:0]
	for _, cm := range a.comparings {
		if cm.p.mode != cm.mode {
			cm.leaveDetails()
			continue
		}
		if _, ok := cm.l.rowByID(cm.l.Anchor); !ok {
			cm.p.LeaveMode()
			cm.l.Anchor = ""
			cm.leaveDetails()
			cmds = append(cmds, a.Toast(Warning, "Diff ended: "+cm.name+" is gone"))
			continue
		}
		if cm.det != nil {
			a.fillDetails(cm)
		}
		kept = append(kept, cm)
	}
	a.comparings = kept
	return tea.Batch(cmds...)
}

func textLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

// diffOp is one line of a diff: ' ' the same on both sides, '-' only in the
// anchor's, '+' only in the other's; a and b are its line in each, -1 when
// it is not there.
type diffOp struct {
	op   byte
	a, b int
}

// diffLines is the shortest edit from x to y, line by line.
func diffLines(x, y []string) []diffOp {
	pre := 0
	for pre < len(x) && pre < len(y) && x[pre] == y[pre] {
		pre++
	}
	suf := 0
	for suf < len(x)-pre && suf < len(y)-pre && x[len(x)-1-suf] == y[len(y)-1-suf] {
		suf++
	}
	var ops []diffOp
	for i := range pre {
		ops = append(ops, diffOp{' ', i, i})
	}
	ops = append(ops, myers(x[pre:len(x)-suf], y[pre:len(y)-suf], pre, pre)...)
	for i := range suf {
		ops = append(ops, diffOp{' ', len(x) - suf + i, len(y) - suf + i})
	}
	return ops
}

// myers is the shortest edit script from a to b (Myers 1986), its lines
// counted on from ao and bo.
func myers(a, b []string, ao, bo int) []diffOp {
	n, m := len(a), len(b)
	if n == 0 && m == 0 {
		return nil
	}
	most := n + m
	off := most + 1
	v := make([]int, 2*most+3)
	var trace [][]int // trace[d]: v around the diagonals -d-1..d+1 before step d
	for d := 0; d <= most; d++ {
		trace = append(trace, slices.Clone(v[off-d-1:off+d+2]))
		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || k != d && v[off+k-1] < v[off+k+1] {
				x = v[off+k+1]
			} else {
				x = v[off+k-1] + 1
			}
			y := x - k
			for x < n && y < m && a[x] == b[y] {
				x++
				y++
			}
			v[off+k] = x
			if x >= n && y >= m {
				return backtrack(trace, n, m, ao, bo)
			}
		}
	}
	return nil
}

func backtrack(trace [][]int, x, y, ao, bo int) []diffOp {
	var rev []diffOp
	for d := len(trace) - 1; d >= 0; d-- {
		v := trace[d]
		at := func(k int) int { return v[k+d+1] }
		k := x - y
		pk := k - 1
		if k == -d || k != d && at(k-1) < at(k+1) {
			pk = k + 1
		}
		px := at(pk)
		py := px - pk
		for x > px && y > py {
			x, y = x-1, y-1
			rev = append(rev, diffOp{' ', ao + x, bo + y})
		}
		if d > 0 {
			if x == px {
				y--
				rev = append(rev, diffOp{'+', -1, bo + y})
			} else {
				x--
				rev = append(rev, diffOp{'-', ao + x, -1})
			}
		}
		x, y = px, py
	}
	slices.Reverse(rev)
	return rev
}

// hunks are the stretches of ops a compare shows: each change with 3 lines
// either side, the ones that meet joined (tdp note.8).
func hunks(ops []diffOp) [][2]int {
	var out [][2]int
	for i, o := range ops {
		if o.op == ' ' {
			continue
		}
		from, to := max(i-3, 0), min(i+4, len(ops))
		if n := len(out); n > 0 && from <= out[n-1][1] {
			out[n-1][1] = max(out[n-1][1], to)
			continue
		}
		out = append(out, [2]int{from, to})
	}
	return out
}

func numWidthOf(x, y []string) int { return len(itoa(max(len(x), len(y), 1))) }

// diffUnified is UNIFIED: both line numbers, then - in Red, + in Green or a
// space, the hunks w cells wide with an Overlay0 line between; at is the op
// each line shows (a line between hunks, the one before the next's first).
func diffUnified(ops []diffOp, x, y []string, w int) (out []string, at []int) {
	nw := numWidthOf(x, y)
	num := func(i int) string {
		if i < 0 {
			return Repeat(" ", nw)
		}
		return PadLeft(itoa(i+1), nw)
	}
	for hi, h := range hunks(ops) {
		if hi > 0 {
			out, at = append(out, Fg(Overlay0).Render(Repeat("─", w))), append(at, h[0]-1)
		}
		for j := h[0]; j < h[1]; j++ {
			o := ops[j]
			at = append(at, j)
			nums := Fg(Overlay0).Render(num(o.a)+" "+num(o.b)) + " "
			switch o.op {
			case '-':
				out = append(out, nums+Fg(Red).Render("-"+x[o.a]))
			case '+':
				out = append(out, nums+Fg(Green).Render("+"+y[o.b]))
			default:
				out = append(out, nums+" "+x[o.a])
			}
		}
	}
	return out, at
}

// diffSplit is SPLIT: the anchor's lines on the left, the other's on the
// right, each half with its numbers. A line too long folds within its half
// as a viewer's does (tdp note.2), and the two across from each other take
// as many rows as the longer, the shorter filled with blank rows, so they
// start on the same row (tdp note.8). A line with nothing across from it
// faces an empty half; at is the op each row shows, as diffUnified's.
func diffSplit(ops []diffOp, x, y []string, w int) (out []string, at []int) {
	nw := numWidthOf(x, y)
	half := max((w-2)/2, nw+2)
	// side is line i's rows in its half, its number on the first.
	side := func(i int, lines []string, st Style) []string {
		if i < 0 {
			return nil
		}
		var rows []string
		for r, part := range foldText(strings.TrimSuffix(lines[i], "\r"), half-nw-1) {
			lead := Repeat(" ", nw+1)
			if r == 0 {
				lead = Fg(Overlay0).Render(PadLeft(itoa(i+1), nw)) + " "
			}
			rows = append(rows, Pad(lead+st.Render(part), half))
		}
		return rows
	}
	pair := func(op int, l, r []string) {
		for k := range max(len(l), len(r)) {
			left, right := Repeat(" ", half), Repeat(" ", half)
			if k < len(l) {
				left = l[k]
			}
			if k < len(r) {
				right = r[k]
			}
			out, at = append(out, left+"  "+right), append(at, op)
		}
	}
	for hi, h := range hunks(ops) {
		if hi > 0 {
			out, at = append(out, Fg(Overlay0).Render(Repeat("─", w))), append(at, h[0]-1)
		}
		var dels, adds []int // ops
		flush := func() {
			for k := range max(len(dels), len(adds)) {
				l, r, op := -1, -1, 0
				if k < len(dels) {
					l, op = ops[dels[k]].a, dels[k]
				} else {
					op = adds[k]
				}
				if k < len(adds) {
					r = ops[adds[k]].b
				}
				pair(op, side(l, x, Fg(Red)), side(r, y, Fg(Green)))
			}
			dels, adds = nil, nil
		}
		for j := h[0]; j < h[1]; j++ {
			o := ops[j]
			switch o.op {
			case '-':
				if len(adds) > 0 {
					flush()
				}
				dels = append(dels, j)
			case '+':
				adds = append(adds, j)
			default:
				flush()
				pair(j, side(o.a, x, Style{}), side(o.b, y, Style{}))
			}
		}
		flush()
	}
	return out, at
}

// foldText is a line as a viewer draws it (tabs as spaces, a control
// character by name) cut into the rows it folds onto at w cells, as the
// viewer folds it (tdp note.2): one row at least.
func foldText(s string, w int) []string {
	s = readingLine(s, tabCells)
	chars := graphemes(stripSGR(s))
	cells, space := make([]int, len(chars)), make([]bool, len(chars))
	for k, c := range chars {
		cells[k], space[k] = Width(c), c == " "
	}
	starts := foldAt(cells, space, w)
	rows := make([]string, len(starts))
	if !strings.Contains(s, "\x1b") { // plain: by its characters
		starts = append(starts, len(chars))
		for r := range rows {
			rows[r] = strings.Join(chars[starts[r]:starts[r+1]], "")
		}
		return rows
	}
	from := 0 // in cells
	for r := range starts {
		to := Width(s)
		if r+1 < len(starts) {
			to = 0
			for _, c := range cells[:starts[r+1]] {
				to += c
			}
		}
		rows[r] = Cut(s, from, to)
		from = to
	}
	return rows
}
