package terminu

import (
	"strings"
	"testing"
)

// TestPanel38PaddedText: blanks padding a fixed-width slot at the end of a
// capsule's text are part of the text, not a space after it: kept as
// written, counted in the width, filled on the chosen part and on Mantle on
// the others, in a panel's tabs and title, a screen chip and a popup's tabs
// alike; the part after it does not move when the slot fills (tdp panel.4,
// L2; panel-38, L2-2; from kbu, the user).
func TestPanel38PaddedText(t *testing.T) {
	arrow := "\U0000F432" // nf-oct-arrow_right
	padded := "RELATIVES " + arrow + " 1 "
	chosen, other := covCStyle{Fg: Base, Bg: Blue, Bold: true}, covCStyle{Fg: Blue, Bg: Mantle}
	blank := func(what, row, text string, want covCStyle) {
		t.Helper()
		s := covCSpan(row, text)
		if s == nil {
			t.Fatalf("%s: %q is not drawn: %q", what, text, covCPlain(row))
		}
		if c := s[len(s)-1]; c.Ch != " " || c.St != want {
			t.Errorf("%s: the blank after the digit is %q %+v, want a blank %+v", what, c.Ch, c.St, want)
		}
	}

	// A panel's tab, chosen and not; the next tab stays put from 1 to 10.
	p := &Panel{Label: "Pods", Tabs: []string{"Relatives " + arrow + " 1 ", "Events"}, Tab: 1}
	row := drawChain(p.capsules(1), Blue, 80)
	if !strings.Contains(covCPlain(row), padded+glyphSlantJoin+"EVENTS") {
		t.Errorf("chosen tab: %q", covCPlain(row))
	}
	blank("chosen tab", row, padded, chosen)
	events := covCSpan(row, "EVENTS")[0].Col
	p.Tabs[0] = "Relatives " + arrow + " 10"
	if got := covCSpan(drawChain(p.capsules(1), Blue, 80), "EVENTS")[0].Col; got != events {
		t.Errorf("the next tab moved from 1 to 10: %d→%d", events, got)
	}
	p.Tabs[0], p.Tab = "Relatives "+arrow+" 1 ", 2
	row = drawChain(p.capsules(1), Blue, 80)
	if !strings.Contains(covCPlain(row), padded+glyphSlantLeft+"EVENTS") {
		t.Errorf("tab not chosen: %q", covCPlain(row))
	}
	blank("tab not chosen", row, padded, other)

	// A panel's title.
	row = drawChain((&Panel{Label: "Level 1 "}).capsules(1), Blue, 80)
	if got := covCPlain(row); got != glyphSlantLeft+"[1] LEVEL 1 "+glyphHard {
		t.Errorf("title: %q", got)
	}
	blank("title", row, "[1] LEVEL 1 ", chosen)

	// A screen chip, chosen and not.
	a := New("x")
	a.AddScreen(&Screen{Name: "Level 1 ", Key: "L", Panels: []*Panel{{Label: "A", Body: &Pager{}}}})
	a.AddScreen(&Screen{Name: "Other", Key: "O", Panels: []*Panel{{Label: "B", Body: &Pager{}}}})
	row = a.chipRow(80)
	if !strings.Contains(covCPlain(row), "[L]EVEL 1 "+glyphSlantJoin+"[O]THER") {
		t.Errorf("chips: %q", covCPlain(row))
	}
	blank("chosen chip", row, "[L]EVEL 1 ", chosen)
	a.cur = 1
	blank("chip not chosen", a.chipRow(80), "[L]EVEL 1 ", other)

	// A popup's tab.
	ta := newTestApp(t)
	ta.Viewer(&Viewer{Title: "DevTools", Tabs: []string{"Level 1 ", "Network"}, Lines: []string{"x"}})
	ta.settle()
	y, _ := covCFind(t, ta.Frame(), "LEVEL 1 ")
	blank("popup tab", ta.Frame()[y], "LEVEL 1 ", covCStyle{Fg: Base, Bg: LayerColor(1), Bold: true})
}
