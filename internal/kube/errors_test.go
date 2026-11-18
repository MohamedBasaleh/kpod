package kube

import (
	"context"
	"errors"
	"testing"
)

func TestClassifyErrorFixtures(t *testing.T) {
	cases := []struct {
		name   string
		stderr string
		want   error
	}{
		{
			name:   "forbidden",
			stderr: `Error from server (Forbidden): namespaces is forbidden: User "arn:aws:iam::1:user/x" cannot list resource "namespaces" in API group "" at the cluster scope`,
			want:   ErrForbidden,
		},
		{
			name:   "unauthorized",
			stderr: `error: You must be logged in to the server (Unauthorized)`,
			want:   ErrUnauthorized,
		},
		{
			name:   "context missing",
			stderr: `error: context "nope" does not exist`,
			want:   ErrContextNotFound,
		},
		{
			name:   "context was not found",
			stderr: `Error in configuration: context was not found for specified context: does-not-exist`,
			want:   ErrContextNotFound,
		},
		{
			name:   "connection refused",
			stderr: `Unable to connect to the server: dial tcp 127.0.0.1:6443: connect: connection refused`,
			want:   ErrClusterUnavailable,
		},
		{
			name:   "io timeout",
			stderr: `Unable to connect to the server: net/http: request canceled (Client.Timeout exceeded while awaiting headers): i/o timeout`,
			want:   ErrClusterUnavailable,
		},
		{
			name:   "unknown stays unknown",
			stderr: `error: something completely unexpected happened`,
			want:   ErrUnknown,
		},
		{
			name:   "exec missing",
			stderr: `error: Internal error occurred: exec: "bash": executable file not found in $PATH`,
			want:   ErrExecNotFound,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := classifyOutput(Result{ExitCode: 1, Stderr: []byte(tc.stderr)}, nil)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
		})
	}
}

func TestClassifyDeadline(t *testing.T) {
	err := classifyRunError(context.DeadlineExceeded, Result{})
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("got %v", err)
	}
}
