package terminu

import (
	"os"
	"strconv"
	"strings"
)

// EnvName is the name of an environment variable app reads, tdp E2:
// EnvName("filu", "icon width") is "FILU__ICON_WIDTH".
func EnvName(app, name string) string {
	up := func(s string) string {
		s = strings.ToUpper(strings.TrimSpace(s))
		return strings.Join(strings.FieldsFunc(s, func(r rune) bool {
			return !(r >= 'A' && r <= 'Z' || r >= '0' && r <= '9')
		}), "_")
	}
	return up(app) + "__" + up(name)
}

// The variables a family app sets for what runs in its PTY (tdp E2, E5).
const (
	FamilyIconWidth      = "TERMINU__ICON_WIDTH"
	FamilyPowerlineWidth = "TERMINU__POWERLINE_WIDTH"
)

// ProbeResult is what Probe found out about the terminal.
type ProbeResult struct {
	IconCells      int // cells a Nerd Font icon takes
	PowerlineCells int // cells a powerline symbol takes
	// IconFrom and PowerlineFrom say where each came from: the variable's
	// name, "probe", or "default".
	IconFrom, PowerlineFrom string
	Palette                 int // how many of the 16 colours the terminal reported (OSC 4)
}

// Probe finds out, before Bubble Tea takes the terminal, how many cells an icon
// and a powerline symbol take (tdp E5) and the terminal's 16 colours
// (components/color, dimming). Each width comes from the app's variable
// (<APP>__ICON_WIDTH, <APP>__POWERLINE_WIDTH), then the family's
// (TERMINU__ICON_WIDTH, TERMINU__POWERLINE_WIDTH), then asking the terminal
// where the cursor ended up after printing one (CPR, unix only). The palette
// is asked for with OSC 4 at the same time; colours the terminal does not
// report stay xterm's. Probe sets the lib's widths and palette and reports
// what it found.
func Probe(app string) ProbeResult {
	res := ProbeResult{IconCells: 1, PowerlineCells: 1, IconFrom: "default", PowerlineFrom: "default"}
	iconSet := widthFromEnv(&res.IconCells, &res.IconFrom, EnvName(app, "icon width"), FamilyIconWidth)
	plSet := widthFromEnv(&res.PowerlineCells, &res.PowerlineFrom, EnvName(app, "powerline width"), FamilyPowerlineWidth)
	ans, ok := askTerminal(!iconSet || !plSet)
	if ok {
		if !iconSet && ans.icon > 0 {
			res.IconCells, res.IconFrom = ans.icon, "probe"
		}
		if !plSet && ans.powerline > 0 {
			res.PowerlineCells, res.PowerlineFrom = ans.powerline, "probe"
		}
		res.Palette = ans.colours
	}
	SetIconWidth(res.IconCells, res.PowerlineCells)
	return res
}

// widthFromEnv takes the first of the variables that says 1 or 2.
func widthFromEnv(cells *int, from *string, names ...string) bool {
	for _, name := range names {
		if n, err := strconv.Atoi(os.Getenv(name)); err == nil && (n == 1 || n == 2) {
			*cells, *from = n, name
			return true
		}
	}
	return false
}

// answers is what the terminal said: cell widths (0 when not asked or no
// answer) and how many palette colours it reported.
type answers struct {
	icon, powerline, colours int
}

// parseAnswers reads the terminal's replies to the probe: two cursor position
// reports (after an icon, then after a powerline symbol, each printed from
// column 1), any OSC 4 colour reports, and the DA1 reply that ends it all.
// Colours it finds go into the palette.
func parseAnswers(b string) (ans answers, done bool) {
	cpr := 0
	for len(b) > 0 {
		i := strings.IndexByte(b, 0x1b)
		if i < 0 {
			break
		}
		b = b[i:]
		switch {
		case strings.HasPrefix(b, "\x1b]4;"):
			rest := b[len("\x1b]4;"):]
			end, term := strings.IndexByte(rest, '\x07'), 1
			if st := strings.Index(rest, "\x1b\\"); st >= 0 && (end < 0 || st < end) {
				end, term = st, 2
			}
			if end < 0 {
				return ans, false
			}
			if setPaletteReply(rest[:end]) {
				ans.colours++
			}
			b = rest[end+term:]
		case strings.HasPrefix(b, "\x1b[?"):
			end := strings.IndexByte(b, 'c')
			if end < 0 {
				return ans, false
			}
			return ans, true
		case strings.HasPrefix(b, "\x1b["):
			end := strings.IndexByte(b, 'R')
			if end < 0 {
				return ans, false
			}
			if semi := strings.IndexByte(b[:end], ';'); semi > 0 {
				if col, err := strconv.Atoi(b[semi+1 : end]); err == nil && col >= 2 {
					switch cpr {
					case 0:
						ans.icon = clampCells(col - 1)
					case 1:
						ans.powerline = clampCells(col - 1)
					}
				}
			}
			cpr++
			b = b[end+1:]
		default:
			b = b[1:]
		}
	}
	return ans, false
}

// setPaletteReply takes "n;rgb:RRRR/GGGG/BBBB" and stores colour n.
func setPaletteReply(body string) bool {
	n, spec, ok := strings.Cut(body, ";")
	idx, err := strconv.Atoi(n)
	if !ok || err != nil || idx < 0 || idx > 15 {
		return false
	}
	spec, ok = strings.CutPrefix(spec, "rgb:")
	if !ok {
		return false
	}
	parts := strings.Split(spec, "/")
	if len(parts) != 3 {
		return false
	}
	var c RGB
	for i, p := range parts {
		if len(p) == 0 || len(p) > 4 {
			return false
		}
		v, err := strconv.ParseUint(p, 16, 16)
		if err != nil {
			return false
		}
		// Keep the high 8 bits whatever the digit count: "ff" and "ffff" are both 255.
		bits := uint(len(p) * 4)
		if bits > 8 {
			v >>= bits - 8
		} else if bits < 8 {
			v = v * 255 / (1<<bits - 1)
		}
		c[i] = uint8(v)
	}
	palette16[idx] = c
	return true
}

// probeQueries is what Probe sends: an icon and a cursor position request, a
// powerline symbol and another, the 16 palette colours, then DA1, which every
// terminal answers, so no reply is waited for past it.
func probeQueries(widths bool) string {
	var b strings.Builder
	if widths {
		b.WriteString("\r" + iconProbe + "\x1b[6n\r" + powerlineProbe + "\x1b[6n")
	}
	for i := range 16 {
		b.WriteString("\x1b]4;" + strconv.Itoa(i) + ";?\x1b\\")
	}
	b.WriteString("\x1b[c")
	return b.String()
}

const (
	iconProbe      = "\U0000F07B" // nf-fa-folder, a representative icon
	powerlineProbe = "\U0000E0B0" // nf-pl-left_hard_divider
)
