package discover

import (
	"sort"

	"github.com/MohamedBasaleh/kpod/internal/config"
	"github.com/MohamedBasaleh/kpod/internal/state"
)

// DiffContexts returns kubeconfig contexts that are new relative to seen/ignored/configured.
func DiffContexts(current []string, cfg config.Config, st state.State) []string {
	var out []string
	for _, name := range current {
		if name == "" {
			continue
		}
		if cfg.ContextIgnored(name) {
			continue
		}
		if _, configured := cfg.Contexts[name]; configured {
			continue
		}
		if state.Contains(st.SeenContexts, name) {
			continue
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// DiffNamespaces returns namespaces in one context that are new relative to seen/ignored/configured.
func DiffNamespaces(kubeContext string, current []string, cfg config.Config, st state.State) []string {
	var out []string
	for _, name := range current {
		if name == "" {
			continue
		}
		if cfg.NamespaceIgnored(kubeContext, name) {
			continue
		}
		if cfg.AliasForNamespace(kubeContext, name) != "" {
			continue
		}
		if state.Contains(st.SeenNamespaces[kubeContext], name) {
			continue
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func ApplyContextBaseline(st *state.State, current []string, cfg config.Config) {
	if st == nil {
		return
	}
	for _, name := range current {
		state.MarkContextSeen(st, name)
	}
	for _, name := range cfg.ConfiguredContextNames() {
		state.MarkContextSeen(st, name)
	}
	st.DiscoveryReady = true
}

func ApplyNamespaceBaseline(st *state.State, kubeContext string, current []string, cfg config.Config) {
	if st == nil || kubeContext == "" {
		return
	}
	state.EnsureNamespaceKey(st, kubeContext)
	for _, name := range current {
		state.MarkNamespaceSeen(st, kubeContext, name)
	}
	for _, env := range cfg.EnvironmentsForContext(kubeContext) {
		state.MarkNamespaceSeen(st, kubeContext, env.Namespace)
	}
}

func NeedsContextBaseline(st state.State) bool {
	return !st.DiscoveryReady
}

func NeedsNamespaceBaseline(st state.State, kubeContext string) bool {
	return !state.NamespacesInitialized(st, kubeContext)
}
