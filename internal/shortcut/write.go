package shortcut

import (
	"os"

	"github.com/MohamedBasaleh/kpod/internal/atomicfile"
)

func writeFile(path string, data []byte) error {
	return atomicfile.Write(path, data, 0o600)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
