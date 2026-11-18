package update

import (
	"os"
	"path/filepath"
	"strings"
)

func InstalledByHomebrew(execPath string) bool {
	p := filepath.ToSlash(filepath.Clean(execPath))
	if strings.Contains(p, "/Cellar/kpod/") {
		return true
	}
	if strings.Contains(p, "/linuxbrew/") && strings.Contains(p, "/kpod/") {
		return true
	}
	if strings.Contains(p, "/Homebrew/") && strings.Contains(p, "/kpod") {
		return true
	}
	return false
}

func CurrentExecutable() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(p)
}
