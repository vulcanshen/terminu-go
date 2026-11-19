package main

import (
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	terminu "github.com/vulcanshen/terminu-go"
)

// relPath is a path of the project as the showcase names it: from the
// project's folder.
func relPath(p string) string {
	if rel, err := filepath.Rel(projectDir, p); err == nil {
		return rel
	}
	return p
}

// isText reports whether s reads as text: no NUL, valid UTF-8.
func isText(s string) bool { return utf8.ValidString(s) && !strings.Contains(s, "\x00") }

// textFile reports whether the file at p is text, by its first 8 KB.
func textFile(p string) bool {
	f, err := os.Open(p)
	if err != nil {
		return false
	}
	defer f.Close()
	b := make([]byte, 8<<10)
	n, _ := io.ReadFull(f, b)
	st, err := f.Stat()
	return err == nil && !st.IsDir() && isText(string(b[:n]))
}

// readOr is the file at p, "" when it is not there.
func readOr(p string) string {
	b, _ := os.ReadFile(p)
	return string(b)
}

// dirEntries are the folder's entries by name, none when it is not there
// (a folder only one side has).
func dirEntries(dir string) (map[string]bool, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, e := range entries {
		out[e.Name()] = e.IsDir()
	}
	return out, nil
}

// dirDiff are the items that differ between the folders a and b, one level,
// by name (tdp panel.9): a file only b has added, one only a has taken out,
// one both have with other bytes changed, compared when both sides are
// text; a folder that differs inside holds many, its items the next level.
// What is the same is left out. Each side is named from the project's
// folder, so that it says where.
func dirDiff(a, b string) ([]terminu.DiffItem, error) {
	ea, err := dirEntries(a)
	if err != nil {
		return nil, err
	}
	eb, err := dirEntries(b)
	if err != nil {
		return nil, err
	}
	var names []string
	for n := range ea {
		names = append(names, n)
	}
	for n := range eb {
		if _, ok := ea[n]; !ok {
			names = append(names, n)
		}
	}
	slices.Sort(names)
	var out []terminu.DiffItem
	for _, n := range names {
		pa, pb := filepath.Join(a, n), filepath.Join(b, n)
		dirA, inA := ea[n]
		dirB, inB := eb[n]
		it := terminu.DiffItem{Mark: terminu.DiffChanged, Name: n, Anchor: relPath(pa), Other: relPath(pb)}
		switch {
		case !inA:
			it.Mark = terminu.DiffAdded
		case !inB:
			it.Mark = terminu.DiffRemoved
		}
		if dirA || dirB {
			if inA && inB {
				if in, err := dirDiff(pa, pb); err == nil && len(in) == 0 {
					continue // the same inside
				}
			}
			it.Name += "/"
			it.Items = func() ([]terminu.DiffItem, error) { return dirDiff(pa, pb) }
			out = append(out, it)
			continue
		}
		ta, tb := readOr(pa), readOr(pb)
		if inA && inB && ta == tb {
			continue
		}
		if isText(ta) && isText(tb) {
			it.Text = func() (string, string, error) { return readOr(pa), readOr(pb), nil }
		}
		out = append(out, it)
	}
	return out, nil
}

// The pretend history of the project: 5d6e7f8 its last commit, 1a2b3c4 the
// one before it; deepPath a file it added, its path longer than a row of the
// comparison on the narrowest terminal.
const (
	lastCommit   = "5d6e7f8"
	parentCommit = "1a2b3c4"
	deepPath     = "installation/docker-compose/production/env.example"
)

// newFolder is new since the last commit: not in it yet.
const newFolder = "releases/v2"

// isNew reports whether rel is newFolder or in it.
func isNew(rel string) bool { return rel == newFolder || strings.HasPrefix(rel, newFolder+"/") }

// committed is the file at rel as its last commit has it: README.md and
// main.go are being edited, what is in newFolder is not there (""), the
// others are as committed.
func committed(rel, now string) string {
	switch {
	case rel == "README.md":
		return strings.Replace(now, "- Terminal\n", "", 1)
	case rel == "main.go":
		return strings.Replace(now, `, 42`, "", 1)
	case isNew(rel):
		return ""
	}
	return now
}

// sinceCommit are the items that differ in the folder at dir since the
// last commit, by name (tdp note.8): what is in newFolder added, a file
// being edited changed, a folder holding either holding many; the rest left
// out.
func sinceCommit(dir string) ([]terminu.DiffItem, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []terminu.DiffItem
	for _, e := range entries {
		p := filepath.Join(dir, e.Name())
		rel := relPath(p)
		it := terminu.DiffItem{Mark: terminu.DiffChanged, Name: e.Name(), Anchor: rel + " @ " + lastCommit, Other: rel}
		if isNew(rel) {
			it.Mark = terminu.DiffAdded
		}
		if e.IsDir() {
			if in, err := sinceCommit(p); err != nil || len(in) == 0 {
				continue
			}
			it.Name += "/"
			it.Items = func() ([]terminu.DiffItem, error) { return sinceCommit(p) }
			out = append(out, it)
			continue
		}
		now := readOr(p)
		if committed(rel, now) == now && !isNew(rel) {
			continue
		}
		if isText(now) {
			it.Text = func() (string, string, error) { now := readOr(p); return committed(rel, now), now, nil }
		}
		out = append(out, it)
	}
	return out, nil
}

// lastCommitItems are what the last commit changed, file by file, as git
// names them (tdp note.8): a rename both sides have, a submodule that holds
// many, an image that is not text, and a path long enough that the
// comparison's first row folds after a / on a narrow terminal.
func lastCommitItems() ([]terminu.DiffItem, error) {
	side := func(path, commit string) string { return path + " @ " + commit }
	text := func(a, b string) func() (string, string, error) {
		return func() (string, string, error) { return a, b, nil }
	}
	item := func(mark terminu.DiffMark, name, was, now string) terminu.DiffItem {
		return terminu.DiffItem{Mark: mark, Name: name, Anchor: side(was, parentCommit), Other: side(now, lastCommit)}
	}
	main := item(terminu.DiffChanged, "main.go", "main.go", "main.go")
	main.Text = text("package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"showcase\")\n}\n",
		"package main\n\nimport \"fmt\"\n\n// main says hello.\nfunc main() {\n\tfmt.Println(\"showcase\")\n}\n")
	notes := item(terminu.DiffChanged, "docs/NOTES.md → docs/notes.md", "docs/NOTES.md", "docs/notes.md")
	notes.Text = text("Notes on the README:\n- keep it short\n", "Notes on the README:\n- keep it short\n- link the CHANGELOG\n")
	build := item(terminu.DiffAdded, "config/build.toml", "config/build.toml", "config/build.toml")
	build.Text = text("", "# how the showcase builds\n[project]\nname = \"showcase\"\n")
	todo := item(terminu.DiffRemoved, "TODO.md", "TODO.md", "TODO.md")
	todo.Text = text("- write the README\n", "")
	logo := item(terminu.DiffChanged, "assets/logo.png", "assets/logo.png", "assets/logo.png")
	theme := item(terminu.DiffChanged, "themes/", "themes", "themes")
	theme.Items = func() ([]terminu.DiffItem, error) {
		mocha := item(terminu.DiffChanged, "mocha.toml", "themes/mocha.toml", "themes/mocha.toml")
		mocha.Text = text("base = \"#1e1e2e\"\ntext = \"#cdd6f4\"\n", "base = \"#1e1e2e\"\ntext = \"#cdd6f4\"\naccent = \"#89b4fa\"\n")
		latte := item(terminu.DiffAdded, "latte.toml", "themes/latte.toml", "themes/latte.toml")
		latte.Text = text("", "base = \"#eff1f5\"\n")
		return []terminu.DiffItem{latte, mocha}, nil
	}
	env := item(terminu.DiffAdded, deepPath, deepPath, deepPath)
	env.Text = text("", "POSTGRES_PASSWORD=not-a-real-password\n")
	return []terminu.DiffItem{logo, build, notes, main, theme, todo, env}, nil
}
