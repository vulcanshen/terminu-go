package main

import (
	_ "embed"

	terminu "github.com/vulcanshen/terminu-go"
)

// icon is terminu's own mark, a copy of tdp's principle/icon.svg (v0.5.4):
// the showcase draws it in its splash (tdp S5, E8).
//
//go:embed icon.svg
var icon []byte

// splash is the showcase's easter egg, V on a panel (Rules S).
func splash() terminu.Splash {
	s, err := terminu.NewSplash(icon, "showcase", version, "Every component of terminu design, in one app")
	if err != nil {
		panic(err) // the embedded icon is broken: a mistake in the program
	}
	return s
}
