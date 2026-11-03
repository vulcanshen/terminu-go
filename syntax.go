package terminu

import (
	"embed"
	"io/fs"
	"path"
	"path/filepath"
	"slices"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
)

// lexerFiles are chroma's lexers for the formats where chroma leaves a gap
// in the syntax colours, with the gaps filled the family's one way, and
// lexers for the formats the family shows that chroma has none for (tdp
// file-picker.3, "Where chroma leaves a gap").
//
//go:embed lexers/*.xml
var lexerFiles embed.FS

// chromaOwn are chroma's own lexers, as its registry had them before the
// lib's went in. A test reads each file of tdp file-picker.3's table with
// the one chroma would pick and checks it still leaves the gap: once chroma
// fills one itself, the lib's fix for it goes.
var chromaOwn []chroma.Lexer

// The lexers go into chroma's global registry under the names of chroma's
// own, which they replace: lexers.Get, Match and MatchMimeType answer with
// them, for the lib and for an app that calls chroma itself.
func init() {
	chromaOwn = slices.Clone(lexers.GlobalLexerRegistry.Lexers)
	paths, err := fs.Glob(lexerFiles, "lexers/*.xml")
	if err != nil {
		panic(err)
	}
	for _, p := range paths {
		lexers.Register(chroma.MustNewXMLLexer(lexerFiles, p))
	}
	lexers.Register(goLexer())
}

// goLexer is chroma's Go lexer, written in Go there, with the gaps filled
// (tdp file-picker.3): the name a type declaration gives, Point in type
// Point struct and each in a type ( … ) block, is NameClass, as Rust's
// struct; the package's name is NameNamespace, as proto's package.
func goLexer() chroma.Lexer {
	theirs := lexers.Get("Go").(*chroma.RegexLexer)
	rules := chroma.Rules{}
	for state, rs := range theirs.MustRules() {
		rules[state] = slices.Clone(rs)
	}
	const ident = `[\p{L}_][\p{L}\p{N}_]*`
	rules["root"] = append([]chroma.Rule{
		{Pattern: `(package)(\s+)(` + ident + `)`, Type: chroma.ByGroups(chroma.KeywordNamespace, chroma.TextWhitespace, chroma.NameNamespace)},
		{Pattern: `(type)(\s+)(` + ident + `)`, Type: chroma.ByGroups(chroma.KeywordDeclaration, chroma.TextWhitespace, chroma.NameClass)},
		{Pattern: `(type)(\s*)(\()`, Type: chroma.ByGroups(chroma.KeywordDeclaration, chroma.TextWhitespace, chroma.Punctuation), Mutator: chroma.Push("types")},
	}, rules["root"]...)
	rules["types"] = []chroma.Rule{ // a type ( … ) block: each line's first name
		{Pattern: `\)`, Type: chroma.Punctuation, Mutator: chroma.Pop(1)},
		{Pattern: `^([^\S\n]*)(` + ident + `)`, Type: chroma.ByGroups(chroma.TextWhitespace, chroma.NameClass)},
		{Pattern: `[({]`, Type: chroma.Punctuation, Mutator: chroma.Push("nested")},
		chroma.Include("root"),
	}
	rules["nested"] = []chroma.Rule{ // brackets inside a type block, read as Go
		{Pattern: `[)}]`, Type: chroma.Punctuation, Mutator: chroma.Pop(1)},
		{Pattern: `[({]`, Type: chroma.Punctuation, Mutator: chroma.Push("nested")},
		chroma.Include("root"),
	}
	config := *theirs.Config()
	return chroma.MustNewLexer(&config, func() chroma.Rules { return rules })
}

// byDirectory are the file names more than one format has: the directory the
// file is in tells which (tdp file-picker.3). A hosts is the hosts file only
// in /etc: elsewhere (an Ansible inventory) it is another format.
var byDirectory = []struct{ dir, name, lexer string }{
	{".ssh", "config", "SSH config"},
	{".git", "config", "Git Config"},
	{"etc", "hosts", "hosts file"},
}

// lexerOfPath is the lexer the directory of the file at p picks, for a name
// more than one format has; nil when the name alone tells.
func lexerOfPath(p string) chroma.Lexer {
	dir, name := path.Split(filepath.ToSlash(p))
	dir = path.Base(path.Clean("/" + dir))
	for _, b := range byDirectory {
		if b.dir == dir && b.name == name {
			return lexers.Get(b.lexer)
		}
	}
	return nil
}

// Syntax is lines in syntax colours, as the lib colours a text file in its
// file preview and in a viewer (tdp file-picker.3, note.2): chroma's
// catppuccin-mocha, its colours, bold, italic and underline but never its
// background, with the gaps chroma leaves filled the family's one way. An app that draws a file in a body of its own colours
// it with this, so that the file looks the same there as in the preview.
//
// lang is what Viewer.Lang takes: a lexer's name or alias ("toml"), a file
// name or path ("Cargo.toml"), or a content type ("application/json"). A
// path also tells a name more than one format has by its directory: a
// config in .ssh is SSH's, one in .git is git's, a hosts in /etc is the hosts
// file and one elsewhere is not. A language it cannot tell
// leaves the lines plain, in Text. The lines are
// plain text, read together (a string may go on to the next line), and each
// gives one line back; a tab or a \r stays as it is, for the caller to draw.
//
// Importing the lib puts its lexers in chroma's global registry in place of
// chroma's own, so an app that calls chroma itself gets the gaps filled too.
func Syntax(lang string, lines []string) []string {
	lexer := lexerFor(lang)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	return colourise(lexer, lines)
}
