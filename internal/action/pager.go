package action

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
)

const pagerBytes = 8000
const pagerLines = 80

func PreferPager(output []byte) bool {
	if len(output) > pagerBytes {
		return true
	}
	return bytes.Count(output, []byte("\n")) >= pagerLines
}

// PagerCommand returns a pager command that reads content from stdin, or nil.
func PagerCommand(content string) *exec.Cmd {
	spec := strings.TrimSpace(os.Getenv("PAGER"))
	if spec == "" {
		if _, err := exec.LookPath("less"); err != nil {
			return nil
		}
		spec = "less -R"
	}
	fields := strings.Fields(spec)
	if len(fields) == 0 {
		return nil
	}
	if _, err := exec.LookPath(fields[0]); err != nil {
		return nil
	}
	cmd := exec.Command(fields[0], fields[1:]...)
	cmd.Stdin = strings.NewReader(content)
	return cmd
}
