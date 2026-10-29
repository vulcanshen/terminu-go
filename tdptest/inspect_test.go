package tdptest

import (
	"testing"
	"time"
)

// TestInspectToastStays: the inspect toast has no timer, and an ordinary
// toast that takes its place gives it back once its own time is up (tdp
// toast.4); Esc does not take it, so closing every popup does not wait for
// it.
func TestInspectToastStays(t *testing.T) {
	ty := newTiny()
	ty.App.Screen().Focused().Inspect = func() []string { return []string{"go.dev/doc"} }
	d := Start(t, ty.App, 80, 24)
	d.Advance(10 * time.Second)
	if !d.Has("INSPECT") || !d.Has("go.dev/doc") {
		t.Fatalf("the inspect toast went in time%s", d.where())
	}
	d.Press("S")
	if !d.Has("Hello") || d.Has("go.dev/doc") {
		t.Errorf("the ordinary toast is not in its place%s", d.where())
	}
	d.Advance(2200 * time.Millisecond)
	if d.Has("Hello") || !d.Has("go.dev/doc") {
		t.Errorf("it is not back once the ordinary toast is gone%s", d.where())
	}
	closeAll(t, d, "an inspect toast")
}
