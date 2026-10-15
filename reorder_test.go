package terminu

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func testReorder(done *[]int) Reorder {
	return Reorder{Title: "Order", Options: []string{"Usage", "Cost", "Model", "Context", "Git"},
		Done: func(o []int) tea.Cmd { *done = o; return nil }}
}

// The rows keep a cell for the glyph; the width is the widest hint's, 35
// (components/input/reorder).
func TestReorderLayout(t *testing.T) {
	ta := newTestApp(t)
	var got []int
	ta.Reorder(testReorder(&got))
	ta.settle()
	lines, layer := covFDraw(ta)
	text := covFText(lines)
	if ta.top().w != 35 || len(lines) != 9 {
		t.Fatalf("%d×%d, want 35×9:\n%s", ta.top().w, len(lines), strings.Join(text, "\n"))
	}
	if !strings.HasPrefix(text[0], "┏━") || !strings.Contains(text[0], "ORDER") {
		t.Errorf("title %q", text[0])
	}
	for i, item := range []string{"Usage", "Cost", "Model", "Context", "Git"} {
		if !strings.HasPrefix(text[2+i], "┃   "+item+" ") {
			t.Errorf("row %d: %q, want the item after the glyph's cell", i, text[2+i])
		}
	}
	// It opens on the first item: the cursor in the layer colour.
	if !strings.Contains(lines[2], On(layer).Render(" "+Pad("  Usage", 31))) {
		t.Errorf("the cursor %q", lines[2])
	}
	if h := covFHint(ta); !strings.Contains(h, " Enter:apply D:drag Esc:cancel ") {
		t.Errorf("hint %q", h)
	}
}

// The example of components/input/reorder: Cost picked up, moved up, dropped,
// then the order applied.
func TestReorderDrag(t *testing.T) {
	ta := newTestApp(t)
	var got []int
	ta.Reorder(testReorder(&got))
	ta.settle()
	rv := ta.topView().(*reorderView)
	ta.press("j", "D")
	if !rv.held || !rv.inMode() {
		t.Fatal("D does not pick up the item")
	}
	lines, _ := covFDraw(ta)
	text := covFText(lines)
	if !strings.Contains(text[0], "┫Drag┣") || !strings.Contains(lines[0], Fg(Yellow).Render("┏")) {
		t.Errorf("the Drag mode on the frame, in Yellow: %q", text[0])
	}
	if !strings.Contains(lines[3], On(Yellow).Render(" "+Pad(Pad(glyphSwap, 2)+"Cost", 31))) || !strings.Contains(lines[3], Fg(Yellow).Render(glyphHard)) {
		t.Errorf("the held item in Yellow with the swap glyph: %q", text[3])
	}
	if h := covFHint(ta); !strings.Contains(h, " Enter:drop Esc:cancel ") {
		t.Errorf("hint while holding %q", h)
	}
	ta.press("k")
	if !slices.Equal(rv.order, []int{1, 0, 2, 3, 4}) || rv.cursor != 0 {
		t.Errorf("k swaps it with the item above: %v, cursor %d", rv.order, rv.cursor)
	}
	ta.press("Enter")
	if rv.held || ta.top() == nil || got != nil {
		t.Fatal("Enter while holding drops it, nothing more")
	}
	ta.press("Enter")
	if !slices.Equal(got, []int{1, 0, 2, 3, 4}) || ta.top() != nil {
		t.Errorf("Enter applies the order: %v", got)
	}
}

// The held item wraps at the ends; the others move by one (Rules K12).
func TestReorderDragWraps(t *testing.T) {
	ta := newTestApp(t)
	var got []int
	ta.Reorder(testReorder(&got))
	ta.settle()
	rv := ta.topView().(*reorderView)
	ta.press("G", "D", "j")
	if !slices.Equal(rv.order, []int{4, 0, 1, 2, 3}) || rv.cursor != 0 {
		t.Errorf("the last moved down goes first: %v, cursor %d", rv.order, rv.cursor)
	}
	ta.press("k")
	if !slices.Equal(rv.order, []int{0, 1, 2, 3, 4}) || rv.cursor != 4 {
		t.Errorf("the first moved up goes last: %v, cursor %d", rv.order, rv.cursor)
	}
	// Not holding, j|k move the cursor and wrap too.
	ta.press("Enter", "j")
	if rv.cursor != 0 {
		t.Errorf("j on the last item: cursor %d", rv.cursor)
	}
}

// Esc goes back one layer at a time: the held item back, then the popup,
// asking first once the order changed (components/input/reorder, Rules K4).
func TestReorderEsc(t *testing.T) {
	ta := newTestApp(t)
	var got []int
	ta.Reorder(testReorder(&got))
	ta.settle()
	rv := ta.topView().(*reorderView)
	ta.press("D", "j", "j", "Esc")
	if rv.held || !slices.Equal(rv.order, []int{0, 1, 2, 3, 4}) || rv.cursor != 0 || ta.top() == nil {
		t.Fatalf("Esc while holding puts it back: held %v, %v, cursor %d", rv.held, rv.order, rv.cursor)
	}
	ta.press("Esc")
	if ta.top() != nil {
		t.Fatal("Esc with nothing changed closes at once")
	}
	// Dropped somewhere else, then moved back: still changed.
	ta.Reorder(testReorder(&got))
	ta.settle()
	rv = ta.topView().(*reorderView)
	ta.press("D", "j", "Enter", "D", "k", "Enter")
	if !slices.Equal(rv.order, []int{0, 1, 2, 3, 4}) || !rv.changed {
		t.Fatalf("moved and moved back: %v, changed %v", rv.order, rv.changed)
	}
	ta.press("Esc")
	if !strings.Contains(ta.screenText(), "Discard changes to Order?") {
		t.Fatalf("Esc after a change asks first:\n%s", ta.screenText())
	}
	ta.press("Esc")
	if ta.topView() != rv {
		t.Fatal("Esc on the question goes back to the list")
	}
	// Back in its first order, Enter writes nothing.
	ta.press("Enter")
	if got != nil || ta.top() != nil {
		t.Errorf("Enter with the order as it opened wrote %v", got)
	}
	ta.Reorder(testReorder(&got))
	ta.settle()
	ta.press("D", "j", "Enter", "Esc", "Enter")
	if got != nil || ta.top() != nil {
		t.Errorf("discarding wrote %v, left %T", got, ta.topView())
	}
}

// In the Drag mode only its own keys and the core keys of Rules K11 work.
func TestReorderDragModeKeys(t *testing.T) {
	ta := newTestApp(t)
	ta.CoreKeys = []CoreKey{{Key: "!", Desc: "the app log", Do: ta.record("log")}}
	var got []int
	ta.Reorder(testReorder(&got))
	ta.settle()
	rv := ta.topView().(*reorderView)
	ta.press("j", "D")
	ta.press("D", "Space", "G", "gg", "d", "!", "Tab")
	if !rv.held || rv.cursor != 1 || len(ta.ran) != 0 || len(ta.stack) != 1 {
		t.Errorf("other keys did something: held %v, cursor %d, ran %v, %d layers", rv.held, rv.cursor, ta.ran, len(ta.stack))
	}
	ta.press("?")
	if n, ok := ta.topView().(*noteView); !ok || n.title != "DRAG KEYS" {
		t.Fatalf("? while holding is the Drag mode's keys: %T", ta.topView())
	}
	text := ta.screenText()
	if !strings.Contains(text, "move it down, up") || strings.Contains(text, "apply the order") {
		t.Errorf("the mode's key reference:\n%s", text)
	}
	ta.press("Esc", "Enter", "!")
	if len(ta.ran) != 1 {
		t.Errorf("the app's core key once dropped: %v", ta.ran)
	}
	ta.press("q")
	if !covAQuitConfirm(ta) {
		t.Error("q does not start the quit flow")
	}
}

// Space does nothing in a reorder (Rules K5): D picks up.
func TestReorderSpace(t *testing.T) {
	ta := newTestApp(t)
	var got []int
	ta.Reorder(testReorder(&got))
	ta.settle()
	ta.press("Space")
	if rv := ta.topView().(*reorderView); rv.held || len(ta.stack) != 1 {
		t.Error("Space did something")
	}
}

// Many items scroll, N of M on the bottom border.
func TestReorderScroll(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(80, 14)
	opts := make([]string, 30)
	for i := range opts {
		opts[i] = fmt.Sprintf("item-%02d", i)
	}
	ta.Reorder(Reorder{Title: "Order", Options: opts})
	ta.settle()
	ta.press("G")
	text := ta.screenText()
	if !strings.Contains(text, "item-29") || !strings.Contains(text, "30 of 30") {
		t.Errorf("G scrolls to the last item:\n%s", text)
	}
}

// The panel's Drag mode: the held row as the cursor in Yellow, the swap
// glyph in the mark column, wrapping at the ends (layout/panel).
func TestPanelDragHeldRow(t *testing.T) {
	ta := newTestApp(t)
	l := ta.list
	l.Draggable = true
	l.ToggleMark() // pod-00 marked
	if !l.Hold() || !l.Holding() {
		t.Fatal("Hold on a row")
	}
	lines := l.Draw(60, 10, true)
	held := strings.TrimSuffix(StripStyles(lines[1]), glyphHard)
	if !strings.HasPrefix(held, " "+Pad(glyphSwap, IconWidth()+1)+"pod-00") || lines[1] != On(Yellow).Render(held)+Fg(Yellow).Render(glyphHard) {
		t.Errorf("the held row: %q", lines[1])
	}
	l.MoveHeld(false)
	if r, _ := l.Current(); r.ID != "0" || l.rows[39].ID != "0" || l.rows[0].ID != "1" {
		t.Errorf("the first moved up goes last: cursor on %s, rows %s…%s", r.ID, l.rows[0].ID, l.rows[39].ID)
	}
	l.MoveHeld(true)
	if l.rows[0].ID != "0" || l.rows[1].ID != "1" || l.cursor != 0 {
		t.Errorf("the last moved down goes first: %s %s, cursor %d", l.rows[0].ID, l.rows[1].ID, l.cursor)
	}
	l.MoveHeld(true)
	if l.rows[0].ID != "1" || l.rows[1].ID != "0" || l.cursor != 1 {
		t.Errorf("j swaps it with the next: %s %s", l.rows[0].ID, l.rows[1].ID)
	}
	l.Drop()
	lines = l.Draw(60, 10, true)
	if !strings.Contains(lines[2], glyphMarked) || strings.Contains(lines[2], glyphSwap) {
		t.Errorf("dropped, its mark is back: %q", StripStyles(lines[2]))
	}
	// A list with a Drag keeps the mark column without marks too.
	plain := &List{Draggable: true}
	plain.SetRows([]Row{{Cells: []string{"one"}}, {Cells: []string{"two"}}})
	if got := StripStyles(plain.Draw(20, 2, true)[1]); !strings.HasPrefix(got, " "+Repeat(" ", IconWidth()+1)+"two") {
		t.Errorf("no mark column kept: %q", got)
	}
}

// The viewer's selection mode does not take the app's own core keys either
// (Rules K11).
func TestK11PopupModeCoreKeys(t *testing.T) {
	ta := newTestApp(t)
	ta.CoreKeys = []CoreKey{{Key: "!", Desc: "the app log", Do: ta.record("log")}}
	v := testViewer()
	v.Select = &Selection{Key: "v", Name: "Visual"}
	ta.Viewer(v)
	ta.settle()
	ta.press("!")
	if len(ta.ran) != 1 {
		t.Fatalf("! on a viewer: %v", ta.ran)
	}
	ta.press(v.Select.Key, "!")
	if len(ta.ran) != 1 {
		t.Errorf("! in the selection mode ran %v", ta.ran)
	}
}
