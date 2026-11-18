package version

import (
	"fmt"
	"strings"
)

// Set at link time by GoReleaser / -ldflags.
var (
	Version = "2.0.0-dev"
	Commit  = "unknown"
	Date    = "unknown"
	Repo    = "MohamedBasaleh/kpod"
)

func Display() string {
	v := strings.TrimPrefix(strings.TrimSpace(Version), "v")
	if v == "" {
		return "2.0.0-dev"
	}
	return v
}

func Line() string {
	return "kpod " + Display()
}

func Verbose() string {
	return fmt.Sprintf("kpod %s\ncommit %s\nbuilt  %s\n", Display(), Commit, Date)
}

func UserAgent() string {
	return "kpod/" + Display()
}
