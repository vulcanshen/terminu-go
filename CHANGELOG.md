# Changelog

Each release names the version of terminu design (tdp) it implements.

## Unreleased — tdp v0.7.4

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
