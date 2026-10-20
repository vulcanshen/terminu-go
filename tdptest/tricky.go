package tdptest

import "strings"

// Tricky are values that have broken screens: control characters and
// escape sequences, wide, joined and combining characters, Nerd Font
// glyphs, empty and very long values. Give them to whatever the app draws
// from outside — a name, a path, a message, a value typed or pasted — and
// the driver checks every frame (tdp L4).
var Tricky = []string{
	"",
	" ",
	"  padded  ",
	"tab\there",
	"line\nbreak",
	"carriage\rreturn",
	"crlf\r\n",
	"nul\x00byte",
	"bell\x07",
	"back\bspace",
	"del\x7f",
	"\x1b",
	"\x1b[31mred\x1b[0m",
	"\x1b[2J\x1b[Hclear",
	"\x1b]0;title\x07",
	"\x1b]52;c;aGk=\x07",
	"\x1b]8;;https://example.com\x1b\\link\x1b]8;;\x1b\\",
	"c1 \U00000085 \U0000009b31m",
	"漢字かな한국어",
	"wide 漢 in the middle",
	"emoji \U0001F44D\U0001F3FD",
	"family \U0001F468\U0000200D\U0001F469\U0000200D\U0001F467",
	"flag \U0001F1F9\U0001F1FC",
	"combining e\U00000301 a\U0000030A",
	"zero\U0000200Bwidth",
	"bidi \U0000202Eevil\U0000202C",
	"bom \U0000FEFF",
	"icon \U0000F07B folder",
	"powerline \U0000E0B0 \U0000E0B6",
	"md \U000F0A9E",
	"[x] [Enter] [g]oto",
	"%s %d %!v",
	"…",
	"─│┌┐└┘━┃",
	strings.Repeat("a", 300),
	strings.Repeat("長", 150),
	"/very/long/path/" + strings.Repeat("directory/", 30) + "file.txt",
	strings.Repeat("word ", 80),
}

// Sizes are terminal sizes that have broken screens: one cell, one row, one
// column, around the default least size (tdp L1), common ones, very wide,
// very tall.
var Sizes = [][2]int{
	{1, 1}, {2, 1}, {1, 2}, {80, 1}, {80, 2}, {80, 5}, {20, 10}, {40, 12}, {60, 20},
	{79, 40}, {80, 39}, {80, 40}, {80, 24}, {120, 40}, {300, 50}, {1000, 10}, {10, 200},
}

// Mashes are keys a user presses faster than the animations play: twice in
// a row, during an opening, closing what is still opening. Give each to
// Mash.
var Mashes = [][]string{
	{"Space", "Space"},
	{"Space", "Esc"},
	{"Space", "j", "j", "Enter"},
	{"?", "?", "Esc", "Esc"},
	{"Esc", "Esc", "Esc", "Esc"},
	{"Enter", "Enter"},
	{"Tab", "Tab", "Tab", "Tab", "Tab"},
	{"q", "Esc", "q", "Esc"},
	{"Space", "G", "Enter", "Esc", "Esc"},
	{"/", "a", "Esc", "Esc"},
}

// Mash presses keys as fast as a user can, the time standing still between
// them, then lets the animations play to their end.
func (d *Driver) Mash(keys ...string) {
	d.t.Helper()
	auto := d.Auto
	d.Auto = false
	d.Press(keys...)
	d.Auto = auto
	d.Settle()
}
