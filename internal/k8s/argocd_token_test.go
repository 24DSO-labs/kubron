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
		t.Fatalf("до сохранения ожидался ErrArgoTokenNotSet, получили %v", err)
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
			t.Errorf("права файла токена %v, нужны 0600", fi.Mode().Perm())
		}
		di, _ := os.Stat(dir)
		if di.Mode().Perm() != 0o700 {
			t.Errorf("права каталога %v, нужны 0700", di.Mode().Perm())
		}
	}

	// Перезапись заменяет значение; временных файлов не остаётся.
	if err := saveArgoToken("c1", "second"); err != nil {
		t.Fatal(err)
	}
	if got, _ := loadArgoToken("c1"); got != "second" {
		t.Errorf("перезапись: %q", got)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("в каталоге должен остаться один файл, есть %d", len(entries))
	}
}

func TestArgoTokenIsolatedPerClusterAndPathSafe(t *testing.T) {
	dir := useTempTokenDir(t)
	_ = saveArgoToken("cluster-a", "tok-a")
	_ = saveArgoToken("cluster-b", "tok-b")
	a, _ := loadArgoToken("cluster-a")
	b, _ := loadArgoToken("cluster-b")
	if a != "tok-a" || b != "tok-b" {
		t.Fatalf("токены перепутаны: %q %q", a, b)
	}

	// Идентификатор с «..» и разделителями не может вывести файл из каталога токенов.
	evil := "../../etc/passwd"
	if err := saveArgoToken(evil, "x"); err != nil {
		t.Fatal(err)
	}
	p, _ := argoTokenPath(evil)
	if filepath.Dir(p) != dir {
		t.Errorf("путь вышел из каталога токенов: %s", p)
	}
}

func TestArgoTokenValidationDoesNotEchoValue(t *testing.T) {
	useTempTokenDir(t)
	long := strings.Repeat("a", argoTokenMaxLen+1)
	cases := map[string]string{
		"пустой":          "   ",
		"пробел внутри":   "SECRET part2",
		"перевод строки":  "SECRET\nline",
		"управляющий":     "SECRET\x00",
		"слишком длинный": long,
	}
	for name, tok := range cases {
		t.Run(name, func(t *testing.T) {
			err := saveArgoToken("c1", tok)
			if err == nil {
				t.Fatal("ожидалась ошибка")
			}
			if strings.Contains(err.Error(), "SECRET") || strings.Contains(err.Error(), long[:50]) {
				t.Errorf("текст ошибки содержит токен: %v", err)
			}
		})
	}
	if _, err := loadArgoToken("c1"); !errors.Is(err, ErrArgoTokenNotSet) {
		t.Errorf("невалидный токен не должен сохраняться: %v", err)
	}
	if err := saveArgoToken("", "tok"); err == nil {
		t.Error("без кластера сохранять нельзя")
	}
}

func TestArgoTokenDeleteIsIdempotent(t *testing.T) {
	useTempTokenDir(t)
	if err := deleteArgoToken("c1"); err != nil {
		t.Fatalf("удаление несуществующего токена не ошибка: %v", err)
	}
	_ = saveArgoToken("c1", "tok")
	if err := deleteArgoToken("c1"); err != nil {
		t.Fatal(err)
	}
	if _, err := loadArgoToken("c1"); !errors.Is(err, ErrArgoTokenNotSet) {
		t.Errorf("после удаления токена быть не должно: %v", err)
	}
}

func TestManagerArgoAPIStatus(t *testing.T) {
	useTempTokenDir(t)
	m := NewManager()
	if _, err := m.GetArgoAPIStatus(); err == nil {
		t.Error("без активного кластера статус недоступен")
	}
	if err := m.SetArgoAPIToken("tok"); err == nil {
		t.Error("без активного кластера сохранять нельзя")
	}

	m.activeCtx = "c1"
	st, err := m.GetArgoAPIStatus()
	if err != nil || st.Configured {
		t.Fatalf("до сохранения: %+v %v", st, err)
	}
	if err := m.SetArgoAPIToken("tok"); err != nil {
		t.Fatal(err)
	}
	if st, _ := m.GetArgoAPIStatus(); !st.Configured {
		t.Error("после сохранения Configured должен быть true")
	}
	// Статус не содержит токена ни в каком виде: структура — один bool.
	if err := m.DeleteArgoAPIToken(); err != nil {
		t.Fatal(err)
	}
	if st, _ := m.GetArgoAPIStatus(); st.Configured {
		t.Error("после удаления Configured должен быть false")
	}
}
