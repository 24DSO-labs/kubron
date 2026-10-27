// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"path/filepath"
	"testing"
)

func TestSetKubronDirOverridesDefault(t *testing.T) {
	t.Cleanup(func() { SetKubronDir("") })
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	// Переменная окружения каталог данных не меняет (десктоп её игнорирует)
	t.Setenv("KUBRON_DATA_DIR", t.TempDir())
	want := filepath.Join(home, ".kubron")
	if got := GetKubronDir(); got != want {
		t.Fatalf("по умолчанию: %q, ждали %q", got, want)
	}
	custom := t.TempDir()
	SetKubronDir(custom)
	if got := GetKubronDir(); got != custom {
		t.Errorf("после SetKubronDir: %q, ждали %q", got, custom)
	}
	SetKubronDir("")
	if got := GetKubronDir(); got != want {
		t.Errorf("после сброса: %q, ждали %q", got, want)
	}
}
