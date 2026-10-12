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

It is built on [Bubble Tea v2](https://github.com/charmbracelet/bubbletea) and [Lip Gloss v2](https://github.com/charmbracelet/lipgloss).

## Install

```sh
go get github.com/vulcanshen/terminu-go
```

Requires Go 1.26 or later. Apps built with it need what terminu design needs from the user's terminal: a
[Nerd Font](https://www.nerdfonts.com/) and 24-bit colour (truecolor).

## Quick start

The API is not settled yet: the library is still checking the spec against a working implementation. This section will
show a minimal app once it is.

## Which terminu design

terminu-go implements **tdp v0.3.0**. The tdp version each release implements is listed in the [CHANGELOG](CHANGELOG.md).

## Links

- [CHANGELOG](CHANGELOG.md)
- [Developer notes](docs/dev-remarks.md) (in Traditional Chinese)
- [terminu design](https://github.com/vulcanshen/terminu)

## License

[GPL-3.0](LICENSE)
