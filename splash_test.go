package terminu

import (
	"strings"
	"testing"
)

// withSplash gives the test app a splash that reveals "MARK" a letter a
// frame.
func withSplash(ta *testApp) *testApp {
	ta.Splash = func(frame, w, h int) ([]string, bool) {
		n := min(frame+1, 4)
		return []string{"MARK"[:n]}, n == 4
	}
	return ta
}

func TestS1SplashOnV(t *testing.T) {
	ta := withSplash(newTestApp(t))
	cmd := ta.press("V")
	if !ta.splashOn {
		t.Fatal("V on a panel did not open the splash")
	}
	for range 10 {
		if cmd == nil {
			break
		}
		cmd = ta.Update2(cmd())
	}
	if got := ta.screenText(); !strings.HasPrefix(got, "MARK") || strings.Contains(got, "PODS") {
		t.Errorf("the splash, revealed, over the whole screen:\n%s", got)
	}
	ta.press("x")

	// A panel may give V to an action of its own; the splash does not open there.
	ta.pods.Actions = append(ta.pods.Actions, Action{Key: "V", Label: "View", Desc: "it", Scope: ItemScope, Do: ta.record("view")})
	ta.press("V")
	if ta.splashOn || len(ta.ran) == 0 || ta.ran[len(ta.ran)-1] != "view" {
		t.Errorf("the panel's own V: splash %v, ran %v", ta.splashOn, ta.ran)
	}
	ta.press("2", "V")
	if !ta.splashOn {
		t.Error("another panel still opens the splash")
	}
}

func TestS1EveryAppHasSplash(t *testing.T) {
	ta := newTestApp(t)
	if ta.checkSplash() == nil {
		t.Error("an app without a splash")
	}
	withSplash(ta)
	if err := ta.checkSplash(); err != nil {
		t.Error(err)
	}
	ta.Global = append(ta.Global, Action{Key: "V", Label: "Vault", Scope: GlobalScope})
	if ta.checkSplash() == nil {
		t.Error("a global hotkey on V")
	}
	ta.Global = ta.Global[:len(ta.Global)-1]
	for _, s := range ta.screens {
		for _, p := range s.Panels {
			p.Actions = append(p.Actions, Action{Key: "V", Label: "View", Scope: PanelScope})
		}
	}
	if ta.checkSplash() == nil {
		t.Error("every panel gave V away")
	}
}

func TestS2SplashNotDisclosed(t *testing.T) {
	ta := withSplash(newTestApp(t))
	seen := StripStyles(strings.Join(ta.Frame(), "\n")) // footer and hints
	for _, keys := range [][]string{{"Space"}, {"?"}, {"Space", "G", "Enter"}} {
		ta.press(keys...)
		seen += ta.screenText()
		for len(ta.stack) > 0 {
			ta.press("Esc")
		}
	}
	if strings.Contains(strings.ToLower(seen), "splash") {
		t.Error("the splash is listed somewhere")
	}
}

func TestS3AnyKeyClosesOnly(t *testing.T) {
	ta := withSplash(newTestApp(t))
	for _, k := range []string{"q", "Ctrl-C", "Esc", "Space", "?", "Tab", "S"} {
		ta.press("V")
		focus := ta.Screen().focus
		cmd := ta.press(k)
		if ta.splashOn {
			t.Errorf("%s did not close the splash", k)
		}
		if cmd != nil || ta.top() != nil || ta.quitting || ta.Screen().focus != focus || len(ta.ran) > 0 {
			t.Errorf("%s did more than close the splash", k)
		}
	}
	ta.press("V", "Ctrl-C", "Ctrl-C")
	if ta.top() == nil || !ta.quitting {
		t.Error("Ctrl-C after the splash is closed starts the quit flow")
	}
}

func TestS4SplashOnlyOnPanel(t *testing.T) {
	ta := withSplash(newTestApp(t))
	if ta.splashOn {
		t.Fatal("the splash plays at the start")
	}
	ta.press("Space", "V")
	if ta.splashOn {
		t.Error("V over a popup")
	}
	ta.press("Esc", "/", "V")
	if ta.splashOn || ta.list.filter.value() != "V" {
		t.Errorf("V on a typing row is a letter: splash %v, filter %q", ta.splashOn, ta.list.filter.value())
	}
	ta.press("Esc", "Esc", "D", "V")
	if !ta.splashOn {
		t.Error("V in a mode, still on the panel")
	}
}
