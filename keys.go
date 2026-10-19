package terminu

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

// KeyName is the name of the key in msg as tdp writes it (Rules M5): the
// keycap name in upper camel case (Enter, Esc, Tab, Space, Backspace, Delete,
// Home, End, PgUp, PgDn), arrows as ↑ ↓ ← →, a letter as typed (q, A),
// modifiers joined with "-" and Ctrl's letter in upper case (Ctrl-C, Alt-t,
// Shift-Tab, Alt-Esc). A paste has no name: it is "".
func KeyName(msg tea.KeyMsg) string {
	if msg.Paste {
		return ""
	}
	s := msg.String()
	alt := strings.HasPrefix(s, "alt+")
	s = strings.TrimPrefix(s, "alt+")
	name := ""
	switch {
	case s == " ":
		name = "Space"
	case strings.HasPrefix(s, "ctrl+") && utf8.RuneCountInString(s) == len("ctrl+")+1:
		name = "Ctrl-" + strings.ToUpper(s[len("ctrl+"):])
	case named[s] != "":
		name = named[s]
	case msg.Type == tea.KeyRunes:
		name = string(msg.Runes)
	default:
		name = s
	}
	if alt {
		return "Alt-" + name
	}
	return name
}

var named = map[string]string{
	"enter": "Enter", "esc": "Esc", "tab": "Tab", "shift+tab": "Shift-Tab", "backspace": "Backspace",
	"ctrl+h": "Backspace", "delete": "Delete", "home": "Home", "end": "End", "pgup": "PgUp", "pgdown": "PgDn",
	"up": "↑", "down": "↓", "left": "←", "right": "→", "insert": "Insert",
	"ctrl+up": "Ctrl-↑", "ctrl+down": "Ctrl-↓", "ctrl+left": "Ctrl-←", "ctrl+right": "Ctrl-→",
	"shift+up": "Shift-↑", "shift+down": "Shift-↓", "shift+left": "Shift-←", "shift+right": "Shift-→",
}

// isLetters reports whether key is made of letters only ("r", "D", "go").
func isLetters(key string) bool {
	if key == "" {
		return false
	}
	for _, r := range key {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

// modifierKey splits "Alt-t" into "Alt-" and "t".
func modifierKey(key string) (mod, rest string, ok bool) {
	for _, m := range []string{"Ctrl-", "Alt-", "Shift-"} {
		if r, found := strings.CutPrefix(key, m); found && r != "" {
			return m, r, true
		}
	}
	return "", "", false
}

// MarkLabel writes a hotkey into its label the way Rules M5 says. A letter
// that is in the label, in either case, goes in place of the first one, written
// as the key is pressed ("[r]ename", "[n]ew", "D[e]lete", "Ex[P]ort": Shift-P);
// one that is not goes in front ("[x] Reset"). A modifier key goes in place of
// the letter it ends with the same way ("[Alt-t]erm"), several letters in place
// at the start ("[go]to"), a digit, a core key or a symbol always in front
// ("[3] Favorites", "[Enter] Edit", "[/] Search"). The label keeps its own
// case. A label that already shows its key, and a row with no key, are left as
// they are.
func MarkLabel(key, label string) string {
	if key == "" || strings.Contains(label, "["+key+"]") {
		return label
	}
	mod, letter := "", key
	if m, rest, ok := modifierKey(key); ok {
		mod, letter = m, rest
	}
	if utf8.RuneCountInString(letter) == 1 && isLetters(letter) {
		if i := indexFold(label, letter); i >= 0 {
			_, n := utf8.DecodeRuneInString(label[i:])
			return label[:i] + "[" + mod + letter + "]" + label[i+n:]
		}
		return "[" + key + "] " + label
	}
	if mod == "" && isLetters(key) && len(label) >= len(key) && strings.EqualFold(label[:len(key)], key) {
		return "[" + key + "]" + label[len(key):] // several letters, in either case: [go]to on Goto
	}
	return "[" + key + "] " + label
}

// indexFold is where the letter first is in s, in either case; -1 if nowhere.
func indexFold(s, letter string) int {
	for i, r := range s {
		if strings.EqualFold(string(r), letter) {
			return i
		}
	}
	return -1
}

// Upper writes text in upper case for a capsule, a title, a header or a
// section title (components, "Uppercase"), leaving the letters in a hotkey
// marker as they are: "[m]anage" becomes "[m]ANAGE".
func Upper(text string) string {
	var b strings.Builder
	inMark := false
	for _, r := range text {
		switch {
		case r == '[':
			inMark = true
		case r == ']':
			inMark = false
		case !inMark:
			r = unicode.ToUpper(r)
		}
		b.WriteRune(r)
	}
	return b.String()
}

// popupTitle is a popup's title text: what it is for in upper case, then
// the name of what it acts on after ": ", as it is (components/layout/popup):
// "EDIT: filu".
func popupTitle(title, subject string) string {
	t := Upper(title)
	if subject != "" {
		t += ": " + subject
	}
	return t
}

// KeyHint is one "key:what" pair of a hint or the footer (Rules M5).
type KeyHint struct {
	Key, Desc string
}

// renderKey draws a key in keyStyle; keys doing one thing are joined with
// "|" in the words' colour, not the key's (Rules M5: "j|k").
func renderKey(key string, keyStyle Style, words Color) string {
	parts := strings.Split(key, "|")
	for i, p := range parts {
		parts[i] = keyStyle.Render(p)
	}
	return strings.Join(parts, Fg(words).Render("|"))
}

// renderHints writes pairs as "Key:desc Key:desc", keys in keyStyle and the
// colon and words in Overlay0, dropping whole pairs from the end until they fit
// in w cells; a last pair that closes with Esc stays, the ones before it going
// (components/layout/popup, hint).
func renderHints(pairs []KeyHint, w int, keyStyle Style, words Color) string {
	return renderHintsOff(pairs, nil, w, keyStyle, words)
}

// renderHintsOff is renderHints with the pairs that cannot be pressed now
// (off) all in the disabled colour (Rules M6).
func renderHintsOff(pairs []KeyHint, off []bool, w int, keyStyle Style, words Color) string {
	if len(pairs) == 0 {
		return ""
	}
	last := len(pairs) - 1
	keepEsc := slices.Contains(strings.Split(pairs[last].Key, "|"), "Esc")
	for n := len(pairs); n > 0; n-- {
		idx := make([]int, 0, n)
		for i := range n {
			idx = append(idx, i)
		}
		if keepEsc && n <= last {
			// The Esc pair stays: the pairs before it go first.
			idx = append(idx[:max(n-1, 0)], last)
		}
		parts := make([]string, len(idx))
		width := len(idx) - 1
		for k, i := range idx {
			p := pairs[i]
			if i < len(off) && off[i] {
				parts[k] = renderKey(p.Key, Fg(Surface2), Surface2) + Fg(Surface2).Render(":"+p.Desc)
			} else {
				parts[k] = renderKey(p.Key, keyStyle, words) + Fg(words).Render(":"+p.Desc)
			}
			width += Width(p.Key) + 1 + Width(p.Desc)
		}
		if width <= w {
			return strings.Join(parts, " ")
		}
	}
	return ""
}
