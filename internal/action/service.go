package action

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MohamedBasaleh/kpod/internal/kube"
)

type Service struct {
	kube *kube.Service
}

func New(k *kube.Service) *Service {
	return &Service{kube: k}
}

func (s *Service) Kube() *kube.Service { return s.kube }

func (s *Service) probeShell(ctx context.Context, req Request, shell string) error {
	args := probeShellArgs(req, shell)
	res, err := s.kube.RunArgs(ctx, args...)
	classified := kube.Classify(err, res)
	if classified == nil {
		return nil
	}
	if errors.Is(classified, kube.ErrExecNotFound) {
		return kube.ErrExecNotFound
	}
	return classified
}

func (s *Service) ResolveShell(ctx context.Context, req Request) (string, error) {
	shells := []string{"bash", "sh", "ash"}
	if req.PreferredShell == "sh" {
		shells = []string{"sh", "ash", "bash"}
	}
	for _, sh := range shells {
		err := s.probeShell(ctx, req, sh)
		if err == nil {
			return sh, nil
		}
		if errors.Is(err, kube.ErrExecNotFound) {
			continue
		}
		return "", err
	}
	return "", fmt.Errorf("no shell found (tried: bash, sh, ash)")
}

func (s *Service) Capture(ctx context.Context, req Request) (kube.Result, error) {
	switch req.Spec.ID {
	case Restart, Rollout:
		return s.captureWorkload(ctx, req)
	case Metrics:
		return s.captureMetrics(ctx, req)
	case Logs:
		if req.Logs == LogsPrevious {
			return s.capturePreviousLogs(ctx, req)
		}
	}
	args, err := PrepareArgs(req)
	if err != nil {
		return kube.Result{}, err
	}
	res, runErr := s.kube.RunArgsRaw(ctx, args...)
	if err := kube.Classify(runErr, res); err != nil {
		return res, err
	}
	return res, nil
}

func (s *Service) capturePreviousLogs(ctx context.Context, req Request) (kube.Result, error) {
	args, err := PrepareArgs(req)
	if err != nil {
		return kube.Result{}, err
	}
	res, runErr := s.kube.RunArgsRaw(ctx, args...)
	if err := kube.Classify(runErr, res); err != nil {
		text := strings.ToLower(string(res.Stderr))
		if strings.Contains(text, "previous terminated container") ||
			strings.Contains(text, "previous terminated") {
			return res, kube.ErrPreviousLogsUnavailable
		}
		return res, err
	}
	return res, nil
}

func (s *Service) captureMetrics(ctx context.Context, req Request) (kube.Result, error) {
	args, err := PrepareArgs(req)
	if err != nil {
		return kube.Result{}, err
	}
	res, runErr := s.kube.RunArgs(ctx, args...)
	if err := kube.Classify(runErr, res); err != nil {
		text := strings.ToLower(string(res.Stderr) + string(res.Stdout))
		if strings.Contains(text, "metrics-server") ||
			strings.Contains(text, "metrics api") ||
			strings.Contains(text, "metrics not available") {
			return res, kube.ErrMetricsUnavailable
		}
		return res, err
	}
	return res, nil
}

func (s *Service) captureWorkload(ctx context.Context, req Request) (kube.Result, error) {
	deploy, err := s.kube.OwnerDeployment(ctx, req.Env.KubeContext, req.Env.Namespace, req.Pod)
	if err != nil {
		return kube.Result{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	if req.Spec.ID == Restart {
		args, err := PrepareWorkloadArgs(req, deploy, true)
		if err != nil {
			return kube.Result{}, err
		}
		res, runErr := s.kube.RunArgsRaw(ctx, args...)
		if err := kube.Classify(runErr, res); err != nil {
			return res, err
		}
		statusReq := req
		statusReq.Spec.ID = Rollout
		statusArgs, err := PrepareWorkloadArgs(statusReq, deploy, false)
		if err != nil {
			return res, err
		}
		st, stErr := s.kube.RunArgsRaw(ctx, statusArgs...)
		if err := kube.Classify(stErr, st); err != nil {
			return st, err
		}
		return st, nil
	}
	args, err := PrepareWorkloadArgs(req, deploy, false)
	if err != nil {
		return kube.Result{}, err
	}
	res, runErr := s.kube.RunArgsRaw(ctx, args...)
	if err := kube.Classify(runErr, res); err != nil {
		return res, err
	}
	return res, nil
}

func (s *Service) Attach(ctx context.Context, req Request) error {
	args, err := s.KubectlArgs(ctx, req)
	if err != nil {
		return err
	}
	return s.kube.Attach(ctx, args...)
}

func (s *Service) KubectlArgs(ctx context.Context, req Request) ([]string, error) {
	if req.Spec.ID == Shell {
		sh, err := s.ResolveShell(ctx, req)
		if err != nil {
			return nil, err
		}
		return PrepareArgs(req, sh)
	}
	if req.Spec.ID == Restart || req.Spec.ID == Rollout {
		deploy, err := s.kube.OwnerDeployment(ctx, req.Env.KubeContext, req.Env.Namespace, req.Pod)
		if err != nil {
			return nil, err
		}
		return PrepareWorkloadArgs(req, deploy, req.Spec.ID == Restart)
	}
	return PrepareArgs(req)
}

func (s *Service) OwnerDeployment(ctx context.Context, req Request) (string, error) {
	return s.kube.OwnerDeployment(ctx, req.Env.KubeContext, req.Env.Namespace, req.Pod)
}
