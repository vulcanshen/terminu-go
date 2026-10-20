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
// path and why; Enter renames it to .bak and starts a new one; Esc leaves
// the app, Run returning the path and the reason (Rules E3: E3-17, E3-18,
// E3-20; components/dialog/confirm).
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
	for _, want := range []string{"MOVE ASIDE", path + " can't be read:", why.Error(), "Move it to profiles.yaml.bak and start a new one?", "Enter:move Esc:quit"} {
		if !strings.Contains(text, want) {
			t.Errorf("%q is not on the confirm:\n%s", want, text)
		}
	}
	ta.press("Enter")
	if _, err := os.Stat(path + ".bak"); err != nil || !fresh || ta.top() != nil {
		t.Errorf("Enter: moved %v, started fresh %v, closed %v", err, fresh, ta.top() == nil)
	}
	if !strings.Contains(strings.Join(ta.logs.lines(), "\n"), "moved "+path+" to "+path+".bak") {
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
	if !strings.Contains(ta.screenText(), "Move it to config.yaml.bak.2 and start") {
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
