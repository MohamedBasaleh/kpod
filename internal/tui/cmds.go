package tui

import (
	"time"

	"github.com/MohamedBasaleh/kpod/internal/config"
	"github.com/MohamedBasaleh/kpod/internal/kube"
	"github.com/MohamedBasaleh/kpod/internal/migration"
	tea "github.com/charmbracelet/bubbletea"
)

const kubeTimeout = 25 * time.Second

type kubectlOKMsg struct{}

type kubectlErrMsg struct{ err error }

type contextsMsg struct {
	list []kube.Context
	err  error
}

type namespacesMsg struct {
	list []kube.Namespace
	err  error
}

type accessMsg struct {
	ns      string
	allowed bool
	err     error
}

type migratedMsg struct {
	result migration.Result
	err    error
}

type savedMsg struct{ err error }

func validateKubectlCmd(svc *kube.Service, newErr error) tea.Cmd {
	return func() tea.Msg {
		if newErr != nil {
			return kubectlErrMsg{newErr}
		}
		if svc == nil {
			return kubectlErrMsg{kube.ErrKubectlNotFound}
		}
		ctx, cancel := timeoutCtx()
		defer cancel()
		if err := svc.Validate(ctx); err != nil {
			return kubectlErrMsg{err}
		}
		return kubectlOKMsg{}
	}
}

func loadContextsCmd(svc *kube.Service) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := timeoutCtx()
		defer cancel()
		list, err := svc.ListContexts(ctx, kube.Refresh())
		return contextsMsg{list: list, err: err}
	}
}

func loadNamespacesCmd(svc *kube.Service, kubeContext string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := timeoutCtx()
		defer cancel()
		list, err := svc.ListNamespaces(ctx, kubeContext, kube.Refresh())
		return namespacesMsg{list: list, err: err}
	}
}

func accessNamespaceCmd(svc *kube.Service, kubeContext, namespace string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := timeoutCtx()
		defer cancel()
		access, err := svc.CanAccessNamespace(ctx, kubeContext, namespace)
		return accessMsg{ns: namespace, allowed: access.Allowed, err: err}
	}
}

func migrateCmd(paths config.Paths) tea.Cmd {
	return func() tea.Msg {
		result, err := migration.Migrate(paths)
		return migratedMsg{result: result, err: err}
	}
}

func saveConfigCmd(paths config.Paths, cfg config.Config) tea.Cmd {
	return func() tea.Msg {
		err := config.NewRepository(paths).Save(cfg)
		return savedMsg{err: err}
	}
}
