package onboard

import (
	"fmt"
	"os"
	"sort"

	"github.com/MohamedBasaleh/kpod/internal/config"
	"github.com/MohamedBasaleh/kpod/internal/migration"
)

// LegacyPreview is a read-only view of v1 mappings for the migration screen.
type LegacyPreview struct {
	Mappings []Mapping
}

func LoadLegacyPreview(paths config.Paths) (LegacyPreview, error) {
	data, err := os.ReadFile(paths.LegacyContextsFile())
	if err != nil {
		return LegacyPreview{}, fmt.Errorf("read legacy contexts: %w", err)
	}
	cfg, err := migration.ParseLegacy(string(data))
	if err != nil {
		return LegacyPreview{}, err
	}
	return PreviewFromConfig(cfg), nil
}

func PreviewFromConfig(cfg config.Config) LegacyPreview {
	var mappings []Mapping
	for ctxName, ctx := range cfg.Contexts {
		for alias, env := range ctx.Environments {
			mappings = append(mappings, Mapping{
				Alias:       alias,
				KubeContext: ctxName,
				Namespace:   env.Namespace,
			})
		}
	}
	sort.Slice(mappings, func(i, j int) bool {
		if mappings[i].Alias != mappings[j].Alias {
			return mappings[i].Alias < mappings[j].Alias
		}
		return mappings[i].KubeContext < mappings[j].KubeContext
	})
	return LegacyPreview{Mappings: mappings}
}
