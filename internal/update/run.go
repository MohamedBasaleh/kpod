package update

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/MohamedBasaleh/kpod/internal/version"
)

type Options struct {
	Stdout   io.Writer
	Stderr   io.Writer
	Client   Client
	ExecPath string
	Current  string
	Force    bool
}

type Result struct {
	Status  string
	Current string
	Latest  string
	Updated bool
}

func Run(opts Options) (Result, error) {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	current := opts.Current
	if current == "" {
		current = version.Display()
	}
	execPath := opts.ExecPath
	if execPath == "" {
		p, err := CurrentExecutable()
		if err != nil {
			return Result{Current: current}, fmt.Errorf("locate kpod binary: %w", err)
		}
		execPath = p
	}
	if InstalledByHomebrew(execPath) {
		msg := "kpod was installed through Homebrew.\n\nUpdate with:\n\n  brew upgrade kpod\n"
		fmt.Fprint(opts.Stdout, msg)
		return Result{Status: "homebrew", Current: current}, nil
	}

	rel, err := opts.Client.LatestStable()
	if err != nil {
		return Result{Current: current}, err
	}
	latest := strings.TrimPrefix(rel.Tag, "v")
	cmp, err := version.Compare(current, latest)
	if err != nil {
		return Result{Current: current, Latest: latest}, err
	}
	if cmp >= 0 && !opts.Force {
		fmt.Fprintf(opts.Stdout, "kpod %s is already the latest release.\n", current)
		return Result{Status: "latest", Current: current, Latest: latest}, nil
	}

	archive, err := download(opts.Client, rel.URL)
	if err != nil {
		return Result{Current: current, Latest: latest}, fmt.Errorf("download release: %w", err)
	}
	sums, err := download(opts.Client, rel.Checksum)
	if err != nil {
		return Result{Current: current, Latest: latest}, fmt.Errorf("download checksums: %w", err)
	}
	want, err := ParseChecksums(string(sums), rel.Archive)
	if err != nil {
		return Result{Current: current, Latest: latest}, err
	}
	if err := Verify(archive, want); err != nil {
		return Result{Current: current, Latest: latest}, err
	}
	bin, err := ExtractBinary(archive, "kpod")
	if err != nil {
		return Result{Current: current, Latest: latest}, err
	}
	if err := ReplaceExecutable(execPath, bin); err != nil {
		return Result{Current: current, Latest: latest}, fmt.Errorf("replace binary: %w", err)
	}
	fmt.Fprintf(opts.Stdout, "Updated kpod %s → %s\n", current, latest)
	return Result{Status: "updated", Current: current, Latest: latest, Updated: true}, nil
}

func download(c Client, url string) ([]byte, error) {
	if c.HTTP == nil {
		c.HTTP = http.DefaultClient
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", version.UserAgent())
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", res.StatusCode)
	}
	return io.ReadAll(io.LimitReader(res.Body, 64<<20))
}
