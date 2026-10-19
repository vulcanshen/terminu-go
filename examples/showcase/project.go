package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	terminu "github.com/vulcanshen/terminu-go"
)

// projectDir is a pretend project under the temp directory, with one file of
// each kind a preview shows differently: text, Go, a PDF, an image, an
// archive, a binary, a private key, a file only its owner may read, a hidden
// one, a link and a broken one; a directory deep enough to be cut.
var projectDir = fakeProject()

func fakeProject() string {
	dir := filepath.Join(dataDir(), "project")
	put := func(name string, data []byte, mode os.FileMode) {
		p := filepath.Join(dir, name)
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		_ = os.WriteFile(p, data, mode)
		_ = os.Chmod(p, mode)
	}
	put("README.md", []byte("# showcase\n\nEvery component of terminu design, on fake data.\n\n- Panels\n- Inputs\n- Dialogs\n- Terminal\n"), 0o644)
	put("README-zh_TW.md", []byte("# showcase\n\n用假資料展示 terminu design 的每一個元件。\n"), 0o644)
	put("main.go", []byte("package main\n\nimport \"fmt\"\n\n// main says hello.\nfunc main() {\n\tfmt.Println(\"showcase\", 42)\n}\n"), 0o644)
	put("docs/notes.md", []byte("Notes on the README:\n- keep it short\n- link the CHANGELOG\n"), 0o644)
	put("docs/guide.pdf", minimalPDF("Showcase guide: every component, on fake data."), 0o644)
	put("assets/logo.png", tinyPNG(), 0o644)
	put("assets/fonts.zip", tinyZip(), 0o644)
	put("bin/tool", []byte("\x7fELF\x02\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\x3e\x00showcase tool"), 0o755)
	put("config/app.yaml", []byte("token: 3f9a-not-a-real-token\nendpoint: https://example.com\n"), 0o600)
	put("keys/deploy_key", []byte("-----BEGIN OPENSSH PRIVATE KEY-----\n(a fake key for the showcase)\n-----END OPENSSH PRIVATE KEY-----\n"), 0o644)
	put(".env", []byte("SHOWCASE_SECRET=not-a-real-secret\n"), 0o600)
	put("docs/scan.pdf", minimalPDF(""), 0o644)
	put("assets/photo-large.png", append(tinyPNG(), make([]byte, 1100<<10)...), 0o644)
	put("secret.bin", []byte("nobody may read this\n"), 0o000)
	put("assets/images/backgrounds/very-long-directory-name/landscape-2026.png", tinyPNG(), 0o644)
	_ = os.Symlink("docs/guide.pdf", filepath.Join(dir, "latest.pdf"))
	_ = os.Symlink("docs/gone.pdf", filepath.Join(dir, "broken.pdf"))
	_ = os.MkdirAll(filepath.Join(dir, "empty"), 0o755)
	// Kept empty, so that rm -rf still removes it.
	_ = os.MkdirAll(filepath.Join(dir, "locked"), 0o755)
	_ = os.Chmod(filepath.Join(dir, "locked"), 0o000)
	return dir
}

// dataDir holds the showcase's pretend files, at a path easy to type in the
// file-picker.
func dataDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.TempDir(), "terminu-showcase")
	}
	return "/tmp/terminu-showcase"
}

// minimalPDF is a one-page PDF that says text.
func minimalPDF(text string) []byte {
	var b bytes.Buffer
	var offsets []int
	obj := func(body string) {
		offsets = append(offsets, b.Len())
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", len(offsets), body)
	}
	b.WriteString("%PDF-1.4\n")
	obj("<< /Type /Catalog /Pages 2 0 R >>")
	obj("<< /Type /Pages /Kids [3 0 R] /Count 1 >>")
	obj("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 400 100] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>")
	stream := "BT /F1 12 Tf 20 50 Td (" + text + ") Tj ET"
	obj(fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream))
	obj("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>")
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(offsets)+1)
	for _, o := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", o)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets)+1, xref)
	return b.Bytes()
}

func tinyPNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := range 4 {
		for x := range 4 {
			img.Set(x, y, color.RGBA{uint8(x * 60), uint8(y * 60), 200, 255})
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

func tinyZip() []byte {
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for _, n := range []string{"fonts/SymbolsNerdFont-Regular.ttf", "fonts/README.txt", "LICENSE"} {
		w, _ := z.Create(n)
		_, _ = w.Write([]byte("fake"))
	}
	_ = z.Close()
	return b.Bytes()
}

// projectFiles are the project's files (not the hidden ones), by path.
func projectFiles() []string {
	var out []string
	_ = filepath.WalkDir(projectDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") && p != projectDir {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			out = append(out, p)
		}
		return nil
	})
	return out
}

func fileRows() []terminu.Row {
	var rows []terminu.Row
	for _, p := range projectFiles() {
		rel, _ := filepath.Rel(projectDir, p)
		st, err := os.Stat(p)
		if err != nil {
			continue
		}
		age := time.Since(st.ModTime()).Round(time.Minute)
		// The files in a folder first, the ones at the top after, however the
		// list is sorted: as a file manager keeps its directories first.
		group := 1
		if strings.Contains(rel, "/") {
			group = 0
		}
		rows = append(rows, terminu.Row{ID: p, Cells: []string{rel, humanSize(int(st.Size())), shortAge(age)},
			Sort: []string{rel, fmt.Sprint(st.Size()), fmt.Sprint(st.ModTime().Unix())}, SortGroup: group})
	}
	return rows
}

func fileItems() []terminu.FinderItem {
	var items []terminu.FinderItem
	for _, p := range projectFiles() {
		rel, _ := filepath.Rel(projectDir, p)
		items = append(items, terminu.FinderItem{ID: p, Text: rel})
	}
	return items
}
