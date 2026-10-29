package tdptest

import (
	"fmt"
	"strings"
	"testing"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"

	terminu "github.com/vulcanshen/terminu-go"
	"github.com/vulcanshen/terminu-go/internal/testhook"
)

// everywhere is an app that draws v in every place a value from outside can
// go: list cells, a value panel, a pager, the screen's top row; open lists
// each popup that can show it.
func everywhere(v string) (*terminu.App, []func(a *terminu.App) tea.Cmd) {
	a := terminu.New("tricky")
	a.MinWidth, a.MinHeight = 20, 10
	a.Splash = func(frame, w, h int) ([]string, bool) { return []string{"tricky"}, true }
	a.Quit = &terminu.Ask{Title: "Quit", Question: "Quit tricky?", Verb: "quit"}
	a.Status = func(int) []terminu.Span {
		return []terminu.Span{{Text: v}, {Text: " " + v, Style: terminu.Fg(terminu.Lavender)}}
	}
	list := &terminu.List{Columns: []terminu.Column{{Title: "Name", Shrink: true}, {Title: "Note"}}, Filterable: true, Marks: true}
	list.SetRows([]terminu.Row{{ID: "1", Cells: []string{v, v}}, {ID: "2", Cells: []string{"plain", v}}})
	pager := &terminu.Pager{}
	pager.SetLines([]string{v, "after"})
	values := &terminu.Values{Fields: []*terminu.Field{{Label: "Name", Value: func() string { return v }}, {Label: "Path", Path: true, Value: func() string { return v }}}}
	a.AddScreen(&terminu.Screen{Name: "One", Key: "O", Panels: []*terminu.Panel{
		{Label: "Items", Body: list, Actions: []terminu.Action{{Key: "r", Label: "Rename", Desc: v, Scope: terminu.ItemScope, Do: func() tea.Cmd { return nil }}}},
		{Label: "Log", Body: pager},
		{Label: "Settings", Body: values},
	}})
	a.AddScreen(&terminu.Screen{Name: "Two", Key: "T", Panels: []*terminu.Panel{{Label: "Empty", Body: &terminu.Pager{EmptyFact: v}}}})
	nop := func() tea.Cmd { return nil }
	open := []func(a *terminu.App) tea.Cmd{
		func(a *terminu.App) tea.Cmd { return a.Toast(terminu.Info, v) },
		func(a *terminu.App) tea.Cmd { return a.Toast(terminu.Warning, v) },
		func(a *terminu.App) tea.Cmd {
			return a.Confirm(terminu.Ask{Title: "Delete", Subject: v, Question: v, Verb: "delete", Details: []string{v, v}, Warn: []int{1}}, nop)
		},
		func(a *terminu.App) tea.Cmd { return a.ErrorAbout("Open", v, v) },
		func(a *terminu.App) tea.Cmd {
			return a.MenuAbout("Pick", v, terminu.MenuSection{Title: v, Actions: []terminu.Action{{Label: v, Desc: v, Do: nop}, {Key: "x", Label: "Other", Desc: v, Do: nop}}})
		},
		func(a *terminu.App) tea.Cmd {
			return a.Viewer(&terminu.Viewer{Title: "Log", Subject: v, Lines: []string{v, v + v, ""}, Tabs: []string{v, "Two"}})
		},
		func(a *terminu.App) tea.Cmd {
			return a.Viewer(&terminu.Viewer{Title: "Wide", Subject: v, Lines: []string{v, v + v}})
		},
		func(a *terminu.App) tea.Cmd {
			return a.Find(&terminu.Finder{Title: "Search", Subject: v, Items: []terminu.FinderItem{{ID: "1", Text: v}, {ID: "2", Text: "b " + v}},
				Preview: func(it terminu.FinderItem) (string, []string) { return it.Text, []string{v, v} }})
		},
		func(a *terminu.App) tea.Cmd {
			return a.Select(terminu.Select{Title: "Pick", Subject: v, Options: []string{v, "two"}})
		},
		func(a *terminu.App) tea.Cmd {
			return a.Checks(terminu.Checks{Title: "Pick", Subject: v, Options: []string{v, "two"}, Checked: []bool{true, false}})
		},
		func(a *terminu.App) tea.Cmd {
			return a.Text(terminu.TextInput{Title: "Rename", Subject: v, Prompt: v, Verb: "rename", Old: v})
		},
		func(a *terminu.App) tea.Cmd {
			return a.Text(terminu.TextInput{Title: "New", Prompt: "Name", Verb: "create", Suggest: v})
		},
		func(a *terminu.App) tea.Cmd {
			return a.Textarea(terminu.Textarea{Title: "Note", Subject: v, Old: v})
		},
		func(a *terminu.App) tea.Cmd {
			return a.Form(&terminu.Form{Title: "Edit", Subject: v, Button: "Save", Fields: []*terminu.FormField{
				{Label: "Name", Value: func() string { return v }},
				{Label: "Path", Path: true, Stacked: true, Value: func() string { return v }},
				{Label: "Kind", Options: []string{v, "two"}},
			}})
		},
	}
	return a, open
}

// TestTrickyValuesEverywhere puts every Tricky value in every place a value
// from outside goes, on a small and a common terminal; a value with a Nerd
// Font glyph again with an icon and a powerline symbol two cells wide. Each
// frame is checked for tdp L4 and each popup closes one layer an Esc (tdp
// K4).
func TestTrickyValuesEverywhere(t *testing.T) {
	for i, v := range Tricky {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			trickyEverywhere(t, v)
			if strings.ContainsFunc(v, func(r rune) bool { return r >= 0xE000 && r <= 0xF8FF || r >= 0xF0000 }) {
				terminu.SetIconWidth(2, 2)
				defer terminu.SetIconWidth(1, 1)
				trickyEverywhere(t, v)
			}
		})
	}
}

func trickyEverywhere(t *testing.T, v string) {
	t.Helper()
	for _, size := range [][2]int{{20, 10}, {80, 24}} {
		a, open := everywhere(v)
		d := Start(t, a, size[0], size[1])
		d.AllowLineBreaks = true // tricky names on purpose
		for _, panel := range []string{"1", "2", "3"} {
			d.Press(panel, "Space")
			closeAll(t, d, "the Space menu with "+fmt.Sprintf("%q", v))
		}
		d.Press("1", "/")
		d.Paste(v)
		d.Press("Esc", "T")
		for n, f := range open {
			d.Do(func() tea.Cmd { return f(a) })
			if n > 1 && testhook.Layers(a) != 1 {
				t.Errorf("popup %d with %q did not open", n, v)
			}
			escape(t, d, fmt.Sprintf("popup %d with %q", n, v))
		}
	}
}

// TestTrickyValuesTyped types, pastes and bursts each value into a text
// input: control characters are drawn, not sent, and Enter holds them back
// (tdp input).
func TestTrickyValuesTyped(t *testing.T) {
	for i, v := range Tricky {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			a, _ := everywhere("")
			d := Start(t, a, 80, 24)
			d.AllowLineBreaks = true // tricky names on purpose
			for _, how := range []string{"paste", "burst"} {
				var got *string
				d.Do(func() tea.Cmd {
					return a.Text(terminu.TextInput{Title: "Rename", Prompt: "New name", Verb: "rename",
						Done: func(s string) tea.Cmd { got = &s; return nil }})
				})
				sent := v
				if how == "paste" {
					d.Paste(v)
				} else {
					sent = strings.Map(func(r rune) rune {
						if unicode.IsControl(r) {
							return -1 // a control character typed is its key, or nothing
						}
						return r
					}, v)
					d.Burst(sent)
				}
				d.Press("Enter")
				control := strings.ContainsFunc(sent, unicode.IsControl)
				switch {
				case control && !d.Has("can't have control characters"):
					t.Errorf("%s %q: Enter took a value with control characters", how, v)
				case !control && strings.TrimSpace(sent) != "" && got == nil:
					t.Errorf("%s %q: Enter did not take it", how, v)
				}
				closeAll(t, d, how)
			}
		})
	}
}

// TestSizesAndMashes opens the app on every size in Sizes and presses each
// of Mashes there.
func TestSizesAndMashes(t *testing.T) {
	for _, size := range Sizes {
		t.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(t *testing.T) {
			a, _ := everywhere("value")
			d := Start(t, a, size[0], size[1])
			for _, keys := range Mashes {
				d.Mash(keys...)
				if d.Quit() {
					t.Fatalf("%v quit the app", keys)
				}
				closeAll(t, d, fmt.Sprint(keys))
				d.Press("Esc")
			}
		})
	}
}
