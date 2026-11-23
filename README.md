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
	"fmt"
	"os"
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
		fmt.Fprintln(os.Stderr, "todo:", err) // errors on stderr after "<app>: " (tdp E1)
		os.Exit(1)
	}
	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "todo:", err)
		os.Exit(1)
	}
}
```

`x` marks the task under the cursor done. `Space` lists what can be done there, `?` lists the keys, `V` opens the
splash, `q` leaves. Every terminu app has a splash with its own mark (tdp S1, S5); `icon.svg` is that mark, and the
example uses a copy of terminu's.

Closing the terminal window ends `Run` as leaving does: it returns nil, and what `main` does after it (saving, say)
still runs. The programs running in the app's terminals are hung up first, as a closed terminal window does, so
nothing after `Run` waits on them for ever. The terminal is gone by then: nothing written to it shows.

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
  `App.Status` returns them. A row whose content has a colour of its own (a ranking's places) may give that colour to
  the cursor resting on it with `Row.CursorColor` (tdp color.2): only the row's own colour, and in a mode the cursor
  is Subtext1 as ever. On a `Canvas` or in a popup of your own, `Escape` draws text from outside the same way.
  A list column of lines of text — a line of source code, of a log — is marked `Column.Line`: a tab in it is spaces.
  Your own text has no line break in it: a message of several rows goes a row at a time,
  `app.Error(title, row1, row2)`, `app.Toast(kind, row1, row2)`, so a red `\n` on the screen is from outside.
- **What is read** (a viewer, a panel's log, previews) is drawn as a terminal would: colour codes colour, `\n` starts
  a line, a progress bar's `\r` leaves its last state, anything else is its name in Red. A body of your own that
  shows a log draws its lines with `Reading`.
- **A document drawn from markup** (a web page, a Markdown file drawn as a document) is drawn as a browser would, and
  the library has no component that draws one: its viewer and previews show a Markdown file's text. Laying one out
  yourself, turn a tab or a `\n` in running text into a space, and in a code block a tab into `App.InputTabWidth`
  spaces (4 when unset) and a `\n` into a new row; then draw the text with `Escape`, so any other control character, a colour
  code's ESC too, is its name in Red and no colour comes from the document.
- **Syntax colours**: a viewer whose `Lang` names a language, and a text file in a preview, are coloured as tdp asks
  (a file's format by its name, and by its directory where formats share a name: a `config` in `.ssh` is SSH's, a `hosts` is the hosts file only in
  `/etc`):
  chroma's catppuccin-mocha, its colours, bold, italic and underline but never its background, with the gaps chroma
  leaves filled the family's one way (TOML's table names and keys, YAML's values without quotes…). A body of your own that shows a file colours it with
  `Syntax(lang, lines)`, to look as it does in the preview. Importing the library puts its lexers in chroma's global
  registry, so code of yours that calls chroma gets the gaps filled too.

## The App Log

`!` on any panel opens the App Log: what the app did and what went wrong, newest last, in memory only. Every toast
and every error popup is in it by itself; `app.Log(kind, text)`
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

With no screen (a command printing to a pipe), everything below and above goes to stderr, a line each after
`<app>: `, in the TUI's words; where the TUI goes on, the command goes on, its exit status saying whether its own job
was done (tdp E3, E1). A file it can't read is as `Esc`: touch no file, write `terminu.CannotReadText(path, err)` (the
words `Run` returns, `~/.config/catu/settings.yaml can't be read: …`), a line saying what to do if you like (`Open catu
to move it aside and start fresh`), and exit 1. `terminu.NoLongerUsedLines(paths...)` are `NoLongerUsed`'s lines.

A file the app reads but with values it can't use (a colour that is not `#rrggbb`, a negative timeout) or entries that
don't fit (a blank or repeated name) is not one it can't read: the app goes on. At the start, copy it aside with
`copied, err := terminu.KeepAside(path)` (to `.bak`, never over another), write the file again with your defaults
for those values and without those entries, and call `app.DidNotFit(path, copied, misfits...)`: one Warning toast
naming the file and the copy, and a WARN in the App Log for each `terminu.Misfit{Key, Was, Now}`. With no screen,
write `terminu.MisfitNotice(path, copied, misfits...).Lines()` to stderr: the toast's row, then each WARN. When
`KeepAside` fails, don't write the file
again: use the defaults for this run only, give its error in an Error toast (or on stderr) and call
`app.DidNotFit(path, "", misfits...)` for the WARNs.

A value naming an entry of the user's own data that is not there (a setting naming a profile deleted by hand) is not
a value that doesn't fit: no `KeepAside`, the file left as it is, the value kept. Use your built-in default for this
run and say so with `app.Toast(terminu.Warning, …)` at the start; every toast goes in the App Log in the same words,
so that is the WARN too, with no `app.Log` beside it (with no screen, write the sentence to stderr). Show the value as
written with the field's `Err` returning true, so its label is Red; while the entry is gone, its select opens with
`Selected: -1`.

When one start has more than one of these to say (a location moved, values that don't fit in one file or several, a
copy that could not be made, a value naming an entry that is not there), each is a toast of its own, stacked as every
toast is. Collect a `terminu.Notice{Kind, Text}` for each in the order they happened,
`terminu.MisfitNotice(path, copied, misfits...)` for a file's misfits in place of `DidNotFit`, and call
`app.Notify(notices...)`: a toast each, and in the App Log an entry each at its own kind, a file's WARNs right after
it. With no screen, write each notice's `Lines()` to stderr instead, a line each.

Toasts stack: a new one comes at the bottom and pushes the ones showing up; each leaves when its time is up; at most
three show, a fourth pushing the oldest out (its words are in the App Log). The same words as a toast showing, of the
same kind, make no new one: that one's time starts over and its title counts them (`INFO ×3`). `Esc` closes one a
press, the oldest first.

One that comes later (a namespace or a kind you can ask a cluster about only once the screen is open) goes to
`app.Notify` too, a toast of its own stacked with the rest, so none replaces another before it is read.

A key in a settings file that the app doesn't read is kept, whether it is one the app no longer reads or one it never
did (a typo, `refesh`), also when the app writes the file again: no toast, no copy, no `Notice`. At every start, log
an INFO entry for each with `app.Log(terminu.Info, …)`: `No longer used: measure in ~/.config/app/config.yaml; delete
it if you don't need it` for a key you know the app used to read, `Not read: refesh in ~/.config/app/config.yaml;
delete it if you don't need it` for any other. With no screen, those lines go to stderr.

Outside the TUI — a screen of your own that is not part of it, or output printed for another program to show (a
status line) — none of the above runs: no `MoveAside`, `KeepAside`, `DidNotFit` or `Notify`, and no file is touched
(nothing moved to `.bak`, nothing written again). Use your defaults for this run (with a file that can't be read, how
the app goes on is yours) and show the user there which file has what wrong, as you see fit (a mark at the end of the
status line, a line on a sidebar): nobody reads stderr there. The next time the TUI opens, it goes as above.

## A success report

A success the user needs to read whole that is too long for a toast (what an export wrote, and where) goes in a
read-only viewer: `app.Report(&terminu.Viewer{Title: "Exported", Lines: lines})`. It came up by itself, so `Enter`
closes it as well as `Esc`, and it goes in the App Log as a Success.

## Raising privileges

To do something as the administrator (copying a temporary file over `/etc/hosts`, say), call
`app.Elevate(terminu.Elevate{Command: ..., Done: ...})` with only the command to run. It runs at once when no password
is needed; otherwise the lib asks in a `PASSWORD` password popup and hands the password to sudo, a wrong one asked for
again in the same popup. The app never gets the password: it is not stored, logged in the App Log, shown or sent
anywhere. `Done` gets how the command ended: an `*terminu.ElevateError` when it failed (each line it printed), for an
error popup; `OnCancel` runs when the user cancels. With doas and no sudo, set `Tool: terminu.Doas`: doas asks for the
password itself in a terminal popup. Not for Windows: use gsudo or the sudo Windows has there, the system asking with UAC.

## Diff

`app.DiffModeAction(panel, list, terminu.DiffMode{Key: "D", Text: ...})` gives a list the Diff mode: the user picks
two rows and sees what differs, UNIFIED or SPLIT. Two rows that hold many (two directories, two namespaces, two
commits) go in instead, to a list of the items that differ: `Many` says which rows hold many, and `Items` gives the
items that differ between two of them, each a `terminu.DiffItem`. The lib draws that list in the panel, each item's
mark (`DiffAdded` +, `DiffRemoved` -, `DiffChanged` ~), and does `Enter`, `Esc` and the location row.
`CompareAction` and `Compare`, the old names, still build; they go in a later minor version.

A `DiffItem` names its two sides in `Anchor` and `Other`, the older one the anchor, so that they say where
(`ui.go @ 1a2b3c4`): its diff, titled `DIFF`, has them for its first row however it is reached, right under the
top border as a location row is, folded and never cut when they are long, and the list it goes into has them for its location row. It has `Text`, each side's text ("" on a side it is not on), or `Items` when it holds many itself,
or neither when it is not text and has no diff. A rename you know of is `DiffChanged`, named `old → new`. Leave
out what is the same on both sides, and read a text only when it is asked for.

An action of your own that shows two versions of one item (a file in a commit against the commit before it, a file
being edited against its last commit) opens the same diff with `app.OpenDiff(panel, item)`: with `Text`, the
comparison; with `Items`, the list of what differs in it, in the panel in place of its rows until `Esc` brings them
back. An action of yours that shows something else in that panel (Enter on a branch putting its commit under the
cursor) calls `panel.LeaveDiff()` first: it leaves every such list, as if `Esc` were pressed until the panel's rows
are back, and does nothing when there is none.

Going into such a list changes what the panel lists, so `OpenDiff` and the Diff mode clear the panel's filter first
(tdp panel.14). Do the same in an action of yours that changes what a panel lists or puts its cursor on a particular
item (the cursor in another panel taking this one to an item, going into an item): call `list.ClearFilter()` first,
whether or not the filter hides that item. The focus merely leaving the panel keeps the filter.

A panel that shows a difference (a details panel showing what changed in the file under the cursor) draws it as
UNIFIED does, never as a tool prints it (`git diff`'s headers and `@@` lines): read the two texts and call
`pager.SetDiff(anchor, other)` on its `Pager`, and give the panel an `Enter` action that opens the diff with
`app.OpenDiff(panel, item)`. The same text on both sides shows the Pager's `EmptyFact`.

When what it shows holds many (a folder that is new, a commit), list the items that differ, never their bare names:
`pager.SetDiffItems(items)` draws each item's mark in its colour, then its name, as the list `OpenDiff` goes into draws
them (a new folder's items are all `DiffAdded`). Its `Enter` goes into that list in the panel: `app.OpenDiff(panel,
terminu.DiffItem{Anchor: ..., Other: ..., Items: ...})` with the same items. None shows the Pager's `EmptyFact`.

A list with a details panel (`panel.Details`) may show its Diff mode there instead, without opening anything or
going into the list: set `InDetails` (which way is yours). The mode then spans the list and its details, both
frames in the mode's colour with `Diff` on them. The details follow the cursor, no key pressed: the lib shows the
anchor against the row under the cursor there, as `SetDiff` and `SetDiffItems` draw it, from your `Text` and
`Items`, and an empty state on the anchor (`Move to another <Noun> to diff with <anchor>`: set `Noun`, `item`
when empty) or on a row the same as it. The diff key or `Enter` on the list sends the focus to the details,
where it opens an item's diff or goes into it; `Esc` comes back the way the user went, to the list with the
anchor kept, then out of the mode, and the details show your own content again. While the mode is on, your
content, location row, `Fail` and loading text in the details are not shown; keep them up to date as ever. The
list's `Details` must be another panel of the same screen.

## An item of several rows

A list item can take rows under its first, on the items that need them (a commit's branches and tags under it):
`Row.Below` is a `[]terminu.BelowRow`, each a row of `Cells` under the list's columns, `Styled` to colour them as
`Row.Styled` does. Leave a cell `""` to leave its column blank, so a row's text starts where you put it; its last cell
with text runs on to the row's end. The first row keeps the columns, the header and the tick. The cursor is drawn on
it, and the rows below are lit with it. Moving, `N of M` and scrolling count items, and the filter matches an item by
any of its rows (tdp panel.5).

## Room left over

On a wide terminal, once every column shows its widest value and none is taken away, `List.Spread` says where the
rest of the width goes, your pick for each list (tdp panel.5): `SpreadEnd`, the default, at the row's end;
`SpreadFill`, all of it to the column you mark `Column.Fill` (a long one, a description), the columns after it at the
right edge; `SpreadEven`, the narrowest columns widened first until they are as even as the room allows;
`SpreadGaps`, every gap between columns widened alike from 2 cells up to 6, the rest to the `Fill` column or the row's
end. Short of room, every one narrows and drops columns as before.

## Modes of your own

`panel.EnterMode(&terminu.Mode{...})` puts a panel in a mode of the app's own (tdp K11): only the mode's `Keys` and the
core keys work there, and `Esc` leaves. A `ModeKey` takes only the keys its `Keys` lists; the library adds none. List
the arrow keys and `PgUp|PgDn` there yourself, as tdp K12 has them; `Key` is what the key reference and the footer
write, the letters alone:

```go
{Key: "j|k", Keys: []string{"j", "k", "↓", "↑"}, Desc: "down, up", NoFooter: true, Do: move},
{Key: "u|d", Keys: []string{"u", "d", "PgUp", "PgDn"}, Desc: "half a page up, down", NoFooter: true, Do: move},
```

The mode's Space menu lists its keys as tdp menu.4 has them. What a mode of your own lists there is up to you (tdp
K11): `NoMenu` keeps a key out of the menu, and `Mode.NoLeaveRow` keeps out the row that leaves the mode. Every key is
still in `?`, the footer stays as it is, and `Esc` still leaves. A menu left with nothing lists the leaving row all the
same, since a mode's Space menu always has a row.

A row of that menu closes the menu before it does what its key does, so a popup the key opens, such as a confirm before
starting over, goes back to the mode when it is cancelled or closed, not to the menu (tdp K11). The popup itself is
drawn and answered as its component: a confirm still takes `Enter` and `Esc`.

Leaving a mode of your own, its `OnExit` may open one popup saying where things are left, such as a confirm asking
whether to go on after a pause (tdp K11). The mode is left by the time the popup shows: `Esc` on it closes it, still
outside the mode, and going back in is the popup's own action (`EnterMode`). A mode that ends by one of its keys or on
its own (a game over) leaves with `Panel.LeaveMode`: the mode's Space menu and `?` close with it, and a popup you open
then is the only one (tdp K11).

Where the user sets the keys of a mode of your own (tdp K11), `Enter` and the movement keys may be set too. Put the key
the user chose in `Key` and `Keys`, and leave the arrow keys and `PgUp|PgDn` in the `Keys` of the `ModeKey` the mode
gives them by default. Moving left set to `a` is `{Key: "a", Keys: []string{"a", "←"}}`, and `?` writes `a|←`, the footer
and the Space menu `a`. A pair, one way each half, whose one half the user moved is two rows in `?` (tdp M5): give them
in `Ref`, and the footer still writes the pair as one.

```go
{Key: "Enter|k", Keys: []string{"Enter", "k", "PgDn", "↑"}, Desc: "rotate", Do: rotate,
	Ref: []terminu.KeyRow{{Key: "Enter|PgDn", Desc: "rotate left"}, {Key: "k", Desc: "rotate right"}}},
```

The user may put one of the mode's actions on `Space` as well; you never put one there by default (tdp K11). It is a
`ModeKey` with a `Do` taking `Space`, such as `{Key: "Space", Desc: "drop", Do: drop}`: `Space` then does it, and the
mode has no Space menu. The footer writes it first, where `Space:menu` was (`Space:drop ?:help …`), and `?` lists it
among the mode's keys. The keys only that menu listed go in the footer then: take their `NoFooter` off. Refusing
`Space` when an action has it already, the Warning saying what `Space` does now and to press `Esc` first for the menu,
and keeping `Space` for the menu in a mode of tdp's (Drag, Diff) are yours. A `ModeKey` taking `Space` with no `Do`
is still the core key, `Space` opening the menu.

What the footer of a mode of your own writes between `Space:menu ?:help` and `Esc:<verb>` is up to you, core keys
included (tdp K11, screen.2). To write `q:quit` there, give the mode a `ModeKey` with no `Do`, `{Key: "q", Desc:
"quit"}`: `q` still starts the quit flow, as in any mode, and so does its row in the Space menu; `?` lists it once,
among the core keys.

## An action from a popup of your own

A key on a popup of your own (`App.Popup`) may run one of the app's actions as its menu runs it (tdp popup.8):
`App.RunAction(key)` runs what `key` runs on the focused panel out of a mode, an item or panel operation, the panel's
`Enter` action or a global one; `q` is the quit flow. An action that asks first opens its confirm over the popup; a
disabled or hidden one does nothing, as a key with no action does. Done with nothing new open, every popup closes.

```go
func (b *settings) Key(key string, _ tea.KeyMsg) (bool, tea.Cmd) {
	if key == "n" { // the panel's [n]ew game
		return true, b.app.RunAction("n")
	}
	return false, nil
}
```

When the popup is over a panel in a mode of your own, the mode is left as the action runs: after `Enter` on its
confirm, so `Esc` there changes nothing. It is left as `Panel.LeaveMode` leaves it, without `OnExit`: do what `OnExit`
would do (a pause) before calling, and when leaving would lose something the user made and wants kept, ask before
calling (tdp K11). `q` leaves no mode; the quit flow asks first.

## An item's note

An item may have a note: where a link goes, a remark (tdp panel.18). Set `panel.Note` to a func giving the note of
the item under the cursor, nil for an item with none (in a mode of your own, `Mode.Note`). While the cursor rests on
an item with a note, its head shows right under the item, a small box with the note glyph and `i`; `i` opens it, a
viewer titled `NOTE` anchored under the item (above it when the item is on the panel's last row), at most half the
panel high, with the viewer's scrolling, selection mode and `/` search; `Esc` closes it back to its head. Opening it
goes in the App Log by itself, `Opened the note of <item>`, so your app logs nothing for it. The lib
puts `[i] Note` among the panel's actions; `i` is the family's key for it (tdp K13), so none of your actions on that
panel may take `i`. A `List` says where its cursor's item is; a body of your own implements
`terminu.NoteSpot`'s `NoteAt() (top, bottom, col int, ok bool)`.

```go
pods.Note = func() []string {
	if r, ok := list.Current(); ok && r.Cells[2] == "Pending" {
		return []string{"0/3 nodes are available: 3 Insufficient memory"}
	}
	return nil
}
```

`Panel.Inspect` and `Mode.Inspect`, the inspect toast the note replaces, still work for now; they go in a later minor
version.

## Panels that follow a leading panel

Some places of a screen may follow whichever of a few leading panels last had the focus (tdp L5): a git tool's right
side showing a branch's commits while the branches lead, a file's diff and history while the changes do. List the
places that follow in `Screen.Following` (indexes into `Panels`) and, for each leading panel, what they show in
`Screen.Leads`: `terminu.Lead{Panel: changes, Shows: []*terminu.Panel{diff, history, nil}}`, `nil` hiding a place.
When the focus moves to a leading panel, the lib puts its panels in those places; moving into them changes nothing.
Numbers go by place: a place keeps its `[N]` whatever it shows, a hidden one too, its digit doing nothing while it is
hidden. Your `Layout` asks `screen.Hidden(i)` to give a hidden place's room to its neighbour.

## A sign

A panel that only tells (a game's NEXT, INFO and KEYS, a legend) is a sign (tdp panel.17): set `Sign` to
`terminu.SignLit` (never dimmed, as streaming content) or `terminu.SignDim` (dimmed, as a panel without the focus),
and put it in `Screen.Panels` with the rest, so it is laid out and resized with them. It never takes the focus:
`Tab` and the digits pass it by, and its frame has its label alone, the other panels numbered among themselves.
It has no actions, tabs, hint or scroll position (`AddScreen` panics on `Actions` or `Tabs`). It does not scroll:
make its content fit the size it gets on the smallest screen (60 × 32). A `Pager`'s lines that still do not fit
are cut at the bottom, the last row shown ending in `…`.

```go
keys := &terminu.Pager{}
keys.SetLines([]string{"h|l  move", "k    rotate"})
sign := &terminu.Panel{Label: "Keys", Body: keys, Sign: terminu.SignLit}
```

## A status on a panel's frame

A panel may show one lasting status of the app's own on its frame: a glyph and a short text, such as when a file the
panel watches was last read (tdp panel.1). Whether a panel has one, what it says and which corner are yours:

```go
logs.FrameStatus = terminu.FrameStatus{Glyph: glyphWatch, Text: "17:43:42", Corner: terminu.BottomRight}
```

The lib draws it on the inner side of what that corner already has: after the capsule chain and the loading icon, left
of the mode name, after the hint, or left of a search's status and the scroll position. It is in the frame's colour;
when what it tells has gone wrong, set `Kind` to `terminu.StatusWarning` (Peach) or `terminu.StatusError` (Red). On a
border too narrow it is the first thing to go, whole, and it comes back when there is room. Change it from the update
loop as what it tells changes, on a timer of yours say (`App.Tick`, its message arriving in `OnMsg`); the zero value
takes it off. Popups have none.

## Content that changes while open

Whether and when to read a source again (a file the app watches) are yours; how what shows it changes is tdp
note.9's, and the lib does it:

```go
cmd := app.ReadAgain(viewer, lines) // a viewer open on it
pager.ReadAgain(lines)              // a panel's Pager, with Follow set
```

`ReadAgain` is as if the viewer had just been opened on the new lines: at the new end when its end was in sight (not in
the selection mode), the line numbers' column only widening, the search found again, the selection mode kept where its
positions still are; it returns the toasts that say what went, or, with the selection mode's Space menu or `?` open, a
note over them that closes with them. When the source cannot be read any more, call
`app.CannotReadAgain(viewer, "Cannot read notes.txt: it is gone")`: your words go in an Error toast, or in an error
popup over that menu. A selection mode of your own, on a reader of your own in a panel, says that its selection is gone
with `app.SelectionGone(panel, terminu.Warning, "Read again", "notes.txt changed: Visual left")` before it leaves the
mode: a toast, or the note over the mode's Space menu or `?`. A Pager that takes lines while it is open, by `Append` or
`ReadAgain`, sets `Follow`. Content read again is not streaming: leave `Panel.Stream` off for it (tdp T2).

What such a read finds goes in the App Log once (tdp note.7). A read that finds something changed is one
`app.Log(terminu.Info, "Changed outside the app: branch feat deleted")` saying what; one that finds nothing new records
nothing. When it fails, the first failure is the Error toast, which records itself; the same failure again (the same
words) shows and records nothing, so give `CannotReadAgain` no words then; a different one toasts again. A read that
works again is one `app.Log(terminu.Info, "The repository can be read again")`, with no toast. Updates that keep coming
in by themselves (streaming, a list a connection keeps up to date) are not recorded one by one.

## The command line

What a command prints takes one form in every app (tdp E1): its results on stdout; its errors and warnings on
stderr, a line each after `<app>: `, worded as the toast for the same thing; a wrong use (an unknown command, a missing
argument) on stderr, `<app>: ` and what is wrong (`Unknown command "x"`), a blank line, then the usage, exit 2; exit 0
when it is done, 1 when it is not. `<app> iconwidth`, where an app has it, prints `terminu.Probe(name).Lines()` on
stdout: how many cells an icon and a powerline symbol take, each with where it came from (`icon 2 (probe)`,
`powerline 1 (TERMINU__POWERLINE_WIDTH)`).

Every app has `<app> update`, which brings it to its latest release (never a pre-release) the way it was installed.
It is the same in every app, so the lib does it:

```go
case "update":
	os.Exit(terminu.SelfUpdate{App: "todo", Version: version.Version, Owner: "vulcanshen", Tap: "vulcanshen/tap"}.Run(os.Stdout, os.Stderr))
```

Installed with Homebrew, it runs `brew upgrade <Tap>/<App>`. Installed with the install script or downloaded from a
release, it downloads the release for this OS and arch from `github.com/<Owner>/<App>`, checks it against the
release's `checksums.txt` and puts it in place of the running file. Installed with Scoop, it prints the
`scoop update <app>` to run. Already the latest, it says so and exits 0. Built from source (`Version` "dev" or
empty), or with no right to write where the file is, it says why and what to do and exits 1. It never asks for
more rights. The release files must be named as the family's goreleaser template names them:
`<app>_<version>_<os>_<arch>.tar.gz` (`.zip` on Windows) with the executable in it, and `checksums.txt`.
`Source` is how a build from source updates (`git pull && make install`). `Then` is a command to run after
updating by hand. `After(exe, updated)` is what your app does once it is at the latest: `exe` is the new
executable after an update.

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
  reports that the app asked to quit (after that nothing more goes to it, as in the program: a popup closing as it
  quits stops where it is), `Clipboard` is what it copied. A failure says the size, the keys pressed and the
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

The test's time also moves on with the animations each input plays, as the time a user waits for them, and a timer
due inside an `Advance` fires at its own time before the time goes on to the end. So a time the app took when its
timer fired (when it last read a file, say) is earlier than `d.Now()` after it, by up to that `Advance` and the
animations since: check that it falls between `d.Now()` before and after, not that it equals the one after.

A screen that moves on by itself, a game's clock say, holds still while the user cannot see it: on each tick ask
`app.Covered()`, true while a popup is over the screen (until one that is closing is gone) or the splash is. A toast
does not cover it.

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

terminu-go implements **tdp v1.3.5**. The tdp version each release implements is listed in the [CHANGELOG](CHANGELOG.md).

## Links

- [CHANGELOG](CHANGELOG.md)
- [Developer notes](docs/dev-remarks.md) (in Traditional Chinese)
- [terminu design](https://github.com/vulcanshen/terminu)

## License

[GPL-3.0](LICENSE)
