package terminu

import (
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ReadAgain puts lines in the open viewer v in place of what it shows, its
// source read again (a file it watches; tdp note.9). Whether and when to
// read again are the app's; it is as if v had just been opened on the new
// lines, keeping only this:
//
//   - Where the view was; at the new end when the end was in sight (never
//     in the selection mode), or when the lines are fewer than fill it.
//   - The line numbers' column, which only widens while v is open.
//   - The lib's search, found again in the new lines: the first hit at or
//     after the current one is current, the first of all when none is (as
//     n wraps), and the view does not move for it. With none left it is a
//     search that finds nothing: an Info toast, the hits and the status
//     gone. An app's own search (Keys taking /, n or N) is the app's to
//     find again.
//   - The selection mode's positions, not its text, a position being its
//     line and that line's column: both still there, nothing changes and
//     the copy is of the new text. Where the selection started gone, the
//     selection goes, back to the mode's first layer as one Esc; the
//     cursor's place gone, the mode is left, as Esc to the end; each said
//     in a Warning toast that names Subject (Title without one). With the
//     selection mode's Space menu or key reference open, it is said in a
//     note over them instead, drawn as an error popup in Peach with the
//     Warning toast's glyph, READ AGAIN: <Subject>; Enter and Esc close it
//     and them, back to the viewer (tdp note-99).
//
// Otherwise an open Space menu or key reference stays as it is. It returns
// the toasts. On a viewer not on the screen it only puts the lines in.
func (a *App) ReadAgain(v *Viewer, lines []string) tea.Cmd {
	if !a.viewerOpen(v) {
		v.Lines = lines
		return nil
	}
	v.sync()
	line, sub := v.maxTop()
	end := v.textW > 0 && !v.inMode && (v.top > line || v.top == line && v.sub >= sub)
	q, at, searched := v.Query, pos{}, v.Search != nil && len(v.Hits) > 0
	if searched {
		h := v.Hits[min(max(v.Current, 0), len(v.Hits)-1)]
		at = pos{h.Line, h.From}
	}

	v.Lines = lines
	v.sync()
	v.readLines()
	if end {
		v.top, v.sub = 1<<30, 0 // the next draw brings it to the new end
	}
	var cmds []tea.Cmd
	if searched {
		v.Hits, v.Current = v.FindText(q), 0
		for i, h := range v.Hits {
			if h.Line > at.line || h.Line == at.line && h.From >= at.col {
				v.Current = i
				break
			}
		}
		if len(v.Hits) == 0 {
			v.Query = ""
			cmds = append(cmds, a.Toast(Info, "No line has "+q))
		}
	}
	subject := cmpOr(v.Subject, v.Title)
	switch {
	case v.inMode && !v.there(v.line, v.col):
		v.inMode, v.selecting = false, false
		cmds = append(cmds, a.viewerGone(v, Warning, "Read again", subject+" changed under the cursor: "+cmpOr(v.Select.Name, "Select")+" left"))
	case v.selecting && !v.there(v.fromL, v.fromC):
		v.selecting = false
		cmds = append(cmds, a.viewerGone(v, Warning, "Read again", subject+" changed: where the selection started is gone"))
	}
	return tea.Batch(cmds...)
}

// CannotReadAgain is for v when its source cannot be read any more (a file
// deleted; tdp note.9): v leaves the selection mode and keeps what it shows,
// and message, the app's words, says why (tdp panel.15): an Error toast; with
// the selection mode's Space menu or key reference open, an error popup over
// them instead, CANNOT READ: <Subject>, message and "The selection is gone",
// whose Enter and Esc close it and them, back to the viewer (tdp note-100).
// With no message it gives no toast: a read the app repeats on its own that
// fails the same way again (the same words) shows and records nothing, a
// different failure toasting again (tdp note.7). On a viewer not on the
// screen it says nothing.
func (a *App) CannotReadAgain(v *Viewer, message ...string) tea.Cmd {
	in := v.inMode
	v.inMode, v.selecting = false, false
	if !a.viewerOpen(v) {
		return nil
	}
	if at := slices.IndexFunc(a.stack, func(l *layer) bool { return l.viewer == v && !l.closing }); in && at >= 0 {
		rows := append(slices.Clone(message), "The selection is gone")
		return a.goneNote(a.stack[at], Error, "Cannot read", cmpOr(v.Subject, v.Title), rows...)
	}
	if len(message) == 0 {
		return nil
	}
	return a.Toast(Error, message...)
}

// SelectionGone says that content read again took away the selection of the
// mode p is in, an app's own selection mode on its own reader (tdp note.9).
// With that mode's Space menu or key reference open, it is a note over them,
// drawn as an error popup in kind's colour, Warning in Peach with the Warning
// toast's glyph, Error in Red, titled title, message its rows; Enter and Esc
// close it and them, back to the screen (tdp note-99, note-100). With neither
// open it is a toast of kind with message. Call it before leaving the mode or
// going back a layer: leaving it then keeps them open under the note, to
// close with it. Kind is Warning or Error.
func (a *App) SelectionGone(p *Panel, kind ToastKind, title string, message ...string) tea.Cmd {
	at := slices.IndexFunc(a.stack, func(l *layer) bool { return l.mode == p && !l.closing })
	if at < 0 {
		return a.Toast(kind, message...)
	}
	return a.goneNote(a.stack[at], kind, title, "", message...)
}

// viewerGone says, in kind, that v's selection is gone: a toast, or the note
// over its selection mode's Space menu or ? (tdp note.9).
func (a *App) viewerGone(v *Viewer, kind ToastKind, title, message string) tea.Cmd {
	at := slices.IndexFunc(a.stack, func(l *layer) bool { return l.viewer == v && !l.closing })
	if at < 0 {
		return a.Toast(kind, message)
	}
	return a.goneNote(a.stack[at], kind, title, cmpOr(v.Subject, v.Title), message)
}

// goneNote opens the note that a selection is gone over the popups from down
// up, closing with them (tdp note.9): an error popup, in Peach with the
// Warning toast's glyph for Warning. It goes in the App Log as the toast it
// stands for would.
func (a *App) goneNote(down *layer, kind ToastKind, title, subject string, message ...string) tea.Cmd {
	a.Log(kind, errorText(title, subject, strings.Join(message, " ")))
	n := errorPopup(title, subject, message...)
	if kind == Warning {
		peach := Peach
		n.glyph, n.color, n.textStyle = glyphWarning, &peach, Fg(Peach)
	}
	n.down = down
	return a.open(n)
}

// there reports that the text has line and that line's column col: a
// position kept only where it still is (tdp note.9). An empty line has its
// column 0, where the cursor can be.
func (v *Viewer) there(line, col int) bool {
	if line < 0 || line >= len(v.plain) || col < 0 {
		return false
	}
	n := len(v.plain[line])
	return col < n || col == 0 && n == 0
}
