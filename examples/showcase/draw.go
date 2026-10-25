package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"
)

const glyphGrid = "\U000F02C1" // nf-md-grid

const drawW, drawH = 12, 8

// drawBody is a popup of the showcase's own: a grid to draw an icon on. It
// has a value to submit, so the lib puts the action area with Save under it
// and does Tab, Esc and q as on a form (tdp popup.8); the showcase draws the
// grid and handles the keys on it.
type drawBody struct {
	a       *terminu.App
	cells   [drawH][drawW]bool
	x, y    int
	focused bool // the focus is on the grid, not on Save
	changed bool
}

func (ds *dialogsScreen) draw() tea.Cmd {
	b := &drawBody{a: ds.s.app}
	for _, c := range [][2]int{{2, 3}, {3, 4}, {4, 5}, {5, 6}, {6, 5}, {7, 4}, {8, 3}} {
		b.cells[c[1]][c[0]] = true
	}
	return b.a.Popup(terminu.Popup{Title: "Draw", Glyph: glyphGrid, Subject: "icon", Body: b, Actions: &terminu.PopupActions{
		Button: "Save", Content: "grid",
		Submit: func() (tea.Cmd, error) { return b.a.Toast(terminu.Info, "Saved icon"), nil },
	}})
}

// Size: two cells a square, one cell from the frame.
func (b *drawBody) Size(sw, sh int) (int, int) { return drawW*2 + 2, drawH }

func (b *drawBody) Focus(on bool) { b.focused = on }

func (b *drawBody) Draw(w, h int, layer terminu.Color) []string {
	var rows []string
	for y := range drawH {
		var line strings.Builder
		for x := range drawW {
			on, cursor := b.cells[y][x], b.focused && x == b.x && y == b.y
			switch {
			case cursor && on:
				line.WriteString(terminu.Style{Fg: terminu.Peach, Bg: layer}.Render("▐▌"))
			case cursor:
				line.WriteString(terminu.On(layer).Render("  "))
			case on:
				line.WriteString(terminu.Fg(terminu.Peach).Render("██"))
			default:
				line.WriteString(terminu.Fg(terminu.Surface2).Render("· "))
			}
		}
		rows = append(rows, " "+terminu.Fit(line.String(), w-2)+" ")
	}
	return rows
}

func (b *drawBody) Key(key string, _ tea.KeyMsg) (bool, tea.Cmd) {
	switch key {
	case "Enter", "Space":
		// A square has two values; Space goes to the next one (tdp K5).
		b.cells[b.y][b.x] = !b.cells[b.y][b.x]
		b.changed = true
	case "x":
		b.cells = [drawH][drawW]bool{}
		b.changed = true
	case "h", "←":
		b.x = max(b.x-1, 0)
	case "l", "→":
		b.x = min(b.x+1, drawW-1)
	case "k", "↑":
		b.y = max(b.y-1, 0)
	case "j", "↓":
		b.y = min(b.y+1, drawH-1)
	default:
		return false, nil
	}
	return true, nil
}

func (b *drawBody) Changed() bool { return b.changed }

// Check is what Save checks: an icon with no square is not saved, and the
// error row says so (tdp popup.7).
func (b *drawBody) Check() string {
	if b.cells == ([drawH][drawW]bool{}) {
		return "Draw at least one square"
	}
	return ""
}

// QuitNote is the line the quit confirm adds while the drawing has changes.
func (b *drawBody) QuitNote() string {
	if b.changed {
		return "Changes to icon are not saved."
	}
	return ""
}

func (b *drawBody) Hint() []terminu.KeyHint {
	return []terminu.KeyHint{{Key: "Enter|Space", Desc: "toggle"}}
}

func (b *drawBody) Keys() []terminu.KeySection {
	return []terminu.KeySection{
		{Rows: []terminu.KeyRow{
			{Key: "Enter|Space", Desc: "toggle the square"},
			{Key: "x", Desc: "clear every square"},
		}},
		{Title: "move", Rows: []terminu.KeyRow{{Key: "h|j|k|l", Desc: "left, down, up, right; stops at the edges"}}},
	}
}
