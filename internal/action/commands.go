package action

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/MohamedBasaleh/kpod/internal/kube"
)

func baseArgs(req Request) []string {
	return []string{"--context", req.Env.KubeContext, "-n", req.Env.Namespace}
}

func withContainer(args []string, container string) []string {
	if strings.TrimSpace(container) == "" {
		return args
	}
	return append(args, "-c", container)
}

func usesUseContext(args []string) bool {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "config" && args[i+1] == "use-context" {
			return true
		}
	}
	return false
}

func requireTarget(req Request) error {
	if strings.TrimSpace(req.Env.KubeContext) == "" || strings.TrimSpace(req.Env.Namespace) == "" {
		return fmt.Errorf("environment context and namespace are required")
	}
	if strings.TrimSpace(req.Pod) == "" {
		return fmt.Errorf("pod is required")
	}
	return nil
}

func requireConfirm(req Request) error {
	if req.Spec.Destructive && !req.Confirmed {
		return kube.ErrNeedsConfirm
	}
	return nil
}

func requireRunning(req Request, running bool) error {
	if req.Spec.NeedsRunning && !running {
		return kube.ErrNoRunningContainer
	}
	return nil
}

// PrepareArgs returns kubectl arguments for req (excluding the binary name).
func PrepareArgs(req Request, extra ...string) ([]string, error) {
	if err := requireTarget(req); err != nil {
		return nil, err
	}
	if err := requireConfirm(req); err != nil {
		return nil, err
	}
	args, err := buildArgs(req, extra...)
	if err != nil {
		return nil, err
	}
	if usesUseContext(args) {
		return nil, fmt.Errorf("refusing to mutate kubeconfig current-context")
	}
	return args, nil
}

func buildArgs(req Request, extra ...string) ([]string, error) {
	b := baseArgs(req)
	switch req.Spec.ID {
	case Shell:
		shell := "bash"
		if len(extra) > 0 && extra[0] != "" {
			shell = extra[0]
		}
		args := append(b, "exec", "-it", req.Pod)
		args = withContainer(args, req.Container)
		return append(args, "--", shell), nil
	case Console:
		args := append(b, "exec", "-it", req.Pod)
		args = withContainer(args, req.Container)
		return append(args, "--", "bundle", "exec", "rails", "console"), nil
	case DBConsole:
		args := append(b, "exec", "-it", req.Pod)
		args = withContainer(args, req.Container)
		return append(args, "--", "bundle", "exec", "rails", "dbconsole"), nil
	case Runner:
		if strings.TrimSpace(req.RunnerCode) == "" {
			return nil, fmt.Errorf("runner expression is required")
		}
		args := append(b, "exec", "-it", req.Pod)
		args = withContainer(args, req.Container)
		return append(args, "--", "bundle", "exec", "rails", "runner", req.RunnerCode), nil
	case Logs:
		args := append(b, "logs", req.Pod)
		args = withContainer(args, req.Container)
		switch req.Logs {
		case LogsFollow:
			args = append(args, "-f")
		case LogsPrevious:
			args = append(args, "--previous")
		case LogsTail500:
			args = append(args, "--tail=500")
		case LogsTail100:
			n := 100
			if req.Tail > 0 {
				n = req.Tail
			}
			args = append(args, "--tail="+strconv.Itoa(n))
		default:
			if req.Tail > 0 {
				args = append(args, "--tail="+strconv.Itoa(req.Tail))
			} else {
				args = append(args, "-f")
			}
		}
		return args, nil
	case Describe:
		return append(b, "describe", "pod", req.Pod), nil
	case YAML:
		return append(b, "get", "pod", req.Pod, "-o", "yaml"), nil
	case Events:
		return append(b, "get", "events",
			"--field-selector", "involvedObject.name="+req.Pod,
			"--sort-by=.lastTimestamp"), nil
	case Metrics:
		return append(b, "top", "pod", req.Pod), nil
	case Delete:
		return append(b, "delete", "pod", req.Pod), nil
	case PortForward:
		if req.LocalPort <= 0 || req.RemotePort <= 0 {
			return nil, fmt.Errorf("local and remote ports are required")
		}
		return append(b, "port-forward", "pod/"+req.Pod, fmt.Sprintf("%d:%d", req.LocalPort, req.RemotePort)), nil
	case Download:
		if err := validateCopyPaths(req, false); err != nil {
			return nil, err
		}
		src := req.Env.Namespace + "/" + req.Pod + ":" + req.RemotePath
		args := append(b, "cp", src, req.LocalPath)
		return withContainer(args, req.Container), nil
	case Upload:
		if err := validateCopyPaths(req, true); err != nil {
			return nil, err
		}
		dst := req.Env.Namespace + "/" + req.Pod + ":" + req.RemotePath
		args := append(b, "cp", req.LocalPath, dst)
		return withContainer(args, req.Container), nil
	case Restart, Rollout:
		return nil, fmt.Errorf("use PrepareWorkloadArgs")
	default:
		return nil, fmt.Errorf("unsupported action %s", req.Spec.ID)
	}
}

func probeShellArgs(req Request, shell string) []string {
	args := append(baseArgs(req), "exec", req.Pod)
	args = withContainer(args, req.Container)
	return append(args, "--", shell, "-c", "exit 0")
}

func validateCopyPaths(req Request, upload bool) error {
	if strings.TrimSpace(req.RemotePath) == "" || strings.TrimSpace(req.LocalPath) == "" {
		return fmt.Errorf("remote and local paths are required")
	}
	if strings.ContainsRune(req.RemotePath, 0) || strings.ContainsRune(req.LocalPath, 0) {
		return fmt.Errorf("invalid path")
	}
	if upload {
		info, err := os.Stat(req.LocalPath)
		if err != nil {
			return fmt.Errorf("local source %q is not readable", req.LocalPath)
		}
		if info.IsDir() {
			return fmt.Errorf("local source must be a file")
		}
	}
	return nil
}

// LooksLikePath reports whether s is likely a filesystem path (CLI copy args).
func LooksLikePath(s string) bool {
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "/") || strings.HasPrefix(s, "./") || strings.HasPrefix(s, "../") || strings.HasPrefix(s, "~/") {
		return true
	}
	return strings.Contains(s, "/")
}

func PrepareWorkloadArgs(req Request, deploy string, restart bool) ([]string, error) {
	if err := requireTarget(req); err != nil {
		return nil, err
	}
	if restart {
		if err := requireConfirm(req); err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(deploy) == "" {
		return nil, kube.ErrDeploymentNotFound
	}
	args := append(baseArgs(req), "rollout")
	if restart {
		args = append(args, "restart", "deployment/"+deploy)
	} else {
		args = append(args, "status", "deployment/"+deploy, "--timeout=300s")
	}
	if usesUseContext(args) {
		return nil, fmt.Errorf("refusing to mutate kubeconfig current-context")
	}
	return args, nil
}
