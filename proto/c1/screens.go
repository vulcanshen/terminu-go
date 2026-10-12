package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Glyphs tdp names by code point (E4: written as code points).
const (
	slantL    = "\U0000E0BA" // nf-ple-lower_right_triangle
	hard      = "\U0000E0B0" // nf-pl-left_hard_divider: slanted right end, seam, cursor arrow
	soft      = "\U0000E0B1" // nf-pl-left_soft_divider
	roundL    = "\U0000E0B6" // nf-ple-left_half_circle_thick
	roundR    = "\U0000E0B4" // nf-ple-right_half_circle_thick
	hexagon   = "\U000F02D8" // nf-md-hexagon
	sortAsc   = "\U0000F160" // nf-fa-sort_amount_asc
	sortDesc  = "\U0000F161" // nf-fa-sort_amount_desc
	marked    = "\U000F0132" // nf-md-checkbox_marked
	unchecked = "\U000F0131" // nf-md-checkbox_blank_outline
	radioOn   = "\U000F043E" // nf-md-radiobox_marked
	radioOff  = "\U000F043D" // nf-md-radiobox_blank
	gMenu     = "\U0000F0C9" // nf-fa-bars
	gKeys     = "\U0000F059" // nf-fa-question_circle
	gConfirm  = "\U000F0ECC" // nf-md-shield_alert
	gTerm     = "\U000F018D" // nf-md-console
	gInfo     = "\U0000F024" // nf-fa-flag
	gWarn     = "\U000F078F" // nf-md-alarm_light
	gError    = "\U000F0238" // nf-md-fire
	gLoading  = "\U000F0AA1" // nf-md-circle_slice_4
)

var (
	cBase     = lipgloss.Color("#1e1e2e")
	cBlue     = lipgloss.Color("#89b4fa")
	cSurface2 = lipgloss.Color("#585b70")
	cLayer1   = lipgloss.Color("#A4C0FA")
	cGreen    = lipgloss.Color("#a6e3a1")
	cOverlay0 = lipgloss.Color("#6c7086")
	cSubtext1 = lipgloss.Color("#bac2de")

	sBlue     = fg(cBlue)
	sSurface2 = fg(cSurface2)
	sDim      = fg(cOverlay0)
	sGreen    = fg(cGreen)
	sLayer    = fg(cLayer1)
	sCursor   = lipgloss.NewStyle().Background(cSubtext1).Foreground(cBase).Bold(true)
)

func fg(c lipgloss.Color) lipgloss.Style { return lipgloss.NewStyle().Foreground(c) }

// filled is text on a block of colour c: Base, bold.
func filled(c lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().Background(c).Foreground(cBase).Bold(true)
}

type popupKind int

const (
	pMenu popupKind = iota
	pKeys
	pConfirm
	pForm
	pToastInfo
	pToastWarn
	pToastError
	pTerminal
	pLoading
	pNone
)

var popupNames = []string{"menu", "keys", "confirm", "form", "toast-info", "toast-warn", "toast-error", "terminal", "loading", "none"}

type oversize int

const (
	fits oversize = iota
	wider
	taller
	both
)

var oversizeNames = []string{"fits", "wider", "taller", "both"}

// popup is what goes in a popup, independent of how it is drawn.
type popup struct {
	title  string
	rows   []string
	cursor int // row index of a whole-row cursor, -1 for none
	hint   string
	width  int  // outer width
	bare   bool // terminal: no padding
	toast  bool
}

func makePopup(k popupKind, over oversize, w, h int) popup {
	p := popup{width: 44, cursor: -1}
	switch k {
	case pMenu:
		p.title, p.hint, p.cursor = gMenu+" [2] PODS", "Enter:run Esc:close", 1
		p.rows = []string{
			hexagon + " ITEM OPERATION",
			"[Enter] Logs              follow its log",
			"[x] Delete               remove this pod",
			hexagon + " PANEL OPERATION",
			"[/] Search                  name, status",
			"Global operation  actions for the whole app",
		}
	case pKeys:
		p.title, p.hint = gKeys+" [2] PODS KEYS", "?/Esc:close"
		p.rows = []string{hexagon + " ITEM OPERATION", " Enter  Logs — follow its log", " x      Delete — remove this pod", hexagon + " CORE KEYS", " Space  what you can do here", " ?      this list"}
	case pConfirm:
		p.title, p.hint = gConfirm+" DELETE", "Enter:delete Esc:cancel"
		p.rows = []string{"web-a8k2 will be deleted.", "", "Delete web-a8k2?"}
	case pForm:
		p.title, p.hint, p.cursor = gMenu+" NEW HOST", "Enter:choose Esc:cancel", 0
		p.rows = []string{"Auth   " + radioOn + " password", "       " + radioOff + " privatekey", "Agent  " + marked + " forward", "       " + unchecked + " x11"}
	case pToastInfo, pToastWarn, pToastError:
		g := map[popupKind]string{pToastInfo: gInfo + " INFO", pToastWarn: gWarn + " WARNING", pToastError: gError + " ERROR"}[k]
		p.title, p.hint, p.toast, p.width = g, "Esc:close", true, 30
		p.rows = []string{"     Copied 3 paths"}
	case pTerminal:
		p.title, p.hint, p.bare, p.width = gTerm+" SHELL", "Alt-Esc:exit", true, w-2
		p.rows = []string{"$ ls", "README.md  go.mod  proto", "$ "}
	case pLoading:
		p.title, p.hint, p.cursor = gMenu+" SORT BY", "Esc:close", 0
		p.rows = []string{"Name", "Age"}
	}
	if over == wider || over == both {
		p.width = w + 10
	}
	if over == taller || over == both {
		for len(p.rows) < h+5 {
			p.rows = append(p.rows, "filler "+marked+" row")
		}
	}
	return p
}

// --- tdp v0.5.0-shaped drawing with the E5 width function -------------------

func hline(n int, ch string) string { return strings.Repeat(ch, max(n, 0)) }

// capsuleChain: "[2] PODS" selected, then two unselected tabs, slanted ends.
func capsuleChain(c lipgloss.Color) string {
	return fg(c).Render(slantL) + filled(c).Render("[2] PODS") +
		lipgloss.NewStyle().Foreground(c).Background(cBase).Render(hard) +
		fg(c).Render("EVENTS") + fg(c).Render(soft) + fg(c).Render("YAML") + fg(cBase).Render(hard)
}

// e5Panel draws the focused list panel: capsule chain in the top border, a
// header with sort glyphs, a marker column, a cursor row ending in an arrow,
// hint and scroll position.
func e5Panel(w, h int) []string {
	inner := w - 2
	chain := capsuleChain(cBlue)
	if dispWidth(chain) > inner {
		chain = cut(chain, 0, inner) // the prototype skips the capsule shrinking rules
	}
	out := []string{sBlue.Render("┏") + chain + sBlue.Render(hline(inner-dispWidth(chain), "━")+"┓")}
	rows := []string{
		sBlue.Render("   NAME " + sortAsc + "       STATUS     AGE " + sortDesc),
		"   nginx-7d4f    Running     3d",
		" " + sGreen.Render(marked) + " web-a8k2      CrashLoo…   2h",
		"   coredns-7w9z  Running     9d",
		"   prome-er-0    Pending     5m",
	}
	// The cursor row: Subtext1 under the whole inner width, Base bold text, the
	// right padding cell an arrow in Subtext1.
	c := 1 + cursorRow%(len(rows)-1)
	rows[c] = sCursor.Render(pad(ansiStrip(rows[c]), inner-1)) + fg(cSubtext1).Render(hard)
	for i := 0; i < h-2; i++ {
		r := ""
		if i < len(rows) {
			r = rows[i]
		}
		out = append(out, sBlue.Render("┃")+pad(r, inner)+sBlue.Render("┃"))
	}
	hint, pos := " Enter:logs ", " 3 of 40 "
	if inner-2-dispWidth(hint)-dispWidth(pos) < 0 {
		hint = ""
	}
	out = append(out, sBlue.Render("┗━"+hint+hline(inner-2-dispWidth(hint)-dispWidth(pos), "━")+pos+"━┛"))
	return out
}

func e5Side(w, h int) []string {
	inner := w - 2
	label := sSurface2.Render(slantL) + filled(cSurface2).Render("[1] KINDS") + sSurface2.Render(hard)
	if dispWidth(label) > inner {
		label = cut(label, 0, inner)
	}
	hint := " Enter:open "
	if dispWidth(hint) > inner-1 {
		hint = ""
	}
	out := []string{sSurface2.Render("┌") + label + sSurface2.Render(hline(inner-dispWidth(label), "─")+"┐")}
	rows := []string{" Pods", " Deployments", " Services"}
	for i := 0; i < h-2; i++ {
		r := ""
		if i < len(rows) {
			r = rows[i]
		}
		out = append(out, sSurface2.Render("│")+pad(r, inner)+sSurface2.Render("│"))
	}
	return append(out, sSurface2.Render("└─"+hint+hline(inner-1-dispWidth(hint), "─")+"┘"))
}

// e5Base is the whole screen: statusbar, two panels side by side, footer.
func e5Base(w, h int) []string {
	out := []string{pad(sDim.Render(" C1 prototype · icon width ")+sBlue.Render(itoa(iconWidth)), w)}
	sw := w / 3
	side, main := e5Side(sw, h-2), e5Panel(w-sw, h-2)
	for i := range side {
		out = append(out, side[i]+main[i])
	}
	return append(out, pad(sBlue.Render("Space")+sDim.Render(":menu ")+sBlue.Render("?")+sDim.Render(":help ")+sBlue.Render("q")+sDim.Render(":quit"), w))
}

// e5Popup draws a popup frame the way components/layout/popup describes.
func e5Popup(p popup) []string {
	inner := p.width - 2
	title := fg(cLayer1).Render(roundL) + filled(cLayer1).Render(p.title) + fg(cLayer1).Render(roundR)
	out := []string{sLayer.Render("┏━") + title + sLayer.Render(hline(inner-1-dispWidth(title), "━")+"┓")}
	side := sLayer.Render("┃")
	if !p.bare {
		out = append(out, side+hline(inner, " ")+side)
	}
	for i, r := range p.rows {
		switch {
		case p.bare:
			out = append(out, side+pad(r, inner)+side)
		case i == p.cursor:
			out = append(out, side+filled(cLayer1).Render(pad(" "+ansiStrip(r), inner-1))+sLayer.Render(hard)+side)
		default:
			out = append(out, side+" "+pad(r, inner-2)+" "+side)
		}
	}
	if !p.bare {
		out = append(out, side+hline(inner, " ")+side)
	}
	hint := " " + p.hint + " "
	return append(out, sLayer.Render("┗━"+hint+hline(inner-1-dispWidth(hint), "━")+"┛"))
}

func e5Screen(w, h int, k popupKind, over oversize) []string {
	base := e5Base(w, h)
	if k == pNone {
		return base
	}
	p := makePopup(k, over, w, h)
	box := e5Popup(p)
	x, y := (w-p.width)/2, (h-len(box))/2
	switch {
	case p.toast:
		y = h - 2 - len(box) // bottom frame on the third row from the bottom (F7)
	case k == pTerminal:
		y = 1 // keeps the statusbar, covers the footer
	}
	return composite(base, box, x, y, w)
}

func itoa(n int) string { return string(rune('0' + n)) }
