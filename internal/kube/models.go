package kube

import "time"

// Context is a Kubernetes kubeconfig context name.
type Context struct {
	Name string
}

// Namespace is a cluster namespace.
type Namespace struct {
	Name string
}

// Pod is a structured view of a Kubernetes pod for the future browser.
type Pod struct {
	Name        string
	Namespace   string
	Node        string
	Phase       string
	Status      string
	Ready       int
	Total       int
	Restarts    int
	Age         time.Duration
	CreatedAt   time.Time
	Terminating bool
	Containers  []Container
}

// Container is a pod container (app or init).
type Container struct {
	Name         string
	Ready        bool
	RestartCount int
	State        string
	Reason       string
	Init         bool
}

// NamespaceAccess is the result of probing a manually supplied namespace.
type NamespaceAccess struct {
	Allowed bool
}

// DebugInfo is a concise record of one kubectl invocation. It must never
// include kubeconfig contents or credentials.
type DebugInfo struct {
	Args     []string
	Duration time.Duration
	ExitCode int
	Stderr   string
}

type listOpts struct {
	refresh bool
}

// ListOption mutates a discovery call.
type ListOption func(*listOpts)

// Refresh bypasses the in-memory cache for this call.
func Refresh() ListOption {
	return func(o *listOpts) { o.refresh = true }
}

func applyListOptions(opts []ListOption) listOpts {
	var o listOpts
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	return o
}
