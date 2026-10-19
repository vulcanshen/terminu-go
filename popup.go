package terminu

import (
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// popupView is one kind of popup's content and keys.
type popupView interface {
	// measure is the popup's outer size on a sw × sh screen, fixed when it
	// opens (Rules F7).
	measure(sw, sh int) (w, h int)
	// draw returns h lines of w cells, the frame included, in the layer
	// colour.
	draw(w, h int, layer Color) []string
	// key handles a key while the popup is the top layer and reports whether
	// it did; the app does the core keys it leaves (Rules K1).
	key(a *App, key string, msg tea.KeyMsg) (bool, tea.Cmd)
	// surface is the popup's name in its key reference's title.
	surface() string
	// keyRef is what its key reference lists: only its own keys (Rules K6).
	keyRef() []refSection
	// typing reports the input state: ? and q are characters (Rules K8).
	typing() bool
}

// animation (Rules F2, components/layout/popup): a line grows from the middle
// (4 frames), holds (1), then the popup opens up and down (4); 32ms a frame.
const (
	frameTime  = 32 * time.Millisecond
	openFrames = 9
)

type animMsg struct{}

func animTick() tea.Cmd { return tea.Tick(frameTime, func(time.Time) tea.Msg { return animMsg{} }) }

// layer is one popup on the stack.
type layer struct {
	view     popupView
	w, h     int
	x, y     int
	frame    int // 1–9 while opening, 9 when open, counting down while closing
	closing  bool
	toast    bool
	loading  bool  // it was loading at the last update
	answered bool  // closed on its answer: not cancelled (OnCancel)
	shade    Color // the layer colour it closes in: the one it had open
	// below runs when the layer is gone, the one under it on top again.
	below func() tea.Cmd
}

func (l *layer) open() bool { return !l.closing && l.frame >= openFrames }

// place centres the layer on a sw × sh screen.
func (l *layer) place(sw, sh int) {
	if v, ok := l.view.(interface{ at(sw, sh int) (int, int) }); ok {
		l.x, l.y = v.at(sw, sh) // a terminal popup has its own place (Rules F7)
		return
	}
	l.x, l.y = (sw-l.w)/2, (sh-l.h)/2
}

// drawFrame is the layer as it looks in its animation frame: nil when there
// is nothing to draw yet; rows are relative to the layer's top.
func (l *layer) drawFrame(full []string, color Color) (rows []string, first, x int) {
	f := l.frame
	switch {
	case f <= 0:
		return nil, 0, 0
	case f <= 5:
		// The line, from the middle out: a quarter of the width a frame.
		n := min(f, 4)
		width := max((l.w*n+3)/4, 1)
		if width%2 != l.w%2 && width < l.w {
			width++
		}
		return []string{Fg(color).Render(Repeat("━", width))}, l.h / 2, (l.w - width) / 2
	case f < openFrames:
		// The rows around the middle: a quarter of the height a frame.
		n := max((l.h*(f-5)+3)/4, 1)
		start := (l.h - n) / 2
		return full[start : start+n], start, 0
	}
	return full, 0, 0
}

// fitRows keeps a popup drawn on a screen too short for it to its height, so
// the footer stays in sight (Rules F7): the rows blank inside the frame go
// first, from the top, then the rows above the bottom border.
func fitRows(rows []string, h int) []string {
	if len(rows) <= h {
		return rows
	}
	rows = slices.Clone(rows)
	for i := 1; len(rows) > h && i < len(rows)-1; {
		if w := Width(rows[i]); strings.TrimSpace(StripStyles(Cut(rows[i], 1, w-1))) == "" {
			rows = slices.Delete(rows, i, i+1)
			continue
		}
		i++
	}
	for len(rows) > max(h, 2) {
		rows = slices.Delete(rows, len(rows)-2, len(rows)-1)
	}
	return rows
}

// refSection is one section of a key reference: a title (none for the first
// of a popup's) and its rows.
type refSection struct {
	title string
	rows  []KeyRow
}

// popupBox draws a popup frame (components/layout/popup): heavy line in
// color, the title capsule one segment from the corner, the hint after one
// segment, the scroll position before the last; one blank row above and
// below the body, which comes in already as wide as the inside.
type popupBox struct {
	color   Color
	glyph   string
	title   string // as given: the caller has made it upper case where it should be
	tabs    []Capsule
	loading bool
	mode    string
	hint    []KeyHint
	scroll  string
	bare    bool // no padding at all (terminal)
}

func (p popupBox) draw(w int, body []string) []string {
	b := box{width: w, lines: heavyLines, color: p.color, hintKey: Fg(Blue), mode: p.mode}
	room := w - 4 - modeRoom(p.mode)
	b.title = titleCapsule(p.glyph, p.title, p.color, room)
	if len(p.tabs) > 0 {
		// The tabs follow one line segment after the title (layout/popup).
		left := room - Width(b.title) - 1
		if left > 2 {
			b.title += Fg(p.color).Render("━") + drawChain(p.tabs, p.color, left)
		}
	}
	if p.loading {
		b.after = Fg(p.color).Bolded().Render(loadingFrame())
	}
	b.hint, b.scroll = p.hint, p.scroll
	inner := w - 2
	if p.bare {
		b.body = body
		return b.draw()
	}
	rows := make([]string, 0, len(body)+2)
	rows = append(rows, Repeat(" ", inner))
	rows = append(rows, body...)
	rows = append(rows, Repeat(" ", inner))
	b.body = rows
	return b.draw()
}

// padRow is one row of a popup's body: one cell from the frame on either side.
func padRow(s string, inner int) string {
	return " " + Fit(s, inner-2) + " "
}

// cursorRow is a list row under the cursor: a block of color from frame to
// frame, the text one cell in, the right-most cell an arrow (layout/popup).
func cursorRow(plain string, inner int, block Style) string {
	arrow := Width(glyphHard)
	return block.Render(" "+Fit(plain, inner-1-arrow)) + Fg(block.Bg).Render(glyphHard)
}

// loadingFrame is the loading icon now: the clock decides which of the eight
// frames, 90ms each (components/layout/popup).
func loadingFrame() string {
	return loadingFrames[(time.Now().UnixMilli()/90)%8]
}

// LoadingFrame is the loading icon as it is now, for a row of the app's own
// that is loading (a transfer in a work list): the same icon, turning with
// the lib's. App.Spinning keeps it turning (components/layout/popup).
func LoadingFrame() string { return loadingFrame() }

type spinMsg struct{}

func spinTick() tea.Cmd {
	return tea.Tick(90*time.Millisecond, func(time.Time) tea.Msg { return spinMsg{} })
}

// popupWidth is Rules F7's width: content + 4 when it is known when the popup
// opens, at least the title and hint, at most min(screen − 2, 120); the
// maximum when it is not known.
func popupWidth(sw, content int, known bool) int {
	most := max(min(sw-2, 120), 1)
	if !known {
		return most
	}
	return min(content+4, most)
}

// scrollWindow keeps focus in sight in a window of n rows over total,
// scrolling the least (components/layout/popup): the new first row.
func scrollWindow(top, focus, n, total int) int {
	if focus < top {
		top = focus
	}
	if focus >= top+n {
		top = focus - n + 1
	}
	return min(max(top, 0), max(total-n, 0))
}
