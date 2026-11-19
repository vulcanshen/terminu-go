package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"
	"github.com/vulcanshen/terminu-go/tdptest"
)

// The first launch of every app, as tdptest walks it.
func TestFirstLaunch(t *testing.T) {
	tdptest.Check(t, tdptest.App{Name: "showcase", New: func() *terminu.App { return newShowcase().app }})
}

// Items in items are walked a level at a time (tdp panel.10, panel-158):
// Enter goes in, the location row naming the path; the filter and N of M
// hold that level only; Esc clears the filter first, then goes up a level
// with the cursor back on the folder it came out of, and at the top does
// nothing.
func TestPanelItemsInItems(t *testing.T) {
	s := newShowcase()
	d := tdptest.Start(t, s.app, 120, 40)
	is := s.inputs
	cur := func() string {
		r, _ := is.fileList.Current()
		return r.ID
	}
	at := func(dir, on string) {
		t.Helper()
		if is.dir != dir || is.fileList.Location != dir || cur() != on {
			t.Fatalf("in %q (location %q) on %q, want in %q on %q", is.dir, is.fileList.Location, cur(), dir, on)
		}
	}
	assets, docs := filepath.Join(projectDir, "assets"), filepath.Join(projectDir, "docs")
	images := filepath.Join(assets, "images")

	d.Press("I", "2", "l", "l") // [2] FILES, its Folders tab
	if is.tab != 2 {
		t.Fatalf("tab %d, want the Folders tab", is.tab)
	}
	at(projectDir, assets) // the folders first, by name
	d.Press("?")
	if d.Has("one level up") {
		t.Error("? has Esc: one level up at the top")
	}
	d.Press("Esc", "Esc")
	at(projectDir, assets)
	if d.Quit() {
		t.Fatal("Esc at the top left the app")
	}

	// A filter typed at the top stays there: docs/ shows all it holds.
	d.Press("/")
	d.Type("doc")
	d.Press("Enter", "Enter") // to the list, then into docs/
	at(docs, filepath.Join(docs, "guide.pdf"))
	if is.fileList.Filtering() {
		t.Error("the filter went into docs/ with it")
	}
	if n := len(is.fileList.Rows()); n != 3 {
		t.Errorf("docs/ has %d rows, want its 3 files", n)
	}
	for _, want := range []string{"guide.pdf", "notes.md", "scan.pdf"} {
		if !d.Has(want) {
			t.Errorf("docs/ has no %q\n%s", want, strings.Join(d.Screen(), "\n"))
		}
	}
	if d.Has("assets/") {
		t.Error("docs/ shows the level above")
	}
	d.Press("Esc")
	at(projectDir, docs)

	// Two levels down and a filter: Esc after Esc to the top.
	d.Press("g", "g", "Enter", "Enter")
	at(images, filepath.Join(images, "backgrounds"))
	d.Press("/")
	d.Type("zz")
	d.Press("Esc")
	if is.fileList.Filtering() {
		t.Fatal("Esc left the filter")
	}
	at(images, filepath.Join(images, "backgrounds"))
	d.Press("?")
	if !d.Has("one level up") {
		t.Error("? has no Esc: one level up below the top")
	}
	d.Press("Esc", "Esc")
	at(assets, images)
	d.Press("Esc")
	at(projectDir, assets)
	d.Press("Esc")
	at(projectDir, assets)
}

// [2] LOG has a status of its own on its frame (tdp panel.1): Status puts it
// in the next corner each time, from the top left round to the bottom right,
// then takes it off; Status trouble turns it from fine to worth noticing to
// gone wrong and back, and cannot be chosen with no status; it says when the
// last line came in, following the log.
func TestLogFrameStatus(t *testing.T) {
	s := newShowcase()
	d := tdptest.Start(t, s.app, 160, 40)
	ps := s.panels
	act := func(label string) func() tea.Cmd {
		for _, a := range ps.log.Actions {
			if a.Label == label {
				return a.Do
			}
		}
		t.Fatalf("no %s on [2] LOG", label)
		return nil
	}
	shown := func() bool { return d.Has(glyphSync + " " + ps.log.FrameStatus.Text) }
	d.Press("P", "2", "Space")
	if !d.Has("Status trouble") || !d.Has("in the next corner of the frame") {
		t.Fatal("its Space menu has Status and Status trouble")
	}
	d.Press("Esc")
	if ps.log.FrameStatus != (terminu.FrameStatus{}) || d.Has(glyphSync) {
		t.Fatal("none to start with")
	}
	for _, corner := range []terminu.Corner{terminu.TopLeft, terminu.TopRight, terminu.BottomLeft, terminu.BottomRight} {
		d.Do(act("Status"))
		if st := ps.log.FrameStatus; st.Corner != corner || st.Text != ps.lastRead || !shown() {
			t.Errorf("corner %d: %+v, shown %v", corner, st, shown())
		}
	}
	d.Do(act("Status"))
	if ps.log.FrameStatus.Text != "" || d.Has(glyphSync) {
		t.Errorf("after the bottom right, none: %+v", ps.log.FrameStatus)
	}
	d.Do(act("Status"))
	for _, kind := range []terminu.StatusKind{terminu.StatusWarning, terminu.StatusError, terminu.StatusFine} {
		d.Do(act("Status trouble"))
		if ps.log.FrameStatus.Kind != kind || !shown() {
			t.Errorf("trouble: %d, want %d", ps.log.FrameStatus.Kind, kind)
		}
	}
	d.Advance(time.Second)
	if ps.log.FrameStatus.Text != ps.lastRead || !shown() {
		t.Errorf("it follows the log: %q, last read %q", ps.log.FrameStatus.Text, ps.lastRead)
	}
}

// The watched file's viewer is read again every 3 seconds (tdp note.9):
// longer, shorter, its lines changed, then gone a while with the app's own
// Error toast, and round again; once the viewer has closed, the reading
// stops.
func TestWatchedFile(t *testing.T) {
	s := newShowcase()
	d := tdptest.Start(t, s.app, 120, 40)
	ds := s.dialogs
	d.Do(ds.watchedFile)
	v := ds.watched
	for _, want := range []int{60, 15, 40} {
		d.Advance(3 * time.Second)
		if len(v.Lines) != want || !d.Has("notes.txt") {
			t.Fatalf("read again: %d lines, want %d", len(v.Lines), want)
		}
	}
	d.Advance(3 * time.Second)
	if !d.Has("Cannot read notes.txt: it is gone") || len(v.Lines) != 40 {
		t.Errorf("gone: no Error toast, or the lines went: %d", len(v.Lines))
	}
	d.Advance(3 * time.Second)
	if len(v.Lines) != 40 || ds.version != 5 {
		t.Errorf("round again: %d lines, version %d", len(v.Lines), ds.version)
	}
	for i := 0; i < 3 && s.app.Covered(); i++ { // the Error toast, then the viewer
		d.Press("Esc")
	}
	d.Advance(3 * time.Second)
	d.Advance(3 * time.Second)
	if ds.watched != nil || ds.version != 5 {
		t.Errorf("closed, it is still read: version %d", ds.version)
	}
}

// A pretend settings file with two values it cannot use is copied to .bak
// and written again with the defaults, said in one Warning toast; in a
// folder that cannot be written there is no copy, the file is left as it
// was and an Error toast says so (tdp E3, E3-28 to E3-32).
func TestValuesDidNotFit(t *testing.T) {
	s := newShowcase()
	d := tdptest.Start(t, s.app, 140, 40)
	path := filepath.Join(dataDir(), "fit", "config.yaml")
	d.Do(func() tea.Cmd { return s.dialogs.didNotFit(false) })
	if !d.Has("2 values in config.yaml did not fit: defaults used, yours kept in config.yaml.bak") {
		t.Errorf("no Warning toast:\n%s", strings.Join(d.Screen(), "\n"))
	}
	kept, _ := os.ReadFile(path + ".bak")
	now, _ := os.ReadFile(path)
	if !strings.Contains(string(kept), "#zzzzzz") || !strings.Contains(string(now), "#89b4fa") {
		t.Errorf("the copy %q, the file %q", kept, now)
	}
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		return // a folder that cannot be written
	}
	d.Advance(5 * time.Second)
	d.Do(func() tea.Cmd { return s.dialogs.didNotFit(true) })
	if !d.Has("config.yaml.bak could not be made: permission denied; config.yaml left as it was") {
		t.Errorf("no Error toast:\n%s", strings.Join(d.Screen(), "\n"))
	}
	if _, err := os.Lstat(path + ".bak"); err == nil {
		t.Error("a copy was made")
	}
	if now, _ := os.ReadFile(path); !strings.Contains(string(now), "#zzzzzz") {
		t.Errorf("the file was written again: %q", now)
	}
}

// A success too long for a toast opens a viewer that Enter closes too
// (tdp toast.2, toast-28).
func TestSuccessReport(t *testing.T) {
	s := newShowcase()
	d := tdptest.Start(t, s.app, 120, 40)
	d.Do(s.dialogs.report)
	if !d.Has("EXPORTED") || !d.Has("Enter|Esc:close") || !d.Has("events since last week.csv") {
		t.Fatalf("no report:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Press("Enter")
	if s.app.Covered() {
		t.Error("Enter did not close it")
	}
}

// A setting naming one of the user's own profiles that is not there is not a
// value that does not fit: shown as written, its label Red, a Warning toast
// that the App Log keeps as a WARN; once it is back, it is used again
// (tdp E3, E3-33, E3-34).
func TestProfileGone(t *testing.T) {
	s := newShowcase()
	d := tdptest.Start(t, s.app, 140, 40)
	d.Press("3") // SETTINGS focused, its labels not dimmed
	var gone terminu.Action
	for _, act := range s.panels.settings.Actions {
		if act.Label == "Profile gone" {
			gone = act
		}
	}
	label := func(c terminu.Color) bool {
		open, _, _ := strings.Cut(terminu.Fg(c).Render("PROFILE"), "PROFILE") // the label is padded
		return strings.Contains(strings.Join(d.Styled(), "\n"), open+"PROFILE")
	}
	if !label(terminu.Blue) {
		t.Fatalf("no Profile row:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Do(gone.Do)
	const said = `Profile "work" in config.yaml is not in profiles.yaml: using the default`
	if !d.Has(said) {
		t.Errorf("no Warning toast:\n%s", strings.Join(d.Screen(), "\n"))
	}
	if !label(terminu.Red) {
		t.Errorf("the label is not Red:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Advance(5 * time.Second)
	d.Press("!")
	if !d.Has("WARN " + said) {
		t.Errorf("no WARN in the App Log:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Press("Esc")
	d.Do(gone.Do)
	if !label(terminu.Blue) {
		t.Errorf("back, the label is still Red:\n%s", strings.Join(d.Screen(), "\n"))
	}
}

// Several things at one start come as one toast, a row each, its kind the
// strongest; the App Log has an entry each (tdp E3, E3-37).
func TestSeveralAtOneStart(t *testing.T) {
	s := newShowcase()
	d := tdptest.Start(t, s.app, 160, 40)
	const gone = `Profile "work" in config.yaml is not in profiles.yaml: using the default`
	d.Do(func() tea.Cmd {
		return s.dialogs.didNotFit(false, terminu.Notice{Kind: terminu.Warning, Text: gone})
	})
	fit := "2 values in config.yaml did not fit: defaults used, yours kept in config.yaml.bak"
	if !d.Has("WARNING") || !d.Has(fit) || !d.Has(gone) {
		t.Errorf("not one toast with both rows:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Advance(5 * time.Second)
	d.Press("!")
	for _, want := range []string{"WARN " + fit, `WARN config.yaml: color "#zzzzzz" did not fit`, "WARN " + gone} {
		if !d.Has(want) {
			t.Errorf("no %q in the App Log:\n%s", want, strings.Join(d.Screen(), "\n"))
		}
	}
}

// The Folders tab compares two folders as rows that hold many (tdp panel.9,
// panel-176 to panel-179): the items that differ, each with its mark, the
// location row the two sides; Enter goes into a folder that differs inside
// or opens a file's comparison, titled with its two sides; Esc goes back up
// a level at a time, the last back to the mode with the anchor kept.
func TestCompareFolders(t *testing.T) {
	s := newShowcase()
	d := tdptest.Start(t, s.app, 140, 40)
	is := s.inputs
	d.Press("I", "2", "l", "l", "/")
	d.Type("rel")
	d.Press("Enter", "Enter") // into releases/
	if is.dir != filepath.Join(projectDir, "releases") {
		t.Fatalf("in %q", is.dir)
	}
	d.Press("C", "j", "Enter") // v1/ against v2/
	if is.files.InMode() == nil || !d.Has("releases/v1 ↔ releases/v2") {
		t.Fatalf("did not go in:\n%s", strings.Join(d.Screen(), "\n"))
	}
	for _, want := range []string{"+ NEWS.md", "~ conf/", "~ logo.png", "~ main.go", "- old-notes.txt"} {
		if !d.Has(want) {
			t.Errorf("no %q:\n%s", want, strings.Join(d.Screen(), "\n"))
		}
	}
	if d.Has("README.md") || d.Has("docs/") {
		t.Error("an item the same on both sides is listed")
	}
	if styled := strings.Join(d.Styled(), "\n"); !strings.Contains(styled, terminu.Fg(terminu.Red).Render("-")) || !strings.Contains(styled, terminu.Fg(terminu.Peach).Render("~")) {
		t.Error("the marks are not in their colours")
	}
	d.Press("j", "Enter") // conf/
	if !d.Has("releases/v1/conf ↔ releases/v2/conf") || !d.Has("~ app.toml") || !d.Has("+ extra.toml") || d.Has("same.toml") {
		t.Fatalf("conf/:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Press("Enter")
	if !d.Has("DIFF — releases/v1/conf/app.toml ↔ releases/v2/conf/app.toml") || !d.Has("+port = 9090") {
		t.Errorf("app.toml's comparison:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Press("Esc", "Esc", "j", "Enter") // up to conf/, then logo.png: not text
	if d.Has("DIFF") {
		t.Error("an image was compared")
	}
	d.Press("Esc")
	if !d.Has("C|Enter:compare") || d.Has("↔") || is.fileList.Anchor == "" {
		t.Errorf("not back in the mode with the anchor kept:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Press("Esc")
	if is.files.InMode() != nil {
		t.Error("Esc did not leave the mode")
	}
}

// An action showing two versions of one item opens the same comparison, the
// older the anchor, titled with the two as the app names them; the last
// commit, which holds many, goes in to the items that differ (tdp note.8,
// note-101, note-102).
func TestTwoVersions(t *testing.T) {
	s := newShowcase()
	d := tdptest.Start(t, s.app, 140, 40)
	is := s.inputs
	d.Press("I", "2")
	if !is.fileList.Select(filepath.Join(projectDir, "main.go")) {
		t.Fatal("no main.go")
	}
	d.Press("H")
	if !d.Has("DIFF — main.go @ 5d6e7f8 ↔ main.go") || !d.Has("UNIFIED") || !d.Has(`+	fmt.Println("showcase", 42)`) && !d.Has(`+    fmt.Println("showcase", 42)`) {
		t.Errorf("main.go against its last commit:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Press("Esc")
	is.fileList.Select(filepath.Join(projectDir, "docs", "notes.md"))
	d.Press("H")
	if !d.Has("No differences between docs/notes.md @ 5d6e7f8 and docs/notes.md") {
		t.Errorf("a file as committed:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Advance(10 * time.Second) // the toast gone: Esc is the list's
	d.Press("K")
	for _, want := range []string{"1a2b3c4 ↔ 5d6e7f8", "~ docs/NOTES.md → docs/notes.md", "+ config/build.toml", "- TODO.md", "~ themes/"} {
		if !d.Has(want) {
			t.Errorf("no %q:\n%s", want, strings.Join(d.Screen(), "\n"))
		}
	}
	if is.files.InMode() != nil || d.Has("README-zh_TW.md") {
		t.Error("the commit is in a mode, or shows the panel's own list")
	}
	d.Press("j", "j", "Enter") // the rename
	if !d.Has("DIFF — docs/NOTES.md @ 1a2b3c4 ↔ docs/notes.md @ 5d6e7f8") {
		t.Errorf("the rename's comparison:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Press("Esc", "j", "j", "Enter") // themes/
	if !d.Has("themes @ 1a2b3c4 ↔ themes @ 5d6e7f8") || !d.Has("+ latte.toml") {
		t.Errorf("into themes/:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Press("Esc", "Esc")
	if d.Has("↔") || !d.Has("README-zh_TW.md") {
		t.Errorf("Esc did not go back to the files:\n%s", strings.Join(d.Screen(), "\n"))
	}
}

// An action showing something in a panel leaves the lists of the items that
// differ it went into first (Panel.LeaveDiff, from gitreu): [1]'s M from
// inside the last commit lands on main.go in [2]'s own files.
func TestShowMainLeavesDiff(t *testing.T) {
	s := newShowcase()
	d := tdptest.Start(t, s.app, 140, 40)
	is := s.inputs
	d.Press("I", "2", "K", "j", "j", "j", "j", "Enter") // into the last commit, then themes/
	if !d.Has("themes @ 1a2b3c4 ↔ themes @ 5d6e7f8") {
		t.Fatalf("not inside themes/:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Press("1", "M")
	if r, _ := is.fileList.Current(); d.Has("↔") || r.ID != filepath.Join(projectDir, "main.go") || s.app.Screen().Focused() != is.files {
		t.Errorf("not on main.go in [2]:\n%s", strings.Join(d.Screen(), "\n"))
	}
	if d.Press("Esc"); d.Has("↔") {
		t.Error("Esc found a list of the items that differ left behind")
	}
}
