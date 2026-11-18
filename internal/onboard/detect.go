package onboard

import (
	"github.com/MohamedBasaleh/kpod/internal/config"
	"github.com/MohamedBasaleh/kpod/internal/migration"
)

// Mode is how kpod should start when invoked with no arguments.
type Mode int

const (
	// ModeConfigured: valid v2 config exists; skip onboarding.
	ModeConfigured Mode = iota
	// ModeMigrate: no v2 config, legacy v1 contexts file exists.
	ModeMigrate
	// ModeFresh: neither v1 nor v2 configuration exists.
	ModeFresh
)

func Detect(paths config.Paths) Mode {
	if _, err := config.NewRepository(paths).Load(); err == nil {
		return ModeConfigured
	}
	if migration.HasLegacyFile(paths) {
		return ModeMigrate
	}
	return ModeFresh
}
