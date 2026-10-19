package terminu

import (
	"strings"
	"unicode/utf8"

	"github.com/rivo/uniseg"
)

// Glyphs tdp names by code point (Rules E4: written as code points).
const (
	glyphSlantLeft  = "\U0000E0BA" // nf-ple-lower_right_triangle: a capsule chain's left end
	glyphHard       = "\U0000E0B0" // nf-pl-left_hard_divider: right end, a join where the colour changes, the cursor arrow
	glyphSoft       = "\U0000E0B1" // nf-pl-left_soft_divider: a join between parts of the same colour
	glyphRoundLeft  = "\U0000E0B6" // nf-ple-left_half_circle_thick: a popup title's left end
	glyphRoundRight = "\U0000E0B4" // nf-ple-right_half_circle_thick: a popup title's right end
	glyphHexagon    = "\U000F02D8" // nf-md-hexagon: before a section title
	glyphTyping     = "\U0000F105" // nf-fa-angle_right: before a typing row that filters
	glyphSortAsc    = "\U0000F160" // nf-fa-sort_amount_asc
	glyphSortDesc   = "\U0000F161" // nf-fa-sort_amount_desc
	glyphMarked     = "\U000F0132" // nf-md-checkbox_marked
	glyphUnchecked  = "\U000F0131" // nf-md-checkbox_blank_outline
	glyphSwap       = "\U000F04E2" // nf-md-swap_vertical: the item held in the Drag mode
	glyphRadioOn    = "\U000F043E" // nf-md-radiobox_marked
	glyphRadioOff   = "\U000F043D" // nf-md-radiobox_blank
	glyphMenu       = "\U0000F0C9" // nf-fa-bars
	glyphKeys       = "\U0000F059" // nf-fa-question_circle
	glyphConfirm    = "\U000F0ECC" // nf-md-shield_alert
	glyphTerminal   = "\U000F018D" // nf-md-console
	glyphInfo       = "\U0000F024" // nf-fa-flag
	glyphWarning    = "\U000F078F" // nf-md-alarm_light
	glyphError      = "\U000F0238" // nf-md-fire
	glyphSuccess    = "\U000F05E0" // nf-md-check_circle
)

// loadingFrames are nf-md-circle_slice_1 to _8 (components/layout/popup).
var loadingFrames = [8]string{
	"\U000F0A9E", "\U000F0A9F", "\U000F0AA0", "\U000F0AA1", "\U000F0AA2", "\U000F0AA3", "\U000F0AA4", "\U000F0AA5",
}

// lineSet is one kind of frame line.
type lineSet struct {
	tl, tr, bl, br, h, v, jl, jr string // jl/jr: the junctions around a mode name
}

var (
	heavyLines = lineSet{"┏", "┓", "┗", "┛", "━", "┃", "┫", "┣"}
	lightLines = lineSet{"┌", "┐", "└", "┘", "─", "│", "┤", "├"}
)

// Capsule is one part of a capsule chain: a panel's [N] label or a tab, a
// screen chip.
type Capsule struct {
	Text     string
	Selected bool
}

// shrunk is the capsule's first character or glyph, or its hotkey marker:
// "[M]ANAGE" becomes "[M]".
func shrunk(text string) string {
	if strings.HasPrefix(text, "[") {
		if i := strings.IndexByte(text, ']'); i > 0 {
			return text[:i+1]
		}
	}
	g, _, _, _ := uniseg.FirstGraphemeClusterInString(text, -1)
	return g
}

// drawChain draws a capsule chain (components/layout/panel) in color, in at
// most maxW cells: slanted ends, selected parts filled, joins U+E0B0 where the
// colour changes and U+E0B1 where it does not. When it does not fit,
// unselected parts shrink to their first character, then are folded away from
// the end behind a "…"; the selected ones always stay. The ends and joins are
// coloured as in Powerline: the left part's ground over the right part's, an
// unselected part's ground taken as Base (layout/panel, capsule chain).
func drawChain(caps []Capsule, color Color, maxW int) string {
	if len(caps) == 0 || maxW <= 0 {
		return ""
	}
	parts := append([]Capsule(nil), caps...)
	draw := func(ps []Capsule, folded bool) string {
		ground := func(c Capsule) Color {
			if c.Selected {
				return color
			}
			return Base
		}
		var b strings.Builder
		b.WriteString(Fg(ground(ps[0])).Render(glyphSlantLeft))
		for i, p := range ps {
			if i > 0 {
				prev := ps[i-1]
				switch {
				case prev.Selected && p.Selected:
					b.WriteString(Style{Fg: Base, Bg: color}.Render(glyphSoft))
				case !prev.Selected && !p.Selected:
					b.WriteString(Fg(color).Render(glyphSoft))
				default:
					b.WriteString(Style{Fg: ground(prev), Bg: ground(p)}.Render(glyphHard))
				}
			}
			if p.Selected {
				b.WriteString(On(color).Render(p.Text))
			} else {
				b.WriteString(Fg(color).Render(p.Text))
			}
		}
		b.WriteString(Fg(ground(ps[len(ps)-1])).Render(glyphHard))
		if folded {
			b.WriteString(Fg(color).Render("…"))
		}
		return b.String()
	}
	if s := draw(parts, false); Width(s) <= maxW {
		return s
	}
	for i := range parts {
		if !parts[i].Selected {
			parts[i].Text = shrunk(parts[i].Text)
		}
	}
	if s := draw(parts, false); Width(s) <= maxW {
		return s
	}
	for j := len(parts) - 1; j >= 0; j-- {
		if parts[j].Selected {
			continue
		}
		parts = append(parts[:j], parts[j+1:]...)
		if s := draw(parts, true); Width(s) <= maxW {
			return s
		}
	}
	return Cut(draw(parts, len(parts) < len(caps)), 0, maxW)
}

// titleCapsule is a popup's title: glyph and text, upper case, in a capsule
// rounded at both ends, filled with color (components/layout/popup).
func titleCapsule(glyph, title string, color Color, maxW int) string {
	text := title
	if glyph != "" {
		text = glyph + " " + title
	}
	ends := Width(glyphRoundLeft) + Width(glyphRoundRight)
	text = Truncate(text, maxW-ends)
	if text == "" {
		return ""
	}
	return Fg(color).Render(glyphRoundLeft) + On(color).Render(text) + Fg(color).Render(glyphRoundRight)
}

// scrollText is a scroll position: "3 of 40" with a cursor, "1-20 of 120"
// for what is seen without one.
// scrollRoom is the width a popup that scrolls needs for its whole hint and
// its scroll position at the widest (components/layout/popup): `┗━ `, the
// hint, at least ` ━ `, the position, ` ━┛`.
func scrollRoom(pairs []KeyHint, widest string) int { return hintWidth(pairs) + Width(widest) + 9 }

func scrollText(cursor bool, first, last, total int) string {
	if cursor {
		return itoa(first) + " of " + itoa(total)
	}
	return itoa(first) + "-" + itoa(last) + " of " + itoa(total)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// box is everything that goes on a frame.
type box struct {
	width     int
	lines     lineSet
	color     Color  // the frame's colour
	head      string // drawn right after the top-left corner (a panel's capsule chain)
	title     string // drawn after one line segment (a popup's title capsule, its tabs)
	after     string // after head or title, one space either side (the loading icon)
	mode      string // the mode name, between junctions at the right of the top border
	hint      []KeyHint
	hintOff   []bool // the pairs that cannot be pressed now, in the disabled colour
	hintKey   Style  // the colour of the hint's keys
	hintWords Color  // the colour of its colons and words; Overlay0 when unset
	scroll    string // the scroll position at the right of the bottom border
	body      []string
}

// draw draws the frame around body, every line exactly b.width cells. Text on
// the frame keeps one line segment from each corner; only a panel's capsule
// chain touches the top-left one.
func (b box) draw() []string {
	inner := b.width - 2
	if inner < 0 {
		return nil
	}
	ls, frame := b.lines, Fg(b.color)
	out := make([]string, 0, len(b.body)+2)

	// Top: corner, chain or title, the loading icon, line, mode, corner.
	var top strings.Builder
	used := 0
	put := func(s string) {
		top.WriteString(s)
		used += Width(s)
	}
	modeW := 0
	if b.mode != "" {
		modeW = Width(b.mode) + 3 // junction, name, junction, one segment before the corner
	}
	room := inner - modeW
	if room < 2 && b.mode != "" {
		b.mode, modeW, room = "", 0, inner
	}
	if b.head != "" {
		put(Cut(b.head, 0, max(room, 0)))
	}
	if b.title != "" && room > 1 {
		put(frame.Render(ls.h))
		put(Cut(b.title, 0, room-used))
	}
	if b.after != "" && room-used >= Width(b.after)+2 {
		put(" " + b.after + " ")
	}
	top.WriteString(frame.Render(Repeat(ls.h, inner-used-modeW)))
	if b.mode != "" {
		top.WriteString(frame.Render(ls.jl) + Fg(Yellow).Bolded().Render(b.mode) + frame.Render(ls.jr+ls.h))
	}
	out = append(out, frame.Render(ls.tl)+top.String()+frame.Render(ls.tr))

	for _, l := range b.body {
		out = append(out, frame.Render(ls.v)+Pad(l, inner)+frame.Render(ls.v))
	}

	// Bottom: corner, one segment, hint, line, scroll position, one segment, corner.
	scroll := ""
	if b.scroll != "" && inner >= Width(b.scroll)+4 {
		scroll = " " + b.scroll + " "
	}
	tail := ls.h
	if scroll == "" {
		tail = ""
	}
	// A segment, the space either side, at least one segment after it, the
	// scroll position and its segment. A popup is made wide enough for both
	// (scrollRoom); pairs go from the end only on a terminal too narrow.
	hintRoom := inner - 1 - 2 - 1 - Width(scroll) - Width(tail)
	hint := ""
	if len(b.hint) > 0 && hintRoom > 0 {
		if h := renderHintsOff(b.hint, b.hintOff, hintRoom, b.hintKey, b.words()); h != "" {
			hint = " " + h + " "
		}
	}
	fill := inner - 1 - Width(hint) - Width(scroll) - Width(tail)
	out = append(out, frame.Render(ls.bl+ls.h)+hint+frame.Render(Repeat(ls.h, fill))+frame.Render(scroll)+frame.Render(tail+ls.br))
	return out
}

// runeCount counts the characters of s.
func runeCount(s string) int { return utf8.RuneCountInString(s) }

func (b box) words() Color {
	if b.hintWords.Set {
		return b.hintWords
	}
	return Overlay0
}
