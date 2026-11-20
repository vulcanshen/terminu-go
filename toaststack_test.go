package terminu

import (
	"slices"
	"strings"
	"testing"
)

// shown is the toasts showing, oldest first: their first row and y.
func shown(ta *testApp) (texts []string, ys []int) {
	for _, t := range ta.showingToasts() {
		texts = append(texts, t.view.(*toastView).text[0])
		ys = append(ys, t.y)
	}
	return texts, ys
}

// TestToastStack: a new toast comes at the bottom, its bottom border on the
// third row from the bottom, pushing the ones showing up; each keeps its own
// time, leaving when it is up, the others closing the gap; at most three
// show, a fourth pushing the oldest out, its words in the App Log (tdp
// toast.2, toast-13).
func TestToastStack(t *testing.T) {
	ta := newTestApp(t)
	ta.Toast(Info, "One")
	ta.settle()
	one := ta.showingToasts()[0]
	bottom := one.y
	if one.y+one.h-1 != ta.h-3 {
		t.Errorf("the bottom border on row %d, not the third from the bottom", one.y+one.h-1)
	}
	ta.Toast(Warning, "Two")
	ta.settle()
	texts, ys := shown(ta)
	if !slices.Equal(texts, []string{"One", "Two"}) || ys[1] != bottom || ys[0] != ys[1]-one.h {
		t.Errorf("not stacked, the new at the bottom: %v at %v", texts, ys)
	}
	if text := ta.screenText(); strings.Index(text, "One") > strings.Index(text, "Two") {
		t.Error("the older is not above")
	}

	// Each its own time: One's goes, Two moves down into its place.
	ta.Update(toastDone{ta.showingToasts()[0].view.(*toastView).id})
	ta.settle()
	texts, ys = shown(ta)
	if !slices.Equal(texts, []string{"Two"}) || ys[0] != ta.h-2-ta.showingToasts()[0].h {
		t.Errorf("after One's time: %v at %v", texts, ys)
	}

	// At most three: a fourth pushes the oldest out.
	ta.Toast(Info, "Three")
	ta.Toast(Info, "Four")
	ta.Toast(Info, "Five")
	ta.settle()
	if texts, _ = shown(ta); !slices.Equal(texts, []string{"Three", "Four", "Five"}) {
		t.Errorf("not the three newest: %v", texts)
	}
	if !strings.Contains(strings.Join(ta.logs.lines(), "\n"), "Two") {
		t.Error("the one pushed out is not in the App Log")
	}
	if strings.Contains(ta.screenText(), "Two") {
		t.Error("the one pushed out still shows")
	}
	// A screen too short for them all: the ones past the top are left out.
	ta.resize(60, 12)
	ta.Frame()
}

// TestToastEscOldest: Esc closes one toast a press, the oldest first, before
// any other popup; Esc:close is drawn on the oldest only (tdp toast.3,
// toast-14).
func TestToastEscOldest(t *testing.T) {
	ta := newTestApp(t)
	ta.press("Space")
	ta.Toast(Info, "One")
	ta.Toast(Info, "Two")
	ta.Toast(Info, "Three")
	ta.settle()
	if n := strings.Count(ta.screenText(), "Esc:close"); n != 1 {
		t.Errorf("Esc:close on %d toasts", n)
	}
	hinted := func() string {
		for _, t := range ta.showingToasts() {
			if v := t.view.(*toastView); len(v.hint()) > 0 {
				return v.text[0]
			}
		}
		return ""
	}
	if h := hinted(); h != "One" {
		t.Errorf("the hint on %q, not the oldest", h)
	}
	for _, want := range [][]string{{"Two", "Three"}, {"Three"}, nil} {
		ta.press("Esc")
		if texts, _ := shown(ta); !slices.Equal(texts, want) {
			t.Errorf("after Esc: %v, want %v", texts, want)
		}
		if len(want) > 0 && hinted() != want[0] {
			t.Errorf("the hint on %q, not %q", hinted(), want[0])
		}
		if ta.top() == nil {
			t.Fatal("Esc closed the Space menu before the toasts")
		}
	}
	ta.press("Esc")
	if ta.top() != nil {
		t.Error("the Space menu is still open")
	}
}

// TestToastRepeat: the same words as a toast showing, of the same kind, make
// no new toast: that one's time starts over and its title counts them
// (INFO ×3); the App Log gets an entry each time (tdp toast.2, toast-29).
func TestToastRepeat(t *testing.T) {
	ta := newTestApp(t)
	before := len(ta.logs.entries)
	ta.Toast(Info, "Copied")
	ta.settle()
	first := ta.showingToasts()[0].view.(*toastView).id
	ta.Toast(Info, "Copied")
	ta.Toast(Info, "Copied")
	ta.settle()
	if n := len(ta.showingToasts()); n != 1 {
		t.Fatalf("%d toasts for the same words", n)
	}
	if text := ta.screenText(); !strings.Contains(text, "INFO ×3") {
		t.Errorf("the title does not count them:\n%s", text)
	}
	ta.Update(toastDone{first})
	ta.settle()
	if len(ta.showingToasts()) != 1 {
		t.Error("the first time closed it: its time starts over")
	}
	if n := len(ta.logs.entries) - before; n != 3 {
		t.Errorf("%d App Log entries, not one each time", n)
	}
	ta.Toast(Warning, "Copied") // another kind: another toast
	ta.settle()
	if n := len(ta.showingToasts()); n != 2 {
		t.Errorf("another kind joined: %d toasts", n)
	}
}
