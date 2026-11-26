package terminu

import (
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// goneNoteOnTop is the note on top saying a selection is gone, failing the
// test when there is none or a toast came with it.
func goneNoteOnTop(t *testing.T, ta *testApp, kind ToastKind, title string, rows ...string) *noteView {
	t.Helper()
	n, ok := ta.topView().(*noteView)
	if !ok || n.down == nil {
		t.Fatalf("no note on top: %T", ta.topView())
	}
	glyph, colour := glyphError, Red
	if kind == Warning {
		glyph, colour = glyphWarning, Peach
	}
	if n.glyph != glyph || n.color == nil || *n.color != colour || n.title != title || !slices.Equal(n.text, rows) {
		t.Errorf("the note: %q %q %v %q", n.glyph, n.title, n.color, n.text)
	}
	if text := ta.screenText(); !strings.Contains(text, title) || !strings.Contains(text, "Enter|Esc:close") {
		t.Errorf("the note as drawn:\n%s", text)
	}
	if ta.lastToast() != nil {
		t.Errorf("a toast came with the note")
	}
	return n
}

// TestNote99ViewerGoneNote: with a viewer's selection mode Space menu or ?
// open, a re-read that takes the selection away says so in a Peach note
// over them, READ AGAIN: <subject>, not in a Warning toast; Enter and Esc
// close it and them together, back to the viewer, the mode as the re-read
// left it (tdp note.9, note-99).
func TestNote99ViewerGoneNote(t *testing.T) {
	for _, opener := range []string{"Space", "?"} {
		for _, closer := range []string{"Enter", "Esc"} {
			// Where the selection started gone: back to Viewing.
			ta, v := openFile(t, fileLines(80))
			v.ScrollTo(10)
			ta.press("j", "j", "j", "j", "j", "v", "v", "k", "k", "k", "k", "k", "k", "k", "k", opener)
			if l := ta.top(); l == nil || l.viewer != v {
				t.Fatalf("%s: the selection mode's popup is not tagged", opener)
			}
			ta.ReadAgain(v, fileLines(12))
			ta.settle()
			goneNoteOnTop(t, ta, Warning, "READ AGAIN: notes.txt", "notes.txt changed: where the selection started is gone")
			if e := ta.logs.entries[len(ta.logs.entries)-1]; e.kind != Warning || !strings.Contains(e.text, "where the selection started is gone") {
				t.Errorf("not in the App Log as a Warning: %+v", e)
			}
			ta.press(closer)
			if vv, ok := ta.topView().(*viewerView); !ok || vv.v != v || !v.inMode || v.selecting {
				t.Errorf("%s then %s: back on %T, in %v selecting %v", opener, closer, ta.topView(), v.inMode, v.selecting)
			}

			// The cursor's place gone: out of the mode.
			ta, v = openFile(t, fileLines(80))
			v.ScrollTo(10)
			ta.press("v", "v", "j", "j", "j", "j", "j", opener)
			ta.ReadAgain(v, fileLines(15))
			ta.settle()
			goneNoteOnTop(t, ta, Warning, "READ AGAIN: notes.txt", "notes.txt changed under the cursor: Viewing left")
			ta.press(closer)
			if vv, ok := ta.topView().(*viewerView); !ok || vv.v != v || v.inMode {
				t.Errorf("%s then %s, cursor gone: back on %T, in %v", opener, closer, ta.topView(), v.inMode)
			}
		}
	}

	// Nothing taken away: the menu stays as it is, no note.
	ta, v := openFile(t, fileLines(80))
	v.ScrollTo(10)
	ta.press("v", "v", "j", "Space")
	menu := ta.topView()
	ta.ReadAgain(v, fileLines(90))
	if ta.settle(); ta.topView() != menu || ta.lastToast() != nil {
		t.Errorf("nothing gone: %T on top", ta.topView())
	}
}

// TestNote100ViewerCannotRead: content that cannot be read, with the
// selection mode's Space menu or ? open, is an error popup over them,
// CANNOT READ: <subject>, the app's words and "The selection is gone", in
// place of the Error toast; Enter and Esc close it and them, back to the
// viewer out of the mode. With neither open it is the app's Error toast;
// with no words, nothing (tdp note.9, note-100).
func TestNote100ViewerCannotRead(t *testing.T) {
	for _, opener := range []string{"Space", "?"} {
		for _, closer := range []string{"Enter", "Esc"} {
			ta, v := openFile(t, fileLines(80))
			ta.press("v", "v", "j", opener)
			before := slices.Clone(v.Lines)
			ta.CannotReadAgain(v, "Cannot read notes.txt: it is gone")
			ta.settle()
			goneNoteOnTop(t, ta, Error, "CANNOT READ: notes.txt", "Cannot read notes.txt: it is gone", "The selection is gone")
			if v.inMode || !slices.Equal(v.Lines, before) {
				t.Errorf("%s: in %v, lines changed", opener, v.inMode)
			}
			ta.press(closer)
			if vv, ok := ta.topView().(*viewerView); !ok || vv.v != v {
				t.Errorf("%s then %s: back on %T", opener, closer, ta.topView())
			}
		}
	}

	ta, v := openFile(t, fileLines(80))
	ta.press("v", "v", "j")
	ta.CannotReadAgain(v, "Cannot read notes.txt: it is gone")
	ta.settle()
	if vv, ok := ta.topView().(*viewerView); !ok || vv.v != v || v.inMode || !strings.Contains(ta.screenText(), "Cannot read notes.txt: it is gone") {
		t.Errorf("no menu: not the Error toast, %T on top", ta.topView())
	}
	ta, v = openFile(t, fileLines(80))
	ta.press("v")
	if cmd := ta.CannotReadAgain(v); cmd != nil || v.inMode || ta.lastToast() != nil {
		t.Error("no words: something said")
	}
	ta.press("Esc")
	if cmd := ta.CannotReadAgain(v, "gone"); cmd != nil || ta.lastToast() != nil {
		t.Error("closed: something said")
	}
}

// TestNote99SelectionGone: an app's own selection mode on its own reader, a
// panel's mode: with its Space menu or ? open, SelectionGone is a note over
// them in kind's colour, Peach or Red, and leaving the mode then keeps them
// open under it; Enter and Esc close the note and them, back to the screen.
// With neither open, it is a toast of kind (tdp note.9, note-99,
// note-100).
func TestNote99SelectionGone(t *testing.T) {
	visual := func(ta *testApp) *Mode {
		m := &Mode{Name: "Viewing", Exit: "leave", Keys: []ModeKey{{Key: "y", Desc: "copy", Do: func(string) tea.Cmd { return nil }}}}
		ta.pods.EnterMode(m)
		return m
	}
	for _, c := range []struct {
		opener, closer string
		kind           ToastKind
		leave          bool
	}{
		{"Space", "Enter", Warning, true}, {"Space", "Esc", Error, false},
		{"?", "Esc", Warning, true}, {"?", "Enter", Error, true},
	} {
		ta := newTestApp(t)
		visual(ta)
		ta.press(c.opener)
		under := ta.top()
		if under == nil || under.mode != ta.pods {
			t.Fatalf("%s: the mode's popup is not on top", c.opener)
		}
		ta.SelectionGone(ta.pods, c.kind, "Read again", "README.md changed: Viewing left")
		ta.settle()
		goneNoteOnTop(t, ta, c.kind, "READ AGAIN", "README.md changed: Viewing left")
		if c.leave {
			ta.pods.LeaveMode()
			ta.settle()
			if under.closing || ta.pods.InMode() != nil {
				t.Errorf("%s, left: the popup under the note closed with the mode", c.opener)
			}
		}
		ta.press(c.closer)
		if ta.top() != nil {
			t.Errorf("%s %s: %T still open", c.opener, c.closer, ta.topView())
		}
	}

	ta := newTestApp(t)
	visual(ta)
	ta.SelectionGone(ta.pods, Warning, "Read again", "README.md changed: where the selection started is gone")
	ta.settle()
	if ta.top() != nil || !strings.Contains(ta.screenText(), "README.md changed: where the selection started is gone") {
		t.Errorf("no menu: not a toast, %T on top", ta.topView())
	}
	ta.press("Space")
	ta.pods.LeaveMode()
	ta.settle()
	if ta.top() != nil {
		t.Errorf("left with no note: the mode's popup stayed, %T", ta.topView())
	}
}
