package config

const Version = 2

// Config is durable user configuration (no Kubernetes credentials).
type Config struct {
	Version         int                 `yaml:"version"`
	Contexts        map[string]Context  `yaml:"contexts"`
	IgnoredContexts []string            `yaml:"ignored_contexts"`
	Shortcuts       map[string]Shortcut `yaml:"shortcuts"`
	Settings        Settings            `yaml:"settings"`
}

// Context is a Kubernetes context managed by kpod.
// Presence in Config.Contexts means kpod manages it.
type Context struct {
	Environments      map[string]Environment `yaml:"environments"`
	IgnoredNamespaces []string               `yaml:"ignored_namespaces,omitempty"`
}

// Environment maps a globally unique alias to a namespace inside a context.
type Environment struct {
	Namespace string `yaml:"namespace"`
}

// Shortcut is a user-defined shell helper (k-<id>) mapped to an environment action.
type Shortcut struct {
	Environment string `yaml:"environment"`
	Action      string `yaml:"action"`
}

// Settings holds user preferences.
type Settings struct {
	Notifications  bool `yaml:"notifications"`
	ShellShortcuts bool `yaml:"shell_shortcuts"`
}

// Default returns an empty valid v2 config.
func Default() Config {
	return Config{
		Version:         Version,
		Contexts:        map[string]Context{},
		IgnoredContexts: []string{},
		Shortcuts:       map[string]Shortcut{},
		Settings: Settings{
			Notifications:  true,
			ShellShortcuts: false,
		},
	}
}

// EnvironmentLookup is a resolved environment alias.
type EnvironmentLookup struct {
	Alias       string
	KubeContext string
	Namespace   string
}

// LookupEnvironment finds an alias across all contexts.
func (c Config) LookupEnvironment(alias string) (EnvironmentLookup, bool) {
	for ctxName, ctx := range c.Contexts {
		if env, ok := ctx.Environments[alias]; ok {
			return EnvironmentLookup{
				Alias:       alias,
				KubeContext: ctxName,
				Namespace:   env.Namespace,
			}, true
		}
	}
	return EnvironmentLookup{}, false
}
