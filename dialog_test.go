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
		"confirm": func(ta *testApp) {
			ta.Confirm(Ask{Title: "Overwrite", Question: "Overwrite it?", Verb: "overwrite"}, nil)
		},
		"error": func(ta *testApp) { ta.Error("Save failed", strings.Repeat("hosts.yaml changed on disk. ", 12)) },
		"terminal": func(ta *testApp) {
			term := &Terminal{Title: "Alterm", Subject: strings.Repeat("myhost", 30), in: &strings.Builder{}, started: true, History: true,
				Keys: []TerminalKey{{Key: "Alt-t", Desc: "hide"}}}
			term.mu.Lock()
			term.vt().write([]byte("中文 wide \U000F024B icons and a line long enough to wrap around the screen\r\n$ "))
			term.mu.Unlock()
			ta.open(&termView{t: term, a: ta.App})
		},
	}
}

func testViewer() *Viewer {
	var lines []string
	for i := range 40 {
		lines = append(lines, fmt.Sprintf("key%d: value %d", i, i))
	}
	return &Viewer{Title: "YAML", Subject: "pod/nginx", Lines: lines}
}

// rowWith is the styled screen row whose text holds s.
func (ta *testApp) rowWith(s string) string {
	for _, l := range ta.Frame() {
		if strings.Contains(StripStyles(l), s) {
			return l
		}
	}
	return ""
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
	if v.Top() != 2+v.visible/2 {
		t.Errorf("j and d scroll: top %d", v.Top())
	}
	if strings.Contains(ta.rowWith(fmt.Sprintf("key%d:", v.Top())), "48;2;164;192;250") {
		t.Error("a viewer has no cursor outside the selection mode")
	}
	if want := fmt.Sprintf("%d-%d of 40", v.Top()+1, v.Top()+v.visible); !strings.Contains(ta.screenText(), want) {
		t.Errorf("no %s, the lines in sight", want)
	}
	ta.press("G")
	if v.Top() != len(v.Lines)-v.visible {
		t.Errorf("G goes to the bottom: top %d", v.Top())
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
	if !v.inMode || v.selecting || v.line != 0 || v.col != 0 {
		t.Fatalf("the first v brings the cursor out at the start of the first line in sight, selecting nothing: %v %v %d,%d", v.inMode, v.selecting, v.line, v.col)
	}
	if h := ta.topView().(*viewerView).hint(); fmt.Sprint(h) != "[{v select} {Esc cancel}]" {
		t.Errorf("hint before selecting: %v", h)
	}
	if !strings.Contains(ta.rowWith("ind: Pod"), "48;2;164;192;250") {
		t.Error("the cursor is a block of the layer colour")
	}
	text := ta.screenText()
	if !strings.Contains(text, "┫Visual┣") {
		t.Errorf("the mode name is not on the frame:\n%s", text)
	}
	if !strings.Contains(strings.Join(ta.Frame(), "\n"), "38;2;249;226;175") {
		t.Error("the frame does not turn the mode colour")
	}
	ta.press("c", "Space", "y")
	if ran != 0 || !v.inMode || got != "" {
		t.Error("a hotkey ran in the mode (Rules K11), or y copied before the selection started")
	}
	ta.press("v")
	if !v.selecting {
		t.Fatal("the second v did not start the selection")
	}
	if h := ta.topView().(*viewerView).hint(); fmt.Sprint(h) != "[{y copy} {Esc cancel}]" {
		t.Errorf("hint while selecting: %v", h)
	}
	ta.press("?")
	if n, ok := ta.topView().(*noteView); !ok || !strings.Contains(n.title, "KEYS") || !strings.Contains(StripStyles(strings.Join(n.lines, "\n")), "copy the selection") {
		t.Fatalf("? in the mode lists the mode's keys: %T", ta.topView())
	}
	ta.press("Esc", "w", "j", "v", "$", "y")
	if got != "kind: Pod\nmetadata:" {
		t.Errorf("v again once selecting does nothing; copied %q", got)
	}
	if v.inMode || v.selecting {
		t.Error("y leaves the whole mode")
	}
	if strings.Contains(ta.rowWith("metadata:"), "48;2;164;192;250") {
		t.Error("the cursor goes with the mode")
	}
	ta.press("v", "w", "w", "v", "e", "y") // w stops at the colon first, as vim's
	if got != "Pod" {
		t.Errorf("the selection starts at the cursor: %q", got)
	}
	ta.press("v", "v", "Esc")
	if v.inMode || v.selecting || ta.top() == nil {
		t.Error("Esc leaves the whole mode and only the mode")
	}
}

func TestViewerSelectScrollPosition(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(80, 20)
	v := testViewer()
	v.Select = &Selection{Key: "v", Name: "Visual"}
	ta.Viewer(v)
	ta.settle()
	ta.press("d")
	top := v.Top()
	ta.press("v")
	if v.line != top {
		t.Errorf("the cursor comes out on the first line in sight: %d, top %d", v.line, top)
	}
	ta.press("j", "j")
	if want := fmt.Sprintf("%d of 40", top+3); !strings.Contains(ta.screenText(), want) {
		t.Errorf("in the mode the scroll position is the cursor's line, %s:\n%s", want, ta.screenText())
	}
	ta.press("G")
	if v.line != 39 || v.Top() != len(v.Lines)-v.visible {
		t.Errorf("the view follows the cursor: line %d, top %d", v.line, v.Top())
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
	ta.Menu("Worklist", MenuSection{Actions: []Action{{Label: "Rotate the certificate", Detail: true, Do: func() tea.Cmd {
		return ta.Viewer(&Viewer{Title: "Task", Lines: []string{"Rotate the certificate before Friday."}})
	}}}})
	ta.settle()
	if h := ta.topView().(*menuView).hint(); h[0] != (KeyHint{"Enter", "detail"}) {
		t.Errorf("a worklist's hint is Enter:detail: %v", h)
	}
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

// TestConfirmKeys: Enter and Esc are the only answers (Rules F6): y and n
// do nothing, and its ? lists just the two.
func TestConfirmKeys(t *testing.T) {
	ta := newTestApp(t)
	accepted := 0
	ask := Ask{Title: "Overwrite", Question: "Overwrite it?", Verb: "overwrite"}
	ta.Confirm(ask, func() tea.Cmd { accepted++; return nil })
	ta.settle()
	ta.press("Space", "y", "n")
	if ta.top() == nil || accepted != 0 {
		t.Fatalf("Space, y or n answered the confirm: open %v, accepted %d", ta.top() != nil, accepted)
	}
	ta.press("?")
	ref := StripStyles(strings.Join(ta.topView().(*noteView).lines, "\n"))
	var rows []string
	for _, l := range strings.Split(ref, "\n") {
		if l = strings.Join(strings.Fields(l), " "); l != "" {
			rows = append(rows, l)
		}
	}
	if strings.Join(rows, "|") != "Enter overwrite|Esc cancel" {
		t.Errorf("its ? lists only Enter and Esc:\n%s", ref)
	}
	ta.press("Esc", "Enter")
	if accepted != 1 || ta.top() != nil {
		t.Errorf("Enter accepts: %d", accepted)
	}
	ta.Confirm(ask, func() tea.Cmd { accepted++; return nil })
	ta.settle()
	ta.press("Esc")
	if accepted != 1 || ta.top() != nil {
		t.Errorf("Esc cancels: accepted %d", accepted)
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
	ta.press("Tab", "Enter")
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

func TestFormSkipDisabledOption(t *testing.T) {
	ta := newTestApp(t)
	order := &FormField{Label: "Order", Options: []string{"asc", "random", "desc"}, OptionDisabled: func(i int) bool { return i == 1 }}
	ta.Form(&Form{Title: "Sort", Fields: []*FormField{order}, Button: "Save", Submit: func() (tea.Cmd, error) { return nil, nil }})
	ta.settle()
	fv := ta.topView().(*formView)
	ta.press("j")
	if fv.focus.option != 2 {
		t.Errorf("j from asc stopped on option %d, not desc", fv.focus.option)
	}
	if row := ta.rowWith("random"); !strings.Contains(row, "38;2;88;91;112") {
		t.Errorf("a disabled option is drawn in Surface2: %q", row)
	}
}

func TestPopupTitleCutKeepsCapsule(t *testing.T) {
	cap := titleCapsule("", "FILE — /srv/projects/showcase/assets", Blue, 20)
	if !strings.Contains(StripStyles(cap), "…") {
		t.Fatalf("the title is not cut: %q", StripStyles(cap))
	}
	// Nothing may end the capsule's background before its "…".
	body := cap[strings.Index(cap, "FILE"):]
	if reset := strings.Index(body, "\x1b[m"); reset >= 0 && reset < strings.Index(body, "…") ||
		strings.Contains(body[:strings.Index(body, "…")], "\x1b[0m") {
		t.Errorf("the capsule's background ends before its …: %q", cap)
	}
	if got := Truncate("plain text here", 8); strings.Contains(got, "\x1b") {
		t.Errorf("Truncate added a style to plain text: %q", got)
	}
}
