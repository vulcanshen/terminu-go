package terminu

import (
	"slices"
	"strings"
	"testing"
	"time"
)

// locCell is the cell of line where sub starts, read from its plain text.
func locCell(t *testing.T, line, sub string) covCCell {
	t.Helper()
	plain := StripStyles(line)
	i := strings.Index(plain, sub)
	if i < 0 {
		t.Fatalf("no %q in %q", sub, plain)
	}
	return covCAt(line, Width(plain[:i]))
}

// TestLocationWork: work running on the place a location row names follows
// the where in Green, the loading icon turning in Green where the glyph
// sits, the row's own glyph back, or none, when it ends; what the place is
// in the middle of stays in the row's Lavender; cut from the front, the work
// stays (tdp panel.10, popup.4; panel-197).
func TestLocationWork(t *testing.T) {
	merge := "\U000F062D"
	for _, iw := range []int{1, 2} {
		withWidths(t, iw, 1)
		l := &List{Columns: []Column{{Title: "Name"}}, Location: "~/repo  Merging: feature/login", LocationGlyph: merge, LocationWork: "Pushing…"}
		l.SetRows([]Row{{ID: "1", Cells: []string{"x"}}})
		line := l.Draw(60, 4, true)[0]
		icon := covCAt(line, 1)
		if !slices.Contains(loadingFrames[:], icon.Ch) || icon.St.Fg != Green {
			t.Errorf("icon %d: the loading icon in Green where the glyph sits: %q %v", iw, icon.Ch, icon.St.Fg)
		}
		if got, want := StripStyles(line), Pad(" "+icon.Ch+" ~/repo  Merging: feature/login Pushing…", 60); got != want || Width(line) != 60 {
			t.Errorf("icon %d: the where, a space, the work:\n%q\n%q", iw, got, want)
		}
		if c := locCell(t, line, "~/repo"); c.St.Fg != Lavender {
			t.Errorf("icon %d: the where in Lavender: %v", iw, c.St.Fg)
		}
		if c := locCell(t, line, "Merging"); c.St.Fg != Lavender {
			t.Errorf("icon %d: the state of the place in Lavender: %v", iw, c.St.Fg)
		}
		if c := locCell(t, line, "Pushing"); c.St.Fg != Green {
			t.Errorf("icon %d: the work in Green: %v", iw, c.St.Fg)
		}

		// Too long: cut from the front, the icon and the work staying, the …
		// in the colour of what it cuts.
		line = l.Draw(30, 4, true)[0]
		plain := strings.TrimRight(StripStyles(line), " ")
		if !strings.HasPrefix(plain, " "+icon.Ch+" …") || !strings.HasSuffix(plain, "feature/login Pushing…") || Width(line) != 30 ||
			Width(plain) != 30-Width(glyphHard) {
			t.Errorf("icon %d: cut in the where: %q", iw, plain)
		}
		if c := covCAt(line, 1+Width(icon.Ch)+1); c.Ch != "…" || c.St.Fg != Lavender {
			t.Errorf("icon %d: the … of the where in Lavender: %q %v", iw, c.Ch, c.St.Fg)
		}
		line = l.Draw(10, 4, true)[0]
		if c := covCAt(line, 1+Width(icon.Ch)+1); c.Ch != "…" || c.St.Fg != Green || !strings.HasSuffix(strings.TrimRight(StripStyles(line), " "), "ing…") {
			t.Errorf("icon %d: the work itself cut, its … in Green: %q %v %q", iw, c.Ch, c.St.Fg, StripStyles(line))
		}

		// Work in the where's place: the row by itself.
		l.Location = ""
		line = l.Draw(60, 4, true)[0]
		if got := StripStyles(line); !strings.HasPrefix(got, " "+icon.Ch+" Pushing…") || locCell(t, line, "Pushing").St.Fg != Green {
			t.Errorf("icon %d: the work with no where: %q", iw, got)
		}

		// It ends: the row's own glyph is back, in Lavender.
		l.Location, l.LocationWork = "~/repo  Merging: feature/login", ""
		line = l.Draw(60, 4, true)[0]
		if c := covCAt(line, 1); c.Ch != merge || c.St.Fg != Lavender || strings.Contains(StripStyles(line), "Pushing") {
			t.Errorf("icon %d: the work done, the glyph back: %q %v", iw, c.Ch, c.St.Fg)
		}

		// A row with no glyph of its own: the loading icon while it runs,
		// none after.
		l.LocationGlyph, l.LocationWork = "", "Pulling…"
		line = l.Draw(60, 4, true)[0]
		if c := covCAt(line, 1); !slices.Contains(loadingFrames[:], c.Ch) || c.St.Fg != Green {
			t.Errorf("icon %d: no glyph, the loading icon still: %q %v", iw, c.Ch, c.St.Fg)
		}
		l.LocationWork = ""
		if got := StripStyles(l.Draw(60, 4, true)[0]); !strings.HasPrefix(got, " ~/repo  Merging") {
			t.Errorf("icon %d: no glyph, none after: %q", iw, got)
		}
		l.Location = ""
		if got := StripStyles(l.Draw(60, 4, true)[0]); strings.TrimSpace(got) != "NAME" {
			t.Errorf("icon %d: nothing to say, no row: %q", iw, got)
		}
	}
}

// TestLocationWorkInPanel: in a panel the work dims with the focus, the
// icon turns with the App's clock, the lib keeps it turning only while work
// runs on a place on the screen; a panel's own work makes the row the
// panel's (tdp panel.10, popup.4, T2; panel-197).
func TestLocationWorkInPanel(t *testing.T) {
	ta := newTestApp(t)
	ta.list.Location, ta.list.LocationWork = "~/repo", "Pushing…"
	row := func() string {
		for _, l := range ta.Frame() {
			if strings.Contains(StripStyles(l), "Pushing…") || strings.Contains(StripStyles(l), "Fetching…") {
				return l
			}
		}
		t.Fatalf("no work on the screen: %q", ta.screenText())
		return ""
	}
	clk := &stillClock{time.UnixMilli(90 * 8 * 1000)}
	ta.testClock = clk
	line := row()
	icon := locCell(t, line, loadingFrames[0])
	if icon.St.Fg != Green || !strings.Contains(StripStyles(line), loadingFrames[0]+" ~/repo Pushing…") {
		t.Errorf("the loading icon in Green, the where, the work: %q %v", StripStyles(line), icon.St.Fg)
	}
	if c := locCell(t, line, "Pushing"); c.St.Fg != Green {
		t.Errorf("the work in Green: %v", c.St.Fg)
	}
	clk.t = clk.t.Add(spinTime)
	if line = row(); !strings.Contains(StripStyles(line), loadingFrames[1]+" ~/repo") {
		t.Errorf("90ms on, the next frame: %q", StripStyles(line))
	}
	ta.testClock = nil

	ta.press("2")
	line = row()
	if c := locCell(t, line, "Pushing"); c.St.Fg != DimColor(Green) {
		t.Errorf("unfocused, the work dims: %v", c.St.Fg)
	}
	if c := covCAt(line, locCell(t, line, "~/repo").Col-2); c.St.Fg != DimColor(Green) {
		t.Errorf("unfocused, the icon dims: %q %v", c.Ch, c.St.Fg)
	}
	ta.press("1")

	ta.spin = false
	if _, cmd := ta.Update(struct{}{}); cmd == nil || !ta.spin {
		t.Error("work running: the ticks keep coming")
	}
	ta.list.LocationWork = ""
	ta.spin = false
	ta.Update(struct{}{})
	if ta.spin {
		t.Error("no work: no ticks")
	}

	// The panel's own work, with no Location of its own: the row is the
	// panel's, not its list's.
	ta.pods.LocationWork = "Fetching…"
	line = row()
	if got := StripStyles(line); strings.Contains(got, "~/repo") || locCell(t, line, "Fetching").St.Fg != Green {
		t.Errorf("the panel's work, its row: %q", got)
	}
	ta.spin = false
	if ta.Update(struct{}{}); !ta.spin {
		t.Error("the panel's work: the ticks keep coming")
	}
}
