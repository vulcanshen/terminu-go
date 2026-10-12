package terminu

import (
	"fmt"
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
	glyphFolder = "\U000F024B" // nf-md-folder
	glyphFile   = "\U000F0214" // nf-md-file
)

// FilePicker picks a file (components/input/file-picker): a finder over a
// directory, Enter going into a directory or choosing a file.
type FilePicker struct {
	// Title is the field's name: the title writes the directory after it
	// ("IDENTITY FILE · ~/.ssh").
	Title, Glyph string
	// Root is the directory the list opens in: a start, not a boundary.
	Root string
	// Value is the file now, if any: the cursor starts on it, in Green.
	Value string
	// Icon is a row's icon and its colour, the app's choice (a file type's
	// colour); a folder or a file glyph in Text when unset.
	Icon func(path string, dir bool) (string, Color)
	// Done gets the chosen file. It is not called when it did not change.
	Done func(path string) tea.Cmd
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
	fv := &finderView{kind: findFiles, glyph: p.Glyph, name: p.Title, ed: newEditor(""), onList: true, written: written}
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
	fv.green = func(_ int, r finderRow) bool { return !r.dir && p.Value != "" && r.path == expandHome(p.Value) }
	fv.verb = func(r finderRow) string {
		if r.dir {
			return "into"
		}
		return "choose"
	}
	fv.enter = func(a *App, _ int, r finderRow) tea.Cmd {
		if r.dir {
			fp.dir = r.path
			fv.ed.set("")
			fv.onList = true
			fv.refilter()
			if len(fv.shown) > 1 && fp.rows[0].text == ".." {
				fv.cursor = 1 // the first entry under ..
			}
			return nil
		}
		if r.path == expandHome(p.Value) || p.Done == nil {
			return a.closeTop()
		}
		fv.wrote()
		return tea.Batch(a.closeTop(), p.Done(r.path))
	}
	fv.preview = func(r finderRow) (string, []string) { return previewFile(r) }
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
	key := dir + "\x00" + glob
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
		fp.rows = fp.readDir(fp.start)
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
// the files, each by name, hidden ones too. TODO(tdp G32): the order is the
// picture's.
func (fp *filePicker) readDir(dir string) []finderRow {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var rows []finderRow
	if dir != "/" {
		rows = append(rows, fp.row(filepath.Dir(dir), "..", true))
	}
	var files []finderRow
	for _, e := range entries {
		p := filepath.Join(dir, e.Name())
		isDir := e.IsDir()
		if e.Type()&fs.ModeSymlink != 0 {
			if st, err := os.Stat(p); err == nil {
				isDir = st.IsDir()
			}
		}
		if isDir {
			rows = append(rows, fp.row(p, e.Name(), true))
		} else {
			files = append(files, fp.row(p, e.Name(), false))
		}
	}
	return append(rows, files...)
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
				}
				return false
			}
			if len(n) == 0 {
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

// previewFile is what the preview shows (components/input/file-picker): a
// directory's entries, a text file's lines numbered in Overlay0, else its kind
// and size. TODO(tdp G31): a private key is a text file and shows whole.
func previewFile(r finderRow) (string, []string) {
	title := filepath.Base(r.path)
	if r.dir {
		entries, err := os.ReadDir(r.path)
		if err != nil {
			return title, []string{Fg(Overlay0).Render("(" + err.Error() + ")")}
		}
		var out []string
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() {
				name += "/"
			}
			out = append(out, Fg(Text).Render(name))
		}
		return title, out
	}
	st, err := os.Stat(r.path)
	if err != nil {
		return title, []string{Fg(Overlay0).Render("(" + err.Error() + ")")}
	}
	f, err := os.Open(r.path)
	if err != nil {
		return title, []string{Fg(Overlay0).Render("(" + err.Error() + ")")}
	}
	defer f.Close()
	buf := make([]byte, 16<<10)
	n, _ := f.Read(buf)
	buf = buf[:n]
	if n == 0 || !utf8.Valid(trimPartialRune(buf)) || slices.Contains(buf, 0) {
		kind := "file"
		if n > 0 {
			kind = "binary"
		}
		return title, []string{Fg(Overlay0).Render(fmt.Sprintf("(%s, %s)", kind, size(st.Size())))}
	}
	lines := strings.Split(strings.TrimRight(string(trimPartialRune(buf)), "\n"), "\n")
	numW := len(strconv.Itoa(len(lines)))
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = Fg(Overlay0).Render(PadLeft(strconv.Itoa(i+1), numW)) + " " + Fg(Text).Render(strings.ReplaceAll(l, "\t", "    "))
	}
	return title, out
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
	switch {
	case n < 1<<10:
		return strconv.FormatInt(n, 10) + " B"
	case n < 1<<20:
		return strconv.FormatInt(n>>10, 10) + " KB"
	}
	return strconv.FormatInt(n>>20, 10) + " MB"
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
