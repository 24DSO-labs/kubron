// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFormatMatchLabels(t *testing.T) {
	cases := []struct {
		name string
		in   *metav1.LabelSelector
		want string
	}{
		{"nil", nil, ""},
		{"empty", &metav1.LabelSelector{}, ""},
		{"single", &metav1.LabelSelector{MatchLabels: map[string]string{"app": "web"}}, "app=web"},
		{"sorted", &metav1.LabelSelector{MatchLabels: map[string]string{"b": "2", "a": "1", "c": "3"}}, "a=1, b=2, c=3"},
	}
	for _, c := range cases {
		// Порядок обхода map случаен: повторяем, чтобы поймать нестабильный результат
		for i := 0; i < 20; i++ {
			if got := formatMatchLabels(c.in); got != c.want {
				t.Fatalf("%s: got %q, want %q", c.name, got, c.want)
			}
		}
	}
}

func TestPromLabelValue(t *testing.T) {
	cases := map[string]string{
		"default":      "default",
		`a"b`:          `a\"b`,
		`a\b`:          `a\\b`,
		"line1\nline2": `line1\nline2`,
	}
	for in, want := range cases {
		if got := promLabelValue(in); got != want {
			t.Errorf("promLabelValue(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPromPodQueriesByScope(t *testing.T) {
	mustScope := func(in ...string) nsScope {
		s, err := parseScope(in)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	cases := []struct {
		name         string
		scope        nsScope
		wantCPU      string // подстрока, которая обязана быть в CPU-запросе
		wantMem      string
		forbidSubstr string // чего не должно быть
	}{
		{"all", mustScope(), `{container!="", container!="POD"}`, `{container!="", container!="POD"}`, "namespace"},
		{"one: exact match", mustScope("a"), `namespace="a"}`, `{namespace="a",`, `namespace=~`},
		{"several: regex", mustScope("b", "a"), `namespace=~"a|b"}`, `{namespace=~"a|b",`, `namespace="`},
	}
	for _, c := range cases {
		cpu, mem := promQueryPodCPUByNs(c.scope), promQueryPodMemByNs(c.scope)
		if !strings.Contains(cpu, c.wantCPU) || !strings.Contains(mem, c.wantMem) {
			t.Errorf("%s: cpu=%q mem=%q", c.name, cpu, mem)
		}
		// «все»: подстрока namespace допустима только в `sum by (pod, namespace)`
		for _, q := range []string{cpu, mem} {
			if strings.Contains(strings.ReplaceAll(q, "by (pod, namespace)", ""), c.forbidSubstr) {
				t.Errorf("%s: unexpected %q in %q", c.name, c.forbidSubstr, q)
			}
		}
	}
	// метасимволы не должны попасть в регулярку без экранирования даже в обход parseScope
	q := promQueryPodCPUByNs(nsScope{names: []string{"a.b", "c"}})
	if !strings.Contains(q, `namespace=~"a\\.b|c"`) {
		t.Errorf("no escaping: %q", q)
	}
}

func TestWorkloadStatus(t *testing.T) {
	cases := []struct {
		desired, ready int32
		want           string
	}{
		{0, 0, "Idle"},
		{3, 3, "Running"},
		{3, 5, "Running"},
		{3, 0, "Unavailable"},
		{3, 1, "Degraded"},
	}
	for _, c := range cases {
		if got := workloadStatus(c.desired, c.ready); got != c.want {
			t.Errorf("workloadStatus(%d,%d) = %q, want %q", c.desired, c.ready, got, c.want)
		}
	}
}
