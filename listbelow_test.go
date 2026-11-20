package terminu

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// belowList is a list of commits, some taking rows under their first: c1 a
// row of refs under DESCRIPTION, Green; c4 two, the second's text in the
// first column running on to the row's end.
func belowList() *List {
	l := &List{Columns: []Column{{Title: "Commit"}, {Title: "Description", Shrink: true}}, Marks: true, Filterable: true}
	var rows []Row
	for i := range 10 {
		rows = append(rows, Row{ID: fmt.Sprint(i), Cells: []string{fmt.Sprintf("c%d", i), fmt.Sprintf("change %d", i)}})
	}
	rows[1].Below = []BelowRow{{Cells: []string{"", "feat, v1.2"}, Styled: [][]Span{nil, {{Text: "feat", Style: Fg(Green)}, {Text: ", v1.2", Style: Fg(Peach)}}}}}
	rows[4].Below = []BelowRow{{Cells: []string{"", "main"}}, {Cells: []string{"release-x is a long line that runs on past its column"}}}
	l.SetRows(rows)
	return l
}

// lineWith is the first of lines whose text has s, and the cell s starts at.
func lineWith(lines []string, s string) (string, int) {
	for _, l := range lines {
		if p := StripStyles(l); strings.Contains(p, s) {
			return l, Width(p[:strings.Index(p, s)])
		}
	}
	return "", -1
}

// TestListSeveralRows: an item may take rows under its first, as many as it
// needs; the first holds the columns, the header and the tick; what the rows
// below hold sits under the columns, the last cell running on; the cursor is
// drawn on the first row only, with its arrow; the other rows keep their own
// background and colours, a Subtext1 bar down the inside's first cell beside
// them and the item's last row underlined across the inside in Subtext1, no
// Surface1 (tdp panel.5, panel-181, from gittu).
func TestListSeveralRows(t *testing.T) {
	l := belowList()
	l.ToggleMark() // c0
	l.Key("j", tea.KeyMsg{})
	l.ToggleMark() // c1, the cursor on it
	lines := l.Draw(50, 14, true)
	green := sgrRGB(true, Green.RGB)
	bar := Fg(Subtext1).Render(glyphBar)
	under := "\x1b[4;" + sgrUnderline(Subtext1.RGB) + "m"
	first, at := lineWith(lines, "change 1")
	below, col := lineWith(lines, "feat, v1.2")
	if first == "" || below == "" || col != at {
		t.Fatalf("the refs are not under DESCRIPTION: %d, %d:\n%s", col, at, strings.Join(lines, "\n"))
	}
	if !strings.Contains(first, glyphHard) || strings.Contains(below, glyphHard) {
		t.Error("the arrow is not on the first row only")
	}
	if !strings.HasPrefix(StripStyles(below), glyphBar) || !strings.Contains(below, bar) {
		t.Errorf("no Subtext1 bar down the first cell: %q", below)
	}
	if !strings.HasPrefix(below, under) || !strings.Contains(below, green) {
		t.Errorf("the item's last row is not underlined in Subtext1, its colours kept: %q", below)
	}
	if strings.Contains(below, sgrRGB(false, Surface1.RGB)) {
		t.Errorf("the row below still has Surface1: %q", below)
	}
	if w := Width(below); w != 50 {
		t.Errorf("the row below is %d wide, not the whole width", w)
	}
	if strings.Contains(StripStyles(below), glyphMarked) || !strings.Contains(StripStyles(first), glyphMarked) {
		t.Error("the tick is not on the first row only")
	}
	if header, _ := lineWith(lines, "DESCRIPTION"); header == "" {
		t.Error("no header")
	}

	// Two rows below (c4): the bar beside both, the underline on the last only.
	l.Key("j", tea.KeyMsg{})
	l.Key("j", tea.KeyMsg{})
	l.Key("j", tea.KeyMsg{})
	lines = l.Draw(50, 14, true)
	mid, _ := lineWith(lines, "main")
	last, _ := lineWith(lines, "release-x")
	if !strings.HasPrefix(StripStyles(mid), glyphBar) || strings.Contains(mid, under) {
		t.Errorf("the middle row: a bar, no underline: %q", mid)
	}
	if !strings.HasPrefix(StripStyles(last), glyphBar) || !strings.HasPrefix(last, under) {
		t.Errorf("the last row: a bar and the underline: %q", last)
	}
	// The last cell with text runs on past its column.
	if !strings.Contains(StripStyles(last), "runs on past") {
		t.Errorf("the last cell is cut at its column: %q", StripStyles(last))
	}

	// Off the cursor: no marks.
	if below, _ = lineWith(lines, "feat, v1.2"); strings.Contains(StripStyles(below), glyphBar) || strings.Contains(below, "\x1b[4;") {
		t.Errorf("a row below off the cursor is marked: %q", below)
	}

	// Typing in the filter dims the cursor once, its marks too (tdp P6).
	l.Key("/", tea.KeyMsg{})
	lines = l.Draw(50, 14, true)
	if last, _ = lineWith(lines, "release-x"); !strings.Contains(last, Fg(DimColor(Subtext1)).Render(glyphBar)) {
		t.Errorf("the marks are not dimmed while typing: %q", last)
	}
}

// TestListAnchorSeveralRows: an anchor of several rows (tdp panel.9,
// panel-196): its first row the Yellow anchor row; its other rows keep their
// colours with a Yellow bar beside them and the last underlined in Yellow;
// with the cursor on it, the first row is the cursor and the others keep the
// Yellow marks. A dimmed panel dims the underline as it dims the text.
func TestListAnchorSeveralRows(t *testing.T) {
	l := belowList()
	l.Anchor, l.inMode = "4", true // c4, two rows below
	bar := Fg(Yellow).Render(glyphBar)
	under := "\x1b[4;" + sgrUnderline(Yellow.RGB) + "m"
	check := func(why string) {
		t.Helper()
		lines := l.Draw(50, 14, true)
		first, _ := lineWith(lines, "change 4")
		mid, _ := lineWith(lines, "main")
		last, _ := lineWith(lines, "release-x")
		if !strings.Contains(first, sgrRGB(false, Yellow.RGB)) && !strings.Contains(first, sgrRGB(false, Subtext1.RGB)) {
			t.Errorf("%s: the first row is neither the anchor nor the cursor: %q", why, first)
		}
		if !strings.Contains(mid, bar) || strings.Contains(mid, under) {
			t.Errorf("%s: the middle row: a Yellow bar, no underline: %q", why, mid)
		}
		if !strings.Contains(last, bar) || !strings.HasPrefix(last, under) {
			t.Errorf("%s: the last row: a Yellow bar and underline: %q", why, last)
		}
		if strings.Contains(mid, sgrRGB(false, Yellow.RGB)) {
			t.Errorf("%s: a row below is filled Yellow: %q", why, mid)
		}
	}
	check("the cursor elsewhere")
	for range 4 {
		l.Key("j", tea.KeyMsg{})
	}
	if r, _ := l.Current(); r.ID != "4" {
		t.Fatalf("the cursor on %s", r.ID)
	}
	check("the cursor on it")
	first, _ := lineWith(l.Draw(50, 14, true), "change 4")
	if !strings.Contains(first, glyphHard) {
		t.Error("the cursor is not on the anchor's first row")
	}

	// Dimmed, the underline's colour dims too.
	dim := strings.Join(DimLines(l.Draw(50, 14, false)), "\n")
	if !strings.Contains(dim, sgrUnderline(DimRGB(Yellow.RGB))) || strings.Contains(dim, sgrUnderline(Yellow.RGB)) {
		t.Error("the underline is not dimmed with the panel")
	}
}

// TestListSeveralRowsMove: moving, N of M and scrolling count items: j|k one
// item, every movement on an item's first row, the item under the cursor
// in sight whole (panel-182, from gittu).
func TestListSeveralRowsMove(t *testing.T) {
	l := belowList()
	press := func(keys ...string) {
		for _, k := range keys {
			l.Key(k, tea.KeyMsg{})
		}
	}
	h := 6 // the header and five rows
	press("j", "j")
	if l.cursor != 2 {
		t.Fatalf("two j from c0: on %d, want c2", l.cursor)
	}
	press("j", "j") // c4, three rows
	lines := l.Draw(50, h, true)
	for _, s := range []string{"change 4", "main", "release-x"} {
		if l, _ := lineWith(lines, s); l == "" {
			t.Errorf("c4 is not in sight whole, no %q:\n%s", s, strings.Join(lines, "\n"))
		}
	}
	if l.Scroll() != "5 of 10" {
		t.Errorf("N of M counts items: %q", l.Scroll())
	}
	press("k")
	if l.cursor != 3 {
		t.Errorf("k: on %d, want c3", l.cursor)
	}
	press("gg", "d") // half a page, two rows: c1 has two
	if l.cursor != 1 {
		t.Errorf("d from c0: on %d, want c1", l.cursor)
	}
	press("d") // c2 and c3, a row each
	if l.cursor != 3 {
		t.Errorf("d from c1: on %d, want c3", l.cursor)
	}
	press("d", "u") // c3 to c4, then half a page back: c3 and c2, a row each
	if l.cursor != 2 {
		t.Errorf("u: on %d, want c2", l.cursor)
	}
	press("G")
	lines = l.Draw(50, h, true)
	if l.cursor != 9 || l.Scroll() != "10 of 10" {
		t.Errorf("G: on %d, %q", l.cursor, l.Scroll())
	}
	if last, _ := lineWith(lines, "change 9"); last == "" {
		t.Errorf("the last item is not in sight:\n%s", strings.Join(lines, "\n"))
	}
	// A view shorter than the item: its first row in sight.
	press("gg", "j", "j", "j", "j")
	lines = l.Draw(50, 3, true)
	if first, _ := lineWith(lines, "change 4"); first == "" {
		t.Errorf("an item taller than the view does not show its first row:\n%s", strings.Join(lines, "\n"))
	}
}

// TestListSeveralRowsFilter: the filter matches an item by the text of any
// of its rows (panel-183, from gittu).
func TestListSeveralRowsFilter(t *testing.T) {
	l := belowList()
	l.OpenFilter()
	for _, r := range "release" {
		l.Key(string(r), tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if len(l.view) != 1 || l.rows[l.view[0]].ID != "4" {
		t.Errorf("text on a row below did not find c4: %v", l.view)
	}
	l.ClearFilter()
	l.OpenFilter()
	for _, r := range "feat" {
		l.Key(string(r), tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if len(l.view) != 1 || l.rows[l.view[0]].ID != "1" {
		t.Errorf("a ref did not find c1: %v", l.view)
	}
}
