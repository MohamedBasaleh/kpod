package kube

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

type podListJSON struct {
	Items []podJSON `json:"items"`
}

type podJSON struct {
	Metadata struct {
		Name              string  `json:"name"`
		Namespace         string  `json:"namespace"`
		CreationTimestamp string  `json:"creationTimestamp"`
		DeletionTimestamp *string `json:"deletionTimestamp"`
	} `json:"metadata"`
	Spec struct {
		NodeName   string          `json:"nodeName"`
		Containers []containerName `json:"containers"`
	} `json:"spec"`
	Status struct {
		Phase                 string            `json:"phase"`
		ContainerStatuses     []containerStatus `json:"containerStatuses"`
		InitContainerStatuses []containerStatus `json:"initContainerStatuses"`
	} `json:"status"`
}

type containerName struct {
	Name string `json:"name"`
}

type containerStatus struct {
	Name         string `json:"name"`
	Ready        bool   `json:"ready"`
	RestartCount int    `json:"restartCount"`
	State        struct {
		Waiting    *containerStateInfo `json:"waiting"`
		Running    *containerStateInfo `json:"running"`
		Terminated *containerStateInfo `json:"terminated"`
	} `json:"state"`
}

type containerStateInfo struct {
	Reason   string `json:"reason"`
	ExitCode int    `json:"exitCode"`
}

// ListPods returns pods for an explicit context and namespace using JSON.
//
//	kubectl --context <context> -n <namespace> get pods -o json
//
// JSON is used instead of tables so READY/STATUS/restarts/containers stay stable.
func (s *Service) ListPods(ctx context.Context, kubeContext, namespace string, opts ...ListOption) ([]Pod, error) {
	if err := s.requireContext(kubeContext); err != nil {
		return nil, err
	}
	if err := s.requireNamespace(namespace); err != nil {
		return nil, err
	}

	key := "pods\x00" + kubeContext + "\x00" + namespace
	o := applyListOptions(opts)
	if !o.refresh {
		if hit, ok := s.cache.get(s.now(), key); ok {
			return clonePods(hit.([]Pod)), nil
		}
	}

	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	res, err := s.run(ctx, "--context", kubeContext, "-n", namespace, "get", "pods", "-o", "json")
	if err := classifyRunError(err, res); err != nil {
		return nil, err
	}

	list, err := parsePods(res.Stdout, s.now())
	if err != nil {
		return nil, newError(ErrMalformedOutput, "malformed kubectl pod JSON", res, err)
	}
	s.cache.set(s.now(), key, clonePods(list), s.ttl.Pods)
	return list, nil
}

func parsePods(raw []byte, now time.Time) ([]Pod, error) {
	var parsed podListJSON
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	out := make([]Pod, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		out = append(out, mapPod(item, now))
	}
	return out, nil
}

func mapPod(item podJSON, now time.Time) Pod {
	created, _ := time.Parse(time.RFC3339, item.Metadata.CreationTimestamp)
	terminating := item.Metadata.DeletionTimestamp != nil && strings.TrimSpace(*item.Metadata.DeletionTimestamp) != ""

	containers := make([]Container, 0, len(item.Spec.Containers)+len(item.Status.InitContainerStatuses))
	for _, ic := range item.Status.InitContainerStatuses {
		containers = append(containers, mapContainer(ic, true))
	}
	statusByName := map[string]containerStatus{}
	for _, cs := range item.Status.ContainerStatuses {
		statusByName[cs.Name] = cs
	}
	if len(item.Spec.Containers) > 0 {
		for _, c := range item.Spec.Containers {
			if cs, ok := statusByName[c.Name]; ok {
				containers = append(containers, mapContainer(cs, false))
				continue
			}
			containers = append(containers, Container{Name: c.Name, State: "unknown"})
		}
	} else {
		for _, cs := range item.Status.ContainerStatuses {
			containers = append(containers, mapContainer(cs, false))
		}
	}

	ready, total, restarts := 0, len(item.Spec.Containers), 0
	if total == 0 {
		total = len(item.Status.ContainerStatuses)
	}
	for _, cs := range item.Status.ContainerStatuses {
		if cs.Ready {
			ready++
		}
		restarts += cs.RestartCount
	}

	var age time.Duration
	if !created.IsZero() && now.After(created) {
		age = now.Sub(created)
	}

	return Pod{
		Name:        item.Metadata.Name,
		Namespace:   item.Metadata.Namespace,
		Node:        item.Spec.NodeName,
		Phase:       item.Status.Phase,
		Status:      effectivePodStatus(item, terminating),
		Ready:       ready,
		Total:       total,
		Restarts:    restarts,
		Age:         age,
		CreatedAt:   created,
		Terminating: terminating,
		Containers:  containers,
	}
}

func mapContainer(cs containerStatus, init bool) Container {
	state, reason := containerState(cs)
	return Container{
		Name:         cs.Name,
		Ready:        cs.Ready,
		RestartCount: cs.RestartCount,
		State:        state,
		Reason:       reason,
		Init:         init,
	}
}

func containerState(cs containerStatus) (string, string) {
	switch {
	case cs.State.Waiting != nil:
		return "waiting", cs.State.Waiting.Reason
	case cs.State.Running != nil:
		return "running", cs.State.Running.Reason
	case cs.State.Terminated != nil:
		return "terminated", cs.State.Terminated.Reason
	default:
		return "unknown", ""
	}
}

func effectivePodStatus(item podJSON, terminating bool) string {
	if terminating {
		return "Terminating"
	}
	for _, cs := range item.Status.InitContainerStatuses {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
			return cs.State.Waiting.Reason
		}
		if cs.State.Terminated != nil && cs.State.Terminated.ExitCode != 0 && cs.State.Terminated.Reason != "" {
			return cs.State.Terminated.Reason
		}
	}
	for _, cs := range item.Status.ContainerStatuses {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
			return cs.State.Waiting.Reason
		}
	}
	switch item.Status.Phase {
	case "Succeeded":
		return "Completed"
	case "Failed":
		for _, cs := range item.Status.ContainerStatuses {
			if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" {
				return cs.State.Terminated.Reason
			}
		}
		return "Error"
	case "":
		return "Unknown"
	default:
		return item.Status.Phase
	}
}

func clonePods(in []Pod) []Pod {
	out := make([]Pod, len(in))
	copy(out, in)
	return out
}
