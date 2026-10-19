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
	writeFile(t, filepath.Join(dir, "tool"), []byte{0x7f, 'E', 'L', 'F', 0, 1, 2, 3, 'a', 'b'}, 0o755)
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

	text := previewText(filePreview(filepath.Join(dir, "main.go"), 40, 10, false, nil, 4))
	if !strings.HasPrefix(text, "1 package main\n2 \n3 func main() {}") {
		t.Errorf("text with line numbers:\n%s", text)
	}
	if lines := filePreview(filepath.Join(dir, "main.go"), 40, 10, false, nil, 4); !strings.Contains(lines[0], "38;2;") {
		t.Error("the text is not coloured")
	}
	want := "7f 45 4c 46 00 01 02 03  .ELF....\n61 62                    ab"
	if hex := previewText(filePreview(filepath.Join(dir, "tool"), 120, 10, false, nil, 4)); hex != want {
		t.Errorf("hex, always eight bytes a row:\n%q\nwant\n%q", hex, want)
	}
	if img := previewText(filePreview(filepath.Join(dir, "logo.png"), 20, 10, false, nil, 4)); !strings.HasPrefix(img, "data:image/png · 12 B\n\ndata:image/png;base6") {
		t.Errorf("image: %s", img)
	}
	big := filepath.Join(dir, "big.png")
	writeFile(t, big, bytes.Repeat([]byte{1}, 1<<20+1), 0o644)
	if img := previewText(filePreview(big, 40, 10, false, nil, 4)); img != "(1.0 MB — too large to inline as a data URI)" {
		t.Errorf("an image over 1 MB: %s", img)
	}
	zipTree := previewText(filePreview(filepath.Join(dir, "fonts.zip"), 40, 10, false, nil, 4))
	if !strings.Contains(zipTree, "fonts\n  ├─") || !strings.Contains(zipTree, "└─ "+glyphFile+" b.ttf") || !strings.HasSuffix(zipTree, "LICENSE") {
		t.Errorf("archive tree:\n%s", zipTree)
	}
	tree := previewText(filePreview(filepath.Join(dir, "tree"), 40, 10, false, nil, 4))
	// The second level two cells in, under the first level's name; three
	// levels and no more (components/input/file-picker).
	wantTree := glyphFolder + " a\n  └─ " + glyphFolder + " b\n     └─ " + glyphFolder + " c\n" + glyphFile + " z.txt"
	if tree != wantTree {
		t.Errorf("a tree is three levels deep:\n%s", tree)
	}
	if short := previewText(filePreview(filepath.Join(dir, "tree"), 40, 2, false, nil, 4)); !strings.HasSuffix(short, "…") {
		t.Errorf("a tree cut short ends in …:\n%s", short)
	}
	if empty := previewText(filePreview(filepath.Join(dir, "empty"), 40, 5, false, nil, 4)); empty != "(empty)" {
		t.Errorf("empty directory: %q", empty)
	}
}

func TestFilePreviewHidesSecrets(t *testing.T) {
	dir := t.TempDir()
	key := filepath.Join(dir, "deploy_key")
	writeFile(t, key, []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nsecret\n-----END OPENSSH PRIVATE KEY-----\n"), 0o644)
	own := filepath.Join(dir, "app.yaml")
	writeFile(t, own, []byte("token: x\n"), 0o600)
	if got := previewText(filePreview(own, 40, 5, false, nil, 4)); !strings.Contains(got, "token: x") {
		t.Errorf("off by default: everything shows: %q", got)
	}
	if got := previewText(filePreview(key, 40, 5, true, nil, 4)); got != "-----BEGIN OPENSSH PRIVATE KEY-----\n(sensitive content not shown)" {
		t.Errorf("a private key shows its first line only: %q", got)
	}
	if got := previewText(filePreview(own, 40, 5, true, nil, 4)); got != "(sensitive content not shown)" {
		t.Errorf("a file only its owner may read shows nothing: %q", got)
	}
	if err := os.Chmod(key, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := previewText(filePreview(key, 40, 5, true, nil, 4)); !strings.HasPrefix(got, "-----BEGIN OPENSSH PRIVATE KEY-----\n") {
		t.Errorf("a private key only its owner may read still shows its header: %q", got)
	}
	if got := previewText(fileInfo(key, 40, Blue)); !strings.Contains(got, "TYPE      private key") {
		t.Errorf("TYPE of a private key:\n%s", got)
	}
	if got := previewText(filePreview(dir, 40, 5, true, nil, 4)); strings.Contains(got, "sensitive") {
		t.Errorf("a directory is never hidden: %q", got)
	}
}

func TestFileInfo(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"), []byte("hello"), 0o640)
	writeFile(t, filepath.Join(dir, "b.bin"), bytes.Repeat([]byte{0}, 4300), 0o644)
	writeFile(t, filepath.Join(dir, "sub/x"), nil, 0o644)
	writeFile(t, filepath.Join(dir, "sub/y"), nil, 0o644)
	_ = os.Symlink("a.txt", filepath.Join(dir, "link"))
	info := previewText(fileInfo(filepath.Join(dir, "a.txt"), 40, Blue))
	for _, want := range []string{"TYPE      text", "SIZE      5 B", "MODIFIED  ", "MODE      -rw-r-----"} {
		if !strings.Contains(info, want) {
			t.Errorf("%q not in\n%s", want, info)
		}
	}
	if b := previewText(fileInfo(filepath.Join(dir, "b.bin"), 40, Blue)); !strings.Contains(b, "SIZE      4.2 KB") || !strings.Contains(b, "TYPE      binary") {
		t.Errorf("4300 bytes:\n%s", b)
	}
	if d := previewText(fileInfo(filepath.Join(dir, "sub"), 40, Blue)); !strings.Contains(d, "TYPE      directory") || !strings.Contains(d, "SIZE      2 items") {
		t.Errorf("directory:\n%s", d)
	}
	if l := previewText(fileInfo(filepath.Join(dir, "link"), 40, Blue)); !strings.Contains(l, "TYPE      link → a.txt") {
		t.Errorf("symlink:\n%s", l)
	}
}

// TestFilePickerMetaDash: a link's SIZE, MODIFIED and MODE are its
// target's; what cannot be had is — and the four rows stay
// (components/input/file-picker).
func TestFilePickerMetaDash(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"), []byte("hello"), 0o600)
	_ = os.Symlink("a.txt", filepath.Join(dir, "link"))
	_ = os.Symlink("gone.txt", filepath.Join(dir, "broken"))
	l := previewText(fileInfo(filepath.Join(dir, "link"), 40, Blue))
	for _, want := range []string{"SIZE      5 B", "MODE      -rw-------"} {
		if !strings.Contains(l, want) {
			t.Errorf("a link writes its target's %q:\n%s", want, l)
		}
	}
	b := strings.Split(previewText(fileInfo(filepath.Join(dir, "broken"), 40, Blue)), "\n")
	want := []string{"TYPE      link → gone.txt", "SIZE      —", "MODIFIED  —", "MODE      —"}
	for i, w := range want {
		if i >= len(b) || !strings.HasPrefix(b[i], w) {
			t.Errorf("a broken link, row %d: want %q in\n%s", i, w, strings.Join(b, "\n"))
		}
	}
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(locked, 0o755)
	if _, err := os.ReadDir(locked); err == nil {
		t.Skip("running as a user who reads any directory")
	}
	d := previewText(fileInfo(locked, 40, Blue))
	for _, want := range []string{"TYPE      directory", "SIZE      —", "MODE      d---------"} {
		if !strings.Contains(d, want) {
			t.Errorf("an unreadable directory: %q not in\n%s", want, d)
		}
	}
	if strings.Contains(d, "items") || strings.Contains(d, "MODIFIED  —") {
		t.Errorf("an unreadable directory still has its time, and no count:\n%s", d)
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
