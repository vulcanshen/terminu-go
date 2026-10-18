package terminu

import (
	"strings"
	"testing"
)

func withWidths(t *testing.T, icon, powerline int) {
	t.Helper()
	oi, op := iconCells, powerlineCells
	SetIconWidth(icon, powerline)
	t.Cleanup(func() { iconCells, powerlineCells = oi, op })
}

func TestE5WidthCountsIconsAndPowerline(t *testing.T) {
	cases := []struct {
		icon, pl int
		s        string
		want     int
	}{
		{1, 1, "a" + glyphMarked + "b", 3},
		{2, 1, "a" + glyphMarked + "b", 4},
		{2, 1, glyphHard + glyphSlantLeft, 2},
		{2, 2, glyphHard + glyphSlantLeft, 4},
		{2, 1, Fg(Blue).Render("中文") + glyphMenu, 6},
		{1, 1, "x\nlonger", 6},
	}
	for _, c := range cases {
		withWidths(t, c.icon, c.pl)
		if got := Width(c.s); got != c.want {
			t.Errorf("icon %d, powerline %d: Width(%q) = %d, want %d", c.icon, c.pl, c.s, got, c.want)
		}
	}
}

func TestCutKeepsWidthAndStyles(t *testing.T) {
	withWidths(t, 2, 1)
	s := Fg(Red).Render("ab" + glyphMarked + "cd")
	if got := Width(Cut(s, 0, 3)); got != 3 {
		t.Errorf("cutting an icon in half: width %d, want 3", got)
	}
	if got := Cut(s, 1, 2); !strings.Contains(got, "38;2;243;139;168") {
		t.Errorf("Cut dropped the colour: %q", got)
	}
	if got := Width(Pad("中", 5)); got != 5 {
		t.Errorf("Pad: %d", got)
	}
}

func TestTruncate(t *testing.T) {
	withWidths(t, 1, 1)
	cases := []struct {
		s    string
		w    int
		want string
	}{
		{"abcdef", 6, "abcdef"},
		{"abcdef", 4, "abc…"},
		{"中文字", 4, "中…"},
		{"abc", 1, "…"},
	}
	for _, c := range cases {
		if got := StripStyles(Truncate(c.s, c.w)); got != c.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", c.s, c.w, got, c.want)
		}
	}
	if got := StripStyles(TruncateLeft("/a/b/c/d", 5)); got != "…/c/d" {
		t.Errorf("TruncateLeft = %q", got)
	}
}

// TestE5CompositeOversized lays boxes bigger than the screen over it, every
// way: they are cut, the lines keep the screen's width, nothing panics.
func TestE5CompositeOversized(t *testing.T) {
	withWidths(t, 2, 1)
	base := blankLines(20, 5)
	big := blankLines(40, 10)
	for i := range big {
		big[i] = strings.Repeat(glyphMarked, 20)
	}
	for _, at := range [][2]int{{-5, -5}, {0, 0}, {10, 3}, {25, 8}} {
		out := Composite(append([]string(nil), base...), big, at[0], at[1], 20)
		for i, l := range out {
			if Width(l) != 20 {
				t.Errorf("at %v line %d: width %d", at, i, Width(l))
			}
		}
	}
}

func TestWrap(t *testing.T) {
	withWidths(t, 1, 1)
	got := Wrap("one two three four", 9)
	want := []string{"one two", "three", "four"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("Wrap = %q, want %q", got, want)
	}
	for _, l := range Wrap("abcdefghijklmnop", 5) {
		if Width(l) > 5 {
			t.Errorf("long word line %q wider than 5", l)
		}
	}
	// Components, wrapping: a wide character breaks on either side, a line
	// fills up first; a word is cut only when longer than a line; a wide
	// character is never cut in half.
	for _, c := range []struct {
		s    string
		w    int
		want string
	}{
		{"\U00009019\U0000662F\U00004E00\U00006BB5\U00004E2D\U00006587\U00007684\U000053E5\U00005B50", 7,
			"\U00009019\U0000662F\U00004E00|\U00006BB5\U00004E2D\U00006587|\U00007684\U000053E5\U00005B50"},
		{"see \U00004E2D\U00006587\U00005B57 ok", 8, "see \U00004E2D\U00006587|\U00005B57 ok"},
		{"abc\U00004E2D", 4, "abc|\U00004E2D"},
		{"ab\U00004E2D\U00006587", 3, "ab|\U00004E2D|\U00006587"},
		{"ab abcdefghij", 5, "ab|abcde|fghij"},
	} {
		if got := strings.Join(Wrap(c.s, c.w), "|"); got != c.want {
			t.Errorf("Wrap(%q, %d) = %q, want %q", c.s, c.w, got, c.want)
		}
	}
}

// TestFoldAt: the textarea and the viewer keep every character when a line
// folds; it breaks after a space or on either side of a wide character, and
// inside a word only when the word is longer than a row (components,
// wrapping).
func TestFoldAt(t *testing.T) {
	withWidths(t, 1, 1)
	fold := func(s string, w int) string {
		var chars []string
		for _, r := range s {
			chars = append(chars, string(r))
		}
		cells, space := make([]int, len(chars)), make([]bool, len(chars))
		for i, c := range chars {
			cells[i], space[i] = Width(c), c == " " || c == "\t"
			if c == "\t" {
				cells[i] = 4
			}
		}
		starts := append(foldAt(cells, space, w), len(chars))
		var rows []string
		for k := range starts[:len(starts)-1] {
			rows = append(rows, strings.Join(chars[starts[k]:starts[k+1]], ""))
		}
		return strings.Join(rows, "|")
	}
	for _, c := range []struct {
		s    string
		w    int
		want string
	}{
		{"hello world", 8, "hello |world"},
		{"hello world", 11, "hello world"},
		{"\U00004E2D\U00006587\U00005B57", 5, "\U00004E2D\U00006587|\U00005B57"},
		{"see \U00004E2D\U00006587", 6, "see \U00004E2D|\U00006587"},
		{"ab abcdefgh", 4, "ab |abcd|efgh"},
		{"ab\tcd", 6, "ab\t|cd"},
		{"hello world", 5, "hello| |world"}, // the space that does not fit starts the next row
	} {
		if got := fold(c.s, c.w); got != c.want {
			t.Errorf("fold(%q, %d) = %q, want %q", c.s, c.w, got, c.want)
		}
	}
}

func TestE2EnvNames(t *testing.T) {
	if got := EnvName("filu", "icon width"); got != "FILU__ICON_WIDTH" {
		t.Errorf("EnvName = %q", got)
	}
	if got := EnvName("kbu", "ALTERM_LOGIN_SHELL"); got != "KBU__ALTERM_LOGIN_SHELL" {
		t.Errorf("EnvName = %q", got)
	}
}

func TestE5ProbeAnswers(t *testing.T) {
	old := palette16
	t.Cleanup(func() { palette16 = old })
	reply := "\x1b[5;3R\x1b[5;2R\x1b]4;4;rgb:8989/b4b4/fafa\x1b\\\x1b]4;1;rgb:f3/8b/a8\x07\x1b[?62;22c"
	ans, done := parseAnswers(reply)
	if !done {
		t.Fatal("DA1 not seen")
	}
	if ans.icon != 2 || ans.powerline != 1 || ans.colours != 2 {
		t.Errorf("answers = %+v", ans)
	}
	if palette16[4] != (RGB{0x89, 0xb4, 0xfa}) || palette16[1] != (RGB{0xf3, 0x8b, 0xa8}) {
		t.Errorf("palette: %v %v", palette16[4], palette16[1])
	}
	if _, done := parseAnswers("\x1b[5;3R\x1b]4;4;rgb:8989"); done {
		t.Error("an unfinished reply counted as done")
	}
}

func TestE5WidthOrder(t *testing.T) {
	withWidths(t, 1, 1)
	t.Setenv("SHOWCASE__ICON_WIDTH", "2")
	t.Setenv(FamilyIconWidth, "1")
	t.Setenv("SHOWCASE__POWERLINE_WIDTH", "")
	t.Setenv(FamilyPowerlineWidth, "2")
	r := Probe("showcase")
	if r.IconCells != 2 || r.IconFrom != "SHOWCASE__ICON_WIDTH" {
		t.Errorf("app variable first: %+v", r)
	}
	if r.PowerlineCells != 2 || r.PowerlineFrom != FamilyPowerlineWidth || PowerlineWidth() != 2 {
		t.Errorf("powerline has its own variables (E5): %+v", r)
	}
	t.Setenv("SHOWCASE__ICON_WIDTH", "")
	t.Setenv(FamilyIconWidth, "2")
	t.Setenv("SHOWCASE__POWERLINE_WIDTH", "1")
	r = Probe("showcase")
	if r.IconCells != 2 || r.IconFrom != FamilyIconWidth {
		t.Errorf("then the family's: %+v", r)
	}
	if r.PowerlineCells != 1 || r.PowerlineFrom != "SHOWCASE__POWERLINE_WIDTH" {
		t.Errorf("the app's powerline variable first: %+v", r)
	}
}
