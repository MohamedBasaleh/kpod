package action

type MenuEntry struct {
	Spec     Spec
	Label    string
	Group    Group
	Logs     LogsMode
	LogsMenu bool
}

func Menu() []MenuEntry {
	return []MenuEntry{
		{Spec: must(Console), Label: "Console", Group: GroupConnect},
		{Spec: must(DBConsole), Label: "DB Console", Group: GroupConnect},
		{Spec: must(Runner), Label: "Runner", Group: GroupConnect},
		{Spec: must(Shell), Label: "Shell", Group: GroupConnect},
		{Spec: must(Logs), Label: "Logs", Group: GroupObserve, LogsMenu: true},
		{Spec: must(Logs), Label: "Previous Logs", Group: GroupObserve, Logs: LogsPrevious},
		{Spec: must(Describe), Label: "Describe", Group: GroupObserve},
		{Spec: must(YAML), Label: "YAML", Group: GroupObserve},
		{Spec: must(Events), Label: "Events", Group: GroupObserve},
		{Spec: must(Metrics), Label: "Metrics", Group: GroupObserve},
		{Spec: must(Rollout), Label: "Rollout Status", Group: GroupManage},
		{Spec: must(Restart), Label: "Restart Deployment", Group: GroupManage},
		{Spec: must(Delete), Label: "Delete Pod", Group: GroupManage},
		{Spec: must(Download), Label: "Download", Group: GroupTransfer},
		{Spec: must(Upload), Label: "Upload", Group: GroupTransfer},
		{Spec: must(PortForward), Label: "Port Forward", Group: GroupNetwork},
	}
}

func LogsChoices() []struct {
	Label string
	Mode  LogsMode
} {
	return []struct {
		Label string
		Mode  LogsMode
	}{
		{Label: "Last 100 lines", Mode: LogsTail100},
		{Label: "Last 500 lines", Mode: LogsTail500},
		{Label: "Follow", Mode: LogsFollow},
	}
}

func must(id ID) Spec {
	s, err := Resolve(string(id))
	if err != nil {
		return Spec{ID: id, Label: string(id)}
	}
	return s
}
