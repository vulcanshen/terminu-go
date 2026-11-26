package terminu

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestDiffLines: the shortest edit, line by line, the anchor's side first.
func TestDiffLines(t *testing.T) {
	ops := diffLines([]string{"a", "b", "c", "d"}, []string{"a", "c", "d", "e"})
	var got []string
	for _, o := range ops {
		got = append(got, fmt.Sprintf("%c%d,%d", o.op, o.a, o.b))
	}
	if want := " 0,0 -1,-1  2,1  3,2 +-1,3"; strings.Join(got, " ") != want {
		t.Errorf("ops %q, want %q", strings.Join(got, " "), want)
	}
	if len(diffLines(nil, []string{"x"})) != 1 || len(diffLines([]string{"x"}, nil)) != 1 {
		t.Error("one side empty")
	}
}

// compareApp is the test app with a Compare action on its pods: rows of the
// same age have the same text. Its Split is set, and does nothing.
func compareApp(t *testing.T) (*testApp, *bool) {
	ta := newTestApp(t)
	split := new(bool)
	ta.pods.Actions = append(ta.pods.Actions, ta.DiffModeAction(ta.pods, ta.list, DiffMode{Key: "C", Split: split,
		Can: func(r Row) bool { return r.ID != "2" },
		Text: func(r Row) (string, error) {
			var b strings.Builder
			for i := range 20 {
				fmt.Fprintf(&b, "line %d\n", i)
			}
			return "age: " + r.Cells[1] + "\n" + b.String() + "kind: pod\n", nil
		}}))
	return ta, split
}

// TestCompareMode: the compare key enters the mode, its row the anchor;
// the compare keys are off on the anchor and on a row that cannot be
// compared; Enter compares; the same text is an Info toast; Esc leaves, the
// anchor gone (tdp panel.9). The mode goes unrecorded in the App Log, its
// toasts in as every toast (tdp note.7, from kbu).
func TestCompareMode(t *testing.T) {
	ta, _ := compareApp(t)
	ta.press("C")
	if m := ta.pods.InMode(); m == nil || m.Name != "Diffing" || ta.list.Anchor != "0" {
		t.Fatalf("not in the compare mode: %v, anchor %q", m, ta.list.Anchor)
	}
	frame := strings.Join(ta.Frame(), "\n")
	if !strings.Contains(covCFooter(ta), "C|Enter:diff") || !strings.Contains(frame, Fg(Surface2).Render(":diff")) {
		t.Errorf("on the anchor the compare keys are off: %q", covCFooter(ta))
	}
	ta.press("Enter")
	if ta.top() != nil {
		t.Error("Enter on the anchor compared")
	}
	ta.press("j", "j") // pod-02 cannot be compared
	if ta.press("C"); ta.top() != nil {
		t.Error("compared a row that cannot be")
	}
	ta.press("Tab")
	if ta.Screen().Focused() != ta.pods || !strings.Contains(ta.screenText(), "Press [Esc] to leave Diffing first") {
		t.Error("Tab left the mode's panel")
	}
	ta.toasts, ta.inspectT = nil, nil
	ta.press("k", "Enter")
	v, ok := ta.topView().(*viewerView)
	if !ok {
		t.Fatalf("no compare popup: %T", ta.topView())
	}
	if text := ta.screenText(); !strings.Contains(text, "pod-00 ↔ pod-01") || !strings.Contains(text, "UNIFIED") {
		t.Errorf("the popup's title:\n%s", text)
	}
	if !slicesContain(v.v.Lines, Fg(Red).Render("-age: 0")) || !slicesContain(v.v.Lines, Fg(Green).Render("+age: 1")) {
		t.Errorf("UNIFIED: %q", v.v.Lines)
	}
	ta.press("Esc")
	if ta.pods.InMode() == nil || ta.list.Anchor != "0" {
		t.Error("closing the compare left the mode")
	}
	ta.press("G", "k", "k", "k", "k") // pod-35, the same age as pod-00
	if r, _ := ta.list.Current(); r.ID != "35" {
		t.Fatalf("at %s", r.ID)
	}
	ta.press("Enter")
	if ta.top() != nil || !strings.Contains(ta.screenText(), "No differences between pod-00 and pod-35") {
		t.Errorf("the same text:\n%s", ta.screenText())
	}
	ta.toasts, ta.inspectT = nil, nil
	if ta.press("g", "g"); ta.list.cursor != 0 {
		t.Errorf("gg in the mode: at %d", ta.list.cursor)
	}
	ta.press("Esc")
	if ta.pods.InMode() != nil || ta.list.Anchor != "" {
		t.Error("Esc did not leave the mode, or the anchor stayed")
	}
	var logged []string
	for _, e := range ta.logs.entries {
		logged = append(logged, e.text)
	}
	if want := []string{"Press [Esc] to leave Diffing first", "No differences between pod-00 and pod-35"}; fmt.Sprint(logged) != fmt.Sprint(want) {
		t.Errorf("logged %q, want %q", logged, want)
	}
}

func slicesContain(lines []string, s string) bool {
	for _, l := range lines {
		if strings.Contains(l, s) {
			return true
		}
	}
	return false
}

// TestCompareHunks: only the stretches that changed, 3 lines either side,
// an Overlay0 line between them (note-81).
func TestCompareHunks(t *testing.T) {
	x := strings.Split("a b c d e f g h i j k l m n o", " ")
	y := append([]string(nil), x...)
	y[1], y[13] = "B", "N"
	lines, _ := diffUnified(diffLines(x, y), x, y, 30)
	plain := make([]string, len(lines))
	for i, l := range lines {
		plain[i] = StripStyles(l)
	}
	if len(lines) != 6+1+6 || !strings.HasPrefix(plain[6], "───") || !strings.Contains(lines[6], Fg(Overlay0).Render(strings.Repeat("─", 30))) {
		t.Errorf("hunks:\n%s", strings.Join(plain, "\n"))
	}
	if plain[0] != " 1  1  a" || plain[1] != " 2    -b" || plain[2] != "    2 +B" {
		t.Errorf("numbers and signs: %q", plain[:3])
	}
}

// TestCompareSplit: SPLIT puts the anchor on the left, each half numbered,
// cut with …; every comparison opens on UNIFIED, the tab chosen not kept,
// Compare.Split doing nothing; under 96 columns SPLIT is off (note-83 to 85).
func TestCompareSplit(t *testing.T) {
	ta, split := compareApp(t)
	ta.press("C", "j", "Enter")
	v := ta.topView().(*viewerView).v
	if v.Tab != 0 {
		t.Fatalf("opened on tab %d, not UNIFIED", v.Tab)
	}
	if ta.press("l"); v.Tab != 1 || *split {
		t.Fatalf("l did not switch to SPLIT, or Split was set: tab %d, split %v", v.Tab, *split)
	}
	if !slicesContain(v.Lines, Fg(Red).Render("age: 0")) || !slicesContain(v.Lines, Fg(Green).Render("age: 1")) {
		t.Errorf("SPLIT: %q", v.Lines)
	}
	row := StripStyles(v.Lines[0])
	if !strings.HasPrefix(row, " 1 age: 0") || !strings.Contains(row, " 1 age: 1") {
		t.Errorf("each half numbered: %q", row)
	}
	// A line too long folds within its half, the short one across from it
	// filled with blank rows, both starting on the same row (note-83).
	got, at := diffSplit([]diffOp{{'-', 0, -1}, {'+', -1, 0}}, []string{strings.Repeat("x", 30)}, []string{"short"}, 30)
	plain := StripStyles(strings.Join(got, "\n"))
	if len(got) != 3 || fmt.Sprint(at) != "[0 0 0]" || strings.Count(plain, "x") != 30 || strings.Contains(plain, "…") {
		t.Fatalf("the long line folds into three rows: %q, ops %v", got, at)
	}
	for i, row := range got {
		right := StripStyles(row)[16:]
		// The rows a line folds onto leave its number's column blank.
		if Width(row) != 30 || i == 0 && !strings.HasPrefix(right, "1 short") || i > 0 && (strings.TrimSpace(right) != "" || !strings.HasPrefix(StripStyles(row), "  x")) {
			t.Errorf("row %d: %q", i, StripStyles(row))
		}
	}
	ta.press("Esc", "Enter")
	if v := ta.topView().(*viewerView).v; v.Tab != 0 {
		t.Error("SPLIT was kept: the next comparison did not open on UNIFIED")
	}
	ta.press("Esc")
	*split = true // the old way to open on SPLIT: nothing now
	ta.press("Enter")
	if v := ta.topView().(*viewerView).v; v.Tab != 0 {
		t.Error("Compare.Split opened it on SPLIT")
	}
	ta.press("Esc")
	ta.resize(90, 30)
	ta.press("Enter")
	v = ta.topView().(*viewerView).v
	if ta.press("l"); v.Tab != 0 {
		t.Errorf("under 96 columns: tab %d", v.Tab)
	}
	if !strings.Contains(strings.Join(ta.Frame(), "\n"), Style{Fg: Surface2, Bg: Mantle}.Render("SPLIT")) {
		t.Error("SPLIT is not drawn off")
	}
}

// TestCompareAnchorGone: the anchor no longer in the list ends the mode with
// a Warning toast (panel-145).
func TestCompareAnchorGone(t *testing.T) {
	ta, _ := compareApp(t)
	ta.press("C")
	rows := ta.list.Rows()
	ta.list.SetRows(rows[1:])
	ta.press("j")
	if ta.pods.InMode() != nil || ta.list.Anchor != "" || !strings.Contains(ta.screenText(), "Diffing ended: pod-00 is gone") {
		t.Errorf("the anchor gone:\n%s", ta.screenText())
	}
}

// TestModeKeyDisabled: a mode key that cannot be pressed now does nothing
// and is in the disabled colour in its key reference (tdp M6).
func TestModeKeyDisabled(t *testing.T) {
	ta := newTestApp(t)
	ran := false
	ta.pods.EnterMode(&Mode{Name: "Pick", Keys: []ModeKey{{Key: "p", Desc: "pick",
		Disabled: func() bool { return true }, Do: func(string) tea.Cmd { ran = true; return nil }}}})
	if ta.press("p"); ran {
		t.Error("a disabled mode key ran")
	}
	ta.press("?")
	if !strings.Contains(strings.Join(ta.Frame(), "\n"), Fg(Surface2).Render("pick")) {
		t.Error("the key reference does not show it off")
	}
}

// TestCompareResize: the compare lays itself out again as the terminal
// changes size: under 96 columns UNIFIED, back over it the SPLIT chosen
// while it is open, the stretch on top kept in sight (note-85).
func TestCompareResize(t *testing.T) {
	ta := newTestApp(t)
	split := new(bool)
	*split = true
	ta.pods.Actions = append(ta.pods.Actions, ta.DiffModeAction(ta.pods, ta.list, DiffMode{Key: "C", Split: split,
		Text: func(r Row) (string, error) {
			var b strings.Builder
			for i := range 61 {
				if i%15 == 0 { // a change every 15 lines: five stretches
					fmt.Fprintf(&b, "age %s at %d\n", r.Cells[1], i)
					continue
				}
				fmt.Fprintf(&b, "line %d\n", i)
			}
			return b.String(), nil
		}}))
	ta.resize(120, 17)
	ta.press("C", "j", "Enter", "l")
	v := ta.topView().(*viewerView).v
	if v.Tab != 1 {
		t.Fatalf("on tab %d, want SPLIT", v.Tab)
	}
	sep := -1
	for i, l := range v.Lines {
		if strings.HasPrefix(StripStyles(l), "───") {
			sep = i
		}
	}
	v.top = sep + 1 // the last stretch on top, as far as it scrolls
	ta.screenText()
	want := regexp.MustCompile(`line \d+`).FindString(StripStyles(v.Lines[v.top]))
	if want == "" || v.top == 0 {
		t.Fatalf("not scrolled: %d %q", v.top, StripStyles(v.Lines[v.top]))
	}
	ta.resize(90, 17)
	if v.Tab != 0 || !strings.Contains(StripStyles(v.Lines[v.top]), want) {
		t.Errorf("under 96: tab %d, on top %q, want %q", v.Tab, StripStyles(v.Lines[v.top]), want)
	}
	if !strings.Contains(ta.screenText(), want) {
		t.Errorf("%q is not in sight:\n%s", want, ta.screenText())
	}
	ta.resize(120, 17)
	if v.Tab != 1 || !strings.Contains(ta.screenText(), want) {
		t.Errorf("back over 96: tab %d, %q in sight:\n%s", v.Tab, want, ta.screenText())
	}
	for i, l := range ta.Frame() {
		if Width(l) != 120 {
			t.Errorf("row %d is %d wide", i, Width(l))
		}
	}
}

// TestHintTab: where h|l switch tabs the hint says h|l:tab, the app giving
// nothing: on a panel of two tabs or more after Enter, in a viewer first;
// not with one tab, nor while h|l do something else, a mode, the selection,
// SPLIT off on a narrow terminal (panel-17, panel-19, popup-10, popup-11;
// from filu).
func TestHintTab(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(240, 40)
	pods := func() string { return covCPlain(Cut(ta.Frame()[38], 0, 80)) }
	if got := pods(); !strings.HasPrefix(got, "┗━ Enter:logs h|l:tab m:mark") {
		t.Errorf("a panel of two tabs: %q", got)
	}
	ta.press("D")
	if got := pods(); strings.Contains(got, "h|l") {
		t.Errorf("in a mode h|l do not switch tabs: %q", got)
	}
	ta.press("Esc")
	ta.pods.Tabs = nil
	if got := pods(); strings.Contains(got, "h|l") {
		t.Errorf("one tab: %q", got)
	}
	ta.Viewer(&Viewer{Title: "DevTools", Tabs: []string{"Console", "Network"}, Lines: []string{"x"}})
	ta.settle()
	h := ta.topView().(*viewerView).hint()
	if len(h) == 0 || h[0] != (KeyHint{"h|l", "tab"}) || h[len(h)-1].Key != "Esc" {
		t.Errorf("a viewer of two tabs: %v", h)
	}
	ta.press("v")
	if fmt.Sprint(ta.topView().(*viewerView).hint()) != "[{v select} {Esc cancel}]" {
		t.Error("the selection mode writes h|l:tab")
	}
	ta.press("Esc", "Esc")
	ta.Viewer(&Viewer{Title: "Log", Lines: []string{"x"}})
	ta.settle()
	if fmt.Sprint(ta.topView().(*viewerView).hint()) != "[{/ search} {v viewing} {Esc close}]" {
		t.Errorf("no tabs: %v", ta.topView().(*viewerView).hint())
	}
	ta.press("Esc")

	tc, _ := compareApp(t)
	tc.press("C", "j", "Enter")
	if h := tc.topView().(*viewerView).hint(); h[0] != (KeyHint{"h|l", "tab"}) {
		t.Errorf("the compare: %v", h)
	}
	tc.press("Esc")
	tc.resize(90, 30)
	tc.press("Enter")
	if h := tc.topView().(*viewerView).hint(); h[0].Key == "h|l" {
		t.Errorf("SPLIT off, h|l switch nothing: %v", h)
	}
}

// TestCompareHead: a comparison is titled DIFF, its two tabs after it; its
// first row names the two sides as the app names them, in Lavender, right
// under the top border in the top padding's place, the padding below kept,
// fixed over the lines as they scroll; too long, it folds before ↔, then
// after a /, never cut (note-107, note-80, note-101, popup-15, from gittu).
func TestCompareHead(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(60, 20)
	an, bn := "installation/docker-compose/env.example @ 1ba6d81", "installation/docker-compose/env.example @ b8a7e3b"
	var a, b strings.Builder
	for i := range 60 {
		fmt.Fprintf(&a, "old %d\n", i)
		fmt.Fprintf(&b, "new %d\n", i)
	}
	ta.OpenDiff(ta.pods, DiffItem{Anchor: an, Other: bn, Text: func() (string, string, error) { return a.String(), b.String(), nil }})
	ta.settle()
	vv, ok := ta.topView().(*viewerView)
	if !ok {
		t.Fatalf("no comparison: %T", ta.topView())
	}
	if got := vv.v.heads; len(got) != 2 || got[0] != an || got[1] != "↔ "+bn {
		t.Errorf("folded before ↔: %q", got)
	}
	rowOf := func(s string) int {
		for i, l := range ta.Frame() {
			if strings.Contains(StripStyles(l), s) {
				return i
			}
		}
		return -1
	}
	text, frame := ta.screenText(), frameText(ta)
	title := ta.Frame()[rowOf("DIFF")]
	if plain := StripStyles(title); !strings.Contains(plain, "UNIFIED") || !strings.Contains(plain, "SPLIT") || strings.Contains(plain, "↔") || strings.Contains(plain, "…") {
		t.Errorf("the title row: %q", plain)
	}
	if !strings.Contains(frame, Fg(Lavender).Render(an)) || !strings.Contains(frame, Fg(Lavender).Render("↔ "+bn)) {
		t.Errorf("the first row is not Lavender:\n%s", text)
	}
	if _, h := vv.measure(60, 20); rowOf("of 120")-rowOf("DIFF")+1 != h {
		t.Errorf("the popup is %d rows, measured %d:\n%s", rowOf("of 120")-rowOf("DIFF")+1, h, text)
	}
	// Drawn to its height as it is: no blank row above for fitRows to drop.
	if w, h := vv.measure(60, 20); len(vv.draw(w, h, Blue)) != h {
		t.Errorf("drawn %d rows, measured %d", len(vv.draw(w, h, Blue)), h)
	}
	first, second := rowOf(an), rowOf("↔ "+bn)
	if first != rowOf("DIFF")+1 || second != first+1 || rowOf("old 0") != first+2 { // right under the border, no blank row
		t.Fatalf("the sides at %d and %d, the lines at %d:\n%s", first, second, rowOf("old 0"), text)
	}
	ta.press("d", "d")
	if rowOf(an) != first || rowOf("↔ "+bn) != second || rowOf("old 0") >= 0 {
		t.Errorf("scrolled, the sides moved or the lines did not:\n%s", ta.screenText())
	}
	if ta.press("G"); rowOf("new 59") < 0 || rowOf("of 120") != rowOf("new 59")+2 {
		t.Errorf("at the end, the last line is not in sight:\n%s", ta.screenText())
	}
	// Narrower: a side too wide goes on after a /.
	ta.resize(40, 20)
	want := []string{"installation/docker-compose/", "env.example @ 1ba6d81", "↔ installation/docker-compose/", "env.example @ b8a7e3b"}
	if got := vv.v.heads; strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("folded after a /: %q", got)
	}
	if strings.Contains(ta.screenText(), "…") {
		t.Errorf("cut with …:\n%s", ta.screenText())
	}
	// Short enough: one row; no / to fold after: where the row is full.
	if got := diffHead("a", "b", 10); len(got) != 1 || got[0] != "a ↔ b" {
		t.Errorf("one row: %q", got)
	}
	if got := diffHead("abcdefghij", "x", 6); strings.Join(got, "|") != "abcdef|ghij|↔ x" {
		t.Errorf("no /: %q", got)
	}
}

// TestCompareWidth: the comparison is as wide as the terminal but 2, with
// no 120-column cap, SPLIT's two sides sharing it; another viewer keeps the
// cap (F7-17, from gittu).
func TestCompareWidth(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(200, 40)
	ta.OpenDiff(ta.pods, DiffItem{Anchor: "a", Other: "b", Text: func() (string, string, error) { return "left\n", "right\n", nil }})
	ta.settle()
	vv := ta.topView().(*viewerView)
	if w, _ := vv.measure(200, 40); w != 198 {
		t.Errorf("the comparison is %d wide on 200 columns, want 198", w)
	}
	// Its height: the borders, the sides' row in the top padding's place,
	// the two lines, the padding below (popup-15).
	if _, h := vv.measure(200, 40); h != 6 {
		t.Errorf("the comparison is %d rows, want 6", h)
	}
	if w, _ := (&viewerView{v: &Viewer{Lines: []string{"x"}}}).measure(200, 40); w != 120 {
		t.Errorf("another viewer is %d wide, want 120", w)
	}
	if text := ta.screenText(); !strings.Contains(text, "left") || !strings.Contains(text, "right") {
		t.Errorf("a short comparison does not show all its lines:\n%s", text)
	}
	ta.press("l") // SPLIT
	for _, l := range vv.v.Lines {
		if p := StripStyles(l); strings.Contains(p, "right") && strings.Index(p, "right") < 90 {
			t.Errorf("the other side starts at %d, in 120 columns: %q", strings.Index(p, "right"), p)
		}
	}
	if !strings.Contains(ta.screenText(), "right") {
		t.Errorf("no other side:\n%s", ta.screenText())
	}
}

// TestCompareDeprecated: Compare and CompareAction, the Diffing mode's old
// names, still build and give the same mode, so an app on them upgrades
// without a change (tdp panel.9 renamed it the Diffing mode).
func TestCompareDeprecated(t *testing.T) {
	ta := newTestApp(t)
	var c Compare = DiffMode{Key: "C", Text: func(r Row) (string, error) { return r.Cells[0], nil }}
	act := ta.CompareAction(ta.pods, ta.list, c)
	if act.Label != "Diff" || act.Key != "C" {
		t.Errorf("CompareAction: %q %q", act.Key, act.Label)
	}
	ta.pods.Actions = append(ta.pods.Actions, act)
	ta.press("C")
	if m := ta.pods.InMode(); m == nil || m.Name != "Diffing" {
		t.Errorf("not in the Diffing mode: %v", m)
	}
}

// TestDiffError: a side that cannot be read is an error popup titled with
// the Diff and the side, DIFF: <side>; one with no side to blame, with the
// item (tdp note.8, panel.9: the Compare mode is the Diffing mode).
func TestDiffError(t *testing.T) {
	ta := newTestApp(t)
	ta.pods.Actions = append(ta.pods.Actions, ta.DiffModeAction(ta.pods, ta.list, DiffMode{Key: "C",
		Text: func(r Row) (string, error) {
			if r.ID == "1" {
				return "", errors.New("permission denied")
			}
			return r.Cells[0], nil
		}}))
	ta.press("C", "j", "Enter") // pod-00 with pod-01, which cannot be read
	if text := ta.screenText(); !strings.Contains(text, "DIFF: pod-01") || !strings.Contains(text, "permission denied") {
		t.Errorf("a side that cannot be read:\n%s", text)
	}
	ta.covBCloseAll()
	ta.OpenDiff(ta.pods, DiffItem{Anchor: "a", Other: "b", Items: func() ([]DiffItem, error) { return nil, errors.New("gone") }})
	ta.settle()
	if text := ta.screenText(); !strings.Contains(text, "DIFF: a ↔ b") {
		t.Errorf("the item's error:\n%s", text)
	}
}
