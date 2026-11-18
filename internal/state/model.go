package state

const Version = 1

// State is runtime/discovery information. It is not user-facing configuration.
type State struct {
	Version         int                 `yaml:"version"`
	LastEnvironment string              `yaml:"last_environment"`
	SeenContexts    []string            `yaml:"seen_contexts"`
	SeenNamespaces  map[string][]string `yaml:"seen_namespaces"`
	// DiscoveryReady is set after the first successful context inventory.
	// An empty seen list before this flag must be treated as uninitialized, not
	// as "everything is new".
	DiscoveryReady bool `yaml:"discovery_ready,omitempty"`
}

// Default returns an empty valid v1 state.
func Default() State {
	return State{
		Version:        Version,
		SeenContexts:   []string{},
		SeenNamespaces: map[string][]string{},
	}
}
