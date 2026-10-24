package tdptest

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	terminu "github.com/vulcanshen/terminu-go"
	"github.com/vulcanshen/terminu-go/internal/testhook"
)

// App is how Check makes the app, as its main does.
type App struct {
	// Name is the app's name, as terminu.New takes it: the variables Check
	// clears are <NAME>__… (tdp E2).
	Name string
	// New makes the app, all but Run. Check calls it once for each launch,
	// in a new sandbox.
	New func() *terminu.App
	// Env is more variables for each launch; home is the sandbox's HOME, for
	// a value that points into it (a file Seed put there).
	Env func(home string) map[string]string
	// Seed puts files in the sandbox before New runs; home is its HOME.
	Seed func(home string)
	// Tricky says Seed's data has tricky values in it (Tricky): a colour
	// code in a name there is the data's, rightly drawn as text, so Check
	// does not take one on the screen for a colour of the app's own left out
	// of its spans (Driver.ColourCodes).
	Tricky bool
}

// Check launches the app as for the first time on terminals of many sizes,
// with an icon and a powerline symbol one cell and two (EachIconWidth), and
// walks what every app has: on each screen and each panel, the Space menu,
// the global operation popup, the App Log (`!`) and the key reference (`?`),
// each closed one layer an Esc (tdp K4); then a terminal under the least size and back
// (tdp L1). Every frame is checked for tdp L4 on the way. With go test
// -short it launches on the least size and 80×24 only, icons one cell.
//
// Each launch is a first one, in a sandbox of its own: a new directory
// under /tmp (short, so a path on the screen is as long on every machine)
// as HOME, the XDG directories and Windows' profile directories in it, the
// app's own variables and the family's (<NAME>__…, TERMINU__…,
// LC_TERMINU_…) cleared, then Env and Seed. Check sets variables, so its
// test does not run in parallel.
func Check(t *testing.T, app App) {
	t.Helper()
	if app.Name == "" || app.New == nil {
		t.Fatal("tdptest: Check needs the app's Name and New")
	}
	mw, mh := testhook.MinSize(launch(t, app))
	sizes, each := scanSizes(mw, mh), EachIconWidth
	if testing.Short() {
		sizes = [][2]int{{mw, mh}}
		if mw != 80 || mh != 24 {
			sizes = append(sizes, [2]int{80, 24})
		}
		each = func(t *testing.T, f func(t *testing.T)) { t.Run("icon1_powerline1", f) }
	}
	each(t, func(t *testing.T) {
		for _, s := range sizes {
			t.Run(fmt.Sprintf("%dx%d", s[0], s[1]), func(t *testing.T) {
				a := launch(t, app)
				d := Start(t, a, s[0], s[1])
				d.ColourCodes, d.AllowLineBreaks = !app.Tricky, app.Tricky
				if s[0] >= mw && s[1] >= mh {
					scan(t, d, a)
					shrink(t, d, mw, mh)
					return
				}
				checkTiny(t, d, mw, mh)
				d.Press("Ctrl-C", "Ctrl-C")
				if !d.Quit() {
					t.Errorf("tdptest: under the least size Ctrl-C twice did not leave (tdp L1)%s", d.where())
				}
			})
		}
	})
}

// scanSizes are the sizes Check launches on: the least, common ones, a big
// one, and under the least (only narrower, only lower, 80×5, two rows, one,
// 1×1; tdp L1, E5).
func scanSizes(mw, mh int) [][2]int {
	var out [][2]int
	for _, s := range [][2]int{{mw, mh}, {80, 40}, {120, 40}, {80, 24}, {300, 50}, {mw - 1, mh}, {mw, mh - 1}, {80, 5}, {80, 2}, {80, 1}, {1, 1}} {
		if s[0] >= 1 && s[1] >= 1 && !contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}

func contains(sizes [][2]int, s [2]int) bool {
	for _, o := range sizes {
		if o == s {
			return true
		}
	}
	return false
}

// launch makes the app in a new sandbox.
func launch(t *testing.T, app App) *terminu.App {
	t.Helper()
	home := sandbox(t, app.Name)
	if app.Env != nil {
		for k, v := range app.Env(home) {
			t.Setenv(k, v)
		}
	}
	if app.Seed != nil {
		app.Seed(home)
	}
	return app.New()
}

// sandbox points HOME and the places derived from it at a new directory and
// clears the variables that would take the app elsewhere; it is HOME.
func sandbox(t *testing.T, name string) string {
	t.Helper()
	dir := ""
	if runtime.GOOS != "windows" {
		if d, err := os.MkdirTemp("/tmp", "tdp-"); err == nil {
			dir = d
			t.Cleanup(func() { _ = os.RemoveAll(d) })
		}
	}
	if dir == "" {
		dir = t.TempDir()
	}
	prefixes := []string{terminu.EnvName(name, ""), "TERMINU__", "LC_TERMINU_"}
	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		for _, p := range prefixes {
			if strings.HasPrefix(k, p) {
				t.Setenv(k, "")
				_ = os.Unsetenv(k)
			}
		}
	}
	for k, v := range map[string]string{
		"HOME": dir, "USERPROFILE": dir,
		"XDG_CONFIG_HOME": filepath.Join(dir, ".config"), "XDG_CACHE_HOME": filepath.Join(dir, ".cache"),
		"XDG_DATA_HOME": filepath.Join(dir, ".local", "share"), "XDG_STATE_HOME": filepath.Join(dir, ".local", "state"),
		"LOCALAPPDATA": filepath.Join(dir, "AppData", "Local"), "APPDATA": filepath.Join(dir, "AppData", "Roaming"),
	} {
		t.Setenv(k, v)
	}
	return dir
}

// scan walks every screen and panel: Space menu, global operation popup,
// key reference, each closed an Esc at a time.
func scan(t *testing.T, d *Driver, a *terminu.App) {
	t.Helper()
	closeAll(t, d, "at the start")
	if a.Canvas != nil {
		return // the app's own screen: no panels to walk
	}
	for _, s := range testhook.Screens(a).([]*terminu.Screen) {
		d.Do(func() tea.Cmd { return a.ShowScreen(s) })
		for _, p := range s.Panels {
			if _, pty := p.Body.(*terminu.Terminal); pty || p.InMode() != nil {
				continue // keys go to the program, or to the mode
			}
			d.Do(func() tea.Cmd { return a.Focus(p) })
			where := fmt.Sprintf("on %s, panel %s", s.Name, p.Label)
			opens(t, d, where+": Space", 1, "Space")
			if p == s.Panels[0] {
				// What does not change from panel to panel, once a screen.
				opens(t, d, where+": ? on the Space menu", 2, "?")
				closeAll(t, d, where)
				opens(t, d, where+": the Space menu's Global operation", 2, "Space", "G", "Enter")
				closeAll(t, d, where)
				opens(t, d, where+": ! (the App Log)", 1, "!")
			}
			closeAll(t, d, where)
			opens(t, d, where+": ?", 1, "?")
			closeAll(t, d, where)
		}
	}
}

// opens presses keys and checks that n popups are open after.
func opens(t *testing.T, d *Driver, what string, n int, keys ...string) {
	t.Helper()
	d.Press(keys...)
	if got := testhook.Layers(d.app); got != n {
		t.Errorf("tdptest: %s: %d popups open, want %d%s", what, got, n, d.where())
	}
}

// closeAll presses Esc until no popup is open, checking each closes one
// layer and no more (tdp K4) and the app stays. Esc takes a toast first.
func closeAll(t *testing.T, d *Driver, where string) {
	t.Helper()
	for n := testhook.Layers(d.app); n > 0 || testhook.Toast(d.app); n = testhook.Layers(d.app) {
		toast := testhook.Toast(d.app)
		d.Press("Esc")
		want := n - 1
		if toast {
			want = n
		}
		if got := testhook.Layers(d.app); got != want || toast && testhook.Toast(d.app) {
			t.Errorf("tdptest: %s: Esc on %d popups (toast %v) left %d, want %d (tdp K4)%s", where, n, toast, got, want, d.where())
			return
		}
	}
	if d.Quit() {
		t.Errorf("tdptest: %s: Esc quit the app (tdp K4)", where)
	}
}

// escape presses Esc until no popup is open, a few more times than there
// are popups, as a popup with a state of its own (a textarea writing)
// leaves it first: each Esc closes one layer at most and opens none (tdp
// K4).
func escape(t *testing.T, d *Driver, where string) {
	t.Helper()
	for tries := 3*testhook.Layers(d.app) + 2; testhook.Layers(d.app) > 0 || testhook.Toast(d.app); tries-- {
		n := testhook.Layers(d.app)
		if tries == 0 {
			t.Errorf("tdptest: %s: Esc does not close the last %d popups (tdp K4)%s", where, n, d.where())
			return
		}
		d.Press("Esc")
		if got := testhook.Layers(d.app); got < n-1 || got > n {
			t.Errorf("tdptest: %s: Esc on %d popups left %d (tdp K4)%s", where, n, got, d.where())
			return
		}
	}
}

// shrink takes the terminal under the least size with a popup open and
// back: the same screen comes back, popup and all (tdp L1).
func shrink(t *testing.T, d *Driver, mw, mh int) {
	t.Helper()
	if d.app.Canvas == nil && len(testhook.Screens(d.app).([]*terminu.Screen)) > 0 {
		d.Press("Space")
	}
	w, h := d.w, d.h
	before := strings.Join(d.Screen(), "\n")
	for _, s := range [][2]int{{mw - 1, mh}, {mw, mh - 1}, {1, 1}} {
		if s[0] < 1 || s[1] < 1 {
			continue
		}
		d.Resize(s[0], s[1])
		checkTiny(t, d, mw, mh)
		d.Resize(w, h)
		if after := strings.Join(d.Screen(), "\n"); after != before {
			t.Errorf("tdptest: back from %d×%d the screen is not what it was (tdp L1)\n  before:\n%s\n  after:\n%s", s[0], s[1], before, after)
		}
	}
}

// checkTiny checks a terminal under the least size: only the app's name
// and the size it needs, in the middle; no key acts, q and one Ctrl-C do
// not leave (tdp L1).
func checkTiny(t *testing.T, d *Driver, mw, mh int) {
	t.Helper()
	need := fmt.Sprintf("needs a terminal of at least %d×%d", mw, mh)
	text := strings.Join(d.Screen(), "\n")
	name := d.app.Name
	shown := terminu.StripStyles(terminu.Cut(name, 0, min(d.w, terminu.Width(name)))) // as much as fits
	if !strings.Contains(text, shown) {
		t.Errorf("tdptest: under the least size the name %q is not on the screen (tdp L1)%s", shown, d.where())
	}
	if fits := d.h >= 2 && terminu.Width(need) <= d.w; fits != strings.Contains(text, need) {
		t.Errorf("tdptest: under the least size %q should be on the screen: %v (tdp L1)%s", need, fits, d.where())
	}
	if rest := strings.TrimSpace(strings.NewReplacer(shown, "", need, "").Replace(text)); rest != "" {
		t.Errorf("tdptest: under the least size more than the name and the size are drawn: %q (tdp L1)%s", rest, d.where())
	}
	quit := d.Quit()
	d.Press("j", "q", "Ctrl-C", "Space", "Enter", "Esc")
	if !quit && d.Quit() {
		t.Errorf("tdptest: under the least size q or one Ctrl-C left the app (tdp L1)%s", d.where())
		return
	}
	if strings.Join(d.Screen(), "\n") != text {
		t.Errorf("tdptest: under the least size a key changed the screen (tdp L1)%s", d.where())
	}
}
