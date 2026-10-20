package terminu

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type menuRowKind int

const (
	menuItem menuRowKind = iota
	menuTitle
	menuSeparator
)

type menuRow struct {
	kind   menuRowKind
	label  string // with its hotkey marker
	desc   string
	action Action
}

func (r menuRow) selectable() bool { return r.kind == menuItem && !r.action.disabled() }

// menuView is a menu popup (components/dialog/menu): a list without search,
// j/k to move, Enter or a hotkey to run a row.
type menuView struct {
	glyph, title string
	subject      string // after ": " in the title, kept as it is
	cased        string // the whole title, in its case already (a panel's with its name)
	rows         []menuRow
	cursor       int // index into rows, always a selectable one when there is one
	top          int
	inner        int
	visible      int // rows the last draw had room for
	// run runs a row's action; the app decides what closes (Rules T1).
	run func(a *App, act Action) tea.Cmd
	// space makes Space close it: the Space menu (Rules K5).
	space bool
	name  string
}

// MenuSection is a group of a menu's rows under a title. A menu of one group
// needs none; in a menu of more, every group has one, the first too (Rules
// M8).
type MenuSection struct {
	Title   string
	Actions []Action
}

func newMenu(title string, sections []MenuSection, tail []Action) *menuView {
	m := &menuView{glyph: glyphMenu, title: title, name: title}
	for i, s := range sections {
		var rows []menuRow
		for _, a := range s.Actions {
			if a.hidden() {
				continue
			}
			rows = append(rows, menuRow{kind: menuItem, label: MarkLabel(a.Key, a.Label), desc: a.Desc, action: a})
		}
		if len(rows) == 0 {
			continue
		}
		if len(m.rows) > 0 {
			m.rows = append(m.rows, menuRow{kind: menuSeparator})
		}
		if s.Title != "" {
			m.rows = append(m.rows, menuRow{kind: menuTitle, label: s.Title})
		}
		m.rows = append(m.rows, rows...)
		_ = i
	}
	if len(tail) > 0 {
		if len(m.rows) > 0 {
			m.rows = append(m.rows, menuRow{kind: menuSeparator})
		}
		for _, a := range tail {
			m.rows = append(m.rows, menuRow{kind: menuItem, label: MarkLabel(a.Key, a.Label), desc: a.Desc, action: a})
		}
	}
	m.cursor = m.step(-1, 1)
	return m
}

// step finds the next selectable row from i going by dir, wrapping (Rules K12).
func (m *menuView) step(i, dir int) int {
	n := len(m.rows)
	for k := 1; k <= n; k++ {
		j := ((i+dir*k)%n + n) % n
		if m.rows[j].selectable() {
			return j
		}
	}
	return max(i, 0)
}

func (m *menuView) arrowRoom() int { return Width(glyphHard) }

// rowWidth is a row's width without padding: label, two cells, description.
func (m *menuView) rowWidth(r menuRow) int {
	switch r.kind {
	case menuTitle:
		return Width(glyphHexagon) + 1 + Width(Escape(r.label))
	case menuSeparator:
		return 0
	}
	if r.desc == "" {
		return Width(Escape(r.label))
	}
	return Width(Escape(r.label)) + 2 + Width(Escape(r.desc))
}

// hint is Enter:run Esc:close, Enter:detail on a row opening its full text
// (components/dialog/menu: what Enter does there).
func (m *menuView) hint() []KeyHint {
	verb := "run"
	if m.cursor < len(m.rows) && m.rows[m.cursor].kind == menuItem {
		switch act := m.rows[m.cursor].action; {
		case act.EnterHint != "":
			verb = act.EnterHint
		case act.Detail:
			verb = "detail"
		}
	}
	return []KeyHint{{"Enter", verb}, {"Esc", "close"}}
}

func (m *menuView) measure(sw, sh int) (int, int) {
	content := 0
	for _, r := range m.rows {
		content = max(content, m.rowWidth(r))
	}
	need := content + 3 + m.arrowRoom() // frame, the left cell, the arrow's cells
	need = max(need, Width(titleCapsule(m.glyph, m.titleText(), Base, 1<<20))+4)
	for i, r := range m.rows {
		if r.kind == menuItem {
			saved := m.cursor
			m.cursor = i
			need = max(need, hintWidth(m.hint())+6) // the widest hint: the width does not change
			m.cursor = saved
		}
	}
	h := min(len(m.rows)+4, max(sh-2, 3))
	if len(m.rows) > h-4 {
		total := 0
		for _, r := range m.rows {
			if r.kind == menuItem {
				total++
			}
		}
		widest := scrollText(true, total, 0, total)
		for i, r := range m.rows {
			if r.kind == menuItem {
				saved := m.cursor
				m.cursor = i
				need = max(need, scrollRoom(m.hint(), widest))
				m.cursor = saved
			}
		}
	}
	w := min(need, max(min(sw-2, 120), 1))
	return w, h
}

// hintWidth is how wide a hint is written whole.
func hintWidth(pairs []KeyHint) int {
	w := 0
	for i, p := range pairs {
		if i > 0 {
			w++
		}
		w += Width(p.Key) + 1 + Width(p.Desc)
	}
	return w
}

func (m *menuView) draw(w, h int, color Color) []string {
	inner := w - 2
	m.inner = inner
	n := max(h-4, 0)
	m.visible = n
	m.top = scrollWindow(m.top, m.cursor, n, len(m.rows))
	body := make([]string, 0, n)
	arrow := m.arrowRoom()
	text := inner - 1 - arrow
	for i := m.top; i < m.top+n && i < len(m.rows); i++ {
		r := m.rows[i]
		if r.kind == menuItem && r.action.Live != nil {
			// Asked again at every draw: a work list's progress (components/dialog/menu).
			label, desc := r.action.Live()
			r.label, r.desc = MarkLabel(r.action.Key, label), desc
		}
		switch r.kind {
		case menuTitle:
			body = append(body, " "+Fit(Fg(Overlay0).Render(glyphHexagon+" "+Escape(Upper(r.label))), text)+Repeat(" ", arrow))
		case menuSeparator:
			body = append(body, " "+Fg(Overlay0).Render(Repeat("╌", inner-2))+" ")
		default:
			line := m.rowText(r, text)
			switch {
			case i == m.cursor && r.selectable():
				body = append(body, cursorRow(plainNames(line), inner, On(color)))
			case r.action.disabled():
				body = append(body, " "+Fg(Surface2).Render(plainNames(line))+Repeat(" ", arrow))
			default:
				body = append(body, " "+line+Repeat(" ", arrow))
			}
		}
	}
	for len(body) < n {
		body = append(body, Repeat(" ", inner))
	}
	scroll := ""
	if len(m.rows) > n {
		pos, total := 0, 0
		for i, r := range m.rows {
			if r.kind == menuItem {
				total++
				if i == m.cursor {
					pos = total
				}
			}
		}
		scroll = scrollText(true, pos, 0, total)
	}
	return popupBox{color: color, glyph: m.glyph, title: m.titleText(), hint: m.hint(), scroll: scroll}.draw(w, body)
}

// titleText is the title as drawn: upper case, the subject as it is.
func (m *menuView) titleText() string {
	if m.cased != "" {
		return m.cased
	}
	return popupTitle(m.title, m.subject)
}

// rowText is a row in w cells: label in Text, description in Overlay0 on the
// right, cut with "…" when it does not fit (components/dialog/menu).
func (m *menuView) rowText(r menuRow, w int) string {
	label := Truncate(Escape(r.label), w)
	room := w - Width(label) - 2
	if r.desc == "" || room < 2 {
		return Pad(Fg(Text).Render(label), w)
	}
	desc := Truncate(Escape(r.desc), room)
	return Fg(Text).Render(label) + Repeat(" ", w-Width(label)-Width(desc)) + Fg(Overlay0).Render(desc)
}

func (m *menuView) key(a *App, key string, msg tea.KeyMsg) (bool, tea.Cmd) {
	half := max(m.visible/2, 1)
	switch key {
	case "j", "l", "↓", "→":
		m.cursor = m.step(m.cursor, 1)
	case "k", "h", "↑", "←":
		m.cursor = m.step(m.cursor, -1)
	case "d", "PgDn":
		m.cursor = m.towards(min(m.cursor+half, len(m.rows)-1), -1)
	case "u", "PgUp":
		m.cursor = m.towards(max(m.cursor-half, 0), 1)
	case "gg":
		m.cursor = m.towards(0, 1)
	case "G":
		m.cursor = m.towards(len(m.rows)-1, -1)
	case "Enter":
		if m.cursor < len(m.rows) && m.rows[m.cursor].selectable() {
			return true, m.run(a, m.rows[m.cursor].action)
		}
		return true, nil
	case "Space":
		if m.space {
			return true, a.closeTop()
		}
		return true, nil
	default:
		for _, r := range m.rows {
			if r.kind == menuItem && r.action.Key == key && key != "Enter" {
				if r.action.disabled() {
					return true, nil
				}
				return true, m.run(a, r.action)
			}
		}
		return false, nil
	}
	return true, nil
}

// towards is the selectable row nearest i, looking first by dir, then back:
// u/d and gg/G stop at the ends (Rules K12).
func (m *menuView) towards(i, dir int) int {
	for j := i; j >= 0 && j < len(m.rows); j += dir {
		if m.rows[j].selectable() {
			return j
		}
	}
	for j := i; j >= 0 && j < len(m.rows); j -= dir {
		if m.rows[j].selectable() {
			return j
		}
	}
	return m.cursor
}

func (m *menuView) surface() string { return m.name }

func (m *menuView) keyRef() []refSection {
	var rows []KeyRow
	rows = append(rows, KeyRow{Key: "Enter", Desc: "the row under the cursor"})
	for _, r := range m.rows {
		if r.kind == menuItem && r.action.Key != "" && r.action.Key != "Enter" {
			rows = append(rows, KeyRow{Key: r.action.Key, Desc: plainLabel(r.action) + descTail(r.action.Desc), Disabled: r.action.disabled()})
		}
	}
	esc := "close this menu"
	if m.space {
		rows = append(rows, KeyRow{Key: "Space|Esc", Desc: esc})
	} else {
		rows = append(rows, KeyRow{Key: "Esc", Desc: esc})
	}
	return []refSection{{rows: rows}, {title: "move", rows: oneWayKeys()}}
}

func (m *menuView) typing() bool { return false }

// plainLabel is an action's label as the key reference writes it: no hotkey
// marker, its case as it is (components/dialog/note).
func plainLabel(a Action) string { return a.Label }

func descTail(d string) string {
	if d == "" {
		return ""
	}
	return " — " + d
}

func joinNonEmpty(parts ...string) string {
	var out []string
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, " ")
}
