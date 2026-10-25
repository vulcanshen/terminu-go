package terminu

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// vt is the screen of a program running in a PTY: a small terminal emulator.
// It places every character with the lib's own width (tdp E5): a program in
// the PTY is told how wide an icon is (TERMINU__ICON_WIDTH) and lays out by
// it, so the screen has to agree, or its frames would not line up in ours.
// It answers what a family app asks when it starts (cursor position, DA1, the
// palette).
type vt struct {
	w, h      int
	main, alt [][]vtCell
	cells     [][]vtCell // the screen in use
	altOn     bool
	x, y      int
	wrapNext  bool // the last column was written: the next character wraps first
	attr      vtAttr
	saved     vtSaved
	savedMain vtSaved
	top, bot  int // the scroll region, inclusive
	autowrap  bool
	cursorOn  bool
	appCursor bool // arrow keys send ESC O A rather than ESC [ A
	bracketed bool // pastes come between ESC [200~ and ESC [201~
	origin    bool
	insert    bool
	gl, g0    byte // the character sets: 'B' ASCII, '0' line drawing
	history   [][]vtCell
	title     string

	// The parser.
	state    int
	params   [][]int
	cur      int
	hasCur   bool
	joinNext bool // the last parameter ended with a colon
	private  byte
	inter    []byte
	osc      []byte
	utf      []byte
	reply    []byte // answers to send back to the program
}

type vtCell struct {
	s    string // the character; "" for a blank and the right half of a wide one
	w    int    // 1, 2, or 0 for the right half of a wide one
	attr vtAttr
}

type vtColor struct {
	kind uint8 // 0 the terminal's own, 1 indexed, 2 RGB
	idx  uint8
	rgb  RGB
}

type vtAttr struct {
	fg, bg                                                  vtColor
	bold, faint, italic, underline, reverse, hidden, strike bool
}

type vtSaved struct {
	x, y     int
	attr     vtAttr
	wrapNext bool
	origin   bool
	g0       byte
}

const historyLines = 2000

const (
	vtGround = iota
	vtEsc
	vtEscCharset
	vtCSI
	vtOSC
	vtOSCEsc
	vtStr
	vtStrEsc
)

func newVT(w, h int) *vt {
	t := &vt{autowrap: true, cursorOn: true, gl: 'B', g0: 'B'}
	t.w, t.h = max(w, 1), max(h, 1)
	t.main, t.alt = blankScreen(t.w, t.h), blankScreen(t.w, t.h)
	t.cells = t.main
	t.top, t.bot = 0, t.h-1
	return t
}

func blankScreen(w, h int) [][]vtCell {
	s := make([][]vtCell, h)
	for i := range s {
		s[i] = blankLine(w, vtAttr{})
	}
	return s
}

func blankLine(w int, a vtAttr) []vtCell {
	l := make([]vtCell, w)
	for i := range l {
		l[i] = vtCell{w: 1, attr: vtAttr{bg: a.bg}}
	}
	return l
}

// resize keeps what fits; a cursor below the new bottom takes the lines above
// it up with it.
func (t *vt) resize(w, h int) {
	w, h = max(w, 1), max(h, 1)
	if w == t.w && h == t.h {
		return
	}
	fit := func(s [][]vtCell, inUse, main bool) [][]vtCell {
		if inUse && t.y >= h {
			shift := t.y - h + 1
			if main {
				t.history = append(t.history, s[:shift]...)
			}
			s = s[shift:]
		}
		out := make([][]vtCell, h)
		for i := range out {
			if i < len(s) {
				l := s[i]
				if len(l) >= w {
					l = l[:w]
				} else {
					l = append(l, blankLine(w-len(l), vtAttr{})...)
				}
				out[i] = l
			} else {
				out[i] = blankLine(w, vtAttr{})
			}
		}
		return out
	}
	shifted := t.y >= h
	t.main = fit(t.main, !t.altOn, true)
	t.alt = fit(t.alt, t.altOn, false)
	if t.altOn {
		t.cells = t.alt
	} else {
		t.cells = t.main
	}
	if shifted {
		t.y = h - 1
	}
	t.w, t.h = w, h
	t.x = min(t.x, w-1)
	t.top, t.bot = 0, h-1
	t.wrapNext = false
	if len(t.history) > historyLines {
		t.history = t.history[len(t.history)-historyLines:]
	}
}

// write feeds what the program printed.
func (t *vt) write(b []byte) {
	for _, c := range b {
		t.step(c)
	}
}

func (t *vt) step(c byte) {
	switch t.state {
	case vtGround:
		t.ground(c)
	case vtEsc:
		t.escape(c)
	case vtEscCharset:
		if t.inter != nil && len(t.inter) > 0 && t.inter[0] == '(' {
			t.g0 = c
			t.gl = c
		}
		t.state = vtGround
	case vtCSI:
		t.csiByte(c)
	case vtOSC:
		switch c {
		case 0x07:
			t.oscDone()
		case 0x1b:
			t.state = vtOSCEsc
		default:
			if len(t.osc) < 4096 {
				t.osc = append(t.osc, c)
			}
		}
	case vtOSCEsc:
		t.oscDone()
		if c != '\\' {
			t.state = vtEsc
			t.escape(c)
		}
	case vtStr:
		switch c {
		case 0x1b:
			t.state = vtStrEsc
		case 0x07:
			t.state = vtGround
		}
	case vtStrEsc:
		if c == '\\' {
			t.state = vtGround
		} else {
			t.state = vtStr
		}
	}
}

func (t *vt) ground(c byte) {
	if len(t.utf) > 0 {
		if c&0xC0 == 0x80 {
			t.utf = append(t.utf, c)
			if utf8.FullRune(t.utf) {
				r, _ := utf8.DecodeRune(t.utf)
				t.utf = t.utf[:0]
				t.put(r)
			}
			return
		}
		t.utf = t.utf[:0]
		t.put(utf8.RuneError)
	}
	switch {
	case c < 0x20:
		t.control(c)
	case c == 0x7f:
	case c < 0x80:
		t.put(rune(c))
	default:
		t.utf = append(t.utf, c)
	}
}

func (t *vt) control(c byte) {
	switch c {
	case 0x08:
		if t.wrapNext {
			t.wrapNext = false
		} else if t.x > 0 {
			t.x--
		}
	case 0x09:
		t.x = min((t.x/8+1)*8, t.w-1)
		t.wrapNext = false
	case 0x0a, 0x0b, 0x0c:
		t.lineFeed()
	case 0x0d:
		t.x, t.wrapNext = 0, false
	case 0x0e:
		t.gl = 'B' // G1 is never designated: shifting to it keeps ASCII
	case 0x0f:
		t.gl = t.g0
	case 0x18, 0x1a:
		t.state = vtGround
	case 0x1b:
		t.state = vtEsc
		t.inter = t.inter[:0]
	}
}

func (t *vt) escape(c byte) {
	t.state = vtGround
	switch c {
	case '[':
		t.state = vtCSI
		t.params, t.cur, t.hasCur, t.private = t.params[:0], 0, false, 0
		t.inter = t.inter[:0]
	case ']':
		t.state, t.osc = vtOSC, t.osc[:0]
	case 'P', 'X', '^', '_':
		t.state = vtStr
	case '(', ')', '*', '+':
		t.state = vtEscCharset
		t.inter = append(t.inter[:0], c)
	case '7':
		t.saved = vtSaved{t.x, t.y, t.attr, t.wrapNext, t.origin, t.g0}
	case '8':
		t.restore(t.saved)
	case 'D':
		t.lineFeed()
	case 'E':
		t.x = 0
		t.lineFeed()
	case 'M':
		t.wrapNext = false
		if t.y == t.top {
			t.scrollDown(1)
		} else if t.y > 0 {
			t.y--
		}
	case 'c':
		w, h, hist := t.w, t.h, t.history
		*t = *newVT(w, h)
		t.history = hist
	default:
		if c < 0x20 {
			t.control(c)
		}
	}
}

func (t *vt) restore(s vtSaved) {
	t.x, t.y = min(s.x, t.w-1), min(s.y, t.h-1)
	t.attr, t.wrapNext, t.origin, t.g0, t.gl = s.attr, s.wrapNext, s.origin, s.g0, s.g0
}

func (t *vt) csiByte(c byte) {
	switch {
	case c >= '0' && c <= '9':
		t.cur = min(t.cur*10+int(c-'0'), 1<<16)
		t.hasCur = true
	case c == ';':
		t.pushParam(false)
	case c == ':':
		t.pushParam(true)
	case c >= '<' && c <= '?' && len(t.params) == 0 && !t.hasCur:
		t.private = c
	case c >= 0x20 && c <= 0x2f:
		t.inter = append(t.inter, c)
	case c >= 0x40 && c <= 0x7e:
		if t.hasCur || len(t.params) > 0 {
			t.pushParam(false)
		}
		t.state = vtGround
		t.dispatchCSI(c)
	case c < 0x20:
		t.control(c)
		if c != 0x1b && c != 0x18 && c != 0x1a {
			t.state = vtCSI
		}
	}
}

// pushParam ends one parameter; sub says it continues the same group (a
// colon, as in 38:2::r:g:b).
func (t *vt) pushParam(sub bool) {
	v := t.cur
	if !t.hasCur {
		v = -1 // left out
	}
	if t.joinNext && len(t.params) > 0 {
		last := len(t.params) - 1
		t.params[last] = append(t.params[last], v)
	} else {
		t.params = append(t.params, []int{v})
	}
	t.joinNext = sub
	t.cur, t.hasCur = 0, false
}

// param is parameter i, or def when it is left out or zero.
func (t *vt) param(i, def int) int {
	if i >= len(t.params) || t.params[i][0] <= 0 {
		return def
	}
	return t.params[i][0]
}

func (t *vt) dispatchCSI(c byte) {
	defer func() { t.joinNext = false }()
	if t.private == '?' {
		switch c {
		case 'h', 'l':
			for _, p := range t.params {
				t.decMode(p[0], c == 'h')
			}
		}
		return
	}
	if t.private == '>' {
		if c == 'c' {
			t.reply = append(t.reply, "\x1b[>1;10;0c"...)
		}
		return
	}
	if t.private != 0 || len(t.inter) > 0 {
		return
	}
	n := t.param(0, 1)
	switch c {
	case '@':
		t.insertChars(n)
	case 'A':
		t.y = max(t.y-n, t.regionTop())
		t.wrapNext = false
	case 'B', 'e':
		t.y = min(t.y+n, t.regionBottom())
		t.wrapNext = false
	case 'C', 'a':
		t.x = min(t.x+n, t.w-1)
		t.wrapNext = false
	case 'D':
		t.x = max(t.x-n, 0)
		t.wrapNext = false
	case 'E':
		t.x, t.y = 0, min(t.y+n, t.regionBottom())
		t.wrapNext = false
	case 'F':
		t.x, t.y = 0, max(t.y-n, t.regionTop())
		t.wrapNext = false
	case 'G', '`':
		t.x = min(max(n-1, 0), t.w-1)
		t.wrapNext = false
	case 'd':
		t.moveTo(t.x, n-1)
	case 'H', 'f':
		t.moveTo(t.param(1, 1)-1, t.param(0, 1)-1)
	case 'J':
		t.eraseDisplay(t.param(0, 0))
	case 'K':
		t.eraseLine(t.param(0, 0))
	case 'L':
		t.insertLines(n)
	case 'M':
		t.deleteLines(n)
	case 'P':
		t.deleteChars(n)
	case 'S':
		t.scrollUp(n)
	case 'T':
		t.scrollDown(n)
	case 'X':
		for i := t.x; i < min(t.x+n, t.w); i++ {
			t.cells[t.y][i] = vtCell{w: 1, attr: vtAttr{bg: t.attr.bg}}
		}
	case 'm':
		t.sgr()
	case 'r':
		top, bot := t.param(0, 1)-1, t.param(1, t.h)-1
		if top < bot && bot < t.h {
			t.top, t.bot = top, bot
			t.moveTo(0, 0)
		}
	case 's':
		t.saved = vtSaved{t.x, t.y, t.attr, t.wrapNext, t.origin, t.g0}
	case 'u':
		t.restore(t.saved)
	case 'h', 'l':
		for _, p := range t.params {
			if p[0] == 4 {
				t.insert = c == 'h'
			}
		}
	case 'n':
		switch t.param(0, 0) {
		case 5:
			t.reply = append(t.reply, "\x1b[0n"...)
		case 6:
			y := t.y
			if t.origin {
				y -= t.top
			}
			t.reply = append(t.reply, fmt.Sprintf("\x1b[%d;%dR", y+1, t.x+1)...)
		}
	case 'c':
		if t.param(0, 0) == 0 {
			t.reply = append(t.reply, "\x1b[?62;22c"...)
		}
	case 't':
		if t.param(0, 0) == 18 {
			t.reply = append(t.reply, fmt.Sprintf("\x1b[8;%d;%dt", t.h, t.w)...)
		}
	}
}

func (t *vt) regionTop() int {
	if t.y >= t.top {
		return t.top
	}
	return 0
}

func (t *vt) regionBottom() int {
	if t.y <= t.bot {
		return t.bot
	}
	return t.h - 1
}

func (t *vt) moveTo(x, y int) {
	if t.origin {
		y += t.top
		y = min(max(y, t.top), t.bot)
	}
	t.x, t.y = min(max(x, 0), t.w-1), min(max(y, 0), t.h-1)
	t.wrapNext = false
}

func (t *vt) decMode(m int, on bool) {
	switch m {
	case 1:
		t.appCursor = on
	case 6:
		t.origin = on
		t.moveTo(0, 0)
	case 7:
		t.autowrap = on
	case 25:
		t.cursorOn = on
	case 47, 1047, 1049:
		if on == t.altOn {
			return
		}
		if m == 1049 && on {
			t.savedMain = vtSaved{t.x, t.y, t.attr, t.wrapNext, t.origin, t.g0}
		}
		t.altOn = on
		if on {
			t.alt = blankScreen(t.w, t.h)
			t.cells = t.alt
		} else {
			t.cells = t.main
			if m == 1049 {
				t.restore(t.savedMain)
			}
		}
		t.top, t.bot = 0, t.h-1
	case 2004:
		t.bracketed = on
	}
}

// lineDrawing is the DEC special graphics set, for programs that draw boxes
// with it.
var lineDrawing = map[rune]rune{
	'`': '◆', 'a': '▒', 'f': '°', 'g': '±', 'j': '┘', 'k': '┐', 'l': '┌', 'm': '└', 'n': '┼', 'o': '⎺', 'p': '⎻',
	'q': '─', 'r': '⎼', 's': '⎽', 't': '├', 'u': '┤', 'v': '┴', 'w': '┬', 'x': '│', 'y': '≤', 'z': '≥', '{': 'π',
	'|': '≠', '}': '£', '~': '·',
}

// zeroWidth reports a character that joins the one before it: a combining
// mark, a variation selector, a zero-width joiner.
func zeroWidth(r rune) bool {
	return unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) || r == 0x200d || r >= 0xfe00 && r <= 0xfe0f
}

// put prints one character at the cursor.
func (t *vt) put(r rune) {
	if t.gl == '0' {
		if m, ok := lineDrawing[r]; ok {
			r = m
		}
	}
	if zeroWidth(r) || t.afterJoiner() {
		if c := t.prevCell(); c != nil {
			c.s += string(r)
			return
		}
		if zeroWidth(r) {
			return
		}
	}
	w := min(Width(string(r)), 2)
	if w <= 0 {
		return
	}
	if t.wrapNext && t.autowrap {
		t.x = 0
		t.lineFeed()
	}
	t.wrapNext = false
	if t.x+w > t.w {
		if !t.autowrap {
			t.x = t.w - w
		} else {
			t.x = 0
			t.lineFeed()
		}
	}
	line := t.cells[t.y]
	if t.insert {
		copy(line[t.x+w:], line[t.x:])
	}
	t.clearWide(t.x)
	if w == 2 {
		t.clearWide(t.x + 1)
	}
	line[t.x] = vtCell{s: string(r), w: w, attr: t.attr}
	if w == 2 {
		line[t.x+1] = vtCell{w: 0, attr: t.attr}
	}
	t.x += w
	if t.x >= t.w {
		t.x = t.w - 1
		t.wrapNext = t.autowrap
	}
}

// prevCell is the character just printed, which a combining mark joins.
func (t *vt) prevCell() *vtCell {
	x := t.x
	if !t.wrapNext {
		x--
	}
	if x < 0 {
		return nil
	}
	if t.cells[t.y][x].w == 0 && x > 0 {
		x--
	}
	return &t.cells[t.y][x]
}

func (t *vt) afterJoiner() bool {
	c := t.prevCell()
	return c != nil && strings.HasSuffix(c.s, "\U0000200D")
}

// clearWide blanks the other half of a wide character that x overwrites.
func (t *vt) clearWide(x int) {
	if x >= t.w {
		return
	}
	line := t.cells[t.y]
	switch {
	case line[x].w == 0 && x > 0:
		line[x-1] = vtCell{w: 1, attr: line[x-1].attr}
	case line[x].w == 2 && x+1 < t.w:
		line[x+1] = vtCell{w: 1, attr: line[x+1].attr}
	}
}

func (t *vt) lineFeed() {
	t.wrapNext = false
	switch {
	case t.y == t.bot:
		t.scrollUp(1)
	case t.y < t.h-1:
		t.y++
	}
}

// scrollUp moves the scroll region up n lines; lines leaving the top of the
// main screen go into the history.
func (t *vt) scrollUp(n int) {
	n = min(n, t.bot-t.top+1)
	if t.top == 0 && !t.altOn {
		t.history = append(t.history, t.cells[:n]...)
		if len(t.history) > historyLines {
			t.history = t.history[len(t.history)-historyLines:]
		}
	}
	copy(t.cells[t.top:], t.cells[t.top+n:t.bot+1])
	for i := t.bot - n + 1; i <= t.bot; i++ {
		t.cells[i] = blankLine(t.w, t.attr)
	}
}

func (t *vt) scrollDown(n int) {
	n = min(n, t.bot-t.top+1)
	copy(t.cells[t.top+n:t.bot+1], t.cells[t.top:t.bot+1-n])
	for i := t.top; i < t.top+n; i++ {
		t.cells[i] = blankLine(t.w, t.attr)
	}
}

func (t *vt) insertLines(n int) {
	if t.y < t.top || t.y > t.bot {
		return
	}
	top := t.top
	t.top = t.y
	t.scrollDown(n)
	t.top = top
	t.x = 0
}

func (t *vt) deleteLines(n int) {
	if t.y < t.top || t.y > t.bot {
		return
	}
	top, alt := t.top, t.altOn
	t.top, t.altOn = t.y, true // deleted lines do not go into the history
	t.scrollUp(n)
	t.top, t.altOn = top, alt
	t.x = 0
}

func (t *vt) insertChars(n int) {
	line := t.cells[t.y]
	n = min(n, t.w-t.x)
	copy(line[t.x+n:], line[t.x:])
	for i := t.x; i < t.x+n; i++ {
		line[i] = vtCell{w: 1, attr: vtAttr{bg: t.attr.bg}}
	}
}

func (t *vt) deleteChars(n int) {
	line := t.cells[t.y]
	n = min(n, t.w-t.x)
	copy(line[t.x:], line[t.x+n:])
	for i := t.w - n; i < t.w; i++ {
		line[i] = vtCell{w: 1, attr: vtAttr{bg: t.attr.bg}}
	}
}

func (t *vt) eraseLine(mode int) {
	from, to := t.x, t.w
	switch mode {
	case 1:
		from, to = 0, t.x+1
	case 2:
		from = 0
	}
	for i := from; i < min(to, t.w); i++ {
		t.cells[t.y][i] = vtCell{w: 1, attr: vtAttr{bg: t.attr.bg}}
	}
}

func (t *vt) eraseDisplay(mode int) {
	switch mode {
	case 0:
		t.eraseLine(0)
		for y := t.y + 1; y < t.h; y++ {
			t.cells[y] = blankLine(t.w, t.attr)
		}
	case 1:
		t.eraseLine(1)
		for y := 0; y < t.y; y++ {
			t.cells[y] = blankLine(t.w, t.attr)
		}
	case 2:
		for y := range t.cells {
			t.cells[y] = blankLine(t.w, t.attr)
		}
	case 3:
		t.history = nil
	}
}

// sgr sets the drawing attributes.
func (t *vt) sgr() {
	if len(t.params) == 0 {
		t.attr = vtAttr{}
		return
	}
	for i := 0; i < len(t.params); i++ {
		p := t.params[i]
		v := max(p[0], 0)
		switch {
		case v == 0:
			t.attr = vtAttr{}
		case v == 1:
			t.attr.bold = true
		case v == 2:
			t.attr.faint = true
		case v == 3:
			t.attr.italic = true
		case v == 4:
			t.attr.underline = len(p) < 2 || p[1] != 0
		case v == 7:
			t.attr.reverse = true
		case v == 8:
			t.attr.hidden = true
		case v == 9:
			t.attr.strike = true
		case v == 22:
			t.attr.bold, t.attr.faint = false, false
		case v == 23:
			t.attr.italic = false
		case v == 24:
			t.attr.underline = false
		case v == 27:
			t.attr.reverse = false
		case v == 28:
			t.attr.hidden = false
		case v == 29:
			t.attr.strike = false
		case v >= 30 && v <= 37:
			t.attr.fg = vtColor{kind: 1, idx: uint8(v - 30)}
		case v >= 90 && v <= 97:
			t.attr.fg = vtColor{kind: 1, idx: uint8(v - 90 + 8)}
		case v == 39:
			t.attr.fg = vtColor{}
		case v >= 40 && v <= 47:
			t.attr.bg = vtColor{kind: 1, idx: uint8(v - 40)}
		case v >= 100 && v <= 107:
			t.attr.bg = vtColor{kind: 1, idx: uint8(v - 100 + 8)}
		case v == 49:
			t.attr.bg = vtColor{}
		case v == 38 || v == 48 || v == 58:
			var c vtColor
			var used int
			if len(p) > 1 {
				c, _ = extColor(p[1:], true)
			} else {
				var rest []int
				for _, q := range t.params[i+1:] {
					rest = append(rest, q[0])
				}
				c, used = extColor(rest, false)
				i += used
			}
			switch v {
			case 38:
				t.attr.fg = c
			case 48:
				t.attr.bg = c
			}
		}
	}
}

// extColor reads 5;n or 2;r;g;b; with colons the colour space may sit before
// r;g;b. It reports how many parameters it took.
func extColor(p []int, colon bool) (vtColor, int) {
	at := func(i int) int {
		if i < len(p) {
			return max(p[i], 0)
		}
		return 0
	}
	switch at(0) {
	case 5:
		return vtColor{kind: 1, idx: uint8(at(1))}, 2
	case 2:
		if colon && len(p) >= 5 {
			return vtColor{kind: 2, rgb: RGB{uint8(at(2)), uint8(at(3)), uint8(at(4))}}, 5
		}
		return vtColor{kind: 2, rgb: RGB{uint8(at(1)), uint8(at(2)), uint8(at(3))}}, 4
	}
	return vtColor{}, 1
}

func (t *vt) oscDone() {
	t.state = vtGround
	body := string(t.osc)
	num, rest, _ := strings.Cut(body, ";")
	switch num {
	case "0", "2":
		t.title = rest
	case "4":
		parts := strings.Split(rest, ";")
		for i := 0; i+1 < len(parts); i += 2 {
			n, err := strconv.Atoi(parts[i])
			if err != nil || parts[i+1] != "?" || n < 0 || n > 255 {
				continue
			}
			c := vtColor{kind: 1, idx: uint8(n)}.rgbOr(RGB{})
			t.reply = append(t.reply, fmt.Sprintf("\x1b]4;%d;%s\x1b\\", n, xColor(c))...)
		}
	case "10", "11":
		if rest == "?" {
			c := Text.RGB
			if num == "11" {
				c = Base.RGB
			}
			t.reply = append(t.reply, fmt.Sprintf("\x1b]%s;%s\x1b\\", num, xColor(c))...)
		}
	}
}

func xColor(c RGB) string {
	return fmt.Sprintf("rgb:%02x%02x/%02x%02x/%02x%02x", c[0], c[0], c[1], c[1], c[2], c[2])
}

// rgbOr is the colour as 24-bit, def for the terminal's own.
func (c vtColor) rgbOr(def RGB) RGB {
	switch c.kind {
	case 1:
		if c.idx < 16 {
			return palette16[c.idx]
		}
		return color256(int(c.idx))
	case 2:
		return c.rgb
	}
	return def
}

// render draws the screen as h lines of w cells, scrolled back by back lines
// into the history; cursor draws the program's cursor.
func (t *vt) render(back int, cursor bool, cursorStyle Style) []string {
	back = min(max(back, 0), len(t.history))
	out := make([]string, 0, t.h)
	for row := range t.h {
		var line []vtCell
		src := row - back
		if src < 0 {
			line = t.history[len(t.history)+src]
		} else {
			line = t.cells[src]
		}
		cx := -1
		if cursor && back == 0 && src == t.y && t.cursorOn {
			cx = t.x
		}
		out = append(out, renderCells(line, t.w, cx, cursorStyle))
	}
	return out
}

func renderCells(line []vtCell, w, cursorX int, cursorStyle Style) string {
	var b strings.Builder
	used := 0
	last := ""
	for x := 0; x < len(line) && used < w; x++ {
		c := line[x]
		if c.w == 0 {
			continue
		}
		text := c.s
		if text == "" || c.attr.hidden {
			text = strings.Repeat(" ", c.w)
		}
		if used+c.w > w {
			text = " "
		}
		if x == cursorX {
			if last != "" {
				b.WriteString("\x1b[0m")
				last = ""
			}
			b.WriteString(cursorStyle.Render(text))
			used += c.w
			continue
		}
		if sgr := c.attr.sgr(); sgr != last {
			if last != "" {
				b.WriteString("\x1b[0m")
			}
			b.WriteString(sgr)
			last = sgr
		}
		b.WriteString(text)
		used += c.w
	}
	if last != "" {
		b.WriteString("\x1b[0m")
	}
	if used < w {
		b.WriteString(strings.Repeat(" ", w-used))
	}
	return b.String()
}

// sgr is the escape that sets the attributes, in 24-bit colour (dimming
// needs it, tdp color).
func (a vtAttr) sgr() string {
	if a == (vtAttr{}) {
		return ""
	}
	var codes []string
	fg, bg := a.fg, a.bg
	if a.reverse {
		f, g := fg.rgbOr(Text.RGB), bg.rgbOr(Base.RGB)
		fg, bg = vtColor{kind: 2, rgb: g}, vtColor{kind: 2, rgb: f}
	}
	if a.bold {
		codes = append(codes, "1")
	}
	if a.faint {
		codes = append(codes, "2")
	}
	if a.italic {
		codes = append(codes, "3")
	}
	if a.underline {
		codes = append(codes, "4")
	}
	if a.strike {
		codes = append(codes, "9")
	}
	if fg.kind != 0 {
		c := fg.rgbOr(RGB{})
		codes = append(codes, fmt.Sprintf("38;2;%d;%d;%d", c[0], c[1], c[2]))
	}
	if bg.kind != 0 {
		c := bg.rgbOr(RGB{})
		codes = append(codes, fmt.Sprintf("48;2;%d;%d;%d", c[0], c[1], c[2]))
	}
	if len(codes) == 0 {
		return ""
	}
	return "\x1b[" + strings.Join(codes, ";") + "m"
}

// text is what the screen says, for tests: the characters of each line, the
// trailing blanks cut.
func (t *vt) text() []string {
	out := make([]string, t.h)
	for y, l := range t.cells {
		var b strings.Builder
		for _, c := range l {
			switch {
			case c.w == 0:
			case c.s == "":
				b.WriteString(" ")
			default:
				b.WriteString(c.s)
			}
		}
		out[y] = strings.TrimRight(b.String(), " ")
	}
	return out
}
