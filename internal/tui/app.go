package tui

import (
	"github.com/MohamedBasaleh/kpod/internal/config"
	"github.com/MohamedBasaleh/kpod/internal/kube"
	"github.com/MohamedBasaleh/kpod/internal/onboard"
	tea "github.com/charmbracelet/bubbletea"
)

type phase int

const (
	phaseOnboard phase = iota
	phaseMain
)

type app struct {
	width, height int
	paths         config.Paths
	kube          *kube.Service
	kubeErr       error
	debug         bool
	phase         phase
	onboard       model
	main          mainModel
}

func newApp(paths config.Paths, mode onboard.Mode, svc *kube.Service, kubeErr error, debug bool, cfg config.Config, open *config.EnvironmentLookup, openPods bool) app {
	a := app{
		paths:   paths,
		kube:    svc,
		kubeErr: kubeErr,
		debug:   debug,
	}
	if mode == onboard.ModeConfigured {
		a.phase = phaseMain
		a.main = newMainModel(paths, svc, kubeErr, cfg, debug, open, openPods)
		return a
	}
	a.phase = phaseOnboard
	a.onboard = newModel(paths, mode, svc, kubeErr)
	return a
}

func (a app) Init() tea.Cmd {
	if a.phase == phaseMain {
		return a.main.Init()
	}
	return a.onboard.Init()
}

func (a app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		a.width, a.height = ws.Width, ws.Height
	}
	switch msg.(type) {
	case onboardCompleteMsg:
		return a.enterMain(nil)
	case startSetupMsg:
		a.phase = phaseOnboard
		a.onboard = newModel(a.paths, onboard.ModeFresh, a.kube, a.kubeErr)
		a.onboard.width, a.onboard.height = a.width, a.height
		return a, a.onboard.Init()
	}
	if a.phase == phaseOnboard {
		m, cmd := a.onboard.Update(msg)
		a.onboard = m.(model)
		return a, cmd
	}
	m, cmd := a.main.Update(msg)
	a.main = m.(mainModel)
	return a, cmd
}

func (a app) View() string {
	if a.phase == phaseOnboard {
		return a.onboard.View()
	}
	return a.main.View()
}

func (a app) enterMain(open *config.EnvironmentLookup) (tea.Model, tea.Cmd) {
	cfg, err := config.NewRepository(a.paths).Load()
	if err != nil {
		a.phase = phaseMain
		a.main = newMainModel(a.paths, a.kube, a.kubeErr, config.Default(), a.debug, nil, false)
		a.main.current.view = viewError
		a.main.status = "Could not load kpod configuration.\n\n" + err.Error()
		a.main.width, a.main.height = a.width, a.height
		return a, nil
	}
	a.phase = phaseMain
	a.main = newMainModel(a.paths, a.kube, a.kubeErr, cfg, a.debug, open, false)
	a.main.width, a.main.height = a.width, a.height
	return a, a.main.Init()
}
