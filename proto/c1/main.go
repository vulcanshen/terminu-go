package main

// Usage:
//
//	c1 dump <pipeline> <W> <H> <popup> <oversize>   print one frame and exit
//	c1 tui                                          bubbletea v2's renderer
//	c1 raw                                          bubbletea v2 input, own line renderer
//
// pipeline: e5 | lipgloss; popup: menu keys confirm form toast-info toast-warn
// toast-error terminal loading none; oversize: fits wider taller both.
// SHOWCASE__ICON_WIDTH=1|2 sets how many cells an icon takes (default 1).
// In tui and raw: j/k move the list cursor, p next popup, o next oversize, q quits.

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"
)

var cursorRow int

func main() {
	if v, err := strconv.Atoi(os.Getenv("SHOWCASE__ICON_WIDTH")); err == nil && (v == 1 || v == 2) {
		iconWidth = v
	}
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: c1 dump|tui|raw ...")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "dump":
		a := os.Args[2:]
		w, _ := strconv.Atoi(a[1])
		h, _ := strconv.Atoi(a[2])
		k := popupKind(slices.Index(popupNames, a[3]))
		o := oversize(slices.Index(oversizeNames, a[4]))
		draw := e5Screen
		if a[0] == "lipgloss" {
			draw = lgScreen
		}
		fmt.Print(strings.Join(draw(w, h, k, o), "\r\n"))
	case "tui", "raw":
		raw := os.Args[1] == "raw"
		opts := []tea.ProgramOption{}
		if raw {
			// Without its renderer bubbletea also skips raw mode and the size.
			st, err := term.MakeRaw(os.Stdin.Fd())
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			defer term.Restore(os.Stdin.Fd(), st)
			w, h, _ := term.GetSize(os.Stdout.Fd())
			opts = append(opts, tea.WithoutRenderer(), tea.WithWindowSize(w, h))
			fmt.Print("\x1b[?1049h\x1b[?25l")
			defer fmt.Print("\x1b[?25h\x1b[?1049l")
		}
		if _, err := tea.NewProgram(model{raw: raw, kind: pMenu}, opts...).Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

type model struct {
	raw  bool
	w, h int
	kind popupKind
	over oversize
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j":
			cursorRow++
		case "k":
			cursorRow += 3
		case "p":
			m.kind = (m.kind + 1) % popupKind(len(popupNames))
		case "o":
			m.over = (m.over + 1) % oversize(len(oversizeNames))
		}
	}
	if m.raw && m.w > 0 {
		// Whole lines from column 0, the way bubbletea v1 drew: the terminal,
		// not a cell model, decides where each glyph lands.
		var b strings.Builder
		b.WriteString("\x1b[?2026h")
		for i, l := range m.frame() {
			fmt.Fprintf(&b, "\x1b[%d;1H%s\x1b[0m", i+1, l)
		}
		b.WriteString("\x1b[?2026l")
		fmt.Print(b.String())
	}
	return m, nil
}

func (m model) frame() []string {
	return e5Screen(m.w, m.h, m.kind, m.over)
}

func (m model) View() tea.View {
	if m.raw || m.w == 0 {
		return tea.NewView("")
	}
	v := tea.NewView(strings.Join(m.frame(), "\n"))
	v.AltScreen = true
	return v
}
