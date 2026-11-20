package terminu

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

// TestE3Notify: several things one start says come as one toast, a row each
// in the order given, its kind the strongest (Error, then Warning, else the
// first row's); in the App Log each row is an entry of its own at its own
// kind, a misfit's WARNs right after its row (tdp E3, E3-37).
func TestE3Notify(t *testing.T) {
	ta := newTestApp(t)
	before := len(ta.logs.entries)
	misfit := MisfitNotice("/data/config.yaml", "/data/config.yaml.bak", Misfit{"color", "#zzzzzz", "#89b4fa"}, Misfit{Key: `entry 3 (name "")`})
	ta.Notify(
		Notice{Kind: Info, Text: "Settings moved from ~/.app to ~/.config/app"},
		misfit,
		Notice{Kind: Error, Text: "hosts.yaml.bak could not be made: permission denied; hosts.yaml left as it was"},
		Notice{Kind: Warning}, // no row
		Notice{Kind: Warning, Text: `Profile "work" in config.yaml is not in profiles.yaml: using the default`},
	)
	ta.settle()
	if ta.toast == nil || ta.toast.view.(*toastView).kind != Error {
		t.Fatalf("not one Error toast:\n%s", ta.screenText())
	}
	rows := []string{
		"Settings moved from ~/.app to ~/.config/app",
		"1 value and 1 entry in config.yaml did not fit: defaults used and entries taken out, yours kept in config.yaml.bak",
		"hosts.yaml.bak could not be made: permission denied; hosts.yaml left as it was",
		`Profile "work" in config.yaml is not in profiles.yaml: using the default`,
	}
	if got := ta.toast.view.(*toastView).text; !slices.Equal(got, rows) {
		t.Errorf("the rows:\n%s", strings.Join(got, "\n"))
	}
	var got []string
	for _, e := range ta.logs.entries[before:] {
		got = append(got, fmt.Sprint(e.kind, " ", e.text))
	}
	want := []string{
		fmt.Sprint(Info, " ", rows[0]),
		fmt.Sprint(Warning, " ", rows[1]),
		fmt.Sprint(Warning, ` config.yaml: color "#zzzzzz" did not fit; using "#89b4fa"`),
		fmt.Sprint(Warning, ` config.yaml: entry 3 (name "") did not fit; taken out`),
		fmt.Sprint(Error, " ", rows[2]),
		fmt.Sprint(Warning, " ", rows[3]),
	}
	if !slices.Equal(got, want) {
		t.Errorf("the App Log:\n%s", strings.Join(got, "\n"))
	}
	if misfit.Text != DidNotFitText("/data/config.yaml", "/data/config.yaml.bak", Misfit{"color", "#zzzzzz", "#89b4fa"}, Misfit{Key: `entry 3 (name "")`}) {
		t.Errorf("MisfitNotice's row: %q", misfit.Text)
	}

	// The strongest kind: Error, then Warning, else the first row's.
	for _, c := range []struct {
		kinds []ToastKind
		want  ToastKind
	}{
		{[]ToastKind{Info, Warning, Success}, Warning},
		{[]ToastKind{Warning, Error}, Error},
		{[]ToastKind{Error, Warning}, Error},
		{[]ToastKind{Success, Info}, Success},
		{[]ToastKind{Info, Success}, Info},
		{[]ToastKind{Success}, Success},
	} {
		ta := newTestApp(t)
		var ns []Notice
		for i, k := range c.kinds {
			ns = append(ns, Notice{Kind: k, Text: fmt.Sprint("Row ", i)})
		}
		ta.Notify(ns...)
		if ta.toast == nil || ta.toast.view.(*toastView).kind != c.want {
			t.Errorf("%v: not %v", c.kinds, c.want)
		}
	}

	// No rows, no toast; a misfit with no copy still logs its WARN.
	ta = newTestApp(t)
	before = len(ta.logs.entries)
	if cmd := ta.Notify(Notice{Kind: Error}, MisfitNotice("/data/hosts.yaml", "", Misfit{Key: `entry 3 (name "")`})); cmd != nil || ta.toast != nil {
		t.Error("no rows: a toast")
	}
	if e := ta.logs.entries[before:]; len(e) != 1 || e[0].kind != Warning || e[0].text != `hosts.yaml: entry 3 (name "") did not fit; taken out` {
		t.Errorf("no rows: the App Log %+v", e)
	}
	if ta.Notify() != nil {
		t.Error("nothing: a command")
	}
}

// TestE3ComingLater: one that comes after the start joins the start's toast
// while it still shows, a row more, its kind the strongest again, its time
// starting over; once that toast is gone it is a toast of its own; the App
// Log keeps an entry a row (tdp E3 "Coming later", E3-43, from kbu). Notify
// does not join an ordinary toast: that one is replaced, as toasts are.
func TestE3ComingLater(t *testing.T) {
	ta := newTestApp(t)
	view := func() *toastView {
		if ta.toast == nil || ta.toast.closing {
			return nil
		}
		return ta.toast.view.(*toastView)
	}
	before := len(ta.logs.entries)
	ta.Notify(Notice{Kind: Info, Text: "Settings moved from ~/.app to ~/.config/app"})
	ta.settle()
	first := view().id
	ta.Notify(Notice{Kind: Warning, Text: `Namespace "dev" is not in the cluster: using the others`})
	ta.settle()
	v := view()
	if v == nil || !slices.Equal(v.text, []string{"Settings moved from ~/.app to ~/.config/app", `Namespace "dev" is not in the cluster: using the others`}) {
		t.Fatalf("the late one did not join the start's toast: %+v", v)
	}
	if v.kind != Warning {
		t.Errorf("the kind is not the strongest again: %v", v.kind)
	}
	ta.Update(toastDone{first})
	ta.settle()
	if view() == nil {
		t.Fatal("the start toast's time closed the joined one: its time starts over")
	}
	ta.Notify(Notice{Kind: Info, Text: `Kind "Widget" is not in the cluster: using the default`})
	ta.settle()
	if v = view(); len(v.text) != 3 || v.kind != Warning {
		t.Errorf("a third: %d rows, %v", len(v.text), v.kind)
	}
	if n := len(ta.logs.entries) - before; n != 3 {
		t.Errorf("the App Log has %d entries, not one a row", n)
	}

	// Gone (its time up, or Esc), or going: on its own.
	ta.Update(toastDone{v.id}) // its time up: closing
	ta.Notify(Notice{Kind: Info, Text: "While it goes"})
	if v = view(); v == nil || !slices.Equal(v.text, []string{"While it goes"}) {
		t.Errorf("a toast on its way out was joined: %+v", v)
	}
	ta.settle()
	ta.Update(toastDone{v.id})
	ta.settle()
	ta.Notify(Notice{Kind: Info, Text: "Later still"})
	ta.settle()
	if v = view(); v == nil || !slices.Equal(v.text, []string{"Later still"}) || v.kind != Info {
		t.Errorf("after the toast went: %+v", v)
	}
	ta.press("Esc")
	ta.Notify(Notice{Kind: Warning, Text: "After Esc"})
	ta.settle()
	if v = view(); v == nil || !slices.Equal(v.text, []string{"After Esc"}) {
		t.Errorf("after Esc: %+v", v)
	}

	// An ordinary toast is not joined, and does not take later ones in.
	ta.Toast(Info, "Saved")
	ta.settle()
	ta.Notify(Notice{Kind: Warning, Text: "A late one"})
	ta.settle()
	if v = view(); v == nil || !slices.Equal(v.text, []string{"A late one"}) {
		t.Errorf("an ordinary toast was joined: %+v", v)
	}
}
