# Changelog

Each release names the version of terminu design (tdp) it implements.

## Unreleased — tdp v0.7.0

- tdp v0.7.0 adds the project spec: how a family app's repo is laid out, versioned and released (Rules E6). The
  library and the showcase follow the Rules E8 table instead, so nothing changes
- A select or checkbox popup that scrolls is wide enough for its whole hint and its scroll position at the widest,
  as a menu already was: with 18 options at 80×24 its frame was 42 cells and dropped `Esc:cancel` (demou's DM3)

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
