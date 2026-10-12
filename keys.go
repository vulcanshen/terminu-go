package terminu

import (
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

// MarkLabel writes a hotkey into its label the way Rules M5 says: in place
// when it is the label's first letter ("[r]ename") or a letter inside it
// ("UR[L]"), in front otherwise ("[n] New"); a modifier key in place of the
// letter it ends with ("[Alt-t]erm"), several letters in place ("[go]to"), a
// digit, a core key or a symbol always in front ("[3] Favorites",
// "[Enter] Edit", "[/] Search"). The case counts: "[A]dd" is Shift-A. A label
// that already shows its key, and a row with no key, are left as they are.
func MarkLabel(key, label string) string {
	if key == "" || strings.Contains(label, "["+key+"]") {
		return label
	}
	if mod, rest, ok := modifierKey(key); ok && utf8.RuneCountInString(rest) == 1 && isLetters(rest) {
		if strings.HasPrefix(label, rest) {
			return "[" + mod + rest + "]" + label[len(rest):]
		}
		return "[" + key + "] " + label
	}
	if isLetters(key) {
		if strings.HasPrefix(label, key) {
			return "[" + key + "]" + label[len(key):]
		}
		if utf8.RuneCountInString(key) == 1 {
			if i := strings.Index(label, key); i >= 0 {
				return label[:i] + "[" + key + "]" + label[i+len(key):]
			}
		}
	}
	return "[" + key + "] " + label
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

// KeyHint is one "key:what" pair of a hint or the footer (Rules M5).
type KeyHint struct {
	Key, Desc string
}

// renderHints writes pairs as "Key:desc Key:desc", keys in keyStyle and the
// colon and words in Overlay0, dropping whole pairs from the end until they fit
// in w cells (components/layout/popup, hint).
func renderHints(pairs []KeyHint, w int, keyStyle Style, words Color) string {
	for n := len(pairs); n > 0; n-- {
		parts := make([]string, n)
		width := n - 1
		for i, p := range pairs[:n] {
			parts[i] = keyStyle.Render(p.Key) + Fg(words).Render(":"+p.Desc)
			width += Width(p.Key) + 1 + Width(p.Desc)
		}
		if width <= w {
			return strings.Join(parts, " ")
		}
	}
	return ""
}
