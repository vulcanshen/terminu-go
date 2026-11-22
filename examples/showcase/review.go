package main

import (
	terminu "github.com/vulcanshen/terminu-go"
)

// reviewScreen is [R]EVIEW: panels that follow a leading panel (tdp L5).
// [1] CHANGES and [2] BRANCHES lead; the right side follows the one of them
// that last had the focus: DIFF and HISTORY while CHANGES leads, the third
// place hidden and the other two taking its room; COMMITS, DESCRIPTION and
// FILES while BRANCHES does. Moving into the right side changes nothing.
type reviewScreen struct {
	screen *terminu.Screen
	panels map[string]*terminu.Panel
}

func newReviewScreen() *reviewScreen {
	rs := &reviewScreen{panels: map[string]*terminu.Panel{}}
	list := func(rows ...string) *terminu.List {
		l := &terminu.List{Columns: []terminu.Column{{Shrink: true}}}
		var rr []terminu.Row
		for _, r := range rows {
			rr = append(rr, terminu.Row{ID: r, Cells: []string{r}})
		}
		l.SetRows(rr)
		return l
	}
	pager := func(lines ...string) *terminu.Pager {
		p := &terminu.Pager{}
		p.SetLines(lines)
		return p
	}
	diff := &terminu.Pager{}
	diff.SetDiff("package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n", "package main\n\nfunc main() {\n\tprintln(\"hello, showcase\")\n}\n")
	add := func(label string, body terminu.Body) *terminu.Panel {
		p := &terminu.Panel{Label: label, Body: body}
		rs.panels[label] = p
		return p
	}
	changes := add("Changes", list("main.go", "README.md", "docs/notes.md"))
	branches := add("Branches", list("main", "release-1.2", "fix-toasts"))
	sign := &terminu.Panel{Label: "Repo", Body: pager("showcase on main", "3 changes, 2 commits ahead"), Sign: terminu.SignDim}
	d := add("Diff", diff)
	history := add("History", pager("5d6e7f8 Say hello to the showcase", "1a2b3c4 Start main.go"))
	commits := add("Commits", list("5d6e7f8 Say hello to the showcase", "1a2b3c4 Start main.go"))
	desc := add("Description", pager("Say hello to the showcase", "", "The greeting names who it is for."))
	files := add("Files", list("main.go", "README.md"))
	var s *terminu.Screen
	s = &terminu.Screen{Name: "Review", Key: "R", Desc: "panels that follow a leading one",
		Panels:    []*terminu.Panel{changes, branches, sign, commits, desc, files},
		Following: []int{3, 4, 5},
		Leads: []terminu.Lead{
			{Panel: changes, Shows: []*terminu.Panel{d, history, nil}},
			{Panel: branches, Shows: []*terminu.Panel{commits, desc, files}},
		},
		NarrowBelow: 60,
		// The leading panels on the left, one over the other; the sign on top
		// of the right side, the following places under it, a hidden one's
		// room going to the others (Screen.Hidden).
		Layout: func(w, h int) []terminu.Rect {
			left := w / 3
			top := min(4, h/4)
			r := []terminu.Rect{
				{X: 0, Y: 0, W: left, H: h / 2}, {X: 0, Y: h / 2, W: left, H: h - h/2},
				{X: left, Y: 0, W: w - left, H: top},
			}
			shown := []int{}
			for _, place := range []int{3, 4, 5} {
				if !s.Hidden(place) {
					shown = append(shown, place)
				}
			}
			rows := map[int]terminu.Rect{}
			y, room := top, h-top
			for k, place := range shown {
				ph := room / len(shown)
				if k == len(shown)-1 {
					ph = h - y
				}
				rows[place] = terminu.Rect{X: left, Y: y, W: w - left, H: ph}
				y += ph
			}
			for _, place := range []int{3, 4, 5} {
				r = append(r, rows[place]) // a hidden place: no room
			}
			return r
		}}
	rs.screen = s
	return rs
}
