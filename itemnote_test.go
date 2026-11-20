package terminu

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// noteApp is the test app, 80 × 20, its pods with a note on every pod but
// pod-00: two lines, the second long.
func noteApp(t *testing.T) *testApp {
	t.Helper()
	ta := newTestApp(t)
	ta.resize(80, 20)
	ta.pods.Note = func() []string {
		r, ok := ta.list.Current()
		if !ok || r.ID == "0" {
			return nil
		}
		return []string{"go.dev/doc/effective_go for " + r.Cells[0], strings.Repeat("a long remark that wraps ", 6)}
	}
	return ta
}

// row is the screen's row y as text.
func row(ta *testApp, y int) string { return StripStyles(ta.Frame()[y]) }

// headRow is the screen row the note's head is on, -1 for none.
func headRow(ta *testApp) int {
	for y, l := range ta.Frame() {
		if strings.Contains(StripStyles(l), glyphNote+" i ") {
			return y
		}
	}
	return -1
}

// TestNoteHead: with the cursor on an item with a note, the note's head
// shows right under it, over the row below: a small box with U+F02FD and i;
// it takes no key and goes when the cursor leaves, or while a popup is open
// (tdp panel.18, panel-192).
func TestNoteHead(t *testing.T) {
	ta := noteApp(t)
	if headRow(ta) >= 0 {
		t.Fatal("a head on an item with no note")
	}
	ta.press("j") // pod-01
	y := headRow(ta)
	item := -1
	for i, l := range ta.Frame() {
		if strings.Contains(StripStyles(l), "pod-01") {
			item = i
		}
	}
	if y < 0 || y != item+1 {
		t.Fatalf("the head on row %d, the item on %d", y, item)
	}
	if got := row(ta, y); !strings.Contains(got, "┌ "+glyphNote+" i ┐") || strings.Contains(got, "pod-02") {
		t.Errorf("the head does not lie over the row below: %q", got)
	}
	// The box and i Blue, as a key that can be pressed now; the glyph Text.
	if styled := ta.Frame()[y]; !strings.Contains(styled, Fg(Blue).Render("┌ ")) || !strings.Contains(styled, Fg(Text).Render(glyphNote)) ||
		!strings.Contains(styled, Fg(Blue).Render(" i ┐")) {
		t.Errorf("the head's colours: %q", styled)
	}
	ta.press("j") // it takes no key: j moves the cursor
	if r, _ := ta.list.Current(); r.ID != "2" || headRow(ta) != y+1 {
		t.Errorf("j did not move the cursor past it: %s, head on %d", r.ID, headRow(ta))
	}
	ta.press("gg")
	if headRow(ta) >= 0 {
		t.Error("the head stays when the cursor leaves")
	}
	ta.press("j", "Space")
	if headRow(ta) >= 0 {
		t.Error("the head shows over the Space menu")
	}
	ta.press("Esc", "Tab")
	if headRow(ta) >= 0 {
		t.Error("the head shows on a panel without the focus")
	}
}

// TestNoteOpen: i opens the note under its item, a viewer titled NOTE with
// the focus in it: it scrolls, has the selection mode and / search, dims
// what is under it; at most half the panel high, long lines wrapped; i is
// the item action [i] Note, off on an item with none (tdp panel.18,
// panel-193, K13-4).
func TestNoteOpen(t *testing.T) {
	ta := noteApp(t)
	ta.press("j")
	head := headRow(ta)
	ta.press("i")
	vv, ok := ta.topView().(*viewerView)
	if !ok {
		t.Fatalf("i opened %T", ta.topView())
	}
	l := ta.top()
	if l.y != head || !strings.Contains(row(ta, head), glyphNote+" NOTE") {
		t.Errorf("not opened from its head: y %d, head %d: %q", l.y, head, row(ta, head))
	}
	if l.h > 18/2 {
		t.Errorf("%d rows, more than half the panel", l.h)
	}
	if text := ta.screenText(); !strings.Contains(text, "a long remark") || !strings.Contains(text, "go.dev/doc/") {
		t.Errorf("the note's lines:\n%s", text)
	}
	if vv.v.Select == nil || vv.v.Search == nil || vv.v.NoNumbers != true {
		t.Error("not a full viewer: selection, search")
	}
	ta.press("/")
	if ta.topView() == popupView(vv) || len(ta.stack) != 2 {
		t.Errorf("/ did not open the search over it: %T", ta.topView())
	}
	ta.press("Esc")
	ta.press("v")
	if !vv.v.inMode {
		t.Error("v does not start the selection mode")
	}
	ta.press("Esc")
	under := ta.Frame()[3] // the pods' header, under the popup's dimming
	if !strings.Contains(under, covBRGB(DimColor(Blue))) {
		t.Errorf("what is under it is not dimmed: %q", under)
	}

	// [i] Note in the Space menu, off on an item with none.
	ta.press("Esc", "gg", "Space")
	if text := ta.screenText(); !strings.Contains(text, "[i] Note") {
		t.Errorf("no [i] Note in the Space menu:\n%s", text)
	}
	off := false // drawn off on an item with no note (tdp M6)
	for _, l := range ta.Frame() {
		if strings.Contains(StripStyles(l), "[i] Note") && strings.Contains(l, covBRGB(Surface2)) {
			off = true
		}
	}
	if !off {
		t.Error("[i] Note is not off on an item with no note")
	}
	ta.press("Esc", "i")
	if ta.top() != nil {
		t.Error("i opened a note on an item with none")
	}
}

// TestNoteLogged: opening a note only looks, and goes in the App Log as
// opening a viewer does, naming the item: a list's row by its first cell,
// else the note's first line; the head coming and going with the cursor
// does not (tdp panel.18, note.7).
func TestNoteLogged(t *testing.T) {
	ta := noteApp(t)
	before := len(ta.logs.entries)
	ta.press("j", "j", "k")
	if n := len(ta.logs.entries) - before; n != 0 {
		t.Errorf("the head moving with the cursor logged %d entries", n)
	}
	ta.press("i")
	got := ta.logs.entries[before:]
	if len(got) != 1 || got[0].kind != Info || got[0].text != "Opened the note of pod-01" {
		t.Errorf("opening the note logged %+v", got)
	}

	// A body of the app's own: the note's first line names it.
	ta = noteApp(t)
	ta.pods.Note = nil
	mode := &Mode{Name: "Focus", Note: func() []string { return []string{"go.dev/doc", "the effective Go page"} }}
	ta.pods.EnterMode(mode)
	ta.list.SetRows([]Row{{ID: "x", Cells: []string{""}}})
	ta.Update(nil)
	before = len(ta.logs.entries)
	ta.press("i")
	if got := ta.logs.entries[before:]; len(got) != 1 || got[0].text != "Opened the note of go.dev/doc" {
		t.Errorf("a row with no name: %+v", got)
	}
}

// TestNoteClose: Esc closes the note back to its head, the focus on the
// list, the cursor where it was (tdp panel.18, panel-194).
func TestNoteClose(t *testing.T) {
	ta := noteApp(t)
	ta.press("j", "j")
	head := headRow(ta)
	ta.press("i", "Esc")
	if ta.top() != nil {
		t.Fatal("Esc did not close it")
	}
	if r, _ := ta.list.Current(); r.ID != "2" || ta.Screen().Focused() != ta.pods || headRow(ta) != head {
		t.Errorf("not back to its head: cursor %s, head %d", r.ID, headRow(ta))
	}
}

// TestNoteUp: with no room below the item, the head and the note go above
// it, the head at the bottom against the item (tdp panel.18, panel-195).
func TestNoteUp(t *testing.T) {
	ta := noteApp(t)
	ta.press("G")
	item := -1
	for i, l := range ta.Frame() {
		if strings.Contains(StripStyles(l), "pod-39") {
			item = i
		}
	}
	if y := headRow(ta); y != item-1 || !strings.Contains(row(ta, y), "└ "+glyphNote+" i ┘") {
		t.Fatalf("the head on row %d, the item on %d: %q", y, item, row(ta, max(y, 0)))
	}
	ta.press("i")
	l := ta.top()
	if l == nil || l.y+l.h != item {
		t.Fatalf("the note does not end against the item: %+v, item %d", l, item)
	}
	if l.unfold != -1 {
		t.Error("it does not unfold upward")
	}
	// Opening, it unfolds from its head: up, the rows nearest the item first.
	full := make([]string, l.h)
	for i := range full {
		full[i] = itoa(i)
	}
	l.frame = 3
	rows, first, _ := l.drawFrame(full, Blue)
	if len(rows) == 0 || len(rows) >= l.h || first != l.h-len(rows) || rows[len(rows)-1] != itoa(l.h-1) {
		t.Errorf("frame 3 upward: rows %q from %d of %d", rows, first, l.h)
	}
	l.unfold = 1
	if rows, first, _ = l.drawFrame(full, Blue); first != 0 || rows[0] != "0" || len(rows) >= l.h {
		t.Errorf("frame 3 downward: rows %q from %d", rows, first)
	}
	l.unfold, l.frame = -1, openFrames
	if r, _ := ta.list.Current(); r.ID != "39" {
		t.Error("the cursor moved")
	}
}

// TestK13NoteKey: i is the family's key for an item's note: a panel with
// notes takes no action of the app's on i (AddScreen panics); in a mode with
// a Note, i is the mode's key for it, in its Space menu (tdp K13, K13-4).
func TestK13NoteKey(t *testing.T) {
	func() {
		defer func() {
			if recover() == nil {
				t.Error("an action on i was taken on a panel with notes")
			}
		}()
		p := &Panel{Label: "L", Body: &List{}, Note: func() []string { return nil },
			Actions: []Action{{Key: "i", Label: "Import", Do: func() tea.Cmd { return nil }}}}
		New("x").AddScreen(&Screen{Name: "S", Panels: []*Panel{p}})
	}()

	ta := noteApp(t)
	ta.pods.Note = nil
	mode := &Mode{Name: "Focus", Note: func() []string { return []string{"where the link goes"} }}
	ta.pods.EnterMode(mode)
	ta.Update(nil)
	if headRow(ta) < 0 {
		t.Fatal("no head in the mode")
	}
	ta.press("Space") // a mode of the app's own lists its keys as they are
	if text := ta.screenText(); !strings.Contains(text, "FOCUS MODE") || !strings.Contains(text, "i                note") {
		t.Errorf("no i in the mode's Space menu:\n%s", text)
	}
	ta.press("Esc", "i")
	if _, ok := ta.topView().(*viewerView); !ok || !strings.Contains(ta.screenText(), "where the link") {
		t.Errorf("i in the mode did not open its note:\n%s", ta.screenText())
	}
	n := len(mode.Keys)
	ta.pods.LeaveMode()
	ta.pods.EnterMode(mode)
	if len(mode.Keys) != n {
		t.Error("the note key went in twice")
	}
}
