package terminu

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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

// TestFilePreviewEscapes: a file's colour codes colour it; an escape
// sequence that is not one is shown, not obeyed: ESC by its red name, the
// rest of it as text; a \r in a line leaves its last state
// (components/input/file-picker, file-picker-93).
func TestFilePreviewEscapes(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "log.txt"), []byte("ok \x1b[31mred\x1b[0m done\n\x1b[2Jclear\n10%\r100%\n"), 0o600)
	lines := filePreview(filepath.Join(dir, "log.txt"), 60, 5, false, nil, 4)
	text := StripStyles(strings.Join(lines, "\n"))
	for _, want := range []string{"1 ok red done", `2 \x1b[2Jclear`, "3 100%"} {
		if !strings.Contains(text, want) {
			t.Errorf("%q is not shown: %q", want, text)
		}
	}
	all := strings.Join(lines, "")
	if !strings.Contains(all, "\x1b[31mred") || !strings.Contains(all, Fg(Red).Render(`\x1b`)) || strings.Contains(all, "\x1b[2J") {
		t.Errorf("the colour code colours, the other ESC by its red name: %q", lines)
	}
}

// TestLsMode: a mode as ls -l writes it (components/input/file-picker).
func TestLsMode(t *testing.T) {
	for _, c := range []struct {
		m    fs.FileMode
		want string
	}{
		{0o600, "-rw-------"},
		{fs.ModeDir | 0o700, "drwx------"},
		{fs.ModeDir | fs.ModeSticky | 0o777, "drwxrwxrwt"},
		{fs.ModeSetuid | 0o755, "-rwsr-xr-x"},
		{fs.ModeSetgid | 0o640, "-rw-r-S---"},
		{fs.ModeSymlink | 0o777, "lrwxrwxrwx"},
		{fs.ModeDevice | fs.ModeCharDevice | 0o666, "crw-rw-rw-"},
		{fs.ModeDevice | 0o660, "brw-rw----"},
		{fs.ModeNamedPipe | 0o644, "prw-r--r--"},
		{fs.ModeSocket | 0o755, "srwxr-xr-x"},
	} {
		if got := lsMode(c.m); got != c.want {
			t.Errorf("lsMode(%v) = %q, want %q", c.m, got, c.want)
		}
	}
	shared := filepath.Join(t.TempDir(), "shared")
	if err := os.Mkdir(shared, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(shared, 0o777|os.ModeSticky); err != nil {
		t.Fatal(err)
	}
	if l := previewText(fileInfo(shared, 40, Blue)); !strings.Contains(l, "MODE      drwxrwxrwt") {
		t.Errorf("the metadata writes it as ls -l does:\n%s", l)
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
	// A directory of one: 1 item.
	one := filepath.Join(dir, "one")
	if err := os.Mkdir(one, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(one, "x"), nil, 0o600)
	if l := previewText(fileInfo(one, 40, Blue)); !strings.Contains(l, "SIZE      1 item ") {
		t.Errorf("one item:\n%s", l)
	}
	// Gone since it was listed: every row a dash, the four rows kept.
	gone := strings.Split(previewText(fileInfo(filepath.Join(dir, "deleted.txt"), 40, Blue)), "\n")
	for i, w := range []string{"TYPE      —", "SIZE      —", "MODIFIED  —", "MODE      —"} {
		if i >= len(gone) || !strings.HasPrefix(gone[i], w) {
			t.Errorf("a file gone, row %d: want %q in\n%s", i, w, strings.Join(gone, "\n"))
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
		text := ta.readAll()
		info := strings.Contains(text, "INFO")
		if info == (sz[0] >= 96) {
			t.Errorf("%dx%d: an INFO box %v (it is one only when the preview is below)", sz[0], sz[1], info)
		}
		if !strings.Contains(text, "TYPE      text") || !strings.Contains(text, "MODE") {
			t.Errorf("%dx%d: no facts:\n%s", sz[0], sz[1], text)
		}
	}
}

// memFS is a PreviewFS in memory, as an app gives a remote host's.
type memFS map[string]string // path to content; a path ending in / is a directory

type memInfo struct {
	name string
	size int64
	dir  bool
}

func (i memInfo) Name() string { return i.name }
func (i memInfo) Size() int64  { return i.size }
func (i memInfo) Mode() fs.FileMode {
	if i.dir {
		return fs.ModeDir | 0o755
	}
	return 0o644
}
func (i memInfo) ModTime() time.Time { return time.Date(2026, 10, 19, 9, 30, 0, 0, time.Local) }
func (i memInfo) IsDir() bool        { return i.dir }
func (i memInfo) Sys() any           { return nil }

type memFile struct{ *bytes.Reader }

func (memFile) Close() error { return nil }

func (m memFS) Open(name string) (PreviewFile, error) {
	if c, ok := m[name]; ok {
		return memFile{bytes.NewReader([]byte(c))}, nil
	}
	return nil, fs.ErrNotExist
}

func (m memFS) Stat(name string) (fs.FileInfo, error) {
	if c, ok := m[name]; ok {
		return memInfo{path.Base(name), int64(len(c)), false}, nil
	}
	if _, ok := m[name+"/"]; ok {
		return memInfo{path.Base(name), 0, true}, nil
	}
	return nil, fs.ErrNotExist
}

func (m memFS) Lstat(name string) (fs.FileInfo, error) { return m.Stat(name) }
func (m memFS) ReadLink(string) (string, error)        { return "", fs.ErrInvalid }

func (m memFS) ReadDir(name string) ([]fs.FileInfo, error) {
	var out []fs.FileInfo
	for p, c := range m {
		if dir, base := path.Split(strings.TrimSuffix(p, "/")); dir == name+"/" {
			out = append(out, memInfo{base, int64(len(c)), strings.HasSuffix(p, "/")})
		}
	}
	return out, nil
}

// TestFinderFilesInFS: a finder of files in a file system the app gives
// previews them, their trees and their facts from it (components/input/
// file-picker, the preview; sshu SS18).
// readAll draws a frame and reads at once what it asked to read in the
// background, as the app does off the update loop; the screen then.
func (ta *testApp) readAll() string {
	ta.Frame()
	for {
		cmd := ta.fetch()
		if cmd == nil {
			break
		}
		ta.runAll(cmd)
		ta.Frame()
	}
	return ta.screenText()
}

// runAll runs cmd and gives the app what it sends, a batch's too; not what
// the app sends back.
func (ta *testApp) runAll(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	switch m := cmd().(type) {
	case tea.BatchMsg:
		for _, c := range m {
			ta.runAll(c)
		}
	default:
		ta.Update(m)
	}
}

// TestFilePreviewReadInTheBackground: a file preview is read off the update
// loop: meanwhile the loading icon after its title, "(loading…)" and blank
// facts; what was read is kept for the row while the popup is open
// (components/input/file-picker; sshu SS40).
func TestFilePreviewReadInTheBackground(t *testing.T) {
	remote := memFS{"/srv/": "", "/srv/app.conf": "port = 22\n", "/srv/b.conf": "host = web\n"}
	ta := newTestApp(t)
	ta.Find(&Finder{Title: "Remote", Files: true, FS: remote,
		Items: []FinderItem{{ID: "/srv/app.conf", Text: "app.conf"}, {ID: "/srv/b.conf", Text: "b.conf"}}})
	ta.settle()
	text := ta.screenText()
	if !strings.Contains(text, "(loading…)") || strings.Contains(text, "port = 22") || strings.Contains(text, "—") ||
		!strings.Contains(text, "TYPE") || strings.Contains(text, "TYPE      text") {
		t.Errorf("before it is read: (loading…), blank facts, not a dash:\n%s", text)
	}
	titled := slices.ContainsFunc(ta.Frame(), func(row string) bool {
		p := StripStyles(row)
		return strings.Contains(p, "app.conf") && slices.ContainsFunc(loadingFrames[:], func(g string) bool { return strings.Contains(p, g) })
	})
	if !titled {
		t.Error("the loading icon after the preview title")
	}
	if !ta.loading() {
		t.Error("frames keep coming while it reads")
	}
	cmd := ta.Update2(struct{}{})
	if cmd == nil {
		t.Fatal("the next update starts reading")
	}
	ta.runAll(cmd)
	if !strings.Contains(ta.screenText(), "port = 22") {
		t.Error("what the update started reading is drawn once it is read")
	}
	if text := ta.readAll(); !strings.Contains(text, "port = 22") || !strings.Contains(text, "TYPE      text") {
		t.Errorf("read:\n%s", text)
	}
	ta.press("Tab", "j")
	ta.readAll()
	ta.press("k")
	ta.Frame()
	fv := ta.topView().(*finderView)
	if fv.read.busy() || !strings.Contains(ta.screenText(), "port = 22") {
		t.Error("back to a row read before: nothing read again")
	}
}

// TestPreviewFromALineThatFits: a file that fits in the preview is drawn
// from its first line even when the preview starts from a later one
// (components/input/finder).
func TestPreviewFromALineThatFits(t *testing.T) {
	fs := memFS{"/a.txt": "one\ntwo\nthree\nfour\nfive\n"}
	lines := previewIn(fs, "/a.txt", 4, 40, 10, false, nil, 4)
	if text := previewText(lines); !strings.HasPrefix(strings.TrimSpace(text), "1 one") {
		t.Errorf("from line 1:\n%s", text)
	}
	long := ""
	for i := 1; i <= 40; i++ {
		long += fmt.Sprintf("line %d\n", i)
	}
	fs["/b.txt"] = long
	if text := previewText(previewIn(fs, "/b.txt", 20, 40, 10, false, nil, 4)); !strings.HasPrefix(strings.TrimSpace(text), "18 line 18") {
		t.Errorf("the line on the third row:\n%s", text)
	}
}

func TestFinderFilesInFS(t *testing.T) {
	remote := memFS{"/srv/": "", "/srv/app.conf": "port = 22\nhost = web\n", "/srv/logs/": "", "/srv/logs/a.log": "x\n"}
	ta := newTestApp(t)
	ta.Find(&Finder{Title: "Remote", Files: true, FS: remote,
		Items: []FinderItem{{ID: "/srv/app.conf", Text: "app.conf"}, {ID: "/srv", Text: "srv"}}})
	ta.settle()
	text := ta.readAll()
	for _, want := range []string{"1 port = 22", "2 host = web", "TYPE", "text", "21 B", "2026-10-19 09:30", "-rw-r--r--"} {
		if !strings.Contains(text, want) {
			t.Errorf("the remote file: no %q in\n%s", want, text)
		}
	}
	ta.press("Tab", "j")
	if text := ta.readAll(); !strings.Contains(text, glyphFolder+" logs") || !strings.Contains(text, "  └─ "+glyphFile+" a.log") ||
		!strings.Contains(text, "2 items") {
		t.Errorf("the remote directory:\n%s", text)
	}
}

// TestFilePreviewPanel: a panel can show a file as a file-picker's preview
// does, from a line a search found, with its facts under it
// (components/input/file-picker; layout/panel, details; filu FL3, FL16).
func TestFilePreviewPanel(t *testing.T) {
	dir := t.TempDir()
	var text []string
	for i := range 40 {
		text = append(text, fmt.Sprintf("line %d", i+1))
	}
	p := filepath.Join(dir, "notes.txt")
	writeFile(t, p, []byte(strings.Join(text, "\n")+"\n"), 0o644)
	fp := &FilePreview{Path: p, Line: 20, Info: true}
	out := plainLines(fp.Draw(40, 12, true))
	if len(out) != 12 || out[0] != " 18 line 18" || out[2] != " 20 line 20" {
		t.Errorf("from line 20:\n%s", strings.Join(out, "\n"))
	}
	if !strings.Contains(out[8], "TYPE") || !strings.Contains(out[8], "text") || !strings.HasPrefix(out[7], " ─") {
		t.Errorf("the facts under a separator:\n%s", strings.Join(out, "\n"))
	}
	fp.Line = 0
	if out := plainLines(fp.Draw(40, 12, true)); out[0] != " 1 line 1" {
		t.Errorf("from the top: %q", out[0])
	}
	// At the end of the file it shows the last lines.
	fp.Line = 39
	if out := plainLines(fp.Draw(40, 12, true)); out[0] != " 34 line 34" || out[6] != " 40 line 40" {
		t.Errorf("near the end:\n%s", strings.Join(out, "\n"))
	}
	if (&FilePreview{}).Empty() != true || fp.Empty() {
		t.Error("Empty reports the path")
	}
}

// TestFilePreviewPanelHidesSecrets: a panel's file preview keeps to the app's
// hide_secrets_in_preview, as a file-picker's does (components/input/
// file-picker).
func TestFilePreviewPanelHidesSecrets(t *testing.T) {
	key := filepath.Join(t.TempDir(), "id_ed25519")
	writeFile(t, key, []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nsecret\n"), 0o600)
	ta := &testApp{App: anySize(New("test"))}
	ta.AddScreen(&Screen{Name: "Files", Key: "F", Panels: []*Panel{{Label: "Key", Body: &FilePreview{Path: key}}}})
	ta.resize(80, 20)
	if text := ta.screenText(); !strings.Contains(text, "secret") {
		t.Errorf("shown in full:\n%s", text)
	}
	ta.HideSecretsInPreview = true
	if text := ta.screenText(); strings.Contains(text, "secret") || !strings.Contains(text, "(sensitive content not shown)") {
		t.Errorf("hidden:\n%s", text)
	}
}
