package terminu

import (
	"path"
	"path/filepath"
	"slices"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

// FinderItem is one row of a Finder.
type FinderItem struct {
	ID   string // the app's own key for it
	Text string // shown, and matched unless Match is set
	// Match, when set, is what the typed words are matched against instead
	// of Text: the item's other content, a file's text (components/input/finder:
	// what is matched is the app's choice).
	Match string
	// Icon, when set, goes before the text in IconColor (a file's type).
	Icon      string
	IconColor Color
	// Line, with Files, is a text file's line a search found: the preview
	// starts near it.
	Line int
}

// Finder is a list with a search that goes somewhere (components/input/finder):
// type to narrow the list, Enter on a row to go there, the row under the
// cursor previewed beside it. Keep the pointer: rows may keep arriving —
// append to Items while Loading is true.
type Finder struct {
	Title, Glyph string
	// Subject is the one thing it acts on, kept as it is after ": " in the
	// title: "EDIT: filu" (components/layout/popup).
	Subject string
	// Verb is what Enter on the list does, for the hint ("open").
	Verb    string
	Items   []FinderItem
	Loading bool
	// Preview is the row under the cursor: a title (its name, or where it
	// is) and the lines to show.
	Preview func(FinderItem) (title string, lines []string)
	// Files says the items are files and directories, their IDs the paths:
	// the preview shows them as a file-picker does, with their facts under
	// it (components/input/file-picker), instead of Preview.
	Files bool
	// FS is where Files reads the paths: the local disk when nil.
	FS PreviewFS
	// Search, when set, is a search the app runs itself (a search tool, a
	// remote host), answering a batch at a time: it gets the typed words each
	// time they change and puts what it finds in Items as it comes, Loading
	// meanwhile; the list is then Items as they are, not filtered again. It
	// must not wait for the search: typing goes on while it runs
	// (components/input/finder).
	Search func(typed string) tea.Cmd
	Done   func(FinderItem) tea.Cmd
}

// Select picks one value from a list (components/input/select).
type Select struct {
	Title, Glyph string
	// Subject is the one thing it acts on, kept as it is after ": " in the
	// title: "EDIT: filu" (components/layout/popup).
	Subject string
	Options []string
	// Keys, when set, are the options' hotkeys (Rules M5); "" for none.
	Keys []string
	// Selected is the value now; -1 for none.
	Selected int
	// Loading, for a select opened with LoadSelect, is true while Options
	// are still coming.
	Loading bool
	// Done gets the chosen option. It is not called when the value did not
	// change.
	Done func(int) tea.Cmd
	// Commit, when set, writes a changed value back when Enter takes it, on
	// the update loop, before Done: an error keeps the popup open with the
	// value in it and opens an error popup over it, "RENAME FAILED"
	// (components/layout/popup, the action failing; Rules F4).
	Commit func(int) error
	// OnCancel runs when it closes with no answer: Esc, or the popup it is
	// over closing it, so whoever waits for the answer (a web page's prompt,
	// ssh's askpass) hears it was cancelled (Rules F6).
	OnCancel func() tea.Cmd
}

// Checks picks several values (components/input/checkbox, the checkbox
// popup): Space checks a row on screen only, Enter writes the whole set back.
type Checks struct {
	Title, Glyph string
	// Subject is the one thing it acts on, kept as it is after ": " in the
	// title: "EDIT: filu" (components/layout/popup).
	Subject string
	Options []string
	Checked []bool
	// Cursor is the option the cursor starts on, scrolled into sight; the
	// app decides (components/input/checkbox). 0, the first, by default.
	Cursor int
	// Unavailable reports an option that is not available: drawn in
	// Surface2, the tick too, but the cursor stops on it and Space flips it
	// like any other; what it means is the app's (components/input/checkbox).
	Unavailable func(i int) bool
	// Loading, for a checkbox popup opened with LoadChecks, is true while
	// Options are still coming; Checked may be shorter than Options.
	Loading bool
	// Done gets the new set. It is not called when nothing changed.
	Done func([]bool) tea.Cmd
	// Commit, when set, writes a changed value back when Enter takes it, on
	// the update loop, before Done: an error keeps the popup open with the
	// value in it and opens an error popup over it, "RENAME FAILED"
	// (components/layout/popup, the action failing; Rules F4).
	Commit func([]bool) error
	// OnCancel runs when it closes with no answer: Esc, or the popup it is
	// over closing it, so whoever waits for the answer (a web page's prompt,
	// ssh's askpass) hears it was cancelled (Rules F6).
	OnCancel func() tea.Cmd
}

type finderKind int

const (
	findGo finderKind = iota
	findSelect
	findChecks
	findFiles
)

// finderRow is one row of a finder's list.
type finderRow struct {
	text  string // shown, and matched unless match is set
	match string // what is matched instead of text
	label string // shown instead of text when set: a hotkey marker, ". (this directory)"
	key   string // a select option's hotkey
	icon  string
	color Color // the icon's colour
	dir   bool  // a file-picker's directory
	// unavailable is a checkbox popup's option that is not available:
	// Surface2, the tick too.
	unavailable bool
	path        string
	item        FinderItem
}

// finderView is the finder skeleton every kind shares: the typing row with
// the count, a separator, the list; a preview beside or below for the kinds
// that have one.
type finderView struct {
	kind  finderKind
	glyph string
	// title is the title as drawn in room cells; a file-picker cuts its path
	// from the front.
	title func(room int) string
	// words is what the typed text filters by; all of it when unset.
	words func(typed string) string
	name  string
	rows  func() []finderRow
	// green says which rows are the value now (select, file-picker) or are
	// checked (checkbox popup): their text is Green.
	green func(i int, r finderRow) bool
	verb  func(r finderRow) string // Enter's word on the list
	enter func(a *App, i int, r finderRow) tea.Cmd
	// preview is the row under the cursor in w × h: a title and lines.
	preview func(r finderRow, w, h int) (string, []string)
	// info, for a preview of files, is the four rows of facts about it
	// (components/input/file-picker).
	info    func(r finderRow, w int, layer Color) []string
	loading func() bool
	known   bool // the width is known when it opens
	// enterRef, when set, is what Enter does in the key reference.
	enterRef string
	// slow marks a preview and facts read from files: they are read off the
	// update loop (components/input/file-picker, read in the background);
	// previewTitle is the preview box's title meanwhile.
	slow         bool
	previewTitle func(r finderRow) string
	read         reads
	// search is the app's own search: it gets what is typed, and the rows
	// are its results, not filtered again.
	search func(typed string) tea.Cmd

	ed       *lineEditor
	onList   bool
	shown    []int // indexes into rows, after the filter
	cursor   int   // index into shown
	top      int
	visible  int
	filtered string
	checked  []bool // the checkbox popup's checks, on screen
	allIn    bool   // its Select all row was in shown at the last refilter
	written  func()
	onCancel func() tea.Cmd // a select's, a checkbox popup's or a file-picker's OnCancel
}

// Find opens a finder.
func (a *App) Find(f *Finder) tea.Cmd {
	fv := &finderView{
		kind: findGo, glyph: f.Glyph, name: f.Title, ed: newEditor(""),
		title: func(int) string { return popupTitle(f.Title, f.Subject) },
		rows: func() []finderRow {
			out := make([]finderRow, len(f.Items))
			for i, it := range f.Items {
				out[i] = finderRow{text: it.Text, match: it.Match, icon: it.Icon, color: it.IconColor, item: it}
			}
			return out
		},
		verb:    func(finderRow) string { return cmpOr(f.Verb, "open") },
		loading: func() bool { return f.Loading },
		search:  f.Search,
	}
	fv.enter = func(a *App, _ int, r finderRow) tea.Cmd {
		if f.Done == nil {
			return a.closeTop()
		}
		return tea.Batch(a.closeTop(), f.Done(r.item))
	}
	switch {
	case f.Files:
		fv.preview = func(r finderRow, w, h int) (string, []string) {
			return path.Base(filepath.ToSlash(r.item.ID)), previewIn(f.FS, r.item.ID, r.item.Line, w, h, a.HideSecretsInPreview, nil, a.inputTabWidth())
		}
		fv.info = func(r finderRow, w int, layer Color) []string { return fileInfoIn(f.FS, r.item.ID, w, layer) }
		fv.slow = true
		fv.previewTitle = func(r finderRow) string { return path.Base(filepath.ToSlash(r.item.ID)) }
	case f.Preview != nil:
		fv.preview = func(r finderRow, _, _ int) (string, []string) {
			// Reading content, drawn as a viewer draws it (components, control characters).
			title, lines := f.Preview(r.item)
			rows := readingRows(lines)
			for i, l := range rows {
				rows[i] = readingLine(cutCR(l), tabCells)
			}
			return title, rows
		}
	}
	return a.open(fv)
}

func (s Select) open(a *App, written func()) tea.Cmd { return a.open(s.view(written, false)) }

// LoadSelect opens a select whose options are still coming: keep the
// pointer and append to Options while Loading is true. Meanwhile the title
// has the loading icon and the popup is as wide as it can be, its width not
// known when it opens (Rules F7).
func (a *App) LoadSelect(s *Select) tea.Cmd { return a.open(s.view(nil, true)) }

// view is the select's popup; live, it reads Options again as they come.
func (s *Select) view(written func(), live bool) *finderView {
	var rows []finderRow
	listRows := func() []finderRow {
		for i := len(rows); i < len(s.Options); i++ {
			r := finderRow{text: s.Options[i]}
			if i < len(s.Keys) && s.Keys[i] != "" {
				r.key, r.label = s.Keys[i], MarkLabel(s.Keys[i], s.Options[i])
			}
			rows = append(rows, r)
		}
		return rows
	}
	fv := &finderView{
		kind: findSelect, glyph: s.Glyph, name: s.Title, ed: newEditor(""), known: !live, onList: true,
		title:    func(int) string { return popupTitle(s.Title, s.Subject) },
		rows:     listRows,
		loading:  func() bool { return live && s.Loading },
		onCancel: s.OnCancel,
		green:    func(i int, _ finderRow) bool { return i == s.Selected },
		verb:     func(finderRow) string { return "choose" },
		written:  written,
	}
	fv.enter = func(a *App, i int, _ finderRow) tea.Cmd {
		if i == s.Selected || s.Done == nil && s.Commit == nil {
			return a.answer()
		}
		if err := runCommit(s.Commit, i); err != nil {
			return a.commitFailed(s.Title, s.Subject, err)
		}
		fv.wrote()
		return tea.Batch(a.answer(), runDone(s.Done, i))
	}
	fv.refilter()
	if k := slices.Index(fv.shown, s.Selected); k >= 0 {
		fv.cursor = k
	}
	return fv
}

func (s Select) typed() bool { return false }

// Select opens a select popup.
func (a *App) Select(s Select) tea.Cmd { return s.open(a, nil) }

func (c Checks) open(a *App, written func()) tea.Cmd { return a.open(c.view(written, false)) }

// LoadChecks opens a checkbox popup whose options are still coming, as
// LoadSelect does: keep the pointer and append to Options (and Checked)
// while Loading is true.
func (a *App) LoadChecks(c *Checks) tea.Cmd { return a.open(c.view(nil, true)) }

// view is the checkbox popup; live, it reads Options again as they come.
func (c *Checks) view(written func(), live bool) *finderView {
	var rows []finderRow
	fv := &finderView{
		kind: findChecks, glyph: c.Glyph, name: c.Title, ed: newEditor(""), known: !live, onList: true,
		title:    func(int) string { return popupTitle(c.Title, c.Subject) },
		verb:     func(finderRow) string { return "apply" },
		loading:  func() bool { return live && c.Loading },
		onCancel: c.OnCancel,
		written:  written,
	}
	fv.rows = func() []finderRow {
		for i := len(rows); i < len(c.Options); i++ {
			rows = append(rows, finderRow{text: c.Options[i], unavailable: c.Unavailable != nil && c.Unavailable(i)})
			fv.checked = append(fv.checked, i < len(c.Checked) && c.Checked[i])
		}
		return rows
	}
	fv.rows()
	// A row checked on screen is Green before Enter applies it (components/input/checkbox).
	fv.green = func(i int, _ finderRow) bool { return fv.isChecked(i) }
	fv.enter = func(a *App, _ int, _ finderRow) tea.Cmd {
		was := make([]bool, len(fv.checked))
		copy(was, c.Checked)
		if slices.Equal(was, fv.checked) || c.Done == nil && c.Commit == nil {
			return a.answer()
		}
		if err := runCommit(c.Commit, slices.Clone(fv.checked)); err != nil {
			return a.commitFailed(c.Title, c.Subject, err)
		}
		fv.wrote()
		return tea.Batch(a.answer(), runDone(c.Done, slices.Clone(fv.checked)))
	}
	fv.refilter()
	// The option the app says, or the first option: not Select all
	// (components/input/checkbox).
	if k := slices.Index(fv.shown, c.Cursor); k >= 0 {
		fv.cursor = k
	}
	return fv
}

func (c Checks) typed() bool { return false }

// Checks opens a checkbox popup.
func (a *App) Checks(c Checks) tea.Cmd { return c.open(a, nil) }

func (f *finderView) wrote() {
	if f.written != nil {
		f.written()
	}
}

// refilter applies what is typed (components/input/finder, matching); a
// change of the typed words puts the cursor on the first match.
func (f *finderView) refilter() {
	rows := f.rows()
	typed := f.ed.value()
	retyped := typed != f.filtered
	if retyped {
		f.cursor, f.top = 0, 0
		f.filtered = typed
	}
	words := typed
	if f.words != nil {
		words = f.words(typed)
	}
	f.shown = f.shown[:0]
	if f.kind == findChecks && typed == "" && len(rows) >= 2 {
		f.shown = append(f.shown, optAll) // Select all, hidden while filtering
	}
	for i, r := range rows {
		if f.search != nil || Matches(cmpOr(r.match, r.text), words) {
			f.shown = append(f.shown, i)
		}
	}
	// Select all coming in as the options do (or going) with nothing typed
	// anew: the cursor stays on its option.
	if all := len(f.shown) > 0 && f.shown[0] == optAll; all != f.allIn && !retyped {
		if all {
			f.cursor++
		} else if f.cursor > 0 {
			f.cursor--
		}
	}
	f.allIn = len(f.shown) > 0 && f.shown[0] == optAll
	f.cursor = min(max(f.cursor, 0), max(len(f.shown)-1, 0))
}

// at is the row under the cursor.
func (f *finderView) at() (int, finderRow, bool) {
	if f.cursor >= len(f.shown) {
		return 0, finderRow{}, false
	}
	i := f.shown[f.cursor]
	return i, f.row(i), true
}

// row is row i, or the Select all row of a checkbox popup.
func (f *finderView) row(i int) finderRow {
	if i == optAll {
		return finderRow{text: selectAllLabel}
	}
	return f.rows()[i]
}

// isChecked reports row i ticked on screen: Select all when every option is.
func (f *finderView) isChecked(i int) bool {
	if i == optAll {
		return allChecked(len(f.checked), f.checked, nil)
	}
	return f.checked[i]
}

// options counts the rows shown that are options, not Select all.
func (f *finderView) options() int {
	if len(f.shown) > 0 && f.shown[0] == optAll {
		return len(f.shown) - 1
	}
	return len(f.shown)
}

// mark is what goes before a row's text: the radio or checkbox glyph, an icon.
func (f *finderView) mark(i int, r finderRow) string {
	switch f.kind {
	case findSelect:
		if f.green(i, r) {
			return glyphRadioOn
		}
		return glyphRadioOff
	case findChecks:
		if f.isChecked(i) {
			return glyphMarked
		}
		return glyphUnchecked
	}
	return r.icon
}

func (f *finderView) rowLabel(r finderRow) string { return Escape(cmpOr(r.label, r.text)) }

// listHint is the list's hint. With no results Enter does nothing and is
// left out; a checkbox popup's Enter still applies the checks, so only Space
// goes (components/input/finder, components/input/checkbox).
func (f *finderView) listHint(results bool) []KeyHint {
	// With no results the finder's own word, not another: the width was
	// taken from it when the popup opened.
	_, r, _ := f.at()
	var h []KeyHint
	if results || f.kind == findChecks {
		h = append(h, KeyHint{"Enter", f.verb(r)})
	}
	if results && f.kind == findChecks {
		h = append(h, KeyHint{"Space", "toggle"})
	}
	return append(h, KeyHint{"Tab|/", "filter"}, KeyHint{"Esc", "cancel"})
}

// typingHint is the typing row's hint: with no results its Enter does
// nothing and is left out (components/input/finder).
func (f *finderView) typingHint(results bool) []KeyHint {
	if !results {
		return []KeyHint{{"Tab", "list"}, {"Esc", "cancel"}}
	}
	return []KeyHint{{"Enter|Tab", "list"}, {"Esc", "cancel"}}
}

func (f *finderView) hint() []KeyHint {
	if f.onList {
		return f.listHint(len(f.shown) > 0)
	}
	return f.typingHint(len(f.shown) > 0)
}

func (f *finderView) measure(sw, sh int) (int, int) {
	most := max(min(sw-2, 120), 1)
	rows := f.rows()
	if !f.known {
		// A finder with a preview takes the most height: the preview differs
		// row by row and the box must not jump (components/input/finder).
		return most, max(sh-2, 3)
	}
	content := 0
	for _, r := range rows {
		content = max(content, Width(f.mark(0, r))+1+Width(f.rowLabel(r)))
	}
	n := len(rows) // the list's rows, a checkbox popup's Select all among them
	if f.kind == findChecks && len(rows) >= 2 {
		n++
		content = max(content, Width(glyphMarked)+1+Width(selectAllLabel))
	}
	count := len(strconv.Itoa(len(rows)))*2 + 1
	content = max(content+Width(glyphHard), Width(glyphTyping)+1+8+2+count) // the typing row: the angle, eight cells, the count
	w := max(content+3, Width(titleCapsule(f.glyph, f.title(1<<20), Base, 1<<20))+4,
		hintWidth(f.listHint(true))+6, hintWidth(f.typingHint(true))+6)
	h := min(n+6, max(sh-2, 3))
	if n > h-6 {
		// Room for the scroll position at its widest too, so the hint stays
		// whole (components/layout/popup, width).
		widest := scrollText(true, n, 0, n)
		w = max(w, scrollRoom(f.listHint(true), widest), scrollRoom(f.typingHint(true), widest))
	}
	return min(w, most), h
}

func (f *finderView) draw(w, h int, layer Color) []string {
	f.refilter()
	if f.preview == nil {
		return f.drawList(w, h, layer)
	}
	// At 96 wide or more (w is the screen's width − 2 under 120) the list
	// and the preview sit side by side, the facts under the preview. Below
	// that the list and the facts share the top half and the preview takes
	// the bottom, as wide as the popup (components/input/finder; G30).
	lw := (w - 1) / 2
	// Each box is cut or padded to its share: on a tiny screen a box would
	// otherwise take more rows than it has.
	box := func(lines []string, bw, bh int) []string { return Pad2(lines, bw, bh) }
	side := func(a, b []string) []string {
		out := make([]string, len(a))
		for i := range out {
			out[i] = a[i] + " " + b[i]
		}
		return out
	}
	if w+2 >= 96 {
		return side(box(f.drawList(lw, h, layer), lw, h), box(f.drawPreview(w-lw-1, h, layer, f.info != nil), w-lw-1, h))
	}
	top := h / 2
	bottom := box(f.drawPreview(w, h-top, layer, false), w, h-top)
	if f.info == nil {
		// Without facts the list and the preview are stacked, half each.
		return append(box(f.drawList(w, top, layer), w, top), bottom...)
	}
	return append(side(box(f.drawList(lw, top, layer), lw, top), box(f.drawInfo(w-lw-1, top, layer), w-lw-1, top)), bottom...)
}

// drawInfo is the facts in a box of their own, titled INFO, when the preview
// is below.
func (f *finderView) drawInfo(w, h int, layer Color) []string {
	inner := w - 2
	var rows []string
	if _, r, ok := f.at(); ok {
		rows = f.facts(r, inner-2, layer)
	}
	body := make([]string, 0, max(h-4, 0))
	for i := range max(h-4, 0) {
		if i < len(rows) {
			body = append(body, padRow(rows[i], inner))
		} else {
			body = append(body, Repeat(" ", inner))
		}
	}
	return popupBox{color: layer, title: "INFO"}.draw(w, body)
}

// drawList is the finder's own box: the typing row and its count, a
// separator one cell from the frame, the list.
func (f *finderView) drawList(w, h int, layer Color) []string {
	inner := w - 2
	rows := f.rows()
	count := strconv.Itoa(f.options()) + "/" + strconv.Itoa(len(rows))
	body := []string{
		typingRow(f.ed, inner, count, !f.onList, layer),
		" " + Fg(Overlay0).Render(Repeat("─", inner-2)) + " ",
	}
	n := max(h-6, 1)
	f.visible = n
	f.top = scrollWindow(f.top, f.cursor, n, len(f.shown))
	arrow := Width(glyphHard)
	text := inner - 1 - arrow
	for k := f.top; k < f.top+n; k++ {
		if k >= len(f.shown) {
			if k == 0 && len(rows) == 0 && f.loading != nil && f.loading() {
				body = append(body, padRow(Fg(Overlay0).Render("(indexing…)"), inner))
				continue
			}
			body = append(body, Repeat(" ", inner))
			continue
		}
		i := f.shown[k]
		r := f.row(i)
		mark := f.mark(i, r)
		label := f.rowLabel(r)
		if k == f.cursor {
			plain := label
			if mark != "" {
				plain = mark + " " + label
			}
			line := cursorRow(plainNames(plain), inner, On(layer))
			if !f.onList {
				line = Dim(line) // the focus is on the typing row (P6)
			}
			body = append(body, line)
			continue
		}
		style := Fg(Text)
		if f.green != nil && f.green(i, r) {
			style = Fg(Green)
		}
		if r.unavailable {
			style = Fg(Surface2) // the tick too: unavailable wins over in effect
		}
		line := style.Render(label)
		if mark != "" {
			markStyle := style
			if f.kind == findGo || f.kind == findFiles {
				markStyle = Fg(cmpColor(r.color, Text))
			}
			line = markStyle.Render(mark) + " " + line
		}
		body = append(body, " "+Fit(line, text)+Repeat(" ", arrow))
	}
	scroll := ""
	if len(f.shown) > n {
		scroll = scrollText(true, f.cursor+1, 0, len(f.shown))
	}
	loading := f.loading != nil && f.loading()
	return popupBox{color: layer, glyph: f.glyph, title: f.title(w - 6), hint: f.hint(), scroll: scroll, loading: loading}.draw(w, body)
}

// drawPreview is the box beside or under the list: the row under the cursor,
// titled with its name; the layer's colour, never dimmed, nothing on its
// bottom line. withInfo puts a separator and the facts under the preview.
func (f *finderView) drawPreview(w, h int, layer Color, withInfo bool) []string {
	inner := w - 2
	rows := max(h-4, 0)
	area := rows
	if withInfo {
		area = max(rows-5, 0) // a separator and four facts
	}
	title, lines := "", []string(nil)
	var info []string
	reading := false
	if _, r, ok := f.at(); ok {
		if f.slow {
			// Read in the background: meanwhile the loading icon after the
			// title and "(loading…)" (components/input/file-picker).
			title = f.previewTitle(r)
			var done bool
			lines, done = f.read.get(rowPath(r)+"\x00p"+itoa(inner-2)+"x"+itoa(area), func() []string {
				_, l := f.preview(r, inner-2, area)
				return l
			})
			if !done {
				lines, reading = []string{Fg(Overlay0).Render("(loading…)")}, true
			}
		} else {
			title, lines = f.preview(r, inner-2, area)
		}
		if withInfo {
			info = f.facts(r, inner-2, layer)
		}
	}
	body := make([]string, 0, rows)
	for i := range area {
		if i < len(lines) {
			body = append(body, padRow(lines[i], inner))
		} else {
			body = append(body, Repeat(" ", inner))
		}
	}
	if withInfo && rows >= 5 {
		body = append(body, " "+Fg(Overlay0).Render(Repeat("─", inner-2))+" ")
		for i := range 4 {
			if i < len(info) {
				body = append(body, padRow(info[i], inner))
			} else {
				body = append(body, Repeat(" ", inner))
			}
		}
	}
	return popupBox{color: layer, title: title, loading: reading}.draw(w, Pad2(body, inner, rows))
}

// facts are the four rows of facts about the row under the cursor; read in
// the background, their values are blank until they are there: not "—",
// which says a value cannot be had (components/input/file-picker).
func (f *finderView) facts(r finderRow, w int, layer Color) []string {
	if !f.slow {
		return f.info(r, w, layer)
	}
	rows, done := f.read.get(rowPath(r)+"\x00i"+itoa(w)+layer.String(), func() []string { return f.info(r, w, layer) })
	if !done {
		return infoRows([4][2]string{{"TYPE", ""}, {"SIZE", ""}, {"MODIFIED", ""}, {"MODE", ""}}, w, layer)
	}
	return rows
}

// rowPath is the path a row of files stands for.
func rowPath(r finderRow) string { return cmpOr(r.path, r.item.ID) }

// reads are what a popup reads off the update loop: kept by what was asked,
// so going back to a row reads nothing again while it is open; wanted until
// the app starts reading them; in flight until they come.
type reads struct {
	done     map[string][]string
	wanted   map[string]func() []string
	inFlight map[string]bool
}

// get is what was read for key, or false, key then wanted, read by fn.
func (r *reads) get(key string, fn func() []string) ([]string, bool) {
	if lines, ok := r.done[key]; ok {
		return lines, true
	}
	if !r.inFlight[key] {
		if r.wanted == nil {
			r.wanted = map[string]func() []string{}
		}
		r.wanted[key] = fn
	}
	return nil, false
}

// readDone is what was read for key.
type readDone struct {
	r     *reads
	key   string
	lines []string
}

// start starts reading what is wanted, each off the update loop.
func (r *reads) start() tea.Cmd {
	var cmds []tea.Cmd
	for key, fn := range r.wanted {
		if r.inFlight == nil {
			r.inFlight = map[string]bool{}
		}
		r.inFlight[key] = true
		cmds = append(cmds, func() tea.Msg { return readDone{r, key, fn()} })
	}
	r.wanted = nil
	return tea.Batch(cmds...)
}

// keep keeps what was read.
func (r *reads) keep(m readDone) {
	if r.done == nil {
		r.done = map[string][]string{}
	}
	r.done[m.key] = m.lines
	delete(r.inFlight, m.key)
}

// busy reports something wanted or being read.
func (r *reads) busy() bool { return len(r.wanted) > 0 || len(r.inFlight) > 0 }

// fetch starts the reads the last frame wanted.
func (f *finderView) fetch() tea.Cmd { return f.read.start() }

// fetching keeps the frames coming while something is read.
func (f *finderView) fetching() bool { return f.read.busy() }

func cmpColor(c, def Color) Color {
	if !c.Set {
		return def
	}
	return c
}

func (f *finderView) move(dir int) {
	if n := len(f.shown); n > 0 {
		f.cursor = (f.cursor + dir + n) % n
	}
}

func (f *finderView) key(a *App, key string, msg tea.KeyMsg) (bool, tea.Cmd) {
	f.refilter()
	if !f.onList {
		switch key {
		case "Esc":
			return true, a.closeTop()
		case "↑":
			f.move(-1)
		case "↓":
			f.move(1)
		case "Enter":
			if len(f.shown) > 0 {
				f.onList = true
			}
		case "Tab":
			f.onList = true
		default:
			if f.ed.key(key, msg) {
				f.refilter()
				if f.search != nil {
					return true, f.search(f.ed.value()) // the app searches; typing goes on
				}
			}
		}
		return true, nil // the typing row takes every key (Rules K8)
	}
	half := max(f.visible/2, 1)
	switch key {
	case "j", "l", "↓", "→":
		f.move(1)
	case "k", "h", "↑", "←":
		f.move(-1)
	case "d", "PgDn":
		f.cursor = min(f.cursor+half, max(len(f.shown)-1, 0))
	case "u", "PgUp":
		f.cursor = max(f.cursor-half, 0)
	case "gg":
		f.cursor = 0
	case "G":
		f.cursor = max(len(f.shown)-1, 0)
	case "Tab", "/":
		f.onList = false
	case "Enter":
		// A checkbox popup applies its checks with no results too: the filter
		// does not change what is checked (components/input/checkbox).
		if i, r, ok := f.at(); ok || f.kind == findChecks {
			return true, f.enter(a, i, r)
		}
	case "Space":
		if f.kind == findChecks {
			if i, _, ok := f.at(); ok && i == optAll {
				f.checked = checkAll(len(f.checked), f.checked, !f.isChecked(optAll), nil)
			} else if ok {
				f.checked[i] = !f.checked[i]
			}
		}
	default:
		if f.kind == findSelect {
			for i, r := range f.rows() {
				if r.key != "" && r.key == key {
					return true, f.enter(a, i, r)
				}
			}
		}
		return false, nil
	}
	return true, nil
}

func (f *finderView) surface() string { return f.name }

func (f *finderView) keyRef() []refSection {
	enter := "go to the row under the cursor"
	switch f.kind {
	case findSelect:
		enter = "choose the row under the cursor"
	case findChecks:
		enter = "apply the checks"
	case findFiles:
		enter = "go into a directory, choose a file"
	}
	rows := []KeyRow{{Key: "Enter", Desc: cmpOr(f.enterRef, enter)}}
	if f.kind == findChecks {
		rows = append(rows, KeyRow{Key: "Space", Desc: "check or uncheck the row"})
	}
	if f.kind == findSelect {
		for _, r := range f.rows() {
			if r.key != "" {
				rows = append(rows, KeyRow{Key: r.key, Desc: "choose " + r.text})
			}
		}
	}
	rows = append(rows, KeyRow{Key: "Tab|/", Desc: "to the typing row, typing on after the words there"},
		KeyRow{Key: "Esc", Desc: "cancel"})
	return []refSection{{rows: rows}, {title: "move", rows: []KeyRow{
		{Key: "j|k", Desc: "down, up"},
		{Key: "u|d", Desc: "half a page up, down"},
		{Key: "gg|G", Desc: "to the top, the bottom"},
	}}}
}

func (f *finderView) typing() bool { return !f.onList }

func (f *finderView) isLoading() bool { return f.loading != nil && f.loading() }

func (f *finderView) cancel() tea.Cmd { return run(f.onCancel) }
