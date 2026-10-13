package terminu

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Tests of a viewer's line numbers (components/dialog/note).

// viewerRows is the top popup's text rows, styled: the margin rows and the
// frame's top and bottom left out.
func viewerRows(ta *testApp) []string {
	l := ta.top()
	rows := covDRows(ta, l)
	return rows[2 : l.h-2]
}

// numberedLines are n lines, "line 1" to "line n".
func numberedLines(n int) []string {
	var out []string
	for i := range n {
		out = append(out, fmt.Sprintf("line %d", i+1))
	}
	return out
}

// TestViewerLineNumbers: each line's first row has its number, right-aligned
// in a column at least three digits wide with a space after it, Overlay0; a
// line folded onto more rows leaves the column blank there; the column is as
// wide as the line count needs and stays so while it scrolls; in the
// selection mode the cursor's line has its number in Text.
func TestViewerLineNumbers(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(80, 24)
	lines := numberedLines(42)
	lines[5] = strings.Repeat("word ", 30) // 150 cells: folds onto a second and a third row
	v := &Viewer{Title: "YAML", Lines: lines, Select: &Selection{Key: "v", Name: "Visual"}}
	ta.Viewer(v)
	ta.settle()
	rows := viewerRows(ta)
	plain := covDPlain(rows)
	if !strings.HasPrefix(plain[0], "┃   1 line 1") || !strings.HasPrefix(plain[4], "┃   5 line 5") {
		t.Errorf("a three-digit column, one cell from the frame, a space after it:\n%s\n%s", plain[0], plain[4])
	}
	if !strings.HasPrefix(plain[5], "┃   6 word") || !strings.HasPrefix(plain[6], "┃     word") ||
		!strings.HasPrefix(plain[7], "┃     ") || !strings.HasPrefix(plain[8], "┃   7 line 7") {
		t.Errorf("a folded line leaves the column blank on the rows under its first:\n%s", strings.Join(plain[5:9], "\n"))
	}
	number := func(row string) Color {
		t.Helper()
		for _, c := range covCCells(row) {
			if c.Col == 4 {
				return c.St.Fg
			}
		}
		t.Fatalf("no cell 4 in %q", StripStyles(row))
		return Color{}
	}
	if got := number(rows[0]); got != Overlay0 {
		t.Errorf("the numbers are in %v, not Overlay0", got)
	}
	ta.press("v", "j")
	rows = viewerRows(ta)
	if number(rows[1]) != Text || number(rows[0]) != Overlay0 {
		t.Errorf("in the selection mode the cursor's line has its number in Text, the others Overlay0: %v, %v", number(rows[1]), number(rows[0]))
	}
	ta.press("Esc", "Esc")

	// 1200 lines: four digits from the first line on, and still four at the end.
	v = &Viewer{Title: "Log", Lines: numberedLines(1200)}
	ta.Viewer(v)
	ta.settle()
	if p := covDPlain(viewerRows(ta)); !strings.HasPrefix(p[0], "┃    1 line 1") {
		t.Errorf("the column takes the line count's four digits from the start: %q", p[0])
	}
	ta.press("G")
	if p := covDPlain(viewerRows(ta)); !strings.HasPrefix(p[len(p)-1], "┃ 1200 line 1200") {
		t.Errorf("at the end: %q", p[len(p)-1])
	}
}

// TestViewerLineNumbersStream: lines that keep arriving start in a
// three-digit column, which widens only past 999.
func TestViewerLineNumbersStream(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(80, 24)
	v := &Viewer{Title: "Log", Loading: true}
	ta.Viewer(v)
	ta.settle()
	for _, n := range []int{1, 99, 999} {
		v.Lines = numberedLines(n)
		ta.growLoading() // a loading popup grows with its lines (Rules F7)
		if p := covDPlain(viewerRows(ta)); !strings.HasPrefix(p[0], "┃   1 line 1") {
			t.Errorf("%d lines: %q, not a three-digit column", n, p[0])
		}
	}
	v.Lines = numberedLines(1000)
	ta.growLoading()
	if p := covDPlain(viewerRows(ta)); !strings.HasPrefix(p[0], "┃    1 line 1") {
		t.Errorf("the 1000th line widens the column: %q", p[0])
	}
}

// TestViewerCopyNoNumbers: what the selection copies is the text alone.
func TestViewerCopyNoNumbers(t *testing.T) {
	ta := newTestApp(t)
	got := ""
	v := &Viewer{Title: "YAML", Lines: []string{"kind: Pod", "metadata:", "  name: nginx"},
		Select: &Selection{Key: "v", Name: "Visual", Copy: func(s string) tea.Cmd { got = s; return nil }}}
	ta.Viewer(v)
	ta.settle()
	ta.press("v", "v", "j", "$", "y")
	if got != "kind: Pod\nmetadata:" {
		t.Errorf("copied %q", got)
	}
}

// TestViewerFoldedScroll: the scroll position counts lines, the numbers on
// the left, however many rows they fold onto; G shows the last line whole.
func TestViewerFoldedScroll(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(80, 14) // 8 rows of text
	var lines []string
	for i := range 10 {
		lines = append(lines, fmt.Sprintf("%02d ", i+1)+strings.Repeat("x", 97)) // 100 cells: two rows each
	}
	v := &Viewer{Title: "Log", Lines: lines, Select: &Selection{Key: "v", Name: "Visual"}}
	ta.Viewer(v)
	ta.settle()
	if !strings.Contains(ta.screenText(), " 1-4 of 10 ") {
		t.Errorf("four lines in sight on eight rows:\n%s", ta.screenText())
	}
	ta.press("G")
	if !strings.Contains(ta.screenText(), " 7-10 of 10 ") || v.Top() != 6 {
		t.Errorf("G: top %d\n%s", v.Top(), ta.screenText())
	}
	ta.press("gg", "v", "j", "j", "j", "j")
	if v.line != 4 || v.Top() != 1 || !strings.Contains(ta.screenText(), " 5 of 10 ") {
		t.Errorf("the view follows the cursor a line at a time: line %d, top %d\n%s", v.line, v.Top(), ta.screenText())
	}
}
