package terminu

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// Tests for the rows of docs/coverage.md under Rules M, L, F and X, T, S.

// covBCell is one cell of a drawn line: its text ("" for the second cell of a
// wide character) and the colours it is drawn in, as "r;g;b" ("" for none).
type covBCell struct {
	text, fg, bg string
	bold         bool
}

// covBCells splits a drawn line into its cells, following its SGR sequences.
func covBCells(line string) []covBCell {
	var out []covBCell
	var cur covBCell
	walk(line, func(seq string, cells int) {
		if cells > 0 {
			c := cur
			c.text = seq
			out = append(out, c)
			for range cells - 1 {
				c.text = ""
				out = append(out, c)
			}
			return
		}
		if !strings.HasPrefix(seq, "\x1b[") || !strings.HasSuffix(seq, "m") {
			return
		}
		ps := strings.Split(seq[2:len(seq)-1], ";")
		for i := 0; i < len(ps); i++ {
			switch ps[i] {
			case "", "0":
				cur = covBCell{}
			case "1":
				cur.bold = true
			case "39":
				cur.fg = ""
			case "49":
				cur.bg = ""
			case "38", "48":
				if i+4 < len(ps) && ps[i+1] == "2" {
					rgb := strings.Join(ps[i+2:i+5], ";")
					if ps[i] == "38" {
						cur.fg = rgb
					} else {
						cur.bg = rgb
					}
					i += 4
				}
			}
		}
	})
	return out
}

// covBFind is the first cell from which the cells read s (one cell a rune).
func covBFind(cells []covBCell, s string) int {
	rs := []rune(s)
	for i := 0; i+len(rs) <= len(cells); i++ {
		k := 0
		for k < len(rs) && cells[i+k].text == string(rs[k]) {
			k++
		}
		if k == len(rs) {
			return i
		}
	}
	return -1
}

// covBAt is the cell where s starts in the columns [from, to) of line; a
// zero cell when it is not there.
func covBAt(line string, from, to int, s string) covBCell {
	cells := covBCells(line)
	to = min(to, len(cells))
	if from >= to {
		return covBCell{}
	}
	if i := covBFind(cells[from:to], s); i >= 0 {
		return cells[from+i]
	}
	return covBCell{}
}

// covBText is the text of the columns [from, to) of line.
func covBText(line string, from, to int) string {
	var b strings.Builder
	cells := covBCells(line)
	for i := from; i < to && i < len(cells); i++ {
		b.WriteString(cells[i].text)
	}
	return b.String()
}

func covBRGB(c Color) string { return fmt.Sprintf("%d;%d;%d", c.R(), c.G(), c.B()) }

func covBDim(c Color) string { return covBRGB(DimColor(c)) }

// covBFooter is the footer as text.
func (ta *testApp) covBFooter() string {
	f := ta.Frame()
	return strings.TrimRight(StripStyles(f[len(f)-1]), " ")
}

// covBPopupRow is row y of the top popup as text, frame to frame.
func (ta *testApp) covBPopupRow(y int) string {
	l := ta.top()
	return covBText(ta.Frame()[y], l.x, l.x+l.w)
}

// covBCloseAll closes every popup.
func (ta *testApp) covBCloseAll() {
	ta.closeAll()
	ta.settle()
}

// covBThreeLayers stacks three menus and a confirm, each smaller than the one
// under it so that every frame shows: layer 1 a wide menu, layer 2 a narrower
// one, layer 3 a confirm.
func (ta *testApp) covBThreeLayers(t *testing.T) (l1, l2, l3 *layer) {
	t.Helper()
	second := func() tea.Cmd {
		acts := []Action{{Label: "confirm", Stay: true, Do: func() tea.Cmd {
			return ta.Confirm(Ask{Title: "Sure", Question: "Sure?", Verb: "go"}, nil)
		}}}
		for i := range 10 {
			acts = append(acts, Action{Label: fmt.Sprintf("second %d", i), Desc: strings.Repeat("mid ", 12)})
		}
		return ta.Menu("Second", MenuSection{Actions: acts})
	}
	acts := []Action{{Label: "open the second", Stay: true, Do: second}}
	for i := range 20 {
		acts = append(acts, Action{Label: fmt.Sprintf("first %d", i), Desc: strings.Repeat("wide ", 18)})
	}
	ta.Menu("First", MenuSection{Actions: acts})
	ta.settle()
	ta.press("Enter", "Enter")
	if len(ta.stack) != 3 {
		t.Fatalf("want three layers, have %d", len(ta.stack))
	}
	return ta.stack[0], ta.stack[1], ta.stack[2]
}

// --- M ----------------------------------------------------------------------

func TestM1FooterEntries(t *testing.T) {
	ta := newTestApp(t)
	if got := ta.covBFooter(); got != "Space:menu ?:help Tab|1–3:panels q:quit" {
		t.Errorf("footer: %q", got)
	}
	ta.press("Space")
	if got := ta.covBFooter(); got != "Space:menu ?:help Tab|1–3:panels q:quit" {
		t.Errorf("with a popup open the footer is the screen's under it: %q", got)
	}
	ta.press("Esc", "O")
	if got := ta.covBFooter(); got != "Space:menu ?:help q:quit" {
		t.Errorf("a screen of one panel: %q", got)
	}
	ta.press("P", "D")
	if got := ta.covBFooter(); !strings.HasPrefix(got, "?:help") || strings.Contains(got, "Space") {
		t.Errorf("in a mode ? stays and Space need not: %q", got)
	}
}

func TestM2HeadingStrings(t *testing.T) {
	ta := newTestApp(t)
	ta.press("Space")
	m := ta.topView().(*menuView)
	var heads []string
	for _, r := range m.rows {
		if r.kind == menuTitle {
			heads = append(heads, r.label)
		}
	}
	if !slices.Equal(heads, []string{"item operation", "panel operation"}) {
		t.Errorf("section titles %q", heads)
	}
	text := ta.screenText()
	for _, h := range []string{glyphHexagon + " ITEM OPERATION", glyphHexagon + " PANEL OPERATION"} {
		if !strings.Contains(text, h) {
			t.Errorf("no %q: titles are drawn in capitals after the hexagon", h)
		}
	}
	if strings.Contains(text, "GLOBAL OPERATION") {
		t.Error("the global row has a section title")
	}
	lines := strings.Split(text, "\n")
	found := false
	for y, l := range lines {
		if strings.Contains(l, "Global operation") {
			found = true
			if !strings.Contains(l, "┃ Global operation") || !strings.Contains(l, "actions for the whole app") {
				t.Errorf("the global row has no hotkey and says what it is: %q", l)
			}
			if !strings.Contains(lines[y-1], "╌") {
				t.Errorf("no separator above the global row: %q", lines[y-1])
			}
		}
	}
	if !found {
		t.Error("no Global operation row")
	}
}

func TestM2NoTargetNoSection(t *testing.T) {
	ta := newTestApp(t)
	ta.list.SetRows(nil)
	ta.press("Space")
	text := ta.screenText()
	if strings.Contains(text, "ITEM OPERATION") || strings.Contains(text, "[m]ark") {
		t.Errorf("an empty list has no item operation section:\n%s", text)
	}
	if !strings.Contains(text, "Global operation") {
		t.Error("the global row is always there")
	}
	ta.press("Esc", "?")
	if strings.Contains(ta.screenText(), "ITEM OPERATION") {
		t.Error("the key reference lists item operations with no item")
	}
}

func TestM2HeadingAlways(t *testing.T) {
	ta := newTestApp(t)
	ta.list.SetRows(nil)
	ta.press("Space")
	if !strings.Contains(ta.screenText(), glyphHexagon+" PANEL OPERATION") {
		t.Error("the panel section alone keeps its title")
	}
	// A panel with item operations only.
	tb := &testApp{App: anySize(New("t"))}
	l := &List{Columns: []Column{{Title: "Name"}}}
	l.SetRows([]Row{{ID: "a", Cells: []string{"a"}}})
	tb.AddScreen(&Screen{Name: "Hosts", Key: "H", Panels: []*Panel{{Label: "Hosts", Body: l,
		Actions: []Action{{Key: "o", Label: "open", Desc: "connect", Scope: ItemScope}}}}})
	tb.resize(80, 30)
	tb.press("Space")
	if text := tb.screenText(); !strings.Contains(text, glyphHexagon+" ITEM OPERATION") || strings.Contains(text, "PANEL OPERATION") {
		t.Errorf("the item section alone keeps its title:\n%s", text)
	}
}

func TestM3EveryHotkeyInMenu(t *testing.T) {
	ta := newTestApp(t)
	for i, p := range ta.Screen().Panels {
		ta.press(fmt.Sprint(i+1), "Space")
		text := ta.screenText()
		for _, scope := range []Scope{ItemScope, PanelScope} {
			for _, act := range ta.panelActions(p, scope) {
				if !strings.Contains(text, MarkLabel(act.Key, act.Label)) {
					t.Errorf("panel %s: %q is not in its Space menu", p.Label, MarkLabel(act.Key, act.Label))
				}
			}
		}
		ta.press("Esc")
	}
	ta.press("1", "Space", "G", "Enter")
	text := ta.screenText()
	for _, act := range ta.globalActions() {
		if !strings.Contains(text, MarkLabel(act.Key, act.Label)) {
			t.Errorf("%q is not in the global operation popup", MarkLabel(act.Key, act.Label))
		}
	}
	// A popup's own keys are in its ?: the Space menu's.
	ta.press("Esc", "?")
	ref := StripStyles(strings.Join(ta.topView().(*noteView).lines, "\n"))
	for _, want := range []string{"mark — this pod", "Delete — this pod", "Drag — this pod", "Filter — narrow the list as you type"} {
		if !strings.Contains(ref, want) {
			t.Errorf("the Space menu's ? lacks %q:\n%s", want, ref)
		}
	}
	ta.covBCloseAll()
	ta.press("z")
	if len(ta.ran) != 0 || ta.top() != nil {
		t.Error("a key no action registered did something")
	}
}

func TestM3TypingPopupHintFits80(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(80, 40)
	full := "Enter:save Tab:edit Backspace:clear Esc:cancel"
	masked := "Enter:save Backspace:clear Esc:cancel"
	for name, c := range map[string]struct {
		open func()
		hint string
	}{
		"text":     {func() { ta.Text(TextInput{Title: "Name", Prompt: "A name", Old: "web-01"}) }, full},
		"password": {func() { ta.Password(PasswordInput{Title: "Password", Prompt: "For root", Set: true}) }, masked},
		"pin":      {func() { ta.Password(PasswordInput{Title: "PIN", Prompt: "Up to 12 digits", Unlock: true, MaxLen: 12}) }, masked},
		"number":   {func() { ta.Number(NumberInput{Title: "Port", Prompt: "1-65535", Min: 1, Max: 65535, Old: "22"}) }, full},
	} {
		c.open()
		ta.settle()
		l := ta.top()
		if bottom := ta.covBPopupRow(l.y + l.h - 1); !strings.Contains(bottom, "┗━ "+c.hint+" ━") {
			t.Errorf("%s: the whole hint at 80 columns: %q", name, bottom)
		}
		ta.press("Esc")
	}
}

func TestM4GlobalPopup(t *testing.T) {
	ta := newTestApp(t)
	ta.press("Space", "G", "Enter")
	if len(ta.stack) != 2 {
		t.Fatalf("want the global operation popup over the Space menu, have %d layers", len(ta.stack))
	}
	under, _ := ta.stack[0].view.(*menuView)
	top, ok := ta.stack[1].view.(*menuView)
	if under == nil || !under.space || !ok || top.space {
		t.Fatalf("a menu over the Space menu: %T over %T", ta.stack[1].view, ta.stack[0].view)
	}
	var labels []string
	for _, r := range top.rows {
		if r.kind == menuItem {
			labels = append(labels, r.label)
		}
	}
	if want := []string{"[P]anels", "[O]ther", "[S]tart", "[q]uit"}; !slices.Equal(labels, want) {
		t.Errorf("rows %q, want every global operation %q", labels, want)
	}
	if !strings.Contains(ta.screenText(), "GLOBAL OPERATION") {
		t.Error("no GLOBAL OPERATION title")
	}
	ta.press("G", "Enter")
	if _, ok := ta.topView().(*confirmView); !ok || !ta.quitting {
		t.Errorf("leaving is here: %T", ta.topView())
	}
}

func TestM4EscBackToSpaceMenu(t *testing.T) {
	ta := newTestApp(t)
	ta.press("Space", "G", "Enter", "Esc")
	if m, ok := ta.topView().(*menuView); !ok || !m.space || len(ta.stack) != 1 {
		t.Fatalf("Esc goes back to the Space menu: %T, %d layers", ta.topView(), len(ta.stack))
	}
	ta.press("G", "Enter", "S")
	if strings.Join(ta.ran, ",") != "start" || ta.top() != nil {
		t.Errorf("running a global operation closes the whole stack: ran %v, top %T", ta.ran, ta.topView())
	}
}

func TestM4CurrentScreenDisabled(t *testing.T) {
	ta := newTestApp(t)
	ta.press("Space", "G", "Enter")
	m := ta.topView().(*menuView)
	if r := m.rows[m.cursor]; r.label != "[O]ther" {
		t.Errorf("the cursor skips the current screen's row: on %q", r.label)
	}
	row := ta.rowWith("[P]anels")
	if !strings.Contains(row, "\x1b[38;2;"+covBRGB(Surface2)+"m[P]anels") || !strings.Contains(StripStyles(row), "switch to this screen") {
		t.Errorf("the current screen's row is in Surface2, its description kept: %q", row)
	}
	ta.press("P")
	if ta.cur != 0 || len(ta.stack) != 2 {
		t.Error("the disabled row's hotkey did something")
	}
	ta.press("O")
	if ta.cur != 1 || ta.top() != nil {
		t.Error("the other screen's row does not switch")
	}
}

func TestKeyReferenceReadOnly(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(100, 20)
	ta.press("?")
	n, ok := ta.topView().(*noteView)
	if !ok {
		t.Fatalf("? opened %T", ta.topView())
	}
	ta.press("Enter", "x", "m", "S", "Space")
	if len(ta.ran) != 0 || len(ta.list.Marked()) != 0 || ta.topView() != n {
		t.Errorf("the key reference ran something: ran %v, top %T", ta.ran, ta.topView())
	}
	l := ta.top()
	f := ta.Frame()
	for y := l.y + 1; y < l.y+l.h-1; y++ {
		cells := covBCells(f[y])
		for x := l.x; x < l.x+l.w; x++ {
			if cells[x].bg != "" {
				t.Fatalf("row %d has a block at %d: a key reference has no cursor", y, x)
			}
		}
	}
	if !strings.Contains(ta.covBPopupRow(l.y+l.h-1), "1-") {
		t.Errorf("no scroll position: %q", ta.covBPopupRow(l.y+l.h-1))
	}
	ta.press("j")
	if n.top != 1 {
		t.Errorf("j scrolls: top %d", n.top)
	}
}

func TestMenuRowLayout(t *testing.T) {
	ta := newTestApp(t)
	ta.press("Space")
	l := ta.top()
	frame := ta.Frame()
	found := false
	for y := l.y; y < l.y+l.h; y++ {
		row := covBText(frame[y], l.x, l.x+l.w)
		if !strings.Contains(row, "[x] Delete") {
			continue
		}
		found = true
		if !strings.HasPrefix(row, "┃ [x] Delete ") || !strings.HasSuffix(row, " this pod ┃") {
			t.Errorf("the name on the left, the description on the right: %q", row)
		}
		if !strings.Contains(frame[y], Fg(Text).Render("[x] Delete")) || !strings.Contains(frame[y], Fg(Overlay0).Render("this pod")) {
			t.Errorf("the name in Text, the description in Overlay0: %q", frame[y])
		}
	}
	if !found {
		t.Error("no [x] Delete row")
	}
	ta.covBCloseAll()
	ta.resize(40, 20)
	ta.Menu("Sort", MenuSection{Actions: []Action{{Label: "Name", Desc: strings.Repeat("a very long description ", 4)}}})
	ta.settle()
	l = ta.top()
	if row := ta.covBPopupRow(l.y + 2); !strings.HasPrefix(row, "┃ Name  ") || !strings.HasSuffix(row, "…"+glyphHard+"┃") {
		t.Errorf("a description too long is cut with …, on one row: %q", row)
	}
}

func TestM5HotkeyMarking(t *testing.T) {
	for _, c := range []struct{ key, label, want string }{
		{"r", "rename", "[r]ename"},
		{"D", "Delete", "[D]elete"},
		{"A", "Add", "[A]dd"},
		{"L", "URL", "UR[L]"},
		{"n", "New", "[n]ew"},
		{"e", "Delete", "D[e]lete"},
		{"P", "Export", "Ex[P]ort"},
		{"x", "Reset", "[x] Reset"},
		{"Alt-T", "Alterm", "Al[Alt-T]erm"},
		{"go", "Goto", "[go]to"},
		{"go", "Go to", "[go] to"},
		{"go", "Ago", "[go] Ago"},
		{"Alt-x", "term", "[Alt-x] term"},
		{"Alt-t", "term", "[Alt-t]erm"},
		{"go", "goto", "[go]to"},
		{"3", "Favorites", "[3] Favorites"},
		{"3", "432hz", "[3] 432hz"},
		{"Enter", "Edit", "[Enter] Edit"},
		{"/", "Search", "[/] Search"},
		{"/", "[/] Search", "[/] Search"},
		{"", "Refresh", "Refresh"},
	} {
		if got := MarkLabel(c.key, c.label); got != c.want {
			t.Errorf("MarkLabel(%q, %q) = %q, want %q", c.key, c.label, got, c.want)
		}
	}
	ta := newTestApp(t)
	ta.press("Space")
	text := ta.screenText()
	for _, want := range []string{"[Enter] Logs", "[m]ark", "[x] Delete", "[D]rag", "[/] Filter", "┃ Refresh", "[P]ANELS", "[O]THER"} {
		if !strings.Contains(text, want) {
			t.Errorf("%q is not written out on the screen", want)
		}
	}
}

func TestM5KeyNames(t *testing.T) {
	runes := func(s string, alt bool) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s), Alt: alt} }
	for _, c := range []struct {
		msg  tea.KeyMsg
		want string
	}{
		{tea.KeyMsg{Type: tea.KeyEsc}, "Esc"},
		{tea.KeyMsg{Type: tea.KeyTab}, "Tab"},
		{tea.KeyMsg{Type: tea.KeyEnter}, "Enter"},
		{tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}, "Space"},
		{tea.KeyMsg{Type: tea.KeyBackspace}, "Backspace"},
		{tea.KeyMsg{Type: tea.KeyDelete}, "Delete"},
		{tea.KeyMsg{Type: tea.KeyHome}, "Home"},
		{tea.KeyMsg{Type: tea.KeyEnd}, "End"},
		{tea.KeyMsg{Type: tea.KeyPgUp}, "PgUp"},
		{tea.KeyMsg{Type: tea.KeyPgDown}, "PgDn"},
		{tea.KeyMsg{Type: tea.KeyUp}, "↑"},
		{tea.KeyMsg{Type: tea.KeyDown}, "↓"},
		{tea.KeyMsg{Type: tea.KeyLeft}, "←"},
		{tea.KeyMsg{Type: tea.KeyRight}, "→"},
		{tea.KeyMsg{Type: tea.KeyCtrlC}, "Ctrl-C"},
		{tea.KeyMsg{Type: tea.KeyCtrlU}, "Ctrl-U"},
		{tea.KeyMsg{Type: tea.KeyShiftTab}, "Shift-Tab"},
		{tea.KeyMsg{Type: tea.KeyEsc, Alt: true}, "Alt-Esc"},
		{runes("t", true), "Alt-t"},
		{runes("z", true), "Alt-z"},
		{runes("q", false), "q"},
		{runes("A", false), "A"},
	} {
		if got := KeyName(c.msg); got != c.want {
			t.Errorf("KeyName(%v) = %q, want %q", c.msg, got, c.want)
		}
	}
	// Several keys doing one thing are joined with |, a range with –, never /.
	ta := newTestApp(t)
	if got := ta.covBFooter(); !strings.Contains(got, "Tab|1–3:panels") {
		t.Errorf("footer: %q", got)
	}
	ta.press("?")
	ref := StripStyles(strings.Join(ta.topView().(*noteView).lines, "\n"))
	for _, want := range []string{"q|Ctrl-C", "1–3", "j|k"} {
		if !strings.Contains(ref, want) {
			t.Errorf("the key reference lacks %q", want)
		}
	}
}

func TestM5WritingByPosition(t *testing.T) {
	ta := newTestApp(t)
	// hint and footer: key:words, no space around the colon, one between pairs.
	if got := ta.covBFooter(); got != "Space:menu ?:help Tab|1–3:panels q:quit" {
		t.Errorf("footer: %q", got)
	}
	f := ta.Frame()
	if got := covBText(f[len(f)-2], 0, 40); !strings.HasPrefix(got, "┗━ Enter:logs m:mark ━") {
		t.Errorf("panel hint: %q", got)
	}
	// The key reference: two columns, the keys bare.
	ta.press("?")
	descAt := map[string]int{}
	for _, l := range ta.topView().(*noteView).lines {
		plain := StripStyles(l)
		fields := strings.Fields(plain)
		if len(fields) < 2 || (fields[0] != "x" && fields[0] != "Space") {
			continue
		}
		if strings.ContainsAny(fields[0], "[]:") {
			t.Errorf("a key with brackets or a colon: %q", plain)
		}
		descAt[fields[0]] = strings.Index(plain, fields[1])
	}
	if len(descAt) != 2 || descAt["x"] != descAt["Space"] {
		t.Errorf("the descriptions are one column: %v", descAt)
	}
	// Sentences: keys in square brackets, as written.
	ta.press("Esc", "D", "Tab")
	if tv, ok := ta.toast.view.(*toastView); !ok || !strings.Contains(tv.text, "[Esc]") {
		t.Errorf("the lib's own sentence names its key in brackets: %+v", ta.toast.view)
	}
	ta.press("Esc", "Esc")
	ta.list.SetRows(nil)
	ta.list.EmptyFact, ta.list.EmptyHint = "No pods", "Press [A] or [Space]"
	if !strings.Contains(ta.screenText(), "Press [A] or [Space]") {
		t.Errorf("the empty state's hint is not as written:\n%s", ta.screenText())
	}
}

func TestM5HintColors(t *testing.T) {
	ta := newTestApp(t)
	f := ta.Frame()
	foot := f[len(f)-1]
	for _, want := range []string{
		Fg(Blue).Render("Space") + Fg(Overlay0).Render(":menu"),
		Fg(Blue).Render("Tab") + Fg(Overlay0).Render("|") + Fg(Blue).Render("1–3") + Fg(Overlay0).Render(":panels"),
	} {
		if !strings.Contains(foot, want) {
			t.Errorf("footer: keys in Blue, colon and words in Overlay0: %q", foot)
		}
	}
	if !strings.Contains(f[len(f)-2], Fg(Blue).Render("Enter")+Fg(Overlay0).Render(":logs")) {
		t.Errorf("panel hint: %q", f[len(f)-2])
	}
	ta.press("Space")
	l := ta.top()
	if !strings.Contains(ta.Frame()[l.y+l.h-1], Fg(Blue).Render("Enter")+Fg(Overlay0).Render(":run")) {
		t.Error("popup hint: the key in Blue, the words in Overlay0")
	}
	ta.press("?")
	ref := strings.Join(ta.topView().(*noteView).lines, "\n")
	if !strings.Contains(ref, Fg(Blue).Render("x")) || !strings.Contains(ref, Fg(Text).Render("Delete — this pod")) {
		t.Errorf("key reference: the key in Blue, what it does in Text:\n%q", ref)
	}
}

// covBDisabledApp is the test app with a disabled item operation Zap and a
// hidden one Yank on the pods.
func covBDisabledApp(t *testing.T) *testApp {
	ta := newTestApp(t)
	ta.pods.Actions = append(ta.pods.Actions,
		Action{Key: "Z", Label: "Zap", Desc: "this pod", Scope: ItemScope, Disabled: func() bool { return true }, Do: ta.record("zap")},
		Action{Key: "Y", Label: "Yank", Desc: "this pod", Scope: ItemScope, Hidden: func() bool { return true }, Do: ta.record("yank")})
	return ta
}

func TestM6DisabledRow(t *testing.T) {
	ta := covBDisabledApp(t)
	ta.press("Z", "Y")
	if len(ta.ran) != 0 {
		t.Errorf("the hotkey of a disabled or hidden action ran: %v", ta.ran)
	}
	ta.press("Space")
	if strings.Contains(ta.screenText(), "Yank") {
		t.Error("an action whose object does not exist has a row")
	}
	row := ta.rowWith("[Z]ap")
	if !strings.Contains(row, "\x1b[38;2;"+covBRGB(Surface2)+"m[Z]ap") || !strings.Contains(StripStyles(row), "[Z]ap") ||
		!strings.Contains(StripStyles(row), "this pod") {
		t.Errorf("a disabled row is Surface2 with its description: %q", row)
	}
	m := ta.topView().(*menuView)
	for range len(m.rows) {
		ta.press("j")
		if m.rows[m.cursor].label == "[Z]ap" {
			t.Fatal("the cursor stopped on a disabled row")
		}
	}
	ta.press("Z")
	if len(ta.ran) != 0 || ta.topView() != m {
		t.Error("the disabled row's hotkey did something in the menu")
	}
}

func TestM6KeyReferenceDisabled(t *testing.T) {
	ta := covBDisabledApp(t)
	ta.press("?")
	ref := strings.Join(ta.topView().(*noteView).lines, "\n")
	if strings.Contains(ref, "Yank") {
		t.Error("the key reference lists an action whose object does not exist")
	}
	for _, want := range []string{"Zap — this pod", "Panels — switch to this screen"} {
		if !strings.Contains(ref, Fg(Surface2).Render(want)) {
			t.Errorf("%q is not listed in Surface2", want)
		}
	}
}

func TestM7PanelEntriesRespond(t *testing.T) {
	ta := newTestApp(t)
	for _, keys := range [][]string{{"2"}, {"O"}} { // LOG: no item, no operation; a screen of one empty panel
		ta.press(keys...)
		ta.press("Space")
		m, ok := ta.topView().(*menuView)
		if !ok || m.rows[len(m.rows)-1].label != "Global operation" {
			t.Fatalf("%v: Space opened %T", keys, ta.topView())
		}
		ta.press("Esc", "?")
		if _, ok := ta.topView().(*noteView); !ok {
			t.Errorf("%v: ? opened %T", keys, ta.topView())
		}
		ta.press("Esc")
	}
	for name, open := range map[string]func(){
		"menu":    func() { ta.Menu("Sort", MenuSection{Actions: []Action{{Label: "Name"}}}) },
		"confirm": func() { ta.Confirm(Ask{Title: "Delete", Question: "Delete it?", Verb: "delete"}, nil) },
		"viewer":  func() { ta.Viewer(&Viewer{Title: "YAML", Lines: []string{"a: 1"}}) },
		"form":    func() { ta.Form(testForm(nil)) },
	} {
		open()
		ta.settle()
		below := ta.topView()
		ta.press("?")
		if n, ok := ta.topView().(*noteView); !ok || popupView(n) == below {
			t.Errorf("%s: ? did not open its key reference: %T", name, ta.topView())
		}
		ta.covBCloseAll()
	}
}

func TestM8GroupedMenuOrder(t *testing.T) {
	ta := newTestApp(t)
	ta.Menu("Open with",
		MenuSection{Title: "default", Actions: []Action{{Label: "Default app", Desc: "for .yaml files"}}},
		MenuSection{Title: "editors", Actions: []Action{{Label: "vim", Desc: "in a terminal"}, {Label: "code", Desc: "an editor"}}},
		MenuSection{Title: "viewers", Actions: []Action{{Label: "less", Desc: "page through it"}}})
	ta.settle()
	m := ta.topView().(*menuView)
	if m.rows[m.cursor].label != "Default app" {
		t.Errorf("the cursor starts in the first group: %q", m.rows[m.cursor].label)
	}
	// Every group has its title, the first too.
	text := ta.screenText()
	order := []string{glyphHexagon + " DEFAULT", "Default app", "╌", glyphHexagon + " EDITORS", "vim", "code", glyphHexagon + " VIEWERS", "less"}
	at := -1
	for _, s := range order {
		i := strings.Index(text, s)
		if i <= at {
			t.Fatalf("%q is not after what comes before it:\n%s", s, text)
		}
		at = i
	}
	ta.covBCloseAll()
	ta.Menu("Sort by", MenuSection{Actions: []Action{{Label: "Name"}, {Label: "Age"}}})
	ta.settle()
	l := ta.top()
	for y := l.y; y < l.y+l.h; y++ {
		if row := ta.covBPopupRow(y); strings.Contains(row, glyphHexagon) || strings.Contains(row, "╌") {
			t.Errorf("a menu with no groups has a title or separator: %q", row)
		}
	}
}

func TestM9ActiveHintBright(t *testing.T) {
	ta := newTestApp(t)
	enter := func(from, to int) string {
		f := ta.Frame()
		return covBAt(f[len(f)-2], from, to, "Enter").fg
	}
	if enter(0, 40) != covBRGB(Blue) || enter(80, 120) != covBDim(Blue) {
		t.Errorf("focus on [1]: its Enter %s should be bright, [3]'s %s dim", enter(0, 40), enter(80, 120))
	}
	ta.press("3")
	if enter(80, 120) != covBRGB(Blue) || enter(0, 40) != covBDim(Blue) {
		t.Errorf("focus on [3]: its Enter %s should be bright, [1]'s %s dim", enter(80, 120), enter(0, 40))
	}
}

// --- L ----------------------------------------------------------------------

func TestL1Core80x40(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(80, 40)
	text := ta.screenText()
	for _, want := range []string{"[1] PODS", "[2] LOG", "[3] SETTINGS", "Space:menu ?:help Tab|1–3:panels q:quit"} {
		if !strings.Contains(text, want) {
			t.Errorf("at 80 × 40 %q is not whole", want)
		}
	}
	for _, keys := range [][]string{{"Space"}, {"Space", "G", "Enter"}, {"?"}, {"x"}, {"q"}} {
		ta.press(keys...)
		l := ta.top()
		if l == nil {
			t.Fatalf("%v opened nothing", keys)
		}
		if l.x < 0 || l.x+l.w > 80 || l.y < 1 || l.y+l.h > 39 {
			t.Errorf("%v: %dx%d at %d,%d is not on the screen with the top row and footer", keys, l.w, l.h, l.x, l.y)
		}
		for y := l.y; y < l.y+l.h; y++ {
			if row := ta.covBPopupRow(y); strings.Contains(row, "…") {
				t.Errorf("%v: cut at 80 × 40: %q", keys, row)
			}
		}
		ta.covBCloseAll()
	}
}

func TestScreenNarrow(t *testing.T) {
	ta := newTestApp(t)
	ta.Screen().NarrowBelow = 70
	ta.resize(60, 30)
	text := ta.screenText()
	if !strings.Contains(text, "[1] PODS") || strings.Contains(text, "[2] LOG") {
		t.Errorf("narrow: only the focused panel:\n%s", text)
	}
	if top := StripStyles(ta.Frame()[2]); !strings.HasPrefix(top, "┏") || !strings.HasSuffix(top, "┓") {
		t.Errorf("the panel takes the whole width: %q", top)
	}
	ta.press("Tab")
	if text := ta.screenText(); !strings.Contains(text, "[2] LOG") || strings.Contains(text, "[1] PODS") {
		t.Errorf("Tab shows the next panel, one at a time:\n%s", text)
	}
	ta.Screen().NarrowBelow = 100
	ta.resize(90, 30)
	if text := ta.screenText(); !strings.Contains(text, "[1] PODS") || !strings.Contains(text, "[3] SETTINGS") {
		t.Error("the threshold is never above 80: at 90 every panel is drawn")
	}
}

func TestL2FixedWidthChrome(t *testing.T) {
	ta := newTestApp(t)
	ta.Status = func(w int) string { return Truncate("copying  4%", w) }
	chips := StripStyles(ta.Frame()[0])
	ta.press("O")
	if got := StripStyles(ta.Frame()[0]); got != chips {
		t.Errorf("the chips moved with the chosen screen:\n%q\n%q", chips, got)
	}
	ta.press("P")
	// A panel's title keeps its place across its tabs.
	where := func() (int, int) {
		cells := covBCells(ta.Frame()[2])
		return covBFind(cells, "PODS"), covBFind(cells, "EVENTS")
	}
	p0, e0 := where()
	if p0 < 0 || e0 < 0 {
		t.Fatalf("no PODS or EVENTS on the panel's top border: %d %d", p0, e0)
	}
	ta.press("l")
	if p1, e1 := where(); p1 != p0 || e1 != e0 {
		t.Errorf("the capsules moved when the tab changed: PODS %d→%d, EVENTS %d→%d", p0, p1, e0, e1)
	}
	ta.press("h")
	// Marking a row keeps its text in place.
	ta.press("j")
	before := covBFind(covBCells(ta.rowWith("pod-01")), "pod-01")
	if before < 0 {
		t.Fatal("no pod-01 row")
	}
	ta.press("m")
	if len(ta.list.Marked()) != 1 {
		t.Fatal("m marked nothing")
	}
	if after := covBFind(covBCells(ta.rowWith("pod-01")), "pod-01"); after != before {
		t.Errorf("the row moved when marked: %d→%d", before, after)
	}
}

func TestL3ChromeRowsFixed(t *testing.T) {
	ta := newTestApp(t)
	ta.Status = func(w int) string { return Truncate("context: prod  namespace: kube-system  2 errors", w) }
	for _, w := range []int{120, 60, 30} {
		ta.resize(w, 20)
		f := ta.Frame()
		if len(f) != 20 {
			t.Fatalf("%d wide: %d rows", w, len(f))
		}
		if Width(f[0]) != w || StripStyles(f[1]) != strings.Repeat("━", w) {
			t.Errorf("%d wide: the chips and status take one row, the divider one: %q", w, StripStyles(f[1]))
		}
		if top := StripStyles(f[2]); !strings.HasPrefix(top, "┏") {
			t.Errorf("%d wide: the panels do not start on the third row: %q", w, top)
		}
		full := "Space:menu ?:help Tab|1–3:panels q:quit"
		got := ta.covBFooter()
		if !strings.HasPrefix(full, got) || (got != full && full[len(got)] != ' ') {
			t.Errorf("%d wide: the footer drops whole pairs from its end: %q", w, got)
		}
	}
	if got := ta.covBFooter(); got != "Space:menu ?:help" {
		t.Errorf("30 wide: %q", got)
	}
}

func TestL5FocusNotColorOnly(t *testing.T) {
	ta := newTestApp(t)
	corner := func(x int) string { return covBCells(ta.Frame()[2])[x].text }
	if corner(0) != "┏" || corner(40) != "┌" || corner(80) != "┌" {
		t.Errorf("focus is a heavy frame, the rest light: %q %q %q", corner(0), corner(40), corner(80))
	}
	ta.press("D")
	if corner(0) != "┏" || !strings.Contains(StripStyles(ta.Frame()[2]), "┫Drag┣") {
		t.Errorf("in a mode the focused frame stays heavy: %q", StripStyles(ta.Frame()[2]))
	}
	if c := covBCells(ta.Frame()[2])[0]; c.fg != covBRGB(Yellow) {
		t.Errorf("the mode turns the frame Yellow: %s", c.fg)
	}
}

func TestL5NoShift(t *testing.T) {
	ta := newTestApp(t)
	light := strings.NewReplacer("┏", "┌", "┓", "┐", "┗", "└", "┛", "┘", "━", "─", "┃", "│", "┫", "┤", "┣", "├")
	plain := func() []string {
		var out []string
		for _, l := range ta.Frame() {
			out = append(out, light.Replace(StripStyles(l)))
		}
		return out
	}
	before := plain()
	for _, k := range []string{"Tab", "Tab"} {
		ta.press(k)
		for i, l := range plain() {
			if l != before[i] {
				t.Errorf("row %d changed with the focus:\n%q\n%q", i, before[i], l)
			}
		}
	}
}

// --- F ----------------------------------------------------------------------

func TestConfirmScrollsContent(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(80, 16)
	var details []string
	for i := range 30 {
		details = append(details, fmt.Sprintf("detail %02d", i+1))
	}
	ta.Confirm(Ask{Title: "Connect", Question: "Connect to web-01?", Verb: "connect", Details: details}, nil)
	ta.settle()
	l := ta.top()
	if got := ta.covBPopupRow(l.y + l.h - 3); !strings.Contains(got, "Connect to web-01?") {
		t.Errorf("the question is the last row: %q", got)
	}
	if got := ta.covBPopupRow(l.y + l.h - 1); !strings.Contains(got, "1-8 of 30") {
		t.Errorf("the details scroll: %q", got)
	}
	ta.press("j", "j")
	if got := ta.covBPopupRow(l.y + l.h - 1); !strings.Contains(got, "3-10 of 30") {
		t.Errorf("j scrolls the details: %q", got)
	}
	if got := ta.covBPopupRow(l.y + 2); !strings.Contains(got, "detail 03") {
		t.Errorf("the first detail in sight: %q", got)
	}
	if got := ta.covBPopupRow(l.y + l.h - 3); !strings.Contains(got, "Connect to web-01?") {
		t.Errorf("the question stays: %q", got)
	}
}

func TestFinderPhases(t *testing.T) {
	ta := newTestApp(t)
	f := &Finder{Title: "Search", Items: []FinderItem{{ID: "a", Text: "alpha"}, {ID: "b", Text: "beta"}},
		Preview: func(FinderItem) (string, []string) { return "Preview", []string{"line one"} }}
	ta.Find(f)
	ta.settle()
	fv := ta.topView().(*finderView)
	l := ta.top()
	angle := func() string { return covBCells(ta.Frame()[l.y+2])[l.x+2].fg }
	cursor := func() string { return covBCells(ta.Frame()[l.y+4])[l.x+1].bg }
	if !fv.typing() || angle() != covBRGB(LayerColor(1)) || cursor() != covBDim(LayerColor(1)) {
		t.Errorf("typing: the typing row lit (%s), the list's cursor dim (%s)", angle(), cursor())
	}
	ta.press("Tab")
	if fv.typing() || angle() != covBRGB(Overlay0) || cursor() != covBRGB(LayerColor(1)) {
		t.Errorf("on the list: the typing row grey (%s), the cursor lit (%s)", angle(), cursor())
	}
	ta.press("Tab")
	if !fv.typing() {
		t.Error("Tab goes back to the typing row: the preview takes no focus")
	}
	ta.press("Tab", "Esc")
	if ta.top() != nil {
		t.Error("Esc on the list closes the whole finder")
	}
	ta.Find(f)
	ta.settle()
	ta.press("Esc")
	if ta.top() != nil {
		t.Error("Esc on the typing row closes the whole finder")
	}
}

func TestToastNotALayer(t *testing.T) {
	ta := newTestApp(t)
	before := ta.Frame()
	ta.Toast(Info, "Copied 3 paths")
	ta.settle()
	after := ta.Frame()
	for _, i := range []int{0, 2, len(before) - 1} {
		if after[i] != before[i] {
			t.Errorf("a toast changed row %d: it dims nothing", i)
		}
	}
	ta.press("Space") // keys go through the toast
	if len(ta.stack) != 1 || ta.toast == nil {
		t.Fatalf("Space under a toast: %d layers, toast %v", len(ta.stack), ta.toast != nil)
	}
	l := ta.top()
	if c := covBCells(ta.Frame()[l.y])[l.x]; c.fg != covBRGB(LayerColor(1)) {
		t.Errorf("the popup under a toast is not dimmed and is layer 1: %s", c.fg)
	}
	ta.press("Esc")
	if ta.toast != nil || ta.top() == nil {
		t.Error("Esc closes the toast first, and only it")
	}
}

func TestF3EscImmediate(t *testing.T) {
	ta := newTestApp(t)
	ta.press("Space")
	l := ta.top()
	ta.Update(keyMsg("Esc"))
	if !l.closing {
		t.Fatal("Esc did not start closing at once")
	}
	frames := 0
	for ta.anim {
		ta.Update(animMsg{})
		frames++
	}
	if frames != openFrames || len(ta.stack) != 0 {
		t.Errorf("closed in %d frames, want %d", frames, openFrames)
	}
	ta.Toast(Warning, "Disk almost full")
	ta.settle()
	ta.Update(keyMsg("Esc"))
	if ta.toast == nil || !ta.toast.closing {
		t.Error("Esc did not start closing the toast before its time")
	}
	ta.settle()
	ta.press("Space", "G", "Enter")
	ta.Update(keyMsg("Esc"))
	ta.Update(keyMsg("Esc"))
	if len(ta.stack) != 2 || !ta.stack[0].closing || !ta.stack[1].closing {
		t.Error("two Escs at once close two layers")
	}
}

func TestF3ClosingIgnoresKeys(t *testing.T) {
	ta := newTestApp(t)
	ta.Menu("Sort by", MenuSection{Actions: []Action{{Key: "n", Label: "Name", Do: ta.record("sort")}, {Label: "Age"}}})
	ta.settle()
	m := ta.topView().(*menuView)
	cur := m.cursor
	ta.Update(keyMsg("Esc"))
	ta.Update(keyMsg("j"))
	ta.Update(keyMsg("n"))
	ta.Update(keyMsg("Enter")) // the panel's Enter: logs
	if m.cursor != cur || slices.Contains(ta.ran, "sort") {
		t.Errorf("a closing menu took keys: cursor %d→%d, ran %v", cur, m.cursor, ta.ran)
	}
}

func TestF3Exceptions(t *testing.T) {
	// In a PTY Esc is the program's; a toast waits for its time.
	tp, _, sent := ptyApp(t)
	tp.Toast(Info, "copied")
	tp.settle()
	tp.press("Esc")
	if tp.toast == nil || tp.toast.closing || sent.String() != "\x1b" {
		t.Errorf("in a PTY Esc went to the toast: sent %q", sent.String())
	}
	// A textarea writing: Esc goes to the moving state.
	ta := newTestApp(t)
	ta.Textarea(Textarea{Title: "Comment", Old: "one"})
	ta.settle()
	tv := ta.topView().(*textareaView)
	ta.press("!", "Esc")
	if ta.topView() != tv || tv.typing() {
		t.Fatal("Esc while writing closed the textarea")
	}
	// Changed, Esc asks first; unchanged, it closes.
	ta.press("Esc")
	if c, ok := ta.topView().(*confirmView); !ok || !strings.Contains(c.ask.Question, "Discard") {
		t.Fatalf("Esc on a changed textarea: %T", ta.topView())
	}
	ta.covBCloseAll()
	ta.Textarea(Textarea{Title: "Comment", Old: "one"})
	ta.settle()
	ta.press("Esc", "Esc")
	if ta.top() != nil {
		t.Error("Esc on an unchanged textarea asks")
	}
	// A changed form asks first.
	ta.Form(testForm(nil))
	ta.settle()
	ta.press("Enter", "w", "Enter", "Esc")
	if _, ok := ta.topView().(*confirmView); !ok {
		t.Errorf("Esc on a changed form: %T", ta.topView())
	}
}

func TestF4KeepSource(t *testing.T) {
	ta := newTestApp(t)
	ta.press("Space", "j")
	menu := ta.topView().(*menuView)
	cur := menu.cursor
	ta.press("x") // Delete asks: a confirm over the menu
	if _, ok := ta.topView().(*confirmView); !ok || len(ta.stack) != 2 {
		t.Fatalf("x: %T, %d layers", ta.topView(), len(ta.stack))
	}
	ta.press("Esc")
	if ta.topView() != menu || menu.cursor != cur {
		t.Errorf("cancelling the confirm goes back to the menu as it was: %T, cursor %d→%d", ta.topView(), cur, menu.cursor)
	}
	ta.press("G", "Enter", "?")
	if len(ta.stack) != 3 {
		t.Fatalf("want three layers, have %d", len(ta.stack))
	}
	global := ta.stack[1].view
	ta.press("Esc")
	if len(ta.stack) != 2 || ta.topView() != global || ta.stack[0].view != menu {
		t.Error("Esc closes only the top layer")
	}
	ta.press("Esc")
	if len(ta.stack) != 1 || ta.topView() != menu {
		t.Error("Esc again goes back to the Space menu")
	}
}

type covBPing struct{}

func TestF5ErrorNonBlocking(t *testing.T) {
	ta := newTestApp(t)
	pings := 0
	ta.OnMsg = func(msg tea.Msg) tea.Cmd {
		if _, ok := msg.(covBPing); ok {
			pings++
		}
		return nil
	}
	ta.Toast(Error, "Couldn't reach the cluster")
	if ta.toast == nil {
		t.Fatal("the error toast is not there at once")
	}
	ta.settle()
	if !strings.Contains(ta.screenText(), "Couldn't reach the cluster") {
		t.Error("the error is not on the screen")
	}
	ta.press("3")
	if ta.Screen().focus != 2 {
		t.Error("an error toast blocked the app")
	}
	ta.press("Esc")
	if ta.toast != nil {
		t.Error("Esc did not close the error toast")
	}
	ta.Error("Save failed", "hosts.yaml changed on disk since it was opened.")
	ta.settle()
	if n, ok := ta.topView().(*noteView); !ok || n.title != "SAVE FAILED" {
		t.Fatalf("no error popup: %T", ta.topView())
	}
	ta.Update(covBPing{})
	if pings != 1 {
		t.Error("the app stopped getting its messages under an error popup")
	}
	ta.press("Esc")
	if ta.top() != nil {
		t.Error("Esc did not close the error popup")
	}
}

func TestF7WidthKnown(t *testing.T) {
	ta := newTestApp(t)
	check := func(what string, want int) {
		t.Helper()
		ta.settle()
		l := ta.top()
		if l.w != want {
			t.Errorf("%s: %d wide, want %d", what, l.w, want)
		}
		if l.x != (ta.w-l.w)/2 {
			t.Errorf("%s: at column %d, not centred", what, l.x)
		}
		ta.covBCloseAll()
	}
	ta.Password(PasswordInput{Title: "PIN", Prompt: "Up to 12 digits", Unlock: true, MaxLen: 12})
	check("a PIN: its hint + 6", 43)
	ta.Confirm(Ask{Title: "Delete", Question: "Delete it?", Verb: "delete"}, nil)
	check("a short confirm: its hint Enter:delete Esc:cancel + 6", 29)
	q := "Delete web-a8k2 and every volume it has mounted?"
	ta.Confirm(Ask{Title: "Delete", Question: q, Verb: "delete"}, nil)
	check("a confirm: its widest row + 4", Width(q)+4)
	ta.Confirm(Ask{Title: "Delete", Question: strings.Repeat("x", 300), Verb: "delete"}, nil)
	check("a confirm wider than the screen", 118)
	menu := func() {
		ta.Menu("Sort by", MenuSection{Actions: []Action{{Label: "Name", Desc: "by the pod's name, A to Z"}}})
	}
	menu()
	check("a menu: its row + 4", Width("Name")+2+Width("by the pod's name, A to Z")+4)
	ta.press("?")
	ta.settle()
	widest := 0
	for _, l := range ta.topView().(*noteView).lines {
		widest = max(widest, Width(l))
	}
	check("a key reference", widest+4)
	ta.Number(NumberInput{Title: "Port", Prompt: "1-65535", Min: 1, Max: 65535})
	ta.settle()
	w0 := ta.top().w
	ta.press("6", "5", "5", "3", "5")
	if ta.top().w != w0 {
		t.Errorf("the width changed while open: %d → %d", w0, ta.top().w)
	}
	ta.covBCloseAll()
	// A full-row cursor: the right padding is as wide as the arrow.
	withWidths(t, 1, 2)
	menu()
	check("a menu, the arrow two cells", Width("Name")+2+Width("by the pod's name, A to Z")+5)
}

func TestF7WidthUnknown(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(100, 30)
	for name, open := range map[string]func(){
		"text":     func() { ta.Text(TextInput{Title: "Rename", Prompt: "New name"}) },
		"viewer":   func() { ta.Viewer(&Viewer{Title: "YAML", Lines: []string{"a: 1"}}) },
		"form":     func() { ta.Form(testForm(nil)) },
		"finder":   func() { ta.Find(testFinder()) },
		"textarea": func() { ta.Textarea(Textarea{Title: "Comment"}) },
	} {
		open()
		ta.settle()
		if l := ta.top(); l.w != 98 || l.x != 1 {
			t.Errorf("%s: %d wide at %d, want 98 at 1", name, l.w, l.x)
		}
		ta.covBCloseAll()
	}
}

func TestF7WidthCap120(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(200, 50)
	for name, open := range map[string]func(){
		"text":    func() { ta.Text(TextInput{Title: "URL", Prompt: "Address"}) },
		"viewer":  func() { ta.Viewer(&Viewer{Title: "Log", Lines: []string{"one"}}) },
		"confirm": func() { ta.Confirm(Ask{Title: "Wide", Question: strings.Repeat("word ", 60), Verb: "go"}, nil) },
	} {
		open()
		ta.settle()
		if l := ta.top(); l.w != 120 || l.x != 40 {
			t.Errorf("%s: %d wide at %d, want 120 at 40", name, l.w, l.x)
		}
		ta.covBCloseAll()
	}
}

func TestF7HeightFixed(t *testing.T) {
	ta := newTestApp(t)
	ta.press("Space")
	m := ta.topView().(*menuView)
	if l := ta.top(); l.h != len(m.rows)+4 {
		t.Errorf("a menu is as tall as its rows: %d, want %d", l.h, len(m.rows)+4)
	}
	ta.covBCloseAll()
	v := &Viewer{Title: "Log", Lines: []string{"one", "two"}}
	ta.Viewer(v)
	ta.settle()
	l := ta.top()
	h0 := l.h
	v.Lines = append(v.Lines, "three", "four", "five")
	ta.Update(spinMsg{})
	if l.h != h0 {
		t.Errorf("the height followed the content: %d → %d", h0, l.h)
	}
	if !strings.Contains(ta.covBPopupRow(l.y+l.h-1), "1-2 of 5") {
		t.Errorf("the rest scrolls inside: %q", ta.covBPopupRow(l.y+l.h-1))
	}
}

func TestF7HeightMax(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(80, 30)
	var acts []Action
	for i := range 60 {
		acts = append(acts, Action{Label: fmt.Sprintf("row %d", i+1)})
	}
	ta.Menu("Long", MenuSection{Actions: acts})
	ta.settle()
	l := ta.top()
	if l.h != 28 || l.y != 1 {
		t.Errorf("a long menu is %d tall at row %d; want the screen's height − 2 at row 1", l.h, l.y)
	}
	if !strings.Contains(ta.covBPopupRow(l.y+l.h-1), "1 of 60") {
		t.Errorf("no scroll position: %q", ta.covBPopupRow(l.y+l.h-1))
	}
	ta.covBCloseAll()
	for name, open := range map[string]func(){
		"finder with a preview": func() { ta.Find(testFinder()) },
		"textarea":              func() { ta.Textarea(Textarea{Title: "Comment", Old: "one"}) },
	} {
		open()
		ta.settle()
		if l := ta.top(); l.h != 28 {
			t.Errorf("%s: %d tall, want the most, 28", name, l.h)
		}
		ta.covBCloseAll()
	}
}

func TestF7LoadingIcon(t *testing.T) {
	ta := newTestApp(t)
	icon := func() (covBCell, int, []covBCell) {
		l := ta.top()
		cells := covBCells(ta.Frame()[l.y])
		for i, c := range cells {
			if slices.Contains(loadingFrames[:], c.text) {
				return c, i, cells
			}
		}
		return covBCell{}, -1, cells
	}
	v := &Viewer{Title: "Log", Loading: true}
	ta.Viewer(v)
	ta.settle()
	c, i, cells := icon()
	if i < 2 || cells[i-1].text != " " || cells[i-2].text != glyphRoundRight {
		t.Fatalf("no loading icon one space after the title: %d", i)
	}
	if c.fg != covBRGB(LayerColor(1)) || !c.bold {
		t.Errorf("the icon is the layer colour, bold: %+v", c)
	}
	v.Loading = false
	ta.Update(spinMsg{})
	if _, i, _ := icon(); i >= 0 {
		t.Error("the icon stays after loading")
	}
	ta.covBCloseAll()
	ta.Viewer(&Viewer{Title: "DevTools", Tabs: []string{"Console", "Network"}, Loading: true})
	ta.settle()
	if _, i, cells := icon(); i < 0 || covBFind(cells, "NETWORK") > i || cells[i-2].text != glyphHard {
		t.Errorf("with tabs the icon follows the whole chain: at %d", i)
	}
}

func TestF7Position(t *testing.T) {
	ta := newTestApp(t)
	for _, sz := range [][2]int{{120, 40}, {81, 31}, {60, 20}} {
		ta.resize(sz[0], sz[1])
		for _, keys := range [][]string{{"Space"}, {"x"}, {"?"}} {
			ta.press(keys...)
			l := ta.top()
			above, below := l.y, ta.h-l.y-l.h
			left, right := l.x, ta.w-l.x-l.w
			if above-below > 1 || below-above > 1 || left-right > 1 || right-left > 1 {
				t.Errorf("%v at %dx%d: %d above, %d below, %d left, %d right", keys, sz[0], sz[1], above, below, left, right)
			}
			if c := covBCells(ta.Frame()[l.y])[l.x]; c.text != "┏" {
				t.Errorf("%v at %dx%d: the corner is %q", keys, sz[0], sz[1], c.text)
			}
			ta.covBCloseAll()
		}
	}
}

func TestF7ErrorRowReserved(t *testing.T) {
	ta := newTestApp(t)
	ta.Number(NumberInput{Title: "Port", Prompt: "1-65535", Min: 1, Max: 65535})
	ta.settle()
	l := ta.top()
	h0 := l.h
	errRow := func() string { return strings.Trim(ta.covBPopupRow(l.y+l.h-3), "┃ ") }
	if errRow() != "" {
		t.Errorf("the error row is blank with no error: %q", errRow())
	}
	ta.press("7", "0", "0", "0", "0", "Enter")
	if ta.top() != l || l.h != h0 {
		t.Fatalf("a wrong value changed the popup: %d → %d", h0, l.h)
	}
	if errRow() != "Port must be 1-65535" {
		t.Errorf("error row: %q", errRow())
	}
	if !strings.Contains(ta.covBPopupRow(l.y+l.h-5), "70000") || strings.Trim(ta.covBPopupRow(l.y+l.h-4), "┃ ") != "" {
		t.Error("the error row is not one blank row under the value")
	}
	if c := covBAt(ta.Frame()[l.y+l.h-3], l.x, l.x+l.w, "Port"); c.fg != covBRGB(Red) {
		t.Errorf("the error is not Red: %s", c.fg)
	}
}

func TestF8DimAllButTop(t *testing.T) {
	ta := newTestApp(t)
	_, _, l3 := ta.covBThreeLayers(t)
	f := ta.Frame()
	base := DimLines(ta.base())
	for _, y := range []int{0, 2, len(f) - 1} {
		if f[y] != base[y] {
			t.Errorf("row %d of the screen under the popups is not dimmed", y)
		}
	}
	lit := map[string]bool{covBRGB(LayerColor(1)): true, covBRGB(LayerColor(2)): true}
	for y, row := range f {
		for x, c := range covBCells(row) {
			if lit[c.fg] || lit[c.bg] {
				t.Fatalf("an undimmed lower layer's colour at %d,%d", x, y)
			}
		}
	}
	if c := covBCells(f[l3.y])[l3.x]; c.fg != covBRGB(LayerColor(3)) {
		t.Errorf("the top popup is not bright: %s", c.fg)
	}
}

func TestF8DimStreamToo(t *testing.T) {
	ta := newTestApp(t)
	ta.Screen().Panels[1].Body.(*Pager).SetLines([]string{Fg(Red).Render("error: boom"), Fg(Peach).Render("warning: slow")})
	fg := func(s string) string { return covBAt(ta.rowWith(s), 40, 80, s).fg }
	if fg("error") != covBRGB(Red) {
		t.Fatalf("an unfocused stream is not dimmed (T2): %s", fg("error"))
	}
	ta.press("Space")
	if fg("error") != covBDim(Red) || fg("warning") != covBDim(Peach) {
		t.Errorf("under a popup the stream and its alert colours dim: %s, %s", fg("error"), fg("warning"))
	}
}

func TestDimEveryColor(t *testing.T) {
	for _, c := range []struct{ in, want RGB }{
		{Text.RGB, RGB{109, 113, 135}},
		{Red.RGB, RGB{126, 79, 101}},
		{RGB{255, 255, 255}, RGB{131, 131, 140}},
		{RGB{0, 0, 0}, RGB{0, 0, 0}}, // never lighter
		{Base.RGB, Base.RGB},
	} {
		if got := DimRGB(c.in); got != c.want {
			t.Errorf("dim(%v) = %v, want %v", c.in, got, c.want)
		}
	}
	rgb := func(c RGB) string { return fmt.Sprintf("%d;%d;%d", c[0], c[1], c[2]) }
	line := "plain " + Fg(Red).Render("red") + " " + Style{Fg: Base, Bg: Blue, Bold: true}.Render("cap") +
		" \x1b[38;5;196mx\x1b[0m \x1b[31;42my\x1b[0m"
	dimmed := Dim(line)
	if StripStyles(dimmed) != StripStyles(line) {
		t.Errorf("the text changed: %q", StripStyles(dimmed))
	}
	cells := covBCells(dimmed)
	at := func(s string) covBCell { return cells[covBFind(cells, s)] }
	for name, c := range map[string]struct {
		got  covBCell
		want covBCell
	}{
		"no colour":  {at("plain"), covBCell{fg: rgb(DimRGB(Text.RGB))}},
		"24-bit":     {at("red"), covBCell{fg: rgb(DimRGB(Red.RGB))}},
		"background": {at("cap"), covBCell{fg: rgb(DimRGB(Base.RGB)), bg: rgb(DimRGB(Blue.RGB)), bold: true}},
		"256":        {at("x"), covBCell{fg: rgb(DimRGB(color256(196)))}},
		"16":         {at("y"), covBCell{fg: rgb(DimRGB(palette16[1])), bg: rgb(DimRGB(palette16[2]))}},
	} {
		c.got.text = ""
		if c.got != c.want {
			t.Errorf("%s: %+v, want %+v", name, c.got, c.want)
		}
	}
	// On the screen: a capsule under a popup keeps its block, dimmed.
	ta := newTestApp(t)
	ta.press("Space")
	if c := covBAt(ta.Frame()[2], 0, 40, "PODS"); c.bg != covBDim(Blue) {
		t.Errorf("the panel's capsule lost its block: %+v", c)
	}
}

func TestF8DimOnce(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(200, 40)
	fg := func(s string, from, to int) string { return covBAt(ta.rowWith(s), from, to, s).fg }
	if fg("LAYOUT", 132, 200) != covBDim(Blue) || fg("AGE", 0, 66) != covBRGB(Blue) {
		t.Fatalf("before: unfocused %s, focused %s", fg("LAYOUT", 132, 200), fg("AGE", 0, 66))
	}
	ta.press("Space")
	if got := fg("LAYOUT", 132, 200); got != covBRGB(DimColor(DimColor(Blue))) {
		t.Errorf("an unfocused panel under a popup is dimmed once more: %s", got)
	}
	if got := fg("AGE", 0, 66); got != covBDim(Blue) {
		t.Errorf("the focused panel under a popup is dimmed once: %s", got)
	}
	ta.press("Esc")
	ta.Toast(Info, "copied")
	ta.settle()
	if fg("LAYOUT", 132, 200) != covBDim(Blue) || fg("AGE", 0, 66) != covBRGB(Blue) {
		t.Error("a toast dimmed the screen")
	}
}

func TestF8BorderKeepsLayerColor(t *testing.T) {
	ta := newTestApp(t)
	l1, l2, l3 := ta.covBThreeLayers(t)
	f := ta.Frame()
	for i, c := range []struct {
		l    *layer
		want string
	}{{l1, covBDim(LayerColor(1))}, {l2, covBDim(LayerColor(2))}, {l3, covBRGB(LayerColor(3))}} {
		if got := covBCells(f[c.l.y])[c.l.x]; got.fg != c.want || got.text != "┏" {
			t.Errorf("layer %d's frame: %q in %s, want %s", i+1, got.text, got.fg, c.want)
		}
	}
	if covBDim(LayerColor(1)) == covBDim(LayerColor(2)) {
		t.Error("two dimmed layers look the same")
	}
}

// TestF7WidthKnownUnlockContent is the part of TestF7WidthKnown the lib does
// not do: an unlock mask of n characters is 2n − 1 cells (layout/popup's PIN:
// 12 + 11 spaces + 4 = 27), so its box is 2n + 3 when the mask is the widest.
func TestF7WidthKnownUnlockContent(t *testing.T) {
	ta := newTestApp(t)
	ta.Password(PasswordInput{Title: "PIN", Prompt: "PIN", Unlock: true, MaxLen: 30})
	ta.settle()
	if w := ta.top().w; w != 30*2-1+4 {
		t.Errorf("a 30-character unlock mask: %d wide, want 63", w)
	}
}

// --- T ----------------------------------------------------------------------

func TestT1KeepOrClear(t *testing.T) {
	ta := newTestApp(t)
	// A short confirm over a menu: answered, the menu stays.
	ta.Menu("Sort by", MenuSection{Actions: []Action{{Label: "Name"}}})
	ta.settle()
	menu := ta.topView()
	ta.Confirm(Ask{Title: "Reset", Question: "Reset the order?", Verb: "reset"}, ta.record("reset"))
	ta.settle()
	ta.press("Enter")
	if ta.topView() != menu || strings.Join(ta.ran, ",") != "reset" {
		t.Errorf("after the confirm: %T, ran %v", ta.topView(), ta.ran)
	}
	ta.covBCloseAll()
	// A value chosen goes back to the form, which stays.
	ta.Form(testForm(nil))
	ta.settle()
	form := ta.topView()
	ta.press("G", "k", "Enter", "j", "Enter")
	if ta.topView() != form {
		t.Errorf("after the select: %T", ta.topView())
	}
	ta.covBCloseAll()
	// Switching screens clears the stack.
	ta.press("Space", "G", "Enter", "O")
	if ta.cur != 1 || ta.top() != nil {
		t.Errorf("after switching screens: screen %d, top %T", ta.cur, ta.topView())
	}
	ta.press("P")
	// A shell opened from the Space menu clears the menu (Action.Clear).
	term := &Terminal{Title: "Shell", in: &strings.Builder{}, started: true}
	ta.pods.Actions = append(ta.pods.Actions, Action{Key: "Alt-t", Label: "term", Desc: "a shell", Scope: PanelScope, Clear: true,
		Do: func() tea.Cmd { return ta.open(&termView{t: term, a: ta.App}) }})
	ta.press("Space", "Alt-t")
	if _, ok := ta.topView().(*termView); !ok || len(ta.stack) != 1 {
		t.Errorf("after opening a terminal from the Space menu: %T, %d layers", ta.topView(), len(ta.stack))
	}
}

func TestT2UnfocusedDim(t *testing.T) {
	ta := newTestApp(t)
	fg := func(s string, from, to int) string { return covBAt(ta.rowWith(s), from, to, s).fg }
	if fg("AGE", 0, 40) != covBRGB(Blue) || fg("LAYOUT", 80, 120) != covBDim(Blue) {
		t.Errorf("focus on [1]: [1] %s bright, [3] %s dimmed", fg("AGE", 0, 40), fg("LAYOUT", 80, 120))
	}
	if c := covBCells(ta.Frame()[2])[80]; c.text != "┌" || c.fg != covBRGB(Surface2) {
		t.Errorf("an unfocused frame is light and Surface2: %q %s", c.text, c.fg)
	}
	ta.press("3")
	if fg("AGE", 0, 40) != covBDim(Blue) || fg("LAYOUT", 80, 120) != covBRGB(Blue) {
		t.Errorf("focus on [3]: [1] %s dimmed, [3] %s bright", fg("AGE", 0, 40), fg("LAYOUT", 80, 120))
	}
}

func TestT2StreamNotDimmed(t *testing.T) {
	ta := newTestApp(t)
	log := ta.Screen().Panels[1]
	log.Body.(*Pager).SetLines([]string{Fg(Green).Render("ok: synced"), "plain line"})
	fg := func(s string) string { return covBAt(ta.rowWith(s), 40, 80, s).fg }
	if plain := covBAt(ta.rowWith("plain line"), 40, 80, "plain"); fg("ok") != covBRGB(Green) || plain.text != "p" || plain.fg != "" {
		t.Errorf("an unfocused stream is not dimmed: %s, %+v", fg("ok"), plain)
	}
	log.Stream = false
	if fg("ok") != covBDim(Green) {
		t.Errorf("the same panel not streaming dims: %s", fg("ok"))
	}
	// A program's screen is live too.
	tb := newTestApp(t)
	term := &Terminal{Title: "Shell", in: &strings.Builder{}, started: true}
	tb.AddScreen(&Screen{Name: "Terminal", Key: "T", Panels: []*Panel{{Label: "Other", Body: &Pager{}}, {Label: "Shell", Body: term}}})
	tb.press("T")
	tb.Frame()
	term.mu.Lock()
	term.vt().write([]byte("\x1b[38;2;1;2;3mlive\x1b[0m"))
	term.mu.Unlock()
	if got := covBAt(tb.rowWith("live"), 60, 120, "live").fg; got != "1;2;3" {
		t.Errorf("an unfocused PTY is dimmed: %s", got)
	}
	// The hint of a stream without the focus dims all the same: it is not
	// what streams (Rules T2).
	log.Stream = true
	log.Actions = []Action{{Key: "Enter", Label: "View", Desc: "the log", Scope: PanelScope, Hint: "view", Do: func() tea.Cmd { return nil }}}
	if got := covBAt(ta.rowWith("Enter:view"), 40, 80, "Enter").fg; got != covBDim(Blue) {
		t.Errorf("an unfocused stream's hint is not dimmed: %s", got)
	}
}

// A menu that clears itself for the popup it opens (Rules T1, a drill-down):
// both animations play at once, and the closing menu is not a layer, so the
// new popup has the first layer's colour from its first frame
// (components/color).
func TestT1SourceClosesWhileTargetOpens(t *testing.T) {
	ta := newTestApp(t)
	ta.Menu("Open", MenuSection{Actions: []Action{{Key: "o", Label: "Open", Desc: "the YAML", Clear: true,
		Do: func() tea.Cmd { return ta.Viewer(testViewer()) }}}})
	ta.settle()
	ta.Update(keyMsg("o"))
	if len(ta.stack) != 2 || !ta.stack[0].closing || ta.stack[1].frame != 1 {
		t.Fatalf("the menu does not close while the viewer opens: %d layers", len(ta.stack))
	}
	for range openFrames - 1 {
		ta.Update(animMsg{})
	}
	if len(ta.stack) != 2 {
		t.Fatalf("the menu is gone before the viewer is open: %d layers", len(ta.stack))
	}
	if row := ta.rowWith("YAML"); !strings.Contains(row, Fg(LayerColor(1)).Render("┏")) {
		t.Errorf("the viewer is not in the first layer's colour: %q", row)
	}
	ta.Update(animMsg{})
	if len(ta.stack) != 1 {
		t.Errorf("%d layers once the menu has closed", len(ta.stack))
	}
}

// TestT1KeepByDefault: run from a menu, what an action opens keeps the menu
// under it, to come back to; an action that opens nothing closes it. A
// confirm before it is only a door (Rules T1, F4; components/dialog/menu).
func TestT1KeepByDefault(t *testing.T) {
	ta := newTestApp(t)
	opens := Action{Key: "y", Label: "YAML", Desc: "look at it", Do: func() tea.Cmd { return ta.Viewer(testViewer()) }}
	nothing := Action{Key: "r", Label: "Refresh", Desc: "again", Do: ta.record("refresh")}
	fails := Action{Key: "f", Label: "Fetch", Desc: "fails", Do: func() tea.Cmd { return ta.Error("Fetch", "connection refused") }}
	asks := Action{Key: "x", Label: "Delete", Desc: "it", Ask: &Ask{Title: "Delete", Question: "Delete it?", Verb: "delete"}, Do: ta.record("delete")}
	asksThenOpens := Action{Key: "e", Label: "Edit", Desc: "it", Ask: &Ask{Title: "Edit", Question: "Edit it?", Verb: "edit"},
		Do: func() tea.Cmd { return ta.Viewer(testViewer()) }}
	open := func() popupView {
		ta.covBCloseAll()
		ta.Menu("Pod", MenuSection{Actions: []Action{opens, nothing, fails, asks, asksThenOpens}})
		ta.settle()
		return ta.topView()
	}
	menu := open()
	ta.press("y")
	ta.settle()
	if _, ok := ta.topView().(*viewerView); !ok || len(ta.stack) != 2 {
		t.Fatalf("a viewer over the menu: %T, %d layers", ta.topView(), len(ta.stack))
	}
	ta.press("Esc")
	ta.settle()
	if ta.topView() != menu {
		t.Errorf("Esc goes back to the menu: %T", ta.topView())
	}
	ta.press("r")
	ta.settle()
	if ta.top() != nil || strings.Join(ta.ran, ",") != "refresh" {
		t.Errorf("opening nothing, the menu closes: %T, ran %v", ta.topView(), ta.ran)
	}
	menu = open()
	ta.press("f")
	ta.settle()
	ta.press("Enter")
	ta.settle()
	if ta.topView() != menu {
		t.Errorf("the error popup of a menu's action closes back to the menu: %T", ta.topView())
	}
	menu = open()
	ta.press("x")
	ta.settle()
	ta.press("Esc")
	ta.settle()
	if ta.topView() != menu {
		t.Errorf("Esc on its confirm goes back to the menu: %T", ta.topView())
	}
	ta.press("x")
	ta.settle()
	ta.press("Enter")
	ta.settle()
	if ta.top() != nil || !strings.Contains(strings.Join(ta.ran, ","), "delete") {
		t.Errorf("accepted, it opens nothing: everything closes: %T", ta.topView())
	}
	menu = open()
	ta.press("e")
	ta.settle()
	ta.press("Enter")
	ta.settle()
	if _, ok := ta.topView().(*viewerView); !ok || len(ta.stack) != 2 || ta.stack[0].view != menu {
		t.Errorf("accepted, the confirm goes and the menu stays under what it opens: %T, %d layers", ta.topView(), len(ta.stack))
	}
}

// TestF2LayersCloseTogether: several popups closing at once each play their
// own closing, at their own size and in their own layer's colour, all in the
// same nine frames (Rules F2; components/layout/popup).
func TestF2LayersCloseTogether(t *testing.T) {
	ta := newTestApp(t)
	ta.press("Space", "G", "Enter")
	ta.settle()
	ta.press("?")
	ta.settle()
	if len(ta.stack) != 3 {
		t.Fatalf("three layers: %d", len(ta.stack))
	}
	colors := []Color{LayerColor(1), LayerColor(2), LayerColor(3)}
	ta.CloseAll()
	for i, l := range ta.stack {
		if !l.closing || l.frame != openFrames || l.shade != colors[i] {
			t.Errorf("layer %d: closing %v at frame %d in %v, want its own colour %v", i+1, l.closing, l.frame, l.shade, colors[i])
		}
	}
	ta.Update(animMsg{})
	for i, l := range ta.stack {
		if l.frame != openFrames-1 {
			t.Errorf("layer %d at frame %d: not all together", i+1, l.frame)
		}
	}
	if frame := strings.Join(ta.Frame(), ""); !strings.Contains(frame, "38;2;"+covBRGB(colors[2])+"m") {
		t.Error("the third layer is not drawn in its own colour as it closes")
	}
	for range openFrames - 1 {
		ta.Update(animMsg{})
	}
	if len(ta.stack) != 0 {
		t.Errorf("%d layers after nine frames", len(ta.stack))
	}
}

// TestF7ShortScreenKeepsFooter: on a screen too short for a popup, it is
// drawn no taller than its height, so the chip row and the footer stay in
// sight (Rules F7; demou DM5).
func TestF7ShortScreenKeepsFooter(t *testing.T) {
	calls := inputOpeners(t)
	for name, open := range dialogOpeners() {
		calls[name] = open
	}
	for name, open := range calls {
		for sh := 5; sh <= 14; sh++ {
			ta := newTestApp(t)
			ta.resize(80, sh)
			before := ta.Frame()
			open(ta)
			ta.settle()
			after := ta.Frame()
			if name == "form" && sh >= 9 && !strings.Contains(StripStyles(strings.Join(after, "\n")), "Name     required") {
				t.Errorf("form at 80x%d: its first field is not in sight", sh)
			}
			rows := []int{0, sh - 1}
			if name == "terminal" {
				rows = rows[:1] // it covers the footer (Rules F7)
			}
			for _, row := range rows {
				if got, want := StripStyles(after[row]), StripStyles(before[row]); got != want {
					t.Errorf("%s at 80x%d: row %d is %q, want %q", name, sh, row, got, want)
				}
			}
		}
	}
}

// TestInputShortScreenKeepsTypingRow: an input too short for its prompt keeps
// the typing row in sight, and the error row while there is room for both
// (components/layout/popup; demou DM6).
func TestInputShortScreenKeepsTypingRow(t *testing.T) {
	for sh := 5; sh <= 10; sh++ {
		ta := newTestApp(t)
		ta.resize(80, sh)
		ta.Number(NumberInput{Title: "Port", Prompt: "Which port", Min: 1, Max: 10})
		ta.settle()
		ta.press("9", "9", "Enter")
		l := ta.top()
		text := strings.Join(plainLines(fitRows(l.view.draw(l.w, l.h, Blue), l.h)), "\n")
		if !strings.Contains(text, "99") {
			t.Errorf("80x%d: no typing row:\n%s", sh, text)
		}
		if l.h-4 >= 2 && !strings.Contains(text, "Port must be 1-10") {
			t.Errorf("80x%d: no error row:\n%s", sh, text)
		}
	}
}

// TestOnCancel: a confirm or an input closed with no answer, by Esc or by the
// popup under it closing, runs its OnCancel, once; an answer does not, even
// one that leaves the value as it was (Rules F6; webu WB3, sshu SS8).
func TestOnCancel(t *testing.T) {
	ta := newTestApp(t)
	cases := []struct {
		name string
		open func()
		keys []string
		want []string
	}{
		{"confirm, Esc", func() {
			ta.Confirm(Ask{Title: "Delete", Question: "Delete it?", Verb: "delete", OnCancel: ta.record("cancel")}, ta.record("yes"))
		}, []string{"Esc"}, []string{"cancel"}},
		{"confirm, Enter", func() {
			ta.Confirm(Ask{Title: "Delete", Question: "Delete it?", Verb: "delete", OnCancel: ta.record("cancel")}, ta.record("yes"))
		}, []string{"Enter"}, []string{"yes"}},
		{"text, Esc", func() { ta.Text(TextInput{Title: "Name", OnCancel: ta.record("cancel")}) }, []string{"a", "Esc"}, []string{"cancel"}},
		{"text, Enter unchanged", func() { ta.Text(TextInput{Title: "Name", Old: "a", OnCancel: ta.record("cancel")}) }, []string{"Enter"}, nil},
		{"password, Esc", func() { ta.Password(PasswordInput{Title: "Password", OnCancel: ta.record("cancel")}) }, []string{"Esc"}, []string{"cancel"}},
		{"number, Esc", func() { ta.Number(NumberInput{Title: "Port", OnCancel: ta.record("cancel")}) }, []string{"Esc"}, []string{"cancel"}},
		{"select, Esc", func() {
			ta.Select(Select{Title: "Pick", Options: []string{"a", "b"}, Selected: 0, OnCancel: ta.record("cancel")})
		}, []string{"Esc"}, []string{"cancel"}},
		{"select, Enter", func() {
			ta.Select(Select{Title: "Pick", Options: []string{"a", "b"}, Selected: 0, OnCancel: ta.record("cancel")})
		}, []string{"Enter"}, nil},
		{"textarea, discarded", func() { ta.Textarea(Textarea{Title: "Notes", Old: "a", OnCancel: ta.record("cancel")}) },
			[]string{"b", "Esc", "Esc", "Enter"}, []string{"cancel"}},
	}
	for _, c := range cases {
		ta.ran = nil
		c.open()
		ta.settle()
		ta.press(c.keys...)
		if !slices.Equal(ta.ran, c.want) || ta.top() != nil {
			t.Errorf("%s: ran %v, want %v; %T still open", c.name, ta.ran, c.want, ta.topView())
		}
		ta.covBCloseAll()
	}
	// The popup under it closes: an input over it is cancelled too.
	ta.ran = nil
	b := &testBody{w: 10, h: 3}
	ta.Popup(Popup{Title: "Draw", Body: b})
	ta.Text(TextInput{Title: "Name", OnCancel: ta.record("cancel")})
	ta.settle()
	ta.ClosePopup(b)
	if !slices.Equal(ta.ran, []string{"cancel"}) {
		t.Errorf("closed from under: ran %v", ta.ran)
	}
}

// TestF6AskFor: an action can make its question when it runs, after Hidden
// and Disabled, or ask nothing; from the Space menu its confirm opens over
// the menu, and Esc goes back to it (Rules F6, T1; kamonu KM12, filu FL15,
// sshu SS29, webu WB15).
func TestF6AskFor(t *testing.T) {
	ta := newTestApp(t)
	var order []string
	ask := true
	ta.pods.Actions = append(ta.pods.Actions, Action{Key: "R", Label: "Restart", Desc: "this pod", Scope: ItemScope,
		Disabled: func() bool { order = append(order, "disabled"); return false },
		AskFor: func() *Ask {
			order = append(order, "ask")
			if !ask {
				return nil
			}
			r, _ := ta.list.Current()
			return &Ask{Title: "Restart", Question: "Restart " + r.ID + "?", Verb: "restart"}
		}, Do: ta.record("restart")})
	r, _ := ta.list.Current()
	ta.press("R")
	c, ok := ta.topView().(*confirmView)
	if !ok || c.ask.Question != "Restart "+r.ID+"?" {
		t.Fatalf("the confirm: %T %+v", ta.topView(), c)
	}
	if i := slices.Index(order, "ask"); i < 1 || order[i-1] != "disabled" {
		t.Errorf("asked before Disabled: %v", order)
	}
	ta.press("Esc")
	if len(ta.ran) != 0 {
		t.Errorf("ran %v after Esc", ta.ran)
	}
	ask = false
	ta.press("R")
	if !slices.Equal(ta.ran, []string{"restart"}) || ta.top() != nil {
		t.Errorf("with no question: ran %v, %T open", ta.ran, ta.topView())
	}
	// From the Space menu: Esc on the confirm goes back to the menu.
	ask = true
	ta.press("Space")
	m := ta.topView().(*menuView)
	m.cursor = slices.IndexFunc(m.rows, func(r menuRow) bool { return r.action.Label == "Restart" })
	ta.press("Enter")
	if _, ok := ta.topView().(*confirmView); !ok {
		t.Fatalf("from the menu: %T", ta.topView())
	}
	ta.press("Esc")
	if ta.topView() != m {
		t.Errorf("Esc on the confirm went to %T, not back to the menu", ta.topView())
	}
}
