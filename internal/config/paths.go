package config

import (
	"os"
	"path/filepath"
)

const (
	configFileName     = "config.yaml"
	stateFileName      = "state.yaml"
	legacyContextsName = "contexts"
	legacyBackupName   = "contexts.v1.bak"
	configDirName      = "kpod"
	envConfigDir       = "KPOD_CONFIG_DIR"
)

// Paths locates kpod files under a configuration directory.
type Paths struct {
	Dir string
}

// DefaultPaths returns ~/.config/kpod, honoring XDG_CONFIG_HOME and KPOD_CONFIG_DIR.
func DefaultPaths() (Paths, error) {
	if override := os.Getenv(envConfigDir); override != "" {
		return NewPaths(override), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, err
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		base = filepath.Join(home, ".config")
	}
	return NewPaths(filepath.Join(base, configDirName)), nil
}

// NewPaths uses dir as the kpod configuration directory.
func NewPaths(dir string) Paths {
	return Paths{Dir: dir}
}

func (p Paths) ConfigFile() string {
	return filepath.Join(p.Dir, configFileName)
}

func (p Paths) StateFile() string {
	return filepath.Join(p.Dir, stateFileName)
}

func (p Paths) LegacyContextsFile() string {
	return filepath.Join(p.Dir, legacyContextsName)
}

func (p Paths) LegacyBackupFile() string {
	return filepath.Join(p.Dir, legacyBackupName)
}

func (p Paths) ShellDir() string {
	return filepath.Join(p.Dir, "shell")
}

func (p Paths) AliasFile(shell string) string {
	switch shell {
	case "bash":
		return filepath.Join(p.ShellDir(), "aliases.bash")
	default:
		return filepath.Join(p.ShellDir(), "aliases.zsh")
	}
}
