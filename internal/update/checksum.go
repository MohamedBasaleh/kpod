package update

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

func ParseChecksums(text, filename string) (string, error) {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		sum, name := fields[0], fields[len(fields)-1]
		name = strings.TrimPrefix(name, "*")
		if name == filename || strings.HasSuffix(name, "/"+filename) {
			if len(sum) != 64 {
				return "", fmt.Errorf("invalid checksum for %s", filename)
			}
			return strings.ToLower(sum), nil
		}
	}
	return "", fmt.Errorf("checksums.txt does not list %s", filename)
}

func SHA256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func Verify(data []byte, wantHex string) error {
	got := SHA256Hex(data)
	if !strings.EqualFold(got, strings.TrimSpace(wantHex)) {
		return fmt.Errorf("checksum mismatch for download (got %s)", got)
	}
	return nil
}
