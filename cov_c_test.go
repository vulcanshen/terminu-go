package terminu

import (
	"bytes"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Tests for components/color, components §圖與間距 and §大寫, layout/screen
// and layout/panel (docs/coverage.md).

// --- reading a drawn line back --------------------------------------------------

// covCStyle is what one cell is drawn with, read back from the SGR codes.
type covCStyle struct {
	Fg, Bg Color
	Bold   bool
}

// covCCell is one character on the screen: where it is and how it is drawn.
type covCCell struct {
	Ch  string
	Col int
	St  covCStyle
}

// covCCells reads one drawn line cell by cell.
func covCCells(line string) []covCCell {
	var out []covCCell
	var st covCStyle
	col := 0
	walk(line, func(seq string, cells int) {
		if cells == 0 {
			if strings.HasPrefix(seq, "\x1b[") && strings.HasSuffix(seq, "m") {
				covCApply(&st, seq[2:len(seq)-1])
			}
			return
		}
		out = append(out, covCCell{Ch: seq, Col: col, St: st})
		col += cells
	})
	return out
}

// covCApply applies one SGR parameter list to st.
func covCApply(st *covCStyle, params string) {
	ps := strings.Split(params, ";")
	for i := 0; i < len(ps); i++ {
		switch ps[i] {
		case "", "0":
			*st = covCStyle{}
		case "1":
			st.Bold = true
		case "22":
			st.Bold = false
		case "39":
			st.Fg = Color{}
		case "49":
			st.Bg = Color{}
		case "38", "48":
			if i+4 >= len(ps) || ps[i+1] != "2" {
				continue
			}
			var c RGB
			for k := range 3 {
				v, _ := strconv.Atoi(ps[i+2+k])
				c[k] = uint8(v)
			}
			if ps[i] == "38" {
				st.Fg = Color{RGB: c, Set: true}
			} else {
				st.Bg = Color{RGB: c, Set: true}
			}
			i += 4
		}
	}
}

// covCSpan is the cells where text is first drawn on line; nil when it is not.
func covCSpan(line, text string) []covCCell {
	cells := covCCells(line)
	var b strings.Builder
	starts := make([]int, len(cells))
	for i, c := range cells {
		starts[i] = b.Len()
		b.WriteString(c.Ch)
	}
	at := strings.Index(b.String(), text)
	if at < 0 {
		return nil
	}
	var out []covCCell
	for i, c := range cells {
		if starts[i] >= at && starts[i] < at+len(text) {
			out = append(out, c)
		}
	}
	return out
}

// covCAfter is the cell drawn right after where text is first drawn.
func covCAfter(line, text string) covCCell {
	s := covCSpan(line, text)
	if s == nil {
		return covCCell{}
	}
	last := s[len(s)-1]
	return covCAt(line, last.Col+Width(last.Ch))
}

// covCAt is the cell covering column col.
func covCAt(line string, col int) covCCell {
	for _, c := range covCCells(line) {
		if col >= c.Col && col < c.Col+Width(c.Ch) {
			return c
		}
	}
	return covCCell{}
}

// covCFind is the first row drawing text, and its cells there.
func covCFind(t *testing.T, lines []string, text string) (int, []covCCell) {
	t.Helper()
	for i, l := range lines {
		if s := covCSpan(l, text); s != nil {
			return i, s
		}
	}
	t.Fatalf("%q is not on the screen:\n%s", text, StripStyles(strings.Join(lines, "\n")))
	return -1, nil
}

// covCStyleOf is the style text starts with where it is first drawn.
func covCStyleOf(t *testing.T, lines []string, text string) covCStyle {
	t.Helper()
	_, s := covCFind(t, lines, text)
	return s[0].St
}

// covCWant checks the colours text is drawn in; an unset bg means no
// background at all.
func covCWant(t *testing.T, what string, lines []string, text string, fg, bg Color) {
	t.Helper()
	if st := covCStyleOf(t, lines, text); st.Fg != fg || st.Bg != bg {
		t.Errorf("%s: %q is drawn %v on %v, want %v on %v", what, text, st.Fg, st.Bg, fg, bg)
	}
}

// covCCrop is the columns [x0, x1) of every line, styles kept.
func covCCrop(lines []string, x0, x1 int) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = Cut(l, x0, x1)
	}
	return out
}

// covCPlain is one line without its styles.
func covCPlain(line string) string { return StripStyles(line) }

// covCFooter is the footer's text.
func covCFooter(ta *testApp) string {
	lines := ta.Frame()
	return strings.TrimSpace(StripStyles(lines[len(lines)-1]))
}

// covCStates open the screens the colour tests look at.
func covCStates() map[string]func(*testApp) {
	return map[string]func(*testApp){
		"panels":    func(*testApp) {},
		"space":     func(ta *testApp) { ta.press("Space") },
		"global":    func(ta *testApp) { ta.press("Space", "G", "Enter") },
		"keys":      func(ta *testApp) { ta.press("?") },
		"confirm":   func(ta *testApp) { ta.press("x") },
		"mode":      func(ta *testApp) { ta.press("D") },
		"mode keys": func(ta *testApp) { ta.press("D", "?") },
		"filter":    func(ta *testApp) { ta.press("/", "1") },
		"values":    func(ta *testApp) { ta.press("3") },
		"marked":    func(ta *testApp) { ta.press("m", "j") },
		"loading":   func(ta *testApp) { ta.pods.Loading = true },
		"toast":     func(ta *testApp) { ta.Toast(Warning, "Disk almost full") },
		"error":     func(ta *testApp) { ta.Error("Save failed", "hosts.yaml changed on disk") },
		"form":      func(ta *testApp) { ta.Form(testForm(nil)) },
		"text": func(ta *testApp) {
			ta.Text(TextInput{Title: "Rename", Prompt: "New name"})
			ta.settle()
			ta.press("a", "b")
		},
		"viewer": func(ta *testApp) {
			ta.Viewer(&Viewer{Title: "DevTools", Tabs: []string{"Console", "Network"}, Lines: []string{"x"}, Loading: true})
		},
		"narrow": func(ta *testApp) { ta.resize(50, 20) },
	}
}

// covCOpen is the test app in one of covCStates, its animations played.
func covCOpen(t *testing.T, state string) *testApp {
	t.Helper()
	ta := newTestApp(t)
	covCStates()[state](ta)
	ta.settle()
	return ta
}

// covCClose checks a dimmed channel against dim(c) = c × 0.45 + base × 0.55,
// never brighter than c; the rounding is not given, so within one.
func covCClose(t *testing.T, what string, got, src RGB) {
	t.Helper()
	for i := range 3 {
		exact := float64(src[i])*0.45 + float64(Base.RGB[i])*0.55
		switch {
		case exact > float64(src[i]):
			if got[i] != src[i] {
				t.Errorf("%s: channel %d of %v dims to %d, want it kept at %d (never brighter)", what, i, src, got[i], src[i])
			}
		case math.Abs(float64(got[i])-exact) >= 1:
			t.Errorf("%s: channel %d of %v dims to %d, want %.2f", what, i, src, got[i], exact)
		}
	}
}

// covCFg is the foreground the first character of a dimmed line gets.
func covCFg(t *testing.T, line string) RGB {
	t.Helper()
	cells := covCCells(Dim(line))
	if len(cells) == 0 || !cells[0].St.Fg.Set {
		t.Fatalf("%q dimmed has no colour", line)
	}
	return cells[0].St.Fg.RGB
}

// --- color -----------------------------------------------------------------------

func TestColorTokensOneMeaning(t *testing.T) {
	tokens := map[string]Color{
		"Base": Base, "Blue": Blue, "Surface2": Surface2, "Lavender": Lavender, "Green": Green, "Yellow": Yellow,
		"Peach": Peach, "Red": Red, "Overlay0": Overlay0, "Text": Text, "Subtext1": Subtext1, "Surface1": Surface1,
	}
	for k := 1; k <= 4; k++ {
		tokens[fmt.Sprintf("layer %d", k)] = LayerColor(k)
	}
	byColor := map[Color]string{}
	for name, c := range tokens {
		if other, ok := byColor[c]; ok {
			t.Errorf("%s and %s are both %v: one colour, one meaning", name, other, c)
		}
		byColor[c] = name
	}
	// The interface draws only with the table's colours, as they are or
	// dimmed (Principle P4); the test app has no content colours of its own.
	allowed := map[Color]bool{}
	for c := range byColor {
		for range 4 {
			allowed[c] = true
			c = DimColor(c)
		}
	}
	for state := range covCStates() {
		ta := covCOpen(t, state)
	rows:
		for i, l := range ta.Frame() {
			for _, c := range covCCells(l) {
				for _, col := range []Color{c.St.Fg, c.St.Bg} {
					if col.Set && !allowed[col] {
						t.Errorf("%s: row %d draws %q in %v, not a colour of the table", state, i, c.Ch, col)
						break rows
					}
				}
			}
		}
	}
}

func TestLayerColors(t *testing.T) {
	table := []Color{Hex("#A4C0FA"), Hex("#94C3F5"), Hex("#84C5F1"), Hex("#74c7ec")}
	for k := 1; k <= 6; k++ {
		got, want := LayerColor(k), table[min(k, 4)-1]
		if got != want {
			t.Errorf("layer %d is %v, want %v", k, got, want)
		}
		// lerp(Lavender, Sapphire, K/4) rounded half up, four layers and up
		// the end colour.
		f := float64(min(k, 4)) / 4
		for i := range 3 {
			exact := float64(Lavender.RGB[i]) + (float64(Sapphire.RGB[i])-float64(Lavender.RGB[i]))*f
			if float64(got.RGB[i]) != math.Floor(exact+0.5) {
				t.Errorf("layer %d channel %d is %d, the lerp gives %.2f", k, i, got.RGB[i], exact)
			}
		}
	}
	// On the screen: each popup opened on top has the frame and title of its layer.
	ta := newTestApp(t)
	top := func(k int) {
		t.Helper()
		l := ta.top()
		row := ta.Frame()[l.y]
		want := LayerColor(k)
		if c := covCAt(row, l.x); c.Ch != "┏" || c.St.Fg != want {
			t.Errorf("layer %d: frame %q in %v, want %v", k, c.Ch, c.St.Fg, want)
		}
		if c := covCAt(row, l.x+3); c.St.Bg != want {
			t.Errorf("layer %d: title on %v, want %v", k, c.St.Bg, want)
		}
	}
	ta.press("Space")
	top(1)
	ta.press("G", "Enter")
	top(2)
	ta.press("q")
	top(3)
	ta.press("?")
	top(4)
	ta.Confirm(Ask{Title: "More", Question: "One more?", Verb: "go"}, nil)
	ta.settle()
	top(5)
	// A toast is not a layer: always the first layer's colour.
	ta.Toast(Info, "copied")
	ta.settle()
	if c := covCAt(ta.Frame()[ta.toast.y], ta.toast.x); c.St.Fg != LayerColor(1) {
		t.Errorf("a toast over five popups is framed in %v, want layer 1 %v", c.St.Fg, LayerColor(1))
	}
}

func TestWarningColorFixed(t *testing.T) {
	ask := Ask{Title: "Wipe", Question: "Wipe it?", Verb: "wipe", Warn: []int{0},
		Details: []string{"Everything on the disk goes, and nothing can bring any of it back again"}}
	ta := newTestApp(t)
	ta.Confirm(ask, nil)
	ta.settle()
	covCWant(t, "a warning on layer 1", ta.Frame(), "Everything", Peach, Color{})
	ta.press("Esc", "Space")
	ta.Confirm(ask, nil)
	ta.settle()
	covCWant(t, "a warning on layer 2", ta.Frame(), "Everything", Peach, Color{})
	// Covered by another popup it is dimmed with the rest (Rules F8).
	ta.Error("Failed", "boom")
	ta.settle()
	lines := ta.Frame()
	covCWant(t, "a warning under a popup", lines, "Everything", DimColor(Peach), Color{})
	covCWant(t, "an error on layer 3", lines, "boom", Red, Color{})
}

func TestBoldOnlyWhereAllowed(t *testing.T) {
	loading := map[string]bool{}
	for _, g := range loadingFrames {
		loading[g] = true
	}
	for state := range covCStates() {
		ta := covCOpen(t, state)
	rows:
		for i, l := range ta.Frame() {
			for _, c := range covCCells(l) {
				if !c.St.Bold || c.Ch == " " {
					continue
				}
				// On a block (a cursor, a capsule, a popup title), a mode name,
				// a loading icon.
				mode := c.St.Fg == Yellow || c.St.Fg == DimColor(Yellow)
				if c.St.Bg.Set || mode || loading[c.Ch] {
					continue
				}
				t.Errorf("%s: row %d has %q in bold", state, i, c.Ch)
				break rows
			}
		}
	}
	// Where it is allowed, it is there.
	ta := newTestApp(t)
	ta.press("D")
	lines := ta.Frame()
	for _, s := range []string{"pod-00", "[1] PODS", "Drag"} {
		if !covCStyleOf(t, lines, s).Bold {
			t.Errorf("%q is not bold", s)
		}
	}
	ta.press("Esc", "x")
	if !covCStyleOf(t, ta.Frame(), "DELETE").Bold {
		t.Error("the popup title is not bold")
	}
}

func TestColorTable(t *testing.T) {
	none := Color{}
	ta := newTestApp(t)
	ta.Progress = func() (float64, bool) { return 0.5, true }
	lines := ta.Frame()
	cell := func(what string, c covCCell, ch string, fg Color) {
		t.Helper()
		if c.Ch != ch || c.St.Fg != fg || c.St.Bg.Set {
			t.Errorf("%s: %q in %v on %v, want %q in %v", what, c.Ch, c.St.Fg, c.St.Bg, ch, fg)
		}
	}
	cell("focused frame", covCAt(lines[2], 0), "┏", Blue)
	cell("unfocused frame", covCAt(lines[2], 40), "┌", Surface2)
	cell("progress", covCAt(lines[1], 0), "━", Green)
	cell("divider", covCAt(lines[1], 119), "━", Overlay0)
	footer := lines[len(lines)-1:]
	covCWant(t, "a key you can press", footer, "Space", Blue, none)
	covCWant(t, "hint words", footer, ":menu", Overlay0, none)
	pods := covCCrop(lines, 0, 40)
	covCWant(t, "header of the focused panel", pods, "NAME", Blue, none)
	covCWant(t, "panel cursor", pods, "pod-00", Base, Subtext1)
	covCWant(t, "screen chip", lines[:1], "[P]ANELS", Base, Blue)

	ta.press("m", "j")
	covCWant(t, "mark", covCCrop(ta.Frame(), 0, 40), glyphMarked, Green, none)

	ta.press("3")
	settings := covCCrop(ta.Frame(), 80, 120)
	covCWant(t, "label", settings, "NAME", Blue, none)
	covCWant(t, "disabled label", settings, "REMOTE", Surface2, none)
	covCWant(t, "value in effect", settings, glyphRadioOn+" row", Green, none)
	covCWant(t, "a value", settings, "column", Text, none)

	ta.press("1", "D")
	covCWant(t, "mode", ta.Frame(), "Drag", Yellow, none)

	ta.press("Esc", "Space")
	l := ta.top()
	lines = ta.Frame()
	cell("popup frame", covCAt(lines[l.y], l.x), "┏", LayerColor(1))
	covCWant(t, "menu label", lines, "Refresh", Text, none)
	covCWant(t, "menu description", lines, "the list", Overlay0, none)
	covCWant(t, "section title", lines, "ITEM OPERATION", Overlay0, none)
	covCWant(t, "popup cursor", lines, "[Enter] Logs", Base, LayerColor(1))

	ta.press("Esc", "x")
	lines = ta.Frame()
	covCWant(t, "warning", lines, "It is gone", Peach, none)
	covCWant(t, "question", lines, "Delete it?", Text, none)

	ta.press("Esc")
	ta.Form(testForm(nil))
	ta.settle()
	covCWant(t, "button without focus", ta.Frame(), "Save", Text, Surface1)

	ta.press("Esc")
	ta.Text(TextInput{Title: "Rename", Prompt: "New name"})
	ta.settle()
	ta.press("a", "b", "c")
	covCWant(t, "being edited", ta.Frame(), "abc", Lavender, none)

	ta.press("Esc")
	ta.list.SetRows(nil)
	ta.pods.Fail = &Failure{Fact: "Forbidden", Hint: "Press [R] to retry"}
	covCWant(t, "error", ta.Frame(), "Forbidden", Red, none)
}

func TestNoBackgroundPainted(t *testing.T) {
	powerline := func(ch string) bool {
		r := []rune(ch)
		return len(r) == 1 && isPowerline(r[0])
	}
	for state := range covCStates() {
		ta := covCOpen(t, state)
	rows:
		for i, l := range ta.Frame() {
			for _, c := range covCCells(l) {
				switch {
				case c.St.Bg == Base && !powerline(c.Ch):
					t.Errorf("%s: row %d paints Base under %q", state, i, c.Ch)
					break rows
				case c.St.Fg == Base && !c.St.Bg.Set && !powerline(c.Ch) && c.Ch != " ":
					t.Errorf("%s: row %d writes %q in Base off a block", state, i, c.Ch)
					break rows
				}
			}
		}
	}
	// Cells with no background of their own are left to the terminal.
	ta := newTestApp(t)
	lines := ta.Frame()
	for _, at := range [][2]int{{0, 119}, {20, 20}, {20, 60}, {20, 100}, {39, 119}} {
		if c := covCAt(lines[at[0]], at[1]); c.Ch != " " || c.St.Bg.Set {
			t.Errorf("row %d column %d: %q on %v, want a blank with no background", at[0], at[1], c.Ch, c.St.Bg)
		}
	}
}

func TestCursorBlock(t *testing.T) {
	ta := newTestApp(t)
	pods := func() []string { return covCCrop(ta.Frame(), 0, 40) }
	if st := covCStyleOf(t, pods(), "pod-00"); st != (covCStyle{Fg: Base, Bg: Subtext1, Bold: true}) {
		t.Errorf("panel cursor: %+v", st)
	}
	// Text on the block turns Base, whatever its own colour: the Green mark too.
	ta.press("m")
	if st := covCStyleOf(t, pods(), glyphMarked); st.Fg != Base || st.Bg != Subtext1 {
		t.Errorf("a mark under the cursor: %v on %v", st.Fg, st.Bg)
	}
	// In a popup the block is the layer colour.
	ta.press("Space")
	if st := covCStyleOf(t, ta.Frame(), "[Enter] Logs"); st != (covCStyle{Fg: Base, Bg: LayerColor(1), Bold: true}) {
		t.Errorf("popup cursor: %+v", st)
	}
	// The block without the focus has its cursor dimmed once more (P6): the
	// list under a panel filter's typing row...
	ta.press("Esc", "/")
	if st := covCStyleOf(t, pods(), "pod-00"); st.Bg != DimColor(Subtext1) || st.Fg != Base {
		t.Errorf("the list's cursor while typing: %v on %v", st.Fg, st.Bg)
	}
	// ...and a finder's list while its typing row has the focus.
	ta.press("Esc")
	ta.Find(testFinder())
	ta.settle()
	l := ta.top()
	list := covCCrop(ta.Frame(), l.x, l.x+l.w/2) // the list's half, not the preview's
	if st := covCStyleOf(t, list, "README.md"); st.Bg != DimColor(LayerColor(1)) || st.Fg != Base {
		t.Errorf("the finder's cursor while typing: %v on %v", st.Fg, st.Bg)
	}
	ta.press("Tab")
	list = covCCrop(ta.Frame(), l.x, l.x+l.w/2)
	if st := covCStyleOf(t, list, "README.md"); st.Bg != LayerColor(1) || st.Fg != Base {
		t.Errorf("the finder's cursor on the list: %v on %v", st.Fg, st.Bg)
	}
}

func TestDimFormula(t *testing.T) {
	for _, c := range []RGB{{137, 180, 250}, {243, 139, 168}, {205, 214, 244}, {255, 255, 255}, {100, 100, 100}} {
		line := fmt.Sprintf("\x1b[38;2;%d;%d;%d;48;2;%d;%d;%dmx", c[0], c[1], c[2], c[0], c[1], c[2])
		cells := covCCells(Dim(line))
		covCClose(t, "foreground", cells[0].St.Fg.RGB, c)
		covCClose(t, "background", cells[0].St.Bg.RGB, c)
	}
	// 256 colours from 16 on: the standard values.
	for n, rgb := range map[int]RGB{
		16: {0, 0, 0}, 21: {0, 0, 255}, 102: {135, 135, 135}, 196: {255, 0, 0}, 231: {255, 255, 255},
		232: {8, 8, 8}, 244: {128, 128, 128}, 255: {238, 238, 238},
	} {
		covCClose(t, fmt.Sprintf("38;5;%d", n), covCFg(t, fmt.Sprintf("\x1b[38;5;%dmx", n)), rgb)
		bg := covCCells(Dim(fmt.Sprintf("\x1b[48;5;%dmx", n)))[0].St.Bg
		covCClose(t, fmt.Sprintf("48;5;%d", n), bg.RGB, rgb)
	}
}

func TestDimPaletteFromOSC4(t *testing.T) {
	old := palette16
	t.Cleanup(func() { palette16 = old })
	palette16 = xterm16
	// The terminal says its blue (colour 4) is #89b4fa.
	if _, done := parseAnswers("\x1b]4;4;rgb:8989/b4b4/fafa\x1b\\\x1b[?62;22c"); !done {
		t.Fatal("the reply was not read to its end")
	}
	blue := RGB{0x89, 0xb4, 0xfa}
	covCClose(t, "34", covCFg(t, "\x1b[34mx"), blue)
	covCClose(t, "38;5;4", covCFg(t, "\x1b[38;5;4mx"), blue)
	covCClose(t, "44", covCCells(Dim("\x1b[44mx"))[0].St.Bg.RGB, blue)
	// A colour it did not report stays xterm's.
	covCClose(t, "94", covCFg(t, "\x1b[94mx"), xterm16[12])
}

func TestDimPaletteFallbackXterm(t *testing.T) {
	old := palette16
	t.Cleanup(func() { palette16 = old })
	palette16 = xterm16
	// A terminal that answers DA1 and only colour 1: the rest are xterm's.
	ans, done := parseAnswers("\x1b]4;1;rgb:f3/8b/a8\x07\x1b[?62;22c")
	if !done || ans.colours != 1 {
		t.Fatalf("answers %+v, done %v", ans, done)
	}
	covCClose(t, "34", covCFg(t, "\x1b[34mx"), RGB{0, 0, 238})
	covCClose(t, "37", covCFg(t, "\x1b[37mx"), RGB{229, 229, 229})
	covCClose(t, "31", covCFg(t, "\x1b[31mx"), RGB{0xf3, 0x8b, 0xa8})
	// One that answers nothing at all.
	palette16 = xterm16
	if _, done := parseAnswers(""); done {
		t.Error("no reply counted as an answer")
	}
	covCClose(t, "33", covCFg(t, "\x1b[33mx"), RGB{205, 205, 0})
}

func TestDimNeverBrighter(t *testing.T) {
	for _, c := range []RGB{{0, 0, 0}, {10, 200, 40}, {30, 30, 46}, {0, 0, 238}} {
		got := covCFg(t, fmt.Sprintf("\x1b[38;2;%d;%d;%dmx", c[0], c[1], c[2]))
		covCClose(t, "foreground", got, c)
	}
	for r := 0; r < 256; r += 17 {
		for g := 0; g < 256; g += 51 {
			for b := 0; b < 256; b += 85 {
				c := RGB{uint8(r), uint8(g), uint8(b)}
				cells := covCCells(Dim(fmt.Sprintf("\x1b[38;2;%d;%d;%d;48;2;%d;%d;%dmx", r, g, b, r, g, b)))
				for i := range 3 {
					if cells[0].St.Fg.RGB[i] > c[i] || cells[0].St.Bg.RGB[i] > c[i] {
						t.Fatalf("%v got brighter: %v on %v", c, cells[0].St.Fg, cells[0].St.Bg)
					}
				}
			}
		}
	}
}

func TestDimTruecolorOut(t *testing.T) {
	in := "\x1b[31ma\x1b[91mb\x1b[38;5;33mc\x1b[44md\x1b[103me\x1b[48;5;200mf\x1b[38;2;1;2;3mg\x1b[1;7mh\x1b[2Ci\x1b[22;27mj"
	out := Dim(in)
	if StripStyles(out) != StripStyles(in) {
		t.Errorf("the text changed: %q", StripStyles(out))
	}
	var params []string
	for _, seq := range strings.Split(out, "\x1b[")[1:] {
		if end := strings.IndexAny(seq, "mC"); end >= 0 && seq[end] == 'm' {
			params = append(params, strings.Split(seq[:end], ";")...)
		}
	}
	for i := 0; i < len(params); i++ {
		p := params[i]
		if p == "38" || p == "48" {
			if i+1 >= len(params) || params[i+1] != "2" {
				t.Errorf("a colour not in 24-bit: %s;%s", p, params[min(i+1, len(params)-1)])
			}
			i += 4
			continue
		}
		if n, err := strconv.Atoi(p); err == nil && (n >= 30 && n <= 37 || n >= 40 && n <= 47 || n >= 90 && n <= 97 || n >= 100 && n <= 107) {
			t.Errorf("a 16-colour code left in: %d", n)
		}
	}
	// Bold, reverse and the cursor's movement are left as they are.
	for _, want := range []string{"\x1b[2C", "1", "7", "22", "27"} {
		if want[0] == '\x1b' {
			if !strings.Contains(out, want) {
				t.Errorf("the cursor movement %q is gone", want)
			}
			continue
		}
		found := false
		for _, p := range params {
			found = found || p == want
		}
		if !found {
			t.Errorf("SGR %s is gone: %q", want, out)
		}
	}
}

func TestDimDefaultFg(t *testing.T) {
	for _, c := range covCCells(Dim("plain")) {
		if c.St.Fg != DimColor(Text) {
			t.Errorf("%q with no colour dims to %v, want dim(Text) %v", c.Ch, c.St.Fg, DimColor(Text))
		}
	}
	for _, line := range []string{"\x1b[38;2;243;139;168mred\x1b[0mafter", "\x1b[38;2;243;139;168mred\x1b[39mafter"} {
		if st := covCStyleOf(t, []string{Dim(line)}, "after"); st.Fg != DimColor(Text) {
			t.Errorf("%q: text after the colour ends is %v, want dim(Text)", line, st.Fg)
		}
	}
}

// --- components §圖與間距、§大寫 ------------------------------------------------------

func TestTwoCellColumnGap(t *testing.T) {
	// Table columns.
	l := &List{Columns: []Column{{Title: "Name"}, {Title: "Kind"}, {Title: "Size", Number: true}}}
	l.SetRows([]Row{{Cells: []string{"ab", "x", "7"}}, {Cells: []string{"abcdef", "pod", "12"}}})
	text := StripStyles(strings.Join(l.Draw(40, 4, false), "\n"))
	for _, s := range []string{"NAME    KIND  SIZE", "abcdef  pod     12"} {
		if !strings.Contains(text, s) {
			t.Errorf("table: %q missing in\n%s", s, text)
		}
	}
	// A label and its value.
	ta := newTestApp(t)
	settings := StripStyles(strings.Join(covCCrop(ta.Frame(), 80, 120), "\n"))
	for _, s := range []string{"NAME    web", "LAYOUT  " + glyphRadioOn, "REMOTE  x"} {
		if !strings.Contains(settings, s) {
			t.Errorf("values: %q missing in\n%s", s, settings)
		}
	}
	// A label and its description, at the widest row of a menu.
	ta.press("Space")
	if s := "Global operation  actions for the whole app"; !strings.Contains(ta.screenText(), s) {
		t.Errorf("menu: %q missing", s)
	}
	// A key and what it does.
	ta.press("Esc", "?")
	if s := "q|Ctrl-C  quit"; !strings.Contains(ta.screenText(), s) {
		t.Errorf("key reference: %q missing", s)
	}
}

func TestGlyphOneCellGap(t *testing.T) {
	for _, icon := range []int{1, 2} {
		t.Run(fmt.Sprintf("icon %d", icon), func(t *testing.T) {
			withWidths(t, icon, 1)
			gap := func(lines []string, glyph, word string) {
				t.Helper()
				_, s := covCFind(t, lines, glyph+" "+word)
				if s[2].Col != s[0].Col+icon+1 {
					t.Errorf("%q starts at %d, the glyph at %d: want one cell between", word, s[2].Col, s[0].Col)
				}
			}
			ta := newTestApp(t)
			ta.press("m", "j")
			lines := ta.Frame()
			gap(lines, glyphMarked, "pod-00")
			gap(lines, glyphRadioOn, "row")
			gap(lines, glyphRadioOff, "column")
			ta.press("Space")
			lines = ta.Frame()
			gap(lines, glyphHexagon, "ITEM OPERATION")
			gap(lines, glyphMenu, "[1] PODS")
			ta.press("Esc", "/", "1")
			gap(ta.Frame(), glyphTyping, "1")
		})
	}
}

func TestUppercaseWhereRequired(t *testing.T) {
	ta := newTestApp(t)
	text := ta.screenText()
	for _, s := range []string{"[1] PODS", "EVENTS", "[2] LOG", "[3] SETTINGS", "[P]ANELS", "[O]THER", "NAME", "AGE", "LAYOUT", "REMOTE"} {
		if !strings.Contains(text, s) {
			t.Errorf("%q is not on the screen in capitals", s)
		}
	}
	for _, s := range []string{"Pods", "Events", "Settings", "Panels", "Layout", "Remote"} {
		if strings.Contains(text, s) {
			t.Errorf("%q is not in capitals", s)
		}
	}
	// Popup titles and the section titles of menus and key references.
	ta.press("Space")
	l := ta.top()
	if title := covCPlain(ta.Frame()[l.y]); !strings.Contains(title, "[1] PODS") {
		t.Errorf("Space menu title: %q", title)
	}
	for _, s := range []string{"ITEM OPERATION", "PANEL OPERATION"} {
		if !strings.Contains(ta.screenText(), s) {
			t.Errorf("menu section title %q missing", s)
		}
	}
	ta.press("Esc", "?")
	for _, s := range []string{"ITEM OPERATION", "MOVE", "CORE KEYS", "APP-WIDE"} {
		if !strings.Contains(ta.screenText(), s) {
			t.Errorf("key reference section title %q missing", s)
		}
	}
	ta.press("Esc", "x")
	if title := covCPlain(ta.Frame()[ta.top().y]); !strings.Contains(title, "DELETE") {
		t.Errorf("confirm title: %q", title)
	}
}

func TestUppercaseKeepsHotkeyAndNames(t *testing.T) {
	ta := newTestApp(t)
	ta.AddScreen(&Screen{Name: "terminal", Key: "t", Panels: []*Panel{{Label: "Shell", Body: &Pager{}}}})
	ta.AddScreen(&Screen{Name: "Manage", Key: "M", Panels: []*Panel{{Label: "Hosts", Body: &Pager{}}}})
	chips := covCPlain(ta.Frame()[0])
	for _, s := range []string{"[t]ERMINAL", "[M]ANAGE"} {
		if !strings.Contains(chips, s) {
			t.Errorf("chip %q missing: %q", s, chips)
		}
	}
	// Names and paths in a title are content.
	ta.Viewer(&Viewer{Title: "Yaml", Subject: "pod/nginx-7d4f", Lines: []string{"kind: Pod"}})
	ta.settle()
	if title := covCPlain(ta.Frame()[ta.top().y]); !strings.Contains(title, "YAML — pod/nginx-7d4f") {
		t.Errorf("viewer title: %q", title)
	}
	ta.press("Esc")
	term := &Terminal{Title: "Alterm", Subject: "myhost", in: &bytes.Buffer{}, started: true}
	ta.open(&termView{t: term, a: ta.App})
	ta.settle()
	if title := covCPlain(ta.Frame()[ta.top().y]); !strings.Contains(title, "ALTERM: myhost") {
		t.Errorf("terminal title: %q", title)
	}
}

// TestPopupTitleSubject: a popup that names what it acts on writes the name
// after ": ", as it is (components/layout/popup, "Title").
func TestPopupTitleSubject(t *testing.T) {
	const subject = "filu ~/Work"
	cases := []struct {
		name, want string
		open       func(ta *testApp)
	}{
		{"form", "EDIT: ", func(ta *testApp) {
			f := testForm(nil)
			f.Title, f.Subject = "Edit", subject
			ta.Form(f)
		}},
		{"text", "RENAME: ", func(ta *testApp) { ta.Text(TextInput{Title: "Rename", Subject: subject}) }},
		{"password", "PASSWORD: ", func(ta *testApp) { ta.Password(PasswordInput{Title: "Password", Subject: subject}) }},
		{"number", "PORT: ", func(ta *testApp) { ta.Number(NumberInput{Title: "Port", Subject: subject}) }},
		{"textarea", "NOTE: ", func(ta *testApp) { ta.Textarea(Textarea{Title: "Note", Subject: subject}) }},
		{"confirm", "DELETE: ", func(ta *testApp) {
			ta.Confirm(Ask{Title: "Delete", Subject: subject, Question: "Delete it?", Verb: "delete"}, nil)
		}},
		{"date", "DUE: ", func(ta *testApp) { ta.PickDate(DatePicker{Title: "Due", Subject: subject}) }},
		{"color", "COLOUR: ", func(ta *testApp) { ta.PickColor(ColorPicker{Title: "Colour", Subject: subject}) }},
		{"slider", "LEVEL: ", func(ta *testApp) { ta.Open(Slider{Title: "Level", Subject: subject, Top: 10}) }},
		{"finder", "GO: ", func(ta *testApp) { ta.Find(&Finder{Title: "Go", Subject: subject}) }},
		{"select", "KIND: ", func(ta *testApp) {
			ta.Select(Select{Title: "Kind", Subject: subject, Options: []string{"a", "b"}, Selected: -1})
		}},
		{"checks", "TAGS: ", func(ta *testApp) {
			ta.Checks(Checks{Title: "Tags", Subject: subject, Options: []string{"a", "b"}, Checked: []bool{false, false}})
		}},
	}
	for _, c := range cases {
		ta := newTestApp(t)
		c.open(ta)
		ta.settle()
		if title := covCPlain(ta.Frame()[ta.top().y]); !strings.Contains(title, c.want+subject) {
			t.Errorf("%s: title %q, want %q", c.name, title, c.want+subject)
		}
	}
}

func TestUppercaseLeavesContent(t *testing.T) {
	ta := newTestApp(t)
	ta.Status = func(int) string { return "[N]arrow: off" }
	text := ta.screenText()
	for _, s := range []string{"web", "pod-00", "Enter:logs", "Space:menu", "?:help", "[N]arrow: off"} {
		if !strings.Contains(text, s) {
			t.Errorf("%q is not on the screen as it is", s)
		}
	}
	ta.press("Space")
	text = ta.screenText()
	for _, s := range []string{"[m]ark", "this pod", "Refresh", "Global operation", "Enter:run"} {
		if !strings.Contains(text, s) {
			t.Errorf("menu: %q is not as it is", s)
		}
	}
	ta.press("Esc", "?")
	text = ta.screenText()
	for _, s := range []string{"Enter", "Space", "q|Ctrl-C", "this list"} {
		if !strings.Contains(text, s) {
			t.Errorf("key reference: %q is not as it is", s)
		}
	}
	ta.press("Esc", "D")
	if text = ta.screenText(); !strings.Contains(text, "┫Drag┣") || strings.Contains(text, "DRAG") {
		t.Error("the mode name is not as it is")
	}
}

// --- layout/screen ------------------------------------------------------------------

func TestScreenRows(t *testing.T) {
	ta := newTestApp(t)
	lines := ta.Frame()
	if !strings.Contains(covCPlain(lines[0]), "[P]ANELS") {
		t.Errorf("the top row is not the screen chips: %q", covCPlain(lines[0]))
	}
	if got := covCPlain(lines[1]); got != strings.Repeat("━", 120) {
		t.Errorf("the second row is not a full-width line: %q", got)
	}
	if !strings.HasPrefix(covCPlain(lines[2]), "┏") || !strings.HasPrefix(covCPlain(lines[38]), "┗") {
		t.Error("the panels do not take the rows between")
	}
	if !strings.HasPrefix(covCPlain(lines[39]), "Space:menu") {
		t.Errorf("the last row is not the footer: %q", covCPlain(lines[39]))
	}
	// Each screen numbers its panels from [1].
	ta.press("O")
	if !strings.Contains(covCPlain(ta.Frame()[2]), "[1] ONLY") {
		t.Errorf("the other screen's panel: %q", covCPlain(ta.Frame()[2]))
	}
	// One screen: no chips; a statusbar when the app has one.
	one := &testApp{App: New("one")}
	one.AddScreen(&Screen{Name: "Main", Panels: []*Panel{{Label: "A", Body: &Pager{}}, {Label: "B", Body: &Pager{}}}})
	one.resize(60, 10)
	lines = one.Frame()
	if !strings.HasPrefix(covCPlain(lines[0]), "┏") || !strings.HasPrefix(covCPlain(lines[9]), "Space:menu") {
		t.Errorf("one screen without a statusbar:\n%s", one.screenText())
	}
	one.Status = func(int) string { return "status" }
	lines = one.Frame()
	if covCPlain(lines[0]) != Pad("status", 60) || !strings.HasPrefix(covCPlain(lines[1]), "┏") {
		t.Errorf("one screen with a statusbar:\n%s", one.screenText())
	}
}

func TestFooterDropsWholePairs(t *testing.T) {
	pairs := []string{"Space:menu", "?:help", "Tab|1–3:panels", "q:quit"}
	ta := newTestApp(t)
	for _, w := range []int{80, 39, 38, 32, 31, 17, 16, 10, 9} {
		ta.resize(w, 20)
		want := ""
		for n := len(pairs); n > 0; n-- {
			if s := strings.Join(pairs[:n], " "); Width(s) <= w {
				want = s
				break
			}
		}
		if got := covCFooter(ta); got != want {
			t.Errorf("%d wide: footer %q, want %q", w, got, want)
		}
	}
	ta.resize(120, 40)
	footer := ta.Frame()[39:]
	covCWant(t, "footer key", footer, "Tab", Blue, Color{})
	covCWant(t, "footer bar", footer, "|1", Overlay0, Color{})
	covCWant(t, "footer words", footer, ":panels", Overlay0, Color{})
}

func TestFooterVariants(t *testing.T) {
	ta := newTestApp(t)
	cases := []struct {
		name string
		keys []string
		want string
	}{
		{"normal", nil, "Space:menu ?:help Tab|1–3:panels q:quit"},
		{"mode", []string{"D"}, "?:help Enter:drop Esc:cancel"},
		{"filter typing", []string{"Esc", "/"}, "Enter|Tab:list Esc:clear"},
		{"filter, no results", []string{"z", "z", "z"}, "Tab:list Esc:clear"},
		{"popup", []string{"Esc", "Space"}, "Space:menu ?:help Tab|1–3:panels q:quit"},
		{"one panel", []string{"Esc", "O"}, "Space:menu ?:help q:quit"},
	}
	for _, c := range cases {
		ta.press(c.keys...)
		if got := covCFooter(ta); got != c.want {
			t.Errorf("%s: footer %q, want %q", c.name, got, c.want)
		}
	}
	pty, _, _ := ptyApp(t)
	if got := covCFooter(pty); !strings.HasPrefix(got, "Alt-Esc:leave") {
		t.Errorf("in a panel's PTY: footer %q", got)
	}
}

func TestFooterUnderPopupDimmed(t *testing.T) {
	ta := newTestApp(t)
	before := covCFooter(ta)
	ta.press("Space")
	if got := covCFooter(ta); got != before {
		t.Errorf("the footer changed under a popup: %q, was %q", got, before)
	}
	footer := ta.Frame()[39:]
	covCWant(t, "key under a popup", footer, "Space", DimColor(Blue), Color{})
	covCWant(t, "words under a popup", footer, ":menu", DimColor(Overlay0), Color{})
	// In a mode, under its key reference: the mode's footer, dimmed.
	ta.press("Esc", "D", "?")
	if got := covCFooter(ta); got != "?:help Enter:drop Esc:cancel" {
		t.Errorf("a mode's footer under its key reference: %q", got)
	}
	covCWant(t, "mode key under a popup", ta.Frame()[39:], "Enter", DimColor(Blue), Color{})
}

func TestScreenChips(t *testing.T) {
	ta := newTestApp(t)
	sl, h := glyphSlantLeft, glyphHard
	check := func(sel, unsel string) {
		t.Helper()
		row := ta.Frame()[0]
		if s := covCSpan(row, sel); s == nil || s[0].St != (covCStyle{Fg: Base, Bg: Blue, Bold: true}) {
			t.Errorf("the chosen screen %q: %+v", sel, s)
		}
		if s := covCSpan(row, unsel); s == nil || s[0].St != (covCStyle{Fg: Blue}) {
			t.Errorf("another screen %q: %+v", unsel, s)
		}
	}
	if got := covCPlain(ta.Frame()[0]); !strings.HasPrefix(got, sl+"[P]ANELS"+h+"[O]THER"+h+" ") {
		t.Errorf("one capsule per screen in a chain: %q", got)
	}
	check("[P]ANELS", "[O]THER")
	ta.press("O")
	check("[O]THER", "[P]ANELS")
}

func TestScreenChipsShrink(t *testing.T) {
	ta := newTestApp(t)
	ta.AddScreen(&Screen{Name: "Inputs", Key: "I", Panels: []*Panel{{Label: "Inputs", Body: &Pager{}}}})
	sl, h, s := glyphSlantLeft, glyphHard, glyphSoft
	chips := func(w int) string {
		ta.resize(w, 20)
		return strings.TrimRight(covCPlain(ta.Frame()[0]), " ")
	}
	cases := []struct {
		w    int
		want string
	}{
		{60, sl + "[P]ANELS" + h + "[O]THER" + s + "[I]NPUTS" + h},
		{20, sl + "[P]ANELS" + h + "[O]" + s + "[I]" + h},
		{14, sl + "[P]ANELS" + h + "…"},
	}
	for _, c := range cases {
		if got := chips(c.w); got != c.want {
			t.Errorf("%d wide: %q, want %q", c.w, got, c.want)
		}
	}
	if got := chips(8); !strings.HasPrefix(got, sl+"[P]") || Width(got) > 8 {
		t.Errorf("8 wide: the chosen chip stays, cut to the width: %q", got)
	}
	ta.press("O")
	if got, want := chips(20), sl+"[P]"+h+"[O]THER"+h+"[I]"+h; got != want {
		t.Errorf("the middle one chosen, 20 wide: %q, want %q", got, want)
	}
	if got, want := chips(14), sl+"[O]THER"+h+"…"; got != want {
		t.Errorf("the middle one chosen, 14 wide: %q, want %q", got, want)
	}
}

func TestChipRowProgress(t *testing.T) {
	ta := newTestApp(t)
	for _, c := range covCCells(ta.Frame()[1]) {
		if c.Ch != "━" || c.St.Fg != Overlay0 {
			t.Fatalf("the line under the chips: %q in %v at %d", c.Ch, c.St.Fg, c.Col)
		}
	}
	ta.Progress = func() (float64, bool) { return 0.25, true }
	for _, c := range covCCells(ta.Frame()[1]) {
		want := Overlay0
		if c.Col < 30 {
			want = Green
		}
		if c.Ch != "━" || c.St.Fg != want {
			t.Fatalf("a quarter done: %q in %v at %d, want %v", c.Ch, c.St.Fg, c.Col, want)
		}
	}
	// Under a terminal popup the chips and the line stay, progress and all.
	ta.open(&termView{t: &Terminal{Title: "Shell", in: &bytes.Buffer{}, started: true}, a: ta.App})
	ta.settle()
	lines := ta.Frame()
	if !strings.Contains(covCPlain(lines[0]), "[P]ANELS") || covCPlain(lines[1]) != strings.Repeat("━", 120) {
		t.Errorf("under a terminal popup:\n%s\n%s", covCPlain(lines[0]), covCPlain(lines[1]))
	}
	if left, right := covCAt(lines[1], 0).St.Fg, covCAt(lines[1], 119).St.Fg; left == right || (left != Green && left != DimColor(Green)) {
		t.Errorf("under a terminal popup the progress is gone: %v then %v", left, right)
	}
}

func TestStatusbar(t *testing.T) {
	ta := newTestApp(t)
	var widths []int
	ta.Status = func(w int) string {
		widths = append(widths, w)
		return Fg(Overlay0).Render("[N]arrow: ") + Fg(Lavender).Render("off")
	}
	lines := ta.Frame()
	// Several screens: the right of the chip row, not a row of its own.
	if row := covCPlain(lines[0]); !strings.HasPrefix(row, glyphSlantLeft+"[P]ANELS") || !strings.HasSuffix(row, "[N]arrow: off") {
		t.Errorf("chip row: %q", row)
	}
	if covCPlain(lines[1]) != strings.Repeat("━", 120) {
		t.Error("the statusbar took a row of its own")
	}
	// The app's colours are kept.
	covCWant(t, "statusbar label", lines[:1], "[N]arrow", Overlay0, Color{})
	covCWant(t, "you are here", lines[:1], "off", Lavender, Color{})
	// It is given the same width every time.
	ta.press("2", "j", "3")
	for _, w := range widths {
		if w != widths[0] {
			t.Errorf("the statusbar's width changed: %v", widths)
			break
		}
	}
	// Not a surface: Tab goes round the panels only.
	ta.press("1")
	for _, want := range []int{1, 2, 0} {
		ta.press("Tab")
		if ta.Screen().focus != want {
			t.Errorf("Tab went to %d, want panel %d", ta.Screen().focus, want)
		}
	}
	// One screen: the top row; too long, it is cut, never wrapped.
	one := &testApp{App: New("one")}
	one.AddScreen(&Screen{Name: "Main", Panels: []*Panel{{Label: "A", Body: &Pager{}}}})
	one.Status = func(int) string { return strings.Repeat("status ", 30) }
	one.resize(60, 10)
	lines = one.Frame()
	if covCPlain(lines[0]) != strings.Repeat("status ", 30)[:60] || !strings.HasPrefix(covCPlain(lines[1]), "┏") {
		t.Errorf("one screen, a long statusbar:\n%s", one.screenText())
	}
}

func TestEmptyState(t *testing.T) {
	ta := newTestApp(t)
	ta.list.EmptyFact, ta.list.EmptyHint = "No pods yet", "Press [A] to add a pod"
	ta.list.SetRows(nil)
	inside := covCCrop(ta.Frame(), 1, 39)
	fr, fact := covCFind(t, inside, "No pods yet")
	hr, hint := covCFind(t, inside, "Press [A] to add a pod")
	if hr != fr+2 {
		t.Errorf("the fact on row %d, the hint on %d: want one blank row between", fr, hr)
	}
	for _, s := range [][]covCCell{fact, hint} {
		if s[0].St.Fg != Overlay0 {
			t.Errorf("%q in %v, want Overlay0", s[0].Ch, s[0].St.Fg)
		}
		last := s[len(s)-1]
		if left, right := s[0].Col, 38-(last.Col+1); left-right > 1 || right-left > 1 {
			t.Errorf("not centred: %d cells left, %d right", left, right)
		}
	}
	// In the middle of the panel's inside, rows 3 to 37.
	if above, below := fr-3, 37-hr; above-below > 1 || below-above > 1 {
		t.Errorf("not in the middle: %d rows above, %d below", above, below)
	}
}

// --- layout/panel -----------------------------------------------------------------------

func TestPanelFrameFocus(t *testing.T) {
	ta := newTestApp(t)
	lines := ta.Frame()
	for _, c := range []struct {
		row, col int
		ch       string
		fg       Color
	}{
		{2, 0, "┏", Blue}, {2, 39, "┓", Blue}, {38, 0, "┗", Blue}, {38, 39, "┛", Blue}, {10, 0, "┃", Blue},
		{2, 40, "┌", Surface2}, {38, 79, "┘", Surface2}, {10, 80, "│", Surface2}, {10, 119, "│", Surface2},
	} {
		if got := covCAt(lines[c.row], c.col); got.Ch != c.ch || got.St.Fg != c.fg {
			t.Errorf("row %d column %d: %q in %v, want %q in %v", c.row, c.col, got.Ch, got.St.Fg, c.ch, c.fg)
		}
	}
	// Without the focus the content is dimmed, but for a stream (Rules T2).
	covCWant(t, "unfocused panel", covCCrop(lines, 80, 120), "NAME", DimColor(Blue), Color{})
	if st := covCStyleOf(t, covCCrop(lines, 40, 80), "one"); st.Fg.Set {
		t.Errorf("an unfocused stream is dimmed: %v", st.Fg)
	}
	// Changing the focus moves nothing (Rules L5).
	r0, before := covCFind(t, lines, "pod-01")
	ta.press("Tab")
	lines = ta.Frame()
	r1, after := covCFind(t, lines, "pod-01")
	if r0 != r1 || before[0].Col != after[0].Col {
		t.Errorf("pod-01 moved from %d,%d to %d,%d", r0, before[0].Col, r1, after[0].Col)
	}
	if c := covCAt(lines[2], 0); c.Ch != "┌" || c.St.Fg != Surface2 {
		t.Errorf("the panel that lost the focus: %q in %v", c.Ch, c.St.Fg)
	}
	if after[0].St.Fg != DimColor(Text) {
		t.Errorf("its rows are not dimmed: %v", after[0].St.Fg)
	}
}

func TestPanelTitle(t *testing.T) {
	ta := newTestApp(t)
	sl, h := glyphSlantLeft, glyphHard
	top := ta.Frame()[2]
	for _, c := range []struct {
		x0, x1 int
		want   string
	}{
		{0, 40, "┏" + sl + "[1] PODS" + h + "EVENTS" + h + "━"},
		{40, 80, "┌" + sl + "[2] LOG" + h + "─"},
		{80, 120, "┌" + sl + "[3] SETTINGS" + h + "─"},
	} {
		if got := covCPlain(Cut(top, c.x0, c.x1)); !strings.HasPrefix(got, c.want) {
			t.Errorf("title %q, want it to start %q", got, c.want)
		}
	}
	ta.press("O")
	if got := covCPlain(ta.Frame()[2]); !strings.HasPrefix(got, "┏"+sl+"[1] ONLY"+h) {
		t.Errorf("the other screen's first panel: %q", got)
	}
}

func TestPanelLoadingIconAfterChain(t *testing.T) {
	ta := newTestApp(t)
	ta.pods.Loading = true
	loading := map[string]bool{}
	for _, g := range loadingFrames {
		loading[g] = true
	}
	after := func(row, chainEnd string) {
		t.Helper()
		i := strings.Index(row, chainEnd)
		if i < 0 {
			t.Fatalf("%q not in %q", chainEnd, row)
		}
		r := []rune(row[i+len(chainEnd):])
		if len(r) < 4 || r[0] != ' ' || !loading[string(r[1])] || r[2] != ' ' || (r[3] != '━' && r[3] != '─') {
			t.Errorf("after the chain: %q, want a space, the loading icon, a space, the line", string(r[:min(len(r), 4)]))
		}
	}
	for _, tab := range []int{0, 1} {
		ta.pods.Tab = tab
		after(covCPlain(Cut(ta.Frame()[2], 0, 40)), "EVENTS"+glyphHard)
	}
	ta.Screen().Panels[1].Loading = true
	after(covCPlain(Cut(ta.Frame()[2], 40, 80)), "[2] LOG"+glyphHard)

	// The icon is in the frame's colour: Blue with focus, Surface2 without.
	color := func(line string) Color {
		t.Helper()
		for _, c := range covCCells(line) {
			if loading[c.Ch] {
				return c.St.Fg
			}
		}
		t.Fatalf("no loading icon in %q", covCPlain(line))
		return Color{}
	}
	if got := color(Cut(ta.Frame()[2], 0, 40)); got != Blue {
		t.Errorf("the icon on the focused panel is %v, not the frame's Blue", got)
	}
	if got := color(Cut(ta.Frame()[2], 40, 80)); got != Surface2 {
		t.Errorf("the icon on a panel without focus is %v, not the frame's Surface2", got)
	}
}

func TestPanelModeLabel(t *testing.T) {
	ta := newTestApp(t)
	ta.press("D")
	pods := covCCrop(ta.Frame(), 0, 40)
	if top := covCPlain(pods[2]); !strings.HasSuffix(top, "━┫Drag┣━┓") {
		t.Errorf("the mode on a focused panel: %q", top)
	}
	if st := covCStyleOf(t, pods, "Drag"); st != (covCStyle{Fg: Yellow, Bold: true}) {
		t.Errorf("the mode name: %+v", st)
	}
	for _, at := range [][2]int{{2, 0}, {2, 39}, {38, 0}} {
		if c := covCAt(pods[at[0]], at[1]); c.St.Fg != Yellow {
			t.Errorf("frame %q in %v, want the mode colour", c.Ch, c.St.Fg)
		}
	}
	for _, j := range []string{"┫", "┣"} {
		if s := covCSpan(pods[2], j); s == nil || s[0].St.Fg != Yellow {
			t.Errorf("junction %s: %+v", j, s)
		}
	}
	if st := covCStyleOf(t, pods, "[1] PODS"); st.Bg != Yellow {
		t.Errorf("the capsule on %v, want the mode colour", st.Bg)
	}
	ta.press("Esc")
	if top := covCPlain(Cut(ta.Frame()[2], 0, 40)); strings.Contains(top, "Drag") {
		t.Errorf("the mode name stayed: %q", top)
	}
	// A panel in a mode without the focus: thin junctions.
	ta.pods.EnterMode(&Mode{Name: "Select", Exit: "cancel"})
	ta.press("Tab")
	if top := covCPlain(Cut(ta.Frame()[2], 0, 40)); !strings.HasSuffix(top, "─┤Select├─┐") {
		t.Errorf("the mode on an unfocused panel: %q", top)
	}
}

func TestPanelModeLabelKeptWhenNarrow(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(42, 40)
	ta.press("D")
	top := covCPlain(Cut(ta.Frame()[2], 0, 14))
	if !strings.HasSuffix(top, "┫Drag┣━┓") {
		t.Errorf("the mode name is gone: %q", top)
	}
	if strings.Contains(top, "[1] PODS") || !strings.Contains(top, "[1]") {
		t.Errorf("the title is not the one cut: %q", top)
	}
	if Width(top) != 14 {
		t.Errorf("the top border is %d wide", Width(top))
	}
}

func TestPanelHint(t *testing.T) {
	ta := newTestApp(t)
	bottom := ta.Frame()[38]
	if got := covCPlain(Cut(bottom, 0, 40)); !strings.HasPrefix(got, "┗━ Enter:logs m:mark ━") || !strings.HasSuffix(got, "━ 1 of 40 ━┛") {
		t.Errorf("pods: %q", got)
	}
	if got := covCPlain(Cut(bottom, 80, 120)); !strings.HasPrefix(got, "└─ Enter:edit ─") {
		t.Errorf("settings: %q", got)
	}
}

func TestPanelNoHintWhileFiltering(t *testing.T) {
	ta := newTestApp(t)
	bottom := func() string { return covCPlain(Cut(ta.Frame()[38], 0, 40)) }
	for _, keys := range [][]string{{"/"}, {"1"}} {
		ta.press(keys...)
		if got := bottom(); !strings.HasPrefix(got, "┗━━") || strings.Contains(got, "Enter:") {
			t.Errorf("while typing in the filter: %q", got)
		}
	}
	ta.press("Enter")
	if got := bottom(); !strings.HasPrefix(got, "┗━ Enter:logs") {
		t.Errorf("back on the list: %q", got)
	}
	// In the panel's PTY.
	pty, term, _ := ptyApp(t)
	pty.Screen().Panels[0].Actions[0].Hint = "type"
	shell := func() string { return covCPlain(Cut(pty.Frame()[38], 0, 60)) }
	if got := shell(); strings.Contains(got, "Enter:") {
		t.Errorf("in the PTY: %q", got)
	}
	pty.press("Alt-Esc", "Enter")
	if term.Inside() {
		t.Fatal("did not leave the PTY")
	}
	if got := shell(); !strings.HasPrefix(got, "┗━ Enter:type") {
		t.Errorf("out of the PTY: %q", got)
	}
}

// TestPanelHintNoItems: with no item in its list, filtered to nothing or
// empty to begin with, the hint leaves out Enter and the keys on an item and
// keeps those on the whole panel (components/layout/panel, hint).
func TestPanelHintNoItems(t *testing.T) {
	ta := &testApp{App: New("test")}
	l := groupList()
	p := &Panel{Label: "Kinds", Body: l, Actions: []Action{
		{Key: "Enter", Label: "Open", Desc: "this kind", Scope: ItemScope, Hint: "open", Do: ta.record("open")},
		{Key: "x", Label: "Delete", Desc: "this kind", Scope: ItemScope, Hint: "delete", Do: ta.record("delete")},
		{Key: "n", Label: "New", Desc: "a kind", Scope: PanelScope, Hint: "new", Do: ta.record("new")},
	}}
	ta.AddScreen(&Screen{Name: "Kinds", Key: "W", Panels: []*Panel{p}})
	ta.resize(80, 24)
	hint := func() string {
		h, _ := p.hint()
		var out []string
		for _, k := range h {
			out = append(out, k.Key+":"+k.Desc)
		}
		return strings.Join(out, " ")
	}
	if got := hint(); got != "Enter:open x:delete n:new" {
		t.Errorf("with items: %q", got)
	}
	ta.press("/", "z", "z", "z", "Tab")
	if got := hint(); got != "n:new" {
		t.Errorf("filtered to nothing: %q", got)
	}
	ta.press("Enter", "x")
	if len(ta.ran) > 0 {
		t.Errorf("with no item, %v ran", ta.ran)
	}
	ta.press("Esc")
	l.SetRows(nil)
	if got := hint(); got != "n:new" {
		t.Errorf("empty: %q", got)
	}
}

func TestPanelFrameTextSpacing(t *testing.T) {
	ta := newTestApp(t)
	lines := ta.Frame()
	pods := func(row int) string { return covCPlain(Cut(lines[row], 0, 40)) }
	if got := pods(2); !strings.HasPrefix(got, "┏"+glyphSlantLeft) {
		t.Errorf("the capsule does not touch the corner: %q", got)
	}
	if got := pods(38); !strings.HasPrefix(got, "┗━ Enter") || !strings.HasSuffix(got, "40 ━┛") {
		t.Errorf("the hint and scroll position keep a segment from the corners: %q", got)
	}
	if got := covCPlain(Cut(lines[38], 80, 120)); !strings.HasPrefix(got, "└─ Enter") {
		t.Errorf("thin line: %q", got)
	}
	ta.press("D")
	lines = ta.Frame()
	if got := pods(2); !strings.HasSuffix(got, "┣━┓") {
		t.Errorf("the mode name keeps a segment from the corner: %q", got)
	}
}

func TestPanelHintContent(t *testing.T) {
	ta := newTestApp(t)
	hint := func() (string, []string) {
		lines := covCCrop(ta.Frame(), 0, 40)
		b := covCPlain(lines[38])
		b = strings.TrimPrefix(strings.TrimPrefix(b, "┗━ "), "└─ ")
		return b[:strings.Index(b, " ")+len(" m:mark")], lines[38:39]
	}
	got, row := hint()
	if got != "Enter:logs m:mark" {
		t.Errorf("hint %q, want the panel's own keys", got)
	}
	covCWant(t, "hint key", row, "Enter", Blue, Color{})
	covCWant(t, "hint words", row, ":logs", Overlay0, Color{})
	// Without the focus it stays, dimmed with the panel.
	ta.press("Tab")
	got, row = hint()
	if got != "Enter:logs m:mark" {
		t.Errorf("unfocused hint %q", got)
	}
	covCWant(t, "unfocused hint key", row, "Enter", DimColor(Blue), Color{})
	covCWant(t, "unfocused hint words", row, ":logs", DimColor(Overlay0), Color{})
}

func TestPanelHintDrop(t *testing.T) {
	ta := newTestApp(t)
	cases := []struct {
		w    int
		hint string
	}{
		{120, " Enter:logs m:mark "},
		{90, " Enter:logs "},
		{60, ""},
	}
	for _, c := range cases {
		ta.resize(c.w, 40)
		pw := c.w / 3
		got := covCPlain(Cut(ta.Frame()[38], 0, pw))
		if !strings.HasSuffix(got, " 1 of 40 ━┛") {
			t.Errorf("%d wide: the scroll position is gone: %q", pw, got)
		}
		want := "┗━" + c.hint + "━"
		if c.hint == "" {
			want = "┗━━"
		}
		if !strings.HasPrefix(got, want) || strings.Count(got, ":") != strings.Count(c.hint, ":") {
			t.Errorf("%d wide: %q, want whole pairs %q", pw, got, c.hint)
		}
	}
}

func TestCapsuleChain(t *testing.T) {
	none := Color{}
	style := func(fg, bg Color, bold bool) covCStyle { return covCStyle{Fg: fg, Bg: bg, Bold: bold} }
	// A panel on its second tab, in Blue: [1] and EVENTS chosen.
	ta := newTestApp(t)
	ta.press("l")
	top := Cut(ta.Frame()[2], 0, 40)
	for _, c := range []struct {
		what string
		cell covCCell
		ch   string
		want covCStyle
	}{
		{"left end", covCAt(top, 1), glyphSlantLeft, style(Blue, none, false)},
		{"[1]", covCSpan(top, "[1]")[0], "[", style(Base, Blue, true)},
		{"join to PODS", covCAfter(top, "[1]"), glyphHard, style(Blue, Base, false)},
		{"PODS", covCSpan(top, "PODS")[0], "P", style(Blue, none, false)},
		{"join to EVENTS", covCAfter(top, "PODS"), glyphHard, style(Base, Blue, false)},
		{"EVENTS", covCSpan(top, "EVENTS")[0], "E", style(Base, Blue, true)},
		{"right end", covCAfter(top, "EVENTS"), glyphHard, style(Blue, none, false)},
	} {
		if c.cell.Ch != c.ch || c.cell.St != c.want {
			t.Errorf("%s: %q %+v, want %q %+v", c.what, c.cell.Ch, c.cell.St, c.ch, c.want)
		}
	}
	// Joins between parts of one kind, and an end on an unchosen part.
	chain := drawChain([]Capsule{{"A", true}, {"B", true}, {"C", false}, {"D", false}}, Green, 40)
	cells := covCCells(chain)
	if got := covCPlain(chain); got != glyphSlantLeft+"A"+glyphSoft+"B"+glyphHard+"C"+glyphSoft+"D"+glyphHard {
		t.Fatalf("chain %q", got)
	}
	for i, want := range []covCStyle{
		style(Green, none, false), style(Base, Green, true), style(Base, Green, false), style(Base, Green, true),
		style(Green, Base, false), style(Green, none, false), style(Green, none, false), style(Green, none, false),
		style(Base, none, false),
	} {
		if cells[i].St != want {
			t.Errorf("cell %d %q: %+v, want %+v", i, cells[i].Ch, cells[i].St, want)
		}
	}
	if c := covCCells(drawChain([]Capsule{{"X", false}, {"Y", true}}, Green, 40))[0]; c.St.Fg != Base {
		t.Errorf("an end on an unchosen part is Base: %v", c.St.Fg)
	}
	// The chain's colour: the frame's on a panel, the layer's in a popup.
	ta.press("Tab")
	if st := covCStyleOf(t, covCCrop(ta.Frame(), 0, 40), "EVENTS"); st.Bg != Surface2 {
		t.Errorf("unfocused panel: chosen tab on %v", st.Bg)
	}
	ta.Viewer(&Viewer{Title: "DevTools", Tabs: []string{"Console", "Network"}, Lines: []string{"x"}})
	ta.settle()
	lines := ta.Frame()
	covCWant(t, "chosen popup tab", lines, "CONSOLE", Base, LayerColor(1))
	covCWant(t, "other popup tab", lines, "NETWORK", LayerColor(1), none)
	// Changing the chosen tab moves no text.
	p := &Panel{Label: "Pods", Tabs: []string{"Events", "Yaml"}, Tab: 1}
	cols := func() (int, int) {
		row := drawChain(p.capsules(1), Blue, 60)
		return covCSpan(row, "EVENTS")[0].Col, covCSpan(row, "YAML")[0].Col
	}
	e1, y1 := cols()
	p.Tab = 2
	if e2, y2 := cols(); e1 != e2 || y1 != y2 {
		t.Errorf("tabs moved: EVENTS %d→%d, YAML %d→%d", e1, e2, y1, y2)
	}
}

func TestCapsuleChainShrink(t *testing.T) {
	sl, h, s := glyphSlantLeft, glyphHard, glyphSoft
	caps := []Capsule{{"[1] PODS", true}, {"EVENTS", false}, {"YAML", false}, {"LOGS", false}}
	for _, c := range []struct {
		w    int
		want string
	}{
		{40, sl + "[1] PODS" + h + "EVENTS" + s + "YAML" + s + "LOGS" + h},
		{20, sl + "[1] PODS" + h + "E" + s + "Y" + s + "L" + h},
		{15, sl + "[1] PODS" + h + "E" + s + "Y" + h + "…"},
		{11, sl + "[1] PODS" + h + "…"},
	} {
		got := drawChain(caps, Blue, c.w)
		if covCPlain(got) != c.want || Width(got) > c.w {
			t.Errorf("%d cells: %q, want %q", c.w, covCPlain(got), c.want)
		}
	}
	if got := drawChain(caps, Blue, 6); Width(got) > 6 || !strings.HasPrefix(covCPlain(got), sl+"[1]") {
		t.Errorf("6 cells: the chosen part stays, cut: %q", covCPlain(got))
	}
	// Chosen parts in the middle stay; the others go from the end.
	mid := []Capsule{{"[1]", true}, {"PODS", false}, {"EVENTS", true}, {"YAML", false}}
	if got, want := covCPlain(drawChain(mid, Blue, 13)), sl+"[1]"+s+"EVENTS"+h+"…"; got != want {
		t.Errorf("13 cells: %q, want %q", got, want)
	}
	// A narrow panel keeps its frame's width.
	ta := newTestApp(t)
	ta.resize(30, 20)
	for i, l := range ta.Frame() {
		if Width(l) != 30 {
			t.Errorf("row %d is %d wide", i, Width(l))
		}
	}
}

func TestListCursor(t *testing.T) {
	ta := newTestApp(t)
	rows := ta.list.Rows()
	rows[0].Cells = []string{Fg(Red).Render("pod-00"), "0"}
	ta.list.SetRows(rows)
	check := func(block, arrow Color) {
		t.Helper()
		lines := covCCrop(ta.Frame(), 0, 40)
		r, _ := covCFind(t, lines, "pod-00")
		for _, c := range covCCells(lines[r]) {
			switch {
			case c.Col >= 1 && c.Col <= 37:
				if c.St != (covCStyle{Fg: Base, Bg: block, Bold: true}) {
					t.Errorf("column %d %q: %+v, want Base on %v", c.Col, c.Ch, c.St, block)
					return
				}
			case c.Col == 38:
				if c.Ch != glyphHard || c.St.Fg != arrow || c.St.Bg.Set {
					t.Errorf("the arrow: %q %+v, want %v", c.Ch, c.St, arrow)
				}
			}
		}
	}
	check(Subtext1, Subtext1)
	ta.press("Tab")
	check(DimColor(Subtext1), DimColor(Subtext1))
}

func TestCursorArrow(t *testing.T) {
	for _, pl := range []int{1, 2} {
		t.Run(fmt.Sprintf("powerline %d", pl), func(t *testing.T) {
			withWidths(t, 1, pl)
			ta := newTestApp(t)
			lines := covCCrop(ta.Frame(), 0, 40)
			r0, _ := covCFind(t, lines, "pod-00")
			if c := covCAt(lines[r0], 39-pl); c.Ch != glyphHard || c.St.Fg != Subtext1 || c.Col != 39-pl {
				t.Errorf("the arrow: %q at %d in %v", c.Ch, c.Col, c.St.Fg)
			}
			r1, before := covCFind(t, lines, "pod-01")
			for col := 39 - pl; col < 39; col++ {
				if c := covCAt(lines[r1], col); c.Ch != " " || c.St.Bg.Set {
					t.Errorf("another row at %d: %q", col, c.Ch)
				}
			}
			ta.press("j")
			if _, after := covCFind(t, covCCrop(ta.Frame(), 0, 40), "pod-01"); after[0].Col != before[0].Col {
				t.Errorf("the cursor moved the text: %d to %d", before[0].Col, after[0].Col)
			}
		})
	}
}

// covCSizes is a list with a text and a number column.
func covCSizes(rows ...[2]string) *List {
	l := &List{Columns: []Column{{Title: "Name"}, {Title: "Size", Number: true}}}
	var rs []Row
	for _, r := range rows {
		rs = append(rs, Row{Cells: []string{r[0], r[1]}})
	}
	l.SetRows(rs)
	return l
}

// covCNames are the names in a drawn list's rows, in order.
func covCNames(lines []string) []string {
	var out []string
	for _, l := range lines[1:] {
		if f := strings.Fields(StripStyles(l)); len(f) > 0 {
			out = append(out, f[0])
		}
	}
	return out
}

func TestListHeader(t *testing.T) {
	l := covCSizes([2]string{"b", "10"}, [2]string{"a", "9"}, [2]string{"c", "100"}, [2]string{"B2", "2"})
	lines := l.Draw(30, 6, false)
	if st := covCStyleOf(t, lines[:1], "NAME"); st != (covCStyle{Fg: Blue}) {
		t.Errorf("header: %+v, want Blue, not bold", st)
	}
	// Text to the left, numbers to the right.
	name, size := covCSpan(lines[0], "NAME"), covCSpan(lines[0], "SIZE")
	end := func(s []covCCell) int { return s[len(s)-1].Col }
	if b := covCSpan(lines[1], "b"); b[0].Col != name[0].Col {
		t.Errorf("text column: value at %d, title at %d", b[0].Col, name[0].Col)
	}
	for i, v := range []string{"10", "9", "100", "2"} {
		if s := covCSpan(Cut(lines[i+1], 3, 30), v); s == nil || end(s)+3 != end(size) {
			t.Errorf("number %s does not end under its title", v)
		}
	}
	// Numbers by value, text as text.
	l.SortBy(SortKey{Col: 1})
	if got := strings.Join(covCNames(l.Draw(30, 6, false)), ","); got != "B2,a,b,c" {
		t.Errorf("by size: %s", got)
	}
	l.SortBy(SortKey{Col: 0})
	if got := strings.Join(covCNames(l.Draw(30, 6, false)), ","); got != "a,b,B2,c" {
		t.Errorf("by name: %s", got)
	}
	// The header stays when the list scrolls.
	ta := newTestApp(t)
	ta.press("G")
	pods := covCCrop(ta.Frame(), 0, 40)
	if !strings.Contains(covCPlain(pods[3]), "NAME") || !strings.Contains(ta.screenText(), "pod-39") {
		t.Errorf("at the bottom of the list the header is gone: %q", covCPlain(pods[3]))
	}
}

func TestListSortGlyphs(t *testing.T) {
	l := covCSizes([2]string{"b", "1"}, [2]string{"a", "2"})
	after := func(title string) string {
		h := covCPlain(l.Draw(40, 3, false)[0])
		return strings.TrimLeft(h[strings.Index(h, title)+len(title):], " ")
	}
	l.SortBy(SortKey{Col: 0})
	if a := after("NAME"); !strings.HasPrefix(a, glyphSortAsc) {
		t.Errorf("ascending: %q after NAME", a)
	}
	if a := after("SIZE"); a != "" {
		t.Errorf("a column not sorted by has %q", a)
	}
	l.SortBy(SortKey{Col: 0, Desc: true})
	if a := after("NAME"); !strings.HasPrefix(a, glyphSortDesc) {
		t.Errorf("descending: %q after NAME", a)
	}
}

func TestListSortNumbers(t *testing.T) {
	l := covCSizes([2]string{"b", "1"}, [2]string{"a", "1"}, [2]string{"c", "5"})
	l.SortBy(SortKey{Col: 1, Desc: true}, SortKey{Col: 0})
	lines := l.Draw(50, 5, false)
	h := covCPlain(lines[0])
	for _, c := range [][2]string{{"SIZE (1)", glyphSortDesc}, {"NAME (2)", glyphSortAsc}} {
		i := strings.Index(h, c[0])
		if i < 0 || !strings.HasPrefix(strings.TrimLeft(h[i+len(c[0]):], " "), c[1]) {
			t.Errorf("header %q: want %q then its arrow", h, c[0])
		}
	}
	if got := strings.Join(covCNames(lines), ","); got != "c,a,b" {
		t.Errorf("by size down, then name: %s", got)
	}
	l.SortBy(SortKey{Col: 1})
	if h := covCPlain(l.Draw(50, 5, false)[0]); strings.Contains(h, "(1)") {
		t.Errorf("one column has no order number: %q", h)
	}
}

func TestPanelScrollPosition(t *testing.T) {
	ta := newTestApp(t)
	lines := ta.Frame()
	bottom := func(x0, x1 int) string { return covCPlain(Cut(lines[38], x0, x1)) }
	if got := bottom(0, 40); !strings.HasSuffix(got, " 1 of 40 ━┛") {
		t.Errorf("pods: %q", got)
	}
	covCWant(t, "scroll position", covCCrop(lines[38:], 0, 40), "1 of 40", Blue, Color{})
	if got := bottom(80, 120); strings.Contains(got, " of ") {
		t.Errorf("everything fits, yet %q", got)
	}
	ta.press("j", "Tab")
	lines = ta.Frame()
	if got := bottom(0, 40); !strings.HasSuffix(got, " 2 of 40 ─┘") {
		t.Errorf("pods after j, unfocused: %q", got)
	}
	covCWant(t, "unfocused scroll position", covCCrop(lines[38:], 0, 40), "2 of 40", Surface2, Color{})
	// Without a cursor: the lines in sight.
	var log []string
	for i := range 100 {
		log = append(log, fmt.Sprintf("line %d", i))
	}
	ta.Screen().Panels[1].Body.(*Pager).SetLines(log)
	lines = ta.Frame()
	if got := bottom(40, 80); !strings.HasSuffix(got, " 1-35 of 100 ━┛") {
		t.Errorf("log: %q", got)
	}
}

func TestListMarkColumn(t *testing.T) {
	for _, icon := range []int{1, 2} {
		t.Run(fmt.Sprintf("icon %d", icon), func(t *testing.T) {
			withWidths(t, icon, 1)
			ta := newTestApp(t)
			ta.press("m", "j")
			lines := covCCrop(ta.Frame(), 0, 40)
			r, word := covCFind(t, lines, "pod-00")
			mark := covCAt(lines[r], 2)
			if mark.Ch != glyphMarked || mark.St.Fg != Green {
				t.Errorf("the mark: %q in %v", mark.Ch, mark.St.Fg)
			}
			_, other := covCFind(t, lines, "pod-02")
			if word[0].Col != 2+icon+1 || other[0].Col != word[0].Col {
				t.Errorf("text at %d marked, %d unmarked; want both at %d", word[0].Col, other[0].Col, 2+icon+1)
			}
			ta.press("k", "m", "j")
			lines = covCCrop(ta.Frame(), 0, 40)
			r, word = covCFind(t, lines, "pod-00")
			if c := covCAt(lines[r], 2); c.Ch != " " || word[0].Col != 2+icon+1 {
				t.Errorf("unmarked: %q in the mark column, text at %d", c.Ch, word[0].Col)
			}
		})
	}
}

func TestListTruncate(t *testing.T) {
	l := &List{Columns: []Column{{Title: "Name"}, {Title: "Kind"}}}
	l.SetRows([]Row{{Cells: []string{"a-very-long-pod-name-that-does-not-fit", "Pod"}}})
	row := covCPlain(l.Draw(60, 2, false)[1])
	if !strings.Contains(row, "a-very-long-pod-name-that-does-not-fit  Pod") {
		t.Errorf("everything fits: %q", row)
	}
	row = covCPlain(l.Draw(24, 2, false)[1])
	if !strings.Contains(row, "a-very") || !strings.Contains(row, "…") {
		t.Errorf("a name cut from the end: %q", row)
	}
	m := &List{Columns: []Column{{Title: "Name", Middle: true}}}
	m.SetRows([]Row{{Cells: []string{"demo-app-7d4f9c8b66-gm72n"}}})
	if row := strings.Trim(covCPlain(m.Draw(1+17+1, 2, false)[1]), " "+glyphHard); row != "demo-app…66-gm72n" {
		t.Errorf("a pod name cut in the middle: %q", row)
	}
	p := &List{Columns: []Column{{Title: "Path", Path: true}}}
	p.SetRows([]Row{{Cells: []string{"/tmp"}}, {Cells: []string{"/home/me/.ssh/keys/id_ed25519"}}})
	row = strings.Trim(covCPlain(p.Draw(20, 3, false)[2]), " "+glyphHard)
	if !strings.HasPrefix(row, "…") || !strings.HasSuffix(row, "/id_ed25519") {
		t.Errorf("a path cut from the front: %q", row)
	}
}

// TestListColumnsGiveWay: columns that do not fit are taken away whole, the
// header with them, in the app's order, the name never; then the rest narrow
// in proportion; they come back as the panel grows (layout/panel).
func TestListColumnsGiveWay(t *testing.T) {
	l := &List{Columns: []Column{{Title: "Name"}, {Title: "Status", Drop: 2}, {Title: "Node", Drop: 1}}}
	l.SetRows([]Row{{Cells: []string{"web-0123456789", "Running", "worker-01"}}})
	// header and row, in the text room of a w-cell body
	draw := func(w int) (string, string) {
		lines := l.Draw(w, 2, false)
		return strings.Trim(covCPlain(lines[0]), " "+glyphHard), strings.Trim(covCPlain(lines[1]), " "+glyphHard)
	}
	if h, r := draw(1 + 14 + 2 + 7 + 2 + 9 + 1); h != "NAME            STATUS   NODE" || r != "web-0123456789  Running  worker-01" {
		t.Errorf("all fit: %q %q", h, r)
	}
	if h, r := draw(1 + 14 + 2 + 7 + 1); h != "NAME            STATUS" || r != "web-0123456789  Running" {
		t.Errorf("Drop 1 goes first, its header too: %q %q", h, r)
	}
	if h, r := draw(1 + 14 + 1); h != "NAME" || r != "web-0123456789" {
		t.Errorf("then Drop 2: %q %q", h, r)
	}
	if h, r := draw(1 + 10 + 1); h != "NAME" || r != "web-01234…" {
		t.Errorf("the name is never taken away; it is cut: %q %q", h, r)
	}
	if h, _ := draw(1 + 14 + 2 + 7 + 2 + 9 + 1); !strings.Contains(h, "NODE") {
		t.Errorf("grown again, they come back: %q", h)
	}
	// An app that marks every column: the last one left stays.
	every := &List{Columns: []Column{{Title: "Name", Drop: 2}, {Title: "Node", Drop: 1}}}
	every.SetRows([]Row{{Cells: []string{"web-0123456789", "worker-01"}}})
	if h := strings.Trim(covCPlain(every.Draw(1+10+1, 2, false)[0]), " "+glyphHard); h != "NAME" {
		t.Errorf("one column always stays: %q", h)
	}
	// Nothing to take away: every column narrows in proportion.
	p := &List{Columns: []Column{{Title: "A"}, {Title: "B"}}}
	p.SetRows([]Row{{Cells: []string{strings.Repeat("a", 30), strings.Repeat("b", 10)}}})
	if ws := p.widths(2 + 20); !slices.Equal(ws, []int{15, 5}) {
		t.Errorf("in proportion: %v, want [15 5]", ws)
	}
	// A list of one column with no title has no header (layout/panel).
	one := &List{Columns: []Column{{}}}
	one.SetRows([]Row{{Cells: []string{"Kinds"}}, {Cells: []string{"Pods"}}})
	if first := strings.Trim(covCPlain(one.Draw(20, 3, false)[0]), " "+glyphHard); first != "Kinds" {
		t.Errorf("no header: the first line is %q", first)
	}
}

func TestTruncateMiddle(t *testing.T) {
	for _, c := range []struct {
		s    string
		w    int
		want string
	}{
		{"demo-app-7d4f9c8b66-gm72n", 17, "demo-app…66-gm72n"},
		{"demo-app-7d4f9c8b66-gm72n", 16, "demo-ap…66-gm72n"},
		{"short", 10, "short"},
		{"abcdef", 1, "…"},
		{"漢字漢字漢字", 6, "漢…字"},
	} {
		if got := TruncateMiddle(c.s, c.w); got != c.want {
			t.Errorf("TruncateMiddle(%q, %d) = %q, want %q", c.s, c.w, got, c.want)
		}
	}
}

// TestListSortGroups: sorting keeps the app's groups apart, the lower first,
// and sorts within each, either way; no title between them (layout/panel).
func TestListSortGroups(t *testing.T) {
	l := &List{Columns: []Column{{Title: "Name"}}}
	l.SetRows([]Row{
		{ID: "b.txt", Cells: []string{"b.txt"}, SortGroup: 1}, {ID: "src", Cells: []string{"src"}},
		{ID: "a.txt", Cells: []string{"a.txt"}, SortGroup: 1}, {ID: "docs", Cells: []string{"docs"}},
	})
	order := func() string {
		var out []string
		for _, r := range l.view {
			out = append(out, l.rows[r].ID)
		}
		return strings.Join(out, " ")
	}
	if got := order(); got != "src docs b.txt a.txt" {
		t.Errorf("not sorted: the groups apart, each in its order: %q", got)
	}
	l.SortBy(SortKey{Col: 0})
	if got := order(); got != "docs src a.txt b.txt" {
		t.Errorf("by name: %q", got)
	}
	l.SortBy(SortKey{Col: 0, Desc: true})
	if got := order(); got != "src docs b.txt a.txt" {
		t.Errorf("by name, descending: the directories still first: %q", got)
	}
	if strings.Contains(StripStyles(strings.Join(l.Draw(20, 6, false), "\n")), "  src") {
		t.Error("no title, no indent between the groups")
	}
}

// TestListFixedOrder: the same rows, set in another order, come out in the
// same order (layout/panel): ties go by ID.
func TestListFixedOrder(t *testing.T) {
	rows := []Row{{ID: "ctx-b", Cells: []string{"prod"}}, {ID: "ctx-a", Cells: []string{"prod"}}, {ID: "ctx-c", Cells: []string{"dev"}}}
	l := &List{Columns: []Column{{Title: "Context"}}}
	l.SortBy(SortKey{Col: 0})
	l.SetRows(rows)
	first := slices.Clone(l.view)
	var ids []string
	for _, r := range first {
		ids = append(ids, l.rows[r].ID)
	}
	l.SetRows([]Row{rows[2], rows[0], rows[1]})
	var again []string
	for _, r := range l.view {
		again = append(again, l.rows[r].ID)
	}
	if !slices.Equal(ids, again) || strings.Join(ids, " ") != "ctx-c ctx-a ctx-b" {
		t.Errorf("one order every time: %v, then %v", ids, again)
	}
}

func TestPanelLoading(t *testing.T) {
	ta := newTestApp(t)
	ta.pods.Loading = true
	// Loading again: the old rows stay.
	if !strings.Contains(ta.screenText(), "pod-00") {
		t.Error("loading cleared the rows")
	}
	// Nothing yet: one sentence in the middle, the icon first.
	ta.list.SetRows(nil)
	ta.pods.LoadingText = "loading pods…"
	inside := covCCrop(ta.Frame(), 1, 39)
	r, s := covCFind(t, inside, " loading pods…")
	icon := covCAt(inside[r], s[0].Col-1)
	found := false
	for _, g := range loadingFrames {
		found = found || icon.Ch == g
	}
	if !found || icon.St.Fg != Overlay0 || s[1].St.Fg != Overlay0 {
		t.Errorf("%q in %v before %q in %v", icon.Ch, icon.St.Fg, "loading pods…", s[1].St.Fg)
	}
	if left, right := icon.Col, 38-(s[len(s)-1].Col+1); left-right > 1 || right-left > 1 {
		t.Errorf("not centred: %d left, %d right", left, right)
	}
}

func TestPanelError(t *testing.T) {
	ta := newTestApp(t)
	rows := ta.list.Rows()
	ta.pods.Fail = &Failure{Fact: "Forbidden: pods in kube-system", Hint: "Press [R] to retry"}
	// The old rows stay; the error goes to a toast, the app's (Rules F5).
	if text := ta.screenText(); !strings.Contains(text, "pod-00") || strings.Contains(text, "Forbidden") {
		t.Error("an error over old rows replaced them")
	}
	ta.list.SetRows(nil)
	inside := covCCrop(ta.Frame(), 1, 39)
	fr, fact := covCFind(t, inside, "Forbidden: pods in kube-system")
	hr, hint := covCFind(t, inside, "Press [R] to retry")
	if fact[0].St.Fg != Red || hint[0].St.Fg != Overlay0 {
		t.Errorf("fact in %v, hint in %v; want Red, Overlay0", fact[0].St.Fg, hint[0].St.Fg)
	}
	if hr != fr+2 {
		t.Errorf("fact on row %d, hint on %d", fr, hr)
	}
	for _, s := range [][]covCCell{fact, hint} {
		if left, right := s[0].Col, 38-(s[len(s)-1].Col+1); left-right > 1 || right-left > 1 {
			t.Errorf("not centred: %d left, %d right", left, right)
		}
	}
	ta.list.SetRows(rows)
	if text := ta.screenText(); !strings.Contains(text, "pod-00") || strings.Contains(text, "Forbidden") {
		t.Error("the rows back, the error should go")
	}
}

// covCValues is an app of one settings panel: Name (an input popup), Layout
// (a radio), Show (checkboxes), Remote (disabled).
type covCValues struct {
	*testApp
	vals    *Values
	name    string
	changes int
}

func covCValuesApp(t *testing.T, tabs ...string) *covCValues {
	t.Helper()
	c := &covCValues{testApp: &testApp{App: New("covc")}, name: "web"}
	changed := func() tea.Cmd { c.changes++; return nil }
	c.vals = &Values{Fields: []*Field{
		{Label: "Name", Value: func() string { return c.name }, Edit: func() tea.Cmd {
			return c.Text(TextInput{Title: "Name", Prompt: "A name", Old: c.name, Done: func(v string) tea.Cmd { c.name = v; return changed() }})
		}},
		{Label: "Layout", Options: []string{"row", "column"}, OnChange: changed},
		{Label: "Show", Options: []string{"status", "clock"}, Multi: true, OnChange: changed},
		{Label: "Remote", Value: func() string { return "x" }, Disabled: func() bool { return true }},
	}}
	c.AddScreen(&Screen{Name: "Main", Panels: []*Panel{{Label: "Settings", Tabs: tabs, Body: c.vals}}})
	c.resize(80, 20)
	return c
}

// item is the name of the item under the cursor: an option, or a label.
func (c *covCValues) item() string {
	it := c.vals.itemList()[c.vals.cursor]
	if it.option >= 0 {
		return it.field.Options[it.option]
	}
	return it.field.Label
}

func TestValuePanelMoves(t *testing.T) {
	c := covCValuesApp(t)
	for _, s := range []struct{ key, want string }{
		{"j", "row"}, {"l", "column"}, {"↓", "status"}, {"→", "clock"}, {"j", "Name"},
		{"k", "clock"}, {"h", "status"}, {"↑", "column"}, {"←", "row"}, {"k", "Name"},
	} {
		c.press(s.key)
		if got := c.item(); got != s.want {
			t.Errorf("%s went to %s, want %s", s.key, got, s.want)
		}
	}
	// With tabs, h|l are the tabs'.
	c = covCValuesApp(t, "Advanced")
	c.press("l")
	p := c.Screen().Focused()
	if p.Tab != 1 || c.item() != "Name" {
		t.Errorf("l with tabs: tab %d, item %s", p.Tab, c.item())
	}
	c.press("j")
	if c.item() != "row" {
		t.Errorf("j with tabs went to %s", c.item())
	}
}

func TestValuePanelEnter(t *testing.T) {
	c := covCValuesApp(t)
	c.press("Enter")
	if _, ok := c.topView().(*lineInput); !ok {
		t.Fatalf("Enter on a value did not open its input popup: %T", c.topView())
	}
	c.press("a", "p", "i", "Enter")
	if c.name != "api" || c.top() != nil || c.item() != "Name" {
		t.Errorf("after the popup: name %q, popup %T, focus on %s", c.name, c.topView(), c.item())
	}
	// A radio and checkboxes are chosen in place.
	c.press("j", "j", "Enter")
	if c.vals.Fields[1].Selected != 1 || c.top() != nil || c.item() != "column" {
		t.Errorf("radio: selected %d, popup %T, focus on %s", c.vals.Fields[1].Selected, c.topView(), c.item())
	}
	c.press("j", "Enter")
	if len(c.vals.Fields[2].Checked) == 0 || !c.vals.Fields[2].Checked[0] || c.item() != "status" {
		t.Errorf("checkbox: %v, focus on %s", c.vals.Fields[2].Checked, c.item())
	}
	c.press("Enter")
	if c.vals.Fields[2].Checked[0] {
		t.Error("a second Enter did not clear the checkbox")
	}
}

func TestValuePanelAppliesNow(t *testing.T) {
	c := covCValuesApp(t)
	c.press("j", "j", "Enter")
	if c.changes != 1 || !strings.Contains(c.screenText(), glyphRadioOn+" column") {
		t.Errorf("a radio takes effect when chosen: %d changes", c.changes)
	}
	c.press("gg", "Enter", "x", "Enter")
	if c.changes != 2 || c.name != "x" || !strings.Contains(c.screenText(), "x") {
		t.Errorf("a value takes effect when its popup is confirmed: %d changes, %q", c.changes, c.name)
	}
	// Nothing to submit, nothing to ask on the way out.
	c.press("Esc")
	if c.top() != nil {
		t.Errorf("Esc after changes opened %T", c.topView())
	}
	if text := c.screenText(); strings.Contains(text, "Save") || strings.Contains(text, "Apply") {
		t.Error("a value panel has an action row")
	}
}

func TestValuePanelFocusBlock(t *testing.T) {
	c := covCValuesApp(t)
	block := func(text string) {
		t.Helper()
		lines := c.Frame()
		r, s := covCFind(t, lines, text)
		start := s[0].Col
		if text != "web" {
			start -= IconWidth() + 1 // an option's block starts at its glyph
		}
		for _, cell := range covCCells(lines[r]) {
			switch {
			case cell.Col < start:
				if cell.St.Bg.Set {
					t.Errorf("%s: column %d before the item is on %v", text, cell.Col, cell.St.Bg)
					return
				}
			case cell.Col < 78:
				if cell.St != (covCStyle{Fg: Base, Bg: Subtext1, Bold: true}) {
					t.Errorf("%s: column %d %q: %+v", text, cell.Col, cell.Ch, cell.St)
					return
				}
			case cell.Col == 78:
				if cell.Ch != glyphHard || cell.St.Fg != Subtext1 || cell.St.Bg.Set {
					t.Errorf("%s: the arrow %q %+v", text, cell.Ch, cell.St)
				}
			}
		}
	}
	block("web")
	c.press("j", "j")
	block("column")
}

func TestValuePanelLabels(t *testing.T) {
	ta := newTestApp(t)
	ta.press("3")
	settings := func() []string { return covCCrop(ta.Frame(), 80, 120) }
	s := settings()
	covCWant(t, "label", s, "NAME", Blue, Color{})
	covCWant(t, "label", s, "LAYOUT", Blue, Color{})
	covCWant(t, "disabled label", s, "REMOTE", Surface2, Color{})
	// The label column is the longest label's, then two cells.
	_, label := covCFind(t, s, "LAYOUT")
	for _, v := range []string{"web", glyphRadioOn, "x  "} {
		if _, cell := covCFind(t, s, v); cell[0].Col != label[0].Col+6+2 {
			t.Errorf("%q at %d, want %d", v, cell[0].Col, label[0].Col+8)
		}
	}
	// A wrong value turns its label Red, before anything else.
	ta.values.Fields[2].Err = func() bool { return true }
	covCWant(t, "wrong and disabled", settings(), "REMOTE", Red, Color{})
	ta.values.Fields[0].Err = func() bool { return true }
	covCWant(t, "wrong", settings(), "NAME", Red, Color{})
	ta.press("Tab")
	covCWant(t, "unfocused label", settings(), "LAYOUT", DimColor(Blue), Color{})
}

// A one-value-per-row panel's header is Overlay0, not the labels' Blue, and
// is dimmed with the panel (components/layout/panel).
func TestValuePanelHeader(t *testing.T) {
	ta := newTestApp(t)
	ta.values.Header = [2]string{"Property", "Value"}
	ta.press("3")
	settings := func() []string { return covCCrop(ta.Frame(), 80, 120) }
	covCWant(t, "header", settings(), "PROPERTY", Overlay0, Color{})
	covCWant(t, "header", settings(), "VALUE", Overlay0, Color{})
	covCWant(t, "label", settings(), "NAME", Blue, Color{})
	ta.press("Tab")
	covCWant(t, "unfocused header", settings(), "PROPERTY", DimColor(Overlay0), Color{})
}

func TestPanelFilterListKeys(t *testing.T) {
	ta := newTestApp(t)
	ta.press("/", "1", "5", "Enter")
	if ta.list.Typing() || len(ta.list.view) != 1 {
		t.Fatalf("after Enter: typing %v, %d rows", ta.list.Typing(), len(ta.list.view))
	}
	// On the list the typing row is grey and the list stays filtered.
	pods := covCCrop(ta.Frame(), 0, 40)
	covCWant(t, "typing row on the list", pods, glyphTyping, Overlay0, Color{})
	covCWant(t, "typed words on the list", pods, "15", Overlay0, Color{})
	// / goes back to the typing row with its words.
	ta.press("/")
	pods = covCCrop(ta.Frame(), 0, 40)
	if !ta.list.Typing() || !strings.Contains(covCPlain(pods[3]), glyphTyping+" 15") {
		t.Errorf("/ on the list: typing %v, %q", ta.list.Typing(), covCPlain(pods[3]))
	}
	covCWant(t, "typing row", pods, glyphTyping, Blue, Color{})
	// Tab on the list: the next panel.
	ta.press("Enter", "Tab")
	if ta.Screen().focus != 1 || !ta.list.Filtering() {
		t.Errorf("Tab on the list: panel %d, filter %v", ta.Screen().focus+1, ta.list.Filtering())
	}
	// Esc on the list clears the filter.
	ta.press("1", "Enter", "Esc")
	if ta.list.Filtering() {
		t.Error("Esc on the list kept the filter")
	}
}

func TestPanelFilterFocusReturn(t *testing.T) {
	ta := newTestApp(t)
	at := func(panel int, typing bool) {
		t.Helper()
		if ta.Screen().focus != panel || ta.list.Typing() != typing {
			t.Errorf("panel %d typing %v, want panel %d typing %v", ta.Screen().focus+1, ta.list.Typing(), panel+1, typing)
		}
	}
	ta.press("/", "1", "5")
	ta.press("Tab")
	at(0, false)
	ta.press("Tab")
	at(1, false)
	// Away from the panel the filter stays.
	if text := StripStyles(strings.Join(covCCrop(ta.Frame(), 0, 40), "\n")); !strings.Contains(text, glyphTyping+" 15") ||
		!strings.Contains(text, "pod-15") || strings.Contains(text, "pod-14") {
		t.Errorf("the filter is gone while away:\n%s", text)
	}
	// Back to it: on the typing row; Tab stops twice on the way through.
	ta.press("Tab", "Tab")
	at(0, true)
	ta.press("Tab")
	at(0, false)
	ta.press("Tab")
	at(1, false)
	ta.press("1")
	at(0, true)
}

// TestValuesEmptyState: a one-value-per-row panel with nothing to list says
// so as a list does, the fact and the hint in Overlay0 (layout/screen, the
// empty state; ccdrawu CD23).
func TestValuesEmptyState(t *testing.T) {
	ta := newTestApp(t)
	ta.values.Fields = nil
	ta.values.EmptyFact, ta.values.EmptyHint = "No settings here", "Press [A] to add one"
	ta.press("3")
	inside := covCCrop(ta.Frame(), 81, 119)
	for _, text := range []string{"No settings here", "Press [A] to add one"} {
		if _, cells := covCFind(t, inside, text); cells[0].St.Fg != Overlay0 {
			t.Errorf("%q in %v, want Overlay0", text, cells[0].St.Fg)
		}
	}
}
