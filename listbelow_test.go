package terminu

import (
	"fmt"
	"math"
	"slices"
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

// cellBgs is each cell's background in a drawn line, "r;g;b", "" for none.
func cellBgs(line string) []string {
	var out []string
	bg := ""
	walk(line, func(seq string, cells int) {
		if cells > 0 {
			for range cells {
				out = append(out, bg)
			}
			return
		}
		if !strings.HasPrefix(seq, "\x1b[") || !strings.HasSuffix(seq, "m") {
			return
		}
		ps := strings.Split(seq[2:len(seq)-1], ";")
		for i := 0; i < len(ps); i++ {
			switch ps[i] {
			case "", "0", "49":
				bg = ""
			case "38", "58":
				if i+1 < len(ps) && ps[i+1] == "2" {
					i += 4
				} else {
					i += 2
				}
			case "48":
				if i+4 < len(ps) && ps[i+1] == "2" {
					bg = strings.Join(ps[i+2:i+5], ";")
					i += 4
				} else {
					bg = "?"
					i += 2
				}
			}
		}
	})
	return out
}

// wantFade is the fade the spec gives n cells from start into bg (tdp
// panel.5): lerp(start, bg, 1 - (1 - x)^3), x from 0 in the first cell to 1
// in the last, each channel rounded half up; "" for a cell within 2 of bg on
// every channel, left unpainted.
func wantFade(start, bg RGB, n int) []string {
	out := make([]string, n)
	for i := range n {
		t := 1 - math.Pow(1-float64(i)/float64(n-1), 3)
		var c [3]int
		near := true
		for k := range 3 {
			c[k] = int(math.Floor(float64(start[k]) + (float64(bg[k])-float64(start[k]))*t + 0.5))
			if d := c[k] - int(bg[k]); d < -2 || d > 2 {
				near = false
			}
		}
		if !near {
			out[i] = fmt.Sprintf("%d;%d;%d", c[0], c[1], c[2])
		}
	}
	return out
}

// The fades' starts the spec gives: Subtext1 and Yellow dimmed twice (tdp
// panel.5, panel.9).
var (
	cursorFade = RGB{0x2f, 0x31, 0x3c}
	anchorFade = RGB{0x3c, 0x37, 0x33}
)

// faded checks that row fades from start into the terminal's background
// across the inside's first n cells, the arrow's cells after them unpainted.
func faded(t *testing.T, why, row string, start RGB, n int) {
	t.Helper()
	bgs := cellBgs(row)
	if len(bgs) < n {
		t.Fatalf("%s: %d cells: %q", why, len(bgs), row)
	}
	if got, want := bgs[:n], wantFade(start, termBg, n); !slices.Equal(got, want) {
		t.Errorf("%s: the fade\n got %q\nwant %q", why, got, want)
	}
	for _, b := range bgs[n:] {
		if b != "" {
			t.Errorf("%s: the arrow's cell is painted: %q", why, row)
		}
	}
	for _, s := range []string{"\x1b[48;7;", "\u258c", "\x1b[4;", "58;2;"} {
		if strings.Contains(row, s) {
			t.Errorf("%s: %q on the row: %q", why, s, row)
		}
	}
}

// TestListSeveralRows: an item may take rows under its first, as many as it
// needs; the first holds the columns, the header and the tick; what the rows
// below hold sits under the columns, the last cell running on; the cursor is
// drawn on the first row only, with its arrow; the other rows keep their
// text's colours on a background fading out across the inside, from the
// first row's block colour dimmed twice (Subtext1: #2f313c) in its first
// cell, dropping fast, into the terminal's own background, a cell within 2
// of it unpainted, the arrow's cell too; no bar, no underline (tdp panel.5,
// panel-181, from gittu).
func TestListSeveralRows(t *testing.T) {
	answered(t, Base.RGB)
	l := belowList()
	l.ToggleMark() // c0
	l.Key("j", tea.KeyMsg{})
	l.ToggleMark() // c1, the cursor on it
	lines := l.Draw(50, 14, true)
	n := 50 - Width(glyphHard)
	first, at := lineWith(lines, "change 1")
	below, col := lineWith(lines, "feat, v1.2")
	if first == "" || below == "" || col != at {
		t.Fatalf("the refs are not under DESCRIPTION: %d, %d:\n%s", col, at, strings.Join(lines, "\n"))
	}
	if !strings.Contains(first, glyphHard) || strings.Contains(below, glyphHard) {
		t.Error("the arrow is not on the first row only")
	}
	if bgs := cellBgs(below); len(bgs) != 50 || bgs[0] != "47;49;60" {
		t.Errorf("the fade does not start from #2f313c in the inside's first cell: %q", below)
	}
	faded(t, "c1", below, cursorFade, n)
	if !strings.Contains(below, sgrRGB(true, Green.RGB)) {
		t.Errorf("the row below lost its colours: %q", below)
	}
	if strings.Contains(StripStyles(below), glyphMarked) || !strings.Contains(StripStyles(first), glyphMarked) {
		t.Error("the tick is not on the first row only")
	}
	if header, _ := lineWith(lines, "DESCRIPTION"); header == "" {
		t.Error("no header")
	}

	// Two rows below (c4): both fade the same.
	l.Key("j", tea.KeyMsg{})
	l.Key("j", tea.KeyMsg{})
	l.Key("j", tea.KeyMsg{})
	lines = l.Draw(50, 14, true)
	mid, _ := lineWith(lines, "main")
	last, _ := lineWith(lines, "release-x")
	faded(t, "c4's middle row", mid, cursorFade, n)
	faded(t, "c4's last row", last, cursorFade, n)
	// The last cell with text runs on past its column.
	if !strings.Contains(StripStyles(last), "runs on past") {
		t.Errorf("the last cell is cut at its column: %q", StripStyles(last))
	}

	// Off the cursor: no background at all.
	below, _ = lineWith(lines, "feat, v1.2")
	for _, b := range cellBgs(below) {
		if b != "" {
			t.Errorf("a row below off the cursor is painted: %q", below)
			break
		}
	}

	// Typing in the filter dims the cursor once, the fade's start with it (tdp P6).
	l.Key("/", tea.KeyMsg{})
	last, _ = lineWith(l.Draw(50, 14, true), "release-x")
	faded(t, "typing", last, DimRGB(cursorFade), n)
}

// TestListAnchorSeveralRows: an anchor of several rows (tdp panel.9,
// panel-196): its first row the Yellow anchor row; its other rows fade out
// as the cursor's do, from Yellow dimmed twice (#3c3733), their text keeping
// its colours; with the cursor on it, the first row is the cursor and the
// others keep the Yellow fade. Dimmed, the start dims and the fade still
// ends in the terminal's background.
func TestListAnchorSeveralRows(t *testing.T) {
	answered(t, Base.RGB)
	l := belowList()
	l.Anchor, l.inMode = "4", true // c4, two rows below
	n := 50 - Width(glyphHard)
	check := func(why string) {
		t.Helper()
		lines := l.Draw(50, 14, true)
		first, _ := lineWith(lines, "change 4")
		mid, _ := lineWith(lines, "main")
		last, _ := lineWith(lines, "release-x")
		if !strings.Contains(first, sgrRGB(false, Yellow.RGB)) && !strings.Contains(first, sgrRGB(false, Subtext1.RGB)) {
			t.Errorf("%s: the first row is neither the anchor nor the cursor: %q", why, first)
		}
		if bgs := cellBgs(mid); bgs[0] != "60;55;51" {
			t.Errorf("%s: the fade does not start from #3c3733: %q", why, mid)
		}
		faded(t, why+", the middle row", mid, anchorFade, n)
		faded(t, why+", the last row", last, anchorFade, n)
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

	// Dimmed (tdp T2): the start dims, the fade still ends in the background.
	mid, _ := lineWith(finishFades(DimLines(l.draw(50, 14, false, true))), "main")
	faded(t, "dimmed", mid, DimRGB(anchorFade), n)
}

// TestListFadeDimmed: on a screen, a fade's start dims with the rest and the
// fade still ends in the terminal's own background, on a terminal whose
// background is not Base too: without the focus (tdp T2) and under a popup
// (tdp F8), each dimming the start once more (tdp panel.5, panel-181).
func TestListFadeDimmed(t *testing.T) {
	answered(t, RGB{0x30, 0x34, 0x46})
	l := belowList()
	l.Key("j", tea.KeyMsg{}) // c1
	a := anySize(New("x"))
	a.AddScreen(&Screen{Name: "S", Panels: []*Panel{{Label: "Commits", Body: l}, {Label: "Other", Body: &Pager{}}},
		Layout: func(w, h int) []Rect { return []Rect{{0, 0, 60, h}, {60, 0, w - 60, h}} }})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	n := 58 - Width(glyphHard) // the inside of the 60 cells, the arrow's cells unpainted
	check := func(why string, start RGB) {
		t.Helper()
		row, _ := lineWith(a.Frame(), "feat, v1.2")
		bgs := cellBgs(row)
		if len(bgs) < 60 || bgs[0] != "" {
			t.Fatalf("%s: %q", why, row)
		}
		faded(t, why, Cut(row, 1, 59), start, n)
	}
	check("with the focus", cursorFade)
	a.Update(keyMsg("2"))
	check("without the focus", DimRGB(cursorFade))
	a.Confirm(Ask{Title: "Delete", Question: "Delete it?", Verb: "delete"}, func() tea.Cmd { return nil })
	check("under a popup", DimRGB(DimRGB(cursorFade)))
}

// answered is the terminal having answered OSC 11 with bg, for this test.
func answered(t *testing.T, bg RGB) {
	t.Helper()
	oldBg, oldKnown := termBg, termBgKnown
	t.Cleanup(func() { termBg, termBgKnown = oldBg, oldKnown })
	termBg, termBgKnown = bg, true
}

// TestFadeBackgroundOSC11: the terminal's background is asked for with the
// palette, first, ahead of the palette and DA1 (OSC 11), so a multiplexer
// holding answers behind a colour left unanswered does not make it miss the
// wait; its answer is where a fade ends (tdp color.2, color.4, panel.5); the
// lib's own terminal answers a program in it with that background, and
// nothing while it is not known.
func TestFadeBackgroundOSC11(t *testing.T) {
	for _, widths := range []bool{false, true} {
		q := probeQueries(widths)
		if at := strings.Index(q, "\x1b]11;?\x1b\\"); at < 0 || at > strings.Index(q, "\x1b]4;") || at > strings.Index(q, "\x1b[c") {
			t.Errorf("the background is not asked for ahead of the palette and DA1: %q", q)
		}
	}
	oldBg, oldKnown, oldPal := termBg, termBgKnown, palette16
	t.Cleanup(func() { termBg, termBgKnown, palette16 = oldBg, oldKnown, oldPal })
	termBg, termBgKnown = Base.RGB, false
	if _, done := parseAnswers("\x1b]4;1;rgb:f3/8b/a8\x07\x1b[?62;22c"); !done || termBgKnown {
		t.Errorf("no answer: the background is known, %v", termBg)
	}
	if r := string(vtAfter(10, 5, "\x1b]11;?\x1b\\").reply); r != "" {
		t.Errorf("not known: the lib's terminal answers %q", r)
	}
	for _, reply := range []string{
		"\x1b]11;rgb:3030/3434/4646\x1b\\\x1b]4;1;rgb:f3/8b/a8\x07\x1b[?62;22c",
		"\x1b]4;1;rgb:f3/8b/a8\x07\x1b]11;rgb:3030/3434/4646\x1b\\\x1b[?62;22c",
		"\x1b]11;rgb:30/34/46\x07\x1b[?62;22c",
	} {
		termBg, termBgKnown = Base.RGB, false
		if _, done := parseAnswers(reply); !done || !termBgKnown || termBg != (RGB{0x30, 0x34, 0x46}) {
			t.Errorf("%q: the background is %v, known %v", reply, termBg, termBgKnown)
		}
	}
	if _, done := parseAnswers("\x1b]11;rgb:3030/34"); done {
		t.Error("an unfinished reply counted as done")
	}
	if r := string(vtAfter(10, 5, "\x1b]11;?\x1b\\").reply); r != "\x1b]11;rgb:3030/3434/4646\x1b\\" {
		t.Errorf("the lib's terminal answers %q", r)
	}
}

// TestPanel200NoOSC11Flat: with no answer to OSC 11 there is no fade: the
// cursor's item's other rows take the fade's start flat across the whole
// inside, the arrow's cells too, up to the right border, on a terminal whose
// background is not Base (gittu's tmux display-popup, #11111a); dimmed, the
// start dims as before (tdp color.2, panel.5, T2, F8; panel-200; from
// gittu, the user).
func TestPanel200NoOSC11Flat(t *testing.T) {
	oldBg, oldKnown := termBg, termBgKnown
	t.Cleanup(func() { termBg, termBgKnown = oldBg, oldKnown })
	termBg, termBgKnown = Base.RGB, false
	l := belowList()
	l.Key("j", tea.KeyMsg{}) // c1
	a := anySize(New("x"))
	a.AddScreen(&Screen{Name: "S", Panels: []*Panel{{Label: "Commits", Body: l}, {Label: "Other", Body: &Pager{}}},
		Layout: func(w, h int) []Rect { return []Rect{{0, 0, 60, h}, {60, 0, w - 60, h}} }})
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	check := func(why string, start RGB) {
		t.Helper()
		row, _ := lineWith(a.Frame(), "feat, v1.2")
		bgs := cellBgs(row)
		want := fmt.Sprintf("%d;%d;%d", start[0], start[1], start[2])
		if len(bgs) < 60 || bgs[0] != "" || bgs[59] != "" {
			t.Fatalf("%s: the frame's cells: %q", why, row)
		}
		for i, b := range bgs[1:59] {
			if b != want {
				t.Errorf("%s: inside cell %d is %q, want %s flat to the right border: %q", why, i, b, want, row)
				break
			}
		}
	}
	check("with the focus", cursorFade)
	a.Update(keyMsg("2"))
	check("without the focus", DimRGB(cursorFade))
	a.Confirm(Ask{Title: "Delete", Question: "Delete it?", Verb: "delete"}, func() tea.Cmd { return nil })
	check("under a popup", DimRGB(DimRGB(cursorFade)))
	// Off the cursor, nothing; the anchor's rows flat in its own colour.
	if below, _ := lineWith(l.Draw(50, 14, true), "main"); slices.ContainsFunc(cellBgs(below), func(b string) bool { return b != "" }) {
		t.Errorf("a row below off the cursor is painted: %q", below)
	}
	l.Anchor, l.inMode = "4", true
	mid, _ := lineWith(l.Draw(50, 14, false), "main")
	for _, b := range cellBgs(mid) {
		if b != "60;55;51" {
			t.Errorf("the anchor's row is not #3c3733 flat across its 50 cells: %q", mid)
			break
		}
	}
	// With an answer, the arrow's cells past the fade stay unpainted, on the
	// narrowest fade too.
	termBgKnown = true
	if c, paint := fadeColor(cursorFade, 3, 2); paint {
		t.Errorf("a cell past a fade of 2 is painted %v", c)
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
