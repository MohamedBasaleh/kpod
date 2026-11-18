package tui

import (
	"errors"
	"fmt"

	"github.com/MohamedBasaleh/kpod/internal/kube"
	"github.com/MohamedBasaleh/kpod/internal/onboard"
)

func kubeContextError(err error, kubeContext string, debug bool) string {
	if err == nil {
		return ""
	}
	var msg string
	switch {
	case errors.Is(err, kube.ErrUnauthorized):
		msg = fmt.Sprintf("Authentication failed for %s.\n\nYour Kubernetes credentials may have expired.", kubeContext)
	case errors.Is(err, kube.ErrClusterUnavailable):
		msg = fmt.Sprintf("Unable to reach %s.\n\nCheck your network or VPN connection.", kubeContext)
	case errors.Is(err, kube.ErrTimeout):
		msg = fmt.Sprintf("%s did not respond in time.\n\n[r] Retry  [esc] Back", kubeContext)
	case errors.Is(err, kube.ErrForbidden):
		msg = fmt.Sprintf("Your Kubernetes account cannot list namespaces in %s.", kubeContext)
	case errors.Is(err, kube.ErrKubectlNotFound):
		msg = onboard.UserMessage(err)
	default:
		msg = onboard.UserMessage(err)
	}
	if debug {
		var ke *kube.Error
		if errors.As(err, &ke) && ke.Stderr != "" {
			msg += "\n\n" + firstLine(ke.Stderr)
		}
	}
	return msg
}

func kubePodError(err error, kubeContext, namespace string, debug bool) string {
	if err == nil {
		return ""
	}
	var msg string
	switch {
	case errors.Is(err, kube.ErrUnauthorized):
		msg = "Authentication failed.\n\nYour Kubernetes credentials may have expired."
	case errors.Is(err, kube.ErrForbidden):
		msg = fmt.Sprintf("You do not have permission to list pods in %s.", namespace)
	case errors.Is(err, kube.ErrClusterUnavailable):
		msg = fmt.Sprintf("Unable to reach %s.\n\nCheck your network or VPN.", kubeContext)
	case errors.Is(err, kube.ErrTimeout):
		msg = "The cluster did not respond in time.\n\n[r] Retry  [esc] Back"
	case errors.Is(err, kube.ErrMalformedOutput):
		msg = "kubectl returned pod data kpod could not read."
	default:
		msg = kubeContextError(err, kubeContext, debug)
		return msg
	}
	if debug {
		var ke *kube.Error
		if errors.As(err, &ke) && ke.Stderr != "" {
			msg += "\n\n" + firstLine(ke.Stderr)
		}
	}
	return msg
}

func firstLine(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			return s[:i]
		}
	}
	return s
}
