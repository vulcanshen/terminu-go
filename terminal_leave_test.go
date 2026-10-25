package terminu

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// runAll runs cmd and the commands it batches.
func runAll(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	if batch, ok := cmd().(tea.BatchMsg); ok {
		for _, c := range batch {
			runAll(c)
		}
	}
}

// TestTerminalOnLeave: the exit key accepted, OnLeave runs once the focus
// is out and its command is run: here it puts the focus on the other panel,
// where the app wants it (Rules K10). Esc on the confirm and the app's own
// Hide do not run it; an exit key that does not ask, on a terminal that
// ends, does.
func TestTerminalOnLeave(t *testing.T) {
	ta, term, _ := ptyApp(t)
	other := ta.Screen().Panels[1]
	left, ran := 0, 0
	term.OnLeave = func() tea.Cmd {
		left++
		if term.Inside() {
			t.Error("OnLeave ran with the focus still inside")
		}
		ta.Focus(other)
		return func() tea.Msg { ran++; return nil }
	}
	ta.press("Alt-Esc", "Esc")
	if left != 0 || !term.Inside() {
		t.Fatalf("Esc on the confirm: OnLeave ran %d times, inside %v", left, term.Inside())
	}
	ta.press("Alt-Esc")
	runAll(ta.press("Enter"))
	if left != 1 || ran != 1 || ta.Screen().Focused() != other {
		t.Errorf("leaving: OnLeave ran %d times, its command %d, the focus on %q", left, ran, ta.Screen().Focused().Label)
	}
	ta.press("1", "Enter")
	term.Hide(ta.App)
	if left != 1 {
		t.Error("the app's own Hide ran OnLeave")
	}
	term.Enter()
	term.ExitKey, term.Ends = "Alt-q", true
	runAll(ta.press("Alt-q"))
	if left != 2 || ran != 2 || term.Inside() {
		t.Errorf("Alt-q on a terminal that ends: OnLeave ran %d times, its command %d", left, ran)
	}
}
