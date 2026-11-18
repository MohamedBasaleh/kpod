package shortcut

import (
	"fmt"
	"os"
	"strings"

	"github.com/MohamedBasaleh/kpod/internal/action"
	"github.com/MohamedBasaleh/kpod/internal/config"
)

type Service struct {
	Paths  config.Paths
	Exists ExistsFunc
	Home   string
	Shell  ShellKind
}

func New(paths config.Paths) Service {
	home, _ := os.UserHomeDir()
	return Service{
		Paths:  paths,
		Exists: LookPathExists,
		Home:   home,
		Shell:  DetectShellFromEnv(),
	}
}

func (s Service) Load() (config.Config, error) {
	return config.NewRepository(s.Paths).Load()
}

func (s Service) Save(cfg config.Config) error {
	return config.NewRepository(s.Paths).Save(cfg)
}

func (s Service) Add(cfg config.Config, id string, env, act string) (config.Config, error) {
	id, sc, err := Prepare(id, config.Shortcut{Environment: env, Action: act})
	if err != nil {
		return cfg, err
	}
	if err := CheckRefs(cfg, id, sc); err != nil {
		return cfg, err
	}
	if err := Collision(cfg, id, s.Exists); err != nil {
		return cfg, err
	}
	spec, _ := action.Resolve(sc.Action)
	if spec.Destructive {
		// allowed, but never bypass confirmation — generated command has no flags
	}
	if cfg.Shortcuts == nil {
		cfg.Shortcuts = map[string]config.Shortcut{}
	}
	cfg.Shortcuts[id] = sc
	return cfg, nil
}

func (s Service) Remove(cfg config.Config, id string) (config.Config, error) {
	id = strings.TrimSpace(id)
	if _, ok := cfg.Shortcuts[id]; !ok {
		return cfg, fmt.Errorf("shortcut %q is not configured", id)
	}
	delete(cfg.Shortcuts, id)
	return cfg, nil
}

func (s Service) Enable(cfg config.Config) config.Config {
	cfg.Settings.ShellShortcuts = true
	return cfg
}

func (s Service) Disable(cfg config.Config) config.Config {
	cfg.Settings.ShellShortcuts = false
	return cfg
}

func (s Service) Generate(cfg config.Config) error {
	return WriteAliasFiles(s.Paths, cfg)
}

func (s Service) Status(cfg config.Config) Status {
	kind := s.Shell
	if kind == "" {
		kind = DetectShellFromEnv()
	}
	rc, alias, snippet := InstallPlan(s.Paths, s.Home, kind)
	st := Status{
		Enabled:   cfg.Settings.ShellShortcuts,
		Shell:     kind,
		RCPath:    rc,
		AliasFile: alias,
		Snippet:   snippet,
		Installed: false,
		Count:     len(cfg.Shortcuts),
		Problems:  nil,
	}
	if b, err := os.ReadFile(rc); err == nil {
		st.Installed = HasManagedBlock(string(b))
	}
	for _, id := range SortedIDs(cfg) {
		if err := CheckRefs(cfg, id, cfg.Shortcuts[id]); err != nil {
			st.Problems = append(st.Problems, err.Error())
		}
	}
	return st
}

type Status struct {
	Enabled   bool
	Shell     ShellKind
	RCPath    string
	AliasFile string
	Snippet   string
	Installed bool
	Count     int
	Problems  []string
}

func (s Service) InstallRC(confirmed bool) error {
	if !confirmed {
		return fmt.Errorf("confirmation required")
	}
	kind := s.Shell
	if kind == "" {
		kind = DetectShellFromEnv()
	}
	_, alias, _ := InstallPlan(s.Paths, s.Home, kind)
	return ApplyRC(RCPath(s.Home, kind), alias, true)
}

func (s Service) UninstallRC(confirmed bool) error {
	if !confirmed {
		return fmt.Errorf("confirmation required")
	}
	kind := s.Shell
	if kind == "" {
		kind = DetectShellFromEnv()
	}
	_, alias, _ := InstallPlan(s.Paths, s.Home, kind)
	return ApplyRC(RCPath(s.Home, kind), alias, false)
}

func ApplyRecommended(cfg config.Config, selected []Candidate) (config.Config, error) {
	s := Service{Exists: func(string) bool { return false }}
	for _, c := range selected {
		var err error
		cfg, err = s.Add(cfg, c.ID, c.Env, string(c.Action))
		if err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}
