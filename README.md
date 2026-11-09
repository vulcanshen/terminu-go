# terminu-go

**Language**: English · [繁體中文](README-zh_TW.md)

**A Go library that draws and drives a terminal UI the way [terminu design](https://github.com/vulcanshen/terminu) says.**

You register your panels and your actions — key, name, scope (item, panel or global), whether to confirm first, and
what to do. terminu-go draws the whole screen from that one registration and handles the core keys: the `Space` menu,
the `?` key reference, the App Log on `!` (every toast and error, and what the app logs), the hint on each panel's
bottom border, the footer, confirm popups, fading what is not in focus, `Esc` closing one layer at a time, and popups
stacked on popups. Because the menu, the key reference and the keys
themselves come from the same registration, they cannot drift apart.

Screens the library does not own (a full-screen lock screen, several terminals side by side) can still use its parts:
draw such a screen yourself with `App.Canvas`, and confirm, toast and every kind of input open over it as they do
anywhere else. A popup of your own (a grid to draw on) goes on the same
stack as the library's: you draw what is inside and handle its keys; the frame, the animation, the fading and `Esc` are
the library's, and its confirms and inputs open over yours. When it has values to submit, it works as a form: the
library puts the buttons under your content and does `Tab`, the error row, and the question before `Esc` discards changes.

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
	list := &terminu.List{Columns: []terminu.Column{{Title: "Task", Shrink: true}}}
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
splash, `q` leaves. Every terminu app has a splash with its own mark (tdp S1, S5); `icon.svg` is that mark, and the
example uses a copy of terminu's.

To see everything it does, run the showcase, an example app that shows every component on fake data:

```sh
go run github.com/vulcanshen/terminu-go/examples/showcase@latest
```

## Text and colours

The strings you give the library are text. Whatever comes from outside — a file's name, a web page's title, a
program's output — never reaches the terminal as it is:

- **Names and short texts** (a list's cells, values on forms and panels, menu labels, titles, toasts, confirms,
  options, the statusbar) draw every control character by its name in Red: a tab is `\t`, a colour code is `\x1b`
  then `[31m`. To colour one yourself, give spans: `Row.Styled`, `Field.Styled`, `FormField.Styled`, and
  `App.Status` returns them. On a `Canvas` or in a popup of your own, `Escape` draws text from outside the same way.
  A list column of lines of text — a line of source code, of a log — is marked `Column.Line`: a tab in it is spaces.
  Your own text has no line break in it: a message of several rows goes a row at a time,
  `app.Error(title, row1, row2)`, `app.Toast(kind, row1, row2)`, so a red `\n` on the screen is from outside.
- **What is read** (a viewer, a panel's log, previews) is drawn as a terminal would: colour codes colour, `\n` starts
  a line, a progress bar's `\r` leaves its last state, anything else is its name in Red. A body of your own that
  shows a log draws its lines with `Reading`.
- **Syntax colours**: a viewer whose `Lang` names a language, and a text file in a preview, are coloured as tdp asks
  (a file's format by its name, and by its directory where formats share a name: a `config` in `.ssh` is SSH's, a `hosts` is the hosts file only in
  `/etc`):
  chroma's catppuccin-mocha, its colours, bold, italic and underline but never its background, with the gaps chroma
  leaves filled the family's one way (TOML's table names and keys, YAML's values without quotes…). A body of your own that shows a file colours it with
  `Syntax(lang, lines)`, to look as it does in the preview. Importing the library puts its lexers in chroma's global
  registry, so code of yours that calls chroma gets the gaps filled too.

## The App Log

`!` on any panel opens the App Log: what the app did and what went wrong, newest last, in memory only. Every toast
but the inspect toast, which only follows the cursor, and every error popup is in it by itself; `app.Log(kind, text)`
adds the rest — an action's result, what the app does in the background — from any goroutine, worded as a toast, with
a capital first. While errors or warnings came in unseen, the footer's `!:log` says how many.

## A file of your own it can't read

When the app can't read its settings, state or data — a bad format, a broken file, no permission — call
`app.MoveAside(path, err, startFresh)` at the start. The user is asked to move the file aside: `Enter` renames it to
`.bak` (never over another) and runs `startFresh`, where the app goes on as on its first start; `Esc` leaves, `Run`
returning the path and the reason for `main` to print. When the data is kept one record per file, call `app.MoveAsideRecord(path, err, reread)` for the record
that can't be read: `Enter` only moves that one aside and runs `reread`, which reads the rest. Files a newer version
no longer uses are left to the user: call `app.NoLongerUsed(paths...)` at every start, and the App Log names each one
still there.

## Raising privileges

To do something as the administrator (copying a temporary file over `/etc/hosts`, say), call
`app.Elevate(terminu.Elevate{Command: ..., Done: ...})` with only the command to run. It runs at once when no password
is needed; otherwise the lib asks in a `PASSWORD` password popup and hands the password to sudo, a wrong one asked for
again in the same popup. The app never gets the password: it is not stored, logged in the App Log, shown or sent
anywhere. `Done` gets how the command ended: an `*terminu.ElevateError` when it failed (each line it printed), for an
error popup; `OnCancel` runs when the user cancels. With doas and no sudo, set `Tool: terminu.Doas`: doas asks for the
password itself in a terminal popup. Not for Windows: use gsudo or the sudo Windows has there, the system asking with UAC.

## Modes of your own

`panel.EnterMode(&terminu.Mode{...})` puts a panel in a mode of the app's own (tdp K11): only the mode's `Keys` and the
core keys work there, and `Esc` leaves. A `ModeKey` takes only the keys its `Keys` lists; the library adds none. List
the arrow keys and `PgUp|PgDn` there yourself, as tdp K12 has them; `Key` is what the key reference and the footer
write, the letters alone:

```go
{Key: "j|k", Keys: []string{"j", "k", "↓", "↑"}, Desc: "down, up", NoFooter: true, Do: move},
{Key: "u|d", Keys: []string{"u", "d", "PgUp", "PgDn"}, Desc: "half a page up, down", NoFooter: true, Do: move},
```

## Settings

Two of the user's settings are shared by every terminu app (tdp E3). The library does not read config files: the
app reads them from its own and hands them over.

| Key | What it does | Default | Set |
|---|---|---|---|
| `hide_secrets_in_preview` | Previews of files hide private keys and files only their owner may read | `false` | `App.HideSecretsInPreview` |
| `input_tab_width` | How many spaces `Tab` puts in a multi-line text, and how wide a tab in text on the screen is drawn | `4` | `App.InputTabWidth` |

## Testing an app

The `tdptest` package drives an app in a Go test the way a user does, and after every message it checks what terminu
design asks of every screen: each row exactly as wide as the terminal (tdp L4).

```go
func TestDone(t *testing.T) {
	d := tdptest.Start(t, newApp(), 80, 40) // newApp is all of main but Run
	d.Press("x")
	if !d.Has("Done: Write the README") {
		t.Error("no toast")
	}
	d.Advance(10 * time.Second)
	if d.Has("Done:") {
		t.Error("the toast stays")
	}
}

// The app's first launch, walked on many terminal sizes.
func TestFirstLaunch(t *testing.T) {
	tdptest.Check(t, tdptest.App{Name: "todo", New: newApp})
}
```

- **Keys and text**: `Press` takes keys by the names terminu design writes (`"j"`, `"Enter"`, `"Ctrl-C"`, `"Alt-Esc"`);
  `Type` types text a key at a time, `Burst` sends it all at once, `Paste` pastes it. `Resize` changes the terminal's
  size, `Send` sends a message of the app's own, `Do` runs code where the app may be changed (`app.Focus`).
- **Time is the test's**: after each input the popups' animations play to their end; `Advance` moves the time on (a
  toast going, a timer); with `d.Auto = false`, `Step` plays one frame at a time. The app's own timers follow it when
  they are made with `App.Tick` (below).
- **Commands** run as they do in the program. One that takes long (a file read, a program's output) comes back with a
  later input; `WaitFor` waits for it.
- **What is on the screen**: `Screen` is the rows as text, `Styled` with their colours, `Has` looks for text, `Quit`
  reports that the app asked to quit, `Clipboard` is what it copied. A failure says the size, the keys pressed and the
  screen; `Note` adds more to it (which of `Tricky` the app was given).
- **`EachIconWidth`** runs a test once with icons and powerline symbols one cell wide, once with icons two, once with
  both two.
- **`Check`** launches the app as for the first time on terminals of many sizes and every icon width, each in a new
  sandbox (`HOME` and the XDG directories under `/tmp`, the app's own variables cleared), and opens the `Space` menu,
  the global operation popup, the App Log and `?` on every screen and panel, closing each with `Esc`; then it shrinks the terminal
  under the app's least size and back. A `MinWidth`, `MinHeight` above 60×32 or a `NarrowBelow` above 60 fails it
  (tdp L1). A colour code drawn as text — one left in a cell or a value instead of
  `Styled` — fails it, unless `Tricky` says the seeded data has tricky values. Any test on a `Driver` fails when a
  line break was drawn by its name in Red (a `\n` in a toast of your own); set `AllowLineBreaks` when your data has
  such names on purpose. `Env` gets the sandbox's `HOME`, so a variable can point to a file `Seed` put
  there. With `go test -short` it launches on the least size and 80×24 only, icons one cell wide.
  It does not read the app's repository: the family's templates and the repository checks are in [terminu-harness](https://github.com/vulcanshen/terminu-harness).
- **`Tricky`, `Sizes` and `Mashes`** are values that have broken screens: control characters, escape sequences, wide
  and joined characters, very long values; odd terminal sizes; keys pressed faster than the animations (`Mash`).

The widths of icons are the whole test program's, so a test that drives an app does not run in parallel with another.

### Timers: `App.Tick`

Make the app's own timers with `App.Tick` instead of `tea.Tick`; it takes the same arguments:

```go
return app.Tick(300*time.Millisecond, func(time.Time) tea.Msg { return searchNow{} }) // was tea.Tick
```

In the program it is `tea.Tick`. Under a test it waits for the test's time, so `Advance` fires it and nothing fires on
its own. A timer made with `tea.Tick` runs on the real clock there too: the test cannot move it, and what it does comes
in whenever it is due.

## What the lib does and what an app does

tdp E8 keeps three principles for any shared lib: an app does not redo what the lib covers, the screen tests split by
who draws what, and the lib does not do its own thing. How the work splits between this lib and an app using it:

| Rule | The lib | The example app | An app using the lib |
|---|---|---|---|
| The on-screen rules (K, M, L, F, X, T, S1–S4, components) | covers them | shows all of them | from the lib, not done again |
| E1 `version`, `help` | — | yes | its own |
| E2 naming environment variables | the variables the lib reads hang under the app's name (e.g. `<APP>__ICON_WIDTH`) | as E2 (e.g. `SHOWCASE__…`) | its own variables follow E2 |
| E3 where config and data live | — | stores nothing, not applicable | its own |
| E4 requirements | stated in the README | — | stated in the README |
| E5 the actual icon width | covers it: the probe (icons and powerline symbols), the display-width functions, setting `TERMINU__ICON_WIDTH` and `TERMINU__POWERLINE_WIDTH` for subprocesses, reading `LC_TERMINU_ICON_WIDTH` and `LC_TERMINU_POWERLINE_WIDTH` at startup; in E5's tests, measuring the bare box of each kind of popup (the lib draws the popups) | — | from the lib; an app with ssh sends `LC_TERMINU_*` itself when connecting. E5's tests cover its own whole screen only: its own layout plus the popups it opens, at E5's widths and sizes, every line full width |
| E6 releases | a git tag (a Go module), no binary; the L4 screen tests are required | not released; the L4 screen tests are required | as E6 (the screen tests split as in the row above); the family's templates and the repository checks are in [terminu-harness](https://github.com/vulcanshen/terminu-harness) |
| S5 the icon the splash draws | the splash mechanism; building a splash, it returns an error for a row of text wider than 50 columns | tdp's `principle/icon.svg` (terminu's mark) | its own `docs/icon.svg` |
| E7 README | a structure of its own: installation is `go get`, the quick start is code, and it says how to run the example app | none of its own | as E7 |
| A requirement that cannot be met, or works badly ("Leaving it to the app" at the start of the Rules) | **the lib does not do its own thing**: it reports a gap in tdp, and the lib changes once the spec does | — | reports a gap in tdp |

This only sets the split when an app uses the lib; whether to use it, the app decides. An app moving to the lib can
delete its on-screen code and its E5 code; E1, E3, E6, E7 and its own icon stay.

## Which terminu design

terminu-go implements **tdp v1.2.1**. The tdp version each release implements is listed in the [CHANGELOG](CHANGELOG.md).

## Links

- [CHANGELOG](CHANGELOG.md)
- [Developer notes](docs/dev-remarks.md) (in Traditional Chinese)
- [terminu design](https://github.com/vulcanshen/terminu)

## License

[GPL-3.0](LICENSE)
