package tdptest

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// tdpVersion is the tdp this lib implements; _project holds its project/
// templates as they are (tdp project.2).
const tdpVersion = "v1.0.1"

//go:embed _project
var templates embed.FS

// libModule is the lib's own module: tdp E8 leaves project/ out of the lib
// and its examples.
const libModule = "github.com/vulcanshen/terminu-go"

// repoRoot is the directory of the go.mod at or above the test's working
// directory: the app's repository.
func repoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		up := filepath.Dir(dir)
		if up == dir {
			return ""
		}
		dir = up
	}
}

// checkRepo checks the app's repository at root against tdp project: the
// templates of project.2 as tdp has them, but for what an app may change in
// each, and the files, Makefile and CHANGELOG checked by reading them alone.
func checkRepo(t testing.TB, root, name string) {
	t.Helper()
	if root == "" {
		t.Errorf("tdptest: no go.mod at or above the test's directory: the repository is not checked (tdp project)")
		return
	}
	r := &repo{t: t, root: root, name: name, account: "vulcanshen"}
	module := r.module()
	if module == libModule {
		return
	}
	r.files()
	r.templates()
	r.makefile()
	r.changelog()
}

// repo is an app's repository as checkRepo reads it.
type repo struct {
	t          testing.TB
	root, name string
	// account owns the repository, vulcanshen in the family; an app made
	// after tdp has its own in place of vulcanshen (tdp project.2).
	account string
}

func (r *repo) read(rel string) (string, bool) {
	b, err := os.ReadFile(filepath.Join(r.root, filepath.FromSlash(rel)))
	if err != nil {
		return "", false
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n"), true
}

func (r *repo) has(rel string) bool {
	_, err := os.Stat(filepath.Join(r.root, filepath.FromSlash(rel)))
	return err == nil
}

// module reads go.mod's module path; it is github.com/<account>/<app>
// (tdp project.3, project-13).
func (r *repo) module() string {
	s, ok := r.read("go.mod")
	if !ok {
		r.t.Errorf("tdptest: there is no go.mod (tdp project.3, project-15)")
		return ""
	}
	m := regexp.MustCompile(`(?m)^module\s+(\S+)`).FindStringSubmatch(s)
	if m == nil {
		r.t.Errorf("tdptest: go.mod names no module (tdp project.3, project-13)")
		return ""
	}
	if m[1] == libModule {
		return m[1]
	}
	own := regexp.MustCompile(`^github\.com/([^/]+)/` + regexp.QuoteMeta(r.name) + `$`).FindStringSubmatch(m[1])
	if own == nil {
		r.t.Errorf("tdptest: go.mod's module is %s, not github.com/<account>/%s: go install github.com/<account>/%s/cmd/%s@latest has to work (tdp project.3, project-13)",
			m[1], r.name, r.name, r.name)
		return m[1]
	}
	r.account = own[1]
	return m[1]
}

// files are the ones tdp project.3 puts in every app.
func (r *repo) files() {
	for _, f := range []struct{ path, item string }{
		{"cmd/" + r.name + "/main.go", "project-12: the main package"},
		{"go.sum", "project-15"},
		{"LICENSE", "project-16"},
		{"docs/icon.svg", "project-18: the family mark the splash draws, its master copy"},
	} {
		if !r.has(f.path) {
			r.t.Errorf("tdptest: there is no %s (tdp project.3, %s)", f.path, f.item)
		}
	}
}

// line is a line of a file and where it was, as the diff shows it.
type line struct {
	text string
	n    int
}

func lines(s string) []line {
	s = strings.TrimSuffix(s, "\n")
	var out []line
	for i, l := range strings.Split(s, "\n") {
		out = append(out, line{l, i + 1})
	}
	return out
}

// template is one of tdp project.2's templates and what an app may change in
// it, as its last column says.
type template struct {
	src, dst, item string
	may            string // what may differ, for the message
	// named: the app's name and account are written as <app> and
	// vulcanshen before comparing.
	named bool
	// fix makes the app's lines and the template's comparable: what the app
	// may change is made the same on both sides.
	fix func(app, tdp []line) ([]line, []line)
}

func (r *repo) templates() {
	install, _ := r.read("install.sh")
	windows := regexp.MustCompile(`(?m)^WINDOWS="yes"$`).MatchString(install)
	known := install != ""
	for _, tm := range []template{
		{src: "family.mk", dst: "family.mk", item: "project-1", may: "nothing in it may differ"},
		{src: "release.yml", dst: ".github/workflows/release.yml", item: "project-2, project-8",
			may: "only windows-latest added to the test matrix may differ, for an app with a Windows build",
			fix: func(app, tdp []line) ([]line, []line) {
				switch {
				case !known: // either
					return matrix(app, withWindows, noWindows), tdp
				case windows:
					return app, matrix(tdp, noWindows, withWindows)
				}
				return app, tdp
			}},
		{src: "goreleaser.yaml", dst: ".goreleaser.yaml", item: "project-3, project-9", named: true,
			may: "only <app>, the value of the key right below each comment marked app: (or the key the comment names, added there when the template has none) and, for an app with a Windows build (WINDOWS=\"yes\" in install.sh), the leading \"# \" of the lines marked Windows and of the commented lines of their blocks may differ; the comments and the Windows marks stay as they are",
			fix: func(app, tdp []line) ([]line, []line) {
				if !known {
					windows = sameWindows(app)
				}
				return appRegions(app), appRegions(openWindows(tdp, windows))
			}},
		{src: "install.sh", dst: "install.sh", item: "project-4, project-10", named: true,
			may: `only <app>, WINDOWS ("yes" for an app with a Windows build) and the blocks marked app's own may differ`,
			fix: func(app, tdp []line) ([]line, []line) {
				for i, l := range app {
					if l.text == `WINDOWS="yes"` {
						app[i].text = `WINDOWS="no"`
					}
				}
				return ownBlocks(app), ownBlocks(tdp)
			}},
		{src: "uninstall.sh", dst: "uninstall.sh", item: "project-5", named: true,
			may: "only <app> and the blocks marked app's own may differ; the <APP> in a comment stays as it is",
			fix: func(app, tdp []line) ([]line, []line) { return ownBlocks(app), ownBlocks(tdp) }},
		{src: "version.go", dst: "internal/version/version.go", item: "project-6", named: true, may: "only <app> may differ"},
		{src: "gitignore", dst: ".gitignore", item: "project-7", named: true,
			may: "only <app> may differ, the app's own lines going after the last one",
			fix: func(app, tdp []line) ([]line, []line) {
				if len(app) > len(tdp) {
					app = app[:len(tdp)]
				}
				return app, tdp
			}},
	} {
		r.template(tm)
	}
	if windows {
		for _, f := range []string{"install.ps1", "uninstall.ps1"} {
			if !r.has(f) {
				r.t.Errorf("tdptest: install.sh says the app has a Windows build (WINDOWS=\"yes\") but there is no %s (tdp project.3, project-19)", f)
			}
		}
	}
}

func (r *repo) template(tm template) {
	b, err := templates.ReadFile("_project/" + tm.src)
	if err != nil {
		panic(err)
	}
	again := fmt.Sprintf("copy tdp %s's project/%s to %s again%s", tdpVersion, tm.src, tm.dst, copyAt(tm.src))
	s, ok := r.read(tm.dst)
	if !ok {
		r.t.Errorf("tdptest: there is no %s (tdp project.2, %s): %s", tm.dst, tm.item, again)
		return
	}
	if tm.named {
		s = r.unname(s)
	}
	app, tdp := lines(s), lines(strings.ReplaceAll(string(b), "\r\n", "\n"))
	if tm.fix != nil {
		app, tdp = tm.fix(app, tdp)
	}
	if d := diff(tdp, app, "project/"+tm.src, tm.dst); d != "" {
		named := ""
		if tm.named {
			named = ", the app's name written as <app>"
		}
		r.t.Errorf("tdptest: %s is not tdp %s's project/%s (tdp project.2, %s): %s; %s.\n%s(- tdp's, + the app's%s)",
			tm.dst, tdpVersion, tm.src, tm.item, again, tm.may, d, named)
	}
}

// unname writes the app's name as <app> and its account as vulcanshen, as in
// the templates. The <APP> in uninstall.sh's comment is not <app>: it stays
// as it is (tdp project.2).
func (r *repo) unname(s string) string {
	if r.account != "vulcanshen" {
		s = regexp.MustCompile(`\b`+regexp.QuoteMeta(r.account)+`\b`).ReplaceAllLiteralString(s, "vulcanshen")
	}
	return regexp.MustCompile(`\b`+regexp.QuoteMeta(r.name)+`\b`).ReplaceAllLiteralString(s, "<app>")
}

// copyAt is where the lib keeps its copy of a template, when it is on the
// disk (the module cache, or the lib's repository with go.work).
func copyAt(src string) string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	p := filepath.Join(filepath.Dir(file), "_project", src)
	if _, err := os.Stat(p); err != nil {
		return ""
	}
	return " (the lib's copy of it: " + p + ")"
}

// The test matrix of release.yml, as the template has it and with
// windows-latest added (tdp project.2).
const (
	noWindows   = "os: [ubuntu-latest, macos-latest]"
	withWindows = "os: [ubuntu-latest, macos-latest, windows-latest]"
)

// matrix writes the test matrix from as to in a copy of ls.
func matrix(ls []line, from, to string) []line {
	out := append([]line(nil), ls...)
	for i, l := range out {
		if strings.TrimSpace(l.text) == from {
			out[i].text = strings.Replace(l.text, from, to, 1)
		}
	}
	return out
}

// ownBlocks leaves out what is between "# --- app's own: …" and "# --- end
// of the app's own ---": the app's (install.sh, uninstall.sh).
func ownBlocks(ls []line) []line {
	var out []line
	in := false
	for _, l := range ls {
		t := strings.TrimSpace(l.text)
		switch {
		case strings.HasPrefix(t, "# --- app's own:"):
			in = true
		case t == "# --- end of the app's own ---":
			in = false
		case in:
			continue
		}
		out = append(out, l)
	}
	return out
}

var (
	// appMark is a comment marked "app:" (.goreleaser.yaml); one that names
	// its key first ("app: dependencies, …") is for a key the template may
	// not have yet.
	appMark = regexp.MustCompile(`^# app: (?:([a-z_]+),)?`)
	yamlKey = regexp.MustCompile(`^(\s*)([A-Za-z_][A-Za-z0-9_]*):`)
)

// appRegions leaves out what each comment marked "app:" lets the app write
// (.goreleaser.yaml, tdp project.2): the value of the key right below the
// comment, a list or a whole block included, the key itself kept; or, for a
// comment naming a key, that key, which the app adds there when the template
// has none. The comment, all its lines, and any other key stay.
func appRegions(ls []line) []line {
	var out []line
	for i := 0; i < len(ls); i++ {
		out = append(out, ls[i])
		m := appMark.FindStringSubmatch(strings.TrimSpace(ls[i].text))
		if m == nil {
			continue
		}
		for i+1 < len(ls) && strings.HasPrefix(strings.TrimSpace(ls[i+1].text), "#") {
			i++
			out = append(out, ls[i])
		}
		if i+1 == len(ls) {
			break
		}
		k := yamlKey.FindStringSubmatch(ls[i+1].text)
		if k == nil || m[1] != "" && k[2] != m[1] {
			continue // not the app's: test: under "app: dependencies, …"
		}
		i++
		if m[1] == "" {
			out = append(out, line{k[1] + k[2] + ":", ls[i].n})
		}
		i = valueEnd(ls, i, len(k[1]))
	}
	return out
}

// valueEnd is the last line of the value of the key at ls[i], indented in:
// the lines below indented more, a list at the key's own indent, and the
// blank lines between them.
func valueEnd(ls []line, i, in int) int {
	end := i
	for j := i + 1; j < len(ls); j++ {
		t := strings.TrimSpace(ls[j].text)
		switch {
		case t == "":
		case indent(ls[j].text) > in || indent(ls[j].text) == in && strings.HasPrefix(t, "- "):
			end = j
		default:
			return end
		}
	}
	return end
}

func indent(s string) int { return len(s) - len(strings.TrimLeft(s, " ")) }

var windowsMark = regexp.MustCompile(`\s+# Windows$`)

// openWindows uncomments the template's lines marked Windows and the
// commented lines of the block right under each, for an app with a Windows
// build: only the leading "# " goes, the mark stays (tdp project.2).
func openWindows(ls []line, windows bool) []line {
	if !windows {
		return ls
	}
	out := append([]line(nil), ls...)
	for i := 0; i < len(out); i++ {
		if !windowsMark.MatchString(out[i].text) || !strings.HasPrefix(strings.TrimSpace(out[i].text), "#") {
			continue
		}
		in := indent(out[i].text)
		out[i].text = uncomment(out[i].text, in)
		for i+1 < len(out) && indent(out[i+1].text) == in && strings.HasPrefix(strings.TrimSpace(out[i+1].text), "#") &&
			!windowsMark.MatchString(out[i+1].text) {
			i++
			out[i].text = uncomment(out[i].text, in)
		}
	}
	return out
}

func uncomment(s string, in int) string {
	rest := strings.TrimPrefix(s[in:], "#")
	rest = strings.TrimPrefix(rest, " ")
	if rest == "" {
		return ""
	}
	return s[:in] + rest
}

// sameWindows, with no install.sh to say, takes .goreleaser.yaml's Windows
// lines as uncommented when its builds have windows.
func sameWindows(app []line) bool {
	for _, l := range app {
		if t := strings.TrimSpace(windowsMark.ReplaceAllString(l.text, "")); t == "- windows" {
			return true
		}
	}
	return false
}

var (
	assignment = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*\s*(:=|::=|\?=|\+=|=)`)
	target     = regexp.MustCompile(`^([A-Za-z0-9_./-]+(?:\s+[A-Za-z0-9_./-]+)*)\s*:(?:[^=]|$)`)
)

// makefile checks the app's Makefile against tdp project.5: its first lines,
// the variables, include family.mk, then the app's own section, last.
func (r *repo) makefile() {
	s, ok := r.read("Makefile")
	if !ok {
		r.t.Errorf("tdptest: there is no Makefile (tdp project.5, project-23)")
		return
	}
	ls := strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	head := []string{"# " + r.name + " — build / run / test / package（家族共用的 target 在 family.mk）", "#", "# 跑 `make`（或 `make help`）列出所有指令。"}
	for i, want := range head {
		if i >= len(ls) || ls[i] != want {
			got := ""
			if i < len(ls) {
				got = ls[i]
			}
			r.t.Errorf("tdptest: Makefile line %d is %q, not %q: its first lines are tdp project.5's (project-26)", i+1, got, want)
			break
		}
	}
	inc := -1
	for i, l := range ls {
		if strings.TrimSpace(l) == "include family.mk" {
			inc = i
			break
		}
	}
	if inc < 0 {
		r.t.Errorf("tdptest: the Makefile does not include family.mk (tdp project.5, project-23)")
		return
	}
	binary := false
	for i := 0; i < inc; i++ {
		t := strings.TrimSpace(ls[i])
		switch {
		case t == "" || strings.HasPrefix(t, "#") || i > 0 && strings.HasSuffix(ls[i-1], "\\"):
		case assignment.MatchString(t):
			binary = binary || regexp.MustCompile(`^BINARY\s*:=\s*`+regexp.QuoteMeta(r.name)+`$`).MatchString(t)
		default:
			r.t.Errorf("tdptest: Makefile line %d, before include family.mk, is not a variable: %q (tdp project.5, project-24)", i+1, ls[i])
		}
	}
	if !binary {
		r.t.Errorf("tdptest: the Makefile does not set BINARY := %s before include family.mk (tdp project.5, project-25)", r.name)
	}
	own, section := -1, "##@ "+r.name+" 自己的"
	for i := inc + 1; i < len(ls); i++ {
		t := strings.TrimSpace(ls[i])
		switch {
		case own < 0 && t == section:
			own = i
		case own < 0 && t != "" && !strings.HasPrefix(t, "#"):
			r.t.Errorf("tdptest: Makefile line %d comes after include family.mk but before %q: %q; the app's own targets go in that section, last (tdp project.5, project-23)", i+1, section, ls[i])
		case own >= 0 && strings.HasPrefix(t, "##@"):
			r.t.Errorf("tdptest: Makefile line %d starts another section after %q: %q; it is the last one (tdp project.5, project-23)", i+1, section, ls[i])
		}
	}
	if own < 0 {
		r.t.Errorf("tdptest: the Makefile has no section %q after include family.mk (tdp project.5, project-23)", section)
	}
	family, _ := templates.ReadFile("_project/family.mk")
	ours := targets(strings.Split(string(family), "\n"))
	for i, l := range ls {
		for name := range targets([]string{l}) {
			if ours[name] {
				r.t.Errorf("tdptest: Makefile line %d makes target %s, which family.mk has (tdp project.5, project-27)", i+1, name)
			}
		}
	}
}

// targets are the target names the rule lines among ls make, special ones
// (.PHONY) left out.
func targets(ls []string) map[string]bool {
	out := map[string]bool{}
	for _, l := range ls {
		m := target.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		for _, name := range strings.Fields(m[1]) {
			if !strings.HasPrefix(name, ".") {
				out[name] = true
			}
		}
	}
	return out
}

var heading = regexp.MustCompile(`^## v\d+\.\d+\.\d+ — tdp v\d+\.\d+\.\d+$`)

// changelog checks CHANGELOG.md against tdp project.6: its first lines, and
// each heading naming a tdp written as a tag, " — tdp ", the version.
func (r *repo) changelog() {
	s, ok := r.read("CHANGELOG.md")
	if !ok {
		r.t.Errorf("tdptest: there is no CHANGELOG.md (tdp project.6, project-28)")
		return
	}
	ls := strings.Split(s, "\n")
	head := []string{"# Changelog", "", "Each release names the version of terminu design (tdp) it follows."}
	for i, want := range head {
		if i >= len(ls) || ls[i] != want {
			got := ""
			if i < len(ls) {
				got = ls[i]
			}
			r.t.Errorf("tdptest: CHANGELOG.md line %d is %q, not %q: it starts as tdp project.6's (project-28)", i+1, got, want)
			break
		}
	}
	for i, l := range ls {
		switch {
		case !strings.HasPrefix(l, "## "):
		case strings.HasPrefix(l, "## Unreleased") && l != "## Unreleased":
			r.t.Errorf("tdptest: CHANGELOG.md line %d is %q: what is not released yet goes under ## Unreleased alone (tdp project.6, project-30)", i+1, l)
		case strings.Contains(l, "tdp") && !heading.MatchString(l):
			r.t.Errorf("tdptest: CHANGELOG.md line %d is %q, not ## vX.Y.Z — tdp vA.B.C: the tag pushed, then the tdp it follows (tdp project.6, project-29)", i+1, l)
		}
	}
}

// diff is a's lines against b's, the ones that differ with two around them,
// "" when they are the same.
func diff(a, b []line, an, bn string) string {
	n, m := len(a), len(b)
	// lcs[i][j] is the longest common run of a[i:] and b[j:].
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i].text == b[j].text {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	type op struct {
		kind byte // ' ', '-', '+'
		l    line
	}
	var ops []op
	i, j := 0, 0
	for i < n || j < m {
		switch {
		case i < n && j < m && a[i].text == b[j].text:
			ops = append(ops, op{' ', a[i]})
			i, j = i+1, j+1
		case i < n && (j == m || lcs[i+1][j] >= lcs[i][j+1]):
			ops = append(ops, op{'-', a[i]})
			i++
		default:
			ops = append(ops, op{'+', b[j]})
			j++
		}
	}
	show := make([]bool, len(ops))
	for k, o := range ops {
		if o.kind != ' ' {
			for x := max(k-2, 0); x <= min(k+2, len(ops)-1); x++ {
				show[x] = true
			}
		}
	}
	var out strings.Builder
	rows := 0
	for k, o := range ops {
		if !show[k] {
			continue
		}
		if k == 0 || !show[k-1] {
			where := an
			if o.kind == '+' {
				where = bn
			}
			fmt.Fprintf(&out, "@@ %s line %d @@\n", where, o.l.n)
		}
		if rows++; rows > 60 {
			out.WriteString("…\n")
			break
		}
		fmt.Fprintf(&out, "%c %s\n", o.kind, o.l.text)
	}
	return out.String()
}
