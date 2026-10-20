package tdptest

import (
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"

	terminu "github.com/vulcanshen/terminu-go"
)

// keyTypes are the keys tdp names by a word or an arrow (tdp M5).
var keyTypes = map[string]tea.KeyType{
	"Enter": tea.KeyEnter, "Esc": tea.KeyEsc, "Tab": tea.KeyTab, "Shift-Tab": tea.KeyShiftTab,
	"Space": tea.KeySpace, "Backspace": tea.KeyBackspace, "Delete": tea.KeyDelete, "Insert": tea.KeyInsert,
	"Home": tea.KeyHome, "End": tea.KeyEnd, "PgUp": tea.KeyPgUp, "PgDn": tea.KeyPgDown,
	"↑": tea.KeyUp, "↓": tea.KeyDown, "←": tea.KeyLeft, "→": tea.KeyRight,
	"Ctrl-↑": tea.KeyCtrlUp, "Ctrl-↓": tea.KeyCtrlDown, "Ctrl-←": tea.KeyCtrlLeft, "Ctrl-→": tea.KeyCtrlRight,
	"Shift-↑": tea.KeyShiftUp, "Shift-↓": tea.KeyShiftDown, "Shift-←": tea.KeyShiftLeft, "Shift-→": tea.KeyShiftRight,
}

// keyMsg is the key named as tdp writes it (terminu.KeyName); ok false for
// a name that is not one.
func keyMsg(name string) (msg tea.KeyMsg, ok bool) {
	rest, alt := strings.CutPrefix(name, "Alt-")
	switch t, named := keyTypes[rest]; {
	case named:
		msg = tea.KeyMsg{Type: t}
		if t == tea.KeySpace {
			msg.Runes = []rune{' '}
		}
	case len(rest) == len("Ctrl-C") && strings.HasPrefix(rest, "Ctrl-") && rest[5] >= 'A' && rest[5] <= 'Z':
		msg = tea.KeyMsg{Type: tea.KeyCtrlA + tea.KeyType(rest[5]-'A')}
	case utf8.RuneCountInString(rest) == 1:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(rest)}
	default:
		return msg, false
	}
	msg.Alt = alt
	return msg, terminu.KeyName(msg) == name
}

// typed is the key a terminal sends for one character typed: a control
// character is its own key (Tab, Enter, Esc, Backspace, Ctrl-…), a space
// is Space.
func typed(r rune) tea.KeyMsg {
	switch {
	case r == ' ':
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	case r < 0x20 || r == 0x7f:
		return tea.KeyMsg{Type: tea.KeyType(r)}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

// burst is text as the keys a terminal sends when it comes all at once:
// the characters together in one key, split where Bubble Tea splits them,
// at a space and a control character.
func burst(s string) []tea.KeyMsg {
	var out []tea.KeyMsg
	var run []rune
	for _, r := range s {
		if r == ' ' || r < 0x20 || r == 0x7f {
			if len(run) > 0 {
				out = append(out, tea.KeyMsg{Type: tea.KeyRunes, Runes: run})
				run = nil
			}
			out = append(out, typed(r))
			continue
		}
		run = append(run, r)
	}
	if len(run) > 0 {
		out = append(out, tea.KeyMsg{Type: tea.KeyRunes, Runes: run})
	}
	return out
}
