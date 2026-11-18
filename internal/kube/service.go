package kube

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

const defaultTimeout = 20 * time.Second

// Service is the Kubernetes discovery layer. Callers must not invoke kubectl.
type Service struct {
	kubectl string
	runner  Runner
	cache   *memoryCache
	ttl     CacheTTL
	timeout time.Duration
	now     func() time.Time
	onDebug func(DebugInfo)
}

// Option configures Service.
type Option func(*Service)

func WithRunner(r Runner) Option {
	return func(s *Service) { s.runner = r }
}

func WithKubectl(path string) Option {
	return func(s *Service) { s.kubectl = path }
}

func WithTimeout(d time.Duration) Option {
	return func(s *Service) { s.timeout = d }
}

func WithCacheTTL(ttl CacheTTL) Option {
	return func(s *Service) { s.ttl = ttl }
}

func WithNow(now func() time.Time) Option {
	return func(s *Service) { s.now = now }
}

func WithDebug(fn func(DebugInfo)) Option {
	return func(s *Service) { s.onDebug = fn }
}

// New constructs a Service. When no runner is supplied, kubectl must exist on PATH.
func New(opts ...Option) (*Service, error) {
	s := &Service{
		kubectl: "kubectl",
		cache:   newMemoryCache(),
		ttl:     DefaultCacheTTL(),
		timeout: defaultTimeout,
		now:     time.Now,
	}
	for _, opt := range opts {
		opt(s)
	}
	if s.runner == nil {
		path, err := exec.LookPath(s.kubectl)
		if err != nil {
			return nil, newError(ErrKubectlNotFound, "kubectl not found on PATH", Result{Name: s.kubectl}, err)
		}
		s.kubectl = path
		s.runner = ExecRunner{}
	}
	return s, nil
}

// Validate checks that kubectl can run a client-only version command.
func (s *Service) Validate(ctx context.Context) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()
	res, err := s.run(ctx, "version", "--client")
	if err != nil {
		return classifyRunError(err, res)
	}
	if res.ExitCode != 0 {
		return classifyOutput(res, nil)
	}
	return nil
}

func (s *Service) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, s.timeout)
}

func (s *Service) run(ctx context.Context, args ...string) (Result, error) {
	res, err := s.runner.Run(ctx, s.kubectl, args...)
	if s.onDebug != nil {
		s.onDebug(DebugInfo{
			Args:     append([]string{s.kubectl}, args...),
			Duration: res.Duration,
			ExitCode: res.ExitCode,
			Stderr:   firstLine(string(res.Stderr)),
		})
	}
	return res, err
}

// InvalidatePods drops cached pods for a context/namespace.
func (s *Service) InvalidatePods(kubeContext, namespace string) {
	if s == nil || s.cache == nil {
		return
	}
	s.cache.delete("pods\x00" + kubeContext + "\x00" + namespace)
}

func (s *Service) requireContext(kubeContext string) error {
	if strings.TrimSpace(kubeContext) == "" {
		return newError(ErrUnknown, "kubernetes context is required", Result{}, nil)
	}
	return nil
}

func (s *Service) requireNamespace(namespace string) error {
	if strings.TrimSpace(namespace) == "" {
		return newError(ErrUnknown, "namespace is required", Result{}, nil)
	}
	return nil
}
