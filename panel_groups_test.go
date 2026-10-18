package terminu

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// groupList is the list of components/layout/panel's groups example.
func groupList() *List {
	l := &List{Filterable: true}
	l.SetRows([]Row{
		{ID: "pods", Cells: []string{"pods"}, Group: "Workloads"},
		{ID: "deployments", Cells: []string{"deployments"}, Group: "Workloads"},
		{ID: "statefulsets", Cells: []string{"statefulsets"}, Group: "Workloads"},
		{ID: "services", Cells: []string{"services"}, Group: "Network"},
		{ID: "ingresses", Cells: []string{"ingresses"}, Group: "Network"},
	})
	return l
}

func plainLines(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = strings.TrimRight(StripStyles(l), " ")
	}
	return out
}

// TestPanelListGroups: a group title is Blue, upper case, not bold, one cell
// from the frame; its items two cells further in; no blank row or separator
// between groups (components/layout/panel, groups).
func TestPanelListGroups(t *testing.T) {
	l := groupList()
	out := l.Draw(24, 10, true)
	for i, line := range out {
		if Width(line) != 24 {
			t.Fatalf("row %d is %d wide", i, Width(line))
		}
	}
	got := plainLines(out)[:7]
	want := []string{" WORKLOADS", "   pods" + strings.Repeat(" ", 24-7-Width(glyphHard)) + glyphHard, "   deployments", "   statefulsets",
		" NETWORK", "   services", "   ingresses"}
	if !slices.Equal(got, want) {
		t.Errorf("rows:\n%q\nwant\n%q", got, want)
	}
	if !strings.Contains(out[0], Fg(Blue).Render("WORKLOADS")) || !strings.Contains(out[4], Fg(Blue).Render("NETWORK")) {
		t.Errorf("a group title is not Blue and plain: %q", out[0])
	}
	// Sorted, the rows stay in their groups, the groups in their order.
	l.SortBy(SortKey{Col: 0})
	got = plainLines(l.Draw(24, 10, true))[:7]
	order := []string{"WORKLOADS", "deployments", "pods", "statefulsets", "NETWORK", "ingresses", "services"}
	for i, w := range order {
		if !strings.Contains(got[i], w) {
			t.Errorf("sorted, row %d is %q, want %s", i, got[i], w)
		}
	}
}

// TestPanelListGroupsMarks: with a mark column, the marks go in the two cells
// the items are further in: the mark one cell from the frame, a space, the
// text three cells in as with no marks; a group title keeps no mark column
// (components/layout/panel, groups).
func TestPanelListGroupsMarks(t *testing.T) {
	withWidths(t, 1, 1)
	l := groupList()
	l.Marks = true
	l.ToggleMark() // pods, under the cursor
	got := plainLines(l.Draw(24, 10, true))[:3]
	want := []string{" WORKLOADS", " " + glyphMarked + " pods" + strings.Repeat(" ", 24-7-Width(glyphHard)) + glyphHard, "   deployments"}
	if !slices.Equal(got, want) {
		t.Errorf("rows:\n%q\nwant\n%q", got, want)
	}
}

// TestPanelListGroupsCursor: the cursor skips group titles: j|k step over
// them, gg|G go to the first and last item, N of M counts items; scrolled to a
// group's first item, its title stays on screen (components/layout/panel,
// groups).
func TestPanelListGroupsCursor(t *testing.T) {
	l := groupList()
	for _, step := range []struct{ key, want string }{
		{"j", "deployments"}, {"j", "statefulsets"}, {"j", "services"}, {"k", "statefulsets"},
		{"G", "ingresses"}, {"gg", "pods"}, {"k", "ingresses"},
	} {
		l.Key(step.key, keyMsg(step.key))
		if r, _ := l.Current(); r.ID != step.want {
			t.Fatalf("%s: on %q, want %q", step.key, r.ID, step.want)
		}
	}
	l.Select("services")
	out := plainLines(l.Draw(24, 4, true))
	if !strings.Contains(out[2], "NETWORK") || !strings.Contains(out[3], "services") {
		t.Errorf("scrolled to services, the group title is not in sight: %q", out)
	}
	if l.Scroll() != scrollText(true, 4, 0, 5) {
		t.Errorf("scroll position %q: only items count", l.Scroll())
	}
	l.Select("pods")
	if out := plainLines(l.Draw(24, 4, true)); out[0] != " WORKLOADS" {
		t.Errorf("back on the first item, its group title is not in sight: %q", out)
	}
}

// TestPanelListGroupsFilter: a group the panel filter leaves no item of has
// no title (components/layout/panel, groups).
func TestPanelListGroupsFilter(t *testing.T) {
	l := groupList()
	l.OpenFilter()
	for _, k := range []string{"s", "e", "r", "v"} {
		l.Key(k, keyMsg(k))
	}
	text := strings.Join(plainLines(l.Draw(24, 10, true)), "\n")
	if strings.Contains(text, "WORKLOADS") || !strings.Contains(text, "NETWORK") || !strings.Contains(text, "services") {
		t.Errorf("filtered to services:\n%s", text)
	}
}

// TestPanelDetailsFollowCursor: a details panel shows the item under the
// other panel's cursor at once, its label too, in the same frame; the focus
// stays (components/layout/panel, details).
func TestPanelDetailsFollowCursor(t *testing.T) {
	ta := &testApp{App: New("test")}
	kinds := groupList()
	details := &Panel{Body: &Pager{}}
	var moves []string
	kinds.OnMove = func(r Row) tea.Cmd {
		moves = append(moves, r.ID)
		details.Label = r.Cells[0]
		details.Body.(*Pager).SetLines([]string{"kind " + r.ID})
		return nil
	}
	ta.AddScreen(&Screen{Name: "Kinds", Key: "W", Panels: []*Panel{{Label: "Kinds", Body: kinds}, details}})
	ta.resize(80, 24)
	if text := ta.screenText(); !strings.Contains(text, "[2] PODS") || !strings.Contains(text, "kind pods") {
		t.Errorf("the first frame does not show the first item:\n%s", text)
	}
	ta.press("j", "j", "j")
	if text := ta.screenText(); !strings.Contains(text, "[2] SERVICES") || !strings.Contains(text, "kind services") {
		t.Errorf("after j the details are not the item under the cursor:\n%s", text)
	}
	if ta.Screen().Focused().Label != "Kinds" {
		t.Errorf("the focus moved to %q", ta.Screen().Focused().Label)
	}
	ta.resize(90, 24)
	if want := []string{"pods", "deployments", "statefulsets", "services"}; !slices.Equal(moves, want) {
		t.Errorf("OnMove ran for %q, want %q: once per new item", moves, want)
	}
	// Unfocused, the group titles dim with the panel (Rules T2).
	ta.press("Tab")
	if strings.Contains(strings.Join(ta.Frame(), "\n"), Fg(Blue).Render("WORKLOADS")) {
		t.Errorf("the group title of an unfocused list is not dimmed")
	}
}

// TestValuesOnMove: a value panel says where its cursor came to: the field and
// the option in it (components/layout/panel, details).
func TestValuesOnMove(t *testing.T) {
	ta := newTestApp(t)
	var got []string
	ta.values.OnMove = func(f *Field, option int) tea.Cmd {
		got = append(got, fmt.Sprintf("%s/%d", f.Label, option))
		return nil
	}
	// The cursor was on Name before OnMove was set: it runs from the next
	// place on.
	ta.press("3", "j", "j")
	if want := []string{"Layout/0", "Layout/1"}; !slices.Equal(got, want) {
		t.Errorf("OnMove: %q, want %q", got, want)
	}
}

// TestK3EnterToDetails: Enter on a list may send the focus to its details
// panel, on the items the app says (Rules K3, components/layout/panel,
// details).
func TestK3EnterToDetails(t *testing.T) {
	ta := &testApp{App: New("test")}
	kinds := groupList()
	details := &Panel{Label: "Details", Body: &Pager{}}
	ran := 0
	list := &Panel{Label: "Kinds", Body: kinds, Details: details, Actions: []Action{{
		Key: "Enter", Label: "Edit", Desc: "the kind", Scope: ItemScope, Hint: "edit",
		ToDetails: func() bool { r, _ := kinds.Current(); return r.ID != "ingresses" },
		Do:        func() tea.Cmd { ran++; return nil },
	}}}
	ta.AddScreen(&Screen{Name: "Kinds", Key: "W", Panels: []*Panel{list, details}})
	ta.resize(80, 24)
	ta.press("Enter")
	if ta.Screen().Focused() != details || ran != 1 {
		t.Errorf("Enter on pods: focus on %q, Do ran %d times", ta.Screen().Focused().Label, ran)
	}
	ta.press("1", "G", "Enter")
	if ta.Screen().Focused() != list || ran != 2 {
		t.Errorf("Enter on ingresses, which the app keeps: focus on %q, Do ran %d times", ta.Screen().Focused().Label, ran)
	}
}

// TestK3ToDetailsOnlyEnter: only Enter, on a panel whose details are another
// panel of its screen, may send the focus there (Rules K3).
func TestK3ToDetailsOnlyEnter(t *testing.T) {
	yes := func() bool { return true }
	details := &Panel{Label: "Details", Body: &Pager{}}
	enter := []Action{{Key: "Enter", Label: "Edit", Scope: ItemScope, ToDetails: yes}}
	for name, s := range map[string]*Screen{
		"not Enter": {Name: "A", Key: "W", Panels: []*Panel{{Label: "L", Body: groupList(), Details: details,
			Actions: []Action{{Key: "e", Label: "Edit", Scope: ItemScope, ToDetails: yes}}}, details}},
		"no details":        {Name: "B", Key: "W", Panels: []*Panel{{Label: "L", Body: groupList(), Actions: enter}}},
		"details elsewhere": {Name: "C", Key: "W", Panels: []*Panel{{Label: "L", Body: groupList(), Details: details, Actions: enter}}},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: AddScreen took it", name)
				}
			}()
			New("test").AddScreen(s)
		}()
	}
}
