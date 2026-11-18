package config

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Clone returns a deep copy of config.
func Clone(c Config) Config {
	data, err := yaml.Marshal(c)
	if err != nil {
		return Default()
	}
	var out Config
	if err := yaml.Unmarshal(data, &out); err != nil {
		return Default()
	}
	normalize(&out)
	return out
}

// TakenAliases returns every configured environment name.
func (c Config) TakenAliases() map[string]struct{} {
	taken := map[string]struct{}{}
	for _, alias := range c.Aliases() {
		taken[alias] = struct{}{}
	}
	return taken
}

// ContextIgnored reports whether the user asked not to be notified about a context.
func (c Config) ContextIgnored(name string) bool {
	return containsString(c.IgnoredContexts, name)
}

// NamespaceIgnored reports whether the user asked not to be notified about a namespace.
func (c Config) NamespaceIgnored(kubeContext, namespace string) bool {
	ctx, ok := c.Contexts[kubeContext]
	if !ok {
		return false
	}
	return containsString(ctx.IgnoredNamespaces, namespace)
}

// AddEnvironment merges one alias mapping into an existing config.
func AddEnvironment(c Config, kubeContext, namespace, alias string) (Config, error) {
	c = Clone(c)
	kubeContext = strings.TrimSpace(kubeContext)
	namespace = strings.TrimSpace(namespace)
	alias = strings.TrimSpace(alias)
	if kubeContext == "" {
		return c, fmt.Errorf("kubernetes context must not be empty")
	}
	if namespace == "" {
		return c, fmt.Errorf("namespace must not be empty")
	}
	if alias == "" {
		return c, fmt.Errorf("environment name must not be empty")
	}
	if strings.ContainsAny(alias, " \t") {
		return c, fmt.Errorf("environment name must not contain spaces")
	}
	if strings.ContainsAny(alias, ";&|`$()<>\\\"'") {
		return c, fmt.Errorf("environment name contains unsupported characters")
	}
	if existing := c.AliasForNamespace(kubeContext, namespace); existing != "" {
		return c, fmt.Errorf("namespace %q is already environment %q", namespace, existing)
	}
	ctx := c.Contexts[kubeContext]
	if ctx.Environments == nil {
		ctx.Environments = map[string]Environment{}
	}
	if ctx.IgnoredNamespaces == nil {
		ctx.IgnoredNamespaces = []string{}
	}
	ctx.Environments[alias] = Environment{Namespace: namespace}
	c.Contexts[kubeContext] = ctx
	if err := Validate(c); err != nil {
		return Config{}, err
	}
	return c, nil
}

// IgnoreContext records that a kubeconfig context should not produce "new" notifications.
func IgnoreContext(c Config, name string) Config {
	c = Clone(c)
	name = strings.TrimSpace(name)
	if name == "" || containsString(c.IgnoredContexts, name) {
		return c
	}
	c.IgnoredContexts = append(c.IgnoredContexts, name)
	return c
}

// IgnoreNamespace records that a namespace should not produce "new" notifications.
func IgnoreNamespace(c Config, kubeContext, namespace string) Config {
	c = Clone(c)
	kubeContext = strings.TrimSpace(kubeContext)
	namespace = strings.TrimSpace(namespace)
	if kubeContext == "" || namespace == "" {
		return c
	}
	ctx := c.Contexts[kubeContext]
	if ctx.Environments == nil {
		ctx.Environments = map[string]Environment{}
	}
	if ctx.IgnoredNamespaces == nil {
		ctx.IgnoredNamespaces = []string{}
	}
	if !containsString(ctx.IgnoredNamespaces, namespace) {
		ctx.IgnoredNamespaces = append(ctx.IgnoredNamespaces, namespace)
	}
	c.Contexts[kubeContext] = ctx
	return c
}

func containsString(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}
