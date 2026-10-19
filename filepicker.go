package terminu

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	glyphFolder     = "\U000F024B" // nf-md-folder
	glyphFile       = "\U000F0214" // nf-md-file
	glyphFolderMove = "\U000F0252" // nf-md-folder_move: ". (this directory)"
)

// FilePicker picks a file (components/input/file-picker): a finder over a
// directory, Enter going into a directory or choosing a file. With Dirs it
// picks a directory instead: only directories are listed, and Enter on the
// first row, ". (this directory)", chooses the one listed.
type FilePicker struct {
	// Title is the field's name: the title writes the directory after it
	// ("IDENTITY FILE · ~/.ssh").
	Title, Glyph string
	// Root is the directory the list opens in: a start, not a boundary.
	Root string
	// Value is the file now, if any: the cursor starts on it, in Green.
	Value string
	// Dirs picks a directory (components/input/file-picker, choosing a
	// directory). Open it at the directory now (Root = Value): the cursor
	// starts on ".", in Green.
	Dirs bool
	// Icon is a row's icon and its colour, the app's choice (a file type's
	// colour); a folder or a file glyph in Text when unset.
	Icon func(path string, dir bool) (string, Color)
	// Done gets the chosen file (or directory). It is not called when it did
	// not change.
	Done func(path string) tea.Cmd
	// Commit, when set, writes a changed value back when Enter takes it, on
	// the update loop, before Done: an error keeps the popup open with the
	// value in it and opens an error popup over it, "RENAME FAILED"
	// (components/layout/popup, the action failing; Rules F4).
	Commit func(path string) error
	// OnCancel runs when it closes with no answer: Esc, or the popup it is
	// over closing it, so whoever waits for the answer (a web page's prompt,
	// ssh's askpass) hears it was cancelled (Rules F6).
	OnCancel func() tea.Cmd
}

func (p FilePicker) typed() bool { return false }

// PickFile opens a file-picker.
func (a *App) PickFile(p FilePicker) tea.Cmd { return p.open(a, nil) }

// filePicker is the state a file-picker keeps beside its finder.
type filePicker struct {
	p     FilePicker
	dir   string // the directory listed when no path is typed
	key   string // what the cached rows were made for
	rows  []finderRow
	start string // the directory the title writes
}

func (p FilePicker) open(a *App, written func()) tea.Cmd {
	fp := &filePicker{p: p, dir: expandHome(cmpOr(p.Root, "."))}
	if abs, err := filepath.Abs(fp.dir); err == nil {
		fp.dir = abs
	}
	fv := &finderView{kind: findFiles, glyph: p.Glyph, name: p.Title, ed: newEditor(""), onList: true, written: written, onCancel: p.OnCancel}
	fv.rows = func() []finderRow { return fp.list(fv.ed.value()) }
	fv.words = func(typed string) string {
		switch {
		case !isPath(typed):
			return typed
		case strings.ContainsAny(typed, "*?"):
			return "" // the wildcard has done the filtering
		}
		return typed[strings.LastIndex(typed, "/")+1:]
	}
	fv.title = func(room int) string {
		name := Upper(p.Title)
		where := homeShort(fp.start)
		room -= Width(p.Glyph) + 1 + Width(name) + 3
		return name + " · " + TruncateLeft(where, max(room, 1))
	}
	value := ""
	if p.Value != "" {
		value = filepath.Clean(expandHome(p.Value))
	}
	// The value now is Green: a file, or with Dirs a directory (not ..).
	fv.green = func(_ int, r finderRow) bool {
		return value != "" && r.dir == p.Dirs && r.text != ".." && filepath.Clean(r.path) == value
	}
	chooses := func(r finderRow) bool { return !r.dir || p.Dirs && r.text == "." }
	fv.verb = func(r finderRow) string {
		if chooses(r) {
			return "choose"
		}
		return "into"
	}
	if p.Dirs {
		fv.enterRef = "go into a directory; on . choose the one listed"
	}
	fv.enter = func(a *App, _ int, r finderRow) tea.Cmd {
		if !chooses(r) {
			fp.dir = r.path
			fv.ed.set("")
			fv.onList = true
			fv.refilter()
			// The cursor rests on the first entry under . and .., or on the
			// first row in a directory with none.
			fv.cursor = 0
			for k, i := range fv.shown {
				if t := fp.rows[i].text; t != "." && t != ".." {
					fv.cursor = k
					break
				}
			}
			return nil
		}
		if filepath.Clean(r.path) == value || p.Done == nil && p.Commit == nil {
			return a.answer()
		}
		if err := runCommit(p.Commit, r.path); err != nil {
			return a.commitFailed(p.Title, "", err)
		}
		fv.wrote()
		return tea.Batch(a.answer(), runDone(p.Done, r.path))
	}
	fv.preview = func(r finderRow, w, h int) (string, []string) {
		return filepath.Base(r.path), filePreview(r.path, w, h, a.HideSecretsInPreview, p.Icon, a.inputTabWidth())
	}
	fv.info = func(r finderRow, w int, layer Color) []string { return fileInfo(r.path, w, layer) }
	fv.refilter()
	for k, i := range fv.shown {
		if fv.green(i, fp.rows[i]) {
			fv.cursor = k
		}
	}
	return a.open(fv)
}

// isPath reports typed text that is a path: it starts at / or ~/.
func isPath(s string) bool { return strings.HasPrefix(s, "/") || strings.HasPrefix(s, "~/") }

// list is the rows for what is typed: the directory listed, a typed path's
// directory, or what a wildcard finds.
func (fp *filePicker) list(typed string) []finderRow {
	dir, glob := fp.dir, ""
	if isPath(typed) {
		dir = typed[:strings.LastIndex(typed, "/")+1]
		if i := strings.IndexAny(typed, "*?"); i >= 0 {
			dir = typed[:strings.LastIndex(typed[:i], "/")+1]
			glob = typed[len(dir):]
		}
		dir = expandHome(dir)
	}
	// Hidden files show when the word typed starts with a dot (".ss" finds
	// .ssh), as a shell glob does.
	word := typed[strings.LastIndex(typed, "/")+1:]
	hidden := strings.HasPrefix(word, ".")
	key := dir + "\x00" + glob + "\x00" + strconv.FormatBool(hidden) + strconv.FormatBool(word != "")
	if key == fp.key {
		return fp.rows
	}
	fp.key, fp.start = key, filepath.Clean(dir)
	switch {
	case slices.Contains(strings.Split(dir, "/"), ".."):
		// A path with .. in it is not followed up: no such directory
		// (components/input/file-picker).
		fp.rows = nil
	case glob != "":
		fp.rows = fp.search(fp.start, glob)
	default:
		fp.rows = fp.readDir(fp.start, hidden)
	}
	if fp.p.Dirs {
		// Directories only; with a word or a wildcard typed there is no "."
		// row (components/input/file-picker).
		fp.rows = slices.DeleteFunc(fp.rows, func(r finderRow) bool { return !r.dir || word != "" && r.text == "." })
	}
	return fp.rows
}

func (fp *filePicker) row(p string, text string, dir bool) finderRow {
	r := finderRow{text: text, path: p, dir: dir}
	if fp.p.Icon != nil {
		r.icon, r.color = fp.p.Icon(p, dir)
	} else if dir {
		r.icon = glyphFolder
	} else {
		r.icon = glyphFile
	}
	return r
}

// readDir lists a directory: .. first (none at /), then the directories, then
// the files, each by name (components/input/file-picker). Choosing a
// directory, ". (this directory)" comes before them all.
func (fp *filePicker) readDir(dir string, hidden bool) []finderRow {
	entries, err := dirEntries(dir, hidden)
	if err != nil {
		return nil
	}
	var rows []finderRow
	if fp.p.Dirs {
		here := fp.row(dir, ".", true)
		here.icon, here.label = glyphFolderMove, ". (this directory)"
		rows = append(rows, here)
	}
	if dir != "/" {
		rows = append(rows, fp.row(filepath.Dir(dir), "..", true))
	}
	for _, e := range entries {
		rows = append(rows, fp.row(e.path, e.name, e.dir))
	}
	return rows
}

type dirEntry struct {
	name, path string
	dir        bool
}

// dirEntries is a directory's entries as lists show them: the directories,
// then the files, each by name ignoring case; hidden ones (a leading dot)
// only when asked for.
func dirEntries(dir string, hidden bool) ([]dirEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []dirEntry
	for _, e := range entries {
		if !hidden && strings.HasPrefix(e.Name(), ".") {
			continue
		}
		p := filepath.Join(dir, e.Name())
		isDir := e.IsDir()
		if e.Type()&fs.ModeSymlink != 0 {
			if st, err := os.Stat(p); err == nil {
				isDir = st.IsDir()
			}
		}
		out = append(out, dirEntry{e.Name(), p, isDir})
	}
	slices.SortStableFunc(out, func(a, b dirEntry) int {
		if a.dir != b.dir {
			if a.dir {
				return -1
			}
			return 1
		}
		return strings.Compare(strings.ToLower(a.name), strings.ToLower(b.name))
	})
	return out, nil
}

// searchLimit keeps a ** search from walking a whole disk.
const searchLimit = 20000

// search is a wildcard: * within one directory, ** any number of them; each
// row is its path from start.
func (fp *filePicker) search(start, glob string) []finderRow {
	var rows []finderRow
	seen := 0
	_ = filepath.WalkDir(start, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if seen++; seen > searchLimit {
			return filepath.SkipAll
		}
		rel, _ := filepath.Rel(start, p)
		if rel == "." {
			return nil
		}
		// A hidden directory is gone into only when the glob names it.
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") && !strings.Contains(glob, "/.") && !strings.HasPrefix(glob, ".") {
			return filepath.SkipDir
		}
		if !strings.Contains(glob, "**") && strings.Count(rel, "/") > strings.Count(glob, "/") {
			return filepath.SkipDir
		}
		if globMatch(glob, filepath.ToSlash(rel)) {
			rows = append(rows, fp.row(p, rel, d.IsDir()))
		}
		return nil
	})
	return rows
}

// globMatch is a shell glob over a slash path: ** stands for any number of
// directories.
func globMatch(glob, name string) bool {
	g, n := strings.Split(glob, "/"), strings.Split(name, "/")
	var match func(g, n []string) bool
	match = func(g, n []string) bool {
		for len(g) > 0 {
			if g[0] == "**" {
				for k := 0; k <= len(n); k++ {
					if match(g[1:], n[k:]) {
						return true
					}
					if k < len(n) && strings.HasPrefix(n[k], ".") {
						return false // ** does not go through hidden directories
					}
				}
				return false
			}
			if len(n) == 0 {
				return false
			}
			// As a shell does, * and ? do not match a leading dot.
			if strings.HasPrefix(n[0], ".") && !strings.HasPrefix(g[0], ".") {
				return false
			}
			if ok, _ := path.Match(g[0], n[0]); !ok {
				return false
			}
			g, n = g[1:], n[1:]
		}
		return len(n) == 0
	}
	return match(g, n)
}

// trimPartialRune drops a character cut in two at the end of what was read.
func trimPartialRune(b []byte) []byte {
	for k := 0; k < utf8.UTFMax && len(b) > 0; k++ {
		if utf8.Valid(b) {
			return b
		}
		b = b[:len(b)-1]
	}
	return b
}

func size(n int64) string {
	if n < 1<<10 {
		return strconv.FormatInt(n, 10) + " B"
	}
	f, units, i := float64(n)/1024, []string{"KB", "MB", "GB", "TB"}, 0
	for f >= 1024 && i < len(units)-1 {
		f /= 1024
		i++
	}
	return strconv.FormatFloat(f, 'f', 1, 64) + " " + units[i]
}

// expandHome turns a leading ~ into the home directory.
func expandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return home + p[1:]
		}
	}
	return p
}

// homeShort writes the home directory as ~ (components/input/README).
func homeShort(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if p == home {
		return "~"
	}
	if strings.HasPrefix(p, home+"/") {
		return "~" + p[len(home):]
	}
	return p
}
