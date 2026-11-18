package action

import "github.com/MohamedBasaleh/kpod/internal/config"

type LogsMode int

const (
	LogsFollow LogsMode = iota
	LogsTail100
	LogsTail500
	LogsPrevious
)

// Request is a fully specified action against one pod.
type Request struct {
	Spec           Spec
	Env            config.EnvironmentLookup
	Pod            string
	Container      string
	Logs           LogsMode
	Tail           int
	RunnerCode     string
	LocalPort      int
	RemotePort     int
	LocalPath      string
	RemotePath     string
	Confirmed      bool
	PreferredShell string
}

type Prepared struct {
	Args        []string
	Interactive bool
	Message     string
}

// Attached reports whether the request should occupy the terminal.
func Attached(req Request) bool {
	switch req.Spec.ID {
	case Shell, Console, DBConsole, Runner, PortForward:
		return true
	case Logs:
		return req.Logs == LogsFollow
	default:
		return false
	}
}
