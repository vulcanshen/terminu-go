package terminu

// A group of checkbox options (two or more) carries a first row of its own,
// Select all (tdp checkbox.3): on a form and a value
// panel on the label's row, the options under it; first in a checkbox
// popup's list, hidden while something is typed. It is not an option: no
// value of its own, not counted among the five drawn in place. Ticking it
// ticks every option, the unavailable ones too, unticking unticks them all;
// it is ticked when every option is.

// optAll is the Select all row where an option's index goes.
const optAll = -2

const selectAllLabel = "Select all"

// optIndent is how much further in a group's options are than Select all, as
// a list group's items are than its title (tdp panel).
const optIndent = "  "

// allChecked reports every option of n checked, those skip leaves aside not
// counted; false with none to count.
func allChecked(n int, checked []bool, skip func(int) bool) bool {
	counted := false
	for i := range n {
		if skip != nil && skip(i) {
			continue
		}
		if i >= len(checked) || !checked[i] {
			return false
		}
		counted = true
	}
	return counted
}

// checkAll ticks every option of n, or unticks them all, but those skip
// leaves as they are.
func checkAll(n int, checked []bool, on bool, skip func(int) bool) []bool {
	for len(checked) < n {
		checked = append(checked, false)
	}
	for i := range n {
		if skip == nil || !skip(i) {
			checked[i] = on
		}
	}
	return checked
}

// selectAllText is the Select all row as drawn: ticked in Green as an
// option is.
func selectAllText(on bool) string {
	if on {
		return Fg(Green).Render(glyphMarked + " " + selectAllLabel)
	}
	return Fg(Text).Render(glyphUnchecked + " " + selectAllLabel)
}
