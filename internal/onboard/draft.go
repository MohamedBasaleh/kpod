package onboard

import (
	"fmt"
	"sort"
	"strings"

	"github.com/MohamedBasaleh/kpod/internal/config"
)

// Mapping is one environment the user is about to save.
type Mapping struct {
	Alias       string
	KubeContext string
	Namespace   string
}

// Draft is the onboarding wizard's in-memory configuration.
type Draft struct {
	SelectedContexts []string
	// Namespaces[context] are namespaces the user chose to manage.
	Namespaces map[string][]string
	Mappings   []Mapping
}

func NewDraft() Draft {
	return Draft{
		SelectedContexts: []string{},
		Namespaces:       map[string][]string{},
		Mappings:         []Mapping{},
	}
}

func (d *Draft) SelectContext(name string) {
	if contains(d.SelectedContexts, name) {
		return
	}
	d.SelectedContexts = append(d.SelectedContexts, name)
}

func (d *Draft) ToggleContext(name string) {
	if contains(d.SelectedContexts, name) {
		d.SelectedContexts = remove(d.SelectedContexts, name)
		delete(d.Namespaces, name)
		d.Mappings = filterMappings(d.Mappings, func(m Mapping) bool {
			return m.KubeContext != name
		})
		return
	}
	d.SelectedContexts = append(d.SelectedContexts, name)
}

func (d *Draft) ContextSelected(name string) bool {
	return contains(d.SelectedContexts, name)
}

func (d *Draft) SetNamespaces(kubeContext string, names []string) {
	copied := append([]string{}, names...)
	d.Namespaces[kubeContext] = copied
}

func (d *Draft) AddNamespace(kubeContext, namespace string) {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return
	}
	current := d.Namespaces[kubeContext]
	if contains(current, namespace) {
		return
	}
	d.Namespaces[kubeContext] = append(current, namespace)
	if !contains(d.SelectedContexts, kubeContext) {
		d.SelectedContexts = append(d.SelectedContexts, kubeContext)
	}
}

func (d *Draft) ToggleNamespace(kubeContext, namespace string) {
	current := d.Namespaces[kubeContext]
	if contains(current, namespace) {
		d.Namespaces[kubeContext] = remove(current, namespace)
		d.Mappings = filterMappings(d.Mappings, func(m Mapping) bool {
			return !(m.KubeContext == kubeContext && m.Namespace == namespace)
		})
		return
	}
	d.AddNamespace(kubeContext, namespace)
}

func (d *Draft) NamespaceSelected(kubeContext, namespace string) bool {
	return contains(d.Namespaces[kubeContext], namespace)
}

func (d *Draft) TakenAliases() map[string]struct{} {
	taken := map[string]struct{}{}
	for _, m := range d.Mappings {
		if m.Alias != "" {
			taken[m.Alias] = struct{}{}
		}
	}
	return taken
}

func (d *Draft) SetMapping(kubeContext, namespace, alias string) error {
	alias = strings.TrimSpace(alias)
	taken := d.TakenAliases()
	for i, m := range d.Mappings {
		if m.KubeContext == kubeContext && m.Namespace == namespace {
			delete(taken, m.Alias)
			if err := ValidateAlias(alias, taken); err != nil {
				return err
			}
			d.Mappings[i].Alias = alias
			return nil
		}
	}
	if err := ValidateAlias(alias, taken); err != nil {
		return err
	}
	d.Mappings = append(d.Mappings, Mapping{
		Alias:       alias,
		KubeContext: kubeContext,
		Namespace:   namespace,
	})
	return nil
}

func (d Draft) PendingAliasTargets() []Mapping {
	var out []Mapping
	for _, ctx := range d.SelectedContexts {
		for _, ns := range d.Namespaces[ctx] {
			if _, ok := d.mappingFor(ctx, ns); !ok {
				out = append(out, Mapping{KubeContext: ctx, Namespace: ns})
			}
		}
	}
	return out
}

func (d Draft) mappingFor(kubeContext, namespace string) (Mapping, bool) {
	for _, m := range d.Mappings {
		if m.KubeContext == kubeContext && m.Namespace == namespace {
			return m, true
		}
	}
	return Mapping{}, false
}

func (d Draft) HasAnyMapping() bool {
	return len(d.Mappings) > 0
}

func (d Draft) ToConfig() (config.Config, error) {
	cfg := config.Default()
	if len(d.Mappings) == 0 {
		return config.Config{}, fmt.Errorf("add at least one environment")
	}
	for _, m := range d.Mappings {
		if err := ValidateAlias(m.Alias, map[string]struct{}{}); err != nil {
			return config.Config{}, err
		}
		entry := cfg.Contexts[m.KubeContext]
		if entry.Environments == nil {
			entry.Environments = map[string]config.Environment{}
		}
		entry.Environments[m.Alias] = config.Environment{Namespace: m.Namespace}
		cfg.Contexts[m.KubeContext] = entry
	}
	if err := config.Validate(cfg); err != nil {
		return config.Config{}, err
	}
	return cfg, nil
}

func (d Draft) ReviewLines() []string {
	lines := make([]string, 0, len(d.Mappings))
	maps := append([]Mapping{}, d.Mappings...)
	sort.Slice(maps, func(i, j int) bool {
		if maps[i].KubeContext != maps[j].KubeContext {
			return maps[i].KubeContext < maps[j].KubeContext
		}
		return maps[i].Alias < maps[j].Alias
	})
	for _, m := range maps {
		lines = append(lines, fmt.Sprintf("%-12s  %s / %s", m.Alias, m.KubeContext, m.Namespace))
	}
	return lines
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

func remove(list []string, v string) []string {
	out := make([]string, 0, len(list))
	for _, item := range list {
		if item != v {
			out = append(out, item)
		}
	}
	return out
}

func filterMappings(list []Mapping, keep func(Mapping) bool) []Mapping {
	out := make([]Mapping, 0, len(list))
	for _, m := range list {
		if keep(m) {
			out = append(out, m)
		}
	}
	return out
}
