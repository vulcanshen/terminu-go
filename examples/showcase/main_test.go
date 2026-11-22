package main

import (
	"os"
	"path/filepath"
	"regexp"
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
	for _, want := range []int{60, 15, 40, 40} {
		d.Advance(3 * time.Second)
		if len(v.Lines) != want || !d.Has("notes.txt") {
			t.Fatalf("read again: %d lines, want %d", len(v.Lines), want)
		}
	}
	d.Advance(3 * time.Second)
	if !d.Has("Cannot read notes.txt: it is gone") || len(v.Lines) != 40 {
		t.Errorf("gone: no Error toast, or the lines went: %d", len(v.Lines))
	}
	d.Advance(3 * time.Second) // still gone
	d.Advance(3 * time.Second)
	if !d.Has("Cannot read notes.txt: permission denied") {
		t.Errorf("another failure, no Error toast:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Advance(3 * time.Second) // as it was again
	d.Advance(3 * time.Second) // the same
	if len(v.Lines) != 40 || ds.version != 9 {
		t.Errorf("round again: %d lines, version %d", len(v.Lines), ds.version)
	}
	for i := 0; i < 4 && s.app.Covered(); i++ { // the Error toast, then the viewer
		d.Press("Esc")
	}
	d.Advance(3 * time.Second)
	d.Advance(3 * time.Second)
	if ds.watched != nil || ds.version != 9 {
		t.Errorf("closed, it is still read: version %d", ds.version)
	}
	// Each change one INFO entry, nothing new none; each failure one ERR
	// entry, the same again none; working again one INFO entry (note-104,
	// note-105, from gittu).
	d.Press("!")
	log := strings.Join(d.Screen(), "\n")
	for want, n := range map[string]int{
		"INFO Changed outside the app: notes.txt went from 40 lines to 60": 1,
		"INFO Changed outside the app: notes.txt went from 60 lines to 15": 1,
		"INFO Changed outside the app: notes.txt went from 15 lines to 40": 1,
		"Changed outside the app":                       3,
		"ERR  Cannot read notes.txt: it is gone":        1,
		"ERR  Cannot read notes.txt: permission denied": 1,
		"INFO notes.txt can be read again":              1,
	} {
		if got := strings.Count(log, want); got != n {
			t.Errorf("%d of %q in the App Log, want %d:\n%s", got, want, n, log)
		}
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
	d.Press("D", "j", "Enter") // v1/ against v2/
	if is.files.InMode() == nil || !d.Has("releases/v1 ↔ releases/v2") {
		t.Fatalf("did not go in:\n%s", strings.Join(d.Screen(), "\n"))
	}
	for _, want := range []string{"+ NEWS.md", "~ conf/", "~ logo.png", "~ main.go", "- old-notes.txt"} {
		if !d.Has(want) {
			t.Errorf("no %q:\n%s", want, strings.Join(d.Screen(), "\n"))
		}
	}
	// [3] lists v2/ under the cursor, new since the last commit, all +: one
	// listed here would be ~.
	if d.Has("~ README.md") || d.Has("~ docs/") {
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
	if !d.Has("releases/v1/conf/app.toml ↔ releases/v2/conf/app.toml") || !d.Has("+port = 9090") {
		t.Errorf("app.toml's comparison:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Press("Esc", "Esc", "j", "Enter") // up to conf/, then logo.png: not text
	if d.Has("DIFF") {
		t.Error("an image was compared")
	}
	d.Press("Esc")
	if !d.Has("D|Enter:diff") || d.Has("↔") || is.fileList.Anchor == "" {
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
	if !d.Has("main.go @ 5d6e7f8 ↔ main.go") || !d.Has("UNIFIED") || !d.Has(`+	fmt.Println("showcase", 42)`) && !d.Has(`+    fmt.Println("showcase", 42)`) {
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
	if !d.Has("docs/NOTES.md @ 1a2b3c4 ↔ docs/notes.md @ 5d6e7f8") {
		t.Errorf("the rename's comparison:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Press("Esc", "j", "j", "Enter") // themes/
	if !d.Has("themes @ 1a2b3c4 ↔ themes @ 5d6e7f8") || !d.Has("+ latte.toml") {
		t.Errorf("into themes/:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Press("Esc", "Esc")
	if d.Has("↔") || !d.Has("MODIFIED") { // the files' own header
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

// A panel showing a difference draws it as UNIFIED does, not as git prints
// a patch, and Enter opens the comparison (tdp note.8 "In a panel",
// note-103): [3] CHANGES follows [2]'s cursor, the file under it against its
// last commit; a file as committed is the empty state, Enter off.
func TestChangesPanel(t *testing.T) {
	s := newShowcase()
	d := tdptest.Start(t, s.app, 140, 40)
	is := s.inputs
	show := func(rel string) {
		t.Helper()
		d.Press("2")
		if !is.fileList.Select(filepath.Join(projectDir, rel)) {
			t.Fatalf("no %s", rel)
		}
		d.Press("3")
	}
	d.Press("I")
	show("main.go")
	screen := strings.Join(d.Screen(), "\n")
	if !regexp.MustCompile(`\+ +fmt\.Println\("showcase", 42\)`).MatchString(screen) || !regexp.MustCompile(`- +fmt\.Println\("showcase"\)`).MatchString(screen) {
		t.Errorf("main.go's changes are not drawn as UNIFIED:\n%s", screen)
	}
	if d.Has("@@") || d.Has("diff --git") || !d.Has("Enter:diff") {
		t.Errorf("a patch's own lines, or no Enter:diff:\n%s", screen)
	}
	d.Press("Enter")
	if !d.Has("main.go @ 5d6e7f8 ↔ main.go") || !d.Has("UNIFIED") || !d.Has("SPLIT") {
		t.Errorf("Enter did not open the comparison:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Press("Esc")
	show("docs/notes.md")
	if d.Press("Enter"); !d.Has("No changes since 5d6e7f8") || d.Has("UNIFIED") {
		t.Errorf("a file as committed:\n%s", strings.Join(d.Screen(), "\n"))
	}
	show("assets/logo.png")
	if !d.Has("Not a text file") {
		t.Errorf("an image:\n%s", strings.Join(d.Screen(), "\n"))
	}
}

// A panel showing the changes of what holds many lists the items that
// differ with their marks, a new folder's all + in Green, not bare names;
// Enter goes into that list (tdp note.8, note-106, from gittu): [3] CHANGES
// on releases/v2/, new since the last commit. [2]'s cursor moving on leaves
// that list.
func TestNewFolderChanges(t *testing.T) {
	s := newShowcase()
	d := tdptest.Start(t, s.app, 140, 40)
	d.Press("I", "2", "l", "l", "/")
	d.Type("rel")
	d.Press("Enter", "Enter") // into releases/, the cursor on v1/
	if !d.Has("No changes since 5d6e7f8") {
		t.Fatalf("v1/, as committed:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Press("j") // v2/
	for _, want := range []string{"+ NEWS.md", "+ README.md", "+ conf/", "+ docs/", "+ logo.png", "+ main.go"} {
		if !d.Has(want) {
			t.Errorf("no %q:\n%s", want, strings.Join(d.Screen(), "\n"))
		}
	}
	if d.Press("3"); !d.Has("Enter:open") || d.Has("Enter:diff") {
		t.Errorf("no Enter:open on [3]:\n%s", strings.Join(d.Screen(), "\n"))
	}
	if styled := strings.Join(d.Styled(), "\n"); !strings.Contains(styled, terminu.Fg(terminu.Green).Render("+")) {
		t.Error("the marks are not in Green")
	}
	d.Press("Enter")
	if !d.Has("releases/v2 @ 5d6e7f8 ↔ releases/v2") {
		t.Fatalf("Enter did not go in:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Press("Enter") // NEWS.md
	if !d.Has("releases/v2/NEWS.md @ 5d6e7f8 ↔ releases/v2/NEWS.md") || !d.Has("+# News") {
		t.Fatalf("an item's comparison:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Press("Esc", "2", "k")
	if d.Has("↔") || !d.Has("No changes since 5d6e7f8") {
		t.Errorf("[2]'s cursor moving on kept the list:\n%s", strings.Join(d.Screen(), "\n"))
	}
}

// The comparison's first row names its two sides, folded and never cut
// (tdp note.8, note-107, from gittu): on 100 columns before ↔, on 60 a side
// too wide on after a /; it stays as the lines scroll.
func TestCompareFirstRow(t *testing.T) {
	s := newShowcase()
	d := tdptest.Start(t, s.app, 100, 30)
	d.Press("I", "2", "K", "G", "Enter") // the last item, a long path
	if !d.Has(deepPath+" @ 1a2b3c4") || !d.Has("↔ "+deepPath+" @ 5d6e7f8") || !d.Has("UNIFIED") || !d.Has("SPLIT") {
		t.Fatalf("not folded before ↔:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Resize(60, 30)
	for _, want := range []string{"installation/docker-compose/production/", "env.example @ 1a2b3c4", "↔ installation/docker-compose/production/", "env.example @ 5d6e7f8"} {
		if !d.Has(want) {
			t.Errorf("no row %q:\n%s", want, strings.Join(d.Screen(), "\n"))
		}
	}
}

// The command line (tdp E1, E1-3 to E1-8; E3-40): results on stdout, errors
// and warnings on stderr a line each after "showcase: ", in the TUI's words;
// exit 0 done, 1 not done, 2 used wrong. iconwidth prints each width with
// where it came from; settings, with no screen, says on stderr what the TUI
// says in its toast and App Log, the settings alone on stdout.
func TestCommandLine(t *testing.T) {
	run := func(args ...string) (string, string, int) {
		var out, errs strings.Builder
		code := command(args, &out, &errs)
		return out.String(), errs.String(), code
	}
	if out, errs, code := run("version"); out != "showcase "+version+"\n" || errs != "" || code != 0 {
		t.Errorf("version: %q %q %d", out, errs, code)
	}
	if out, errs, code := run("help"); out != usage || errs != "" || code != 0 {
		t.Errorf("help: %q %q %d", out, errs, code)
	}
	if out, errs, code := run("x"); out != "" || errs != "showcase: Unknown command \"x\"\n\n"+usage || code != 2 {
		t.Errorf("used wrong: %q %q %d", out, errs, code)
	}
	// Never released, it is a build from source: update says so, exit 1,
	// nothing looked up (tdp E1, E1-10).
	if out, errs, code := run("update"); out != "" || code != 1 ||
		errs != "showcase: This showcase is built from source: update it there (`git pull && go install ./examples/showcase`)\n" {
		t.Errorf("update: %q %q %d", out, errs, code)
	}

	icon, powerline := terminu.IconWidth(), terminu.PowerlineWidth()
	t.Cleanup(func() { terminu.SetIconWidth(icon, powerline) })
	t.Setenv("SHOWCASE__ICON_WIDTH", "2")
	t.Setenv("SHOWCASE__POWERLINE_WIDTH", "1")
	if out, errs, code := run("iconwidth"); out != "icon 2 (SHOWCASE__ICON_WIDTH)\npowerline 1 (SHOWCASE__POWERLINE_WIDTH)\n" || errs != "" || code != 0 {
		t.Errorf("iconwidth: %q %q %d", out, errs, code)
	}

	out, errs, code := run("settings")
	if code != 0 || !strings.Contains(out, `color: "#89b4fa"`) || strings.Contains(out, "showcase:") {
		t.Errorf("settings, stdout %q, exit %d", out, code)
	}
	for _, want := range []string{
		"showcase: 2 values in config.yaml did not fit: defaults used, yours kept in config.yaml.bak",
		`showcase: config.yaml: color "#zzzzzz" did not fit; using "#89b4fa"`,
		`showcase: config.yaml: timeout "-5" did not fit; using "30"`,
		"showcase: No longer used: " + filepath.Join(dataDir(), "layout.yaml") + "; delete it if you don't need it",
	} {
		if !strings.Contains(errs, want+"\n") {
			t.Errorf("settings, no %q on stderr:\n%s", want, errs)
		}
	}
	if strings.Count(errs, "\n") != 4 {
		t.Errorf("settings: %d lines on stderr, want 4:\n%s", strings.Count(errs, "\n"), errs)
	}
	out, errs, code = run("settings", "broken")
	want := "showcase: " + filepath.Join(dataDir(), "settings.yaml") + " can't be read: line 1: did not find expected ',' or ']'\n" +
		"showcase: Open showcase to move it aside and start fresh\n"
	if out != "" || errs != want || code != 1 {
		t.Errorf("a file it can't read: %q %q %d", out, errs, code)
	}
	if _, _, code := run("settings", "x"); code != 2 {
		t.Errorf("settings used wrong: exit %d", code)
	}
}

// An item of several rows (tdp panel.5, panel-181 to panel-183, from gittu):
// [1] PODS' EVENTS tab has each event's message under it, starting under
// REASON, only some events taller; the rows below the cursor's item marked,
// a bar down beside them, another item's not; j moves an item; the filter
// finds an event by its message.
func TestEventsSeveralRows(t *testing.T) {
	s := newShowcase()
	d := tdptest.Start(t, s.app, 140, 40)
	d.Press("P", "1", "l", "g", "g") // EVENTS, sorted by object: postgres-0 first
	lit := func(text string) bool {  // the bar beside the row (tdp panel.5)
		for _, l := range d.Screen() {
			if strings.Contains(l, text) {
				return strings.Contains(l, "\u258c")
			}
		}
		t.Fatalf("no %q:\n%s", text, strings.Join(d.Screen(), "\n"))
		return false
	}
	screen := d.Screen()
	row := func(text string) int {
		for i, l := range screen {
			if strings.Contains(l, text) {
				return i
			}
		}
		return -1
	}
	reason, message := row("BackOff"), row("Back-off restarting")
	if message != reason+1 || row("Last seen on node worker-2") != reason+2 || strings.Index(screen[message], "Back-off") != strings.Index(screen[reason], "BackOff") {
		t.Errorf("the message is not under REASON:\n%s", strings.Join(screen, "\n"))
	}
	if lit("Pulled image") || lit("Back-off restarting") {
		t.Error("on postgres-0, another event's rows are lit")
	}
	d.Press("j") // prometheus-server-0: one item down, its row below lit
	if r, _ := s.panels.podList.Current(); r.ID != "e2" || !lit("Pulled image") || lit("Back-off restarting") {
		t.Errorf("j: on %s, or the wrong rows lit", r.ID)
	}
	d.Press("j") // web-a8k2, two rows below
	if r, _ := s.panels.podList.Current(); r.ID != "e1" || !lit("Back-off restarting") || !lit("Last seen on node") || lit("Pulled image") {
		t.Errorf("j again: on %s, or the wrong rows lit", r.ID)
	}
	d.Press("/")
	d.Type("worker-2")
	if !d.Has("web-a8k2") || d.Has("prometheus-server-0") || d.Has("postgres-0") {
		t.Errorf("the filter did not find web-a8k2 by its message:\n%s", strings.Join(d.Screen(), "\n"))
	}
}

// Room left over on a wide terminal (tdp panel.5, panel-184): [1] PODS'
// "Room left over" goes round the four ways; with NAME filling, AGE sits at
// the panel's right edge, at the end there is room after it; the header and
// the rows keep the same widths.
func TestRoomLeftOver(t *testing.T) {
	s := newShowcase()
	d := tdptest.Start(t, s.app, 240, 40)
	d.Press("P", "1")
	room := func() int { // the blank between AGE and the frame on the header
		for _, l := range d.Screen() {
			if i := strings.Index(l, "AGE"); i >= 0 && strings.Contains(l, "NAME") {
				rest := l[i+len("AGE"):]
				return len(rest) - len(strings.TrimLeft(rest, " "))
			}
		}
		t.Fatalf("no header:\n%s", strings.Join(d.Screen(), "\n"))
		return -1
	}
	end := room()
	var spread terminu.Action
	for _, act := range s.panels.pods.Actions {
		if act.Label == "Room left over" {
			spread = act
		}
	}
	d.Do(spread.Do) // NAME fills
	if s.panels.podList.Spread != terminu.SpreadFill || !d.Has("Room left over: NAME fills") {
		t.Fatalf("not NAME fills: %d", s.panels.podList.Spread)
	}
	if fill := room(); end <= 4 || fill > 3 {
		t.Errorf("AGE is not at the right edge with NAME filling: %d cells after it, %d at the end", fill, end)
	}
	for range 3 {
		d.Do(spread.Do)
	}
	if s.panels.podList.Spread != terminu.SpreadEnd || room() != end {
		t.Errorf("round the four, not back at the end: %d", s.panels.podList.Spread)
	}
}

// D is the Diff mode's key in every app, written [D]iff; x deletes the item
// under the cursor, X many at once; nothing that removes for good is on D or
// d (tdp K13, K13-1 to K13-3): [1] PODS has [D]iff, its Drag on r, D[r]ag,
// [x] Delete and [X] Delete marked; [2] FILES' Diff is on D too.
func TestK13Keys(t *testing.T) {
	s := newShowcase()
	d := tdptest.Start(t, s.app, 160, 40)
	d.Press("P", "1", "Space")
	for _, want := range []string{"[D]iff", "D[r]ag", "[x] Delete", "[X] Delete marked"} {
		if !d.Has(want) {
			t.Errorf("[1] PODS' menu has no %q:\n%s", want, strings.Join(d.Screen(), "\n"))
		}
	}
	d.Press("Esc", "I", "2", "l", "l", "Space") // FOLDERS
	if !d.Has("[D]iff") {
		t.Errorf("[2] FILES' menu has no [D]iff:\n%s", strings.Join(d.Screen(), "\n"))
	}
	for _, sc := range []*terminu.Screen{s.panels.screen, s.inputs.screen, s.dialogs.screen, s.terminal.screen, s.kinds.screen} {
		for _, p := range sc.Panels {
			for _, act := range p.Actions {
				removes := strings.Contains(act.Label, "Delete") || strings.Contains(act.Label, "Discard") || strings.Contains(act.Label, "Drop")
				if removes && (act.Key == "D" || act.Key == "d") {
					t.Errorf("%s removes on %s", act.Label, act.Key)
				}
				if act.Label == "Diff" && act.Key != "Enter" && act.Key != "D" { // the mode's, not an Enter that opens one
					t.Errorf("the Diff mode on %s in %s", act.Key, p.Label)
				}
			}
		}
	}
}

// [2] FILES may show its Diff mode in [3] CHANGES, its details (tdp
// panel.9 "With a details panel", panel-185, panel-186): the showcase
// switches between the two ways. The mode spans [2] and [3]; [3] follows
// the cursor with the anchor's diff, an empty state on the anchor; D sends
// the focus to [3], Enter there goes into a folder, Esc goes back the way
// the user came, and out of the mode [3] shows the file under the cursor
// against its last commit again.
func TestDiffInChanges(t *testing.T) {
	s := newShowcase()
	d := tdptest.Start(t, s.app, 160, 40)
	is := s.inputs
	d.Press("I", "2", "l", "l", "/")
	d.Type("rel")
	d.Press("Enter", "Enter") // into releases/
	var in terminu.Action
	for _, act := range is.files.Actions {
		if act.Label == "Diff in [3]" {
			in = act
		}
	}
	d.Do(in.Do)
	if !d.Has("The Diff mode shows in [3] CHANGES") {
		t.Fatal("no toast")
	}
	d.Press("Esc", "D") // the toast closed; v1/ the anchor
	if is.files.InMode() == nil || is.changes.InMode() == nil || s.app.Screen().Focused() != is.files {
		t.Fatal("the mode does not span [2] and [3], the focus on [2]")
	}
	if !d.Has("Move to another file or folder to diff with releases/v1") {
		t.Errorf("on the anchor:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Press("j") // v2/: [3] lists the items that differ, no key pressed
	for _, want := range []string{"releases/v1 ↔ releases/v2", "+ NEWS.md", "~ conf/", "- old-notes.txt"} {
		if !d.Has(want) {
			t.Errorf("no %q in [3]:\n%s", want, strings.Join(d.Screen(), "\n"))
		}
	}
	if s.app.Screen().Focused() != is.files || d.Has("DIFF") {
		t.Error("moving the cursor did more than fill [3]")
	}
	d.Press("D")
	if s.app.Screen().Focused() != is.changes {
		t.Fatal("D did not send the focus to [3]")
	}
	d.Press("j", "Enter") // conf/: in
	if !d.Has("releases/v1/conf ↔ releases/v2/conf") || !d.Has("~ app.toml") {
		t.Fatalf("conf/ in [3]:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Press("Enter")
	if !d.Has("+port = 9090") {
		t.Errorf("app.toml's comparison:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Press("Esc", "Esc", "Esc") // the comparison, conf/, [3]
	if s.app.Screen().Focused() != is.files || is.files.InMode() == nil || is.fileList.Anchor == "" {
		t.Fatal("Esc did not go back to [2], the mode kept")
	}
	d.Press("Esc")
	if is.files.InMode() != nil || is.changes.InMode() != nil || d.Has("releases/v1 ↔ releases/v2") {
		t.Errorf("out of the mode:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Do(in.Do)
	d.Press("Esc", "k", "D", "j", "Enter") // as before: v1/ against v2/, into the list in [2]
	if s.app.Screen().Focused() != is.files || is.changes.InMode() != nil || !d.Has("releases/v1 ↔ releases/v2") {
		t.Errorf("the Diff mode as before:\n%s", strings.Join(d.Screen(), "\n"))
	}
}

// In [3] the Diff mode says it cannot diff a row with the anchor, and shows
// a side it cannot read as its error state, no error popup (tdp panel.9,
// panel-187): assets/ against main.go, then against locked/.
func TestDiffInChangesCannot(t *testing.T) {
	s := newShowcase()
	d := tdptest.Start(t, s.app, 200, 40)
	is := s.inputs
	d.Press("I", "2", "l", "l")
	for _, act := range is.files.Actions {
		if act.Label == "Diff in [3]" {
			d.Do(act.Do)
		}
	}
	d.Press("Esc")
	if !is.fileList.Select(filepath.Join(projectDir, "assets")) {
		t.Fatal("no assets/")
	}
	d.Press("D")
	is.fileList.Select(filepath.Join(projectDir, "main.go"))
	d.Press("Space", "Esc") // a message, so the details fill
	if !d.Has("Can't diff main.go with assets") {
		t.Errorf("a file against a folder:\n%s", strings.Join(d.Screen(), "\n"))
	}
	is.fileList.Select(filepath.Join(projectDir, "locked"))
	d.Press("Space", "Esc")
	if !d.Has("locked: ") || !d.Has("permission denied") || d.Has("DIFF: ") {
		t.Errorf("a folder that cannot be read:\n%s", strings.Join(d.Screen(), "\n"))
	}
}

// [K]INDS has two signs under the details (tdp panel.17, panel-188 to
// panel-191): LEGEND lit and SUMMARY dim, drawn as panels without the
// focus, their labels alone; Tab and the digits pass them by; their lines
// fit the smallest screen.
func TestKindsSigns(t *testing.T) {
	s := newShowcase()
	d := tdptest.Start(t, s.app, 60, 32)
	d.Press("K")
	screen := strings.Join(d.Screen(), "\n")
	for _, want := range []string{"LEGEND", "SUMMARY", "[1] KINDS", "[2] KIND: pods", "the kind it opens on", "Cluster-wide: read-only",
		"16 kinds in 5 groups", "Opens on pods", "Tab|1–2:panels"} {
		if !strings.Contains(screen, want) {
			t.Errorf("no %q at 60 × 32:\n%s", want, screen)
		}
	}
	if strings.Contains(screen, "[3]") || strings.Contains(screen, "[4]") {
		t.Errorf("a sign numbered:\n%s", screen)
	}
	focused := func() *terminu.Panel { return s.app.Screen().Focused() }
	d.Press("Tab")
	if focused() != s.kinds.details {
		t.Errorf("Tab went to %s", focused().Label)
	}
	d.Press("Tab")
	if focused() == s.kinds.legend || focused() == s.kinds.summary {
		t.Error("Tab stopped on a sign")
	}
	d.Press("3")
	d.Press("4")
	if focused() == s.kinds.legend || focused() == s.kinds.summary {
		t.Error("a digit went to a sign")
	}
	d.Press("1", "j", "f")
	if !d.Has("Opens on deployments") {
		t.Errorf("SUMMARY did not follow:\n%s", strings.Join(d.Screen(), "\n"))
	}
}

// A pod that is not running says why in its note (tdp panel.18,
// panel-192 to panel-194): the head under it, i opening the note under the
// pod, Esc closing it back to its head; a running pod has none.
func TestPodNote(t *testing.T) {
	s := newShowcase()
	d := tdptest.Start(t, s.app, 140, 40)
	d.Press("P", "1", "g", "g")
	head := "┌ \U000F02FD i ┐"
	if d.Has(head) {
		t.Fatal("a head on a running pod")
	}
	for range 40 {
		if r, ok := s.panels.podList.Current(); ok && strings.Contains(r.Cells[2], "CrashLoopBackOff") {
			break
		}
		d.Press("j")
	}
	if !d.Has(head) {
		t.Fatalf("no head under a pod in CrashLoopBackOff:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Press("i")
	if !d.Has("NOTE") || !d.Has("Back-off restarting failed container app") {
		t.Errorf("i did not open its note:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Press("Esc")
	if d.Has("NOTE") || !d.Has(head) {
		t.Errorf("Esc did not close it back to its head:\n%s", strings.Join(d.Screen(), "\n"))
	}
}

// Toasts stack (tdp toast.2, toast.3, toast-13, toast-14, toast-29): the
// start's two Warnings a toast each; the same words again count on one; Esc
// closes the oldest first.
func TestToastsStack(t *testing.T) {
	s := newShowcase()
	d := tdptest.Start(t, s.app, 160, 40)
	const gone = `Profile "work" in config.yaml is not in profiles.yaml: using the default`
	d.Do(func() tea.Cmd { return s.dialogs.didNotFit(false, terminu.Notice{Kind: terminu.Warning, Text: gone}) })
	screen := strings.Join(d.Screen(), "\n")
	if strings.Count(screen, "WARNING") != 2 || strings.Index(screen, "did not fit") > strings.Index(screen, gone) {
		t.Errorf("not two toasts, the first above:\n%s", screen)
	}
	if strings.Count(screen, "Esc:close") != 1 {
		t.Errorf("Esc:close on more than the oldest:\n%s", screen)
	}
	for range 3 {
		d.Do(func() tea.Cmd { return s.app.Toast(terminu.Info, "Copied 3 paths") })
	}
	if !d.Has("INFO ×3") {
		t.Errorf("the same words do not count on one toast:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Press("Esc")
	if d.Has("did not fit") || !d.Has(gone) {
		t.Errorf("Esc did not close the oldest first:\n%s", strings.Join(d.Screen(), "\n"))
	}
}

// A name with a colour code in it is drawn by its names, in the toast and
// in the App Log alike, the code never applied (tdp note.7, note-108; from
// hostu): D "Toast: a colour code in a name".
func TestAppLogNames(t *testing.T) {
	s := newShowcase()
	d := tdptest.Start(t, s.app, 140, 40)
	d.Do(func() tea.Cmd { return s.app.Toast(terminu.Info, "Opened lab\x1b[31mX") })
	if !d.Has(`Opened lab\x1b[31mX`) {
		t.Errorf("the toast does not show the code by name:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Press("Esc", "!")
	if !d.Has(`INFO Opened lab\x1b[31mX`) {
		t.Errorf("the App Log does not show the code by name:\n%s", strings.Join(d.Screen(), "\n"))
	}
}

// [R]EVIEW's right side follows the leading panel that last had the focus
// (tdp L5, L5-4; from gittu): DIFF and HISTORY under CHANGES, the third
// place hidden and its digit doing nothing; COMMITS, DESCRIPTION, FILES
// under BRANCHES; moving into them changes nothing; numbers by place.
func TestReviewFollows(t *testing.T) {
	s := newShowcase()
	d := tdptest.Start(t, s.app, 140, 40)
	d.Press("R")
	focused := func() *terminu.Panel { return s.app.Screen().Focused() }
	if !d.Has("[3] DIFF") || !d.Has("[4] HISTORY") || d.Has("[5]") || !d.Has("Tab|1–5:panels") {
		t.Fatalf("CHANGES leading:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Press("5")
	if focused() != s.review.panels["Changes"] {
		t.Errorf("the hidden place's digit went to %s", focused().Label)
	}
	d.Press("2")
	if !d.Has("[3] COMMITS") || !d.Has("[4] DESCRIPTION") || !d.Has("[5] FILES") || d.Has("DIFF") {
		t.Errorf("BRANCHES leading:\n%s", strings.Join(d.Screen(), "\n"))
	}
	d.Press("5")
	if focused() != s.review.panels["Files"] || !d.Has("[3] COMMITS") {
		t.Errorf("into the right side something changed: %s\n%s", focused().Label, strings.Join(d.Screen(), "\n"))
	}
	d.Press("1")
	if !d.Has("[3] DIFF") || d.Has("FILES") {
		t.Errorf("CHANGES again:\n%s", strings.Join(d.Screen(), "\n"))
	}
}
