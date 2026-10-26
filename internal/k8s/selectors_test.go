// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
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
