package main

// Usage:
//
//	c1 dump <W> <H> <popup> <oversize>   print one frame and exit
//	c1 tui                               bubbletea v1's renderer
//	c1 replay <W> <H> <keys...>          print the frame tui shows after keys
//
// popup: menu keys confirm form toast-info toast-warn toast-error terminal
// loading none; oversize: fits wider taller both.
// SHOWCASE__ICON_WIDTH=1|2 sets how many cells an icon takes (default 1).
// In tui: j/k move the list cursor, p next popup, o next oversize, q quits.

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var cursorRow int

func main() {
	if v, err := strconv.Atoi(os.Getenv("SHOWCASE__ICON_WIDTH")); err == nil && (v == 1 || v == 2) {
		iconWidth = v
	}
	lipgloss.SetColorProfile(termenv.TrueColor)
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: c1 dump|tui ...")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "dump":
		a := os.Args[2:]
		w, _ := strconv.Atoi(a[0])
		h, _ := strconv.Atoi(a[1])
		k := popupKind(slices.Index(popupNames, a[2]))
		o := oversize(slices.Index(oversizeNames, a[3]))
		fmt.Print(strings.Join(e5Screen(w, h, k, o), "\r\n"))
	case "replay":
		// The frame tui shows after these keys, printed whole: what the
		// renderer's output is compared against.
		w, _ := strconv.Atoi(os.Args[2])
		h, _ := strconv.Atoi(os.Args[3])
		var m tea.Model = model{w: w, h: h, kind: pMenu}
		for _, k := range os.Args[4:] {
			m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
		}
		fmt.Print(strings.ReplaceAll(m.View(), "\n", "\r\n"))
	case "tui":
		if _, err := tea.NewProgram(model{kind: pMenu}, tea.WithAltScreen()).Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

type model struct {
	w, h int
	kind popupKind
	over oversize
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case tea.KeyMsg:
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
	return m, nil
}

func (m model) View() string {
	if m.w == 0 {
		return ""
	}
	return strings.Join(e5Screen(m.w, m.h, m.kind, m.over), "\n")
}
