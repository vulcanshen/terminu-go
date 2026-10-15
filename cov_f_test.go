package terminu

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Tests for docs/coverage.md, "input/README" and "input — 各種".

// covFKey is keyMsg plus the keys an input needs: Home, End, Delete, Ctrl-
// and Alt- combinations.
func covFKey(name string) tea.KeyMsg {
	switch name {
	case "Home":
		return tea.KeyMsg{Type: tea.KeyHome}
	case "End":
		return tea.KeyMsg{Type: tea.KeyEnd}
	case "Delete":
		return tea.KeyMsg{Type: tea.KeyDelete}
	case "Ctrl-U":
		return tea.KeyMsg{Type: tea.KeyCtrlU}
	case "Ctrl-W":
		return tea.KeyMsg{Type: tea.KeyCtrlW}
	case "Ctrl-A":
		return tea.KeyMsg{Type: tea.KeyCtrlA}
	}
	if r, ok := strings.CutPrefix(name, "Alt-"); ok {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(r), Alt: true}
	}
	return keyMsg(name)
}

func covFPress(ta *testApp, keys ...string) {
	for _, k := range keys {
		ta.Update(covFKey(k))
		ta.settle()
	}
}

// covFType types s one character (one key) at a time.
func covFType(ta *testApp, s string) {
	for _, r := range s {
		ta.press(string(r))
	}
}

func covFPaste(ta *testApp, s string) {
	ta.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s), Paste: true})
	ta.settle()
}

// covFDraw is the top popup alone, drawn in its layer colour.
func covFDraw(ta *testApp) ([]string, Color) {
	top := ta.top()
	for i, l := range ta.stack {
		if l == top {
			c := LayerColor(i + 1)
			return l.view.draw(l.w, l.h, c), c
		}
	}
	return nil, Color{}
}

func covFText(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = StripStyles(l)
	}
	return out
}

// covFScreen is the top popup's text, covFHint its bottom line's.
func covFScreen(ta *testApp) []string {
	lines, _ := covFDraw(ta)
	return covFText(lines)
}

func covFHint(ta *testApp) string {
	text := covFScreen(ta)
	return text[len(text)-1]
}

// covFRow is the drawn line whose text holds s.
func covFRow(lines []string, s string) string {
	for _, l := range lines {
		if strings.Contains(StripStyles(l), s) {
			return l
		}
	}
	return ""
}

// covFValue is the value cell of a one-line input popup: its fifth row
// without the frame, the cell of padding on each side and the angle before
// the value (the unlock look has none).
func covFValue(ta *testApp) string {
	lines, _ := covFDraw(ta)
	row := StripStyles(lines[4])
	return strings.TrimPrefix(strings.TrimSuffix(strings.TrimPrefix(row, "┃ "), " ┃"), glyphTyping+" ")
}

func covFLine(ta *testApp) *lineInput { return ta.topView().(*lineInput) }

// covFOpen is the escape sequence a text drawn in s starts with.
func covFOpen(s Style) string { return "\x1b[" + s.sgr() + "m" }

func covFBlank(row string) bool { return strings.TrimSpace(strings.Trim(row, "┃")) == "" }

// covFClock fixes today at 2026-10-07 14:12.
func covFClock(t *testing.T) {
	old := clock
	clock = func() time.Time { return time.Date(2026, 10, 7, 14, 12, 0, 0, time.Local) }
	t.Cleanup(func() { clock = old })
}

func covFDay(d, h, m int) time.Time { return time.Date(2026, 10, d, h, m, 0, 0, time.Local) }

// --- input/README ------------------------------------------------------------

func TestInputCommon(t *testing.T) {
	ta := newTestApp(t)
	got := "unset"
	text := TextInput{Title: "Name", Done: func(v string) tea.Cmd { got = v; return nil }}
	// The typing row is the input state: q, ? and Space are characters.
	ta.Text(text)
	ta.settle()
	ta.press("q", "?", "Space", "x")
	if li, ok := ta.topView().(*lineInput); !ok || li.ed.value() != "q? x" || ta.quitting {
		t.Fatalf("q ? Space are typed: %T, quitting %v", ta.topView(), ta.quitting)
	}
	// Esc closes, the value unchanged.
	ta.press("Esc")
	if ta.top() != nil || got != "unset" {
		t.Fatalf("Esc closes without writing: top %v, got %q", ta.top(), got)
	}
	// Enter writes the value back and closes (Rules K3).
	ta.Text(text)
	ta.settle()
	ta.press("o", "k", "Enter")
	if ta.top() != nil || got != "ok" {
		t.Errorf("Enter writes back and closes: top %v, got %q", ta.top(), got)
	}

	// The part one does not type in (a select's list) keeps the core keys
	// (Rules K1): Space does nothing, j moves, ? opens the key reference,
	// q and Ctrl-C start the quit flow.
	ta.Select(Select{Title: "Profile", Options: []string{"a", "b", "c"}})
	ta.settle()
	fv := ta.topView().(*finderView)
	ta.press("Space")
	if ta.topView() != popupView(fv) {
		t.Errorf("Space on the list opened %T", ta.topView())
	}
	ta.press("j")
	if fv.cursor != 1 {
		t.Errorf("j moves: cursor %d", fv.cursor)
	}
	ta.press("?")
	if n, ok := ta.topView().(*noteView); !ok || n.title != "PROFILE KEYS" {
		t.Errorf("? on the list opens the key reference: %T", ta.topView())
	}
	ta.press("Esc")
	for _, quit := range []string{"q", "Ctrl-C"} {
		ta.press(quit)
		if _, ok := ta.topView().(*confirmView); !ok || !ta.quitting {
			t.Errorf("%s on the list starts the quit flow: %T", quit, ta.topView())
		}
		ta.press("Esc")
	}

	// Opened from a form, the focus stays on the field after Enter.
	ta = newTestApp(t)
	r := &formResult{}
	ta.Form(testForm(r))
	ta.settle()
	form := ta.topView().(*formView)
	ta.press("j", "Enter", "2", "2", "Enter")
	if r.port != "22" || ta.topView() != popupView(form) || formStop(form) != "Port" {
		t.Errorf("back on the form at Port: port %q, top %T, focus %s", r.port, ta.topView(), formStop(form))
	}
	ta.press("j", "j", "j", "j", "j", "Enter", "j", "Enter")
	if formStop(form) != "Profile" {
		t.Errorf("after a select the focus stays on Profile, not %s", formStop(form))
	}
}

// Every one-line input's typing row starts with the angle in the layer
// colour, one cell from the frame, then a space; the suggestion and the grey
// dots start right after it, the cursor on their first character
// (components/input/README).
func TestInputAngle(t *testing.T) {
	ta := newTestApp(t)
	angle := typingAngle(layerColors[0])
	opens := []struct {
		name string
		open func()
		text string // what follows the angle
	}{
		{"text", func() { ta.Text(TextInput{Title: "Name"}) }, ""},
		{"text, the old value", func() { ta.Text(TextInput{Title: "Name", Old: "web-01"}) }, "web-01"},
		{"number", func() { ta.Number(NumberInput{Title: "Port", Min: 1, Max: 65535}) }, ""},
		{"password", func() { ta.Password(PasswordInput{Title: "Password"}) }, ""},
		{"password, set", func() { ta.Password(PasswordInput{Title: "Password", Set: true}) }, "••••••••"},
	}
	for _, o := range opens {
		o.open()
		ta.settle()
		lines, _ := covFDraw(ta)
		plain := StripStyles(lines[4])
		if !strings.Contains(lines[4], angle) || !strings.HasPrefix(plain, "┃ "+glyphTyping+" ") ||
			!strings.HasPrefix(covFValue(ta), cmpOr(o.text, " ")) {
			t.Errorf("%s: %q", o.name, plain)
		}
		ta.press("Esc")
	}

	// A text field a form opens is the same popup.
	ta.Form(testForm(nil))
	ta.settle()
	ta.press("Enter")
	if lines, _ := covFDraw(ta); !strings.HasPrefix(StripStyles(lines[4]), "┃ "+glyphTyping+" ") {
		t.Errorf("from a form: %q", StripStyles(lines[4]))
	}
	for len(ta.stack) > 0 {
		ta.press("Esc")
	}

	// The width a length limit gives counts the angle.
	ta.Text(TextInput{Title: "Code", MaxLen: 80})
	ta.settle()
	if w := ta.top().w; w != 80+angleWidth()+1+4 {
		t.Errorf("80 characters at most: %d wide, want %d", w, 80+angleWidth()+1+4)
	}
}

func TestSingleLineSkeleton(t *testing.T) {
	ta := newTestApp(t)
	ta.Text(TextInput{Title: "Rename", Prompt: "New name for assets", Verb: "rename"})
	ta.settle()
	ta.press("a", "s", "s", "e", "t", "s")
	lines, _ := covFDraw(ta)
	text := covFText(lines)
	if len(lines) != 9 {
		t.Fatalf("%d rows:\n%s", len(lines), strings.Join(text, "\n"))
	}
	// A blank row, the prompt, a blank row, the value, a blank row, the error row.
	if !covFBlank(text[1]) || !strings.HasPrefix(text[2], "┃ New name for assets ") || !covFBlank(text[3]) ||
		!strings.HasPrefix(text[4], "┃ "+glyphTyping+" assets ") || !covFBlank(text[5]) || !covFBlank(text[6]) {
		t.Errorf("skeleton:\n%s", strings.Join(text, "\n"))
	}
	if !strings.Contains(lines[2], Fg(Overlay0).Render("New name for assets")) {
		t.Error("the prompt is not Overlay0")
	}
	if !strings.Contains(lines[4], Fg(Lavender).Render("a")) || !strings.Contains(lines[4], Style{Fg: Base, Bg: Lavender}.Render(" ")) {
		t.Error("the value is Lavender, the cursor a Lavender block with Base text")
	}
	if !strings.Contains(text[8], " Enter:rename Esc:cancel ") {
		t.Errorf("hint: %s", text[8])
	}
	// Width: free typing takes Rules F7's most.
	if w := ta.top().w; w != 118 {
		t.Errorf("free typing is %d wide, want 118", w)
	}
	ta.press("Esc")

	// The error row is under the value, a blank row between, in Red.
	ta.Text(TextInput{Title: "Rename", Prompt: "New name", Check: func(string) error { return errors.New("Name is taken") }})
	ta.settle()
	ta.press("x", "Enter")
	lines, _ = covFDraw(ta)
	if !strings.Contains(lines[6], Fg(Red).Render("Name is taken")) || !covFBlank(StripStyles(lines[5])) {
		t.Errorf("error row: %q", lines[6])
	}
	ta.press("Esc")

	// With a length limit the width is known when it opens.
	ta.Text(TextInput{Title: "Code", Prompt: "Six letters", MaxLen: 6})
	ta.settle()
	if w := ta.top().w; w >= 118 {
		t.Errorf("a value with a length limit is %d wide: its content's width", w)
	}
}

func TestInputLongValue(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(30, 20)
	ta.Text(TextInput{Title: "Path"})
	ta.settle()
	path := "/Users/vulcan/Documents/sideproj/terminu/components/layout"
	covFType(ta, path)
	v := covFValue(ta)
	if Width(v) != 24-angleWidth() || !strings.HasPrefix(v, "…") || !strings.HasSuffix(v, "layout ") {
		t.Errorf("typing at the end: the front is cut, the cursor shows: %q", v)
	}
	covFPress(ta, "Home")
	v = covFValue(ta)
	if !strings.HasPrefix(v, "/Users") || !strings.HasSuffix(v, "…") {
		t.Errorf("after Home the end is cut: %q", v)
	}
	// It scrolls only as far as it must: moving right from the start does
	// not scroll until the cursor reaches the edge.
	for range 10 {
		covFPress(ta, "→")
	}
	if v = covFValue(ta); !strings.HasPrefix(v, "/Users") {
		t.Errorf("ten to the right scrolled: %q", v)
	}
	// Nor when the cursor steps back from the end: it is still in sight.
	covFPress(ta, "End")
	end := covFValue(ta)
	covFPress(ta, "←")
	if v = covFValue(ta); v != end {
		t.Errorf("one to the left of the end scrolled: %q, was %q", v, end)
	}
	ta.press("Esc")

	// Display width: a wide character is never cut in half.
	ta.Text(TextInput{Title: "Name"})
	ta.settle()
	covFType(ta, strings.Repeat("中文", 20))
	for _, k := range []string{"End", "Home"} {
		covFPress(ta, k)
		v = covFValue(ta)
		inner := strings.Trim(v, "… ")
		if Width(v) != 24-angleWidth() || strings.Trim(inner, "中文") != "" || Width(inner)%2 != 0 {
			t.Errorf("%s: %q", k, v)
		}
	}
	ta.press("Esc")

	// Nor is a red \n, wherever the cursor is.
	ta.Text(TextInput{Title: "Name"})
	ta.settle()
	covFPaste(ta, strings.Repeat("ab\n", 15))
	covFPress(ta, "Home")
	for range 46 {
		v = covFValue(ta)
		if Width(v) != 24-angleWidth() || strings.Count(v, `\`) != strings.Count(v, `\n`) {
			t.Fatalf("a \\n cut in half: %q", v)
		}
		covFPress(ta, "→")
	}
}

func TestInputEditKeys(t *testing.T) {
	ta := newTestApp(t)
	ta.Text(TextInput{Title: "Name"})
	ta.settle()
	li := covFLine(ta)
	steps := []struct {
		keys []string
		want string
	}{
		{[]string{"a", "c"}, "ac"},
		{[]string{"←", "b"}, "abc"},
		{[]string{"Home", "x"}, "xabc"},
		{[]string{"End", "y"}, "xabcy"},
		{[]string{"Home", "Delete"}, "abcy"},
		{[]string{"End", "Backspace"}, "abc"},
		{[]string{"←", "Space"}, "ab c"},
		{[]string{"→", "→", "z"}, "ab cz"},
	}
	for _, s := range steps {
		covFPress(ta, s.keys...)
		if li.ed.value() != s.want {
			t.Errorf("%v: %q, want %q", s.keys, li.ed.value(), s.want)
		}
	}
}

func TestInputGraphemes(t *testing.T) {
	ta := newTestApp(t)
	ta.Text(TextInput{Title: "Name"})
	ta.settle()
	li := covFLine(ta)
	// A character is one that looks like one: a CJK character, an emoji with
	// its skin tone, a letter with its combining mark.
	covFPress(ta, "中", "e\U00000301", "\U0001F44D\U0001F3FD", "x")
	if len(li.ed.g) != 4 {
		t.Fatalf("%q is %d characters, want 4", li.ed.value(), len(li.ed.g))
	}
	covFPress(ta, "←", "←", "Backspace")
	if li.ed.value() != "中\U0001F44D\U0001F3FDx" {
		t.Errorf("Backspace took one character: %q", li.ed.value())
	}
	covFPress(ta, "Delete")
	if li.ed.value() != "中x" {
		t.Errorf("Delete took one character: %q", li.ed.value())
	}
	// A pasted line break and tab are one each.
	covFPaste(ta, "a\tb\nc")
	if len(li.ed.g) != 7 {
		t.Errorf("after a paste: %d characters, want 7", len(li.ed.g))
	}

	t.Run("combining mark typed alone", func(t *testing.T) {
		ta := newTestApp(t)
		ta.Text(TextInput{Title: "Name"})
		ta.settle()
		covFPress(ta, "e", "\U00000301")
		if li := covFLine(ta); len(li.ed.g) != 1 {
			t.Errorf("e and its combining mark are %d characters", len(li.ed.g))
		}
	})
}

func TestInputCtrlAltIgnored(t *testing.T) {
	ta := newTestApp(t)
	ta.Text(TextInput{Title: "Name"})
	ta.settle()
	ta.press("a", "b", "c")
	covFPress(ta, "Ctrl-U", "Ctrl-W", "Ctrl-A", "Alt-b", "Alt-x")
	li := covFLine(ta)
	if li.ed.value() != "abc" || li.ed.pos != 3 || ta.top() == nil {
		t.Errorf("Ctrl- and Alt- keys did something: %q at %d", li.ed.value(), li.ed.pos)
	}
	ta.press("Esc")
	ta.Find(testFinder())
	ta.settle()
	ta.press("r", "e")
	covFPress(ta, "Ctrl-U", "Ctrl-W", "Alt-b")
	if fv := ta.topView().(*finderView); fv.ed.value() != "re" {
		t.Errorf("a finder's typing row: %q", fv.ed.value())
	}

	t.Run("number", func(t *testing.T) {
		ta := newTestApp(t)
		ta.Number(NumberInput{Title: "Port", Min: 1, Max: 65535})
		ta.settle()
		covFPress(ta, "Alt-1")
		if v := covFLine(ta).ed.value(); v != "" {
			t.Errorf("Alt-1 typed %q", v)
		}
	})
}

func TestInputSuggestion(t *testing.T) {
	ta := newTestApp(t)
	got, calls := "", 0
	in := TextInput{Title: "Rename", Verb: "rename", Old: "report-2025.pdf", Done: func(v string) tea.Cmd { got = v; calls++; return nil }}
	ta.Text(in)
	ta.settle()
	li := covFLine(ta)
	lines, _ := covFDraw(ta)
	if !li.ed.empty() || !strings.Contains(lines[4], Style{Fg: Base, Bg: Lavender}.Render("r")+Fg(Overlay0).Render("eport-2025.pdf")) {
		t.Fatalf("it opens empty, the old value grey from the cursor on, the cursor on its r: %q", lines[4])
	}
	if h := StripStyles(lines[8]); !strings.Contains(h, " Enter:rename Tab:edit Backspace:clear Esc:cancel ") {
		t.Errorf("hint: %s", h)
	}
	// Nothing touched: Enter changes nothing.
	ta.press("Enter")
	if ta.top() != nil || calls != 0 {
		t.Fatalf("Enter with nothing touched wrote %q", got)
	}
	// Typing replaces it; deleting what was typed brings it back.
	ta.Text(in)
	ta.settle()
	li = covFLine(ta)
	ta.press("x")
	if li.showingOld() {
		t.Error("typing did not hide the suggestion")
	}
	ta.press("Backspace")
	if !li.showingOld() {
		t.Error("deleting what was typed did not bring it back")
	}
	// Tab takes it in, the cursor at its end.
	ta.press("Tab")
	if li.ed.value() != "report-2025.pdf" || li.ed.pos != len(li.ed.g) {
		t.Errorf("Tab: %q at %d", li.ed.value(), li.ed.pos)
	}
	ta.press("Esc")
	// Backspace on the empty value drops it; Enter then saves the empty value.
	ta.Text(in)
	ta.settle()
	li = covFLine(ta)
	ta.press("Backspace")
	lines, _ = covFDraw(ta)
	if li.showingOld() || strings.Contains(StripStyles(lines[4]), "report") || strings.Contains(StripStyles(lines[8]), "Tab:edit") {
		t.Error("Backspace did not drop the suggestion")
	}
	ta.press("Enter")
	if calls != 1 || got != "" {
		t.Errorf("Enter after dropping it saves the empty value: %d calls, %q", calls, got)
	}
	// A number has the same suggestion; with no old value there is none and
	// Tab does nothing.
	ta.Number(NumberInput{Title: "Port", Old: "22"})
	ta.settle()
	ta.press("Tab")
	if v := covFLine(ta).ed.value(); v != "22" {
		t.Errorf("a number's suggestion: %q", v)
	}
	ta.press("Esc")
	ta.Text(TextInput{Title: "Name"})
	ta.settle()
	ta.press("Tab")
	if li := covFLine(ta); !li.ed.empty() {
		t.Errorf("Tab with no suggestion: %q", li.ed.value())
	}
}

func TestInputNoPlaceholder(t *testing.T) {
	ta := newTestApp(t)
	grey := sgrRGB(true, Overlay0.RGB)
	for name, open := range map[string]func(){
		"text":     func() { ta.Text(TextInput{Title: "Name", Prompt: "A name"}) },
		"number":   func() { ta.Number(NumberInput{Title: "Port", Prompt: "1-65535", Min: 1, Max: 65535}) },
		"password": func() { ta.Password(PasswordInput{Title: "Password", Prompt: "A password"}) },
	} {
		open()
		ta.settle()
		lines, _ := covFDraw(ta)
		if strings.TrimSpace(covFValue(ta)) != "" || strings.Contains(lines[4], grey) {
			t.Errorf("%s: the value row holds %q", name, StripStyles(lines[4]))
		}
		ta.press("Esc")
	}
	// The angle before a finder's typing row is a glyph, not a placeholder:
	// Tab does not take it in, typing does not hide it.
	ta.Find(testFinder())
	ta.settle()
	fv := ta.topView().(*finderView)
	ta.press("Tab", "Tab", "r")
	lines, _ := covFDraw(ta)
	if fv.ed.value() != "r" || !strings.HasPrefix(StripStyles(lines[2]), "┃ "+glyphTyping+" r") {
		t.Errorf("typing row %q, value %q", StripStyles(lines[2]), fv.ed.value())
	}
}

func TestPasteMarkers(t *testing.T) {
	ta := newTestApp(t)
	ta.Text(TextInput{Title: "Name"})
	ta.settle()
	covFPaste(ta, "a\r\nb\tc")
	li := covFLine(ta)
	if !slices.Equal(li.ed.g, []string{"a", "\r", "\n", "b", "\t", "c"}) {
		t.Fatalf("pasted: %q (\\r\\n is two)", li.ed.g)
	}
	lines, _ := covFDraw(ta)
	if !strings.Contains(StripStyles(lines[4]), `a\r\nb\tc`) || !strings.Contains(lines[4], Fg(Red).Render(`\r`)) ||
		!strings.Contains(lines[4], Fg(Red).Render(`\n`)) || !strings.Contains(lines[4], Fg(Red).Render(`\t`)) {
		t.Errorf("drawn as red \\r, \\n and \\t: %q", lines[4])
	}
	ta.press("Backspace", "Backspace")
	if li.ed.value() != "a\r\nb" {
		t.Errorf("Backspace takes the whole \\t: %q", li.ed.value())
	}
	ta.press("Esc")
	// A masked value stays masked.
	ta.Password(PasswordInput{Title: "Password"})
	ta.settle()
	covFPaste(ta, "p\nw")
	if v := covFValue(ta); !strings.HasPrefix(v, "•••") || strings.Contains(v, `\n`) {
		t.Errorf("password: %q", v)
	}
	ta.press("Esc")

	t.Run("grey typing row", func(t *testing.T) {
		ta := newTestApp(t)
		ta.Find(testFinder())
		ta.settle()
		covFPaste(ta, "re\nad")
		ta.press("Tab")
		lines, _ := covFDraw(ta)
		if !strings.Contains(lines[2], Fg(Overlay0).Render(`\n`)) || strings.Contains(lines[2], sgrRGB(true, Red.RGB)) {
			t.Errorf("typing row on the list: %q", lines[2])
		}
	})
	t.Run("grey suggestion", func(t *testing.T) {
		ta := newTestApp(t)
		ta.Text(TextInput{Title: "Name", Old: "x\ny"})
		ta.settle()
		lines, _ := covFDraw(ta)
		if Width(lines[4]) != ta.top().w || !strings.Contains(lines[4], Fg(Overlay0).Render(`\n`)) {
			t.Errorf("suggestion row: %q", lines[4])
		}
	})
}

// Every pasted control character stays, drawn in red: \r, \n and \t by
// name, the rest as \xNN, never cut apart; Backspace takes one whole
// (components/input/README).
func TestPasteControlChars(t *testing.T) {
	ta := newTestApp(t)
	ta.Text(TextInput{Title: "Name"})
	ta.settle()
	pasted := "a\x01b\x7fc\U00000085d\U0000009be\x1bf\rg"
	covFPaste(ta, pasted)
	li := covFLine(ta)
	if v := li.ed.value(); v != pasted {
		t.Errorf("C0, DEL and C1 are kept: %q", v)
	}
	lines, _ := covFDraw(ta)
	if v := covFValue(ta); !strings.HasPrefix(v, `a\x01b\x7fc\x85d\x9be\x1bf\rg`) {
		t.Errorf("drawn as: %q", v)
	}
	for _, name := range []string{`\x01`, `\x7f`, `\x85`, `\x9b`, `\x1b`, `\r`} {
		if !strings.Contains(lines[4], Fg(Red).Render(name)) {
			t.Errorf("%s is not red: %q", name, lines[4])
		}
	}
	ta.press("Backspace", "Backspace", "Backspace", "Backspace")
	if v := li.ed.value(); v != "a\x01b\x7fc\U00000085d\U0000009be" {
		t.Errorf("Backspace takes \\r and \\x1b whole: %q", v)
	}

	// Never cut in half, wherever the cursor is.
	ta.press("Esc")
	ta.resize(30, 20)
	ta.Text(TextInput{Title: "Name"})
	ta.settle()
	covFPaste(ta, strings.Repeat("ab\x1b", 12))
	covFPress(ta, "Home")
	for range 37 {
		v := covFValue(ta)
		if Width(v) != 24-angleWidth() || strings.Count(v, `\`) != strings.Count(v, `\x1b`) {
			t.Fatalf("a \\x1b cut in half: %q", v)
		}
		covFPress(ta, "→")
	}
}

func TestPasteBlockedOnEnter(t *testing.T) {
	ta := newTestApp(t)
	got := "unset"
	done := func(v string) tea.Cmd { got = v; return nil }
	ta.Text(TextInput{Title: "Name", Done: done})
	ta.settle()
	covFPaste(ta, "web\n01")
	ta.press("Enter")
	lines, _ := covFDraw(ta)
	if ta.top() == nil || got != "unset" || !strings.Contains(lines[6], Fg(Red).Render("Name can't have control characters")) {
		t.Fatalf("a used value with a line break is stopped on Enter: got %q, error row %q", got, StripStyles(lines[6]))
	}
	ta.press("Esc")
	// Any control character: a lone \r too, before the trim would take it.
	ta.Text(TextInput{Title: "Name", Done: done})
	ta.settle()
	ta.press("I", "c", "o", "n")
	covFPaste(ta, "\r")
	ta.press("Enter")
	if ta.top() == nil || got != "unset" {
		t.Errorf("Icon\\r was written: %q", got)
	}
	ta.press("Esc")
	// A password too, masked.
	pw := "unset"
	ta.Password(PasswordInput{Title: "Password", Done: func(v string, _ bool) tea.Cmd { pw = v; return nil }})
	ta.settle()
	covFPaste(ta, "secret\n")
	ta.press("Enter")
	lines, _ = covFDraw(ta)
	if ta.top() == nil || pw != "unset" || !strings.Contains(lines[6], Fg(Red).Render("Password can't have control characters")) {
		t.Errorf("a password with a line break: got %q, error row %q", pw, StripStyles(lines[6]))
	}
	ta.press("Esc")
	// Checked before the white space is trimmed: a trailing tab is caught.
	ta.Text(TextInput{Title: "Name", Done: done})
	ta.settle()
	ta.press("I", "c", "o", "n")
	covFPaste(ta, "\t")
	ta.press("Enter")
	if ta.top() == nil || got != "unset" {
		t.Errorf("Icon\\t was trimmed before the check: got %q", got)
	}
	ta.press("Esc")
	// A value not typed (the old one, taken in with Tab) goes through the
	// same filter and the same check.
	ta.Text(TextInput{Title: "Name", Old: "a\x01b", Done: done})
	ta.settle()
	ta.press("Tab")
	if v := covFValue(ta); !strings.HasPrefix(v, `a\x01b`) {
		t.Errorf("the old value was not filtered: %q", v)
	}
	ta.press("Esc")
	ta.Text(TextInput{Title: "Name", Old: "x\ty", Done: done})
	ta.settle()
	ta.press("Tab", "Enter")
	if ta.top() == nil || got != "unset" {
		t.Errorf("an old value with a tab was written: %q", got)
	}
}

func TestPasteSearchNotBlocked(t *testing.T) {
	ta := newTestApp(t)
	ta.Find(testFinder())
	ta.settle()
	fv := ta.topView().(*finderView)
	covFPaste(ta, "read\tme")
	lines, _ := covFDraw(ta)
	if fv.ed.value() != "read\tme" || !strings.Contains(lines[2], Fg(Red).Render(`\t`)) {
		t.Errorf("the finder's typing row draws the tab: %q", StripStyles(lines[2]))
	}
	ta.press("Enter")
	if !fv.onList || ta.topView() != popupView(fv) {
		t.Error("Enter on a search with a tab was stopped")
	}
	ta.press("Esc")
	// The panel filter too.
	ta.press("/")
	covFPaste(ta, "pod-1\n")
	if !ta.list.Typing() || len(ta.list.view) != 10 || ta.top() != nil {
		t.Errorf("panel filter: typing %v, %d rows", ta.list.Typing(), len(ta.list.view))
	}
	if !strings.Contains(ta.screenText(), `pod-1\n`) {
		t.Error("the panel filter does not draw the \\n")
	}
	ta.press("Enter")
	if ta.list.Typing() {
		t.Error("Enter on the panel filter was stopped")
	}
}

func TestInputTrimWhitespace(t *testing.T) {
	ta := newTestApp(t)
	got := ""
	ta.Text(TextInput{Title: "Host", Done: func(v string) tea.Cmd { got = v; return nil }})
	ta.settle()
	covFPress(ta, "\U00003000", "w", "e", "b", "Space")
	covFPaste(ta, "\U000000A0")
	ta.press("Enter")
	if got != "web" {
		t.Errorf("every Unicode white space is trimmed: %q", got)
	}
	pw := "unset"
	ta.Password(PasswordInput{Title: "Password", Done: func(v string, _ bool) tea.Cmd { pw = v; return nil }})
	ta.settle()
	ta.press("Space", "p", "w", "Space", "Enter")
	if pw != " pw " {
		t.Errorf("a password is not trimmed: %q", pw)
	}
}

func TestValueDisplay(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}
	value := func(s string) func() string { return func() string { return s } }
	long := strings.Repeat("long ", 20)
	deep := "/very/long/path/to/some/where/deep/down/.ssh/id_ed25519"
	bg := Hex("#2A2A3C")
	check := func(where string, lines []string) {
		t.Helper()
		if r := covFRow(lines, "web-01"); !strings.Contains(r, Fg(Text).Render("web-01")) {
			t.Errorf("%s: a value is Text: %q", where, r)
		}
		if r := covFRow(lines, "not set"); !strings.Contains(r, Fg(Overlay0).Render("not set")) {
			t.Errorf("%s: an empty value says what it means in grey: %q", where, r)
		}
		if r := StripStyles(covFRow(lines, "long long")); !strings.HasSuffix(strings.TrimRight(r, " ┃"), "…") || strings.Count(r, "long") == 20 {
			t.Errorf("%s: a long value is cut at its end: %q", where, r)
		}
		if r := StripStyles(covFRow(lines, "id_ed25519")); !strings.Contains(r, "…") || !strings.Contains(r, "/.ssh/id_ed25519") {
			t.Errorf("%s: a long path is cut at its front: %q", where, r)
		}
		if r := StripStyles(covFRow(lines, "Downloads")); !strings.Contains(r, "~/Downloads") {
			t.Errorf("%s: the home directory is ~: %q", where, r)
		}
		if r := covFRow(lines, "#2A2A3C"); !strings.Contains(r, Fg(bg).Render("███")+" "+Fg(Text).Render("#2A2A3C")) {
			t.Errorf("%s: a colour is a swatch and its hex: %q", where, r)
		}
	}
	v := &Values{Fields: []*Field{
		{Label: "Cursor", Value: value("here")},
		{Label: "Name", Value: value("web-01")},
		{Label: "Nickname", Value: value(""), Empty: "not set"},
		{Label: "Note", Value: value(long)},
		{Label: "Key", Path: true, Value: value(deep)},
		{Label: "Save to", Path: true, Value: value(filepath.Join(home, "Downloads"))},
		{Label: "Background", Value: value(ColorText(bg))},
	}}
	check("panel", v.Draw(50, 9, true))

	ta := newTestApp(t)
	ta.resize(56, 30)
	ta.Form(&Form{Title: "Settings", Fields: []*FormField{
		{Label: "Cursor", Value: value("here")},
		{Label: "Name", Value: value("web-01")},
		{Label: "Nickname", Value: value(""), Empty: "not set"},
		{Label: "Note", Value: value(long)},
		{Label: "Key", Path: true, Value: value(deep)},
		{Label: "Save to", Path: true, Value: value(filepath.Join(home, "Downloads"))},
		{Label: "Background", Value: value(ColorText(bg))},
	}})
	ta.settle()
	lines, _ := covFDraw(ta)
	check("form", lines)

	t.Run("empty path on a form", func(t *testing.T) {
		ta := newTestApp(t)
		ta.Form(&Form{Title: "Settings", Fields: []*FormField{
			{Label: "Cursor", Value: value("here")},
			{Label: "Key", Path: true, Value: value(""), Empty: "ssh decides"},
		}})
		ta.settle()
		lines, _ := covFDraw(ta)
		if r := covFRow(lines, "ssh decides"); !strings.Contains(r, Fg(Overlay0).Render("ssh decides")) {
			t.Errorf("empty path: %q", r)
		}
	})
}

// --- input — 各種 --------------------------------------------------------------

func TestTextInput(t *testing.T) {
	ta := newTestApp(t)
	got := ""
	ta.Text(TextInput{Title: "Rename", Prompt: "New name for assets", Verb: "rename", Old: "assets",
		Done: func(v string) tea.Cmd { got = v; return nil }})
	ta.settle()
	// The one-line skeleton with the old value as the suggestion; Tab takes
	// it, the editing keys change it, Enter trims and writes it.
	text := covFScreen(ta)
	if !strings.HasPrefix(text[2], "┃ New name for assets") || !strings.Contains(text[4], "assets") {
		t.Errorf("skeleton:\n%s", strings.Join(text, "\n"))
	}
	ta.press("Tab")
	covFPress(ta, "Home", "Delete", "A", "End", "Space", "2", "Space")
	ta.press("Enter")
	if got != "Assets 2" || ta.top() != nil {
		t.Errorf("wrote %q", got)
	}
}

func TestPasswordMasks(t *testing.T) {
	ta := newTestApp(t)
	// The ordinary look: a • per character from the left, in Lavender.
	ta.Password(PasswordInput{Title: "Password"})
	ta.settle()
	ta.press("a", "b", "c")
	lines, _ := covFDraw(ta)
	if v := covFValue(ta); !strings.HasPrefix(v, "••• ") || strings.Contains(v, "abc") || !strings.Contains(lines[4], Fg(Lavender).Render("•")) {
		t.Errorf("ordinary: %q", v)
	}
	ta.press("Esc")
	ta.Password(PasswordInput{Title: "Password", Set: true})
	ta.settle()
	if lines, _ := covFDraw(ta); !strings.Contains(lines[4], Style{Fg: Base, Bg: Lavender}.Render("•")+Fg(Overlay0).Render("•••••••")) {
		t.Errorf("ordinary, set: eight •, the cursor on the first: %q", StripStyles(lines[4]))
	}
	ta.press("Esc")

	// The unlock look: a ● per character a cell apart, centred, growing both ways.
	ta.Password(PasswordInput{Title: "PIN", Unlock: true, MaxLen: 12})
	ta.settle()
	pads := func() (int, int) {
		v := covFValue(ta)
		return len(v) - len(strings.TrimLeft(v, " ")), len(v) - len(strings.TrimRight(v, " "))
	}
	ta.press("1", "2", "3")
	l3, r3 := pads()
	lines, _ = covFDraw(ta)
	// The cursor after them counts: a space and a block, blank without styles.
	if v := covFValue(ta); strings.TrimSpace(v) != "● ● ●" || l3-(r3-2) > 1 || (r3-2)-l3 > 1 || !strings.Contains(lines[4], Fg(Lavender).Render("● ● ● ")) {
		t.Errorf("unlock: %q", v)
	}
	ta.press("4")
	if l4, r4 := pads(); l4 >= l3 || r4 >= r3 {
		t.Errorf("unlock grows both ways: %d %d, was %d %d", l4, r4, l3, r3)
	}
	ta.press("Esc")
	ta.Password(PasswordInput{Title: "PIN", Unlock: true, MaxLen: 12, Set: true})
	ta.settle()
	if lines, _ := covFDraw(ta); !strings.Contains(lines[4], Style{Fg: Base, Bg: Lavender}.Render("●")+Fg(Overlay0).Render(" ● ● ● ● ● ● ●")) {
		t.Errorf("unlock, set: eight ● a cell apart, the cursor on the first: %q", StripStyles(lines[4]))
	}
	ta.press("Esc")

	// Too long for the box: as "the value is longer than the box".
	ta.resize(30, 20)
	ta.Password(PasswordInput{Title: "Password"})
	ta.settle()
	covFType(ta, strings.Repeat("x", 40))
	if v := covFValue(ta); !strings.HasPrefix(v, "…") || !strings.HasSuffix(v, "• ") {
		t.Errorf("a long password: the front is cut, the cursor shows: %q", v)
	}

	t.Run("unlock too long", func(t *testing.T) {
		ta := newTestApp(t)
		ta.resize(30, 20)
		ta.Password(PasswordInput{Title: "PIN", Unlock: true})
		ta.settle()
		covFType(ta, strings.Repeat("1", 30))
		if v := covFValue(ta); !strings.HasPrefix(v, "…") {
			t.Errorf("unlock, too long: %q", v)
		}
	})
}

// The unlock look has a cursor where the next ● goes; once all the digits
// are typed it is on the last ●, no cell more; centring counts it
// (components/input/password).
func TestUnlockCursor(t *testing.T) {
	ta := newTestApp(t)
	ta.Password(PasswordInput{Title: "PIN", Prompt: "Up to 4 digits", Unlock: true, MaxLen: 4})
	ta.settle()
	block := Style{Fg: Base, Bg: Lavender}
	row := func() (string, string) {
		lines, _ := covFDraw(ta)
		return lines[4], covFValue(ta)
	}
	if styled, v := row(); !strings.Contains(styled, block.Render(" ")) || strings.TrimSpace(v) != "" {
		t.Errorf("nothing typed: the cursor alone: %q", styled)
	}
	ta.press("1", "2", "3")
	styled, v := row()
	l, r := len(v)-len(strings.TrimLeft(v, " ")), len(v)-len(strings.TrimRight(v, " "))
	// 2n + 1 cells: the dots, a space, the cursor.
	if !strings.Contains(styled, Fg(Lavender).Render("● ● ● ")+block.Render(" ")) || l-(r-2) > 1 || (r-2)-l > 1 {
		t.Errorf("three typed: %q", styled)
	}
	if strings.Contains(styled, glyphTyping) {
		t.Errorf("the unlock look has no angle: %q", StripStyles(styled))
	}
	ta.press("4")
	styled, v = row()
	l, r = len(v)-len(strings.TrimLeft(v, " ")), len(v)-len(strings.TrimRight(v, " "))
	// 2N - 1 cells: the cursor is on the last ●.
	if !strings.Contains(styled, Fg(Lavender).Render("● ● ● ")+block.Render("●")) || strings.TrimSpace(v) != "● ● ● ●" || l-r > 1 || r-l > 1 {
		t.Errorf("all four typed: %q", styled)
	}
	ta.press("5")
	if _, v2 := row(); v2 != v {
		t.Errorf("a fifth digit: %q", v2)
	}
	ta.press("Esc")

	// A PIN of 12: 23 cells full, the box 43 wide for its hint (layout/popup).
	ta.Password(PasswordInput{Title: "PIN", Prompt: "Up to 12 digits", Unlock: true, MaxLen: 12})
	ta.settle()
	covFType(ta, strings.Repeat("7", 12))
	if _, v := row(); strings.TrimSpace(v) != strings.TrimSpace(strings.Repeat("● ", 12)) || ta.top().w != 43 {
		t.Errorf("a PIN of 12, full: %q in %d", v, ta.top().w)
	}
}

func TestPasswordUnlockWidth(t *testing.T) {
	ta := newTestApp(t)
	ta.Password(PasswordInput{Title: "PIN", Prompt: "Up to 12 digits", Unlock: true, MaxLen: 12})
	ta.settle()
	// 12 digits are 23 cells, the hint "Enter:save Backspace:clear
	// Esc:cancel" 37 + 6: 43 (layout/popup, width), from the first digit on.
	if w := ta.top().w; w != 43 {
		t.Errorf("the PIN box is %d wide, want 43", w)
	}
	covFType(ta, "1234567890123")
	if n := len(covFLine(ta).ed.g); n != 12 {
		t.Errorf("%d digits, the most is 12", n)
	}
	if v := covFValue(ta); strings.TrimSpace(v) != strings.TrimSpace(strings.Repeat("● ", 12)) || ta.top().w != 43 {
		t.Errorf("twelve digits fit: %q, %d wide", v, ta.top().w)
	}
}

func TestPasswordExisting(t *testing.T) {
	ta := newTestApp(t)
	calls := 0
	got := ""
	in := PasswordInput{Title: "Password", Set: true, Done: func(v string, _ bool) tea.Cmd { got = v; calls++; return nil }}
	ta.Password(in)
	ta.settle()
	li := covFLine(ta)
	lines, _ := covFDraw(ta)
	if !li.ed.empty() || !strings.Contains(lines[4], Style{Fg: Base, Bg: Lavender}.Render("•")+Fg(Overlay0).Render("•••••••")) {
		t.Fatalf("it opens empty with eight dots, the cursor on the first: %q", StripStyles(lines[4]))
	}
	if h := StripStyles(lines[8]); strings.Contains(h, "Tab") {
		t.Errorf("no Tab in the hint: %s", h)
	}
	ta.press("Tab")
	if !li.ed.empty() || !li.showingOld() {
		t.Error("Tab took something in")
	}
	for _, k := range []string{"Enter", "Esc"} {
		ta.press(k)
		if ta.top() != nil || calls != 0 {
			t.Errorf("%s with nothing typed changed the password", k)
		}
		ta.Password(in)
		ta.settle()
	}
	ta.press("n", "e", "w")
	if v := covFValue(ta); strings.Contains(v, "••••••••") || !strings.HasPrefix(v, "••• ") {
		t.Errorf("typing replaces the dots: %q", v)
	}
	ta.press("Enter")
	if calls != 1 || got != "new" {
		t.Errorf("wrote %q (%d calls)", got, calls)
	}
}

func TestPasswordBackspaceClears(t *testing.T) {
	ta := newTestApp(t)
	got, cleared, calls := "", false, 0
	ta.Password(PasswordInput{Title: "Password", Set: true, Done: func(v string, c bool) tea.Cmd { got, cleared = v, c; calls++; return nil }})
	ta.settle()
	li := covFLine(ta)
	if h := covFHint(ta); !strings.Contains(h, " Enter:save Backspace:clear Esc:cancel ") {
		t.Errorf("hint: %s", h)
	}
	ta.press("a", "b", "c", "Backspace")
	if !li.ed.empty() || !li.showingOld() {
		t.Errorf("Backspace clears all that was typed, back to the dots: %q", li.ed.value())
	}
	ta.press("Backspace")
	if li.showingOld() {
		t.Error("Backspace on the empty value did not drop the old password")
	}
	ta.press("Enter")
	if calls != 1 || got != "" || !cleared {
		t.Errorf("Enter then saves no password: %q, cleared %v, %d calls", got, cleared, calls)
	}
}

func TestPasswordAppendOnly(t *testing.T) {
	ta := newTestApp(t)
	ta.Password(PasswordInput{Title: "Password"})
	ta.settle()
	li := covFLine(ta)
	ta.press("a", "b")
	covFPress(ta, "←", "Home", "c", "Delete", "→", "End", "←", "d")
	if li.ed.value() != "abcd" || li.ed.pos != 4 {
		t.Errorf("a password only grows at its end: %q at %d", li.ed.value(), li.ed.pos)
	}
}

func TestPasswordDisplay(t *testing.T) {
	ta := newTestApp(t)
	pw := "unset"
	ta.Password(PasswordInput{Title: "Password", Done: func(v string, _ bool) tea.Cmd { pw = v; return nil }})
	ta.settle()
	covFPress(ta, "Space", "x", "\U00003000")
	ta.press("Enter")
	if pw != " x\U00003000" {
		t.Errorf("a password keeps its white space: %q", pw)
	}
	// On a panel: what the app shows for a set password is drawn in Text, an
	// unset one by the empty-value rule.
	v := &Values{Fields: []*Field{
		{Label: "Cursor", Value: func() string { return "here" }},
		{Label: "Password", Value: func() string { return "••••••••" }},
		{Label: "PIN", Value: func() string { return "" }, Empty: "not set"},
	}}
	lines := v.Draw(40, 4, true)
	if !strings.Contains(lines[1], Fg(Text).Render("••••••••")) || !strings.Contains(lines[2], Fg(Overlay0).Render("not set")) {
		t.Errorf("panel:\n%s", strings.Join(covFText(lines), "\n"))
	}
}

func TestNumberFilter(t *testing.T) {
	ta := newTestApp(t)
	ta.Number(NumberInput{Title: "Port", Prompt: "1-65535", Min: 1, Max: 65535})
	ta.settle()
	li := covFLine(ta)
	// The range is known: the width is known when it opens.
	if w := ta.top().w; w >= 118 {
		t.Errorf("a number with a range is %d wide", w)
	}
	ta.press("2", "a", "-", ".", "2")
	if li.ed.value() != "22" {
		t.Errorf("only digits are typed: %q", li.ed.value())
	}
	covFPaste(ta, "8 0-8.x0")
	if li.ed.value() != "228080" {
		t.Errorf("a paste keeps only what is taken: %q", li.ed.value())
	}
	ta.press("↑", "↓")
	if li.ed.value() != "228080" {
		t.Errorf("↑ and ↓ do not count: %q", li.ed.value())
	}
	ta.press("Esc")
	ta.Number(NumberInput{Title: "Ratio", Min: -1, Max: 1, Negative: true, Decimal: true})
	ta.settle()
	ta.press("-", "0", ".", "5", "x")
	if v := covFLine(ta).ed.value(); v != "-0.5" {
		t.Errorf("negative and decimal: %q", v)
	}
	ta.press("Esc")
	ta.Number(NumberInput{Title: "Count"})
	ta.settle()
	if w := ta.top().w; w != 118 {
		t.Errorf("a number with no range is %d wide, want F7's most", w)
	}

	t.Run("space", func(t *testing.T) {
		ta := newTestApp(t)
		ta.Number(NumberInput{Title: "Port", Min: 1, Max: 65535})
		ta.settle()
		ta.press("2", "Space", "2")
		if v := covFLine(ta).ed.value(); v != "22" {
			t.Errorf("Space typed: %q", v)
		}
	})
}

func TestNumberRange(t *testing.T) {
	ta := newTestApp(t)
	got := "unset"
	ta.Number(NumberInput{Title: "Port", Min: 1, Max: 65535, Done: func(v string) tea.Cmd { got = v; return nil }})
	ta.settle()
	li := covFLine(ta)
	// Out of range while typing is fine: the range is checked on Enter.
	ta.press("0")
	if li.err != "" || li.ed.value() != "0" {
		t.Fatalf("typing 0 was stopped: %q %q", li.ed.value(), li.err)
	}
	ta.press("Enter")
	lines, _ := covFDraw(ta)
	if ta.top() == nil || got != "unset" || !strings.Contains(lines[6], Fg(Red).Render("Port must be 1-65535")) {
		t.Fatalf("0 on Enter: got %q, error row %q", got, StripStyles(lines[6]))
	}
	ta.press("Backspace", "7", "0", "0", "0", "0", "Enter")
	if got != "unset" || li.err != "Port must be 1-65535" {
		t.Errorf("70000: got %q, error %q", got, li.err)
	}
	ta.press("Backspace", "Backspace", "Backspace", "Backspace", "2", "Enter")
	if got != "72" || ta.top() != nil {
		t.Errorf("72: got %q", got)
	}
}

func TestTextareaWriteKeys(t *testing.T) {
	ta := newTestApp(t)
	ta.Textarea(Textarea{Title: "Comment", Old: "abc\ndef"})
	ta.settle()
	tv := ta.topView().(*textareaView)
	at := func(line, col int, what string) {
		t.Helper()
		if tv.line != line || tv.col != col {
			t.Errorf("%s: at %d:%d, want %d:%d", what, tv.line, tv.col, line, col)
		}
	}
	covFPress(ta, "←", "←", "←", "←")
	at(0, 3, "← at a line's start goes to the end of the line before")
	covFPress(ta, "→")
	at(1, 0, "→ at a line's end goes to the start of the next")
	covFPress(ta, "Backspace")
	at(0, 3, "Backspace at a line's start joins it to the line before")
	if tv.value() != "abcdef" {
		t.Errorf("joined: %q", tv.value())
	}
	covFPress(ta, "End")
	at(0, 6, "End")
	covFPress(ta, "Home", "Delete")
	if tv.value() != "bcdef" {
		t.Errorf("Delete: %q", tv.value())
	}
	covFPaste(ta, "1\n2")
	if tv.value() != "1\n2bcdef" {
		t.Errorf("a pasted line break is a line break: %q", tv.value())
	}
	ta.press("Esc", "Esc", "Enter") // discard

	// ↑ and ↓ go by the rows drawn: a wrapped line is two rows.
	ta.resize(40, 20)
	ta.Textarea(Textarea{Title: "Comment", Old: strings.Repeat("y", 50)})
	ta.settle()
	tv = ta.topView().(*textareaView)
	w := tv.textW
	covFPress(ta, "↑")
	at(0, 50-w, "↑ to the row above in the same line")
	covFPress(ta, "↓")
	at(0, 50, "↓ back")
}

func TestTextareaMoveKeys(t *testing.T) {
	ta := newTestApp(t)
	ta.Textarea(Textarea{Title: "Comment", Old: "one two\nthree four\nfive"})
	ta.settle()
	tv := ta.topView().(*textareaView)
	ta.press("Esc")
	if tv.typing() {
		t.Fatal("Esc did not move to the moving state")
	}
	steps := []struct {
		key       string
		line, col int
	}{
		{"gg", 0, 0}, {"w", 0, 4}, {"e", 0, 6}, {"b", 0, 4}, {"$", 0, 6}, {"0", 0, 0},
		{"j", 1, 0}, {"l", 1, 1}, {"h", 1, 0}, {"G", 2, 0}, {"k", 1, 0},
	}
	for _, s := range steps {
		if s.key == "gg" {
			ta.press("g", "g")
		} else {
			ta.press(s.key)
		}
		if tv.line != s.line || tv.col != s.col {
			t.Errorf("%s: at %d:%d, want %d:%d", s.key, tv.line, tv.col, s.line, s.col)
		}
	}
	// Moving is neither a mode nor the input state: ? opens the key reference.
	ta.press("?")
	if n, ok := ta.topView().(*noteView); !ok || n.title != "COMMENT KEYS" {
		t.Fatalf("? opens the key reference: %T", ta.topView())
	}
	ta.press("Esc")
	if ta.topView() != popupView(tv) || tv.typing() {
		t.Error("Esc on the key reference goes back to moving")
	}
	ta.press("Esc", "Enter")

	// u and d go half a page.
	ta.Textarea(Textarea{Title: "Comment", Old: strings.Repeat("x\n", 39) + "x"})
	ta.settle()
	tv = ta.topView().(*textareaView)
	ta.Frame() // the page is what the last frame showed
	ta.press("Esc", "g", "g", "d")
	half := tv.visible / 2
	if tv.line != half {
		t.Errorf("d went to line %d, want %d", tv.line, half)
	}
	ta.press("u")
	if tv.line != 0 {
		t.Errorf("u went to line %d", tv.line)
	}
}

func TestFinderLayout(t *testing.T) {
	ta := newTestApp(t)
	ta.Find(testFinder())
	ta.settle()
	lines, layer := covFDraw(ta)
	text := covFText(lines)
	// The typing row: the angle one cell from the frame, a space, what is
	// typed; the count on the right in Overlay0.
	if !strings.HasPrefix(text[2], "┃ "+glyphTyping+" ") || !strings.Contains(text[2], "5/5 ┃") {
		t.Errorf("typing row %q", text[2])
	}
	if !strings.Contains(lines[2], Fg(layer).Render(glyphTyping)) || !strings.Contains(lines[2], Fg(Overlay0).Render("5/5")) {
		t.Error("the angle is the layer colour while typing, the count Overlay0")
	}
	// The separator, one cell from the frame on each side.
	if !strings.HasPrefix(text[3], "┃ ─") || !strings.Contains(text[3], "─ ┃") {
		t.Errorf("separator %q", text[3])
	}
	// The angle is in line with the list's text.
	if strings.Index(text[2], glyphTyping) != strings.Index(text[5], "README-zh_TW.md") {
		t.Errorf("angle and list text not aligned:\n%s\n%s", text[2], text[5])
	}
	// The cursor goes from frame to frame: no gap after ┃, the arrow before ┃.
	ta.press("Tab")
	lines, _ = covFDraw(ta)
	cur := lines[4]
	if !strings.HasPrefix(cur, Fg(layer).Render("┃")+covFOpen(On(layer))+" README.md ") ||
		!strings.Contains(cur, Fg(layer).Render(glyphHard)+Fg(layer).Render("┃")) {
		t.Errorf("cursor row %q", cur)
	}
	if !strings.Contains(lines[2], Fg(Overlay0).Render(glyphTyping)) {
		t.Error("on the list the angle is Overlay0")
	}
}

func TestFinderMarkers(t *testing.T) {
	ta := newTestApp(t)
	ta.Select(Select{Title: "Profile", Options: []string{"clock", "matrix", "pipes"}, Selected: 1})
	ta.settle()
	lines, layer := covFDraw(ta)
	// The cursor on the value is drawn as the cursor.
	if r := covFRow(lines, "matrix"); !strings.Contains(r, covFOpen(On(layer))+" "+glyphRadioOn+" matrix ") || strings.Contains(r, sgrRGB(true, Green.RGB)) {
		t.Errorf("cursor on the value: %q", r)
	}
	ta.press("j")
	lines, _ = covFDraw(ta)
	if r := covFRow(lines, "matrix"); !strings.Contains(r, Fg(Green).Render(glyphRadioOn)+" "+Fg(Green).Render("matrix")) {
		t.Errorf("the value is Green with a marked radio: %q", r)
	}
	if r := covFRow(lines, "clock"); !strings.Contains(r, Fg(Text).Render(glyphRadioOff)+" "+Fg(Text).Render("clock")) {
		t.Errorf("another option: %q", r)
	}
	ta.press("Esc")

	ta.Checks(Checks{Title: "Namespaces", Options: []string{"default", "kube-system", "monitoring"}, Checked: []bool{true, false, false}})
	ta.settle()
	ta.press("j", "j")
	lines, _ = covFDraw(ta)
	if r := covFRow(lines, "default"); !strings.Contains(r, Fg(Green).Render(glyphMarked)+" "+Fg(Green).Render("default")) {
		t.Errorf("a checked row: %q", r)
	}
	if r := covFRow(lines, "kube-system"); !strings.Contains(r, Fg(Text).Render(glyphUnchecked)+" "+Fg(Text).Render("kube-system")) {
		t.Errorf("an unchecked row: %q", r)
	}
	// Green follows the check on screen, before Enter.
	ta.press("k", "Space", "j")
	lines, _ = covFDraw(ta)
	if r := covFRow(lines, "kube-system"); !strings.Contains(r, Fg(Green).Render(glyphMarked)+" "+Fg(Green).Render("kube-system")) {
		t.Errorf("checked with Space: %q", r)
	}
}

func TestFinderFocusBrightness(t *testing.T) {
	ta := newTestApp(t)
	ta.Find(testFinder())
	ta.settle()
	lines, layer := covFDraw(ta)
	bright, dim := sgrRGB(false, layer.RGB), sgrRGB(false, DimRGB(layer.RGB))
	lavender := sgrRGB(false, Lavender.RGB)
	// Typing: the typing row lit with its cursor, the list's cursor dimmed once.
	if !strings.Contains(lines[2], lavender) || !strings.Contains(lines[2], Fg(layer).Render(glyphTyping)) {
		t.Errorf("typing row while typing: %q", lines[2])
	}
	if !strings.Contains(lines[4], dim) || strings.Contains(lines[4], bright) {
		t.Errorf("the list's cursor while typing: %q", lines[4])
	}
	// On the list: the typing row all grey with no cursor, the list's cursor lit.
	ta.press("Tab")
	lines, _ = covFDraw(ta)
	if strings.Contains(lines[2], lavender) || strings.Contains(lines[2], Fg(layer).Render(glyphTyping)) || !strings.Contains(lines[2], Fg(Overlay0).Render(glyphTyping)) {
		t.Errorf("typing row on the list: %q", lines[2])
	}
	if !strings.Contains(lines[4], bright) {
		t.Errorf("the list's cursor on the list: %q", lines[4])
	}
}

func TestFinderTypingKeys(t *testing.T) {
	ta := newTestApp(t)
	ta.Find(testFinder())
	ta.settle()
	fv := ta.topView().(*finderView)
	ta.press("↑")
	if fv.cursor != 4 {
		t.Errorf("↑ on the first wraps to the last: %d", fv.cursor)
	}
	ta.press("↓")
	if fv.cursor != 0 {
		t.Errorf("↓ on the last wraps to the first: %d", fv.cursor)
	}
	ta.press("g", "o")
	if len(fv.shown) != 2 || fv.onList {
		t.Errorf("go matches %d", len(fv.shown))
	}
	ta.press("↓", "Enter")
	if _, r, _ := fv.at(); !fv.onList || r.text != "go.mod" || ta.top() == nil {
		t.Errorf("Enter goes to the row under the cursor without opening it: %q", r.text)
	}
	ta.press("Tab", "Esc")
	if ta.top() != nil {
		t.Error("Esc on the typing row closes the whole finder")
	}
	// A panel filter's Esc clears it.
	ta.press("/", "1")
	ta.press("Esc")
	if ta.list.Filtering() {
		t.Error("Esc on the panel filter's typing row did not clear it")
	}
}

func TestFinderListKeys(t *testing.T) {
	ta := newTestApp(t)
	opened := ""
	f := testFinder()
	f.Done = func(it FinderItem) tea.Cmd { opened = it.ID; return nil }
	ta.Find(f)
	ta.settle()
	fv := ta.topView().(*finderView)
	ta.press("r", "e", "a", "d", "Tab")
	if !fv.onList || len(fv.shown) != 3 {
		t.Fatalf("on the list with 3 rows: %v %d", fv.onList, len(fv.shown))
	}
	ta.press("k")
	if fv.cursor != 2 {
		t.Errorf("k on the first wraps: %d", fv.cursor)
	}
	ta.press("j")
	if fv.cursor != 0 {
		t.Errorf("j on the last wraps: %d", fv.cursor)
	}
	ta.press("/", "m")
	if fv.onList || fv.ed.value() != "readm" {
		t.Errorf("/ goes back to type on after the old words: %q", fv.ed.value())
	}
	ta.press("Tab", "Enter")
	if opened != "README.md" || ta.top() != nil {
		t.Errorf("Enter on the list goes there: %q", opened)
	}
	ta.Find(f)
	ta.settle()
	ta.press("Tab", "Esc")
	if ta.top() != nil {
		t.Error("Esc on the list closes the whole finder")
	}
}

func TestMatchContains(t *testing.T) {
	cases := []struct {
		text, typed string
		want        bool
	}{
		{"Prod-Web-01", "prod", true},
		{"db-replica-tokyo-ap-northeast-1", "prod", false},
		{"README.md", "READ", true},
		{"README.md", "rdm", false},
		{"anything", "", true},
	}
	for _, c := range cases {
		if got := Matches(c.text, c.typed); got != c.want {
			t.Errorf("Matches(%q, %q) = %v", c.text, c.typed, got)
		}
	}
	// A select's typing row filters the same way.
	ta := newTestApp(t)
	ta.Select(Select{Title: "Host", Options: []string{"Prod-Web-01", "db-replica-tokyo-ap-northeast-1", "staging"}})
	ta.settle()
	ta.press("/", "p", "r", "o", "d")
	if fv := ta.topView().(*finderView); !slices.Equal(fv.shown, []int{0}) {
		t.Errorf("prod in a select: %v", fv.shown)
	}
}

func TestMatchWords(t *testing.T) {
	if !Matches("prod-web-01 22", "prod 22") || !Matches("prod-web-01 22", "22 prod") || Matches("prod-web-01 22", "prod 23") {
		t.Error("every word must appear, in any order")
	}
	ta := newTestApp(t)
	ta.Find(testFinder())
	ta.settle()
	covFType(ta, "READ me")
	if fv := ta.topView().(*finderView); len(fv.shown) != 3 {
		t.Errorf("READ me finds %d", len(fv.shown))
	}
	ta.press("Esc")
	// A panel filter's words may fall in different columns: pod-10 and
	// pod-17 are 3 days old, pod-13 has a 3 in its name.
	ta.press("/")
	covFType(ta, "pod-1 3")
	if n := len(ta.list.view); n != 3 {
		t.Errorf("pod-1 3 matches %d rows, want 3", n)
	}
}

func TestMatchKeepsOrder(t *testing.T) {
	if got := Filter([]string{"9", "10", "20", "120", "200", "205", "2"}, "20"); !slices.Equal(got, []int{2, 3, 4, 5}) {
		t.Errorf("Filter keeps the order: %v", got)
	}
	// A list sorted by a number column stays sorted by value when filtered:
	// 10 20 100, not 10 100 20.
	l := &List{Columns: []Column{{Title: "Size", Number: true}}, Filterable: true}
	var rows []Row
	for _, n := range []string{"100", "9", "20", "10"} {
		rows = append(rows, Row{ID: n, Cells: []string{n}})
	}
	l.SetRows(rows)
	l.SortBy(SortKey{Col: 0})
	l.OpenFilter()
	l.Key("0", keyMsg("0"))
	var got []string
	for _, i := range l.view {
		got = append(got, l.rows[i].Cells[0])
	}
	if !slices.Equal(got, []string{"10", "20", "100"}) {
		t.Errorf("filtered: %v", got)
	}
}

func TestFinderHint(t *testing.T) {
	ta := newTestApp(t)
	f := testFinder()
	f.Verb = "open"
	ta.Find(f)
	ta.settle()
	if h := covFHint(ta); !strings.Contains(h, " Enter|Tab:list Esc:cancel ") {
		t.Errorf("typing row: %s", h)
	}
	ta.press("Tab")
	if h := covFHint(ta); !strings.Contains(h, " Enter:open Tab|/:filter Esc:cancel ") {
		t.Errorf("list: %s", h)
	}
	ta.press("Esc")
	ta.Select(Select{Title: "Profile", Options: []string{"a", "b"}})
	ta.settle()
	if h := covFHint(ta); !strings.Contains(h, " Enter:choose Tab|/:filter Esc:cancel ") {
		t.Errorf("select list: %s", h)
	}
	ta.press("/")
	if h := covFHint(ta); !strings.Contains(h, " Enter|Tab:list Esc:cancel ") {
		t.Errorf("select typing row: %s", h)
	}
	ta.press("Esc")
	ta.Checks(Checks{Title: "Namespaces", Options: []string{"a", "b"}})
	ta.settle()
	if h := covFHint(ta); !strings.Contains(h, " Enter:apply Space:toggle Tab|/:filter Esc:cancel ") {
		t.Errorf("checkbox popup list: %s", h)
	}
}

func TestRadio(t *testing.T) {
	changed := 0
	layout := &Field{Label: "Layout", Options: []string{"row", "column", "grid"}, OnChange: func() tea.Cmd { changed++; return nil }}
	v := &Values{Fields: []*Field{{Label: "Name", Value: func() string { return "web" }}, layout}}
	lines := v.Draw(40, 5, true)
	text := covFText(lines)
	// One option a row: the chosen Green and marked, the rest Text.
	if !strings.Contains(lines[1], Fg(Green).Render(glyphRadioOn+" row")) || !strings.HasPrefix(text[1], " LAYOUT  ") ||
		!strings.Contains(lines[2], Fg(Text).Render(glyphRadioOff+" column")) || !strings.Contains(lines[3], Fg(Text).Render(glyphRadioOff+" grid")) {
		t.Errorf("radio:\n%s", strings.Join(text, "\n"))
	}
	// Each option is an item; Enter chooses it at once, the cursor stays.
	v.Key("j", keyMsg("j"))
	v.Key("j", keyMsg("j"))
	v.Key("Enter", keyMsg("Enter"))
	if layout.Selected != 1 || changed != 1 || v.cursor != 2 {
		t.Errorf("Enter on column: selected %d, %d changes, cursor %d", layout.Selected, changed, v.cursor)
	}
	v.Key("j", keyMsg("j"))
	lines = v.Draw(40, 5, true)
	if !strings.Contains(lines[2], Fg(Green).Render(glyphRadioOn+" column")) || !strings.Contains(lines[1], Fg(Text).Render(glyphRadioOff+" row")) {
		t.Errorf("after choosing column:\n%s", strings.Join(covFText(lines), "\n"))
	}
	// On a form too: Enter chooses, the focus stays.
	ta := newTestApp(t)
	ta.Form(testForm(nil))
	ta.settle()
	fv := ta.topView().(*formView)
	ta.press("j", "j", "j", "Enter")
	if formStop(fv) != "Auth:1" || fv.f.Fields[2].Selected != 1 {
		t.Errorf("form: focus %s, selected %d", formStop(fv), fv.f.Fields[2].Selected)
	}
}

func TestCheckbox(t *testing.T) {
	notify := &Field{Label: "Notify", Options: []string{"on"}, Multi: true}
	agent := &Field{Label: "Agent", Options: []string{"forward", "x11"}, Multi: true, Checked: []bool{true, false}}
	v := &Values{Fields: []*Field{{Label: "Name", Value: func() string { return "web" }}, notify, agent}}
	lines := v.Draw(40, 5, true)
	if !strings.Contains(lines[1], Fg(Text).Render(glyphUnchecked+" on")) || !strings.Contains(lines[2], Fg(Green).Render(glyphMarked+" forward")) ||
		!strings.Contains(lines[3], Fg(Text).Render(glyphUnchecked+" x11")) {
		t.Errorf("checkboxes:\n%s", strings.Join(covFText(lines), "\n"))
	}
	// Enter turns one over in place.
	v.Key("j", keyMsg("j"))
	v.Key("Enter", keyMsg("Enter"))
	v.Key("j", keyMsg("j"))
	lines = v.Draw(40, 5, true)
	if v.cursor != 2 || !strings.Contains(lines[1], Fg(Green).Render(glyphMarked+" on")) {
		t.Errorf("Enter on Notify:\n%s", strings.Join(covFText(lines), "\n"))
	}
	// The checkbox popup: the finder skeleton with no preview, a checkbox
	// glyph on each row, as wide as its options.
	width := func(opts ...string) int {
		ta := newTestApp(t)
		ta.Checks(Checks{Title: "Namespaces", Options: opts})
		ta.settle()
		text := covFScreen(ta)
		if n := strings.Count(strings.Join(text, "\n"), "┏"); n != 1 {
			t.Errorf("%d boxes, want one: no preview", n)
		}
		if !strings.Contains(text[4], glyphUnchecked+" "+opts[0]) {
			t.Errorf("row %q", text[4])
		}
		return ta.top().w
	}
	short, long := width("a", "b"), width("a", strings.Repeat("kube-system-", 5))
	if short >= long || long >= 118 {
		t.Errorf("widths by the options: %d, %d", short, long)
	}
}

func TestDatePickerLayout(t *testing.T) {
	covFClock(t)
	ta := newTestApp(t)
	ta.PickDate(DatePicker{Title: "Due date", Value: covFDay(15, 0, 0), Min: covFDay(3, 0, 0), Max: covFDay(28, 0, 0)})
	ta.settle()
	lines, layer := covFDraw(ta)
	text := covFText(lines)
	// The month and the year two cells apart; the calendar one cell from the
	// frame; a day 4 cells, its number on the right; six weeks always.
	want := []string{
		"┃ October  2026",
		"┃ Mo  Tu  We  Th  Fr  Sa  Su",
		"┃              1   2   3   4",
		"┃  5   6   7   8   9  10  11",
		"┃ 12  13  14  15  16  17  18",
		"┃ 19  20  21  22  23  24  25",
		"┃ 26  27  28  29  30  31",
	}
	for i, w := range want {
		if strings.TrimRight(strings.TrimSuffix(text[2+i], "┃"), " ") != w {
			t.Errorf("row %d: %q, want %q", 2+i, text[2+i], w)
		}
	}
	if !covFBlank(text[9]) || len(lines) != 12 {
		t.Errorf("six weeks: the sixth blank, %d rows", len(lines))
	}
	if !strings.Contains(lines[3], Fg(Overlay0).Render("Mo  Tu  We  Th  Fr  Sa  Su")) {
		t.Error("the weekdays are Overlay0")
	}
	// The cursor opens on the value; out of range is Surface2; today is
	// underlined; the weekend has no colour of its own.
	if !strings.Contains(lines[6], On(layer).Render("15")) {
		t.Errorf("the cursor is on the value: %q", lines[6])
	}
	if !strings.Contains(lines[4], Fg(Surface2).Render(" 1")) || !strings.Contains(lines[8], Fg(Surface2).Render("29")) {
		t.Error("out of range is Surface2")
	}
	if !strings.Contains(lines[5], Style{Fg: Text, Underline: true}.Render(" 7")) {
		t.Errorf("today is underlined: %q", lines[5])
	}
	if !strings.Contains(lines[5], Fg(Text).Render("10")+"  "+Fg(Text).Render("11")) {
		t.Errorf("the weekend has no colour: %q", lines[5])
	}
	// The value is Green once the cursor leaves it.
	ta.press("l")
	if lines, _ = covFDraw(ta); !strings.Contains(lines[6], Fg(Green).Render("15")) || !strings.Contains(lines[6], On(layer).Render("16")) {
		t.Errorf("the value Green, the cursor on 16: %q", lines[6])
	}
	// The month and the year take the focus as the cursor.
	ta.press("Tab")
	if lines, _ = covFDraw(ta); !strings.Contains(lines[2], On(layer).Render("October")) {
		t.Errorf("month: %q", lines[2])
	}
	ta.press("Tab")
	if lines, _ = covFDraw(ta); !strings.Contains(lines[2], On(layer).Render("2026")) {
		t.Errorf("year: %q", lines[2])
	}
	h := ta.top().h
	ta.press("Esc")

	// No value: the cursor opens on today. A month of four weeks keeps six rows.
	ta.PickDate(DatePicker{Title: "Due date"})
	ta.settle()
	if lines, _ = covFDraw(ta); !strings.Contains(lines[5], On(layer).Render(" 7")) {
		t.Errorf("no value: the cursor on today: %q", lines[5])
	}
	ta.press("Esc")
	ta.PickDate(DatePicker{Title: "Due date", Value: time.Date(2027, 2, 10, 0, 0, 0, 0, time.Local)})
	ta.settle()
	text = covFScreen(ta)
	if ta.top().h != h || !strings.HasPrefix(text[7], "┃ 22  23") || !covFBlank(text[8]) || !covFBlank(text[9]) {
		t.Errorf("February 2027, four weeks: %d high (October %d)\n%s", ta.top().h, h, strings.Join(text, "\n"))
	}
	ta.press("Esc")

	// A week from Sunday, the app's choice.
	ta.PickDate(DatePicker{Title: "Due date", Value: covFDay(15, 0, 0), SundayFirst: true})
	ta.settle()
	text = covFScreen(ta)
	if !strings.HasPrefix(text[3], "┃ Su  Mo  Tu  We  Th  Fr  Sa") || !strings.HasPrefix(text[4], "┃                  1   2   3") {
		t.Errorf("Sunday first:\n%s\n%s", text[3], text[4])
	}
}

func TestDatetimePickerLayout(t *testing.T) {
	covFClock(t)
	ta := newTestApp(t)
	ta.PickDate(DatePicker{Title: "Meeting", Time: true, Value: covFDay(15, 10, 7)})
	ta.settle()
	lines, layer := covFDraw(ta)
	text := covFText(lines)
	// The separator: two cells either side, on the content rows only.
	col := strings.Index(text[2], "│")
	for i := 2; i <= 9; i++ {
		if !strings.Contains(text[i], "  │  ") || strings.Index(text[i], "│") != col {
			t.Errorf("row %d: %q", i, text[i])
		}
	}
	if strings.Contains(text[1], "│") || strings.Contains(text[10], "│") {
		t.Error("the separator touches the frame")
	}
	// The hour and the minute, two cells apart, as high as the calendar.
	var hours []string
	for i := 2; i <= 9; i++ {
		f := strings.Fields(text[i][col+len("│"):])
		if len(f) < 2 || len(f[0]) != 2 || len(f[1]) != 2 || !strings.Contains(text[i], "│  "+f[0]+"  "+f[1]+" ") {
			t.Errorf("time row %d: %q", i, text[i])
			continue
		}
		hours = append(hours, f[0])
	}
	if !slices.Contains(hours, "10") {
		t.Errorf("the hours shown %v hold the value's 10", hours)
	}
	// 10:07 is not on the grid: the minute's cursor is on 05, the one before
	// (dimmed: the days have the focus).
	if dv := ta.topView().(*dateView); dv.minute != 1 {
		t.Errorf("the minute's cursor is at %02d, want 05", dv.minute*5)
	}
	for i := 2; i <= 9; i++ {
		if f := strings.Fields(text[i][col+len("│"):]); len(f) == 2 && f[1] == "05" && !strings.Contains(lines[i], sgrRGB(false, DimRGB(layer.RGB))) {
			t.Errorf("05 is not the minute's cursor: %q", lines[i])
		}
	}
	// The value is Green once the cursor leaves it.
	ta.press("Tab", "j")
	if lines, _ = covFDraw(ta); !strings.Contains(strings.Join(lines, ""), Fg(Green).Render("10")) {
		t.Error("the value's hour is Green")
	}
	// The width is fixed: it does not depend on the value.
	w := ta.top().w
	ta.press("Esc")
	ta.PickDate(DatePicker{Title: "Meeting", Time: true, Value: time.Date(2027, 5, 31, 23, 55, 0, 0, time.Local)})
	ta.settle()
	if ta.top().w != w || w >= 118 {
		t.Errorf("widths %d and %d", w, ta.top().w)
	}
}

func TestDatetimeOneBrightCursor(t *testing.T) {
	covFClock(t)
	ta := newTestApp(t)
	ta.PickDate(DatePicker{Title: "Meeting", Time: true, Value: covFDay(15, 10, 5)})
	ta.settle()
	count := func() (int, int) {
		lines, layer := covFDraw(ta)
		body := strings.Join(lines[1:len(lines)-1], "\n")
		return strings.Count(body, sgrRGB(false, layer.RGB)), strings.Count(body, sgrRGB(false, DimRGB(layer.RGB)))
	}
	for _, step := range []struct {
		key      string
		bright   int
		dimCount int
	}{
		{"", 1, 2},    // the days: the hour and the minute dimmed
		{"Tab", 1, 2}, // the hour: the day and the minute dimmed
		{"l", 1, 2},   // the minute: the day and the hour dimmed
		{"Tab", 1, 3}, // the month: the day, the hour and the minute dimmed
		{"Tab", 1, 3}, // the year
	} {
		if step.key != "" {
			ta.press(step.key)
		}
		if b, d := count(); b != step.bright || d != step.dimCount {
			t.Errorf("after %q: %d bright cursors, %d dimmed; want %d, %d", step.key, b, d, step.bright, step.dimCount)
		}
	}
}

func TestDatetimeTimeWraps(t *testing.T) {
	covFClock(t)
	ta := newTestApp(t)
	ta.PickDate(DatePicker{Title: "Meeting", Time: true, Value: covFDay(15, 23, 55)})
	ta.settle()
	dv := ta.topView().(*dateView)
	ta.press("Tab", "j")
	if dv.hour != 0 || dv.day.Day() != 15 {
		t.Errorf("23 then 00, the date unchanged: %d, day %d", dv.hour, dv.day.Day())
	}
	ta.press("l", "j")
	if dv.minute != 0 || dv.hour != 0 {
		t.Errorf("55 then 00, the hour unchanged: %d:%02d", dv.hour, dv.minute*5)
	}
	ta.press("k")
	if dv.minute != 11 || dv.hour != 0 {
		t.Errorf("00 back to 55: %d:%02d", dv.hour, dv.minute*5)
	}
	ta.press("h", "k")
	if dv.hour != 23 || dv.day.Day() != 15 {
		t.Errorf("00 back to 23, the date unchanged: %d, day %d", dv.hour, dv.day.Day())
	}
}

func TestDatetimeEnter(t *testing.T) {
	covFClock(t)
	ta := newTestApp(t)
	var got time.Time
	done := func(v time.Time) tea.Cmd { got = v; return nil }
	ta.PickDate(DatePicker{Title: "Due", Value: covFDay(15, 0, 0), Done: done})
	ta.settle()
	dv := ta.topView().(*dateView)
	for _, keys := range [][]string{{"Tab"}, {"Tab", "Tab"}} { // to the month, to the year
		ta.press(keys...)
		f := dv.focus
		ta.press("Enter")
		if f == onDays || dv.focus != onDays || ta.top() == nil || !got.IsZero() {
			t.Errorf("Enter on %v goes to the days without choosing: %v", f, dv.focus)
		}
	}
	ta.press("l", "Enter")
	if !got.Equal(covFDay(16, 0, 0)) || ta.top() != nil {
		t.Errorf("Enter on the days of a date picker chooses: %v", got)
	}
	ta.PickDate(DatePicker{Title: "Meeting", Time: true, Value: covFDay(15, 10, 5), Done: done})
	ta.settle()
	dv = ta.topView().(*dateView)
	ta.press("Enter")
	if dv.focus != onTime || ta.top() == nil {
		t.Fatalf("Enter on the days of a datetime goes to the time: %v", dv.focus)
	}
	ta.press("j", "Enter")
	if !got.Equal(covFDay(15, 11, 5)) || ta.top() != nil {
		t.Errorf("Enter on the time chooses the day and the time: %v", got)
	}
}

func TestDatetimeHint(t *testing.T) {
	covFClock(t)
	ta := newTestApp(t)
	ta.PickDate(DatePicker{Title: "Due"})
	ta.settle()
	if h := covFHint(ta); !strings.Contains(h, " Enter:choose t:today Tab:month Esc:cancel ") {
		t.Errorf("date, days: %s", h)
	}
	ta.press("Esc")
	ta.PickDate(DatePicker{Title: "Meeting", Time: true})
	ta.settle()
	for _, want := range []string{
		"Enter:time t:today Tab:time Esc:cancel",
		"Enter:choose t:today Tab:month Esc:cancel",
		"Enter:days h|l:month t:today Tab:year Esc:cancel",
		"Enter:days h|l:year t:today Tab:days Esc:cancel",
	} {
		if h := covFHint(ta); !strings.Contains(h, " "+want+" ") {
			t.Errorf("hint %s, want %s", h, want)
		}
		ta.press("Tab")
	}
}

func TestColorPickerLayout(t *testing.T) {
	ta := newTestApp(t)
	old := Hex("#2A2A3C")
	ta.PickColor(ColorPicker{Title: "Background", Value: old})
	ta.settle()
	lines, _ := covFDraw(ta)
	text := covFText(lines)
	want := []string{
		"┃ ████████  ████████                   ┃",
		"┃ ████████  ████████  ███  ███  ███    ┃",
		"┃ old       new       R    G    B      ┃",
		"┃ #2A2A3C   #2A2A3C   2A   2A   3C     ┃",
		"┃                                      ┃",
		"┃ ──────────────────────────────────── ┃",
		"┃                              Choose  ┃",
		"┃                                      ┃",
	}
	// As wide as the widest hint, Enter:choose Tab:fields Esc:cancel.
	if len(lines) != 11 || ta.top().w != 40 {
		t.Fatalf("%d×%d:\n%s", ta.top().w, len(lines), strings.Join(text, "\n"))
	}
	for i, w := range want {
		if text[2+i] != w {
			t.Errorf("row %d: %q, want %q", 2+i, text[2+i], w)
		}
	}
	// R, G and B in their own channel's colour; old and new in theirs.
	for _, c := range []string{"#2A0000", "#002A00", "#00003C"} {
		if !strings.Contains(lines[3], Fg(Hex(c)).Render("███")) {
			t.Errorf("no %s swatch", c)
		}
	}
	if !strings.Contains(lines[2], Fg(old).Render("████████")+"  "+Fg(old).Render("████████")) {
		t.Error("old and new swatches")
	}
	// The button as a form's: Surface1 and Text without the focus.
	if !strings.Contains(lines[8], Style{Fg: Text, Bg: Surface1}.Render(" Choose ")) {
		t.Errorf("button %q", lines[8])
	}
}

func TestColorPickerKeys(t *testing.T) {
	ta := newTestApp(t)
	var got Color
	in := ColorPicker{Title: "Background", Value: Hex("#2A2A3C"), Done: func(c Color) tea.Cmd { got = c; return nil }}
	ta.PickColor(in)
	ta.settle()
	cv := ta.topView().(*colorView)
	lines, layer := covFDraw(ta)
	// It opens on R: its name and hex drawn as the cursor, as wide as the swatch.
	if cv.focus != 0 || !strings.Contains(lines[4], On(layer).Render("R  ")) || !strings.Contains(lines[5], On(layer).Render("2A ")) {
		t.Errorf("opens on R: focus %d, %q", cv.focus, lines[4])
	}
	if h := covFHint(ta); !strings.Contains(h, " Enter:edit Tab:actions Esc:cancel ") {
		t.Errorf("hint on R: %s", h)
	}
	for i, k := range []string{"l", "l", "l", "l", "k", "j", "h"} {
		ta.press(k)
		want := []int{1, 2, 3, 0, 3, 0, 3}[i]
		if cv.focus != want {
			t.Errorf("%s: focus %d, want %d (R G B Choose, wrapping)", k, cv.focus, want)
		}
	}
	lines, _ = covFDraw(ta)
	if !strings.Contains(lines[8], On(layer).Render(" Choose ")) || !strings.Contains(covFHint(ta), " Enter:choose Tab:fields Esc:cancel ") {
		t.Errorf("on Choose: %q", StripStyles(lines[len(lines)-1]))
	}
	ta.press("Esc")
	if got.Set || ta.top() != nil {
		t.Error("Esc changed the colour")
	}
	ta.PickColor(in)
	ta.settle()
	ta.press("h", "Enter")
	if got.Set || ta.top() != nil {
		t.Error("Choose with nothing changed wrote the colour")
	}
}

// Tab and Shift-Tab go between the channels and Choose, back to the channel
// left (Rules K2, components/input/color-picker).
func TestColorPickerTab(t *testing.T) {
	ta := newTestApp(t)
	ta.PickColor(ColorPicker{Title: "Background", Value: Hex("#2A2A3C")})
	ta.settle()
	cv := ta.topView().(*colorView)
	for i, k := range []string{"Tab", "Tab", "l", "Tab", "Shift-Tab", "Shift-Tab", "l", "l", "Tab", "Tab"} {
		ta.press(k)
		want := []int{3, 0, 1, 3, 1, 3, 0, 1, 3, 1}[i]
		if cv.focus != want {
			t.Errorf("%d %s: focus %d, want %d", i, k, cv.focus, want)
		}
	}
	// h|l still go one by one and wrap through Choose; Tab from there goes
	// back to the channel left, B.
	ta.press("l", "l")
	if cv.focus != colorChoose {
		t.Fatalf("l to Choose: focus %d", cv.focus)
	}
	ta.press("Tab")
	if cv.focus != 2 {
		t.Errorf("Tab from Choose reached by l: focus %d, want B", cv.focus)
	}
	if ta.top() == nil || len(ta.stack) != 1 {
		t.Error("Tab left the color picker")
	}
}

func TestColorPickerChannelSelect(t *testing.T) {
	ta := newTestApp(t)
	ta.PickColor(ColorPicker{Title: "Background", Value: Hex("#2A2A3C")})
	ta.settle()
	cv := ta.topView().(*colorView)
	ta.press("l", "Enter")
	sel := ta.topView().(*finderView)
	rows := sel.rows()
	if len(rows) != 256 || rows[0].text != "00 (0)" || rows[255].text != "FF (255)" {
		t.Fatalf("00–FF: %d rows, %q … %q", len(rows), rows[0].text, rows[len(rows)-1].text)
	}
	if _, r, _ := sel.at(); r.text != "2A (42)" {
		t.Errorf("the cursor on the value: %q", r.text)
	}
	shown := func() []string {
		var out []string
		for _, i := range sel.shown {
			out = append(out, rows[i].text)
		}
		return out
	}
	ta.press("/", "f", "f")
	if !slices.Equal(shown(), []string{"FF (255)"}) {
		t.Errorf("ff finds %v", shown())
	}
	ta.press("Backspace", "Backspace", "4", "2")
	if s := shown(); !slices.Contains(s, "2A (42)") || !slices.Contains(s, "42 (66)") {
		t.Errorf("42 finds the hex and the decimal: %v", s)
	}
	ta.press("Esc")
	if ta.topView() != popupView(cv) {
		t.Fatalf("Esc closes the select only: %T", ta.topView())
	}
	ta.press("Enter", "j", "Enter")
	lines, _ := covFDraw(ta)
	if cv.focus != 1 || !strings.Contains(lines[3], Fg(Hex("#2A2B3C")).Render("████████")) || !strings.Contains(StripStyles(lines[5]), "#2A2B3C") {
		t.Errorf("new follows G, the focus stays: focus %d, %q", cv.focus, StripStyles(lines[5]))
	}
}

func TestFilePickerLayout(t *testing.T) {
	dir := fakeTree(t)
	ta := newTestApp(t)
	ta.PickFile(FilePicker{Title: "Identity file", Root: dir, Value: filepath.Join(dir, "config")})
	ta.settle()
	fv := ta.topView().(*finderView)
	lines, _ := covFDraw(ta)
	text := covFText(lines)
	// The title is the field and the directory, the path cut from its front.
	if full := fv.title(1 << 20); full != "IDENTITY FILE · "+homeShort(dir) {
		t.Errorf("title %q", full)
	}
	if short := StripStyles(fv.title(30)); !strings.HasPrefix(short, "IDENTITY FILE · …") || !strings.HasSuffix(short, filepath.Base(dir)) {
		t.Errorf("a long directory is cut from the front: %q", short)
	}
	// The typing row opens empty, the angle before it.
	if !fv.ed.empty() || !strings.HasPrefix(text[2], "┃ "+glyphTyping+" ") {
		t.Errorf("typing row %q", text[2])
	}
	// .. on top, an icon before every row; the cursor on the value.
	if !strings.HasPrefix(text[4], "┃ "+glyphFolder+" ..") || !strings.Contains(text[5], glyphFolder+" old") || !strings.Contains(text[7], glyphFile+" config") {
		t.Errorf("rows:\n%s", strings.Join(text[4:9], "\n"))
	}
	if _, r, _ := fv.at(); r.text != "config" {
		t.Errorf("the cursor starts on the value, not %q", r.text)
	}
	ta.press("j")
	if lines, _ = covFDraw(ta); !strings.Contains(covFRow(lines, " config"), Fg(Green).Render("config")) {
		t.Error("the value is Green")
	}
	ta.press("Esc")
	// The icon and its colour are the app's.
	ta.PickFile(FilePicker{Title: "Identity file", Root: dir, Icon: func(string, bool) (string, Color) { return "X", Red }})
	ta.settle()
	if lines, _ = covFDraw(ta); !strings.Contains(covFRow(lines, "X old"), Fg(Red).Render("X")) {
		t.Error("the app's icon colour")
	}
	ta.press("Esc")
	// The root / has no ..
	ta.PickFile(FilePicker{Title: "File", Root: "/"})
	ta.settle()
	if rows := ta.topView().(*finderView).rows(); len(rows) == 0 || rows[0].text == ".." {
		t.Error("/ has a .. row")
	}
}

func TestFilePickerHint(t *testing.T) {
	dir := fakeTree(t)
	ta := newTestApp(t)
	ta.PickFile(FilePicker{Title: "Identity file", Root: dir})
	ta.settle()
	if h := covFHint(ta); !strings.Contains(h, " Enter:into Tab|/:filter Esc:cancel ") {
		t.Errorf("on a directory: %s", h)
	}
	ta.press("G")
	if h := covFHint(ta); !strings.Contains(h, " Enter:choose Tab|/:filter Esc:cancel ") {
		t.Errorf("on a file: %s", h)
	}
	ta.press("Tab")
	if h := covFHint(ta); !strings.Contains(h, " Enter|Tab:list Esc:cancel ") {
		t.Errorf("typing row: %s", h)
	}
}

// An error names the value by its name (components/input/README): an input
// whose title says what it does ("Save as") is given the name; without one
// the title names it, as on a form's field.
func TestInputErrorNamesValue(t *testing.T) {
	ta := newTestApp(t)
	for _, c := range []struct {
		open func() tea.Cmd
		want string
	}{
		{func() tea.Cmd { return ta.Text(TextInput{Title: "Save as", Name: "Name"}) }, "Name can't have control characters"},
		{func() tea.Cmd { return ta.Password(PasswordInput{Title: "Log in", Name: "Password"}) }, "Password can't have control characters"},
		{func() tea.Cmd { return ta.Text(TextInput{Title: "Name"}) }, "Name can't have control characters"},
	} {
		c.open()
		ta.settle()
		covFPaste(ta, "web\n01")
		ta.press("Enter")
		if text := ta.screenText(); ta.top() == nil || !strings.Contains(text, c.want) {
			t.Errorf("want %q:\n%s", c.want, text)
		}
		ta.press("Esc")
	}
	ta.Textarea(Textarea{Title: "Edit", Subject: "notes", Name: "Notes"})
	ta.settle()
	covFPaste(ta, "x\x1by")
	ta.press("Esc", "Enter")
	if text := ta.screenText(); !strings.Contains(text, "Notes can't have control characters") {
		t.Errorf("the textarea's error:\n%s", text)
	}
}
