package terminu

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

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
	if ta.splashOn || ta.pods.InMode() == nil {
		t.Errorf("V in a mode does nothing (tdp S4, K11): splash %v, in the mode %v", ta.splashOn, ta.pods.InMode() != nil)
	}
	ta.press("Esc")
	viewed := 0
	ta.pods.EnterMode(&Mode{Name: "Look", Keys: []ModeKey{{Key: "V", Desc: "view", Do: func(string) tea.Cmd { viewed++; return nil }}}})
	ta.press("V")
	if ta.splashOn || viewed != 1 {
		t.Errorf("V a mode's own key is that key: splash %v, ran %d", ta.splashOn, viewed)
	}
	ta.press("Esc", "V")
	if !ta.splashOn {
		t.Error("out of the mode V on the panel summons it")
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
	goldN := 0
	// The frame's ticks: a mirrored pair, or the middle cell, of a row.
	type step struct{ row, off int }
	steps := map[step]bool{}
	for r := 2; r <= 22; r++ {
		for c := range 25 {
			switch want[r][c] {
			case gold:
				goldN++
			case frame:
				steps[step{r - 2, abs(c - 12)}] = true
			}
		}
	}
	// One letter (the T) a stage, 2 cells a tick; the frame a pair a tick; 25
	// ticks between stages; the text 40 and 50 ticks after the mark.
	letter := 20 + 25
	letterEnd := letter + (goldN+1)/2 - 1
	rise := letterEnd + 25
	riseEnd := rise + len(steps) - 1
	text, text2 := riseEnd+40, riseEnd+90

	var shownGold [][2]int
	shownFrame := map[[2]int]bool{}
	prevGold := 0
	last := step{99, -1} // the frame's last step: rows rise, offsets grow
	for f := 0; f <= text2; f++ {
		lines, done := s(f, 120, 40)
		if done != (f == text2) {
			t.Fatalf("frame %d: done %v, want it at %d", f, done, text2)
		}
		rows, nGold := 0, 0
		var added [][2]int
		for r := range 21 {
			sq := squares(lines[5+r])
			if len(sq) > 0 {
				rows++
			}
			for c, col := range sq {
				switch col {
				case gold:
					nGold++
					if f > 0 && !containsCell(shownGold, c, r) {
						shownGold = append(shownGold, [2]int{c, r})
					}
				case frame:
					if !shownFrame[[2]int{c, r}] {
						shownFrame[[2]int{c, r}] = true
						added = append(added, [2]int{c, r})
					}
				}
			}
		}
		if want := min(f+1, 21); rows != want {
			t.Errorf("frame %d: %d rows of background, want %d (a row a tick)", f, rows, want)
		}
		switch {
		case f < letter && nGold > 0, f < rise && len(shownFrame) > 0:
			t.Errorf("frame %d: a stage before its time: %d gold, %d frame", f, nGold, len(shownFrame))
		case f >= letter && f <= letterEnd && nGold-prevGold != min(2, goldN-prevGold):
			t.Errorf("frame %d: %d letter cells more, want 2 a tick", f, nGold-prevGold)
		case f >= rise && f <= riseEnd:
			// One step a tick: the two cells mirrored about the middle, or
			// the middle cell alone; the row below first, the middle out.
			ok := len(added) == 1 && added[0][0] == 12 ||
				len(added) == 2 && added[0][1] == added[1][1] && added[0][0]+added[1][0] == 24
			if !ok {
				t.Errorf("frame %d: the frame grew by %v, not a mirrored pair or the middle cell", f, added)
				break
			}
			now := step{added[0][1], abs(added[0][0] - 12)}
			if now.row > last.row || now.row == last.row && now.off <= last.off {
				t.Errorf("frame %d: the frame grew at %v after %v (bottom up, middle out)", f, now, last)
			}
			last = now
		case len(added) > 0:
			t.Errorf("frame %d: the frame grew outside its stage: %v", f, added)
		}
		prevGold = nGold
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

// TestS5SplashTooSmall: a terminal under 50 columns or 30 rows gets the text
// alone, centred both ways, in its two batches; too narrow, it wraps between
// words; too short, the hint goes, then the author, then the intro, each with
// the blank row above it, and the name and version stay (tdp S5).
func TestS5SplashTooSmall(t *testing.T) {
	icon, err := os.ReadFile("examples/showcase/icon.svg")
	if err != nil {
		t.Fatal(err)
	}
	intro := "Every component of terminu design, in one app"
	s, err := NewSplash(icon, "showcase", "0.0.0-dev", intro)
	if err != nil {
		t.Fatal(err)
	}
	// rows is what is written on the screen, row by row, the blank ones as "".
	rows := func(f, w, h int) []string {
		t.Helper()
		lines, _ := s(f, w, h)
		var out []string
		for i, l := range Pad2(lines, w, h) {
			if Width(l) != w {
				t.Errorf("%d × %d: row %d is %d wide", w, h, i, Width(l))
			}
			if strings.Contains(l, "\U0000F0C8") {
				t.Errorf("%d × %d: the mark is drawn", w, h)
			}
			out = append(out, strings.TrimSpace(StripStyles(l)))
		}
		return out
	}
	// text is the block of text: from its first written row to its last.
	text := func(rs []string) []string {
		first, last := -1, -1
		for i, r := range rs {
			if r != "" {
				if first < 0 {
					first = i
				}
				last = i
			}
		}
		if first < 0 {
			return nil
		}
		return rs[first : last+1]
	}
	join := func(rs []string) string { return strings.Join(rs, "|") }
	all := []string{"showcase", "0.0.0-dev", intro, "", "developed by", "vulcan.shen.2304@gmail.com", "", "Press any key to close"}

	for _, size := range [][2]int{{49, 40}, {120, 29}} {
		w, h := size[0], size[1]
		if got := text(rows(500, w, h)); join(got) != join(all) {
			t.Errorf("%d × %d: %q, want %q", w, h, got, all)
		}
		if rs := rows(500, w, h); rs[(h-len(all))/2] != "showcase" {
			t.Errorf("%d × %d: the text is not in the middle: %q", w, h, rs)
		}
		lines, _ := s(500, w, h)
		for _, l := range Pad2(lines, w, h) {
			if p := StripStyles(l); strings.TrimSpace(p) != "" && p != Center(strings.TrimSpace(p), w) {
				t.Errorf("%d × %d: a row not centred: %q", w, h, p)
			}
		}
	}

	// The two batches count from the opening: 400ms, then 500ms more.
	for f, want := range map[int]string{
		39: "",
		40: "showcase|0.0.0-dev|" + intro,
		89: "showcase|0.0.0-dev|" + intro,
		90: join(all),
	} {
		if got := join(text(rows(f, 120, 20))); got != want {
			t.Errorf("tick %d: %q, want %q", f, got, want)
		}
	}

	// Too narrow: wrapped between words, each piece centred.
	got := text(rows(500, 30, 40))
	wantNarrow := []string{"showcase", "0.0.0-dev", "Every component of terminu", "design, in one app", "",
		"developed by", "vulcan.shen.2304@gmail.com", "", "Press any key to close"}
	if join(got) != join(wantNarrow) {
		t.Errorf("30 wide: %q, want %q", got, wantNarrow)
	}

	// Too short: cut from the bottom, a group and the blank row above it.
	for h, want := range map[int][]string{
		8: all,
		7: all[:6],
		5: all[:3],
		2: all[:2],
	} {
		if got := text(rows(500, 120, h)); join(got) != join(want) {
			t.Errorf("%d rows: %q, want %q", h, got, want)
		}
	}
}

// TestS5SplashRefusesWideText: no row of the text is wider than the mark (50
// columns); the lib refuses one when it builds the splash (tdp S5, E8).
// TestPlaySplash: the app can play a splash of its own accord, any splash it
// has; any key closes it, as the app's own on V (tdp S3; kamonu KM8).
func TestPlaySplash(t *testing.T) {
	ta := withSplash(newTestApp(t))
	played := 0
	other := Splash(func(f, w, h int) ([]string, bool) {
		played++
		return []string{"another mark"}, true
	})
	ta.PlaySplash(other)
	if !ta.splashOn || !strings.Contains(ta.screenText(), "another mark") || played == 0 {
		t.Fatalf("the splash it plays is drawn: %q", ta.screenText())
	}
	ta.press("j")
	if ta.splashOn || ta.list.cursor != 0 {
		t.Error("a key closes it, and only that")
	}
	ta.press("V")
	if !ta.splashOn || strings.Contains(ta.screenText(), "another mark") {
		t.Error("V plays the app's own")
	}
}

// TestS5SplashKeepsTime: the reveal goes by the time since it opened, not by
// counting ticks that come late (hostu HU9).
func TestS5SplashKeepsTime(t *testing.T) {
	ta := newTestApp(t)
	ta.Splash, _ = showcaseSplash(t)
	ta.press("V")
	ta.stepSplash(splashMsg{ta.splashID})
	if ta.splashFrame != 1 {
		t.Errorf("a tick on time: frame %d", ta.splashFrame)
	}
	ta.splashStart = time.Now().Add(-time.Second)
	ta.stepSplash(splashMsg{ta.splashID})
	if ta.splashFrame < 99 {
		t.Errorf("a second in, frame %d", ta.splashFrame)
	}
}

// TestS5SplashOneRow: one row: the name only (tdp S5).
func TestS5SplashOneRow(t *testing.T) {
	s, _ := showcaseSplash(t)
	lines, _ := s(10000, 80, 1)
	if got := Pad2(lines, 80, 1); len(got) != 1 || strings.TrimSpace(StripStyles(got[0])) != "showcase" {
		t.Errorf("one row: %q", got)
	}
}

func TestS5SplashRefusesWideText(t *testing.T) {
	icon, err := os.ReadFile("examples/showcase/icon.svg")
	if err != nil {
		t.Fatal(err)
	}
	fits := strings.Repeat("x", 50)
	if _, err := NewSplash(icon, "showcase", "v1", fits); err != nil {
		t.Errorf("50 columns: %v", err)
	}
	for _, args := range [][3]string{
		{"showcase", "v1", fits + "x"},
		{fits + "x", "v1", "intro"},
		{"showcase", fits + "x", "intro"},
		{"showcase", "v1", strings.Repeat("中", 26)},
	} {
		if _, err := NewSplash(icon, args[0], args[1], args[2]); err == nil {
			t.Errorf("no error for a row wider than the mark: %q", args)
		}
	}
}
