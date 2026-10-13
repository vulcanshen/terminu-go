# Changelog

Each release names the version of terminu design (tdp) it implements.

## Unreleased — tdp v0.5.6

- The repository: module `github.com/vulcanshen/terminu-go`, GPL-3.0, READMEs in English and Traditional Chinese
- Built on Bubble Tea v1 and Lip Gloss v1, the same as the terminu apps: Bubble Tea v2's renderer and Lip Gloss v2's
  layout cannot be given the icon width, so a two-cell icon misplaces the frames
- `docs/coverage.md`: every requirement of tdp v0.5.6, where the showcase is to show it, and the test that is to guard it
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
- `App.InputTabWidth` for the user's `input_tab_width`: how many spaces Tab puts in a textarea
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
