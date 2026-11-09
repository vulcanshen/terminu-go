package main

import (
	"path/filepath"
	"strings"
	"testing"

	terminu "github.com/vulcanshen/terminu-go"
	"github.com/vulcanshen/terminu-go/tdptest"
)

// The first launch of every app, as tdptest walks it.
func TestFirstLaunch(t *testing.T) {
	tdptest.Check(t, tdptest.App{Name: "showcase", New: func() *terminu.App { return newShowcase().app }})
}

// Items in items are walked a level at a time (tdp panel.10, panel-158):
// Enter goes in, the location row naming the path; the filter and N of M
// hold that level only; Esc clears the filter first, then goes up a level
// with the cursor back on the folder it came out of, and at the top does
// nothing.
func TestPanelItemsInItems(t *testing.T) {
	s := newShowcase()
	d := tdptest.Start(t, s.app, 120, 40)
	is := s.inputs
	cur := func() string {
		r, _ := is.fileList.Current()
		return r.ID
	}
	at := func(dir, on string) {
		t.Helper()
		if is.dir != dir || is.fileList.Location != dir || cur() != on {
			t.Fatalf("in %q (location %q) on %q, want in %q on %q", is.dir, is.fileList.Location, cur(), dir, on)
		}
	}
	assets, docs := filepath.Join(projectDir, "assets"), filepath.Join(projectDir, "docs")
	images := filepath.Join(assets, "images")

	d.Press("I", "2", "l", "l") // [2] FILES, its Folders tab
	if is.tab != 2 {
		t.Fatalf("tab %d, want the Folders tab", is.tab)
	}
	at(projectDir, assets) // the folders first, by name
	d.Press("?")
	if d.Has("one level up") {
		t.Error("? has Esc: one level up at the top")
	}
	d.Press("Esc", "Esc")
	at(projectDir, assets)
	if d.Quit() {
		t.Fatal("Esc at the top left the app")
	}

	// A filter typed at the top stays there: docs/ shows all it holds.
	d.Press("/")
	d.Type("doc")
	d.Press("Enter", "Enter") // to the list, then into docs/
	at(docs, filepath.Join(docs, "guide.pdf"))
	if is.fileList.Filtering() {
		t.Error("the filter went into docs/ with it")
	}
	if n := len(is.fileList.Rows()); n != 3 {
		t.Errorf("docs/ has %d rows, want its 3 files", n)
	}
	for _, want := range []string{"guide.pdf", "notes.md", "scan.pdf"} {
		if !d.Has(want) {
			t.Errorf("docs/ has no %q\n%s", want, strings.Join(d.Screen(), "\n"))
		}
	}
	if d.Has("assets/") {
		t.Error("docs/ shows the level above")
	}
	d.Press("Esc")
	at(projectDir, docs)

	// Two levels down and a filter: Esc after Esc to the top.
	d.Press("g", "g", "Enter", "Enter")
	at(images, filepath.Join(images, "backgrounds"))
	d.Press("/")
	d.Type("zz")
	d.Press("Esc")
	if is.fileList.Filtering() {
		t.Fatal("Esc left the filter")
	}
	at(images, filepath.Join(images, "backgrounds"))
	d.Press("?")
	if !d.Has("one level up") {
		t.Error("? has no Esc: one level up below the top")
	}
	d.Press("Esc", "Esc")
	at(assets, images)
	d.Press("Esc")
	at(projectDir, assets)
	d.Press("Esc")
	at(projectDir, assets)
}
