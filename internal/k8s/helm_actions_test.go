// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"strings"
	"testing"
)

func TestTextDiff(t *testing.T) {
	if d := textDiff("a\nb\n", "a\nb\n"); d != "" {
		t.Errorf("identical texts: %q", d)
	}
	d := textDiff("replicas: 1\n", "replicas: 3\n")
	if !strings.Contains(d, "-replicas: 1") || !strings.Contains(d, "+replicas: 3") {
		t.Errorf("diff:\n%s", d)
	}
	if d := textDiff("", "x: 1\n"); !strings.Contains(d, "+x: 1") {
		t.Errorf("addition to an empty text:\n%s", d)
	}
	big := strings.Repeat("x\n", argoDiffMaxLines+1)
	if d := textDiff(big, ""); !strings.Contains(d, "too large") {
		t.Errorf("large input: %q", d)
	}
}

func TestRollbackUninstallValidateBeforeNetwork(t *testing.T) {
	m := NewManager() // без подключения: проверка входа должна сработать раньше
	if err := m.RollbackHelmRelease("prod", "web", 0, ""); err == nil || !strings.Contains(err.Error(), "revision") {
		t.Errorf("rollback revision=0: %v", err)
	}
	if err := m.RollbackHelmRelease("Prod", "web", 1, ""); err == nil {
		t.Error("an invalid namespace must be rejected")
	}
	if err := m.UninstallHelmRelease("prod", "", false, ""); err == nil {
		t.Error("an empty name must be rejected")
	}
}
