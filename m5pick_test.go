package terminu

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestM5AppPicks: the app may mark another appearance of its key's letter
// than the first, writing the brackets itself ("Stage [a]ll", "Mark
// [r]esolved"); the first stays the default. Menus and chips leave it as it
// is, and the key reference writes it without them, as it writes every
// label (tdp M5, M5-1; from gittu).
func TestM5AppPicks(t *testing.T) {
	for _, c := range []struct{ key, label, want string }{
		{"a", "Stage all", "St[a]ge all"},
		{"a", "Stage [a]ll", "Stage [a]ll"},
		{"r", "Mark [r]esolved", "Mark [r]esolved"},
		{"P", "Ex[P]ort", "Ex[P]ort"},
	} {
		if got := MarkLabel(c.key, c.label); got != c.want {
			t.Errorf("MarkLabel(%q, %q) = %q, want %q", c.key, c.label, got, c.want)
		}
	}
	for _, c := range []struct{ key, label, want string }{
		{"a", "Stage [a]ll", "Stage all"},
		{"r", "Mark [r]esolved", "Mark resolved"},
		{"x", "[x] Reset", "Reset"},
		{"r", "Rename", "Rename"},
		{"", "Refresh", "Refresh"},
	} {
		if got := plainLabel(Action{Key: c.key, Label: c.label}); got != c.want {
			t.Errorf("plainLabel(%q, %q) = %q, want %q", c.key, c.label, got, c.want)
		}
	}

	ta := newTestApp(t)
	ta.pods.Actions = append(ta.pods.Actions,
		Action{Key: "a", Label: "Stage [a]ll", Desc: "every change", Scope: PanelScope, Do: func() tea.Cmd { return nil }})
	ta.AddScreen(&Screen{Name: "Stats [S]ummary", Key: "S", Panels: []*Panel{{Label: "Stats", Body: &Pager{}}}})
	ta.Global = nil // its S is the screen's now
	ta.press("Space")
	if text := ta.screenText(); !strings.Contains(text, "Stage [a]ll") || strings.Contains(text, "St[a]ge") {
		t.Errorf("the Space menu does not keep the app's bracket:\n%s", text)
	}
	ta.press("Esc", "?")
	row := ""
	for _, l := range strings.Split(ta.screenText(), "\n") {
		if strings.Contains(l, "every change") {
			row = l
		}
	}
	if !strings.Contains(row, "Stage all") || strings.Contains(row, "[a]") {
		t.Errorf("the key reference row: %q", row)
	}
	ta.press("Esc")
	if text := ta.screenText(); !strings.Contains(text, "STATS [S]UMMARY") || strings.Contains(text, "[S]TATS") {
		t.Errorf("the chip does not keep the app's bracket:\n%s", strings.Split(text, "\n")[0])
	}
}
