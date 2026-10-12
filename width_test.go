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
	if r := Probe("showcase"); r.IconCells != 2 || r.From != "SHOWCASE__ICON_WIDTH" {
		t.Errorf("app variable first: %+v", r)
	}
	t.Setenv("SHOWCASE__ICON_WIDTH", "")
	t.Setenv(FamilyIconWidth, "2")
	if r := Probe("showcase"); r.IconCells != 2 || r.From != FamilyIconWidth {
		t.Errorf("then the family's: %+v", r)
	}
}
