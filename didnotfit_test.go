package terminu

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestToast28Report: a success report too long for a toast opens in a
// read-only viewer, titled with what was done; Enter closes it as well as
// Esc, but not in its selection mode; the hint and ? say Enter|Esc; it goes
// in the App Log as a Success. A viewer that is not a report still closes
// with Esc only (tdp toast.2, note.1, note.3, toast-28).
func TestToast28Report(t *testing.T) {
	ta := newTestApp(t)
	report := func() *Viewer {
		v := &Viewer{Title: "Exported", Lines: []string{Fg(Green).Render("wrote 3 files"), "to /home/me/exports/2026-11-18"}, NoNumbers: true}
		ta.Report(v)
		ta.settle()
		return v
	}
	v := report()
	if vv, ok := ta.topView().(*viewerView); !ok || vv.v != v || !strings.Contains(ta.screenText(), "Enter|Esc:close") {
		t.Fatalf("not a report viewer: %T\n%s", ta.topView(), ta.screenText())
	}
	if e := ta.logs.entries[len(ta.logs.entries)-1]; e.kind != Success || e.text != "Exported: wrote 3 files to /home/me/exports/2026-11-18" {
		t.Errorf("the App Log: %+v", e)
	}
	ta.press("Enter")
	if ta.top() != nil {
		t.Errorf("Enter did not close it: %T", ta.topView())
	}

	report()
	ta.press("?")
	if text := ta.screenText(); !strings.Contains(text, "Enter|Esc") || !strings.Contains(text, "close") {
		t.Errorf("? does not say Enter|Esc:\n%s", text)
	}
	ta.press("Esc", "v", "Enter")
	if vv, ok := ta.topView().(*viewerView); !ok || !vv.v.inMode {
		t.Fatalf("Enter in the selection mode closed it: %T", ta.topView())
	}
	ta.press("Esc", "Esc")
	if ta.top() != nil {
		t.Errorf("Esc did not close it: %T", ta.topView())
	}

	ta.Viewer(&Viewer{Title: "YAML", Lines: []string{"kind: Pod"}})
	ta.settle()
	ta.press("Enter")
	if _, ok := ta.topView().(*viewerView); !ok || strings.Contains(ta.screenText(), "Enter|Esc:close") {
		t.Errorf("a viewer that is not a report closed with Enter, or says so")
	}
}

// TestE3KeepAside: a file with values that do not fit is copied to
// <name>.bak, then .bak.1…, never over another, with its mode, the file
// itself left as it was; a copy that cannot be made says so in the Error
// toast's words, wrapping why, and leaves no copy (tdp E3, E3-29, E3-32).
func TestE3KeepAside(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("color: \"#zzzzzz\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cp, err := KeepAside(path)
	if err != nil || cp != path+".bak" {
		t.Fatalf("KeepAside: %q %v", cp, err)
	}
	got, _ := os.ReadFile(cp)
	info, _ := os.Stat(cp)
	if string(got) != "color: \"#zzzzzz\"\n" || runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Errorf("the copy: %q %v", got, info.Mode())
	}
	if orig, _ := os.ReadFile(path); string(orig) != "color: \"#zzzzzz\"\n" {
		t.Errorf("the file changed: %q", orig)
	}
	_ = os.WriteFile(path, []byte("color: red\n"), 0o600)
	cp, err = KeepAside(path)
	if err != nil || cp != path+".bak.1" {
		t.Fatalf("again: %q %v", cp, err)
	}
	if old, _ := os.ReadFile(path + ".bak"); string(old) != "color: \"#zzzzzz\"\n" {
		t.Errorf("the first copy was written over: %q", old)
	}

	missing := filepath.Join(dir, "gone.yaml")
	if cp, err := KeepAside(missing); cp != "" || !errors.Is(err, fs.ErrNotExist) ||
		!strings.HasPrefix(err.Error(), "gone.yaml.bak could not be made: ") || !strings.HasSuffix(err.Error(), "; gone.yaml left as it was") ||
		strings.Contains(err.Error(), dir) {
		t.Errorf("no file: %q %v", cp, err)
	}
	if _, err := os.Lstat(missing + ".bak"); err == nil {
		t.Error("a copy that failed was left")
	}
	sub := filepath.Join(dir, "folder")
	_ = os.Mkdir(sub, 0o700)
	if cp, err := KeepAside(sub); cp != "" || err == nil {
		t.Errorf("a folder: %q %v", cp, err)
	}
	if _, err := os.Lstat(sub + ".bak"); err == nil {
		t.Error("a copy that failed halfway was left")
	}
	if err := copyNew(path, path+".bak"); err == nil {
		t.Error("copyNew wrote over a file that was there")
	}
	if old, _ := os.ReadFile(path + ".bak"); string(old) != "color: \"#zzzzzz\"\n" {
		t.Errorf("copyNew changed a file that was there: %q", old)
	}

	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		return // a folder that cannot be written
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o700) })
	cp, err = KeepAside(path)
	if cp != "" || !errors.Is(err, fs.ErrPermission) || err.Error() != "config.yaml.bak.2 could not be made: permission denied; config.yaml left as it was" {
		t.Errorf("a folder that cannot be written: %q %v", cp, err)
	}
}

// TestE3DidNotFit: one Warning toast naming the file and its copy, a WARN
// in the App Log for each value or entry, its key, what it had and what is
// used now; the sentence alone for stderr; with no copy only the WARNs (tdp
// E3, E3-30, E3-31, E3-32).
func TestE3DidNotFit(t *testing.T) {
	ta := newTestApp(t)
	values := []Misfit{{"color", "#zzzzzz", "#89b4fa"}, {"timeout", "-5", "30"}, {"name", "", "web"}}
	before := len(ta.logs.entries)
	ta.DidNotFit("/home/me/.config/app/config.yaml", "/home/me/.config/app/config.yaml.bak", values...)
	ta.settle()
	want := "3 values in config.yaml did not fit: defaults used, yours kept in config.yaml.bak"
	if !strings.Contains(ta.screenText(), want) {
		t.Errorf("the toast:\n%s", ta.screenText())
	}
	var got []string
	for _, e := range ta.logs.entries[before:] {
		if e.kind != Warning {
			t.Errorf("not a WARN: %+v", e)
		}
		got = append(got, e.text)
	}
	if strings.Join(got, "\n") != want+"\n"+
		`config.yaml: color "#zzzzzz" did not fit; using "#89b4fa"`+"\n"+
		`config.yaml: timeout "-5" did not fit; using "30"`+"\n"+
		`config.yaml: name "" did not fit; using "web"` {
		t.Errorf("the App Log:\n%s", strings.Join(got, "\n"))
	}

	entry := Misfit{Key: `entry 3 (name "")`}
	for _, c := range []struct {
		misfits []Misfit
		want    string
	}{
		{[]Misfit{values[0]}, "1 value in hosts.yaml did not fit: defaults used, yours kept in hosts.yaml.bak.1"},
		{[]Misfit{entry, entry}, "2 entries in hosts.yaml did not fit: taken out, yours kept in hosts.yaml.bak.1"},
		{[]Misfit{entry}, "1 entry in hosts.yaml did not fit: taken out, yours kept in hosts.yaml.bak.1"},
		{append(values, entry, entry), "3 values and 2 entries in hosts.yaml did not fit: defaults used and entries taken out, yours kept in hosts.yaml.bak.1"},
		{[]Misfit{values[0], entry}, "1 value and 1 entry in hosts.yaml did not fit: defaults used and entries taken out, yours kept in hosts.yaml.bak.1"},
		{nil, ""},
	} {
		if got := DidNotFitText("/data/hosts.yaml", "/data/hosts.yaml.bak.1", c.misfits...); got != c.want {
			t.Errorf("%d misfits: %q", len(c.misfits), got)
		}
	}

	ta = newTestApp(t)
	before = len(ta.logs.entries)
	if cmd := ta.DidNotFit("/data/hosts.yaml", "", entry); cmd != nil || ta.toast != nil {
		t.Error("no copy: a toast")
	}
	if e := ta.logs.entries[before:]; len(e) != 1 || e[0].kind != Warning || e[0].text != `hosts.yaml: entry 3 (name "") did not fit; taken out` {
		t.Errorf("no copy: the App Log %+v", e)
	}
	if DidNotFitText("/data/hosts.yaml", "", entry) != "" {
		t.Error("no copy: a sentence")
	}
}
