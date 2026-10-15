package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"
)

const glyphGrid = "\U000F02C1" // nf-md-grid

const drawW, drawH = 12, 8

// drawBody is a popup of the showcase's own: a grid to draw an icon on, with
// a Save button under it. The showcase draws the inside and handles its keys;
// the lib draws the frame, opens and closes it, and stacks its own popups on
// it (the discard confirm, the quit confirm, the key reference).
type drawBody struct {
	a       *terminu.App
	cells   [drawH][drawW]bool
	x, y    int
	button  bool // the focus is on Save
	changed bool
}

func (ds *dialogsScreen) draw() tea.Cmd {
	b := &drawBody{a: ds.s.app}
	for _, c := range [][2]int{{2, 3}, {3, 4}, {4, 5}, {5, 6}, {6, 5}, {7, 4}, {8, 3}} {
		b.cells[c[1]][c[0]] = true
	}
	return b.a.Popup(terminu.Popup{Title: "Draw", Glyph: glyphGrid, Subject: "icon", Body: b})
}

// Size: two cells a square, one cell from the frame; a blank row, the button.
func (b *drawBody) Size(sw, sh int) (int, int) { return drawW*2 + 2, drawH + 2 }

func (b *drawBody) Draw(w, h int, layer terminu.Color) []string {
	var rows []string
	for y := range drawH {
		var line strings.Builder
		for x := range drawW {
			on, cursor := b.cells[y][x], !b.button && x == b.x && y == b.y
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
	save := terminu.Style{Fg: terminu.Text, Bg: terminu.Surface1}.Render(" Save ")
	if b.button {
		save = terminu.On(layer).Render(" Save ")
	}
	return append(rows, "", terminu.PadLeft(save, w-1)+" ")
}

func (b *drawBody) Key(key string, _ tea.KeyMsg) (bool, tea.Cmd) {
	switch key {
	case "Tab", "Shift-Tab":
		b.button = !b.button
	case "Esc":
		if !b.changed {
			return false, nil // the lib closes it
		}
		return true, b.a.Confirm(terminu.Ask{Title: "Discard", Question: "Discard changes to icon?", Verb: "discard"}, b.close)
	case "q":
		// q does nothing here, as on a form (Rules K9).
	case "Enter", "Space":
		if b.button {
			if key == "Space" {
				return true, nil
			}
			return true, tea.Batch(b.close(), b.a.Toast(terminu.Info, "Saved icon"))
		}
		b.cells[b.y][b.x] = !b.cells[b.y][b.x]
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

func (b *drawBody) close() tea.Cmd { return b.a.ClosePopup(b) }

// QuitNote is the line the quit confirm adds while the drawing has changes.
func (b *drawBody) QuitNote() string {
	if b.changed {
		return "Changes to icon are not saved."
	}
	return ""
}

func (b *drawBody) Hint() []terminu.KeyHint {
	if b.button {
		return []terminu.KeyHint{{Key: "Enter", Desc: "save"}, {Key: "Tab", Desc: "grid"}, {Key: "Esc", Desc: "cancel"}}
	}
	return []terminu.KeyHint{{Key: "Enter|Space", Desc: "toggle"}, {Key: "Tab", Desc: "actions"}, {Key: "Esc", Desc: "cancel"}}
}

func (b *drawBody) Keys() []terminu.KeySection {
	return []terminu.KeySection{
		{Rows: []terminu.KeyRow{
			{Key: "Enter|Space", Desc: "toggle the square; Enter on Save saves"},
			{Key: "Tab", Desc: "between the grid and Save"},
			{Key: "Esc", Desc: "cancel (asks first when changed)"},
		}},
		{Title: "move", Rows: []terminu.KeyRow{{Key: "h|j|k|l", Desc: "left, down, up, right; stops at the edges"}}},
	}
}
