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
//     goes on as on its first start (its defaults for settings, empty state
//     or data): a file its first start makes is made again, one it does not
//     waits for the first save. The lib makes nothing in its place.
//   - Esc (quit) leaves the app: Run returns the path and the reason, for
//     main to write to stderr as it does any error of Run.
//
// A cache that cannot be read is not asked about: throw it away and build it
// again. Something outside the app that cannot be reached (a cluster, the
// network) is not the app's own: the app goes on and says so where it shows
// it (layout/panel, loading and errors). Data kept one record per file uses
// MoveAsideRecord.
func (a *App) MoveAside(path string, err error, fresh func() tea.Cmd) tea.Cmd {
	return a.moveAside(path, err, fresh, "start fresh")
}

// MoveAsideRecord is MoveAside for data kept one record per file (a mark per
// file): Enter only renames the record that cannot be read to <name>.bak,
// making nothing new in its place, and runs reread, where the app reads the
// rest as usual (Rules E3, when it cannot be read). The question is "Move it
// to <name>.bak and go on without it?"; Esc leaves the app as MoveAside's.
func (a *App) MoveAsideRecord(path string, err error, reread func() tea.Cmd) tea.Cmd {
	return a.moveAside(path, err, reread, "go on without it")
}

// moveAside asks about path; then is what the question says comes after the
// move.
func (a *App) moveAside(path string, err error, fresh func() tea.Cmd, then string) tea.Cmd {
	shown := homeShort(path)
	c := newConfirm(Ask{Title: "Move aside", Verb: "move",
		Details:  []string{shown + " can't be read:", err.Error()},
		Question: "Move it to " + filepath.Base(asideName(path)) + " and " + then + "?"}, nil)
	c.accept = func() tea.Cmd {
		to := asideName(path) // the next free one, if one was taken meanwhile
		if rerr := os.Rename(path, to); rerr != nil {
			return tea.Batch(a.closeTop(), a.moveAside(path, err, fresh, then), a.ErrorAbout("Move aside failed", filepath.Base(path), rerr.Error()))
		}
		a.Log(Info, "Moved "+shown+" to "+homeShort(to))
		return tea.Batch(a.closeTop(), run(fresh))
	}
	c.quit = func() tea.Cmd {
		a.exitErr = fmt.Errorf("%s can't be read: %v", shown, err)
		return tea.Quit
	}
	return a.open(c)
}

// NoLongerUsed notes each file of the app's own that a newer version no
// longer uses and that is still there (settings, state or data): an INFO in
// the App Log naming its path, "No longer used: <path>; delete it if you
// don't need it". The file is not deleted and there is no toast: what to do
// with it is the user's. Call it at every start; the App Log starts empty
// (Rules E3, files no longer used).
func (a *App) NoLongerUsed(paths ...string) {
	for _, p := range paths {
		if _, err := os.Lstat(p); err == nil {
			a.Log(Info, "No longer used: "+homeShort(p)+"; delete it if you don't need it")
		}
	}
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
