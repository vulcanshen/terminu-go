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
