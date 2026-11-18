package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/MohamedBasaleh/kpod/internal/action"
	"github.com/MohamedBasaleh/kpod/internal/config"
	"github.com/MohamedBasaleh/kpod/internal/kube"
	"github.com/MohamedBasaleh/kpod/internal/resolve"
	"golang.org/x/term"
)

type Options struct {
	Paths     config.Paths
	Inv       Invocation
	Out       io.Writer
	In        io.Reader
	Err       io.Writer
	Debug     bool
	Kube      *kube.Service
	Confirmed bool
	SelectFn  func(label string, names []string) (string, error)
}

func Run(opts Options) error {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	if opts.In == nil {
		opts.In = os.Stdin
	}
	if opts.Err == nil {
		opts.Err = os.Stderr
	}
	cfg, err := config.NewRepository(opts.Paths).Load()
	if err != nil {
		return fmt.Errorf("could not load kpod configuration: %w", err)
	}
	env, err := resolve.Lookup(cfg, opts.Inv.Env)
	if err != nil {
		return err
	}
	svc := opts.Kube
	if svc == nil {
		var kubeOpts []kube.Option
		if opts.Debug {
			kubeOpts = append(kubeOpts, kube.WithDebug(func(info kube.DebugInfo) {
				fmt.Fprintf(opts.Err, "kubectl args=%q exit=%d duration=%s stderr=%q\n",
					info.Args, info.ExitCode, info.Duration, info.Stderr)
			}))
		}
		svc, err = kube.New(kubeOpts...)
		if err != nil {
			return err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	pods, err := svc.ListPods(ctx, env.KubeContext, env.Namespace)
	if err != nil {
		return err
	}
	matched := action.MatchPods(pods, opts.Inv.PodQuery, true)
	if opts.Inv.Spec.ID == action.Runner && opts.Inv.PodQuery != "" && len(matched) == 0 && len(opts.Inv.Extra) == 0 {
		opts.Inv.Extra = []string{opts.Inv.PodQuery}
		opts.Inv.PodQuery = ""
		matched = action.MatchPods(pods, "", true)
	}
	pod, err := pickPod(matched, opts)
	if err != nil {
		return err
	}
	container := ""
	if opts.Inv.Spec.NeedsContainer {
		container, err = pickContainer(pod, opts)
		if err != nil {
			return err
		}
	}
	if opts.Inv.Spec.NeedsRunning && !action.IsRunning(pod) {
		return fmt.Errorf("%s requires a running container (pod %s is %s)", opts.Inv.Spec.Label, pod.Name, pod.Status)
	}
	req := action.Request{
		Spec:           opts.Inv.Spec,
		Env:            env,
		Pod:            pod.Name,
		Container:      container,
		PreferredShell: opts.Inv.ShellPref,
	}
	if req.Spec.ID == action.Logs {
		req.Logs = action.LogsFollow
	}
	if err := fillExtras(&req, opts); err != nil {
		return err
	}
	if req.Spec.Destructive && !req.Confirmed {
		ok, err := confirmDestructive(req, env, opts)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(opts.Out, "Cancelled.")
			return nil
		}
		req.Confirmed = true
	}
	acts := action.New(svc)
	if action.Attached(req) {
		runCtx := context.Background()
		return acts.Attach(runCtx, req)
	}
	runCtx, cancelRun := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelRun()
	res, err := acts.Capture(runCtx, req)
	if err != nil {
		if res.Stderr != nil {
			fmt.Fprint(opts.Err, string(res.Stderr))
		}
		return err
	}
	out := string(res.Stdout)
	if len(res.Stderr) > 0 && out == "" {
		out = string(res.Stderr)
	}
	fmt.Fprint(opts.Out, out)
	if !strings.HasSuffix(out, "\n") && out != "" {
		fmt.Fprintln(opts.Out)
	}
	return nil
}

func pickPod(matched []kube.Pod, opts Options) (kube.Pod, error) {
	switch len(matched) {
	case 0:
		q := opts.Inv.PodQuery
		if q == "" {
			return kube.Pod{}, fmt.Errorf("no pods found")
		}
		return kube.Pod{}, fmt.Errorf("no pods matching %q", q)
	case 1:
		return matched[0], nil
	default:
		names := make([]string, len(matched))
		for i, p := range matched {
			names[i] = p.Name
		}
		if !isTTY() && opts.SelectFn == nil {
			return kube.Pod{}, fmt.Errorf("multiple pods match; refine the name:\n  %s", strings.Join(names, "\n  "))
		}
		choice, err := selectName("Select pod", names, opts)
		if err != nil {
			return kube.Pod{}, err
		}
		for _, p := range matched {
			if p.Name == choice {
				return p, nil
			}
		}
		return kube.Pod{}, fmt.Errorf("invalid pod selection")
	}
}

func pickContainer(pod kube.Pod, opts Options) (string, error) {
	name, err := action.PickContainer(pod, "")
	if err == nil {
		return name, nil
	}
	if err != action.ErrNeedContainer {
		return "", err
	}
	names := action.AppContainerNames(pod)
	if !isTTY() && opts.SelectFn == nil {
		return "", fmt.Errorf("container selection required: %s", strings.Join(names, ", "))
	}
	return selectName("Select container", names, opts)
}

func selectName(label string, names []string, opts Options) (string, error) {
	if opts.SelectFn != nil {
		return opts.SelectFn(label, names)
	}
	fmt.Fprintf(opts.Out, "%s\n", label)
	for i, n := range names {
		fmt.Fprintf(opts.Out, "  %d) %s\n", i+1, n)
	}
	fmt.Fprint(opts.Out, "Select: ")
	scanner := bufio.NewScanner(opts.In)
	if !scanner.Scan() {
		return "", fmt.Errorf("cancelled")
	}
	line := strings.TrimSpace(scanner.Text())
	idx, err := strconv.Atoi(line)
	if err != nil || idx < 1 || idx > len(names) {
		return "", fmt.Errorf("invalid selection")
	}
	return names[idx-1], nil
}

func fillExtras(req *action.Request, opts Options) error {
	extra := opts.Inv.Extra
	switch req.Spec.ID {
	case action.Runner:
		if len(extra) > 0 {
			req.RunnerCode = strings.Join(extra, " ")
		}
		if req.RunnerCode == "" {
			code, err := prompt(opts, "Runner expression: ")
			if err != nil {
				return err
			}
			req.RunnerCode = code
		}
	case action.PortForward:
		if len(extra) >= 2 {
			lp, err1 := strconv.Atoi(extra[0])
			rp, err2 := strconv.Atoi(extra[1])
			if err1 != nil || err2 != nil {
				return fmt.Errorf("ports must be numbers")
			}
			req.LocalPort, req.RemotePort = lp, rp
		} else {
			lp, err := prompt(opts, "Local port: ")
			if err != nil {
				return err
			}
			rp, err := prompt(opts, "Remote port: ")
			if err != nil {
				return err
			}
			var err1, err2 error
			req.LocalPort, err1 = strconv.Atoi(strings.TrimSpace(lp))
			req.RemotePort, err2 = strconv.Atoi(strings.TrimSpace(rp))
			if err1 != nil || err2 != nil || req.LocalPort <= 0 || req.RemotePort <= 0 {
				return fmt.Errorf("ports must be numbers")
			}
		}
	case action.Download, action.Upload:
		if len(extra) >= 2 {
			if req.Spec.ID == action.Download {
				req.RemotePath, req.LocalPath = extra[0], extra[1]
			} else {
				req.LocalPath, req.RemotePath = extra[0], extra[1]
			}
		} else {
			if req.Spec.ID == action.Download {
				r, err := prompt(opts, "Remote path (in pod): ")
				if err != nil {
					return err
				}
				l, err := prompt(opts, "Local destination: ")
				if err != nil {
					return err
				}
				req.RemotePath, req.LocalPath = r, l
			} else {
				l, err := prompt(opts, "Local source: ")
				if err != nil {
					return err
				}
				r, err := prompt(opts, "Remote path (in pod): ")
				if err != nil {
					return err
				}
				req.LocalPath, req.RemotePath = l, r
			}
		}
	}
	if opts.Confirmed {
		req.Confirmed = true
	}
	return nil
}

func prompt(opts Options, label string) (string, error) {
	if !isTTY() && opts.SelectFn == nil {
		return "", fmt.Errorf("%srequired", label)
	}
	fmt.Fprint(opts.Out, label)
	scanner := bufio.NewScanner(opts.In)
	if !scanner.Scan() {
		return "", fmt.Errorf("cancelled")
	}
	return strings.TrimSpace(scanner.Text()), nil
}

func confirmDestructive(req action.Request, env config.EnvironmentLookup, opts Options) (bool, error) {
	if opts.Confirmed {
		return true, nil
	}
	if req.Spec.ID == action.Delete {
		fmt.Fprintf(opts.Out, "Delete pod?\n\n%s\n\nEnvironment: %s\nContext: %s\nNamespace: %s\n\n[y/N] ",
			req.Pod, env.Alias, env.KubeContext, env.Namespace)
	} else {
		fmt.Fprintf(opts.Out, "Restart workload?\n\nEnvironment: %s\nNamespace: %s\nPod: %s\n\n[y/N] ",
			env.Alias, env.Namespace, req.Pod)
	}
	scanner := bufio.NewScanner(opts.In)
	if !scanner.Scan() {
		return false, nil
	}
	ans := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return ans == "y" || ans == "yes", nil
}

func isTTY() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}
