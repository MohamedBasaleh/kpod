package kube

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type ownerList struct {
	Metadata struct {
		OwnerReferences []ownerRef `json:"ownerReferences"`
	} `json:"metadata"`
}

type ownerRef struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

// OwnerDeployment walks Pod → ReplicaSet → Deployment using JSON ownerReferences.
func (s *Service) OwnerDeployment(ctx context.Context, kubeContext, namespace, pod string) (string, error) {
	if err := s.requireContext(kubeContext); err != nil {
		return "", err
	}
	if err := s.requireNamespace(namespace); err != nil {
		return "", err
	}
	if strings.TrimSpace(pod) == "" {
		return "", fmt.Errorf("pod is required")
	}

	rs, err := s.ownerOfKind(ctx, kubeContext, namespace, "pod", pod, "ReplicaSet")
	if err != nil {
		return "", err
	}
	if rs == "" {
		return "", newError(ErrDeploymentNotFound, "this pod is not managed by a Deployment", Result{}, nil)
	}
	deploy, err := s.ownerOfKind(ctx, kubeContext, namespace, "rs", rs, "Deployment")
	if err != nil {
		return "", err
	}
	if deploy == "" {
		return "", newError(ErrDeploymentNotFound, "this pod is not managed by a Deployment", Result{}, nil)
	}
	return deploy, nil
}

func (s *Service) ownerOfKind(ctx context.Context, kubeContext, namespace, resource, name, kind string) (string, error) {
	res, err := s.RunArgs(ctx, "--context", kubeContext, "-n", namespace, "get", resource, name, "-o", "json")
	if err := classifyRunError(err, res); err != nil {
		return "", err
	}
	var parsed ownerList
	if err := json.Unmarshal(res.Stdout, &parsed); err != nil {
		return "", newError(ErrMalformedOutput, "malformed owner references JSON", res, err)
	}
	for _, ref := range parsed.Metadata.OwnerReferences {
		if ref.Kind == kind && strings.TrimSpace(ref.Name) != "" {
			return ref.Name, nil
		}
	}
	return "", nil
}
