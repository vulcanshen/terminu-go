# Changelog

Each release names the version of terminu design (tdp) it implements.

## Unreleased — tdp v0.5.1

- The repository: module `github.com/vulcanshen/terminu-go`, GPL-3.0, READMEs in English and Traditional Chinese
- Built on Bubble Tea v1 and Lip Gloss v1, the same as the terminu apps: Bubble Tea v2's renderer and Lip Gloss v2's
  layout cannot be given the icon width, so a two-cell icon misplaces the frames
- `docs/coverage.md`: every requirement of tdp v0.5.1, where the showcase is to show it, and the test that is to guard it
- The library's first parts: icon width (probe, variables, one width function), colours and dimming, screens with the
  chip row and footer, panels (lists, one value per row, pagers) with frames, capsules, hints, filter and modes, the
  Space menu, the global operation popup, key references, confirm, toast, the popup animation, and text, password and
  number inputs
- Every kind of input: select, checkbox popup, finder with a preview, file-picker, form, textarea, date and datetime
  pickers, color picker and slider
- Powerline symbols are measured apart from icons, with their own variables `<APP>__POWERLINE_WIDTH` and
  `TERMINU__POWERLINE_WIDTH`
- Viewers (notes) with tabs, keys of their own and a selection mode; menus whose rows open their full text and
  multi-step menus; confirms with hotkeys of their own; a loading popup grows with its content
- Terminals: a program in a PTY, in a panel or in a terminal popup, with the exit key, the app's own keys and
  history; its screen is laid out with the same widths as everything else (Unix only)
- The showcase's Panels, Inputs, Dialogs and Terminal screens
