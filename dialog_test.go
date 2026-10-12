package terminu

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// dialogOpeners open the dialogs of this file for the L4 screen test.
func dialogOpeners() map[string]func(*testApp) {
	lines := make([]string, 50)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d of the YAML, long enough to need some room on a narrow screen", i+1)
	}
	return map[string]func(*testApp){
		"viewer": func(ta *testApp) {
			ta.Viewer(&Viewer{Title: "YAML", Subject: "pod/nginx", Lines: lines, Select: &Selection{Key: "v", Name: "Visual"}})
		},
		"viewer mode": func(ta *testApp) {
			ta.Viewer(&Viewer{Title: "YAML", Subject: strings.Repeat("very long ", 20), Lines: lines, Select: &Selection{Key: "v", Name: "Visual"}})
			ta.settle()
			ta.press("v", "j", "w", "$")
		},
		"viewer tabs": func(ta *testApp) {
			ta.Viewer(&Viewer{Title: "DevTools", Tabs: []string{"Console", "Network", "Sources"}, Lines: lines[:3], Loading: true})
		},
		"confirm keys": func(ta *testApp) {
			ta.Confirm(Ask{Title: "Overwrite", Question: "Overwrite it?", Verb: "overwrite", Keys: []AskKey{{Key: "y", Desc: "overwrite", Accept: true}}}, nil)
		},
		"error": func(ta *testApp) { ta.Error("Save failed", strings.Repeat("hosts.yaml changed on disk. ", 12)) },
	}
}

func testViewer() *Viewer {
	var lines []string
	for i := range 40 {
		lines = append(lines, fmt.Sprintf("key%d: value %d", i, i))
	}
	return &Viewer{Title: "YAML", Subject: "pod/nginx", Lines: lines}
}

func TestNoteKeys(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(80, 20)
	copied := 0
	v := testViewer()
	v.Keys = []ViewerKey{{Key: "y", Desc: "copy", Do: func() tea.Cmd { copied++; return nil }}}
	v.Select = &Selection{Key: "v", Name: "Visual"}
	ta.Viewer(v)
	ta.settle()
	vv := ta.topView().(*viewerView)
	if h := vv.hint(); len(h) != 3 || h[0].Key != "y" || h[1].Key != "v" || h[2] != (KeyHint{"Esc", "close"}) {
		t.Errorf("the note's own keys come first, Esc:close last: %v", h)
	}
	ta.press("Enter", "Space")
	if ta.top() == nil {
		t.Fatal("Enter or Space closed the note")
	}
	ta.press("j", "j", "d")
	if v.Top() < 3 {
		t.Errorf("j and d scroll: top %d", v.Top())
	}
	ta.press("G")
	if v.Top() != len(v.Lines)-v.visible {
		t.Errorf("G goes to the bottom: top %d", v.Top())
	}
	if !strings.Contains(ta.screenText(), "of 40") {
		t.Error("no N-M of T")
	}
	ta.press("y")
	if copied != 1 {
		t.Error("the hotkey did not run")
	}
	ta.press("Esc")
	if ta.top() != nil {
		t.Error("Esc did not close the note")
	}
}

func TestNoteHotkeysAndMode(t *testing.T) {
	ta := newTestApp(t)
	got, ran := "", 0
	v := &Viewer{Title: "YAML", Lines: []string{"kind: Pod", "metadata:", "  name: nginx"},
		Keys:   []ViewerKey{{Key: "c", Desc: "count", Do: func() tea.Cmd { ran++; return nil }}},
		Select: &Selection{Key: "v", Name: "Visual", Copy: func(s string) tea.Cmd { got = s; return nil }}}
	ta.Viewer(v)
	ta.settle()
	ta.press("v")
	if !v.selecting {
		t.Fatal("v did not start the selection mode")
	}
	text := ta.screenText()
	if !strings.Contains(text, "┫Visual┣") {
		t.Errorf("the mode name is not on the frame:\n%s", text)
	}
	if !strings.Contains(strings.Join(ta.Frame(), "\n"), "38;2;249;226;175") {
		t.Error("the frame does not turn the mode colour")
	}
	ta.press("c", "Space")
	if ran != 0 || !v.selecting {
		t.Error("a hotkey ran in the mode (Rules K11)")
	}
	ta.press("?")
	if n, ok := ta.topView().(*noteView); !ok || !strings.Contains(n.title, "KEYS") || !strings.Contains(StripStyles(strings.Join(n.lines, "\n")), "copy the selection") {
		t.Fatalf("? in the mode lists the mode's keys: %T", ta.topView())
	}
	ta.press("Esc", "w", "j", "$", "y")
	if got != "kind: Pod\nmetadata:" {
		t.Errorf("copied %q", got)
	}
	if v.selecting {
		t.Error("y did not leave the mode")
	}
	ta.press("v", "Esc")
	if v.selecting || ta.top() == nil {
		t.Error("Esc leaves the mode and only the mode")
	}
}

func TestPopupTitleTabs(t *testing.T) {
	ta := newTestApp(t)
	var tabs []int
	v := &Viewer{Title: "DevTools", Tabs: []string{"Console", "Network"}, Lines: []string{"a"},
		OnTab: func(i int) tea.Cmd { tabs = append(tabs, i); return nil }}
	ta.Viewer(v)
	ta.settle()
	if !strings.Contains(ta.screenText(), "DEVTOOLS"+glyphRoundRight+"━"+glyphSlantLeft+"CONSOLE") {
		t.Errorf("one segment of line between the title and the tabs:\n%s", ta.screenText())
	}
	ta.press("l", "l", "h")
	if fmt.Sprint(tabs) != "[1 0 1]" {
		t.Errorf("h/l switch the tabs, wrapping: %v", tabs)
	}
}

func TestMenuEnterOpensFullText(t *testing.T) {
	ta := newTestApp(t)
	ta.Menu("Worklist", MenuSection{Actions: []Action{{Label: "Rotate the certificate", Stay: true, Do: func() tea.Cmd {
		return ta.Viewer(&Viewer{Title: "Task", Lines: []string{"Rotate the certificate before Friday."}})
	}}}})
	ta.settle()
	ta.press("Enter")
	if _, ok := ta.topView().(*viewerView); !ok {
		t.Fatalf("Enter did not open the full text: %T", ta.topView())
	}
	ta.press("Esc")
	if _, ok := ta.topView().(*menuView); !ok {
		t.Errorf("Esc goes back to the menu: %T", ta.topView())
	}
}

func TestMultiStepStacks(t *testing.T) {
	ta := newTestApp(t)
	chose := ""
	ta.Menu("Sort by", MenuSection{Actions: []Action{{Label: "Name", Stay: true, Do: func() tea.Cmd {
		return ta.Menu("Sort by Name", MenuSection{Actions: []Action{{Key: "1", Label: "Ascending", Do: func() tea.Cmd { chose = "asc"; return nil }}}})
	}}}})
	ta.settle()
	ta.press("Enter")
	if len(ta.stack) != 2 {
		t.Fatalf("the second step is a popup of its own on the first: %d layers", len(ta.stack))
	}
	ta.press("Esc")
	if m, ok := ta.topView().(*menuView); !ok || m.title != "Sort by" {
		t.Fatal("Esc goes back to the first step")
	}
	ta.press("Enter", "1")
	if chose != "asc" || ta.top() != nil {
		t.Errorf("the last step runs and closes them all: %q, top %v", chose, ta.top())
	}
}

func TestK12HotkeyCannotTakeMovementInMenu(t *testing.T) {
	ta := newTestApp(t)
	defer func() {
		if recover() == nil {
			t.Error("a menu took d, kept for half a page")
		}
	}()
	ta.Menu("Sort", MenuSection{Actions: []Action{{Key: "d", Label: "Descending"}}})
}

func TestK1DigitsFreeInPopups(t *testing.T) {
	ta := newTestApp(t)
	ran := false
	ta.Menu("Sort", MenuSection{Actions: []Action{{Key: "2", Label: "Descending", Do: func() tea.Cmd { ran = true; return nil }}}})
	ta.settle()
	ta.press("2")
	if !ran {
		t.Error("a digit hotkey in a popup did not run")
	}
}

func TestConfirmKeys(t *testing.T) {
	ta := newTestApp(t)
	accepted, kept := 0, 0
	ask := Ask{Title: "Overwrite", Question: "Overwrite it?", Verb: "overwrite",
		Keys: []AskKey{{Key: "y", Desc: "overwrite", Accept: true}, {Key: "n", Desc: "keep the file", Do: func() tea.Cmd { kept++; return nil }}}}
	ta.Confirm(ask, func() tea.Cmd { accepted++; return nil })
	ta.settle()
	ta.press("Space")
	if ta.top() == nil {
		t.Fatal("Space closed the confirm")
	}
	ta.press("?")
	ref := StripStyles(strings.Join(ta.topView().(*noteView).lines, "\n"))
	rows := map[string]bool{}
	for _, l := range strings.Split(ref, "\n") {
		rows[strings.Join(strings.Fields(l), " ")] = true
	}
	if !rows["y overwrite"] || !rows["n keep the file"] {
		t.Errorf("its own hotkeys are in its ?:\n%s", ref)
	}
	ta.press("Esc", "y")
	if accepted != 1 || ta.top() != nil {
		t.Errorf("y accepts: %d", accepted)
	}
	ta.Confirm(ask, func() tea.Cmd { accepted++; return nil })
	ta.settle()
	ta.press("n")
	if accepted != 1 || kept != 1 || ta.top() != nil {
		t.Errorf("n cancels and runs its Do: accepted %d, kept %d", accepted, kept)
	}
}

func TestF7LoadingGrows(t *testing.T) {
	ta := newTestApp(t)
	v := &Viewer{Title: "Log", Loading: true}
	ta.Viewer(v)
	ta.settle()
	h0 := ta.top().h
	if !strings.Contains(ta.screenText(), loadingFrames[0][:0]) || !ta.loading() {
		t.Fatal("a loading popup keeps its icon turning")
	}
	v.Lines = append(v.Lines, "one", "two", "three")
	ta.Update(spinMsg{})
	if ta.top().h != h0+3 {
		t.Errorf("a loading popup grows with its content: %d → %d", h0, ta.top().h)
	}
	v.Loading = false
	v.Lines = append(v.Lines, "four")
	ta.Update(spinMsg{})
	h1 := ta.top().h
	v.Lines = append(v.Lines, "five")
	ta.Update(spinMsg{})
	if ta.top().h != h1 {
		t.Error("the height changed after loading ended")
	}
}

func TestFormSaveFails(t *testing.T) {
	ta := newTestApp(t)
	f := testForm(&formResult{name: "web-01"})
	f.Submit = func() (tea.Cmd, error) { return nil, errors.New("hosts.yaml changed on disk since it was opened.") }
	ta.Form(f)
	ta.settle()
	ta.press("G", "Enter")
	n, ok := ta.topView().(*noteView)
	if !ok || n.title != "SAVE FAILED" {
		t.Fatalf("a failing save opens SAVE FAILED: %T", ta.topView())
	}
	ta.press("Enter")
	fv, ok := ta.topView().(*formView)
	if !ok || fv.focus.kind != itemButton {
		t.Error("closing it goes back to the form, the focus on the button")
	}
}
