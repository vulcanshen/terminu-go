package terminu

import tea "github.com/charmbracelet/bubbletea"

// FilePreview is a panel's body that shows a file or a directory as a
// file-picker's preview does: a tree, a text file in colour with line numbers,
// an image, a PDF, hex; with Info, its four facts under it
// (components/input/file-picker). As a details panel it follows its list's
// cursor: set Path in the list's OnMove (layout/panel, details). It takes no
// keys and does not scroll, as a preview does not.
type FilePreview struct {
	Path string
	// Line is a text file's line to start near: a search's hit.
	Line int
	// FS is where it reads: the local disk when nil.
	FS PreviewFS
	// Icon is a tree entry's icon and its colour, as a FilePicker's.
	Icon func(path string, dir bool) (string, Color)
	// Info puts TYPE, SIZE, MODIFIED and MODE under the preview, after a
	// separator.
	Info bool
	// EmptyFact and EmptyHint are what it says with no Path.
	EmptyFact, EmptyHint string

	hide bool // the app's HideSecretsInPreview at the last draw
}

// Draw draws the preview in w × h, one cell from the frame on either side.
func (p *FilePreview) Draw(w, h int, _ bool) []string {
	if p.Path == "" {
		return EmptyState(w, h, p.EmptyFact, p.EmptyHint)
	}
	area := h
	if p.Info {
		area = max(h-5, 0) // a separator and four facts
	}
	lines := previewIn(p.FS, p.Path, p.Line, w-2, area, p.hide, p.Icon, tabCells)
	out := make([]string, 0, h)
	for i := range area {
		if i < len(lines) {
			out = append(out, padRow(lines[i], w))
		} else {
			out = append(out, Repeat(" ", w))
		}
	}
	if p.Info && h >= 5 {
		out = append(out, " "+Fg(Overlay0).Render(Repeat("─", w-2))+" ")
		for _, r := range fileInfoIn(p.FS, p.Path, w-2, Blue) {
			out = append(out, padRow(r, w))
		}
	}
	return Pad2(out, w, h)
}

// Key takes no keys: a preview does not scroll.
func (p *FilePreview) Key(string, tea.KeyMsg) (bool, tea.Cmd) { return false, nil }

// HasItem is false: a preview has no item of its own.
func (p *FilePreview) HasItem() bool { return false }

// Scroll is "": a preview does not scroll.
func (p *FilePreview) Scroll() string { return "" }

// MoveKeys are none.
func (p *FilePreview) MoveKeys() []KeyRow { return nil }

// Empty reports whether there is nothing to show, for the panel's loading
// and failure.
func (p *FilePreview) Empty() bool { return p.Path == "" }
