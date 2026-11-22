package terminu

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// leadApp is a screen whose right side follows the leading panel that last
// had the focus: [1] CHANGES and [2] BRANCHES lead, a sign, then three
// places: DIFF, HISTORY and none while CHANGES leads; COMMITS, DESCRIPTION
// and FILES while BRANCHES does.
func leadApp(t *testing.T) (ta *testApp, s *Screen, ps map[string]*Panel) {
	t.Helper()
	ta = &testApp{App: anySize(New("test"))}
	ps = map[string]*Panel{}
	for _, n := range []string{"Changes", "Branches", "Diff", "History", "Commits", "Description", "Files"} {
		pg := &Pager{}
		pg.SetLines([]string{"the " + strings.ToLower(n)})
		ps[n] = &Panel{Label: n, Body: pg}
	}
	sign := &Panel{Label: "Repo", Body: &Pager{}, Sign: SignDim}
	s = &Screen{Name: "Git", Key: "G",
		Panels:    []*Panel{ps["Changes"], ps["Branches"], sign, ps["Commits"], ps["Description"], ps["Files"]},
		Following: []int{3, 4, 5},
		Leads: []Lead{
			{Panel: ps["Changes"], Shows: []*Panel{ps["Diff"], ps["History"], nil}},
			{Panel: ps["Branches"], Shows: []*Panel{ps["Commits"], ps["Description"], ps["Files"]}},
		}}
	ta.AddScreen(s)
	ta.resize(160, 30)
	return ta, s, ps
}

// TestL5Follow: panels that follow a leading panel (tdp L5, L5-4; from
// gittu): moving the focus to another leading panel changes what the
// following places show, hiding one and its neighbours sharing its room;
// the same every time for the same leading panel; moving the focus into the
// following places changes nothing. Numbers go by place: a place keeps its
// [N] whatever it shows, the footer and ? count the hidden one, whose digit
// does nothing while it is hidden.
func TestL5Follow(t *testing.T) {
	ta, s, ps := leadApp(t)
	focused := func() *Panel { return ta.Screen().Focused() }
	text := ta.screenText()
	for _, want := range []string{"[1] CHANGES", "[2] BRANCHES", "[3] DIFF", "[4] HISTORY", "Tab|1–5:panels"} {
		if !strings.Contains(text, want) {
			t.Errorf("CHANGES leading: no %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "COMMITS") || strings.Contains(text, "[5]") || !s.Hidden(5) {
		t.Errorf("the fifth place is not hidden while CHANGES leads:\n%s", text)
	}
	ta.press("5")
	if focused() != ps["Changes"] {
		t.Errorf("the hidden place's digit went to %s", focused().Label)
	}

	// BRANCHES takes the focus: the right side follows, the hidden place back.
	ta.press("2")
	text = ta.screenText()
	for _, want := range []string{"[3] COMMITS", "[4] DESCRIPTION", "[5] FILES"} {
		if !strings.Contains(text, want) {
			t.Errorf("BRANCHES leading: no %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "DIFF") || s.Hidden(5) {
		t.Errorf("CHANGES's panels still show:\n%s", text)
	}

	// Into the following places: nothing changes.
	ta.press("Tab")
	if focused() != ps["Commits"] {
		t.Fatalf("Tab went to %s", focused().Label)
	}
	ta.press("Tab", "Tab")
	if focused() != ps["Files"] || !strings.Contains(ta.screenText(), "[3] COMMITS") {
		t.Errorf("the focus moving into them changed what they show:\n%s", ta.screenText())
	}
	ta.press("?")
	if text := ta.screenText(); !strings.Contains(text, "1–5") {
		t.Errorf("? does not count every place:\n%s", text)
	}
	ta.press("Esc")

	// Back to CHANGES: the same as before, every time.
	ta.press("1")
	if text := ta.screenText(); !strings.Contains(text, "[3] DIFF") || strings.Contains(text, "FILES") || !s.Hidden(5) {
		t.Errorf("CHANGES again does not show the same:\n%s", text)
	}
	ta.press("3") // DIFF: CHANGES still leads
	if focused() != ps["Diff"] || !strings.Contains(ta.screenText(), "[4] HISTORY") {
		t.Errorf("3 with CHANGES leading: %s\n%s", focused().Label, ta.screenText())
	}
	ta.press("Tab", "Tab") // HISTORY, then past the hidden place and the sign: CHANGES
	if focused() != ps["Changes"] {
		t.Errorf("Tab did not pass the hidden place: %s", focused().Label)
	}
	if ps["Diff"].app != ta.App || ps["Files"].app != ta.App {
		t.Error("a following panel does not know its app")
	}

	// An app's Layout giving the hidden place room: it is not drawn there.
	s.Layout = func(w, h int) []Rect {
		var r []Rect
		for i := range 6 {
			r = append(r, Rect{X: i * (w / 6), W: w / 6, H: h})
		}
		return r
	}
	if !s.Hidden(5) || strings.Contains(ta.screenText(), "FILES") {
		t.Errorf("the hidden place is drawn where Layout put it:\n%s", ta.screenText())
	}

	// A Lead whose Shows do not match Following is refused.
	defer func() {
		if recover() == nil {
			t.Error("a Lead with too few Shows was taken")
		}
	}()
	New("x").AddScreen(&Screen{Name: "S", Panels: []*Panel{ps["Changes"], ps["Diff"]}, Following: []int{1},
		Leads: []Lead{{Panel: ps["Changes"], Shows: nil}}})
}

// TestL5DetailsK3: Enter may send the focus to the details when the
// details are on the screen whenever its panel has the focus (tdp K3, L5;
// from gittu): a leading panel's details in its own Shows, a following
// panel's in the Shows of the lead showing it; a details only another lead
// shows is refused, Enter there would go to a hidden panel.
func TestL5DetailsK3(t *testing.T) {
	build := func(set func(ps map[string]*Panel)) (*App, map[string]*Panel, error) {
		ps := map[string]*Panel{}
		for _, n := range []string{"Changes", "Branches", "Diff", "History", "Commits", "Description", "Files"} {
			ps[n] = &Panel{Label: n, Body: &Pager{}}
		}
		set(ps)
		a := anySize(New("x"))
		var err error
		func() {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("%v", r)
				}
			}()
			a.AddScreen(&Screen{Name: "S", // in BRANCHES's layout, as gittu starts it
				Panels:    []*Panel{ps["Changes"], ps["Branches"], ps["Commits"], ps["Description"], ps["Files"]},
				Following: []int{2, 3, 4},
				Leads: []Lead{
					{Panel: ps["Changes"], Shows: []*Panel{ps["Diff"], ps["History"], nil}},
					{Panel: ps["Branches"], Shows: []*Panel{ps["Commits"], ps["Description"], ps["Files"]}},
				}})
		}()
		return a, ps, err
	}
	toDetails := func(p, d *Panel) {
		p.Details = d
		p.Actions = []Action{{Key: "Enter", Label: "Show", Scope: PanelScope, ToDetails: func() bool { return true }, Do: func() tea.Cmd { return nil }}}
	}

	a, ps, err := build(func(ps map[string]*Panel) { toDetails(ps["Changes"], ps["Diff"]) })
	if err != nil {
		t.Fatalf("a lead's details in its own Shows was refused: %v", err)
	}
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	a.Update(keyMsg("Enter"))
	if f := a.Screen().Focused(); f != ps["Diff"] {
		t.Errorf("Enter on CHANGES went to %s, not its details", f.Label)
	}
	if _, _, err = build(func(ps map[string]*Panel) { toDetails(ps["Changes"], ps["Commits"]) }); err == nil {
		t.Error("a lead's details only another lead shows was taken")
	}
	if _, _, err = build(func(ps map[string]*Panel) { toDetails(ps["Commits"], ps["Description"]) }); err != nil {
		t.Errorf("a following panel's details shown by the same lead was refused: %v", err)
	}
	if _, _, err = build(func(ps map[string]*Panel) { toDetails(ps["Commits"], ps["History"]) }); err == nil {
		t.Error("a following panel's details another lead shows was taken")
	}
	if _, _, err = build(func(ps map[string]*Panel) { ps["Diff"].Actions = []Action{{Key: "q", Label: "Quit"}} }); err == nil {
		t.Error("a following panel's action on a core key was taken: its actions are not checked")
	}
}
