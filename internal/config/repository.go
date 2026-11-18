package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/MohamedBasaleh/kpod/internal/atomicfile"
	"gopkg.in/yaml.v3"
)

const filePerm os.FileMode = 0o600

// ErrNotExist means config.yaml is missing.
var ErrNotExist = errors.New("config not found")

// Repository loads and saves config.yaml.
type Repository struct {
	paths Paths
}

func NewRepository(paths Paths) Repository {
	return Repository{paths: paths}
}

func (r Repository) Paths() Paths {
	return r.paths
}

func (r Repository) Exists() bool {
	_, err := os.Stat(r.paths.ConfigFile())
	return err == nil
}

func (r Repository) Load() (Config, error) {
	data, err := os.ReadFile(r.paths.ConfigFile())
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, ErrNotExist
		}
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("malformed YAML in %s: %w", r.paths.ConfigFile(), err)
	}
	normalize(&c)
	if err := Validate(c); err != nil {
		return Config{}, err
	}
	return c, nil
}

func (r Repository) Save(c Config) error {
	normalize(&c)
	if err := Validate(c); err != nil {
		return err
	}
	data, err := marshalConfig(c)
	if err != nil {
		return err
	}
	return atomicfile.Write(r.paths.ConfigFile(), data, filePerm)
}

func normalize(c *Config) {
	if c.Contexts == nil {
		c.Contexts = map[string]Context{}
	}
	for name, ctx := range c.Contexts {
		if ctx.Environments == nil {
			ctx.Environments = map[string]Environment{}
		}
		if ctx.IgnoredNamespaces == nil {
			ctx.IgnoredNamespaces = []string{}
		}
		c.Contexts[name] = ctx
	}
	if c.IgnoredContexts == nil {
		c.IgnoredContexts = []string{}
	}
	if c.Shortcuts == nil {
		c.Shortcuts = map[string]Shortcut{}
	}
}

func marshalConfig(c Config) ([]byte, error) {
	data, err := yaml.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("serialize config: %w", err)
	}
	return data, nil
}
