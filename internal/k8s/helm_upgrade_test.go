// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"strings"
	"testing"
)

func TestParseHelmValues(t *testing.T) {
	for _, in := range []string{"", "  \n", "null", "{}"} {
		got, err := parseHelmValues(in)
		if err != nil || got == nil || len(got) != 0 {
			t.Fatalf("%q: got %v, %v", in, got, err)
		}
	}
	got, err := parseHelmValues("replicaCount: 3\nimage:\n  tag: v1\n")
	if err != nil {
		t.Fatal(err)
	}
	if got["replicaCount"] != float64(3) {
		t.Fatalf("replicaCount = %#v", got["replicaCount"])
	}
	for name, in := range map[string]string{
		"список":       "- a\n- b\n",
		"скаляр":       "just a string",
		"битый YAML":   "a: [unclosed",
		"слишком длин": strings.Repeat("a", helmValuesMaxBytes+1),
	} {
		if _, err := parseHelmValues(in); err == nil {
			t.Errorf("%s: ожидалась ошибка", name)
		}
	}
}

func TestMarshalHelmValues(t *testing.T) {
	if s, err := marshalHelmValues(nil); err != nil || s != "" {
		t.Fatalf("пустые values: %q, %v", s, err)
	}
	s, err := marshalHelmValues(map[string]interface{}{"b": 1, "a": "x"})
	if err != nil || s != "a: x\nb: 1\n" {
		t.Fatalf("got %q, %v", s, err)
	}
}

func TestUpgradeValidatesBeforeNetwork(t *testing.T) {
	m := NewManager()
	if _, err := m.PreviewHelmUpgrade("Prod", "web", ""); err == nil {
		t.Error("некорректный namespace должен отклоняться")
	}
	if _, err := m.PreviewHelmUpgrade("prod", "web", "- x"); err == nil || !strings.Contains(err.Error(), "YAML") {
		t.Errorf("не-map values: %v", err)
	}
	if _, err := m.UpgradeHelmRelease("prod", "", ""); err == nil {
		t.Error("пустое имя должно отклоняться")
	}
}
