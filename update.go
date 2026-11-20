package terminu

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"cmp"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// SelfUpdate is <app> update (tdp E1): it brings the app to its latest
// release, never a pre-release, the way it was installed. The app's command
// line runs it: os.Exit(terminu.SelfUpdate{...}.Run(os.Stdout, os.Stderr)).
//
//   - Already the latest: "<app> 1.2.0 is the latest release.", exit 0.
//   - Installed with Homebrew (the executable under Cellar/<app>): it runs
//     brew upgrade <Tap>/<App>, brew's output passed through.
//   - Installed with the install script or downloaded from a release: it
//     downloads the release for this OS and arch, holds it to the release's
//     checksums.txt and puts its executable in place of the running one.
//   - Installed with Scoop (under scoop/apps/<app>; Windows, where a running
//     file cannot be replaced): it prints the scoop update <app> to run.
//   - Built from source (Version "dev" or ""): it says to update it there,
//     exit 1, nothing looked up.
//   - No right to write where the executable is: it says so and what to do,
//     exit 1, nothing downloaded. It never asks for more rights.
//
// Progress goes to stdout, errors to stderr after "<app>: ", a line each;
// exit 0 done, 1 not (tdp E1). The release is GitHub's,
// github.com/<Owner>/<App>, its files named as the family's goreleaser
// names them: <app>_<version>_<os>_<arch>.tar.gz (.zip on Windows) with the
// executable <app> (<app>.exe) in it, and checksums.txt.
type SelfUpdate struct {
	// App is the app's name: its executable, its release files' prefix, its
	// Homebrew formula and its Scoop app.
	App string
	// Version is the running version, without its v ("1.2.0"); "dev" or ""
	// for a build from source.
	Version string
	// Owner is the GitHub owner of the app's repository ("vulcanshen").
	Owner string
	// Tap is the Homebrew tap the app is in ("vulcanshen/tap", tdp E6).
	Tap string
	// Source, when set, is how a build from source updates, written after
	// "update it there" ("git pull && make install").
	Source string
	// Then, when set, is a command the user runs after updating by hand (a
	// build from source, Scoop), written after the rest ("ccdrawu install
	// claude").
	Then string
	// After, when set, runs once the app is at its latest release, updated
	// now or already (tdp E1: what the app does after updating is its own):
	// exe is the executable that is the latest now, the new one after an
	// update (Homebrew's link in its bin). Its error goes to stderr, exit 1.
	After func(exe string, updated bool) error
}

// updateEnv is where an update runs: the executable, the platform, where
// releases are looked up and downloaded, and how brew runs. Tests give their
// own.
type updateEnv struct {
	exe          string
	goos, goarch string
	api, files   string // https://api.github.com, https://github.com
	client       *http.Client
	brew         func(args ...string) *exec.Cmd
}

// Run does the update, its words on stdout and stderr, and gives the exit
// status.
func (u SelfUpdate) Run(stdout, stderr io.Writer) int {
	env := updateEnv{goos: runtime.GOOS, goarch: runtime.GOARCH, api: "https://api.github.com", files: "https://github.com",
		client: &http.Client{Timeout: 2 * time.Minute},
		brew:   func(args ...string) *exec.Cmd { return exec.Command("brew", args...) }}
	return u.run(env, stdout, stderr)
}

func (u SelfUpdate) run(env updateEnv, stdout, stderr io.Writer) int {
	fail := func(lines ...string) int {
		for _, l := range lines {
			fmt.Fprintln(stderr, u.App+": "+l)
		}
		return 1
	}
	cur := strings.TrimPrefix(u.Version, "v")
	if cur == "" || cur == "dev" {
		s := "This " + u.App + " is built from source: update it there"
		if u.Source != "" {
			s += " (`" + u.Source + "`)"
		}
		if u.Then != "" {
			s += ", then run `" + u.Then + "`"
		}
		return fail(s)
	}
	exe := env.exe
	if exe == "" {
		var err error
		if exe, err = os.Executable(); err == nil {
			exe, err = filepath.EvalSymlinks(exe)
		}
		if err != nil {
			return fail("Cannot find its own executable: " + err.Error())
		}
	}
	latest, err := latestRelease(env, u.Owner, u.App)
	if err != nil {
		return fail("Cannot look up the latest release: " + err.Error())
	}
	after := func(exe string, updated bool) int {
		if u.After != nil {
			if err := u.After(exe, updated); err != nil {
				return fail(err.Error())
			}
		}
		return 0
	}
	switch {
	case cur == latest:
		fmt.Fprintf(stdout, "%s %s is the latest release.\n", u.App, cur)
		return after(exe, false)
	case !newer(latest, cur):
		// TODO(tdp): a pre-release newer than the latest release is not in tdp yet.
		fmt.Fprintf(stdout, "%s %s is newer than the latest release, %s.\n", u.App, cur, latest)
		return after(exe, false)
	}
	switch installedBy(exe, u.App) {
	case byHomebrew:
		formula := u.App
		if u.Tap != "" {
			formula = u.Tap + "/" + u.App
		}
		fmt.Fprintf(stdout, "Updating %s %s to %s with Homebrew…\n", u.App, cur, latest)
		brew := env.brew("upgrade", formula)
		brew.Stdout, brew.Stderr = stdout, stderr
		if err := brew.Run(); err != nil {
			return fail("brew upgrade " + formula + ": " + err.Error())
		}
		return after(brewBin(exe, u.App), true)
	case byScoop:
		// Scoop does not replace an app while it runs, and this is it.
		fmt.Fprintf(stdout, "%s %s is out. Scoop installed this one: close %s and run\n\n  scoop update %s\n", u.App, latest, u.App, u.App)
		if u.Then != "" {
			fmt.Fprintln(stdout, "  "+u.Then)
		}
		return 0 // TODO(tdp): Scoop's exit status is not in tdp yet; it told how, so 0.
	}
	fmt.Fprintf(stdout, "Updating %s %s to %s…\n", u.App, cur, latest)
	if err := replaceExe(env, u.Owner, u.App, exe, latest); err != nil {
		if errors.Is(err, fs.ErrPermission) {
			return fail(err.Error(), "Run the install script again, or `"+u.App+" update` as the folder's owner")
		}
		return fail(err.Error())
	}
	fmt.Fprintf(stdout, "%s %s is in %s.\n", u.App, latest, exe)
	return after(exe, true)
}

// A release's archive is a few MB; anything past this is not one.
const maxRelease = 64 << 20

func releaseGet(env updateEnv, url string) ([]byte, error) {
	resp, err := env.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", url, resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxRelease))
}

// latestRelease is the version of owner/app's latest release, without its
// v. GitHub's latest is never a pre-release.
func latestRelease(env updateEnv, owner, app string) (string, error) {
	url := env.api + "/repos/" + owner + "/" + app + "/releases/latest"
	body, err := releaseGet(env, url)
	if err != nil {
		return "", err
	}
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &release); err != nil || release.TagName == "" {
		return "", fmt.Errorf("%s: no release tag in its answer", url)
	}
	return strings.TrimPrefix(release.TagName, "v"), nil
}

// newer reports whether version a is newer than b, as semver orders them:
// part by numeric part, 0.1.10 newer than 0.1.9; a release newer than its
// pre-releases, 1.0.1 than 1.0.1-beta.1, and those newer than 1.0.0.
func newer(a, b string) bool {
	na, pa := splitVersion(a)
	nb, pb := splitVersion(b)
	if c := compareParts(strings.Split(na, "."), strings.Split(nb, "."), true); c != 0 {
		return c > 0
	}
	if pa == "" || pb == "" {
		return pa == "" && pb != ""
	}
	return compareParts(strings.Split(pa, "."), strings.Split(pb, "."), false) > 0
}

// splitVersion takes a version apart into its numbers and its pre-release,
// the build metadata left out: 1.0.1-beta.1+dirty is 1.0.1 and beta.1.
func splitVersion(v string) (numbers, pre string) {
	v, _, _ = strings.Cut(v, "+")
	numbers, pre, _ = strings.Cut(v, "-")
	return numbers, pre
}

// compareParts compares dotted parts in turn: numbers by value and before
// words, words by their bytes. Numbers (the version's) count a missing part
// as 0; a pre-release with more parts, the rest the same, is newer.
func compareParts(a, b []string, numbers bool) int {
	n := min(len(a), len(b))
	if numbers {
		n = max(len(a), len(b))
	}
	at := func(p []string, i int) string {
		if i < len(p) {
			return p[i]
		}
		return "0"
	}
	for i := range n {
		x, errX := strconv.Atoi(at(a, i))
		y, errY := strconv.Atoi(at(b, i))
		c := 0
		switch {
		case errX == nil && errY == nil:
			c = cmp.Compare(x, y)
		case errX == nil:
			c = -1
		case errY == nil:
			c = 1
		default:
			c = strings.Compare(at(a, i), at(b, i))
		}
		if c != 0 {
			return c
		}
	}
	if numbers {
		return 0
	}
	return cmp.Compare(len(a), len(b))
}

type installMethod int

const (
	byRelease  installMethod = iota // the install script, or a download: it replaces its own executable
	byHomebrew                      // under Cellar/<app>
	byScoop                         // under scoop/apps/<app>
)

// installedBy tells how app was installed from where its executable really
// is, symlinks followed.
func installedBy(exe, app string) installMethod {
	p := strings.ToLower(strings.ReplaceAll(exe, `\`, "/"))
	switch {
	case strings.Contains(p, "/cellar/"+strings.ToLower(app)+"/"):
		return byHomebrew
	case strings.Contains(p, "/scoop/apps/"+strings.ToLower(app)+"/"):
		return byScoop
	}
	return byRelease
}

// brewBin is the app Homebrew links into its prefix's bin, for an executable
// under its Cellar: after an upgrade that is the new one, where the Cellar
// path is the old version's.
func brewBin(exe, app string) string {
	if i := strings.Index(exe, "/Cellar/"); i >= 0 {
		return exe[:i] + "/bin/" + app
	}
	return exe
}

// releaseArchive is the name of a release's archive for a platform, as the
// family's goreleaser names it: a zip for Windows, a tar.gz for the rest.
func releaseArchive(app, version, goos, goarch string) string {
	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("%s_%s_%s_%s.%s", app, version, goos, goarch, ext)
}

// replaceExe puts the executable of app's release version in place of exe:
// its archive downloaded and held to the release's checksums.txt, the
// executable written beside exe, then renamed over it. Windows does not
// overwrite a running executable but renames one, so there exe moves to
// exe.old first, which the next update removes. A folder it cannot write
// in fails before anything downloads.
func replaceExe(env updateEnv, owner, app, exe, version string) error {
	dir := filepath.Dir(exe)
	tmp, err := os.CreateTemp(dir, "."+app+"-update-*")
	if err != nil {
		var pe *fs.PathError
		if errors.As(err, &pe) {
			err = pe.Err
		}
		return fmt.Errorf("Cannot write in %s: %w", dir, err)
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	file := func(name string) string {
		return env.files + "/" + owner + "/" + app + "/releases/download/v" + version + "/" + name
	}
	name := releaseArchive(app, version, env.goos, env.goarch)
	sums, err := releaseGet(env, file("checksums.txt"))
	if err != nil {
		return err
	}
	archive, err := releaseGet(env, file(name))
	if err != nil {
		return err
	}
	if err := checkSum(archive, name, sums); err != nil {
		return err
	}
	bin := app
	if env.goos == "windows" {
		bin += ".exe"
	}
	b, err := extractExe(archive, bin, env.goos == "windows")
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if _, err := tmp.Write(b); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		return err
	}
	if env.goos != "windows" {
		return os.Rename(tmp.Name(), exe)
	}
	old := exe + ".old"
	_ = os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), exe); err != nil {
		_ = os.Rename(old, exe)
		return err
	}
	return nil
}

// checkSum holds archive to its line in checksums.txt, "<sha256>  <name>".
func checkSum(archive []byte, name string, sums []byte) error {
	for line := range strings.Lines(string(sums)) {
		fields := strings.Fields(line)
		if len(fields) != 2 || fields[1] != name {
			continue
		}
		sum := sha256.Sum256(archive)
		if hex.EncodeToString(sum[:]) != strings.ToLower(fields[0]) {
			return fmt.Errorf("%s does not match its checksum in checksums.txt: not installed", name)
		}
		return nil
	}
	return fmt.Errorf("checksums.txt has no line for %s: not installed", name)
}

// extractExe is the executable bin out of a release archive, a zip or a
// tar.gz.
func extractExe(archive []byte, bin string, isZip bool) ([]byte, error) {
	if isZip {
		zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
		if err != nil {
			return nil, err
		}
		for _, f := range zr.File {
			if path.Base(f.Name) != bin {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(io.LimitReader(rc, maxRelease))
		}
		return nil, errors.New("no " + bin + " in it")
	}
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil, errors.New("no " + bin + " in it")
		}
		if err != nil {
			return nil, err
		}
		if h.Typeflag == tar.TypeReg && path.Base(h.Name) == bin {
			return io.ReadAll(io.LimitReader(tr, maxRelease))
		}
	}
}
