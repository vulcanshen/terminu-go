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
		launch(t, App{Name: "tiny", Env: map[string]string{"TINY__THEME": "dark"},
			Seed: func(h string) { seeded = h },
			New: func() *terminu.App {
				for _, k := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "TINY__THEME"} {
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
	if got["TINY__THEME"] != "dark" {
		t.Error("Env is not set")
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
	Check(t, App{Name: "tiny", New: func() *terminu.App { return newTiny().App }})
}
