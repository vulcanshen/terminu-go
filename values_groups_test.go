package terminu

import (
	"slices"
	"strings"
	"testing"
)

// groupValues are the fields of the example in tdp panel.12 with groups, a
// header set as well, wider than the labels.
func groupValues() *Values {
	val := func(s string) func() string { return func() string { return s } }
	return &Values{Header: [2]string{"Property_of_the_setting", "Value"}, Fields: []*Field{
		{Label: "Pin", Group: "Lock", Value: val(""), Empty: "not set"},
		{Label: "Pin_prompt_timeout", Group: "Lock", Value: val("30")},
		{Label: "Profile", Group: "Display", Value: val("clock")},
		{Label: "Show_status", Group: "Display", Value: val("on")},
	}}
}

// valueLines are the rows as text, the cursor's arrow taken off.
func valueLines(out []string) []string {
	got := plainLines(out)
	for i, l := range got {
		got[i] = strings.TrimRight(strings.TrimSuffix(l, glyphHard), " ")
	}
	return got
}

// TestPanel12Groups: each group under a title in Overlay0, upper case, not
// bold, no glyph, one cell from the frame; its rows two cells further in, the
// values in one column across the groups; no blank row or separator between
// groups; the first row a title; no header (tdp panel.12, panel-161 to 164,
// 168, 171).
func TestPanel12Groups(t *testing.T) {
	v := groupValues()
	out := v.Draw(40, 8, true)
	for i, line := range out {
		if Width(line) != 40 {
			t.Fatalf("row %d is %d wide", i, Width(line))
		}
	}
	label := func(s string) string { return "   " + s + strings.Repeat(" ", 18-len(s)) + "  " }
	want := []string{" LOCK", label("PIN") + "not set", label("PIN_PROMPT_TIMEOUT") + "30",
		" DISPLAY", label("PROFILE") + "clock", label("SHOW_STATUS") + "on", "", ""}
	if got := valueLines(out); !slices.Equal(got, want) {
		t.Errorf("rows:\n%q\nwant\n%q", got, want)
	}
	for _, i := range []int{0, 3} {
		title := strings.TrimSpace(StripStyles(out[i]))
		if !strings.Contains(out[i], Fg(Overlay0).Render(title)) || strings.Contains(out[i], Fg(Blue).Render(title)) {
			t.Errorf("group title %q is not plain Overlay0: %q", title, out[i])
		}
	}
	if !strings.Contains(out[2], Fg(Blue).Render(Pad("PIN_PROMPT_TIMEOUT", 18))) {
		t.Errorf("a label in a group is not Blue: %q", out[2])
	}
	if strings.Contains(strings.Join(plainLines(out), "\n"), "PROPERTY") {
		t.Error("a panel with groups draws a header")
	}
	// A value too long is cut two cells sooner, the arrow at the edge.
	long := strings.Repeat("x", 40)
	v.Fields[0].Value = func() string { return long }
	if row := plainLines(v.Draw(40, 8, true))[1]; !strings.HasSuffix(row, "x…"+glyphHard) {
		t.Errorf("a long value in a group: %q", row)
	}
	// Without groups, the same fields have the header, one cell from the frame.
	for _, f := range v.Fields {
		f.Group = ""
	}
	got := valueLines(v.Draw(40, 8, true))
	if !strings.HasPrefix(got[0], " PROPERTY") || !strings.HasPrefix(got[1], " PIN ") {
		t.Errorf("without groups: %q", got[:2])
	}
}

// TestPanel12GroupsOrder: a group's fields are drawn together, the groups in
// the order they first appear (tdp panel.12, as panel.11).
func TestPanel12GroupsOrder(t *testing.T) {
	v := groupValues()
	v.Fields[1], v.Fields[2] = v.Fields[2], v.Fields[1] // Lock, Display, Lock, Display
	got := valueLines(v.Draw(40, 8, true))[:6]
	for i, w := range []string{"LOCK", "PIN", "PIN_PROMPT_TIMEOUT", "DISPLAY", "PROFILE", "SHOW_STATUS"} {
		if f := strings.Fields(got[i]); len(f) == 0 || f[0] != w {
			t.Errorf("row %d is %q, want %s", i, got[i], w)
		}
	}
	v.Key("j", keyMsg("j"))
	if c := v.Current(); c == nil || c.Label != "Pin_prompt_timeout" {
		t.Errorf("j from the first field goes to %v, want the next drawn", c)
	}
}

// TestPanel12GroupsCursor: the focus skips the titles; N of M counts fields
// only (tdp panel.12, panel-165).
func TestPanel12GroupsCursor(t *testing.T) {
	v := groupValues()
	v.Fields[1], v.Fields[2] = v.Fields[2], v.Fields[1] // drawn as before
	v.Draw(40, 3, true)
	steps := []struct{ key, want string }{
		{"j", "Pin_prompt_timeout"}, {"j", "Profile"}, {"k", "Pin_prompt_timeout"},
		{"G", "Show_status"}, {"gg", "Pin"}, {"k", "Show_status"}, {"j", "Pin"},
	}
	for _, s := range steps {
		v.Key(s.key, keyMsg(s.key))
		v.Draw(40, 3, true)
		if c := v.Current(); c == nil || c.Label != s.want {
			t.Fatalf("%s: on %v, want %s", s.key, c, s.want)
		}
	}
	v.Key("j", keyMsg("j"))
	v.Key("j", keyMsg("j"))
	v.Draw(40, 3, true)
	if got := v.Scroll(); got != "3 of 4" {
		t.Errorf("scroll position %q, want 3 of 4 (fields, no titles)", got)
	}
}

// TestPanel12GroupsScrollKeepsTitle: when scrolling puts the focus on a
// group's first field, its title stays in sight; a radio's options are one
// field; when the title and the focus do not both fit, the focus is shown
// (tdp panel.12, panel-166).
func TestPanel12GroupsScrollKeepsTitle(t *testing.T) {
	v := groupValues()
	v.Draw(40, 3, true)
	for _, k := range []string{"G", "k", "k", "k"} { // up to the first field
		v.Key(k, keyMsg(k))
		v.Draw(40, 3, true)
	}
	if got := valueLines(v.Draw(40, 3, true)); got[0] != " LOCK" {
		t.Errorf("up to the first field, rows %q", got)
	}
	v.Key("j", keyMsg("j"))
	v.Key("j", keyMsg("j")) // down to Profile
	if got := valueLines(v.Draw(40, 3, true)); !strings.Contains(got[1], "DISPLAY") || !strings.Contains(got[2], "PROFILE") {
		t.Errorf("down to a group's first field, rows %q", got)
	}

	val := func(s string) func() string { return func() string { return s } }
	r := &Values{Fields: []*Field{
		{Label: "Left", Group: "Keys", Value: val("h")},
		{Label: "Right", Group: "Keys", Value: val("l")},
		{Label: "Speed", Group: "Play", Options: []string{"slow", "normal", "fast"}},
		{Label: "Ghost", Group: "Play", Value: val("on")},
	}}
	r.Draw(40, 4, true)
	r.Key("G", keyMsg("G"))
	r.Draw(40, 4, true)
	r.Key("k", keyMsg("k")) // on fast, the last option of Play's first field
	if got := valueLines(r.Draw(40, 4, true)); got[0] != " PLAY" || !strings.Contains(got[3], "fast") {
		t.Errorf("on a radio option of a group's first field, rows %q", got)
	}
	r.Draw(40, 3, true)
	r.Key("G", keyMsg("G"))
	r.Draw(40, 3, true)
	r.Key("k", keyMsg("k"))
	if got := valueLines(r.Draw(40, 3, true)); !slices.ContainsFunc(got, func(l string) bool { return strings.Contains(l, "fast") }) {
		t.Errorf("the title and the focus do not fit: the focus is not shown: %q", got)
	}
}

// TestPanel12GroupsEmptyGroup: a group with no fields has no title (tdp
// panel.12, panel-167).
func TestPanel12GroupsEmptyGroup(t *testing.T) {
	v := groupValues()
	v.Fields = v.Fields[2:]
	got := valueLines(v.Draw(40, 6, true))
	if got[0] != " DISPLAY" || strings.Contains(strings.Join(got, "\n"), "LOCK") {
		t.Errorf("rows %q", got)
	}
}

// TestPanel12GroupsKeyReference: with groups, the key reference's "what each
// row is" stays one section, in the panel's order, with no group titles; the
// group titles are dimmed with the panel (tdp panel.12, T2, panel-169).
func TestPanel12GroupsKeyReference(t *testing.T) {
	ta := newTestApp(t)
	ta.values.Fields = []*Field{
		{Label: "Name", Group: "General", Desc: "a name", Value: func() string { return "web" }},
		{Label: "Port", Group: "Network", Desc: "a port", Value: func() string { return "80" }},
		{Label: "Owner", Group: "General", Desc: "an owner", Value: func() string { return "me" }},
	}
	ta.press("3", "?")
	_, plain := covEPopup(ta)
	keys := covEKeys(plain)
	if !slices.Equal(keys["WHAT EACH ROW IS"], []string{"NAME", "OWNER", "PORT"}) {
		t.Errorf("the section: %v", keys)
	}
	text := strings.Join(plain, "\n")
	if strings.Contains(text, "GENERAL") || strings.Contains(text, "NETWORK") {
		t.Errorf("a group title in the key reference:\n%s", text)
	}
	ta.press("Esc")
	if row := ta.rowWith("GENERAL"); !strings.Contains(row, Fg(Overlay0).Render("GENERAL")) {
		t.Errorf("focused, the title: %q", row)
	}
	ta.press("1")
	if row, on := ta.rowWith("GENERAL"), strings.SplitN(Fg(DimColor(Overlay0)).Render("x"), "x", 2)[0]; !strings.Contains(row, on+"GENERAL") {
		t.Errorf("unfocused, the title is not dimmed: %q", row)
	}
}
