package tui

import (
	"regexp"
	"strings"
	"testing"

	"github.com/MohamedBasaleh/kpod/internal/kube"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func plain(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

func lineContaining(view, needle string) string {
	for _, line := range strings.Split(plain(view), "\n") {
		if strings.Contains(line, needle) {
			return line
		}
	}
	return ""
}

func detailModel(t *testing.T, width int) mainModel {
	t.Helper()
	m := envModel(t)
	m.width = width
	m.current.view = viewPodDetail
	m.current.podName = "dev-core-api-586c8c88dc-9p5xs"
	m.pods = []kube.Pod{{
		Name:     "dev-core-api-586c8c88dc-9p5xs",
		Status:   "Running",
		Ready:    1,
		Total:    1,
		Restarts: 0,
		Node:     "i-049cefc4fba36d434",
		Containers: []kube.Container{
			{Name: "init-db", Init: true},
			{Name: "dev-core-api"},
		},
	}}
	return m
}

func TestPodDetailWideTwoColumns(t *testing.T) {
	m := detailModel(t, 100)
	view := m.View()
	p := plain(view)
	if strings.Contains(p, "Environment\n") {
		t.Fatalf("stacked labels:\n%s", p)
	}
	envLine := lineContaining(view, "Environment")
	if !strings.Contains(envLine, "dev") || !strings.Contains(envLine, "Status") || !strings.Contains(envLine, "Running") {
		t.Fatalf("wide row should pair Environment with Status: %q", envLine)
	}
	ctxLine := lineContaining(view, "Context")
	if !strings.Contains(ctxLine, "Nawaqis-Dev") || !strings.Contains(ctxLine, "Ready") || !strings.Contains(ctxLine, "1/1") {
		t.Fatalf("%q", ctxLine)
	}
	nsLine := lineContaining(view, "Namespace")
	if !strings.Contains(nsLine, "dev-core") || !strings.Contains(nsLine, "Restarts") {
		t.Fatalf("%q", nsLine)
	}
	if !strings.Contains(p, "dev-core-api") || !strings.Contains(p, "init-db") {
		t.Fatalf("containers:\n%s", p)
	}
	if strings.Index(p, "Pod Actions") < strings.Index(p, "Environment") {
		t.Fatal("actions should follow pod info")
	}
}

func TestPodDetailNarrowSingleColumn(t *testing.T) {
	m := detailModel(t, 48)
	view := m.View()
	envLine := lineContaining(view, "Environment")
	if envLine == "" || !strings.Contains(envLine, "dev") {
		t.Fatalf("%q", envLine)
	}
	if strings.Contains(envLine, "Status") {
		t.Fatalf("narrow should not place Status on the Environment row: %q", envLine)
	}
	st := lineContaining(view, "Status")
	if !strings.Contains(st, "Running") {
		t.Fatalf("%q", st)
	}
	if strings.Contains(plain(view), "Environment\n  ") {
		t.Fatal("must not stack label above indented value")
	}
	for _, line := range strings.Split(plain(view), "\n") {
		if len([]rune(line)) > 48 && strings.Contains(line, "Environment") {
			t.Fatalf("narrow line too long: %q (%d)", line, len([]rune(line)))
		}
	}
}

func TestPodDetailMediumTruncatesLongValues(t *testing.T) {
	m := detailModel(t, 68)
	pod := m.pods[0]
	pod.Node = "ip-0123456789abcdef0123456789abcdef-very-long-eks-node"
	longCtx := "arn:aws:eks:eu-central-1:133136949499:cluster/nawaqis-very-long"
	info := m.renderPodInfo("dev", longCtx, "dev-core", pod)
	p := plain(info)
	for _, line := range strings.Split(strings.TrimSuffix(p, "\n"), "\n") {
		if lipglossWidth(line) > 68 {
			t.Fatalf("line exceeds width 68: %q (%d)", line, lipglossWidth(line))
		}
	}
	ctxLine := lineContaining(info, "Context")
	if strings.Contains(ctxLine, longCtx) {
		t.Fatalf("long context should truncate in medium layout: %q", ctxLine)
	}
	if !strings.Contains(ctxLine, "…") {
		t.Fatalf("expected ellipsis: %q", ctxLine)
	}
}

func TestPodDetailKeepsFullValuesInModel(t *testing.T) {
	m := detailModel(t, 68)
	long := "ip-0123456789abcdef0123456789abcdef-very-long-eks-node"
	m.pods[0].Node = long
	_ = m.View()
	if m.pods[0].Node != long {
		t.Fatal("truncation must be visual only")
	}
}

func TestPodDetailNoColorKeepsStatusText(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	m := detailModel(t, 100)
	p := plain(m.View())
	if !strings.Contains(p, "Running") {
		t.Fatalf("%s", p)
	}
	if strings.Contains(p, "●") {
		t.Fatal("NO_COLOR should not add a status glyph")
	}
}

func TestKVLayoutThresholds(t *testing.T) {
	if kvLayoutFor(100).Columns != 2 || kvLayoutFor(70).Columns != 2 || kvLayoutFor(50).Columns != 1 {
		t.Fatalf("%+v %+v %+v", kvLayoutFor(100), kvLayoutFor(70), kvLayoutFor(48))
	}
}

func lipglossWidth(s string) int {
	return len([]rune(s))
}
