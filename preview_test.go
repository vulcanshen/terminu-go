package terminu

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func previewText(lines []string) string { return StripStyles(strings.Join(lines, "\n")) }

func writeFile(t *testing.T, p string, data []byte, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, data, mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, mode); err != nil {
		t.Fatal(err)
	}
}

func TestFilePreviewKinds(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644)
	writeFile(t, filepath.Join(dir, "tool"), []byte{0x7f, 'E', 'L', 'F', 0, 1, 2, 3}, 0o755)
	writeFile(t, filepath.Join(dir, "logo.png"), []byte("\x89PNG\r\n\x1a\nfake"), 0o644)
	var zb bytes.Buffer
	z := zip.NewWriter(&zb)
	for _, n := range []string{"fonts/a.ttf", "fonts/b.ttf", "LICENSE"} {
		w, _ := z.Create(n)
		_, _ = w.Write([]byte("x"))
	}
	_ = z.Close()
	writeFile(t, filepath.Join(dir, "fonts.zip"), zb.Bytes(), 0o644)
	writeFile(t, filepath.Join(dir, "tree/a/b/c/deep.txt"), nil, 0o644)
	writeFile(t, filepath.Join(dir, "tree/z.txt"), nil, 0o644)
	_ = os.Mkdir(filepath.Join(dir, "empty"), 0o755)

	text := previewText(filePreview(filepath.Join(dir, "main.go"), 40, 10, false, nil))
	if !strings.HasPrefix(text, "1 package main\n2 \n3 func main() {}") {
		t.Errorf("text with line numbers:\n%s", text)
	}
	if lines := filePreview(filepath.Join(dir, "main.go"), 40, 10, false, nil); !strings.Contains(lines[0], "38;2;") {
		t.Error("the text is not coloured")
	}
	if hex := previewText(filePreview(filepath.Join(dir, "tool"), 80, 10, false, nil)); !strings.HasPrefix(hex, "00000000  7f 45 4c 46") || !strings.Contains(hex, ".ELF") {
		t.Errorf("hex: %s", hex)
	}
	if img := previewText(filePreview(filepath.Join(dir, "logo.png"), 20, 10, false, nil)); !strings.HasPrefix(img, "data:image/png;base6") {
		t.Errorf("image: %s", img)
	}
	zipTree := previewText(filePreview(filepath.Join(dir, "fonts.zip"), 40, 10, false, nil))
	if !strings.Contains(zipTree, "fonts\n├─") || !strings.Contains(zipTree, "└─ "+glyphFile+" b.ttf") || !strings.HasSuffix(zipTree, "LICENSE") {
		t.Errorf("archive tree:\n%s", zipTree)
	}
	tree := previewText(filePreview(filepath.Join(dir, "tree"), 40, 10, false, nil))
	if !strings.Contains(tree, "└─ "+glyphFolder+" b") || strings.Contains(tree, "deep.txt") {
		t.Errorf("a tree is three levels deep:\n%s", tree)
	}
	if short := previewText(filePreview(filepath.Join(dir, "tree"), 40, 2, false, nil)); !strings.HasSuffix(short, "…") {
		t.Errorf("a tree cut short ends in …:\n%s", short)
	}
	if empty := previewText(filePreview(filepath.Join(dir, "empty"), 40, 5, false, nil)); empty != "(empty)" {
		t.Errorf("empty directory: %q", empty)
	}
}

func TestFilePreviewHidesSecrets(t *testing.T) {
	dir := t.TempDir()
	key := filepath.Join(dir, "deploy_key")
	writeFile(t, key, []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nsecret\n-----END OPENSSH PRIVATE KEY-----\n"), 0o644)
	own := filepath.Join(dir, "app.yaml")
	writeFile(t, own, []byte("token: x\n"), 0o600)
	if got := previewText(filePreview(own, 40, 5, false, nil)); !strings.Contains(got, "token: x") {
		t.Errorf("off by default: everything shows: %q", got)
	}
	if got := previewText(filePreview(key, 40, 5, true, nil)); got != "-----BEGIN OPENSSH PRIVATE KEY-----\n(sensitive content not shown)" {
		t.Errorf("a private key shows its first line only: %q", got)
	}
	if got := previewText(filePreview(own, 40, 5, true, nil)); got != "(sensitive content not shown)" {
		t.Errorf("a file only its owner may read shows nothing: %q", got)
	}
}

func TestFileInfo(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"), []byte("hello"), 0o640)
	writeFile(t, filepath.Join(dir, "sub/x"), nil, 0o644)
	writeFile(t, filepath.Join(dir, "sub/y"), nil, 0o644)
	_ = os.Symlink("a.txt", filepath.Join(dir, "link"))
	info := previewText(fileInfo(filepath.Join(dir, "a.txt"), 40, Blue))
	for _, want := range []string{"TYPE      text", "SIZE      5 B", "MODIFIED  ", "MODE      -rw-r-----"} {
		if !strings.Contains(info, want) {
			t.Errorf("%q not in\n%s", want, info)
		}
	}
	if d := previewText(fileInfo(filepath.Join(dir, "sub"), 40, Blue)); !strings.Contains(d, "TYPE      directory") || !strings.Contains(d, "SIZE      2 items") {
		t.Errorf("directory:\n%s", d)
	}
	if l := previewText(fileInfo(filepath.Join(dir, "link"), 40, Blue)); !strings.Contains(l, "TYPE      link → a.txt") {
		t.Errorf("symlink:\n%s", l)
	}
}

func TestFinderFileLayout(t *testing.T) {
	dir := fakeTree(t)
	for _, sz := range [][2]int{{120, 30}, {80, 24}} {
		ta := newTestApp(t)
		ta.resize(sz[0], sz[1])
		ta.PickFile(FilePicker{Title: "Identity file", Root: dir, Value: filepath.Join(dir, "config")})
		ta.settle()
		text := ta.screenText()
		info := strings.Contains(text, "INFO")
		if info == (sz[0] >= 96) {
			t.Errorf("%dx%d: an INFO box %v (it is one only when the preview is below)", sz[0], sz[1], info)
		}
		if !strings.Contains(text, "TYPE      text") || !strings.Contains(text, "MODE") {
			t.Errorf("%dx%d: no facts:\n%s", sz[0], sz[1], text)
		}
	}
}
