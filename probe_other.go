//go:build !unix

package terminu

// askTerminal does not probe outside unix: an icon is one cell unless a
// variable says otherwise (tdp E5), and the palette stays xterm's.
func askTerminal(bool) (answers, bool) { return answers{}, false }
