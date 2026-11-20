package terminu

import (
	"slices"
	"strings"
	"testing"
)

// spreadList is a list whose columns at their widest are 4, 4, 20 and 10
// cells (the third a description that may shrink and fill), on a row 60 wide
// with 16 cells to spare.
func spreadList(s Spread) *List {
	l := &List{Columns: []Column{{Title: "Name"}, {Title: "Size", Number: true}, {Title: "Description", Shrink: true, Fill: true}, {Title: "Date", Drop: 1}},
		Spread: s}
	l.SetRows([]Row{
		{ID: "1", Cells: []string{"a", "7", "a short description", "2026-11-20"}},
		{ID: "2", Cells: []string{"bb", "1234", "another one, longer.", "2026-11-19"}},
	})
	return l
}

// TestListSpread: when every column already shows its widest value, the room
// left over goes where Spread says: Pack at the row's end; Fill to the Fill
// column, the columns after it at the right edge; Even to the narrowest
// columns first, a wider one kept, a number still right-aligned; Gaps to
// every gap alike up to six cells, the rest to the Fill column. With the room
// short, every Spread is Pack (a trial, from gittu).
func TestListSpread(t *testing.T) {
	cases := []struct {
		s    Spread
		want []int
		gap  int
	}{
		{SpreadPack, []int{4, 4, 20, 10}, 0},
		{SpreadFill, []int{4, 4, 36, 10}, 0},
		{SpreadEven, []int{12, 11, 20, 11}, 0},
		{SpreadGaps, []int{4, 4, 24, 10}, 4},
	}
	for _, c := range cases {
		l := spreadList(c.s)
		if got := l.widths(60); !slices.Equal(got, c.want) || l.gapMore != c.gap {
			t.Errorf("Spread %d: widths %v, gaps +%d; want %v, +%d", c.s, got, l.gapMore, c.want, c.gap)
		}
	}
	// Fill: the last column at the right edge of the row.
	l := spreadList(SpreadFill)
	if row := l.rowText(0, l.widths(60)); Width(row) != 60 || !strings.HasSuffix(row, "2026-11-20") {
		t.Errorf("Fill: %q", row)
	}
	// Even: a number right-aligned in its wider column.
	l = spreadList(SpreadEven)
	ws := l.widths(60)
	if row := l.rowText(0, ws); !strings.HasPrefix(StripStyles(row)[ws[0]+2:], strings.Repeat(" ", ws[1]-1)+"7") {
		t.Errorf("Even: the number is not right-aligned: %q", row)
	}
	// Gaps: six cells between the columns, header and rows alike.
	l = spreadList(SpreadGaps)
	ws = l.widths(60)
	if h := StripStyles(l.header(ws)); !strings.HasPrefix(h, "NAME"+strings.Repeat(" ", 6)+"SIZE") {
		t.Errorf("Gaps: the header's gap: %q", h)
	}
	l.rows[0].Below = []BelowRow{{Cells: []string{"", "", "refs"}}}
	if b := l.belowText(0, 0, ws, 60); strings.Index(b, "refs") != 4+6+4+6 || Width(b) != 60 {
		t.Errorf("Gaps: a row below the item is not under DESCRIPTION, or not the row's width: %q", b)
	}
	if row := l.rowText(1, ws); !strings.HasPrefix(row, "bb  "+strings.Repeat(" ", 6)+"1234") || Width(row) != 60 {
		t.Errorf("Gaps: the row's gaps: %q", row)
	}
	// No Fill column: Fill is Pack, Gaps leaves the rest at the row's end.
	for _, s := range []Spread{SpreadFill, SpreadGaps} {
		l := spreadList(s)
		l.Columns[2].Fill = false
		if got := l.widths(60); !slices.Equal(got, []int{4, 4, 20, 10}) {
			t.Errorf("Spread %d with no Fill column: %v", s, got)
		}
	}
	// A column gone for room: no Spread places what is left.
	dropped := spreadList(SpreadPack).widths(35)
	for _, c := range cases {
		if got := spreadList(c.s).widths(35); !slices.Equal(got, dropped) || got[3] >= 0 {
			t.Errorf("Spread %d with a column gone: %v, want %v", c.s, got, dropped)
		}
	}
	// Narrower again, the gaps are two cells as before.
	if l.widths(36); l.gapMore != 0 {
		t.Errorf("Gaps: narrower, the gaps are still +%d", l.gapMore)
	}
	// Short of room: every Spread is Pack, the description shrinking.
	pack := spreadList(SpreadPack).widths(36)
	for _, c := range cases {
		l := spreadList(c.s)
		if got := l.widths(36); !slices.Equal(got, pack) || l.gapMore != 0 {
			t.Errorf("Spread %d short of room: %v +%d, want %v", c.s, got, l.gapMore, pack)
		}
	}
}
