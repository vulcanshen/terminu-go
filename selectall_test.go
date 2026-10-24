package terminu

import (
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestCheckboxSelectAllPopup: a checkbox popup's first row is Select all,
// not an option: the cursor starts on the first option, stops on it; Space
// on it ticks every option, the unavailable too, on screen only, Enter
// writing back the options alone; it is ticked only while every option is;
// typing hides it, clearing brings it back (checkbox-12, 26 to 30).
func TestCheckboxSelectAllPopup(t *testing.T) {
	ta := newTestApp(t)
	var got []bool
	ta.Checks(Checks{Title: "Namespaces", Options: []string{"default", "kube-system", "kube-public", "monitoring"},
		Checked: []bool{false, true, true, false}, Unavailable: func(i int) bool { return i == 3 },
		Done: func(v []bool) tea.Cmd { got = v; return nil }})
	ta.settle()
	fv := ta.topView().(*finderView)
	if i, _, _ := fv.at(); i != 0 {
		t.Fatalf("the cursor opened on %d, not the first option", i)
	}
	if row := StripStyles(ta.rowWith("Select all")); !strings.Contains(row, glyphUnchecked+" Select all") {
		t.Errorf("Select all: %q", row)
	}
	ta.press("k", "Space")
	if !slices.Equal(fv.checked, []bool{true, true, true, true}) || got != nil {
		t.Errorf("Space on Select all: %v, written %v", fv.checked, got)
	}
	if !strings.Contains(strings.Join(ta.Frame(), "\n"), Fg(Green).Render("default")) || !strings.Contains(StripStyles(ta.rowWith("Select all")), glyphMarked) {
		t.Error("ticked, Select all is not ticked")
	}
	ta.press("j", "Space")
	if fv.checked[0] || strings.Contains(StripStyles(ta.rowWith("Select all")), glyphMarked) {
		t.Error("an option unticked, Select all stays ticked")
	}
	ta.press("Space")
	if !strings.Contains(StripStyles(ta.rowWith("Select all")), glyphMarked) {
		t.Error("ticked full one by one, Select all is not ticked")
	}
	ta.press("Tab", "k", "u", "b", "e")
	if strings.Contains(ta.screenText(), "Select all") {
		t.Errorf("Select all is drawn while filtering:\n%s", ta.screenText())
	}
	ta.press("Backspace", "Backspace", "Backspace", "Backspace", "Tab")
	if !strings.Contains(ta.screenText(), "Select all") {
		t.Error("Select all does not come back")
	}
	ta.press("Enter")
	if !slices.Equal(got, []bool{true, true, true, true}) {
		t.Errorf("written %v: the options alone", got)
	}
	// Opened with every option ticked, it is ticked; a single switch has none.
	ta.Checks(Checks{Title: "Two", Options: []string{"a", "b"}, Checked: []bool{true, true}})
	ta.settle()
	if !strings.Contains(StripStyles(ta.rowWith("Select all")), glyphMarked) {
		t.Error("opened all ticked, Select all is not")
	}
	ta.press("Esc")
	ta.Checks(Checks{Title: "One", Options: []string{"on"}})
	ta.settle()
	if strings.Contains(ta.screenText(), "Select all") {
		t.Error("a single switch has Select all")
	}
}

// TestCheckboxSelectAllInPlace: on a form and a value panel Select all is on
// the label's row, a focus item of its own, the options under it; Enter or
// Space on it ticks or unticks the group at once; the form's focus starts on
// the first option and lands on the first ticked one, never on it (form-49,
// form-62, checkbox-26 to 29).
func TestCheckboxSelectAllInPlace(t *testing.T) {
	ta := newTestApp(t)
	agent := &FormField{Label: "Agent", Options: []string{"forward", "x11"}, Multi: true}
	ta.Form(&Form{Title: "Host", Fields: []*FormField{agent}, Submit: func() (tea.Cmd, error) { return nil, nil }})
	ta.settle()
	fv := ta.topView().(*formView)
	if fv.focus.option != 0 {
		t.Fatalf("the form opened on %d, not the first option", fv.focus.option)
	}
	if row := StripStyles(ta.rowWith("Agent")); !strings.Contains(row, "Select all") {
		t.Errorf("Select all is not on the label's row: %q", row)
	}
	ta.press("k", "Space")
	if fv.focus.option != optAll || !slices.Equal(agent.Checked, []bool{true, true}) || !fv.dirty {
		t.Errorf("Select all on the form: focus %d, %v, changed %v", fv.focus.option, agent.Checked, fv.dirty)
	}
	ta.press("Enter")
	if !slices.Equal(agent.Checked, []bool{false, false}) {
		t.Errorf("Enter on Select all again: %v", agent.Checked)
	}
	ta.press("Esc", "Enter")

	x11 := &FormField{Label: "Agent", Options: []string{"forward", "x11"}, Multi: true, Checked: []bool{false, true}}
	ta.Form(&Form{Title: "Host", Fields: []*FormField{{Label: "Name", Value: func() string { return "x" }}, x11}, Focus: x11,
		Submit: func() (tea.Cmd, error) { return nil, nil }})
	ta.settle()
	if fv := ta.topView().(*formView); fv.focus.field != x11 || fv.focus.option != 1 {
		t.Errorf("landing on the field: option %d, not the first ticked", fv.focus.option)
	}
	ta.covBCloseAll()
	ta.Form(&Form{Title: "One", Fields: []*FormField{{Label: "Agent", Options: []string{"forward"}, Multi: true}},
		Submit: func() (tea.Cmd, error) { return nil, nil }})
	ta.settle()
	if strings.Contains(ta.screenText(), "Select all") {
		t.Error("a single switch on a form has Select all")
	}
	ta.covBCloseAll()

	changed := 0
	notify := &Field{Label: "Notify", Options: []string{"errors", "success"}, Multi: true, OnChange: func() tea.Cmd { changed++; return nil }}
	v := &Values{Fields: []*Field{notify}}
	lines := v.Draw(40, 3, true)
	if !strings.Contains(lines[0], "NOTIFY") || !strings.Contains(lines[0], "Select all") || !strings.Contains(lines[1], "errors") {
		t.Errorf("on a panel:\n%s", strings.Join(lines, "\n"))
	}
	v.Key("Enter", keyMsg("Enter"))
	if !slices.Equal(notify.Checked, []bool{true, true}) || changed != 1 {
		t.Errorf("Select all on a panel: %v, %d changes", notify.Checked, changed)
	}
	v.Key("j", keyMsg("j"))
	v.Key("Enter", keyMsg("Enter"))
	if lines := v.Draw(40, 3, true); strings.Contains(StripStyles(lines[0]), glyphMarked) {
		t.Error("an option unticked, Select all stays ticked on the panel")
	}
}

// colOf is the cell where s starts in the first line of lines holding it.
func colOf(lines []string, s string) int {
	for _, l := range lines {
		if i := strings.Index(l, s); i >= 0 {
			return Width(l[:i])
		}
	}
	return -1
}

// TestCheckboxSelectAllIndent: the options are two cells further in than
// Select all, on a form, a value panel and in a checkbox popup, where they
// stay so while filtering; with every option disabled Select all is too
// (checkbox-31, checkbox-32, form-14).
func TestCheckboxSelectAllIndent(t *testing.T) {
	ta := newTestApp(t)
	ta.Checks(Checks{Title: "Namespaces", Options: []string{"default", "kube-system"}})
	ta.settle()
	lines := strings.Split(ta.screenText(), "\n")
	all, opt := colOf(lines, "Select all"), colOf(lines, "kube-system")
	if all < 0 || opt != all+2 {
		t.Errorf("in the popup Select all at %d, an option at %d", all, opt)
	}
	ta.press("Tab", "k", "u")
	if got := colOf(strings.Split(ta.screenText(), "\n"), "kube-system"); got != opt {
		t.Errorf("filtering moved the option from %d to %d", opt, got)
	}
	ta.covBCloseAll()

	agent := &FormField{Label: "Agent", Options: []string{"forward", "x11"}, Multi: true}
	ta.Form(&Form{Title: "Host", Fields: []*FormField{agent}, Submit: func() (tea.Cmd, error) { return nil, nil }})
	ta.settle()
	lines = strings.Split(ta.screenText(), "\n")
	if all, opt := colOf(lines, "Select all"), colOf(lines, "x11"); all < 0 || opt != all+2 {
		t.Errorf("on a form Select all at %d, an option at %d", all, opt)
	}
	ta.covBCloseAll()

	v := &Values{Fields: []*Field{{Label: "Notify", Options: []string{"errors", "success"}, Multi: true}}}
	lines = v.Draw(40, 3, true)
	for i := range lines {
		lines[i] = StripStyles(lines[i])
	}
	if all, opt := colOf(lines, "Select all"), colOf(lines, "success"); all < 0 || opt != all+2 {
		t.Errorf("on a panel Select all at %d, an option at %d", all, opt)
	}

	off := &FormField{Label: "Agent", Options: []string{"forward", "x11"}, Multi: true, OptionDisabled: func(int) bool { return true }}
	ta.Form(&Form{Title: "Host", Fields: []*FormField{{Label: "Name", Value: func() string { return "x" }}, off},
		Submit: func() (tea.Cmd, error) { return nil, nil }})
	ta.settle()
	fv := ta.topView().(*formView)
	for _, it := range fv.fieldItems() {
		if it.field == off {
			t.Errorf("a group with every option disabled has a focus item: %d", it.option)
		}
	}
	if !strings.Contains(ta.rowWith("Select all"), Fg(Surface2).Render(glyphUnchecked+" Select all")) {
		t.Error("Select all is not drawn disabled")
	}
}
