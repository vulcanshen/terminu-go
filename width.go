// Package terminu draws and drives a terminal UI the way terminu design (tdp)
// says, on top of Bubble Tea v1.
package terminu

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"
)

// Every width on the screen goes through Width (tdp E5): padding, cutting,
// centring, side by side, frames and popups laid over the screen. A Nerd Font
// glyph advances the cursor as many cells as the terminal decides, which
// x/ansi cannot know; the lib measures it at startup (Probe).
var (
	iconCells      = 1 // a Nerd Font icon
	powerlineCells = 1 // a powerline symbol: capsule ends, joins, the cursor arrow (tdp E5)
)

// IconWidth is how many cells a Nerd Font icon takes on this terminal.
func IconWidth() int { return iconCells }

// PowerlineWidth is how many cells a powerline symbol takes on this terminal.
func PowerlineWidth() int { return powerlineCells }

// SetIconWidth sets how many cells a Nerd Font icon takes, and a powerline
// symbol. Probe sets them; tests and apps that already know may set them.
func SetIconWidth(icon, powerline int) {
	iconCells, powerlineCells = clampCells(icon), clampCells(powerline)
}

func clampCells(n int) int { return min(max(n, 1), 2) }

// isPowerline reports whether r is a powerline symbol (U+E0A0–U+E0D7). Where a
// terminal draws file icons two cells wide, these often stay one (tdp E5), so
// they are measured on their own.
func isPowerline(r rune) bool { return r >= 0xE0A0 && r <= 0xE0D7 }

// isIcon reports whether r is in a private use area, where Nerd Font glyphs live.
func isIcon(r rune) bool {
	return (r >= 0xE000 && r <= 0xF8FF) || (r >= 0xF0000 && r <= 0xFFFFD) || (r >= 0x100000 && r <= 0x10FFFD)
}

// clusterWidth corrects the width x/ansi measured for one grapheme cluster.
func clusterWidth(g string, measured int) int {
	r, n := utf8.DecodeRuneInString(g)
	if n != len(g) {
		return measured
	}
	switch {
	case isPowerline(r):
		return powerlineCells
	case isIcon(r):
		return iconCells
	}
	return measured
}

// walk calls f for every escape sequence (cells == 0) and every grapheme
// cluster (cells > 0) of one line.
func walk(s string, f func(seq string, cells int)) {
	var state byte
	for len(s) > 0 {
		seq, w, n, next := ansi.DecodeSequence(s, state, nil)
		if n <= 0 {
			return
		}
		if w > 0 {
			w = clusterWidth(seq, w)
		}
		f(seq, w)
		state = next
		s = s[n:]
	}
}

// Width is how many cells s takes on the screen: its widest line, escape
// sequences ignored.
func Width(s string) int {
	widest := 0
	for line := range strings.SplitSeq(s, "\n") {
		w := 0
		walk(line, func(_ string, cells int) { w += cells })
		widest = max(widest, w)
	}
	return widest
}

// Cut keeps the cells [from, to) of one line. Escape sequences are kept so the
// styles stay right, and the result ends with a style reset. A wide character
// cut in half becomes spaces, so the result is always exactly as wide as the
// part of [from, to) the line covers.
func Cut(s string, from, to int) string {
	var b strings.Builder
	col := 0
	walk(s, func(seq string, cells int) {
		if cells == 0 {
			b.WriteString(seq)
			return
		}
		start, end := col, col+cells
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

// Pad cuts or pads one line with spaces to exactly w cells.
func Pad(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if d := Width(s); d < w {
		return s + strings.Repeat(" ", w-d)
	} else if d == w {
		return s
	}
	return Cut(s, 0, w)
}

// PadLeft right-aligns one line in w cells.
func PadLeft(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if d := Width(s); d < w {
		return strings.Repeat(" ", w-d) + s
	}
	return Pad(s, w)
}

// Center centres one line in w cells; the smaller half of the gap goes left.
func Center(s string, w int) string {
	d := Width(s)
	if d >= w {
		return Pad(s, w)
	}
	gap := w - d
	return strings.Repeat(" ", gap/2) + s + strings.Repeat(" ", gap-gap/2)
}

// Truncate fits one line in w cells, cutting from the end and adding "…" when
// it had to cut. The result may be narrower than w when a wide character
// would straddle the cut.
func Truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if Width(s) <= w {
		return s
	}
	if w == 1 {
		return "…"
	}
	cut := trimRight(Cut(s, 0, w-1))
	// Plain text stays plain: a reset in it would end the style of whatever
	// it is drawn in (the title capsule's background before its "…").
	if !strings.Contains(s, "\x1b") {
		cut = strings.TrimSuffix(cut, ansi.ResetStyle)
	}
	return cut + "…"
}

// TruncateLeft fits one line in w cells, cutting from the front and putting
// "…" first: for paths and for a value whose end matters.
func TruncateLeft(s string, w int) string {
	if w <= 0 {
		return ""
	}
	d := Width(s)
	if d <= w {
		return s
	}
	if w == 1 {
		return "…"
	}
	return "…" + trimLeft(Cut(s, d-(w-1), d))
}

// trimRight drops the spaces Cut put where a wide character was cut in half,
// so "…" sits right after the last whole character.
func trimRight(s string) string {
	reset := ansi.ResetStyle
	body := strings.TrimSuffix(s, reset)
	return strings.TrimRight(body, " ") + reset
}

func trimLeft(s string) string { return strings.TrimLeft(s, " ") }

// Fit pads or truncates (with "…") one line to exactly w cells.
func Fit(s string, w int) string { return Pad(Truncate(s, w), w) }

// FitLeft is Fit cutting from the front.
func FitLeft(s string, w int) string { return Pad(TruncateLeft(s, w), w) }

// Composite lays the lines of top over base, top's first cell at column x and
// row y. Whatever falls off the screen is cut; it never panics, however big top
// is (tdp E5). base is changed in place and returned; every line keeps base's
// width.
func Composite(base, top []string, x, y, width int) []string {
	x, y = max(x, 0), max(y, 0)
	if x >= width {
		return base
	}
	for i, line := range top {
		row := y + i
		if row >= len(base) {
			break
		}
		mid := Cut(line, 0, width-x)
		right := x + Width(mid)
		base[row] = Cut(base[row], 0, x) + mid + Cut(base[row], right, width)
	}
	return base
}

// Repeat is strings.Repeat that takes a negative count as zero.
func Repeat(s string, n int) string { return strings.Repeat(s, max(n, 0)) }

// Wrap breaks plain text into lines of at most w cells, at spaces where it
// can and inside a word that is longer than a line. Line breaks in s are kept.
func Wrap(s string, w int) []string {
	if w <= 0 {
		return nil
	}
	var out []string
	for para := range strings.SplitSeq(s, "\n") {
		line, lineW := "", 0
		flush := func() {
			out = append(out, line)
			line, lineW = "", 0
		}
		for _, word := range strings.Fields(para) {
			ww := Width(word)
			if lineW > 0 && lineW+1+ww > w {
				flush()
			}
			if ww > w {
				// A word longer than a line: on lines of its own, cut by cells.
				if lineW > 0 {
					flush()
				}
				for ww > w {
					head := strings.TrimSuffix(Cut(word, 0, w), ansi.ResetStyle)
					hw := max(Width(head), 1)
					line, lineW = head, hw
					flush()
					word = strings.TrimSuffix(Cut(word, hw, ww), ansi.ResetStyle)
					ww = Width(word)
				}
			}
			if ww == 0 {
				continue
			}
			if lineW > 0 {
				line += " "
				lineW++
			}
			line += word
			lineW += ww
		}
		flush()
	}
	return out
}

// StripStyles drops the escape sequences of s.
func StripStyles(s string) string { return ansi.Strip(s) }
