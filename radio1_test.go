package terminu

import (
	"fmt"
	"strings"
	"testing"
)

// TestRadio1ValuesAtMostFive: a one-value-per-row panel draws at most five
// options in place, as a form does; more is a mistake in the program, the
// panel panicking as it draws, radio or checkbox group (tdp radio.1, form.5).
func TestRadio1ValuesAtMostFive(t *testing.T) {
	draw := func(f *Field) (msg string) {
		defer func() {
			if r := recover(); r != nil {
				msg = fmt.Sprint(r)
			}
		}()
		(&Values{Fields: []*Field{{Label: "Name", Value: func() string { return "web" }}, f}}).Draw(60, 12, true)
		return ""
	}
	five := []string{"one", "two", "three", "four", "five"}
	if msg := draw(&Field{Label: "Count", Options: five}); msg != "" {
		t.Errorf("five options: %s", msg)
	}
	six := append(five, "six")
	for _, multi := range []bool{false, true} {
		msg := draw(&Field{Label: "Count", Options: six, Multi: multi, Checked: make([]bool, 6)})
		if !strings.Contains(msg, `value field "Count" has 6 options; more than five go in a select or a checkbox popup`) {
			t.Errorf("six options, Multi %v: %q", multi, msg)
		}
	}
}
