package kube

import (
	"context"
	"errors"
	"testing"
	"time"
)

const nsJSON = `{"items":[{"metadata":{"name":"dev-core"}},{"metadata":{"name":"staging-core"}}]}`

func TestListNamespacesSuccess(t *testing.T) {
	r := &fakeRunner{stdout: nsJSON}
	svc := newTestService(t, r)
	got, err := svc.ListNamespaces(context.Background(), "Nawaqis-Dev")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "dev-core" || got[1].Name != "staging-core" {
		t.Fatalf("got %+v", got)
	}
	requireArgPair(t, r.lastArgs(), "--context", "Nawaqis-Dev")
	assertNoUseContext(t, r)
}

func TestListNamespacesZero(t *testing.T) {
	r := &fakeRunner{stdout: `{"items":[]}`}
	svc := newTestService(t, r)
	got, err := svc.ListNamespaces(context.Background(), "Nawaqis-Dev")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("got %#v", got)
	}
}

func TestListNamespacesForbidden(t *testing.T) {
	r := &fakeRunner{
		exit:   1,
		stderr: `Error from server (Forbidden): namespaces is forbidden: User "x" cannot list resource "namespaces" in API group "" at the cluster scope`,
	}
	svc := newTestService(t, r)
	got, err := svc.ListNamespaces(context.Background(), "Nawaqis-Dev")
	if got != nil {
		t.Fatalf("forbidden must not become a list: %+v", got)
	}
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("got %v", err)
	}
}

func TestListNamespacesUnauthorized(t *testing.T) {
	r := &fakeRunner{exit: 1, stderr: `error: You must be logged in to the server (Unauthorized)`}
	svc := newTestService(t, r)
	_, err := svc.ListNamespaces(context.Background(), "Nawaqis-Dev")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
}

func TestListNamespacesContextNotFound(t *testing.T) {
	r := &fakeRunner{exit: 1, stderr: `error: context "missing" does not exist`}
	svc := newTestService(t, r)
	_, err := svc.ListNamespaces(context.Background(), "missing")
	if !errors.Is(err, ErrContextNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestListNamespacesUnreachable(t *testing.T) {
	r := &fakeRunner{exit: 1, stderr: `Unable to connect to the server: dial tcp 10.0.0.1:443: connect: connection refused`}
	svc := newTestService(t, r)
	_, err := svc.ListNamespaces(context.Background(), "Nawaqis-Dev")
	if !errors.Is(err, ErrClusterUnavailable) {
		t.Fatalf("got %v", err)
	}
}

func TestListNamespacesMalformed(t *testing.T) {
	r := &fakeRunner{stdout: `{not json`}
	svc := newTestService(t, r)
	_, err := svc.ListNamespaces(context.Background(), "Nawaqis-Dev")
	if !errors.Is(err, ErrMalformedOutput) {
		t.Fatalf("got %v", err)
	}
}

func TestCanAccessNamespaceYes(t *testing.T) {
	r := &fakeRunner{stdout: "yes\n"}
	svc := newTestService(t, r)
	got, err := svc.CanAccessNamespace(context.Background(), "Nawaqis-Dev", "dev-core")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Allowed {
		t.Fatal("expected allowed")
	}
	requireArgPair(t, r.lastArgs(), "--context", "Nawaqis-Dev")
	requireArgPair(t, r.lastArgs(), "-n", "dev-core")
	assertNoUseContext(t, r)
}

func TestCanAccessNamespaceNo(t *testing.T) {
	r := &fakeRunner{stdout: "no\n", exit: 1}
	svc := newTestService(t, r)
	got, err := svc.CanAccessNamespace(context.Background(), "Nawaqis-Dev", "kube-system")
	if err != nil {
		t.Fatalf("no must not be an error: %v", err)
	}
	if got.Allowed {
		t.Fatal("expected denied")
	}
}

func TestCanAccessNamespaceForbidden(t *testing.T) {
	r := &fakeRunner{exit: 1, stderr: `Error from server (Forbidden): User "x" cannot create resource "selfsubjectaccessreviews"`}
	svc := newTestService(t, r)
	_, err := svc.CanAccessNamespace(context.Background(), "Nawaqis-Dev", "dev-core")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("got %v", err)
	}
}

func TestCanAccessNamespaceUnauthorized(t *testing.T) {
	r := &fakeRunner{exit: 1, stderr: `error: You must be logged in to the server (Unauthorized)`}
	svc := newTestService(t, r)
	_, err := svc.CanAccessNamespace(context.Background(), "Nawaqis-Dev", "dev-core")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("got %v", err)
	}
}

func TestCanAccessNamespaceBadContext(t *testing.T) {
	r := &fakeRunner{exit: 1, stderr: `error: context "nope" does not exist`}
	svc := newTestService(t, r)
	_, err := svc.CanAccessNamespace(context.Background(), "nope", "dev-core")
	if !errors.Is(err, ErrContextNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestCanAccessNamespaceTimeout(t *testing.T) {
	r := &fakeRunner{
		handle: func(ctx context.Context, name string, args []string) (Result, error) {
			return Result{Name: name, Args: args}, context.DeadlineExceeded
		},
	}
	svc := newTestService(t, r, WithTimeout(time.Millisecond))
	_, err := svc.CanAccessNamespace(context.Background(), "Nawaqis-Dev", "dev-core")
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("got %v", err)
	}
}
