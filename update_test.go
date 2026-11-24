package terminu

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// fakeReleases is GitHub for app "tool" of owner "me": its latest release
// latest, its files for linux/amd64 and windows/amd64; hits counts the
// requests. A bad sum makes checksums.txt wrong for the tar.gz.
type fakeReleases struct {
	latest  string
	badSum  bool
	hits    int
	srv     *httptest.Server
	newExe  []byte
	tarball []byte
	zipball []byte
}

func newFakeReleases(t *testing.T, latest string) *fakeReleases {
	t.Helper()
	f := &fakeReleases{latest: latest, newExe: []byte("#!/bin/sh\necho tool " + latest + "\n")}
	var tb bytes.Buffer
	gz := gzip.NewWriter(&tb)
	tw := tar.NewWriter(gz)
	for _, file := range []struct {
		name string
		body []byte
	}{{"README.md", []byte("readme")}, {"tool", f.newExe}} {
		_ = tw.WriteHeader(&tar.Header{Name: file.name, Mode: 0o755, Size: int64(len(file.body)), Typeflag: tar.TypeReg})
		_, _ = tw.Write(file.body)
	}
	_ = tw.Close()
	_ = gz.Close()
	f.tarball = tb.Bytes()
	var zb bytes.Buffer
	zw := zip.NewWriter(&zb)
	w, _ := zw.Create("tool.exe")
	_, _ = w.Write(f.newExe)
	_ = zw.Close()
	f.zipball = zb.Bytes()
	sum := func(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.hits++
		dl := "/me/tool/releases/download/v" + f.latest + "/"
		switch r.URL.Path {
		case "/me/tool/releases/latest":
			// As github.com: to the latest release's page; with none yet,
			// to the releases page.
			to := f.srv.URL + "/me/tool/releases"
			if f.latest != "" {
				to += "/tag/v" + f.latest
			}
			http.Redirect(w, r, to, http.StatusFound)
		case "/me/tool/releases", "/me/tool/releases/tag/v" + f.latest:
			_, _ = w.Write([]byte("<html>a page</html>"))
		case "/repos/me/tool/releases/latest":
			// GitHub's API, its 60 an hour used up: not asked.
			w.Header().Set("X-Ratelimit-Remaining", "0")
			http.Error(w, "API rate limit exceeded", http.StatusForbidden)
		case dl + "checksums.txt":
			tarSum := sum(f.tarball)
			if f.badSum {
				tarSum = sum([]byte("something else"))
			}
			_, _ = w.Write([]byte(tarSum + "  tool_" + f.latest + "_linux_amd64.tar.gz\n" +
				sum(f.zipball) + "  tool_" + f.latest + "_windows_amd64.zip\n"))
		case dl + "tool_" + f.latest + "_linux_amd64.tar.gz":
			_, _ = w.Write(f.tarball)
		case dl + "tool_" + f.latest + "_windows_amd64.zip":
			_, _ = w.Write(f.zipball)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

// env is the update's world: exe the running executable, on goos/amd64.
func (f *fakeReleases) env(exe, goos string, brew func(args ...string) *exec.Cmd) updateEnv {
	return updateEnv{exe: exe, goos: goos, goarch: "amd64", github: f.srv.URL, client: f.srv.Client(), brew: brew}
}

// installed puts an old tool at dir/rel and gives its path.
func installed(t *testing.T, dir, rel string) string {
	t.Helper()
	exe := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(exe), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, []byte("old tool"), 0o755); err != nil {
		t.Fatal(err)
	}
	return exe
}

type afterCall struct {
	exe     string
	updated bool
}

func runUpdate(u SelfUpdate, env updateEnv) (code int, stdout, stderr string, after []afterCall) {
	var out, errs bytes.Buffer
	u.After = func(exe string, updated bool) error { after = append(after, afterCall{exe, updated}); return nil }
	code = u.run(env, &out, &errs)
	return code, out.String(), errs.String(), after
}

// TestE1Update: <app> update brings the app to its latest release the way it
// was installed (tdp E1, E1-9): a release download or the install script's
// put in place of the running file, held to checksums.txt (a zip and the
// file moved to .old on Windows); Homebrew runs brew upgrade <tap>/<app>;
// Scoop prints the command to run. Progress on stdout, then the app's After
// with the new executable. The latest is where github.com's releases/latest
// leads; the fake's API has used up its 60 an hour, as a shared network's
// does.
func TestE1Update(t *testing.T) {
	u := SelfUpdate{App: "tool", Version: "1.2.0", Owner: "me", Tap: "me/tap"}
	f := newFakeReleases(t, "1.3.0")

	// The install script's, or a download.
	exe := installed(t, t.TempDir(), "bin/tool")
	code, out, errs, after := runUpdate(u, f.env(exe, "linux", nil))
	if code != 0 || errs != "" {
		t.Fatalf("exit %d: %s", code, errs)
	}
	if out != "Updating tool 1.2.0 to 1.3.0…\ntool 1.3.0 is in "+exe+".\n" {
		t.Errorf("stdout %q", out)
	}
	if b, _ := os.ReadFile(exe); !bytes.Equal(b, f.newExe) {
		t.Errorf("the file was not replaced: %q", b)
	}
	if st, _ := os.Stat(exe); runtime.GOOS != "windows" && st.Mode().Perm()&0o100 == 0 {
		t.Errorf("not executable: %v", st.Mode())
	}
	if !slices.Equal(after, []afterCall{{exe, true}}) {
		t.Errorf("After %v", after)
	}
	if left, _ := filepath.Glob(filepath.Join(filepath.Dir(exe), ".tool-update-*")); len(left) > 0 {
		t.Errorf("left behind: %v", left)
	}

	// Windows: the zip, and the running file moved aside first.
	exe = installed(t, t.TempDir(), "tool.exe")
	if code, _, errs, _ = runUpdate(u, f.env(exe, "windows", nil)); code != 0 {
		t.Fatalf("Windows: exit %d: %s", code, errs)
	}
	if b, _ := os.ReadFile(exe); !bytes.Equal(b, f.newExe) {
		t.Errorf("Windows: not replaced: %q", b)
	}
	if b, _ := os.ReadFile(exe + ".old"); string(b) != "old tool" {
		t.Errorf("Windows: the old one is not in .old: %q", b)
	}

	// Homebrew: brew upgrade <tap>/<app>, After with the link in its bin.
	root := t.TempDir()
	exe = installed(t, root, "Cellar/tool/1.2.0/bin/tool")
	var ran []string
	brew := func(args ...string) *exec.Cmd { ran = args; return exec.Command("go", "version") }
	code, out, errs, after = runUpdate(u, f.env(exe, "darwin", brew))
	if code != 0 || !slices.Equal(ran, []string{"upgrade", "me/tap/tool"}) {
		t.Fatalf("Homebrew: exit %d, ran %v: %s", code, ran, errs)
	}
	if !strings.HasPrefix(out, "Updating tool 1.2.0 to 1.3.0 with Homebrew…\ngo version") {
		t.Errorf("Homebrew: brew's output not passed through: %q", out)
	}
	if !slices.Equal(after, []afterCall{{filepath.Join(root, "bin", "tool"), true}}) {
		t.Errorf("Homebrew: After %v", after)
	}
	if b, _ := os.ReadFile(exe); string(b) != "old tool" {
		t.Error("Homebrew: the file was replaced by the lib")
	}
	failing := func(args ...string) *exec.Cmd { return exec.Command("go", "no-such-command") }
	if code, _, errs, after = runUpdate(u, f.env(exe, "darwin", failing)); code != 1 || !strings.HasSuffix(errs, "tool: brew upgrade me/tap/tool: exit status 2\n") || after != nil {
		t.Errorf("Homebrew failing: exit %d, %q, After %v", code, errs, after)
	}

	// Scoop: the command to run, and the app's own after it.
	exe = installed(t, t.TempDir(), `scoop/apps/tool/current/tool.exe`)
	f.hits = 0
	code, out, _, after = runUpdate(SelfUpdate{App: "tool", Version: "1.2.0", Owner: "me", Then: "tool install claude"}, f.env(exe, "windows", nil))
	if code != 0 || out != "tool 1.3.0 is out. Scoop installed this one: close tool and run\n\n  scoop update tool\n  tool install claude\n" || after != nil {
		t.Errorf("Scoop: exit %d, %q, After %v", code, out, after)
	}
	if f.hits != 1 {
		t.Errorf("Scoop: %d requests, not the latest alone", f.hits)
	}
}

// TestE1UpdateNot: already the latest it says so, exit 0, the app's After
// still told; built from source, or with no right to write where the file
// is, it says why and what to do, exit 1, never asking for more rights; a
// checksum that does not match or a lookup that fails installs nothing
// (tdp E1, E1-10).
func TestE1UpdateNot(t *testing.T) {
	u := SelfUpdate{App: "tool", Version: "1.3.0", Owner: "me", Tap: "me/tap"}
	f := newFakeReleases(t, "1.3.0")
	exe := installed(t, t.TempDir(), "tool")
	code, out, errs, after := runUpdate(u, f.env(exe, "linux", nil))
	if code != 0 || out != "tool 1.3.0 is the latest release.\n" || errs != "" || !slices.Equal(after, []afterCall{{exe, false}}) {
		t.Errorf("the latest: exit %d, %q %q, After %v", code, out, errs, after)
	}
	u.Version = "v1.3.0" // written with its v: the same
	if code, out, _, _ = runUpdate(u, f.env(exe, "linux", nil)); code != 0 || out != "tool 1.3.0 is the latest release.\n" {
		t.Errorf("with a v: exit %d, %q", code, out)
	}
	u.Version = "1.4.0-beta.1"
	if code, out, _, _ = runUpdate(u, f.env(exe, "linux", nil)); code != 0 || out != "tool 1.4.0-beta.1 is newer than the latest release, 1.3.0.\n" {
		t.Errorf("a pre-release ahead: exit %d, %q", code, out)
	}

	// Built from source: nothing looked up.
	f.hits = 0
	for _, c := range []struct {
		u    SelfUpdate
		want string
	}{
		{SelfUpdate{App: "tool", Version: "dev"}, "tool: This tool is built from source: update it there\n"},
		{SelfUpdate{App: "tool", Source: "git pull && make install", Then: "tool install claude"},
			"tool: This tool is built from source: update it there (`git pull && make install`), then run `tool install claude`\n"},
	} {
		code, out, errs, after = runUpdate(c.u, f.env(exe, "linux", nil))
		if code != 1 || out != "" || errs != c.want || after != nil {
			t.Errorf("from source: exit %d, %q, %q", code, out, errs)
		}
	}
	if f.hits != 0 {
		t.Errorf("from source: %d requests", f.hits)
	}

	// No right to write where it is: why and what to do, nothing downloaded.
	u.Version = "1.2.0"
	if runtime.GOOS != "windows" && os.Geteuid() != 0 {
		dir := t.TempDir()
		exe = installed(t, dir, "tool")
		if err := os.Chmod(dir, 0o555); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
		f.hits = 0
		code, _, errs, after = runUpdate(u, f.env(exe, "linux", nil))
		want := "tool: Cannot write in " + dir + ": permission denied\ntool: Run the install script again, or `tool update` as the folder's owner\n"
		if code != 1 || errs != want || after != nil {
			t.Errorf("no right to write: exit %d, %q", code, errs)
		}
		if f.hits != 1 {
			t.Errorf("no right to write: %d requests, not the latest alone", f.hits)
		}
		if b, _ := os.ReadFile(exe); string(b) != "old tool" {
			t.Error("no right to write: the file changed")
		}
	}

	// A checksum that does not match: nothing installed.
	exe = installed(t, t.TempDir(), "tool")
	f.badSum = true
	code, _, errs, _ = runUpdate(u, f.env(exe, "linux", nil))
	if code != 1 || errs != "tool: tool_1.3.0_linux_amd64.tar.gz does not match its checksum in checksums.txt: not installed\n" {
		t.Errorf("a bad checksum: exit %d, %q", code, errs)
	}
	if b, _ := os.ReadFile(exe); string(b) != "old tool" {
		t.Error("a bad checksum: the file changed")
	}
	f.badSum = false

	// A platform the release does not have: nothing installed.
	if code, _, errs, _ = runUpdate(u, updateEnv{exe: exe, goos: "linux", goarch: "riscv64", github: f.srv.URL, client: f.srv.Client()}); code != 1 ||
		!strings.Contains(errs, "tool_1.3.0_linux_riscv64.tar.gz: 404") {
		t.Errorf("no such platform: exit %d, %q", code, errs)
	}

	// The lookup fails: no such repository; no release yet.
	nobody := u
	nobody.Owner = "nobody"
	if code, _, errs, _ = runUpdate(nobody, f.env(exe, "linux", nil)); code != 1 ||
		errs != "tool: Cannot look up the latest release: "+f.srv.URL+"/nobody/tool/releases/latest: 404 Not Found\n" {
		t.Errorf("no such repository: exit %d, %q", code, errs)
	}
	f.latest = ""
	if code, _, errs, _ = runUpdate(u, f.env(exe, "linux", nil)); code != 1 ||
		errs != "tool: Cannot look up the latest release: "+f.srv.URL+"/me/tool/releases/latest: leads to \""+f.srv.URL+"/me/tool/releases\", not a release\n" {
		t.Errorf("no release yet: exit %d, %q", code, errs)
	}

	// The app's After failing is a failure.
	f.latest = "1.2.0"
	u.After = func(string, bool) error { return errors.New("The plugin could not be updated") }
	var o, e bytes.Buffer
	if code = u.run(f.env(exe, "linux", nil), &o, &e); code != 1 || e.String() != "tool: The plugin could not be updated\n" {
		t.Errorf("After failing: exit %d, %q", code, e.String())
	}
}

// TestNewer: versions as semver orders them.
func TestNewer(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"0.1.10", "0.1.9", true}, {"0.1.9", "0.1.10", false}, {"1.0.1", "1.0.1-beta.1", true},
		{"1.0.1-beta.1", "1.0.0", true}, {"1.0.1-beta.2", "1.0.1-beta.10", false}, {"1.0.1-rc.1", "1.0.1-beta.9", true},
		{"1.2", "1.2.0", false}, {"1.2.0", "1.2.0+dirty", false}, {"1.0.0-beta.1.1", "1.0.0-beta.1", true},
		{"1.0.0-beta", "1.0.0-1", true},
	} {
		if got := newer(c.a, c.b); got != c.want {
			t.Errorf("newer(%s, %s) = %v", c.a, c.b, got)
		}
	}
}

// TestUpdateInstalledBy: the way it was installed, from where the file is.
func TestUpdateInstalledBy(t *testing.T) {
	for _, c := range []struct {
		exe  string
		want installMethod
	}{
		{"/opt/homebrew/Cellar/tool/1.2.0/bin/tool", byHomebrew},
		{"/home/linuxbrew/.linuxbrew/Cellar/tool/1.2.0/bin/tool", byHomebrew},
		{`C:\Users\me\scoop\apps\tool\current\tool.exe`, byScoop},
		{"/home/me/.local/bin/tool", byRelease},
		{"/opt/homebrew/Cellar/toolkit/1.0/bin/toolkit", byRelease},
	} {
		if got := installedBy(c.exe, "tool"); got != c.want {
			t.Errorf("%s: %v", c.exe, got)
		}
	}
	if got := brewBin("/opt/homebrew/Cellar/tool/1.2.0/bin/tool", "tool"); got != "/opt/homebrew/bin/tool" {
		t.Errorf("brewBin %s", got)
	}
}
