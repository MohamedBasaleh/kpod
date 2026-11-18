package tui

import (
	"context"

	"github.com/MohamedBasaleh/kpod/internal/config"
	"github.com/MohamedBasaleh/kpod/internal/kube"
	"github.com/MohamedBasaleh/kpod/internal/onboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type screen int

const (
	screenWelcome screen = iota
	screenKubectlWait
	screenFatal
	screenMigrate
	screenMigrateView
	screenMigrateWait
	screenMigrateDone
	screenContextsWait
	screenContexts
	screenNamespacesWait
	screenNamespaces
	screenForbidden
	screenManualNS
	screenAccessWait
	screenAccessDenied
	screenAlias
	screenReview
	screenSaveWait
	screenDone
)

type model struct {
	width, height int
	paths         config.Paths
	mode          onboard.Mode
	kube          *kube.Service
	kubeErr       error
	spinner       spinner.Model
	input         textinput.Model
	screen        screen
	status        string
	fatal         string
	help          string

	migrateMenu   menu
	preview       onboard.LegacyPreview
	previewErr    error
	forbiddenMenu menu
	deniedMenu    menu

	contexts   checklist
	namespaces checklist
	queue      []string
	current    string
	draft      onboard.Draft
	pending    []onboard.Mapping
	pendingI   int
	accessNS   string
	doneMsg    string
}

func newModel(paths config.Paths, mode onboard.Mode, svc *kube.Service, kubeErr error) model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	in := textinput.New()
	in.CharLimit = 128
	in.Width = 40
	m := model{
		paths:   paths,
		mode:    mode,
		kube:    svc,
		kubeErr: kubeErr,
		spinner: sp,
		input:   in,
		draft:   onboard.NewDraft(),
		migrateMenu: menu{items: []string{
			"Migrate to kpod v2",
			"View mappings",
			"Exit",
		}},
		forbiddenMenu: menu{items: []string{
			"Add namespace manually",
			"Retry",
			"Skip this context",
		}},
		deniedMenu: menu{items: []string{
			"Try another namespace",
			"Back",
		}},
	}
	if mode == onboard.ModeMigrate {
		m.screen = screenMigrate
		preview, err := onboard.LoadLegacyPreview(paths)
		m.preview = preview
		m.previewErr = err
	} else {
		m.screen = screenWelcome
	}
	return m
}

func (m model) Init() tea.Cmd {
	if m.screen == screenKubectlWait {
		return tea.Batch(m.spinner.Tick, validateKubectlCmd(m.kube, m.kubeErr))
	}
	return m.spinner.Tick
}

func (m *model) listHeight() int {
	h := m.height - 10
	if h < 5 {
		return 5
	}
	if h > 16 {
		return 16
	}
	return h
}

func (m *model) listWidth() int {
	w := m.width - 4
	if w < 20 {
		return 20
	}
	return w
}

func timeoutCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), kubeTimeout)
}
