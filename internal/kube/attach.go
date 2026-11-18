package kube

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
)

// Attacher runs a command with the caller's terminal attached.
type Attacher interface {
	RunAttached(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, name string, args ...string) error
}

func (ExecRunner) RunAttached(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if err == nil {
		return nil
	}
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return ctx.Err()
	}
	var pathErr *exec.Error
	if errors.As(err, &pathErr) {
		return ErrKubectlNotFound
	}
	return err
}

// KubectlBin returns the kubectl executable path used by the service.
func (s *Service) KubectlBin() string {
	return s.kubectl
}

// RunArgs runs kubectl with the given args and the service timeout unless ctx already has a deadline.
func (s *Service) RunArgs(ctx context.Context, args ...string) (Result, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()
	return s.run(ctx, args...)
}

// RunArgsRaw runs kubectl without applying the default discovery timeout.
func (s *Service) RunArgsRaw(ctx context.Context, args ...string) (Result, error) {
	return s.run(ctx, args...)
}

// Attach runs kubectl attached to the process stdio.
func (s *Service) Attach(ctx context.Context, args ...string) error {
	att, ok := s.runner.(Attacher)
	if !ok {
		return errors.New("kubectl runner does not support attached processes")
	}
	if s.onDebug != nil {
		s.onDebug(DebugInfo{Args: append([]string{s.kubectl}, args...)})
	}
	return att.RunAttached(ctx, os.Stdin, os.Stdout, os.Stderr, s.kubectl, args...)
}

// Command returns an exec.Cmd for kubectl args (for Bubble Tea tea.ExecProcess).
func (s *Service) Command(args ...string) *exec.Cmd {
	return exec.Command(s.kubectl, args...)
}
