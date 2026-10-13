package terminu

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// clock is now; tests set it.
var clock = time.Now

// DatePicker picks a date, or a date and a time (components/input/datetime-picker).
type DatePicker struct {
	Title, Glyph string
	// Value is the value now; zero for none.
	Value time.Time
	// Time adds the hour and the minute (five minutes a step).
	Time bool
	// Min and Max limit the dates; zero for no limit.
	Min, Max time.Time
	// SundayFirst starts a week on Sunday instead of Monday.
	SundayFirst bool
	// Done gets the new value. It is not called when it did not change.
	Done func(time.Time) tea.Cmd
}

func (d DatePicker) typed() bool { return false }

// PickDate opens a date (or date and time) picker.
func (a *App) PickDate(d DatePicker) tea.Cmd { return d.open(a, nil) }

type dateFocus int

const (
	onMonth dateFocus = iota
	onYear
	onDays
	onTime
)

type dateView struct {
	d        DatePicker
	day      time.Time // the calendar's cursor: the date written back
	hour     int
	minute   int // 0–11: five minutes a step
	focus    dateFocus
	onMinute bool // in the time block, the minute column has the focus
	hTop     int  // the first hour shown
	mTop     int  // the first minute shown
	written  func()
}

func (d DatePicker) open(a *App, written func()) tea.Cmd {
	v := d.Value
	if v.IsZero() {
		v = clock()
	}
	dv := &dateView{d: d, focus: onDays, written: written,
		day: dateOnly(v), hour: v.Hour(), minute: v.Minute() / 5}
	dv.day = dv.clamp(dv.day)
	return a.open(dv)
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

func (dv *dateView) inRange(t time.Time) bool {
	return (dv.d.Min.IsZero() || !t.Before(dateOnly(dv.d.Min))) && (dv.d.Max.IsZero() || !t.After(dateOnly(dv.d.Max)))
}

func (dv *dateView) clamp(t time.Time) time.Time {
	if !dv.d.Min.IsZero() && t.Before(dateOnly(dv.d.Min)) {
		return dateOnly(dv.d.Min)
	}
	if !dv.d.Max.IsZero() && t.After(dateOnly(dv.d.Max)) {
		return dateOnly(dv.d.Max)
	}
	return t
}

func (dv *dateView) hint() []KeyHint {
	switch dv.focus {
	case onMonth:
		return []KeyHint{{"Enter", "days"}, {"h|l", "month"}, {"t", "today"}, {"Tab", "year"}, {"Esc", "cancel"}}
	case onYear:
		return []KeyHint{{"Enter", "days"}, {"h|l", "year"}, {"t", "today"}, {"Tab", "days"}, {"Esc", "cancel"}}
	case onTime:
		return []KeyHint{{"Enter", "choose"}, {"t", "today"}, {"Tab", "month"}, {"Esc", "cancel"}}
	}
	if dv.d.Time {
		return []KeyHint{{"Enter", "time"}, {"t", "today"}, {"Tab", "time"}, {"Esc", "cancel"}}
	}
	return []KeyHint{{"Enter", "choose"}, {"t", "today"}, {"Tab", "month"}, {"Esc", "cancel"}}
}

// calendar widths: a day is 4 cells (two digits on the right, two apart).
const calW = 7*4 - 2

func (dv *dateView) contentWidth() int {
	if dv.d.Time {
		return calW + 5 + 6 // "  │  ", then "HH  MM"
	}
	return calW
}

func (dv *dateView) measure(sw, sh int) (int, int) {
	w := dv.contentWidth() + 4
	w = max(w, Width(titleCapsule(dv.d.Glyph, Upper(dv.d.Title), Base, 1<<20))+4)
	saved := dv.focus
	for f := onMonth; f <= onTime; f++ {
		dv.focus = f
		w = max(w, hintWidth(dv.hint())+6)
	}
	dv.focus = saved
	// The month row, the weekdays, six weeks always (Rules F7: the height
	// does not change with the month).
	return min(w, max(min(sw-2, 120), 1)), min(8+4, max(sh-2, 3))
}

// cursor draws s as the cursor: the layer colour, Base bold; dimmed once when
// another part has the focus (Principle P6).
func cursorText(s string, layer Color, bright bool) string {
	c := On(layer).Render(s)
	if !bright {
		return Dim(c)
	}
	return c
}

func (dv *dateView) draw(w, h int, layer Color) []string {
	inner := w - 2
	first := time.Date(dv.day.Year(), dv.day.Month(), 1, 0, 0, 0, 0, time.Local)
	weekStart := time.Monday
	names := []string{"Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"}
	if dv.d.SundayFirst {
		weekStart = time.Sunday
		names = []string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}
	}
	month, year := dv.day.Month().String(), fmt.Sprint(dv.day.Year())
	if dv.focus == onMonth {
		month = cursorText(month, layer, true)
	} else {
		month = Fg(Text).Render(month)
	}
	if dv.focus == onYear {
		year = cursorText(year, layer, true)
	} else {
		year = Fg(Text).Render(year)
	}
	cal := []string{month + "  " + year}
	var head string
	for i, n := range names {
		if i > 0 {
			head += "  "
		}
		head += n
	}
	cal = append(cal, Fg(Overlay0).Render(head))
	offset := (int(first.Weekday()) - int(weekStart) + 7) % 7
	start := first.AddDate(0, 0, -offset)
	today := dateOnly(clock())
	value := time.Time{}
	if !dv.d.Value.IsZero() {
		value = dateOnly(dv.d.Value)
	}
	for wk := range 6 {
		var row string
		for wd := range 7 {
			if wd > 0 {
				row += "  "
			}
			day := start.AddDate(0, 0, wk*7+wd)
			if day.Month() != first.Month() {
				row += "  "
				continue
			}
			text := fmt.Sprintf("%2d", day.Day())
			style := Fg(Text)
			switch {
			case !dv.inRange(day):
				style = Fg(Surface2)
			case day.Equal(value):
				style = Fg(Green)
			}
			if day.Equal(today) {
				style.Underline = true
			}
			if day.Equal(dv.day) {
				row += cursorText(text, layer, dv.focus == onDays)
				continue
			}
			row += style.Render(text)
		}
		cal = append(cal, row)
	}
	dv.hTop = scrollWindow(dv.hTop, dv.hour, 8, 24)
	dv.mTop = scrollWindow(dv.mTop, dv.minute, 8, 12)
	body := make([]string, 0, h-4)
	for i := range 8 {
		line := Pad(cal[i], calW)
		if dv.d.Time {
			line += "  " + Fg(Overlay0).Render("│") + "  " + dv.timeRow(i, layer)
		}
		body = append(body, padRow(line, inner))
	}
	return popupBox{color: layer, glyph: dv.d.Glyph, title: Upper(dv.d.Title), hint: dv.hint()}.draw(w, Pad2(body, inner, h-4))
}

// timeRow is row i of the time block: the hour column and the minute column,
// each scrolled to keep its cursor in sight.
func (dv *dateView) timeRow(i int, layer Color) string {
	col := func(top, cur, n, step int, bright bool, value int) string {
		k := top + i
		if k >= n {
			return "  "
		}
		text := fmt.Sprintf("%02d", k*step)
		if k == cur {
			return cursorText(text, layer, bright)
		}
		if k == value {
			return Fg(Green).Render(text)
		}
		return Fg(Text).Render(text)
	}
	vh, vm := -1, -1
	if !dv.d.Value.IsZero() {
		vh, vm = dv.d.Value.Hour(), dv.d.Value.Minute()/5
	}
	return col(dv.hTop, dv.hour, 24, 1, dv.focus == onTime && !dv.onMinute, vh) + "  " +
		col(dv.mTop, dv.minute, 12, 5, dv.focus == onTime && dv.onMinute, vm)
}

// moveDays moves the cursor n days, staying put when that leaves the range.
func (dv *dateView) moveDays(n int) {
	t := dv.day.AddDate(0, 0, n)
	if dv.inRange(t) {
		dv.day = t
	}
}

// moveMonths keeps the day, or the month's last when it has none.
func (dv *dateView) moveMonths(n int) {
	y, m, d := dv.day.Date()
	first := time.Date(y, m+time.Month(n), 1, 0, 0, 0, 0, time.Local)
	last := first.AddDate(0, 1, -1).Day()
	dv.day = dv.clamp(time.Date(first.Year(), first.Month(), min(d, last), 0, 0, 0, 0, time.Local))
}

func (dv *dateView) result() time.Time {
	if !dv.d.Time {
		return dv.day
	}
	return dv.day.Add(time.Duration(dv.hour)*time.Hour + time.Duration(dv.minute*5)*time.Minute)
}

func (dv *dateView) confirm(a *App) tea.Cmd {
	v := dv.result()
	old := dv.d.Value
	if !old.IsZero() && !dv.d.Time {
		old = dateOnly(old)
	} else if !old.IsZero() {
		old = old.Truncate(time.Minute)
	}
	if v.Equal(old) || dv.d.Done == nil {
		return a.closeTop()
	}
	return tea.Batch(a.closeTop(), then(dv.written, dv.d.Done(v)))
}

func (dv *dateView) key(a *App, key string, _ tea.KeyMsg) (bool, tea.Cmd) {
	switch key {
	case "Tab":
		dv.focus++
		if dv.focus > onTime || dv.focus == onTime && !dv.d.Time {
			dv.focus = onMonth
		}
		return true, nil
	case "t":
		now := clock()
		dv.day = dv.clamp(dateOnly(now))
		if dv.focus == onTime {
			dv.hour, dv.minute = now.Hour(), now.Minute()/5
		}
		return true, nil
	case "Enter":
		switch dv.focus {
		case onMonth, onYear:
			dv.focus = onDays
		case onDays:
			if dv.d.Time {
				dv.focus = onTime
				return true, nil
			}
			return true, dv.confirm(a)
		case onTime:
			return true, dv.confirm(a)
		}
		return true, nil
	case "Space":
		return true, nil
	}
	dir := map[string]int{"h": -1, "←": -1, "l": 1, "→": 1, "k": -1, "↑": -1, "j": 1, "↓": 1}[key]
	vertical := key == "j" || key == "k" || key == "↑" || key == "↓"
	if dir == 0 {
		return false, nil
	}
	switch dv.focus {
	case onMonth:
		if !vertical {
			dv.moveMonths(dir)
		}
	case onYear:
		if !vertical {
			dv.moveMonths(12 * dir)
		}
	case onDays:
		if vertical {
			dv.moveDays(7 * dir)
		} else {
			dv.moveDays(dir)
		}
	case onTime:
		if !vertical {
			dv.onMinute = !dv.onMinute // two columns: a step wraps (Rules K12)
		} else if dv.onMinute {
			dv.minute = (dv.minute + dir + 12) % 12
		} else {
			dv.hour = (dv.hour + dir + 24) % 24
		}
	}
	return true, nil
}

func (dv *dateView) surface() string { return dv.d.Title }

func (dv *dateView) keyRef() []refSection {
	rows := []KeyRow{
		{Key: "Enter", Desc: "to the days; choose the day (the time)"},
		{Key: "t", Desc: "to today (and now)"},
		{Key: "Tab", Desc: "month, year, days, time"},
		{Key: "Esc", Desc: "cancel"},
	}
	return []refSection{{rows: rows}, {title: "move", rows: []KeyRow{
		{Key: "h|l", Desc: "a day; a month or a year; hour and minute"},
		{Key: "j|k", Desc: "a week; the hour or the minute"},
	}}}
}

func (dv *dateView) typing() bool { return false }
