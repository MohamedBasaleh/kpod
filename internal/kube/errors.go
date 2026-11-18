package kube

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

var (
	ErrKubectlNotFound         = errors.New("kubectl not found")
	ErrUnauthorized            = errors.New("unauthorized")
	ErrForbidden               = errors.New("forbidden")
	ErrContextNotFound         = errors.New("kubernetes context not found")
	ErrClusterUnavailable      = errors.New("cluster unavailable")
	ErrTimeout                 = errors.New("kubernetes operation timed out")
	ErrMalformedOutput         = errors.New("malformed kubectl output")
	ErrUnknown                 = errors.New("kubernetes error")
	ErrMetricsUnavailable      = errors.New("metrics unavailable")
	ErrDeploymentNotFound      = errors.New("deployment not found")
	ErrContainerNotFound       = errors.New("container not found")
	ErrNoRunningContainer      = errors.New("no running container")
	ErrNeedsConfirm            = errors.New("confirmation required")
	ErrExecNotFound            = errors.New("executable not found in container")
	ErrPreviousLogsUnavailable = errors.New("previous logs are not available")
)

// Classify maps a kubectl result to a sentinel error. Callers should use errors.Is.
func Classify(err error, res Result) error {
	return classifyRunError(err, res)
}

// Error is a classified kubectl failure. Callers should use errors.Is.
type Error struct {
	Class   error
	Message string
	Stderr  string
	Args    []string
	Err     error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Class != nil {
		return e.Class.Error()
	}
	return ErrUnknown.Error()
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	if e.Class != nil {
		return e.Class
	}
	return e.Err
}

func newError(class error, message string, res Result, err error) *Error {
	stderr := strings.TrimSpace(string(res.Stderr))
	if message == "" {
		message = class.Error()
		if stderr != "" {
			message = fmt.Sprintf("%s: %s", class.Error(), firstLine(stderr))
		}
	}
	return &Error{
		Class:   class,
		Message: message,
		Stderr:  stderr,
		Args:    append([]string{res.Name}, res.Args...),
		Err:     err,
	}
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

func classifyRunError(err error, res Result) error {
	if err == nil && res.ExitCode == 0 {
		return nil
	}

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return newError(ErrTimeout, "kubernetes operation timed out", res, err)
		}
		if errors.Is(err, context.Canceled) {
			return newError(ErrTimeout, "kubernetes operation canceled", res, err)
		}
		if errors.Is(err, ErrKubectlNotFound) {
			return err
		}
		var pathErr *exec.Error
		if errors.As(err, &pathErr) {
			return newError(ErrKubectlNotFound, "kubectl not found on PATH", res, err)
		}
	}

	if res.ExitCode == 0 && err == nil {
		return nil
	}

	return classifyOutput(res, err)
}

func classifyOutput(res Result, err error) error {
	text := strings.ToLower(string(res.Stderr) + "\n" + string(res.Stdout))

	switch {
	case containsAll(text, "context", "does not exist"),
		containsAll(text, "context", "was not found"),
		strings.Contains(text, "no context exists"),
		strings.Contains(text, "context not found"):
		return newError(ErrContextNotFound, "", res, err)
	case strings.Contains(text, "unauthorized"),
		strings.Contains(text, "you must be logged in"):
		return newError(ErrUnauthorized, "", res, err)
	case strings.Contains(text, "forbidden"):
		return newError(ErrForbidden, "", res, err)
	case strings.Contains(text, "executable file not found"),
		strings.Contains(text, "command not found"):
		return newError(ErrExecNotFound, "", res, err)
	case containsAny(text,
		"connection refused",
		"no such host",
		"i/o timeout",
		"unable to connect to the server",
		"tls handshake timeout",
		"network is unreachable",
		"connection reset",
		"dial tcp"):
		return newError(ErrClusterUnavailable, "", res, err)
	default:
		msg := firstLine(string(res.Stderr))
		if msg == "" {
			msg = fmt.Sprintf("kubectl exited %d", res.ExitCode)
		}
		return newError(ErrUnknown, msg, res, err)
	}
}

func containsAll(text string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(text, p) {
			return false
		}
	}
	return true
}

func containsAny(text string, parts ...string) bool {
	for _, p := range parts {
		if strings.Contains(text, p) {
			return true
		}
	}
	return false
}
