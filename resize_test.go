package terminu

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// The layouts the resize tests use, in a panel area of w × h.
func sideBySide2(w, h int) []Rect { c := w / 2; return []Rect{{0, 0, c, h}, {c, 0, w - c, h}} }
func stacked2(w, h int) []Rect    { r := h / 2; return []Rect{{0, 0, w, r}, {0, r, w, h - r}} }
func columns3(w, h int) []Rect {
	c := w / 3
	return []Rect{{0, 0, c, h}, {c, 0, c, h}, {2 * c, 0, w - 2*c, h}}
}
func grid4(w, h int) []Rect {
	c, r := w/2, h/2
	return []Rect{{0, 0, c, r}, {c, 0, w - c, r}, {0, r, c, h - r}, {c, r, w - c, h - r}}
}
func twoOverOne(w, h int) []Rect {
	c, r := w/2, h/2
	return []Rect{{0, 0, c, r}, {c, 0, w - c, r}, {0, r, w, h - r}}
}

// resizeApp is one screen, Grid, of panels named by labels laid out by
// layout, on a terminal of w × h: the panel area is w × (h-1), the footer
// under it.
func resizeApp(t *testing.T, w, h int, layout func(w, h int) []Rect, labels ...string) (*testApp, *Screen, []*Panel) {
	t.Helper()
	ta := &testApp{App: anySize(New("test"))}
	var ps []*Panel
	for _, l := range labels {
		pg := &Pager{}
		pg.SetLines([]string{strings.ToLower(l)})
		ps = append(ps, &Panel{Label: l, Body: pg})
	}
	s := &Screen{Name: "Grid", Key: "R", Panels: ps, Layout: layout}
	ta.AddScreen(s)
	ta.resize(w, h)
	return ta, s, ps
}

func (ta *testApp) rectOf(t *testing.T, p *Panel) Rect {
	t.Helper()
	r, ok := ta.panelRect(p)
	if !ok {
		t.Fatalf("%s is not drawn", p.Label)
	}
	return r
}

// showing are the toasts showing, oldest first, as "TITLE: text".
func (ta *testApp) showing() []string {
	var out []string
	for _, l := range ta.showingToasts() {
		v := l.view.(*toastView)
		out = append(out, v.title()+": "+strings.Join(v.text, " "))
	}
	return out
}

func (ta *testApp) newestToast(t *testing.T) (*layer, *toastView) {
	t.Helper()
	s := ta.showingToasts()
	if len(s) == 0 {
		t.Fatal("no toast")
	}
	l := s[len(s)-1]
	return l, l.view.(*toastView)
}

// TestScreen37WiderNarrower: > makes the focused panel 2 columns wider, < 2
// narrower, the line on its edge moving (tdp screen.9, screen-37).
func TestScreen37WiderNarrower(t *testing.T) {
	ta, _, ps := resizeApp(t, 100, 31, nil, "Left", "Right")
	ta.press(">")
	if l, r := ta.rectOf(t, ps[0]), ta.rectOf(t, ps[1]); l.W != 52 || r.X != 52 || r.W != 48 {
		t.Errorf(">: [1] %+v, [2] %+v; want [1] 52 wide, [2] from 52, 48 wide", l, r)
	}
	ta.press("<", "<")
	if l, r := ta.rectOf(t, ps[0]), ta.rectOf(t, ps[1]); l.W != 48 || r.X != 48 || r.W != 52 {
		t.Errorf("< <: [1] %+v, [2] %+v", l, r)
	}
	// Drawn so: [2]'s frame starts at column 48.
	if row := StripStyles(ta.Frame()[0]); !strings.HasPrefix(string([]rune(row)[48:]), "┌") {
		t.Errorf("[2]'s frame is not at column 48: %q", row)
	}
	if len(ta.showing()) != 0 {
		t.Errorf("a line that moves says nothing: %v", ta.showing())
	}
}

// TestScreen38TallerShorter: + makes the focused panel 1 row taller, - 1
// row shorter (tdp screen.9, screen-38).
func TestScreen38TallerShorter(t *testing.T) {
	ta, _, ps := resizeApp(t, 100, 31, stacked2, "Top", "Bottom")
	ta.press("+")
	if a, b := ta.rectOf(t, ps[0]), ta.rectOf(t, ps[1]); a.H != 16 || b.Y != 16 || b.H != 14 {
		t.Errorf("+: [1] %+v, [2] %+v; want [1] 16 high, [2] from 16, 14 high", a, b)
	}
	ta.press("-", "-")
	if a, b := ta.rectOf(t, ps[0]), ta.rectOf(t, ps[1]); a.H != 14 || b.Y != 14 || b.H != 16 {
		t.Errorf("- -: [1] %+v, [2] %+v", a, b)
	}
}

// TestScreen39SharedLine: every panel on the line moves with it; a line is
// as long as the panels along it touch: under a panel across the whole
// width the upright line stops (tdp screen.9, screen-39).
func TestScreen39SharedLine(t *testing.T) {
	ta, _, ps := resizeApp(t, 100, 31, grid4, "A", "B", "C", "D")
	ta.press(">")
	for i, want := range []Rect{{0, 0, 52, 15}, {52, 0, 48, 15}, {0, 15, 52, 15}, {52, 15, 48, 15}} {
		if got := ta.rectOf(t, ps[i]); got != want {
			t.Errorf("four, > on [1]: [%d] %+v, want %+v", i+1, got, want)
		}
	}
	ta.press("+")
	for i, want := range []Rect{{0, 0, 52, 16}, {52, 0, 48, 16}, {0, 16, 52, 14}, {52, 16, 48, 14}} {
		if got := ta.rectOf(t, ps[i]); got != want {
			t.Errorf("four, + on [1]: [%d] %+v, want %+v", i+1, got, want)
		}
	}

	ta, _, ps = resizeApp(t, 100, 31, twoOverOne, "A", "B", "C")
	ta.press(">", "+")
	for i, want := range []Rect{{0, 0, 52, 16}, {52, 0, 48, 16}, {0, 16, 100, 14}} {
		if got := ta.rectOf(t, ps[i]); got != want {
			t.Errorf("two over one, > + on [1]: [%d] %+v, want %+v", i+1, got, want)
		}
	}
}

// TestScreen40WhichLine: a panel with a line on both sides moves the right
// one for its width and the one under it for its height; one on the area's
// right or bottom edge moves the other, so > always widens it and + always
// makes it taller (tdp screen.9, screen-40).
func TestScreen40WhichLine(t *testing.T) {
	ta, _, ps := resizeApp(t, 99, 31, columns3, "A", "B", "C")
	ta.press("2", ">")
	if b, c := ta.rectOf(t, ps[1]), ta.rectOf(t, ps[2]); b != (Rect{33, 0, 35, 30}) || c != (Rect{68, 0, 31, 30}) {
		t.Errorf("> on the middle one: [2] %+v, [3] %+v; want its right line moved", b, c)
	}
	ta.press("3", ">")
	if b, c := ta.rectOf(t, ps[1]), ta.rectOf(t, ps[2]); b != (Rect{33, 0, 33, 30}) || c != (Rect{66, 0, 33, 30}) {
		t.Errorf("> on the right one: [2] %+v, [3] %+v; want its left line moved, it wider", b, c)
	}
	ta.press("<")
	if c := ta.rectOf(t, ps[2]); c != (Rect{68, 0, 31, 30}) {
		t.Errorf("< on the right one: [3] %+v, narrower", c)
	}

	ta, _, ps = resizeApp(t, 100, 31, stacked2, "Top", "Bottom")
	ta.press("2", "+")
	if a, b := ta.rectOf(t, ps[0]), ta.rectOf(t, ps[1]); a.H != 14 || b != (Rect{0, 14, 100, 16}) {
		t.Errorf("+ on the bottom one: [1] %+v, [2] %+v; want its top line moved, it taller", a, b)
	}
}

// TestScreen41Least: a line stops once a panel it moves is at its least
// size, the app's or the lib's 10 × 3; the last move goes as far as there is
// room, less than 2 columns (tdp screen.9, screen-41).
func TestScreen41Least(t *testing.T) {
	ta, _, ps := resizeApp(t, 100, 31, nil, "Left", "Right")
	ps[1].MinWidth = 45
	ta.press(">", ">")
	if r := ta.rectOf(t, ps[1]); r.W != 46 {
		t.Fatalf("[2] %d wide, want 46", r.W)
	}
	ta.press(">") // 1 column left before [2] is at its least
	if r := ta.rectOf(t, ps[1]); r.W != 45 || len(ta.showing()) != 0 {
		t.Errorf("the last column: [2] %d wide, toasts %v", r.W, ta.showing())
	}
	ta.press(">")
	if r := ta.rectOf(t, ps[1]); r.W != 45 || !slices.Equal(ta.showing(), []string{"INFO: [2] is at its minimum width"}) {
		t.Errorf("at its least: [2] %d wide, toasts %v", r.W, ta.showing())
	}
	for range 30 {
		ta.press("<")
	}
	if r := ta.rectOf(t, ps[0]); r.W != leastPanelW {
		t.Errorf("[1] %d wide after many <, want the lib's least %d", r.W, leastPanelW)
	}
	if got := ta.showing(); !strings.HasSuffix(got[len(got)-1], ": [1] is at its minimum width") {
		t.Errorf("[1] at its least: %v", got)
	}
}

// TestScreen42NoLine: with no line to move a key still answers, an Info
// toast naming the panel that stops the line, the focused one first; narrow
// or zoomed the one panel is full width and full height (tdp screen.9,
// screen-42).
func TestScreen42NoLine(t *testing.T) {
	ta, _, ps := resizeApp(t, 100, 31, stacked2, "Top", "Bottom")
	ta.press(">")
	ta.press("<")
	if got := ta.showing(); !slices.Equal(got, []string{"INFO ×2: [1] is already full width"}) {
		t.Errorf("> and < on a full-width panel: %v", got)
	}
	if !strings.Contains(strings.Join(ta.logs.lines(), "\n"), "[1] is already full width") {
		t.Error("the toast is not in the App Log")
	}

	ta, _, ps = resizeApp(t, 100, 31, grid4, "A", "B", "C", "D")
	ps[0].MinWidth, ps[2].MinWidth = 50, 50
	ta.press("3", "<")
	if _, v := ta.newestToast(t); strings.Join(v.text, "") != "[3] is at its minimum width" {
		t.Errorf("[1] and [3] at their least, < on [3]: %q, want the focused one", v.text)
	}
	ta.press("1", "<")
	if _, v := ta.newestToast(t); strings.Join(v.text, "") != "[1] is at its minimum width" {
		t.Errorf("< on [1]: %q", v.text)
	}

	for _, narrow := range []func(*testApp){
		func(ta *testApp) { ta.Screen().NarrowBelow = 60; ta.resize(50, 31) },
		func(ta *testApp) { ta.ForceNarrow = true },
	} {
		ta, _, _ = resizeApp(t, 100, 31, nil, "Left", "Right")
		narrow(ta)
		ta.press(">", "+")
		if got := ta.showing(); !slices.Equal(got, []string{"INFO: [1] is already full width", "INFO: [1] is already full height"}) {
			t.Errorf("narrow or zoomed: %v", got)
		}
	}
}

// TestScreen43ToastPlace: the toast is on that panel, on the edge that would
// move, inside its frame: the right one at the top right, the left one at
// the top left, the bottom one at the bottom in the middle, the top one at
// the top in the middle; full width is the right, full height the bottom;
// > and < on a panel put it in the same place; more stack into the panel;
// one wider than the panel keeps to its side, on the screen (tdp screen.9,
// toast.2, screen-43).
func TestScreen43ToastPlace(t *testing.T) {
	ta, _, ps := resizeApp(t, 100, 31, nil, "Left", "Right")
	ps[0].MinWidth, ps[1].MinWidth = 50, 50
	ta.press(">") // [2] stops [1]'s right line
	l, _ := ta.newestToast(t)
	r := ta.rectOf(t, ps[0])
	if l.x+l.w != r.X+r.W-1 || l.y != r.Y+1 {
		t.Errorf("the right edge: toast at %d,%d %d wide; want its right at %d, its top at %d", l.x, l.y, l.w, r.X+r.W-1, r.Y+1)
	}
	ta.press("<")
	if l2, _ := ta.newestToast(t); l2.x+l2.w != r.X+r.W-1 {
		t.Errorf("< on [1] puts it on the same edge: %d", l2.x+l2.w)
	}
	// The edge moves while the toast shows: the toast goes with it.
	ps[0].MinWidth = 10
	ta.press("<")
	ta.Frame()
	r = ta.rectOf(t, ps[0])
	if l2, _ := ta.newestToast(t); r.W != 48 || l2.x+l2.w != r.X+r.W-1 {
		t.Errorf("the edge moved to %d: the toast's right at %d", r.X+r.W-1, l2.x+l2.w)
	}
	ps[0].MinWidth = 50 // under it now: it cannot shrink
	ta.press("2", ">")  // [2] on the right edge: its left line, stopped by [1]
	l, _ = ta.newestToast(t)
	r = ta.rectOf(t, ps[1])
	if l.x != r.X+1 || l.y != r.Y+1 {
		t.Errorf("the left edge: toast at %d,%d; want %d,%d", l.x, l.y, r.X+1, r.Y+1)
	}
	ta.press("+") // full height: the bottom
	l, _ = ta.newestToast(t)
	if l.y+l.h != r.Y+r.H-1 || l.x != r.X+(r.W-l.w)/2 {
		t.Errorf("full height: toast at %d,%d %dx%d; want its bottom at %d, in the middle", l.x, l.y, l.w, l.h, r.Y+r.H-1)
	}
	// Drawn inside [2]'s frame: the frame's left column untouched.
	frame := ta.Frame()
	row := []rune(StripStyles(frame[l.y+1]))
	if row[r.X] != '┃' { // [2] has the focus
		t.Errorf("the toast covers [2]'s frame: %q", string(row))
	}

	ta, _, ps = resizeApp(t, 100, 31, stacked2, "Top", "Bottom")
	ps[0].MinHeight = 15
	ta.press("2", "+") // [2] on the bottom edge: its top line, stopped by [1]
	l, _ = ta.newestToast(t)
	r = ta.rectOf(t, ps[1])
	if l.y != r.Y+1 || l.x != r.X+(r.W-l.w)/2 {
		t.Errorf("the top edge: toast at %d,%d; want %d at the top, in the middle", l.x, l.y, r.Y+1)
	}

	// Two on one edge stack into the panel, the newest at the edge; = goes
	// at the bottom.
	ta, _, ps = resizeApp(t, 100, 31, grid4, "A", "B", "C", "D")
	ps[1].MinWidth = 50
	ta.press(">")
	ps[1].MinWidth, ps[3].MinWidth = 0, 50
	ta.press(">")
	ts := ta.showingToasts()
	if len(ts) != 2 || ts[0].y != ts[1].y+ts[1].h || ts[0].x+ts[0].w != ts[1].x+ts[1].w {
		t.Fatalf("two on [1]'s right edge: %v at %d and %d", ta.showing(), ts[0].y, ts[1].y)
	}
	ta.press("=")
	if l, v := ta.newestToast(t); v.at != nil || l.y+l.h != ta.h-2 {
		t.Errorf("= puts its toast where toasts go: %d", l.y+l.h)
	}

	// Wider than the panel: at its side, over the next one, on the screen.
	ta, _, ps = resizeApp(t, 100, 31, func(w, h int) []Rect {
		return []Rect{{0, 0, 12, h}, {12, 0, w - 12, h}}
	}, "Narrow", "Wide")
	ps[1].MinWidth = 88
	ta.press(">")
	l, _ = ta.newestToast(t)
	if l.w <= 10 || l.x != 0 {
		t.Errorf("a toast wider than [1]: %d wide at %d, want kept on the screen from 0", l.w, l.x)
	}
	ta.press("2", "<")
	l, _ = ta.newestToast(t)
	r = ta.rectOf(t, ps[1])
	if l.x != r.X+1 {
		t.Errorf("[2]'s left edge: %d, want %d", l.x, r.X+1)
	}
}

// TestScreen44Default: = puts every line of the screen back as Layout has
// it; with every line there already it says so, where toasts go (tdp
// screen.9, screen-44).
func TestScreen44Default(t *testing.T) {
	ta, _, ps := resizeApp(t, 100, 31, grid4, "A", "B", "C", "D")
	ta.press(">", "+", "4", ">")
	ta.press("=")
	for i, want := range grid4(100, 30) {
		if got := ta.rectOf(t, ps[i]); got != want {
			t.Errorf("after =: [%d] %+v, want %+v", i+1, got, want)
		}
	}
	if len(ta.showing()) != 0 || len(ta.Lines) != 0 {
		t.Errorf("= says nothing and keeps nothing: %v %v", ta.showing(), ta.Lines)
	}
	ta.press("=")
	if got := ta.showing(); !slices.Equal(got, []string{"INFO: The layout is already the default"}) {
		t.Errorf("= again: %v", got)
	}
}

// TestScreen45Kept: where a line is kept for the screen, by the terminal's
// width for an upright one and its height for a level one, and used again
// at that size; a size never set up has the default (tdp screen.9,
// screen-45).
func TestScreen45Kept(t *testing.T) {
	ta, _, ps := resizeApp(t, 100, 31, grid4, "A", "B", "C", "D")
	ta.press(">", "+")
	ta.resize(120, 31)
	if r := ta.rectOf(t, ps[0]); r != (Rect{0, 0, 60, 16}) {
		t.Errorf("120 × 31: [1] %+v; want the default width, the height kept for 31", r)
	}
	ta.resize(100, 41)
	if r := ta.rectOf(t, ps[0]); r != (Rect{0, 0, 52, 20}) {
		t.Errorf("100 × 41: [1] %+v; want the width kept for 100, the default height", r)
	}
	ta.resize(100, 31)
	if r := ta.rectOf(t, ps[0]); r != (Rect{0, 0, 52, 16}) {
		t.Errorf("back to 100 × 31: [1] %+v", r)
	}
	// Each screen its own.
	other := &Panel{Label: "Other", Body: &Pager{}}
	ta.AddScreen(&Screen{Name: "More", Key: "M", Panels: []*Panel{other, {Label: "Too", Body: &Pager{}}}})
	if _, ok := ta.Lines["More"]; ok {
		t.Error("another screen has the first one's lines")
	}
	// Read back from the app's state by another start of the app.
	ta2, _, ps2 := resizeApp(t, 100, 31, grid4, "A", "B", "C", "D")
	ta2.Lines = ta.Lines
	if r := ta2.rectOf(t, ps2[3]); r != (Rect{52, 16, 48, 14}) {
		t.Errorf("read back: [4] %+v", r)
	}
}

// TestScreen46NothingWritten: nothing goes to the state before the user
// moves a line; each move gives the app the lines to write, and = takes this
// screen's at this width and height away (tdp screen.9, screen-46).
func TestScreen46NothingWritten(t *testing.T) {
	ta, _, _ := resizeApp(t, 100, 31, grid4, "A", "B", "C", "D")
	var got []Lines
	ta.OnLines = func(l Lines) tea.Cmd { // what the app writes then
		c := Lines{}
		for k, v := range l {
			sl := ScreenLines{Widths: map[int]map[string]int{}, Heights: map[int]map[string]int{}}
			for size, at := range v.Widths {
				sl.Widths[size] = maps.Clone(at)
			}
			for size, at := range v.Heights {
				sl.Heights[size] = maps.Clone(at)
			}
			c[k] = sl
		}
		got = append(got, c)
		return nil
	}
	ta.press("2", "1", "Space", "Esc", "?", "Esc")
	ta.resize(90, 31)
	ta.resize(100, 31)
	if len(got) != 0 {
		t.Fatalf("written before a line moved: %v", got)
	}
	ta.press(">", "+")
	want := Lines{"Grid": {Widths: map[int]map[string]int{100: {"50": 52}}, Heights: map[int]map[string]int{31: {"15": 16}}}}
	if len(got) != 2 || !reflect.DeepEqual(got[1], want) {
		t.Errorf("after > +: %v, want %v", got, want)
	}
	ta.resize(120, 41)
	ta.press(">")
	ta.resize(100, 31)
	ta.press("=")
	want = Lines{"Grid": {Widths: map[int]map[string]int{120: {"60": 62}}, Heights: map[int]map[string]int{}}}
	if n := len(got); n != 4 || !reflect.DeepEqual(got[n-1]["Grid"].Widths, want["Grid"].Widths) || len(got[n-1]["Grid"].Heights) != 0 {
		t.Errorf("after = at 100 × 31: %v, want what is kept for 120 alone", got[len(got)-1])
	}
	ta.press("<") // moved, then back where Layout has it: not kept
	ta.press(">")
	if n := len(got); fmt.Sprint(got[n-2]["Grid"].Widths[100]) != "map[50:48]" || got[n-1]["Grid"].Widths[100] != nil {
		t.Errorf("a line moved and back: %v, then %v", got[n-2], got[n-1])
	}
}

// TestScreen47Menu: the Space menu of every panel has the resize keys among
// its panel operations, the global operation popup [=] Default layout;
// a row whose key would only say why not is disabled, [=] while every line
// is where Layout has it (tdp screen.9, screen-47).
func TestScreen47Menu(t *testing.T) {
	ta, _, _ := resizeApp(t, 100, 31, stacked2, "Top", "Bottom")
	rows := func() map[string]bool {
		out := map[string]bool{}
		for _, r := range ta.topView().(*menuView).rows {
			if r.kind == menuItem {
				out[MarkLabel(r.action.Key, r.action.Label)+"  "+r.action.Desc] = r.action.disabled()
			}
		}
		return out
	}
	ta.press("Space")
	got := rows()
	for row, off := range map[string]bool{
		"[>] Wider  2 columns, kept for this width":    true, // full width: > only says so
		"[<] Narrower  2 columns, kept for this width": true,
		"[+] Taller  1 row, kept for this height":      false,
		"[-] Shorter  1 row, kept for this height":     false,
	} {
		if d, ok := got[row]; !ok || d != off {
			t.Errorf("%q: there %v, disabled %v; want disabled %v", row, ok, d, off)
		}
	}
	ta.press("Esc")
	ta.openGlobal()
	ta.settle()
	if d, ok := rows()["[=] Default layout  this screen's lines back"]; !ok || !d {
		t.Errorf("[=] Default layout: there %v, disabled %v; want there, disabled", ok, d)
	}
	ta.press("Esc", "+")
	ta.openGlobal()
	ta.settle()
	if d := rows()["[=] Default layout  this screen's lines back"]; d {
		t.Error("[=] is disabled with a line moved")
	}
	ta.press("=") // run from the popup
	if r := ta.rectOf(t, ta.Screen().Panels[0]); r.H != 15 || ta.top() != nil {
		t.Errorf("[=] from the popup: [1] %+v, popup %v", r, ta.top())
	}
	// Run from the Space menu: the line moves and the menu closes.
	ta.press("Space", "+")
	if r := ta.rectOf(t, ta.Screen().Panels[0]); r.H != 16 || ta.top() != nil {
		t.Errorf("[+] from the menu: [1] %+v, popup %v", r, ta.top())
	}
}

// TestScreen48OnePanel: a screen with one panel has no resize rows in its
// Space menu, nor [=] in the global operation popup, but the keys still
// answer with a toast (tdp screen.9, screen-48).
func TestScreen48OnePanel(t *testing.T) {
	ta, _, _ := resizeApp(t, 100, 31, nil, "Only")
	ta.press("Space")
	for _, r := range ta.topView().(*menuView).rows {
		if r.kind == menuItem && isResizeKey(r.action.Key) && !r.action.hidden() {
			t.Errorf("%q in a one-panel screen's menu", r.action.Label)
		}
	}
	ta.press("Esc")
	ta.openGlobal()
	ta.settle()
	for _, r := range ta.topView().(*menuView).rows {
		if r.kind == menuItem && r.action.Key == "=" && !r.action.hidden() {
			t.Error("[=] Default layout in the global operation popup of a one-panel screen")
		}
	}
	ta.press("Esc", ">", "-", "=")
	want := []string{"INFO: [1] is already full width", "INFO: [1] is already full height", "INFO: The layout is already the default"}
	if got := ta.showing(); !slices.Equal(got, want) {
		t.Errorf("the keys on one panel: %v, want %v", got, want)
	}
}

// TestScreen49KeysKept: on a panel > < + - = only resize: Run refuses them
// as a panel's, a screen's or a global hotkey; a mode may have them; the
// panel's key reference lists the four with its panel operations and = with
// the app-wide ones (tdp screen.9, screen-49).
func TestScreen49KeysKept(t *testing.T) {
	for _, c := range []struct {
		name string
		set  func(a *App, p *Panel, s *Screen)
	}{
		{"a panel's", func(_ *App, p *Panel, _ *Screen) {
			p.Actions = []Action{{Key: ">", Label: "Next", Scope: PanelScope}}
		}},
		{"a screen's", func(_ *App, _ *Panel, s *Screen) { s.Key = "=" }},
		{"a global one", func(a *App, _ *Panel, _ *Screen) {
			a.Global = []Action{{Key: "+", Label: "Add", Scope: GlobalScope}}
		}},
	} {
		ta, s, ps := resizeApp(t, 100, 31, nil, "Left", "Right")
		c.set(ta.App, ps[0], s)
		if err := ta.checkKeys(); err == nil || !strings.Contains(err.Error(), "resizes the panels") {
			t.Errorf("%s hotkey on a resize key: %v", c.name, err)
		}
	}
	ta, _, ps := resizeApp(t, 100, 31, nil, "Left", "Right")
	ps[0].EnterMode(&Mode{Name: "Pick", Keys: []ModeKey{{Key: ">", Desc: "next", Do: func(string) tea.Cmd { return nil }}}})
	if err := ta.checkKeys(); err != nil {
		t.Errorf("a mode's key: %v", err)
	}
	ps[0].LeaveMode()
	ta.press("?")
	ref := ta.screenText()
	for _, want := range []string{"Wider — 2 columns, kept for this width", "Shorter — 1 row, kept for this height",
		"Default layout — this screen's lines back"} {
		if !strings.Contains(ref, want) {
			t.Errorf("the key reference has no %q:\n%s", want, ref)
		}
	}
}

// TestScreen9Typing: the keys go where the keys go: typing in a filter, a
// mode, a popup; there they are not resize keys (tdp screen.9, K8, K11).
func TestScreen9Typing(t *testing.T) {
	ta, _, ps := resizeApp(t, 100, 31, nil, "Left", "Right")
	var got []string
	ps[0].EnterMode(&Mode{Name: "Pick", Keys: []ModeKey{{Key: ">", Desc: "next", Do: func(k string) tea.Cmd { got = append(got, k); return nil }}}})
	ta.press(">")
	if r := ta.rectOf(t, ps[0]); r.W != 50 || !slices.Equal(got, []string{">"}) {
		t.Errorf("in a mode: [1] %d wide, the mode got %v", r.W, got)
	}
	ps[0].LeaveMode()
	ta.press("Space", ">") // the menu's row runs: the line moves
	if r := ta.rectOf(t, ps[0]); r.W != 52 {
		t.Errorf("[>] in the menu: [1] %d wide", r.W)
	}
}

// TestScreen9Hidden: a place hidden while its leading panel shows nothing
// there takes its line with it; the lines it does not touch stay where the
// user left them; shown again, its line is back where the user left it; a
// line the Layout moves when it hides one is another line, with its own
// default (tdp screen.9, L5).
func TestScreen9Hidden(t *testing.T) {
	ta := &testApp{App: anySize(New("test"))}
	mk := func(n string) *Panel { pg := &Pager{}; pg.SetLines([]string{n}); return &Panel{Label: n, Body: pg} }
	changes, branches, diff, history, commits, files := mk("Changes"), mk("Branches"), mk("Diff"), mk("History"), mk("Commits"), mk("Files")
	var s *Screen
	s = &Screen{Name: "Git", Key: "G", Panels: []*Panel{changes, branches, commits, files}, Following: []int{2, 3},
		Leads: []Lead{{Panel: changes, Shows: []*Panel{diff, nil}}, {Panel: branches, Shows: []*Panel{commits, files}}},
		// The leading panels on the left; on the right the two places, the
		// first taking it all when the second is hidden.
		Layout: func(w, h int) []Rect {
			c, r := w/3, h/2
			out := []Rect{{0, 0, c, r}, {0, r, c, h - r}, {c, 0, w - c, r}, {c, r, w - c, h - r}}
			if s.Hidden(3) {
				out[2], out[3] = Rect{c, 0, w - c, h}, Rect{}
			}
			return out
		}}
	ta.AddScreen(s)
	ta.resize(90, 31)
	_ = history
	ta.press("2")      // BRANCHES leads: both places shown
	ta.press("3", "+") // the line under COMMITS
	ta.press(">")      // ... and to the right, the line between left and right
	if r := ta.rectOf(t, commits); r != (Rect{28, 0, 62, 16}) {
		t.Fatalf("COMMITS %+v", r)
	}
	ta.press("1") // CHANGES leads: the fourth place hidden
	if r := ta.rectOf(t, diff); r != (Rect{28, 0, 62, 30}) {
		t.Errorf("CHANGES leading: DIFF %+v; want the line on the left kept, the level line gone with the place", r)
	}
	ta.press("2")
	if r, f := ta.rectOf(t, commits), ta.rectOf(t, files); r != (Rect{28, 0, 62, 16}) || f.Y != 16 {
		t.Errorf("BRANCHES again: COMMITS %+v, FILES %+v; want the level line back where it was", r, f)
	}
}

// TestScreen9Sign: a sign moves with the line, as it resizes with the
// panels; at its least the toast names it by its label (tdp screen.9,
// panel.17).
func TestScreen9Sign(t *testing.T) {
	ta, _, ps := resizeApp(t, 100, 31, stacked2, "Top", "Info")
	ps[1].Sign, ps[1].MinHeight = SignDim, 14
	ta.press("+")
	if r := ta.rectOf(t, ps[1]); r.H != 14 {
		t.Errorf("the sign %d high, want 14", r.H)
	}
	ta.press("+")
	if got := ta.showing(); !slices.Equal(got, []string{"INFO: INFO is at its minimum height"}) {
		t.Errorf("the sign at its least: %v", got)
	}
}

// TestScreen9Stale: lines kept for a size that no longer fit (the app's
// Layout or least sizes changed) are not used there: the default ones are,
// and what is kept for that width or height is taken away and written, with
// no toast and no WARN, E3's one exception; a size they fit keeps its own
// (tdp screen.9).
func TestScreen9Stale(t *testing.T) {
	ta, _, ps := resizeApp(t, 100, 31, nil, "Left", "Right")
	var written []Lines
	ta.OnLines = func(l Lines) tea.Cmd { written = append(written, l); return nil }
	ta.Lines = Lines{"Grid": {Widths: map[int]map[string]int{100: {"50": 95, "70": 80}, 120: {"60": 70}}}}
	if r := ta.rectOf(t, ps[1]); r.X != 50 {
		t.Errorf("a kept line that leaves [2] 5 wide: [2] %+v, want the default", r)
	}
	ta.resize(100, 31)
	if got := ta.Lines["Grid"].Widths; !reflect.DeepEqual(got, map[int]map[string]int{120: {"60": 70}}) || len(written) != 1 {
		t.Errorf("after the next message: %v kept, written %d times; want 100's taken away, 120's kept", got, len(written))
	}
	if len(ta.showing()) != 0 || strings.Contains(strings.Join(ta.logs.lines(), "\n"), "WARN") {
		t.Errorf("something said: %v %v", ta.showing(), ta.logs.lines())
	}
	ta.resize(120, 31)
	if r := ta.rectOf(t, ps[1]); r.X != 70 || len(written) != 1 {
		t.Errorf("at 120: [2] %+v, written %d times; want the line kept for 120", r, len(written))
	}
	// A line moved in the same message as they are found starts afresh.
	ta.Lines = Lines{"Grid": {Widths: map[int]map[string]int{120: {"60": 115}}}}
	ta.press(">")
	if got := ta.Lines["Grid"].Widths[120]; !reflect.DeepEqual(got, map[string]int{"60": 62}) {
		t.Errorf("after a move: %v", got)
	}
}
