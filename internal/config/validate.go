package config

import (
	"fmt"
	"strings"
)

// Validate checks a v2 config. Environment aliases must be unique across all contexts.
func Validate(c Config) error {
	if c.Version != Version {
		return fmt.Errorf("unsupported config version %d (supported: %d)", c.Version, Version)
	}
	seen := map[string]string{}
	for ctxName, ctx := range c.Contexts {
		if strings.TrimSpace(ctxName) == "" {
			return fmt.Errorf("context name must not be empty")
		}
		for alias, env := range ctx.Environments {
			if strings.TrimSpace(alias) == "" {
				return fmt.Errorf("environment alias must not be empty (context %q)", ctxName)
			}
			if strings.TrimSpace(env.Namespace) == "" {
				return fmt.Errorf("namespace must not be empty (environment %q, context %q)", alias, ctxName)
			}
			if other, ok := seen[alias]; ok {
				return fmt.Errorf("duplicate environment alias %q in contexts %q and %q", alias, other, ctxName)
			}
			seen[alias] = ctxName
		}
	}
	return nil
}
