# terminu-go

**Language**: English · [繁體中文](README-zh_TW.md)

**A Go library that draws and drives a terminal UI the way [terminu design](https://github.com/vulcanshen/terminu) says.**

You register your panels and your actions — key, name, scope (item, panel or global), whether to confirm first, and
what to do. terminu-go draws the whole screen from that one registration and handles the core keys: the `Space` menu,
the `?` key reference, the hint on each panel's bottom border, the footer, confirm popups, fading what is not in focus,
`Esc` closing one layer at a time, and popups stacked on popups. Because the menu, the key reference and the keys
themselves come from the same registration, they cannot drift apart.

Screens the library does not own (a full-screen lock screen, several terminals side by side) can still use its parts:
confirm, toast and every kind of input work on their own. A popup of your own (a grid to draw on) goes on the same
stack as the library's: you draw what is inside and handle its keys; the frame, the animation, the fading and `Esc` are
the library's, and its confirms and inputs open over yours.

It is built on [Bubble Tea](https://github.com/charmbracelet/bubbletea) v1 and [Lip Gloss](https://github.com/charmbracelet/lipgloss) v1.

## Install

```sh
go get github.com/vulcanshen/terminu-go
```

Requires Go 1.26 or later. Apps built with it need what terminu design needs from the user's terminal: a
[Nerd Font](https://www.nerdfonts.com/) and 24-bit colour (truecolor). The terminal component (a program running in a
PTY) works on Unix only.

Some fonts make an icon take two cells, and some make the powerline symbols (the ends of the capsules on the frames, the
arrow at the end of the cursor) take two as well. The library measures both when the app starts; the user can set them
instead with `<APP>__ICON_WIDTH` and `<APP>__POWERLINE_WIDTH` (for one app, e.g. `LOCKU__ICON_WIDTH=2`) or
`TERMINU__ICON_WIDTH` and `TERMINU__POWERLINE_WIDTH` (shared by every terminu app: set once, every app in the family reads
them). An app that runs a terminal inside itself sets both `TERMINU__` variables for the programs it starts.

## Quick start

A whole app, a list of tasks with one action on it ([`examples/minimal`](examples/minimal/main.go)):

```go
// minimal is the smallest terminu app: one list and one action on it.
package main

import (
	_ "embed"
	"log"
	"slices"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"
)

//go:embed icon.svg
var icon []byte // the app's mark, drawn by its splash (V)

func main() {
	app := terminu.New("todo")
	list := &terminu.List{Columns: []terminu.Column{{Title: "Task", Flex: true}}}
	list.SetRows([]terminu.Row{{ID: "1", Cells: []string{"Write the README"}}, {ID: "2", Cells: []string{"Tag v1.0.0"}}})
	done := terminu.Action{Key: "x", Label: "Done", Desc: "this task", Scope: terminu.ItemScope, Hint: "done",
		Do: func() tea.Cmd {
			row, _ := list.Current()
			list.SetRows(slices.DeleteFunc(list.Rows(), func(r terminu.Row) bool { return r.ID == row.ID }))
			return app.Toast(terminu.Info, "Done: "+row.Cells[0])
		}}
	app.AddScreen(&terminu.Screen{Name: "Tasks", Panels: []*terminu.Panel{{Label: "Tasks", Body: list, Actions: []terminu.Action{done}}}})
	var err error
	if app.Splash, err = terminu.NewSplash(icon, "todo", "v0.1.0", "Things to do"); err != nil {
		log.Fatal(err)
	}
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
```

`x` marks the task under the cursor done. `Space` lists what can be done there, `?` lists the keys, `V` opens the
splash, `q` leaves. Every terminu app has a splash with its own mark (tdp Rules S); `icon.svg` is that mark, and the
example uses a copy of terminu's.

To see everything it does, run the showcase, an example app that shows every component on fake data:

```sh
go run github.com/vulcanshen/terminu-go/examples/showcase@latest
```

## Settings

Two of the user's settings are shared by every terminu app (tdp Rules E3). The library does not read config files: the
app reads them from its own and hands them over.

| Key | What it does | Default | Set |
|---|---|---|---|
| `hide_secrets_in_preview` | Previews of files hide private keys and files only their owner may read | `false` | `App.HideSecretsInPreview` |
| `input_tab_width` | How many spaces `Tab` puts in a multi-line text | `4` | `App.InputTabWidth` |

## Which terminu design

terminu-go implements **tdp v0.6.14**. The tdp version each release implements is listed in the [CHANGELOG](CHANGELOG.md).

## Links

- [CHANGELOG](CHANGELOG.md)
- [Developer notes](docs/dev-remarks.md) (in Traditional Chinese)
- [terminu design](https://github.com/vulcanshen/terminu)

## License

[GPL-3.0](LICENSE)
