package terminu

import (
	"embed"
	"io/fs"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
)

// lexerFiles are chroma's lexers for the formats where chroma leaves a gap
// in the syntax colours, with the gaps filled the family's one way (tdp
// file-picker.3, "Where chroma leaves a gap").
//
//go:embed lexers/*.xml
var lexerFiles embed.FS

// chromaLexers are chroma's own lexers that lexerFiles replace, by the name
// of the lexer replacing them. A test checks that each still leaves its
// gaps: once chroma fills them itself, the file in lexers goes.
var chromaLexers = map[string]chroma.Lexer{}

// The lexers go into chroma's global registry under the names of chroma's
// own, which they replace: lexers.Get, Match and MatchMimeType answer with
// them, for the lib and for an app that calls chroma itself.
func init() {
	paths, err := fs.Glob(lexerFiles, "lexers/*.xml")
	if err != nil {
		panic(err)
	}
	for _, p := range paths {
		l := chroma.MustNewXMLLexer(lexerFiles, p)
		c := l.Config()
		was := lexers.Get(c.Name)
		if was == nil && len(c.Filenames) > 0 {
			was = lexers.Match(c.Filenames[0]) // a format chroma reads with another lexer
		}
		chromaLexers[c.Name] = was
		lexers.Register(l)
	}
}

// Syntax is lines in syntax colours, as the lib colours a text file in its
// file preview and in a viewer (tdp file-picker.3, note.2): chroma's
// catppuccin-mocha, foreground only, with the gaps chroma leaves filled the
// family's one way. An app that draws a file in a body of its own colours
// it with this, so that the file looks the same there as in the preview.
//
// lang is what Viewer.Lang takes: a lexer's name or alias ("toml"), a file
// name or path ("Cargo.toml"), or a content type ("application/json"). A
// language it cannot tell leaves the lines plain, in Text. The lines are
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
