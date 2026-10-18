package terminu

import (
	"fmt"
	"slices"
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

// TestViewerTabs: a tab in a viewer's text is drawn as spaces, coloured or
// not, cursor on it or not, and every row stays its width (Rules L4); a copy
// keeps the tab. The showcase's main.go broke the frame with one.
func TestViewerTabs(t *testing.T) {
	for _, lang := range []string{"", "go"} {
		ta := newTestApp(t)
		got := ""
		v := &Viewer{Title: "File", Lang: lang, Lines: []string{"func main() {", "\tfmt.Println(\"x\")", "}"},
			Select: &Selection{Key: "v", Name: "Visual", Copy: func(s string) tea.Cmd { got = s; return nil }}}
		ta.Viewer(v)
		ta.settle()
		check := func(when string) {
			t.Helper()
			for i, l := range ta.Frame() {
				if strings.Contains(l, "\t") || Width(l) != ta.w {
					t.Fatalf("lang %q, %s: row %d is %d wide or has a tab: %q", lang, when, i, Width(l), StripStyles(l))
				}
			}
			if !strings.Contains(ta.screenText(), "2     fmt.Println") {
				t.Errorf("lang %q, %s: the tab is not four spaces:\n%s", lang, when, ta.screenText())
			}
		}
		check("drawn")
		ta.press("v", "j")
		check("the cursor on the tab")
		ta.press("v", "$", "y")
		if got != "\tfmt.Println(\"x\")" {
			t.Errorf("lang %q: copied %q, want the tab kept", lang, got)
		}
	}
}

// TestViewerControlChars: in a viewer and a file preview a tab is
// input_tab_width spaces, the \r of a CRLF line break is not drawn, and any
// other control character is its name in Red, never sent to the terminal:
// every row stays its width (Rules L4). A copy keeps the text as it is
// (components/dialog/note).
func TestViewerControlChars(t *testing.T) {
	for _, lang := range []string{"", "go"} {
		ta := newTestApp(t)
		ta.InputTabWidth = 2
		got := ""
		ta.Viewer(&Viewer{Title: "File", Lang: lang,
			Lines:  []string{"a := 1\r", "b\x07c\x7fd\re", "\tx", Fg(Red).Render("ERROR") + " y"},
			Select: &Selection{Key: "v", Name: "Visual", Copy: func(s string) tea.Cmd { got = s; return nil }}})
		ta.settle()
		for i, l := range ta.Frame() {
			if Width(l) != ta.w || strings.ContainsAny(StripStyles(l), "\r\x07\x7f\t") {
				t.Fatalf("lang %q: row %d is %d wide or has a control character: %q", lang, i, Width(l), StripStyles(l))
			}
		}
		text := ta.screenText()
		if !strings.Contains(text, " 1 a := 1 ") || strings.Contains(text, `1\r`) {
			t.Errorf("lang %q: the \\r of a CRLF line break is drawn:\n%s", lang, text)
		}
		if !strings.Contains(text, `b\x07c\x7fd\re`) || !strings.Contains(text, " 3   x") {
			t.Errorf("lang %q: names, a tab of input_tab_width spaces:\n%s", lang, text)
		}
		frame := strings.Join(ta.Frame(), "\n")
		for _, name := range []string{`\x07`, `\x7f`, `\r`} {
			if !strings.Contains(frame, Fg(Red).Render(name)) {
				t.Errorf("lang %q: %s is not Red", lang, name)
			}
		}
		ta.press("v", "v", "j", "y")
		if got != "a := 1\r\nb" {
			t.Errorf("lang %q: copied %q, want the CRLF kept", lang, got)
		}
	}
	for _, l := range highlight("crlf.go", "package main\r\n\tfunc main() {}\r\nx\x07\n", 10, 2) {
		if plain := StripStyles(l); strings.ContainsAny(plain, "\r\t\x07") || strings.Contains(plain, `\r`) {
			t.Errorf("a preview line: %q", plain)
		}
		if strings.Contains(StripStyles(l), `\x07`) && !strings.Contains(l, Fg(Red).Render(`\x07`)) {
			t.Errorf("a preview name is not Red: %q", l)
		}
	}
}

// TestViewerFoldedScroll: the scroll position counts lines, the numbers on
// the left, however many rows they fold onto; G shows the last line whole.
func TestViewerFoldedScroll(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(80, 14) // 8 rows of text
	var lines []string
	for i := range 10 {
		lines = append(lines, fmt.Sprintf("%02d", i+1)+strings.Repeat("x", 98)) // one word of 100 cells: two rows each
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

// TestViewerFoldsAtSpaces: a line longer than a row folds at a space, a
// word going to the next row whole (components, wrapping).
func TestViewerFoldsAtSpaces(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(80, 14)
	// 70 cells a row: cut by cells, the second row would start "s words".
	ta.Viewer(&Viewer{Title: "Log", Lines: []string{strings.Repeat("words ", 25)}})
	ta.settle()
	text := ta.screenText()
	if !strings.Contains(text, "┃   1 words") || !strings.Contains(text, "┃     words words") {
		t.Errorf("the line does not fold at a space:\n%s", text)
	}
}

// TestViewerTabGutterOnlyGrows: with tabs, the line numbers' column is as
// wide as the widest tab seen since the viewer opened: a tab with more lines
// widens it, going back to one with fewer keeps it; opening again starts
// from the tab it opens on (components/dialog/note).
func TestViewerTabGutterOnlyGrows(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(80, 24)
	small, big := numberedLines(5), numberedLines(2000)
	v := &Viewer{Title: "DevTools", Tabs: []string{"Console", "Network"}, Lines: small}
	v.OnTab = func(tab int) tea.Cmd {
		v.Lines = [][]string{small, big}[tab]
		return nil
	}
	first := func() string { return covDPlain(viewerRows(ta))[0] }
	ta.Viewer(v)
	ta.settle()
	if got := first(); !strings.HasPrefix(got, "┃   1 line 1") {
		t.Errorf("opened on 5 lines: %q, not three digits", got)
	}
	ta.press("l")
	if got := first(); !strings.HasPrefix(got, "┃    1 line 1") {
		t.Errorf("a tab of 2000 lines widens it: %q", got)
	}
	ta.press("h")
	if got := first(); !strings.HasPrefix(got, "┃    1 line 1") {
		t.Errorf("back on 5 lines it keeps the width: %q", got)
	}
	ta.press("Esc")
	ta.Viewer(v)
	ta.settle()
	if got := first(); !strings.HasPrefix(got, "┃   1 line 1") {
		t.Errorf("opened again on 5 lines: %q", got)
	}
}

// TestViewerScrollsRows: a folded viewer scrolls a drawn row at a time, as
// less does; the scroll position names the lines its first and last rows
// belong to; a line taller than the frame is read whole
// (components/dialog/note, components/layout/popup).
func TestViewerScrollsRows(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(80, 14) // 8 rows of text, 70 cells each
	lines := numberedLines(42)
	lines[5] = strings.Repeat("word ", 30) // line 6: three rows
	v := &Viewer{Title: "YAML", Lines: lines}
	ta.Viewer(v)
	ta.settle()
	ta.press("j", "j", "j", "j", "j")
	if p := covDPlain(viewerRows(ta)); v.Top() != 5 || !strings.HasPrefix(p[0], "┃   6 word") {
		t.Fatalf("five rows down, line 6 on top: top %d, %q", v.Top(), p[0])
	}
	ta.press("j")
	p := covDPlain(viewerRows(ta))
	if v.Top() != 5 || !strings.HasPrefix(p[0], "┃     word") || !strings.HasPrefix(p[2], "┃   7 line 7") {
		t.Errorf("j: one row on, the second row of line 6 on top:\n%s", strings.Join(p[:3], "\n"))
	}
	if !strings.Contains(ta.screenText(), " 6-12 of 42 ") {
		t.Errorf("the lines of the first and the last row:\n%s", ta.screenText())
	}
	ta.press("k")
	if p := covDPlain(viewerRows(ta)); !strings.HasPrefix(p[0], "┃   6 word") {
		t.Errorf("k: one row back: %q", p[0])
	}
	ta.press("d") // half a page: four rows
	if p := covDPlain(viewerRows(ta)); !strings.HasPrefix(p[0], "┃   8 line 8") {
		t.Errorf("d: four rows on: %q", p[0])
	}

	// A line taller than the frame: every row of it comes into sight.
	var words []string
	for i := range 400 {
		words = append(words, fmt.Sprintf("w%03d", i+1))
	}
	v = &Viewer{Title: "JSON", Lines: []string{"start", strings.Join(words, " "), "end"}}
	ta.Viewer(v)
	ta.settle()
	seen := ""
	for range 60 {
		seen += ta.screenText()
		ta.press("j")
	}
	for _, w := range []string{"w200", "w399", "end"} {
		if !strings.Contains(seen, w) {
			t.Errorf("%s never came into sight", w)
		}
	}
	ta.press("G")
	if text := ta.screenText(); !strings.Contains(text, "w400") || !strings.Contains(text, " 2-3 of 3 ") {
		t.Errorf("G: the last rows, of lines 2 and 3:\n%s", text)
	}
}

// TestViewerSyntaxColours: a viewer colours its lines as the file preview
// colours a text file (components/dialog/note): the language given by name,
// by file name or by content type; plain text in Text; the numbers in
// Overlay0; a selection Yellow with Base text, the syntax colours not shown
// in it.
func TestViewerSyntaxColours(t *testing.T) {
	text := []string{"kind: Pod", "# restarted", "spec:"}
	// fgs is the foreground of each cell of text, from cell from of row.
	fgs := func(row string, from, n int) []Color {
		var out []Color
		for _, c := range covCCells(row) {
			if c.Col >= from && len(out) < n {
				out = append(out, c.St.Fg)
			}
		}
		return out
	}
	preview := highlight("pod.yaml", strings.Join(text, "\n"), 10, 4)
	for _, lang := range []string{"yaml", "pod.yaml", "text/x-yaml; charset=utf-8"} {
		ta := newTestApp(t)
		ta.Viewer(&Viewer{Title: "YAML", Lines: text, Lang: lang})
		ta.settle()
		rows := viewerRows(ta)
		for i, l := range text {
			got, want := fgs(rows[i], 6, len(l)), fgs(preview[i], 2, len(l))
			if fmt.Sprint(got) != fmt.Sprint(want) {
				t.Errorf("%s, line %d: %v, the preview's %v", lang, i+1, got, want)
			}
		}
		if fgs(rows[0], 6, 1)[0] == Text {
			t.Errorf("%s: the key is not coloured", lang)
		}
		if fgs(rows[0], 4, 1)[0] != Overlay0 {
			t.Errorf("%s: the numbers are not Overlay0", lang)
		}
	}
	for _, lang := range []string{"", "notes.nothing-knows-this"} {
		ta := newTestApp(t)
		ta.Viewer(&Viewer{Title: "Log", Lines: []string{"started 08:14:07", "kind: Pod"}, Lang: lang})
		ta.settle()
		for i, row := range viewerRows(ta)[:2] {
			for _, c := range fgs(row, 6, 9) {
				if c != Text {
					t.Errorf("Lang %q, line %d: plain text in %v, not Text", lang, i+1, c)
					break
				}
			}
		}
	}
	// A line the app styled keeps its styles, the rest of it in Text.
	ta := newTestApp(t)
	ta.Viewer(&Viewer{Title: "Log", Lines: []string{"08:14 " + Fg(Red).Render("ERROR") + " gone"}})
	ta.settle()
	if got := fmt.Sprint(fgs(viewerRows(ta)[0], 6, 16)); got != fmt.Sprint(append(append(
		slices.Repeat([]Color{Text}, 6), slices.Repeat([]Color{Red}, 5)...), slices.Repeat([]Color{Text}, 5)...)) {
		t.Errorf("a styled line: %s", got)
	}
	// A selection: Yellow, Base text, whatever the language.
	ta = newTestApp(t)
	ta.Viewer(&Viewer{Title: "YAML", Lines: text, Lang: "yaml", Select: &Selection{Key: "v", Name: "Visual"}})
	ta.settle()
	ta.press("v", "v", "l", "l", "l")
	for _, c := range covCCells(viewerRows(ta)[0]) {
		if c.Col >= 6 && c.Col < 9 && (c.St.Fg != Base || c.St.Bg != Yellow) {
			t.Errorf("cell %d of the selection: %v on %v, not Base on Yellow", c.Col, c.St.Fg, c.St.Bg)
		}
	}
}
