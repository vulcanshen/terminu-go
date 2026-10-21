package terminu

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestMoveAside: a file of the app's own it cannot read asks MOVE ASIDE, its
// path and why; Enter renames it to .bak and runs fresh, the lib making
// nothing in its place (the app goes on as on its first start); Esc leaves
// the app, Run returning the path and the reason (Rules E3: E3-17, E3-18,
// E3-20; components/dialog/confirm). The App Log's entry starts with a
// capital, as a toast does (note-78).
func TestMoveAside(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "profiles.yaml")
	writeFile(t, path, []byte("bad: [\n"), 0o600)
	why := errors.New("line 3: mapping values are not allowed here")
	ta := newTestApp(t)
	fresh := false
	ta.MoveAside(path, why, func() tea.Cmd { fresh = true; return nil })
	ta.settle()
	text := ta.screenText()
	for _, want := range []string{"MOVE ASIDE", path + " can't be read:", why.Error(), "Move it to profiles.yaml.bak and start fresh?", "Enter:move Esc:quit"} {
		if !strings.Contains(text, want) {
			t.Errorf("%q is not on the confirm:\n%s", want, text)
		}
	}
	ta.press("Enter")
	if _, err := os.Stat(path + ".bak"); err != nil || !fresh || ta.top() != nil {
		t.Errorf("Enter: moved %v, started fresh %v, closed %v", err, fresh, ta.top() == nil)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("the lib made something in its place: %v", err)
	}
	if !strings.Contains(strings.Join(ta.logs.lines(), "\n"), "Moved "+path+" to "+path+".bak") {
		t.Error("the move is not in the App Log")
	}
	// Esc leaves the app, saying what and why.
	writeFile(t, path, []byte("bad again"), 0o600)
	ta.MoveAside(path, why, func() tea.Cmd { return nil })
	ta.settle()
	cmd := ta.press("Esc")
	if cmd == nil {
		t.Fatal("Esc did nothing")
	}
	if _, quit := cmd().(tea.QuitMsg); !quit || ta.exitErr == nil || !strings.Contains(ta.exitErr.Error(), path+" can't be read: "+why.Error()) {
		t.Errorf("Esc: quit, with %v", ta.exitErr)
	}
	if _, err := os.Stat(path); err != nil {
		t.Error("Esc moved the file")
	}
}

// TestMoveAsideRecord: data kept one record per file: Enter only renames the
// record that cannot be read to .bak, and the app reads the rest again; the
// question says go on without it (E3-22; kamonu KM18). A rename that fails
// asks the same again.
func TestMoveAsideRecord(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kiri.yaml")
	writeFile(t, path, []byte("bad: [\n"), 0o600)
	ta := newTestApp(t)
	reread := false
	ta.MoveAsideRecord(path, errors.New("bad"), func() tea.Cmd { reread = true; return nil })
	ta.settle()
	text := ta.screenText()
	for _, want := range []string{"MOVE ASIDE", path + " can't be read:", "Move it to kiri.yaml.bak and go on without it?", "Enter:move Esc:quit"} {
		if !strings.Contains(text, want) {
			t.Errorf("%q is not on the confirm:\n%s", want, text)
		}
	}
	ta.press("Enter")
	if _, err := os.Stat(path + ".bak"); err != nil || !reread || ta.top() != nil {
		t.Errorf("Enter: moved %v, read again %v, closed %v", err, reread, ta.top() == nil)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("something is in its place: %v", err)
	}
	// The rename fails: the error, and under it the same question again.
	writeFile(t, path, []byte("bad again"), 0o600)
	ta.MoveAsideRecord(path, errors.New("bad"), func() tea.Cmd { return nil })
	ta.settle()
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0o700)
	ta.press("Enter")
	ta.settle()
	if text := ta.screenText(); !strings.Contains(text, "MOVE ASIDE FAILED") {
		t.Errorf("a failed rename says nothing:\n%s", text)
	}
	ta.press("Esc")
	if text := ta.screenText(); !strings.Contains(text, "Move it to kiri.yaml.bak.1 and go on without it?") {
		t.Errorf("not asked again:\n%s", text)
	}
}

// TestNoLongerUsed: a file the app no longer uses that is still there gets an
// INFO in the App Log with its path, at every start; it is not deleted and
// there is no toast; one that is gone says nothing (Rules E3, E3-23).
func TestNoLongerUsed(t *testing.T) {
	dir := t.TempDir()
	old, gone := filepath.Join(dir, "errors.yaml"), filepath.Join(dir, "theme.yaml")
	writeFile(t, old, []byte("x"), 0o600)
	for range 2 { // every start: the App Log starts empty
		ta := newTestApp(t)
		ta.NoLongerUsed(old, gone)
		log := StripStyles(strings.Join(ta.logs.lines(), "\n"))
		if !strings.Contains(log, "INFO No longer used: "+old+"; delete it if you don't need it") || strings.Contains(log, gone) {
			t.Errorf("the App Log:\n%s", log)
		}
		if ta.toast != nil {
			t.Error("a toast")
		}
	}
	if _, err := os.Stat(old); err != nil {
		t.Errorf("the file was deleted: %v", err)
	}
}

// TestMoveAsideNeverOverAnother: with .bak there it goes to .bak.1, then
// .bak.2, the question naming the one it uses (E3-19).
func TestMoveAsideNeverOverAnother(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	for _, p := range []string{path, path + ".bak", path + ".bak.1"} {
		writeFile(t, p, []byte(p), 0o600)
	}
	ta := newTestApp(t)
	ta.MoveAside(path, errors.New("bad"), func() tea.Cmd { return nil })
	ta.settle()
	if !strings.Contains(ta.screenText(), "Move it to config.yaml.bak.2 and start fresh?") {
		t.Errorf("the question:\n%s", ta.screenText())
	}
	ta.press("Enter")
	for _, p := range []string{path + ".bak", path + ".bak.1"} {
		if b, _ := os.ReadFile(p); string(b) != p {
			t.Errorf("%s was written over: %q", p, b)
		}
	}
	if b, _ := os.ReadFile(path + ".bak.2"); string(b) != path {
		t.Errorf(".bak.2 is not the file moved aside: %q", b)
	}
}
