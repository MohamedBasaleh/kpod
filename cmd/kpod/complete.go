package main

import (
	"strings"

	"github.com/MohamedBasaleh/kpod/internal/action"
	"github.com/MohamedBasaleh/kpod/internal/config"
	"github.com/spf13/cobra"
)

func completeRoot(configDir string, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	none := cobra.ShellCompDirectiveNoFileComp
	if len(args) >= 2 {
		return nil, none
	}
	if len(args) == 1 {
		return filterPrefix(actionCompletions(), toComplete), none
	}
	paths, err := resolvePaths(configDir)
	if err != nil {
		return nil, none
	}
	cfg, err := config.NewRepository(paths).Load()
	if err != nil {
		return nil, none
	}
	return filterPrefix(cfg.Aliases(), toComplete), none
}

func actionCompletions() []string {
	var out []string
	seen := map[string]struct{}{}
	for _, spec := range action.All() {
		for _, name := range spec.Aliases {
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			out = append(out, name)
		}
	}
	return out
}

func filterPrefix(items []string, prefix string) []string {
	if prefix == "" {
		return items
	}
	var out []string
	for _, item := range items {
		if strings.HasPrefix(item, prefix) {
			out = append(out, item)
		}
	}
	return out
}
