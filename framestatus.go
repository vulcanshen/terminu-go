package terminu

// FrameStatus is one lasting status of a panel's own on its frame (tdp
// panel.1), such as a file the panel watches and when it was last read: a
// glyph and a short text (U+F04E6 17:43:42), in the corner the app chooses.
// Whether a panel has one, what it says and where are the app's; the lib
// draws it while Glyph or Text is not "", and keeping to a glyph and a short
// text is the app's. In its corner it goes on the inner side of what is
// there, a space either side and a line segment between: top left after the
// capsule chain and the loading icon, top right left of the mode name,
// bottom left after the hint, bottom right left of a search's status and the
// scroll position; alone in its corner, a segment from it. On a border too
// narrow it is the first thing to go, whole, before the hint's pairs, and it
// comes back when there is room. The app sets and clears it as what it tells
// changes, from the update loop (a timer of its own made with App.Tick, its
// message in OnMsg; an action's Do), as any other field of the panel; popups
// have none.
type FrameStatus struct {
	Glyph, Text string
	Corner      Corner
	// Kind is how what it tells is going: StatusFine draws it in the
	// frame's colour, as the scroll position (Yellow in a mode); when it has
	// gone wrong, StatusWarning and StatusError draw it in Peach and Red,
	// dimmed on a panel without the focus, a streaming one too (tdp T2).
	Kind StatusKind
}

// Corner is a corner of a panel's frame.
type Corner int

const (
	TopLeft     Corner = iota // after the capsule chain and the loading icon
	TopRight                  // left of the mode name
	BottomLeft                // after the hint
	BottomRight               // left of a search's status and the scroll position
)

func (c Corner) top() bool    { return c == TopLeft || c == TopRight }
func (c Corner) bottom() bool { return c == BottomLeft || c == BottomRight }

// StatusKind is how what a FrameStatus tells is going (tdp panel.1).
type StatusKind int

const (
	StatusFine    StatusKind = iota // in the frame's colour
	StatusWarning                   // worth noticing: Peach
	StatusError                     // gone wrong: Red
)

// text is the status as drawn, glyph and text with control characters by
// name (components); "" with neither.
func (s FrameStatus) text() string {
	switch {
	case s.Glyph == "":
		return Escape(s.Text)
	case s.Text == "":
		return Escape(s.Glyph)
	}
	return Escape(s.Glyph) + " " + Escape(s.Text)
}

// style is the status's colour on a frame of colour frame.
func (s FrameStatus) style(frame Color, focused bool) Style {
	var c Color
	switch s.Kind {
	case StatusWarning:
		c = Peach
	case StatusError:
		c = Red
	default:
		return Fg(frame)
	}
	if !focused { // on the frame, as the hint: dimmed on a stream too (tdp T2)
		c = DimColor(c)
	}
	return Fg(c)
}
