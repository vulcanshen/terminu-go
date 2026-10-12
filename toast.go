package terminu

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ToastKind is what a toast says (components/dialog/toast).
type ToastKind int

const (
	Info ToastKind = iota
	Warning
	Error
)

func (k ToastKind) glyph() string {
	return [...]string{glyphInfo, glyphWarning, glyphError}[k]
}

func (k ToastKind) title() string { return [...]string{"INFO", "WARNING", "ERROR"}[k] }

func (k ToastKind) color() Color { return [...]Color{Text, Peach, Red}[k] }

// duration: Info 2200ms, Warning and Error 4400ms.
func (k ToastKind) duration() time.Duration {
	if k == Info {
		return 2200 * time.Millisecond
	}
	return 4400 * time.Millisecond
}

// toastView is a short message from the bottom: not a layer, it neither dims
// nor is dimmed, and every key but Esc goes through it.
type toastView struct {
	kind  ToastKind
	text  string
	lines []string
	id    int
}

type toastDone struct{ id int }

func (t *toastView) hint() []KeyHint { return []KeyHint{{"Esc", "close"}} }

func (t *toastView) measure(sw, sh int) (int, int) {
	most := max(min(sw-2, 120), 1)
	w := min(max(Width(t.text)+4, Width(titleCapsule(t.kind.glyph(), t.kind.title(), Base, 1<<20))+4, hintWidth(t.hint())+6), most)
	lines := Wrap(t.text, w-4)
	if len(lines) > 3 {
		// Three lines at most, the third ending in "…".
		lines = lines[:3]
		lines[2] = Truncate(lines[2]+" …", w-4)
		if Width(lines[2]) < w-4 && lines[2][len(lines[2])-len("…"):] != "…" {
			lines[2] += "…"
		}
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
	return popupBox{color: LayerColor(1), glyph: t.kind.glyph(), title: t.kind.title(), hint: t.hint()}.draw(w, body)
}

func (t *toastView) key(*App, string, tea.KeyMsg) (bool, tea.Cmd) { return false, nil }
func (t *toastView) surface() string                              { return "toast" }
func (t *toastView) keyRef() []refSection                         { return nil }
func (t *toastView) typing() bool                                 { return false }
