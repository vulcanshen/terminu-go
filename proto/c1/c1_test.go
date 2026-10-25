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
// (tdp E5), with every popup fitting, wider, taller, and both.
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
