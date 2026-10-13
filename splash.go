package terminu

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Splash draws frame n (from 0) of the family's easter egg in w × h cells
// (Rules S). done reports that the reveal is over; the last frame then stays
// until a key. NewSplash makes the one Rules S5 describes.
type Splash func(frame, w, h int) (lines []string, done bool)

type splashMsg struct{ id int }

// splashTickTime is one tick of the reveal (Rules S5).
const splashTickTime = 10 * time.Millisecond

func splashTick(id int) tea.Cmd {
	return tea.Tick(splashTickTime, func(time.Time) tea.Msg { return splashMsg{id} })
}

// The mark's colours (Rules S5): the U frame, the same in every family app;
// the letters in it; the squares under them, Surface0.
var (
	markFrame = Hex("#205090")
	markGold  = Hex("#f2b753")
	markBack  = Hex("#313244")
)

const (
	// markCols × markRows is the mark as drawn: the svg's 25 columns, and its
	// 19 rows with one more of background above and below (svg rows 2–22).
	markCols, markRows = 25, 21
	markTop            = 2 // the svg row drawn first
	markSquare         = "\U0000F0C8"

	// The reveal's holds, in ticks: between stages, then before each batch
	// of the text.
	stageHold = 25 // 250ms
	textHold  = 40 // 400ms after the mark is complete
	textHold2 = 50 // 500ms more

	splashAuthor = "vulcan.shen.2304@gmail.com"
)

type markCell struct{ col, row int }

// NewSplash is the family's splash (Rules S5): the mark in icon, the app's
// docs/icon.svg, cell for cell, and under it the app's name (its command's),
// its version as `<app> version` prints it and a one-line intro, then the
// family's author and the hint.
//
// icon is read as S5 describes it: a 250 × 250 viewBox of 10 × 10 rects, the
// U frame in #205090 and the letters in #f2b753 within rows 3–21. The letters
// are revealed one at a time: each XML comment starts the next one, so the
// svg lists them in the order of the app's name with a comment before each
// (`<!-- K -->`). Without comments all the gold is one letter.
func NewSplash(icon []byte, name, version, intro string) (Splash, error) {
	frame, letters, err := parseMark(icon)
	if err != nil {
		return nil, err
	}

	// at is the tick a cell turns from background to its colour.
	var colour [markRows][markCols]Color
	var at [markRows][markCols]int
	for r := range at {
		for c := range at[r] {
			at[r][c] = -1
		}
	}
	t := markRows - 1 // the background: a row a tick, top to bottom
	rng := rand.New(rand.NewPCG(5, 5))
	for _, cells := range letters { // a letter a stage, 2 cells a tick, scattered
		rng.Shuffle(len(cells), func(i, j int) { cells[i], cells[j] = cells[j], cells[i] })
		start := t + stageHold
		for i, c := range cells {
			colour[c.row][c.col], at[c.row][c.col] = markGold, start+i/2
		}
		t = start + (len(cells)-1)/2
	}
	// The U frame rises from the bottom, 3 cells a tick.
	// TODO(tdp G69): the order within a row; left to right for now.
	slices.SortFunc(frame, func(a, b markCell) int {
		if a.row != b.row {
			return b.row - a.row
		}
		return a.col - b.col
	})
	start := t + stageHold
	for i, c := range frame {
		colour[c.row][c.col], at[c.row][c.col] = markFrame, start+i/3
	}
	t = start + (len(frame)-1)/3
	text, text2 := t+textHold, t+textHold+textHold2

	under := []struct {
		s     string
		style Style
		at    int
	}{
		{"", Style{}, 0}, // a blank row under the mark
		{name, Fg(Blue).Bolded(), text},
		{version, Fg(Blue), text},
		{intro, Fg(Text), text},
		{"", Style{}, 0},
		{"developed by", Fg(Overlay0), text2},
		{splashAuthor, Fg(markGold), text2},
		{"", Style{}, 0},
		{"Press any key to close", Fg(Overlay0), text2},
	}
	markW := 2 * markCols
	// TODO(tdp G70): a line wider than the mark; centred on the same middle,
	// sticking out on both sides, for now.
	blockW := markW
	for _, u := range under {
		blockW = max(blockW, Width(u.s))
	}

	return func(f, w, h int) ([]string, bool) {
		square := markSquare
		if IconWidth() == 1 {
			square += " " // a square takes two columns (E5)
		}
		// TODO(tdp G67): where the splash sits; in the middle for now.
		// TODO(tdp G68): a terminal smaller than the splash; cut at the right
		// and the bottom for now.
		x := max((w-blockW)/2, 0)
		y := max((h-markRows-len(under))/2, 0)
		lines := make([]string, y, y+markRows+len(under))
		markPad := strings.Repeat(" ", x+(blockW-markW)/2)
		for r := range markRows {
			if f < r {
				lines = append(lines, "")
				continue
			}
			var b strings.Builder
			b.WriteString(markPad)
			run, n := markBack, 0
			for c := range markCols + 1 {
				cur := markBack
				switch {
				case c == markCols:
					cur = Color{} // ends the last run
				case at[r][c] >= 0 && f >= at[r][c]:
					cur = colour[r][c]
				}
				if cur == run {
					n++
					continue
				}
				b.WriteString(Fg(run).Render(strings.Repeat(square, n)))
				run, n = cur, 1
			}
			lines = append(lines, b.String())
		}
		pad := strings.Repeat(" ", x)
		for _, u := range under {
			if u.s == "" || f < u.at {
				lines = append(lines, "")
				continue
			}
			lines = append(lines, pad+Center(u.style.Render(u.s), blockW))
		}
		return lines, f >= text2
	}, nil
}

// parseMark reads the cells of a family mark (Rules S5): the U frame's, and
// each letter's in the order the svg lists them.
func parseMark(icon []byte) (frame []markCell, letters [][]markCell, err error) {
	fail := func(format string, a ...any) error {
		return fmt.Errorf("terminu: the splash's icon: "+format, a...)
	}
	d := xml.NewDecoder(bytes.NewReader(icon))
	next := true // the next gold cell starts a letter
	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fail("%v", err)
		}
		switch t := tok.(type) {
		case xml.Comment:
			next = true
		case xml.StartElement:
			if t.Name.Local != "rect" {
				continue
			}
			attr := map[string]string{}
			for _, a := range t.Attr {
				attr[a.Name.Local] = a.Value
			}
			fill := strings.ToLower(attr["fill"])
			if fill == "#000000" {
				continue // the background
			}
			x, errX := strconv.Atoi(attr["x"])
			y, errY := strconv.Atoi(attr["y"])
			if errX != nil || errY != nil || attr["width"] != "10" || attr["height"] != "10" ||
				x%10 != 0 || y%10 != 0 || x < 0 || x > 240 || y < 30 || y > 210 {
				return nil, nil, fail("a %s cell not on the 25 × 19 grid at x 0–240, y 30–210 (x %q, y %q)", fill, attr["x"], attr["y"])
			}
			c := markCell{col: x / 10, row: y/10 - markTop}
			switch fill {
			case "#205090":
				frame = append(frame, c)
			case "#f2b753":
				if next {
					letters, next = append(letters, nil), false
				}
				letters[len(letters)-1] = append(letters[len(letters)-1], c)
			default:
				return nil, nil, fail("a cell in %s; the mark has only #205090 and #f2b753", fill)
			}
		}
	}
	if len(frame) == 0 || len(letters) == 0 {
		return nil, nil, fail("no U frame (#205090) or no letters (#f2b753)")
	}
	return frame, letters, nil
}

// takesV reports that p gave V to an action of its own: the splash does not
// open there (Rules S1).
func (p *Panel) takesV() bool {
	for _, act := range p.Actions {
		if act.Key == "V" {
			return true
		}
	}
	if b, ok := p.Body.(interface{ Actions() []Action }); ok {
		for _, act := range b.Actions() {
			if act.Key == "V" {
				return true
			}
		}
	}
	if p.mode != nil {
		for _, k := range p.mode.Keys {
			if k.takes("V") {
				return true
			}
		}
	}
	return false
}

// openSplash opens the splash: only from a panel, only on V (Rules S4).
func (a *App) openSplash() tea.Cmd {
	a.splashOn, a.splashFrame = true, 0
	a.splashID++
	return splashTick(a.splashID)
}

// stepSplash moves the reveal on by one tick until it is over.
func (a *App) stepSplash(msg splashMsg) tea.Cmd {
	if !a.splashOn || msg.id != a.splashID {
		return nil
	}
	if _, done := a.Splash(a.splashFrame, a.w, a.h); done {
		return nil
	}
	a.splashFrame++
	return splashTick(a.splashID)
}

// checkSplash refuses an app the splash cannot be opened in (Rules S1): none
// given, a global hotkey on V, or every panel giving V away.
func (a *App) checkSplash() error {
	if a.Splash == nil {
		return errors.New("terminu: no Splash (Rules S1: every terminu app has one)")
	}
	for _, act := range a.Global {
		if act.Key == "V" {
			return errors.New("terminu: global action " + act.Label + " takes V, kept for the splash (Rules S1)")
		}
	}
	for _, s := range a.screens {
		for _, p := range s.Panels {
			if !p.takesV() {
				return nil
			}
		}
	}
	return errors.New("terminu: every panel takes V; at least one must keep it for the splash (Rules S1)")
}
