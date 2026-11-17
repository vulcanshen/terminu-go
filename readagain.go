package terminu

import tea "github.com/charmbracelet/bubbletea"

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
//     in a Warning toast that names Subject (Title without one).
//
// An open Space menu or key reference stays as it is. It returns the
// toasts. On a viewer not on the screen it only puts the lines in.
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
		cmds = append(cmds, a.Toast(Warning, subject+" changed under the cursor: "+cmpOr(v.Select.Name, "Select")+" left"))
	case v.selecting && !v.there(v.fromL, v.fromC):
		v.selecting = false
		cmds = append(cmds, a.Toast(Warning, subject+" changed: where the selection started is gone"))
	}
	return tea.Batch(cmds...)
}

// CannotReadAgain is for v when its source cannot be read any more (a file
// deleted; tdp note.9): v leaves the selection mode, with no toast of its
// own, and keeps what it shows; the app's Error toast says why (tdp
// panel.15).
func (a *App) CannotReadAgain(v *Viewer) { v.inMode, v.selecting = false, false }

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
