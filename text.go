package terminu

import (
	"strings"

	"github.com/rivo/uniseg"
)

// Control characters in text that comes from outside (a file's name, a web
// page's title, a program's output) are never sent to the terminal; how they
// are drawn depends on the text (components, control characters): a typing
// row's as input says, reading content as a terminal draws it (the viewer,
// a panel's log, a file preview), and a name or a short text with every one
// of them by its name in Red. Escape is that last one.

// nameSGR starts a control character's name. It is Red with an attribute
// that changes nothing (24, not underlined), so the lib knows its own mark:
// on a block (a cursor, a capsule) the name is drawn Red under Base instead.
var nameSGR = "\x1b[" + Fg(Red).sgr() + ";24m"

// nameOnBlock is a name on a block: Red under Base, as a typing row's cursor
// on a control character (tdp input); with the same mark as
// nameSGR.
var nameOnBlock = "\x1b[" + Style{Fg: Base, Bg: Red}.sgr() + ";24m"

// nameMarked reports an SGR sequence carrying the names' mark, 24, as a
// parameter of its own, not a colour's: so it is found on a dimmed row too,
// where the colours were written again (tdptest's line break check).
func nameMarked(seq string) bool {
	params := strings.Split(strings.TrimSuffix(strings.TrimPrefix(seq, "\x1b["), "m"), ";")
	for i := 0; i < len(params); i++ {
		switch params[i] {
		case "38", "48":
			if i+1 < len(params) && params[i+1] == "2" {
				i += 4
			} else if i+1 < len(params) && params[i+1] == "5" {
				i += 2
			}
		case "24":
			return true
		}
	}
	return false
}

// showsLineBreak reports a drawn row with a line break's name in it, \n or
// \r drawn by its name in Red: in a name or a short text, dimmed or on a
// block too.
func showsLineBreak(row string) bool {
	name, run := false, ""
	found := false
	walk(row, func(seq string, cells int) {
		if cells == 0 && isSGR(seq) {
			if name && (strings.HasPrefix(run, `\n`) || strings.HasPrefix(run, `\r`)) {
				found = true
			}
			name, run = nameMarked(seq), ""
			return
		}
		if name {
			run += seq
		}
	})
	return found || name && (strings.HasPrefix(run, `\n`) || strings.HasPrefix(run, `\r`))
}

// Escape is text as a name or a short text is drawn (components, control
// characters): each control character (C0, DEL, C1), Tab, \n and ESC too,
// by its name in Red (\t, \n, \x1b, as a pasted one in an input), nothing
// read as a style or a line break, nothing dropped. The lib draws every name
// and short text an app gives it this way; an app drawing text from outside
// itself (on a Canvas, in a popup of its own) uses it too.
func Escape(s string) string {
	if !strings.ContainsFunc(s, isControlRune) {
		return s
	}
	var b strings.Builder
	gr := uniseg.NewGraphemes(s)
	for gr.Next() {
		c := gr.Str()
		if c == "\r\n" {
			b.WriteString(nameSGR + `\r\n` + "\x1b[0m")
			continue
		}
		if name, ok := ctrlName(c); ok {
			b.WriteString(nameSGR + name + "\x1b[0m")
			continue
		}
		if strings.ContainsFunc(c, isControlRune) {
			// A control character a mark joined: each rune on its own.
			for _, r := range c {
				if name, ok := ctrlName(string(r)); ok {
					b.WriteString(nameSGR + name + "\x1b[0m")
				} else {
					b.WriteRune(r)
				}
			}
			continue
		}
		b.WriteString(c)
	}
	return b.String()
}

// plainNames is s with its styles gone but the control characters' names,
// for a row drawn on a block (a cursor): the names stay Red under Base.
func plainNames(s string) string {
	if !strings.Contains(s, "\x1b") {
		return s
	}
	var b strings.Builder
	name := false
	for len(s) > 0 {
		if s[0] == 0x1b {
			seq := sgrAt.FindString(s)
			switch {
			case seq == "":
				seq = s[:1]
			case seq == nameSGR || seq == nameOnBlock:
				name = true
				b.WriteString(seq)
			case name && (seq == "\x1b[0m" || seq == "\x1b[m"):
				name = false
				b.WriteString(seq)
			}
			s = s[len(seq):]
			continue
		}
		end := strings.IndexByte(s, 0x1b)
		if end < 0 {
			end = len(s)
		}
		b.WriteString(s[:end])
		s = s[end:]
	}
	return b.String()
}

// readingRows are lines of reading content (a viewer's, a panel's log) as a
// terminal shows them (components, control characters): a line with \n in
// it is two, a \n ending a line being its own end; a \r inside a line
// leaves only its last state. The \r of a CRLF stays at the end of its row,
// not drawn but copied (crlf).
func readingRows(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		if !strings.ContainsAny(l, "\r\n") {
			out = append(out, l)
			continue
		}
		parts := strings.Split(l, "\n")
		if len(parts) > 1 && stripSGR(parts[len(parts)-1]) == "" {
			parts = parts[:len(parts)-1] // the \n ending the line is its own end
		}
		for _, p := range parts {
			cr := strings.HasSuffix(stripSGR(p), "\r")
			p = lastState(cutCR(p))
			if cr {
				p += "\r"
			}
			out = append(out, p)
		}
	}
	return out
}

// Reading is lines of reading content (a log, a program's output) as a
// terminal draws them, the way the lib's viewer and Pager draw theirs
// (components, control characters): a line with \n in it is two rows, a \r
// inside a line leaves its last state, the \r of a CRLF goes, colour codes
// colour, a tab is input_tab_width spaces, any other control character or
// escape sequence is its name in Red. For a body of the app's own that shows
// what is read (a container's log behind a prefix, folded): each string it
// gives back is one row's text, to fold or cut as the app does; Width,
// Truncate and Cut measure it right.
func Reading(lines ...string) []string {
	rows := readingRows(lines)
	for i, r := range rows {
		rows[i] = readingLine(cutCR(r), tabCells)
	}
	return rows
}

// cutCR is line without the \r it ends with, its styles kept.
func cutCR(line string) string {
	if k := strings.LastIndex(line, "\r"); k >= 0 && strings.HasSuffix(stripSGR(line), "\r") {
		return line[:k] + line[k+1:]
	}
	return line
}

// lastState is line as a terminal leaves it when a \r in it sends the
// cursor back to the start: each part written over the one before, what is
// left of a longer one staying ("10%\r100%" is "100%", a progress bar);
// each character keeps the colours it was written in.
func lastState(line string) string {
	if !strings.Contains(line, "\r") {
		return line
	}
	type cell struct{ sgr, g string }
	var cells []cell
	pos, sgr := 0, ""
	walk(line, func(seq string, cells0 int) {
		switch {
		case seq == "\r":
			pos = 0
		case cells0 == 0 && isSGR(seq):
			if seq == "\x1b[0m" || seq == "\x1b[m" {
				sgr = ""
			} else {
				sgr += seq
			}
		default:
			c := cell{sgr, seq}
			if pos < len(cells) {
				cells[pos] = c
			} else {
				cells = append(cells, c)
			}
			pos++
		}
	})
	var b strings.Builder
	cur := ""
	for _, c := range cells {
		if c.sgr != cur {
			b.WriteString("\x1b[0m" + c.sgr)
			cur = c.sgr
		}
		b.WriteString(c.g)
	}
	if cur != "" {
		b.WriteString("\x1b[0m")
	}
	return b.String()
}

// readingLine is a row of reading content drawn: SGR colouring it, a tab as
// tabW spaces, any other control character and escape sequence by its name
// in Red (components, control characters).
func readingLine(row string, tabW int) string {
	if !strings.ContainsFunc(row, isControlRune) {
		return row
	}
	return redNames(visibleText(row, tabW), graphemes(stripSGR(row)), tabW)
}

// isControlRune reports a C0 or C1 control character, or DEL.
func isControlRune(r rune) bool { return r < 0x20 || r >= 0x7f && r <= 0x9f }

// Span is a piece of text in a style of the app's own, for the places an
// app colours text it gives the lib (a list's cell, a value, the statusbar):
// the text is drawn as any name or short text, a control character in it by
// its name (Escape); the style is the app's.
type Span struct {
	Text  string
	Style Style
}

// spansText is spans drawn: each piece's text escaped, in its style.
func spansText(spans []Span) string {
	var b strings.Builder
	for _, s := range spans {
		b.WriteString(s.Style.Render(Escape(s.Text)))
	}
	return b.String()
}

// spansPlain is the text of spans, as it is.
func spansPlain(spans []Span) string {
	var b strings.Builder
	for _, s := range spans {
		b.WriteString(s.Text)
	}
	return b.String()
}
