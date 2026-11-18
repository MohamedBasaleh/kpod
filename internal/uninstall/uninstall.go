package uninstall

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/MohamedBasaleh/kpod/internal/config"
	"github.com/MohamedBasaleh/kpod/internal/onboard"
	"github.com/MohamedBasaleh/kpod/internal/shortcut"
	"github.com/MohamedBasaleh/kpod/internal/update"
)

const envInstallDir = "KPOD_INSTALL_DIR"

// Options control a conservative uninstall.
type Options struct {
	Home           string
	Paths          config.Paths
	Purge          bool
	Yes            bool
	ExplicitConfig bool
	ExecPath       string
	LookPath func(string) (string, error)
	In       io.Reader
	Out      io.Writer
	Err      io.Writer
}

// Plan is the preview of an uninstall. It never includes kubeconfig or AWS paths.
type Plan struct {
	Remove      []string
	Skip        []string
	Leftover    []string
	RC          []shortcut.RCMatch
	ConfigDir   string
	Purge       bool
	Homebrew    string
	SelfRemove  string
	ManualRM    string
	Protected   []string
}

// Result is what happened after Execute.
type Result struct {
	Removed []string
	Notes   []string
	Fresh   bool
}

func (o Options) home() string {
	if o.Home != "" {
		return o.Home
	}
	h, _ := os.UserHomeDir()
	return h
}

func (o Options) stdout() io.Writer {
	if o.Out != nil {
		return o.Out
	}
	return os.Stdout
}

func (o Options) stderr() io.Writer {
	if o.Err != nil {
		return o.Err
	}
	return os.Stderr
}

func (o Options) stdin() io.Reader {
	if o.In != nil {
		return o.In
	}
	return os.Stdin
}

func knownBinaryPaths(home string) []string {
	var out []string
	if d := strings.TrimSpace(os.Getenv(envInstallDir)); d != "" {
		out = append(out, filepath.Join(d, "kpod"))
	}
	if home != "" {
		out = append(out, filepath.Join(home, ".local", "bin", "kpod"))
		out = append(out, filepath.Join(home, "bin", "kpod"))
	}
	return uniquePaths(out)
}

func uniquePaths(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range in {
		p = filepath.Clean(p)
		if p == "" || p == "." || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

func sameFile(a, b string) bool {
	fa, errA := os.Stat(a)
	fb, errB := os.Stat(b)
	if errA != nil || errB != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return os.SameFile(fa, fb)
}

func isKpodBinary(path string) bool {
	if filepath.Base(path) != "kpod" {
		return false
	}
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return false
	}
	return true
}

func protectedPaths(home string) []string {
	if home == "" {
		return []string{"~/.kube/config", "Kubernetes contexts", "AWS credentials"}
	}
	return []string{
		filepath.Join(home, ".kube", "config"),
		filepath.Join(home, ".aws"),
		"Kubernetes contexts",
		"AWS credentials",
	}
}

func canPurgeDir(dir string, home string, explicit bool) error {
	if dir == "" {
		return fmt.Errorf("kpod config directory is empty; refusing to purge")
	}
	clean := filepath.Clean(dir)
	if clean == string(filepath.Separator) || clean == filepath.VolumeName(clean)+string(filepath.Separator) {
		return fmt.Errorf("refusing to purge %s", clean)
	}
	if home != "" && clean == filepath.Clean(home) {
		return fmt.Errorf("refusing to purge the home directory")
	}
	slash := filepath.ToSlash(clean)
	if strings.Contains(slash, "/.kube") || strings.HasSuffix(slash, "/.aws") || strings.Contains(slash, "/.aws/") {
		return fmt.Errorf("refusing to purge %s", clean)
	}
	base := filepath.Base(clean)
	if base != "kpod" && os.Getenv("KPOD_CONFIG_DIR") == "" && !explicit {
		return fmt.Errorf("refusing to purge %s (not a kpod config directory)", clean)
	}
	return nil
}

// Preview builds the uninstall plan without deleting anything.
func Preview(opts Options) (Plan, error) {
	home := opts.home()
	p := Plan{
		Protected: protectedPaths(home),
		Purge:     opts.Purge,
		ConfigDir: opts.Paths.Dir,
	}
	execPath := opts.ExecPath
	if execPath == "" {
		if got, err := update.CurrentExecutable(); err == nil {
			execPath = got
		}
	}
	if execPath != "" && update.InstalledByHomebrew(execPath) {
		p.Homebrew = execPath
		p.Skip = append(p.Skip, execPath+" (Homebrew; not removed)")
	}

	for _, cand := range knownBinaryPaths(home) {
		if !isKpodBinary(cand) {
			continue
		}
		if update.InstalledByHomebrew(cand) {
			p.Skip = append(p.Skip, cand+" (Homebrew; not removed)")
			continue
		}
		p.Remove = append(p.Remove, cand)
		if execPath != "" && sameFile(execPath, cand) {
			p.SelfRemove = cand
		}
	}

	look := opts.LookPath
	if look == nil {
		look = exec.LookPath
	}
	if found, err := look("kpod"); err == nil && found != "" {
		matched := false
		for _, r := range p.Remove {
			if sameFile(found, r) {
				matched = true
				break
			}
		}
		if !matched && p.Homebrew != "" && sameFile(found, p.Homebrew) {
			matched = true
		}
		if !matched {
			p.Leftover = append(p.Leftover, found)
		}
	}

	if opts.Purge {
		if err := canPurgeDir(opts.Paths.Dir, home, opts.ExplicitConfig); err != nil {
			return p, err
		}
		if _, err := os.Stat(opts.Paths.Dir); err == nil {
			p.Remove = append(p.Remove, opts.Paths.Dir)
		}
	}

	p.RC = shortcut.FindManagedLines(home)
	p.Remove = uniquePaths(p.Remove)
	return p, nil
}

func writePlan(w io.Writer, p Plan) {
	fmt.Fprintln(w, "kpod will remove:")
	fmt.Fprintln(w)
	if len(p.Remove) == 0 {
		fmt.Fprintln(w, "  (nothing in known kpod install/config locations)")
	}
	for _, r := range p.Remove {
		fmt.Fprintf(w, "  %s\n", r)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "kpod will NOT modify:")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  ~/.kube/config")
	fmt.Fprintln(w, "  Kubernetes contexts")
	fmt.Fprintln(w, "  AWS credentials")
	if !p.Purge {
		fmt.Fprintln(w, "  kpod configuration (~/.config/kpod)")
	}
	fmt.Fprintln(w)
	for _, s := range p.Skip {
		fmt.Fprintf(w, "Skipped: %s\n", s)
	}
	for _, s := range p.Leftover {
		fmt.Fprintf(w, "Also found, not removed: %s\n", s)
		fmt.Fprintln(w, "  (not a known kpod install path)")
	}
	if len(p.RC) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Shell rc files still source kpod shortcuts. kpod will not edit them.")
		for _, rc := range p.RC {
			fmt.Fprintf(w, "  %s\n", rc.Path)
			for _, line := range rc.Lines {
				fmt.Fprintf(w, "    %s\n", line)
			}
		}
		fmt.Fprintln(w, "Remove the managed line with: kpod shortcut uninstall")
		fmt.Fprintln(w, "Or delete that line by hand, then open a new terminal.")
	}
}

func confirm(opts Options) bool {
	fmt.Fprint(opts.stdout(), "Continue? [y/N] ")
	sc := bufio.NewScanner(opts.stdin())
	if !sc.Scan() {
		return false
	}
	ans := strings.ToLower(strings.TrimSpace(sc.Text()))
	return ans == "y" || ans == "yes"
}

// Execute performs the plan after confirmation.
func Execute(opts Options) (Result, error) {
	plan, err := Preview(opts)
	if err != nil {
		return Result{}, err
	}
	writePlan(opts.stdout(), plan)
	if !opts.Yes {
		if !confirm(opts) {
			fmt.Fprintln(opts.stdout(), "Cancelled.")
			return Result{}, nil
		}
	}

	var res Result
	for _, path := range plan.Remove {
		if path == opts.Paths.Dir && opts.Purge {
			if err := os.RemoveAll(path); err != nil {
				return res, fmt.Errorf("remove %s: %w", path, err)
			}
			res.Removed = append(res.Removed, path)
			continue
		}
		if err := os.Remove(path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			res.Notes = append(res.Notes, fmt.Sprintf("could not remove %s: %s", path, err.Error()))
			if plan.SelfRemove == path {
				res.Notes = append(res.Notes, "Remove it after this process exits:\n  rm -f "+path)
			}
			continue
		}
		res.Removed = append(res.Removed, path)
	}

	if opts.Purge {
		_, statErr := os.Stat(opts.Paths.ConfigFile())
		res.Fresh = os.IsNotExist(statErr) && onboard.Detect(opts.Paths) == onboard.ModeFresh
	}

	if len(res.Removed) == 0 && len(res.Notes) == 0 {
		fmt.Fprintln(opts.stdout(), "Nothing to remove.")
	} else {
		fmt.Fprintln(opts.stdout(), "Removed:")
		for _, r := range res.Removed {
			fmt.Fprintf(opts.stdout(), "  %s\n", r)
		}
	}
	for _, n := range res.Notes {
		fmt.Fprintln(opts.stderr(), n)
	}
	if opts.Purge && res.Fresh {
		fmt.Fprintln(opts.stdout(), "kpod config is gone. After reinstall, `kpod` will start first-run setup.")
	}
	return res, nil
}
