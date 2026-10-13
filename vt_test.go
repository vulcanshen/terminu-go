package terminu

import (
	"strings"
	"testing"
)

func vtAfter(w, h int, out string) *vt {
	t := newVT(w, h)
	t.write([]byte(out))
	return t
}

func TestVTPrintAndWrap(t *testing.T) {
	v := vtAfter(5, 3, "abcdefg\r\nxy")
	if got := strings.Join(v.text(), "|"); got != "abcde|fg|xy" {
		t.Errorf("wrap: %q", got)
	}
	// Writing the last column holds the wrap until the next character.
	v = vtAfter(5, 2, "abcde\r\nz")
	if got := strings.Join(v.text(), "|"); got != "abcde|z" {
		t.Errorf("a full line then CR LF: %q", got)
	}
}

func TestVTWideCharacters(t *testing.T) {
	withWidths(t, 2, 1)
	v := vtAfter(6, 2, "中文"+glyphFolder+"x")
	if got := strings.Join(v.text(), "|"); got != "中文"+glyphFolder+"|x" {
		t.Errorf("CJK and a two-cell icon: %q", got)
	}
	if v.x != 1 || v.y != 1 {
		t.Errorf("cursor at %d,%d", v.x, v.y)
	}
	line := renderCells(v.cells[0], 6, -1, Style{})
	if Width(line) != 6 {
		t.Errorf("a rendered line is %d wide", Width(line))
	}
	// Combining marks join the character before.
	v = vtAfter(4, 1, "e\U00000301x")
	if got := v.text()[0]; got != "e\U00000301x" {
		t.Errorf("combining: %q", got)
	}
}

func TestVTCursorAndErase(t *testing.T) {
	v := vtAfter(6, 3, "aaaaaa\r\nbbbbbb\r\ncccccc\x1b[2;3H\x1b[K\x1b[3;1H\x1b[1K\x1b[1;4HX")
	if got := strings.Join(v.text(), "|"); got != "aaaXaa|bb| ccccc" {
		t.Errorf("CUP, EL: %q", got)
	}
	v = vtAfter(4, 3, "1111\r\n2222\r\n3333\x1b[2;1H\x1b[J")
	if got := strings.Join(v.text(), "|"); got != "1111||" {
		t.Errorf("ED: %q", got)
	}
}

func TestVTScrollRegionAndHistory(t *testing.T) {
	v := vtAfter(3, 3, "a\r\nb\r\nc\r\nd")
	if got := strings.Join(v.text(), "|"); got != "b|c|d" || len(v.history) != 1 {
		t.Errorf("scroll: %q, history %d", got, len(v.history))
	}
	// A region: only lines 2–3 move.
	v = vtAfter(3, 3, "a\r\nb\r\nc\x1b[2;3r\x1b[3;1H\nd")
	if got := strings.Join(v.text(), "|"); got != "a|c|d" {
		t.Errorf("region: %q", got)
	}
	lines := v.render(0, false, Style{})
	if len(lines) != 3 {
		t.Fatal(len(lines))
	}
}

func TestVTAltScreen(t *testing.T) {
	v := vtAfter(4, 2, "main\x1b[?1049h\x1b[Halt")
	if v.text()[0] != "alt" {
		t.Errorf("alt screen: %q", v.text())
	}
	v.write([]byte("\x1b[?1049l"))
	if v.text()[0] != "main" {
		t.Errorf("back to main: %q", v.text())
	}
}

func TestVTAnswers(t *testing.T) {
	v := vtAfter(10, 5, "\x1b[3;4H\x1b[6n\x1b[c\x1b]4;1;?\x1b\\")
	r := string(v.reply)
	if !strings.HasPrefix(r, "\x1b[3;4R\x1b[?62;22c\x1b]4;1;rgb:") {
		t.Errorf("replies %q", r)
	}
}

func TestVTSplitSequences(t *testing.T) {
	v := newVT(10, 2)
	for _, part := range []string{"\x1b", "[3", "1m", "re", "\xe4\xb8", "\xad"} {
		v.write([]byte(part))
	}
	if v.text()[0] != "re中" {
		t.Errorf("text %q", v.text()[0])
	}
	if c := v.cells[0][0]; c.attr.fg != (vtColor{kind: 1, idx: 1}) {
		t.Errorf("SGR split across writes: %+v", c.attr)
	}
}

func TestVTColours(t *testing.T) {
	v := vtAfter(10, 1, "\x1b[38;2;1;2;3;48;5;196mA\x1b[38:2::4:5:6mB\x1b[0;7mC")
	line := renderCells(v.cells[0], 10, -1, Style{})
	for _, want := range []string{"38;2;1;2;3", "38;2;4;5;6", "48;2;255;0;0"} {
		if !strings.Contains(line, want) {
			t.Errorf("%q not in %q", want, line)
		}
	}
	if c := v.cells[0][2]; !c.attr.reverse {
		t.Error("reverse lost")
	}
}

func TestVTResize(t *testing.T) {
	v := vtAfter(4, 4, "a\r\nb\r\nc\r\nd")
	v.resize(6, 2)
	if got := strings.Join(v.text(), "|"); got != "c|d" || v.y != 1 {
		t.Errorf("shrinking keeps the cursor's lines: %q at %d", got, v.y)
	}
}

func TestTerminalCursorBlock(t *testing.T) {
	// The program asks for a bar (DECSCUSR 6): not drawn; the cursor stays a
	// block in Text with Base on it (components/dialog/terminal).
	v := vtAfter(4, 1, "ab\x1b[6 q")
	if got := v.text()[0]; got != "ab" {
		t.Errorf("DECSCUSR left %q", got)
	}
	if line := v.render(0, true, Style{Fg: Base, Bg: Text})[0]; !strings.Contains(line, "48;2;205;214;244") || Width(line) != 4 {
		t.Errorf("no block cursor in Text: %q", line)
	}
}
