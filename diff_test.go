package terminu

import (
	"errors"
	"strings"
	"testing"
)

// manyApp is the test app with a Compare action on its pods, pod-03 to
// pod-05 holding many: pod-03 and pod-04 differ in five items, one of them
// holding many itself, the others in none.
func manyApp(t *testing.T) *testApp {
	ta := newTestApp(t)
	text := func(a, b string) func() (string, string, error) {
		return func() (string, string, error) { return a, b, nil }
	}
	ta.pods.Actions = append(ta.pods.Actions, ta.DiffModeAction(ta.pods, ta.list, DiffMode{Key: "C",
		Can:  func(r Row) bool { return r.ID != "2" },
		Many: func(r Row) bool { return r.ID == "3" || r.ID == "4" || r.ID == "5" },
		Text: func(r Row) (string, error) { return "age: " + r.Cells[1] + "\n", nil },
		Items: func(a, b Row) ([]DiffItem, error) {
			if b.ID != "4" {
				return nil, nil
			}
			return []DiffItem{
				{Mark: DiffChanged, Name: "main.go", Anchor: "pod-03/main.go", Other: "pod-04/main.go", Text: text("package main\nv1\n", "package main\nv2\n")},
				{Mark: DiffAdded, Name: "README.md", Anchor: "pod-03/README.md", Other: "pod-04/README.md", Text: text("", "hello\n")},
				{Mark: DiffRemoved, Name: "old.txt", Anchor: "pod-03/old.txt", Other: "pod-04/old.txt", Text: text("bye\n", "")},
				{Mark: DiffChanged, Name: "logo.png", Anchor: "pod-03/logo.png", Other: "pod-04/logo.png"},
				{Mark: DiffChanged, Name: "src", Anchor: "pod-03/src", Other: "pod-04/src", Items: func() ([]DiffItem, error) {
					return []DiffItem{{Mark: DiffChanged, Name: "util.go", Anchor: "pod-03/src/util.go", Other: "pod-04/src/util.go", Text: text("a\n", "b\n")}}, nil
				}},
			}, nil
		}}))
	return ta
}

func frameText(ta *testApp) string { return strings.Join(ta.Frame(), "\n") }

// TestCompareMany: comparing two rows that hold many goes in to a list of
// the items that differ, the location row the two sides as the app names
// them, each item's mark in its colour; Enter opens an item's comparison,
// titled with its two sides, or goes in again; one that is not text has no
// comparison; Esc goes back up a level at a time, the cursor on the item it
// came out of, the last back to the mode with the anchor kept; a row that
// holds many is not compared with one that does not; nothing that differs
// is the Info toast (panel-176 to panel-179).
func TestCompareMany(t *testing.T) {
	ta := manyApp(t)
	ta.press("j", "j", "j", "C") // pod-03, the anchor
	if ta.press("k"); !strings.Contains(covCFooter(ta), "C|Enter:diff") || !strings.Contains(frameText(ta), Fg(Surface2).Render(":diff")) {
		t.Errorf("on pod-02 the compare keys are off: %q", covCFooter(ta))
	}
	ta.press("k") // pod-01: text against one that holds many
	if ta.press("Enter"); ta.top() != nil || ta.pods.level() != nil || !strings.Contains(frameText(ta), Fg(Surface2).Render(":diff")) {
		t.Error("a row that holds many compared with one that does not")
	}
	ta.press("j", "j", "j", "j") // pod-05: nothing differs
	if ta.press("Enter"); ta.pods.level() != nil || !strings.Contains(ta.screenText(), "No differences between pod-03 and pod-05") {
		t.Errorf("nothing differs:\n%s", ta.screenText())
	}
	ta.toasts, ta.inspectT = nil, nil
	ta.press("k", "Enter") // pod-04
	if ta.pods.level() == nil || ta.pods.InMode() == nil || ta.pods.InMode().Name != "Diff" {
		t.Fatalf("did not go in, in the mode:\n%s", ta.screenText())
	}
	text, frame := ta.screenText(), frameText(ta)
	for _, want := range []string{"pod-03 ↔ pod-04", "README.md", "old.txt", "logo.png", "src"} {
		if !strings.Contains(text, want) {
			t.Errorf("the list has no %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "pod-05") {
		t.Error("the panel's own rows show in the list")
	}
	if !strings.Contains(frame, Fg(Lavender).Render("pod-03 ↔ pod-04")) {
		t.Error("the location row is not the two sides, in Lavender")
	}
	for _, m := range []string{Fg(Green).Render("+"), Fg(Red).Render("-"), Fg(Peach).Render("~")} {
		if !strings.Contains(frame, m) {
			t.Errorf("no mark %q", m)
		}
	}
	if !strings.Contains(covCFooter(ta), "Esc:up") {
		t.Errorf("the footer: %q", covCFooter(ta))
	}
	ta.press("?")
	if ref := ta.screenText(); !strings.Contains(ref, "one level up") || strings.Contains(ref, "back to before Compare") {
		t.Errorf("the mode's ? in the list:\n%s", ref)
	}
	ta.press("Esc", "Space")
	if menu := ta.screenText(); !strings.Contains(menu, "its two sides") || strings.Contains(menu, "with pod-03") || !strings.Contains(menu, "Up") {
		t.Errorf("the mode's Space menu in the list:\n%s", menu)
	}
	ta.press("Esc")
	if ta.top() != nil || ta.pods.level() == nil {
		t.Fatal("closing the menu left the list")
	}
	// Enter: the comparison, titled with the item's two sides.
	ta.press("Enter")
	v, ok := ta.topView().(*viewerView)
	if !ok || !strings.Contains(ta.screenText(), "pod-03/main.go ↔ pod-04/main.go") || !slicesContain(v.v.Lines, Fg(Green).Render("+v2")) {
		t.Fatalf("the item's comparison:\n%s", ta.screenText())
	}
	ta.press("Esc")
	if ta.top() != nil || ta.pods.level() == nil {
		t.Fatal("closing the comparison left the list")
	}
	// Only the other has it: one side empty.
	ta.press("j", "Enter")
	if v, ok := ta.topView().(*viewerView); !ok || !slicesContain(v.v.Lines, Fg(Green).Render("+hello")) {
		t.Errorf("an item only the other has:\n%s", ta.screenText())
	}
	ta.press("Esc", "j", "j") // logo.png: not text
	if ta.press("Enter"); ta.top() != nil || !strings.Contains(frameText(ta), Fg(Surface2).Render(":diff")) {
		t.Error("an item that is not text was compared, or its key is not off")
	}
	ta.press("j", "Enter") // src: in again
	if !strings.Contains(ta.screenText(), "pod-03/src ↔ pod-04/src") || !strings.Contains(ta.screenText(), "util.go") || len(ta.pods.levels) != 2 {
		t.Fatalf("did not go in again:\n%s", ta.screenText())
	}
	ta.press("Enter")
	if !strings.Contains(ta.screenText(), "pod-03/src/util.go ↔ pod-04/src/util.go") {
		t.Errorf("a nested item's title:\n%s", ta.screenText())
	}
	ta.press("Esc", "Esc")
	if it, _ := ta.pods.level().current(); len(ta.pods.levels) != 1 || it.Name != "src" {
		t.Errorf("Esc did not go up one level to src: %d levels, on %q", len(ta.pods.levels), it.Name)
	}
	ta.press("Esc")
	if ta.pods.level() != nil || ta.pods.InMode() == nil || ta.list.Anchor != "3" {
		t.Fatal("the last Esc did not go back to the mode with the anchor kept")
	}
	if r, _ := ta.list.Current(); r.ID != "4" || strings.Contains(ta.screenText(), "↔") {
		t.Errorf("back on %s:\n%s", r.ID, ta.screenText())
	}
	ta.press("Esc")
	if ta.pods.InMode() != nil || ta.list.Anchor != "" {
		t.Error("Esc did not leave the mode")
	}
}

// TestCompareManyAnchorGone: the anchor gone while in the list ends the
// mode with its Warning toast, the lists gone into closed (panel-145).
func TestCompareManyAnchorGone(t *testing.T) {
	ta := manyApp(t)
	ta.press("j", "j", "j", "C", "j", "Enter")
	if ta.pods.level() == nil {
		t.Fatal("did not go in")
	}
	rows := ta.list.Rows()
	ta.list.SetRows(append(rows[:3:3], rows[4:]...))
	ta.press("j")
	if ta.pods.InMode() != nil || ta.pods.level() != nil || !strings.Contains(ta.screenText(), "Diff ended: pod-03 is gone") {
		t.Errorf("the anchor gone:\n%s", ta.screenText())
	}
}

// TestOpenDiff: an action of the app's own showing two versions of one item
// opens the same comparison, titled with the two as the app names them, the
// older the anchor; one that holds many goes in to the list of the items
// that differ as the compare mode does, out of a mode: Enter opens, Esc goes
// up, Tab goes to another panel and back with the list kept, the Space menu
// and ? have the list's keys, not the panel's own (note-101, note-102).
func TestOpenDiff(t *testing.T) {
	ta := newTestApp(t)
	// The panel's own location row, inspect and other keys are about its
	// own rows: none of them shows in the list.
	ta.pods.Location = "~/proj"
	ta.pods.Inspect = func() []string { return []string{"pod inspect"} }
	ta.pods.OtherKeys = []KeySection{{Title: "terminal", Rows: []KeyRow{{Key: "Ctrl-X", Desc: "zap the pod"}}}}
	item := DiffItem{Anchor: "ui.go @ 1a2b3c4", Other: "ui.go", Text: func() (string, string, error) { return "x\n", "y\n", nil }}
	ta.OpenDiff(ta.pods, item)
	ta.settle()
	if text := ta.screenText(); !strings.Contains(text, "ui.go @ 1a2b3c4 ↔ ui.go") || !strings.Contains(text, "UNIFIED") || !strings.Contains(text, "SPLIT") {
		t.Fatalf("two versions of one item:\n%s", text)
	}
	// It opens on UNIFIED every time, the tab chosen not kept (note-84).
	ta.press("l", "Esc")
	ta.OpenDiff(ta.pods, item)
	ta.settle()
	if v, ok := ta.topView().(*viewerView); !ok || v.v.Tab != 0 {
		t.Error("the next comparison did not open on UNIFIED")
	}
	ta.press("Esc")
	item.Text = func() (string, string, error) { return "x\n", "x\n", nil }
	ta.OpenDiff(ta.pods, item)
	ta.settle()
	if ta.top() != nil || !strings.Contains(ta.screenText(), "No differences between ui.go @ 1a2b3c4 and ui.go") {
		t.Errorf("the same twice:\n%s", ta.screenText())
	}
	ta.toasts, ta.inspectT = nil, nil
	commit := DiffItem{Anchor: "1a2b3c4", Other: "5d6e7f8", Items: func() ([]DiffItem, error) {
		return []DiffItem{
			{Mark: DiffChanged, Name: "old.go → new.go", Anchor: "old.go @ 1a2b3c4", Other: "new.go @ 5d6e7f8", Text: func() (string, string, error) { return "a\n", "b\n", nil }},
			{Mark: DiffAdded, Name: "internal", Anchor: "internal @ 1a2b3c4", Other: "internal @ 5d6e7f8", Items: func() ([]DiffItem, error) {
				return []DiffItem{{Mark: DiffAdded, Name: "x.go", Anchor: "internal/x.go @ 1a2b3c4", Other: "internal/x.go @ 5d6e7f8", Text: func() (string, string, error) { return "", "x\n", nil }}}, nil
			}},
		}, nil
	}}
	ta.press("j") // the panel's own inspect shows on its rows
	if !strings.Contains(ta.screenText(), "pod inspect") {
		t.Fatalf("no inspect on the panel's rows:\n%s", ta.screenText())
	}
	ta.OpenDiff(ta.pods, commit)
	ta.press("g", "g") // a message through, still the top row
	ta.settle()
	text := ta.screenText()
	if ta.pods.InMode() != nil || !strings.Contains(text, "1a2b3c4 ↔ 5d6e7f8") || strings.Contains(text, "~/proj") || !strings.Contains(text, "old.go → new.go") {
		t.Fatalf("going into a commit:\n%s", text)
	}
	if strings.Contains(text, "pod inspect") || !strings.Contains(text, "Enter:diff") {
		t.Errorf("the panel's inspect shows, or the hint is not Enter:diff:\n%s", text)
	}
	ta.press("Space")
	menu := ta.screenText()
	if !strings.Contains(menu, "Diff") || strings.Contains(menu, "Logs") || strings.Contains(menu, "Delete") {
		t.Errorf("the Space menu has the panel's own actions:\n%s", menu)
	}
	ta.press("Esc", "?")
	if ref := ta.screenText(); !strings.Contains(ref, "one level up") || strings.Contains(ref, "Logs") || strings.Contains(ref, "zap the pod") {
		t.Errorf("the key reference:\n%s", ref)
	}
	ta.press("Esc", "x") // the panel's own Delete is not a key here
	if ta.top() != nil {
		t.Error("the panel's own action ran in the list")
	}
	ta.press("Enter")
	if !strings.Contains(ta.screenText(), "old.go @ 1a2b3c4 ↔ new.go @ 5d6e7f8") {
		t.Errorf("a renamed item's comparison:\n%s", ta.screenText())
	}
	ta.press("Esc", "Tab", "1")
	if ta.Screen().Focused() != ta.pods || ta.pods.level() == nil {
		t.Fatal("Tab and back lost the list")
	}
	ta.press("j", "Enter")
	// The location row cut at the front, as panel.10 cuts a long one.
	if len(ta.pods.levels) != 2 || !strings.Contains(ta.screenText(), "…rnal @ 1a2b3c4 ↔ internal @ 5d6e7f8") || !strings.Contains(ta.screenText(), "x.go") {
		t.Fatalf("did not go in again:\n%s", ta.screenText())
	}
	ta.press("Esc", "Esc")
	if ta.pods.level() != nil || !strings.Contains(ta.screenText(), "~/proj") || !strings.Contains(ta.screenText(), "pod-00") {
		t.Errorf("Esc did not go back to the panel as it was:\n%s", ta.screenText())
	}
	if ta.press("Enter"); len(ta.ran) == 0 || ta.ran[len(ta.ran)-1] != "logs" {
		t.Errorf("the panel's own Enter after: %v", ta.ran)
	}
	ta.ran = nil
	// An error reading is the error popup; nothing is gone into.
	commit.Items = func() ([]DiffItem, error) { return nil, errors.New("bad object") }
	ta.OpenDiff(ta.pods, commit)
	ta.settle()
	if ta.pods.level() != nil || !strings.Contains(ta.screenText(), "bad object") {
		t.Errorf("an error:\n%s", ta.screenText())
	}
}

// TestLeaveDiff: LeaveDiff leaves every list of the items that differ the
// panel went into, back to its own rows with the cursor where the app put
// it, and does nothing when there is none; a compare mode stays, its anchor
// kept (from gitreu).
func TestLeaveDiff(t *testing.T) {
	ta := newTestApp(t)
	ta.pods.LeaveDiff() // none: nothing
	if ta.pods.level() != nil || !strings.Contains(ta.screenText(), "pod-00") {
		t.Fatal("LeaveDiff with no list changed the panel")
	}
	text := func(a, b string) func() (string, string, error) {
		return func() (string, string, error) { return a, b, nil }
	}
	inner := DiffItem{Mark: DiffChanged, Name: "src", Anchor: "src @ a", Other: "src @ b", Items: func() ([]DiffItem, error) {
		return []DiffItem{{Mark: DiffAdded, Name: "x.go", Anchor: "x.go @ a", Other: "x.go @ b", Text: text("", "x\n")}}, nil
	}}
	ta.OpenDiff(ta.pods, DiffItem{Anchor: "a", Other: "b", Items: func() ([]DiffItem, error) { return []DiffItem{inner}, nil }})
	ta.press("Enter") // in again, into src
	if len(ta.pods.levels) != 2 {
		t.Fatalf("%d levels, want 2", len(ta.pods.levels))
	}
	// The app's own action: its row under the cursor, the panel back.
	ta.list.Select("5")
	ta.pods.LeaveDiff()
	ta.settle()
	if r, _ := ta.list.Current(); ta.pods.level() != nil || r.ID != "5" || strings.Contains(ta.screenText(), "↔") || !strings.Contains(ta.screenText(), "pod-05") {
		t.Errorf("not back on pod-05:\n%s", ta.screenText())
	}
	if ta.press("Enter"); len(ta.ran) == 0 || ta.ran[len(ta.ran)-1] != "logs" {
		t.Errorf("the panel's own Enter after: %v", ta.ran)
	}

	// In a compare mode: the mode stays, its anchor kept.
	tm := manyApp(t)
	tm.press("j", "j", "j", "C", "j", "Enter", "j", "j", "j", "j", "Enter")
	if len(tm.pods.levels) != 2 {
		t.Fatalf("%d levels in the mode, want 2", len(tm.pods.levels))
	}
	tm.pods.LeaveDiff()
	if tm.pods.level() != nil || tm.pods.InMode() == nil || tm.list.Anchor != "3" {
		t.Error("LeaveDiff left the compare mode, or its anchor")
	}
}

// TestDiffClearsFilter: going into a list of the items that differ changes
// what the panel lists, so its filter is cleared first, whether a compare
// mode goes in or OpenDiff does; back up, every row is there, the cursor on
// the one it was on. Nothing that differs goes nowhere and keeps the filter
// (panel-180).
func TestDiffClearsFilter(t *testing.T) {
	ta := manyApp(t)
	ta.press("/", "p", "o", "d", "-", "0", "Enter") // pod-00 to pod-09; pod-39 hidden
	if len(ta.list.view) != 10 {
		t.Fatalf("the filter kept %d rows, want 10", len(ta.list.view))
	}
	ta.press("j", "j", "j", "C", "j", "j", "Enter") // pod-03 with pod-05: nothing differs
	if ta.pods.level() != nil || !ta.list.Filtering() {
		t.Fatal("nothing that differs cleared the filter")
	}
	ta.toasts, ta.inspectT = nil, nil
	ta.press("k", "Enter") // pod-04: in
	if ta.pods.level() == nil || ta.list.Filtering() {
		t.Fatalf("going in from the compare mode kept the filter:\n%s", ta.screenText())
	}
	ta.press("Esc")
	if r, _ := ta.list.Current(); ta.pods.level() != nil || len(ta.list.view) != 40 || r.ID != "4" {
		t.Errorf("back up: %d rows, the cursor on %q", len(ta.list.view), r.ID)
	}

	tb := newTestApp(t)
	tb.press("/", "p", "o", "d", "-", "0", "Enter", "j", "j")
	commit := DiffItem{Anchor: "1a2b3c4", Other: "5d6e7f8", Items: func() ([]DiffItem, error) {
		return []DiffItem{{Mark: DiffAdded, Name: "x.go", Anchor: "x.go @ 1a2b3c4", Other: "x.go @ 5d6e7f8",
			Text: func() (string, string, error) { return "", "x\n", nil }}}, nil
	}}
	tb.OpenDiff(tb.pods, commit)
	tb.settle()
	if tb.pods.level() == nil || tb.list.Filtering() {
		t.Fatalf("OpenDiff going in kept the filter:\n%s", tb.screenText())
	}
	tb.press("Esc")
	if r, _ := tb.list.Current(); len(tb.list.view) != 40 || r.ID != "2" {
		t.Errorf("back up after OpenDiff: %d rows, the cursor on %q", len(tb.list.view), r.ID)
	}
}
