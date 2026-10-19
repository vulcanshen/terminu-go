package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"
)

// inputsScreen is [I]NPUTS: every kind of input. [1] INPUTS has one row per
// input and per way of drawing one, Enter opening its popup; [2] FILES is a
// list whose [/] Find opens a general finder with a preview.
type inputsScreen struct {
	s        *showcase
	screen   *terminu.Screen
	values   *terminu.Values
	inputs   *terminu.Panel
	files    *terminu.Panel
	fileList *terminu.List
	disabled bool
	index    *terminu.Finder // the slow index, while it fills
	indexed  int
}

// The showcase's own glyphs for its inputs (the app's choice: what each
// input edits).
const (
	glyphPerson   = "\U000F0004" // nf-md-account
	glyphFolder   = "\U000F024B" // nf-md-folder
	glyphKey      = "\U000F0306" // nf-md-key
	glyphLan      = "\U000F0317" // nf-md-lan
	glyphComment  = "\U000F0182" // nf-md-comment_text
	glyphPalette  = "\U000F03D8" // nf-md-palette
	glyphCalendar = "\U000F00ED" // nf-md-calendar
	glyphTune     = "\U000F062E" // nf-md-tune
	glyphSearch   = "\U0000F002" // nf-fa-search
	glyphTag      = "\U000F04F9" // nf-md-tag
)

func newInputsScreen(s *showcase) *inputsScreen {
	is := &inputsScreen{s: s}
	a := s.app
	var (
		name, nickname = "assets", ""
		path           = filepath.Join(projectDir, "assets/images/backgrounds/very-long-directory-name")
		password       = true
		pin            = ""
		port, ratio    = "22", "0.5"
		comment        = "Thanks for the quick reply. The new build fixes the crash on start.\n\nOne more thing: the export button"
		profiles       = []string{"clock", "matrix", "pipes", "rain", "stars", "tunnel", "fire"}
		profile        = 0
		namespaces     = []string{"default", "kube-system", "monitoring", "ingress-nginx", "cert-manager", "argocd", "logging", "vault"}
		nsChecked      = []bool{true, true, false, false, false, false, false, false}
		segments       = []string{"Model", "Usage", "Cost", "Context", "Git", "Branch", "Clock"}
		brightness     = 70.0
		opacity        = 0.85
		today          = time.Now()
		due            = time.Date(today.Year(), today.Month(), today.Day()+3, 0, 0, 0, 0, time.Local)
		meeting        = time.Date(today.Year(), today.Month(), today.Day(), 10, 7, 0, 0, time.Local)
		background     = terminu.Hex("#2A2A3C")
		sshDir         = fakeSSH()
		home, _        = os.UserHomeDir()
		downloads      = filepath.Join(home, "Downloads")
		host           = "web-01"
		identity       = filepath.Join(sshDir, "id_ed25519")
		archive        = ""
		highlight      terminu.Color // none yet: the theme's own
		refresh        = ""
		wrongPIN       = 0
		pausedUntil    time.Time
	)
	disabled := func() bool { return is.disabled }
	is.values = &terminu.Values{Header: [2]string{"Input", "Value"}}
	is.values.Fields = []*terminu.Field{
		{Label: "Name", Value: func() string { return name }, Edit: func() tea.Cmd {
			return a.Text(terminu.TextInput{Title: "Rename", Subject: name, Name: "Name", Glyph: glyphPencil, Prompt: "New name for " + name, Verb: "rename", Old: name,
				Check: func(v string) error {
					if v == "" {
						return fmt.Errorf("Name can't be empty")
					}
					return nil
				},
				Done: func(v string) tea.Cmd { name = v; return nil }})
		}},
		{Label: "Nickname", Value: func() string { return nickname }, Empty: "none", Edit: func() tea.Cmd {
			return a.Text(terminu.TextInput{Title: "Nickname", Glyph: glyphPerson, Prompt: "A nickname, if you like", Verb: "save",
				Done: func(v string) tea.Cmd { nickname = v; return nil }})
		}},
		// A directory: the file-picker's "choosing a directory", opened at it.
		{Label: "Path", Path: true, Value: func() string { return path }, Edit: func() tea.Cmd {
			return a.PickFile(terminu.FilePicker{Title: "Path", Glyph: glyphFolder, Root: path, Value: path, Dirs: true,
				Done: func(p string) tea.Cmd { path = p; return nil }})
		}},
		{Label: "Password", Value: func() string { return dots(password) }, Empty: "not set", Disabled: disabled, Edit: func() tea.Cmd {
			return a.Password(terminu.PasswordInput{Title: "Password", Glyph: glyphKey, Prompt: "The password for web-01", Verb: "save", Set: password,
				Done: func(v string, clear bool) tea.Cmd { password = !clear; return nil }})
		}},
		// The unlock look has no prompt: its title says what to type.
		{Label: "PIN", Value: func() string { return dots(pin != "") }, Empty: "not set", Edit: func() tea.Cmd {
			return a.Password(terminu.PasswordInput{Title: "New PIN", Glyph: glyphLock, Verb: "save", Set: pin != "", Unlock: true, MaxLen: 12,
				Done: func(v string, clear bool) tea.Cmd { pin = v; return nil }})
		}},
		// Three wrong PINs pause it for 30 seconds, counted down on the error
		// row; the PIN is 1234.
		{Label: "Unlock", Value: func() string { return "the PIN is 1234" }, Edit: func() tea.Cmd {
			return a.Password(terminu.PasswordInput{Title: "Unlock", Glyph: glyphLock, Verb: "unlock", Unlock: true, MaxLen: 4,
				Check: func(v string) error {
					if v == "1234" {
						wrongPIN = 0
						return nil
					}
					if wrongPIN++; wrongPIN%3 == 0 {
						pausedUntil = time.Now().Add(30 * time.Second)
					}
					return fmt.Errorf("Wrong PIN")
				},
				PausedUntil: func() time.Time { return pausedUntil },
				Done:        func(string, bool) tea.Cmd { return a.Toast(terminu.Success, "Unlocked") }})
		}},
		// A control character in a value is written by name, in Red: a tab too.
		{Label: "Separator", Value: func() string { return "\t" }},
		// A new thing with a name the app proposes: Enter takes it.
		{Label: "Archive", Value: func() string { return archive }, Empty: "not packed", Edit: func() tea.Cmd {
			return a.Text(terminu.TextInput{Title: "Pack", Name: "Name", Glyph: glyphPencil, Prompt: "The archive to pack the files into", Verb: "pack",
				Suggest: "assets.zip",
				Check: func(v string) error {
					if v == "" {
						return terminu.ErrEmpty
					}
					return nil
				},
				Done: func(v string) tea.Cmd { archive = v; return nil }})
		}},
		// A number with a unit that may be empty: empty means the default.
		{Label: "Refresh", Value: func() string {
			if refresh == "" {
				return ""
			}
			return refresh + " seconds"
		}, Empty: "30 seconds (default)", Edit: func() tea.Cmd {
			return a.Number(terminu.NumberInput{Title: "Refresh", Glyph: glyphTune, Name: "Refresh", Prompt: "Seconds between runs while nothing happens, 1-60",
				Verb: "save", Old: refresh, Min: 1, Max: 60, Optional: true, Unit: "seconds", Done: func(v string) tea.Cmd { refresh = v; return nil }})
		}},
		{Label: "Port", Value: func() string { return port }, Edit: func() tea.Cmd {
			return a.Number(terminu.NumberInput{Title: "Port", Glyph: glyphLan, Name: "Port", Prompt: "The SSH port, 1-65535", Verb: "save",
				Old: port, Min: 1, Max: 65535, Done: func(v string) tea.Cmd { port = v; return nil }})
		}},
		{Label: "Ratio", Value: func() string { return ratio }, Edit: func() tea.Cmd {
			return a.Number(terminu.NumberInput{Title: "Ratio", Glyph: glyphTune, Name: "Ratio", Prompt: "A ratio from -10 to 10", Verb: "save",
				Old: ratio, Min: -10, Max: 10, Negative: true, Decimal: true, Done: func(v string) tea.Cmd { ratio = v; return nil }})
		}},
		{Label: "Comment", Value: func() string { return firstLine(comment) }, Empty: "none", Edit: func() tea.Cmd {
			return a.Textarea(terminu.Textarea{Title: "Comment", Glyph: glyphComment, Old: comment,
				Done: func(v string) tea.Cmd { comment = v; return nil }})
		}},
		{Label: "Profile", Value: func() string { return profiles[profile] }, Edit: func() tea.Cmd {
			return a.Select(terminu.Select{Title: "Profile", Glyph: glyphTag, Options: profiles, Selected: profile,
				Keys: []string{"1", "2", "3", "4", "5", "6", "7"}, Done: func(i int) tea.Cmd { profile = i; return nil }})
		}},
		{Label: "Namespaces", Value: func() string { return joinChecked(namespaces, nsChecked) }, Empty: "none", Edit: func() tea.Cmd {
			// kube-system and vault are out of reach: options not available,
			// kube-system ticked from before (components/input/checkbox).
			return a.Checks(terminu.Checks{Title: "Namespaces", Glyph: glyphTag, Options: namespaces, Checked: nsChecked,
				Unavailable: func(i int) bool { return i == 1 || i == 7 },
				Done:        func(c []bool) tea.Cmd { nsChecked = c; return nil }})
		}},
		{Label: "Segments", Value: func() string { return strings.Join(segments, ", ") }, Edit: func() tea.Cmd {
			return a.Reorder(terminu.Reorder{Title: "Segments", Options: segments, Done: func(o []int) tea.Cmd {
				next := make([]string, len(o))
				for i, k := range o {
					next[i] = segments[k]
				}
				segments = next
				return nil
			}})
		}},
		{Label: "Layout", Options: []string{"row", "column", "grid"}},
		{Label: "Order", Options: []string{"asc", "desc"}},
		{Label: "Notify", Options: []string{"notify on errors"}, Multi: true, Checked: []bool{true}},
		// No display here: X11 is not available (components/input/checkbox).
		{Label: "Agent", Options: []string{"forward", "X11", "compression"}, Multi: true, Checked: []bool{true, false, true},
			OptionUnavailable: func(i int) bool { return i == 1 }},
		{Label: "Brightness", Value: func() string { return terminu.SliderTrack(brightness, 0, 100, 12, terminu.Text) }, Edit: func() tea.Cmd {
			return a.Open(terminu.Slider{Title: "Brightness", Glyph: glyphTune, Bottom: 0, Top: 100, Value: brightness,
				Done: func(v float64) tea.Cmd { brightness = v; return nil }})
		}},
		{Label: "Opacity", Value: func() string { return terminu.SliderTrack(opacity, 0, 1, 12, terminu.Text) }, Edit: func() tea.Cmd {
			return a.Open(terminu.Slider{Title: "Opacity", Glyph: glyphTune, Bottom: 0, Top: 1, Step: 0.05, Value: opacity,
				Done: func(v float64) tea.Cmd { opacity = v; return nil }})
		}},
		{Label: "Due date", Value: func() string { return due.Format("2006-01-02") }, Edit: func() tea.Cmd {
			return a.PickDate(terminu.DatePicker{Title: "Due date", Glyph: glyphCalendar, Value: due,
				Min: today.AddDate(0, 0, -7), Max: today.AddDate(0, 0, 40), Done: func(t time.Time) tea.Cmd { due = t; return nil }})
		}},
		{Label: "Meeting", Value: func() string { return meeting.Format("2006-01-02 15:04") }, Edit: func() tea.Cmd {
			return a.PickDate(terminu.DatePicker{Title: "Meeting", Glyph: glyphCalendar, Value: meeting, Time: true,
				Done: func(t time.Time) tea.Cmd { meeting = t; return nil }})
		}},
		{Label: "Background", Value: func() string { return terminu.ColorText(background) }, Edit: func() tea.Cmd {
			return a.PickColor(terminu.ColorPicker{Title: "Background", Glyph: glyphPalette, Value: background,
				Done: func(c terminu.Color) tea.Cmd { background = c; return nil }})
		}},
		// No colour yet: old says none, new starts from #000000.
		{Label: "Highlight", Value: func() string {
			if !highlight.Set {
				return ""
			}
			return terminu.ColorText(highlight)
		}, Empty: "the theme's own", Edit: func() tea.Cmd {
			return a.PickColor(terminu.ColorPicker{Title: "Highlight", Glyph: glyphPalette, Value: highlight,
				Done: func(c terminu.Color) tea.Cmd { highlight = c; return nil }})
		}},
		{Label: "Identity file", Path: true, Value: func() string { return identity }, Edit: func() tea.Cmd {
			return a.PickFile(terminu.FilePicker{Title: "Identity file", Glyph: glyphKey, Root: sshDir, Value: identity,
				Done: func(p string) tea.Cmd { identity = p; return nil }})
		}},
		{Label: "Remote", Value: func() string { return "" }, Empty: "ssh decides", Disabled: disabled},
		// Under the home directory, which shows as ~.
		{Label: "Download to", Path: true, Value: func() string { return downloads }, Edit: func() tea.Cmd {
			return a.PickFile(terminu.FilePicker{Title: "Download to", Glyph: glyphFolder, Root: downloads, Value: downloads, Dirs: true,
				Done: func(p string) tea.Cmd { downloads = p; return nil }})
		}},
		{Label: "Host", Value: func() string { return host }, Edit: func() tea.Cmd {
			return a.Find(&terminu.Finder{Title: "Host", Glyph: glyphLan, Verb: "choose", Items: hostItems(), Preview: hostPreview,
				Done: func(it terminu.FinderItem) tea.Cmd { host = it.Text; return nil }})
		}},
	}
	is.inputs = &terminu.Panel{Label: "Inputs", Body: is.values}
	is.inputs.Actions = is.stateActions(is.inputs)

	is.fileList = &terminu.List{
		Columns:   []terminu.Column{{Title: "Name", Path: true}, {Title: "Size", Number: true, Drop: 2}, {Title: "Modified", Drop: 1}},
		EmptyFact: "No files here",
		EmptyHint: "Press [Space] to see what you can do here",
	}
	is.fileList.SetRows(fileRows())
	is.fileList.SortBy(terminu.SortKey{Col: 0})
	is.fileList.Location = projectDir // where the files are (layout/panel, location row)
	// This panel's hint lists only what can be pressed now (Rules M6: the
	// app's choice): View, disabled, leaves it.
	is.files = &terminu.Panel{Label: "Files", Body: is.fileList, HintPressableOnly: true}
	is.files.Actions = append([]terminu.Action{
		{Key: "/", Label: "Find", Desc: "search the files, with a preview", Scope: terminu.PanelScope, Hint: "find", Do: is.find},
		{Key: "S", Label: "Slow index", Desc: "a search whose results keep arriving", Scope: terminu.PanelScope, Do: is.slowIndex},
		{Key: "L", Label: "Copy URL", Desc: "the file's address", Scope: terminu.ItemScope,
			Do: func() tea.Cmd { return a.Toast(terminu.Info, "Copied the URL") }},
		// This panel gives V to an action of its own: the splash does not open
		// here (Rules S1).
		{Key: "V", Label: "View", Desc: "the file's text", Scope: terminu.ItemScope, Hint: "view", Do: is.view,
			Disabled: func() bool { return is.disabled }},
	}, is.stateActions(is.files)...)

	is.screen = &terminu.Screen{Name: "Inputs", Key: "I", Desc: "every kind of input",
		Panels: []*terminu.Panel{is.inputs, is.files}, NarrowBelow: 60, Layout: halves}
	return is
}

// halves puts two panels side by side.
func halves(w, h int) []terminu.Rect {
	left := w / 2
	return []terminu.Rect{{X: 0, Y: 0, W: left, H: h}, {X: left, Y: 0, W: w - left, H: h}}
}

func dots(set bool) string {
	if set {
		return "••••••••"
	}
	return ""
}

func firstLine(s string) string {
	line, rest, more := strings.Cut(s, "\n")
	if more && rest != "" {
		return line + "…"
	}
	return line
}

func joinChecked(opts []string, checked []bool) string {
	var on []string
	for i, o := range opts {
		if i < len(checked) && checked[i] {
			on = append(on, o)
		}
	}
	return strings.Join(on, ", ")
}

// stateActions are the showcase's own switches for this screen's panels.
func (is *inputsScreen) stateActions(p *terminu.Panel) []terminu.Action {
	a := is.s.app
	return []terminu.Action{
		{Label: "Empty", Desc: "clear this panel, or fill it again", Scope: terminu.PanelScope,
			Hidden: func() bool { return p != is.files },
			Do: func() tea.Cmd {
				if is.fileList.Empty() {
					is.fileList.SetRows(fileRows())
				} else {
					is.fileList.SetRows(nil)
				}
				return nil
			}},
		{Label: "Loading", Desc: "start loading, or stop", Scope: terminu.PanelScope,
			Do: func() tea.Cmd { p.Loading = !p.Loading; return nil }},
		{Label: "Disable", Desc: "some rows cannot be used now", Scope: terminu.PanelScope,
			Hidden: func() bool { return p != is.inputs },
			Do:     func() tea.Cmd { is.disabled = !is.disabled; return nil }},
		{Label: "Toast info", Desc: "a short message", Scope: terminu.PanelScope,
			Do: func() tea.Cmd { return a.Toast(terminu.Info, "Saved") }},
	}
}

func humanSize(n int) string {
	switch {
	case n < 1<<10:
		return fmt.Sprintf("%d B", n)
	case n < 1<<20:
		return fmt.Sprintf("%d KB", n>>10)
	}
	return fmt.Sprintf("%d MB", n>>20)
}

// view opens the file under the cursor in a viewer.
func (is *inputsScreen) view() tea.Cmd {
	row, ok := is.fileList.Current()
	if !ok {
		return nil
	}
	b, err := os.ReadFile(row.ID)
	if err != nil {
		return is.s.app.Error("View", err.Error())
	}
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	lang := row.ID // coloured as the file's name says, as in the preview
	if bytes.IndexByte(b, 0) >= 0 {
		lines, lang = []string{"(binary)"}, ""
	}
	return is.s.app.Viewer(&terminu.Viewer{Title: "File", Subject: row.Cells[0], Glyph: glyphYaml, Lines: lines, Lang: lang})
}

func (is *inputsScreen) find() tea.Cmd {
	return is.s.app.Find(&terminu.Finder{Title: "Search", Glyph: glyphSearch, Verb: "open", Items: fileItems(), Files: true,
		Done: func(it terminu.FinderItem) tea.Cmd { return is.s.app.Toast(terminu.Info, "Opened "+it.Text) }})
}

type indexTick struct{}

// slowIndex is a finder whose results arrive one by one, the loading icon
// turning until the last.
func (is *inputsScreen) slowIndex() tea.Cmd {
	is.index = &terminu.Finder{Title: "Search", Glyph: glyphSearch, Verb: "open", Loading: true, Files: true,
		Done: func(it terminu.FinderItem) tea.Cmd { return is.s.app.Toast(terminu.Info, "Opened "+it.Text) }}
	is.indexed = 0
	return tea.Batch(is.s.app.Find(is.index), tea.Tick(time.Second, func(time.Time) tea.Msg { return indexTick{} }))
}

func (is *inputsScreen) indexStep() tea.Cmd {
	if is.index == nil {
		return nil
	}
	items := fileItems()
	is.index.Items = append(is.index.Items, items[is.indexed])
	is.indexed++
	if is.indexed == len(items) {
		is.index.Loading = false
		is.index = nil
		return nil
	}
	return tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg { return indexTick{} })
}

// hosts are pretend ssh hosts for a finder whose preview the app gives.
var hosts = []struct{ name, addr, user, note string }{
	{"web-01", "10.0.0.5", "deploy", "the public web server"},
	{"web-02", "10.0.0.6", "deploy", "the second web server"},
	{"db-primary", "10.0.4.21", "postgres", "the production database"},
	{"db-replica", "10.0.4.22", "postgres", "a read-only copy of the database"},
	{"bastion-02", "203.0.113.7", "jump", "the way in from outside"},
	{"build", "10.0.8.3", "ci", "where the images are built"},
}

func hostItems() []terminu.FinderItem {
	var items []terminu.FinderItem
	for _, h := range hosts {
		items = append(items, terminu.FinderItem{ID: h.name, Text: h.name})
	}
	return items
}

// hostPreview is the app's own preview: no file, so no facts under it.
func hostPreview(it terminu.FinderItem) (string, []string) {
	for _, h := range hosts {
		if h.name == it.ID {
			return h.name, []string{"HostName  " + h.addr, "User      " + h.user, "Port      22", "", h.note}
		}
	}
	return it.Text, nil
}

// fakeSSH makes a pretend ~/.ssh for the file-picker, so the showcase never
// shows the real one.
func fakeSSH() string {
	dir := filepath.Join(dataDir(), ".ssh")
	put := func(name, text string) {
		p := filepath.Join(dir, name)
		_ = os.MkdirAll(filepath.Dir(p), 0o700)
		_ = os.WriteFile(p, []byte(text), 0o600)
	}
	put("config", "Host web-01\n  HostName 10.0.0.5\n  User deploy\n  IdentityFile ~/.ssh/id_ed25519\n")
	put("id_ed25519", "-----BEGIN OPENSSH PRIVATE KEY-----\n(a fake key for the showcase)\n-----END OPENSSH PRIVATE KEY-----\n")
	put("id_ed25519.pub", "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFAKEFAKEFAKE showcase@example\n")
	put("known_hosts", "10.0.0.5 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFAKE\n")
	put("work/id_work", "-----BEGIN OPENSSH PRIVATE KEY-----\n(another fake key)\n-----END OPENSSH PRIVATE KEY-----\n")
	put("work/keys/a.pem", "-----BEGIN CERTIFICATE-----\n(fake)\n-----END CERTIFICATE-----\n")
	put("old/b.pem", "-----BEGIN CERTIFICATE-----\n(fake)\n-----END CERTIFICATE-----\n")
	return dir
}
