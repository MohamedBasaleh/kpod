package onboard

import (
	"errors"
	"strings"

	"github.com/MohamedBasaleh/kpod/internal/kube"
)

// UserMessage returns a concise, actionable explanation. It never surfaces
// raw "exit status N" text.
func UserMessage(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, kube.ErrKubectlNotFound):
		return "kubectl was not found.\n\nInstall kubectl and try again."
	case errors.Is(err, kube.ErrUnauthorized):
		return "Kubernetes authentication failed.\n\nRefresh your cluster credentials and try again."
	case errors.Is(err, kube.ErrForbidden):
		return "Your Kubernetes account cannot list namespaces.\n\nYou can still add a namespace you already have access to."
	case errors.Is(err, kube.ErrContextNotFound):
		return "That Kubernetes context was not found."
	case errors.Is(err, kube.ErrClusterUnavailable):
		return "The cluster is unreachable.\n\nCheck VPN and cluster access, then retry."
	case errors.Is(err, kube.ErrTimeout):
		return "The Kubernetes request timed out.\n\nCheck VPN and try again."
	case errors.Is(err, kube.ErrMalformedOutput):
		return "kubectl returned data kpod could not read."
	default:
		msg := err.Error()
		if strings.Contains(strings.ToLower(msg), "exit status") {
			return "The Kubernetes request failed.\n\nRetry, or check cluster access."
		}
		return msg
	}
}
