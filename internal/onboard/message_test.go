package onboard

import (
	"errors"
	"strings"
	"testing"

	"github.com/MohamedBasaleh/kpod/internal/kube"
)

func TestUserMessageKubectlMissing(t *testing.T) {
	msg := UserMessage(kube.ErrKubectlNotFound)
	if !strings.Contains(msg, "kubectl was not found") {
		t.Fatalf("%q", msg)
	}
	if strings.Contains(msg, "exit status") {
		t.Fatalf("%q", msg)
	}
}

func TestUserMessageForbidden(t *testing.T) {
	msg := UserMessage(kube.ErrForbidden)
	if !strings.Contains(strings.ToLower(msg), "cannot list namespaces") {
		t.Fatalf("%q", msg)
	}
}

func TestUserMessageStripsExitStatus(t *testing.T) {
	msg := UserMessage(errors.New("exit status 1"))
	if strings.Contains(msg, "exit status") {
		t.Fatalf("%q", msg)
	}
}
