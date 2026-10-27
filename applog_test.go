package terminu

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// logApp is a test app on a still clock at 14:02:11.
func logApp(t *testing.T) (*testApp, *stillClock) {
	ta := newTestApp(t)
	clk := &stillClock{time.Date(2026, 10, 20, 14, 2, 11, 0, time.Local)}
	ta.testClock = clk
	return ta, clk
}

// TestAppLogOnBang: ! on a panel opens the App Log, a viewer; not in a mode,
// not on a typing row (tdp K1, K11; tdp note, App Log).
func TestAppLogOnBang(t *testing.T) {
	ta, _ := logApp(t)
	ta.press("!")
	if vv, ok := ta.topView().(*viewerView); !ok || vv.title() != "APP LOG" {
		t.Fatalf("! opened %T", ta.topView())
	}
	ta.press("Esc", "D") // Drag: a mode
	ran := false
	ta.pods.InMode().Keys = append(ta.pods.InMode().Keys, ModeKey{Key: "!", Desc: "x", Do: func(string) tea.Cmd { ran = true; return nil }})
	ta.press("!")
	if ta.top() != nil || ran {
		t.Error("! acted in a mode")
	}
	ta.press("Esc", "/", "!")
	if ta.top() != nil || ta.list.filter.value() != "!" {
		t.Error("! on a typing row is a character")
	}
	if checkPanelKey("!") == nil {
		t.Error("! was taken as a panel's hotkey")
	}
	ta.CoreKeys = []CoreKey{{Key: "!", Desc: "mine", Do: func() tea.Cmd { return nil }}}
	if ta.checkKeys() == nil {
		t.Error("an app's core key took !")
	}
}

// TestToastKeepsCase: the words are the app's: a row that starts with a name
// keeps its case, in the toast and in the App Log (tdp toast.2).
func TestToastKeepsCase(t *testing.T) {
	ta, _ := logApp(t)
	ta.Toast(Success, "kbu.yaml saved")
	ta.settle()
	if !strings.Contains(ta.screenText(), "kbu.yaml saved") {
		t.Errorf("the toast changed the name:\n%s", ta.screenText())
	}
	ta.press("Esc", "!")
	if !strings.Contains(ta.screenText(), "OK   kbu.yaml saved") {
		t.Errorf("the App Log changed the name:\n%s", ta.screenText())
	}
}

// TestAppLogScreenSwitch: a switch of screens the user runs, by the screen's
// key or from the global operation popup, goes in by itself; what is not an
// operation does not: moving the cursor or the focus, the Space menu, ?, !.
// A switch the app makes (ShowScreen) is the app's to log (tdp note.7,
// note-77).
func TestAppLogScreenSwitch(t *testing.T) {
	ta, _ := logApp(t)
	ta.press("j", "k", "Tab", "1", "Space", "Esc", "?", "Esc", "!", "Esc")
	ta.settle()
	ta.press("O")
	ta.press("Space", "G", "Enter", "P")
	ta.settle()
	ta.ShowScreen(ta.screens[1])
	var got []string
	for _, e := range ta.logs.entries {
		got = append(got, e.text)
	}
	if want := []string{"Switched to Other", "Switched to Panels"}; fmt.Sprint(got) != fmt.Sprint(want) || ta.cur != 1 {
		t.Errorf("logged %q, want %q; on screen %d", got, want, ta.cur)
	}
}

// TestAppLogEntries: each entry is its time HH:MM:SS in Overlay0, its level
// four cells wide in its colour, what happened; every toast and error popup
// goes in, the app logs its own; newest at the bottom, the log open at its
// end, new entries coming in while it is open (note-76, note-77, note-78;
// tdp F5).
func TestAppLogEntries(t *testing.T) {
	ta, clk := logApp(t)
	ta.Toast(Info, "Switched to Pods")
	ta.Toast(Success, "Scaled nginx")
	ta.Log(Warning, "CRD discovery: unable")
	ta.Error("Fetch", "Forbidden")
	ta.settle()
	ta.press("Esc", "Esc")
	ta.press("!")
	text := ta.screenText()
	for _, want := range []string{"14:02:11 INFO Switched to Pods", "14:02:11 OK   Scaled nginx", "14:02:11 WARN CRD discovery: unable", "14:02:11 ERR  Fetch: Forbidden"} {
		if !strings.Contains(text, want) {
			t.Errorf("%q is not in the App Log:\n%s", want, text)
		}
	}
	frame := strings.Join(ta.Frame(), "\n")
	for _, want := range []string{Fg(Overlay0).Render("14:02:11"), Fg(Text).Render("INFO"), Fg(Green).Render("OK  "), Fg(Peach).Render("WARN"), Fg(Red).Render("ERR ")} {
		if !strings.Contains(frame, want) {
			t.Errorf("not in its colour: %q", want)
		}
	}
	if strings.Index(text, "Switched") > strings.Index(text, "Forbidden") {
		t.Error("the newest is not at the bottom")
	}
	clk.t = clk.t.Add(time.Minute)
	ta.Log(Info, "Came in while open")
	if !strings.Contains(ta.screenText(), "14:03:11 INFO Came in while open") {
		t.Error("an entry logged while it is open does not come in")
	}
	// Opened at its end: with more than fits, the newest is in sight.
	for i := range 60 {
		ta.Log(Info, fmt.Sprintf("entry %d", i))
	}
	if !strings.Contains(ta.screenText(), "entry 59") {
		t.Error("the open log does not follow its end")
	}
	ta.press("Esc", "!")
	if !strings.Contains(ta.screenText(), "entry 59") {
		t.Error("it does not open at its end")
	}
}

// TestAppLogInMemory: in memory only, so many entries and the oldest go
// first (note-79).
func TestAppLogInMemory(t *testing.T) {
	ta, _ := logApp(t)
	for i := range logMost + 10 {
		ta.Log(Info, fmt.Sprintf("entry %d", i))
	}
	if n := len(ta.logs.entries); n != logMost || ta.logs.entries[0].text != "entry 10" {
		t.Errorf("%d entries kept, the first %q", n, ta.logs.entries[0].text)
	}
}

// TestAppLogFooter: the footer's !:log is !:N errors in Red while there are
// unseen errors, !:N warning(s) in Peach while only warnings; opening the
// App Log sees them (screen-35).
func TestAppLogFooter(t *testing.T) {
	ta, _ := logApp(t)
	footer := func() string { return StripStyles(ta.Frame()[ta.h-1]) }
	if !strings.Contains(footer(), "?:help !:log Tab") {
		t.Fatalf("the footer: %q", footer())
	}
	ta.Log(Warning, "w")
	if !strings.Contains(footer(), "!:1 warning ") || !strings.Contains(ta.Frame()[ta.h-1], Fg(Peach).Render("1 warning")) {
		t.Errorf("a warning: %q", footer())
	}
	ta.Log(Warning, "w")
	ta.Log(Error, "e")
	if !strings.Contains(footer(), "!:1 error ") || !strings.Contains(ta.Frame()[ta.h-1], Fg(Red).Render("1 error")) {
		t.Errorf("an error before the warnings: %q", footer())
	}
	ta.Log(Error, "e")
	if !strings.Contains(footer(), "!:2 errors") {
		t.Errorf("two errors: %q", footer())
	}
	ta.press("!")
	ta.Log(Error, "seen as it comes")
	ta.Frame()
	ta.press("Esc")
	if !strings.Contains(footer(), "!:log") {
		t.Errorf("opened, all seen: %q", footer())
	}
	ta.Log(Error, "after")
	if !strings.Contains(footer(), "!:1 error") {
		t.Errorf("closed, unseen again: %q", footer())
	}
}
