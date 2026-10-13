package terminu

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// withSplash gives the test app a splash that reveals "MARK" a letter a
// frame.
func withSplash(ta *testApp) *testApp {
	ta.Splash = func(frame, w, h int) ([]string, bool) {
		n := min(frame+1, 4)
		return []string{"MARK"[:n]}, n == 4
	}
	return ta
}

func TestS1SplashOnV(t *testing.T) {
	ta := withSplash(newTestApp(t))
	cmd := ta.press("V")
	if !ta.splashOn {
		t.Fatal("V on a panel did not open the splash")
	}
	for range 10 {
		if cmd == nil {
			break
		}
		cmd = ta.Update2(cmd())
	}
	if got := ta.screenText(); !strings.HasPrefix(got, "MARK") || strings.Contains(got, "PODS") {
		t.Errorf("the splash, revealed, over the whole screen:\n%s", got)
	}
	ta.press("x")

	// A panel may give V to an action of its own; the splash does not open there.
	ta.pods.Actions = append(ta.pods.Actions, Action{Key: "V", Label: "View", Desc: "it", Scope: ItemScope, Do: ta.record("view")})
	ta.press("V")
	if ta.splashOn || len(ta.ran) == 0 || ta.ran[len(ta.ran)-1] != "view" {
		t.Errorf("the panel's own V: splash %v, ran %v", ta.splashOn, ta.ran)
	}
	ta.press("2", "V")
	if !ta.splashOn {
		t.Error("another panel still opens the splash")
	}
}

func TestS1EveryAppHasSplash(t *testing.T) {
	ta := newTestApp(t)
	if ta.checkSplash() == nil {
		t.Error("an app without a splash")
	}
	withSplash(ta)
	if err := ta.checkSplash(); err != nil {
		t.Error(err)
	}
	ta.Global = append(ta.Global, Action{Key: "V", Label: "Vault", Scope: GlobalScope})
	if ta.checkSplash() == nil {
		t.Error("a global hotkey on V")
	}
	ta.Global = ta.Global[:len(ta.Global)-1]
	for _, s := range ta.screens {
		for _, p := range s.Panels {
			p.Actions = append(p.Actions, Action{Key: "V", Label: "View", Scope: PanelScope})
		}
	}
	if ta.checkSplash() == nil {
		t.Error("every panel gave V away")
	}
}

func TestS2SplashNotDisclosed(t *testing.T) {
	ta := withSplash(newTestApp(t))
	seen := StripStyles(strings.Join(ta.Frame(), "\n")) // footer and hints
	for _, keys := range [][]string{{"Space"}, {"?"}, {"Space", "G", "Enter"}} {
		ta.press(keys...)
		seen += ta.screenText()
		for len(ta.stack) > 0 {
			ta.press("Esc")
		}
	}
	if strings.Contains(strings.ToLower(seen), "splash") {
		t.Error("the splash is listed somewhere")
	}
}

func TestS3AnyKeyClosesOnly(t *testing.T) {
	ta := withSplash(newTestApp(t))
	for _, k := range []string{"q", "Ctrl-C", "Esc", "Space", "?", "Tab", "S"} {
		ta.press("V")
		focus := ta.Screen().focus
		cmd := ta.press(k)
		if ta.splashOn {
			t.Errorf("%s did not close the splash", k)
		}
		if cmd != nil || ta.top() != nil || ta.quitting || ta.Screen().focus != focus || len(ta.ran) > 0 {
			t.Errorf("%s did more than close the splash", k)
		}
	}
	ta.press("V", "Ctrl-C", "Ctrl-C")
	if ta.top() == nil || !ta.quitting {
		t.Error("Ctrl-C after the splash is closed starts the quit flow")
	}
}

func TestS4SplashOnlyOnPanel(t *testing.T) {
	ta := withSplash(newTestApp(t))
	if ta.splashOn {
		t.Fatal("the splash plays at the start")
	}
	ta.press("Space", "V")
	if ta.splashOn {
		t.Error("V over a popup")
	}
	ta.press("Esc", "/", "V")
	if ta.splashOn || ta.list.filter.value() != "V" {
		t.Errorf("V on a typing row is a letter: splash %v, filter %q", ta.splashOn, ta.list.filter.value())
	}
	ta.press("Esc", "Esc", "D", "V")
	if !ta.splashOn {
		t.Error("V in a mode, still on the panel")
	}
}

// showcaseSplash is the showcase's splash, made from its copy of tdp's
// principle/icon.svg.
func showcaseSplash(t *testing.T) (Splash, []byte) {
	t.Helper()
	icon, err := os.ReadFile("examples/showcase/icon.svg")
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSplash(icon, "showcase", "0.0.0-dev", "Every component of terminu design")
	if err != nil {
		t.Fatal(err)
	}
	return s, icon
}

// svgColours is the colour of every cell of the svg's 25 × 25 grid: its rects,
// and Surface0 where it has none.
func svgColours(t *testing.T, icon []byte) [25][25]Color {
	t.Helper()
	var out [25][25]Color
	for r := range out {
		for c := range out[r] {
			out[r][c] = Hex("#313244")
		}
	}
	re := regexp.MustCompile(`<rect x="(\d+)" y="(\d+)" width="10" height="10" fill="(#[0-9a-fA-F]{6})"`)
	for _, m := range re.FindAllStringSubmatch(string(icon), -1) {
		x, _ := strconv.Atoi(m[1])
		y, _ := strconv.Atoi(m[2])
		out[y/10][x/10] = Hex(m[3])
	}
	return out
}

// squares is the foreground colour of every square on a line, in order.
func squares(line string) []Color {
	var out []Color
	var fg Color
	re := regexp.MustCompile("\x1b" + `\[([0-9;]*)m|\x{F0C8}`)
	for _, m := range re.FindAllStringSubmatch(line, -1) {
		if m[0] == "\U0000F0C8" {
			out = append(out, fg)
			continue
		}
		fg = Color{}
		p := strings.Split(m[1], ";")
		for i := 0; i+4 < len(p); i++ {
			if p[i] == "38" && p[i+1] == "2" {
				r, _ := strconv.Atoi(p[i+2])
				g, _ := strconv.Atoi(p[i+3])
				b, _ := strconv.Atoi(p[i+4])
				fg = Color{RGB: RGB{uint8(r), uint8(g), uint8(b)}, Set: true}
			}
		}
	}
	return out
}

func TestS5SplashMatchesIcon(t *testing.T) {
	icon0, power0 := IconWidth(), PowerlineWidth()
	defer SetIconWidth(icon0, power0)
	for _, iw := range []int{1, 2} {
		SetIconWidth(iw, 1)
		s, icon := showcaseSplash(t)
		want := svgColours(t, icon)
		var lines []string
		for f := 0; ; f++ {
			var done bool
			if lines, done = s(f, 120, 40); done {
				break
			}
			if f > 1000 {
				t.Fatal("the reveal never ends")
			}
		}
		// 21 rows of mark, a blank row and eight of text: 30 rows in the
		// middle of 40; 50 columns in the middle of 120.
		const y, x = 5, 35
		for r := range 21 {
			line := lines[y+r]
			if got := Width(line); got != x+50 {
				t.Errorf("icon width %d, mark row %d: %d cells, want %d", iw, r, got, x+50)
			}
			if iw == 1 && strings.Count(StripStyles(line), "\U0000F0C8 ") != 25 {
				t.Errorf("icon width 1, mark row %d: a square with no space after it: %q", r, StripStyles(line))
			}
			got := squares(line)
			if len(got) != 25 {
				t.Fatalf("icon width %d, mark row %d: %d squares", iw, r, len(got))
			}
			for c := range 25 {
				if got[c] != want[r+2][c] {
					t.Errorf("icon width %d, cell (%d, %d): %v, want %v", iw, c, r+2, got[c], want[r+2][c])
				}
			}
		}
		under := []struct {
			s     string
			style Style
		}{
			{"", Style{}},
			{"showcase", Fg(Blue).Bolded()},
			{"0.0.0-dev", Fg(Blue)},
			{"Every component of terminu design", Fg(Text)},
			{"", Style{}},
			{"developed by", Fg(Overlay0)},
			{"vulcan.shen.2304@gmail.com", Fg(Hex("#f2b753"))},
			{"", Style{}},
			{"Press any key to close", Fg(Overlay0)},
		}
		for i, u := range under {
			line := lines[y+21+i]
			if u.s == "" {
				if strings.TrimSpace(line) != "" {
					t.Errorf("row %d under the mark is not blank: %q", i, line)
				}
				continue
			}
			if StripStyles(line) != strings.Repeat(" ", x)+Center(u.s, 50) || !strings.Contains(line, u.style.Render(u.s)) {
				t.Errorf("row %d under the mark, centred under it in its colour: %q", i, line)
			}
		}
	}
}

func TestS5SplashReveal(t *testing.T) {
	s, icon := showcaseSplash(t)
	want := svgColours(t, icon)
	gold, frame := Hex("#f2b753"), Hex("#205090")
	var goldN, frameN int
	for r := 2; r <= 22; r++ {
		for c := range 25 {
			switch want[r][c] {
			case gold:
				goldN++
			case frame:
				frameN++
			}
		}
	}
	// One letter (the T) a stage, 2 cells a tick; the frame 3 a tick; 25
	// ticks between stages; the text 40 and 50 ticks after the mark.
	letter := 20 + 25
	letterEnd := letter + (goldN+1)/2 - 1
	rise := letterEnd + 25
	riseEnd := rise + (frameN+2)/3 - 1
	text, text2 := riseEnd+40, riseEnd+90

	var shownGold [][2]int
	prevGold, prevFrame := 0, 0
	for f := 0; f <= text2; f++ {
		lines, done := s(f, 120, 40)
		if done != (f == text2) {
			t.Fatalf("frame %d: done %v, want it at %d", f, done, text2)
		}
		rows, nGold, nFrame := 0, 0, 0
		lowest := 99 // the highest row the frame has reached
		var unshown []int
		for r := range 21 {
			sq := squares(lines[5+r])
			if len(sq) > 0 {
				rows++
			}
			for c, col := range sq {
				switch {
				case col == gold:
					nGold++
					if f > 0 && !containsCell(shownGold, c, r) {
						shownGold = append(shownGold, [2]int{c, r})
					}
				case col == frame:
					nFrame++
					lowest = min(lowest, r)
				case want[r+2][c] == frame:
					unshown = append(unshown, r)
				}
			}
		}
		if want := min(f+1, 21); rows != want {
			t.Errorf("frame %d: %d rows of background, want %d (a row a tick)", f, rows, want)
		}
		switch {
		case f < letter && nGold > 0, f < rise && nFrame > 0:
			t.Errorf("frame %d: a stage before its time: %d gold, %d frame", f, nGold, nFrame)
		case f >= letter && f <= letterEnd && nGold-prevGold != min(2, goldN-prevGold):
			t.Errorf("frame %d: %d letter cells more, want 2 a tick", f, nGold-prevGold)
		case f >= rise && f <= riseEnd && nFrame-prevFrame != min(3, frameN-prevFrame):
			t.Errorf("frame %d: %d frame cells more, want 3 a tick", f, nFrame-prevFrame)
		}
		for _, r := range unshown {
			if r > lowest {
				t.Errorf("frame %d: the frame reached row %d before row %d (it rises from the bottom)", f, lowest, r)
			}
		}
		prevGold, prevFrame = nGold, nFrame
		name := strings.Contains(lines[5+22], "showcase")
		author := strings.Contains(lines[5+26], "developed by")
		if name != (f >= text) || author != (f >= text2) {
			t.Errorf("frame %d: name %v (from %d), author %v (from %d)", f, name, text, author, text2)
		}
	}
	if len(shownGold) != goldN {
		t.Fatalf("%d letter cells revealed, want %d", len(shownGold), goldN)
	}
	inOrder := true
	for i := 1; i < len(shownGold); i++ {
		a, b := shownGold[i-1], shownGold[i]
		if a[1] > b[1] || a[1] == b[1] && a[0] > b[0] {
			inOrder = false
		}
	}
	if inOrder {
		t.Error("the letter's cells appear in reading order, not scattered")
	}
}

func containsCell(cells [][2]int, c, r int) bool {
	for _, x := range cells {
		if x == [2]int{c, r} {
			return true
		}
	}
	return false
}

func TestS5SplashFillsScreen(t *testing.T) {
	s, _ := showcaseSplash(t)
	ta := newTestApp(t)
	ta.Splash = s
	ta.press("V")
	for range 1000 {
		if ta.Update2(splashMsg{ta.splashID}) == nil {
			break
		}
	}
	for _, size := range [][2]int{{120, 40}, {80, 24}, {40, 12}, {50, 30}} {
		ta.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		lines := ta.Frame()
		if len(lines) != size[1] {
			t.Errorf("%v: %d lines", size, len(lines))
		}
		for i, l := range lines {
			if Width(l) != size[0] {
				t.Errorf("%v: line %d is %d wide", size, i, Width(l))
			}
		}
		if strings.Contains(ta.screenText(), "PODS") {
			t.Errorf("%v: the screen shows under the splash", size)
		}
	}
}

func TestS5SplashRefusesBadIcon(t *testing.T) {
	for _, icon := range []string{
		`<svg><rect x="20" y="30" width="10" height="10" fill="#205090"/></svg>`,                                                             // no letters
		`<svg><rect x="20" y="30" width="10" height="10" fill="#205090"/><rect x="60" y="40" width="10" height="10" fill="#ff0000"/></svg>`,  // a third colour
		`<svg><rect x="20" y="30" width="10" height="10" fill="#205090"/><rect x="60" y="220" width="10" height="10" fill="#f2b753"/></svg>`, // under the mark
		`<svg><rect`,
	} {
		if _, err := NewSplash([]byte(icon), "x", "v1", "y"); err == nil {
			t.Errorf("no error for %s", icon)
		}
	}
}
