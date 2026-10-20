package main

import (
	"testing"

	terminu "github.com/vulcanshen/terminu-go"
	"github.com/vulcanshen/terminu-go/tdptest"
)

// The first launch of every app, as tdptest walks it.
func TestFirstLaunch(t *testing.T) {
	tdptest.Check(t, tdptest.App{Name: "showcase", New: func() *terminu.App { return newShowcase().app }})
}
