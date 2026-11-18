package kube

import (
	"context"
	"errors"
	"testing"
)

func TestListContextsMultiple(t *testing.T) {
	r := &fakeRunner{stdout: "Nawaqis-Dev\nNawaqis-Prod\n"}
	svc := newTestService(t, r)
	got, err := svc.ListContexts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "Nawaqis-Dev" || got[1].Name != "Nawaqis-Prod" {
		t.Fatalf("got %+v", got)
	}
	assertNoUseContext(t, r)
}

func TestListContextsZero(t *testing.T) {
	r := &fakeRunner{stdout: "\n\n"}
	svc := newTestService(t, r)
	got, err := svc.ListContexts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got %+v", got)
	}
}

func TestListContextsARNAndWhitespace(t *testing.T) {
	arn := "arn:aws:eks:eu-central-1:133136949499:cluster/Nawaqis-Development"
	r := &fakeRunner{stdout: "\n  " + arn + "  \nNawaqis-Dev\n"}
	svc := newTestService(t, r)
	got, err := svc.ListContexts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != arn || got[1].Name != "Nawaqis-Dev" {
		t.Fatalf("got %+v", got)
	}
}

func TestListContextsFailure(t *testing.T) {
	r := &fakeRunner{exit: 1, stderr: "Unable to connect to the server: connection refused"}
	svc := newTestService(t, r)
	_, err := svc.ListContexts(context.Background())
	if !errors.Is(err, ErrClusterUnavailable) {
		t.Fatalf("got %v", err)
	}
}

func TestListContextsUsesGetContextsNotUseContext(t *testing.T) {
	r := &fakeRunner{stdout: "Nawaqis-Dev\n"}
	svc := newTestService(t, r)
	if _, err := svc.ListContexts(context.Background()); err != nil {
		t.Fatal(err)
	}
	args := r.lastArgs()
	if len(args) < 3 || args[0] != "config" || args[1] != "get-contexts" {
		t.Fatalf("args %q", args)
	}
	assertNoUseContext(t, r)
}
