package terminu

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// detailsApp is a test app whose [1] FILES list shows its Diffing mode in its
// details, [2] CHANGES, which otherwise shows the row under the cursor; [3]
// is another panel. Rows: a.txt, b.txt (one line changed), c.txt (as
// a.txt), d.bin (not compared), e.txt (cannot be read), v1/, v2/ and v3/
// (folders: v2/ adds new.go and changes sub/, which removes old.go; v3/ is
// as v1/).
func detailsApp(t *testing.T) (ta *testApp, files, changes *Panel) {
	t.Helper()
	ta = &testApp{App: anySize(New("test"))}
	ta.list = &List{Columns: []Column{{Title: "Name"}}}
	var rows []Row
	for i, n := range []string{"a.txt", "b.txt", "c.txt", "d.bin", "e.txt", "v1/", "v2/", "v3/"} {
		rows = append(rows, Row{ID: itoa(i), Cells: []string{n}})
	}
	ta.list.SetRows(rows)
	own := &Pager{}
	ta.list.OnMove = func(r Row) tea.Cmd { own.SetLines([]string{"own: " + r.Cells[0]}); return nil }
	folder := func(r Row) bool { return strings.HasSuffix(r.Cells[0], "/") }
	texts := map[string]string{"a.txt": "1\n2\n3\n", "b.txt": "1\nX\n3\n", "c.txt": "1\n2\n3\n"}
	files = &Panel{Label: "Files", Body: ta.list}
	changes = &Panel{Label: "Changes", Body: own, Fail: &Failure{Fact: "the app's own failure"}}
	files.Details = changes
	files.Actions = []Action{ta.DiffModeAction(files, ta.list, DiffMode{Key: "D", InDetails: true, Noun: "file",
		Can:  func(r Row) bool { return r.Cells[0] != "d.bin" },
		Many: folder,
		Text: func(r Row) (string, error) {
			if r.Cells[0] == "e.txt" {
				return "", errors.New("permission denied")
			}
			return texts[r.Cells[0]], nil
		},
		Items: func(x, y Row) ([]DiffItem, error) {
			if y.Cells[0] == "v3/" {
				return nil, nil
			}
			return []DiffItem{
				{Mark: DiffAdded, Name: "new.go", Anchor: "new.go @ v1", Other: "new.go @ v2",
					Text: func() (string, string, error) { return "", "package x\n", nil }},
				{Mark: DiffChanged, Name: "sub/", Anchor: "sub @ v1", Other: "sub @ v2",
					Items: func() ([]DiffItem, error) {
						return []DiffItem{{Mark: DiffRemoved, Name: "old.go", Anchor: "old.go @ v1", Other: "old.go @ v2",
							Text: func() (string, string, error) { return "gone\n", "", nil }}}, nil
					}},
			}, nil
		}})}
	other := &Panel{Label: "Other", Body: &Pager{EmptyFact: "nothing"}}
	ta.AddScreen(&Screen{Name: "Files", Key: "F", Panels: []*Panel{files, changes, other}})
	ta.resize(240, 40) // each panel 80 wide: the details sentences on one row
	return ta, files, changes
}

// detFrame is panel p's frame as the screen draws it now, by itself.
func detFrame(ta *testApp, p *Panel) []string {
	s := ta.Screen()
	i := 0
	for k, q := range s.Panels {
		if q == p {
			i = k
		}
	}
	return p.draw(i+1, 60, 12, s.Focused() == p, false)
}

// TestDiffInDetails: a list's Diffing mode shown in its details (tdp panel.9
// "With a details panel", panel-185): the mode spans both frames, each in
// the mode's colour with Diff on it, the heavy line on the one with the
// focus, which stays on the list; the details follow the cursor: an empty
// state on the anchor and on a row the same as it, no toast; UNIFIED for
// two texts; the items that differ with their marks for two folders, the
// location row <anchor> ↔ <the other>.
func TestDiffInDetails(t *testing.T) {
	ta, files, changes := detailsApp(t)
	if !strings.Contains(ta.screenText(), "own: a.txt") {
		t.Fatalf("the details show the cursor's row:\n%s", ta.screenText())
	}
	ta.press("D")
	if m := files.InMode(); m == nil || m.Name != "Diffing" {
		t.Fatalf("the list is not in the mode: %v", m)
	}
	if m := changes.InMode(); m == nil || m.Name != "Diffing" {
		t.Fatalf("the details are not in the mode: %v", m)
	}
	if ta.Screen().Focused() != files {
		t.Error("entering the mode moved the focus")
	}
	for _, p := range []*Panel{files, changes} {
		top := detFrame(ta, p)[0]
		if c := covBCells(top)[0]; c.fg != covBRGB(Yellow) {
			t.Errorf("[%s] is not in the mode's colour: %s", p.Label, c.fg)
		}
		if !strings.Contains(StripStyles(top), "Diff") {
			t.Errorf("[%s] has no Diff on it: %q", p.Label, StripStyles(top))
		}
		heavy := strings.HasPrefix(StripStyles(top), "┏")
		if heavy != (p == files) {
			t.Errorf("[%s] heavy %v with the focus on the list", p.Label, heavy)
		}
	}
	text := ta.screenText()
	if !strings.Contains(text, "Move to another file to diff with a.txt") || strings.Contains(text, "own: a.txt") {
		t.Errorf("on the anchor:\n%s", text)
	}
	if strings.Contains(text, "the app's own failure") {
		t.Error("the app's own failure shows over the diff")
	}
	ta.press("j") // b.txt: UNIFIED
	text = ta.screenText()
	if !strings.Contains(text, "-2") || !strings.Contains(text, "+X") {
		t.Errorf("b.txt against a.txt is not UNIFIED:\n%s", text)
	}
	ta.press("j") // c.txt: the same
	if text = ta.screenText(); !strings.Contains(text, "No differences between a.txt and c.txt") || ta.lastToast() != nil {
		t.Errorf("the same text: a toast %v\n%s", ta.lastToast(), text)
	}
	ta.press("j") // d.bin: not compared
	if text = ta.screenText(); !strings.Contains(text, "Can't diff d.bin with a.txt") {
		t.Errorf("a row that cannot be compared:\n%s", text)
	}
	ta.press("j") // e.txt: an error, in Red, as a panel's (tdp panel.15)
	if text = ta.screenText(); !strings.Contains(text, "e.txt: permission denied") || ta.top() != nil {
		t.Errorf("a failure is not the details':\n%s", text)
	}
	if !strings.Contains(strings.Join(changes.draw(2, 60, 12, true, false), "\n"), Fg(Red).Render("e.txt: permission denied")) {
		t.Error("the failure is not Red")
	}
	ta.press("j") // v1/: a folder against a text
	if text = ta.screenText(); !strings.Contains(text, "Can't diff v1/ with a.txt") {
		t.Errorf("a folder against a text:\n%s", text)
	}
	if ta.lastToast() != nil {
		t.Errorf("a toast: %v", ta.lastToast())
	}

	// Two folders: the items that differ, marked, the location row.
	ta.press("Esc", "D", "j")
	text = ta.screenText()
	for _, want := range []string{"v1/ ↔ v2/", "new.go", "sub/"} {
		if !strings.Contains(text, want) {
			t.Errorf("v1/ against v2/: no %q\n%s", want, text)
		}
	}
	det := strings.Join(changes.draw(2, 60, 12, true, false), "\n") // undimmed
	// new.go is under the details' cursor, in its colours; sub/'s mark is Peach.
	if !strings.Contains(StripStyles(det), "+ new.go") || !strings.Contains(det, DiffChanged.span().Style.Render("~")) {
		t.Errorf("the items have no marks:\n%s", det)
	}
	ta.press("j") // v3/: the same
	if text = ta.screenText(); !strings.Contains(text, "No differences between v1/ and v3/") || strings.Contains(text, "v1/ ↔ v3/") {
		t.Errorf("two folders the same:\n%s", text)
	}

	// Out of the mode the details are the app's again, following the cursor.
	ta.press("Esc")
	if files.InMode() != nil || changes.InMode() != nil || ta.list.Anchor != "" {
		t.Fatal("Esc on the list did not leave the mode")
	}
	if text = ta.screenText(); !strings.Contains(text, "own: v3/") || strings.Contains(text, "Diff") {
		t.Errorf("the details after the mode:\n%s", text)
	}
}

// TestDiffInDetailsKeys: in a Diffing mode shown in the details (tdp panel.9,
// panel-186), the diff key or Enter on the list sends the focus to the
// details, off where they have nothing; there Enter opens an item's diff or
// goes in; Esc goes back the way the user came: the diff, a level, the
// details, the mode; Tab and the two digits move between the list and the
// details, the other panels wait (tdp K11).
func TestDiffInDetailsKeys(t *testing.T) {
	ta, files, changes := detailsApp(t)
	focused := func() *Panel { return ta.Screen().Focused() }
	ta.press("D", "Enter")
	if focused() != files {
		t.Error("Enter on the anchor went to the details")
	}
	ta.press("j", "D")
	if focused() != changes {
		t.Fatal("the diff key did not send the focus to the details")
	}
	if f := covCFooter(ta); !strings.Contains(f, "Esc:back") || !strings.Contains(f, "D|Enter:diff") {
		t.Errorf("the details' footer: %q", f)
	}
	ta.press("Enter") // the comparison of b.txt with a.txt
	if _, ok := ta.topView().(*viewerView); !ok {
		t.Fatalf("Enter on the details opened %T", ta.topView())
	}
	ta.press("Esc")
	if ta.top() != nil || focused() != changes {
		t.Fatal("Esc on the comparison did not go back to the details")
	}
	ta.press("Esc")
	if focused() != files || files.InMode() == nil || changes.InMode() == nil || ta.list.Anchor != "0" {
		t.Fatal("Esc on the details did not go back to the list, the mode kept")
	}

	// Tab and the digits: between the two only.
	ta.press("Tab")
	if focused() != changes {
		t.Error("Tab did not go to the details")
	}
	ta.press("Tab")
	if focused() != files {
		t.Error("Tab did not come back to the list")
	}
	ta.press("2")
	if focused() != changes {
		t.Error("2 did not go to the details")
	}
	ta.press("3")
	if focused() != changes || !strings.Contains(ta.screenText(), "Press [Esc] to leave Diffing first") {
		t.Error("3 left the mode's two panels")
	}
	ta.toasts, ta.inspectT = nil, nil
	ta.press("1")
	if focused() != files {
		t.Error("1 did not go back to the list")
	}

	// Folders: Enter goes in, a level at a time, and Esc back up.
	ta.press("Esc", "j", "j", "j", "j", "D", "j", "Enter")
	if focused() != changes {
		t.Fatal("Enter on the list did not send the focus to the details")
	}
	ta.press("j", "Enter") // sub/: in
	if text := ta.screenText(); !strings.Contains(text, "sub @ v1 ↔ sub @ v2") || !strings.Contains(text, "old.go") {
		t.Fatalf("Enter on sub/ did not go in:\n%s", text)
	}
	if f := covCFooter(ta); !strings.Contains(f, "Esc:up") {
		t.Errorf("a level in, Esc goes up: %q", f)
	}
	ta.press("Enter") // old.go's comparison
	if _, ok := ta.topView().(*viewerView); !ok {
		t.Fatalf("Enter on old.go opened %T", ta.topView())
	}
	ta.press("Esc", "Esc")
	if text := ta.screenText(); focused() != changes || !strings.Contains(text, "v1/ ↔ v2/") {
		t.Fatalf("Esc did not go back up a level:\n%s", text)
	}
	ta.press("Esc", "Esc")
	if files.InMode() != nil || changes.InMode() != nil || focused() != files {
		t.Error("Esc, Esc did not leave the mode from the list")
	}

	// The key reference and the Space menu of the details.
	ta.press("D", "k", "D", "?") // v2/ against v1/
	if text := ta.screenText(); !strings.Contains(text, "back to the list, still Diffing") {
		t.Errorf("the details' key reference:\n%s", text)
	}
	ta.press("Esc", "Space")
	if text := ta.screenText(); !strings.Contains(text, "back") || !strings.Contains(text, "its two sides") {
		t.Errorf("the details' Space menu:\n%s", text)
	}
}

// TestDiffInDetailsEnds: entering leaves what the app went into in the
// details and hides their own location row; an anchor gone from the list
// ends the mode, the details with it; the cursor moved by the app fills the
// details again, dropping what was gone into there; a panel with no details
// refuses the action (AddScreen).
func TestDiffInDetailsEnds(t *testing.T) {
	ta, files, changes := detailsApp(t)
	changes.Location = "the app's own place"
	ta.OpenDiff(changes, DiffItem{Anchor: "x", Other: "y", Items: func() ([]DiffItem, error) {
		return []DiffItem{{Mark: DiffAdded, Name: "the app's item"}}, nil
	}})
	ta.press("j", "j", "j", "j", "j", "D")
	if text := ta.screenText(); strings.Contains(text, "the app's item") || strings.Contains(text, "the app's own place") ||
		!strings.Contains(text, "Move to another file to diff with v1/") {
		t.Errorf("the app's own in the details:\n%s", text)
	}
	ta.press("j", "D", "j", "Enter")
	if changes.level() == nil {
		t.Fatal("not in sub/")
	}
	ta.list.Select("7") // the app moves the cursor: v3/
	ta.Update(nil)
	ta.settle()
	if changes.level() != nil || !strings.Contains(ta.screenText(), "No differences between v1/ and v3/") {
		t.Errorf("the details did not follow the cursor:\n%s", ta.screenText())
	}
	var rows []Row
	for _, r := range ta.list.Rows() {
		if r.ID != "5" {
			rows = append(rows, r)
		}
	}
	ta.list.SetRows(rows)
	ta.Update(nil)
	ta.settle()
	if files.InMode() != nil || changes.InMode() != nil || changes.over != nil {
		t.Error("the anchor gone did not end the mode in the details")
	}
	if !strings.Contains(ta.screenText(), "Diffing ended: v1/ is gone") {
		t.Error("no Warning toast")
	}

	defer func() {
		if recover() == nil {
			t.Error("a Diffing mode in no details was taken")
		}
	}()
	a := New("x")
	l := &List{}
	p := &Panel{Label: "List", Body: l}
	p.Actions = []Action{a.DiffModeAction(p, l, DiffMode{Key: "D", InDetails: true})}
	a.AddScreen(&Screen{Name: "S", Panels: []*Panel{p}})
}

// hintText is p's bottom-border hint as text, a key off marked with !.
func hintText(p *Panel) string {
	hs, off := p.hint()
	var out []string
	for i, h := range hs {
		s := h.Key + ":" + h.Desc
		if off[i] {
			s = "!" + s
		}
		out = append(out, s)
	}
	return strings.Join(out, " ")
}

// TestK11ModeHint: while a panel is in a mode its bottom-border hint is the
// mode's keys, not the panel's own actions, which do not work there; the
// movement keys and the core keys the footer has (Space, ?, Esc) are left
// out, a key that cannot be pressed now drawn off, or left out on a panel
// that hints only what can be pressed (tdp K11, panel.1, M6). A Diffing mode in
// the details writes each panel's own layer; out of the mode the panel's
// actions are back.
func TestK11ModeHint(t *testing.T) {
	ta, _ := compareApp(t)
	if h := hintText(ta.pods); !strings.Contains(h, "Enter:logs") {
		t.Fatalf("out of a mode: %q", h)
	}
	ta.press("C")
	if h := hintText(ta.pods); h != "!C|Enter:diff" {
		t.Errorf("in the Diffing mode, on the anchor: %q", h)
	}
	ta.press("j")
	if h := hintText(ta.pods); h != "C|Enter:diff" {
		t.Errorf("in the Diffing mode: %q", h)
	}
	ta.pods.HintPressableOnly = true
	ta.press("k")
	if h := hintText(ta.pods); h != "" {
		t.Errorf("pressable only, on the anchor: %q", h)
	}
	ta.pods.HintPressableOnly = false
	ta.press("Esc")
	if h := hintText(ta.pods); !strings.Contains(h, "Enter:logs") {
		t.Errorf("out of the mode again: %q", h)
	}

	// An app's own mode: its keys.
	ta = newTestApp(t)
	ta.press("D")
	if h := hintText(ta.pods); h != "Enter:drop" {
		t.Errorf("the app's Dragging mode: %q", h)
	}

	// In the details, each panel its own layer.
	ta, files, changes := detailsApp(t)
	ta.press("D", "j")
	if hf, hc := hintText(files), hintText(changes); hf != "D|Enter:diff" || hc != "D|Enter:diff" {
		t.Errorf("the list %q, the details %q", hf, hc)
	}
	ta.press("j") // c.txt, the same: nothing to go to, nothing to open
	if hf, hc := hintText(files), hintText(changes); hf != "!D|Enter:diff" || hc != "!D|Enter:diff" {
		t.Errorf("on a row the same: the list %q, the details %q", hf, hc)
	}
}
