package migration

import (
	"fmt"
	"os"
	"strings"

	"github.com/MohamedBasaleh/kpod/internal/atomicfile"
	"github.com/MohamedBasaleh/kpod/internal/config"
)

// Result describes a migration attempt.
type Result struct {
	Migrated bool
	Skipped  string
	Config   config.Config
	Backup   string
}

// Legacy mapping policy:
//   - blank lines and lines whose first non-space character is '#' are ignored
//   - every other row must be exactly alias|kube-context|namespace
//   - malformed rows abort with a line-numbered error (nothing is written)
//   - environment aliases must be unique across the whole file
//   - existing valid v2 config.yaml always wins; migration does not run
//   - the original contexts file is never deleted
func HasLegacyFile(paths config.Paths) bool {
	_, err := os.Stat(paths.LegacyContextsFile())
	return err == nil
}

func HasV2Config(paths config.Paths) bool {
	repo := config.NewRepository(paths)
	_, err := repo.Load()
	return err == nil
}

// Migrate converts ~/.config/kpod/contexts into config.yaml when no valid v2 config exists.
func Migrate(paths config.Paths) (Result, error) {
	repo := config.NewRepository(paths)
	if _, err := repo.Load(); err == nil {
		return Result{Skipped: "valid v2 config.yaml already exists"}, nil
	}

	if !HasLegacyFile(paths) {
		return Result{Skipped: "no legacy contexts file"}, nil
	}

	data, err := os.ReadFile(paths.LegacyContextsFile())
	if err != nil {
		return Result{}, fmt.Errorf("read legacy contexts: %w", err)
	}

	cfg, err := ParseLegacy(string(data))
	if err != nil {
		return Result{}, err
	}

	backup := paths.LegacyBackupFile()
	if _, err := os.Stat(backup); os.IsNotExist(err) {
		if err := atomicfile.Write(backup, data, 0o600); err != nil {
			return Result{}, fmt.Errorf("write migration backup: %w", err)
		}
	}

	if err := repo.Save(cfg); err != nil {
		return Result{}, fmt.Errorf("write v2 config: %w", err)
	}

	return Result{
		Migrated: true,
		Config:   cfg,
		Backup:   backup,
	}, nil
}

// ParseLegacy parses v1 alias|context|namespace text into a v2 Config.
func ParseLegacy(text string) (config.Config, error) {
	cfg := config.Default()
	aliasOwner := map[string]string{}
	lines := strings.Split(text, "\n")
	for i, raw := range lines {
		lineNo := i + 1
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) != 3 {
			return config.Config{}, fmt.Errorf("invalid legacy configuration at line %d:\nexpected alias|context|namespace", lineNo)
		}
		alias := strings.TrimSpace(parts[0])
		ctx := strings.TrimSpace(parts[1])
		ns := strings.TrimSpace(parts[2])
		if alias == "" || ctx == "" || ns == "" {
			return config.Config{}, fmt.Errorf("invalid legacy configuration at line %d:\nexpected alias|context|namespace", lineNo)
		}
		if owner, ok := aliasOwner[alias]; ok {
			return config.Config{}, fmt.Errorf("duplicate environment alias %q at line %d (already defined for context %q)", alias, lineNo, owner)
		}
		aliasOwner[alias] = ctx
		entry := cfg.Contexts[ctx]
		if entry.Environments == nil {
			entry.Environments = map[string]config.Environment{}
		}
		entry.Environments[alias] = config.Environment{Namespace: ns}
		cfg.Contexts[ctx] = entry
	}
	if err := config.Validate(cfg); err != nil {
		return config.Config{}, err
	}
	return cfg, nil
}
