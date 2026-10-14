# terminu-go

**Language**: English · [繁體中文](README-zh_TW.md)

**A Go library that draws and drives a terminal UI the way [terminu design](https://github.com/vulcanshen/terminu) says.**

You register your panels and your actions — key, name, scope (item, panel or global), whether to confirm first, and
what to do. terminu-go draws the whole screen from that one registration and handles the core keys: the `Space` menu,
the `?` key reference, the hint on each panel's bottom border, the footer, confirm popups, fading what is not in focus,
`Esc` closing one layer at a time, and popups stacked on popups. Because the menu, the key reference and the keys
themselves come from the same registration, they cannot drift apart.

Screens the library does not own (a full-screen lock screen, several terminals side by side) can still use its parts:
confirm, toast and every kind of input work on their own.

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

The API is not settled yet: the library is still checking the spec against a working implementation. This section will
show a minimal app once it is.

To see what it does, run the showcase, an example app that shows every component on fake data:

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

terminu-go implements **tdp v0.6.2**. The tdp version each release implements is listed in the [CHANGELOG](CHANGELOG.md).

## Links

- [CHANGELOG](CHANGELOG.md)
- [Developer notes](docs/dev-remarks.md) (in Traditional Chinese)
- [terminu design](https://github.com/vulcanshen/terminu)

## License

[GPL-3.0](LICENSE)
