package terminu

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vulcanshen/terminu-go/internal/testhook"
)

// inspected is the text of the inspect toast on the screen, nil when there
// is none or it is going.
func inspected(ta *testApp) []string {
	if t := ta.toast; t != nil && !t.closing && t.view.(*toastView).inspect {
		return t.view.(*toastView).text
	}
	return nil
}

// TestToastInspect: the inspect toast is there while the cursor is on an
// item that has one, titled INSPECT with no hint, every key going through
// it, Esc too; none while a popup is open, and none in the App Log (tdp
// toast.4, toast-21 to toast-23, toast-26, toast-27).
func TestToastInspect(t *testing.T) {
	ta := newTestApp(t)
	ta.pods.Inspect = func() []string {
		r, ok := ta.list.Current()
		if !ok || r.ID == "1" {
			return nil // pod-01 has nothing to show
		}
		return []string{"go.dev/" + r.Cells[0]}
	}
	logged := len(ta.logs.entries)
	ta.Update(nil)
	ta.settle()
	if got := inspected(ta); len(got) != 1 || got[0] != "go.dev/pod-00" {
		t.Fatalf("on pod-00: %q", got)
	}
	_, plain := covEBox(ta, ta.toast)
	if !strings.Contains(plain[0], glyphInspect+" INSPECT") || strings.Contains(plain[len(plain)-1], "Esc") {
		t.Errorf("titled INSPECT, with no hint:\n%s", strings.Join(plain, "\n"))
	}
	if testhook.Toast(ta.App) || len(ta.logs.entries) != logged {
		t.Error("Esc does not take it, and the App Log does not keep it (toast-27)")
	}
	ta.press("j")
	if ta.toast != nil {
		t.Error("it stays on pod-01, which has none")
	}
	ta.press("j")
	ta.Update(keyMsg("j"))
	if got := inspected(ta); len(got) != 1 || got[0] != "go.dev/pod-03" || ta.toast.frame != openFrames {
		t.Fatalf("from pod-02 to pod-03 it is pod-03's at once: %q", got)
	}
	ta.settle()

	ta.press("k", "k") // pod-01
	focus := &Mode{Name: "Focus", Inspect: func() []string { return []string{"go.dev/doc"} }}
	ta.pods.EnterMode(focus)
	ta.pods.EnterLayer(&Mode{Name: "select"})
	ta.Update(nil)
	ta.settle()
	if got := inspected(ta); len(got) != 1 || got[0] != "go.dev/doc" {
		t.Errorf("in a layer of a mode with one, the mode's: %q", got)
	}
	ta.press("Esc")
	if ta.pods.InMode() != focus || inspected(ta) == nil {
		t.Errorf("Esc goes through it and leaves the layer: in %v, toast %q", ta.pods.InMode(), inspected(ta))
	}
	ta.press("Esc")
	if ta.pods.InMode() != nil || ta.toast != nil {
		t.Error("Esc leaves the mode, and the toast goes with it, pod-01 having none")
	}

	// None while a popup is open, back once it closes (toast-26).
	ta.press("k", "Space")
	if ta.toast != nil {
		t.Error("it is there under the Space menu")
	}
	ta.press("Esc")
	if inspected(ta) == nil {
		t.Error("it is not back once the Space menu is gone")
	}
}

// TestToastInspectReplaced: an ordinary toast takes the inspect toast's
// place at once, and once that one is gone it is back (tdp toast.4,
// toast-24).
func TestToastInspectReplaced(t *testing.T) {
	ta := newTestApp(t)
	ta.pods.Inspect = func() []string { return []string{"go.dev/doc"} }
	ta.Update(nil)
	ta.settle()
	ta.Toast(Success, "Copied")
	ta.Update(nil)
	if inspected(ta) != nil || ta.toast.frame != openFrames {
		t.Fatalf("the ordinary toast is not in its place at once: %q, frame %d", inspected(ta), ta.toast.frame)
	}
	ta.Update(toastDone{ta.toastSeq})
	if inspected(ta) == nil || ta.toast.frame != openFrames {
		t.Error("it is not back at once when the ordinary toast's time is up")
	}
	ta.Toast(Info, "Saved")
	ta.press("Esc")
	if inspected(ta) == nil {
		t.Error("Esc takes the ordinary toast, and it is back")
	}
	ta.press("Esc")
	if inspected(ta) == nil {
		t.Error("Esc goes through it")
	}
}

// uncovered is a body of the app's own keeping what Uncover told it.
type uncovered struct{ rows int }

func (b *uncovered) Draw(w, h int, focused bool) []string   { return blankLines(w, h) }
func (b *uncovered) Key(string, tea.KeyMsg) (bool, tea.Cmd) { return false, nil }
func (b *uncovered) HasItem() bool                          { return true }
func (b *uncovered) Scroll() string                         { return "" }
func (b *uncovered) MoveKeys() []KeyRow                     { return nil }
func (b *uncovered) Uncover(rows int)                       { b.rows = rows }

// TestToastInspectUncovers: the item under the cursor shows above the
// inspect toast, a list and a panel of values scrolled, a body of the app's
// own told the rows the toast covers (tdp toast.4, toast-25).
func TestToastInspectUncovers(t *testing.T) {
	list := &List{Columns: []Column{{Title: "Name"}}}
	var rows []Row
	values := &Values{}
	for i := range 40 {
		rows = append(rows, Row{ID: fmt.Sprint(i), Cells: []string{fmt.Sprintf("pod-%02d", i)}})
		values.Fields = append(values.Fields, &Field{Label: fmt.Sprintf("F%02d", i), Value: func() string { return "x" }})
	}
	list.SetRows(rows)
	there := func() []string { return []string{"go.dev/doc"} }
	for _, c := range []struct {
		body Body
		last string
	}{{list, "pod-39"}, {values, "F39"}} {
		ta := &testApp{App: anySize(New("test"))}
		ta.AddScreen(&Screen{Name: "One", Key: "O", Panels: []*Panel{{Label: "One", Body: c.body, Inspect: there}}})
		ta.resize(60, 20)
		ta.press("G")
		at := slices.IndexFunc(strings.Split(ta.screenText(), "\n"), func(s string) bool { return strings.Contains(s, c.last) })
		if ta.toast == nil || at < 0 || at >= ta.toast.y {
			t.Errorf("%s on row %d, under the inspect toast:\n%s", c.last, at, ta.screenText())
		}
	}

	a, b := &uncovered{}, &uncovered{}
	show := true
	ta := &testApp{App: anySize(New("test"))}
	ta.AddScreen(&Screen{Name: "Two", Key: "T", Panels: []*Panel{
		{Label: "A", Body: a, Inspect: func() []string {
			if show {
				return []string{"go.dev/doc"}
			}
			return nil
		}},
		{Label: "B", Body: b, Inspect: there},
	}})
	ta.resize(60, 20)
	ta.settle()
	ta.Frame()
	if a.rows != ta.toast.h || b.rows != 0 {
		t.Errorf("the focused body is told the %d rows the toast covers, the other none: %d, %d", ta.toast.h, a.rows, b.rows)
	}
	show = false
	ta.Update(nil)
	ta.settle()
	ta.Frame()
	if a.rows != 0 {
		t.Errorf("with the toast gone it covers %d rows", a.rows)
	}
}
