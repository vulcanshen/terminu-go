package tdptest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	terminu "github.com/vulcanshen/terminu-go"
	"github.com/vulcanshen/terminu-go/internal/testhook"
)

func TestSandbox(t *testing.T) {
	t.Setenv("TINY__CONFIG", "/nowhere")
	t.Setenv("TERMINU__ICON_WIDTH", "2")
	t.Setenv("XDG_CONFIG_HOME", "/real/config")
	home := os.Getenv("HOME")
	var seeded string
	got := map[string]string{}
	t.Run("launch", func(t *testing.T) {
		launch(t, App{Name: "tiny", Env: func(home string) map[string]string {
			return map[string]string{"TINY__THEME": "dark", "TINY__HOSTS_FILE": filepath.Join(home, "hosts")}
		},
			Seed: func(h string) { seeded = h },
			New: func() *terminu.App {
				for _, k := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "TINY__THEME", "TINY__HOSTS_FILE"} {
					got[k] = os.Getenv(k)
				}
				for _, k := range []string{"TINY__CONFIG", "TERMINU__ICON_WIDTH"} {
					if v, ok := os.LookupEnv(k); ok {
						got[k] = v
					}
				}
				return newTiny().App
			}})
		if _, err := os.Stat(seeded); err != nil {
			t.Errorf("the sandbox is not there while the test runs: %v", err)
		}
	})
	if !strings.HasPrefix(got["HOME"], "/tmp/tdp-") || seeded != got["HOME"] {
		t.Errorf("HOME %q, seeded in %q", got["HOME"], seeded)
	}
	if got["XDG_CONFIG_HOME"] != filepath.Join(got["HOME"], ".config") || got["XDG_CACHE_HOME"] != filepath.Join(got["HOME"], ".cache") {
		t.Errorf("XDG: %v", got)
	}
	if got["TINY__THEME"] != "dark" || got["TINY__HOSTS_FILE"] != filepath.Join(got["HOME"], "hosts") {
		t.Errorf("Env is not set, into the sandbox: %v", got)
	}
	if v, ok := got["TINY__CONFIG"]; ok {
		t.Errorf("the app's own variable is still there: %q", v)
	}
	if v, ok := got["TERMINU__ICON_WIDTH"]; ok {
		t.Errorf("the family's variable is still there: %q", v)
	}
	if os.Getenv("HOME") != home || os.Getenv("TINY__CONFIG") != "/nowhere" {
		t.Error("the variables were not put back")
	}
	if _, err := os.Stat(seeded); !os.IsNotExist(err) {
		t.Errorf("the sandbox stays after the test: %v", err)
	}
}

// colouredApp is the small app with a colour of its own left in a cell,
// not in Styled.
func colouredApp() *terminu.App {
	ty := newTiny()
	ty.list.SetRows([]terminu.Row{{ID: "1", Cells: []string{terminu.Fg(terminu.Red).Render("red")}}})
	return ty.App
}

// TestCheckFindsColourCodes: Check fails an app with a colour code left in a
// cell; it runs in a child test process, which must fail.
func TestCheckFindsColourCodes(t *testing.T) {
	if os.Getenv("TDPTEST_CHILD") == "coloured" {
		Check(t, App{Name: "tiny", New: colouredApp})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestCheckFindsColourCodes$", "-test.short")
	cmd.Env = append(os.Environ(), "TDPTEST_CHILD=coloured")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "shows the colour code") {
		t.Errorf("Check did not fail on a colour left in a cell: %v\n%s", err, out)
	}
}

// TestCheckTrickySeed: seeded data with a colour code in a name, drawn as
// text as it should be, is not taken for a colour left out of spans when the
// App says it is Tricky (locku LK21).
func TestCheckTrickySeed(t *testing.T) {
	Check(t, App{Name: "tiny", Tricky: true, New: func() *terminu.App {
		ty := newTiny()
		ty.list.SetRows([]terminu.Row{{ID: "1", Cells: []string{"\x1b[31mred\x1b[0m"}}})
		return ty.App
	}})
}

func TestCheckTiny(t *testing.T) {
	var homes []string
	seen := map[string]bool{}
	Check(t, App{Name: "tiny", New: func() *terminu.App { return newTiny().App },
		Seed: func(home string) {
			homes = append(homes, home)
			if seen[home] {
				t.Errorf("a launch in the sandbox of another: %s", home)
			}
			seen[home] = true
			if entries, _ := os.ReadDir(home); len(entries) != 0 {
				t.Errorf("a first launch with something in its HOME already: %v", entries)
			}
			_ = os.WriteFile(filepath.Join(home, "launched"), nil, 0o600)
		}})
	want := 1 + 3*len(scanSizes(30, 12))
	if testing.Short() {
		want = 1 + 2 // the least size and 80x24, icons one cell
	}
	if len(homes) != want {
		t.Errorf("%d launches, want %d: one for the least size, then each size with each icon width", len(homes), want)
	}
}

// TestCheckLimits: a least size above 60×32, wider or taller, and a narrow
// width above 60 fail, naming the setting; at the most they pass (tdp L1,
// screen.5).
func TestCheckLimits(t *testing.T) {
	for _, c := range []struct {
		w, h, narrow int
		want         string
	}{
		{60, 32, 60, ""}, {0, 0, 0, ""},
		{80, 24, 0, "MinWidth, MinHeight"}, {60, 33, 0, "above 60×32"}, {60, 32, 61, "NarrowBelow"},
	} {
		ty := newTiny()
		ty.MinWidth, ty.MinHeight = c.w, c.h
		testhook.Screens(ty.App).([]*terminu.Screen)[0].NarrowBelow = c.narrow
		ft := &fakeT{T: t}
		limits(ft, ty.App)
		if c.want == "" && len(ft.errs) > 0 || c.want != "" && (len(ft.errs) != 1 || !strings.Contains(ft.errs[0], c.want)) {
			t.Errorf("%d×%d, narrow below %d: %q", c.w, c.h, c.narrow, ft.errs)
		}
	}
}

// TestScanSizes: Check launches on the standard 60×32 (tdp L1) as well as
// the least size.
func TestScanSizes(t *testing.T) {
	if sizes := scanSizes(30, 12); !contains(sizes, [2]int{60, 32}) || !contains(sizes, [2]int{30, 12}) {
		t.Errorf("%v", sizes)
	}
}

// TestCheckFindsLimits: Check fails an app with a least size above 60×32;
// it runs in a child test process, which must fail (tdp L1).
func TestCheckFindsLimits(t *testing.T) {
	if os.Getenv("TDPTEST_CHILD") == "limits" {
		Check(t, App{Name: "tiny", New: func() *terminu.App {
			ty := newTiny()
			ty.MinWidth, ty.MinHeight = 80, 24
			return ty.App
		}})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestCheckFindsLimits$", "-test.short")
	cmd.Env = append(os.Environ(), "TDPTEST_CHILD=limits")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "above 60×32") {
		t.Errorf("Check did not fail on a least size of 80×24: %v\n%s", err, out)
	}
}

// TestTinyProblems: under the least size the name, the size and how to quit
// are drawn top down, nothing else; short of rows the size goes first, then
// how to quit; a row wider than the screen does not fit (tdp L1).
func TestTinyProblems(t *testing.T) {
	need := "needs a terminal of at least 30×12"
	for _, c := range []struct {
		w, h  int
		lines []string
		want  string // a problem it has; "" for none
	}{
		{80, 5, []string{"", "tiny", need, tinyQuit, ""}, ""},
		{80, 2, []string{"tiny", tinyQuit}, ""},
		{30, 4, []string{"tiny", tinyQuit, "", ""}, ""},
		{33, 5, []string{"", "tiny", tinyQuit, "", ""}, ""},
		{27, 5, []string{"", "tiny", "", "", ""}, ""},
		{2, 1, []string{"ti"}, ""},
		{80, 2, []string{"tiny", need}, "short of room the size goes first"},
		{80, 5, []string{"", "tiny", tinyQuit, need, ""}, "top down"},
		{80, 5, []string{"", "tiny", need, "", ""}, "how to quit"},
		{30, 4, []string{"tiny", need, tinyQuit, ""}, "should be on the screen: false"},
		{80, 5, []string{"tiny", need, tinyQuit, "PODS", ""}, "more than"},
		{80, 5, []string{"", "", need, tinyQuit, ""}, "the name"},
	} {
		got := tinyProblems(c.lines, "tiny", c.w, c.h, 30, 12)
		if c.want == "" && len(got) > 0 || c.want != "" && !strings.Contains(strings.Join(got, "\n"), c.want) {
			t.Errorf("%d×%d %q: %q, want %q", c.w, c.h, c.lines, got, c.want)
		}
	}
}

// breakingApp is the small app with a line break in a toast of its own, at
// start.
func breakingApp() *terminu.App {
	ty := newTiny()
	ty.App.Start = func() tea.Cmd { return ty.App.Toast(terminu.Info, "Saved.\nIt takes effect now.") }
	return ty.App
}

// TestCheckFindsLineBreaks: Check fails an app whose own text carries a line
// break, drawn by its name in Red; it runs in a child test process, which
// must fail (components-30; sshu SS43). An app saying its data is Tricky is
// not failed for one.
func TestCheckFindsLineBreaks(t *testing.T) {
	if os.Getenv("TDPTEST_CHILD") == "breaking" {
		Check(t, App{Name: "tiny", New: breakingApp})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestCheckFindsLineBreaks$", "-test.short")
	cmd.Env = append(os.Environ(), "TDPTEST_CHILD=breaking")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "shows a line break by its name") {
		t.Errorf("Check did not fail on a line break in a toast: %v\n%s", err, out)
	}
	d := Start(t, breakingApp(), 80, 24)
	d.AllowLineBreaks = true // shown, the check let off
	if !d.Has(`\n`) {
		t.Error("the toast does not show the line break by its name")
	}
}

// TestDriverFindsLineBreaks: a Driver checks for line breaks by default, in
// an app's own tests too, the frames from Start on: one shown fails the test
// when it ends (a child test process, which must fail) unless the test lets
// it be with AllowLineBreaks, set after Start.
func TestDriverFindsLineBreaks(t *testing.T) {
	if os.Getenv("TDPTEST_CHILD") == "driver" {
		Start(t, breakingApp(), 80, 24)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestDriverFindsLineBreaks$")
	cmd.Env = append(os.Environ(), "TDPTEST_CHILD=driver")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "shows a line break by its name") {
		t.Errorf("a Driver did not fail on a line break in a toast: %v\n%s", err, out)
	}
}

// TestDriverFindsDimmedLineBreaks: the Driver finds a line break drawn by
// its name in a list cell under a popup, dimmed (a child test process, which
// must fail; sshu, on v0.4.19).
func TestDriverFindsDimmedLineBreaks(t *testing.T) {
	if os.Getenv("TDPTEST_CHILD") == "dimmed" {
		ty := newTiny()
		d := Start(t, ty.App, 80, 24)
		d.Press("Space")
		d.Do(func() tea.Cmd {
			ty.list.SetRows([]terminu.Row{{ID: "1", Cells: []string{"first"}}, {ID: "2", Cells: []string{"line\nbreak"}}})
			return nil
		})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestDriverFindsDimmedLineBreaks$")
	cmd.Env = append(os.Environ(), "TDPTEST_CHILD=dimmed")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "shows a line break by its name") {
		t.Errorf("a Driver did not fail on a line break in a dimmed list: %v\n%s", err, out)
	}
}
