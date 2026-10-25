package terminu

import (
	"math"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Slider is a number in a range, shown as a track (tdp slider).
// It has no keys of its own: Enter opens a select of the numbers from Bottom
// to Top, Step apart.
type Slider struct {
	Title, Glyph string
	// Subject is the one thing it acts on, kept as it is after ": " in the
	// title: "EDIT: filu" (tdp popup).
	Subject     string
	Bottom, Top float64
	// Step is the distance between two numbers: (Top − Bottom) / 10 when 0.
	Step  float64
	Value float64
	// Done gets the new number. It is not called when it did not change.
	Done func(float64) tea.Cmd
}

func (s Slider) typed() bool { return false }

func (s Slider) step() float64 {
	if s.Step > 0 {
		return s.Step
	}
	return (s.Top - s.Bottom) / 10
}

// numbers is every number the slider takes, rounded to the step's decimals.
func (s Slider) numbers() []float64 {
	step := s.step()
	if step <= 0 || s.Top < s.Bottom {
		return []float64{s.Bottom}
	}
	dec := 0
	if i := strings.IndexByte(formatNum(step), '.'); i >= 0 {
		dec = len(formatNum(step)) - i - 1
	}
	scale := math.Pow(10, float64(dec))
	n := int(math.Floor((s.Top-s.Bottom)/step+1e-9)) + 1
	out := make([]float64, n)
	for i := range out {
		out[i] = math.Round((s.Bottom+float64(i)*step)*scale) / scale
	}
	return out
}

func (s Slider) open(a *App, written func()) tea.Cmd {
	nums := s.numbers()
	opts := make([]string, len(nums))
	sel := 0
	for i, n := range nums {
		opts[i] = formatNum(n)
		if math.Abs(n-s.Value) < math.Abs(nums[sel]-s.Value) {
			sel = i
		}
	}
	return Select{
		Title: s.Title, Subject: s.Subject, Glyph: s.Glyph, Options: opts, Selected: sel,
		Done: func(i int) tea.Cmd {
			if s.Done == nil {
				return nil
			}
			return s.Done(nums[i])
		},
	}.open(a, written)
}

// SliderSpans is a value on a track of width cells in color, then the number
// (tdp slider): a heavy line with a dot where the value is. For
// Styled, the number in Value.
func SliderSpans(value, bottom, top float64, width int, color Color) []Span {
	at := 0
	if top > bottom && width > 1 {
		at = int(math.Round((value - bottom) / (top - bottom) * float64(width-1)))
		at = min(max(at, 0), width-1)
	}
	track := Repeat("━", at) + "●" + Repeat("━", max(width-at-1, 0))
	return []Span{{Text: track, Style: Fg(color)}, {Text: " "}, {Text: strconv.FormatFloat(value, 'f', -1, 64), Style: Fg(Text)}}
}
