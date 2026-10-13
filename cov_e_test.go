package terminu

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Tests of components/dialog/form, dialog/note and dialog/toast, as listed in
// docs/coverage.md.

// covEBox is a layer as the user sees it: its rows with their styles, and
// without.
func covEBox(ta *testApp, l *layer) (styled, plain []string) {
	frame := ta.Frame()
	for i := range l.h {
		row := Cut(frame[l.y+i], l.x, l.x+l.w)
		styled = append(styled, row)
		plain = append(plain, StripStyles(row))
	}
	return styled, plain
}

// covEPopup is covEBox of the top popup.
func covEPopup(ta *testApp) (styled, plain []string) { return covEBox(ta, ta.top()) }

// covERow is the first row holding s, -1 when none does.
func covERow(rows []string, s string) int {
	for i, r := range rows {
		if strings.Contains(r, s) {
			return i
		}
	}
	return -1
}

// covECol is the cell where s starts in row, -1 when it is not there.
func covECol(row, s string) int {
	i := strings.Index(row, s)
	if i < 0 {
		return -1
	}
	return Width(row[:i])
}

// covEInside is a plain row of a layer w wide without its frame.
func covEInside(row string, w int) string { return StripStyles(Cut(row, 1, w-1)) }

// covEBlock is the escape sequence that starts a block of c: a cursor, a
// focused item, a focused button.
func covEBlock(c Color) string { return "\x1b[" + On(c).sgr() + "m" }

// covEBottom is the bottom border of the top popup, where the hint is.
func covEBottom(ta *testApp) string {
	_, plain := covEPopup(ta)
	return plain[len(plain)-1]
}

// covEKeys are the keys a key reference lists, section by section: "" for the
// rows before the first section title.
func covEKeys(plain []string) map[string][]string {
	out := map[string][]string{}
	section := ""
	for _, row := range plain[1 : len(plain)-1] {
		s := strings.TrimSpace(strings.Trim(row, "┃"))
		switch {
		case s == "":
		case strings.HasPrefix(s, glyphHexagon):
			section = strings.TrimSpace(strings.TrimPrefix(s, glyphHexagon))
		default:
			out[section] = append(out[section], strings.Fields(s)[0])
		}
	}
	return out
}

// --- form --------------------------------------------------------------------

func TestFormNoTyping(t *testing.T) {
	ta := newTestApp(t)
	r := &formResult{}
	ta.Form(testForm(r))
	ta.settle()
	ta.press("Enter")
	if _, ok := ta.topView().(*lineInput); !ok || len(ta.stack) != 2 {
		t.Fatalf("Enter on Name opens its input popup over the form: %T, %d layers", ta.topView(), len(ta.stack))
	}
	ta.press("w", "e", "b", "Enter")
	if _, ok := ta.topView().(*formView); !ok || r.name != "web" {
		t.Fatalf("what is typed in the popup is written back to the form: %q, top %T", r.name, ta.topView())
	}
	ta.press("G", "k", "k", "Enter") // Profile: its options are in a select
	if f, ok := ta.topView().(*finderView); !ok || f.kind != findSelect {
		t.Fatalf("Enter on Profile opens a select: %T", ta.topView())
	}
	ta.press("Esc")
	_, plain := covEPopup(ta)
	for _, opt := range []string{glyphRadioOn + " password", glyphRadioOff + " privatekey", glyphRadioOff + " credential", glyphUnchecked + " forward"} {
		if covERow(plain, opt) < 0 {
			t.Errorf("the option %q is not drawn on the form:\n%s", opt, strings.Join(plain, "\n"))
		}
	}
	ta.press("gg", "Tab", "Tab", "j", "Enter") // Auth: privatekey
	_, plain = covEPopup(ta)
	if len(ta.stack) != 1 || covERow(plain, glyphRadioOn+" privatekey") < 0 {
		t.Errorf("an option is chosen on the form, no popup: %d layers", len(ta.stack))
	}
}

func TestFormLayouts(t *testing.T) {
	ta := newTestApp(t)
	f := &Form{Title: "New profile", Button: "Save", Fields: []*FormField{
		{Label: "Name", Value: func() string { return "clock3" }, Check: func() string { return "Name is taken" }},
		{Label: "Timeout", Value: func() string { return "30" }},
		{Label: "Command line", Stacked: true, Value: func() string { return "tmux new-session -A -s main" }},
		{Label: "Layout", Value: func() string { return "row" }},
	}}
	ta.Form(f)
	ta.settle()
	ta.press("G", "Enter") // the check fails: the error row shows Name is taken
	_, plain := covEPopup(ta)
	name, timeout, cmd := covERow(plain, "Name"), covERow(plain, "Timeout"), covERow(plain, "Command line")
	value, layout, errRow := covERow(plain, "tmux new-session"), covERow(plain, "Layout"), covERow(plain, "Name is taken")
	if min(name, timeout, cmd, value, layout, errRow) < 0 {
		t.Fatalf("a field or the error is missing:\n%s", strings.Join(plain, "\n"))
	}
	if timeout != name+1 || cmd != timeout+1 || value != cmd+1 || layout != value+1 {
		t.Errorf("no blank row between fields, nor between a stacked label and its value: rows %d %d %d %d %d",
			name, timeout, cmd, value, layout)
	}
	left := covECol(plain[name], "Name")
	for _, c := range []struct{ row, s string }{
		{plain[timeout], "Timeout"}, {plain[cmd], "Command line"}, {plain[value], "tmux"}, {plain[layout], "Layout"}, {plain[errRow], "Name is taken"},
	} {
		if got := covECol(c.row, c.s); got != left {
			t.Errorf("%q starts at cell %d, the labels at %d: labels, values and errors share the left edge", c.s, got, left)
		}
	}
	// The label column is the longest side-by-side label (Timeout, not the
	// stacked Command line), then two cells.
	want := left + Width("Timeout") + 2
	for _, c := range []struct{ row, s string }{{plain[name], "clock3"}, {plain[timeout], "30"}, {plain[layout], "row"}} {
		if got := covECol(c.row, c.s); got != want {
			t.Errorf("the value %q starts at cell %d, want %d", c.s, got, want)
		}
	}
}

func TestFormOptionsDrawn(t *testing.T) {
	ta := newTestApp(t)
	ta.Form(&Form{Title: "New host", Fields: []*FormField{
		{Label: "Auth", Options: []string{"password", "privatekey", "credential"}},
		{Label: "Agent", Options: []string{"forward", "x11"}, Multi: true, Checked: []bool{true, false}},
	}})
	ta.settle()
	_, plain := covEPopup(ta)
	opts := []string{glyphRadioOn + " password", glyphRadioOff + " privatekey", glyphRadioOff + " credential",
		glyphMarked + " forward", glyphUnchecked + " x11"}
	first := covERow(plain, opts[0])
	if first < 0 {
		t.Fatalf("the options are not drawn on the form:\n%s", strings.Join(plain, "\n"))
	}
	for i, o := range opts {
		if r := covERow(plain, o); r != first+i || covECol(plain[r], o) != covECol(plain[first], opts[0]) {
			t.Errorf("one option a row, one under the other: %q on row %d", o, r)
		}
	}

	t.Run("more than five", func(t *testing.T) {
		ta := newTestApp(t)
		six := []string{"one", "two", "three", "four", "five", "six"}
		refused := func() (r bool) {
			defer func() { r = recover() != nil }()
			ta.Form(&Form{Title: "Pick", Fields: []*FormField{{Label: "Count", Options: six}}})
			return false
		}()
		if refused {
			return
		}
		ta.settle()
		_, plain := covEPopup(ta)
		if covERow(plain, glyphRadioOff+" six") >= 0 {
			t.Error("more than five options go in a select or checkbox popup, not on the form")
		}
	})
}

func TestFormFocusBlock(t *testing.T) {
	ta := newTestApp(t)
	ta.Form(testForm(nil))
	ta.settle()
	layer := LayerColor(1)
	styled, plain := covEPopup(ta)
	row := covERow(plain, "Name")
	start, arrow := covECol(plain[row], "required"), covECol(plain[row], glyphHard)
	if got := covECol(styled[row], covEBlock(layer)); got != start {
		t.Errorf("the block starts at the value, cell %d, not %d", start, got)
	}
	if !strings.HasSuffix(plain[row], glyphHard+"┃") {
		t.Errorf("the block goes to the frame, its last cell an arrow: %q", plain[row])
	}
	// One block from the value to the arrow, an empty value too.
	if want := On(layer).Render(Pad("required", arrow-start)) + Fg(layer).Render(glyphHard); !strings.Contains(styled[row], want) {
		t.Errorf("not one block of the layer colour, Base bold, ending in a layer-colour arrow: %q", styled[row])
	}
	ta.press("Tab", "Tab") // Auth: password
	styled, plain = covEPopup(ta)
	row = covERow(plain, "password")
	if covECol(styled[row], covEBlock(layer)) != covECol(plain[row], glyphRadioOn) || !strings.HasSuffix(plain[row], glyphHard+"┃") {
		t.Errorf("on an option the block starts at the option: %q", plain[row])
	}
	ta.press("G")
	styled, plain = covEPopup(ta)
	row = covERow(plain, "Save")
	if !strings.Contains(styled[row], On(layer).Render(" Save ")) || strings.Contains(plain[row], glyphHard) {
		t.Errorf("the button's block is the button, no arrow: %q", plain[row])
	}
}

func TestFormLabelColors(t *testing.T) {
	ta := newTestApp(t)
	ta.Form(&Form{Title: "New host", Fields: []*FormField{
		{Label: "Name", Value: func() string { return "" }, Empty: "required", Check: func() string { return "Name is required" }},
		{Label: "Port", Value: func() string { return "22" }},
		{Label: "Credential", Value: func() string { return "vault" }, Disabled: func() bool { return true }},
		{Label: "Key", Value: func() string { return "id_rsa" }, Disabled: func() bool { return true },
			Check: func() string { return "Key is not allowed" }},
	}})
	ta.settle()
	layer := LayerColor(1)
	styled, plain := covEPopup(ta)
	row := func(label string) string { return styled[covERow(plain, label)] }
	if !strings.Contains(row("Name"), Fg(layer).Render("Name")) {
		t.Error("the label of the focused field stays the layer colour")
	}
	if !strings.Contains(row("Port"), Fg(layer).Render("Port")) || !strings.Contains(row("Port"), Fg(Text).Render("22")) {
		t.Errorf("a label is the layer colour, its value another: %q", row("Port"))
	}
	cred := covERow(plain, "Credential")
	if cred != covERow(plain, "Port")+1 || !strings.Contains(row("Credential"), Fg(Surface2).Render("Credential")) ||
		!strings.Contains(row("Credential"), Fg(Text).Render("vault")) {
		t.Errorf("a disabled label is Surface2, the field in its place, its value another colour: %q", row("Credential"))
	}
	ta.press("G", "Enter", "j") // submit: Name and Key are wrong; the focus away from Name
	styled, plain = covEPopup(ta)
	if !strings.Contains(row("Name"), Fg(Red).Render("Name")) || !strings.Contains(row("Name"), Fg(Overlay0).Render("required")) {
		t.Errorf("a wrong value's label is Red, its value another colour: %q", row("Name"))
	}
	if !strings.Contains(row("Key"), Fg(Red).Render("Key")) {
		t.Errorf("Red comes before the disabled colour: %q", row("Key"))
	}
}

func TestFormButton(t *testing.T) {
	ta := newTestApp(t)
	ta.Form(testForm(nil))
	ta.settle()
	styled, plain := covEPopup(ta)
	row := covERow(plain, "Save")
	if !strings.Contains(styled[row], Style{Fg: Text, Bg: Surface1}.Render(" Save ")) {
		t.Errorf("without the focus the button is a Surface1 block, Text, one space either side: %q", styled[row])
	}
	for _, s := range []string{"[", "]", glyphRoundLeft, glyphRoundRight, glyphSlantLeft, glyphHard} {
		if strings.Contains(plain[row], s) {
			t.Errorf("the button is neither [ Save ] nor a capsule: %q", plain[row])
		}
	}
	ta.press("G")
	styled, _ = covEPopup(ta)
	if !strings.Contains(styled[row], On(LayerColor(1)).Render(" Save ")) {
		t.Errorf("with the focus the button is a block of the layer colour, Base bold: %q", styled[row])
	}
}

func TestFormActionRowFixed(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(80, 14) // the fields need to scroll
	ta.Form(testForm(nil))
	ta.settle()
	ta.press("G", "Enter", "gg") // Name is required
	check := func(when string) int {
		_, plain := covEPopup(ta)
		b := covERow(plain, "Save")
		inside := func(i int) string { return strings.TrimSuffix(strings.TrimPrefix(plain[i], "┃"), "┃") }
		if strings.TrimSpace(inside(b)) != "Save" || !strings.HasSuffix(plain[b], " Save  ┃") {
			t.Errorf("%s: one button, on the right, one cell from the frame: %q", when, plain[b])
		}
		sep := inside(b - 1)
		if strings.Trim(sep, " ─") != "" || !strings.HasPrefix(sep, " ─") || !strings.HasSuffix(sep, "─ ") {
			t.Errorf("%s: the separator is right above the button row, not touching the frame: %q", when, plain[b-1])
		}
		if strings.TrimSpace(inside(b-2)) != "Name is required" || strings.TrimSpace(inside(b-3)) != "" {
			t.Errorf("%s: a blank row, then the error row above the separator: %q %q", when, plain[b-3], plain[b-2])
		}
		if strings.TrimSpace(inside(b+1)) != "" || !strings.HasPrefix(plain[b+2], "┗") {
			t.Errorf("%s: the button row is the last of the body", when)
		}
		if strings.Contains(strings.ToLower(strings.Join(plain[:len(plain)-1], "\n")), "cancel") {
			t.Errorf("%s: there is no cancel button", when)
		}
		return b
	}
	b := check("on Name")
	ta.press("Tab", "Tab", "Tab", "Tab", "Tab") // Command, the last field
	_, plain := covEPopup(ta)
	if covERow(plain, "Name ") >= 0 && covERow(plain, "Name ") < b-3 {
		t.Fatalf("the fields did not scroll:\n%s", strings.Join(plain, "\n"))
	}
	if check("on Command") != b {
		t.Error("the error row, the separator and the button row do not scroll")
	}
}

func TestFormItems(t *testing.T) {
	ta := newTestApp(t)
	ta.Form(testForm(nil))
	ta.settle()
	fv := ta.topView().(*formView)
	var seen []string
	for range 9 {
		seen = append(seen, formStop(fv))
		ta.press("j")
	}
	want := []string{"Name", "Port", "Auth:0", "Auth:1", "Auth:2", "Agent:0", "Profile", "Command", "button"}
	if !slices.Equal(seen, want) {
		t.Errorf("items %v, want %v: a field, each option, the button", seen, want)
	}
	ta.press("G", "k")
	if formStop(fv) != "Command" {
		t.Errorf("with no error there is no error row: %s", formStop(fv))
	}
	ta.press("G", "Enter", "G", "k") // Name is required
	if formStop(fv) != "error" {
		t.Errorf("with an error the error row is an item, before the button: %s", formStop(fv))
	}
}

func TestFormMoveItems(t *testing.T) {
	ta := newTestApp(t)
	ta.Form(testForm(nil))
	ta.settle()
	fv := ta.topView().(*formView)
	for _, k := range []string{"j", "l", "↓", "→"} {
		ta.press("gg", k)
		if formStop(fv) != "Port" {
			t.Errorf("%s on Name goes to %s, want the next item", k, formStop(fv))
		}
	}
	for _, k := range []string{"k", "h", "↑", "←"} {
		ta.press("gg", "Tab", "Tab", k) // from Auth's first option
		if formStop(fv) != "Port" {
			t.Errorf("%s on Auth:0 goes to %s, want the item before", k, formStop(fv))
		}
	}
	ta.press("gg", "Tab", "Tab", "j", "j", "j")
	if formStop(fv) != "Agent:0" {
		t.Errorf("after a field's last option comes the next field: %s", formStop(fv))
	}
	ta.press("k")
	if formStop(fv) != "Auth:2" {
		t.Errorf("before a field's first item comes the last of the one above: %s", formStop(fv))
	}
	ta.press("G")
	if formStop(fv) != "button" {
		t.Errorf("G goes to the button: %s", formStop(fv))
	}
	ta.press("gg")
	if formStop(fv) != "Name" {
		t.Errorf("gg goes to the first item: %s", formStop(fv))
	}
}

func TestFormWrap(t *testing.T) {
	ta := newTestApp(t)
	ta.Form(testForm(nil))
	ta.settle()
	fv := ta.topView().(*formView)
	for _, c := range []struct{ start, key, want string }{
		{"G", "j", "Name"}, {"gg", "k", "button"}, {"G", "↓", "Name"}, {"gg", "↑", "button"},
		{"G", "Tab", "Name"}, {"gg", "Shift-Tab", "button"},
	} {
		ta.press(c.start, c.key)
		if formStop(fv) != c.want {
			t.Errorf("%s then %s: %s, want %s", c.start, c.key, formStop(fv), c.want)
		}
	}
}

func TestFormHalfPage(t *testing.T) {
	ta := newTestApp(t)
	ta.Form(testForm(nil))
	ta.settle()
	ta.Frame()
	fv := ta.topView().(*formView)
	half := max(fv.visible/2, 1)
	items := []string{"Name", "Port", "Auth:0", "Auth:1", "Auth:2", "Agent:0", "Profile", "Command", "button"}
	last := len(items) - 1
	ta.press("d")
	if want := items[min(half, last)]; formStop(fv) != want {
		t.Errorf("d moves half a page of items: %s, want %s", formStop(fv), want)
	}
	ta.press("G", "d")
	if formStop(fv) != "button" {
		t.Errorf("d stops at the end, no wrapping: %s", formStop(fv))
	}
	ta.press("u")
	if want := items[max(last-half, 0)]; formStop(fv) != want {
		t.Errorf("u moves half a page back: %s, want %s", formStop(fv), want)
	}
	ta.press("gg", "u")
	if formStop(fv) != "Name" {
		t.Errorf("u stops at the start, no wrapping: %s", formStop(fv))
	}
}

func TestFormSkipDisabled(t *testing.T) {
	ta := newTestApp(t)
	off := func() bool { return true }
	ta.Form(&Form{Title: "New host", Fields: []*FormField{
		{Label: "Name", Value: func() string { return "web" }},
		{Label: "Credential", Value: func() string { return "vault" }, Disabled: off},
		{Label: "Mode", Options: []string{"fast", "safe"}, Disabled: off},
		{Label: "Port", Value: func() string { return "22" }},
	}})
	ta.settle()
	fv := ta.topView().(*formView)
	for _, c := range []struct{ keys, want string }{
		{"gg j", "Port"}, {"gg Tab", "Port"}, {"G k k", "Name"}, {"G k Shift-Tab", "Name"}, // G k: Port
	} {
		ta.press(strings.Fields(c.keys)...)
		if formStop(fv) != c.want {
			t.Errorf("%s goes to %s, want %s: disabled fields and their options are skipped", c.keys, formStop(fv), c.want)
		}
	}
	_, plain := covEPopup(ta)
	name := covERow(plain, "Name")
	if covERow(plain, "Credential") != name+1 || covERow(plain, "fast") != name+2 || covERow(plain, "safe") != name+3 ||
		covERow(plain, "Port") != name+4 {
		t.Errorf("disabled fields stay drawn in their place:\n%s", strings.Join(plain, "\n"))
	}
}

func TestFormOpenFocus(t *testing.T) {
	ta := newTestApp(t)
	f := testForm(nil)
	f.Fields[0].Disabled = func() bool { return true }
	ta.Form(f)
	ta.settle()
	if fv, ok := ta.topView().(*formView); !ok || len(ta.stack) != 1 || formStop(fv) != "Port" {
		t.Fatalf("a form opens on itself, the focus on the first item that can take it: %T, %d layers", ta.topView(), len(ta.stack))
	}
	ta.press("Esc")
	if ta.top() != nil {
		t.Fatal("one Esc leaves a form just opened")
	}
	f = testForm(nil)
	f.Focus = f.Fields[1]
	ta.Form(f)
	ta.settle()
	if fv := ta.topView().(*formView); formStop(fv) != "Port" {
		t.Errorf("opened from Port, the focus is on Port: %s", formStop(fv))
	}
	ta.press("Esc")
	f = testForm(nil)
	f.Fields[2].Selected = 1
	f.Focus = f.Fields[2]
	ta.Form(f)
	ta.settle()
	if fv := ta.topView().(*formView); formStop(fv) != "Auth:1" {
		t.Errorf("opened from a radio, the focus is on its chosen option: %s", formStop(fv))
	}
}

func TestFormEnterField(t *testing.T) {
	ta := newTestApp(t)
	r := &formResult{}
	ta.Form(testForm(r))
	ta.settle()
	fv := ta.topView().(*formView)
	ta.press("Tab", "Enter") // Port
	if _, ok := ta.topView().(*lineInput); !ok || len(ta.stack) != 2 {
		t.Fatalf("Enter on Port opens its input popup over the form: %T", ta.topView())
	}
	ta.press("2", "2", "2", "2", "Enter")
	_, plain := covEPopup(ta)
	if ta.topView() != fv || r.port != "2222" || formStop(fv) != "Port" || covERow(plain, "2222") < 0 {
		t.Errorf("Enter writes the value back, closes the popup and keeps the focus on the field: %q, %s", r.port, formStop(fv))
	}
	ta.press("Enter", "1", "Esc")
	if ta.topView() != fv || r.port != "2222" || formStop(fv) != "Port" {
		t.Errorf("Esc in the popup keeps the value: %q", r.port)
	}

	ns := []bool{true, false, false}
	ta = newTestApp(t)
	ta.Form(&Form{Title: "Filter", Fields: []*FormField{{Label: "Namespaces", Value: func() string { return fmt.Sprint(ns) },
		Edit: func() Input {
			return Checks{Title: "Namespaces", Options: []string{"default", "kube-system", "monitoring"}, Checked: ns,
				Done: func(v []bool) tea.Cmd { ns = v; return nil }}
		}}}})
	ta.settle()
	fv = ta.topView().(*formView)
	ta.press("Enter")
	if f, ok := ta.topView().(*finderView); !ok || f.kind != findChecks {
		t.Fatalf("Enter opens the checkbox popup: %T", ta.topView())
	}
	ta.press("j", "Space")
	if !slices.Equal(ns, []bool{true, false, false}) {
		t.Error("Space checks in the popup, it does not write back")
	}
	ta.press("Esc")
	if ta.topView() != fv || !slices.Equal(ns, []bool{true, false, false}) {
		t.Errorf("Esc cancels the checkbox popup: %v", ns)
	}
	ta.press("Enter", "j", "Space", "Enter")
	if ta.topView() != fv || !slices.Equal(ns, []bool{true, true, false}) || formStop(fv) != "Namespaces" {
		t.Errorf("Enter writes the whole group back, the focus on the field: %v", ns)
	}
}

func TestFormEnterOption(t *testing.T) {
	ta := newTestApp(t)
	f := testForm(nil)
	auth, agent := f.Fields[2], f.Fields[3]
	ta.Form(f)
	ta.settle()
	fv := ta.topView().(*formView)
	ta.press("Tab", "Tab", "j", "Enter") // Auth: privatekey
	_, plain := covEPopup(ta)
	if auth.Selected != 1 || len(ta.stack) != 1 || formStop(fv) != "Auth:1" ||
		covERow(plain, glyphRadioOn+" privatekey") < 0 || covERow(plain, glyphRadioOff+" password") < 0 {
		t.Errorf("Enter on a radio option chooses it at once, no popup, the focus stays: %d, %s", auth.Selected, formStop(fv))
	}
	ta.press("Tab", "Enter") // Agent: forward
	_, plain = covEPopup(ta)
	if len(agent.Checked) == 0 || !agent.Checked[0] || len(ta.stack) != 1 || formStop(fv) != "Agent:0" ||
		covERow(plain, glyphMarked+" forward") < 0 {
		t.Errorf("Enter on a checkbox option turns it on at once: %v, %s", agent.Checked, formStop(fv))
	}
	ta.press("Enter")
	_, plain = covEPopup(ta)
	if agent.Checked[0] || covERow(plain, glyphUnchecked+" forward") < 0 {
		t.Error("Enter again turns it off")
	}
}

func TestFormEnterButton(t *testing.T) {
	ta := newTestApp(t)
	r := &formResult{name: "web-01"}
	ta.Form(testForm(r))
	ta.settle()
	ta.press("G", "Enter")
	if r.submitted != 1 || ta.top() != nil {
		t.Errorf("Enter on the button submits the form: %d, top %T", r.submitted, ta.topView())
	}
}

func TestFormSubmitOnlyButton(t *testing.T) {
	ta := newTestApp(t)
	r := &formResult{name: "web-01"}
	ta.Form(testForm(r))
	ta.settle()
	fv := ta.topView().(*formView)
	ta.press("Tab", "Tab", "s", "S", "w", "Space") // on Auth's first option
	if r.submitted != 0 || ta.topView() != fv || formStop(fv) != "Auth:0" {
		t.Errorf("no letter submits the form: submitted %d, focus %s", r.submitted, formStop(fv))
	}
	ta.press("gg", "Enter", "Esc") // Enter on a field opens its popup
	if r.submitted != 0 || ta.topView() != fv {
		t.Error("Enter on a field does not submit")
	}
	ta.press("Esc")
	if r.submitted != 0 || ta.top() != nil {
		t.Error("Esc closes the form; it does not go to the button")
	}
}

func TestFormOperationKeys(t *testing.T) {
	ta := newTestApp(t)
	ta.Form(testForm(nil))
	ta.settle()
	fv := ta.topView().(*formView)
	for _, k := range []string{"j", "k", "h", "l", "↓", "↑", "←", "→", "Tab", "Shift-Tab", "u", "d", "G", "e", "Space"} {
		ta.press("gg", k)
		if ta.topView() != fv {
			t.Fatalf("%s on Name opened %T: it is a form key, no typing note", k, ta.topView())
		}
		if (k == "e" || k == "Space") && formStop(fv) != "Name" {
			t.Errorf("%s does nothing here, the focus went to %s", k, formStop(fv))
		}
	}
	ta.press("gg", "g")
	if ta.topView() != fv || formStop(fv) != "Name" {
		t.Error("g alone does nothing")
	}
	ta.press("g") // gg
	ta.press("?")
	if n, ok := ta.topView().(*noteView); !ok || n.title != "NEW HOST KEYS" {
		t.Fatalf("? opens the form's key reference: %T", ta.topView())
	}
	ta.press("Esc", "Enter")
	if _, ok := ta.topView().(*lineInput); !ok {
		t.Fatalf("Enter on Name opens its input popup: %T", ta.topView())
	}
	ta.press("Esc", "Esc")
	if ta.top() != nil {
		t.Errorf("Esc closes an unchanged form: %T", ta.topView())
	}
}

func TestFormHint(t *testing.T) {
	ta := newTestApp(t)
	ta.Form(testForm(nil))
	ta.settle()
	for _, c := range []struct{ keys, want string }{
		{"gg", "Enter:edit Esc:cancel"},
		{"Tab Tab", "Enter:choose Esc:cancel"}, // Auth: password
		{"Tab", "Enter:toggle Esc:cancel"},     // Agent: forward
		{"G", "Enter:save Esc:cancel"},
		{"Enter", "Enter:edit e:error Esc:cancel"}, // Name is required: the focus on Name
		{"e", "Enter:show e:edit Esc:cancel"},      // the error row
	} {
		ta.press(strings.Fields(c.keys)...)
		if b := covEBottom(ta); !strings.HasPrefix(b, "┗━ "+c.want+" ━") {
			t.Errorf("after %s the hint is %q, want %s", c.keys, b, c.want)
		}
	}
	f := testForm(nil)
	f.Button = "Connect"
	ta.Form(f)
	ta.settle()
	ta.press("G")
	if b := covEBottom(ta); !strings.HasPrefix(b, "┗━ Enter:connect Esc:cancel ━") {
		t.Errorf("on the button Enter: is its text in lower case: %q", b)
	}
}

func TestFormDirtyFlag(t *testing.T) {
	ta := newTestApp(t)
	r := &formResult{name: "web"}
	open := func() {
		ta.Form(testForm(r))
		ta.settle()
	}
	// asks presses Esc on the form and reports whether it asked first; it
	// leaves the form closed.
	asks := func() bool {
		ta.press("Esc")
		_, ok := ta.topView().(*confirmView)
		if ok {
			ta.press("Enter")
		}
		if ta.top() != nil {
			t.Fatalf("the form did not close: %T", ta.topView())
		}
		return ok
	}
	open()
	ta.press("Enter", "Enter") // the value confirmed as it was
	if asks() {
		t.Error("a value confirmed unchanged is not a change")
	}
	open()
	ta.press("Enter", "Tab", "2", "Enter")         // web2
	ta.press("Enter", "Tab", "Backspace", "Enter") // web again
	if r.name != "web" || !asks() {
		t.Errorf("a value changed and changed back is still a change: %q", r.name)
	}
	open()
	ta.press("Tab", "Tab", "j", "Enter") // Auth: privatekey
	if !asks() {
		t.Error("choosing a radio option is a change")
	}
	open()
	ta.press("Tab", "Tab", "Tab", "Enter", "Enter") // Agent: on and off again
	if !asks() {
		t.Error("a checkbox turned on and off is still a change: the flag is never cleared")
	}
}

func TestFormValidation(t *testing.T) {
	ta := newTestApp(t)
	r := &formResult{}
	f := testForm(r)
	f.Fields[2].Selected = 2 // Auth: credential, so Command is wrong too
	ta.Form(f)
	ta.settle()
	fv := ta.topView().(*formView)
	ta.press("Tab", "Enter", "0", "Enter") // Port 0
	if _, ok := ta.topView().(*lineInput); !ok || r.port != "" || !strings.Contains(ta.screenText(), "Port must be 1-65535") {
		t.Fatalf("a field's own rule stops the value in its input popup: %T, %q", ta.topView(), r.port)
	}
	ta.press("Esc", "G", "Enter")
	if r.submitted != 0 || ta.topView() != fv {
		t.Fatalf("a form that fails its checks is not saved and stays: submitted %d", r.submitted)
	}
	styled, plain := covEPopup(ta)
	row := func(s string) string { return styled[covERow(plain, s)] }
	if !strings.Contains(row("Name"), Fg(Red).Render("Name")) || !strings.Contains(row("Command"), Fg(Red).Render("Command")) {
		t.Error("every wrong field's label turns Red")
	}
	if !strings.Contains(row("Port"), Fg(LayerColor(1)).Render("Port")) {
		t.Error("a field that is right keeps its label colour")
	}
	if covERow(plain, "Name is required") < 0 || covERow(plain, "Command can't be used") >= 0 {
		t.Errorf("the error row shows the first problem in field order:\n%s", strings.Join(plain, "\n"))
	}
	if formStop(fv) != "Name" {
		t.Errorf("the focus goes to the first problem: %s", formStop(fv))
	}
}

func TestFormNoCheckBeforeSubmit(t *testing.T) {
	ta := newTestApp(t)
	r := &formResult{}
	ta.Form(testForm(r))
	ta.settle()
	red := "38;2;243;139;168"
	for range 9 {
		if styled, _ := covEPopup(ta); strings.Contains(strings.Join(styled, ""), red) {
			t.Fatal("nothing is Red before the first submit")
		}
		ta.press("j")
	}
	ta.press("G", "Enter")
	ta.press("Enter", "w", "e", "b", "Enter") // Name fixed
	styled, plain := covEPopup(ta)
	if !strings.Contains(styled[covERow(plain, "Name")], Fg(Red).Render("Name")) || covERow(plain, "Name is required") < 0 {
		t.Error("the error stays until the next submit, the value fixed or not")
	}
	ta.press("Tab", "Tab", "j", "j", "Enter", "G", "Enter") // Auth: credential, submit
	styled, plain = covEPopup(ta)
	if strings.Contains(styled[covERow(plain, "Name")], Fg(Red).Render("Name")) || covERow(plain, "Command can't be used") < 0 {
		t.Errorf("the next submit checks again:\n%s", strings.Join(plain, "\n"))
	}
	ta.press("gg", "Tab", "Tab", "k", "k", "Enter", "G", "Enter") // Auth: password
	if r.submitted != 1 {
		t.Error("with nothing wrong the form is saved")
	}
}

// --- note --------------------------------------------------------------------

func TestNoteLayout(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(80, 20)
	v := testViewer()
	v.Glyph = "\U0000F15C"
	ta.Viewer(v)
	ta.settle()
	styled, plain := covEPopup(ta)
	blank := func(row string) bool { return strings.TrimSpace(strings.Trim(row, "┃")) == "" }
	if !blank(plain[1]) || !blank(plain[len(plain)-2]) {
		t.Error("a viewer has a blank row above and below its lines")
	}
	if !strings.Contains(styled[0], On(LayerColor(1)).Render("\U0000F15C YAML — pod/nginx")) {
		t.Errorf("the title capsule is the glyph and what it shows: %q", plain[0])
	}
	if want := fmt.Sprintf(" 1-%d of 40 ", v.visible); !strings.Contains(plain[len(plain)-1], want) {
		t.Errorf("the scroll position is the lines in sight, %q: %q", want, plain[len(plain)-1])
	}
	if l := ta.top(); l.w != 78 {
		t.Errorf("a viewer takes the most width, 78 on 80 columns: %d", l.w)
	}
	ta.press("Esc", "?")
	_, plain = covEPopup(ta)
	l := ta.top()
	widest := 0
	for _, row := range plain[1 : len(plain)-1] {
		widest = max(widest, Width(strings.TrimRight(covEInside(row, l.w), " ")))
	}
	if l.w >= 78 || widest != l.w-3 {
		t.Errorf("a key reference is as wide as its content: width %d, widest row %d", l.w, widest)
	}
}

// TestNoteHint is TestNoteKeys: it checks the hint, the note's own keys first
// and Esc:close last.

func TestKeyReferenceLayout(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(80, 40)
	ta.pods.Actions = append(ta.pods.Actions, Action{Key: "c", Label: "SSH config", Scope: ItemScope,
		Desc: strings.Repeat("a long description ", 10)})
	ta.press("?")
	_, plain := covEPopup(ta)
	if !strings.Contains(plain[0], glyphKeys+" [1] PODS KEYS") {
		t.Errorf("the title is the question circle and <surface> keys: %q", plain[0])
	}
	widest := 0
	for _, keys := range covEKeys(plain) {
		for _, k := range keys {
			widest = max(widest, Width(k))
		}
	}
	for _, row := range plain[1 : len(plain)-1] {
		s := strings.TrimSpace(strings.Trim(row, "┃"))
		if s == "" || strings.HasPrefix(s, glyphHexagon) {
			continue
		}
		key := strings.Fields(s)[0]
		if lead := "┃  " + Pad(key, widest) + "  "; !strings.HasPrefix(row, lead) || strings.HasPrefix(row, lead+" ") {
			t.Errorf("two cells in, the key padded to the widest (%d), two cells, then the text: %q", widest, row)
		}
	}
	row := plain[covERow(plain, "SSH config")]
	if !strings.Contains(row, "SSH config — a long description") || strings.Contains(row, "[c]") {
		t.Errorf("a row from the menu is label — description, the label without its marker, its case kept: %q", row)
	}
	wrapped := slices.ContainsFunc(plain[covERow(plain, "SSH config")+1:], func(r string) bool { return strings.Contains(r, "long desc") })
	if !strings.HasSuffix(row, "… ┃") || wrapped {
		t.Errorf("a long description is cut with …, not wrapped: %q", row)
	}
	ta.press("Esc", "Space", "?")
	if _, plain = covEPopup(ta); !strings.Contains(plain[0], glyphKeys+" SPACE MENU KEYS") {
		t.Errorf("the Space menu's key reference: %q", plain[0])
	}
}

func TestKeyReferenceSections(t *testing.T) {
	ta := newTestApp(t)
	ta.press("?")
	styled, plain := covEPopup(ta)
	last := 0
	for _, title := range []string{"ITEM OPERATION", "PANEL OPERATION", "MOVE", "CORE KEYS", "APP-WIDE"} {
		head := glyphHexagon + " " + title
		i := covERow(plain, "┃ "+head)
		if i <= last {
			t.Errorf("%s is on row %d, after row %d: the sections are in order, one cell from the frame", title, i, last)
			continue
		}
		last = i
		if !strings.Contains(styled[i], Fg(Overlay0).Render(head)) {
			t.Errorf("a section title is Overlay0, not bold: %q", styled[i])
		}
	}
}

func TestKeyReferenceMoveSection(t *testing.T) {
	ta := newTestApp(t)
	ta.press("?")
	_, plain := covEPopup(ta)
	if move := covEKeys(plain)["MOVE"]; !slices.Equal(move, []string{"j|k", "u|d", "gg|G", "h|l"}) {
		t.Errorf("a list with tabs moves with j|k u|d gg|G and switches tabs with h|l: %v", move)
	}
	ta.press("Esc", "2", "?")
	_, plain = covEPopup(ta)
	if move := covEKeys(plain)["MOVE"]; slices.Contains(move, "h|l") || !slices.Contains(move, "j|k") {
		t.Errorf("without tabs there is no h|l: %v", move)
	}
}

func TestKeyReferenceCoreKeys(t *testing.T) {
	ta := newTestApp(t)
	ta.press("?")
	_, plain := covEPopup(ta)
	if core := covEKeys(plain)["CORE KEYS"]; !slices.Equal(core, []string{"Space", "?", "Tab", "1–3", "q|Ctrl-C"}) {
		t.Errorf("core keys on a panel with nothing for Esc to do: %v", core)
	}
	ta.press("Esc", "/", "1", "Enter", "?") // a filter Esc can clear
	_, plain = covEPopup(ta)
	if core := covEKeys(plain)["CORE KEYS"]; !slices.Contains(core, "Esc") || slices.Contains(core, "Enter") {
		t.Errorf("Esc is listed when it does something; Enter is not repeated: %v", core)
	}
	ta.press("Esc", "Esc", "O", "?") // one panel
	_, plain = covEPopup(ta)
	if core := covEKeys(plain)["CORE KEYS"]; !slices.Equal(core, []string{"Space", "?", "q|Ctrl-C"}) {
		t.Errorf("with one panel there is no Tab nor 1–N: %v", core)
	}
}

func TestKeyReferenceKeys(t *testing.T) {
	ta := newTestApp(t)
	ta.pods.Actions = append(ta.pods.Actions, Action{Key: "z", Label: "Zap", Desc: "not now", Scope: ItemScope,
		Disabled: func() bool { return true }})
	ta.press("Space", "?")
	ref := ta.topView()
	styled, plain := covEPopup(ta)
	var keys []string
	for _, k := range covEKeys(plain) {
		keys = append(keys, k...)
	}
	for _, k := range []string{"Enter", "m", "x", "D", "z", "j|k", "u|d", "gg|G"} {
		if !slices.Contains(keys, k) {
			t.Errorf("the Space menu's key reference lacks %s: %v", k, keys)
		}
	}
	if !slices.ContainsFunc(keys, func(k string) bool { return strings.HasSuffix(k, "Esc") }) {
		t.Errorf("Esc is listed: %v", keys)
	}
	for _, k := range []string{"Tab", "1–3", "q|Ctrl-C", "?"} {
		if slices.Contains(keys, k) {
			t.Errorf("a popup lists only its own keys, not %s", k)
		}
	}
	enter := styled[covERow(plain, "┃  Enter")]
	if !strings.Contains(enter, Fg(Blue).Render("Enter")) || !strings.Contains(enter, Fg(Text).Render("the row under the cursor")) {
		t.Errorf("a key is Blue, not bold; its text is Text: %q", enter)
	}
	zap := styled[covERow(plain, "Zap")]
	if !strings.Contains(zap, "\x1b["+Fg(Surface2).sgr()+"mz") || !strings.Contains(zap, Fg(Surface2).Render("Zap — not now")) {
		t.Errorf("a key that cannot be pressed now is Surface2, its text too: %q", zap)
	}
	if b := covEBottom(ta); !strings.HasPrefix(b, "┗━ ?|Esc:close ━") {
		t.Errorf("hint: %q", b)
	}
	ta.press("Enter")
	if ta.topView() != ref {
		t.Fatal("Enter does nothing on a key reference")
	}
	ta.press("?")
	if _, ok := ta.topView().(*menuView); !ok {
		t.Fatalf("? closes the key reference: %T", ta.topView())
	}
	ta.press("?", "Esc")
	if _, ok := ta.topView().(*menuView); !ok {
		t.Fatalf("Esc closes the key reference: %T", ta.topView())
	}
}

func TestErrorPopupLayout(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(60, 10)
	msg := strings.Repeat("hosts.yaml changed on disk. ", 12)
	ta.Error("Save failed", msg)
	ta.settle()
	n := ta.topView().(*noteView)
	styled, plain := covEPopup(ta)
	if !strings.Contains(styled[0], On(Red).Render("\U000F0238 SAVE FAILED")) || !strings.Contains(styled[0], Fg(Red).Render("┏")) {
		t.Errorf("a Red frame; the fire glyph and what failed in a Red capsule, Base bold: %q", styled[0])
	}
	var all []string
	for _, l := range n.lines {
		all = append(all, StripStyles(l))
	}
	if strings.Join(strings.Fields(strings.Join(all, " ")), " ") != strings.TrimSpace(msg) || len(all) < 2 {
		t.Errorf("the message wraps, nothing cut: %q", all)
	}
	if !strings.Contains(styled[2], Fg(Red).Render(all[0])) {
		t.Errorf("the text is Red: %q", styled[2])
	}
	if want := fmt.Sprintf(" 1-%d of %d ", n.visible, len(all)); !strings.Contains(plain[len(plain)-1], want) {
		t.Fatalf("a long message scrolls: %q, want %q", plain[len(plain)-1], want)
	}
	ta.press("j")
	if want := fmt.Sprintf(" 2-%d of %d ", n.visible+1, len(all)); !strings.Contains(covEBottom(ta), want) {
		t.Errorf("j scrolls it: %q", covEBottom(ta))
	}

	ta = newTestApp(t)
	ta.Form(testForm(nil))
	ta.settle()
	ta.press("G", "Enter", "e", "Enter") // Name is required: the error row's full message
	if _, plain = covEPopup(ta); !strings.Contains(plain[0], "\U000F0238 NAME") {
		t.Errorf("opened from the form's error row its title is the field's name: %q", plain[0])
	}
}

func TestErrorPopupClose(t *testing.T) {
	ta := newTestApp(t)
	r := &formResult{port: "2222"}
	ta.Form(testForm(r))
	ta.settle()
	fv := ta.topView().(*formView)
	ta.press("G", "Enter", "e") // Name is required; on the error row
	for _, k := range []string{"Enter", "Esc"} {
		ta.press("Enter")
		if _, ok := ta.topView().(*noteView); !ok {
			t.Fatalf("Enter on the error row opens the error popup: %T", ta.topView())
		}
		if b := covEBottom(ta); !strings.HasPrefix(b, "┗━ Enter|Esc:close ━") {
			t.Errorf("hint: %q", b)
		}
		ta.press(k)
		if ta.topView() != fv || formStop(fv) != "error" || r.port != "2222" || !strings.Contains(ta.screenText(), "2222") {
			t.Errorf("%s closes it, back to the form, the focus where it was, the values kept: %T %s", k, ta.topView(), formStop(fv))
		}
	}
}

// --- toast -------------------------------------------------------------------

func TestToastWrapThreeLines(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(60, 30)
	// body is the toast's text rows, inside the frame.
	body := func(text string) []string {
		ta.Toast(Info, text)
		ta.settle()
		_, plain := covEBox(ta, ta.toast)
		var out []string
		for _, row := range plain[2 : len(plain)-2] {
			out = append(out, covEInside(row, ta.toast.w))
		}
		return out
	}
	centred := func(row string) bool {
		left, right := len(row)-len(strings.TrimLeft(row, " ")), len(row)-len(strings.TrimRight(row, " "))
		return right-left == 0 || right-left == 1
	}
	if rows := body("Copied 3 paths"); len(rows) != 1 || !centred(rows[0]) {
		t.Errorf("one short line, centred: %q", rows)
	}
	rows := body("Copied the paths of every file under the directory to the clipboard, ready to paste")
	if len(rows) != 2 || !centred(rows[0]) || !centred(rows[1]) || strings.Contains(strings.Join(rows, ""), "…") {
		t.Errorf("a long text wraps, each line centred: %q", rows)
	}
	rows = body(strings.Repeat("Copied the paths of every file under the directory. ", 6))
	if len(rows) != 3 || !strings.HasSuffix(strings.TrimRight(rows[2], " "), "…") {
		t.Errorf("three lines at most, the third ending in …: %q", rows)
	}
}

func TestToastKinds(t *testing.T) {
	for _, c := range []struct {
		kind         ToastKind
		glyph, title string
		color        Color
	}{
		{Info, "\U0000F024", "INFO", Text}, {Warning, "\U000F078F", "WARNING", Peach}, {Error, "\U000F0238", "ERROR", Red},
	} {
		ta := newTestApp(t)
		ta.Toast(c.kind, "Copied 3 paths")
		ta.settle()
		styled, plain := covEBox(ta, ta.toast)
		if !strings.Contains(plain[0], c.glyph+" "+c.title) {
			t.Errorf("%s: title %q", c.title, plain[0])
		}
		if !strings.Contains(styled[2], Fg(c.color).Render("Copied 3 paths")) {
			t.Errorf("%s: the text is in the wrong colour: %q", c.title, styled[2])
		}
	}
}

func TestToastDuration(t *testing.T) {
	for _, c := range []struct {
		kind ToastKind
		want time.Duration
	}{{Info, 2200 * time.Millisecond}, {Warning, 4400 * time.Millisecond}, {Error, 4400 * time.Millisecond}} {
		if got := c.kind.duration(); got != c.want {
			t.Errorf("%s: %v, want %v", c.kind.title(), got, c.want)
		}
	}
	ta := newTestApp(t)
	ta.Toast(Info, "Copied 3 paths")
	ta.settle()
	ta.Update(toastDone{ta.toast.view.(*toastView).id})
	ta.settle()
	if ta.toast != nil {
		t.Error("the toast stays after its time")
	}
}

func TestToastReplace(t *testing.T) {
	ta := newTestApp(t)
	ta.Toast(Info, "Copied 3 paths")
	ta.settle()
	first := ta.toast.view.(*toastView).id
	ta.Toast(Warning, "Disk almost full")
	ta.settle()
	text := ta.screenText()
	if !strings.Contains(text, "Disk almost full") || strings.Contains(text, "Copied 3 paths") || strings.Contains(text, "INFO") {
		t.Errorf("the second toast takes the first one's place:\n%s", text)
	}
	ta.Update(toastDone{first})
	ta.settle()
	if ta.toast == nil {
		t.Fatal("the first toast's time closed the second: its time starts again")
	}
	ta.Update(toastDone{ta.toast.view.(*toastView).id})
	ta.settle()
	if ta.toast != nil {
		t.Error("the second toast stays after its own time")
	}
}

func TestToastEscFirst(t *testing.T) {
	ta := newTestApp(t)
	ta.press("Space")
	ta.Toast(Info, "Copied 3 paths")
	ta.settle()
	if _, plain := covEBox(ta, ta.toast); !strings.HasPrefix(plain[len(plain)-1], "┗━ Esc:close ━") {
		t.Errorf("hint: %q", plain[len(plain)-1])
	}
	ta.Update(keyMsg("Esc"))
	if !ta.toast.closing {
		t.Error("Esc closes the toast at once")
	}
	ta.settle()
	if _, ok := ta.topView().(*menuView); !ok || ta.toast != nil {
		t.Fatalf("Esc takes the toast before the popup under it: %T", ta.topView())
	}
	ta.press("Esc")
	if ta.top() != nil {
		t.Error("the next Esc closes the popup")
	}
}

func TestToastKeysPassThrough(t *testing.T) {
	// In a PTY Esc goes to the program too: TestToastPositionOverTerminal.
	ta := newTestApp(t)
	ta.Toast(Info, "Copied 3 paths")
	ta.settle()
	ta.press("j")
	if r, _ := ta.list.Current(); r.ID != "1" || ta.toast == nil || ta.toast.closing {
		t.Errorf("j goes through the toast to the list: on %s", r.ID)
	}
	ta.press("Space")
	m, ok := ta.topView().(*menuView)
	if !ok || ta.toast == nil || ta.toast.closing {
		t.Fatalf("Space goes through the toast: %T", ta.topView())
	}
	before := m.cursor
	ta.press("j")
	if m.cursor == before || ta.toast == nil || ta.toast.closing {
		t.Error("j goes through the toast to the menu")
	}
}
