package terminu

import (
	"fmt"
	"slices"

	tea "github.com/charmbracelet/bubbletea"
)

// CoreKey is a core key of the app's own (Rules K1): "!" for the app's log.
type CoreKey struct {
	Key, Desc string
	Do        func() tea.Cmd
}

func (a *App) coreKey(key string) *CoreKey {
	for i := range a.CoreKeys {
		if a.CoreKeys[i].Key == key {
			return &a.CoreKeys[i]
		}
	}
	return nil
}

// isCoreKey reports a key that means the same on every surface: the table of
// Rules K1, and the app's own.
func (a *App) isCoreKey(key string) bool {
	return slices.Contains([]string{"Tab", "Shift-Tab", "Enter", "Esc", "Space", "?", "q", "Ctrl-C"}, key) ||
		len(key) == 1 && key[0] >= '1' && key[0] <= '9' || a.coreKey(key) != nil
}

// typingOrMode reports the surfaces where the app's core keys do not work:
// a typing row (Rules K8) and a panel in a mode (K11). A PTY never gets here.
func (a *App) typingOrMode() bool {
	if l := a.top(); l != nil {
		return l.view.typing()
	}
	s := a.Screen()
	if s == nil {
		return false
	}
	p := s.Focused()
	return p != nil && (p.typing() || p.mode != nil)
}

// checkKeys refuses hotkeys that are not the app's to take: a global or a
// screen key on a core key or a movement letter (Rules K1, K12), and any
// hotkey on one of the app's own core keys.
func (a *App) checkKeys() error {
	for _, ck := range a.CoreKeys {
		if err := checkPanelKey(ck.Key); err != nil || ck.Key == "" {
			return fmt.Errorf("terminu: core key %q: %v", ck.Key, err)
		}
	}
	taken := func(where, key string) error {
		if a.coreKey(key) != nil {
			return fmt.Errorf("terminu: %s takes %s, a core key of the app (Rules K1)", where, key)
		}
		return nil
	}
	for _, act := range a.Global {
		if err := checkPopupKey(act.Key); err != nil {
			return fmt.Errorf("terminu: global action %q: %v", act.Label, err)
		}
		if err := taken("global action "+act.Label, act.Key); err != nil {
			return err
		}
	}
	for _, s := range a.screens {
		if err := checkPanelKey(s.Key); err != nil {
			return fmt.Errorf("terminu: screen %q: %v", s.Name, err)
		}
		if err := taken("screen "+s.Name, s.Key); err != nil {
			return err
		}
		for _, p := range s.Panels {
			for _, act := range p.Actions {
				if err := taken("panel "+p.Label+", action "+act.Label, act.Key); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
