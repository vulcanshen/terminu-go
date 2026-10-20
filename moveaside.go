package terminu

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

// MoveAside asks what to do with a file of the app's own it cannot read
// (Rules E3, when it cannot be read): its settings, its state or its data,
// whatever the reason (a bad format, a broken file, no permission). The
// confirm, MOVE ASIDE, names the path and why:
//   - Enter (move) renames the file to <name>.bak (.bak.1, .bak.2… when that
//     is taken: never over another), logs it and runs fresh, where the app
//     starts a new one in the same place (its defaults for settings, empty
//     state or data) and goes on.
//   - Esc (quit) leaves the app: Run returns the path and the reason, for
//     main to write to stderr as it does any error of Run.
//
// A cache that cannot be read is not asked about: throw it away and build it
// again. Something outside the app that cannot be reached (a cluster, the
// network) is not the app's own: the app goes on and says so where it shows
// it (layout/panel, loading and errors).
func (a *App) MoveAside(path string, err error, fresh func() tea.Cmd) tea.Cmd {
	shown := homeShort(path)
	c := newConfirm(Ask{Title: "Move aside", Verb: "move",
		Details:  []string{shown + " can't be read:", err.Error()},
		Question: "Move it to " + filepath.Base(asideName(path)) + " and start a new one?"}, nil)
	c.accept = func() tea.Cmd {
		to := asideName(path) // the next free one, if one was taken meanwhile
		if rerr := os.Rename(path, to); rerr != nil {
			return tea.Batch(a.closeTop(), a.MoveAside(path, err, fresh), a.ErrorAbout("Move aside failed", filepath.Base(path), rerr.Error()))
		}
		a.Log(Info, "moved "+shown+" to "+homeShort(to))
		return tea.Batch(a.closeTop(), run(fresh))
	}
	c.quit = func() tea.Cmd {
		a.exitErr = fmt.Errorf("%s can't be read: %v", shown, err)
		return tea.Quit
	}
	return a.open(c)
}

// asideName is where path goes aside: path.bak, or the first of .bak.1,
// .bak.2… that is free.
func asideName(path string) string {
	bak := path + ".bak"
	for n := 1; ; n++ {
		if _, err := os.Lstat(bak); os.IsNotExist(err) {
			return bak
		}
		bak = path + ".bak." + strconv.Itoa(n)
	}
}
