package terminu

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestControlCharsNamesAndShortTexts: in a name or a short text every
// control character, a tab, \n and the ESC of a colour code too, is drawn by
// its name in Red, nothing read, nothing dropped, nothing sent to the
// terminal (components, control characters: components-27; panel-136).
func TestControlCharsNamesAndShortTexts(t *testing.T) {
	v := "a\tb\x1b[31mc\nd"
	shown := `a\tb\x1b[31mc\nd`
	// A title's letters in upper case, the names as they are; after a "[" as a
	// hotkey marker would be (Upper).
	upper := `A\tB\x1b[31mc\nd`
	nop := func() tea.Cmd { return nil }
	titles := map[string]bool{"a group" + "'s title": true, "a panel's tab": true, "a menu's section title": true, "a viewer's tab": true}
	for _, c := range []struct {
		where string
		set   func(ta *testApp)
	}{
		{"a list's cell", func(ta *testApp) {
			ta.list.SetRows([]Row{{ID: "1", Cells: []string{"first", "1"}}, {ID: "2", Cells: []string{v, "2"}}})
		}},
		{"a list's location", func(ta *testApp) { ta.list.Location = v }},
		{"a group's title", func(ta *testApp) {
			ta.list.SetRows([]Row{{ID: "1", Cells: []string{"x"}, Group: v}})
		}},
		{"a value", func(ta *testApp) {
			ta.values.Fields[0].Value = func() string { return v }
			ta.press("3", "j")
		}},
		{"an option in place", func(ta *testApp) {
			ta.values.Fields[1].Options = []string{"row", v}
			ta.press("3")
		}},
		{"an empty fact", func(ta *testApp) { ta.list.SetRows(nil); ta.list.EmptyFact = v }},
		{"the statusbar", func(ta *testApp) { ta.Status = func(int) []Span { return []Span{{Text: v}} } }},
		{"a panel's tab", func(ta *testApp) { ta.pods.Tabs = []string{v} }},
		{"a menu's label", func(ta *testApp) {
			ta.Menu("Pick", MenuSection{Actions: []Action{{Label: "first", Do: nop}, {Label: v, Do: nop}}})
		}},
		{"a menu's description", func(ta *testApp) {
			ta.Menu("Pick", MenuSection{Actions: []Action{{Label: "first", Desc: v, Do: nop}}})
		}},
		{"a menu's section title", func(ta *testApp) {
			ta.Menu("Pick", MenuSection{Title: v, Actions: []Action{{Label: "first", Do: nop}}})
		}},
		{"a popup's subject", func(ta *testApp) {
			ta.Confirm(Ask{Title: "Delete", Subject: v, Question: "Sure?", Verb: "delete"}, nop)
		}},
		{"a confirm's question", func(ta *testApp) { ta.Confirm(Ask{Title: "Delete", Question: v, Verb: "delete"}, nop) }},
		{"a confirm's details", func(ta *testApp) {
			ta.Confirm(Ask{Title: "Delete", Question: "Sure?", Details: []string{v}, Verb: "delete"}, nop)
		}},
		{"a toast", func(ta *testApp) { ta.Toast(Info, v) }},
		{"an error popup", func(ta *testApp) { ta.Error("Open", v) }},
		{"a viewer's tab", func(ta *testApp) { ta.Viewer(&Viewer{Title: "Log", Lines: []string{"x"}, Tabs: []string{"one", v}}) }},
		{"a finder's item", func(ta *testApp) {
			ta.Find(&Finder{Title: "Find", Items: []FinderItem{{ID: "1", Text: "first"}, {ID: "2", Text: v}}})
		}},
		{"a select's option", func(ta *testApp) { ta.Select(Select{Title: "Pick", Options: []string{"first", v}}) }},
		{"a checkbox's option", func(ta *testApp) {
			ta.Checks(Checks{Title: "Pick", Options: []string{"first", v}, Checked: []bool{false, false}})
		}},
		{"an order's item", func(ta *testApp) { ta.Reorder(Reorder{Title: "Order", Options: []string{"first", v}}) }},
		{"an input's prompt", func(ta *testApp) { ta.Text(TextInput{Title: "Rename", Prompt: v, Verb: "rename"}) }},
		{"a form's value", func(ta *testApp) {
			ta.Form(&Form{Title: "Edit", Fields: []*FormField{{Label: "Note", Value: func() string { return "x" }}, {Label: "Name", Value: func() string { return v }}}})
		}},
		{"a form's option", func(ta *testApp) {
			ta.Form(&Form{Title: "Edit", Fields: []*FormField{{Label: "Note", Value: func() string { return "x" }}, {Label: "Kind", Options: []string{"first", v}}}})
		}},
	} {
		t.Run(c.where, func(t *testing.T) {
			ta := newTestApp(t)
			ta.resize(160, 40)
			c.set(ta)
			ta.settle()
			frame := strings.Join(ta.Frame(), "\n")
			shown := shown
			if titles[c.where] {
				shown = upper
			}
			if !strings.Contains(StripStyles(frame), shown) {
				t.Errorf("%s does not show %s:\n%s", c.where, shown, ta.screenText())
			}
			if strings.Contains(frame, "\x1b[31m") || strings.ContainsAny(StripStyles(frame), "\t") {
				t.Errorf("%s sends a control character to the terminal", c.where)
			}
			if !strings.Contains(frame, nameSGR+`\t`) && !strings.Contains(frame, nameOnBlock+`\t`) {
				t.Errorf("%s: the names are not Red", c.where)
			}
		})
	}
}

// TestControlCharsLineColumn: in a list column marked a line of text (Line)
// a Tab is input_tab_width spaces, its layout; any other control character
// is still its name in Red, and a column not marked keeps the Tab's name
// (components, control characters: components-29; webu WB21).
func TestControlCharsLineColumn(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(160, 40)
	ta.list.Columns = []Column{{Title: "Text", Line: true, Shrink: true}, {Title: "Name"}}
	ta.list.SetRows([]Row{{ID: "1", Cells: []string{"first", "x"}},
		{ID: "2", Cells: []string{"x\ty\x1b[31m {", "a\tb"}},
		{ID: "3", Cells: []string{"z", "c"}, Styled: [][]Span{{{Text: "\treturn", Style: Fg(Green)}}}}})
	frame := strings.Join(ta.Frame(), "\n")
	text := StripStyles(frame)
	for _, want := range []string{`x    y\x1b[31m {`, `a\tb`, "    return"} {
		if !strings.Contains(text, want) {
			t.Errorf("%q is not shown:\n%s", want, ta.screenText())
		}
	}
	if !strings.Contains(frame, nameSGR+`\x1b`) || strings.Contains(frame, "\x1b[31m") || !strings.Contains(frame, Fg(Green).Render("    return")) {
		t.Errorf("the ESC is not a red name, or the spans lost their style: %q", frame)
	}
	if got := ta.list.rows[2].Styled[0][0].Text; got != "\treturn" {
		t.Errorf("the app's spans were changed: %q", got)
	}
	ta.InputTabWidth = 2
	if text := ta.screenText(); !strings.Contains(text, "x  y") || strings.Contains(text, "x    y") {
		t.Errorf("a tab is not input_tab_width: %s", text)
	}
}

// TestControlCharsOnABlock: a name under a cursor is Red under Base, as a
// typing row's cursor on a control character.
func TestControlCharsOnABlock(t *testing.T) {
	ta := newTestApp(t)
	ta.list.SetRows([]Row{{ID: "1", Cells: []string{"a\tb"}}})
	if frame := strings.Join(ta.Frame(), "\n"); !strings.Contains(frame, nameOnBlock+`\t`) {
		t.Errorf("the cursor row's name: %q", frame)
	}
	open := "\x1b[" + On(Blue).sgr() + "m"
	if got := On(Blue).Render("x" + Escape("\t") + "y"); got != open+"x"+nameOnBlock+`\t`+"\x1b[0m"+open+"y\x1b[0m" {
		t.Errorf("a block's style goes on after the name: %q", got)
	}
}

// TestSpansStyled: where an app colours text it gives spans: each piece in
// its style, its text escaped (Row.Styled, Field.Styled, FormField.Styled,
// App.Status).
func TestSpansStyled(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(160, 40)
	green := Fg(Green)
	ta.list.SetRows([]Row{{ID: "1", Cells: []string{"x"}}, {ID: "2", Cells: []string{"Running"}, Styled: [][]Span{{{Text: "Run\tning", Style: green}}}}})
	ta.values.Fields[0].Styled = func() []Span { return []Span{{Text: "on", Style: green}} }
	ta.Status = func(int) []Span {
		return []Span{{Text: "ctx: ", Style: Fg(Overlay0)}, {Text: "prod", Style: Fg(Lavender)}}
	}
	frame := strings.Join(ta.Frame(), "\n")
	for _, want := range []string{green.Render(`Run` + Escape("\t") + `ning`), Fg(Lavender).Render("prod")} {
		if !strings.Contains(frame, want) {
			t.Errorf("not drawn in its style: %q", want)
		}
	}
	ta.press("3", "j") // the value off the cursor, in a panel with the focus: its own colour
	if frame := strings.Join(ta.Frame(), "\n"); !strings.Contains(frame, green.Render("on")) {
		t.Errorf("a value not drawn in its style")
	}
}

// TestTabWidthWithoutRun: the app's input_tab_width is the one text is
// drawn with when it is driven without Run, as in its own tests (Rules E3).
func TestTabWidthWithoutRun(t *testing.T) {
	ta := newTestApp(t)
	ta.InputTabWidth = 8
	p := &Pager{}
	p.SetLines([]string{"a\tb"})
	ta.Screen().Panels[1].Body = p
	if !strings.Contains(ta.screenText(), "a        b") {
		t.Errorf("a tab is not 8 cells: %q", ta.screenText())
	}
	ta.InputTabWidth = 0
	if !strings.Contains(ta.screenText(), "a    b") {
		t.Error("back to 4 when unset")
	}
}

// TestReadingForAppBodies: Reading draws reading content for a body of the
// app's own as the Pager does (components, control characters; kbu KB24).
func TestReadingForAppBodies(t *testing.T) {
	rows := Reading("\x1b[32mgreen\x1b[0m", "10%\r100%", "x\x1b[2Jy", "one\ntwo", "t\tt", "crlf\r")
	plain := make([]string, len(rows))
	for i, r := range rows {
		plain[i] = StripStyles(r)
	}
	if want := []string{"green", "100%", `x\x1b[2Jy`, "one", "two", "t    t", "crlf"}; strings.Join(plain, "|") != strings.Join(want, "|") {
		t.Errorf("rows: %q", plain)
	}
	if !strings.Contains(rows[0], "\x1b[32mgreen") || !strings.Contains(rows[2], Fg(Red).Render(`\x1b`)) || strings.Contains(rows[2], "\x1b[2J") {
		t.Errorf("the colour code colours, the other ESC is a red name: %q", rows[:3])
	}
	for _, r := range rows {
		if strings.ContainsAny(StripStyles(r), "\r\n\t") {
			t.Errorf("a control character left: %q", r)
		}
	}
}

// TestReadingContent: a panel's log and a viewer draw what they read as a
// terminal would: \n starts another line (a line the app gave with one in it
// too), a \r in a line leaves its last state, SGR colours, a tab is spaces,
// any other control character or escape sequence is its name in Red
// (components, control characters: components-28; note-74, note-75).
func TestReadingContent(t *testing.T) {
	lines := []string{"\x1b[32mgreen\x1b[0m", "10%\r100%", "x\x1b[2Jy", "one\ntwo", "t\tt", "crlf\r"}
	check := func(where, text, frame string) {
		t.Helper()
		for _, want := range []string{"green", "100%", `x\x1b[2Jy`, "t    t"} {
			if !strings.Contains(text, want) {
				t.Errorf("%s: %q is not shown:\n%s", where, want, text)
			}
		}
		if strings.Contains(text, "10%") || strings.Contains(text, "one\ntwo") || strings.Contains(text, `\r`) {
			t.Errorf("%s: a \\r or a \\n drawn:\n%s", where, text)
		}
		if !strings.Contains(frame, "\x1b[32mgreen") || strings.Contains(frame, "\x1b[2J") || !strings.Contains(frame, Fg(Red).Render(`\x1b`)) {
			t.Errorf("%s: the colour code colours, the other ESC is a red name", where)
		}
		var one, two int
		for i, l := range strings.Split(text, "\n") {
			if strings.Contains(l, "one") {
				one = i
			}
			if strings.Contains(l, "two") {
				two = i
			}
		}
		if two != one+1 {
			t.Errorf("%s: one and two are not on two rows: %d, %d", where, one, two)
		}
	}
	ta := newTestApp(t)
	p := &Pager{}
	p.SetLines(lines)
	ta.Screen().Panels[1].Body = p
	check("a panel's log", ta.screenText(), strings.Join(ta.Frame(), "\n"))
	if s := ta.Screen().Panels[1]; len(p.rows) != 7 || s == nil {
		t.Errorf("the log's rows: %d", len(p.rows))
	}
	ta.Viewer(&Viewer{Title: "Log", Lines: lines})
	ta.settle()
	check("a viewer", ta.screenText(), strings.Join(ta.Frame(), "\n"))
}
