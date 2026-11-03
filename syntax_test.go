package terminu

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2/lexers"
)

// The colours of catppuccin-mocha that a syntax colour can be and the
// family colours leave out.
var (
	mauve     = Hex("#cba6f7")
	rosewater = Hex("#f5e0dc")
)

// gapPart is a part of a line and the colour it is drawn in: the first
// time text comes in line n (from 1).
type gapPart struct {
	n    int
	text string
	c    Color
}

// syntaxGaps are the rows of tdp file-picker.3's "Where chroma leaves a
// gap", each with a file of that format and the parts of it the row
// colours.
var syntaxGaps = []struct {
	row, file string
	lines     []string
	parts     []gapPart
}{
	{"TOML: a table's name", "Cargo.toml",
		[]string{"[project]", "[[bin]]", "[a.b]", `[dog."tater.man"]`},
		[]gapPart{{1, "project", mauve}, {2, "bin", mauve}, {3, "a", mauve}, {3, "b", mauve}, {4, "dog", mauve}, {4, `"tater.man"`, mauve}}},
	{"TOML: a key", "Cargo.toml",
		[]string{`name = "x"`, "a.b = 1", `owner = { first = "Tom", last = "X" }`, `"quoted key" = 2`, `1234 = "digits"`},
		[]gapPart{{1, "name", Blue}, {2, "a", Blue}, {2, "b", Blue}, {3, "owner", Blue}, {3, "first", Blue}, {3, "last", Blue}, {4, `"quoted key"`, Blue}, {5, "1234", Blue}}},
	{"TOML: a number in any form", "Cargo.toml",
		[]string{"h = 0xDEADBEEF", "o = 0o755", "b = 0b1010", "e = 5e+22", "i = inf", "n = -nan"},
		[]gapPart{{1, "0xDEADBEEF", Peach}, {2, "0o755", Peach}, {3, "0b1010", Peach}, {4, "5e+22", Peach}, {5, "inf", Peach}, {6, "-nan", Peach}}},
	{"TOML: a date or a time in any form", "Cargo.toml",
		[]string{"d = 1979-05-27", "t = 07:32:00", "dt = 1979-05-27T07:32:00Z"},
		[]gapPart{{1, "1979-05-27", Text}, {2, "07:32:00", Text}, {3, "1979-05-27T07:32:00Z", Text}}},
	{"YAML: a value without quotes", "pod.yaml",
		[]string{"kind: Pod", "args:", "- serve", "- --port=8080", `image: "nginx"`},
		[]gapPart{{1, "Pod", Green}, {3, "serve", Green}, {4, "--port=8080", Green}, {5, `"nginx"`, Green}}},
	{"YAML: a key after the first in a flow mapping", "pod.yaml",
		[]string{"labels: {app: nginx, tier: web}"},
		[]gapPart{{1, "app", mauve}, {1, "tier", mauve}, {1, "nginx", Green}, {1, "web", Green}}},
	{"YAML: a number with a sign or a leading dot", "pod.yaml",
		[]string{"neg: -3", "ratio: .5", "n: 3"},
		[]gapPart{{1, "-3", Peach}, {2, ".5", Peach}, {3, "3", Peach}}},
	{"INI (.gitconfig): an indented key after the first", ".gitconfig",
		[]string{"[user]", "\tname = Someone", "\temail = someone@example.com"},
		[]gapPart{{2, "name", Blue}, {3, "email", Blue}, {3, "someone@example.com", Green}}},
	{"Makefile: a directive", "Makefile",
		[]string{"include other.mk", "ifeq ($(OS),Windows_NT)", "X = 1", "endif"},
		[]gapPart{{1, "include", Overlay0}, {2, "ifeq", Overlay0}, {4, "endif", Overlay0}}},
	{"Makefile: a variable in a recipe", "Makefile",
		[]string{"BIN := bin/app", "build: $(BIN)", "\t$(GO) build -o $(BIN)", "\t@echo \"built $(BIN)\""},
		[]gapPart{{2, "BIN", rosewater}, {3, "GO", rosewater}, {3, "build -o", Text}, {4, "BIN", rosewater}, {4, `"built `, Green}}},
	{"Dockerfile: a LABEL's key", "Dockerfile",
		[]string{"ENV MODE=prod", `LABEL a.b="x"`},
		[]gapPart{{1, "MODE", rosewater}, {2, "a.b", rosewater}, {2, `"x"`, Green}}},
	{"Dockerfile: AS after an image with a variable", "Dockerfile",
		[]string{"FROM golang:${GO} AS build", "FROM alpine AS run"},
		[]gapPart{{1, "AS", mauve}, {1, "build", Green}, {2, "AS", mauve}, {2, "run", Green}}},
	{"Dockerfile: a version with a dot", "Dockerfile",
		[]string{"ARG GO=1.24", "ENV A=1 B=2.0.1"},
		[]gapPart{{1, "1.24", Peach}, {2, "2.0.1", Peach}}},
	{"Markdown: an underlined heading", "README.md",
		[]string{"Title", "=====", "", "Part", "----", "", "# Hash"},
		[]gapPart{{1, "Title", Peach}, {2, "=====", Peach}, {4, "Part", Peach}, {5, "----", Peach}, {7, "# Hash", Peach}}},
	{"Markdown: a list item's +", "README.md",
		[]string{"+ plus", "- minus", "* star"},
		[]gapPart{{1, "+", mauve}, {2, "-", mauve}, {3, "*", mauve}}},
	{"Markdown: a reference link", "README.md",
		[]string{"See [the docs][ref] or [here](https://x.org).", "", "[ref]: https://example.com"},
		[]gapPart{{1, "the docs", mauve}, {1, "ref", Blue}, {1, "here", mauve}, {1, "https://x.org", Blue}, {3, "ref", mauve}, {3, "https://example.com", Blue}}},
}

// fgRunes is the characters of a styled line, a tab among them, and the
// foreground each is drawn in.
func fgRunes(line string) ([]rune, []Color) {
	var st covCStyle
	var rs []rune
	var fgs []Color
	for i := 0; i < len(line); {
		if m := sgrAt.FindString(line[i:]); m != "" {
			covCApply(&st, m[2:len(m)-1])
			i += len(m)
			continue
		}
		r, n := utf8.DecodeRuneInString(line[i:])
		rs, fgs = append(rs, r), append(fgs, st.Fg)
		i += n
	}
	return rs, fgs
}

// syntaxGapsIn is what is wrong with the parts of a row in lines coloured:
// nothing when every part has its colour.
func syntaxGapsIn(coloured []string, lines []string, parts []gapPart) []string {
	var wrong []string
	for _, p := range parts {
		var rs []rune
		var fgs []Color
		if p.n-1 < len(coloured) {
			rs, fgs = fgRunes(coloured[p.n-1])
		}
		if string(rs) != lines[p.n-1] {
			wrong = append(wrong, fmt.Sprintf("line %d is %q", p.n, string(rs)))
			continue
		}
		at := strings.Index(lines[p.n-1], p.text)
		if at < 0 {
			wrong = append(wrong, fmt.Sprintf("line %d has no %q", p.n, p.text))
			continue
		}
		at = utf8.RuneCountInString(lines[p.n-1][:at])
		for i, r := range []rune(p.text) {
			if r != ' ' && fgs[at+i] != p.c {
				wrong = append(wrong, fmt.Sprintf("%q in line %d is %v, not %v", p.text, p.n, fgs[at+i], p.c))
				break
			}
		}
	}
	return wrong
}

// TestSyntaxGaps: the lib fills each gap chroma leaves as tdp file-picker.3
// lists it (file-picker-94), in Syntax, which a viewer and the preview
// colour with.
func TestSyntaxGaps(t *testing.T) {
	for _, g := range syntaxGaps {
		for _, w := range syntaxGapsIn(Syntax(g.file, g.lines), g.lines, g.parts) {
			t.Errorf("%s (%s): %s", g.row, g.file, w)
		}
	}
}

// TestSyntaxGapsInChroma: chroma's own lexer still leaves each gap the lib
// fills. When chroma fills one itself, this fails: the lib's fix in lexers
// has no more to do, and goes.
func TestSyntaxGapsInChroma(t *testing.T) {
	for _, g := range syntaxGaps {
		mine := lexerFor(g.file)
		theirs := chromaLexers[mine.Config().Name]
		if theirs == nil || theirs == mine {
			t.Errorf("%s: no lexer of chroma's own kept for %s", g.row, mine.Config().Name)
			continue
		}
		if len(syntaxGapsIn(colourise(theirs, g.lines), g.lines, g.parts)) == 0 {
			t.Errorf("%s: chroma's own %s lexer colours it now: the fix in lexers/ can go", g.row, theirs.Config().Name)
		}
	}
}

// TestSyntaxLexersReplaceChroma: the lib's lexers answer for their formats
// in chroma's own registry, for an app that calls chroma itself, and chroma
// keeps one lexer of each name.
func TestSyntaxLexersReplaceChroma(t *testing.T) {
	for name := range chromaLexers {
		l := lexers.Get(name)
		if l == nil || chromaLexers[name] == l {
			t.Errorf("%s: chroma still answers with its own", name)
			continue
		}
		n := 0
		for _, other := range lexers.GlobalLexerRegistry.Lexers {
			if other.Config().Name == name {
				n++
			}
		}
		if n != 1 {
			t.Errorf("%s: %d lexers of that name in chroma", name, n)
		}
	}
	for file, name := range map[string]string{"Cargo.toml": "TOML", "pod.yml": "YAML", ".gitconfig": "Git Config",
		"build.mk": "Makefile", "Dockerfile": "Docker", "README.md": "markdown", "setup.cfg": "INI"} {
		if l := lexers.Match(file); l == nil || l.Config().Name != name {
			t.Errorf("%s: read by %v, want %s", file, l, name)
		}
	}
	for _, mime := range []string{"text/x-toml", "text/x-yaml", "text/x-markdown"} {
		if l := lexers.MatchMimeType(mime); l == nil || l == chromaLexers[l.Config().Name] {
			t.Errorf("%s: chroma's own lexer", mime)
		}
	}
}

// TestSyntax: Syntax takes what Viewer.Lang takes; a language it cannot
// tell leaves the lines in Text; a line in gives a line out, a string
// going on to the next line coloured in both.
func TestSyntax(t *testing.T) {
	lines := []string{"[project]", "name = \"\"\"two", "lines\"\"\"", ""}
	want := Syntax("pyproject.toml", lines)
	if len(want) != len(lines) {
		t.Fatalf("%d lines out of %d", len(want), len(lines))
	}
	if w := syntaxGapsIn(want, lines, []gapPart{{1, "project", mauve}, {2, `"""two`, Green}, {3, `lines"""`, Green}}); len(w) > 0 {
		t.Errorf("%v", w)
	}
	for _, lang := range []string{"toml", "TOML", "/home/me/pyproject.toml", "text/x-toml; charset=utf-8"} {
		if got := Syntax(lang, lines); fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("%q: %q, not as pyproject.toml", lang, got)
		}
	}
	for _, lang := range []string{"", "notes.nothing-knows-this"} {
		for i, l := range Syntax(lang, []string{"name = 1", ""}) {
			if want := []string{Fg(Text).Render("name = 1"), ""}[i]; l != want {
				t.Errorf("%q, line %d: %q, want %q", lang, i+1, l, want)
			}
		}
	}
}

// TestSyntaxFrontMatter: a Markdown file's YAML front matter at the top is
// YAML, as in chroma; a --- further down is not.
func TestSyntaxFrontMatter(t *testing.T) {
	lines := []string{"---", "title: Notes", "---", "", "Text", "---", "", "kind: Pod"}
	if w := syntaxGapsIn(Syntax("notes.md", lines), lines, []gapPart{{2, "title", mauve}, {2, "Notes", Green},
		{5, "Text", Peach}, {8, "kind: Pod", Text}}); len(w) > 0 {
		t.Errorf("%v", w)
	}
}

// TestSyntaxPreviewViewerSame: a file has the same colours in a viewer
// and in the file preview, the gaps chroma leaves filled in both (note-87).
func TestSyntaxPreviewViewerSame(t *testing.T) {
	for _, g := range syntaxGaps {
		preview := highlight(g.file, strings.Join(g.lines, "\n"), 20, 4, 1)
		ta := newTestApp(t)
		ta.Viewer(&Viewer{Title: "File", Lines: g.lines, Lang: g.file})
		ta.settle()
		rows := viewerRows(ta)
		for i := range g.lines {
			var got, want []Color
			for _, c := range covCCells(preview[i])[2:] { // after the number and its space
				want = append(want, c.St.Fg)
			}
			for _, c := range covCCells(rows[i]) {
				if c.Col >= 6 && len(got) < len(want) {
					got = append(got, c.St.Fg)
				}
			}
			if fmt.Sprint(got) != fmt.Sprint(want) {
				t.Errorf("%s (%s), line %d: %v, the preview's %v", g.row, g.file, i+1, got, want)
			}
		}
	}
}
