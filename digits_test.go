package terminu

import (
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// digitsApp is one screen, Monitor: [1] PINS, its pages on the digits
// (Panel.Digits), and a sign that does not count; got are the digits Do ran
// for.
func digitsApp(t *testing.T, pages *int) (ta *testApp, s *Screen, pins *Panel, got *[]int) {
	t.Helper()
	ta = &testApp{App: anySize(New("test"))}
	got = &[]int{}
	pg := &Pager{}
	pg.SetLines([]string{"cells"})
	pins = &Panel{Label: "Pins", Body: pg, Tabs: []string{"Stats"},
		Digits: &Digits{Count: func() int { return *pages }, Hint: "page", Desc: "go to page N",
			Do: func(n int) tea.Cmd { *got = append(*got, n); return nil }},
		Actions: []Action{
			{Key: "Enter", Label: "Open", Desc: "the pin", Scope: PanelScope, Hint: "open", Do: ta.record("open")},
			{Key: "x", Label: "Remove", Desc: "the pin", Scope: PanelScope, Hint: "remove", Do: ta.record("remove")},
		}}
	sign := &Panel{Label: "Legend", Body: &Pager{}, Sign: SignDim}
	s = ta.AddScreen(&Screen{Name: "Monitor", Key: "M", Panels: []*Panel{pins, sign}})
	ta.resize(100, 32)
	return ta, s, pins, got
}

// screenLine is the first line of the screen holding want, "" for none.
func (ta *testApp) screenLine(want string) string {
	for _, l := range strings.Split(ta.screenText(), "\n") {
		if strings.Contains(l, want) {
			return l
		}
	}
	return ""
}

// TestDigitsPages: on a screen where it is the one place with a number, a
// sign not counting, the panel's Digits take 1–9: the digits up to Count run
// Do, the others nothing; the hint writes the range after Enter and h|l and
// before the panel's actions, 1–9 above nine, nothing for one (Panel.Digits,
// tdp K1).
func TestDigitsPages(t *testing.T) {
	pages := 3
	ta, _, _, got := digitsApp(t, &pages)
	ta.press("2", "3", "4", "9", "1")
	if !slices.Equal(*got, []int{2, 3, 1}) {
		t.Errorf("1–3 pressed 2 3 4 9 1: Do ran for %v", *got)
	}
	hint := ta.screenLine("1–3:page")
	e, h, d, x := strings.Index(hint, "Enter:open"), strings.Index(hint, "h|l:tab"), strings.Index(hint, "1–3:page"), strings.Index(hint, "x:remove")
	if e < 0 || h < e || d < h || x < d {
		t.Errorf("the hint: %q", hint)
	}
	pages = 12
	*got = nil
	ta.press("9")
	if !strings.Contains(ta.screenText(), "1–9:page") || !slices.Equal(*got, []int{9}) {
		t.Errorf("twelve pages: Do ran for %v, the hint %q", *got, ta.screenLine(":page"))
	}
	pages = 1
	*got = nil
	ta.press("1", "2")
	if strings.Contains(ta.screenText(), ":page") || !slices.Equal(*got, []int{1}) {
		t.Errorf("one page: Do ran for %v, the hint %q", *got, ta.screenLine(":page"))
	}
	pages = 0
	*got = nil
	ta.press("1")
	if len(*got) != 0 {
		t.Errorf("no page: Do ran for %v", *got)
	}
}

// TestDigitsKeyReference: the key reference lists the range in its move
// section, after the tabs' row, with Desc, and no "jump to panel N"; the
// Space menu has no row for them (Panel.Digits).
func TestDigitsKeyReference(t *testing.T) {
	pages := 3
	ta, _, _, _ := digitsApp(t, &pages)
	ta.press("?")
	lines := strings.Split(ta.screenText(), "\n")
	at := func(want string) int {
		return slices.IndexFunc(lines, func(l string) bool { return strings.Contains(l, want) })
	}
	tabs, row, core := at("previous tab, next tab"), at("go to page N"), at("what you can do here")
	if tabs < 0 || row < tabs || core < row || !strings.Contains(lines[row], "1–3 ") {
		t.Errorf("the key reference:\n%s", ta.screenText())
	}
	if at("jump to panel N") >= 0 {
		t.Error("the key reference has jump to panel N")
	}
	ta.press("Esc", "Space")
	for _, l := range strings.Split(ta.screenText(), "\n") {
		// The panel's own hint shows under the menu, dimmed.
		if (strings.Contains(l, "page") || strings.Contains(l, "1–3")) && !strings.Contains(l, "x:remove") {
			t.Errorf("the Space menu has the digits: %q", l)
		}
	}
}

// TestDigitsSecondPanel: once the app puts another panel on the screen, the
// digits jump to the panels again and the range leaves the hint; with it
// gone they are the panel's again. A hidden place (L5) counts: its digit does
// nothing, as on any screen (Panel.Digits, tdp K1).
func TestDigitsSecondPanel(t *testing.T) {
	pages := 3
	ta, s, pins, got := digitsApp(t, &pages)
	other := &Panel{Label: "Other", Body: &Pager{}}
	s.Panels = append(s.Panels, other)
	ta.press("2")
	if ta.Screen().Focused() != other || len(*got) != 0 {
		t.Errorf("2 with a second panel: the focus on %s, Do ran for %v", ta.Screen().Focused().Label, *got)
	}
	if text := ta.screenText(); strings.Contains(text, ":page") || !strings.Contains(text, "Tab|1–2:panels") {
		t.Errorf("with a second panel:\n%s", text)
	}
	ta.press("1")
	s.Panels = s.Panels[:2]
	ta.press("2")
	if ta.Screen().Focused() != pins || !slices.Equal(*got, []int{2}) || !strings.Contains(ta.screenText(), "1–3:page") {
		t.Errorf("the second panel gone: the focus on %s, Do ran for %v", ta.Screen().Focused().Label, *got)
	}

	lead := &Panel{Label: "Pins", Body: &Pager{}, Digits: pins.Digits}
	ta.AddScreen(&Screen{Name: "Lead", Key: "L", Panels: []*Panel{lead, {Label: "Place", Body: &Pager{}}},
		Following: []int{1}, Leads: []Lead{{Panel: lead, Shows: []*Panel{nil}}}})
	ta.press("L")
	*got = nil
	ta.press("2")
	if ta.Screen().Focused() != lead || len(*got) != 0 || strings.Contains(ta.screenText(), ":page") {
		t.Errorf("a hidden place: the focus on %s, Do ran for %v", ta.Screen().Focused().Label, *got)
	}
}

// TestDigitsModeAndFilter: in a mode the digits do what they do on any
// panel, nothing here, and the range leaves the hint; typing in a filter they
// are typed (Panel.Digits, tdp K8, K11).
func TestDigitsModeAndFilter(t *testing.T) {
	pages := 3
	ta, _, pins, got := digitsApp(t, &pages)
	pins.EnterMode(&Mode{Name: "Drag", Exit: "cancel", Keys: []ModeKey{{Key: "Enter", Desc: "drop", Do: func(string) tea.Cmd { return nil }}}})
	ta.settle()
	ta.press("2")
	if len(*got) != 0 || strings.Contains(ta.screenText(), ":page") {
		t.Errorf("in a mode: Do ran for %v, the hint %q", *got, ta.screenLine(":page"))
	}
	ta.press("Esc")

	l := &List{Columns: []Column{{Title: "Name"}}, Filterable: true}
	l.SetRows([]Row{{ID: "a", Cells: []string{"pin-2"}}, {ID: "b", Cells: []string{"pin-3"}}})
	pins.Body = l
	ta.press("/", "2")
	if !l.Typing() || len(*got) != 0 || len(l.view) != 1 {
		t.Errorf("typing in the filter: Do ran for %v, %d rows", *got, len(l.view))
	}
}

// TestDigitsRefused: Run refuses Digits on a screen where another place
// has a number, a following place's panel included, a sign not counting, and
// Digits with no Count or Do; AddScreen panics on a sign with Digits
// (Panel.Digits, tdp K1, panel.17).
func TestDigitsRefused(t *testing.T) {
	d := func() *Digits {
		return &Digits{Count: func() int { return 2 }, Hint: "page", Desc: "go to page N", Do: func(int) tea.Cmd { return nil }}
	}
	check := func(s *Screen) error {
		a := New("x")
		a.AddScreen(s)
		return a.checkDigits()
	}
	if err := check(&Screen{Name: "A", Panels: []*Panel{{Label: "Pins", Digits: d()}, {Label: "Legend", Sign: SignLit}}}); err != nil {
		t.Errorf("one panel and a sign: %v", err)
	}
	if err := check(&Screen{Name: "A", Panels: []*Panel{{Label: "Pins", Digits: d()}, {Label: "Other"}}}); err == nil || !strings.Contains(err.Error(), "takes the digits") {
		t.Errorf("two panels: %v", err)
	}
	// Shown is not on the screen yet: the first leading panel has the focus.
	shown := &Panel{Label: "Shown", Digits: d()}
	lead, other := &Panel{Label: "Lead"}, &Panel{Label: "Other"}
	if err := check(&Screen{Name: "A", Panels: []*Panel{lead, other, {Label: "Place"}}, Following: []int{2},
		Leads: []Lead{{Panel: lead, Shows: []*Panel{nil}}, {Panel: other, Shows: []*Panel{shown}}}}); err == nil || !strings.Contains(err.Error(), "takes the digits") {
		t.Errorf("a following place's panel: %v", err)
	}
	if err := check(&Screen{Name: "A", Panels: []*Panel{{Label: "Pins", Digits: &Digits{Hint: "page"}}}}); err == nil || !strings.Contains(err.Error(), "need a Count and a Do") {
		t.Errorf("no Count or Do: %v", err)
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("a sign with Digits was taken")
			}
		}()
		New("x").AddScreen(&Screen{Name: "A", Panels: []*Panel{{Label: "Pins"}, {Label: "Legend", Sign: SignLit, Digits: d()}}})
	}()
}
