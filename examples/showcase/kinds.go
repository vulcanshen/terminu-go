package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"
)

// kindsScreen is [K]INDS: a list in groups and a details panel that follows
// its cursor (components/layout/panel, groups and details). [1] KINDS is the
// resource kinds in groups; [2] is the kind under its cursor, named after it.
// Enter on a namespaced kind goes into its details (Rules K3).
type kindsScreen struct {
	screen  *terminu.Screen
	list    *terminu.List
	details *terminu.Panel
	kind    *kind
}

type kind struct {
	name, group, api, short string
	cluster                 bool
	count                   int
}

var kinds = []kind{
	{"pods", "Workloads", "v1", "po", false, 42},
	{"deployments", "Workloads", "apps/v1", "deploy", false, 7},
	{"statefulsets", "Workloads", "apps/v1", "sts", false, 2},
	{"daemonsets", "Workloads", "apps/v1", "ds", false, 3},
	{"jobs", "Workloads", "batch/v1", "", false, 12},
	{"cronjobs", "Workloads", "batch/v1", "cj", false, 4},
	{"services", "Network", "v1", "svc", false, 9},
	{"ingresses", "Network", "networking.k8s.io/v1", "ing", false, 3},
	{"networkpolicies", "Network", "networking.k8s.io/v1", "netpol", false, 1},
	{"configmaps", "Config", "v1", "cm", false, 18},
	{"secrets", "Config", "v1", "", false, 21},
	{"persistentvolumeclaims", "Storage", "v1", "pvc", false, 5},
	{"persistentvolumes", "Storage", "v1", "pv", true, 5},
	{"storageclasses", "Storage", "storage.k8s.io/v1", "sc", true, 2},
	{"nodes", "Cluster", "v1", "no", true, 3},
	{"namespaces", "Cluster", "v1", "ns", true, 6},
}

func newKindsScreen(a *terminu.App) *kindsScreen {
	ks := &kindsScreen{kind: &kinds[0]}
	ks.list = &terminu.List{Filterable: true, EmptyFact: "No kinds match."}
	rows := make([]terminu.Row, len(kinds))
	for i, k := range kinds {
		rows[i] = terminu.Row{ID: k.name, Cells: []string{k.name}, Group: k.group}
	}
	ks.list.SetRows(rows)
	// The details follow the cursor at once, with no Enter; the focus stays
	// (components/layout/panel, details).
	ks.list.OnMove = func(r terminu.Row) tea.Cmd {
		for i := range kinds {
			if kinds[i].name == r.ID {
				ks.kind = &kinds[i]
			}
		}
		ks.details.Label = ks.kind.name
		return nil
	}
	value := func(f func(k kind) string) func() string { return func() string { return f(*ks.kind) } }
	ks.details = &terminu.Panel{Label: ks.kind.name, Body: &terminu.Values{Fields: []*terminu.Field{
		{Label: "Group", Value: value(func(k kind) string { return k.group })},
		{Label: "API version", Value: value(func(k kind) string { return k.api })},
		{Label: "Short name", Value: value(func(k kind) string { return k.short }), Empty: "none", Edit: func() tea.Cmd {
			k := ks.kind
			return a.Text(terminu.TextInput{Title: "Short name", Subject: k.name, Glyph: glyphPencil,
				Prompt: "What kubectl takes for " + k.name, Old: k.short, Done: func(v string) tea.Cmd { k.short = v; return nil }})
		}},
		{Label: "Scope", Value: value(func(k kind) string {
			if k.cluster {
				return "cluster"
			}
			return "namespace"
		})},
		{Label: "Objects", Value: value(func(k kind) string { return fmt.Sprint(k.count) })},
	}}}
	kindsPanel := &terminu.Panel{Label: "Kinds", Body: ks.list, Details: ks.details, Actions: []terminu.Action{
		// Enter goes into a namespaced kind's details, the one place Enter
		// changes panels (Rules K3); a cluster-wide kind is read-only here.
		{Key: "Enter", Label: "Edit", Desc: "go into this kind's details", Scope: terminu.ItemScope, Hint: "edit",
			ToDetails: func() bool { return !ks.kind.cluster },
			Do: func() tea.Cmd {
				if ks.kind.cluster {
					return a.Toast(terminu.Info, ks.kind.name+" is cluster-wide: read-only here")
				}
				return nil
			}},
	}}
	ks.screen = &terminu.Screen{Name: "Kinds", Key: "K", Desc: "a list in groups and details that follow its cursor",
		Panels: []*terminu.Panel{kindsPanel, ks.details}, NarrowBelow: 60, Layout: ks.layout}
	return ks
}

// layout puts the kinds in a sidebar and the details beside it.
func (ks *kindsScreen) layout(w, h int) []terminu.Rect {
	left := min(30, w/2)
	return []terminu.Rect{{X: 0, Y: 0, W: left, H: h}, {X: left, Y: 0, W: w - left, H: h}}
}
