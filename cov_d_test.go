package terminu

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Tests for docs/coverage.md's layout/popup, dialog/menu and dialog/confirm.
// The default test app is 120 × 40; its Space menu on [1] PODS is, row by row:
// top, blank, ITEM OPERATION, [Enter] Logs, [m]ark, [x] Delete, [D]rag,
// separator, PANEL OPERATION, Refresh, [/] Filter, separator, Global
// operation, blank, bottom: eleven rows, seven of them actions.

// covDCell is how one cell of a drawn line looks.
type covDCell struct {
	fg, bg RGB
	bold   bool
}

// covDStyleAt is the look of cell col of a drawn line, read from its SGR
// sequences.
func covDStyleAt(line string, col int) covDCell {
	var c, at covDCell
	cell, found := 0, false
	walk(line, func(seq string, cells int) {
		if found {
			return
		}
		if cells == 0 {
			if strings.HasPrefix(seq, "\x1b[") && strings.HasSuffix(seq, "m") {
				c = covDApply(c, seq[2:len(seq)-1])
			}
			return
		}
		if col >= cell && col < cell+cells {
			at, found = c, true
		}
		cell += cells
	})
	return at
}

func covDApply(c covDCell, params string) covDCell {
	ps := strings.Split(params, ";")
	num := func(i int) uint8 { n, _ := strconv.Atoi(ps[i]); return uint8(n) }
	for i := 0; i < len(ps); i++ {
		switch ps[i] {
		case "", "0":
			c = covDCell{}
		case "1":
			c.bold = true
		case "22":
			c.bold = false
		case "39":
			c.fg = RGB{}
		case "49":
			c.bg = RGB{}
		case "38", "48":
			if i+4 < len(ps) && ps[i+1] == "2" {
				rgb := RGB{num(i + 2), num(i + 3), num(i + 4)}
				if ps[i] == "38" {
					c.fg = rgb
				} else {
					c.bg = rgb
				}
				i += 4
			}
		}
	}
	return c
}

// covDRows is a popup's rows as they are on the screen, cut to its width.
func covDRows(ta *testApp, l *layer) []string {
	frame := ta.Frame()
	rows := make([]string, l.h)
	for i := range rows {
		rows[i] = Cut(frame[l.y+i], l.x, l.x+l.w)
	}
	return rows
}

func covDPlain(rows []string) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = StripStyles(r)
	}
	return out
}

// covDTop is the top popup's rows, without styles.
func covDTop(ta *testApp) []string { return covDPlain(covDRows(ta, ta.top())) }

// covDBlank is an empty row inside a w-wide frame.
func covDBlank(w int) string { return "┃" + Repeat(" ", w-2) + "┃" }

// covDCell1 is the text of cell col of a drawn line.
func covDCell1(line string, col int) string { return StripStyles(Cut(line, col, col+1)) }

// covDGaps are the blank cells left and right of a row's text, inside the
// frame.
func covDGaps(row string) (left, right int) {
	inner := strings.TrimSuffix(strings.TrimPrefix(row, "┃"), "┃")
	return Width(inner) - Width(strings.TrimLeft(inner, " ")), Width(inner) - Width(strings.TrimRight(inner, " "))
}

// covDCursor is the top popup's row under the cursor, a block of the top
// layer's colour, without styles.
func covDCursor(ta *testApp) string {
	l := ta.top()
	rows := covDRows(ta, l)
	for _, r := range rows[1 : l.h-1] {
		if covDStyleAt(r, 1).bg == LayerColor(len(ta.stack)).RGB {
			return StripStyles(r)
		}
	}
	return ""
}

// covDHintKeys are the keys of the hint on a popup's bottom border, in order.
func covDHintKeys(ta *testApp, l *layer) []string {
	p := covDPlain(covDRows(ta, l))
	b := strings.TrimPrefix(p[l.h-1], "┗━ ")
	if i := strings.Index(b, " ━"); i >= 0 {
		b = b[:i]
	}
	var keys []string
	for _, pair := range strings.Fields(b) {
		k, _, _ := strings.Cut(pair, ":")
		keys = append(keys, k)
	}
	return keys
}

// covDTerminal opens a terminal popup with text already on its screen.
func covDTerminal(ta *testApp, text string, keys ...TerminalKey) *Terminal {
	term := &Terminal{Title: "Alterm", Subject: "myHost ~/Work", in: &strings.Builder{}, started: true, Keys: keys}
	term.mu.Lock()
	term.vt().write([]byte(text))
	term.mu.Unlock()
	ta.open(&termView{t: term, a: ta.App})
	ta.settle()
	return term
}

func TestPopupFrame(t *testing.T) {
	ta := newTestApp(t)
	ta.press("Space")
	l := ta.top()
	rows := covDRows(ta, l)
	p := covDPlain(rows)
	last := l.h - 1
	if !strings.HasPrefix(p[0], "┏━") || !strings.HasSuffix(p[0], "━┓") ||
		!strings.HasPrefix(p[last], "┗━") || !strings.HasSuffix(p[last], "━┛") {
		t.Errorf("a popup's frame is the heavy line:\n%s", strings.Join(p, "\n"))
	}
	for i := 1; i < last; i++ {
		if !strings.HasPrefix(p[i], "┃") || !strings.HasSuffix(p[i], "┃") {
			t.Errorf("row %d has no heavy sides: %q", i, p[i])
		}
	}
	for _, at := range [][2]int{{0, 0}, {0, l.w - 1}, {last, 0}, {last, l.w - 1}, {1, 0}, {1, l.w - 1}} {
		if c := covDStyleAt(rows[at[0]], at[1]); c.fg != LayerColor(1).RGB {
			t.Errorf("the frame at %v is %v, not the layer colour", at, c.fg)
		}
	}
	// Covered by another popup and dimmed, it is still the heavy line.
	ta.press("G", "Enter")
	if len(ta.stack) != 2 {
		t.Fatalf("the global operation popup did not open: %d layers", len(ta.stack))
	}
	top := StripStyles(Cut(ta.Frame()[l.y], l.x, l.x+l.w))
	if !strings.HasPrefix(top, "┏━") || !strings.HasSuffix(top, "━┓") {
		t.Errorf("the popup underneath lost its heavy line: %q", top)
	}
}

func TestPopupTitleCapsule(t *testing.T) {
	ta := newTestApp(t)
	ta.press("Space")
	row := covDRows(ta, ta.top())[0]
	if want := "┏━" + glyphRoundLeft + glyphMenu + " [1] PODS" + glyphRoundRight + "━"; !strings.HasPrefix(StripStyles(row), want) {
		t.Errorf("the title is a capsule one segment after the corner, upper case: %q", StripStyles(row))
	}
	layer := LayerColor(1)
	if !strings.Contains(row, On(layer).Render(glyphMenu+" [1] PODS")) {
		t.Error("the capsule is filled with the layer colour, its text Base bold")
	}
	if !strings.Contains(row, Fg(layer).Render(glyphRoundLeft)) || !strings.Contains(row, Fg(layer).Render(glyphRoundRight)) {
		t.Error("the rounded ends are the layer colour")
	}
	// Names and paths in a title keep their case.
	ta.press("Esc")
	covDTerminal(ta, "")
	if top := covDTop(ta)[0]; !strings.Contains(top, glyphTerminal+" ALTERM: myHost ~/Work"+glyphRoundRight) {
		t.Errorf("a name in the title is not kept as it is: %q", top)
	}
}

func TestPopupTitleTruncate(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(40, 20)
	ta.Confirm(Ask{Title: "Delete every pod in the namespace kube-system", Question: "Sure?", Verb: "delete"}, nil)
	ta.settle()
	l := ta.top()
	top := covDTop(ta)[0]
	if !strings.HasPrefix(top, "┏━"+glyphRoundLeft+glyphConfirm+" DELETE EVERY") ||
		!strings.Contains(top, "…"+glyphRoundRight+"━") || !strings.HasSuffix(top, "━┓") || Width(top) != l.w {
		t.Errorf("a long title is cut from its end with …, the frame whole: %q", top)
	}
}

func TestPopupTitleGlyph(t *testing.T) {
	const pencil = "\U000F03EB" // the app's choice for an input, a form, a note
	cases := []struct {
		name  string
		open  func(ta *testApp)
		glyph string
	}{
		{"menu", func(ta *testApp) { ta.press("Space") }, "\U0000F0C9"},
		{"key reference", func(ta *testApp) { ta.press("?") }, "\U0000F059"},
		{"confirm", func(ta *testApp) { ta.press("x") }, "\U000F0ECC"},
		{"terminal", func(ta *testApp) { covDTerminal(ta, "") }, "\U000F018D"},
		{"input", func(ta *testApp) { ta.Text(TextInput{Title: "Rename", Glyph: pencil}) }, pencil},
		{"form", func(ta *testApp) {
			ta.Form(&Form{Title: "New host", Glyph: pencil, Fields: []*FormField{{Label: "Name"}}})
		}, pencil},
		{"note", func(ta *testApp) { ta.Viewer(&Viewer{Title: "YAML", Glyph: pencil, Lines: []string{"a"}}) }, pencil},
	}
	for _, c := range cases {
		ta := newTestApp(t)
		c.open(ta)
		ta.settle()
		if top := covDTop(ta)[0]; !strings.HasPrefix(top, "┏━"+glyphRoundLeft+c.glyph+" ") {
			t.Errorf("%s: the title does not start with its glyph: %q", c.name, top)
		}
	}
	for kind, glyph := range map[ToastKind]string{Info: "\U0000F024", Warning: "\U000F078F", Error: "\U000F0238"} {
		ta := newTestApp(t)
		ta.Toast(kind, "done")
		ta.settle()
		if top := covDPlain(covDRows(ta, ta.toast))[0]; !strings.HasPrefix(top, "┏━"+glyphRoundLeft+glyph+" ") {
			t.Errorf("toast %d: the title's glyph is the kind's: %q", kind, top)
		}
	}
}

func TestPopupHintOrder(t *testing.T) {
	// Enter first, the popup's own keys, a Tab that switches blocks, Esc last;
	// a terminal's exit key first. No key that moves between items.
	cases := []struct {
		name string
		open func(ta *testApp)
		want string
	}{
		{"menu", func(ta *testApp) { ta.press("Space") }, "Enter Esc"},
		{"confirm", func(ta *testApp) { ta.press("x") }, "Enter Esc"},
		{"viewer", func(ta *testApp) {
			v := testViewer()
			v.Keys = []ViewerKey{{Key: "y", Desc: "copy"}}
			v.Select = &Selection{Key: "v", Name: "Visual"}
			ta.Viewer(v)
		}, "y v Esc"},
		{"checkbox list", func(ta *testApp) { ta.Checks(Checks{Title: "Namespaces", Options: []string{"a", "b"}}) }, "Enter Space Tab|/ Esc"},
		{"finder typing", func(ta *testApp) { ta.Find(testFinder()) }, "Enter|Tab Esc"},
		{"text", func(ta *testApp) { ta.Text(TextInput{Title: "Rename", Old: "assets"}) }, "Enter Tab Backspace Esc"},
		{"form", func(ta *testApp) { ta.Form(testForm(nil)) }, "Enter Tab Esc"},
		{"terminal", func(ta *testApp) { covDTerminal(ta, "", TerminalKey{Key: "Alt-t", Desc: "hide"}) }, "Alt-Esc Alt-t"},
	}
	for _, c := range cases {
		ta := newTestApp(t)
		c.open(ta)
		ta.settle()
		if got := strings.Join(covDHintKeys(ta, ta.top()), " "); got != c.want {
			t.Errorf("%s: hint keys %q, want %q", c.name, got, c.want)
		}
	}
}

func TestPopupHintDrop(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(26, 12) // the Space menu is 24 wide and scrolls: 1 of 7
	ta.press("Space")
	l := ta.top()
	bottom := covDTop(ta)[l.h-1]
	if keys := covDHintKeys(ta, l); fmt.Sprint(keys) != "[Enter]" || !strings.Contains(bottom, "Enter:run ") {
		t.Errorf("Esc:close does not fit: the whole pair goes, Enter:run stays whole: %q", bottom)
	}
	if !strings.HasSuffix(bottom, " 1 of 7 ━┛") {
		t.Errorf("the scroll position stays: %q", bottom)
	}
}

// TestPopupWidthCountsScroll: a popup that scrolls is wide enough for its
// whole hint and its scroll position at the widest (components/layout/popup):
// Esc stays in sight, and the frame keeps its width while it scrolls.
func TestPopupWidthCountsScroll(t *testing.T) {
	check := func(name string, open func(ta *testApp), hint string, widest string) {
		t.Helper()
		ta := newTestApp(t)
		ta.resize(120, 12)
		open(ta)
		ta.settle()
		l := ta.top()
		bottom := covDTop(ta)[l.h-1]
		if !strings.Contains(bottom, " "+hint+" ") {
			t.Errorf("%s: the hint is not whole: %q", name, bottom)
		}
		if want := Width(hint) + Width(widest) + 9; l.w < want {
			t.Errorf("%s: %d wide, the hint and %q need %d", name, l.w, widest, want)
		}
		w := l.w
		for range 30 {
			ta.press("j")
		}
		if ta.top().w != w {
			t.Errorf("%s: the width changed while it scrolled: %d, then %d", name, w, ta.top().w)
		}
	}
	var rows []Action
	for i := range 20 {
		rows = append(rows, Action{Label: string(rune('a' + i))})
	}
	check("menu", func(ta *testApp) { ta.Menu("Pick", MenuSection{Actions: rows}) }, "Enter:run Esc:close", "20 of 20")
	var details []string
	for i := range 30 {
		details = append(details, string(rune('a'+i%26)))
	}
	check("confirm", func(ta *testApp) {
		ta.Confirm(Ask{Title: "Wipe", Question: "Wipe?", Verb: "wipe", Details: details}, nil)
	}, "Enter:wipe Esc:cancel", "30-30 of 30")
	// The finders too (demou's DM3: 18 options left Esc:cancel out).
	options := make([]string, 18)
	for i := range options {
		options[i] = string(rune('a' + i))
	}
	check("select", func(ta *testApp) { ta.Select(Select{Title: "Pick", Options: options, Selected: -1}) },
		"Enter:choose Tab|/:filter Esc:cancel", "18 of 18")
	check("checks", func(ta *testApp) { ta.Checks(Checks{Title: "Pick", Options: options}) },
		"Enter:apply Space:toggle Tab|/:filter Esc:cancel", "18 of 18")
}

func TestPopupFrameTextSpacing(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(120, 12)
	ta.press("Space")
	l := ta.top()
	p := covDTop(ta)
	if !strings.HasPrefix(p[0], "┏━"+glyphRoundLeft) || !strings.HasSuffix(p[0], "━┓") {
		t.Errorf("one segment between a corner and the title: %q", p[0])
	}
	if !strings.HasPrefix(p[l.h-1], "┗━ Enter:run") || !strings.HasSuffix(p[l.h-1], " 1 of 7 ━┛") {
		t.Errorf("one segment between a corner and the hint, the scroll position: %q", p[l.h-1])
	}
}

func TestPopupMargins(t *testing.T) {
	ta := newTestApp(t)
	ta.press("Space")
	l := ta.top()
	p := covDTop(ta)
	if p[1] != covDBlank(l.w) || p[l.h-2] != covDBlank(l.w) {
		t.Errorf("a blank row above and below the content:\n%s", strings.Join(p, "\n"))
	}
	if !strings.HasPrefix(p[4], "┃ [m]ark") || !strings.HasSuffix(p[4], "this pod ┃") {
		t.Errorf("the content is one cell from the frame: %q", p[4])
	}
	ta.press("Esc")
	ta.Viewer(&Viewer{Title: "YAML", Lines: []string{"kind: Pod", "metadata:"}})
	ta.settle()
	l = ta.top()
	if p = covDTop(ta); p[1] != covDBlank(l.w) || p[l.h-2] != covDBlank(l.w) || !strings.HasPrefix(p[2], "┃   1 kind: Pod") {
		t.Errorf("a viewer has the same margins:\n%s", strings.Join(p, "\n"))
	}
	// A terminal has none: its screen fills the frame.
	ta.press("Esc")
	term := covDTerminal(ta, "hello")
	l = ta.top()
	if p = covDTop(ta); !strings.HasPrefix(p[1], "┃hello") {
		t.Errorf("a terminal has no margin above or on the left: %q", p[1])
	}
	if term.w != l.w-2 || term.h != l.h-2 {
		t.Errorf("a terminal's screen is %dx%d inside a %dx%d frame", term.w, term.h, l.w, l.h)
	}
}

func TestPopupCursorCoversPadding(t *testing.T) {
	for _, pl := range []int{1, 2} {
		withWidths(t, 1, pl)
		ta := newTestApp(t)
		ta.press("Space")
		l := ta.top()
		rows := covDRows(ta, l)
		cur, other := rows[3], rows[4] // [Enter] Logs under the cursor, [m]ark not
		layer := LayerColor(1).RGB
		if c := covDStyleAt(cur, 1); c.bg != layer || covDCell1(cur, 1) != " " {
			t.Errorf("powerline %d: the cursor block covers the left margin", pl)
		}
		if !strings.HasPrefix(StripStyles(cur), "┃ [Enter] Logs") {
			t.Errorf("powerline %d: the text on the block stays one cell in: %q", pl, StripStyles(cur))
		}
		end := l.w - 1 - pl // the right margin: as wide as the arrow
		if got := StripStyles(Cut(cur, end, l.w-1)); got != glyphHard {
			t.Errorf("powerline %d: the right margin under the cursor is the arrow: %q", pl, got)
		}
		if c := covDStyleAt(cur, end); c.fg != layer || c.bg == layer {
			t.Errorf("powerline %d: the arrow is the block's colour on no ground: %+v", pl, c)
		}
		if got := StripStyles(Cut(other, end, l.w-1)); got != Repeat(" ", pl) {
			t.Errorf("powerline %d: every other row keeps a margin as wide as the arrow: %q", pl, got)
		}
		if c := covDStyleAt(cur, end-1); c.bg != layer {
			t.Error("the block goes on up to the arrow")
		}
	}
}

func TestPopupSeparator(t *testing.T) {
	ta := newTestApp(t)
	ta.press("Space")
	l := ta.top()
	rows := covDRows(ta, l)
	if got, want := StripStyles(rows[7]), "┃ "+Repeat("╌", l.w-4)+" ┃"; got != want {
		t.Errorf("a separator keeps one cell from either side of the frame: %q", got)
	}
	if c := covDStyleAt(rows[7], 2); c.fg != Overlay0.RGB {
		t.Error("a separator is Overlay0")
	}
	ta.press("Esc")
	ta.Form(testForm(nil))
	ta.settle()
	l = ta.top()
	if got, want := covDTop(ta)[l.h-4], "┃ "+Repeat("─", l.w-4)+" ┃"; got != want {
		t.Errorf("the form's separator does not touch the frame: %q", got)
	}
	// A vertical one stays out of the blank rows above and below.
	ta.press("Esc")
	ta.PickDate(DatePicker{Title: "Meeting", Time: true, Value: time.Date(2026, 10, 13, 9, 30, 0, 0, time.Local)})
	ta.settle()
	l = ta.top()
	rows = covDRows(ta, l)
	p := covDPlain(rows)
	if strings.Contains(p[1], "│") || strings.Contains(p[l.h-2], "│") {
		t.Errorf("the vertical separator runs into the margins:\n%s", strings.Join(p, "\n"))
	}
	if i := strings.Index(p[2], "│"); i < 0 {
		t.Error("the date and the time have no separator")
	} else if c := covDStyleAt(rows[2], Width(p[2][:i])); c.fg != Overlay0.RGB {
		t.Error("the vertical separator is Overlay0")
	}
}

func TestPopupNarrowContentAlign(t *testing.T) {
	ta := newTestApp(t)
	ta.Confirm(Ask{Title: "Delete", Question: "Sure?", Verb: "delete"}, nil)
	ta.settle()
	l := ta.top()
	if p := covDTop(ta); l.w <= Width("Sure?")+4 || !strings.HasPrefix(p[2], "┃ Sure? ") {
		t.Errorf("narrow content is on the left, one cell from the frame: %q", p[2])
	}
	ta.press("Esc")
	ta.Toast(Info, "Saved")
	ta.settle()
	row := covDPlain(covDRows(ta, ta.toast))[2]
	if left, right := covDGaps(row); left <= 1 || left-right > 1 || right-left > 1 {
		t.Errorf("a toast's text is centred: %q", row)
	}
	ta.press("Esc")
	ta.Password(PasswordInput{Title: "PIN", Prompt: "Up to 12 digits", Unlock: true, MaxLen: 12})
	ta.settle()
	ta.press("1", "2", "3")
	row = covDTop(ta)[4]
	// The cursor after the dots counts, a space and a block, blank without
	// styles (components/input/password).
	if left, right := covDGaps(row); !strings.Contains(row, "●") || left <= 1 || left-(right-2) > 1 || (right-2)-left > 1 {
		t.Errorf("the unlock mask is centred: %q", row)
	}
}

func TestF2DimFromFirstFrame(t *testing.T) {
	ta := newTestApp(t)
	footer := func() covDCell { f := ta.Frame(); return covDStyleAt(f[len(f)-1], 0) }
	if footer().fg != Blue.RGB {
		t.Fatal("the footer's first key is Blue")
	}
	ta.Update(keyMsg("Space"))
	if l := ta.top(); l == nil || l.frame != 1 {
		t.Fatal("the Space menu is not on its first frame")
	}
	if got := footer().fg; got != DimRGB(Blue.RGB) {
		t.Errorf("the screen is not dimmed from the opening's first frame: %v", got)
	}
	ta.settle()
	ta.Update(keyMsg("Esc"))
	if !ta.stack[0].closing {
		t.Fatal("Esc did not start the closing")
	}
	if got := footer().fg; got != Blue.RGB {
		t.Errorf("the screen is not bright again as soon as the closing starts: %v", got)
	}
	// Two layers: the one underneath is bright again at once.
	ta.settle()
	ta.press("Space", "G", "Enter")
	under := ta.stack[0]
	ta.Update(keyMsg("Esc"))
	if c := covDStyleAt(covDRows(ta, under)[0], 0); c.fg != LayerColor(1).RGB {
		t.Errorf("the Space menu is not bright as the popup over it starts closing: %v", c.fg)
	}
}

func TestLoadingIconFrames(t *testing.T) {
	for i, f := range loadingFrames {
		if r := []rune(f); len(r) != 1 || r[0] != rune(0xF0A9E+i) {
			t.Errorf("frame %d is %U, want U+%X", i, r, 0xF0A9E+i)
		}
		if Width(f) != 1 {
			t.Errorf("frame %d is %d cells, not one", i, Width(f))
		}
	}
	// The clock decides the frame: frames[(now / 90ms) % 8].
	before := time.Now().UnixMilli() / 90
	got := loadingFrame()
	after := time.Now().UnixMilli() / 90
	if got != loadingFrames[before%8] && got != loadingFrames[after%8] {
		t.Errorf("loadingFrame is %U, not the clock's frame", []rune(got))
	}
	// On a popup: after the whole tab chain, a space either side, the layer
	// colour in bold.
	ta := newTestApp(t)
	ta.Viewer(&Viewer{Title: "DevTools", Tabs: []string{"Console", "Network"}, Lines: []string{"a"}, Loading: true})
	ta.settle()
	row := covDRows(ta, ta.top())[0]
	found := false
	for _, f := range loadingFrames {
		if strings.Contains(StripStyles(row), "NETWORK"+glyphHard+" "+f+" ━") &&
			strings.Contains(row, Fg(LayerColor(1)).Bolded().Render(f)) {
			found = true
		}
	}
	if !found {
		t.Errorf("no loading icon after the tabs, in the layer colour bold: %q", StripStyles(row))
	}
}

func TestLoadingTickOnlyWhenLoading(t *testing.T) {
	ta := newTestApp(t)
	if cmd := ta.Update2(spinMsg{}); cmd != nil {
		t.Error("a tick is scheduled with nothing loading")
	}
	v := &Viewer{Title: "Log", Loading: true}
	ta.Viewer(v)
	ta.settle()
	cmd := ta.Update2(spinMsg{})
	if cmd == nil {
		t.Fatal("no next tick while a popup is loading")
	}
	start := time.Now()
	if _, ok := cmd().(spinMsg); !ok {
		t.Error("the tick does not come back as a spin")
	}
	if d := time.Since(start); d < 80*time.Millisecond || d > time.Second {
		t.Errorf("a frame lasts %v, not 90ms", d)
	}
	v.Loading = false
	if cmd := ta.Update2(spinMsg{}); cmd != nil {
		t.Error("the tick goes on after loading ended")
	}
}

func TestPopupScroll(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(120, 12) // six of the menu's eleven rows show
	ta.press("Space")
	l := ta.top()
	first := covDTop(ta)
	ta.press("j", "j", "j", "j", "j", "j") // down to Global operation, the last row
	p := covDTop(ta)
	if p[0] != first[0] || p[1] != covDBlank(l.w) || p[l.h-2] != covDBlank(l.w) || !strings.HasPrefix(p[l.h-1], "┗━ Enter:run") {
		t.Errorf("the frame and the margins do not scroll:\n%s", strings.Join(p, "\n"))
	}
	if !strings.Contains(p[l.h-3], "Global operation") || !strings.Contains(p[3], "PANEL OPERATION") {
		t.Errorf("scrolling the least puts the cursor on the last row, not the middle:\n%s", strings.Join(p, "\n"))
	}
	ta.press("k") // [/] Filter: still in sight, nothing scrolls
	if q := covDTop(ta); q[2] != p[2] || q[3] != p[3] || !strings.Contains(covDCursor(ta), "[/] Filter") {
		t.Errorf("the content scrolled while the cursor was in sight:\n%s", strings.Join(q, "\n"))
	}
}

func TestPopupScrollPosition(t *testing.T) {
	ta := newTestApp(t)
	ta.press("Space")
	if bottom := covDTop(ta)[ta.top().h-1]; strings.Contains(bottom, " of ") {
		t.Errorf("a position is written though everything fits: %q", bottom)
	}
	ta = newTestApp(t)
	ta.resize(120, 12)
	ta.press("Space")
	l := ta.top()
	bottom := covDRows(ta, l)[l.h-1]
	if !strings.HasSuffix(StripStyles(bottom), " 1 of 7 ━┛") || !strings.Contains(bottom, Fg(LayerColor(1)).Render(" 1 of 7 ")) {
		t.Errorf("N of M for the cursor, in the layer colour: %q", StripStyles(bottom))
	}
	ta.press("j", "j", "j", "j") // headings and separators do not count
	if bottom := covDTop(ta)[l.h-1]; !strings.HasSuffix(bottom, " 5 of 7 ━┛") {
		t.Errorf("Refresh is the fifth of seven rows: %q", bottom)
	}
	// A note has no cursor: the lines in sight.
	ta.press("Esc")
	ta.Viewer(testViewer())
	ta.settle()
	l = ta.top()
	if bottom := covDTop(ta)[l.h-1]; !strings.HasSuffix(bottom, fmt.Sprintf(" 1-%d of 40 ━┛", l.h-4)) {
		t.Errorf("a note writes N-M of T: %q", bottom)
	}
}

func TestErrorRow(t *testing.T) {
	ta := newTestApp(t)
	ta.Number(NumberInput{Title: "Port", Prompt: "1-65535", Min: 1, Max: 65535})
	ta.settle()
	l := ta.top()
	h0 := l.h
	// top, blank, prompt, blank, value, blank, error row, blank, bottom
	if p := covDTop(ta); p[6] != covDBlank(l.w) {
		t.Errorf("the error row is kept, blank, before any error: %q", p[6])
	}
	ta.press("0", "Enter")
	rows := covDRows(ta, l)
	p := covDPlain(rows)
	if l.h != h0 || p[5] != covDBlank(l.w) || !strings.HasPrefix(p[6], "┃ Port must be 1-65535") || !strings.HasPrefix(p[2], "┃ 1-65535") {
		t.Errorf("the error goes under the content, a row between, its left edge with the content's:\n%s", strings.Join(p, "\n"))
	}
	if c := covDStyleAt(rows[6], 2); c.fg != Red.RGB {
		t.Error("the error is Red")
	}
	if covDStyleAt(rows[0], 0).fg != LayerColor(1).RGB || covDStyleAt(rows[0], 3).bg != LayerColor(1).RGB ||
		covDStyleAt(rows[l.h-1], 0).fg != LayerColor(1).RGB {
		t.Error("the frame and the title keep the layer colour")
	}
	// Centred content, a centred error.
	ta.press("Esc")
	ta.Password(PasswordInput{Title: "PIN", Prompt: "Up to 12 digits", Unlock: true, MaxLen: 12,
		Check: func(string) error { return errors.New("Wrong PIN") }})
	ta.settle()
	ta.press("1", "2", "3", "4", "Enter")
	row := covDTop(ta)[6]
	if left, right := covDGaps(row); !strings.Contains(row, "Wrong PIN") || left <= 1 || left-right > 1 || right-left > 1 {
		t.Errorf("under centred content the error is centred: %q", row)
	}
	// A popup whose values cannot be wrong keeps no row for an error.
	ta.press("Esc", "Esc", "Space")
	if p := covDTop(ta); !strings.Contains(p[ta.top().h-3], "Global operation") {
		t.Errorf("the menu's last row is right above the bottom margin: %q", p[ta.top().h-3])
	}
}

func TestErrorRowTruncate(t *testing.T) {
	long := "the name may only hold letters, digits and dashes, and must start with a letter"
	ta := newTestApp(t)
	ta.resize(60, 20)
	ta.Text(TextInput{Title: "Rename", Prompt: "New name", Check: func(string) error { return errors.New(long) }})
	ta.settle()
	ta.press("x", "Enter")
	l := ta.top()
	p := covDTop(ta)
	if !strings.Contains(p[6], "the name may") || !strings.HasSuffix(strings.TrimRight(p[6], " ┃"), "…") ||
		strings.Contains(strings.Join(p, "\n"), "with a letter") || p[7] != covDBlank(l.w) {
		t.Errorf("the error takes one row, cut with …:\n%s", strings.Join(p, "\n"))
	}
	// In an input popup the error row takes no focus: typing goes on.
	ta.press("y")
	if p := covDTop(ta); !strings.Contains(p[4], "xy") {
		t.Errorf("typing after the error did not reach the value: %q", p[4])
	}
	// On a form the error row takes the focus and Enter shows the whole message.
	ta.press("Esc")
	ta.Form(&Form{Title: "New host", Fields: []*FormField{{Label: "Name", Check: func() string { return long }}}})
	ta.settle()
	ta.press("Tab", "Enter")
	if text := strings.Join(covDTop(ta), "\n"); !strings.Contains(text, "…") || strings.Contains(text, "with a letter") {
		t.Errorf("the form's error row is cut with …:\n%s", text)
	}
	ta.press("e", "Enter")
	n, ok := ta.topView().(*noteView)
	if !ok {
		t.Fatalf("Enter on the error row opened %T", ta.topView())
	}
	if got := strings.Join(strings.Fields(StripStyles(strings.Join(n.lines, " "))), " "); got != long {
		t.Errorf("the whole message: %q", got)
	}
}

func TestErrorRowChangesOnCheckOnly(t *testing.T) {
	ta := newTestApp(t)
	check := func(v string) error {
		switch {
		case v == "":
			return ErrEmpty
		case strings.Contains(v, " "):
			return errors.New("no spaces")
		}
		return nil
	}
	ta.Text(TextInput{Title: "Rename", Prompt: "New name", Check: check})
	ta.settle()
	errRow := func() string { return covDTop(ta)[6] }
	ta.press("Enter")
	if !strings.Contains(errRow(), "can't be empty") {
		t.Fatalf("Enter checks: %q", errRow())
	}
	ta.press("a", "Space", "b")
	if !strings.Contains(errRow(), "can't be empty") {
		t.Errorf("the error changed between two checks: %q", errRow())
	}
	ta.press("Enter")
	if !strings.Contains(errRow(), "no spaces") {
		t.Errorf("the next check writes its own error: %q", errRow())
	}
	ta.press("Backspace", "Backspace")
	if !strings.Contains(errRow(), "no spaces") {
		t.Errorf("a right value does not clear the error before the check: %q", errRow())
	}
	ta.press("Enter")
	if ta.top() != nil {
		t.Error("a value that passes the check is written back")
	}
	// A form checks when it is submitted, not when a value changes.
	r := &formResult{}
	ta.Form(testForm(r))
	ta.settle()
	ta.press("Tab", "Enter")
	if !strings.Contains(ta.screenText(), "Name is required") {
		t.Fatal("submitting with no name wrote no error")
	}
	ta.press("Enter", "w", "e", "b", "Enter") // the focus is on Name: edit it
	if r.name != "web" || !strings.Contains(ta.screenText(), "Name is required") {
		t.Errorf("the form's error changed before the next check: name %q", r.name)
	}
}

func TestActionFailureErrorPopup(t *testing.T) {
	ta := newTestApp(t)
	f := testForm(&formResult{name: "web-01", port: "2222"})
	f.Submit = func() (tea.Cmd, error) { return nil, errors.New("hosts.yaml changed on disk") }
	ta.Form(f)
	ta.settle()
	form := ta.topView()
	ta.press("Tab", "Enter")
	if _, ok := ta.topView().(*noteView); !ok || len(ta.stack) != 2 || ta.stack[0].view != form {
		t.Fatalf("the error popup goes over the form: %T, %d layers", ta.topView(), len(ta.stack))
	}
	ta.press("Esc")
	if ta.topView() != form {
		t.Fatalf("closing it goes back to the form: %T", ta.topView())
	}
	if text := ta.screenText(); !strings.Contains(text, "web-01") || !strings.Contains(text, "2222") {
		t.Errorf("the values are not all there:\n%s", text)
	}
}

func TestMenuHeadings(t *testing.T) {
	ta := newTestApp(t)
	ta.press("Space")
	rows := covDRows(ta, ta.top())
	for i, want := range map[int]string{2: "ITEM OPERATION", 8: "PANEL OPERATION"} {
		if got := StripStyles(rows[i]); !strings.HasPrefix(got, "┃ "+glyphHexagon+" "+want+" ") {
			t.Errorf("a heading is the hexagon, a space and the title in capitals, one cell in: %q", got)
		}
		for _, col := range []int{2, 4} {
			if c := covDStyleAt(rows[i], col); c.fg != Overlay0.RGB || c.bold {
				t.Errorf("%s: Overlay0, not bold: %+v", want, c)
			}
		}
	}
	ta.press("k")
	if got := covDCursor(ta); !strings.Contains(got, "Global operation") {
		t.Errorf("k on the first row skips the heading: %q", got)
	}
	ta.press("j")
	if got := covDCursor(ta); !strings.Contains(got, "[Enter] Logs") {
		t.Errorf("j round to the top skips the heading: %q", got)
	}
}

func TestMenuSeparator(t *testing.T) {
	ta := newTestApp(t)
	ta.press("Space")
	l := ta.top()
	rows := covDRows(ta, l)
	for _, i := range []int{7, 11} {
		if got, want := StripStyles(rows[i]), "┃ "+Repeat("╌", l.w-4)+" ┃"; got != want {
			t.Errorf("row %d: a dashed separator not touching the frame: %q", i, got)
		}
		if c := covDStyleAt(rows[i], 2); c.fg != Overlay0.RGB {
			t.Errorf("row %d: the separator is Overlay0", i)
		}
	}
	ta.press("j", "j", "j", "j") // from [D]rag over a separator and a heading
	if got := covDCursor(ta); !strings.Contains(got, "Refresh") {
		t.Errorf("the cursor stopped between sections: %q", got)
	}
}

func TestMenuCursor(t *testing.T) {
	ta := newTestApp(t)
	ta.press("Space")
	l := ta.top()
	cur := covDRows(ta, l)[3]
	layer := LayerColor(1).RGB
	for _, col := range []int{1, 2, l.w - 3} { // the margin, the label, the end of the description
		if c := covDStyleAt(cur, col); c.bg != layer || c.fg != Base.RGB || !c.bold {
			t.Errorf("cell %d of the cursor is not Base bold on the layer colour: %+v", col, c)
		}
	}
	if got := covDCell1(cur, l.w-2); got != glyphHard || covDStyleAt(cur, l.w-2).fg != layer {
		t.Errorf("the right-most cell is the arrow in the layer colour: %q", got)
	}
	// On the second layer, the second layer's colour.
	ta.press("G", "Enter")
	if got := covDCursor(ta); !strings.Contains(got, "[O]ther") {
		t.Errorf("the global popup's cursor is not a block of its layer colour: %q", got)
	}
}

func TestMenuTitle(t *testing.T) {
	ta := newTestApp(t)
	ta.press("Space")
	if top := covDTop(ta)[0]; !strings.Contains(top, glyphRoundLeft+glyphMenu+" [1] PODS"+glyphRoundRight) {
		t.Errorf("the Space menu is titled with the focused panel: %q", top)
	}
	ta.press("G", "Enter")
	if top := covDTop(ta)[0]; !strings.Contains(top, glyphMenu+" GLOBAL OPERATION"+glyphRoundRight) {
		t.Errorf("the global popup's title: %q", top)
	}
	ta.press("Esc", "Esc", "3", "Space")
	if top := covDTop(ta)[0]; !strings.Contains(top, glyphMenu+" [3] SETTINGS"+glyphRoundRight) {
		t.Errorf("the Space menu of [3]: %q", top)
	}
	ta.press("Esc")
	ta.Menu("Sort by", MenuSection{Actions: []Action{{Label: "Name", Desc: "a to z"}, {Label: "Modified time", Desc: "newest first, then by name"}}})
	ta.settle()
	if top := covDTop(ta)[0]; !strings.Contains(top, glyphMenu+" SORT BY"+glyphRoundRight) {
		t.Errorf("another menu says what it does: %q", top)
	}
	if w, want := ta.top().w, Width("Modified time  newest first, then by name")+4; w != want {
		t.Errorf("the width is known when it opens: the widest row + 4 = %d, have %d", want, w)
	}
}

func TestMenuGlobalRow(t *testing.T) {
	ta := newTestApp(t)
	// The second panel has nothing to act on: the row is under the cursor.
	for _, open := range [][]string{{"Space"}, {"Esc", "O", "Space"}} {
		ta.press(open...)
		row := strings.TrimRight(covDTop(ta)[ta.top().h-3], "┃ "+glyphHard)
		if !strings.HasPrefix(row, "┃ Global operation ") || !strings.HasSuffix(row, " actions for the whole app") {
			t.Errorf("the Global operation row has no hotkey and its own description: %q", row)
		}
	}
}

func TestMenuKeys(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(120, 14) // eight of twenty rows show: half a page is four
	var ran []string
	var acts []Action
	for i := range 20 {
		name := fmt.Sprintf("Row %02d", i)
		acts = append(acts, Action{Label: name, Desc: "a row of the list to run", Do: func() tea.Cmd { ran = append(ran, name); return nil }})
	}
	acts[5].Key = "r"
	open := func() {
		ta.Menu("Rows", MenuSection{Actions: acts})
		ta.settle()
		ta.Frame()
	}
	open()
	l := ta.top()
	if bottom := covDTop(ta)[l.h-1]; !strings.HasPrefix(bottom, "┗━ Enter:run Esc:close ━") {
		t.Errorf("the hint: %q", bottom)
	}
	for _, step := range []struct{ keys, want string }{
		{"G", "Row 19"}, {"d", "Row 19"}, {"g g", "Row 00"}, {"u", "Row 00"}, {"d", "Row 04"}, {"u", "Row 00"},
	} {
		ta.press(strings.Fields(step.keys)...)
		if got := covDCursor(ta); !strings.Contains(got, step.want) {
			t.Errorf("%s: the cursor is on %q, want %s", step.keys, strings.TrimSpace(got), step.want)
		}
	}
	ta.press("d", "Enter")
	if fmt.Sprint(ran) != "[Row 04]" || ta.top() != nil {
		t.Errorf("Enter runs the cursor's row and closes: %v", ran)
	}
	open()
	ta.press("r")
	if fmt.Sprint(ran) != "[Row 04 Row 05]" || ta.top() != nil {
		t.Errorf("a hotkey runs its row at once: %v", ran)
	}
	open()
	ta.press("Esc")
	if len(ran) != 2 || ta.top() != nil {
		t.Errorf("Esc closes, running nothing: %v", ran)
	}
}

func TestMenuWraps(t *testing.T) {
	ta := newTestApp(t)
	ta.press("Space", "k")
	if got := covDCursor(ta); !strings.Contains(got, "Global operation") {
		t.Errorf("k on the first row goes to the last: %q", got)
	}
	ta.press("j")
	if got := covDCursor(ta); !strings.Contains(got, "[Enter] Logs") {
		t.Errorf("j on the last row goes to the first: %q", got)
	}
	// The global popup's first row is the current screen, disabled: skipped.
	ta.press("G", "Enter", "k")
	if got := covDCursor(ta); !strings.Contains(got, "[q] Quit") {
		t.Errorf("k from the first usable row goes round to the last: %q", got)
	}
	ta.press("j")
	if got := covDCursor(ta); !strings.Contains(got, "[O]ther") {
		t.Errorf("j from the last goes round past the disabled row: %q", got)
	}
}

func TestConfirmLayout(t *testing.T) {
	ta := newTestApp(t)
	ta.press("x")
	l := ta.top()
	p := covDTop(ta)
	inner := func(i int) string { return strings.TrimSpace(strings.Trim(p[i], "┃")) }
	if l.h != 7 || !strings.HasPrefix(p[0], "┏━"+glyphRoundLeft+glyphConfirm+" DELETE"+glyphRoundRight) ||
		inner(1) != "" || inner(2) != "It is gone for good." || inner(3) != "" || inner(4) != "Delete it?" ||
		inner(5) != "" || !strings.HasPrefix(p[6], "┗━ Enter:delete") {
		t.Errorf("title, the details, a blank row, the question last, then the hint:\n%s", strings.Join(p, "\n"))
	}
	ta.press("Esc")
	ta.Confirm(Ask{Title: "Connect", Question: "Connect to web-01?", Verb: "connect"}, nil)
	ta.settle()
	if p = covDTop(ta); ta.top().h != 5 || inner(2) != "Connect to web-01?" {
		t.Errorf("without details there is only the question:\n%s", strings.Join(p, "\n"))
	}
}

func TestConfirmColors(t *testing.T) {
	ta := newTestApp(t)
	ta.Confirm(Ask{Title: "Delete", Question: "Delete 3 pods?", Verb: "delete",
		Details: []string{"3 pods match the filter.", "This can't be undone."}, Warn: []int{1}}, nil)
	ta.settle()
	rows := covDRows(ta, ta.top()) // top, blank, two details, blank, question, blank, bottom
	if c := covDStyleAt(rows[2], 2); c.fg != Overlay0.RGB {
		t.Errorf("the details are Overlay0: %v", c.fg)
	}
	if c := covDStyleAt(rows[3], 2); c.fg != Peach.RGB {
		t.Errorf("the warning is Peach: %v", c.fg)
	}
	if c := covDStyleAt(rows[5], 2); c.fg != Text.RGB || c.bold {
		t.Errorf("the question is Text, not bold: %+v", c)
	}
}

func TestConfirmWrapAndScroll(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(60, 16)
	var details []string
	for i := range 12 {
		details = append(details, fmt.Sprintf("pod-%02d in namespace default goes, with its volumes and its logs", i))
	}
	q := "Delete these twelve pods and everything that belongs to them, for good?"
	ta.Confirm(Ask{Title: "Delete", Question: q, Verb: "delete", Details: details}, nil)
	ta.settle()
	l := ta.top()
	if l.w != 58 {
		t.Errorf("content wider than the screen takes min(60 − 2, 120): %d", l.w)
	}
	inner := func(p []string, from, to int) string {
		var words []string
		for _, r := range p[from:to] {
			words = append(words, strings.Fields(strings.Trim(r, "┃"))...)
		}
		return strings.Join(words, " ")
	}
	p := covDTop(ta) // top, blank, 7 detail rows, blank, 2 question rows, blank, bottom
	if strings.Contains(strings.Join(p, "\n"), "…") || inner(p, l.h-4, l.h-2) != q ||
		!strings.HasPrefix(strings.Join(details, " "), inner(p, 2, l.h-5)) {
		t.Errorf("long text wraps, nothing is cut:\n%s", strings.Join(p, "\n"))
	}
	ta.press("j")
	after := covDTop(ta)
	if after[2] == p[2] || after[l.h-4] != p[l.h-4] || after[l.h-3] != p[l.h-3] {
		t.Errorf("j scrolls the details only, the question stays:\n%s", strings.Join(after, "\n"))
	}
	ta.press("Esc")
	short := "Delete web-01 and its 3 volumes now?"
	ta.Confirm(Ask{Title: "Delete", Question: short, Verb: "delete"}, nil)
	ta.settle()
	if w := ta.top().w; w != Width(short)+4 {
		t.Errorf("a short confirm is its widest row + 4: %d", w)
	}
}

func TestConfirmHint(t *testing.T) {
	ta := newTestApp(t)
	ta.press("x")
	if bottom := covDTop(ta)[ta.top().h-1]; !strings.HasPrefix(bottom, "┗━ Enter:delete Esc:cancel ━") {
		t.Errorf("Enter:<verb> Esc:cancel: %q", bottom)
	}
	ta.press("Esc", "q")
	if bottom := covDTop(ta)[ta.top().h-1]; !strings.HasPrefix(bottom, "┗━ Enter:quit Esc:cancel ━") {
		t.Errorf("Enter:<verb> Esc:cancel: %q", bottom)
	}
}
