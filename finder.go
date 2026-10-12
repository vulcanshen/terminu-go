package terminu

import (
	"slices"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// FinderItem is one row of a Finder.
type FinderItem struct {
	ID   string // the app's own key for it
	Text string // shown and matched
	// Icon, when set, goes before the text in IconColor (a file's type).
	Icon      string
	IconColor Color
}

// Finder is a list with a search that goes somewhere (components/input/finder):
// type to narrow the list, Enter on a row to go there, the row under the
// cursor previewed beside it. Keep the pointer: rows may keep arriving —
// append to Items while Loading is true.
type Finder struct {
	Title, Glyph string
	// Verb is what Enter on the list does, for the hint ("open").
	Verb    string
	Items   []FinderItem
	Loading bool
	// Preview is the row under the cursor: a title (its name, or where it
	// is) and the lines to show.
	Preview func(FinderItem) (title string, lines []string)
	Done    func(FinderItem) tea.Cmd
}

// Select picks one value from a list (components/input/select).
type Select struct {
	Title, Glyph string
	Options      []string
	// Keys, when set, are the options' hotkeys (Rules M5); "" for none.
	Keys []string
	// Selected is the value now; -1 for none.
	Selected int
	// Done gets the chosen option. It is not called when the value did not
	// change.
	Done func(int) tea.Cmd
}

// Checks picks several values (components/input/checkbox, the checkbox
// popup): Space checks a row on screen only, Enter writes the whole set back.
type Checks struct {
	Title, Glyph string
	Options      []string
	Checked      []bool
	// Done gets the new set. It is not called when nothing changed.
	Done func([]bool) tea.Cmd
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
	text  string // matched and shown
	label string // shown instead of text when set: a hotkey marker
	key   string // a select option's hotkey
	icon  string
	color Color // the icon's colour
	dir   bool  // a file-picker's directory
	path  string
	item  FinderItem
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
	green   func(i int, r finderRow) bool
	verb    func(r finderRow) string // Enter's word on the list
	enter   func(a *App, i int, r finderRow) tea.Cmd
	preview func(r finderRow) (string, []string)
	loading func() bool
	known   bool // the width is known when it opens

	ed       *lineEditor
	onList   bool
	shown    []int // indexes into rows, after the filter
	cursor   int   // index into shown
	top      int
	visible  int
	filtered string
	checked  []bool // the checkbox popup's checks, on screen
	written  func()
}

// Find opens a finder.
func (a *App) Find(f *Finder) tea.Cmd {
	fv := &finderView{
		kind: findGo, glyph: f.Glyph, name: f.Title, ed: newEditor(""),
		title: func(int) string { return Upper(f.Title) },
		rows: func() []finderRow {
			out := make([]finderRow, len(f.Items))
			for i, it := range f.Items {
				out[i] = finderRow{text: it.Text, icon: it.Icon, color: it.IconColor, item: it}
			}
			return out
		},
		verb:    func(finderRow) string { return cmpOr(f.Verb, "open") },
		loading: func() bool { return f.Loading },
	}
	fv.enter = func(a *App, _ int, r finderRow) tea.Cmd {
		if f.Done == nil {
			return a.closeTop()
		}
		return tea.Batch(a.closeTop(), f.Done(r.item))
	}
	if f.Preview != nil {
		fv.preview = func(r finderRow) (string, []string) { return f.Preview(r.item) }
	}
	return a.open(fv)
}

func (s Select) open(a *App, written func()) tea.Cmd {
	rows := make([]finderRow, len(s.Options))
	for i, o := range s.Options {
		rows[i] = finderRow{text: o}
		if i < len(s.Keys) && s.Keys[i] != "" {
			rows[i].key, rows[i].label = s.Keys[i], MarkLabel(s.Keys[i], o)
		}
	}
	fv := &finderView{
		kind: findSelect, glyph: s.Glyph, name: s.Title, ed: newEditor(""), known: true, onList: true,
		title:   func(int) string { return Upper(s.Title) },
		rows:    func() []finderRow { return rows },
		green:   func(i int, _ finderRow) bool { return i == s.Selected },
		verb:    func(finderRow) string { return "choose" },
		written: written,
	}
	fv.enter = func(a *App, i int, _ finderRow) tea.Cmd {
		if i == s.Selected || s.Done == nil {
			return a.closeTop()
		}
		fv.wrote()
		return tea.Batch(a.closeTop(), s.Done(i))
	}
	fv.refilter()
	if k := slices.Index(fv.shown, s.Selected); k >= 0 {
		fv.cursor = k
	}
	return a.open(fv)
}

func (s Select) typed() bool { return false }

// Select opens a select popup.
func (a *App) Select(s Select) tea.Cmd { return s.open(a, nil) }

func (c Checks) open(a *App, written func()) tea.Cmd {
	rows := make([]finderRow, len(c.Options))
	for i, o := range c.Options {
		rows[i] = finderRow{text: o}
	}
	fv := &finderView{
		kind: findChecks, glyph: c.Glyph, name: c.Title, ed: newEditor(""), known: true, onList: true,
		title:   func(int) string { return Upper(c.Title) },
		rows:    func() []finderRow { return rows },
		verb:    func(finderRow) string { return "apply" },
		checked: make([]bool, len(c.Options)),
		written: written,
	}
	copy(fv.checked, c.Checked)
	// TODO(tdp G38): a row checked on screen is Green before Enter applies it.
	fv.green = func(i int, _ finderRow) bool { return fv.checked[i] }
	fv.enter = func(a *App, _ int, _ finderRow) tea.Cmd {
		was := make([]bool, len(c.Options))
		copy(was, c.Checked)
		if slices.Equal(was, fv.checked) || c.Done == nil {
			return a.closeTop()
		}
		fv.wrote()
		return tea.Batch(a.closeTop(), c.Done(slices.Clone(fv.checked)))
	}
	fv.refilter()
	return a.open(fv)
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
	if typed != f.filtered {
		f.cursor, f.top = 0, 0
		f.filtered = typed
	}
	words := typed
	if f.words != nil {
		words = f.words(typed)
	}
	f.shown = f.shown[:0]
	for i, r := range rows {
		if Matches(r.text, words) {
			f.shown = append(f.shown, i)
		}
	}
	f.cursor = min(max(f.cursor, 0), max(len(f.shown)-1, 0))
}

// at is the row under the cursor.
func (f *finderView) at() (int, finderRow, bool) {
	if f.cursor >= len(f.shown) {
		return 0, finderRow{}, false
	}
	rows := f.rows()
	i := f.shown[f.cursor]
	return i, rows[i], true
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
		if f.checked[i] {
			return glyphMarked
		}
		return glyphUnchecked
	}
	return r.icon
}

func (f *finderView) rowLabel(r finderRow) string { return cmpOr(r.label, r.text) }

func (f *finderView) listHint() []KeyHint {
	verb := "choose"
	if _, r, ok := f.at(); ok {
		verb = f.verb(r)
	}
	h := []KeyHint{{"Enter", verb}}
	if f.kind == findChecks {
		h = append(h, KeyHint{"Space", "toggle"})
	}
	return append(h, KeyHint{"Tab", "filter"}, KeyHint{"Esc", "cancel"})
}

func (f *finderView) typingHint() []KeyHint {
	return []KeyHint{{"Enter/Tab", "list"}, {"Esc", "cancel"}}
}

func (f *finderView) hint() []KeyHint {
	if f.onList {
		return f.listHint()
	}
	return f.typingHint()
}

// previewSplit says how the list and the preview share w × h: side by side,
// half each, when the screen is at least 96 wide; else the preview below
// (components/input/finder). The list box comes first.
func previewSplit(sw, w, h int) (side bool, lw, lh int) {
	if sw >= 96 {
		return true, (w - 1) / 2, h
	}
	// TODO(tdp G30): stacked, the two boxes take half the height each.
	return false, w, h / 2
}

func (f *finderView) measure(sw, sh int) (int, int) {
	most := max(min(sw-2, 120), 1)
	rows := f.rows()
	if !f.known {
		// TODO(tdp G30): a preview's lines differ row by row and are not known
		// when it opens: the finder takes the most height.
		return most, max(sh-2, 3)
	}
	content := 0
	for _, r := range rows {
		content = max(content, Width(f.mark(0, r))+1+Width(f.rowLabel(r)))
	}
	count := len(strconv.Itoa(len(rows)))*2 + 1
	content = max(content+Width(glyphHard), 8+2+count)
	w := max(content+3, Width(titleCapsule(f.glyph, f.title(1<<20), Base, 1<<20))+4,
		hintWidth(f.listHint())+6, hintWidth(f.typingHint())+6)
	return min(w, most), min(len(rows)+6, max(sh-2, 3))
}

func (f *finderView) draw(w, h int, layer Color) []string {
	f.refilter()
	if f.preview == nil {
		return f.drawList(w, h, layer)
	}
	// w is the screen's width − 2 when it is under 120 wide.
	side, lw, lh := previewSplit(w+2, w, h)
	list := f.drawList(lw, lh, layer)
	if !side {
		return append(list, f.drawPreview(w, h-lh, layer)...)
	}
	prev := f.drawPreview(w-lw-1, h, layer)
	out := make([]string, h)
	for i := range out {
		out[i] = list[i] + " " + prev[i]
	}
	return out
}

// drawList is the finder's own box: the typing row and its count, a
// separator one cell from the frame, the list.
func (f *finderView) drawList(w, h int, layer Color) []string {
	inner := w - 2
	rows := f.rows()
	count := strconv.Itoa(len(f.shown)) + "/" + strconv.Itoa(len(rows))
	room := max(inner-2-Width(count)-2, 1)
	var typed string
	if f.onList {
		typed = f.ed.view(room, Fg(Overlay0), Fg(Overlay0), false, nil, "")
	} else {
		typed = f.ed.view(room, Fg(Lavender), Style{Fg: Base, Bg: Lavender}, true, nil, "")
	}
	body := []string{
		// TODO(tdp G36): the count in Overlay0.
		" " + typed + "  " + Fg(Overlay0).Render(count) + " ",
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
		r := rows[i]
		mark := f.mark(i, r)
		label := f.rowLabel(r)
		if k == f.cursor {
			plain := label
			if mark != "" {
				plain = mark + " " + label
			}
			line := cursorRow(ansi.Strip(plain), inner, On(layer))
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

// drawPreview is the box beside the list: the row under the cursor, titled
// with its name. TODO(tdp G27): the layer's colour, not dimmed, nothing on
// its bottom line.
func (f *finderView) drawPreview(w, h int, layer Color) []string {
	inner := w - 2
	title, lines := "", []string(nil)
	if _, r, ok := f.at(); ok {
		title, lines = f.preview(r)
	}
	body := make([]string, 0, h-4)
	for i := range max(h-4, 0) {
		if i < len(lines) {
			body = append(body, padRow(lines[i], inner))
		} else {
			body = append(body, Repeat(" ", inner))
		}
	}
	return popupBox{color: layer, title: title}.draw(w, body)
}

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
	case "gg", "Home":
		f.cursor = 0
	case "G", "End":
		f.cursor = max(len(f.shown)-1, 0)
	case "Tab", "/":
		f.onList = false
	case "Enter":
		if i, r, ok := f.at(); ok {
			return true, f.enter(a, i, r)
		}
	case "Space":
		if f.kind == findChecks {
			if i, _, ok := f.at(); ok {
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
	rows := []KeyRow{{Key: "Enter", Desc: enter}}
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
	rows = append(rows, KeyRow{Key: "Tab", Desc: "to the typing row"}, KeyRow{Key: "/", Desc: "type on after the words there"},
		KeyRow{Key: "Esc", Desc: "cancel"})
	return []refSection{{rows: rows}, {title: "move", rows: []KeyRow{
		{Key: "j/k", Desc: "down, up"},
		{Key: "u/d", Desc: "half a page up, down"},
		{Key: "gg/G", Desc: "to the top, the bottom"},
	}}}
}

func (f *finderView) typing() bool { return !f.onList }

func (f *finderView) isLoading() bool { return f.loading != nil && f.loading() }
