package main

import (
	"fmt"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"
)

// panelsScreen is [P]ANELS: every look and state of a panel. [1] PODS is an
// ordinary list (tabs, header and sorting, marks, a cursor, 40 rows, a panel
// filter, a mode), [2] LOG streaming content, [3] SETTINGS one value per row.
type panelsScreen struct {
	s        *showcase
	screen   *terminu.Screen
	pods     *terminu.Panel
	podList  *terminu.List
	log      *terminu.Panel
	logText  *terminu.Pager
	settings *terminu.Panel
	values   *terminu.Values
	disabled bool
	logN     int
	created  int // pods added with New
	errors   int // ERROR lines in the log, for the statusbar
	warnings int // WARN lines
	dragFrom string
}

var podNames = []string{
	"nginx-7d4f", "web-a8k2", "coredns-7w9z", "prometheus-server-0", "api-gateway-5c9b8d7f6-x2k4p", "redis-master-0",
	"postgres-0", "kafka-broker-1", "zookeeper-0", "grafana-6b8d", "loki-0", "tempo-ingester-2", "cert-manager-7f9c",
	"ingress-nginx-controller-8d4", "metrics-server-5f7", "argocd-repo-server-3", "vault-0", "consul-server-1",
	"jaeger-collector-4", "elasticsearch-master-2", "kibana-9c2", "fluent-bit-x7k2", "node-exporter-lq8", "alertmanager-0",
	"mysql-primary-0", "rabbitmq-1", "memcached-5d", "minio-3", "etcd-2", "harbor-core-6", "keycloak-1", "sonarqube-0",
	"jenkins-agent-8x", "nexus-0", "traefik-4b", "linkerd-proxy-9", "istiod-7c", "cilium-operator-2", "velero-5", "dex-3",
}

var statuses = []string{"Running", "Running", "CrashLoopBackOff", "Running", "Pending", "Running", "Completed", "Running"}

func newPanelsScreen(s *showcase) *panelsScreen {
	ps := &panelsScreen{s: s}
	ps.podList = &terminu.List{
		Columns: []terminu.Column{
			{Title: "Name", Middle: true, Shrink: true}, {Title: "Ready", Drop: 3}, {Title: "Status"}, {Title: "Restarts", Number: true, Drop: 2}, {Title: "Age", Number: true, Drop: 1},
		},
		Marks:      true,
		Draggable:  true,
		Filterable: true,
		EmptyFact:  "No pods here",
		EmptyHint:  "Press [Space] to see what you can do here",
	}
	ps.podList.SetRows(podRows())
	ps.podList.SortBy(terminu.SortKey{Col: 0})
	ps.pods = &terminu.Panel{Label: "Pods", Tabs: []string{"Events"}, Body: ps.podList, LoadingText: "loading pods…"}
	ps.pods.OnTab = ps.switchTab
	ps.pods.Actions = ps.podActions()
	// A pod that is not running says why in an inspect toast while the
	// cursor is on it (tdp toast.4).
	ps.pods.Inspect = ps.podReason
	// C compares two pods' pretend YAML (tdp panel.9): the
	// lib does the mode and the popup; the app gives the key and the text.
	ps.pods.Actions = append(ps.pods.Actions, s.app.CompareAction(ps.pods, ps.podList, terminu.Compare{Key: "C",
		Text: func(r terminu.Row) (string, error) { return podYAML(r), nil }}))

	ps.logText = &terminu.Pager{Follow: true, EmptyFact: "No log lines yet"}
	ps.log = &terminu.Panel{Label: "Log", Body: ps.logText, Stream: true, Actions: []terminu.Action{
		{Key: "Enter", Label: "View", Desc: "the whole log in a viewer", Scope: terminu.PanelScope, Hint: "view",
			Do: func() tea.Cmd {
				return s.app.Viewer(&terminu.Viewer{Title: "Log", Subject: "api", Glyph: glyphYaml, Lines: slices.Clone(ps.logText.Lines())})
			}},
		{Key: "F", Label: "Freeze", Desc: "stop following new lines until Esc", Scope: terminu.PanelScope, Do: ps.freeze},
	}}
	for range 12 {
		ps.appendLog()
	}

	ps.values = &terminu.Values{Header: [2]string{"Property", "Value"}}
	name, pin, timeout := "showcase", "", "30"
	a := s.app
	ps.values.Fields = []*terminu.Field{
		// Each row says what it is in the panel's ? (what each row is).
		{Label: "Name", Desc: "the name in the splash and the window title", Value: func() string { return name }, Edit: func() tea.Cmd {
			return a.Text(terminu.TextInput{Title: "Name", Glyph: glyphPencil, Prompt: "A name for this showcase", Verb: "rename", Old: name,
				Check: func(v string) error {
					if v == "" {
						return fmt.Errorf("Name can't be empty")
					}
					return nil
				},
				Done: func(v string) tea.Cmd { name = v; return nil }})
		}},
		{Label: "PIN", Desc: "a PIN, kept as a lock screen keeps one", Value: func() string {
			if pin == "" {
				return ""
			}
			return "••••••••"
		}, Empty: "not set", Edit: func() tea.Cmd {
			return a.Password(terminu.PasswordInput{Title: "PIN", Glyph: glyphLock, Prompt: "Up to 12 digits", Verb: "set", Set: pin != "", Unlock: true, MaxLen: 12,
				Done: func(v string, clear bool) tea.Cmd { pin = v; return nil }})
		}},
		{Label: "Layout", Desc: "how the panels share the screen", Options: []string{"row", "column"}},
		{Label: "Notify", Desc: "when a toast tells of a job", Options: []string{"on errors", "on success"}, Multi: true, Checked: []bool{true, false}},
		{Label: "Remote", Desc: "the host the jobs run on", Value: func() string { return "ssh decides" }, Disabled: func() bool { return ps.disabled }},
		// Fail on this panel makes Timeout wrong: its label turns Red.
		{Label: "Timeout", Desc: "how long a job may run", Value: func() string { return timeout }, Err: func() bool { return ps.settings.Fail != nil }, Edit: func() tea.Cmd {
			return a.Number(terminu.NumberInput{Title: "Timeout", Glyph: glyphTimer, Name: "Timeout", Prompt: "Seconds, 1-3600", Verb: "save",
				Old: timeout, Min: 1, Max: 3600, Done: func(v string) tea.Cmd { timeout = v; return nil }})
		}},
	}
	ps.settings = &terminu.Panel{Label: "Settings", Body: ps.values, Actions: []terminu.Action{
		{Key: "e", Label: "Explain", Desc: "what this setting does", Scope: terminu.ItemScope, Hint: "explain",
			Do: func() tea.Cmd {
				return s.app.Toast(terminu.Info, "Settings take effect as soon as they are set")
			}},
	}}
	ps.settings.Actions = append(ps.settings.Actions, ps.stateActions(ps.settings)...)

	ps.screen = &terminu.Screen{Name: "Panels", Key: "P", Desc: "every look and state of a panel",
		Panels: []*terminu.Panel{ps.pods, ps.log, ps.settings}, NarrowBelow: 60, Layout: ps.layout}
	return ps
}

// layout: PODS on the left, LOG and SETTINGS stacked on the right.
func (ps *panelsScreen) layout(w, h int) []terminu.Rect {
	left := w * 3 / 5
	top := h / 2
	return []terminu.Rect{
		{X: 0, Y: 0, W: left, H: h},
		{X: left, Y: 0, W: w - left, H: top},
		{X: left, Y: top, W: w - left, H: h - top},
	}
}

// podReason is why the pod under the cursor is not running, for its
// inspect toast; a running pod and an event have none.
func (ps *panelsScreen) podReason() []string {
	r, ok := ps.podList.Current()
	if !ok || len(r.Cells) < 5 {
		return nil
	}
	switch r.Cells[2] {
	case "CrashLoopBackOff":
		return []string{"Back-off restarting failed container app"}
	case "Pending":
		return []string{"0/3 nodes are available: 3 Insufficient memory"}
	}
	return nil
}

// podYAML is a pod's pretend manifest, from what its row says.
func podYAML(r terminu.Row) string {
	if len(r.Cells) < 5 { // an event: its own cells, a line each
		return strings.Join(r.Cells, "\n") + "\n"
	}
	return "apiVersion: v1\nkind: Pod\nmetadata:\n  name: " + r.Cells[0] + "\n  namespace: default\nspec:\n  containers:\n" +
		"    - name: app\n      image: registry.local/app:1.4\n      ports:\n        - containerPort: 8080\n" +
		"status:\n  phase: " + r.Cells[2] + "\n  ready: " + r.Cells[1] + "\n  restartCount: " + r.Cells[3] + "\n  age: " + r.Cells[4] + "\n"
}

func podRows() []terminu.Row {
	rows := make([]terminu.Row, len(podNames))
	for i, n := range podNames {
		st := statuses[i%len(statuses)]
		ready := "1/1"
		var style terminu.Style
		switch st {
		case "CrashLoopBackOff":
			ready, style = "0/1", terminu.Fg(terminu.Red)
		case "Pending":
			ready, style = "0/2", terminu.Fg(terminu.Peach)
		case "Completed":
			ready, style = "0/1", terminu.Fg(terminu.Overlay0)
		}
		restarts := (i * 7) % 13
		age := time.Duration((i*37)%400+1) * time.Hour / 3
		rows[i] = terminu.Row{
			ID:    n,
			Cells: []string{n, ready, st, fmt.Sprint(restarts), shortAge(age)},
			// The status in its colour: a cell's colours come as spans.
			Styled: [][]terminu.Span{2: {{Text: st, Style: style}}},
			Sort:   []string{n, ready, st, fmt.Sprint(restarts), fmt.Sprint(int(age.Seconds()))},
		}
	}
	return rows
}

func shortAge(d time.Duration) string {
	switch {
	case d >= 24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	case d >= time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dm", int(d.Minutes()))
}

var eventRows = []terminu.Row{
	{ID: "e1", Cells: []string{"web-a8k2", "Warning", "BackOff", "12", "2m"}},
	{ID: "e2", Cells: []string{"prometheus-server-0", "Normal", "Pulled", "1", "5m"}},
	{ID: "e3", Cells: []string{"postgres-0", "Normal", "Started", "1", "9m"}},
}

func (ps *panelsScreen) switchTab(tab int) tea.Cmd {
	cur := ps.podList.Sorting()
	if tab == 1 {
		ps.podList.Columns = []terminu.Column{{Title: "Object", Middle: true, Shrink: true}, {Title: "Type", Drop: 3}, {Title: "Reason"}, {Title: "Count", Number: true, Drop: 2}, {Title: "Age", Number: true, Drop: 1}}
		ps.podList.SetRows(eventRows)
	} else {
		ps.podList.Columns = []terminu.Column{{Title: "Name", Middle: true, Shrink: true}, {Title: "Ready", Drop: 3}, {Title: "Status"}, {Title: "Restarts", Number: true, Drop: 2}, {Title: "Age", Number: true, Drop: 1}}
		ps.podList.SetRows(podRows())
	}
	ps.podList.SortBy(cur...)
	return nil
}

func (ps *panelsScreen) current() string {
	r, ok := ps.podList.Current()
	if !ok {
		return ""
	}
	return r.ID
}

func (ps *panelsScreen) podActions() []terminu.Action {
	a := ps.s.app
	acts := []terminu.Action{
		{Key: "Enter", Label: "Logs", Desc: "follow its log", Scope: terminu.ItemScope, Hint: "logs",
			Do: func() tea.Cmd {
				ps.logText.Append(terminu.Fg(terminu.Lavender).Render("── following " + ps.current() + " ──"))
				return a.Toast(terminu.Info, "Following the log of "+ps.current())
			}},
		// Disabled, it stays in the hint in the disabled colour (tdp M6).
		{Key: "m", Label: "mark", Desc: "this pod, for a batch", Scope: terminu.ItemScope, Hint: "mark",
			Disabled: func() bool { return ps.disabled },
			Do:       func() tea.Cmd { ps.podList.ToggleMark(); return nil }},
		{Key: "x", Label: "Delete", Desc: "this pod", Scope: terminu.ItemScope,
			Disabled: func() bool { return ps.disabled },
			Ask:      &terminu.Ask{Title: "Delete", Question: "Delete this pod?", Verb: "delete", Details: []string{"The pod is removed; its deployment starts a new one.", "This cannot be undone."}, Warn: []int{1}},
			Do: func() tea.Cmd {
				return a.Toast(terminu.Info, "Deleted "+ps.current())
			}},
		{Label: "Delete marked", Desc: "every marked pod", Scope: terminu.PanelScope,
			Disabled: func() bool { return len(ps.podList.Marked()) == 0 || ps.disabled },
			Ask:      &terminu.Ask{Title: "Delete marked", Question: "Delete the marked pods?", Verb: "delete"},
			Do: func() tea.Cmd {
				n := len(ps.podList.Marked())
				ps.podList.ClearMarks()
				return a.Toast(terminu.Info, fmt.Sprintf("Deleted %d pods", n))
			}},
		{Key: "D", Label: "Drag", Desc: "this pod to another place in the list", Scope: terminu.ItemScope,
			Do: ps.startDrag},
		// After adding an item the cursor goes to it, scrolled into sight
		// (tdp K3): sorted by name, it lands far from where the cursor was.
		{Key: "n", Label: "New", Desc: "a pod, the cursor going to it", Scope: terminu.PanelScope,
			Hidden: func() bool { return ps.pods.Tab != 0 },
			Do: func() tea.Cmd {
				ps.created++
				name := fmt.Sprintf("new-pod-%d", ps.created)
				podNames = append(podNames, name)
				ps.podList.SetRows(podRows())
				ps.podList.Select(name)
				return a.Toast(terminu.Success, "Created "+name)
			}},
		{Key: "s", Label: "Sort by", Desc: "a column, then a direction", Scope: terminu.PanelScope,
			Do: ps.sortMenu},
		{Key: "go", Label: "goto", Desc: "the first pod that is not running", Scope: terminu.PanelScope,
			Do: func() tea.Cmd {
				for _, r := range ps.podList.Rows() {
					if !strings.Contains(r.Sort[2], "Running") {
						ps.podList.Select(r.ID)
						break
					}
				}
				return nil
			}},
	}
	return append(acts, ps.stateActions(nil)...)
}

// stateActions are the showcase's own actions that switch a panel's state
// (Empty, Loading, Fail, Disable, Toast …): they are panel operations, so
// they also show the M chapter at work.
func (ps *panelsScreen) stateActions(p *terminu.Panel) []terminu.Action {
	a := ps.s.app
	target := func() *terminu.Panel {
		if p != nil {
			return p
		}
		return ps.pods
	}
	return []terminu.Action{
		{Label: "Empty", Desc: "clear this panel, or fill it again", Scope: terminu.PanelScope,
			Do: func() tea.Cmd {
				if target() == ps.pods {
					if ps.podList.Empty() {
						ps.switchTab(ps.pods.Tab)
					} else {
						ps.podList.SetRows(nil)
					}
				}
				return nil
			}},
		{Label: "Loading", Desc: "start loading, or stop", Scope: terminu.PanelScope,
			Do: func() tea.Cmd { t := target(); t.Loading = !t.Loading; return nil }},
		{Label: "Fail", Desc: "this panel cannot get its content", Scope: terminu.PanelScope,
			Do: func() tea.Cmd {
				t := target()
				if t.Fail != nil {
					t.Fail = nil
					return nil
				}
				t.Fail = &terminu.Failure{Fact: "Forbidden: pods in kube-system", Hint: "Press [Space] and choose Fail again to retry"}
				return a.Toast(terminu.Error, "Forbidden: pods in kube-system")
			}},
		{Label: "Disable", Desc: "some rows and actions cannot be used now", Scope: terminu.PanelScope,
			Do: func() tea.Cmd { ps.disabled = !ps.disabled; return nil }},
		{Label: "Toast success", Desc: "something is done", Scope: terminu.PanelScope,
			Do: func() tea.Cmd { return a.Toast(terminu.Success, "Copied 3 paths") }},
		{Label: "Toast info", Desc: "a short message", Scope: terminu.PanelScope,
			Do: func() tea.Cmd { return a.Toast(terminu.Info, "Switched to context prod") }},
		{Label: "Toast warning", Desc: "something worth a look", Scope: terminu.PanelScope,
			Do: func() tea.Cmd { return a.Toast(terminu.Warning, "The certificate expires in 3 days") }},
		{Label: "Toast error", Desc: "something failed", Scope: terminu.PanelScope,
			Do: func() tea.Cmd { return a.Toast(terminu.Error, "Could not reach the cluster") }},
		{Label: "Toast long", Desc: "a message of several lines", Scope: terminu.PanelScope,
			Do: func() tea.Cmd {
				return a.Toast(terminu.Info, "A toast says one short thing. This one goes on and on, longer than any toast should, so that it wraps onto several lines: a toast has as many lines as it needs, unless the app sets a cap, and then the last line it keeps ends in an ellipsis.")
			}},
	}
}

func (ps *panelsScreen) notYet(what string) func() tea.Cmd {
	return func() tea.Cmd { return ps.s.app.Toast(terminu.Info, "The "+what+" comes with the Inputs screen") }
}

// freeze is a mode that does not pause Tab: the panel can lose the focus in
// it, its mode name then between light joints.
func (ps *panelsScreen) freeze() tea.Cmd {
	ps.logText.Follow = false
	ps.log.EnterMode(&terminu.Mode{Name: "Frozen", Exit: "resume", OnExit: func() tea.Cmd {
		ps.logText.Follow = true
		return nil
	}})
	return nil
}

// startDrag enters the Drag mode (tdp K11): j|k move the pod, Enter drops
// it, Esc puts it back.
func (ps *panelsScreen) startDrag() tea.Cmd {
	ps.dragFrom = ps.current()
	before := append([]terminu.Row(nil), ps.podList.Rows()...)
	sorting := ps.podList.Sorting()
	ps.podList.SortBy() // dragging sets the order by hand
	ps.podList.Hold()
	ps.pods.EnterMode(&terminu.Mode{
		Name: "Drag", Exit: "cancel", Pause: true,
		Keys: []terminu.ModeKey{
			{Key: "j|k", Keys: []string{"j", "k", "↓", "↑"}, Desc: "move it down, up", NoFooter: true,
				Do: func(key string) tea.Cmd { ps.podList.MoveHeld(key == "j" || key == "↓"); return nil }},
			// The drop acts on the pod held: ITEM OPERATION in the mode's
			// Space menu (tdp menu.4).
			{Key: "Enter", Desc: "drop", Item: true, MenuDesc: "the pod here", Do: func(string) tea.Cmd {
				ps.podList.Drop()
				ps.pods.LeaveMode()
				return ps.s.app.Toast(terminu.Info, "Dropped "+ps.dragFrom)
			}},
		},
		OnExit: func() tea.Cmd {
			ps.podList.Drop()
			ps.podList.SetRows(before)
			ps.podList.SortBy(sorting...)
			ps.podList.Select(ps.dragFrom)
			return nil
		},
	})
	return nil
}

// sortMenu is a two-step sort: a column, then a direction (tdp F1: each
// step its own popup).
func (ps *panelsScreen) sortMenu() tea.Cmd {
	a := ps.s.app
	var cols []terminu.Action
	for i, c := range ps.podList.Columns {
		i, c := i, c
		cols = append(cols, terminu.Action{Label: c.Title, Desc: "sort by " + strings.ToLower(c.Title), Stay: true,
			Do: func() tea.Cmd {
				return a.Menu("Sort by "+c.Title, terminu.MenuSection{Actions: []terminu.Action{
					{Label: "Ascending", Desc: "smallest first", Do: func() tea.Cmd { ps.podList.SortBy(terminu.SortKey{Col: i}); return nil }},
					{Label: "Descending", Desc: "largest first", Do: func() tea.Cmd { ps.podList.SortBy(terminu.SortKey{Col: i, Desc: true}); return nil }},
					// Two keys in two directions: both sort glyphs in the header at once,
					// to see which way each one looks (the walkthrough's sort glyphs).
					{Label: "Then by age", Desc: "this column first, then the age, oldest first", Do: func() tea.Cmd {
						ps.podList.SortBy(terminu.SortKey{Col: i}, terminu.SortKey{Col: 4, Desc: true})
						return nil
					}},
				}})
			}})
	}
	return a.Menu("Sort by", terminu.MenuSection{Actions: cols})
}

var logLevels = []string{"INFO", "INFO", "DEBUG", "WARN", "INFO", "ERROR"}

func (ps *panelsScreen) appendLog() {
	ps.logN++
	lvl := logLevels[ps.logN%len(logLevels)]
	color := terminu.Text
	switch lvl {
	case "WARN":
		color = terminu.Peach
		ps.warnings++
	case "ERROR":
		color = terminu.Red
		ps.errors++
	case "DEBUG":
		color = terminu.Overlay0
	}
	line := fmt.Sprintf("%s %s request %d served in %dms", time.Now().Format("15:04:05"),
		terminu.Fg(color).Render(fmt.Sprintf("%-5s", lvl)), ps.logN, 3+ps.logN*7%90)
	ps.logText.Append(line)
}
