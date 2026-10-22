# Changelog

Each release names the version of terminu design (tdp) it implements.

## Unreleased — tdp v0.9.10

- A mode's name opens a character a frame: the junctions first, then one more character every 32ms, so every frame
  changes and the time goes with the name (`Drag` about 160ms, `Compare` about 256ms), no longer spread over 288ms
  (kbu KB30)

## v0.4.11 — tdp v0.9.9

- A mode's name opens from its middle on the frame and closes back to the line (components/layout/panel, the
  frame; kbu KB30): the junctions side by side first, then apart, the name showing from its middle, about 288ms by
  the clock whatever its length; switched halfway it turns where it is. Only the name moves: the mode and the
  frame's colour change at once, keys work all the while, and the title keeps the name's room. On panels, in popups
  and on PTYs. tdptest plays it to its end with `Auto` on and steps it with `Auto` off
- `App.QuitFlow`'s doc says what cannot be undone goes in the command it returns: with a popup open the lib runs the
  flow first, then may hold that command for `Enter`

## v0.4.10 — tdp v0.9.8

- With a popup open (a toast is not one) the first `q` or `Ctrl-C` never leaves at once (Rules K9, kbu KB29): an app
  with no `App.Quit` asks `Quit <app>?`, and a `QuitFlow` that would leave there and then asks it first, leaving on
  `Enter`; a flow that asks something of its own asks only that. `Esc` goes back to the popup; on a panel the app's
  flow is as it was; the second `Ctrl-C` still leaves at once
- A reorder's Drag mode menu leaves with `[Esc] Cancel`, the verb of its hint's `Esc:cancel`, not `[Esc] Put back`

## v0.4.9 — tdp v0.9.7

- `Space` in a mode opens the mode's Space menu (Rules K11; components/dialog/menu, in a mode; kbu KB28), on a panel,
  in a popup (a viewer's selection, a reorder's Drag) and on a PTY (History, selection: `Space` no longer goes to the
  program there). It is titled as where the mode is; `ITEM OPERATION` lists what can be done to the item under the
  cursor, run as its key (Compare's `[C]ompare with <anchor>`, `[v] Select`, then `[y] Copy`, `[Enter] Drop`), off
  when it cannot be done now; `<NAME> MODE` the mode's own keys, the movement keys only shown in Blue, the cursor
  passing them, leaving (`[Esc] Leave`) to run. `Esc` in the menu only closes it
- `ModeKey.Item`, `Label` and `MenuDesc` place an app's mode key in that menu; a `NoFooter` key is shown as a
  movement key
- In a mode the footer starts with `Space:menu`: `Space:menu ?:help <the mode's keys> Esc:<leave>`

## v0.4.8 — tdp v0.9.6

- The compare popup lays itself out again when the terminal changes size, as other popups do: under 96 columns it is
  UNIFIED at once, back over 96 the remembered SPLIT; the stretch on top stays in sight

## v0.4.7 — tdp v0.9.6

- The compare mode and the compare popup (components/layout/panel, compare mode; components/dialog/note, compare):
  `App.CompareAction(panel, list, Compare{Key, Can, Name, Text, Split})` is the Compare item action. Its key on a row
  enters the mode with that row as the anchor (`List.Anchor`); `j|k`, `u|d`, `gg|G` move; the key or `Enter`
  compares with the anchor in a `DIFF — <anchor> ↔ <other>` viewer, UNIFIED or SPLIT (`h|l`, remembered in `Split`,
  SPLIT off under 96 columns), only the stretches that changed with 3 lines either side; the same text is an Info
  toast; `Esc` leaves; an anchor no longer in the list ends it with a Warning toast (kbu)
- `ModeKey.Disabled`: a mode key that cannot be pressed now, in the disabled colour in the footer and the key
  reference, doing nothing (Rules M6)
- `gg` reaches a mode as one key

## v0.4.6 — tdp v0.9.5

- `List.Anchor`: the row a mode keeps picked while the cursor moves on (a Compare mode's first pick) is drawn whole
  in Yellow with Base text, not bold, no arrow, as a selected range; the cursor on it wins (kbu KB27)

## v0.4.5 — tdp v0.9.5

- Capsule chains (panel titles and tabs, screen chips, a popup's tabs) say every colour they use, so they look the
  same on any terminal ground (kbu KB26, tdp v0.9.4 and v0.9.5): an unselected part is on Mantle `#181825`, every join
  is a slant the way of the left end, in the chain's colour on Mantle — U+E0BC out of a selected part, U+E0BA into
  one, the thin U+E0BB between two unselected ones; the ends take the ground of the part beside them, nothing behind
  them. U+E0B1 is no longer used
- Dimming fades toward `#0c0c13`, one step darker than Crust, not Base: `dim(c) = c × 0.45 + t × 0.55`, never
  lighter. Colours darker than Base dim too: under a popup an unselected capsule part's Mantle becomes Crust
  `#11111b`, and Base text on a dimmed cursor block becomes `#14141f`
- `Mantle` joins the colours

## v0.4.4 — tdp v0.9.3

- The lib sends its own timers — a popup's or a toast's opening and closing frames, a toast's time — whatever the app
  returns: an app that keeps only one of two commands (two toasts in one update, a popup and a toast) or drops them
  all no longer stops every animation for good, and the toast still goes. App methods return the same commands; one
  returned and also sent by the lib runs once. tdptest's `Do` sends them too (filu, 2026-10-21)
- `MoveAside`'s question is `Move it to <name>.bak and start fresh?` (Rules E3)
- `App.NoLongerUsed(paths...)` (Rules E3, files no longer used): at every start, an INFO in the App Log for each file
  of the app's own a newer version no longer uses that is still there, `No longer used: <path>; delete it if you
  don't need it`; nothing deleted, no toast

## v0.4.3 — tdp v0.9.2

- The App Log's entry for a file moved aside starts with a capital, as a toast does: `Moved … to …` (note-78)
- `MoveAside`: `Enter` has the app go on as on its first start (Rules E3): the lib made no file before and still
  makes none; the doc says what `fresh` does now. The showcase's settings file comes back only with a save.

## v0.4.2 — tdp v0.9.1

- `App.MoveAsideRecord` (Rules E3, when a file cannot be read): for data kept one record per file, `Enter` only
  renames the record that cannot be read to `.bak` and runs the app's reread, nothing new made in its place; the
  question is `Move it to <name>.bak and go on without it?` (kamonu KM18)
- `Column.Line` marks a list column of lines of text (a line of source code, of a log): a Tab in it is drawn as
  `input_tab_width` spaces, other control characters still by name in Red (components, control characters; webu
  WB21)

## v0.4.1 — tdp v0.9.0

- tdptest: `App.Tricky` says the data `Seed` puts in has tricky values (`Tricky`): `Check` then leaves a colour code
  drawn as text alone, the data's, not a colour of the app's own left out of its spans (locku LK21)

## v0.4.0 — tdp v0.9.0

tdp v0.9.0 settles how control characters are drawn (G100), gives every app an App Log and a way out of a file it
cannot read, and keeps History and the selection for interactive terminals. Upgrading: see "Changes to make" below.

- Control characters are drawn in three groups (components, control characters):
  - In names and short texts — a list's cells and location, values and options on forms and panels, menu labels,
    descriptions and section titles, popup titles and subjects, toasts, confirms, error popups, an input's prompt and
    error rows, empty states, the statusbar, tab names, finder items — every control character, Tab, `\n` and the
    ESC of a colour code too, is its name in Red (`\t`, `\n`, `\x1b`), on a cursor Red under Base. `Escape` does
    it for an app's own drawing (a Canvas, a popup of its own)
  - Colours of the app's own in those places come as `Span`s: `Row.Styled`, `Field.Styled`, `FormField.Styled`, and
    `App.Status` returns `[]Span`; `ColorSpans` replaces `ColorText`, `SliderSpans` replaces `SliderTrack`
  - Reading content — a viewer, a panel's log (`Pager`), a finder's preview, a file preview — is drawn as a terminal
    draws it: `\n` starts another line, one in a line the app gave too; a `\r` inside a line leaves its last state
    (`10%\r100%` is `100%`); colour codes colour, a file's own too; any other control character or escape sequence
    is its name in Red. `Reading` draws reading content that way for a body of the app's own (kbu KB24)
- A proposal (`TextInput.Suggest`) taken with nothing touched is checked for control characters too (input-52)
- Every app has an App Log (components/dialog/note, Rules K1, F5): `!` on a panel opens it, a viewer, newest last;
  each entry is its time, its level (`INFO` `OK` `WARN` `ERR`) and what happened. Every toast and error popup goes
  in by itself; `App.Log` logs the rest, from any goroutine. The footer shows `!:log`, or `!:N errors` in Red,
  `!:N warnings` in Peach, until it is opened. `!` is a core key: no hotkey and no `CoreKey` may take it
- Interactive terminals (components/dialog/terminal): `Terminal.Interactive` replaces `History` and `SelectKey`. An
  interactive terminal has History on `PgUp` and the selection on `Alt-v`, the family's key; any other PTY gives
  every key to its program, `PgUp` too. While the program fills the whole screen `PgUp` is its own and the hint
  leaves `PgUp:history` out. `Terminal.AllKeys` hands every key to the program but the exit key and the app's own
  `Keys`
- `App.MoveAside` (Rules E3, when a file cannot be read): a `MOVE ASIDE` confirm with the path and why,
  `Enter:move Esc:quit`; `Enter` renames the file to `.bak` (or `.bak.1`, `.bak.2`…, never over another) and runs
  the app's fresh start; `Esc` leaves the app and `Run` returns the path and the reason, for main to write to stderr
- `FormResult.Info` makes a form action's toast an Info one instead of Success: which a thing done is, is the app's
  (components/dialog/toast)
- `Style.Render` goes on in its style after a styled piece inside it ends
- The app's `input_tab_width` is used without `Run` too, as in an app's own tests
- tdptest: `Check` opens the App Log too, and with `Driver.ColourCodes`, which it sets, a colour code drawn as text
  (one left in a cell or a value) fails the test; its tricky values go everywhere as they are now

### Changes to make

- Colours you put in `Row.Cells`, `Field.Value`, `FormField.Value` or anything else listed above now show as
  `\x1b[…` in Red: give them as spans instead (`Row.Styled`, `Field.Styled`, `FormField.Styled`), the text staying
  in `Cells` and `Value`. Text from outside you draw yourself goes through `Escape`
- `App.Status` returns `[]Span`; `ColorText(c)` becomes `ColorSpans(c)` in a `Styled`, `HexText(c)` in `Value`;
  `SliderTrack(…)` becomes `SliderSpans(…)` in a `Styled`, the number in `Value`. `tdptest.Check` finds a colour
  code left in a cell or a value
- A body of your own that shows a log or a program's output draws its lines with `Reading`
- `Terminal.History` and `SelectKey` are gone: mark a terminal the user types in `Interactive: true`
- A `CoreKey` or a hotkey on `!` is refused: move it
- A file of your own you cannot read: `app.MoveAside(path, err, startFresh)` at the start
- Make your own timers with `app.Tick`, and add `tdptest.Check` to your tests (README, "Testing an app")

## v0.3.0 — tdp v0.8.3

- New package `tdptest`: drives an app in a Go test as a user does — keys by their tdp names, text typed, sent at
  once or pasted, the terminal resized, messages of the app's own — and checks every frame for tdp L4 (each row
  exactly as wide as the terminal; no control character reaching the terminal)
  - The app runs on the test's clock: after each input the animations play to their end; `Advance` moves the time
    (a toast going, a timer), `Step` plays one frame, `Driver.Auto` off stops the time between inputs
  - The app's commands run as in the program; `WaitFor` waits for a slow one. `tea.Batch` and `tea.Sequence` are
    run as Bubble Tea runs them, a quit is noted; a command that panics fails the test
  - `EachIconWidth` runs a test with icons and powerline symbols one cell wide, icons two, both two (tdp E5)
  - `Check` launches the app as for the first time on many terminal sizes and icon widths, each launch in a new
    sandbox (`Env` gets its `HOME`), and walks the `Space` menu, the global operation popup and `?` on every screen
    and panel, each closed one layer an `Esc` (tdp K4), then a terminal under the least size and back (tdp L1); with
    `go test -short`, the least size and 80×24 only
  - A failure says the size, the keys pressed, what `Note` added and the screen; a control character reaching the
    terminal is shown in its row
  - `Tricky`, `Sizes` and `Mashes` (with `Driver.Mash`): values, terminal sizes and fast keys that have broken screens
- `App.Tick` is `tea.Tick` on the app's clock: under a test that drives the app, the test moves its time. An app's own
  timers are made with it to be tested (README, "Timers: `App.Tick`")
- Characters that come in one key are characters, however many: typed fast, `end` is three letters, not the End
  key, and `ctrl+c` does not quit (Rules M5)
- An escape sequence in a viewer's line that is not a style is shown, not obeyed: its ESC drawn by name in Red, the
  rest as it is; a style stays a style (components/dialog/note)
- A textarea whose old text has its line breaks as `\r` or `\r\n` is not changed by opening it: `Esc` closes it
  without asking and `Enter` writes nothing back (components/input/textarea)
- README: the quick start's column is `Shrink` (`Flex` went in v0.2.1)

## v0.2.2 — tdp v0.8.3

- A one-line input's prompt folds when it is too long, several sentences and its own line breaks (a web page's
  `prompt()`), the popup growing with it; at its tallest the last row it keeps ends in `…`
  (components/input/README, tdp v0.8.3)
- A column the panel's edge falls in is cut its own way, with `…` (from the end, or in the middle), the columns
  after it not drawn; it was chopped at the frame (layout/panel)

## v0.2.1 — tdp v0.8.2

- tdp v0.8.2 rewrites how a list's columns give way, reads file previews in the background and folds the "what each
  row is" section of a key reference; the library follows it below, with what the family found upgrading to v0.2.0

- `TextInput.Unchanged`, `PasswordInput.Unchanged` and `NumberInput.Unchanged` run when `Enter` takes the value the
  input opened with: nothing is written back and `Done` is not called, but a flow can end (`CloseTop`) and a page
  waiting for an answer gets the value it offered (Rules K3, F6)
- `Form.OnCancel` runs when a form closes without being submitted, as an `Ask`'s and an input's do (Rules F6)
- `Terminal.Empty` reports a program that has printed nothing yet, so the panel's `LoadingText` shows meanwhile
- A hotkey of several letters goes in place at the start of its label in either case too: `[go]to` on `Goto`
- A form's key reference says what `Enter` does on the kinds of items the form has, not all of them (Rules M6)
- A form button's `FormResult.Toast` is a Success toast: the action is done
- A list's column widths follow tdp v0.8.2: `Column.Shrink` marks a column of long values, which with the other
  shrinkable ones shares the width the short columns leave, cut its own way, down to `Column.MinWidth` (12 when
  unset); short columns keep the width of their widest value. Only when the shrinkable ones are at their least are
  columns taken away, in `Drop` order. Nothing narrows in proportion any more: a list with no `Shrink` column is
  cut at its right edge (layout/panel)
- A finder's and a file-picker's file previews and facts are read in the background, off the update loop: typing
  and moving never wait for a slow or remote file system. Meanwhile the preview's title has the loading icon, the
  box says `(loading…)` in grey and the facts' values are blank; what was read is kept for the popup's life
  (components/input/file-picker)
- A key reference folds a long description in its "what each row is" section in the right column, the name column
  blank under it; the sections of keys stay one line a key (components/dialog/note)
- `App.CloseAll`'s comment says every popup plays its closing at once, as it does

## v0.2.0 — tdp v0.8.1

- tdp v0.8.1 writes down G98 and G99 as the library does them (a list's location row stays when it is empty; an
  unwrapped viewer scrolls across under the selection cursor); its Windows cache folder and item-by-item moves are
  the app's
- tdp v0.8.0 settles the family's refactor; the library follows it in the third batch below. G96 is settled as
  the library had it but for values: text on the screen draws a tab as `input_tab_width` spaces, a value on a
  form or a panel as a red `\t`; G97 as the library had it (a preview from a line puts it on the third row)

- tdp v0.7.3 writes down that text which is only shown (a confirm, a toast, an error popup, the splash) does not
  draw the space it breaks at, as the library already did (G95); the checklist now counts genes, which the library
  does not apply
- tdp v0.7.4 changes only the checklist (an item can be judged "cannot tell"), so nothing changes
- A popup on a terminal shorter than its frame (5 rows and under, 1×1) is cut off instead of panicking; on a short
  screen a popup is never taller than its height, the blank rows inside its frame giving way first, so a form on a
  10-row screen no longer covers the footer (Rules F7)
- An input too short for its prompt keeps its typing row and its error row in sight
- An input closes without calling `Done` when its value is the one it opened with: typed and deleted back to
  empty, or its suggestion thrown away and typed again (Rules K3)
- `ErrEmpty` in an error row names the value: `Name can't be empty`
- A truncated styled line draws its `…` in the style of the text it stands for, not the terminal's default colour;
  `TruncateLeft` too, and plain text gets no reset
- A tab in the text of a panel or a popup (a pager, a list, a menu, a select…) is drawn as `InputTabWidth`
  spaces, as in the viewer, and `Width` counts it so; it was sent to the terminal and pushed the row past the frame
- A terminal resized while its program ends no longer races the reader closing its PTY (`go test -race`)
- A file preview's tree draws its second level two cells in, under the first level's name
- A viewer's key reference keeps the name of what it shows as it is: `FIRST PAGE — Pride and Prejudice KEYS`
- A form's key reference lists `Space` only when the form has checkbox options and `e` only when a field is wrong,
  either in the disabled colour where the focus cannot use it (Rules M6)
- A form's focused field no longer gets a `…` after a value that fits when the arrow is two cells wide
- With `App.ShiftTab` a panel's key reference writes `Tab|Shift-Tab`; its app-wide section leaves out hidden
  global actions
- `Commit` on every input (text, password, number, textarea, select, checkbox popup, colour picker, date picker,
  file-picker, reorder) writes a changed value back when `Enter` takes it: an error keeps the input open with its
  value and opens an error popup over it, instead of the input closing first and the value being lost
- `OnCancel` on a confirm (`Ask`) and every input runs when it closes with no answer, by `Esc` or by the popup
  under it closing, so a web page's prompt or ssh's askpass hears it was cancelled
- `Action.AskFor` makes an action's question when it runs, naming the item under the cursor, or asks nothing; run
  from a menu, `Esc` on its confirm goes back to the menu
- `App.ErrorAbout` and `App.MenuAbout` keep the name of the one thing they are about as it is in the title:
  `RENAME FAILED: notes.md`, `OPEN WITH: notes.md`
- `Textarea.Check` holds back a changed text that fails it, as a one-line input does
- `TextInput.Search` takes a pasted control character in a value that is only searched for
- A one-value-per-row panel puts what its `Enter` does in the Space menu and the key reference, and has an empty
  state of its own (`Values.EmptyFact`, `EmptyHint`)
- `App.LoadSelect` and `App.LoadChecks` open a select or checkbox popup before its options are there, loading in
  its title, as a `Finder` does
- An app popup's body may have `Scroll()`: its scroll position on the bottom frame line
- `Stay` makes a one-line input a step of a flow (current PIN, new PIN, confirm PIN): it stays under the next step
  and `Esc` goes back to it; `App.CloseTop(n)` ends the flow, `App.CloseAll` the whole stack
- `TerminalKey.Keys` lets one of the app's keys in a PTY take several keys, written once (`Alt-←|→|↑|↓`);
  `Terminal.OnFirstOutput` and `Spoke` tell the app its program has started printing
- `PreviewFS` lets a file preview read a file system the app gives, such as a remote host's over SFTP
  (`Finder.FS`); `FinderItem.Line` starts a text file's preview near a search's hit; `FilePreview` is a panel's
  body that previews a file, for a details panel
- `App.Canvas` and `CanvasKey` let an app draw a whole screen itself, a lock screen, with the lib's popups and
  toasts over it
- `App.OnResize` tells the app the terminal changed size
- A `Success` toast says an action is done: `SUCCESS` in Green, 2200ms (components/dialog/toast)
- A toast has as many lines as it needs, no longer three at most; `App.ToastLines` caps them, the last one kept
  ending in `…`
- A hotkey letter goes in place of the first such letter in its label in either case, written as the key is
  pressed: `[n]ew`, `D[e]lete`, `Ex[P]ort` for Shift-P, `[q]uit`; only a letter not in the label goes in front,
  `[x] Reset` (Rules M5)
- An icon's and a powerline symbol's width are also read from `LC_TERMINU_ICON_WIDTH` and
  `LC_TERMINU_POWERLINE_WIDTH`, which an app with ssh brings to the remote end (`RemoteIconWidth`,
  `RemotePowerlineWidth`): after `<APP>__` and `TERMINU__`, before asking the terminal (Rules E5)
- A list's columns that do not fit are taken away whole, their headers too, in the order `Column.Drop` gives,
  the name (`Drop` 0) never; only then are the columns left narrowed, in proportion. `Column.Flex` does nothing
  now (layout/panel)
- `Column.Middle` cuts a value in the middle, `demo-app…66-gm72n`; `TruncateMiddle` does it to any line
- A list of one column with no `Title` draws no header
- `Row.SortGroup` keeps a list's rows apart when it is sorted, the lower group first and sorted within each:
  directories before files (layout/panel)
- A sorted list puts the rows its keys cannot tell apart in the order of their IDs, so the same rows always come
  out in the same order (layout/panel)
- `List.SetMarked` marks rows by ID, rows the panel filter hides too
- `List.Select` reports whether the row is shown: after adding an item the app selects it, and the next frame
  scrolls it into sight (Rules K3)
- `Panel.Name` is the name of what a panel shows, kept as it is after its label: `[2] PODS: nginx-7d4f`, or the
  whole label, `[1] prod-web-01`; its Space menu and key reference write it so too
- On a panel's first tab the blank between `[N]` and the label is as wide as a join, so the text does not move
  when the tab changes where a powerline symbol takes two cells (layout/panel)
- A popup's hint that does not fit keeps its last `Esc` pair, the pairs before it going first:
  `Enter:install Esc:cancel`, then `Esc:cancel` (components/layout/popup)
- A capsule whose own text is too long is cut from the end with `…`, its slanted ends drawn and a segment of
  line kept before the corner: `┏[2] DEPLOYM…━┓` (layout/panel)
- An empty state too long for its panel folds at its spaces, every line centred one cell from the frame, instead
  of being cut (layout/screen)
- An input closes on the value it opened with only when that value passes its check: one that opened empty and
  may not be empty holds `Enter` back with its error, nothing typed or typed and deleted (Rules K3); a password
  that opened empty closes on `Enter` with nothing typed instead of writing an empty one back
- `TextInput.Suggest` and `NumberInput.Suggest` offer a value the app proposes in grey when there is no old one;
  `Enter` with nothing touched takes it, checked and written back (components/input/README)
- `NumberInput.Optional` lets a number be empty, not checked against its range; one that may not be empty says
  `Port can't be empty`; `NumberInput.Unit` puts the unit under the value, in grey (components/input/number)
- A password's unlock look has no prompt row: two blank rows, the typing row in the middle, a blank row, the error
  row, 7 rows with the frame (components/input/password)
- `PasswordInput.PausedUntil` pauses it, after too many wrong PINs: the error row counts down each second and
  every key but `Esc` is swallowed (components/layout/popup)
- A colour picker for no colour yet (`Value` not `Set`) shows `none` in grey and no swatch for old, starts new from
  `#000000`, and always writes back on Choose, `#000000` too, which used to count as no change
  (components/input/color-picker)
- A value on a form or a panel writes its control characters as their red names, `\n`, `\x1b`, and a tab as
  `\t`, as a typing row does, the app's own colours kept (components/input/README, Rules E3)
- A form's typing note takes only `Enter` and `Esc`: `q`, `?` and whatever else is typed after it go nowhere,
  `Ctrl-C` still works (Rules K9); it folds when too narrow, as an error popup now also does on every resize
- Every row of a form keeps as much room on its right as the focus block's arrow, two cells where a powerline
  symbol takes two (components/dialog/form)
- `FormButton.Open` opens a popup over the form, a terminal popup say, and `Back` reads what was done there into
  the form as it closes (components/dialog/form)
- Run from a menu, an action that opens a popup now keeps the menu under it, to come back to, as Rules T1 keeps the
  source by default; `Action.Clear` clears the menu as it opens (the item gone, a shell, another screen, a
  drill-down). One that opens nothing still closes the menu. A confirm before it is only a door: `Esc` goes
  back to the menu, and an error popup an action opens closes back to it
- Several popups closing together (`CloseAll`, `CloseTop`, a source cleared) each play their own closing at the
  same time, in their own layer's colour, instead of only the top one (Rules F2)
- `Action.Live` gives a menu row's label and description each time it is drawn: a work list follows its work;
  `Action.EnterHint` is the word after `Enter` on that row (`Enter:cancel`) (components/dialog/menu)
- `LoadingFrame` is the loading icon as it is now, for a row of the app's own; `App.Spinning` keeps it turning
- The hint of a streaming panel or a panel's PTY without the focus dims as every other panel's does; only its
  content stays bright (Rules T2)
- While a terminal popup is open the footer is not drawn: the ends of the last row are blank, no footer words
  show beside the popup (components/dialog/terminal)
- A file preview's metadata writes `1 item` for a directory of one, and a dash on every row for a file gone
  since it was listed (components/input/file-picker)
- A key reference lists movement and scrolling keys only where they move something: none for an empty list,
  none to scroll content that all fits (a pager, a confirm, an error popup) (Rules M6)
- A form, a menu, a value panel without tabs and a reorder are one-dimensional lists: their key references write
  `h|k` previous and `j|l` next, and a reorder moves with `h|l` too, held or not (Rules K12)
- A value panel's scroll position counts fields, a radio's options one, as a form's does (layout/panel)
- `Field.Desc` says what a value panel's row means: its key reference starts with a "what each row is" section,
  the names in the text colour (layout/panel)
- `App.Focus` moves the focus to a panel, switching to its screen, and `App.ShowScreen` switches screens, for an
  action the user expects to move them (Rules K3). The focus stays on its panel when the app adds or takes away
  panels, and a screen whose panels got fewer no longer panics
- `App.QuitFlow` makes the quit flow an action of the app's own, a menu of ways to leave; a second `Ctrl-C`
  during it still leaves at once (Rules K9)
- An app has a least size, 80 × 40 unless `App.MinWidth` and `MinHeight` say less (never more): on a smaller
  terminal only its name and `needs a terminal of at least W×H` are drawn in the middle, no key acts nor reaches
  a PTY but `Ctrl-C` twice, which leaves, and all comes back as the terminal grows (Rules L1). Apps that work in
  a smaller terminal set their own
- `App.PlaySplash` plays a splash of the app's own accord, any splash it has; the reveal now keeps to the time
  since it opened, so ticks that come late no longer put the text off
- Every viewer has a selection mode: with no `Viewer.Select` it is `v`, Visual; with no `Copy`, `y` puts the text
  on the terminal's clipboard (`Clipboard`, OSC 52) and a Success toast says `Copied` (components/dialog/note)
- `Viewer.NoWrap` keeps a long line on its row, cut with `…`; `Viewer.NoNumbers` leaves out the line numbers, for
  a table the app lays out. In the selection mode every line scrolls across to keep the cursor in sight, as vim's
  nowrap does, `…` where each is cut on either side; a terminal's selection mode does the same (G99)
- `Viewer.Hits` and `Current` draw a search the app runs: every hit underlined in its own colours, the current
  one as the cursor block; `FindText` finds them (components/dialog/note)
- A viewer's key reference lists its scroll keys only when its lines do not all fit
- An error popup opened before the terminal's size is known (in `App.Start`) folds at the real width once it is,
  not a character a row
- A file preview shows an escape sequence written in the file instead of obeying it: ESC by its red name, the rest
  as text, nothing lost; its metadata writes `MODE` as `ls -l` does (`drwxrwxrwt`, `-rwsr-xr-x`)
  (components/input/file-picker)
- A terminal's exit key is the app's (`Terminal.ExitKey`, Alt-Esc when unset; another asks first with
  `ExitAsks`), shown as it is now; its reserved keys may change while it runs (Rules K10)
- PgUp puts a terminal in the History mode: `j|k`, `u|d`, `gg|G`, `PgUp|PgDn` scroll back, `Esc` goes back to the
  live screen, any other key leaves the mode and goes to the program; the mode is on its frame (Rules K11)
- `Terminal.SelectKey` puts a terminal in the selection mode, a viewer's: the screen holds still, the cursor goes
  up into the history, `v` selects, `y` copies (`Terminal.Copy`, or the clipboard with a Success toast)
  (components/dialog/terminal)
- What a finder matches is the app's: `FinderItem.Match` instead of the text shown; `Finder.Search` runs the
  app's own search at every change of the typed words, its results shown as they come, typing never waiting
  (components/input/finder)
- The README says how a screen the library does not own uses its parts: drawn with `App.Canvas`, popups over it
- `List.Location` is a list's location row, where the user is, in Lavender above its header, cut from the front
  (layout/panel)

## v0.1.13 — tdp v0.7.2

- tdp v0.7.0 adds the project spec: how a family app's repo is laid out, versioned and released (Rules E6). The
  library and the showcase follow the Rules E8 table instead, so nothing changes
- tdp v0.7.1 adds a checklist, tdp broken into items for judging how much of it an app meets; the library does not
  apply it
- Text wraps one way wherever it wraps (tdp v0.7.1, the components' "Wrapping"): at spaces, a word moving whole to the
  next row, and between any two wide characters, so Chinese fills a row first; a word is cut by cells only when it is
  longer than a row, and a wide character never in half. The textarea and the viewer cut a word wherever the row
  ended; a confirm, a toast, an error popup and the splash moved a run of Chinese whole to the next line
- In the textarea, moving up or down onto a row that folded short stops on its last character, not on the first of
  the next row
- A finder with no results, a search still listing before its first result included, leaves `Enter` out of the hint,
  as `Enter` does nothing there (`Tab:list Esc:cancel`, `Tab|/:filter Esc:cancel`), and puts it back once there are
  results (G93). On a checkbox popup's list `Enter` still applies the checks, now with no results too, and only
  `Space` is left out. The width stays what it was when the popup opened
- The colour picker's hjkl move only among R, G and B, wrapping there; `Choose` is reached only with `Tab`, as a
  form's two areas
- A grouped list with marks draws them in the two cells its items are further in: the text stays three cells in, as
  with no marks
- Every group of the showcase's `Open with` menu has a title, the first too (Rules M8); `MenuSection` says so
- A select or checkbox popup that scrolls is wide enough for its whole hint and its scroll position at the widest,
  as a menu already was: with 18 options at 80×24 its frame was 42 cells and dropped `Esc:cancel` (demou's DM3)
- Typing in a panel filter that matches nothing, the footer leaves `Enter` out: `Tab:list Esc:clear` (tdp v0.7.2)
- tdp v0.7.2 also writes down what the library already did: in the textarea and the viewer a space that does not
  fit starts the next row (G94); a panel's hint with no item in its list leaves out `Enter` and the keys on an item;
  a popup at its tallest leaves the chip row or the statusbar and the footer in view, covering the divider

## v0.1.12 — tdp v0.6.15

- A checkbox option can be marked not available: `Checks.Unavailable` in a checkbox popup, `FormField.OptionUnavailable`
  and `Field.OptionUnavailable` for a group drawn in place. It is drawn in Surface2, the tick too, and otherwise works
  like any other option: the cursor stops on it and it flips. What it means is up to the app
- The showcase marks `kube-system` and `vault` in the `Namespaces` popup and `X11` in both `Agent` groups not available

## v0.1.11 — tdp v0.6.12

- The header row of a one-value-per-row panel (`Values.Header`) is Overlay0, so it no longer runs into the Blue labels
  under it. A list's header stays Blue

## v0.1.10 — tdp v0.6.11

- `TextInput`, `PasswordInput` and `Textarea` take a `Name`, as `NumberInput` does: the value's name in its errors when
  the title says what the popup does (`Name can't have control characters` on a `SAVE AS`, not `Save as can't have
  control characters`). Left unset, the title names it, as before
- The showcase's `RENAME` input names its value `Name`

## v0.1.9 — tdp v0.6.10

- `App.Popup`, `Popup` and `PopupBody`: a popup of the app's own on the library's popup stack. The app draws what is
  inside the frame and handles its keys; the library draws the frame, the title and the hint in the layer colour, plays
  the animation, fades what is under it, and does the `Esc`, `?` and `q` the app leaves. The library's confirms, inputs,
  error popups and key references open over it. `App.ClosePopup` closes it with everything over it
- A `PopupBody` with `QuitNote() string` adds a Peach line to the quit confirm while it has something unsaved; the quit
  flow then asks even when `App.Quit` is nil
- A form whose submit opens another popup closes only itself: the new popup stays. A popup's layer colour counts only
  the open layers under it (tdp v0.6.10): a popup that opens while the one under it closes (a menu clearing itself for
  the terminal it opens, a form's submit) has its own layer's colour from its first frame, where it took the next
  layer's until the closing one was gone
- The quit confirm closed together with the popup under it ends the quit flow: the next `Ctrl-C` asks again
- `examples/minimal`: the smallest app, shown in the README's Quick start

## v0.1.8 — tdp v0.6.9

- `FormButton.Subject` and `Form.ButtonSubject`: the name a button acts on, after its verb on the button. The hint turns
  only the verb to lower case and keeps the name as it is (`Overwrite KBU.svg` → `Enter:overwrite KBU.svg`). A failure's
  error popup puts the name after `: ` in its title, kept as it is

## v0.1.7 — tdp v0.6.8

- `Form.Buttons` and `FormButton`: a form's action area can hold more buttons, in rows on the right, the main submit
  at the bottom right. A button is another submit (`Submit`, e.g. Save As) or an action that keeps the form open
  (`Run`, e.g. Test): it checks only the fields it `Needs`, runs off the update loop with the loading icon after the
  title, and its `FormResult` shows as a toast, a note, or an error popup that Enter after it tries again
- While a button's work runs, Enter on it does nothing and the hint leaves its `Enter` out. `Form.SubmitLong` and
  `FormButton.SubmitLong` are submits that take long (Connect), run the same way. Closing the form ends the `ctx` that
  `Run` and `SubmitLong` get, and drops what they return: no toast, note or error popup, no `Apply`
- A form moves in two areas: `j|k` and `gg|G` stay in the field area (the error row its last item) or the action area,
  each wrapping on its own, and `Tab|Shift-Tab` go between them. `G` no longer goes to the button; the first `Tab`
  lands on the main submit. On the error row the hint says `Tab:actions`
- On a cursor block a swatch (a run of `█` in a colour, as `ColorText` writes) keeps its colour; the rest turns Base.
  This is so on a form, a value panel and a list

## v0.1.6 — tdp v0.6.6

- `App.Reorder`: a popup for putting items in an order by hand, the panel's Drag mode in a popup. `D` picks up an
  item, `j|k` move it (wrapping at the ends), `Enter` drops it, `Esc` puts it back; `Enter` then applies the order,
  and `Esc` asks first once the order changed. `Done` gets the options' indexes in their new order
- `List.Draggable`, `Hold`, `MoveHeld`, `Drop`: the row held in a panel's Drag mode is drawn as the cursor in Yellow
  with the swap glyph in the mark column, and moves wrapping at the ends. The Drag mode and its keys stay the app's
- In a popup's mode (a reorder's Drag, a viewer's selection mode) the app's own core keys do nothing, as in a panel's

## v0.1.5 — tdp v0.6.5

- `Checks.Cursor`: the option a checkbox popup opens with the cursor on, scrolled into sight; the first by default

## v0.1.4 — tdp v0.6.5

- The colour picker: `Tab|Shift-Tab` go between R, G, B and `Choose`, back to the channel left, as on a form; the hint
  says `Tab:actions` on a channel and `Tab:fields` on `Choose`, and the popup is 40 cells wide instead of 37 (Rules K2)
- tdp v0.6.4 adds Windows to where an app keeps its settings, data and cache (Rules E3). The library keeps none of
  these, so nothing changes; a `~/` path in the file-picker is the user's home directory, `%USERPROFILE%` on Windows

## v0.1.3 — tdp v0.6.3

- `Panel.Details` and `Action.ToDetails`: Enter on a list may send the focus to its details panel, on the items the app
  says. This is the one place Enter changes panels, and the library refuses it on any other key or panel

## v0.1.2 — tdp v0.6.2

- Groups in a list: `Row.Group`. Each group's title is Blue and upper case, and the cursor skips it. The group's rows are
  two cells further in and stay together when sorted. A group with no row shown has no title
- `List.OnMove` and `Values.OnMove`: run when the cursor comes to another item, before the frame is drawn, so a
  details panel follows another panel's cursor in the same frame

## v0.1.1 — tdp v0.6.1

- A popup that acts on one thing can name it: `Subject` on forms, text, password and number inputs, textareas,
  confirms, date and colour pickers, sliders, finders, selects and checkbox popups. The title reads `EDIT: filu`, the
  name as it is, as a terminal's already did

## v0.1.0 — tdp v0.6.0

- The repository: module `github.com/vulcanshen/terminu-go`, GPL-3.0, READMEs in English and Traditional Chinese
- Built on Bubble Tea v1 and Lip Gloss v1, the same as the terminu apps: Bubble Tea v2's renderer and Lip Gloss v2's
  layout cannot be given the icon width, so a two-cell icon misplaces the frames
- `docs/coverage.md`: every requirement of tdp v0.6.0, where the showcase is to show it, and the test that is to guard it
- The library's first parts: icon width (probe, variables, one width function), colours and dimming, screens with the
  chip row and footer, panels (lists, one value per row, pagers) with frames, capsules, hints, filter and modes, the
  Space menu, the global operation popup, key references, confirm, toast, the popup animation, and text, password and
  number inputs
- Every kind of input: select, checkbox popup, finder with a preview, file-picker, form, textarea, date and datetime
  pickers, color picker and slider
- Powerline symbols are measured apart from icons, with their own variables `<APP>__POWERLINE_WIDTH` and
  `TERMINU__POWERLINE_WIDTH`
- Viewers (notes) with tabs, keys of their own and a selection mode; menus whose rows open their full text and
  multi-step menus; a loading popup grows with its content
- Terminals: a program in a PTY, in a panel or in a terminal popup, with the exit key, the app's own keys and
  history; its screen is laid out with the same widths as everything else (Unix only)
- File previews: a directory's or an archive's tree, an image, a PDF's first pages, text in colour, anything else in
  hex (eight bytes a row), with the file's type, size, time and mode beside it; `App.HideSecretsInPreview` for the user's
  `hide_secrets_in_preview`
- `App.InputTabWidth` for the user's `input_tab_width`: how many spaces Tab puts in a textarea, and how wide a tab
  is drawn in viewers and file previews
- Keys doing one thing are joined with `|` in hints, the footer and key references, the bar in the words' colour
- Every typing row that filters starts with an angle; finder lists say `Tab|/:filter`
- A viewer has no cursor until its selection key: the first press brings the cursor out, the second starts the
  selection
- `Action.Detail` for a menu row that opens its full text: the menu stays and says `Enter:detail`
- `App.Splash`: the mechanism of tdp chapter S; `Run` refuses an app without one. `NewSplash` draws the family splash
  from an app's `docs/icon.svg`: the mark cell for cell, the text under it and the reveal
- `App.CoreKeys`: core keys of the app's own, the same on every panel and popup; `Run` refuses a hotkey on one, and a
  global or screen key on a core key or a movement letter
- `Panel.OtherKeys`: a section of a panel's key reference about another surface (a PTY, a grid)
- A panel's hint lists what cannot run now in the disabled colour; `Panel.HintPressableOnly` leaves it out
- `FormField.OptionDisabled`: an option of a form field that cannot be chosen now; a form refuses a field with more
  than five options
- A test for every requirement in `docs/coverage.md` that a unit test can check
- The showcase's Panels, Inputs, Dialogs and Terminal screens
- `FilePicker.Dirs`: the file-picker that chooses a directory, with a `. (this directory)` row
- A form's `Tab|Shift-Tab` go between the field area and the action row; Space ticks a checkbox drawn on a form
- Every one-line input starts with an angle; pasted control characters are kept and drawn in red
- Viewers number their lines, fold long ones and scroll a row at a time; `Viewer.Lang` colours their content as the
  file preview does
