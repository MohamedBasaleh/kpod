package kube

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type namespaceListJSON struct {
	Items []namespaceJSON `json:"items"`
}

type namespaceJSON struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
}

// ListNamespaces returns namespaces for an explicit Kubernetes context.
//
//	kubectl --context <context> get namespaces -o json
//
// Forbidden is a classified error, not an empty list.
func (s *Service) ListNamespaces(ctx context.Context, kubeContext string, opts ...ListOption) ([]Namespace, error) {
	if err := s.requireContext(kubeContext); err != nil {
		return nil, err
	}
	key := "namespaces\x00" + kubeContext
	o := applyListOptions(opts)
	if !o.refresh {
		if hit, ok := s.cache.get(s.now(), key); ok {
			return cloneNamespaces(hit.([]Namespace)), nil
		}
	}

	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	res, err := s.run(ctx, "--context", kubeContext, "get", "namespaces", "-o", "json")
	if err := classifyRunError(err, res); err != nil {
		return nil, err
	}

	list, err := parseNamespaces(res.Stdout)
	if err != nil {
		return nil, newError(ErrMalformedOutput, "malformed kubectl namespace JSON", res, err)
	}
	s.cache.set(s.now(), key, cloneNamespaces(list), s.ttl.Namespaces)
	return list, nil
}

func parseNamespaces(raw []byte) ([]Namespace, error) {
	var parsed namespaceListJSON
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	out := make([]Namespace, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		name := strings.TrimSpace(item.Metadata.Name)
		if name == "" {
			continue
		}
		out = append(out, Namespace{Name: name})
	}
	return out, nil
}

func cloneNamespaces(in []Namespace) []Namespace {
	out := make([]Namespace, len(in))
	copy(out, in)
	return out
}

// CanAccessNamespace probes whether the user can get pods in a namespace.
//
//	kubectl --context <context> -n <namespace> auth can-i get pods
//
// "yes" and "no" are results, not command failures.
func (s *Service) CanAccessNamespace(ctx context.Context, kubeContext, namespace string) (NamespaceAccess, error) {
	if err := s.requireContext(kubeContext); err != nil {
		return NamespaceAccess{}, err
	}
	if err := s.requireNamespace(namespace); err != nil {
		return NamespaceAccess{}, err
	}

	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	res, err := s.run(ctx, "--context", kubeContext, "-n", namespace, "auth", "can-i", "get", "pods")
	if err != nil {
		return NamespaceAccess{}, classifyRunError(err, res)
	}

	answer := strings.ToLower(strings.TrimSpace(string(res.Stdout)))
	switch answer {
	case "yes":
		return NamespaceAccess{Allowed: true}, nil
	case "no":
		return NamespaceAccess{Allowed: false}, nil
	}

	if classified := classifyRunError(fmt.Errorf("kubectl exited %d", res.ExitCode), res); classified != nil && res.ExitCode != 0 {
		return NamespaceAccess{}, classified
	}
	if res.ExitCode != 0 {
		return NamespaceAccess{}, classifyOutput(res, nil)
	}
	return NamespaceAccess{}, newError(ErrMalformedOutput, "malformed auth can-i output", res, nil)
}
