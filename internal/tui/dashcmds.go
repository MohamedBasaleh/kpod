package tui

import (
	"github.com/MohamedBasaleh/kpod/internal/config"
	"github.com/MohamedBasaleh/kpod/internal/kube"
	"github.com/MohamedBasaleh/kpod/internal/state"
	tea "github.com/charmbracelet/bubbletea"
)

type onboardCompleteMsg struct{}

type startSetupMsg struct{}

type dashContextsMsg struct {
	list []kube.Context
	err  error
}

type dashNamespacesMsg struct {
	kubeContext string
	list        []kube.Namespace
	err         error
}

type dashPodsMsg struct {
	alias   string
	list    []kube.Pod
	err     error
	refresh bool
}

type lastEnvSavedMsg struct{}

func loadDashContextsCmd(svc *kube.Service, refresh bool) tea.Cmd {
	return func() tea.Msg {
		if svc == nil {
			return dashContextsMsg{err: kube.ErrKubectlNotFound}
		}
		ctx, cancel := timeoutCtx()
		defer cancel()
		var list []kube.Context
		var err error
		if refresh {
			list, err = svc.ListContexts(ctx, kube.Refresh())
		} else {
			list, err = svc.ListContexts(ctx)
		}
		return dashContextsMsg{list: list, err: err}
	}
}

func loadDashNamespacesCmd(svc *kube.Service, kubeContext string, refresh bool) tea.Cmd {
	return func() tea.Msg {
		if svc == nil {
			return dashNamespacesMsg{kubeContext: kubeContext, err: kube.ErrKubectlNotFound}
		}
		ctx, cancel := timeoutCtx()
		defer cancel()
		var list []kube.Namespace
		var err error
		if refresh {
			list, err = svc.ListNamespaces(ctx, kubeContext, kube.Refresh())
		} else {
			list, err = svc.ListNamespaces(ctx, kubeContext)
		}
		return dashNamespacesMsg{kubeContext: kubeContext, list: list, err: err}
	}
}

func loadDashPodsCmd(svc *kube.Service, env config.EnvironmentLookup, refresh bool) tea.Cmd {
	return func() tea.Msg {
		if svc == nil {
			return dashPodsMsg{alias: env.Alias, err: kube.ErrKubectlNotFound, refresh: refresh}
		}
		ctx, cancel := timeoutCtx()
		defer cancel()
		var list []kube.Pod
		var err error
		if refresh {
			list, err = svc.ListPods(ctx, env.KubeContext, env.Namespace, kube.Refresh())
		} else {
			list, err = svc.ListPods(ctx, env.KubeContext, env.Namespace)
		}
		return dashPodsMsg{alias: env.Alias, list: list, err: err, refresh: refresh}
	}
}

func saveLastEnvCmd(paths config.Paths, alias string) tea.Cmd {
	return func() tea.Msg {
		_ = state.NewRepository(paths).Mutate(func(st *state.State) error {
			st.LastEnvironment = alias
			return nil
		})
		return lastEnvSavedMsg{}
	}
}
