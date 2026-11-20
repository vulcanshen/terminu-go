package terminu

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

// TestE3Notify: several things one start says are a toast each, in the
// order given, stacked as every toast is; in the App Log each is an entry
// of its own at its own kind, a misfit's WARNs right after it (tdp E3,
// E3-37).
func TestE3Notify(t *testing.T) {
	ta := newTestApp(t)
	before := len(ta.logs.entries)
	misfit := MisfitNotice("/data/config.yaml", "/data/config.yaml.bak", Misfit{"color", "#zzzzzz", "#89b4fa"}, Misfit{Key: `entry 3 (name "")`})
	ta.Notify(
		Notice{Kind: Info, Text: "Settings moved from ~/.app to ~/.config/app"},
		misfit,
		Notice{Kind: Warning}, // no toast
		Notice{Kind: Warning, Text: `Profile "work" in config.yaml is not in profiles.yaml: using the default`},
	)
	ta.settle()
	rows := []string{
		"Settings moved from ~/.app to ~/.config/app",
		"1 value and 1 entry in config.yaml did not fit: defaults used and entries taken out, yours kept in config.yaml.bak",
		`Profile "work" in config.yaml is not in profiles.yaml: using the default`,
	}
	var got []string
	var kinds []ToastKind
	for _, l := range ta.showingToasts() {
		v := l.view.(*toastView)
		got, kinds = append(got, strings.Join(v.text, " ")), append(kinds, v.kind)
	}
	if !slices.Equal(got, rows) || !slices.Equal(kinds, []ToastKind{Info, Warning, Warning}) {
		t.Errorf("not a toast each, in order: %q %v", got, kinds)
	}
	var log []string
	for _, e := range ta.logs.entries[before:] {
		log = append(log, fmt.Sprint(e.kind, " ", e.text))
	}
	want := []string{
		fmt.Sprint(Info, " ", rows[0]),
		fmt.Sprint(Warning, " ", rows[1]),
		fmt.Sprint(Warning, ` config.yaml: color "#zzzzzz" did not fit; using "#89b4fa"`),
		fmt.Sprint(Warning, ` config.yaml: entry 3 (name "") did not fit; taken out`),
		fmt.Sprint(Warning, " ", rows[2]),
	}
	if !slices.Equal(log, want) {
		t.Errorf("the App Log:\n%s", strings.Join(log, "\n"))
	}
	if misfit.Text != DidNotFitText("/data/config.yaml", "/data/config.yaml.bak", Misfit{"color", "#zzzzzz", "#89b4fa"}, Misfit{Key: `entry 3 (name "")`}) {
		t.Errorf("MisfitNotice's text: %q", misfit.Text)
	}

	// No text, no toast; a misfit with no copy still logs its WARN.
	ta = newTestApp(t)
	before = len(ta.logs.entries)
	if cmd := ta.Notify(Notice{Kind: Error}, MisfitNotice("/data/hosts.yaml", "", Misfit{Key: `entry 3 (name "")`})); cmd != nil || len(ta.toasts) != 0 {
		t.Error("no text: a toast")
	}
	if e := ta.logs.entries[before:]; len(e) != 1 || e[0].kind != Warning || e[0].text != `hosts.yaml: entry 3 (name "") did not fit; taken out` {
		t.Errorf("no text: the App Log %+v", e)
	}
	if ta.Notify() != nil {
		t.Error("nothing: a command")
	}
}
