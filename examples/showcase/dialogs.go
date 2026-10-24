package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"
)

// moveAside breaks a pretend settings file and asks what to do with it, as
// an app does when it cannot read a file of its own (Rules E3).
func (ds *dialogsScreen) moveAside() tea.Cmd {
	a := ds.s.app
	path := filepath.Join(dataDir(), "settings.yaml")
	if err := os.MkdirAll(dataDir(), 0o755); err == nil {
		err = os.WriteFile(path, []byte("theme: [dark\n"), 0o600)
	}
	// Moved aside, the showcase goes on as on its first start, which makes
	// no settings file: the next one comes with the first save.
	return a.MoveAside(path, errors.New("line 1: did not find expected ',' or ']'"), func() tea.Cmd {
		return a.Toast(terminu.Info, "Settings back to their defaults, settings.yaml moved aside")
	})
}

// moveAsideRecord breaks one of a few pretend records kept one per file and
// asks what to do with it: moved aside, the showcase reads the rest (Rules
// E3).
func (ds *dialogsScreen) moveAsideRecord() tea.Cmd {
	a := ds.s.app
	dir := filepath.Join(dataDir(), "records")
	_ = os.MkdirAll(dir, 0o755)
	for _, n := range []string{"alpha", "beta"} {
		_ = os.WriteFile(filepath.Join(dir, n+".yaml"), []byte("name: "+n+"\n"), 0o600)
	}
	path := filepath.Join(dir, "gamma.yaml")
	_ = os.WriteFile(path, []byte("name: [gamma\n"), 0o600)
	return a.MoveAsideRecord(path, errors.New("line 1: did not find expected ',' or ']'"), func() tea.Cmd {
		read, _ := filepath.Glob(filepath.Join(dir, "*.yaml"))
		return a.Toast(terminu.Info, fmt.Sprintf("Read %d records, gamma.yaml moved aside", len(read)))
	})
}

// dialogsScreen is [D]IALOGS: every kind of dialog and how they stack. Each
// row of [1] DIALOGS is one scenario; Enter opens it.
type dialogsScreen struct {
	s       *showcase
	screen  *terminu.Screen
	list    *terminu.List
	loading *terminu.Viewer // the loading popup, while its lines arrive
	loaded  int
}

type scenario struct {
	name, kind, desc string
	open             func() tea.Cmd
}

// The showcase's own glyphs for its dialogs (the app's choice).
const (
	glyphYaml    = "\U000F0219" // nf-md-file_document
	glyphConsole = "\U000F018D" // nf-md-console
	glyphServer  = "\U000F048B" // nf-md-server
	glyphProfile = "\U000F0004" // nf-md-account
	glyphList    = "\U000F0279" // nf-md-format_list_bulleted
)

type toastLater struct {
	kind terminu.ToastKind
	text string
}

type loadTick struct{}

func newDialogsScreen(s *showcase) *dialogsScreen {
	ds := &dialogsScreen{s: s}
	scenarios := ds.scenarios()
	ds.list = &terminu.List{Columns: []terminu.Column{{Title: "Scenario"}, {Title: "Kind", Drop: 2}, {Title: "What it shows", Shrink: true, Drop: 1}}}
	rows := make([]terminu.Row, len(scenarios))
	for i, sc := range scenarios {
		rows[i] = terminu.Row{ID: sc.name, Cells: []string{sc.name, sc.kind, sc.desc},
			Styled: [][]terminu.Span{2: {{Text: sc.desc, Style: terminu.Fg(terminu.Overlay0)}}}}
	}
	ds.list.SetRows(rows)
	panel := &terminu.Panel{Label: "Dialogs", Body: ds.list, Actions: []terminu.Action{
		{Key: "Enter", Label: "Open", Desc: "the scenario under the cursor", Scope: terminu.ItemScope, Hint: "open",
			Do: func() tea.Cmd {
				for _, sc := range scenarios {
					if r, ok := ds.list.Current(); ok && sc.name == r.ID {
						return sc.open()
					}
				}
				return nil
			}},
	}}
	ds.screen = &terminu.Screen{Name: "Dialogs", Key: "D", Desc: "every kind of dialog, and how they stack",
		Panels: []*terminu.Panel{panel}}
	return ds
}

func (ds *dialogsScreen) scenarios() []scenario {
	a := ds.s.app
	toast := func(text string) func() tea.Cmd {
		return func() tea.Cmd { return a.Toast(terminu.Info, text) }
	}
	return []scenario{
		{"Menu: Open with", "menu", "the cursor's group first, then titled groups", ds.openWith},
		{"Menu: Sort by", "menu", "two steps, two popups: the column, then the order", ds.sortBy},
		{"Menu: Worklist", "menu", "Enter opens a row's full text over the menu", ds.worklist},
		{"Menu: long", "menu", "more rows than fit: the menu scrolls", ds.longMenu},
		{"Menu: long descriptions", "menu", "descriptions cut with …", ds.longDescriptions},
		{"Confirm", "confirm", "Enter accepts, Esc cancels, Space does nothing", func() tea.Cmd {
			return a.Confirm(terminu.Ask{Title: "Delete", Subject: "web-a8k2", Question: "Delete web-a8k2?", Verb: "delete"}, toast("Deleted web-a8k2"))
		}},
		{"Confirm: with details", "confirm", "what to read first, a warning in Peach", func() tea.Cmd {
			return a.Confirm(terminu.Ask{Title: "Connect", Question: "Connect to prod-db-01?", Verb: "connect",
				Details: []string{"Host  10.0.4.21:5432", "User  deploy", "Via   bastion-02", "", "This is the production database."},
				Warn:    []int{4}}, toast("Connected to prod-db-01"))
		}},
		{"Confirm: long", "confirm", "the details scroll with [j]|[k], the question stays", func() tea.Cmd {
			var details []string
			for i := range 60 {
				details = append(details, fmt.Sprintf("assets/images/backgrounds/landscape-%02d.png", i+1))
			}
			return a.Confirm(terminu.Ask{Title: "Delete", Question: "Delete these 60 files?", Verb: "delete", Details: details},
				toast("Deleted 60 files"))
		}},
		{"Form: New host", "form", "every kind of field; submit empty to see the errors", func() tea.Cmd { return a.Form(ds.hostForm(nil, false)) }},
		{"Form: open at Port", "form", "opened from a field: the focus starts there", func() tea.Cmd {
			f := ds.hostForm(nil, false)
			f.Focus = f.Fields[2]
			return a.Form(f)
		}},
		{"Form: New profile", "form", "stacked and side-by-side fields mixed", ds.profileForm},
		{"Form: long", "form", "more fields than fit: only the fields scroll; Tab to the button and back", ds.longForm},
		{"Form: save fails", "form", "the action fails: an error popup over the form", func() tea.Cmd {
			return a.Form(ds.hostForm(nil, true))
		}},
		{"Form: with actions", "form", "more buttons: [Test] and [Scan] keep the form open, [Save As] submits too, [Connect] takes long; Host bad-… fails", ds.actionsForm},
		{"Note viewer", "note", "its own keys [/] [n] [N] [y]; [v] then [v] again selects text", ds.yaml},
		{"Popup with tabs", "note", "tabs after the title; [h]|[l] switch", ds.devtools},
		{"Viewer: a table", "note", "lines kept on their rows and cut, no line numbers", ds.table},
		{"Loading popup", "note", "lines arriving: the loading icon turns", ds.loadingPopup},
		{"Popup: long title", "note", "a title too long is cut with …", func() tea.Cmd {
			return a.Viewer(&terminu.Viewer{Title: "File", Glyph: glyphYaml,
				Subject: "/srv/projects/showcase/assets/images/backgrounds/very-long-directory-name/and-even-longer-sub-directory/landscape-2026.png",
				Lines:   []string{"(binary, 1 MB)"}})
		}},
		{"Error popup", "note", "an action failed; Enter or Esc closes it", func() tea.Cmd {
			return a.Error("Rename failed", "assets.png could not be renamed: a file called backgrounds.png is already there.")
		}},
		{"Error popup: long", "note", "a message longer than the popup: [j]|[k] scroll it", func() tea.Cmd {
			// A row each, no line break in the app's own text (components,
			// control characters).
			msg := []string{"The deploy of api to prod failed: 3 of 5 pods did not become ready. The last error from each:"}
			for i := range 40 {
				msg = append(msg, fmt.Sprintf("api-7d4f-%02d: readiness probe failed: GET /healthz: dial tcp 10.244.1.%d:8080: connect: connection refused", i+1, 17+i))
			}
			return a.Error("Deploy failed", msg...)
		}},
		{"Popup: the app’s own", "app’s own", "the app draws the inside, the lib the frame: draw on the grid, [Tab] to Save; Esc and Ctrl-C say when changed", ds.draw},
		{"Quit: the app’s own flow", "menu", "the next [q] opens a menu of ways to leave; Ctrl-C again leaves at once", ds.quitFlow},
		{"Move aside: a file it can’t read", "confirm", "its settings broken: Enter moves the file to .bak and starts as the first time, Esc quits the showcase", ds.moveAside},
		{"Move aside: a record it can’t read", "confirm", "one record per file, one broken: Enter moves just that one to .bak and reads the rest", ds.moveAsideRecord},
		{"Toast", "toast", "a short message on the third row from the bottom", toast("Copied 3 paths")},
		{"Toast over a popup", "toast", "a popup, then a toast: Esc takes the toast first", func() tea.Cmd {
			return tea.Batch(ds.yaml(), ds.s.app.Tick(600*time.Millisecond, func(time.Time) tea.Msg {
				return toastLater{terminu.Warning, "The YAML changed on the cluster"}
			}))
		}},
		{"Stack: Space menu → Global operation → Quit", "stack", "three layers; Esc closes one at a time", toast(
			"Press Space, then G and Enter for Global operation, then q: three layers. Esc closes them one at a time.")},
	}
}

func (ds *dialogsScreen) onMsg(msg tea.Msg) tea.Cmd {
	switch m := msg.(type) {
	case toastLater:
		return ds.s.app.Toast(m.kind, m.text)
	case loadTick:
		return ds.loadStep()
	}
	return nil
}

func (ds *dialogsScreen) openWith() tea.Cmd {
	a := ds.s.app
	open := func(with string) func() tea.Cmd {
		return func() tea.Cmd { return a.Toast(terminu.Info, "Opened README.md with "+with) }
	}
	return a.Menu("Open with", terminu.MenuSection{Title: "Default", Actions: []terminu.Action{
		{Key: "Enter", Label: "Open", Desc: "with the default app", Do: open("the default app")},
		{Key: "t", Label: "Open in a new tab", Desc: "keep this one", Do: open("a new tab")},
	}}, terminu.MenuSection{Title: "Editors", Actions: []terminu.Action{
		{Key: "v", Label: "Vim", Desc: "in this terminal", Do: open("Vim")},
		{Key: "c", Label: "VS Code", Desc: "a window of its own", Do: open("VS Code")},
	}}, terminu.MenuSection{Title: "Viewers", Actions: []terminu.Action{
		{Key: "m", Label: "Markdown preview", Desc: "rendered", Do: open("Markdown preview")},
		{Key: "x", Label: "Hex view", Desc: "the bytes", Do: open("Hex view"), Disabled: func() bool { return true }},
	}})
}

func (ds *dialogsScreen) sortBy() tea.Cmd {
	a := ds.s.app
	var cols []terminu.Action
	for _, c := range []string{"Name", "Size", "Modified"} {
		c := c
		cols = append(cols, terminu.Action{Label: c, Desc: "sort by " + strings.ToLower(c), Stay: true, Do: func() tea.Cmd {
			order := func(dir string) func() tea.Cmd {
				return func() tea.Cmd { return a.Toast(terminu.Info, "Sorted by "+strings.ToLower(c)+", "+dir) }
			}
			return a.Menu("Sort by "+c, terminu.MenuSection{Actions: []terminu.Action{
				{Key: "1", Label: "Ascending", Desc: "smallest first", Do: order("ascending")},
				{Key: "2", Label: "Descending", Desc: "largest first", Do: order("descending")},
			}})
		}})
	}
	return a.Menu("Sort by", terminu.MenuSection{Actions: cols})
}

var tasks = []string{
	"Rotate the TLS certificate on ingress-nginx before it expires on Friday; the new one is in vault under certs/ingress/2026, and the old one has to stay until every client has reloaded.",
	"Move the backups of postgres-0 from the local disk to the minio bucket backups/pg, then check that a restore from the bucket works on a scratch namespace.",
	"Find out why coredns-7w9z keeps crashing: the log says the upstream resolver times out, which started after the network change on Monday.",
	"Write the runbook for adding a node: labels, taints, the cilium agent, and the order in which to drain the old one.",
	"Clean up the stale namespaces left by the preview environments of merged pull requests.",
}

// transferring reports the work list's transfer running: its loading icon
// keeps turning (App.Spinning).
var transferring = func() bool { return false }

// quitFlow makes the next quit the app's own flow, a menu of ways to leave
// (App.QuitFlow; Rules K9); after it the confirm comes back.
func (ds *dialogsScreen) quitFlow() tea.Cmd {
	a := ds.s.app
	a.QuitFlow = func() tea.Cmd {
		a.QuitFlow = nil
		return a.Menu("Quit", terminu.MenuSection{Actions: []terminu.Action{
			{Key: "q", Label: "Quit", Desc: "leave the showcase", Do: func() tea.Cmd { return tea.Quit }},
			{Key: "c", Label: "Quit and clean up", Desc: "leave, the pretend files in " + projectDir + " taken away",
				Do: func() tea.Cmd { os.RemoveAll(projectDir); return tea.Quit }},
		}})
	}
	return a.Toast(terminu.Info, "The next q opens the showcase’s own way to quit")
}

func (ds *dialogsScreen) worklist() tea.Cmd {
	a := ds.s.app
	// A transfer that runs while the list is open: its row follows it, the
	// loading icon turning; Enter asks to cancel it, with the row's full
	// text (components/dialog/menu, work lists).
	start, cancelled := time.Now(), false
	progress := func() int { return min(int(time.Since(start)/(600*time.Millisecond)), 10) }
	transferring = func() bool { return !cancelled && progress() < 10 }
	rows := []terminu.Action{{Label: "upload.tar", EnterHint: "cancel", Stay: true,
		Live: func() (string, string) {
			switch {
			case cancelled:
				return "upload.tar", "cancelled"
			case progress() < 10:
				return terminu.LoadingFrame() + " upload.tar", fmt.Sprintf("%d/10", progress())
			}
			return "upload.tar", "done"
		},
		Disabled: func() bool { return !transferring() },
		Ask: &terminu.Ask{Title: "Cancel", Subject: "upload.tar", Verb: "cancel", Question: "Cancel the transfer?",
			Details: []string{"upload.tar to web-01:/srv/releases, 10 parts of 64 MB"}},
		Do: func() tea.Cmd { cancelled = true; return a.Toast(terminu.Info, "Cancelled upload.tar") }}}
	for i, t := range tasks {
		t := t
		first, _, _ := strings.Cut(t, ";")
		first, _, _ = strings.Cut(first, ",")
		rows = append(rows, terminu.Action{Label: first, Detail: true, Do: func() tea.Cmd {
			return a.Viewer(&terminu.Viewer{Title: "Task", Subject: fmt.Sprintf("#%d", i+1), Glyph: glyphList, Lines: []string{t}})
		}})
	}
	return a.Menu("Worklist", terminu.MenuSection{Actions: rows})
}

func (ds *dialogsScreen) longMenu() tea.Cmd {
	a := ds.s.app
	var rows []terminu.Action
	for _, n := range podNames {
		n := n
		rows = append(rows, terminu.Action{Label: n, Desc: "switch to " + n, Do: func() tea.Cmd { return a.Toast(terminu.Info, "Switched to "+n) }})
	}
	return a.Menu("Context", terminu.MenuSection{Actions: rows})
}

func (ds *dialogsScreen) longDescriptions() tea.Cmd {
	a := ds.s.app
	rows := []terminu.Action{
		{Key: "e", Label: "Export", Desc: "write every pod, service, config map, secret, ingress and volume claim in this namespace to a single YAML file", Do: func() tea.Cmd { return a.Toast(terminu.Info, "Exported") }},
		{Key: "i", Label: "Import", Desc: "read a YAML file and create or update everything in it, asking first when something would be replaced", Do: func() tea.Cmd { return a.Toast(terminu.Info, "Imported") }},
		{Key: "r", Label: "Restart", Desc: "roll every deployment", Do: func() tea.Cmd { return a.Toast(terminu.Info, "Restarted") }},
	}
	return a.Menu("Namespace", terminu.MenuSection{Actions: rows})
}

// hostForm is NEW HOST: every kind of field. With fail, saving fails.
func (ds *dialogsScreen) hostForm(_ *terminu.Form, fail bool) *terminu.Form {
	a := ds.s.app
	var (
		name, host, port, user = "", "", "", "deploy"
		profiles               = []string{"default", "staging", "production", "sandbox", "ci", "legacy"}
		profile                = 0
		credentials            = []string{"none", "deploy-key", "ci-token", "admin-cert", "readonly", "backup"}
		credential             = 0
		namespaces             = []string{"default", "kube-system", "monitoring", "ingress-nginx", "cert-manager", "argocd"}
		nsChecked              = []bool{true, false, false, false, false, false}
	)
	if fail {
		name, host = "web-01", "10.0.0.5" // ready to save, so the save itself fails
	}
	auth := &terminu.FormField{Label: "Auth", Options: []string{"password", "privatekey", "credential"}}
	text := func(label, prompt string, v *string, required bool) *terminu.FormField {
		f := &terminu.FormField{Label: label, Value: func() string { return *v }, Empty: "not set",
			Edit: func() terminu.Input {
				return terminu.TextInput{Title: label, Glyph: glyphPencil, Prompt: prompt, Old: *v,
					Done: func(s string) tea.Cmd { *v = s; return nil }}
			}}
		if required {
			f.Empty = "required"
			f.Check = func() string {
				if *v == "" {
					return label + " is required"
				}
				return ""
			}
		}
		return f
	}
	return &terminu.Form{
		Title: "New host", Name: "New host", Glyph: glyphServer, Button: "Save",
		Fields: []*terminu.FormField{
			text("Name", "A name for the host", &name, true),
			text("Host", "Its address or name", &host, true),
			{Label: "Port", Value: func() string { return port }, Empty: "22", Edit: func() terminu.Input {
				return terminu.NumberInput{Title: "Port", Glyph: glyphLan, Name: "Port", Prompt: "The SSH port, 1-65535", Old: port,
					Min: 1, Max: 65535, Done: func(s string) tea.Cmd { port = s; return nil }}
			}},
			text("User", "Who to log in as", &user, false),
			auth,
			{Label: "Credential", Value: func() string { return credentials[credential] },
				Disabled: func() bool { return auth.Selected != 2 },
				Edit: func() terminu.Input {
					return terminu.Select{Title: "Credential", Glyph: glyphKey, Options: credentials, Selected: credential,
						Done: func(i int) tea.Cmd { credential = i; return nil }}
				},
				Check: func() string {
					if auth.Selected == 2 && credential == 0 {
						return "Credential is required with Auth credential"
					}
					return ""
				}},
			// No display here: X11 is not available (components/input/checkbox).
			{Label: "Agent", Options: []string{"forward", "X11"}, Multi: true, OptionUnavailable: func(i int) bool { return i == 1 }},
			{Label: "Namespaces", Value: func() string { return joinChecked(namespaces, nsChecked) }, Empty: "none",
				Edit: func() terminu.Input {
					return terminu.Checks{Title: "Namespaces", Glyph: glyphTag, Options: namespaces, Checked: nsChecked,
						Done: func(c []bool) tea.Cmd { nsChecked = c; return nil }}
				}},
			// An option that cannot be chosen: grey, the focus skips it.
			{Label: "Order", Options: []string{"asc", "desc", "random"}, OptionDisabled: func(i int) bool { return i == 2 }},
			{Label: "Profile", Value: func() string { return profiles[profile] }, Edit: func() terminu.Input {
				return terminu.Select{Title: "Profile", Glyph: glyphProfile, Options: profiles, Selected: profile,
					Done: func(i int) tea.Cmd { profile = i; return nil }}
			}},
		},
		Submit: func() (tea.Cmd, error) {
			if fail {
				return nil, errors.New("hosts.yaml changed on disk since it was opened.")
			}
			return a.Toast(terminu.Info, "Saved "+name), nil
		},
	}
}

// actionsForm is the host form with more buttons (components/dialog/form,
// the action area): Test and Scan keep the form open and need only Host;
// Save As is another submit; the main Connect takes long. Tab lands on
// Connect, bottom right.
func (ds *dialogsScreen) actionsForm() tea.Cmd {
	a := ds.s.app
	f := ds.hostForm(nil, false)
	hostF, portF := f.Fields[1], f.Fields[2]
	needs := []*terminu.FormField{hostF}
	// dial stands for a connection that takes a while, refused by a host
	// named bad-…; closing the form stops it.
	dial := func(ctx context.Context, host string) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1500 * time.Millisecond): // long enough to see the loading icon
		}
		if strings.HasPrefix(host, "bad") {
			return errors.New("ssh: connect to host " + host + " port 22: Connection refused")
		}
		return nil
	}
	test := &terminu.FormButton{Label: "Test", Needs: needs, Run: func(ctx context.Context) terminu.FormResult {
		host := hostF.Value()
		if err := dial(ctx, host); err != nil {
			return terminu.FormResult{Err: err}
		}
		return terminu.FormResult{Toast: "Connected to " + host}
	}}
	scan := &terminu.FormButton{Label: "Scan", Needs: needs, Run: func(context.Context) terminu.FormResult {
		host := hostF.Value()
		return terminu.FormResult{
			Lines: []string{
				"# " + host + ":22 SSH-2.0-OpenSSH_9.6",
				host + " ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFakeKeyForTheShowcaseOnly0000000000",
				host + " ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYFakeKeyForTheShowcase",
			},
			// The port it answered on goes into Port: the form counts as changed.
			Apply: func() { portF.Edit().(terminu.NumberInput).Done("22") },
		}
	}}
	saveAs := &terminu.FormButton{Label: "Save As", Submit: func() (tea.Cmd, error) {
		return a.Toast(terminu.Info, "Saved a copy of "+f.Fields[0].Value()), nil
	}}
	// A button naming what it acts on: the hint keeps the name as it is,
	// Enter:export hosts.yaml (components/dialog/form, hint).
	export := &terminu.FormButton{Label: "Export", Subject: "hosts.yaml", Submit: func() (tea.Cmd, error) {
		return a.Toast(terminu.Info, "Exported "+f.Fields[0].Value()+" to hosts.yaml"), nil
	}}
	// A button that runs a program in a terminal popup over the form; once it
	// ends, the app reads back what it did: the key is on the host now, so
	// Auth is privatekey (components/dialog/form).
	copyKey := &terminu.FormButton{Label: "Copy key", Open: func() tea.Cmd {
		host := hostF.Value()
		if host == "" {
			host = "the host"
		}
		return a.OpenTerminal(&terminu.Terminal{Title: "ssh-copy-id", Subject: host, Ends: true,
			Command: []string{"sh", "-c", "echo Copying ~/.ssh/id_ed25519.pub to " + host + "; echo '(the showcase only pretends)'; sleep 2"}})
	}, Back: func() bool {
		f.Fields[3].Selected = 1
		return true
	}}
	f.Buttons = [][]*terminu.FormButton{{test, scan, export}, {copyKey, saveAs}}
	f.Button, f.Submit = "Connect", nil
	f.SubmitLong = func(ctx context.Context) (tea.Cmd, error) {
		host := hostF.Value()
		if err := dial(ctx, host); err != nil {
			return nil, err
		}
		return a.Toast(terminu.Info, "Connected to "+host+" and saved it"), nil
	}
	return a.Form(f)
}

func (ds *dialogsScreen) profileForm() tea.Cmd {
	a := ds.s.app
	name, timeout, command := "clock3", "30", "tmux new-session -A -s main"
	return a.Form(&terminu.Form{Title: "New profile", Name: "New profile", Glyph: glyphProfile, Button: "Create",
		Fields: []*terminu.FormField{
			{Label: "Name", Value: func() string { return name }, Edit: func() terminu.Input {
				return terminu.TextInput{Title: "Name", Glyph: glyphPencil, Prompt: "A name for the profile", Old: name,
					Done: func(s string) tea.Cmd { name = s; return nil }}
			}},
			{Label: "Timeout", Value: func() string { return timeout }, Edit: func() terminu.Input {
				return terminu.NumberInput{Title: "Timeout", Glyph: glyphTimer, Name: "Timeout", Prompt: "Seconds, 1-3600", Old: timeout,
					Min: 1, Max: 3600, Done: func(s string) tea.Cmd { timeout = s; return nil }}
			}},
			{Label: "Command", Stacked: true, Value: func() string { return command }, Edit: func() terminu.Input {
				return terminu.TextInput{Title: "Command", Glyph: glyphConsole, Prompt: "What to run", Old: command,
					Done: func(s string) tea.Cmd { command = s; return nil }}
			}},
			{Label: "Layout", Options: []string{"row", "column"}},
		},
		Submit: func() (tea.Cmd, error) { return a.Toast(terminu.Info, "Created "+name), nil },
	})
}

func (ds *dialogsScreen) longForm() tea.Cmd {
	a := ds.s.app
	values := map[string]string{}
	var fields []*terminu.FormField
	for _, label := range []string{"Name", "Host", "Port", "User", "Proxy", "Identity", "Shell", "Term", "Locale",
		"Timezone", "Editor", "Pager", "Prompt", "History", "Theme", "Font", "Font size", "Scrollback", "Bell",
		"Cursor", "Blink", "Tab width", "Line height", "Opacity", "Padding", "Title", "Icon", "Startup", "Shutdown",
		"Proxy jump", "Forward agent", "Compression", "Keep alive", "Retries", "Log file", "Log level", "Backup",
		"Sync", "Notes"} {
		label := label
		fields = append(fields, &terminu.FormField{Label: label, Value: func() string { return values[label] }, Empty: "not set",
			Edit: func() terminu.Input {
				return terminu.TextInput{Title: label, Glyph: glyphPencil, Prompt: "The " + strings.ToLower(label), Old: values[label],
					Done: func(s string) tea.Cmd { values[label] = s; return nil }}
			}})
	}
	return a.Form(&terminu.Form{Title: "Edit", Subject: "prod-db-01", Name: "prod-db-01", Glyph: glyphServer, Button: "Save",
		Fields: fields, Submit: func() (tea.Cmd, error) { return a.Toast(terminu.Info, "Saved"), nil }})
}

var yamlLines = strings.Split(`apiVersion: v1
kind: Pod
metadata:
  name: nginx-7d4f
  namespace: default
  labels:
    app: nginx
    tier: frontend
spec:
  containers:
    - name: nginx
      image: nginx:1.27
      ports:
        - containerPort: 80
      resources:
        limits:
          cpu: 500m
          memory: 128Mi
      readinessProbe:
        httpGet:
          path: /healthz
          port: 80
  restartPolicy: Always
status:
  phase: Running
  podIP: 10.244.1.17
  hostIP: 192.168.49.2
  startTime: "2026-10-12T08:14:03Z"
  qosClass: Burstable
  conditions:
    - type: Initialized
      status: "True"
      lastTransitionTime: "2026-10-12T08:14:03Z"
    - type: Ready
      status: "True"
      lastTransitionTime: "2026-10-12T08:14:09Z"
    - type: ContainersReady
      status: "True"
      lastTransitionTime: "2026-10-12T08:14:09Z"
    - type: PodScheduled
      status: "True"
      lastTransitionTime: "2026-10-12T08:14:03Z"
  containerStatuses:
    - name: nginx
      image: nginx:1.27
      imageID: docker.io/library/nginx@sha256:5f0574409b3add89581b96c68afe9e9c7b284651c3a974b6e8bac46bf95e6b7f
      containerID: containerd://8d1c0e4b9f3a2e7d6c5b4a39281706f5e4d3c2b1a0f9e8d7c6b5a49382716051
      ready: true
      started: true
      restartCount: 0
      state:
        running:
          startedAt: "2026-10-12T08:14:07Z"
  podIPs:
    - ip: 10.244.1.17
  ephemeralContainerStatuses: []
  initContainerStatuses: []
  hostIPs:
    - ip: 192.168.49.2`, "\n")

func (ds *dialogsScreen) yaml() tea.Cmd {
	a := ds.s.app
	v := &terminu.Viewer{Title: "YAML", Subject: "pod/nginx-7d4f", Glyph: glyphYaml, Lines: yamlLines, Lang: "yaml"}
	// The app's own search: the lib underlines every hit and draws the
	// current one as the cursor block (Viewer.Hits).
	step := func(by int) func() tea.Cmd {
		return func() tea.Cmd {
			if len(v.Hits) == 0 {
				return nil
			}
			v.Current = (v.Current + by + len(v.Hits)) % len(v.Hits)
			v.ScrollTo(v.Hits[v.Current].Line)
			return nil
		}
	}
	v.Keys = []terminu.ViewerKey{
		{Key: "y", Desc: "copy", Do: func() tea.Cmd { return a.Toast(terminu.Success, "Copied the YAML") }},
		{Key: "/", Desc: "search", Do: func() tea.Cmd {
			return a.Text(terminu.TextInput{Title: "Search", Glyph: glyphSearch, Prompt: "Find in the YAML", Verb: "find", Search: true,
				Done: func(s string) tea.Cmd {
					v.Hits, v.Current = v.FindText(s), 0
					if len(v.Hits) == 0 {
						return a.Toast(terminu.Info, "No line has "+s)
					}
					v.ScrollTo(v.Hits[0].Line)
					return nil
				}})
		}},
		{Key: "n", Desc: "next", Do: step(1)},
		{Key: "N", Desc: "previous", Do: step(-1)},
	}
	// No Select: the selection mode every viewer has, copying to the
	// terminal's clipboard (components/dialog/note).
	return a.Viewer(v)
}

func (ds *dialogsScreen) devtools() tea.Cmd {
	console := []string{"[log] app started", "[warn] slow response from /api/pods (1.2s)", "[error] websocket closed: 1006"}
	network := []string{"GET  /api/pods        200  48ms", "GET  /api/events      200  31ms", "POST /api/exec        101  —"}
	v := &terminu.Viewer{Title: "DevTools", Glyph: glyphConsole, Tabs: []string{"Console", "Network"}, Lines: console}
	v.OnTab = func(tab int) tea.Cmd {
		v.Lines = []([]string){console, network}[tab]
		return nil
	}
	// h|l switch the tabs, and move the cursor in the selection mode (Rules K12).
	// A Copy of the app's own takes the text instead of the clipboard.
	v.Select = &terminu.Selection{Key: "v", Name: "Visual", Copy: func(text string) tea.Cmd {
		return ds.s.app.Toast(terminu.Success, fmt.Sprintf("Copied %d characters", len([]rune(text))))
	}}
	return ds.s.app.Viewer(v)
}

// table is a viewer of a table the app lays out: lines kept on their rows,
// cut at the end, and no line numbers (components/dialog/note).
func (ds *dialogsScreen) table() tea.Cmd {
	return ds.s.app.Viewer(&terminu.Viewer{Title: "Info", Subject: "assets/photo-large.png", Glyph: glyphList, NoWrap: true, NoNumbers: true,
		Lines: []string{
			"TYPE      image",
			"SIZE      1.2 MB",
			"MODIFIED  2026-10-19 14:02",
			"MODE      -rw-r--r--",
			"EXIF      Camera Fujifilm X100V · 23mm · f/2 · 1/250s · ISO 160 · taken in the afternoon by the river",
		}})
}

func (ds *dialogsScreen) loadingPopup() tea.Cmd {
	ds.loading = &terminu.Viewer{Title: "Log", Subject: "deploy/api", Glyph: glyphYaml, Loading: true}
	ds.loaded = 0
	return tea.Batch(ds.s.app.Viewer(ds.loading), ds.s.app.Tick(400*time.Millisecond, func(time.Time) tea.Msg { return loadTick{} }))
}

func (ds *dialogsScreen) loadStep() tea.Cmd {
	if ds.loading == nil {
		return nil
	}
	ds.loaded++
	ds.loading.Lines = append(ds.loading.Lines, fmt.Sprintf("%s INFO  step %d of 16 done", time.Now().Format("15:04:05"), ds.loaded))
	if ds.loaded == 16 {
		ds.loading.Loading = false
		ds.loading = nil
		return nil
	}
	return ds.s.app.Tick(250*time.Millisecond, func(time.Time) tea.Msg { return loadTick{} })
}
