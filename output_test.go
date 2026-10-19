package terminu

import (
	"io"
	"os"
	"testing"
)

// bubbletea uses the output as the terminal only when it is a file with Fd.
var _ interface {
	io.ReadWriteCloser
	Fd() uintptr
} = screenOutput{}

func TestE5OutputDropsEraseRight(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	in := "\x1b[38;2;1;2;3m┃ab┃\x1b[K\r\n\x1b[2K┃cd┃\x1b[K"
	n, err := screenOutput{w}.Write([]byte(in))
	if err != nil || n != len(in) {
		t.Fatalf("Write = %d, %v; want %d, nil", n, err, len(in))
	}
	w.Close()
	got, _ := io.ReadAll(r)
	want := "\x1b[38;2;1;2;3m┃ab┃\r\n\x1b[2K┃cd┃"
	if string(got) != want {
		t.Errorf("wrote %q, want %q", got, want)
	}
}

// TestClipboardGoesWithTheNextFrame: a copy goes to the terminal's clipboard
// (OSC 52) in the same write as the next frame, before it.
func TestClipboardGoesWithTheNextFrame(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	Clipboard("kubectl get pods")
	screenOutput{w}.Write([]byte("frame\x1b[K"))
	screenOutput{w}.Write([]byte("next"))
	w.Close()
	got, _ := io.ReadAll(r)
	if want := "\x1b]52;c;a3ViZWN0bCBnZXQgcG9kcw==\aframenext"; string(got) != want {
		t.Errorf("wrote %q, want %q", got, want)
	}
}
