package terminu

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ToastKind is what a toast says (tdp toast): which one goes
// with the meaning of its colour.
type ToastKind int

const (
	Info    ToastKind = iota // a neutral word, neither good nor bad
	Warning                  // done, or still usable, with something to notice
	Error                    // an action failed
	Success                  // an action is done
)

func (k ToastKind) glyph() string {
	return [...]string{glyphInfo, glyphWarning, glyphError, glyphSuccess}[k]
}

func (k ToastKind) title() string { return [...]string{"INFO", "WARNING", "ERROR", "SUCCESS"}[k] }

func (k ToastKind) color() Color { return [...]Color{Text, Peach, Red, Green}[k] }

// duration: Success and Info 2200ms, Warning and Error 4400ms.
func (k ToastKind) duration() time.Duration {
	if k == Info || k == Success {
		return 2200 * time.Millisecond
	}
	return 4400 * time.Millisecond
}

// toastView is a short message from the bottom: not a layer, it neither dims
// nor is dimmed, and every key but Esc goes through it.
type toastView struct {
	kind  ToastKind
	text  []string // its rows, each folded on its own
	lines []string
	id    int
	// most is the app's cap on its lines (App.ToastLines), 0 for none.
	most int
	// inspect makes it the inspect toast (tdp toast.4): no timer and no hint,
	// Esc going through it too; its kind is Info, for its text in Text.
	inspect bool
}

type toastDone struct{ id int }

func (t *toastView) hint() []KeyHint {
	if t.inspect {
		return nil // no key closes it
	}
	return []KeyHint{{"Esc", "close"}}
}

func (t *toastView) glyph() string {
	if t.inspect {
		return glyphInspect
	}
	return t.kind.glyph()
}

func (t *toastView) title() string {
	if t.inspect {
		return "INSPECT"
	}
	return t.kind.title()
}

func (t *toastView) measure(sw, sh int) (int, int) {
	most := max(min(sw-2, 120), 1)
	text := make([]string, len(t.text))
	widest := 0
	for i, s := range t.text {
		text[i] = Escape(s) // a short text (components, control characters)
		widest = max(widest, Width(text[i]))
	}
	w := min(max(widest+4, Width(titleCapsule(t.glyph(), t.title(), Base, 1<<20))+4, hintWidth(t.hint())+6), most)
	var lines []string
	for _, s := range text {
		lines = append(lines, Wrap(s, w-4)...)
	}
	// As many lines as the app allows, and as fit above the bottom two rows;
	// the last one kept ends in "…".
	n := len(lines)
	if t.most > 0 {
		n = min(n, t.most)
	}
	n = max(min(n, sh-6), 1)
	if n < len(lines) {
		lines = lines[:n]
		last := Truncate(lines[n-1]+" …", w-4)
		if Width(last) < w-4 && last[len(last)-len("…"):] != "…" {
			last += "…"
		}
		lines[n-1] = last
	}
	t.lines = lines
	return w, len(lines) + 4
}

func (t *toastView) draw(w, h int, _ Color) []string {
	inner := w - 2
	body := make([]string, 0, len(t.lines))
	for _, l := range t.lines {
		body = append(body, " "+Center(Fg(t.kind.color()).Render(l), inner-2)+" ")
	}
	return popupBox{color: LayerColor(1), glyph: t.glyph(), title: t.title(), hint: t.hint()}.draw(w, body)
}

func (t *toastView) key(*App, string, tea.KeyMsg) (bool, tea.Cmd) { return false, nil }
func (t *toastView) surface() string                              { return "toast" }
func (t *toastView) keyRef() []refSection                         { return nil }
func (t *toastView) typing() bool                                 { return false }
