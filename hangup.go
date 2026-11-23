package terminu

import (
	"os"
	"os/signal"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
)

// onHangup makes closing the terminal (SIGHUP; on Windows closing the
// console window) end p as a quit does, the programs in Terminals hung up
// first. The function it returns, called once p has ended, reports whether
// the terminal was closed.
func onHangup(p *tea.Program) (hungUp func() bool) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, closedSignal())
	var hung atomic.Bool
	hangUp := func() {
		hung.Store(true)
		stopRunning()
	}
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-sig:
			hangUp()
			p.Send(tea.QuitMsg{}) // a no-op once p has ended
		case <-done:
		}
	}()
	return func() bool {
		close(done)
		wg.Wait()
		select {
		case <-sig: // it came as p ended on the input failing
			hangUp()
		default:
		}
		// Once hung up it stays caught: a shell closing passes its hangup
		// on, and main is still saving.
		if !hung.Load() {
			signal.Stop(sig)
		}
		return hung.Load()
	}
}

// closedSignal is what closing the terminal sends: SIGHUP; on Windows a
// console window closing (CTRL_CLOSE_EVENT), which Go hands over as SIGTERM.
// Caught, Windows lets the app end on its own for a few seconds before it
// ends it.
func closedSignal() os.Signal {
	if runtime.GOOS == "windows" {
		return syscall.SIGTERM
	}
	return syscall.SIGHUP
}
