package terminu

import (
	"slices"
	"strings"
	"testing"
)

const testFlag = "\U000025CF" // ●

// flagRows are three rows, the one with flag set drawing a Green ●.
func flagRows(flag string) []Row {
	rows := []Row{{ID: "home", Cells: []string{"home"}}, {ID: "work", Cells: []string{"work"}}, {ID: "lab", Cells: []string{"lab"}}}
	for i := range rows {
		if rows[i].ID == flag {
			rows[i].Flag = Span{Text: testFlag, Style: Fg(Green)}
		}
	}
	return rows
}

// TestPanelFlags: a list that cannot be marked may draw one flag a row in
// its mark column, the app's glyph in the app's colour; every row keeps the
// column, blank with no flag, so the text starts in the same cell and does
// not move when the flag goes to another row; on the cursor it is on the
// block as a tick is (tdp panel.7, panel-137, L2).
func TestPanelFlags(t *testing.T) {
	withWidths(t, 1, 1)
	l := &List{Flags: true}
	l.SetRows(flagRows("work"))
	out := l.Draw(24, 6, true)
	got := plainLines(out)[:3]
	want := []string{"   home" + strings.Repeat(" ", 24-7-Width(glyphHard)) + glyphHard, " " + testFlag + " work", "   lab"}
	if !slices.Equal(got, want) {
		t.Errorf("rows:\n%q\nwant\n%q", got, want)
	}
	if !strings.Contains(out[1], Fg(Green).Render(testFlag)) {
		t.Errorf("the flag is not in its colour: %q", out[1])
	}
	l.Key("j", keyMsg("j"))
	if out := l.Draw(24, 6, true); strings.Contains(out[1], Fg(Green).Render(testFlag)) || !strings.HasPrefix(StripStyles(out[1]), " "+testFlag+" work") {
		t.Errorf("on the cursor: %q", out[1])
	}
	l.SetRows(flagRows("lab"))
	got = plainLines(l.Draw(24, 6, true))[:3]
	if !strings.HasPrefix(got[1], "   work") || got[2] != " "+testFlag+" lab" {
		t.Errorf("the flag moved, the text moved too: %q", got)
	}
}

// TestPanelFlagsNotWithMarks: a list that can be marked keeps its mark
// column for the tick: a flag is not drawn there, and Marks with Flags is a
// mistake AddScreen refuses (tdp panel.7, panel-138).
func TestPanelFlagsNotWithMarks(t *testing.T) {
	withWidths(t, 1, 1)
	l := &List{Marks: true, Flags: true}
	l.SetRows(flagRows("home"))
	if text := strings.Join(plainLines(l.Draw(24, 6, true)), "\n"); strings.Contains(text, testFlag) {
		t.Errorf("a flag in a list that can be marked:\n%s", text)
	}
	defer func() {
		if recover() == nil {
			t.Error("AddScreen took a list with Marks and Flags")
		}
	}()
	New("test").AddScreen(&Screen{Name: "S", Panels: []*Panel{{Label: "L", Body: l}}})
}

// TestPanelListGroupsFlags: in a list in groups a flag goes where a tick
// does, in the two cells the items are further in: one cell from the frame,
// a space, the text three cells in; a group title keeps no mark column
// (tdp panel.11, panel-86, panel-87).
func TestPanelListGroupsFlags(t *testing.T) {
	withWidths(t, 1, 1)
	l := groupList()
	rows := slices.Clone(l.rows)
	rows[1].Flag = Span{Text: testFlag, Style: Fg(Green)}
	l.SetRows(rows)
	l.Flags = true
	got := plainLines(l.Draw(24, 10, true))[:5]
	want := []string{" WORKLOADS", "   pods" + strings.Repeat(" ", 24-7-Width(glyphHard)) + glyphHard, " " + testFlag + " deployments", "   statefulsets", " NETWORK"}
	if !slices.Equal(got, want) {
		t.Errorf("rows:\n%q\nwant\n%q", got, want)
	}
}

// TestPanelFlagsDrag: the row held in a Drag mode shows the swap glyph in
// its mark column, and its flag again once it is put down (tdp panel.8).
func TestPanelFlagsDrag(t *testing.T) {
	withWidths(t, 1, 1)
	l := &List{Flags: true, Draggable: true}
	l.SetRows(flagRows("home"))
	l.Hold()
	if got := plainLines(l.Draw(24, 6, true)); !strings.HasPrefix(got[0], " "+glyphSwap+" home") {
		t.Errorf("held: %q", got[0])
	}
	l.MoveHeld(true)
	l.Drop()
	if got := plainLines(l.Draw(24, 6, true)); !strings.HasPrefix(got[1], " "+testFlag+" home") || !strings.HasPrefix(got[0], "   work") {
		t.Errorf("put down: %q", got[:2])
	}
}
