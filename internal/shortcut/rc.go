package shortcut

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/MohamedBasaleh/kpod/internal/config"
)

const (
	managedComment = "# kpod shortcuts"
)

type ShellKind string

const (
	ShellZsh  ShellKind = "zsh"
	ShellBash ShellKind = "bash"
)

func DetectShell(shellEnv string) ShellKind {
	base := filepath.Base(shellEnv)
	switch {
	case strings.Contains(base, "bash"):
		return ShellBash
	default:
		return ShellZsh
	}
}

func DetectShellFromEnv() ShellKind {
	return DetectShell(os.Getenv("SHELL"))
}

func RCPath(home string, kind ShellKind) string {
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	if kind == ShellBash {
		bashrc := filepath.Join(home, ".bashrc")
		profile := filepath.Join(home, ".bash_profile")
		if _, err := os.Stat(bashrc); err == nil {
			return bashrc
		}
		if runtime.GOOS == "darwin" {
			if _, err := os.Stat(profile); err == nil {
				return profile
			}
		}
		return bashrc
	}
	return filepath.Join(home, ".zshrc")
}

func SourceLine(aliasFile string) string {
	return fmt.Sprintf("source %q", aliasFile)
}

func ManagedBlock(aliasFile string) string {
	return managedComment + "\n" + SourceLine(aliasFile) + "\n"
}

func HasManagedBlock(content string) bool {
	return strings.Contains(content, managedComment) ||
		strings.Contains(content, "/kpod/shell/aliases.zsh") ||
		strings.Contains(content, "/kpod/shell/aliases.bash") ||
		strings.Contains(content, filepath.ToSlash("/shell/aliases.zsh")) ||
		strings.Contains(content, filepath.ToSlash("/shell/aliases.bash"))
}

func InsertManagedBlock(content, aliasFile string) (string, bool) {
	if HasManagedBlock(content) {
		return content, false
	}
	block := ManagedBlock(aliasFile)
	if content == "" {
		return block, true
	}
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return content + "\n" + block, true
}

func RemoveManagedBlock(content, aliasFile string) (string, bool) {
	if !HasManagedBlock(content) {
		return content, false
	}
	lines := strings.Split(content, "\n")
	var out []string
	skipNext := false
	changed := false
	src := SourceLine(aliasFile)
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if skipNext {
			skipNext = false
			changed = true
			continue
		}
		trim := strings.TrimSpace(line)
		if trim == managedComment {
			skipNext = true
			changed = true
			continue
		}
		if trim == strings.TrimSpace(src) || strings.Contains(trim, "/shell/aliases.zsh") || strings.Contains(trim, "/shell/aliases.bash") {
			if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == managedComment {
				out = out[:len(out)-1]
			}
			changed = true
			continue
		}
		out = append(out, line)
	}
	result := strings.Join(out, "\n")
	return result, changed
}

// RCMatch is a kpod-managed source line found in a shell rc file.
type RCMatch struct {
	Path  string
	Lines []string
}

// FindManagedLines reports kpod source lines in common shell rc files.
// It does not modify any file.
func FindManagedLines(home string) []RCMatch {
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	names := []string{".zshrc", ".bashrc", ".bash_profile", ".profile"}
	var out []RCMatch
	for _, name := range names {
		path := filepath.Join(home, name)
		body, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := string(body)
		if !HasManagedBlock(content) {
			continue
		}
		var lines []string
		for _, line := range strings.Split(content, "\n") {
			trim := strings.TrimSpace(line)
			if trim == managedComment || strings.Contains(trim, "/kpod/shell/aliases.") || strings.Contains(trim, "/shell/aliases.zsh") || strings.Contains(trim, "/shell/aliases.bash") {
				if trim != "" {
					lines = append(lines, trim)
				}
			}
		}
		if len(lines) > 0 {
			out = append(out, RCMatch{Path: path, Lines: lines})
		}
	}
	return out
}

func InstallPlan(paths config.Paths, home string, kind ShellKind) (rcPath, aliasFile, snippet string) {
	aliasFile = paths.AliasFile(string(kind))
	rcPath = RCPath(home, kind)
	snippet = ManagedBlock(aliasFile)
	return rcPath, aliasFile, snippet
}

func ApplyRC(rcPath, aliasFile string, install bool) error {
	var content []byte
	existing, err := os.ReadFile(rcPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil {
		content = existing
	}
	var next string
	var changed bool
	if install {
		next, changed = InsertManagedBlock(string(content), aliasFile)
	} else {
		next, changed = RemoveManagedBlock(string(content), aliasFile)
	}
	if !changed {
		return nil
	}
	return writeFile(rcPath, []byte(next))
}
