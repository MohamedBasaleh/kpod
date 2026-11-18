package config

import "sort"

// Environments returns every configured environment, sorted by alias.
func (c Config) Environments() []EnvironmentLookup {
	var list []EnvironmentLookup
	for ctxName, ctx := range c.Contexts {
		for alias, env := range ctx.Environments {
			list = append(list, EnvironmentLookup{
				Alias:       alias,
				KubeContext: ctxName,
				Namespace:   env.Namespace,
			})
		}
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Alias < list[j].Alias
	})
	return list
}

// Aliases returns sorted environment names.
func (c Config) Aliases() []string {
	envs := c.Environments()
	out := make([]string, 0, len(envs))
	for _, env := range envs {
		out = append(out, env.Alias)
	}
	return out
}

// ConfiguredContextNames returns sorted Kubernetes context names present in config.
func (c Config) ConfiguredContextNames() []string {
	names := make([]string, 0, len(c.Contexts))
	for name := range c.Contexts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// EnvironmentsForContext returns aliases for one Kubernetes context, sorted by alias.
func (c Config) EnvironmentsForContext(kubeContext string) []EnvironmentLookup {
	var list []EnvironmentLookup
	ctx, ok := c.Contexts[kubeContext]
	if !ok {
		return list
	}
	for alias, env := range ctx.Environments {
		list = append(list, EnvironmentLookup{
			Alias:       alias,
			KubeContext: kubeContext,
			Namespace:   env.Namespace,
		})
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Alias < list[j].Alias
	})
	return list
}

// AliasForNamespace returns the environment alias mapped to a namespace in a context.
func (c Config) AliasForNamespace(kubeContext, namespace string) string {
	ctx, ok := c.Contexts[kubeContext]
	if !ok {
		return ""
	}
	for alias, env := range ctx.Environments {
		if env.Namespace == namespace {
			return alias
		}
	}
	return ""
}
