// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// Токен Argo CD API не должен переживать удаление кластера, к которому он относится.
func TestDeleteClusterRemovesArgoToken(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // Windows
	useTempTokenDir(t)

	if err := os.MkdirAll(getClustersDir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(getClustersDir(), "c1.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := saveArgoToken("c1", "tok.en"); err != nil {
		t.Fatal(err)
	}
	if err := saveArgoToken("c2", "other.tok"); err != nil {
		t.Fatal(err)
	}

	if err := DeleteCluster("c1"); err != nil {
		t.Fatal(err)
	}
	if _, err := loadArgoToken("c1"); !errors.Is(err, ErrArgoTokenNotSet) {
		t.Errorf("токен удалённого кластера остался: %v", err)
	}
	if got, err := loadArgoToken("c2"); err != nil || got != "other.tok" {
		t.Errorf("токен другого кластера пострадал: %q %v", got, err)
	}
	// Повторное удаление (токена и файлов уже нет) — не ошибка.
	if err := DeleteCluster("c1"); err != nil {
		t.Errorf("повторное удаление: %v", err)
	}
}
