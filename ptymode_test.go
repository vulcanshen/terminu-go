package terminu

import (
	"bytes"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// ptyModeApp is the test app with a PTY panel whose Enter goes into a mode
// of the app's own holding it, its Esc out of the mode sending the program
// an Esc; the terminal writes its keys into sent.
func ptyModeApp(t *testing.T) (ta *testApp, term *Terminal, sent *bytes.Buffer, p *Panel, left *int) {
	t.Helper()
	ta = newTestApp(t)
	sent, left = &bytes.Buffer{}, new(int)
	term = &Terminal{Title: "Agent", in: sent, started: true,
		Keys: []TerminalKey{{Key: "Alt-t", Desc: "tasks", Do: ta.record("tasks")}}}
	p = &Panel{Label: "Agent", Body: term}
	p.Actions = []Action{{Key: "Enter", Label: "Type", Desc: "go in to type", Scope: PanelScope, Hint: "type",
		Do: func() tea.Cmd {
			p.EnterMode(&Mode{Name: "Typing", Exit: "leave", PTY: true, OnExit: func() tea.Cmd { *left++; return nil }})
			return nil
		}}}
	p.OnEsc = func() tea.Cmd { term.Send([]byte{0x1b}); return nil } // what the agent is doing, cancelled (tdp K4)
	ta.AddScreen(&Screen{Name: "Agent", Key: "A", Panels: []*Panel{p, {Label: "Other", Body: &Pager{}}}})
	ta.press("A", "Enter")
	ta.settle()
	if p.InMode() == nil {
		t.Fatal("Enter did not go into the mode")
	}
	return ta, term, sent, p, left
}

// TestK11ModeHoldsPTY: a mode of the app's own may hold its panel's PTY:
// it shows itself as a mode; every key but Esc goes to the program, Space,
// ?, q and Ctrl-C too; Esc leaves the mode asking nothing, the footer
// writing it; two Escs read as one Alt-Esc leave it and send one Esc; out of
// the mode the panel is the family's again (tdp K11, K10; K11-28, K10-3,
// K10-4; from ccodu, the user).
func TestK11ModeHoldsPTY(t *testing.T) {
	ta, term, sent, p, left := ptyModeApp(t)
	text := ta.screenText()
	if !strings.Contains(text, "Typing") || p.modeName() != "Typing" {
		t.Errorf("the mode's name is not on the frame:\n%s", text)
	}
	if strings.Contains(text, "Enter:type") {
		t.Error("in the mode the panel's hint is still there")
	}
	cursor := func() bool { return strings.Contains(strings.Join(ta.Frame(), "\n"), "48;2;205;214;244") } // the program's, in Text
	if !cursor() {
		t.Error("in the mode the program's cursor is not drawn")
	}
	footer := StripStyles(ta.Frame()[ta.h-1])
	if !strings.Contains(footer, "Esc:leave") || !strings.Contains(footer, "Alt-t:tasks") || strings.Contains(footer, "Alt-Esc") ||
		strings.Contains(footer, "Space:menu") {
		t.Errorf("the footer: %q", footer)
	}

	ta.press("a", "Space", "?", "q", "Ctrl-C")
	if got := sent.String(); got != "a ?q\x03" || ta.top() != nil || p.InMode() == nil {
		t.Errorf("Space, ?, q, Ctrl-C are the program's: sent %q, %T open, in the mode %v", got, ta.topView(), p.InMode() != nil)
	}
	ta.press("Alt-t")
	if len(ta.ran) != 1 || ta.ran[0] != "tasks" {
		t.Errorf("the app's own key in the PTY (K10): ran %v", ta.ran)
	}

	sent.Reset()
	ta.press("Esc")
	if p.InMode() != nil || ta.top() != nil || *left != 1 || sent.Len() != 0 {
		t.Errorf("Esc: in the mode %v, %T open, OnExit %d, sent %q", p.InMode() != nil, ta.topView(), *left, sent.String())
	}
	if !strings.Contains(ta.screenText(), "Enter:type") || term.keysIn() || cursor() {
		t.Error("out of the mode the panel is the family's again")
	}
	ta.press("j", "Esc") // the panel's Esc: one Esc for the program (K4)
	if got := sent.String(); got != "\x1b" {
		t.Errorf("out of the mode: sent %q", got)
	}

	sent.Reset()
	ta.press("Enter", "Alt-Esc")
	if p.InMode() != nil || ta.top() != nil || *left != 2 || sent.String() != "\x1b" {
		t.Errorf("Alt-Esc: in the mode %v, %T open, OnExit %d, sent %q", p.InMode() != nil, ta.topView(), *left, sent.String())
	}
}

// TestK11ModeHoldsPTYOverExitKey: in the mode the Terminal's own exit key is
// a key like any for the program, and its History is not there: PgUp is the
// program's (tdp K11).
func TestK11ModeHoldsPTYOverExitKey(t *testing.T) {
	ta, term, sent, p, _ := ptyModeApp(t)
	term.ExitKey, term.ExitAsks, term.Interactive = KeyName(tea.KeyMsg{Type: tea.KeyCtrlX}), true, true
	ta.Update(tea.KeyMsg{Type: tea.KeyCtrlX})
	ta.press("PgUp")
	if p.InMode() == nil || ta.top() != nil || term.Mode() != "" || sent.String() != "\x18\x1b[5~" {
		t.Errorf("in the mode %v, %T open, terminal mode %q, sent %q", p.InMode() != nil, ta.topView(), term.Mode(), sent.String())
	}
}

// TestTerminalSend: Send writes to the program from outside its PTY, and
// does nothing while it is not running (tdp K11, K4).
func TestTerminalSend(t *testing.T) {
	sent := &bytes.Buffer{}
	term := &Terminal{in: sent, started: true}
	term.Send([]byte("\x1b"))
	term.Send(nil)
	if sent.String() != "\x1b" {
		t.Errorf("sent %q", sent.String())
	}
	term.exited = true
	term.Send([]byte("x"))
	if (&Terminal{}).Send([]byte("x")); sent.String() != "\x1b" {
		t.Errorf("not running, sent %q", sent.String())
	}
}

// TestK11ModePTYNeedsTerminal: a mode holding a PTY on a panel with none is
// the app's mistake, found at once.
func TestK11ModePTYNeedsTerminal(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("no panic")
		}
	}()
	(&Panel{Label: "Pods", Body: &Pager{}}).EnterMode(&Mode{Name: "Typing", PTY: true})
}
