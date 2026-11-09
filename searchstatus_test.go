package terminu

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// statusRe is a search's status on a bottom border (tdp note.3).
var statusRe = regexp.MustCompile(`━ (/\S* \d+ of \d+) ━`)

// TestSearchStatusBorder: while a search has hits the bottom border has its
// status left of the scroll position, a segment between: "/", the text,
// the current hit of how many, in the scroll position's colour. When the
// border is short the hint's pairs go from the end first; only then is the
// text cut at its end with …, the numbers and the scroll position kept
// (tdp note.3, panel.1, note-89).
func TestSearchStatusBorder(t *testing.T) {
	pairs := []KeyHint{{"/", "search"}, {"v", "visual"}, {"Esc", "close"}}
	draw := func(w int, s SearchStatus, scroll string) string {
		b := box{width: w, lines: heavyLines, color: Peach, hintKey: Fg(Blue), hint: pairs, scroll: scroll, found: s}
		out := b.draw()
		return out[len(out)-1]
	}
	found := SearchStatus{Text: "goroutine", Current: 1, Count: 5}
	line := draw(80, found, "6-10 of 42")
	plain := StripStyles(line)
	if !strings.HasSuffix(plain, "━ /goroutine 2 of 5 ━ 6-10 of 42 ━┛") || !strings.Contains(plain, "Esc:close") || Width(line) != 80 {
		t.Errorf("the status left of the scroll position, the whole hint: %q", plain)
	}
	slash, digit := strings.Index(plain, "/goroutine"), strings.Index(plain, "6-10")
	if s, d := covCAt(line, Width(plain[:slash])), covCAt(line, Width(plain[:digit])); s.St.Fg != Peach || d.St.Fg != Peach {
		t.Errorf("in the scroll position's colour, the frame's: %v %v", s.St.Fg, d.St.Fg)
	}
	if plain := StripStyles(draw(80, found, "")); !strings.HasSuffix(plain, "━ /goroutine 2 of 5 ━┛") {
		t.Errorf("no scroll position, the status at the end: %q", plain)
	}
	if plain := StripStyles(draw(80, SearchStatus{Text: "goroutine"}, "6-10 of 42")); strings.Contains(plain, "goroutine") {
		t.Errorf("no hits, no status: %q", plain)
	}
	sawPart, sawCut := false, false
	for w := 80; w >= 28; w-- {
		line := draw(w, found, "6-10 of 42")
		plain := StripStyles(line)
		if Width(line) != w || !strings.HasSuffix(plain, " 2 of 5 ━ 6-10 of 42 ━┛") {
			t.Fatalf("%d wide: the numbers and the scroll position stay: %q", w, plain)
		}
		has := func(p string) bool { return strings.Contains(plain, p) }
		if has("━ /goroutine 2 of 5 ━") {
			// The pairs go as the hint's do (renderHints; Esc:close, a
			// popup's last, stays).
			sawPart = sawPart || has(":") && !(has("/:search") && has("v:visual") && has("Esc:close"))
			continue
		}
		sawCut = true
		if has(":") || !has("━ /") || !has("… 2 of 5 ━") {
			t.Errorf("%d wide: the text is cut only with the hint gone, … before the numbers: %q", w, plain)
		}
	}
	if plain := StripStyles(draw(27, found, "6-10 of 42")); strings.Contains(plain, "of 5") || !strings.HasSuffix(plain, "━ 6-10 of 42 ━┛") {
		t.Errorf("no room for /… and the numbers: no status, the scroll position stays: %q", plain)
	}
	if !sawPart || !sawCut {
		t.Errorf("some pairs dropped with the status whole %v; the text cut %v", sawPart, sawCut)
	}
}

// TestViewerSearchStatus: a viewer's / search shows its status on the
// bottom border; the count follows n|N, wrapping at the ends; it goes when
// the viewer closes, and with no hit there is none; the key reference's
// search shows it too (tdp note.3, note.5, note-89).
func TestViewerSearchStatus(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(100, 20)
	lines := make([]string, 42)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d", i+1)
		if i%9 == 1 {
			lines[i] = fmt.Sprintf("goroutine %d [running]", i)
		}
	}
	v := &Viewer{Title: "Log", Lines: lines}
	ta.Viewer(v)
	ta.settle()
	status := func() string {
		t.Helper()
		m := statusRe.FindStringSubmatch(StripStyles(strings.Join(ta.Frame(), "\n")))
		if m == nil {
			return ""
		}
		return m[1]
	}
	if s := status(); s != "" {
		t.Errorf("no search, no status: %q", s)
	}
	ta.press("/", "g", "o", "r", "o", "u", "t", "i", "n", "e", "Enter")
	if s := status(); s != "/goroutine 1 of 5" || v.Query != "goroutine" {
		t.Fatalf("found: %q, Query %q", s, v.Query)
	}
	for i := 2; i <= 5; i++ {
		if ta.press("n"); status() != fmt.Sprintf("/goroutine %d of 5", i) {
			t.Errorf("n to %d: %q", i, status())
		}
	}
	if ta.press("n"); status() != "/goroutine 1 of 5" {
		t.Errorf("n past the last goes round to the first: %q", status())
	}
	if ta.press("N"); status() != "/goroutine 5 of 5" {
		t.Errorf("N past the first goes round to the last: %q", status())
	}
	ta.press("/", "z", "z", "z", "Enter", "Esc")
	if s := status(); s != "" {
		t.Errorf("no hit, no status: %q", s)
	}
	ta.press("/", "l", "i", "n", "e", "Enter")
	if s := status(); !strings.HasPrefix(s, "/line") {
		t.Fatalf("found again: %q", s)
	}
	ta.press("Esc")
	if ta.top() != nil || v.Query != "" || v.Hits != nil {
		t.Errorf("closed: the status goes with the hits: Query %q", v.Query)
	}
	ta.Viewer(v)
	ta.settle()
	if s := status(); s != "" {
		t.Errorf("opened again, no status: %q", s)
	}
	ta.press("Esc", "?", "/", "t", "h", "i", "s", "Enter")
	n := ta.topView().(*noteView)
	if want := fmt.Sprintf("/this 1 of %d", len(n.hits)); len(n.hits) == 0 || status() != want {
		t.Errorf("the key reference: %q, want %q", status(), want)
	}
	if ta.press("N"); status() != fmt.Sprintf("/this %d of %d", len(n.hits), len(n.hits)) {
		t.Errorf("the key reference, N past the first: %q", status())
	}
}

// TestPanelSearchStatus: the app's own search in a panel gives the panel
// its status (Panel.SearchStatus), drawn as a viewer's: left of the scroll
// position, or at the end with none, in the frame's colour, dimming with
// it; cleared, it goes (tdp note.3, panel.1, panel.14).
func TestPanelSearchStatus(t *testing.T) {
	ta := newTestApp(t)
	ta.resize(120, 14)
	ta.pods.SearchStatus = SearchStatus{Text: "pod", Current: 2, Count: 40}
	row := func() (string, string) {
		t.Helper()
		for _, l := range ta.Frame() {
			if p := StripStyles(l); strings.Contains(p, "/pod 3 of 40") {
				return l, p
			}
		}
		return "", ""
	}
	line, plain := row()
	if !strings.Contains(plain, "━ /pod 3 of 40 ━ 1 of 40 ━┛") {
		t.Fatalf("left of the scroll position: %q", plain)
	}
	colours := func(line, plain string) (Color, Color) {
		i, j := strings.Index(plain, "/pod"), strings.Index(plain, " 1 of 40")
		return covCAt(line, Width(plain[:i])).St.Fg, covCAt(line, Width(plain[:j])+1).St.Fg
	}
	if s, d := colours(line, plain); s != d || s != covCAt(line, 0).St.Fg {
		t.Errorf("the scroll position's colour, the frame's: %v %v", s, d)
	}
	ta.press("2")
	line, plain = row()
	if s, d := colours(line, plain); s != d || s != covCAt(line, 0).St.Fg || s == Blue {
		t.Errorf("unfocused, it dims with the frame: %v %v", s, d)
	}
	ta.press("1")
	ta.pods.SearchStatus = SearchStatus{}
	if _, p := row(); p != "" || statusRe.MatchString(StripStyles(strings.Join(ta.Frame(), "\n"))) {
		t.Errorf("cleared, it goes: %q", p)
	}
	ta.pods.SearchStatus = SearchStatus{Text: "pod", Current: 2, Count: 40}
	ta.list.SetRows(ta.list.Rows()[:3])
	if _, p := row(); !strings.Contains(p, "━ /pod 3 of 40 ━┛") {
		t.Errorf("no scroll position, the status at the end: %q", p)
	}
}
