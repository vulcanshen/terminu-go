package terminu

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ColorPicker picks a colour (components/input/color-picker): old and new
// side by side, the red, green and blue channels each chosen from a select.
type ColorPicker struct {
	Title, Glyph string
	Value        Color
	// Done gets the new colour. It is not called when it did not change.
	Done func(Color) tea.Cmd
}

func (c ColorPicker) typed() bool { return false }

// PickColor opens a color picker.
func (a *App) PickColor(c ColorPicker) tea.Cmd { return c.open(a, nil) }

type colorView struct {
	c       ColorPicker
	rgb     RGB
	focus   int // 0 new, 1 R, 2 G, 3 B
	written func()
}

func (c ColorPicker) open(a *App, written func()) tea.Cmd {
	return a.open(&colorView{c: c, rgb: c.Value.RGB, focus: 1, written: written})
}

// HexText writes a colour as the family does: "#2A2A3C".
func HexText(c Color) string { return strings.ToUpper(c.String()) }

// ColorText is a colour on a form or a panel: a small swatch and its hex
// (components/input/color-picker).
func ColorText(c Color) string {
	return Fg(c).Render("███") + " " + Fg(Text).Render(HexText(c))
}

// The row: old and new 8 cells wide, R, G, B 3 cells, two cells apart.
var colorCols = [5]int{8, 8, 3, 3, 3}

const colorRowW = 8 + 2 + 8 + 2 + 3 + 2 + 3 + 2 + 3

func (cv *colorView) hint() []KeyHint {
	if cv.focus == 0 {
		return []KeyHint{{"Enter", "choose"}, {"Esc", "cancel"}}
	}
	return []KeyHint{{"Enter", "edit"}, {"Esc", "cancel"}}
}

func (cv *colorView) measure(sw, sh int) (int, int) {
	w := max(colorRowW+4, Width(titleCapsule(cv.c.Glyph, Upper(cv.c.Title), Base, 1<<20))+4,
		hintWidth([]KeyHint{{"Enter", "choose"}, {"Esc", "cancel"}})+6)
	return min(w, max(min(sw-2, 120), 1)), min(4+4, max(sh-2, 3))
}

func (cv *colorView) draw(w, h int, layer Color) []string {
	inner := w - 2
	newC := Color{RGB: cv.rgb, Set: true}
	chans := [3]Color{
		{RGB: RGB{cv.rgb[0], 0, 0}, Set: true},
		{RGB: RGB{0, cv.rgb[1], 0}, Set: true},
		{RGB: RGB{0, 0, cv.rgb[2]}, Set: true},
	}
	cell := func(i int, s string) string { return Pad(s, colorCols[i]) }
	join := func(parts ...string) string { return strings.Join(parts, "  ") }
	big := func(c Color) string { return Fg(c).Render(Repeat("█", 8)) }
	small := func(c Color) string { return Fg(c).Render(Repeat("█", 3)) }
	labels := []string{"old", "new", "R", "G", "B"}
	var labelRow []string
	for i, l := range labels {
		text := Fg(Overlay0).Render(l)
		// TODO(tdp G34): the focus is drawn on the name only, the name's width.
		if i > 0 && i-1 == cv.focus {
			text = On(layer).Render(l)
		}
		labelRow = append(labelRow, cell(i, text))
	}
	hexes := []string{HexText(cv.c.Value), HexText(newC)}
	for i := range 3 {
		hexes = append(hexes, fmt.Sprintf("%02X", cv.rgb[i]))
	}
	var hexRow []string
	for i, x := range hexes {
		hexRow = append(hexRow, cell(i, Fg(Text).Render(x)))
	}
	body := []string{
		padRow(join(big(cv.c.Value), big(newC)), inner),
		padRow(join(big(cv.c.Value), big(newC), small(chans[0]), small(chans[1]), small(chans[2])), inner),
		padRow(join(labelRow...), inner),
		padRow(join(hexRow...), inner),
	}
	return popupBox{color: layer, glyph: cv.c.Glyph, title: Upper(cv.c.Title), hint: cv.hint()}.draw(w, Pad2(body, inner, h-4))
}

func (cv *colorView) key(a *App, key string, _ tea.KeyMsg) (bool, tea.Cmd) {
	switch key {
	case "h", "k", "←", "↑":
		cv.focus = (cv.focus + 3) % 4
	case "l", "j", "→", "↓":
		cv.focus = (cv.focus + 1) % 4
	case "Enter":
		if cv.focus == 0 {
			if cv.rgb == cv.c.Value.RGB || cv.c.Done == nil {
				return true, a.closeTop()
			}
			return true, tea.Batch(a.closeTop(), then(cv.written, cv.c.Done(Color{RGB: cv.rgb, Set: true})))
		}
		ch := cv.focus - 1
		opts := make([]string, 256)
		for i := range opts {
			opts[i] = fmt.Sprintf("%02X (%d)", i, i)
		}
		return true, a.Select(Select{
			Title: []string{"Red", "Green", "Blue"}[ch], Glyph: cv.c.Glyph, Options: opts, Selected: int(cv.rgb[ch]),
			Done: func(i int) tea.Cmd { cv.rgb[ch] = uint8(i); return nil },
		})
	case "Space":
	default:
		return false, nil
	}
	return true, nil
}

func (cv *colorView) surface() string { return cv.c.Title }

func (cv *colorView) keyRef() []refSection {
	return []refSection{{rows: []KeyRow{
		{Key: "Enter", Desc: "choose the channel's value; on new, choose the colour"},
		{Key: "Esc", Desc: "cancel"},
	}}, {title: "move", rows: []KeyRow{{Key: "h/l", Desc: "new, R, G, B"}}}}
}

func (cv *colorView) typing() bool { return false }
