package terminu

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// fileLines is n lines of a pretend file, "needle" in the lines named.
func fileLines(n int, needles ...int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("line %d of the file", i)
		if slices.Contains(needles, i) {
			out[i] += " with a needle"
		}
	}
	return out
}

// openFile opens a viewer of a pretend file and draws it.
func openFile(t *testing.T, lines []string) (*testApp, *Viewer) {
	ta := newTestApp(t)
	v := &Viewer{Title: "File", Subject: "notes.txt", Lines: lines}
	ta.Viewer(v)
	ta.settle()
	ta.screenText()
	return ta, v
}

// TestNote9ReadAgainFollow: read again, a viewer whose end was in sight
// goes to the new end; scrolled up it stays where it was; in the selection
// mode it never follows; fewer lines than fill it, it is at the new end
// (tdp note.9, note-90, note-91, note-92).
func TestNote9ReadAgainFollow(t *testing.T) {
	ta, v := openFile(t, fileLines(100))
	ta.press("G")
	ta.ReadAgain(v, fileLines(120))
	if text := ta.screenText(); !strings.Contains(text, "line 119 of") {
		t.Errorf("at its end, it does not follow:\n%s", text)
	}
	ta.press("k", "k", "k")
	top := v.top
	ta.ReadAgain(v, fileLines(140))
	if ta.screenText(); v.top != top {
		t.Errorf("scrolled up, it moved: %d, was %d", v.top, top)
	}
	ta.press("G", "v", "j", "j")
	top, line := v.top, v.line
	ta.ReadAgain(v, fileLines(160))
	if text := ta.screenText(); v.top != top || v.line != line || strings.Contains(text, "line 159 of") {
		t.Errorf("in the selection mode it followed: top %d line %d, were %d %d", v.top, v.line, top, line)
	}
	ta.press("Esc")
	ta.ReadAgain(v, fileLines(5))
	if text := ta.screenText(); v.top != 0 || !strings.Contains(text, "line 4 of") {
		t.Errorf("fewer lines: not at the new end, top %d:\n%s", v.top, text)
	}
}

// TestNote9ReadAgainGutter: read again, the line numbers' column only
// widens while the viewer is open; opened again it starts over (tdp note.9,
// note.2, note-93, note-13).
func TestNote9ReadAgainGutter(t *testing.T) {
	ta, v := openFile(t, fileLines(50))
	if v.numW != 3 {
		t.Fatalf("50 lines: %d", v.numW)
	}
	ta.ReadAgain(v, fileLines(1200))
	ta.screenText()
	if v.numW != 4 {
		t.Errorf("1200 lines: %d", v.numW)
	}
	ta.ReadAgain(v, fileLines(50))
	ta.screenText()
	if v.numW != 4 {
		t.Errorf("50 lines again: it narrowed to %d", v.numW)
	}
	ta.press("Esc")
	ta.Viewer(v)
	ta.settle()
	ta.screenText()
	if v.numW != 3 {
		t.Errorf("opened again: %d", v.numW)
	}
}

// TestNote9ReadAgainSearch: read again, the lib's search is found again:
// the first hit at or after the current one is current, the first of all
// when none is; the view does not move; with none left, an Info toast as a
// search that finds nothing, the hits and the status gone; an open key
// reference stays as it was (tdp note.9, note-94, note-95).
func TestNote9ReadAgainSearch(t *testing.T) {
	ta, v := openFile(t, fileLines(80, 10, 30, 50))
	ta.press("/", "needle", "Enter", "n")
	if v.Current != 1 || len(v.Hits) != 3 {
		t.Fatalf("searched: current %d of %d", v.Current, len(v.Hits))
	}
	top := v.top
	ta.ReadAgain(v, fileLines(80, 5, 35, 55, 70))
	text := ta.screenText()
	if len(v.Hits) != 4 || v.Hits[v.Current].Line != 35 || v.top != top || !strings.Contains(text, "/needle 2 of 4") {
		t.Errorf("found again: current on line %d of %d hits, top %d (was %d)", v.Hits[v.Current].Line, len(v.Hits), v.top, top)
	}
	ta.ReadAgain(v, fileLines(80, 5, 35, 55, 70))
	if v.Current != 1 {
		t.Errorf("the same text: current %d, was 1", v.Current)
	}
	ta.ReadAgain(v, fileLines(80, 5, 20))
	if ta.screenText(); v.Current != 0 || len(v.Hits) != 2 || v.top != top {
		t.Errorf("none at or after it: current %d of %d, top %d", v.Current, len(v.Hits), v.top)
	}

	ta.press("?")
	ref := ta.topView()
	ta.ReadAgain(v, fileLines(80))
	ta.settle()
	text = ta.screenText()
	if len(v.Hits) != 0 || v.Query != "" || !strings.Contains(text, "No line has needle") || strings.Contains(text, "/needle") {
		t.Errorf("none left: hits %v query %q\n%s", v.Hits, v.Query, text)
	}
	if ta.topView() != ref || !strings.Contains(text, "n|N") {
		t.Errorf("the open key reference changed: %T\n%s", ta.topView(), text)
	}
	ta.press("Esc")

	// An app's own search is the app's to find again.
	ta2 := newTestApp(t)
	own := &Viewer{Title: "File", Lines: fileLines(20, 3), Keys: []ViewerKey{{Key: "/", Desc: "search", Do: func() tea.Cmd { return nil }}}}
	ta2.Viewer(own)
	ta2.settle()
	own.Hits, own.Query = []TextMatch{{Line: 3, From: 0, To: 4}}, "line"
	ta2.ReadAgain(own, fileLines(20))
	if len(own.Hits) != 1 || own.Query != "line" {
		t.Errorf("an app's own search was found again: %v %q", own.Hits, own.Query)
	}
}

// TestNote9ReadAgainSelect: read again in the selection mode, its positions
// are kept, not its text: both still there, nothing changes and y copies the
// new text; where the selection started gone, back to Viewing with a
// Warning; the cursor's place gone, the mode is left with a Warning; the
// source unreadable, the mode is left with no toast of the lib's (tdp
// note.9, note-96, note-97, note-98).
func TestNote9ReadAgainSelect(t *testing.T) {
	copied := ""
	open := func() (*testApp, *Viewer) {
		ta := newTestApp(t)
		v := &Viewer{Title: "File", Subject: "notes.txt", Lines: fileLines(80),
			Select: &Selection{Key: "v", Name: "Viewing", Copy: func(s string) tea.Cmd { copied = s; return nil }}}
		ta.Viewer(v)
		ta.settle()
		ta.screenText()
		v.ScrollTo(10)
		return ta, v
	}

	ta, v := open()
	ta.press("v", "v", "j", "j", "$")
	changed := fileLines(80)
	for i := 10; i <= 12; i++ {
		changed[i] = strings.ToUpper(changed[i]) // as long: the cursor at its end is still there
	}
	ta.ReadAgain(v, changed)
	if text := ta.screenText(); !v.inMode || !v.selecting || v.line != 12 || v.fromL != 10 || strings.Contains(text, "changed under") {
		t.Fatalf("both there: in %v selecting %v line %d from %d", v.inMode, v.selecting, v.line, v.fromL)
	}
	ta.press("y")
	if copied != "LINE 10 OF THE FILE\nLINE 11 OF THE FILE\nLINE 12 OF THE FILE" {
		t.Errorf("y copied %q, not the new text", copied)
	}

	ta, v = open()
	ta.press("j", "j", "j", "j", "j", "v", "v", "k", "k", "k", "k", "k", "k", "k", "k") // from line 15 up to 7
	ta.ReadAgain(v, fileLines(12))
	ta.settle()
	text := ta.screenText()
	if !v.inMode || v.selecting || v.line != 7 || !strings.Contains(text, "notes.txt changed: where the selection started is gone") {
		t.Errorf("the start gone: in %v selecting %v line %d\n%s", v.inMode, v.selecting, v.line, text)
	}

	ta, v = open()
	ta.press("v", "v", "j", "j", "j", "j", "j") // from line 10 down to 15
	ta.ReadAgain(v, fileLines(15))              // lines 0 to 14
	ta.settle()
	text = ta.screenText()
	if v.inMode || v.selecting || !strings.Contains(text, "notes.txt changed under the cursor: Viewing left") {
		t.Errorf("the cursor gone: in %v selecting %v\n%s", v.inMode, v.selecting, text)
	}

	ta, v = open()
	ta.press("v", "$") // the cursor at the end of a line, then that line shorter
	short := fileLines(80)
	short[10] = strings.Repeat("x", 18) // the cursor was on its 19th character
	ta.ReadAgain(v, short)
	ta.settle()
	if text := ta.screenText(); v.inMode || !strings.Contains(text, "changed under the cursor") {
		t.Errorf("the cursor's column gone: in %v", v.inMode)
	}

	ta, v = open()
	ta.press("v", "j")
	empty := fileLines(80)
	empty[11] = ""
	ta.press("0")
	ta.ReadAgain(v, empty)
	if ta.screenText(); !v.inMode {
		t.Error("an empty line's column 0 is gone")
	}

	ta, v = open()
	ta.press("v", "v", "j")
	before := slices.Clone(v.Lines)
	ta.CannotReadAgain(v)
	text = ta.screenText()
	if v.inMode || v.selecting || !slices.Equal(v.Lines, before) || strings.Contains(text, "changed") {
		t.Errorf("unreadable: in %v selecting %v\n%s", v.inMode, v.selecting, text)
	}
}

// TestNote9ReadAgainClosed: on a viewer not on the screen it only puts the
// lines in, with no toast.
func TestNote9ReadAgainClosed(t *testing.T) {
	ta, v := openFile(t, fileLines(30, 3))
	ta.press("/", "needle", "Enter", "Esc")
	if ta.top() != nil {
		t.Fatalf("still open: %T", ta.topView())
	}
	if cmd := ta.ReadAgain(v, fileLines(10)); cmd != nil || len(v.Lines) != 10 || strings.Contains(ta.screenText(), "No line has") {
		t.Errorf("closed: a toast, or the lines not in: %d", len(v.Lines))
	}
	ta, v = openFile(t, fileLines(80))
	v.ScrollTo(10)
	ta.press("v", "v", "j", "j")
	ta.CloseAll()
	ta.settle()
	if cmd := ta.ReadAgain(v, fileLines(5)); cmd != nil || strings.Contains(ta.screenText(), "changed") {
		t.Error("closed in the selection mode: a Warning")
	}
}

// TestNote9PagerReadAgain: a Pager read again with Follow goes to the new
// end when its end was in sight; scrolled up, or without Follow, it stays;
// fewer lines than fill it, it is at the new end; SetLines is new content
// and goes nowhere (tdp note.9, note-90, note-92).
func TestNote9PagerReadAgain(t *testing.T) {
	p := &Pager{Follow: true}
	p.SetLines(fileLines(20))
	p.Draw(40, 10, true)
	p.Key("G", tea.KeyMsg{})
	p.Draw(40, 10, true)
	p.ReadAgain(fileLines(30))
	if p.Draw(40, 10, true); p.top != 20 {
		t.Errorf("at its end: top %d", p.top)
	}
	p.Key("k", tea.KeyMsg{})
	p.Draw(40, 10, true)
	p.ReadAgain(fileLines(40))
	if p.Draw(40, 10, true); p.top != 19 {
		t.Errorf("scrolled up: top %d", p.top)
	}
	p.ReadAgain(fileLines(5))
	if p.Draw(40, 10, true); p.top != 0 {
		t.Errorf("fewer lines: top %d", p.top)
	}
	p.SetLines(fileLines(20))
	p.Draw(40, 10, true)
	p.Key("G", tea.KeyMsg{})
	p.Draw(40, 10, true)
	p.SetLines(fileLines(30))
	if p.Draw(40, 10, true); p.top != 10 {
		t.Errorf("SetLines followed: top %d", p.top)
	}
	p.Follow = false
	p.Key("G", tea.KeyMsg{})
	p.Draw(40, 10, true)
	p.ReadAgain(fileLines(40))
	if p.Draw(40, 10, true); p.top != 20 {
		t.Errorf("no Follow, it followed: top %d", p.top)
	}
}
