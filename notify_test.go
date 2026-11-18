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
		Notice{Kind: Warning, Text: `profile "work" in config.yaml is not in profiles.yaml: using the default`},
	)
	ta.settle()
	if ta.toast == nil || ta.toast.view.(*toastView).kind != Error {
		t.Fatalf("not one Error toast:\n%s", ta.screenText())
	}
	rows := []string{
		"Settings moved from ~/.app to ~/.config/app",
		"1 value and 1 entry in config.yaml did not fit: defaults used and entries taken out, yours kept in config.yaml.bak",
		"hosts.yaml.bak could not be made: permission denied; hosts.yaml left as it was",
		`profile "work" in config.yaml is not in profiles.yaml: using the default`,
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
