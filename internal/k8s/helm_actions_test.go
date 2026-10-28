// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"strings"
	"testing"
)

func TestTextDiff(t *testing.T) {
	if d := textDiff("a\nb\n", "a\nb\n"); d != "" {
		t.Errorf("одинаковые тексты: %q", d)
	}
	d := textDiff("replicas: 1\n", "replicas: 3\n")
	if !strings.Contains(d, "-replicas: 1") || !strings.Contains(d, "+replicas: 3") {
		t.Errorf("diff:\n%s", d)
	}
	if d := textDiff("", "x: 1\n"); !strings.Contains(d, "+x: 1") {
		t.Errorf("добавление в пустой текст:\n%s", d)
	}
	big := strings.Repeat("x\n", argoDiffMaxLines+1)
	if d := textDiff(big, ""); !strings.Contains(d, "слишком большой") {
		t.Errorf("большой вход: %q", d)
	}
}

func TestRollbackUninstallValidateBeforeNetwork(t *testing.T) {
	m := NewManager() // без подключения: проверка входа должна сработать раньше
	if err := m.RollbackHelmRelease("prod", "web", 0, ""); err == nil || !strings.Contains(err.Error(), "ревизия") {
		t.Errorf("rollback revision=0: %v", err)
	}
	if err := m.RollbackHelmRelease("Prod", "web", 1, ""); err == nil {
		t.Error("некорректный namespace должен отклоняться")
	}
	if err := m.UninstallHelmRelease("prod", "", false, ""); err == nil {
		t.Error("пустое имя должно отклоняться")
	}
}
