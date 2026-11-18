package action

import (
	"fmt"
	"strings"

	"github.com/MohamedBasaleh/kpod/internal/kube"
)

// MatchPods returns pods whose names contain query (case-insensitive).
// Empty query matches all. When preferRunning is true and any running match
// exists, only running matches are returned; non-running remain if none run.
func MatchPods(pods []kube.Pod, query string, preferRunning bool) []kube.Pod {
	q := strings.ToLower(strings.TrimSpace(query))
	var matched []kube.Pod
	for _, p := range pods {
		if q == "" || strings.Contains(strings.ToLower(p.Name), q) {
			matched = append(matched, p)
		}
	}
	if !preferRunning || len(matched) == 0 {
		return matched
	}
	var running []kube.Pod
	for _, p := range matched {
		if p.Status == "Running" && !p.Terminating {
			running = append(running, p)
		}
	}
	if len(running) > 0 {
		return running
	}
	return matched
}

func AppContainerNames(pod kube.Pod) []string {
	var names []string
	for _, c := range pod.Containers {
		if !c.Init {
			names = append(names, c.Name)
		}
	}
	return names
}

// PickContainer auto-selects a single app container, validates an explicit
// name, or returns ErrNeedContainer when the caller must prompt.
var ErrNeedContainer = fmt.Errorf("container selection required")

func PickContainer(pod kube.Pod, explicit string) (string, error) {
	names := AppContainerNames(pod)
	explicit = strings.TrimSpace(explicit)
	if explicit != "" {
		for _, n := range names {
			if n == explicit {
				return n, nil
			}
		}
		return "", fmt.Errorf("%w: %s", kube.ErrContainerNotFound, explicit)
	}
	switch len(names) {
	case 0:
		return "", kube.ErrContainerNotFound
	case 1:
		return names[0], nil
	default:
		return "", ErrNeedContainer
	}
}

func IsRunning(pod kube.Pod) bool {
	return pod.Status == "Running" && !pod.Terminating
}
