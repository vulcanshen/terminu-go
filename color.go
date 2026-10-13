package terminu

import (
	"fmt"
	"strconv"
	"strings"
)

// RGB is a 24-bit colour.
type RGB [3]uint8

// Color is a colour or no colour (the terminal's own).
type Color struct {
	RGB
	Set bool
}

// Hex makes a Color from "#rrggbb".
func Hex(s string) Color {
	v, err := strconv.ParseUint(strings.TrimPrefix(s, "#"), 16, 32)
	if err != nil {
		panic("terminu: bad colour " + s)
	}
	return Color{RGB: RGB{uint8(v >> 16), uint8(v >> 8), uint8(v)}, Set: true}
}

func (c Color) String() string {
	if !c.Set {
		return "none"
	}
	return fmt.Sprintf("#%02x%02x%02x", c.R(), c.G(), c.B())
}

func (c Color) R() uint8 { return c.RGB[0] }
func (c Color) G() uint8 { return c.RGB[1] }
func (c Color) B() uint8 { return c.RGB[2] }

// The family colours, catppuccin-mocha (components/color). Each meaning has
// one colour (Principle P4); the comment is what tdp uses it for.
var (
	Base     = Hex("#1e1e2e") // text on blocks, the unfilled side of a capsule join, what dimming fades to
	Blue     = Hex("#89b4fa") // focus: frames, keys you can press now, the screen chips
	Surface2 = Hex("#585b70") // not usable: unfocused frames, disabled rows
	Lavender = Hex("#b4befe") // being edited, the user's footprint
	Sapphire = Hex("#74c7ec") // the last layer colour
	Green    = Hex("#a6e3a1") // in effect: the chosen value, live sessions, progress
	Yellow   = Hex("#f9e2af") // a mode
	Peach    = Hex("#fab387") // a warning
	Red      = Hex("#f38ba8") // an error
	Overlay0 = Hex("#6c7086") // quiet text: descriptions, hint words, separators
	Text     = Hex("#cdd6f4") // ordinary text
	Subtext1 = Hex("#bac2de") // the panel cursor block
	Surface1 = Hex("#45475a") // an unfocused button
)

// layerColors are the popup layer colours, lerp(Lavender, Sapphire, K/4)
// with each channel rounded half up (components/color).
var layerColors = [4]Color{Hex("#A4C0FA"), Hex("#94C3F5"), Hex("#84C5F1"), Hex("#74c7ec")}

// LayerColor is the colour of the popup at layer k, the bottom one being 1;
// layer 4 and up are Sapphire. A toast is always layer 1.
func LayerColor(k int) Color { return layerColors[min(max(k, 1), 4)-1] }

// Style is what a piece of text is drawn with. The zero Style draws the text
// as it is.
type Style struct {
	Fg, Bg    Color
	Bold      bool
	Underline bool
}

// Fg is a Style with a foreground only.
func Fg(c Color) Style { return Style{Fg: c} }

// On is text in Base, bold, on a block of c: a cursor, a capsule, a button.
func On(c Color) Style { return Style{Fg: Base, Bg: c, Bold: true} }

// Bolded returns s with bold on.
func (s Style) Bolded() Style { s.Bold = true; return s }

func (s Style) sgr() string {
	var p []string
	if s.Bold {
		p = append(p, "1")
	}
	if s.Underline {
		p = append(p, "4")
	}
	if s.Fg.Set {
		p = append(p, sgrRGB(true, s.Fg.RGB))
	}
	if s.Bg.Set {
		p = append(p, sgrRGB(false, s.Bg.RGB))
	}
	return strings.Join(p, ";")
}

// Render draws text in this style, resetting after it. text is one line.
func (s Style) Render(text string) string {
	p := s.sgr()
	if p == "" || text == "" {
		return text
	}
	return "\x1b[" + p + "m" + text + "\x1b[0m"
}

func sgrRGB(fg bool, c RGB) string {
	lead := "48"
	if fg {
		lead = "38"
	}
	return lead + ";2;" + strconv.Itoa(int(c[0])) + ";" + strconv.Itoa(int(c[1])) + ";" + strconv.Itoa(int(c[2]))
}

// --- dimming (Rules F8, T2; components/color) -------------------------------

// dimKeep is how much of a colour survives: dim(c) = c × 0.45 + base × 0.55.
const dimKeep = 0.45

// DimRGB fades c toward Base. It never lightens: a channel darker than Base
// keeps its own value.
func DimRGB(c RGB) RGB {
	var out RGB
	for i := range c {
		// Each channel rounds half up, then is compared with the original.
		v := float64(c[i])*dimKeep + float64(Base.RGB[i])*(1-dimKeep) + 0.5
		out[i] = min(c[i], uint8(v))
	}
	return out
}

// DimColor is DimRGB on a Color; no colour stays no colour.
func DimColor(c Color) Color {
	if !c.Set {
		return c
	}
	return Color{RGB: DimRGB(c.RGB), Set: true}
}

// palette16 is the terminal's 16 colours: xterm's until Probe asks the
// terminal (OSC 4).
var palette16 = xterm16

var xterm16 = [16]RGB{
	{0, 0, 0}, {205, 0, 0}, {0, 205, 0}, {205, 205, 0}, {0, 0, 238}, {205, 0, 205}, {0, 205, 205}, {229, 229, 229},
	{127, 127, 127}, {255, 0, 0}, {0, 255, 0}, {255, 255, 0}, {92, 92, 255}, {255, 0, 255}, {0, 255, 255}, {255, 255, 255},
}

// color256 is the RGB of a 256-colour index: 0–15 from the terminal's palette,
// the rest by the standard formula.
func color256(n int) RGB {
	switch {
	case n < 16:
		return palette16[max(n, 0)]
	case n < 232:
		n -= 16
		step := func(v int) uint8 {
			if v == 0 {
				return 0
			}
			return uint8(55 + v*40)
		}
		return RGB{step(n / 36), step(n / 6 % 6), step(n % 6)}
	default:
		v := uint8(8 + (min(n, 255)-232)*10)
		return RGB{v, v, v}
	}
}

// dimText is what text with no colour of its own becomes: dim(Text).
func dimText() string { return sgrRGB(true, DimRGB(Text.RGB)) }

// Dim redraws one drawn line dimmed: every foreground and background colour in
// its SGR sequences fades toward Base, always written as 24-bit, and text with
// no colour of its own gets dim(Text). Bold, reverse, cursor movement and the
// text itself are left as they are.
func Dim(line string) string {
	var b strings.Builder
	b.Grow(len(line) + len(line)/4 + 24)
	b.WriteString("\x1b[" + dimText() + "m")
	for i := 0; i < len(line); {
		if strings.HasPrefix(line[i:], "\x1b[") {
			end := i + 2
			for end < len(line) && (line[end] < 0x40 || line[end] > 0x7e) {
				end++
			}
			if end >= len(line) {
				b.WriteString(line[i:])
				break
			}
			if line[end] == 'm' {
				b.WriteString("\x1b[" + dimSGR(line[i+2:end]) + "m")
			} else {
				b.WriteString(line[i : end+1])
			}
			i = end + 1
			continue
		}
		b.WriteByte(line[i])
		i++
	}
	b.WriteString("\x1b[0m")
	return b.String()
}

// DimLines dims every line.
func DimLines(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = Dim(l)
	}
	return out
}

// dimSGR rewrites one SGR parameter list with its colours dimmed. After a
// reset or "default foreground" the dimmed default text colour follows, so
// what comes next stays dim.
func dimSGR(params string) string {
	ps := strings.Split(params, ";")
	var out []string
	needText := false
	for i := 0; i < len(ps); i++ {
		p := ps[i]
		n, err := strconv.Atoi(p)
		if p == "" {
			n, err = 0, nil
		}
		if err != nil {
			out = append(out, p)
			continue
		}
		switch {
		case n == 0:
			out = append(out, "0")
			needText = true
		case n == 39:
			needText = true
		case (n == 38 || n == 48) && i+1 < len(ps):
			fg := n == 38
			var c RGB
			switch ps[i+1] {
			case "2":
				if i+4 >= len(ps) {
					out = append(out, ps[i:]...)
					i = len(ps)
					continue
				}
				for k := range 3 {
					v, _ := strconv.Atoi(ps[i+2+k])
					c[k] = uint8(v)
				}
				i += 4
			case "5":
				if i+2 >= len(ps) {
					out = append(out, ps[i:]...)
					i = len(ps)
					continue
				}
				idx, _ := strconv.Atoi(ps[i+2])
				c = color256(idx)
				i += 2
			default:
				out = append(out, p)
				continue
			}
			out = append(out, sgrRGB(fg, DimRGB(c)))
			if fg {
				needText = false
			}
		case n >= 30 && n <= 37, n >= 90 && n <= 97:
			idx := n - 30
			if n >= 90 {
				idx = n - 90 + 8
			}
			out = append(out, sgrRGB(true, DimRGB(palette16[idx])))
			needText = false
		case n >= 40 && n <= 47, n >= 100 && n <= 107:
			idx := n - 40
			if n >= 100 {
				idx = n - 100 + 8
			}
			out = append(out, sgrRGB(false, DimRGB(palette16[idx])))
		default:
			out = append(out, p)
		}
	}
	if needText {
		out = append(out, dimText())
	}
	return strings.Join(out, ";")
}
