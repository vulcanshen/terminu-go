package terminu

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

// KeepAside copies path to <name>.bak, or the first of .bak.1, .bak.2…
// that is free, never over another, keeping its mode, and returns the copy's
// path (tdp E3, a value that does not fit): the app then writes path again
// with its defaults for the values that did not fit and without the entries
// that did not. It needs no App, for a start with no screen too.
//
// When the copy cannot be made (a folder that cannot be written, a full
// disk) its error says so as the Error toast does, "config.yaml.bak could
// not be made: permission denied; config.yaml left as it was", wrapping
// why: the app does not write path again, uses the defaults for this run
// only, and says the error (an Error toast, or stderr with no screen); the
// next start finds the values again (tdp E3).
func KeepAside(path string) (string, error) {
	to := asideName(path)
	if err := copyNew(path, to); err != nil {
		why := err
		var pe *fs.PathError
		if errors.As(err, &pe) {
			why = pe.Err
		}
		return "", fmt.Errorf("%s could not be made: %w; %s left as it was", filepath.Base(to), why, filepath.Base(path))
	}
	return to, nil
}

// copyNew copies from to a new file to, with from's mode; to is not there
// after a copy that fails.
func copyNew(from, to string) error {
	src, err := os.Open(from)
	if err != nil {
		return err
	}
	defer src.Close()
	info, err := src.Stat()
	if err != nil {
		return err
	}
	dst, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return err
	}
	_, err = io.Copy(dst, src)
	if cerr := dst.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(to)
	}
	return err
}

// Misfit is a value or an entry of a file of the app's own that did not fit
// (tdp E3). Key names it: a value's key ("color"), or which entry it is with
// what it had (`entry 3 (name "")`). Was is what the file had for a value;
// Now is what is used now, "" for an entry taken out (a value whose default
// is empty says so in words: "none").
type Misfit struct{ Key, Was, Now string }

// DidNotFit says what KeepAside and the app's writing path again did (tdp
// E3): one Warning toast naming the file and the copy, as DidNotFitText
// words it, and a WARN in the App Log for each misfit:
// `config.yaml: color "#zzzzzz" did not fit; using "#89b4fa"`, or
// `hosts.yaml: entry 3 (name "") did not fit; taken out`. With copy ""
// (the copy could not be made) it only logs the misfits: the app gives
// KeepAside's error in an Error toast. When the start has more to say, the
// app gives MisfitNotice to Notify with the rest instead.
func (a *App) DidNotFit(path, copy string, misfits ...Misfit) tea.Cmd {
	return a.Notify(MisfitNotice(path, copy, misfits...))
}

// Notice is one row of the toast a start gives, with its own kind (tdp E3,
// several at one start): a location moved, values that did not fit
// (MisfitNotice), a copy that could not be made (KeepAside's error), a value
// naming an entry that is not there. Notify shows them as one toast; with no
// screen, its Lines go to stderr.
type Notice struct {
	Kind ToastKind
	Text string
	// warns are the WARNs logged right after the row: a misfit's.
	warns []string
}

// Lines are what n says with no screen, for stderr a line each after
// "<app>: " (tdp E3, E1): its Text, when it has one, then each WARN Notify
// logs for it, in the words the TUI gives them. It needs no App.
func (n Notice) Lines() []string {
	var out []string
	if n.Text != "" {
		out = append(out, n.Text)
	}
	return append(out, n.warns...)
}

// MisfitNotice is DidNotFit's row, a Warning in DidNotFitText's words,
// carrying a WARN for each misfit that Notify logs right after it. With copy
// "" its Text is "": no row, the WARNs still logged.
func MisfitNotice(path, copy string, misfits ...Misfit) Notice {
	n := Notice{Kind: Warning, Text: DidNotFitText(path, copy, misfits...)}
	file := filepath.Base(path)
	for _, m := range misfits {
		if m.Now == "" {
			n.warns = append(n.warns, file+": "+m.Key+" did not fit; taken out")
			continue
		}
		n.warns = append(n.warns, file+": "+m.Key+" "+strconv.Quote(m.Was)+" did not fit; using "+strconv.Quote(m.Now))
	}
	return n
}

// Notify shows notices as one toast, a row each in the order given, its kind
// the strongest: Error, then Warning, else the first row's (tdp E3, several
// at one start). In the App Log each row is an entry of its own at its own
// kind, a misfit's WARNs right after its row. A notice with no Text has no
// row; with no rows there is no toast.
func (a *App) Notify(notices ...Notice) tea.Cmd {
	var rows []string
	kind := Info
	for _, n := range notices {
		if n.Text != "" {
			a.Log(n.Kind, n.Text)
			if len(rows) == 0 || n.Kind == Error || n.Kind == Warning && kind != Error {
				kind = n.Kind
			}
			rows = append(rows, n.Text)
		}
		for _, w := range n.warns {
			a.Log(Warning, w)
		}
	}
	if len(rows) == 0 {
		return nil
	}
	return a.showToast(kind, rows)
}

// DidNotFitText is DidNotFit's sentence, for an app with no screen to write
// to stderr, where it does not mix with what it prints (tdp E3): "3 values
// in config.yaml did not fit: defaults used, yours kept in config.yaml.bak";
// "2 entries in hosts.yaml did not fit: taken out, yours kept in
// hosts.yaml.bak"; with both, "3 values and 2 entries in config.yaml did not
// fit: defaults used and entries taken out, yours kept in config.yaml.bak".
// It is "" with no misfits or no copy.
func DidNotFitText(path, copy string, misfits ...Misfit) string {
	if copy == "" {
		return ""
	}
	values, entries := 0, 0
	for _, m := range misfits {
		if m.Now == "" {
			entries++
		} else {
			values++
		}
	}
	count := func(n int, one, many string) string {
		if n == 1 {
			return "1 " + one
		}
		return strconv.Itoa(n) + " " + many
	}
	file, kept := filepath.Base(path), ", yours kept in "+filepath.Base(copy)
	switch {
	case values > 0 && entries > 0:
		return count(values, "value", "values") + " and " + count(entries, "entry", "entries") + " in " + file +
			" did not fit: defaults used and entries taken out" + kept
	case values > 0:
		return count(values, "value", "values") + " in " + file + " did not fit: defaults used" + kept
	case entries > 0:
		return count(entries, "entry", "entries") + " in " + file + " did not fit: taken out" + kept
	}
	return ""
}
