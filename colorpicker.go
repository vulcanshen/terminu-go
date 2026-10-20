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
	// Subject is the one thing it acts on, kept as it is after ": " in the
	// title: "EDIT: filu" (components/layout/popup).
	Subject string
	// Value is the colour now; one not Set is no colour yet (the theme's
	// own): old shows none and no swatch, new starts from #000000, and Choose
	// always counts as a change (components/input/color-picker).
	Value Color
	// Done gets the new colour. It is not called when it did not change.
	Done func(Color) tea.Cmd
	// Commit, when set, writes a changed value back when Enter takes it, on
	// the update loop, before Done: an error keeps the popup open with the
	// value in it and opens an error popup over it, "RENAME FAILED"
	// (components/layout/popup, the action failing; Rules F4).
	Commit func(Color) error
	// OnCancel runs when it closes with no answer: Esc, or the popup it is
	// over closing it, so whoever waits for the answer (a web page's prompt,
	// ssh's askpass) hears it was cancelled (Rules F6).
	OnCancel func() tea.Cmd
}

func (c ColorPicker) typed() bool { return false }

// PickColor opens a color picker.
func (a *App) PickColor(c ColorPicker) tea.Cmd { return c.open(a, nil) }

type colorView struct {
	c     ColorPicker
	rgb   RGB
	focus int // 0 R, 1 G, 2 B, 3 Choose
	// back is the channel the focus was last on: Tab on Choose returns
	// there (components/input/color-picker).
	back    int
	written func()
}

func (c ColorPicker) open(a *App, written func()) tea.Cmd {
	return a.open(&colorView{c: c, rgb: c.Value.RGB, written: written})
}

// HexText writes a colour as the family does: "#2A2A3C".
func HexText(c Color) string { return strings.ToUpper(c.String()) }

// ColorSpans is a colour as a value on a form or a panel shows it, for
// Styled: a small swatch and its hex (components/input/color-picker).
func ColorSpans(c Color) []Span {
	return []Span{{Text: "███", Style: Fg(c)}, {Text: " "}, {Text: HexText(c), Style: Fg(Text)}}
}

// The row: old and new 8 cells wide, R, G, B 3 cells, two cells apart.
var colorCols = [5]int{8, 8, 3, 3, 3}

const colorRowW = 8 + 2 + 8 + 2 + 3 + 2 + 3 + 2 + 3

// The focus goes R, G, B, then the Choose button (components/input/color-picker).
const colorChoose = 3

// Tab names the area it goes to, as on a form (components/input/README).
var (
	colorChannelHint = []KeyHint{{"Enter", "edit"}, {"Tab", "actions"}, {"Esc", "cancel"}}
	colorChooseHint  = []KeyHint{{"Enter", "choose"}, {"Tab", "fields"}, {"Esc", "cancel"}}
)

func (cv *colorView) hint() []KeyHint {
	if cv.focus == colorChoose {
		return colorChooseHint
	}
	return colorChannelHint
}

const colorButton = " Choose "

func (cv *colorView) measure(sw, sh int) (int, int) {
	w := max(colorRowW+4, Width(titleCapsule(cv.c.Glyph, popupTitle(cv.c.Title, cv.c.Subject), Base, 1<<20))+4,
		max(hintWidth(colorChannelHint), hintWidth(colorChooseHint))+6)
	// The swatches, their names and hex, a blank row, the divider, the button.
	return min(w, max(min(sw-2, 120), 1)), min(7+4, max(sh-2, 3))
}

func (cv *colorView) draw(w, h int, layer Color) []string {
	inner := w - 2
	newC := Color{RGB: cv.rgb, Set: true}
	chans := [3]Color{
		{RGB: RGB{cv.rgb[0], 0, 0}, Set: true},
		{RGB: RGB{0, cv.rgb[1], 0}, Set: true},
		{RGB: RGB{0, 0, cv.rgb[2]}, Set: true},
	}
	join := func(parts ...string) string { return strings.Join(parts, "  ") }
	big := func(c Color) string {
		if !c.Set {
			return Repeat(" ", 8) // no colour: no swatch
		}
		return Fg(c).Render(Repeat("█", 8))
	}
	small := func(c Color) string { return Fg(c).Render(Repeat("█", 3)) }
	labels := []string{"old", "new", "R", "G", "B"}
	hexes := []string{HexText(cv.c.Value), HexText(newC)}
	if !cv.c.Value.Set {
		hexes[0] = "none" // the 8 cells take no app's words for nothing
	}
	for i := range 3 {
		hexes = append(hexes, fmt.Sprintf("%02X", cv.rgb[i]))
	}
	var labelRow, hexRow []string
	for i := range labels {
		label, hex := Fg(Overlay0).Render(Pad(labels[i], colorCols[i])), Fg(Text).Render(Pad(hexes[i], colorCols[i]))
		if i == 0 && !cv.c.Value.Set {
			hex = Fg(Overlay0).Render(Pad(hexes[i], colorCols[i]))
		}
		if i >= 2 && i-2 == cv.focus {
			// The channel with the focus: its name and hex as the cursor, as
			// wide as the swatch.
			label, hex = On(layer).Render(Pad(labels[i], colorCols[i])), On(layer).Render(Pad(hexes[i], colorCols[i]))
		}
		labelRow, hexRow = append(labelRow, label), append(hexRow, hex)
	}
	button := Style{Fg: Text, Bg: Surface1}.Render(colorButton)
	if cv.focus == colorChoose {
		button = On(layer).Render(colorButton)
	}
	body := []string{
		padRow(join(big(cv.c.Value), big(newC)), inner),
		padRow(join(big(cv.c.Value), big(newC), small(chans[0]), small(chans[1]), small(chans[2])), inner),
		padRow(join(labelRow...), inner),
		padRow(join(hexRow...), inner),
		Repeat(" ", inner),
		" " + Fg(Overlay0).Render(Repeat("─", inner-2)) + " ",
		PadLeft(button, inner-1) + " ",
	}
	return popupBox{color: layer, glyph: cv.c.Glyph, title: popupTitle(cv.c.Title, cv.c.Subject), hint: cv.hint()}.draw(w, Pad2(body, inner, h-4))
}

func (cv *colorView) key(a *App, key string, _ tea.KeyMsg) (bool, tea.Cmd) {
	defer func() {
		if cv.focus != colorChoose {
			cv.back = cv.focus
		}
	}()
	switch key {
	// hjkl move inside an area, as on a form: among R, G and B, round at the
	// ends (Rules K12); only Tab reaches Choose.
	case "h", "k", "←", "↑":
		if cv.focus != colorChoose {
			cv.focus = (cv.focus + 2) % 3
		}
	case "l", "j", "→", "↓":
		if cv.focus != colorChoose {
			cv.focus = (cv.focus + 1) % 3
		}
	case "Tab", "Shift-Tab":
		// The channels and the action area, as a form's two areas (Rules K2):
		// either way is the other one.
		if cv.focus == colorChoose {
			cv.focus = cv.back
		} else {
			cv.focus = colorChoose
		}
	case "Enter":
		if cv.focus == colorChoose {
			// From no colour to one is a change, #000000 too.
			if cv.c.Value.Set && cv.rgb == cv.c.Value.RGB || cv.c.Done == nil && cv.c.Commit == nil {
				return true, a.answer()
			}
			v := Color{RGB: cv.rgb, Set: true}
			if err := runCommit(cv.c.Commit, v); err != nil {
				return true, a.commitFailed(cv.c.Title, cv.c.Subject, err)
			}
			return true, tea.Batch(a.answer(), then(cv.written, runDone(cv.c.Done, v)))
		}
		ch := cv.focus
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

func (cv *colorView) cancel() tea.Cmd { return run(cv.c.OnCancel) }

func (cv *colorView) keyRef() []refSection {
	return []refSection{{rows: []KeyRow{
		{Key: "Enter", Desc: "on R, G or B choose its value; on Choose, choose the colour"},
		{Key: "Tab|Shift-Tab", Desc: "between R, G, B and Choose, back to the channel left"},
		{Key: "Esc", Desc: "cancel"},
	}}, {title: "move", rows: []KeyRow{{Key: "h|l", Desc: "R, G, B"}}}}
}

func (cv *colorView) typing() bool { return false }
