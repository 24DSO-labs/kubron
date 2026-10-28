// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func useTempTokenDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "argocd")
	prev := argoTokenDir
	argoTokenDir = func() string { return dir }
	t.Cleanup(func() { argoTokenDir = prev })
	return dir
}

func TestArgoTokenRoundTripAndPermissions(t *testing.T) {
	dir := useTempTokenDir(t)

	if _, err := loadArgoToken("c1"); !errors.Is(err, ErrArgoTokenNotSet) {
		t.Fatalf("expected ErrArgoTokenNotSet before saving, got %v", err)
	}
	if err := saveArgoToken("c1", "  eyJ.tok.en \n"); err != nil {
		t.Fatal(err)
	}
	got, err := loadArgoToken("c1")
	if err != nil || got != "eyJ.tok.en" {
		t.Fatalf("load: %q %v", got, err)
	}

	if runtime.GOOS != "windows" {
		path, _ := argoTokenPath("c1")
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != 0o600 {
			t.Errorf("token file permissions %v, expected 0600", fi.Mode().Perm())
		}
		di, _ := os.Stat(dir)
		if di.Mode().Perm() != 0o700 {
			t.Errorf("directory permissions %v, expected 0700", di.Mode().Perm())
		}
	}

	// Перезапись заменяет значение; временных файлов не остаётся.
	if err := saveArgoToken("c1", "second"); err != nil {
		t.Fatal(err)
	}
	if got, _ := loadArgoToken("c1"); got != "second" {
		t.Errorf("overwrite: %q", got)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("one file must remain in the directory, got %d", len(entries))
	}
}

func TestArgoTokenIsolatedPerClusterAndPathSafe(t *testing.T) {
	dir := useTempTokenDir(t)
	_ = saveArgoToken("cluster-a", "tok-a")
	_ = saveArgoToken("cluster-b", "tok-b")
	a, _ := loadArgoToken("cluster-a")
	b, _ := loadArgoToken("cluster-b")
	if a != "tok-a" || b != "tok-b" {
		t.Fatalf("tokens are mixed up: %q %q", a, b)
	}

	// Идентификатор с «..» и разделителями не может вывести файл из каталога токенов.
	evil := "../../etc/passwd"
	if err := saveArgoToken(evil, "x"); err != nil {
		t.Fatal(err)
	}
	p, _ := argoTokenPath(evil)
	if filepath.Dir(p) != dir {
		t.Errorf("path escaped the token directory: %s", p)
	}
}

func TestArgoTokenValidationDoesNotEchoValue(t *testing.T) {
	useTempTokenDir(t)
	long := strings.Repeat("a", argoTokenMaxLen+1)
	cases := map[string]string{
		"empty":        "   ",
		"inner space":  "SECRET part2",
		"newline":      "SECRET\nline",
		"control char": "SECRET\x00",
		"too long":     long,
	}
	for name, tok := range cases {
		t.Run(name, func(t *testing.T) {
			err := saveArgoToken("c1", tok)
			if err == nil {
				t.Fatal("expected an error")
			}
			if strings.Contains(err.Error(), "SECRET") || strings.Contains(err.Error(), long[:50]) {
				t.Errorf("error text contains the token: %v", err)
			}
		})
	}
	if _, err := loadArgoToken("c1"); !errors.Is(err, ErrArgoTokenNotSet) {
		t.Errorf("an invalid token must not be saved: %v", err)
	}
	if err := saveArgoToken("", "tok"); err == nil {
		t.Error("saving without a cluster is not allowed")
	}
}

func TestArgoTokenDeleteIsIdempotent(t *testing.T) {
	useTempTokenDir(t)
	if err := deleteArgoToken("c1"); err != nil {
		t.Fatalf("deleting a nonexistent token is not an error: %v", err)
	}
	_ = saveArgoToken("c1", "tok")
	if err := deleteArgoToken("c1"); err != nil {
		t.Fatal(err)
	}
	if _, err := loadArgoToken("c1"); !errors.Is(err, ErrArgoTokenNotSet) {
		t.Errorf("the token must be gone after deletion: %v", err)
	}
}

func TestManagerArgoAPIStatus(t *testing.T) {
	useTempTokenDir(t)
	m := NewManager()
	if _, err := m.GetArgoAPIStatus(); err == nil {
		t.Error("status is unavailable without an active cluster")
	}
	if err := m.SetArgoAPIToken("tok"); err == nil {
		t.Error("saving without an active cluster is not allowed")
	}

	m.activeCtx = "c1"
	st, err := m.GetArgoAPIStatus()
	if err != nil || st.Configured {
		t.Fatalf("before saving: %+v %v", st, err)
	}
	if err := m.SetArgoAPIToken("tok"); err != nil {
		t.Fatal(err)
	}
	if st, _ := m.GetArgoAPIStatus(); !st.Configured {
		t.Error("Configured must be true after saving")
	}
	// Статус не содержит токена ни в каком виде: структура — один bool.
	if err := m.DeleteArgoAPIToken(); err != nil {
		t.Fatal(err)
	}
	if st, _ := m.GetArgoAPIStatus(); st.Configured {
		t.Error("Configured must be false after deletion")
	}
}
