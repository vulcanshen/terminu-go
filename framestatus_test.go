package terminu

import (
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// statusBody draws nothing and says a scroll position.
type statusBody struct{ scroll string }

func (statusBody) Draw(w, h int, _ bool) []string         { return blankLines(w, h) }
func (statusBody) Key(string, tea.KeyMsg) (bool, tea.Cmd) { return false, nil }
func (statusBody) HasItem() bool                          { return false }
func (b statusBody) Scroll() string                       { return b.scroll }
func (statusBody) MoveKeys() []KeyRow                     { return nil }

const statusGlyph = "\U000F04E6" // nf-md-sync

// statusText is the status the tests give a panel, as drawn.
var statusText = statusGlyph + " 17:43:42"

func statusPanel(corner Corner, hints ...string) *Panel {
	p := &Panel{Label: "README.md", Body: statusBody{},
		FrameStatus: FrameStatus{Glyph: statusGlyph, Text: "17:43:42", Corner: corner}}
	for _, h := range hints {
		p.Actions = append(p.Actions, Action{Key: h[:1], Label: h, Scope: PanelScope, Hint: h, Do: func() tea.Cmd { return nil }})
	}
	return p
}

// frameOf is a panel's top and bottom border, w wide, and whether every line
// is w wide.
func frameOf(p *Panel, w int, focused bool) (top, bottom string, exact bool) {
	out := p.draw(2, w, 4, focused, false)
	exact = true
	for _, l := range out {
		exact = exact && Width(l) == w
	}
	return out[0], out[len(out)-1], exact
}

// TestPanel1FrameStatusCorners: a panel's own status is a glyph and a short
// text in the corner the app chose, on the inner side of what is there, a
// space either side and a segment between: top left after the chain and
// the loading icon, top right left of the mode name, bottom left after the
// hint, bottom right left of the search's status and the scroll position;
// alone in its corner, a segment from it (tdp panel.1, panel-172,
// panel-173).
func TestPanel1FrameStatusCorners(t *testing.T) {
	S := statusText
	chain := StripStyles(drawChain(statusPanel(TopLeft).capsules(2), Blue, 100))
	check := func(what string, p *Panel, top bool, ok func(string) bool) {
		t.Helper()
		tl, bl, exact := frameOf(p, 70, true)
		line := StripStyles(bl)
		if top {
			line = StripStyles(tl)
		}
		if !exact || !ok(line) || strings.Count(line, "17:43:42") > 1 {
			t.Errorf("%s: %q (exact %v)", what, line, exact)
		}
	}
	has, ends, starts := strings.Contains, strings.HasSuffix, strings.HasPrefix

	check("top left, after the chain", statusPanel(TopLeft), true, func(l string) bool {
		return starts(l, "┏"+chain+"━ "+S+" ━") && ends(l, "━┓")
	})
	p := statusPanel(TopLeft)
	p.Loading = true
	check("top left, after the loading icon", p, true, func(l string) bool {
		rest, ok := strings.CutPrefix(l, "┏"+chain+" ")
		icon, after, found := strings.Cut(rest, " ━ "+S+" ━")
		return ok && found && Width(icon) == 1 && ends(after, "━┓")
	})
	check("top right, alone", statusPanel(TopRight), true, func(l string) bool {
		return ends(l, "━━ "+S+" ━┓")
	})
	p = statusPanel(TopRight)
	p.EnterMode(&Mode{Name: "Select", Exit: "cancel"})
	check("top right, left of the mode name", p, true, func(l string) bool {
		return ends(l, "━━ "+S+" ━┫Select┣━┓")
	})
	p = statusPanel(TopLeft)
	p.EnterMode(&Mode{Name: "Select", Exit: "cancel"})
	check("top left, the mode name where it was", p, true, func(l string) bool {
		return starts(l, "┏"+chain+"━ "+S+" ━") && ends(l, "━━┫Select┣━┓")
	})
	check("bottom left, after the hint", statusPanel(BottomLeft, "open"), false, func(l string) bool {
		return starts(l, "┗━ o:open ━ "+S+" ━━") && ends(l, "━┛")
	})
	check("bottom left, no hint", statusPanel(BottomLeft), false, func(l string) bool {
		return starts(l, "┗━ "+S+" ━━") && ends(l, "━┛")
	})
	check("bottom right, alone", statusPanel(BottomRight, "open"), false, func(l string) bool {
		return starts(l, "┗━ o:open ━") && ends(l, "━━ "+S+" ━┛")
	})
	searched := func(corner Corner) *Panel {
		p := statusPanel(corner, "open")
		p.Body = statusBody{scroll: "1-2 of 9"}
		p.SearchStatus = SearchStatus{Text: "needle", Current: 1, Count: 5}
		return p
	}
	check("bottom right, left of the search and the scroll position", searched(BottomRight), false, func(l string) bool {
		return starts(l, "┗━ o:open ━") && ends(l, "━━ "+S+" ━ /needle 2 of 5 ━ 1-2 of 9 ━┛")
	})
	check("bottom left, with the search and the scroll position", searched(BottomLeft), false, func(l string) bool {
		return starts(l, "┗━ o:open ━ "+S+" ━━") && ends(l, "━━ /needle 2 of 5 ━ 1-2 of 9 ━┛")
	})
	p = statusPanel(BottomRight)
	p.Body = statusBody{scroll: "1-2 of 9"}
	check("bottom right, before the scroll position alone", p, false, func(l string) bool {
		return ends(l, "━━ "+S+" ━ 1-2 of 9 ━┛")
	})

	// A glyph or a text alone is drawn as it is; control characters by name.
	p = statusPanel(BottomLeft)
	p.FrameStatus.Glyph = ""
	check("a text alone", p, false, func(l string) bool { return starts(l, "┗━ 17:43:42 ━━") })
	p.FrameStatus = FrameStatus{Glyph: statusGlyph, Corner: TopRight}
	check("a glyph alone", p, true, func(l string) bool { return ends(l, "━━ "+statusGlyph+" ━┓") })
	p.FrameStatus = FrameStatus{Text: "17:43:42\nx", Corner: BottomRight}
	check("a control character by its name", p, false, func(l string) bool { return ends(l, "━━ 17:43:42\\nx ━┛") })

	// Cleared, the frame is as it was without one.
	for _, c := range []Corner{TopLeft, TopRight, BottomLeft, BottomRight} {
		p := searched(c)
		p.Loading = true
		p.FrameStatus = FrameStatus{Corner: c}
		bare := searched(c)
		bare.Loading = true
		bare.FrameStatus = FrameStatus{}
		got, want := p.draw(2, 70, 4, true, false), bare.draw(2, 70, 4, true, false)
		if strings.Join(got, "\n") != strings.Join(want, "\n") || has(StripStyles(got[0]+got[3]), "17:43") {
			t.Errorf("corner %d with no glyph or text: no status, the frame as ever", c)
		}
	}
}

// TestPanel1FrameStatusColour: in the frame's colour, as the scroll
// position (Yellow in a mode); Peach or Red when what it tells has gone
// wrong, no other colour; without the focus dimmed with the panel, a
// streaming one too (tdp panel.1, color.2, T2, panel-174).
func TestPanel1FrameStatusColour(t *testing.T) {
	colour := func(p *Panel, focused, pointed bool) (status, frame Color) {
		t.Helper()
		out := p.draw(2, 70, 4, focused, pointed)
		line := out[0]
		if !p.FrameStatus.Corner.top() {
			line = out[len(out)-1]
		}
		plain := StripStyles(line)
		i := strings.Index(plain, "17:43:42")
		if i < 0 {
			t.Fatalf("no status: %q", plain)
		}
		g := strings.Index(plain, statusGlyph)
		if c := covCAt(line, Width(plain[:g])).St.Fg; c != covCAt(line, Width(plain[:i])).St.Fg {
			t.Errorf("the glyph and the text in one colour: %v", c)
		}
		return covCAt(line, Width(plain[:i])).St.Fg, covCAt(line, 0).St.Fg
	}
	for _, corner := range []Corner{TopLeft, TopRight, BottomLeft, BottomRight} {
		p := statusPanel(corner, "open")
		for _, c := range []struct {
			focused, pointed bool
			frame            Color
		}{{true, false, Blue}, {false, false, Surface2}, {false, true, Lavender}} {
			p.FrameStatus.Kind = StatusFine
			if s, f := colour(p, c.focused, c.pointed); s != f || f != c.frame {
				t.Errorf("corner %d focused %v pointed %v: fine, the frame's colour %v: %v", corner, c.focused, c.pointed, f, s)
			}
			for kind, want := range map[StatusKind]Color{StatusWarning: Peach, StatusError: Red} {
				p.FrameStatus.Kind = kind
				if !c.focused {
					want = DimColor(want)
				}
				if s, _ := colour(p, c.focused, c.pointed); s != want {
					t.Errorf("corner %d focused %v pointed %v kind %d: %v, want %v", corner, c.focused, c.pointed, kind, s, want)
				}
			}
		}
		p.FrameStatus.Kind = StatusFine
		p.EnterMode(&Mode{Name: "Select", Exit: "cancel"})
		for _, focused := range []bool{true, false} {
			if s, f := colour(p, focused, false); s != Yellow || f != Yellow {
				t.Errorf("corner %d in a mode, focused %v: Yellow as the frame: %v %v", corner, focused, s, f)
			}
		}
		p.FrameStatus.Kind = StatusError
		if s, _ := colour(p, true, false); s != Red {
			t.Errorf("corner %d in a mode, gone wrong: Red: %v", corner, s)
		}
		p.LeaveMode()
		p.Stream = true
		p.FrameStatus.Kind = StatusWarning
		if s, _ := colour(p, false, false); s != DimColor(Peach) {
			t.Errorf("corner %d a stream without the focus: dimmed: %v", corner, s)
		}
		p.FrameStatus.Kind = StatusKind(9) // not one of the kinds: the frame's colour
		if s, f := colour(p, true, false); s != f {
			t.Errorf("corner %d an unknown kind: the frame's colour: %v %v", corner, s, f)
		}
	}
}

// TestPanel1FrameStatusNarrow: on a border too narrow the status is the
// first to go, whole, before the hint's pairs, the search's text, the chain,
// the loading icon or the mode name, in every corner; it comes back when
// there is room (tdp panel.1, panel-175).
func TestPanel1FrameStatusNarrow(t *testing.T) {
	S := statusText
	for _, iw := range []int{1, 2} {
		icon0, power0 := IconWidth(), PowerlineWidth()
		SetIconWidth(iw, 1)
		for _, corner := range []Corner{TopLeft, TopRight, BottomLeft, BottomRight} {
			p := statusPanel(corner, "open", "edit", "copy")
			p.Tabs = []string{"Raw", "History"}
			p.Body = statusBody{scroll: "1-2 of 9"}
			p.SearchStatus = SearchStatus{Text: "needle", Current: 1, Count: 5}
			p.Loading = true
			if corner.top() {
				p.EnterMode(&Mode{Name: "Select", Exit: "cancel"})
			}
			chain := StripStyles(drawChain(p.capsules(2), Blue, 1000))
			whole := func(top, bottom string) bool {
				if corner.top() {
					rest, ok := strings.CutPrefix(top, "┏"+chain+" ")
					icon, _, _ := strings.Cut(rest, " ")
					return ok && slices.Contains(loadingFrames[:], icon) && strings.HasSuffix(top, "┫Select┣━┓")
				}
				return strings.HasPrefix(bottom, "┗━ h|l:tab o:open e:edit c:copy ━") && strings.HasSuffix(bottom, "━ /needle 2 of 5 ━ 1-2 of 9 ━┛")
			}
			gone, sawGone := 0, false
			for w := 100; w >= 4; w-- {
				tl, bl, exact := frameOf(p, w, true)
				top, bottom := StripStyles(tl), StripStyles(bl)
				if !exact {
					t.Fatalf("icons %d corner %d %d wide: every line exactly as wide", iw, corner, w)
				}
				both := top + bottom
				shown := strings.Contains(both, " "+S+" ")
				if !shown && strings.Contains(both, "17:4") {
					t.Fatalf("icons %d corner %d %d wide: cut: %q %q", iw, corner, w, top, bottom)
				}
				switch {
				case shown && !strings.Contains(both, "━ "+S+" ━"):
					t.Fatalf("icons %d corner %d %d wide: a segment either side: %q %q", iw, corner, w, top, bottom)
				case shown && sawGone:
					t.Fatalf("icons %d corner %d %d wide: back on a narrower border: %q %q", iw, corner, w, top, bottom)
				case shown && !whole(top, bottom):
					t.Fatalf("icons %d corner %d %d wide: shown with the rest not whole: %q %q", iw, corner, w, top, bottom)
				case !shown && !sawGone:
					sawGone, gone = true, w
					if !whole(top, bottom) {
						t.Errorf("icons %d corner %d %d wide: the status went first, the rest still whole: %q %q", iw, corner, w, top, bottom)
					}
				}
			}
			if !sawGone || gone < 20 {
				t.Errorf("icons %d corner %d: it went at %d", iw, corner, gone)
			}
			// Back with room: on the narrowest border that has it, one
			// segment of line is all that is left over.
			tl, bl, _ := frameOf(p, gone+1, true)
			top, bottom := StripStyles(tl), StripStyles(bl)
			tight := strings.Contains(bottom, "c:copy ━ "+S+" ━ /needle 2 of 5 ━ 1-2 of 9 ━┛")
			if corner.top() {
				tight = strings.HasPrefix(top, "┏"+chain+" ") && strings.HasSuffix(top, " ━ "+S+" ━┫Select┣━┓")
			}
			if !tight {
				t.Errorf("icons %d corner %d: back with room, %d wide, a segment left over: %q %q", iw, corner, gone+1, top, bottom)
			}
		}
		SetIconWidth(icon0, power0)
	}
}

// TestPanel1FrameStatusApp: the app sets it on a panel of its screen and
// changes it as what it tells changes; the next frame has it, and cleared it
// goes (tdp panel.1).
func TestPanel1FrameStatusApp(t *testing.T) {
	ta := newTestApp(t)
	frame := func() string { return StripStyles(strings.Join(ta.Frame(), "\n")) }
	ta.pods.FrameStatus = FrameStatus{Glyph: statusGlyph, Text: "17:43:42", Corner: BottomRight}
	if f := frame(); strings.Contains(f, "17:43") {
		t.Errorf("no room beside the whole hint and the scroll position, 40 wide: none: %q", f)
	}
	ta.resize(200, 30)
	if f := frame(); !strings.Contains(f, "━ "+statusText+" ━ 1 of 40 ━┛") {
		t.Errorf("left of the scroll position: %q", f)
	}
	ta.pods.FrameStatus.Text = "17:44:10"
	if f := frame(); !strings.Contains(f, statusGlyph+" 17:44:10 ━") || strings.Contains(f, "17:43:42") {
		t.Errorf("changed, the next frame has it: %q", f)
	}
	ta.pods.FrameStatus = FrameStatus{}
	if f := frame(); strings.Contains(f, "17:44") {
		t.Errorf("cleared, it goes: %q", f)
	}
}
