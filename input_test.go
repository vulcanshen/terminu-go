package terminu

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// inputOpeners open every kind of input popup, and some stacked ones, for the
// L4 screen test.
func inputOpeners(t *testing.T) map[string]func(*testApp) {
	dir := fakeTree(t)
	long := make([]string, 60)
	for i := range long {
		long[i] = "option number " + strings.Repeat("x", i)
	}
	return map[string]func(*testApp){
		"text": func(ta *testApp) { ta.Text(TextInput{Title: "Rename", Prompt: "New name", Old: "assets"}) },
		"pin": func(ta *testApp) {
			ta.Password(PasswordInput{Title: "PIN", Prompt: "Up to 12 digits", Unlock: true, MaxLen: 12})
		},
		"number": func(ta *testApp) { ta.Number(NumberInput{Title: "Port", Prompt: "1-65535", Min: 1, Max: 65535}) },
		"select": func(ta *testApp) { ta.Select(Select{Title: "Profile", Glyph: glyphMenu, Options: long, Selected: 30}) },
		"checks": func(ta *testApp) { ta.Checks(Checks{Title: "Namespaces", Options: []string{"a", "b", "c"}}) },
		"find":   func(ta *testApp) { ta.Find(testFinder()) },
		"files":  func(ta *testApp) { ta.PickFile(FilePicker{Title: "Identity file", Root: dir}) },
		"textarea": func(ta *testApp) {
			ta.Textarea(Textarea{Title: "Comment", Old: strings.Repeat("a long line ", 30) + "\nend"})
		},
		"date":     func(ta *testApp) { ta.PickDate(DatePicker{Title: "Due date"}) },
		"datetime": func(ta *testApp) { ta.PickDate(DatePicker{Title: "Meeting", Time: true}) },
		"color":    func(ta *testApp) { ta.PickColor(ColorPicker{Title: "Background", Value: Hex("#2A2A3C")}) },
		"reorder":  func(ta *testApp) { ta.Reorder(Reorder{Title: "Order", Options: long}) },
		"form":     func(ta *testApp) { ta.Form(testForm(nil)) },
		"form+select": func(ta *testApp) {
			ta.Form(testForm(nil))
			ta.settle()
			ta.press("G", "k", "Enter")
		},
	}
}

// fakeTree is a directory for the file-picker:
//
//	config  id_ed25519  work/id_work  work/keys/a.pem  old/b.pem
func fakeTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for name, text := range map[string]string{
		"config": "Host web\n", "id_ed25519": "KEY\n", "work/id_work": "KEY\n", "work/keys/a.pem": "PEM\n", "old/b.pem": "PEM\n",
	} {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func testFinder() *Finder {
	var items []FinderItem
	for _, n := range []string{"README.md", "README-zh_TW.md", "docs/readme-notes.md", "main.go", "go.mod"} {
		items = append(items, FinderItem{ID: n, Text: n})
	}
	return &Finder{Title: "Search", Items: items, Preview: func(it FinderItem) (string, []string) {
		return it.Text, []string{"line one of " + it.Text}
	}}
}

type formResult struct {
	name, port string
	submitted  int
}

// testForm is NEW HOST: Name (text), Port (number), Auth (radio), Agent (a
// checkbox), Profile (select), Command (stacked).
func testForm(r *formResult) *Form {
	if r == nil {
		r = &formResult{}
	}
	profile := 0
	auth := &FormField{Label: "Auth", Options: []string{"password", "privatekey", "credential"}, Selected: 0}
	return &Form{
		Title: "New host", Name: "New host", Button: "Save",
		Fields: []*FormField{
			{Label: "Name", Value: func() string { return r.name }, Empty: "required",
				Edit: func() Input {
					return TextInput{Title: "Name", Prompt: "A name", Old: r.name, Done: func(v string) tea.Cmd { r.name = v; return nil }}
				},
				Check: func() string {
					if r.name == "" {
						return "Name is required"
					}
					return ""
				}},
			{Label: "Port", Value: func() string { return r.port }, Empty: "22",
				Edit: func() Input {
					return NumberInput{Title: "Port", Prompt: "1-65535", Min: 1, Max: 65535, Old: r.port,
						Done: func(v string) tea.Cmd { r.port = v; return nil }}
				}},
			auth,
			{Label: "Agent", Options: []string{"forward"}, Multi: true},
			{Label: "Profile", Value: func() string { return []string{"a", "b"}[profile] },
				Edit: func() Input {
					return Select{Title: "Profile", Options: []string{"a", "b"}, Selected: profile,
						Done: func(i int) tea.Cmd { profile = i; return nil }}
				}},
			{Label: "Command", Stacked: true, Value: func() string { return "tmux new-session -A -s main" },
				Check: func() string {
					if auth.Selected == 2 {
						return "Command can't be used with a credential"
					}
					return ""
				}},
		},
		Submit: func() (tea.Cmd, error) { r.submitted++; return nil, nil },
	}
}

func TestSelect(t *testing.T) {
	ta := newTestApp(t)
	got := -1
	sel := Select{Title: "Profile", Options: []string{"clock", "matrix", "pipes"}, Keys: []string{"1", "2", "3"}, Selected: 1,
		Done: func(i int) tea.Cmd { got = i; return nil }}
	ta.Select(sel)
	ta.settle()
	f := ta.topView().(*finderView)
	if !f.onList || f.cursor != 1 {
		t.Fatalf("a select opens on the list at the value: onList %v, cursor %d", f.onList, f.cursor)
	}
	ta.press("Enter")
	if got != -1 || ta.top() != nil {
		t.Errorf("Enter on the value now closes without writing: got %d, top %v", got, ta.top())
	}
	ta.Select(sel)
	ta.settle()
	ta.press("j", "Enter")
	if got != 2 {
		t.Errorf("j Enter chose %d, want 2", got)
	}
	ta.Select(sel)
	ta.settle()
	ta.press("1")
	if got != 0 {
		t.Errorf("the hotkey 1 chose %d, want 0", got)
	}
}

func TestCheckboxPopup(t *testing.T) {
	ta := newTestApp(t)
	var got []bool
	c := Checks{Title: "Namespaces", Options: []string{"default", "kube-system", "monitoring"}, Checked: []bool{true, false, false},
		Done: func(v []bool) tea.Cmd { got = v; return nil }}
	ta.Checks(c)
	ta.settle()
	ta.press("j", "Space", "j", "Space")
	if got != nil {
		t.Fatal("Space wrote the value back")
	}
	ta.press("Esc")
	if got != nil || ta.top() != nil {
		t.Fatal("Esc did not cancel")
	}
	ta.Checks(c)
	ta.settle()
	ta.press("j", "Space", "j", "Space", "Enter")
	if !slices.Equal(got, []bool{true, true, true}) {
		t.Errorf("Enter wrote %v", got)
	}
	got = nil
	ta.Checks(c)
	ta.settle()
	ta.press("Enter")
	if got != nil || ta.top() != nil {
		t.Error("Enter with nothing changed should just close")
	}
}

// Checks.Cursor opens a checkbox popup with the cursor on that option, in
// sight even past the first page (ccdrawu's CD8: the app decides).
func TestCheckboxPopupCursor(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(80, 20) // too short for all 29
	opts := make([]string, 29)
	for i := range opts {
		opts[i] = fmt.Sprintf("item-%02d", i)
	}
	ta.Checks(Checks{Title: "Show", Options: opts, Checked: make([]bool, len(opts)), Cursor: 20})
	ta.settle()
	fv := ta.topView().(*finderView)
	if i, r, ok := fv.at(); !ok || i != 20 || r.text != "item-20" {
		t.Fatalf("the cursor on %d %q, want item-20", i, r.text)
	}
	if text := ta.screenText(); !strings.Contains(text, "item-20") || strings.Contains(text, "item-00") {
		t.Errorf("scrolled to item-20:\n%s", text)
	}
	ta.press("Space")
	if !fv.checked[20] {
		t.Error("Space did not check the row the cursor opened on")
	}
	// Out of range: the first, as with no Cursor.
	ta.press("Esc")
	ta.Checks(Checks{Title: "Show", Options: opts, Cursor: 99})
	ta.settle()
	if i, _, _ := ta.topView().(*finderView).at(); i != 0 {
		t.Errorf("Cursor out of range opened on %d, want 0", i)
	}
}

func TestFinderTypingEnter(t *testing.T) {
	ta := newTestApp(t)
	opened := ""
	f := testFinder()
	f.Done = func(it FinderItem) tea.Cmd { opened = it.ID; return nil }
	ta.Find(f)
	ta.settle()
	fv := ta.topView().(*finderView)
	if fv.onList || !fv.typing() {
		t.Fatal("a finder opens on its typing row")
	}
	ta.press("q", "?", "z", "z")
	if ta.top() == nil || fv.ed.value() != "q?zz" {
		t.Fatalf("q and ? are characters on the typing row: %q", fv.ed.value())
	}
	ta.press("Enter")
	if fv.onList {
		t.Error("Enter with no results moved to the list")
	}
	for range 4 {
		ta.press("Backspace")
	}
	ta.press("r", "e", "a", "d")
	if len(fv.shown) != 3 {
		t.Errorf("read matches %d, want 3", len(fv.shown))
	}
	ta.press("↓", "Enter")
	if !fv.onList || opened != "" {
		t.Fatalf("Enter on the typing row moves to the list without going: onList %v, opened %q", fv.onList, opened)
	}
	ta.press("Enter")
	if opened != "README-zh_TW.md" {
		t.Errorf("Enter on the list opened %q", opened)
	}
}

func TestFinderTabBlocks(t *testing.T) {
	ta := newTestApp(t)
	ta.Find(testFinder())
	ta.settle()
	fv := ta.topView().(*finderView)
	ta.press("Tab")
	if !fv.onList {
		t.Fatal("Tab did not move to the list")
	}
	ta.press("Tab")
	if fv.onList {
		t.Fatal("Tab did not move back to the typing row")
	}
	ta.press("Tab", "/", "x")
	if fv.onList || fv.ed.value() != "x" {
		t.Errorf("/ goes back to type on: %q", fv.ed.value())
	}
}

func TestFinderTypingAngle(t *testing.T) {
	// The typing row starts with nf-fa-angle_right one cell from the frame:
	// the layer colour while typing, Overlay0 with the rest of the row while
	// the focus is on the list (components/input/finder).
	row := func(lit bool) string {
		ed := &lineEditor{}
		return typingRow(ed, 30, "3/41", lit, LayerColor(1))
	}
	if r := row(true); !strings.HasPrefix(r, " "+Fg(LayerColor(1)).Render(glyphTyping)+" ") {
		t.Errorf("typing: %q", r)
	}
	if r := row(false); !strings.HasPrefix(r, " "+Fg(Overlay0).Render(glyphTyping)+" ") || !strings.Contains(r, Fg(Overlay0).Render("3/41")) {
		t.Errorf("on the list: %q", r)
	}
	if w := Width(row(true)); w != 30 {
		t.Errorf("the typing row is %d wide", w)
	}
}

func TestFinderPreview(t *testing.T) {
	for _, sz := range [][2]int{{120, 30}, {80, 30}} {
		ta := newTestApp(t)
		ta.resize(sz[0], sz[1])
		ta.Find(testFinder())
		ta.settle()
		text := ta.screenText()
		tops := strings.Count(text, "┏")
		if tops < 2 {
			t.Errorf("%dx%d: %d frames, want the list and the preview", sz[0], sz[1], tops)
		}
		lines := strings.Split(text, "\n")
		side := false
		for _, l := range lines {
			if strings.Count(l, "┏") == 2 {
				side = true
			}
		}
		if side != (sz[0] >= 96) {
			t.Errorf("%dx%d: side by side %v", sz[0], sz[1], side)
		}
	}
}

func TestFinderStreaming(t *testing.T) {
	ta := newTestApp(t)
	f := &Finder{Title: "Search", Loading: true, Preview: func(FinderItem) (string, []string) { return "", nil }}
	ta.Find(f)
	ta.settle()
	if !strings.Contains(ta.screenText(), "(indexing…)") {
		t.Error("no (indexing…) before the first result")
	}
	f.Items = append(f.Items, FinderItem{Text: "first"})
	if strings.Contains(ta.screenText(), "(indexing…)") || !strings.Contains(ta.screenText(), "first") {
		t.Error("the first result did not show")
	}
	if !ta.loading() {
		t.Error("a loading finder keeps the icon turning")
	}
}

func TestFilePicker(t *testing.T) {
	dir := fakeTree(t)
	ta := newTestApp(t)
	chosen := ""
	ta.PickFile(FilePicker{Title: "Identity file", Root: dir, Value: filepath.Join(dir, "id_ed25519"),
		Done: func(p string) tea.Cmd { chosen = p; return nil }})
	ta.settle()
	fv := ta.topView().(*finderView)
	_, r, _ := fv.at()
	if r.text != "id_ed25519" {
		t.Fatalf("the cursor starts on the value, not %q", r.text)
	}
	var names []string
	for _, i := range fv.shown {
		names = append(names, fv.rows()[i].text)
	}
	if want := []string{"..", "old", "work", "config", "id_ed25519"}; !slices.Equal(names, want) {
		t.Errorf("rows %v, want %v", names, want)
	}
	if fv.verb(r) != "choose" {
		t.Errorf("Enter on a file is %q", fv.verb(r))
	}
	ta.press("gg", "j", "j") // work
	ta.press("Enter")
	_, r, _ = fv.at()
	if !fv.onList || r.text != "id_work" && r.text != "keys" {
		t.Errorf("into work: cursor on %q", r.text)
	}
	if fv.cursor != 1 {
		t.Errorf("the cursor goes to the first entry under .., not %d", fv.cursor)
	}
	ta.press("Tab")
	for _, c := range dir + "/work/keys/a" {
		ta.press(string(c))
	}
	names = names[:0]
	for _, i := range fv.shown {
		names = append(names, fv.rows()[i].text)
	}
	if !slices.Equal(names, []string{"a.pem"}) {
		t.Errorf("a typed path lists its directory, filtered: %v", names)
	}
	ta.press("Enter", "Enter")
	if chosen != filepath.Join(dir, "work/keys/a.pem") {
		t.Errorf("chose %q", chosen)
	}
}

// TestFilePickerDirMode: choosing a directory lists . → .. → directories, no
// files; Enter on . chooses the directory listed, on another goes into it
// (components/input/file-picker, choosing a directory).
func TestFilePickerDirMode(t *testing.T) {
	dir := fakeTree(t)
	ta := newTestApp(t)
	chosen := ""
	ta.PickFile(FilePicker{Title: "Download to", Root: dir, Value: dir, Dirs: true,
		Done: func(p string) tea.Cmd { chosen = p; return nil }})
	ta.settle()
	fv := ta.topView().(*finderView)
	names := func() []string {
		var out []string
		for _, i := range fv.shown {
			out = append(out, fv.rows()[i].text)
		}
		return out
	}
	at := func() string {
		_, r, _ := fv.at()
		return r.text
	}
	if want := []string{".", "..", "old", "work"}; !slices.Equal(names(), want) {
		t.Errorf("rows %v, want %v: ., .., the directories, no files", names(), want)
	}
	if i, r, _ := fv.at(); r.text != "." || !fv.green(i, r) {
		t.Errorf("opened at the value, the cursor starts on . in Green: %q", r.text)
	}
	if !strings.Contains(ta.screenText(), glyphFolderMove+" . (this directory)") {
		t.Errorf("the first row is . (this directory) with nf-md-folder_move:\n%s", ta.screenText())
	}
	if h := covFHint(ta); !strings.Contains(h, " Enter:choose Tab|/:filter Esc:cancel ") {
		t.Errorf("on .: %s", h)
	}
	ta.press("j")
	if h := covFHint(ta); !strings.Contains(h, " Enter:into Tab|/:filter Esc:cancel ") {
		t.Errorf("on ..: %s", h)
	}
	ta.press("j", "Enter") // into old: only a file in it
	if !slices.Equal(names(), []string{".", ".."}) || at() != "." {
		t.Errorf("into old: rows %v, cursor on %q", names(), at())
	}
	ta.press("j", "Enter") // .. back up
	if at() != "old" {
		t.Errorf("going in, the cursor rests on the first entry under ..: %q", at())
	}
	ta.press("j", "Enter") // into work
	if at() != "keys" {
		t.Errorf("into work: cursor on %q", at())
	}
	ta.press("gg", "Enter")
	if chosen != filepath.Join(dir, "work") || ta.top() != nil {
		t.Errorf("Enter on . chooses the directory listed and closes: %q, top %T", chosen, ta.topView())
	}
	chosen = "unchanged"
	ta.PickFile(FilePicker{Title: "Download to", Root: dir, Value: dir, Dirs: true,
		Done: func(p string) tea.Cmd { chosen = p; return nil }})
	ta.settle()
	ta.press("?")
	if !strings.Contains(ta.screenText(), "on . choose the one listed") {
		t.Errorf("the key reference says what Enter does here:\n%s", ta.screenText())
	}
	ta.press("Esc", "Enter")
	if chosen != "unchanged" || ta.top() != nil {
		t.Errorf("the same directory again does not call Done: %q", chosen)
	}

	// A word or a wildcard typed: no . row; a wildcard finds directories
	// only. A path typed to its / lists the . row again.
	fp := &filePicker{p: FilePicker{Dirs: true}, dir: dir}
	for _, typed := range []string{dir + "/", dir + "/o", dir + "/.", dir + "/**/*"} {
		var here, file bool
		for _, r := range fp.list(typed) {
			here = here || r.text == "."
			file = file || !r.dir
		}
		if wantHere := strings.HasSuffix(typed, "/"); here != wantHere || file {
			t.Errorf("typed %q: a . row %v (want %v), a file %v", typed, here, wantHere, file)
		}
	}
}

func TestFilePickerPathRules(t *testing.T) {
	dir := fakeTree(t)
	fp := &filePicker{p: FilePicker{}, dir: dir}
	if rows := fp.list(dir + "/work/../old/"); len(rows) != 0 {
		t.Errorf("a path with .. lists %d rows, want none", len(rows))
	}
	if rows := fp.list(dir + "/nope/x"); len(rows) != 0 {
		t.Errorf("a missing directory lists %d rows", len(rows))
	}
	var found []string
	for _, r := range fp.list(dir + "/**/*.pem") {
		found = append(found, r.text)
	}
	slices.Sort(found)
	if !slices.Equal(found, []string{"old/b.pem", "work/keys/a.pem"}) {
		t.Errorf("**/*.pem found %v", found)
	}
	found = found[:0]
	for _, r := range fp.list(dir + "/*/id_*") {
		found = append(found, r.text)
	}
	if !slices.Equal(found, []string{"work/id_work"}) {
		t.Errorf("*/id_* found %v (a * stays in one directory)", found)
	}
}

func TestGlobMatch(t *testing.T) {
	cases := []struct {
		glob, name string
		want       bool
	}{
		{"*.png", "a.png", true}, {"*.png", "x/a.png", false}, {"**/*.pem", "a.pem", true},
		{"**/*.pem", "x/y/a.pem", true}, {"x/**", "x/y/z", true}, {"?.go", "a.go", true}, {"?.go", "ab.go", false},
	}
	for _, c := range cases {
		if got := globMatch(c.glob, c.name); got != c.want {
			t.Errorf("globMatch(%q, %q) = %v", c.glob, c.name, got)
		}
	}
}

// TestFormTabBlocks: Tab|Shift-Tab go between the field area and the action
// row, back to the item left; the error row goes with the button
// (components/dialog/form, Rules K2).
func TestFormTabBlocks(t *testing.T) {
	ta := newTestApp(t)
	ta.Form(testForm(nil))
	ta.settle()
	fv := ta.topView().(*formView)
	for _, at := range [][]string{{"gg"}, {"gg", "j", "j", "j"}, {"gg", "j", "j", "j", "j", "j"}, {"G", "k"}} {
		ta.press(at...)
		left := formStop(fv)
		for _, tab := range []string{"Tab", "Shift-Tab"} {
			ta.press(tab)
			if formStop(fv) != "button" {
				t.Errorf("%s on %s goes to %s, want the button", tab, left, formStop(fv))
			}
			ta.press(tab)
			if formStop(fv) != left {
				t.Errorf("%s on the button goes to %s, want %s, the item left", tab, formStop(fv), left)
			}
		}
	}
	for _, c := range []struct{ keys, want string }{
		{"gg k", "Command"}, // k wraps within the field area, never to the button
		{"G", "Command"},    // G is the field area's last item
		{"Tab k", "button"}, // the action area wraps within itself
	} {
		ta.press(strings.Fields(c.keys)...)
		if formStop(fv) != c.want {
			t.Errorf("%s: %s, want %s", c.keys, formStop(fv), c.want)
		}
	}
	ta.press("Enter") // Name is required: the focus on Name
	ta.press("j", "j", "Tab")
	if formStop(fv) != "button" {
		t.Errorf("Tab from the field area goes to the button: %s", formStop(fv))
	}
	ta.press("k", "Tab")
	if formStop(fv) != "Auth:0" {
		t.Errorf("Tab on the button returns to the item left: %s", formStop(fv))
	}
	ta.press("gg", "e", "Tab")
	if formStop(fv) != "button" {
		t.Errorf("the error row is in the field area: Tab goes to the actions: %s", formStop(fv))
	}
	ta.press("Tab")
	if formStop(fv) != "error" {
		t.Errorf("and Tab comes back to the error row: %s", formStop(fv))
	}
}

// TestFormCheckboxSpace: on a checkbox option drawn on a form Space ticks it
// as Enter does; it does nothing elsewhere on the form (Rules K5).
func TestFormCheckboxSpace(t *testing.T) {
	ta := newTestApp(t)
	f := testForm(nil)
	auth, agent := f.Fields[2], f.Fields[3]
	ta.Form(f)
	ta.settle()
	fv := ta.topView().(*formView)
	ta.press("j", "j", "j", "j", "j", "Space") // Agent: forward
	if len(agent.Checked) == 0 || !agent.Checked[0] || formStop(fv) != "Agent:0" || len(ta.stack) != 1 {
		t.Fatalf("Space on a checkbox option ticks it in place: %v, %s, %d layers", agent.Checked, formStop(fv), len(ta.stack))
	}
	ta.press("Space")
	if agent.Checked[0] {
		t.Error("Space again unticks it")
	}
	ta.press("k", "Space") // Auth: credential, a radio
	if auth.Selected != 0 || formStop(fv) != "Auth:2" {
		t.Errorf("Space on a radio option does nothing: selected %d", auth.Selected)
	}
	for _, at := range []string{"gg", "G"} {
		ta.press(at, "Space")
		if ta.topView() != fv || len(ta.stack) != 1 {
			t.Errorf("Space on %s does nothing: %T", formStop(fv), ta.topView())
		}
	}
	ta.press("gg", "?")
	ref := ta.screenText()
	for _, want := range []string{"Space", "toggle the checkbox option", "Tab|Shift-Tab", "between the fields and the button"} {
		if !strings.Contains(ref, want) {
			t.Errorf("the key reference has no %q:\n%s", want, ref)
		}
	}
}

func formStop(fv *formView) string {
	switch fv.focus.kind {
	case itemButton:
		return "button"
	case itemError:
		return "error"
	}
	if fv.focus.option >= 0 {
		return fv.focus.field.Label + ":" + string(rune('0'+fv.focus.option))
	}
	return fv.focus.field.Label
}

func TestFormSubmitChecks(t *testing.T) {
	ta := newTestApp(t)
	r := &formResult{}
	ta.Form(testForm(r))
	ta.settle()
	fv := ta.topView().(*formView)
	ta.press("gg", "j", "j", "j", "j", "Enter") // Auth: credential
	ta.press("Tab", "Enter")
	if r.submitted != 0 {
		t.Fatal("the form was submitted with errors")
	}
	if formStop(fv) != "Name" || fv.errOn.Label != "Name" {
		t.Errorf("the focus goes to the first problem: %s, error row %v", formStop(fv), fv.errOn.Label)
	}
	if len(fv.errs) != 2 {
		t.Errorf("every field with a problem is marked: %v", fv.errs)
	}
	ta.press("e")
	if formStop(fv) != "error" {
		t.Errorf("e on a wrong field goes to the error row: %s", formStop(fv))
	}
	ta.press("e")
	if formStop(fv) != "Name" {
		t.Errorf("e on the error row goes back: %s", formStop(fv))
	}
	ta.press("j", "j", "j", "j", "j", "j", "j", "e") // Command
	if fv.errOn.Label != "Command" {
		t.Errorf("e shows this field's message on the error row: %s", fv.errOn.Label)
	}
	ta.press("e", "Enter")
	if _, ok := ta.topView().(*lineInput); ok {
		t.Fatal("Enter on Command opened an input")
	}
	// Fix both and submit.
	r.name = "web-01"
	ta.press("gg", "j", "j", "Enter", "Tab", "Enter")
	if r.submitted != 1 {
		t.Errorf("submitted %d times", r.submitted)
	}
}

func TestFormDiscardConfirm(t *testing.T) {
	ta := newTestApp(t)
	ta.Form(testForm(nil))
	ta.settle()
	ta.press("Esc")
	if ta.top() != nil {
		t.Fatal("Esc on an unchanged form closes it")
	}
	ta.Form(testForm(nil))
	ta.settle()
	ta.press("Enter", "w", "e", "b", "Enter") // Name: web
	ta.press("Esc")
	c, ok := ta.topView().(*confirmView)
	if !ok || !strings.Contains(c.ask.Question, "Discard changes to New host?") {
		t.Fatalf("Esc on a changed form asks first: %T", ta.topView())
	}
	ta.press("Esc")
	if _, ok := ta.topView().(*formView); !ok {
		t.Fatal("Esc on the confirm goes back to the form")
	}
	ta.press("Esc", "Enter")
	if ta.top() != nil {
		t.Error("Enter on the confirm discards the form")
	}
}

func TestFormTypingNote(t *testing.T) {
	ta := newTestApp(t)
	ta.Form(testForm(nil))
	ta.settle()
	ta.press("x")
	n, ok := ta.topView().(*noteView)
	if !ok || !strings.Contains(StripStyles(strings.Join(n.lines, "")), "Press [Enter] to edit Name.") {
		t.Fatalf("typing on Name opens the note: %T", ta.topView())
	}
	ta.press("Enter")
	ta.press("j", "j", "q")
	if _, ok := ta.topView().(*formView); !ok || ta.quitting {
		t.Errorf("q on an option does nothing: %T, quitting %v", ta.topView(), ta.quitting)
	}
}

func TestTextarea(t *testing.T) {
	ta := newTestApp(t)
	got := ""
	ta.Textarea(Textarea{Title: "Comment", Old: "one", Done: func(v string) tea.Cmd { got = v; return nil }})
	ta.settle()
	tv := ta.topView().(*textareaView)
	if !tv.typing() || tv.col != 3 {
		t.Fatalf("a textarea opens writing with the cursor at the end: %v %d", tv.typing(), tv.col)
	}
	ta.press("Enter", "Tab", "t", "w", "o", "Space", "q", "?")
	if tv.value() != "one\n    two q?" {
		t.Fatalf("Enter breaks the line, Tab indents, q and ? are characters: %q", tv.value())
	}
	if l := ta.top(); l.h != ta.h-2 {
		t.Errorf("a textarea is as tall as it can be: %d, want %d", l.h, ta.h-2)
	}
	lavender := "48;2;180;190;254"
	if !strings.Contains(strings.Join(ta.Frame(), ""), lavender) {
		t.Error("the writing cursor is not Lavender")
	}
	ta.press("Esc")
	if tv.typing() {
		t.Fatal("Esc moves to the moving state")
	}
	frame := strings.Join(ta.Frame(), "")
	if strings.Contains(frame, lavender) || !strings.Contains(frame, "48;2;164;192;250") {
		t.Error("the moving cursor is a block of the layer colour")
	}
	ta.press("0", "k", "a", "!", "Esc", "Enter")
	if got != "o!ne\n    two q?" {
		t.Errorf("saved %q", got)
	}
}

func TestTextareaWrapAndNumbers(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(40, 20)
	ta.Textarea(Textarea{Title: "Comment", Old: strings.Repeat("word ", 12) + "\nshort"})
	ta.settle()
	text := ta.screenText()
	if !strings.Contains(text, "┃  1 word") || !strings.Contains(text, "┃  2 short") || !strings.Contains(text, "┃    word word") {
		t.Errorf("wrapped rows have no number, each line's first does:\n%s", text)
	}
	// It folds at a space: a word goes to the next row whole (components,
	// wrapping).
	if strings.Contains(text, "┃    ord") {
		t.Errorf("a word was cut in half:\n%s", text)
	}
}

// TestTextareaUpToShorterRow: a row that folded at a space is shorter than
// the one under it; moving up from far right stops on its last character,
// not on the first of the row it was on.
func TestTextareaUpToShorterRow(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(40, 20)
	ta.Textarea(Textarea{Title: "Comment", Old: "ab " + strings.Repeat("c", 29) + "\nnext"})
	ta.settle()
	tv := ta.topView().(*textareaView)
	ta.press("Esc", "gg", "j", "$", "k")
	if tv.line != 0 || tv.col != 2 {
		t.Errorf("k from the end of the second row: col %d, want 2\n%s", tv.col, ta.screenText())
	}
}

// The line numbers' column is at least two digits: it widens at line 100,
// not at line 10 (components/input/textarea).
func TestTextareaGutterTwoDigits(t *testing.T) {
	start := func(lines int) string {
		ta := newTestApp(t)
		ta.resize(40, 20)
		ta.Textarea(Textarea{Title: "Comment", Old: strings.TrimSuffix(strings.Repeat("x\n", lines), "\n")})
		ta.settle()
		ta.press("Esc", "gg")
		return ta.screenText()
	}
	for _, c := range []struct {
		lines int
		first string
	}{{1, "┃  1 x"}, {9, "┃  1 x"}, {10, "┃  1 x"}, {99, "┃  1 x"}, {100, "┃   1 x"}} {
		if text := start(c.lines); !strings.Contains(text, c.first) {
			t.Errorf("%d lines: want %q\n%s", c.lines, c.first, text)
		}
	}
}

func TestDatetimePicker(t *testing.T) {
	old := clock
	clock = func() time.Time { return time.Date(2026, 10, 7, 14, 12, 0, 0, time.Local) }
	t.Cleanup(func() { clock = old })
	ta := newTestApp(t)
	var got time.Time
	value := time.Date(2026, 10, 15, 10, 7, 0, 0, time.Local)
	ta.PickDate(DatePicker{Title: "Meeting", Time: true, Value: value, Done: func(v time.Time) tea.Cmd { got = v; return nil }})
	ta.settle()
	dv := ta.topView().(*dateView)
	if dv.focus != onDays || dv.day.Day() != 15 || dv.minute != 1 {
		t.Fatalf("opens on the days at the value, the minute before it: %v %v %d", dv.focus, dv.day, dv.minute)
	}
	ta.press("l", "j", "k", "h", "h")
	if dv.day.Day() != 14 {
		t.Errorf("h/l a day, j/k a week: %v", dv.day)
	}
	ta.press("Tab")
	if dv.focus != onTime {
		t.Fatalf("Tab after the days goes to the time: %v", dv.focus)
	}
	ta.press("k", "l", "k")
	if dv.hour != 9 || dv.minute != 0 {
		t.Errorf("time: %d:%d", dv.hour, dv.minute*5)
	}
	ta.press("Tab", "l", "Tab", "Tab")
	if dv.focus != onDays || dv.day.Month() != time.November || dv.day.Day() != 14 {
		t.Errorf("Tab month, l the next month, Tab Tab back to the days: %v %v", dv.focus, dv.day)
	}
	ta.press("t")
	if dv.day.Day() != 7 || dv.day.Month() != time.October {
		t.Errorf("t goes to today: %v", dv.day)
	}
	ta.press("Enter", "Enter")
	if want := time.Date(2026, 10, 7, 9, 0, 0, 0, time.Local); !got.Equal(want) {
		t.Errorf("chose %v, want %v", got, want)
	}
}

func TestDatePickerRange(t *testing.T) {
	ta := newTestApp(t)
	day := time.Date(2026, 10, 15, 0, 0, 0, 0, time.Local)
	ta.PickDate(DatePicker{Title: "Due", Value: day, Min: day.AddDate(0, 0, -1), Max: day.AddDate(0, 0, 3)})
	ta.settle()
	dv := ta.topView().(*dateView)
	ta.press("j")
	if !dv.day.Equal(day) {
		t.Errorf("a week on leaves the range: it stays, not %v", dv.day)
	}
	ta.press("h", "h")
	if dv.day.Day() != 14 {
		t.Errorf("the day before the first in range: it stays, not %v", dv.day)
	}
}

func TestColorPickerEnterChannel(t *testing.T) {
	ta := newTestApp(t)
	var got Color
	ta.PickColor(ColorPicker{Title: "Background", Value: Hex("#2A2A3C"), Done: func(c Color) tea.Cmd { got = c; return nil }})
	ta.settle()
	ta.press("Enter")
	sel, ok := ta.topView().(*finderView)
	if !ok || sel.kind != findSelect {
		t.Fatalf("Enter on R opens a select: %T", ta.topView())
	}
	if _, r, _ := sel.at(); r.text != "2A (42)" {
		t.Errorf("the select starts on the value: %q", r.text)
	}
	ta.press("j", "Enter")
	if got.Set {
		t.Fatal("choosing R chose the colour")
	}
	cv := ta.topView().(*colorView)
	if cv.focus != 0 {
		t.Errorf("the focus stays on R, not %d", cv.focus)
	}
	ta.press("Tab") // only Tab reaches Choose (new takes no focus)
	if cv.focus != colorChoose {
		t.Fatalf("focus %d, want Choose", cv.focus)
	}
	ta.press("Enter")
	if HexText(got) != "#2B2A3C" {
		t.Errorf("chose %s", HexText(got))
	}
}

func TestSlider(t *testing.T) {
	s := Slider{Bottom: 0, Top: 1, Step: 0.05}
	nums := s.numbers()
	if len(nums) != 21 || nums[3] != 0.15 || nums[20] != 1 {
		t.Errorf("0–1 by 0.05: %d numbers, %v %v", len(nums), nums[3], nums[20])
	}
	if d := (Slider{Bottom: 0, Top: 100}).numbers(); len(d) != 11 || d[1] != 10 {
		t.Errorf("the default step is a tenth: %v", d)
	}
	if got := StripStyles(SliderTrack(128, 0, 255, 12, Text)); got != "━━━━━━●━━━━━ 128" {
		t.Errorf("track %q", got)
	}
}

func TestFilePickerHiddenAndOrder(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"beta", "Alpha", ".env", "zeta/x", ".ssh/key"} {
		p := filepath.Join(dir, n)
		_ = os.MkdirAll(filepath.Dir(p), 0o700)
		_ = os.WriteFile(p, nil, 0o600)
	}
	fp := &filePicker{dir: dir}
	var names []string
	for _, r := range fp.list("") {
		names = append(names, r.text)
	}
	if want := []string{"..", "zeta", "Alpha", "beta"}; !slices.Equal(names, want) {
		t.Errorf("rows %v, want %v (directories first, by name ignoring case, no hidden ones)", names, want)
	}
	names = names[:0]
	for _, r := range fp.list(dir + "/.") {
		names = append(names, r.text)
	}
	if !slices.Contains(names, ".ssh") || !slices.Contains(names, ".env") {
		t.Errorf("a word starting with a dot lists the hidden ones: %v", names)
	}
	if rows := fp.list(dir + "/**/key"); len(rows) != 0 {
		t.Errorf("** went into a hidden directory: %v", rows)
	}
}

func TestFormScrollPosition(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(80, 14)
	ta.Form(testForm(nil))
	ta.settle()
	ta.press("j", "j")
	if !strings.Contains(ta.screenText(), "3 of 6") {
		t.Errorf("Auth is the third of six fields:\n%s", ta.screenText())
	}
	ta.press("Tab")
	if strings.Contains(ta.screenText(), " of 6") {
		t.Error("on the button the position is not written")
	}
}

// The grey suggestion starts at the cursor's cell: the cursor sits on its
// first character like any cursor, the rest is Overlay0, cut with "…" when it
// is wider than the row (components/input/README).
func TestSuggestionUnderCursor(t *testing.T) {
	cursor := Style{Fg: Base, Bg: Lavender}
	ta := newTestApp(t)
	ta.Text(TextInput{Title: "Name", Old: "日本語.txt"})
	ta.settle()
	lines, _ := covFDraw(ta)
	if !strings.Contains(lines[4], cursor.Render("日")+Fg(Overlay0).Render("本語.txt")) {
		t.Errorf("a wide first character under the cursor: %q", lines[4])
	}
	if v := covFValue(ta); !strings.HasPrefix(v, "日本語.txt ") {
		t.Errorf("the suggestion starts right after the angle and its space: %q", v)
	}
	// Typing hides it; deleting what was typed brings it back, the cursor on
	// its first character again.
	ta.press("x")
	if v := covFValue(ta); strings.Contains(v, "本") {
		t.Errorf("typing did not hide the suggestion: %q", v)
	}
	ta.press("Backspace")
	if lines, _ := covFDraw(ta); !strings.Contains(lines[4], cursor.Render("日")) {
		t.Errorf("the suggestion came back without the cursor on it: %q", lines[4])
	}
	ta.press("Esc")

	// A control character first: its whole name is the cursor, Base on Red.
	ta.Text(TextInput{Title: "Name", Old: "\x1bweb"})
	ta.settle()
	if lines, _ := covFDraw(ta); !strings.Contains(lines[4], Style{Fg: Base, Bg: Red}.Render(`\x1b`)+Fg(Overlay0).Render("web")) {
		t.Errorf("a control character under the cursor: %q", lines[4])
	}
	ta.press("Esc")

	// Wider than the row: cut at the end with "…".
	ta.resize(40, 20)
	ta.Text(TextInput{Title: "Name", Old: strings.Repeat("long-", 20)})
	ta.settle()
	if v := strings.TrimRight(covFValue(ta), " "); !strings.HasPrefix(v, "long-") || !strings.HasSuffix(v, "…") {
		t.Errorf("a long suggestion: %q", v)
	}
	l := ta.top()
	for i, row := range l.view.draw(l.w, l.h, LayerColor(1)) {
		if Width(row) != l.w {
			t.Errorf("row %d is %d wide, not %d", i, Width(row), l.w)
		}
	}
}

// A password that is set shows eight dots from the cursor's cell on, the
// cursor on the first; the unlock look's are a cell apart and centred, 15
// cells (components/input/password).
func TestOldPasswordDotsCursor(t *testing.T) {
	cursor := Style{Fg: Base, Bg: Lavender}
	ta := newTestApp(t)
	ta.Password(PasswordInput{Title: "Password", Set: true})
	ta.settle()
	lines, _ := covFDraw(ta)
	if !strings.Contains(lines[4], cursor.Render("•")+Fg(Overlay0).Render("•••••••")) || !strings.HasPrefix(covFValue(ta), "•••••••• ") {
		t.Errorf("ordinary: %q", StripStyles(lines[4]))
	}
	ta.press("Esc")

	// A PIN of four: the eight dots are wider than four digits, and the box
	// holds them.
	ta.Password(PasswordInput{Title: "PIN", Unlock: true, MaxLen: 4, Set: true})
	ta.settle()
	lines, _ = covFDraw(ta)
	row := StripStyles(lines[4])
	inner := strings.TrimSuffix(strings.TrimPrefix(row, "┃"), "┃")
	dots := strings.TrimSpace(inner)
	left := len(inner) - len(strings.TrimLeft(inner, " "))
	right := len(inner) - len(strings.TrimRight(inner, " "))
	if dots != "● ● ● ● ● ● ● ●" || Width(dots) != 15 || left-right > 1 || right-left > 1 {
		t.Errorf("unlock: %q (left %d, right %d)", row, left, right)
	}
	if !strings.Contains(lines[4], cursor.Render("●")+Fg(Overlay0).Render(" ● ● ● ● ● ● ●")) {
		t.Errorf("unlock: the cursor is not on the first dot: %q", lines[4])
	}
}

// In the textarea's writing state Home and End go to the start and the end of
// the whole text; the line's ends are 0|$ in the moving state
// (components/input/textarea).
func TestTextareaHomeEnd(t *testing.T) {
	ta := newTestApp(t)
	ta.Textarea(Textarea{Title: "Comment", Old: "ab\ncde\nf"})
	ta.settle()
	tv := ta.topView().(*textareaView)
	at := func(line, col int, what string) {
		t.Helper()
		if tv.line != line || tv.col != col {
			t.Errorf("%s: at %d:%d, want %d:%d", what, tv.line, tv.col, line, col)
		}
	}
	covFPress(ta, "↑", "Home")
	at(0, 0, "Home from the middle line")
	covFPress(ta, "↓", "End")
	at(2, 1, "End from the middle line")
	covFPress(ta, "Esc", "k", "0")
	at(1, 0, "0 in the moving state")
	covFPress(ta, "$")
	at(1, 2, "$ in the moving state")
}

// In a textarea a pasted line break (\n, \r\n, a lone \r) is a line break;
// any other control character stays, drawn as its red name and deleted whole,
// and the text is not saved while it holds one: the error row names the value
// (components/input/textarea).
func TestTextareaPasteControlChars(t *testing.T) {
	ta := newTestApp(t)
	got, calls := "", 0
	ta.Textarea(Textarea{Title: "Notes", Done: func(v string) tea.Cmd { got = v; calls++; return nil }})
	ta.settle()
	tv := ta.topView().(*textareaView)
	covFPaste(ta, "1\r2\r\n3\n4")
	if tv.value() != "1\n2\n3\n4" {
		t.Errorf("line breaks: %q", tv.value())
	}
	covFPaste(ta, "x\x1by\tz")
	if tv.value() != "1\n2\n3\n4x\x1by\tz" {
		t.Errorf("a control character stays, a tab stays: %q", tv.value())
	}
	lines, _ := covFDraw(ta)
	if !strings.Contains(strings.Join(lines, "\n"), Fg(Red).Render(`\x1b`)) {
		t.Errorf("not drawn as its red name:\n%s", strings.Join(lines, "\n"))
	}
	if text := ta.screenText(); !strings.Contains(text, `4x\x1by`) {
		t.Errorf("the name takes its four cells:\n%s", text)
	}

	// Enter in the moving state: held back, the error row in Red.
	ta.press("Esc", "Enter")
	if calls != 0 || ta.top() == nil {
		t.Fatalf("saved with a control character: %q", got)
	}
	msg := "Notes can't have control characters"
	lines, _ = covFDraw(ta)
	l := ta.top()
	if !strings.Contains(lines[l.h-3], Fg(Red).Render(msg)) || !covFBlank(StripStyles(lines[l.h-4])) {
		t.Errorf("the error row, a blank row above it:\n%s", strings.Join(lines[l.h-5:], "\n"))
	}

	// Backspace deletes it whole; then it saves.
	ta.press("i")
	covFPress(ta, "End", "←", "←", "←", "Backspace")
	if tv.value() != "1\n2\n3\n4xy\tz" {
		t.Fatalf("Backspace on the control character: %q", tv.value())
	}
	ta.press("Esc", "Enter")
	if calls != 1 || got != "1\n2\n3\n4xy\tz" || ta.top() != nil {
		t.Errorf("a tab does not hold it back: %d %q", calls, got)
	}
}

// TestCursorOnControlChar: a cursor on a control character makes the whole
// name the block, Base on Red, on a typing row and in both textarea states
// (components/input/README, textarea).
func TestCursorOnControlChar(t *testing.T) {
	block := Style{Fg: Base, Bg: Red}.Render(`\x1b`)
	ta := newTestApp(t)
	ta.Text(TextInput{Title: "Name"})
	ta.settle()
	covFPaste(ta, "a\x1bb")
	covFPress(ta, "←", "←")
	if lines, _ := covFDraw(ta); !strings.Contains(lines[4], block) {
		t.Errorf("a typing row: %q", lines[4])
	}

	ta = newTestApp(t)
	ta.Textarea(Textarea{Title: "Notes"})
	ta.settle()
	covFPaste(ta, "a\x1bb")
	covFPress(ta, "←", "←")
	if lines, _ := covFDraw(ta); !strings.Contains(strings.Join(lines, "\n"), block) {
		t.Errorf("writing in a textarea:\n%s", strings.Join(lines, "\n"))
	}
	ta.press("Esc") // moving: the cursor stays on it
	if lines, _ := covFDraw(ta); !strings.Contains(strings.Join(lines, "\n"), block) {
		t.Errorf("moving in a textarea:\n%s", strings.Join(lines, "\n"))
	}
}

// An option not available in a checkbox popup (components/input/checkbox):
// drawn in Surface2, the tick too, but the cursor stops on it, Space flips it
// and Enter writes back what is on screen; filtering lists it.
func TestCheckboxPopupUnavailable(t *testing.T) {
	ta := newTestApp(t)
	var got []bool
	c := Checks{Title: "Items", Options: []string{"alpha", "beta", "kappa", "delta"}, Checked: []bool{false, false, true, false},
		Unavailable: func(i int) bool { return i == 0 || i == 2 },
		Done:        func(v []bool) tea.Cmd { got = v; return nil }}
	ta.Checks(c)
	ta.settle()
	fv := ta.topView().(*finderView)
	cursorOn := func(want string) {
		t.Helper()
		if _, r, ok := fv.at(); !ok || r.text != want {
			t.Errorf("the cursor on %q (%v), want %q", r.text, ok, want)
		}
	}
	cursorOn("alpha") // the first, not available or not
	ta.press("j")
	lines, _ := covFDraw(ta)
	for _, want := range []string{
		Fg(Surface2).Render(glyphUnchecked) + " " + Fg(Surface2).Render("alpha"),
		Fg(Surface2).Render(glyphMarked) + " " + Fg(Surface2).Render("kappa"),
	} {
		if !strings.Contains(strings.Join(lines, "\n"), want) {
			t.Errorf("not all Surface2: %q\n%s", StripStyles(want), StripStyles(strings.Join(lines, "\n")))
		}
	}
	ta.press("j")
	cursorOn("kappa")
	if row := ta.rowWith("kappa"); strings.Contains(row, "38;2;88;91;112") {
		t.Errorf("the cursor on kappa is drawn in Surface2, not as the cursor block: %q", row)
	}
	ta.press("Space", "Enter")
	if !slices.Equal(got, []bool{false, false, false, false}) {
		t.Errorf("Enter wrote %v: Space unticks kappa", got)
	}
	// Filtering lists it, and Enter on the typing row goes to it.
	ta.Checks(c)
	ta.settle()
	fv = ta.topView().(*finderView)
	ta.press("/", "k", "a", "p", "Enter")
	if fv.typing() || len(fv.shown) != 1 {
		t.Errorf("filtered to kappa: typing %v, %d rows", fv.typing(), len(fv.shown))
	}
	cursorOn("kappa")
}

// TestInputErrEmptyNamesValue: ErrEmpty in the error row names the value, as
// every input error does (components/input/README; sshu SS15).
func TestInputErrEmptyNamesValue(t *testing.T) {
	ta := newTestApp(t)
	notEmpty := func(v string) error {
		if v == "" {
			return ErrEmpty
		}
		return nil
	}
	for _, c := range []struct{ name, want string }{{"Name", "Name can't be empty"}, {"", "Rename can't be empty"}} {
		ta.Text(TextInput{Title: "Rename", Name: c.name, Old: "assets", Check: notEmpty})
		ta.settle()
		ta.press("Backspace", "Enter")
		if li, ok := ta.topView().(*lineInput); !ok || li.err != c.want {
			t.Errorf("Name %q: the error row is %q, want %q", c.name, li.err, c.want)
		}
		ta.covBCloseAll()
	}
}

// TestTextareaCheck: a textarea with a check holds back a changed text that
// fails it, the error row naming the text, the popup open; an unchanged
// text closes unchecked (Rules K3; hostu, sshu SS19).
func TestTextareaCheck(t *testing.T) {
	ta := newTestApp(t)
	got, calls := "", 0
	short := func(v string) error {
		if len(v) > 3 {
			return errors.New("Notes must be three letters at most")
		}
		return nil
	}
	ta.Textarea(Textarea{Title: "Notes", Old: "ab", Check: short, Done: func(v string) tea.Cmd { got = v; calls++; return nil }})
	ta.settle()
	tv := ta.topView().(*textareaView)
	ta.press("c", "d", "Esc", "Enter")
	if calls != 0 || ta.top() == nil || tv.err != "Notes must be three letters at most" {
		t.Fatalf("a text that fails: written %d, error %q", calls, tv.err)
	}
	ta.press("a", "Backspace", "Esc", "Enter")
	if calls != 1 || got != "abc" || ta.top() != nil {
		t.Errorf("a text that passes: written %d %q", calls, got)
	}
	ta.Textarea(Textarea{Title: "Notes", Old: "abcdef", Check: short, Done: func(v string) tea.Cmd { calls++; return nil }})
	ta.settle()
	ta.press("Esc", "Enter")
	if calls != 1 || ta.top() != nil {
		t.Errorf("unchanged: checked or written (%d)", calls)
	}
}

// TestInputSteps: a flow of inputs, each step a popup of its own over the one
// before; Esc goes back a step with its value still there, and the last step
// ends the flow, no step cancelled (Rules F1, F4, T1; locku LK7).
func TestInputSteps(t *testing.T) {
	ta := newTestApp(t)
	var pins []string
	step := func(title string, next func() tea.Cmd) PasswordInput {
		return PasswordInput{Title: title, Stay: next != nil, OnCancel: ta.record("cancel " + title),
			Done: func(v string, _ bool) tea.Cmd {
				pins = append(pins, v)
				if next == nil {
					return ta.CloseTop(2)
				}
				return next()
			}}
	}
	confirm := func() tea.Cmd { return ta.Password(step("Confirm PIN", nil)) }
	newPIN := func() tea.Cmd { return ta.Password(step("New PIN", confirm)) }
	ta.Password(step("Current PIN", newPIN))
	ta.settle()
	current := ta.topView().(*lineInput)
	ta.press("1", "2", "3", "4", "Enter")
	ta.settle()
	if li, ok := ta.topView().(*lineInput); !ok || li.title != "New PIN" || len(ta.stack) != 2 {
		t.Fatalf("after the first step: %T, %d popups", ta.topView(), len(ta.stack))
	}
	ta.press("Esc")
	ta.settle()
	if ta.topView() != current || current.ed.value() != "1234" {
		t.Fatalf("Esc did not go back a step with its value: %T %q", ta.topView(), current.ed.value())
	}
	ta.press("Enter", "5", "6", "7", "8", "Enter", "5", "6", "7", "8", "Enter")
	ta.settle()
	if ta.top() != nil || len(ta.stack) != 0 {
		t.Errorf("the flow is done, %d popups still open", len(ta.stack))
	}
	if !slices.Equal(pins, []string{"1234", "1234", "5678", "5678"}) {
		t.Errorf("the steps got %v", pins)
	}
	if !slices.Equal(ta.ran, []string{"cancel New PIN"}) {
		t.Errorf("ran %v: only the step left with Esc is cancelled", ta.ran)
	}
}
