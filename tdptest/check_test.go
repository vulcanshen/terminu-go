package tdptest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	terminu "github.com/vulcanshen/terminu-go"
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
