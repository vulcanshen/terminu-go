package main

import (
	"strings"

	terminu "github.com/vulcanshen/terminu-go"
)

// mark is the family mark the splash draws, one character a cell: L Lavender,
// G Green, a space nothing. TODO(tdp G48): a stand-in; the showcase has no
// docs/icon.svg of the family mark to draw it from.
var mark = []string{
	" LLLLLLLLLLLLLLLLLL ",
	"L                  L",
	"L  G               L",
	"L   G              L",
	"L    G             L",
	"L   G              L",
	"L  G    GGGGGG     L",
	"L                  L",
	" LLLLLLLLLLLLLLLLLL ",
}

// splash draws the mark in the middle of the screen, two columns more each
// frame from the left (Rules S5: the reveal is the app's).
func splash(frame, w, h int) ([]string, bool) {
	markW := len(mark[0])
	shown := min(2*(frame+1), markW)
	x, y := max((w-markW)/2, 0), max((h-len(mark))/2, 0)
	out := make([]string, h)
	for i, row := range mark {
		if y+i >= h {
			break
		}
		var b strings.Builder
		for _, c := range row[:shown] {
			switch c {
			case 'L':
				b.WriteString(terminu.Fg(terminu.Lavender).Render("█"))
			case 'G':
				b.WriteString(terminu.Fg(terminu.Green).Render("█"))
			default:
				b.WriteByte(' ')
			}
		}
		out[y+i] = strings.Repeat(" ", x) + b.String()
	}
	return out, shown == markW
}
