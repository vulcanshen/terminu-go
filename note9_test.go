package terminu

import (
	"fmt"
	"strings"
	"testing"
)

// TestNote9AppLogFollow: the open App Log follows its end while its last
// line is in sight; scrolled up, it stays on its lines; in the selection
// mode it never follows, the view and the cursor staying where they were
// (tdp note.9, note-90, note-91).
func TestNote9AppLogFollow(t *testing.T) {
	ta, _ := logApp(t)
	for i := range 60 {
		ta.Log(Info, fmt.Sprintf("entry %d", i))
	}
	ta.press("!")
	v := ta.topView().(*viewerView).v
	ta.Log(Info, "came in at the end")
	if !strings.Contains(ta.screenText(), "came in at the end") {
		t.Fatal("at its end, it does not follow")
	}

	ta.press("k", "k", "k")
	top := v.top
	ta.Log(Info, "came in scrolled up")
	if text := ta.screenText(); v.top != top || strings.Contains(text, "came in scrolled up") {
		t.Errorf("scrolled up, it moved: top %d, was %d\n%s", v.top, top, text)
	}

	ta.press("G", "v", "j", "j", "j") // the cursor below the first row in sight
	top, line := v.top, v.line
	ta.Log(Info, "came in with the cursor out")
	if text := ta.screenText(); v.top != top || v.line != line || strings.Contains(text, "came in with the cursor out") {
		t.Errorf("Viewing: it followed: top %d line %d, were %d %d", v.top, v.line, top, line)
	}
	ta.press("v", "j")
	top, line = v.top, v.line
	ta.Log(Info, "came in while selecting")
	if text := ta.screenText(); v.top != top || v.line != line || !v.selecting || strings.Contains(text, "came in while selecting") {
		t.Errorf("Viewing:selecting: it followed: top %d line %d, were %d %d", v.top, v.line, top, line)
	}
}

// TestNote9AppLogOldestGo: when the App Log is full and its oldest entries
// go, the lines in sight and the selection mode's cursor and start stay on
// the entries they were on (tdp note.9, note-90).
func TestNote9AppLogOldestGo(t *testing.T) {
	ta, _ := logApp(t)
	for i := range logMost {
		ta.Log(Info, fmt.Sprintf("entry %04d", i))
	}
	ta.press("!", "g", "g")
	for range 30 {
		ta.press("j")
	}
	v := ta.topView().(*viewerView).v
	ta.screenText()
	first := v.Lines[v.top]
	for i := range 5 {
		ta.Log(Info, fmt.Sprintf("newer %d", i))
	}
	ta.screenText()
	if got := v.Lines[v.top]; got != first {
		t.Errorf("scrolled up, the first line in sight changed: %q, was %q", StripStyles(got), StripStyles(first))
	}

	ta.press("v", "j", "v", "j", "j")
	cur, from := v.Lines[v.line], v.Lines[v.fromL]
	first = v.Lines[v.top]
	for i := range 7 {
		ta.Log(Info, fmt.Sprintf("newest %d", i))
	}
	ta.screenText()
	if v.Lines[v.line] != cur || v.Lines[v.fromL] != from || v.Lines[v.top] != first {
		t.Errorf("selecting: cursor %q start %q top %q, were %q %q %q", StripStyles(v.Lines[v.line]), StripStyles(v.Lines[v.fromL]),
			StripStyles(v.Lines[v.top]), StripStyles(cur), StripStyles(from), StripStyles(first))
	}
}
