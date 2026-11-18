package kube

import (
	"context"
	"strings"
)

const cacheKeyContexts = "contexts"

// ListContexts returns kubeconfig context names from:
//
//	kubectl config get-contexts -o name
//
// It never changes the current-context.
func (s *Service) ListContexts(ctx context.Context, opts ...ListOption) ([]Context, error) {
	o := applyListOptions(opts)
	if !o.refresh {
		if hit, ok := s.cache.get(s.now(), cacheKeyContexts); ok {
			return cloneContexts(hit.([]Context)), nil
		}
	}

	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	res, err := s.run(ctx, "config", "get-contexts", "-o", "name")
	if err := classifyRunError(err, res); err != nil {
		return nil, err
	}

	list := parseContextNames(string(res.Stdout))
	s.cache.set(s.now(), cacheKeyContexts, cloneContexts(list), s.ttl.Contexts)
	return list, nil
}

func parseContextNames(stdout string) []Context {
	lines := strings.Split(stdout, "\n")
	out := make([]Context, 0, len(lines))
	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		out = append(out, Context{Name: name})
	}
	if out == nil {
		return []Context{}
	}
	return out
}

func cloneContexts(in []Context) []Context {
	out := make([]Context, len(in))
	copy(out, in)
	return out
}
