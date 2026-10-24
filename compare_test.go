package terminu

import (
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
// same age have the same text.
func compareApp(t *testing.T) (*testApp, *bool) {
	ta := newTestApp(t)
	split := new(bool)
	ta.pods.Actions = append(ta.pods.Actions, ta.CompareAction(ta.pods, ta.list, Compare{Key: "C", Split: split,
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
// anchor gone (components/layout/panel, compare mode: panel-138 to 144).
func TestCompareMode(t *testing.T) {
	ta, _ := compareApp(t)
	ta.press("C")
	if m := ta.pods.InMode(); m == nil || m.Name != "Compare" || ta.list.Anchor != "0" {
		t.Fatalf("not in the compare mode: %v, anchor %q", m, ta.list.Anchor)
	}
	frame := strings.Join(ta.Frame(), "\n")
	if !strings.Contains(covCFooter(ta), "C|Enter:compare") || !strings.Contains(frame, Fg(Surface2).Render(":compare")) {
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
	if ta.Screen().Focused() != ta.pods || !strings.Contains(ta.screenText(), "Press [Esc] to leave Compare first") {
		t.Error("Tab left the mode's panel")
	}
	ta.toast = nil
	ta.press("k", "Enter")
	v, ok := ta.topView().(*viewerView)
	if !ok {
		t.Fatalf("no compare popup: %T", ta.topView())
	}
	if text := ta.screenText(); !strings.Contains(text, "DIFF — pod-00 ↔ pod-01") || !strings.Contains(text, "UNIFIED") {
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
	ta.toast = nil
	if ta.press("g", "g"); ta.list.cursor != 0 {
		t.Errorf("gg in the mode: at %d", ta.list.cursor)
	}
	ta.press("Esc")
	if ta.pods.InMode() != nil || ta.list.Anchor != "" {
		t.Error("Esc did not leave the mode, or the anchor stayed")
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
// cut with …; it is remembered; under 96 columns it is off and UNIFIED is
// shown (note-83 to 85).
func TestCompareSplit(t *testing.T) {
	ta, split := compareApp(t)
	ta.press("C", "j", "Enter", "l")
	v := ta.topView().(*viewerView).v
	if v.Tab != 1 || !*split || !v.NoWrap {
		t.Fatalf("l did not switch to SPLIT: tab %d, split %v", v.Tab, *split)
	}
	if !slicesContain(v.Lines, Fg(Red).Render("age: 0")) || !slicesContain(v.Lines, Fg(Green).Render("age: 1")) {
		t.Errorf("SPLIT: %q", v.Lines)
	}
	row := StripStyles(v.Lines[0])
	if !strings.HasPrefix(row, " 1 age: 0") || !strings.Contains(row, " 1 age: 1") {
		t.Errorf("each half numbered: %q", row)
	}
	if got, _ := diffSplit([]diffOp{{'-', 0, -1}}, []string{strings.Repeat("x", 40)}, nil, 30); !strings.Contains(StripStyles(got[0]), "…") || Width(got[0]) > 30 {
		t.Errorf("a long line is cut: %q", got)
	}
	ta.press("Esc", "Enter")
	if v := ta.topView().(*viewerView).v; v.Tab != 1 {
		t.Error("SPLIT is not remembered")
	}
	ta.press("Esc")
	ta.resize(90, 30)
	ta.press("Enter")
	v = ta.topView().(*viewerView).v
	if ta.press("l"); v.Tab != 0 || !*split {
		t.Errorf("under 96 columns: tab %d, remembered %v", v.Tab, *split)
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
	if ta.pods.InMode() != nil || ta.list.Anchor != "" || !strings.Contains(ta.screenText(), "Compare ended: pod-00 is gone") {
		t.Errorf("the anchor gone:\n%s", ta.screenText())
	}
}

// TestModeKeyDisabled: a mode key that cannot be pressed now does nothing
// and is in the disabled colour in its key reference (Rules M6).
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
// changes size: under 96 columns UNIFIED, back over it the remembered
// SPLIT, the stretch on top kept in sight (note-85).
func TestCompareResize(t *testing.T) {
	ta := newTestApp(t)
	split := new(bool)
	*split = true
	ta.pods.Actions = append(ta.pods.Actions, ta.CompareAction(ta.pods, ta.list, Compare{Key: "C", Split: split,
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
	ta.resize(120, 16)
	ta.press("C", "j", "Enter")
	v := ta.topView().(*viewerView).v
	if v.Tab != 1 {
		t.Fatalf("opened on tab %d, want the remembered SPLIT", v.Tab)
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
	ta.resize(90, 16)
	if v.Tab != 0 || v.NoWrap || !strings.Contains(StripStyles(v.Lines[v.top]), want) {
		t.Errorf("under 96: tab %d, on top %q, want %q", v.Tab, StripStyles(v.Lines[v.top]), want)
	}
	if !strings.Contains(ta.screenText(), want) {
		t.Errorf("%q is not in sight:\n%s", want, ta.screenText())
	}
	ta.resize(120, 16)
	if v.Tab != 1 || !*split || !strings.Contains(ta.screenText(), want) {
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
// filu FL27).
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
	if fmt.Sprint(ta.topView().(*viewerView).hint()) != "[{v visual} {Esc close}]" {
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
