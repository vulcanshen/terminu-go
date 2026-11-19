package terminu

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// TestPagerDiff: a panel showing a difference draws it as UNIFIED does,
// only the parts that differ with 3 lines around, a line between them as
// wide as the panel, both line numbers, - Red and + Green, no tool's own
// output; the same text on both sides is the empty state; SetLines goes
// back to plain lines (note-103, from gitreu).
func TestPagerDiff(t *testing.T) {
	ta := newTestApp(t)
	pg := ta.Screen().Panels[1].Body.(*Pager)
	pg.EmptyFact = "No changes"
	var a, b strings.Builder
	for i := range 40 {
		fmt.Fprintf(&a, "line %d\n", i)
		switch i {
		case 5, 30:
			fmt.Fprintf(&b, "edit %d\n", i)
		default:
			fmt.Fprintf(&b, "line %d\n", i)
		}
	}
	pg.SetDiff(a.String(), b.String())
	frame, text := frameText(ta), ta.screenText()
	if !strings.Contains(frame, Fg(Red).Render("-line 5")) || !strings.Contains(frame, Fg(Green).Render("+edit 30")) {
		t.Errorf("no - Red, + Green:\n%s", text)
	}
	if strings.Contains(text, "line 15") || strings.Contains(text, "@@") || strings.Contains(text, "diff --git") {
		t.Errorf("more than the parts that differ, or a tool's output:\n%s", text)
	}
	plain := func() []string {
		var out []string
		for _, l := range pg.Lines() {
			out = append(out, StripStyles(l))
		}
		return out
	}
	lines := plain()
	if !regexp.MustCompile(`^ ?6 +-line 5$`).MatchString(lines[3]) || !regexp.MustCompile(`^ ?3 +3  line 2$`).MatchString(lines[0]) {
		t.Errorf("the line numbers: %q", lines[:4])
	}
	sep := func() int {
		for _, l := range plain() {
			if strings.Trim(l, "─") == "" && l != "" {
				return Width(l)
			}
		}
		return -1
	}
	if got := sep(); got != pg.width || got <= 0 {
		t.Errorf("the line between the parts is %d wide, the text %d", got, pg.width)
	}
	ta.resize(150, 40)
	ta.Frame()
	if got := sep(); got != pg.width || got <= 0 {
		t.Errorf("after a resize the line is %d wide, the text %d", got, pg.width)
	}

	// The same on both sides: the empty state.
	pg.SetDiff(a.String(), a.String())
	if !pg.Empty() || !strings.Contains(ta.screenText(), "No changes") {
		t.Errorf("the same text:\n%s", ta.screenText())
	}
	// SetLines: plain lines again, kept as they are on a resize.
	pg.SetDiff(a.String(), b.String())
	pg.SetLines([]string{"plain"})
	ta.resize(120, 40)
	if text := ta.screenText(); !strings.Contains(text, "plain") || strings.Contains(text, "-line 5") {
		t.Errorf("SetLines did not go back to plain lines:\n%s", text)
	}
}

// TestPagerDiffItems: a panel showing what holds many lists the items that
// differ, each with its mark in its colour, drawn as the list OpenDiff goes
// into draws them, a long name cut from the front, laid out again as the
// panel's width changes; none is the empty state; SetLines goes back to
// plain lines (note-106, from gittu).
func TestPagerDiffItems(t *testing.T) {
	ta := newTestApp(t)
	pg := ta.Screen().Panels[1].Body.(*Pager)
	pg.EmptyFact = "No changes"
	long := "internal/very/long/path/to/some/deeply/nested/file.go"
	items := []DiffItem{
		{Mark: DiffAdded, Name: "new.go"},
		{Mark: DiffRemoved, Name: "old.txt"},
		{Mark: DiffChanged, Name: "src/", Items: func() ([]DiffItem, error) { return nil, nil }},
		{Mark: DiffAdded, Name: long},
	}
	pg.SetDiffItems(items)
	frame, text := frameText(ta), ta.screenText()
	for _, m := range []string{Fg(Green).Render("+"), Fg(Red).Render("-"), Fg(Peach).Render("~")} {
		if !strings.Contains(frame, m) {
			t.Errorf("no %q in its colour:\n%s", m, text)
		}
	}
	for _, n := range []string{"new.go", "old.txt", "src/"} {
		if !strings.Contains(text, n) {
			t.Errorf("no %s:\n%s", n, text)
		}
	}
	// Drawn as the list going in draws them, the first row aside (the
	// cursor's).
	ta.OpenDiff(ta.pods, DiffItem{Anchor: "a", Other: "b", Items: func() ([]DiffItem, error) { return items, nil }})
	in := ta.pods.level().list.Draw(40, 12, false)
	ta.pods.LeaveDiff()
	got := pg.Draw(40, 10, false)
	line := func(lines []string, name string) string {
		for _, l := range lines {
			if s := StripStyles(l); strings.Contains(s, name) {
				return strings.TrimRight(s, " ")
			}
		}
		return ""
	}
	for _, n := range []string{"old.txt", "src/", "file.go"} {
		if a, b := line(got, n), line(in, n); a == "" || a != b {
			t.Errorf("%s: %q in the panel, %q in the list", n, a, b)
		}
	}
	if l := line(got, "file.go"); !strings.Contains(l, "…") || strings.Contains(l, "internal") {
		t.Errorf("a long name is not cut from the front: %q", l)
	}
	if l := line(pg.Draw(80, 10, false), "file.go"); !strings.Contains(l, long) {
		t.Errorf("wider, the long name is not laid out again: %q", l)
	}

	pg.SetDiffItems(nil)
	ta.Frame()
	if !pg.Empty() || !strings.Contains(ta.screenText(), "No changes") {
		t.Errorf("none:\n%s", ta.screenText())
	}
	pg.SetDiffItems(items)
	pg.SetLines([]string{"plain"})
	ta.resize(120, 40)
	if text := ta.screenText(); !strings.Contains(text, "plain") || strings.Contains(text, "new.go") {
		t.Errorf("SetLines did not go back to plain lines:\n%s", text)
	}
}
