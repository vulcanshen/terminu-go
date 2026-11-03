package terminu

import (
	"errors"
	"fmt"
	"slices"
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
			term := &Terminal{Title: "Alterm", Subject: strings.Repeat("myhost", 30), in: &strings.Builder{}, started: true, Interactive: true,
				Keys: []TerminalKey{{Key: "Alt-t", Desc: "hide"}}}
			term.mu.Lock()
			term.vt().write([]byte("中文 wide \U000F024B icons and a line long enough to wrap around the screen\r\n$ "))
			term.mu.Unlock()
			ta.open(&termView{t: term, a: ta.App})
		},
		// Bigger than the narrow screens, then bigger than any, then rows
		// the app drew too wide and too many.
		"app popup": func(ta *testApp) { ta.Popup(testPopup(&testBody{w: 52, h: 21})) },
		"app popup huge": func(ta *testApp) {
			ta.Popup(Popup{Title: "Draw", Subject: strings.Repeat("kamonu", 30), Body: &testBody{w: 300, h: 100}})
		},
		"app popup sloppy": func(ta *testApp) { ta.Popup(testPopup(&testBody{w: 40, h: 6, sloppy: true})) },
	}
}

func testViewer() *Viewer {
	var lines []string
	for i := range 40 {
		lines = append(lines, fmt.Sprintf("key%d: value %d", i, i))
	}
	return &Viewer{Title: "YAML", Subject: "pod/nginx", Lines: lines}
}

// TestViewerSelectsByDefault: every viewer has a selection mode; with no
// Copy of the app's, y puts the text on the terminal's clipboard and a
// Success toast says so (tdp note).
func TestViewerSelectsByDefault(t *testing.T) {
	ta := newTestApp(t)
	takeClipboard()
	ta.Viewer(testViewer())
	ta.settle()
	if h := StripStyles(ta.rowWith("Esc:close")); !strings.Contains(h, "v:visual Esc:close") {
		t.Errorf("v in its hint: %q", h)
	}
	ta.press("v", "v", "l", "l", "y")
	if got := string(takeClipboard()); got != "\x1b]52;c;a2V5\a" { // "key"
		t.Errorf("on the clipboard: %q", got)
	}
	if ta.toast == nil || ta.toast.view.(*toastView).kind != Success || strings.Join(ta.toast.view.(*toastView).text, " ") != "Copied" {
		t.Error("a Success toast says it is copied")
	}
}

// TestViewerAlwaysWraps: a viewer folds a line too long for its row and
// never cuts one with …, NoWrap set or not, so nothing in it is out of sight
// sideways: not at the selection's cursor at the end of a long line either;
// an app may leave out its line numbers (tdp note.2, note-4).
func TestViewerAlwaysWraps(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(40, 20)
	digits := strings.Repeat("0123456789", 6) // 60 characters
	var got string
	ta.Viewer(&Viewer{Title: "Info", Lines: []string{"NAME    SIZE", digits, "end"}, NoWrap: true, NoNumbers: true,
		Select: &Selection{Key: "v", Name: "Visual", Copy: func(s string) tea.Cmd { got = s; return nil }}})
	ta.settle()
	lines, _ := covFDraw(ta)
	text := covFText(lines)
	row := func(i int) string { return strings.TrimSpace(strings.Trim(text[i], "┃")) }
	if !strings.HasPrefix(text[2], "┃ NAME    SIZE") {
		t.Errorf("no line numbers, the text one cell in: %q", text[2])
	}
	if row(3)+row(4) != digits || row(5) != "end" || strings.Contains(strings.Join(text, "\n"), "…") {
		t.Errorf("the long line folds onto the next row, the next line under it: %q", text[2:6])
	}
	ta.press("v", "j", "v", "$")
	lines, _ = covFDraw(ta)
	if text = covFText(lines); row(3)+row(4) != digits || strings.Contains(strings.Join(text, "\n"), "…") {
		t.Errorf("the cursor at the end of the long line scrolled sideways: %q", text[2:6])
	}
	if ta.press("y"); got != digits {
		t.Errorf("all of it copied: %q", got)
	}
	// Four rows that fit: no scroll keys in its key reference (tdp M6).
	ta.press("?")
	if keys := covAKeys(covARef(t, ta)); keys["j|k"] {
		t.Error("lines that fit list j|k")
	}
}

// TestViewerSearchHits: a search's hits are underlined in their own colours,
// the current one drawn as the cursor block; no Yellow (tdp note).
func TestViewerSearchHits(t *testing.T) {
	ta := newTestApp(t)
	v := &Viewer{Title: "YAML", Lines: []string{"name: Nginx", "image: nginx:1.27"}, Lang: "yaml"}
	v.Hits = v.FindText("NGINX")
	if want := []TextMatch{{0, 6, 11}, {1, 7, 12}}; !slices.Equal(v.Hits, want) {
		t.Fatalf("FindText: %v, want %v", v.Hits, want)
	}
	if got := (&Viewer{Lines: []string{"aaaa"}}).FindText("aa"); !slices.Equal(got, []TextMatch{{0, 0, 2}, {0, 2, 4}}) {
		t.Errorf("hits do not overlap: %v", got)
	}
	if got := (&Viewer{Lines: []string{"\x1b[31mred\x1b[0m line"}}).FindText("red line"); !slices.Equal(got, []TextMatch{{0, 0, 8}}) {
		t.Errorf("in a coloured line, the characters as drawn: %v", got)
	}
	v.Current = 1
	ta.Viewer(v)
	ta.settle()
	first, second := ta.rowWith("name: Nginx"), ta.rowWith("image: nginx")
	if !strings.Contains(first, "\x1b[4m") || strings.Contains(first, sgrRGB(false, Yellow.RGB)) {
		t.Errorf("a hit underlined, no Yellow: %q", first)
	}
	if !strings.Contains(second, On(LayerColor(1)).Render("n")) {
		t.Errorf("the current hit is the cursor block: %q", second)
	}
	if strings.Contains(StripStyles(first), "\x1b") {
		t.Error("no stray escapes")
	}
}

// TestViewerSearch: / asks for the text in an input popup with the app's
// words; the hits are found and the first is in sight; n|N go to the next and
// the previous, wrapping at the ends and scrolling only when it is out of
// sight; with none an Info toast; the hint has /:search but not n|N, which
// are in ? while there are hits; the hits go when the viewer closes (tdp
// note.3, note-86).
func TestViewerSearch(t *testing.T) {
	ta := newTestApp(t)
	lines := make([]string, 60)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d", i+1)
	}
	lines[4], lines[39], lines[40] = "image: nginx", "name: nginx", "app: NGINX"
	v := &Viewer{Title: "YAML", Lines: lines, Search: &ViewerSearch{Title: "Search", Prompt: "Find in the YAML"}}
	ta.Viewer(v)
	ta.settle()
	if h := fmt.Sprint(ta.topView().(*viewerView).hint()); h != "[{/ search} {v visual} {Esc close}]" {
		t.Errorf("hint: %s", h)
	}
	inSight := func(line int) bool { return v.top <= line && line < v.top+v.visible }
	find := func(q string) {
		t.Helper()
		ta.press("/")
		if text := ta.screenText(); !strings.Contains(text, "SEARCH") || !strings.Contains(text, "Find in the YAML") {
			t.Fatalf("/ did not ask for the text:\n%s", text)
		}
		for _, c := range q {
			ta.press(string(c))
		}
		ta.press("Enter")
	}
	find("nginx")
	if len(v.Hits) != 3 || v.Current != 0 || v.top != 0 {
		t.Fatalf("found %v, current %d, top %d: the first in sight, nothing scrolled", v.Hits, v.Current, v.top)
	}
	if ta.press("n"); v.Current != 1 || !inSight(39) {
		t.Errorf("n: current %d, top %d", v.Current, v.top)
	}
	if ta.press("n", "n"); v.Current != 0 || !inSight(4) {
		t.Errorf("n past the last: current %d, top %d", v.Current, v.top)
	}
	if ta.press("N"); v.Current != 2 || !inSight(40) {
		t.Errorf("N past the first: current %d, top %d", v.Current, v.top)
	}
	top := v.top // the hit before is in sight above it
	if ta.press("N"); v.Current != 1 || v.top != top {
		t.Errorf("N to a hit in sight: current %d, scrolled from %d to %d", v.Current, top, v.top)
	}
	var move []string
	for _, s := range ta.topView().keyRef() {
		for _, r := range s.rows {
			move = append(move, r.Key)
		}
	}
	if !slices.Contains(move, "/") || !slices.Contains(move, "n|N") {
		t.Errorf("? with hits: %q", move)
	}
	find("zzz")
	if tv, ok := ta.toast.view.(*toastView); !ok || tv.kind != Info || !strings.Contains(ta.screenText(), "No line has zzz") || len(v.Hits) != 0 {
		t.Errorf("no hit: toast %v, hits %v", ta.toast, v.Hits)
	}
	ta.press("n", "Esc", "G") // n with no hit does nothing; Esc the toast
	if find("nginx"); !inSight(4) {
		t.Errorf("found from the bottom: top %d, the first hit out of sight", v.top)
	}
	ta.press("Esc")
	if ta.top() != nil || v.Hits != nil || v.Current != 0 {
		t.Errorf("closed: popup %v, hits %v", ta.top() != nil, v.Hits)
	}
}

// TestNoteEveryViewerSearches: every viewer has / search, the App Log and
// one the app opens without a Search too; a viewer whose own keys take / runs
// the app's search; the key reference has one, its hint /:search ?|Esc:close
// (tdp note.3, note.5, note-88, note-61).
func TestNoteEveryViewerSearches(t *testing.T) {
	ta := newTestApp(t)
	asks := func(what string) {
		t.Helper()
		ta.press("/")
		if _, ok := ta.topView().(*lineInput); !ok || !strings.Contains(ta.screenText(), "SEARCH") {
			t.Errorf("%s: / does not ask for the text: %T", what, ta.topView())
		}
		ta.press("Esc")
	}
	ta.press("!")
	if v := ta.topView().(*viewerView); v.v.Search == nil || !slices.Contains(v.hint(), KeyHint{"/", "search"}) {
		t.Errorf("the App Log has no search: %v", v.hint())
	}
	asks("the App Log")
	ta.press("Esc")
	v := &Viewer{Title: "Log", Lines: []string{"one", "two"}}
	ta.Viewer(v)
	ta.settle()
	asks("a viewer the app opens without a Search")
	ta.press("Esc")
	own := 0
	v = &Viewer{Title: "Page", Lines: []string{"one"}, Keys: []ViewerKey{{Key: "/", Desc: "find", Do: func() tea.Cmd { own++; return nil }}}}
	ta.Viewer(v)
	ta.settle()
	if ta.press("/"); v.Search != nil || own != 1 {
		t.Errorf("a viewer whose keys take /: the lib search %v, the app's ran %d times", v.Search != nil, own)
	}
	ta.press("Esc", "?")
	if n := ta.topView().(*noteView); fmt.Sprint(n.hint) != "[{/ search} {?|Esc close}]" {
		t.Errorf("the key reference's hint: %v", n.hint)
	}
	asks("the key reference")
	if _, ok := ta.topView().(*noteView); !ok {
		t.Errorf("Esc on the search goes back to the key reference: %T", ta.topView())
	}
}

// TestKeyReferenceSearch: a key reference searches as a viewer does: the
// hits underlined in their own colours, the current one the cursor block, the
// first in sight; n|N go to the next and the previous, wrapping at the ends
// and scrolling to it; with none an Info toast (tdp note.5, note.3).
func TestKeyReferenceSearch(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(120, 12)
	ta.press("?")
	ta.Frame() // drawn, as after every key
	n := ta.topView().(*noteView)
	want := 0
	for _, l := range n.lines {
		want += strings.Count(strings.ToLower(StripStyles(l)), "this")
	}
	if want < 4 {
		t.Fatalf("the test wants hits far apart: %d", want)
	}
	inSight := func(line int) bool { return n.top <= line && line < n.top+n.visible }
	press := func(keys ...string) { ta.press(keys...); ta.Frame() } // drawn after every key, as in an app
	press("G")
	press("/", "t", "h", "i", "s", "Enter")
	if len(n.hits) != want || n.current != 0 || !inSight(n.hits[0].Line) {
		t.Fatalf("found from the bottom %v of %d, current %d, top %d: the first in sight", n.hits, want, n.current, n.top)
	}
	row := ta.rowWith(StripStyles(n.lines[n.hits[0].Line])[1:])
	if !strings.Contains(row, On(LayerColor(1)).Render("t")) {
		t.Errorf("the current hit is the cursor block: %q", row)
	}
	last := n.hits[len(n.hits)-1]
	if last.Line == n.hits[0].Line {
		t.Fatal("the test wants hits on more than one row")
	}
	for i := 1; i < len(n.hits); i++ {
		line, below := n.hits[i].Line, n.hits[i].Line >= n.top+n.visible
		if press("n"); n.current != i || !inSight(line) {
			t.Errorf("n to hit %d: current %d, top %d", i, n.current, n.top)
		}
		if below && n.top != line-n.visible+1 {
			t.Errorf("n to hit %d below: top %d, scrolled more than it takes", i, n.top)
		}
	}
	if press("n"); n.current != 0 || !inSight(n.hits[0].Line) {
		t.Errorf("n past the last: current %d, top %d", n.current, n.top)
	}
	if other := n.hits[1]; !inSight(other.Line) || other.Line == n.hits[0].Line {
		t.Fatalf("the test wants the second hit in sight on a row of its own: %v, top %d", other, n.top)
	} else if row := ta.rowWith(StripStyles(n.lines[other.Line])[1:]); !strings.Contains(row, "\x1b[4m") || strings.Contains(row, On(LayerColor(1)).Render("t")) {
		t.Errorf("a hit not current is underlined, not the cursor block: %q", row)
	}
	if press("N"); n.current != len(n.hits)-1 || !inSight(last.Line) {
		t.Errorf("N past the first: current %d, top %d", n.current, n.top)
	}
	keys := map[string]bool{}
	for _, sec := range n.keyRef() {
		for _, r := range sec.rows {
			keys[r.Key] = true
		}
	}
	if !keys["/"] || !keys["n|N"] {
		t.Errorf("its keys with hits: %v", keys)
	}
	ta.resize(110, 12)
	ta.Frame()
	if len(n.hits) != want || n.current != len(n.hits)-1 {
		t.Errorf("the hits after a resize: %d of %d, current %d", len(n.hits), want, n.current)
	}
	press("/", "z", "z", "z", "Enter")
	if tv, ok := ta.toast.view.(*toastView); !ok || tv.kind != Info || !strings.Contains(ta.screenText(), "No line has zzz") || len(n.hits) != 0 {
		t.Errorf("no hit: toast %v, hits %v", ta.toast, n.hits)
	}

	// A row's description folds to the room there is: narrower, the hit is
	// found where it is now.
	a := anySize(New("t"))
	vals := &Values{Fields: []*Field{{Label: "Timeout", Value: func() string { return "5" },
		Desc: "how long the screen waits before it locks itself again, counted in whole minutes"}}}
	a.AddScreen(&Screen{Name: "S", Panels: []*Panel{{Label: "Settings", Body: vals}}})
	tb := &testApp{App: a}
	tb.resize(160, 30)
	tb.press("?")
	tb.Frame()
	n = tb.topView().(*noteView)
	tb.press("/", "m", "i", "n", "u", "t", "e", "s", "Enter")
	tb.resize(40, 30)
	tb.Frame()
	if len(n.hits) != 1 || !strings.Contains(StripStyles(n.lines[n.hits[0].Line]), "minutes") {
		t.Errorf("narrower: hits %v in %q", n.hits, n.lines)
	}
}

// TestT1SearchFromMenu: a search run from the Space menu clears the menu
// once it runs, found or not, the input popup and the menu closing together;
// Esc on the input popup goes back to the menu. The lib's own searches are
// on popups, which have no Space menu (tdp K5); a panel's search is the app's,
// its Done ending with CloseAll; the list's filter opens nothing, so the menu
// goes. One Esc on a menu left over the hits closes only the menu (tdp T1,
// T1-10, K4).
func TestT1SearchFromMenu(t *testing.T) {
	for _, closeAll := range []bool{true, false} {
		ta := &testApp{App: anySize(New("t"))}
		hits := ""
		pg := &Pager{}
		pg.SetLines([]string{"alpha", "beta"})
		p := &Panel{Label: "Text", Body: pg, OnEsc: func() tea.Cmd { hits = ""; return nil }}
		p.Actions = []Action{{Key: "/", Label: "Search", Desc: "the text", Scope: PanelScope, Do: func() tea.Cmd {
			return ta.Text(TextInput{Title: "Search", Verb: "find", Search: true, Done: func(q string) tea.Cmd {
				hits = q
				if closeAll {
					return ta.CloseAll()
				}
				return nil
			}})
		}}}
		ta.AddScreen(&Screen{Name: "S", Panels: []*Panel{p}})
		ta.resize(80, 24)
		ta.press("Space", "/")
		if _, ok := ta.topView().(*lineInput); !ok || len(ta.stack) != 2 {
			t.Fatalf("the search does not ask over the menu: %T, %d layers", ta.topView(), len(ta.stack))
		}
		if ta.press("Esc"); len(ta.stack) != 1 {
			t.Fatalf("Esc on the input popup: %d layers", len(ta.stack))
		}
		if _, ok := ta.topView().(*menuView); !ok {
			t.Fatalf("Esc on the input popup goes back to the menu: %T", ta.topView())
		}
		ta.press("/", "a")
		ta.Update(keyMsg("Enter"))
		if !closeAll {
			ta.settle()
			if _, ok := ta.topView().(*menuView); !ok || hits != "a" {
				t.Fatalf("a menu left over the hits: %T, hits %q", ta.topView(), hits)
			}
			if ta.press("Esc"); ta.top() != nil || hits != "a" {
				t.Errorf("one Esc on the menu: popup %v, hits %q; it closes the menu only", ta.top() != nil, hits)
			}
			continue
		}
		if len(ta.stack) != 2 || !ta.stack[0].closing || !ta.stack[1].closing || hits != "a" {
			t.Fatalf("Enter: the input popup and the menu do not close together: %d layers, hits %q", len(ta.stack), hits)
		}
		if ta.settle(); ta.top() != nil {
			t.Errorf("a popup left: %T", ta.topView())
		}
		if ta.press("Esc"); hits != "" {
			t.Error("Esc on the panel clears the hits")
		}
	}
	ta := newTestApp(t)
	ta.press("Space", "/")
	if ta.top() != nil || !ta.list.Typing() {
		t.Errorf("the filter from the menu: popup %v, typing %v", ta.top() != nil, ta.list.Typing())
	}
}

// TestErrorPopupBeforeTheSize: an error popup opened before the terminal's
// size is known (App.Start) folds at the width it gets once it is: not a
// character a row (from sshu).
func TestErrorPopupBeforeTheSize(t *testing.T) {
	ta := &testApp{App: anySize(New("t"))}
	ta.AddScreen(&Screen{Name: "S", Panels: []*Panel{{Label: "P", Body: &Pager{}}}})
	ta.Error("Connect", "ssh: connect to host web-01 port 22: Connection refused")
	ta.resize(100, 30)
	n := ta.topView().(*noteView)
	if len(n.lines) != 1 || !strings.Contains(StripStyles(n.lines[0]), "Connection refused") {
		t.Errorf("folded at the real width: %q", n.lines)
	}
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
	if h := vv.hint(); len(h) != 4 || h[0].Key != "y" || h[1] != (KeyHint{"/", "search"}) || h[2].Key != "v" || h[3] != (KeyHint{"Esc", "close"}) {
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
	if h := ta.topView().(*viewerView).hint(); fmt.Sprint(h) != "[{y copy} {Esc unselect}]" {
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
	// Esc a layer at a time: selecting, back to Visual, the cursor where it
	// is; then out of the whole mode, and only the mode (note-37, K11-3).
	ta.press("v", "l", "v", "l", "Esc")
	if !v.inMode || v.selecting || v.col != 2 || ta.top() == nil {
		t.Errorf("Esc while selecting: in the mode %v, selecting %v, cursor at %d", v.inMode, v.selecting, v.col)
	}
	ta.press("Esc")
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

// TestConfirmKeys: Enter and Esc are the only answers (tdp F6): y and n
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

// TestKeyReferenceTitleKeepsName: the key reference of a viewer upper-cases
// what it shows and "keys", not the name of the thing (components, upper
// case; tdp note).
func TestKeyReferenceTitleKeepsName(t *testing.T) {
	ta := newTestApp(t)
	v := testViewer()
	v.Title, v.Subject = "First page", "Pride and Prejudice"
	ta.Viewer(v)
	ta.settle()
	ta.press("?")
	if n, ok := ta.topView().(*noteView); !ok || n.title != "FIRST PAGE — Pride and Prejudice KEYS" {
		t.Errorf("title %q", ta.topView().(*noteView).title)
	}
}

// TestPopupSubjectErrorMenu: an error popup and a menu about one thing keep
// its name as it is after ": " in the title (components, upper case;
// tdp note; from sshu, filu).
func TestPopupSubjectErrorMenu(t *testing.T) {
	ta := newTestApp(t)
	ta.ErrorAbout("Rename failed", "notes.md", "a file called notes.md is already there")
	ta.settle()
	if n, ok := ta.topView().(*noteView); !ok || n.title != "RENAME FAILED: notes.md" {
		t.Errorf("error popup title %q", n.title)
	}
	ta.covBCloseAll()
	ta.MenuAbout("Open with", "notes.md", MenuSection{Actions: []Action{{Label: "Editor", Desc: "edit it"}}})
	ta.settle()
	if text := ta.screenText(); !strings.Contains(text, "OPEN WITH: notes.md") {
		t.Errorf("menu title:\n%s", text)
	}
}

// TestMessageRows: an error popup's message and a toast's text come a row at
// a time: each on a row of its own, folded when too long, no blank row
// between; a \n inside a row is from outside and drawn by its name in Red
// (components-31; components, control characters).
func TestMessageRows(t *testing.T) {
	ta := newTestApp(t)
	ta.Error("Deploy failed", "Stopped.", "api-7d4f-01: readiness probe failed")
	ta.settle()
	text := ta.screenText()
	lines := strings.Split(text, "\n")
	first, second := -1, -1
	for i, l := range lines {
		if strings.Contains(l, "Stopped.") {
			first = i
		}
		if strings.Contains(l, "api-7d4f-01") {
			second = i
		}
	}
	if first < 0 || second != first+1 {
		t.Errorf("the rows of an error popup: %d, %d\n%s", first, second, text)
	}
	ta.press("Esc")
	ta.Toast(Info, "Saved.", "It takes effect now.")
	ta.settle()
	text = ta.screenText()
	if r1, r2 := strings.Index(text, "Saved."), strings.Index(text, "It takes effect now."); r1 < 0 || r2 < 0 ||
		strings.Count(text[r1:r2], "\n") != 1 {
		t.Errorf("the rows of a toast:\n%s", text)
	}
	if !strings.Contains(strings.Join(ta.logs.lines(), "\n"), "Saved. It takes effect now.") {
		t.Error("the App Log does not have the toast on one line")
	}
	ta.Toast(Info, "a\nb")
	if !strings.Contains(strings.Join(ta.Frame(), "\n"), nameSGR+`\n`) {
		t.Error("a \\n inside a row is not drawn by its name")
	}
}
