package terminu

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ptyApp is the test app with a terminal panel on a screen of its own; the
// terminal writes its keys into sent instead of a PTY.
func ptyApp(t *testing.T) (*testApp, *Terminal, *bytes.Buffer) {
	t.Helper()
	ta := newTestApp(t)
	sent := &bytes.Buffer{}
	term := &Terminal{Title: "Shell", Interactive: true, in: sent, started: true}
	ta.AddScreen(&Screen{Name: "Terminal", Key: "T", Panels: []*Panel{
		{Label: "Shell", Body: term, Actions: []Action{{Key: "Enter", Label: "Type", Scope: PanelScope, Do: func() tea.Cmd { term.Enter(); return nil }}}},
		{Label: "Other", Body: &Pager{}},
	}})
	ta.press("T", "Enter")
	if !term.Inside() {
		t.Fatal("Enter did not go into the terminal")
	}
	return ta, term, sent
}

// termWithHistory is a panel PTY with lines 1 to 100 printed, most of them
// gone into the history.
func termWithHistory(t *testing.T) (*testApp, *Terminal, *bytes.Buffer) {
	ta, term, sent := ptyApp(t)
	var b strings.Builder
	for i := 1; i <= 100; i++ {
		fmt.Fprintf(&b, "line %d\r\n", i)
	}
	ta.Frame() // sized
	term.mu.Lock()
	term.vt().write([]byte(b.String()))
	term.mu.Unlock()
	return ta, term, sent
}

// TestTerminalEmptyUntilItSpeaks: started, it is empty until its program
// prints, so the panel's LoadingText shows in its middle (tdp panel).
func TestTerminalEmptyUntilItSpeaks(t *testing.T) {
	ta, term, _ := ptyApp(t)
	ta.Screen().Panels[0].Loading, ta.Screen().Panels[0].LoadingText = true, "connecting to web-01…"
	if !term.Empty() || !strings.Contains(ta.screenText(), "connecting to web-01…") {
		t.Errorf("started, nothing printed: empty, LoadingText shown: %v", term.Empty())
	}
	term.spoke = true
	if term.Empty() || strings.Contains(ta.screenText(), "connecting to web-01…") {
		t.Error("once it prints, its screen")
	}
}

// TestTerminalHistoryMode: PgUp goes into the History mode; its keys scroll
// back, Esc goes back to the live screen, any other key leaves it and goes to
// the program, q too (tdp terminal, tdp K11).
// altV is Alt-v as the terminal sends it.
var altV = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v"), Alt: true}

// TestTerminalNotInteractive: a PTY the app opens to run one program has no
// History mode and no selection: PgUp and Alt-v go to the program, the hint
// does not offer them (tdp terminal, terminal-21).
func TestTerminalNotInteractive(t *testing.T) {
	ta, term, sent := termWithHistory(t)
	term.Interactive = false
	ta.press("PgUp")
	ta.Update(altV)
	ta.settle()
	if term.Mode() != "" || sent.String() != "\x1b[5~\x1bv" {
		t.Errorf("mode %q, sent %q: both keys are the program's", term.Mode(), sent.String())
	}
	if f := StripStyles(ta.Frame()[ta.h-1]); strings.Contains(f, "select") || strings.Contains(f, "history") {
		t.Errorf("the footer offers a mode: %q", f)
	}
}

// TestTerminalFullScreen: while the program fills the whole screen (the
// alternate one: vim, less) PgUp is the program's and the hint leaves
// PgUp:history out; the selection holds that screen alone; back on the
// normal screen PgUp goes into the history again (terminal-22).
func TestTerminalFullScreen(t *testing.T) {
	ta, term, sent := termWithHistory(t)
	write := func(s string) {
		term.mu.Lock()
		term.vt().write([]byte(s))
		term.mu.Unlock()
	}
	write("\x1b[?1049hless screen")
	if f := StripStyles(ta.Frame()[ta.h-1]); !strings.Contains(f, "Alt-v:select") || strings.Contains(f, "PgUp:history") {
		t.Errorf("the footer while full screen: %q", f)
	}
	ta.press("PgUp")
	if term.Mode() != "" || sent.String() != "\x1b[5~" {
		t.Errorf("PgUp went to the history: mode %q, sent %q", term.Mode(), sent.String())
	}
	ta.Update(altV)
	ta.settle()
	if term.Mode() != "Visual" || len(term.sel.Lines) != term.vt().h || term.sel.line != 0 {
		t.Errorf("the selection holds the screen alone: %q, %d lines", term.Mode(), len(term.sel.Lines))
	}
	ta.press("Esc")
	write("\x1b[?1049l")
	ta.press("PgUp")
	if term.Mode() != "History" {
		t.Errorf("back on the normal screen PgUp is the history again: %q", term.Mode())
	}
}

// TestTerminalAllKeys: when the user hands every key to the program, Alt-v
// and PgUp|PgDn go to it; the app's own keys (the one taking them back) and
// the exit key still work; taken back, the modes return (terminal-23).
func TestTerminalAllKeys(t *testing.T) {
	ta, term, sent := termWithHistory(t)
	term.AllKeys = true
	term.Keys = []TerminalKey{{Key: "Alt-u", Desc: "unlock", Do: func() tea.Cmd { term.AllKeys = false; return nil }}}
	if f := StripStyles(ta.Frame()[ta.h-1]); !strings.Contains(f, "Alt-u:unlock") || strings.Contains(f, "select") || strings.Contains(f, "history") {
		t.Errorf("the footer while every key is the program's: %q", f)
	}
	ta.Update(altV)
	ta.settle()
	ta.press("PgUp", "PgDn")
	if term.Mode() != "" || sent.String() != "\x1bv\x1b[5~\x1b[6~" {
		t.Errorf("mode %q, sent %q", term.Mode(), sent.String())
	}
	ta.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u"), Alt: true})
	ta.settle()
	ta.press("PgUp")
	if term.AllKeys || term.Mode() != "History" {
		t.Errorf("taken back, PgUp goes into the history: %v %q", term.AllKeys, term.Mode())
	}
}

func TestTerminalHistoryMode(t *testing.T) {
	ta, term, sent := termWithHistory(t)
	ta.press("PgUp")
	if term.Mode() != "History" || term.back == 0 {
		t.Fatalf("PgUp: the History mode, scrolled back: %q %d", term.Mode(), term.back)
	}
	if top := StripStyles(ta.rowWith("History")); !strings.Contains(top, "┫History┣") {
		t.Errorf("the mode on the frame: %q", top)
	}
	if f := StripStyles(ta.Frame()[ta.h-1]); !strings.Contains(f, "?:help j|k:scroll Esc:live") {
		t.Errorf("the footer has the mode's keys: %q", f)
	}
	back := term.back
	ta.press("k")
	if term.back != back+1 {
		t.Errorf("k: a line further back: %d", term.back)
	}
	ta.press("g", "g")
	if term.back != len(term.vt().history) {
		t.Errorf("gg: the oldest line: %d", term.back)
	}
	ta.press("G")
	if term.back != 0 || term.Mode() != "History" {
		t.Errorf("G: the end, still in the mode: %d %q", term.back, term.Mode())
	}
	if sent.Len() != 0 {
		t.Errorf("the mode's keys went to the program: %q", sent.String())
	}
	ta.press("Esc")
	if term.Mode() != "" || sent.Len() != 0 {
		t.Error("Esc goes back to the live screen, sending nothing")
	}
	ta.press("PgUp", "x")
	if term.Mode() != "" || term.back != 0 || sent.String() != "x" {
		t.Errorf("another key leaves the mode and goes to the program: %q %q", term.Mode(), sent.String())
	}
	ta.press("PgUp", "q")
	if term.Mode() != "" || sent.String() != "xq" || ta.quitting {
		t.Errorf("q too: %q, quitting %v", sent.String(), ta.quitting)
	}
	ta.press("PgUp", "?")
	if n, ok := ta.topView().(*noteView); !ok || n.title != "HISTORY KEYS" {
		t.Errorf("? is the mode's key reference: %T", ta.topView())
	}
}

// TestTerminalSelectMode: Alt-v puts an interactive terminal in the
// selection mode, a viewer's: the screen holds still, the cursor goes up
// into the history, y copies with a Success toast (tdp terminal).
func TestTerminalSelectMode(t *testing.T) {
	ta, term, sent := termWithHistory(t)
	takeClipboard()
	if f := StripStyles(ta.Frame()[ta.h-1]); !strings.Contains(f, "Alt-v:select") {
		t.Errorf("the app's key in the footer: %q", f)
	}
	ta.press("Alt-v")
	if term.Mode() != "Visual" {
		t.Fatalf("the selection mode: %q", term.Mode())
	}
	before := ta.screenText()
	term.mu.Lock()
	term.vt().write([]byte("printed meanwhile\r\n"))
	term.mu.Unlock()
	if ta.screenText() != before {
		t.Error("the screen holds still")
	}
	if top := StripStyles(ta.rowWith("Visual")); !strings.Contains(top, "┫Visual┣") {
		t.Errorf("the mode on the frame: %q", top)
	}
	ta.press("k", "v", "l", "l", "l", "y")
	if got := string(takeClipboard()); !strings.HasPrefix(got, "\x1b]52;c;") || term.Mode() != "" {
		t.Errorf("copied, back to the live screen: %q %q", got, term.Mode())
	}
	if ta.toast == nil || ta.toast.view.(*toastView).kind != Success {
		t.Error("a Success toast says so")
	}
	if !strings.Contains(ta.screenText(), "printed meanwhile") || sent.Len() != 0 {
		t.Errorf("live again; nothing sent: %q", sent.String())
	}
	ta.press("Alt-v", "Esc")
	if term.Mode() != "" || sent.Len() != 0 {
		t.Error("Esc leaves, sending nothing")
	}
}

// TestTerminalExitKey: the exit key is the app's, Alt-Esc unless it says;
// the hint shows it as it is now; another asks first only when the app says
// (tdp K10; sshu SS5, SS6).
func TestTerminalExitKey(t *testing.T) {
	ta, term, sent := ptyApp(t)
	term.ExitKey = "Alt-q"
	if f := StripStyles(ta.Frame()[ta.h-1]); !strings.Contains(f, "Alt-q:leave") {
		t.Errorf("the footer shows the exit key as it is: %q", f)
	}
	ta.Update(tea.KeyMsg{Type: tea.KeyEscape, Alt: true})
	if !term.Inside() || sent.String() != "\x1b\x1b" {
		t.Errorf("Alt-Esc is the program's now: %q", sent.String())
	}
	ta.press("Alt-q")
	if term.Inside() || ta.top() != nil {
		t.Error("Alt-q leaves without asking")
	}
	ta.press("Enter")
	term.ExitAsks = true
	ta.press("Alt-q")
	ta.settle()
	if _, ok := ta.topView().(*confirmView); !ok {
		t.Errorf("asks first when the app says: %T", ta.topView())
	}
}

func TestK10KeysToChild(t *testing.T) {
	ta, _, sent := ptyApp(t)
	ta.press("Space", "?", "q", "Tab", "Esc", "1", "Ctrl-C")
	if got := sent.String(); got != " ?q\t\x1b1\x03" {
		t.Errorf("the program got %q", got)
	}
	if ta.top() != nil || ta.quitting || ta.Screen().focus != 0 {
		t.Error("a core key worked in the PTY")
	}
}

func TestK10ExitKeyShown(t *testing.T) {
	ta, _, _ := ptyApp(t)
	foot := StripStyles(ta.footer(120))
	if !strings.HasPrefix(foot, "Alt-Esc:leave") || !strings.Contains(foot, "PgUp:history") {
		t.Errorf("footer in the PTY: %q", foot)
	}
}

func TestK10AltEscConfirms(t *testing.T) {
	ta, term, sent := ptyApp(t)
	ta.press("Alt-Esc")
	if _, ok := ta.topView().(*confirmView); !ok {
		t.Fatalf("Alt-Esc did not ask: %T", ta.topView())
	}
	ta.press("Esc", "x")
	if !term.Inside() || sent.String() != "x" {
		t.Errorf("Esc on the confirm goes back into the PTY: inside %v, sent %q", term.Inside(), sent.String())
	}
	ta.press("Alt-Esc", "Enter")
	if term.Inside() {
		t.Error("Enter on the confirm did not leave")
	}
	ta.press("Tab")
	if ta.Screen().focus != 1 {
		t.Error("out of the PTY, Tab works again")
	}
}

func TestK10NotReadyForwardsCtrlC(t *testing.T) {
	ta, term, sent := ptyApp(t)
	ready := false
	term.NotReady = func() bool { return !ready }
	ta.press("a", "Ctrl-C")
	if sent.String() != "\x03" {
		t.Errorf("before it is ready only Ctrl-C goes through: %q", sent.String())
	}
	ta.press("Alt-Esc")
	if _, ok := ta.topView().(*confirmView); !ok {
		t.Error("the exit key works before it is ready")
	}
}

func TestEncodeKey(t *testing.T) {
	cases := []struct {
		msg       tea.KeyMsg
		appCursor bool
		want      string
	}{
		{tea.KeyMsg{Type: tea.KeyUp}, false, "\x1b[A"},
		{tea.KeyMsg{Type: tea.KeyUp}, true, "\x1bOA"},
		{tea.KeyMsg{Type: tea.KeyEnter}, false, "\r"},
		{tea.KeyMsg{Type: tea.KeyBackspace}, false, "\x7f"},
		{tea.KeyMsg{Type: tea.KeyCtrlA}, false, "\x01"},
		{tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x"), Alt: true}, false, "\x1bx"},
		{tea.KeyMsg{Type: tea.KeyF5}, false, "\x1b[15~"},
		{tea.KeyMsg{Type: tea.KeyShiftTab}, false, "\x1b[Z"},
		{tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("中")}, false, "中"},
	}
	for _, c := range cases {
		if got := string(encodeKey(c.msg, c.appCursor, false)); got != c.want {
			t.Errorf("%v: %q, want %q", c.msg, got, c.want)
		}
	}
	paste := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a\nb"), Paste: true}
	if got := string(encodeKey(paste, false, true)); got != "\x1b[200~a\nb\x1b[201~" {
		t.Errorf("bracketed paste: %q", got)
	}
}

// runTerminal starts t and feeds its news to the app until the program ends.
func runTerminal(t *testing.T, ta *testApp, term *Terminal, start tea.Cmd) {
	t.Helper()
	cmds := []tea.Cmd{start}
	deadline := time.Now().Add(5 * time.Second)
	for len(cmds) > 0 && time.Now().Before(deadline) {
		cmd := cmds[0]
		cmds = cmds[1:]
		if cmd == nil {
			continue
		}
		switch msg := cmd().(type) {
		case tea.BatchMsg:
			cmds = append(cmds, msg...)
		case termOutput, termExited:
			cmds = append(cmds, ta.Update2(msg))
		}
		if term.Exited() {
			return
		}
	}
	if !term.Exited() {
		t.Fatal("the program did not end")
	}
}

func TestE5ChildEnv(t *testing.T) {
	withWidths(t, 2, 1)
	ta := newTestApp(t)
	term := &Terminal{Command: []string{"sh", "-c", `printf "%s%s" "$TERMINU__ICON_WIDTH" "$TERMINU__POWERLINE_WIDTH"`}}
	runTerminal(t, ta, term, term.Start())
	if got := term.vt().text()[0]; got != "21" {
		t.Errorf("the program saw %q, want the app's widths 21", got)
	}
}

func TestTerminalPopupSize(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(100, 30)
	term := &Terminal{Title: "Alterm", Subject: "myhost", Command: []string{"sh", "-c", "printf ready; sleep 2"}, Ends: true}
	open := ta.OpenTerminal(term)
	defer term.Stop()
	ta.settle()
	l := ta.top()
	if l.w != 98 || l.h != 28 || l.x != 1 || l.y != 2 {
		t.Errorf("the terminal popup is %dx%d at %d,%d; want 98x28 at 1,2 (under the chips and divider)", l.w, l.h, l.x, l.y)
	}
	_ = open
	lines := ta.Frame()
	last := StripStyles(lines[len(lines)-1])
	if !strings.Contains(last, "┗━ Alt-Esc:exit") {
		t.Errorf("the popup covers the footer; the last row is %q", last)
	}
	// No footer: the ends of the last row are blank, none of its words show
	// (tdp terminal).
	if !strings.HasPrefix(last, " ┗") || !strings.HasSuffix(last, "┛ ") {
		t.Errorf("the footer shows at the ends of the last row: %q", last)
	}
	if !strings.Contains(StripStyles(lines[2]), "ALTERM: myhost") {
		t.Errorf("title: %q", StripStyles(lines[2]))
	}
	// No padding: the program's first row is the row under the top border.
	term.mu.Lock()
	term.vt().write([]byte("\x1b[Hhello"))
	term.mu.Unlock()
	if row := StripStyles(ta.Frame()[3]); !strings.HasPrefix(row, "┃┃hello") {
		t.Errorf("the PTY touches the frame: %q", row)
	}
}

func TestToastPositionOverTerminal(t *testing.T) {
	ta, _, sent := ptyApp(t)
	term := &Terminal{Title: "Shell", in: sent, started: true}
	ta.open(&termView{t: term, a: ta.App})
	ta.settle()
	ta.Toast(Info, "copied")
	ta.settle()
	if ta.toast.y+ta.toast.h != ta.h-2 {
		t.Errorf("the toast's bottom border is on row %d, want the third from the bottom", ta.toast.y+ta.toast.h-1)
	}
	ta.press("Esc")
	if ta.toast.closing || sent.String() != "\x1b" {
		t.Error("in a PTY Esc goes to the program, the toast waits for its time")
	}
}

// TestTerminalResizeWhileExiting: drawn at a new size while its program ends,
// before the update loop hears of it, a terminal resizes the PTY its reader
// is closing; the two are kept apart (sshu SS10, go test -race).
func TestTerminalResizeWhileExiting(t *testing.T) {
	term := &Terminal{Command: []string{"sh", "-c", "exit 0"}}
	term.Start()
	if term.Exited() {
		t.Skip("no PTY here")
	}
	ch := term.out
	for i := 0; ; i++ {
		term.resize(80+i%2, 24)
		select {
		case _, ok := <-ch:
			if !ok {
				term.resize(90, 30) // a draw after the PTY closed
				return
			}
		case <-time.After(time.Millisecond):
		}
		if i > 5000 {
			t.Fatal("the program did not end")
		}
	}
}

// TestTerminalKeyTakesSeveral: one of the app's keys in a PTY can take
// several keys, written once in the footer, and its DoKey hears which
// (tdp K10, M5; sshu SS12).
func TestTerminalKeyTakesSeveral(t *testing.T) {
	ta, term, sent := ptyApp(t)
	var got []string
	term.Keys = []TerminalKey{{Key: "Alt-←|→|↑|↓", Keys: []string{"Alt-←", "Alt-→", "Alt-↑", "Alt-↓"}, Desc: "pane",
		DoKey: func(key string) tea.Cmd { got = append(got, key); return nil }}}
	for _, k := range []tea.KeyMsg{{Type: tea.KeyRight, Alt: true}, {Type: tea.KeyUp, Alt: true}} {
		ta.Update2(k)
	}
	if strings.Join(got, " ") != "Alt-→ Alt-↑" || sent.Len() != 0 {
		t.Errorf("the app got %v, the program %q", got, sent.String())
	}
	if footer := covCFooter(ta); strings.Count(footer, "Alt-←|→|↑|↓:pane") != 1 {
		t.Errorf("footer %q", footer)
	}
}

// TestTerminalFirstOutput: a terminal tells the app once when its program
// first prints, so the app can end the panel's loading (tdp K10, F7; sshu
// SS11).
func TestTerminalFirstOutput(t *testing.T) {
	ta := newTestApp(t)
	calls := 0
	term := &Terminal{Command: []string{"sh", "-c", "sleep 0.1; printf a; sleep 0.1; printf b"},
		OnFirstOutput: func() tea.Cmd { calls++; return nil }}
	start := term.Start()
	if term.Exited() {
		t.Skip("no PTY here")
	}
	if term.Spoke() {
		t.Error("it spoke before the program printed")
	}
	runTerminal(t, ta, term, start)
	if !term.Spoke() || calls != 1 {
		t.Errorf("spoke %v, heard %d times", term.Spoke(), calls)
	}
}
