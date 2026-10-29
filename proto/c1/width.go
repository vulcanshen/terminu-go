// Package main is a throwaway prototype that checked one question: does
// lipgloss v2's layer compositing keep frames straight when a Nerd Font icon
// advances the cursor two cells (tdp E5)? It is not the lib.
package main

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

// iconWidth is how many cells a Nerd Font icon advances the cursor (tdp E5).
var iconWidth = 1

// isIcon reports whether r is in a private use area, where Nerd Font glyphs live.
func isIcon(r rune) bool {
	return (r >= 0xE000 && r <= 0xF8FF) ||
		(r >= 0xF0000 && r <= 0xFFFFD) ||
		(r >= 0x100000 && r <= 0x10FFFD)
}

// cellWidth corrects the width ansi measured for one grapheme cluster.
func cellWidth(g string, measured int) int {
	if r, n := utf8.DecodeRuneInString(g); n == len(g) && isIcon(r) {
		return iconWidth
	}
	return measured
}

// dispWidth is the one display-width function E5 asks for: widest line, icons
// counted as iconWidth, escape sequences ignored.
func dispWidth(s string) int {
	widest := 0
	for _, line := range strings.Split(s, "\n") {
		w := 0
		walk(line, func(seq string, cw int) {
			w += cw
		})
		widest = max(widest, w)
	}
	return widest
}

// walk calls f for every escape sequence (cw == 0) and grapheme cluster (cw > 0).
func walk(s string, f func(seq string, cw int)) {
	var state byte
	for len(s) > 0 {
		seq, w, n, next := ansi.DecodeSequence(s, state, nil)
		if w > 0 {
			w = cellWidth(seq, w)
		}
		f(seq, w)
		state = next
		s = s[n:]
	}
}

// cut keeps the cells [from, to) of one line. Escape sequences are all kept, so
// styles stay right; a wide cluster cut in half becomes spaces.
func cut(s string, from, to int) string {
	var b strings.Builder
	col := 0
	walk(s, func(seq string, cw int) {
		if cw == 0 {
			b.WriteString(seq)
			return
		}
		start, end := col, col+cw
		col = end
		switch {
		case end <= from || start >= to:
		case start >= from && end <= to:
			b.WriteString(seq)
		default:
			b.WriteString(strings.Repeat(" ", min(end, to)-max(start, from)))
		}
	})
	b.WriteString(ansi.ResetStyle)
	return b.String()
}

// ansiStrip drops escape sequences.
func ansiStrip(s string) string { return ansi.Strip(s) }

// pad cuts or pads one line to exactly w cells.
func pad(s string, w int) string {
	if d := dispWidth(s); d < w {
		return s + strings.Repeat(" ", w-d)
	}
	return cut(s, 0, w)
}

// composite lays top over base at (x, y). Anything off screen is cut; it never
// panics, whatever the sizes (tdp E5).
func composite(base, top []string, x, y, width int) []string {
	out := append([]string(nil), base...)
	x, y = max(x, 0), max(y, 0)
	for i, line := range top {
		row := y + i
		if row >= len(out) {
			break
		}
		if x >= width {
			continue
		}
		mid := cut(line, 0, width-x)
		right := x + dispWidth(mid)
		out[row] = cut(out[row], 0, x) + mid + cut(out[row], right, width)
	}
	return out
}
