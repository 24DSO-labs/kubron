// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"path/filepath"
	"testing"
)

func TestGetKubronDirHonorsEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("KUBRON_DATA_DIR", dir)
	if got := GetKubronDir(); got != dir {
		t.Errorf("GetKubronDir() = %q, ждали %q", got, dir)
	}
	t.Setenv("KUBRON_DATA_DIR", "")
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	if got, want := GetKubronDir(), filepath.Join(dir, ".kubron"); got != want {
		t.Errorf("без переменной: %q, ждали %q", got, want)
	}
}
