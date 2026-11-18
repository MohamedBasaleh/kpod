package action

import (
	"fmt"
	"strings"
)

type ID string

const (
	Console     ID = "console"
	DBConsole   ID = "dbconsole"
	Runner      ID = "runner"
	Shell       ID = "shell"
	Logs        ID = "logs"
	Describe    ID = "describe"
	YAML        ID = "yaml"
	Events      ID = "events"
	Metrics     ID = "metrics"
	Rollout     ID = "rollout"
	Restart     ID = "restart"
	Delete      ID = "delete"
	Download    ID = "download"
	Upload      ID = "upload"
	PortForward ID = "port-forward"
)

type Group string

const (
	GroupConnect  Group = "Connect"
	GroupObserve  Group = "Observe"
	GroupManage   Group = "Manage"
	GroupTransfer Group = "Transfer"
	GroupNetwork  Group = "Network"
)

// Spec describes a pod action.
type Spec struct {
	ID             ID
	Label          string
	Group          Group
	Aliases        []string
	Interactive    bool
	Destructive    bool
	NeedsContainer bool
	NeedsRunning   bool
	Shortcut       string
}

var catalog = []Spec{
	{ID: Console, Label: "Console", Group: GroupConnect, Aliases: []string{"console", "c", "rails"}, Interactive: true, NeedsContainer: true, NeedsRunning: true, Shortcut: "c"},
	{ID: DBConsole, Label: "DB Console", Group: GroupConnect, Aliases: []string{"dbconsole", "db"}, Interactive: true, NeedsContainer: true, NeedsRunning: true},
	{ID: Runner, Label: "Runner", Group: GroupConnect, Aliases: []string{"runner"}, Interactive: true, NeedsContainer: true, NeedsRunning: true},
	{ID: Shell, Label: "Shell", Group: GroupConnect, Aliases: []string{"shell", "bash", "sh", "exec"}, Interactive: true, NeedsContainer: true, NeedsRunning: true, Shortcut: "b"},
	{ID: Logs, Label: "Logs", Group: GroupObserve, Aliases: []string{"logs", "log", "l"}, Interactive: true, NeedsContainer: true, Shortcut: "l"},
	{ID: Describe, Label: "Describe", Group: GroupObserve, Aliases: []string{"describe"}, Shortcut: "d"},
	{ID: YAML, Label: "YAML", Group: GroupObserve, Aliases: []string{"yaml"}},
	{ID: Events, Label: "Events", Group: GroupObserve, Aliases: []string{"events"}},
	{ID: Metrics, Label: "Metrics", Group: GroupObserve, Aliases: []string{"metrics", "top"}},
	{ID: Rollout, Label: "Rollout Status", Group: GroupManage, Aliases: []string{"rollout"}},
	{ID: Restart, Label: "Restart Deployment", Group: GroupManage, Aliases: []string{"restart"}, Destructive: true},
	{ID: Delete, Label: "Delete Pod", Group: GroupManage, Aliases: []string{"delete", "rm"}, Destructive: true},
	{ID: Download, Label: "Download", Group: GroupTransfer, Aliases: []string{"download", "dl", "get"}, NeedsContainer: true},
	{ID: Upload, Label: "Upload", Group: GroupTransfer, Aliases: []string{"upload", "ul", "put"}, NeedsContainer: true},
	{ID: PortForward, Label: "Port Forward", Group: GroupNetwork, Aliases: []string{"port-forward", "pf", "forward"}, Interactive: true},
}

var aliasIndex = func() map[string]Spec {
	m := map[string]Spec{}
	for _, s := range catalog {
		m[string(s.ID)] = s
		for _, a := range s.Aliases {
			m[strings.ToLower(a)] = s
		}
	}
	return m
}()

func All() []Spec {
	out := make([]Spec, len(catalog))
	copy(out, catalog)
	return out
}

func Resolve(name string) (Spec, error) {
	key := strings.ToLower(strings.TrimSpace(name))
	if spec, ok := aliasIndex[key]; ok {
		return spec, nil
	}
	return Spec{}, fmt.Errorf("unknown action %q", name)
}

func MenuItems() []Spec {
	return All()
}
