package main

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// Glyphs tdp names by code point (E4: written as code points).
const (
	capL      = "\ue0b6"     // nf-ple-left_half_circle_thick
	capR      = "\ue0b4"     // nf-ple-right_half_circle_thick
	seamHard  = "\ue0b0"     // nf-pl-left_hard_divider
	seamSoft  = "\ue0b1"     // nf-pl-left_soft_divider
	sortAsc   = "\uf160"     // nf-fa-sort_amount_asc
	sortDesc  = "\uf161"     // nf-fa-sort_amount_desc
	marked    = "\U000F0132" // nf-md-checkbox_marked
	unchecked = "\U000F0131" // nf-md-checkbox_blank_outline
	radioOn   = "\U000F043E" // nf-md-radiobox_marked
	radioOff  = "\U000F043D" // nf-md-radiobox_blank
	gMenu     = "\uf0c9"     // nf-fa-bars
	gKeys     = "\uf059"     // nf-fa-question_circle
	gConfirm  = "\U000F0ECC" // nf-md-shield_alert
	gTerm     = "\U000F018D" // nf-md-console
	gInfo     = "\uf024"     // nf-fa-flag
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

	sBlue    = lipgloss.NewStyle().Foreground(cBlue)
	sDim     = lipgloss.NewStyle().Foreground(cOverlay0)
	sGreen   = lipgloss.NewStyle().Foreground(cGreen)
	sLayer   = lipgloss.NewStyle().Foreground(cLayer1)
	sCursor  = lipgloss.NewStyle().Background(cSubtext1).Foreground(cBase).Bold(true)
	sCapsule = lipgloss.NewStyle().Background(cBlue).Foreground(cBase).Bold(true)
)

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
	title string
	rows  []string
	hint  string
	width int  // outer width
	bare  bool // terminal: no blank margin rows
	toast bool
}

func makePopup(k popupKind, over oversize, w, h int) popup {
	p := popup{width: 44}
	switch k {
	case pMenu:
		p.title, p.hint = gMenu+" [2] Pods", "Enter:run Esc:close"
		p.rows = []string{"item operation", "[Enter] Logs       follow its log", "[d]elete    remove this pod", "panel operation", "[/] Search   name, status", "Global operation   actions for the whole app"}
	case pKeys:
		p.title, p.hint = gKeys+" [2] Pods keys", "?/Esc:close"
		p.rows = []string{"item operation", "  Enter  logs — follow its log", "  d      delete — remove this pod", "core keys", "  Space  what you can do here", "  ?      this list"}
	case pConfirm:
		p.title, p.hint = gConfirm+" Delete", "Enter:delete Esc:cancel"
		p.rows = []string{"web-a8k2 will be deleted.", "", "Delete web-a8k2?"}
	case pForm:
		p.title, p.hint = gMenu+" New host", "Enter:edit Esc:cancel"
		p.rows = []string{"Auth    " + radioOn + " password", "        " + radioOff + " privatekey", "Agent   " + marked + " forward", "        " + unchecked + " x11"}
	case pToastInfo, pToastWarn, pToastError:
		g := map[popupKind]string{pToastInfo: gInfo + " Info", pToastWarn: gWarn + " Warning", pToastError: gError + " Error"}[k]
		p.title, p.hint, p.toast, p.width = g, "Esc:close", true, 30
		p.rows = []string{"   Copied 3 paths"}
	case pTerminal:
		p.title, p.hint, p.bare, p.width = gTerm+" Shell", "Alt-Esc:exit", true, w-2
		p.rows = []string{"$ ls", "README.md  go.mod  proto", "$ "}
	case pLoading:
		p.title, p.hint = gMenu+" Sort by "+gLoading, "Esc:close"
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

// --- tdp-shaped drawing with the E5 width function -------------------------

func hline(n int, ch string) string { return strings.Repeat(ch, max(n, 0)) }

// capsuleChain draws "[2] Pods" selected, then two unselected tabs.
func capsuleChain() string {
	return sBlue.Render(capL) + sCapsule.Render("[2] Pods") + sBlue.Render(seamHard) +
		sBlue.Render("Logs") + sBlue.Render(seamSoft) + sBlue.Render("Events") + sBlue.Render(capR)
}

// e5Panel draws the focused list panel: capsule chain in the top border, a
// header with sort glyphs, a marker column, a cursor row, hint and position.
func e5Panel(w, h int) []string {
	inner := w - 2
	chain := capsuleChain()
	if dispWidth(chain) > inner {
		chain = cut(chain, 0, inner) // the prototype skips the capsule shrinking rules
	}
	out := []string{sBlue.Render("╔") + chain + sBlue.Render(hline(inner-dispWidth(chain), "═")+"╗")}
	rows := []string{
		sBlue.Render("   Name " + sortAsc + "          Status   Age " + sortDesc),
		"   nginx-7d4f       Running   3d",
		" " + sGreen.Render(marked) + " web-a8k2         CrashLoo…  2h",
		"   coredns-7w9z     Running   9d",
		"   prome-er-0       Pending   5m",
	}
	// The cursor row: Subtext1 under the whole inner width, Base bold text.
	c := 1 + cursorRow%(len(rows)-1)
	rows[c] = sCursor.Render(pad(ansiStrip(rows[c]), inner))
	for i := 0; i < h-2; i++ {
		r := ""
		if i < len(rows) {
			r = rows[i]
		}
		out = append(out, sBlue.Render("║")+pad(r, inner)+sBlue.Render("║"))
	}
	hint, pos := " Enter:logs ", " 3 of 40 "
	if inner-1-dispWidth(hint)-dispWidth(pos) < 0 {
		hint = ""
	}
	out = append(out, sBlue.Render("╚═"+hint+hline(inner-1-dispWidth(hint)-dispWidth(pos), "═")+pos+"╝"))
	return out
}

func e5Side(w, h int) []string {
	inner := w - 2
	label := sSurface2().Render(capL) + lipgloss.NewStyle().Background(cSurface2).Foreground(cBase).Render("[1] Kinds") + sSurface2().Render(capR)
	if dispWidth(label) > inner {
		label = cut(label, 0, inner)
	}
	hint := " Enter:open "
	if dispWidth(hint) > inner-1 {
		hint = ""
	}
	out := []string{sSurface2().Render("╭") + label + sSurface2().Render(hline(inner-dispWidth(label), "─")+"╮")}
	rows := []string{" Pods", " Deployments", " Services"}
	for i := 0; i < h-2; i++ {
		r := ""
		if i < len(rows) {
			r = rows[i]
		}
		out = append(out, sSurface2().Render("│")+pad(r, inner)+sSurface2().Render("│"))
	}
	return append(out, sSurface2().Render("╰─"+hint+hline(inner-1-dispWidth(hint), "─")+"╯"))
}

func sSurface2() lipgloss.Style { return lipgloss.NewStyle().Foreground(cSurface2) }

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
	title := " " + p.title + " "
	out := []string{sLayer.Render("╭─") + sLayer.Bold(true).Render(title) + sLayer.Render(hline(inner-1-dispWidth(title), "─")+"╮")}
	side := sLayer.Render("│")
	if !p.bare {
		out = append(out, side+hline(inner, " ")+side)
	}
	for _, r := range p.rows {
		if p.bare {
			out = append(out, side+pad(r, inner)+side)
		} else {
			out = append(out, side+" "+pad(r, inner-2)+" "+side)
		}
	}
	if !p.bare {
		out = append(out, side+hline(inner, " ")+side)
	}
	hint := " " + p.hint + " "
	return append(out, sLayer.Render("╰─"+hint+hline(inner-1-dispWidth(hint), "─")+"╯"))
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
		y = h - 2 - len(box) // bottom frame sits on the panels' bottom border
	case k == pTerminal:
		y = 1 // keeps the statusbar, covers the footer
	}
	return composite(base, box, x, y, w)
}

// --- the same screen drawn the lipgloss v2 way -----------------------------

func lgPanel(w, h int, focused bool) string {
	b, c := lipgloss.RoundedBorder(), cSurface2
	rows := []string{capL + "[1] Kinds" + capR, " Pods", " Deployments", " Services"}
	if focused {
		b, c = lipgloss.DoubleBorder(), cBlue
		rows = []string{
			capsuleChain(),
			sBlue.Render("   Name " + sortAsc + "          Status   Age " + sortDesc),
			"   nginx-7d4f       Running   3d",
			" " + sGreen.Render(marked) + " web-a8k2         CrashLoo…  2h",
			"   coredns-7w9z     Running   9d",
		}
	}
	return lipgloss.NewStyle().Border(b).BorderForeground(c).Width(w).Height(h).Render(strings.Join(rows, "\n"))
}

func lgPopup(p popup) string {
	content := append([]string{sLayer.Bold(true).Render(p.title)}, p.rows...)
	content = append(content, sDim.Render(p.hint))
	st := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(cLayer1).Width(p.width)
	if !p.bare {
		st = st.Padding(1, 1)
	}
	return st.Render(strings.Join(content, "\n"))
}

func lgScreen(w, h int, k popupKind, over oversize) []string {
	status := lipgloss.NewStyle().Width(w).Render(sDim.Render(" C1 prototype · icon width ") + sBlue.Render(itoa(iconWidth)))
	panels := lipgloss.JoinHorizontal(lipgloss.Top, lgPanel(w/3, h-2, false), lgPanel(w-w/3, h-2, true))
	footer := lipgloss.NewStyle().Width(w).Render(sBlue.Render("Space") + sDim.Render(":menu ") + sBlue.Render("?") + sDim.Render(":help ") + sBlue.Render("q") + sDim.Render(":quit"))
	// Layer X/Y only count through a Compositor; Canvas.Compose draws a bare
	// layer over the whole canvas.
	comp := lipgloss.NewCompositor(lipgloss.NewLayer(lipgloss.JoinVertical(lipgloss.Left, status, panels, footer)))
	if k != pNone {
		p := makePopup(k, over, w, h)
		box := lgPopup(p)
		x, y := (w-lipgloss.Width(box))/2, (h-lipgloss.Height(box))/2
		switch {
		case p.toast:
			y = h - 2 - lipgloss.Height(box)
		case k == pTerminal:
			y = 1
		}
		comp.AddLayers(lipgloss.NewLayer(box).X(max(x, 0)).Y(max(y, 0)).Z(1))
	}
	canvas := lipgloss.NewCanvas(w, h)
	canvas.Compose(comp)
	lines := strings.Split(canvas.Render(), "\n")
	// Canvas.Render trims trailing spaces; pad back with lipgloss's own
	// measure, so lipgloss gets the most favourable reading.
	for i, l := range lines {
		if d := lipgloss.Width(l); d < w {
			lines[i] = l + strings.Repeat(" ", w-d)
		}
	}
	return lines
}

func itoa(n int) string { return string(rune('0' + n)) }
