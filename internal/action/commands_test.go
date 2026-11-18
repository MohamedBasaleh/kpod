package action

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MohamedBasaleh/kpod/internal/config"
	"github.com/MohamedBasaleh/kpod/internal/kube"
)

func env() config.EnvironmentLookup {
	return config.EnvironmentLookup{Alias: "dev", KubeContext: "Nawaqis-Dev", Namespace: "dev-core"}
}

func req(id ID, pod, container string) Request {
	spec, _ := Resolve(string(id))
	return Request{Spec: spec, Env: env(), Pod: pod, Container: container}
}

func TestPrepareArgsContextNamespace(t *testing.T) {
	args, err := PrepareArgs(req(Describe, "api-0", ""))
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--context Nawaqis-Dev") || !strings.Contains(joined, "-n dev-core") {
		t.Fatalf("%q", args)
	}
	if strings.Contains(joined, "config use-context") {
		t.Fatal(joined)
	}
}

func TestRailsCommands(t *testing.T) {
	c, err := PrepareArgs(req(Console, "api-0", "api"))
	if err != nil {
		t.Fatal(err)
	}
	if !containsSeq(c, "exec", "-it", "api-0", "-c", "api", "--", "bundle", "exec", "rails", "console") {
		t.Fatalf("%q", c)
	}
	d, err := PrepareArgs(req(DBConsole, "api-0", "api"))
	if err != nil {
		t.Fatal(err)
	}
	if !containsSeq(d, "rails", "dbconsole") {
		t.Fatalf("%q", d)
	}
	r := req(Runner, "api-0", "api")
	r.RunnerCode = "puts 1"
	got, err := PrepareArgs(r)
	if err != nil {
		t.Fatal(err)
	}
	if !containsSeq(got, "rails", "runner", "puts 1") {
		t.Fatalf("%q", got)
	}
}

func TestLogsModes(t *testing.T) {
	cases := []struct {
		mode LogsMode
		want string
	}{
		{LogsTail100, "--tail=100"},
		{LogsTail500, "--tail=500"},
		{LogsFollow, "-f"},
		{LogsPrevious, "--previous"},
	}
	for _, tc := range cases {
		r := req(Logs, "api-0", "api")
		r.Logs = tc.mode
		args, err := PrepareArgs(r)
		if err != nil {
			t.Fatal(err)
		}
		if !containsSeq(args, "-c", "api") || !containsArg(args, tc.want) {
			t.Fatalf("%s: %q", tc.want, args)
		}
	}
}

func TestDeleteRequiresConfirm(t *testing.T) {
	r := req(Delete, "api-0", "")
	_, err := PrepareArgs(r)
	if !errors.Is(err, kube.ErrNeedsConfirm) {
		t.Fatalf("%v", err)
	}
	r.Confirmed = true
	args, err := PrepareArgs(r)
	if err != nil {
		t.Fatal(err)
	}
	if !containsSeq(args, "delete", "pod", "api-0") {
		t.Fatalf("%q", args)
	}
}

func TestRestartRequiresConfirmAndDeploy(t *testing.T) {
	r := req(Restart, "api-0", "")
	_, err := PrepareWorkloadArgs(r, "api", true)
	if !errors.Is(err, kube.ErrNeedsConfirm) {
		t.Fatalf("%v", err)
	}
	r.Confirmed = true
	args, err := PrepareWorkloadArgs(r, "api", true)
	if err != nil {
		t.Fatal(err)
	}
	if !containsSeq(args, "rollout", "restart", "deployment/api") {
		t.Fatalf("%q", args)
	}
	if !containsSeq(args, "--context", "Nawaqis-Dev", "-n", "dev-core") {
		t.Fatalf("%q", args)
	}
	_, err = PrepareWorkloadArgs(r, "", true)
	if !errors.Is(err, kube.ErrDeploymentNotFound) {
		t.Fatalf("%v", err)
	}
}

func TestPortForwardMapping(t *testing.T) {
	r := req(PortForward, "api-0", "")
	r.LocalPort, r.RemotePort = 3000, 3000
	args, err := PrepareArgs(r)
	if err != nil {
		t.Fatal(err)
	}
	if !containsSeq(args, "port-forward", "pod/api-0", "3000:3000") {
		t.Fatalf("%q", args)
	}
}

func TestCopyUsesContainer(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(local, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	up := req(Upload, "api-0", "api")
	up.LocalPath, up.RemotePath = local, "/tmp/file.txt"
	args, err := PrepareArgs(up)
	if err != nil {
		t.Fatal(err)
	}
	if !containsSeq(args, "cp", local, "dev-core/api-0:/tmp/file.txt", "-c", "api") {
		t.Fatalf("%q", args)
	}
	dl := req(Download, "api-0", "api")
	dl.RemotePath, dl.LocalPath = "/tmp/file.txt", local
	args, err = PrepareArgs(dl)
	if err != nil {
		t.Fatal(err)
	}
	if !containsSeq(args, "cp", "dev-core/api-0:/tmp/file.txt", local, "-c", "api") {
		t.Fatalf("%q", args)
	}
}

func TestYAMLDescribeEventsMetrics(t *testing.T) {
	y, _ := PrepareArgs(req(YAML, "api-0", ""))
	if !containsSeq(y, "get", "pod", "api-0", "-o", "yaml") {
		t.Fatalf("%q", y)
	}
	d, _ := PrepareArgs(req(Describe, "api-0", ""))
	if !containsSeq(d, "describe", "pod", "api-0") {
		t.Fatalf("%q", d)
	}
	e, _ := PrepareArgs(req(Events, "api-0", ""))
	if !containsSeq(e, "get", "events", "--field-selector", "involvedObject.name=api-0") {
		t.Fatalf("%q", e)
	}
	m, _ := PrepareArgs(req(Metrics, "api-0", ""))
	if !containsSeq(m, "top", "pod", "api-0") {
		t.Fatalf("%q", m)
	}
}

func containsArg(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}

func containsSeq(args []string, seq ...string) bool {
	if len(seq) == 0 {
		return true
	}
	for i := 0; i <= len(args)-len(seq); i++ {
		ok := true
		for j := range seq {
			if args[i+j] != seq[j] {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}
