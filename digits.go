package terminu

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// Digits gives the digit keys 1–9 to a panel (Panel.Digits): the pages of
// what it shows, say. They are its own only on a screen where it is the one
// place with a number: there 2–9 have no panel to jump to and 1 is the panel
// itself, so no [N] on a frame stops meaning "press N to come here" (tdp K1).
type Digits struct {
	// Count is how many there are now: the digits 1 to Count work, 1–9 when
	// it is more than nine. The hint and the key reference write the range
	// from it, nothing at one or none.
	Count func() int
	// Hint is the word after the range in the panel's bottom-border hint:
	// "page" for 1–3:page.
	Hint string
	// Desc is the key reference's row for them, in its move section: "go
	// to page N".
	Desc string
	// Do runs for the digit n pressed, 1 to Count.
	Do func(n int) tea.Cmd
}

// digits is how many of the digit keys are p's own now, 0 for none: p's
// Digits while p is the one place of s with a number, signs not counting and
// a hidden place (L5) counting, as the key reference's 1–N and the footer's
// Tab|1–N count them (tdp K1), up to 9.
func (s *Screen) digits(p *Panel) int {
	d := p.Digits
	if d == nil || d.Count == nil || d.Do == nil || s.count() != 1 || s.Panels[s.numbered(1)] != p {
		return 0
	}
	return min(max(d.Count(), 0), 9)
}

// checkDigits refuses a panel's Digits where the digits cannot be its own:
// on a screen with another place that has a number, a following place (L5)
// or a Details among them, signs not counting (tdp K1); and Digits with no
// Count or no Do. Panels the app puts on a screen later are not refused:
// while another place has a number, the digits jump to the panels as ever.
func (a *App) checkDigits() error {
	for _, s := range a.screens {
		all := s.Panels
		for _, l := range s.Leads {
			all = append(all[:len(all):len(all)], l.Shows...)
		}
		for _, p := range all {
			switch {
			case p == nil || p.Digits == nil:
			case p.Digits.Count == nil || p.Digits.Do == nil:
				return fmt.Errorf("terminu: screen %s, panel %s: its Digits need a Count and a Do", s.Name, p.Label)
			case s.count() > 1:
				return fmt.Errorf("terminu: screen %s, panel %s takes the digits, and %d panels there have a number: they jump to them (Rules K1)", s.Name, p.Label, s.count())
			}
		}
	}
	return nil
}
