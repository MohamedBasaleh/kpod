package discover

import (
	"context"
	"errors"

	"github.com/MohamedBasaleh/kpod/internal/config"
	"github.com/MohamedBasaleh/kpod/internal/kube"
	"github.com/MohamedBasaleh/kpod/internal/state"
)

// Cluster is the Kubernetes inventory kpod needs for discovery.
type Cluster interface {
	ListContexts(ctx context.Context, opts ...kube.ListOption) ([]kube.Context, error)
	ListNamespaces(ctx context.Context, kubeContext string, opts ...kube.ListOption) ([]kube.Namespace, error)
}

type Options struct {
	Refresh bool
}

type Outcome struct {
	Result     Result
	State      state.State
	StateDirty bool
}

func names(list []kube.Context) []string {
	out := make([]string, 0, len(list))
	for _, c := range list {
		if c.Name != "" {
			out = append(out, c.Name)
		}
	}
	return out
}

func nsNames(list []kube.Namespace) []string {
	out := make([]string, 0, len(list))
	for _, n := range list {
		if n.Name != "" {
			out = append(out, n.Name)
		}
	}
	return out
}

func listOpts(refresh bool) []kube.ListOption {
	if refresh {
		return []kube.ListOption{kube.Refresh()}
	}
	return nil
}

func nsUnavailable(err error) bool {
	return errors.Is(err, kube.ErrForbidden) || errors.Is(err, kube.ErrUnauthorized)
}

// Run compares live Kubernetes inventory with seen state.
// It never writes configuration and never mutates kubeconfig.
func Run(ctx context.Context, cluster Cluster, cfg config.Config, st state.State, opts Options) Outcome {
	st = state.Clone(st)
	out := Outcome{
		Result: Result{NewNamespaces: map[string][]string{}},
		State:  st,
	}
	if cluster == nil {
		out.Result.ContextsErr = kube.ErrKubectlNotFound
		return out
	}

	current, err := cluster.ListContexts(ctx, listOpts(opts.Refresh)...)
	if err != nil {
		out.Result.ContextsErr = err
	} else {
		currentNames := names(current)
		if NeedsContextBaseline(st) {
			ApplyContextBaseline(&st, currentNames, cfg)
			out.StateDirty = true
		} else {
			out.Result.NewContexts = DiffContexts(currentNames, cfg, st)
		}
	}

	for _, kubeContext := range cfg.ConfiguredContextNames() {
		if cfg.ContextIgnored(kubeContext) {
			continue
		}
		list, err := cluster.ListNamespaces(ctx, kubeContext, listOpts(opts.Refresh)...)
		if err != nil {
			if nsUnavailable(err) {
				out.Result.Unavailable = append(out.Result.Unavailable, kubeContext)
				continue
			}
			out.Result.Unavailable = append(out.Result.Unavailable, kubeContext)
			continue
		}
		currentNS := nsNames(list)
		if NeedsNamespaceBaseline(st, kubeContext) {
			ApplyNamespaceBaseline(&st, kubeContext, currentNS, cfg)
			out.StateDirty = true
			continue
		}
		found := DiffNamespaces(kubeContext, currentNS, cfg, st)
		if len(found) > 0 {
			out.Result.NewNamespaces[kubeContext] = found
		}
	}

	out.State = st
	return out
}

func PersistSeen(repo state.Repository, fn func(*state.State)) error {
	if fn == nil {
		return nil
	}
	return repo.Mutate(func(s *state.State) error {
		fn(s)
		return nil
	})
}
