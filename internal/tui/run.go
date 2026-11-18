package tui

import (
	"fmt"
	"io"
	"os"

	"github.com/MohamedBasaleh/kpod/internal/config"
	"github.com/MohamedBasaleh/kpod/internal/kube"
	"github.com/MohamedBasaleh/kpod/internal/onboard"
	"github.com/MohamedBasaleh/kpod/internal/resolve"
	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

// Options configure the interactive session.
type Options struct {
	Paths    config.Paths
	Out      io.Writer
	Env      string
	Debug    bool
	OpenPods bool
}

// Start is the entrypoint for `kpod` and `kpod <environment>`.
func Start(opts Options) error {
	repo := config.NewRepository(opts.Paths)
	if repo.Exists() {
		cfg, err := repo.Load()
		if err != nil {
			return fmt.Errorf("could not load kpod configuration: %w", err)
		}
		mode := onboard.ModeConfigured
		var open *config.EnvironmentLookup
		if opts.Env != "" {
			env, err := resolve.Lookup(cfg, opts.Env)
			if err != nil {
				return err
			}
			open = &env
		}
		return runApp(opts, mode, cfg, open, opts.OpenPods)
	}
	mode := onboard.Detect(opts.Paths)
	if opts.Env != "" {
		return fmt.Errorf("kpod is not configured yet.\n\nEnvironment %q cannot be opened until setup is complete.", opts.Env)
	}
	return runApp(opts, mode, config.Config{}, nil, false)
}

func runApp(opts Options, mode onboard.Mode, cfg config.Config, open *config.EnvironmentLookup, openPods bool) error {
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return fmt.Errorf("kpod needs an interactive terminal")
	}
	var kubeOpts []kube.Option
	if opts.Debug {
		kubeOpts = append(kubeOpts, kube.WithDebug(func(info kube.DebugInfo) {
			fmt.Fprintf(os.Stderr, "kubectl args=%q exit=%d duration=%s stderr=%q\n",
				info.Args, info.ExitCode, info.Duration, info.Stderr)
		}))
	}
	svc, kubeErr := kube.New(kubeOpts...)
	a := newApp(opts.Paths, mode, svc, kubeErr, opts.Debug, cfg, open, openPods)
	_, err := tea.NewProgram(a, tea.WithAltScreen()).Run()
	return err
}
