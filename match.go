package terminu

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Matches reports whether text matches what was typed in a filter
// (components/input/finder, matching): every space-separated word must appear
// in text as typed, in one piece, ignoring case; the words may come in any
// order. Nothing typed matches everything.
func Matches(text, typed string) bool {
	text = strings.ToLower(ansi.Strip(text))
	for _, w := range strings.Fields(strings.ToLower(typed)) {
		if !strings.Contains(text, w) {
			return false
		}
	}
	return true
}

// Filter returns the indexes of the items that match typed, in their order:
// a filter never reorders.
func Filter(items []string, typed string) []int {
	out := make([]int, 0, len(items))
	for i, it := range items {
		if Matches(it, typed) {
			out = append(out, i)
		}
	}
	return out
}
