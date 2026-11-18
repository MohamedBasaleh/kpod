package tui

import (
	"fmt"
	"sort"
	"time"

	"github.com/MohamedBasaleh/kpod/internal/kube"
	"github.com/charmbracelet/lipgloss"
)

// formatAge renders a Kubernetes age using the duration already derived
// from creationTimestamp (no extra kubectl, no wall clock).
func formatAge(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	sec := int(d / time.Second)
	if sec < 60 {
		return fmt.Sprintf("%ds", sec)
	}
	min := int(d / time.Minute)
	if min < 60 {
		return fmt.Sprintf("%dm", min)
	}
	hour := int(d / time.Hour)
	if hour < 24 {
		return fmt.Sprintf("%dh", hour)
	}
	return fmt.Sprintf("%dd", hour/24)
}

func readyString(p kube.Pod) string {
	return fmt.Sprintf("%d/%d", p.Ready, p.Total)
}

func sortPods(pods []kube.Pod) []kube.Pod {
	out := append([]kube.Pod{}, pods...)
	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := podRank(out[i]), podRank(out[j])
		if ri != rj {
			return ri < rj
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func podRank(p kube.Pod) int {
	if p.Status == "Running" && !p.Terminating {
		return 0
	}
	return 1
}

func appContainers(p kube.Pod) []kube.Container {
	out := make([]kube.Container, 0, len(p.Containers))
	for _, c := range p.Containers {
		if !c.Init {
			out = append(out, c)
		}
	}
	return out
}

func initContainers(p kube.Pod) []kube.Container {
	out := make([]kube.Container, 0, len(p.Containers))
	for _, c := range p.Containers {
		if c.Init {
			out = append(out, c)
		}
	}
	return out
}

func statusStyle(status string) lipgloss.Style {
	switch status {
	case "Running":
		return styleOK
	case "Pending", "Terminating":
		return styleWarn
	case "Completed":
		return styleMuted
	case "CrashLoopBackOff", "ImagePullBackOff", "Error", "Failed":
		return styleErr
	default:
		if status != "" {
			return styleWarn
		}
		return lipgloss.NewStyle()
	}
}

func indexPodByName(pods []kube.Pod, name string) int {
	if name == "" {
		return 0
	}
	for i, p := range pods {
		if p.Name == name {
			return i
		}
	}
	return 0
}

func selectedPodName(pods []kube.Pod, cursor int) string {
	if cursor < 0 || cursor >= len(pods) {
		return ""
	}
	return pods[cursor].Name
}

type podColSet struct {
	name, ready, status, restarts, age, node int
	showNode                                 bool
	showRestarts                             bool
	showAge                                  bool
}

func podColumns(width int) podColSet {
	if width < 40 {
		width = 40
	}
	cols := podColSet{
		name:         18,
		ready:        5,
		status:       16,
		restarts:     8,
		age:          5,
		node:         12,
		showNode:     false,
		showRestarts: true,
		showAge:      true,
	}
	fixed := 2 + cols.ready + 2 + cols.status
	switch {
	case width >= 100:
		cols.showNode = true
		cols.name = width - (fixed + 2 + cols.restarts + 2 + cols.age + 2 + cols.node + 4)
	case width >= 72:
		cols.name = width - (fixed + 2 + cols.restarts + 2 + cols.age + 4)
	case width >= 56:
		cols.showAge = true
		cols.showRestarts = false
		cols.name = width - (fixed + 2 + cols.age + 4)
	default:
		cols.showAge = false
		cols.showRestarts = false
		cols.name = width - (fixed + 4)
	}
	if cols.name < 10 {
		cols.name = 10
	}
	return cols
}
