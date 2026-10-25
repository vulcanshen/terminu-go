package terminu

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// modeMenu opens a mode's Space menu (tdp menu.4,
// K11), titled as where the mode is: ITEM OPERATION, what can be done
// to the item under the cursor or the range selected, and <NAME> MODE, the
// mode's own keys, its movement keys only shown. A row closes the menu and
// does as its key does; Esc in the menu only closes it, back to the mode.
func (a *App) modeMenu(title, mode string, item, keys []Action) tea.Cmd {
	m := newMenu("", []MenuSection{{Title: "item operation", Actions: item}, {Title: mode + " mode", Actions: keys}}, nil)
	m.cased, m.space, m.name = title, true, "Space menu"
	m.run = func(a *App, act Action) tea.Cmd {
		if act.disabled() {
			return nil
		}
		return tea.Batch(a.closeTop(), run(act.Do))
	}
	return a.open(m)
}

// moveRow is a mode's movement key in its Space menu: the key where a label
// goes, in Blue, shown only.
func moveRow(key, desc string) Action { return Action{Label: key, Desc: desc, show: true} }

// leaveRow is the row leaving the mode: [Esc] and its verb.
func leaveRow(verb string, do func() tea.Cmd) Action {
	return Action{Key: "Esc", Label: capital(verb), Do: do}
}

func capital(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// modeTitle is where a mode is: the top popup's title, or the focused
// panel's [N] label.
func (a *App) modeTitle() string {
	if l := a.top(); l != nil {
		return refTitle(l.view)
	}
	s := a.Screen()
	return s.Focused().title(s.at() + 1)
}

// panelModeMenu is the Space menu of the mode p is in: its keys marked Item
// first, its movement keys (NoFooter) shown, the others to run, then leaving.
func (a *App) panelModeMenu(p *Panel) tea.Cmd {
	m := p.mode
	var item, keys []Action
	for _, k := range m.Keys {
		if k.NoFooter {
			keys = append(keys, moveRow(k.Key, k.Desc))
			continue
		}
		key, do := k.first(), k.Do
		act := Action{Key: key, Label: cmpOr(k.Label, capital(k.Desc)), Desc: k.MenuDesc, Disabled: k.Disabled,
			Do: func() tea.Cmd {
				if do == nil {
					return nil
				}
				return do(key)
			}}
		if k.Item {
			item = append(item, act)
		} else {
			keys = append(keys, act)
		}
	}
	keys = append(keys, leaveRow(cmpOr(m.Exit, "leave"), func() tea.Cmd { return a.leavePanelMode(p) }))
	return a.modeMenu(a.modeTitle(), m.Name, item, keys)
}

// leavePanelMode leaves the mode p is in, back to before it, as Esc does.
func (a *App) leavePanelMode(p *Panel) tea.Cmd {
	m := p.mode
	if m == nil {
		return nil
	}
	p.mode = nil
	return run(m.OnExit)
}

// selectModeMenu is the Space menu of a viewer's selection mode: Select,
// or Copy once selecting; press presses a key in the mode.
func (a *App) selectModeMenu(v *Viewer, press func(key string) tea.Cmd) tea.Cmd {
	do := func(key string) func() tea.Cmd { return func() tea.Cmd { return press(key) } }
	item := []Action{{Key: v.Select.Key, Label: "Select", Desc: "from the cursor", Do: do(v.Select.Key)}}
	if v.selecting {
		ck := v.Select.copyKey()
		item = []Action{{Key: ck, Label: "[" + ck + "] Copy", Desc: "the selection, and leave", Do: do(ck)}}
	}
	var keys []Action
	for _, s := range (&viewerView{v: v}).keyRef() {
		if s.title == "move" {
			for _, r := range s.rows {
				keys = append(keys, moveRow(r.Key, r.Desc))
			}
		}
	}
	verb := "cancel" // the hint's Esc
	if v.selecting {
		verb = "unselect"
	}
	keys = append(keys, leaveRow(verb, do("Esc")))
	return a.modeMenu(a.modeTitle(), v.modeName(), item, keys)
}

// reorderModeMenu is the Space menu of a reorder's Drag mode.
func (a *App) reorderModeMenu(rv *reorderView) tea.Cmd {
	do := func(key string) func() tea.Cmd {
		return func() tea.Cmd { _, cmd := rv.dragKey(a, key); return cmd }
	}
	item := []Action{{Key: "Enter", Label: "Drop", Desc: "it here", Do: do("Enter")}}
	// Leaving says the hint's verb, Esc:cancel (tdp menu.4).
	keys := []Action{moveRow("h|k", "move it up"), moveRow("j|l", "move it down"), leaveRow("cancel", do("Esc"))}
	return a.modeMenu(a.modeTitle(), "Drag", item, keys)
}

// terminalModeMenu is the Space menu of a terminal's History or selection
// mode; Space no longer goes to the program there.
func (a *App) terminalModeMenu(t *Terminal) tea.Cmd {
	if t.mode == visualMode {
		return a.selectModeMenu(t.sel, func(key string) tea.Cmd { cmd, _ := t.modeKey(a, key); return cmd })
	}
	keys := []Action{
		moveRow("j|k", "a line down, up"), moveRow("u|d", "half a page up, down"),
		moveRow("PgUp|PgDn", "half a page up, down"), moveRow("gg|G", "the oldest line, the live screen's last"),
		leaveRow("live", func() tea.Cmd { t.leaveMode(); return nil }),
	}
	return a.modeMenu(a.modeTitle(), historyMode, nil, keys)
}
