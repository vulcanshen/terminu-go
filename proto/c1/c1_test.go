package main

import (
	"fmt"
	"testing"
)

var sizes = [][2]int{{80, 40}, {120, 40}, {100, 30}, {60, 20}, {30, 12}}

// l4Failures draws every popup, in every oversize, at every size, and returns
// each line that is not exactly the terminal width (rules L4), plus screens
// that do not have exactly h lines.
func l4Failures(t *testing.T, draw func(w, h int, k popupKind, o oversize) []string) []string {
	t.Helper()
	var bad []string
	for _, s := range sizes {
		w, h := s[0], s[1]
		for k := range popupKind(len(popupNames)) {
			for o := range oversize(len(oversizeNames)) {
				name := fmt.Sprintf("%dx%d %s %s", w, h, popupNames[k], oversizeNames[o])
				lines := draw(w, h, k, o)
				if len(lines) != h {
					bad = append(bad, fmt.Sprintf("%s: %d lines", name, len(lines)))
				}
				for i, l := range lines {
					if d := dispWidth(l); d != w {
						bad = append(bad, fmt.Sprintf("%s: line %d is %d wide", name, i, d))
					}
				}
			}
		}
	}
	return bad
}

func withIconWidth(t *testing.T, n int) {
	t.Helper()
	old := iconWidth
	iconWidth = n
	t.Cleanup(func() { iconWidth = old })
}

// TestC1E5L4 is the L4 screen test run twice, icon one cell and two cells
// (E5), with every popup fitting, wider, taller, and both.
func TestC1E5L4(t *testing.T) {
	for _, n := range []int{1, 2} {
		t.Run(fmt.Sprintf("icon%d", n), func(t *testing.T) {
			withIconWidth(t, n)
			for _, b := range l4Failures(t, e5Screen) {
				t.Error(b)
			}
		})
	}
}

// TestC1LipglossL4 records the finding rather than a goal: lipgloss v2's own
// layout and compositing hold L4 when an icon is one cell, and break it when
// an icon is two.
func TestC1LipglossL4(t *testing.T) {
	withIconWidth(t, 1)
	if bad := l4Failures(t, lgScreen); len(bad) > 0 {
		t.Errorf("icon width 1: %d lines off, first: %s", len(bad), bad[0])
	}
	withIconWidth(t, 2)
	bad := l4Failures(t, lgScreen)
	if len(bad) == 0 {
		t.Fatal("icon width 2: expected lipgloss to misplace frames, none found")
	}
	t.Logf("icon width 2: %d lines are not the terminal width; e.g. %s", len(bad), bad[0])
}
