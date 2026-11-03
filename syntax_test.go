package terminu

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
)

// The colours of catppuccin-mocha that a syntax colour can be and the
// family colours leave out.
var (
	mauve     = Hex("#cba6f7")
	rosewater = Hex("#f5e0dc")
	sky       = Hex("#89dceb")
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
	{"Makefile: the lines between define and endef", "Makefile",
		[]string{`X = echo "hi"`, "define HELP", "Usage: make build", "builds the app", `echo "hi"`, "endef", "all: x"},
		[]gapPart{{1, "echo", sky}, {1, `"hi"`, Green}, {2, "define HELP", Overlay0}, {3, "Usage: make build", Text},
			{4, "builds the app", Text}, {5, "echo", sky}, {5, `"hi"`, Green}, {6, "endef", Overlay0}, {7, "all", Blue}}},
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
	{"Markdown: bold or italic at the start of a line", "README.md",
		// chroma finds them after a space, a line break among them, so only
		// where a rule took the line break before: the top of the file, after
		// a heading, a list item or a quote, and in a list item's text.
		[]string{"**Why**: because", "# Head", "*Note* this", "- item", "__Bold__ too", "> quote", "_It_ too",
			"* item", "- **Lead** item", "Mid **bold** and *em*"},
		[]gapPart{{1, "**Why**", Text}, {8, "*", mauve}, {9, "-", mauve}}},
}

// stylePart is a part of a line and whether it is bold and italic: the
// first time text comes in line n (from 1).
type stylePart struct {
	n            int
	text         string
	bold, italic bool
}

// syntaxGapStyles are the bold and italic a row of syntaxGaps asks for,
// where its colour is the same either way.
var syntaxGapStyles = map[string][]stylePart{
	"Markdown: bold or italic at the start of a line": {
		{1, "**Why**", true, false}, {1, "because", false, false}, {3, "*Note*", false, true}, {5, "__Bold__", true, false},
		{7, "_It_", false, true}, {8, "item", false, false}, {9, "**Lead**", true, false}, {9, "item", false, false},
		{10, "**bold**", true, false}, {10, "*em*", false, true}, {10, "Mid", false, false},
	},
}

// syntaxStylesIn is what is wrong with the bold and italic of parts in
// lines coloured: nothing when each is as it should be.
func syntaxStylesIn(coloured []string, parts []stylePart) []string {
	var wrong []string
	for _, p := range parts {
		var rs []rune
		var sts []covCStyle
		if p.n-1 < len(coloured) {
			rs, sts = styleRunes(coloured[p.n-1])
		}
		at := strings.Index(string(rs), p.text)
		if at < 0 {
			wrong = append(wrong, fmt.Sprintf("line %d has no %q", p.n, p.text))
			continue
		}
		at = utf8.RuneCountInString(string(rs)[:at])
		for k, r := range []rune(p.text) {
			if st := sts[at+k]; r != ' ' && (st.Bold != p.bold || st.Italic != p.italic) {
				wrong = append(wrong, fmt.Sprintf("%q in line %d is bold %v, italic %v; want %v, %v", p.text, p.n, st.Bold, st.Italic, p.bold, p.italic))
				break
			}
		}
	}
	return wrong
}

// styleRunes is the characters of a styled line, a tab among them, and the
// style each is drawn in.
func styleRunes(line string) ([]rune, []covCStyle) {
	var st covCStyle
	var rs []rune
	var sts []covCStyle
	for i := 0; i < len(line); {
		if m := sgrAt.FindString(line[i:]); m != "" {
			covCApply(&st, m[2:len(m)-1])
			i += len(m)
			continue
		}
		r, n := utf8.DecodeRuneInString(line[i:])
		rs, sts = append(rs, r), append(sts, st)
		i += n
	}
	return rs, sts
}

// fgRunes is the characters of a styled line, a tab among them, and the
// foreground each is drawn in.
func fgRunes(line string) ([]rune, []Color) {
	rs, sts := styleRunes(line)
	fgs := make([]Color, len(sts))
	for i, st := range sts {
		fgs[i] = st.Fg
	}
	return rs, fgs
}

// TestSyntaxStyles: syntax colours keep catppuccin-mocha's bold, italic
// and underline, never its background (file-picker-95): Markdown's
// **bold** is bold and *italic* italic, a # heading bold, a comment
// italic, an operator bold; a diff's added and removed lines have no
// background behind them. The preview draws the same.
func TestSyntaxStyles(t *testing.T) {
	type want struct {
		n                       int
		text                    string
		bold, italic, underline bool
	}
	for _, g := range []struct {
		file  string
		lines []string
		parts []want
	}{
		{"README.md", []string{"Some **bold** and *em* text", "# Title"},
			[]want{{1, "Some", false, false, false}, {1, "**bold**", true, false, false}, {1, "*em*", false, true, false}, {2, "# Title", true, false, false}}},
		{"main.go", []string{"// a note", "ok := a && b"},
			[]want{{1, "// a note", false, true, false}, {2, "&&", true, false, false}, {2, "ok", false, false, false}}},
		{"change.diff", []string{"--- a/x", "+++ b/x", "@@ -1 +1 @@", "-old", "+new", " same"},
			[]want{{4, "-old", false, false, false}, {5, "+new", false, false, false}, {6, " same", false, false, false}}},
		{"Makefile", []string{"include other.mk"}, []want{{1, "include other.mk", false, true, false}}},
	} {
		for _, src := range []struct {
			lines []string
			skip  int // the preview's line number and its space
		}{{Syntax(g.file, g.lines), 0}, {highlight(g.file, strings.Join(g.lines, "\n"), 20, 4, 1), 2}} {
			for n, l := range src.lines {
				rs, sts := styleRunes(l)
				rs, sts = rs[src.skip:], sts[src.skip:]
				for i, st := range sts {
					if st.Bg.Set {
						t.Errorf("%s, line %d: %q on a background %v", g.file, n+1, string(rs[i]), st.Bg)
						break
					}
				}
				for _, p := range g.parts {
					if p.n != n+1 {
						continue
					}
					at := strings.Index(string(rs), p.text)
					if at < 0 {
						t.Errorf("%s, line %d has no %q: %q", g.file, n+1, p.text, string(rs))
						continue
					}
					at = utf8.RuneCountInString(string(rs)[:at])
					for k, r := range []rune(p.text) {
						st := sts[at+k]
						if r != ' ' && (st.Bold != p.bold || st.Italic != p.italic || st.Underline != p.underline) {
							t.Errorf("%s, line %d: %q is bold %v, italic %v, underline %v; want %v, %v, %v",
								g.file, n+1, p.text, st.Bold, st.Italic, st.Underline, p.bold, p.italic, p.underline)
							break
						}
					}
				}
			}
		}
	}
	// No lexer the lib has gives a GenericUnderline: one that does.
	under := chroma.MustNewLexer(&chroma.Config{Name: "underlined"}, func() chroma.Rules {
		return chroma.Rules{"root": {{Pattern: `[^\n]+`, Type: chroma.GenericUnderline}, {Pattern: `\n`, Type: chroma.Text}}}
	})
	if _, sts := styleRunes(colourise(under, []string{"x"})[0]); len(sts) != 1 || !sts[0].Underline || sts[0].Bg.Set {
		t.Errorf("GenericUnderline: %+v, not underlined without a background", sts)
	}
	// Each diff line in its colour, as catppuccin-mocha has it.
	for i, c := range map[int]Color{3: Red, 4: Green} {
		if _, fgs := fgRunes(Syntax("change.diff", []string{"--- a/x", "+++ b/x", "@@ -1 +1 @@", "-old", "+new"})[i]); fgs[0] != c {
			t.Errorf("diff line %d is %v, not %v", i+1, fgs[0], c)
		}
	}
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
		coloured := Syntax(g.file, g.lines)
		for _, w := range append(syntaxGapsIn(coloured, g.lines, g.parts), syntaxStylesIn(coloured, syntaxGapStyles[g.row])...) {
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
		coloured := colourise(theirs, g.lines)
		if len(syntaxGapsIn(coloured, g.lines, g.parts))+len(syntaxStylesIn(coloured, syntaxGapStyles[g.row])) == 0 {
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

// TestSyntaxCR: a \r inside a line (a progress bar's 10%\r100%) stays in
// that line as it is, for the caller to draw, and the lines after it keep
// their places and colours; chroma reads a lone \r as a line break unless
// told not to.
func TestSyntaxCR(t *testing.T) {
	for _, g := range []struct {
		file  string
		lines []string
		parts []gapPart
	}{
		{"a.txt", []string{"a\tb", "10%\r100%", "z"}, nil},
		{"main.go", []string{"// 10%\r100%", "func main() {}", "x"}, []gapPart{{2, "main", Blue}}},
		{"pod.yaml", []string{"a: 10%\r100%", "kind: Pod", "x: 1"}, []gapPart{{2, "kind", mauve}, {3, "1", Peach}}},
		{"Cargo.toml", []string{`a = "10%` + "\r" + `100%"`, "[project]", "name = 1"}, []gapPart{{2, "project", mauve}, {3, "name", Blue}}},
		{"README.md", []string{"10%\r100%", "# Title", "+ item"}, []gapPart{{2, "# Title", Peach}, {3, "+", mauve}}},
		{"fence.md", []string{"```go", "x := 1 // 10%\r100%", "```", "# Title"}, []gapPart{{4, "# Title", Peach}}},
		{"Dockerfile", []string{"RUN echo 10%\r100%", "FROM alpine AS run"}, []gapPart{{2, "AS", mauve}}},
		// A line that has the stand-ins too keeps them, and its \r.
		{"b.txt", []string{"x\r\U00000085y", "z"}, nil},
		{"c.txt", []string{"x\r\U00000085\U00002028\U00002029y", "z"}, nil},
		{"crlf.yaml", []string{"kind: Pod\r", "x: 1\r"}, []gapPart{{1, "kind", mauve}, {2, "1", Peach}}},
	} {
		got := Syntax(g.file, g.lines)
		if len(got) != len(g.lines) {
			t.Errorf("%s: %d lines out of %d", g.file, len(got), len(g.lines))
			continue
		}
		for i, l := range got {
			if StripStyles(l) != g.lines[i] {
				t.Errorf("%s, line %d: %q, want %q", g.file, i+1, StripStyles(l), g.lines[i])
			}
		}
		for _, w := range syntaxGapsIn(got, g.lines, g.parts) {
			t.Errorf("%s: %s", g.file, w)
		}
	}
}

// TestViewerSelectionPlain: in a viewer's selection the syntax colours,
// bold, italic and underline are not shown: Base on Yellow (note-17); out
// of it they are.
func TestViewerSelectionPlain(t *testing.T) {
	line := "Some **bold** and *em* text"
	ta := newTestApp(t)
	ta.Viewer(&Viewer{Title: "README", Lines: []string{line, "# Title"}, Lang: "README.md", Select: &Selection{Key: "v", Name: "Visual"}})
	ta.settle()
	bold := func() bool {
		for _, c := range covCCells(viewerRows(ta)[0]) {
			if c.St.Bold || c.St.Italic {
				return true
			}
		}
		return false
	}
	if !bold() {
		t.Fatal("no bold or italic before the selection")
	}
	ta.press("v", "v", "$")
	cells := covCCells(viewerRows(ta)[0])
	for _, c := range cells {
		if c.Col < 6 || c.Col >= 6+len(line)-1 { // the last one is under the cursor
			continue
		}
		if c.St != (covCStyle{Fg: Base, Bg: Yellow}) {
			t.Errorf("cell %d %q of the selection: %+v, not Base on Yellow alone", c.Col, c.Ch, c.St)
			break
		}
	}
	if _, sts := styleRunes(viewerRows(ta)[1]); !slices.ContainsFunc(sts, func(st covCStyle) bool { return st.Bold }) {
		t.Error("the heading out of the selection is not bold")
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

// TestSyntaxPreviewViewerSame: a file has the same colours, bold, italic
// and underline in a viewer and in the file preview, the gaps chroma leaves
// filled in both (note-87).
func TestSyntaxPreviewViewerSame(t *testing.T) {
	gaps := append(slices.Clone(syntaxGaps), struct {
		row, file string
		lines     []string
		parts     []gapPart
	}{"bold, italic", "README.md", []string{"Some **bold** and *em* text", "# Title", "<!-- a note -->"}, nil})
	for _, g := range gaps {
		preview := highlight(g.file, strings.Join(g.lines, "\n"), 20, 4, 1)
		ta := newTestApp(t)
		ta.Viewer(&Viewer{Title: "File", Lines: g.lines, Lang: g.file})
		ta.settle()
		rows := viewerRows(ta)
		for i := range g.lines {
			var got, want []covCStyle
			for _, c := range covCCells(preview[i])[len(fmt.Sprint(len(g.lines)))+1:] { // after the number and its space
				want = append(want, c.St)
			}
			for _, c := range covCCells(rows[i]) {
				if c.Col >= 6 && len(got) < len(want) {
					got = append(got, c.St)
				}
			}
			if fmt.Sprint(got) != fmt.Sprint(want) {
				t.Errorf("%s (%s), line %d: %v, the preview's %v", g.row, g.file, i+1, got, want)
			}
		}
	}
}
