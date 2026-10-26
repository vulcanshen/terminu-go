package tdptest

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	terminu "github.com/vulcanshen/terminu-go"
)

const goodMakefile = `# hostu — build / run / test / package（家族共用的 target 在 family.mk）
#
# 跑 ` + "`make`（或 `make help`）" + `列出所有指令。

BINARY := hostu
CHECK := mod-check
include family.mk

##@ hostu 自己的

.PHONY: mod-check
mod-check: ## go mod verify
	go mod verify
`

const goodChangelog = `# Changelog

Each release names the version of terminu design (tdp) it follows.

## Unreleased

- Hosts can be renamed

## v0.2.0 — tdp v0.10.2

- The host list

## v0.1.0

- First release, before tdp wrote the version in the heading
`

// goodRepo is a repository of an app named hostu as tdp project has it, its
// account's: every template copied as it is, the app's name and account in
// place of <app> and vulcanshen.
func goodRepo(t *testing.T, account string) string {
	t.Helper()
	dir := t.TempDir()
	put := func(rel, s string) {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	put("go.mod", "module github.com/"+account+"/hostu\n\ngo 1.26.0\n")
	put("go.sum", "")
	put("LICENSE", "GPL\n")
	put("docs/icon.svg", "<svg/>\n")
	put("cmd/hostu/main.go", "package main\n\nfunc main() {}\n")
	put("Makefile", goodMakefile)
	put("CHANGELOG.md", goodChangelog)
	for src, dst := range map[string]string{"family.mk": "family.mk", "release.yml": ".github/workflows/release.yml",
		"goreleaser.yaml": ".goreleaser.yaml", "install.sh": "install.sh", "uninstall.sh": "uninstall.sh",
		"version.go": "internal/version/version.go", "gitignore": ".gitignore"} {
		b, err := templates.ReadFile("_project/" + src)
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)
		if src != "family.mk" {
			s = strings.ReplaceAll(strings.ReplaceAll(s, "<app>", "hostu"), "vulcanshen", account)
		}
		put(dst, s)
	}
	return dir
}

// edit changes old to new in rel, which has old once.
func edit(t *testing.T, dir, rel, old, new string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(b), old); n != 1 {
		t.Fatalf("%s has %q %d times", rel, old, n)
	}
	if err := os.WriteFile(p, []byte(strings.Replace(string(b), old, new, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
}

func repoErrs(t *testing.T, dir string) []string {
	t.Helper()
	ft := &fakeT{T: t}
	checkRepo(ft, dir, "hostu")
	return ft.errs
}

// TestRepoAsTdpHasIt: a repository with every template as tdp has it, the
// app's name and account in them, passes; so does one made after tdp with an
// account of its own in place of vulcanshen (tdp project.2).
func TestRepoAsTdpHasIt(t *testing.T) {
	for _, account := range []string{"vulcanshen", "someone"} {
		if errs := repoErrs(t, goodRepo(t, account)); len(errs) > 0 {
			t.Errorf("%s: %q", account, errs)
		}
	}
}

// TestRepoWhatAnAppMayChange: what the last column of tdp project.2 lets an
// app change passes: the app's own blocks, what comes under app:, the app's
// own lines after the template's in .gitignore (project-3, 4, 5, 7).
func TestRepoWhatAnAppMayChange(t *testing.T) {
	dir := goodRepo(t, "vulcanshen")
	edit(t, dir, "install.sh", "# --- app's own: checks before the download (none by default) ---\n",
		"# --- app's own: checks before the download (none by default) ---\ncommand -v ssh >/dev/null || { echo \"ssh is needed\"; exit 1; }\n")
	edit(t, dir, "install.sh", `echo "Run '$APP' to start."`, `echo "Run '$APP' and add a host with a."`)
	edit(t, dir, "uninstall.sh", "# --- app's own: anything left for the user to do (none by default) ---\n",
		"# --- app's own: anything left for the user to do (none by default) ---\necho \"~/.ssh/config is yours, kept\"\n")
	edit(t, dir, ".goreleaser.yaml", "    ignore: []\n", "    ignore:\n      - goos: linux\n        goarch: arm64\n")
	edit(t, dir, ".goreleaser.yaml", "\n    description: \"hostu: what it is (terminu family)\"", "\n    description: \"hostu: ssh hosts at hand (terminu family)\"")
	edit(t, dir, ".goreleaser.yaml", "    test: |\n", "    dependencies:\n      - name: openssh\n    test: |\n")
	edit(t, dir, ".gitignore", ".claude/worktrees/\n", ".claude/worktrees/\n/hosts.local\n")
	edit(t, dir, "uninstall.sh", "(tdp E2): <APP>__CONFIG", "(tdp E2): HOSTU__CONFIG")
	if errs := repoErrs(t, dir); len(errs) > 0 {
		t.Errorf("%q", errs)
	}
}

// TestRepoFamilyMk: a family.mk that is not tdp's fails, naming the file,
// with the difference and the template to copy again (project-1).
func TestRepoFamilyMk(t *testing.T) {
	dir := goodRepo(t, "vulcanshen")
	edit(t, dir, "family.mk", ".PHONY: dev\ndev: build ## 編譯再跑 ./$(BINARY)（跟 build 同一個執行檔；讀真正的設定與資料）\n\t./$(BINARY)\n\n", "")
	errs := repoErrs(t, dir)
	if len(errs) != 1 {
		t.Fatalf("%d errors: %q", len(errs), errs)
	}
	for _, want := range []string{"family.mk is not tdp " + tdpVersion + "'s project/family.mk", "project-1",
		"copy tdp " + tdpVersion + "'s project/family.mk to family.mk again", "@@ project/family.mk line ", "- .PHONY: dev", "- dev: build", "\tgo run $(PKG)"} {
		if !strings.Contains(errs[0], want) {
			t.Errorf("no %q in:\n%s", want, errs[0])
		}
	}
}

// TestRepoChangedTemplates: a change no column allows fails, naming the
// file; a missing one says what to copy (project-1 to project-7).
func TestRepoChangedTemplates(t *testing.T) {
	for _, c := range []struct{ file, old, new string }{
		{".github/workflows/release.yml", "      - run: go test -race ./...\n", "      - run: go test ./...\n"},
		{".goreleaser.yaml", "      - -trimpath\n", ""},
		{".goreleaser.yaml", "project_name: hostu", "project_name: hosts"},
		{"install.sh", `REPO="vulcanshen/$APP"`, `REPO="me/$APP"`},
		{"uninstall.sh", "# --- end of the app's own ---\n\nrm \"$FOUND\"", "# --- end of the app's own ---\n\nrm -f \"$FOUND\""},
		{"internal/version/version.go", `var Version = "dev"`, `var Version = "0.0.0"`},
		{".gitignore", ".idea/\n", ""},
	} {
		dir := goodRepo(t, "vulcanshen")
		edit(t, dir, c.file, c.old, c.new)
		if errs := repoErrs(t, dir); len(errs) != 1 || !strings.Contains(errs[0], c.file+" is not tdp") {
			t.Errorf("%s with %q: %q", c.file, c.new, errs)
		}
		if err := os.Remove(filepath.Join(dir, filepath.FromSlash(c.file))); err != nil {
			t.Fatal(err)
		}
		if errs := repoErrs(t, dir); len(errs) != 1 || !strings.Contains(errs[0], "there is no "+c.file) || !strings.Contains(errs[0], "copy tdp") {
			t.Errorf("no %s: %q", c.file, errs)
		}
	}
}

// openWindowsIn makes dir's app one with a Windows build: install.sh says so,
// .goreleaser.yaml's Windows lines are uncommented, the matrix has
// windows-latest, the PowerShell scripts are there (project-8, 9, 10, 19).
func openWindowsIn(t *testing.T, dir string, which ...string) {
	t.Helper()
	all := len(which) == 0
	does := func(w string) bool {
		for _, x := range which {
			if x == w {
				return true
			}
		}
		return all
	}
	if does("install") {
		edit(t, dir, "install.sh", `WINDOWS="no"`, `WINDOWS="yes"`)
	}
	if does("goreleaser") {
		b, err := os.ReadFile(filepath.Join(dir, ".goreleaser.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)
		s = strings.Replace(s, "      # - windows   # Windows", "      - windows", 1)
		s = strings.Replace(s, "    # format_overrides:   # Windows\n    #   - goos: windows\n    #     formats:\n    #       - zip\n",
			"    format_overrides:   # Windows\n      - goos: windows\n        formats:\n          - zip\n", 1)
		head, tail, _ := strings.Cut(s, "# scoops:   # Windows\n")
		tail = regexp.MustCompile(`(?m)^# ?`).ReplaceAllString(tail, "")
		tail = strings.Replace(tail, `"hostu: what it is (terminu family)"`, `"hostu: ssh hosts at hand (terminu family)"`, 1)
		if err := os.WriteFile(filepath.Join(dir, ".goreleaser.yaml"), []byte(head+"scoops:\n"+tail), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if does("matrix") {
		edit(t, dir, ".github/workflows/release.yml", "os: [ubuntu-latest, macos-latest]", "os: [ubuntu-latest, macos-latest, windows-latest]")
	}
	if does("ps1") {
		for _, f := range []string{"install.ps1", "uninstall.ps1"} {
			if err := os.WriteFile(filepath.Join(dir, f), []byte("# "+f+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// TestRepoWindows: an app with a Windows build has all of it; with
// WINDOWS="yes" in install.sh, each missing piece fails (project-8, 9, 10,
// 19); without, an uncommented Windows line is a change.
func TestRepoWindows(t *testing.T) {
	dir := goodRepo(t, "vulcanshen")
	openWindowsIn(t, dir)
	if errs := repoErrs(t, dir); len(errs) > 0 {
		t.Errorf("a Windows build: %q", errs)
	}
	for _, c := range []struct{ missing, file string }{
		{"goreleaser", ".goreleaser.yaml is not tdp"}, {"matrix", "release.yml is not tdp"}, {"ps1", "no install.ps1"},
	} {
		dir := goodRepo(t, "vulcanshen")
		var rest []string
		for _, w := range []string{"install", "goreleaser", "matrix", "ps1"} {
			if w != c.missing {
				rest = append(rest, w)
			}
		}
		openWindowsIn(t, dir, rest...)
		errs := repoErrs(t, dir)
		if len(errs) == 0 || !strings.Contains(errs[0], c.file) {
			t.Errorf("WINDOWS=yes without %s: %q", c.missing, errs)
		}
	}
	dir = goodRepo(t, "vulcanshen")
	openWindowsIn(t, dir, "goreleaser")
	if errs := repoErrs(t, dir); len(errs) != 1 || !strings.Contains(errs[0], ".goreleaser.yaml is not tdp") {
		t.Errorf("Windows lines open, WINDOWS=no: %q", errs)
	}
}

// TestRepoFiles: the files every app has (tdp project.3): the main package,
// go.mod's module github.com/<account>/<app>, go.sum, LICENSE, docs/icon.svg.
func TestRepoFiles(t *testing.T) {
	dir := goodRepo(t, "vulcanshen")
	for _, f := range []string{"cmd/hostu/main.go", "go.sum", "LICENSE", "docs/icon.svg"} {
		if err := os.Remove(filepath.Join(dir, filepath.FromSlash(f))); err != nil {
			t.Fatal(err)
		}
	}
	errs := strings.Join(repoErrs(t, dir), "\n")
	for _, want := range []string{"no cmd/hostu/main.go (tdp project.3, project-12", "no go.sum (tdp project.3, project-15)",
		"no LICENSE (tdp project.3, project-16)", "no docs/icon.svg (tdp project.3, project-18"} {
		if !strings.Contains(errs, want) {
			t.Errorf("no %q in:\n%s", want, errs)
		}
	}
	dir = goodRepo(t, "vulcanshen")
	edit(t, dir, "go.mod", "module github.com/vulcanshen/hostu", "module example.com/hostu")
	if errs := repoErrs(t, dir); len(errs) != 1 || !strings.Contains(errs[0], "project-13") {
		t.Errorf("another module: %q", errs)
	}
}

// TestRepoLibModule: the lib's own module is not an app's repository (tdp
// E8): nothing in it is checked.
func TestRepoLibModule(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module "+libModule+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if errs := repoErrs(t, dir); len(errs) > 0 {
		t.Errorf("%q", errs)
	}
}

// TestRepoMakefile: tdp project.5: the first lines, variables before include
// family.mk, BINARY, the app's own section last, no target of family.mk's
// (project-23 to 27).
func TestRepoMakefile(t *testing.T) {
	for _, c := range []struct{ old, new, item string }{
		{"（家族共用的 target 在 family.mk）", "", "project-26"},
		{"CHECK := mod-check\n", "CHECK := mod-check\nall: build\n", "project-24"},
		{"BINARY := hostu", "BINARY := hosts", "project-25"},
		{"include family.mk\n", "include family.mk\nGOFMT_DIRS := cmd\n", "project-23"},
		{"##@ hostu 自己的", "##@ mine", "project-23"},
		{"mod-check: ## go mod verify", "test: ## go mod verify", "project-27"},
		{"\tgo mod verify\n", "\tgo mod verify\n\n##@ more\n", "project-23"},
	} {
		dir := goodRepo(t, "vulcanshen")
		edit(t, dir, "Makefile", c.old, c.new)
		errs := strings.Join(repoErrs(t, dir), "\n")
		if !strings.Contains(errs, c.item) || !strings.Contains(errs, "Makefile") {
			t.Errorf("%q to %q: want %s, got %q", c.old, c.new, c.item, errs)
		}
	}
}

// TestRepoChangelog: tdp project.6: the first lines, ## Unreleased alone, a
// heading naming a tdp written ## vX.Y.Z — tdp vA.B.C; an older heading
// without the tdp passes (project-28 to 30).
func TestRepoChangelog(t *testing.T) {
	for _, c := range []struct{ old, new, item string }{
		{"# Changelog\n", "# Change log\n", "project-28"},
		{"(tdp) it follows.", "(tdp) it implements.", "project-28"},
		{"## Unreleased\n", "## Unreleased — tdp v0.10.2\n", "project-30"},
		{"## v0.2.0 — tdp v0.10.2", "## v0.2.0 - tdp v0.10.2", "project-29"},
		{"## v0.2.0 — tdp v0.10.2", "## 0.2.0 — tdp v0.10.2", "project-29"},
	} {
		dir := goodRepo(t, "vulcanshen")
		edit(t, dir, "CHANGELOG.md", c.old, c.new)
		errs := repoErrs(t, dir)
		if len(errs) != 1 || !strings.Contains(errs[0], c.item) || !strings.Contains(errs[0], "CHANGELOG.md line") {
			t.Errorf("%q to %q: want %s, got %q", c.old, c.new, c.item, errs)
		}
	}
}

// TestCheckReadsTheRepo: Check reads the repository of the test it runs in,
// with no call of the app's own: run in an app whose family.mk lacks a
// line, it fails there. The test runs itself again, in that app's cmd/.
func TestCheckReadsTheRepo(t *testing.T) {
	if dir := os.Getenv("TDPTEST_REPO_AT"); dir != "" {
		t.Chdir(dir)
		Check(t, App{Name: "hostu", New: func() *terminu.App { return newTiny().App }})
		return
	}
	dir := goodRepo(t, "vulcanshen")
	edit(t, dir, "family.mk", ".PHONY: dev\n", "")
	cmd := exec.Command(os.Args[0], "-test.run=^TestCheckReadsTheRepo$", "-test.short", "-test.count=1")
	cmd.Env = append(os.Environ(), "TDPTEST_REPO_AT="+filepath.Join(dir, "cmd", "hostu"))
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "--- FAIL: TestCheckReadsTheRepo/repo") ||
		!strings.Contains(string(out), "family.mk is not tdp") || !strings.Contains(string(out), "- .PHONY: dev") {
		t.Errorf("Check did not fail on the repository: %v\n%s", err, out)
	}
}

// TestRepoRoot: the repository is the go.mod at or above the test's
// directory.
func TestRepoRoot(t *testing.T) {
	dir := goodRepo(t, "vulcanshen")
	t.Chdir(filepath.Join(dir, "cmd", "hostu"))
	if got, _ := filepath.EvalSymlinks(repoRoot()); got != mustEval(t, dir) {
		t.Errorf("root %q, want %q", got, dir)
	}
}

func mustEval(t *testing.T, p string) string {
	t.Helper()
	q, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return q
}

// TestTemplatesAreTdps: _project is tdp's project/ templates at the version
// the lib implements, byte for byte (tdp project.2), when the spec's
// repository is on this machine.
func TestTemplatesAreTdps(t *testing.T) {
	home, _ := os.UserHomeDir()
	spec := filepath.Join(home, "Documents", "sideproj", "terminu")
	if _, err := os.Stat(filepath.Join(spec, ".git")); err != nil {
		t.Skip("no tdp repository here")
	}
	entries, err := templates.ReadDir("_project")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		want, err := exec.Command("git", "-C", spec, "show", tdpVersion+":project/"+e.Name()).Output()
		if err != nil {
			t.Errorf("%s: %v", e.Name(), err)
			continue
		}
		if got, _ := templates.ReadFile("_project/" + e.Name()); string(got) != string(want) {
			t.Errorf("_project/%s is not tdp %s's project/%s", e.Name(), tdpVersion, e.Name())
		}
	}
	out, err := exec.Command("git", "-C", spec, "ls-tree", "--name-only", tdpVersion, "project/").Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range strings.Fields(string(out)) {
		if name := strings.TrimPrefix(f, "project/"); !strings.HasPrefix(name, "README") {
			if _, err := templates.ReadFile("_project/" + name); err != nil {
				t.Errorf("tdp %s has project/%s, _project does not", tdpVersion, name)
			}
		}
	}
}

// TestTdpVersionIsTheChangelogs: tdpVersion is the tdp the lib's CHANGELOG
// names at its top.
func TestTdpVersionIsTheChangelogs(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^## .* — tdp (v\d+\.\d+\.\d+)$`).FindStringSubmatch(string(b))
	if m == nil || m[1] != tdpVersion {
		t.Errorf("CHANGELOG's top names %v, tdpVersion is %s", m, tdpVersion)
	}
}
