package terminu

import (
	"strings"
	"testing"
)

// signApp is a screen of a sign, a list, a dim sign and a log: KEYS (lit)
// first, then [1] PLAYGROUND, NEXT (dim), [2] LOG.
func signApp(t *testing.T) (ta *testApp, keys, play, next, log *Panel) {
	t.Helper()
	ta = &testApp{App: anySize(New("test"))}
	ta.list = &List{Columns: []Column{{Title: "Name"}}}
	ta.list.SetRows([]Row{{ID: "a", Cells: []string{"a"}}, {ID: "b", Cells: []string{"b"}}})
	keysText := &Pager{}
	keysText.SetLines([]string{"h|l  move", "j    drop", "k    rotate"})
	nextText := &Pager{}
	nextText.SetLines([]string{"the next piece"})
	logText := &Pager{}
	logText.SetLines([]string{"started"})
	keys = &Panel{Label: "Keys", Body: keysText, Sign: SignLit}
	play = &Panel{Label: "Playground", Body: ta.list}
	next = &Panel{Label: "Next", Body: nextText, Sign: SignDim}
	log = &Panel{Label: "Log", Body: logText}
	ta.AddScreen(&Screen{Name: "Game", Key: "G", Panels: []*Panel{keys, play, next, log}})
	ta.resize(160, 30)
	return ta, keys, play, next, log
}

// TestSignFrame: a sign is drawn as a panel without the focus, Surface2's
// light line, its label alone, no [N]: only the panels that can take the
// focus are numbered, [1] and [2] here; no hint, no scroll position (tdp
// panel.17, panel-188).
func TestSignFrame(t *testing.T) {
	ta, keys, play, _, log := signApp(t)
	text := ta.screenText()
	for _, want := range []string{"KEYS", "[1] PLAYGROUND", "NEXT", "[2] LOG", "Tab|1–2:panels"} {
		if !strings.Contains(text, want) {
			t.Errorf("no %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "] KEYS") || strings.Contains(text, "] NEXT") || strings.Contains(text, "[3]") || strings.Contains(text, "[4]") {
		t.Errorf("a sign has a number:\n%s", text)
	}
	if !strings.Contains(text, "┏") || strings.Count(text, "┏") != 1 {
		t.Errorf("not one heavy frame:\n%s", text)
	}
	if c := covBCells(keys.draw(0, 40, 6, false, false)[0])[0]; c.fg != covBRGB(Surface2) {
		t.Errorf("a sign's frame is not Surface2: %s", c.fg)
	}
	if top := StripStyles(keys.draw(0, 40, 6, true, true)[0]); !strings.HasPrefix(top, "┌") || !strings.Contains(top, "KEYS") {
		t.Errorf("a sign drawn as focused: %q", top)
	}
	long := &Pager{}
	long.SetLines(strings.Split(strings.Repeat("line\n", 30), "\n"))
	keys.Body = long
	if bottom := StripStyles(keys.draw(0, 40, 6, false, false)[5]); strings.Contains(bottom, "of") {
		t.Errorf("a sign has a scroll position: %q", bottom)
	}
	if ta.Screen().Focused() != play || ta.Screen().num(3) != 2 || log == nil {
		t.Error("the focus is not on the first panel that can take it")
	}
}

// TestSignNoFocus: a sign never takes the focus: Tab and Shift-Tab pass it
// by, the digits count only the panels that can take the focus, App.Focus
// leaves it, and the Space menu and ? are the focused panel's (tdp
// panel.17, K1, K2, panel-189). A sign with actions or tabs, a screen of
// signs alone and a sign as a list's details are refused.
func TestSignNoFocus(t *testing.T) {
	ta, keys, play, next, log := signApp(t)
	ta.ShiftTab = true
	focused := func() *Panel { return ta.Screen().Focused() }
	for _, c := range []struct {
		key  string
		want *Panel
	}{{"Tab", log}, {"Tab", play}, {"Shift-Tab", log}, {"Shift-Tab", play}, {"2", log}, {"1", play}, {"3", play}, {"4", play}} {
		ta.press(c.key)
		if focused() != c.want {
			t.Errorf("%s: the focus on %s", c.key, focused().Label)
		}
	}
	ta.Focus(keys)
	ta.Focus(next)
	if focused() != play {
		t.Errorf("App.Focus gave a sign the focus: %s", focused().Label)
	}
	other := ta.AddScreen(&Screen{Name: "Other", Key: "O", Panels: []*Panel{{Label: "Only", Body: &Pager{}}}})
	ta.press("O")
	ta.Focus(keys)
	if ta.Screen() != other {
		t.Error("App.Focus on a sign switched the screen")
	}
	ta.switchScreen(0)
	ta.press("?")
	jump := ""
	for _, l := range strings.Split(ta.screenText(), "\n") {
		if strings.Contains(l, "jump to panel N") {
			jump = l
		}
	}
	if !strings.Contains(jump, "1–2 ") || !strings.Contains(ta.screenText(), "[1] PLAYGROUND") {
		t.Errorf("the key reference: %q", jump)
	}

	refused := func(what string, s *Screen) {
		t.Helper()
		defer func() {
			if recover() == nil {
				t.Errorf("%s was taken", what)
			}
		}()
		New("x").AddScreen(s)
	}
	refused("a sign with actions", &Screen{Name: "S", Panels: []*Panel{{Label: "L"}, {Label: "K", Sign: SignLit, Actions: []Action{{Key: "a", Label: "A"}}}}})
	refused("a sign with tabs", &Screen{Name: "S", Panels: []*Panel{{Label: "L"}, {Label: "K", Sign: SignDim, Tabs: []string{"More"}}}})
	refused("a screen of signs", &Screen{Name: "S", Panels: []*Panel{{Label: "K", Sign: SignLit}}})
	sign := &Panel{Label: "K", Sign: SignLit}
	l := &List{}
	lp := &Panel{Label: "L", Body: l, Details: sign}
	lp.Actions = []Action{{Key: "Enter", Label: "Open", ToDetails: func() bool { return true }}}
	refused("a sign as the details", &Screen{Name: "S", Panels: []*Panel{lp, sign}})
}

// TestSignLight: a lit sign is never dimmed without the focus, a dim one
// always is, and under a popup both dim with everything (tdp panel.17,
// T2, F8, panel-190).
func TestSignLight(t *testing.T) {
	ta, keys, _, next, _ := signApp(t)
	row := func(p *Panel) string { return p.draw(0, 40, 6, false, false)[1] }
	if strings.Contains(row(keys), covBRGB(DimColor(Text))) || covBCells(row(keys))[2].fg == covBRGB(DimColor(Text)) {
		t.Errorf("the lit sign is dimmed: %q", row(keys))
	}
	if c := covBCells(row(next))[2]; c.fg == "" || c.fg == covBRGB(Text) {
		t.Errorf("the dim sign is not dimmed: %+v", c)
	}
	// On the screen: the lit sign's text with no colour of its own, dimmed
	// under the Space menu.
	fg := func() string {
		for _, l := range ta.Frame() {
			if strings.Contains(StripStyles(l), "h|l  move") {
				for _, c := range covBCells(l) {
					if c.text == "h" {
						return c.fg
					}
				}
			}
		}
		return "?"
	}
	if got := fg(); got != "" {
		t.Errorf("the lit sign on the screen: %q", got)
	}
	ta.press("Space")
	if got := fg(); got == "" || got == "?" {
		t.Errorf("the lit sign under a popup: %q", got)
	}
}

// TestSignFits: a sign does not scroll: a Pager's lines that do not fit are
// cut at the bottom, the last row shown ending in …; lines that fit are
// left as they are (tdp panel.17, panel-191).
func TestSignFits(t *testing.T) {
	_, keys, _, _, _ := signApp(t)
	rows := keys.draw(0, 20, 5, false, false) // 3 rows inside
	if got := StripStyles(rows[3]); got != "│ k    rotate      │" {
		t.Errorf("lines that fit changed: %q", got)
	}
	p := keys.Body.(*Pager)
	p.SetLines([]string{"one", "two", "three", "four", "a very long line here, longer than the sign"})
	p.top = 2 // as an Append with Follow would leave it: a sign shows the top
	rows = keys.draw(0, 20, 5, false, false)
	for i, want := range []string{"│ one              │", "│ two              │", "│ three…           │"} {
		if got := StripStyles(rows[i+1]); got != want {
			t.Errorf("row %d: %q, want %q", i, got, want)
		}
	}
	p.SetLines([]string{"one", "two", "a row as wide as it", "four"})
	if got := StripStyles(keys.draw(0, 20, 5, false, false)[3]); got != "│ a row as wide a… │" { // 16 cells of text
		t.Errorf("a full row cut: %q", got)
	}
}
