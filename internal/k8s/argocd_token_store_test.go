// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFileTokenStoreRoundTrip(t *testing.T) {
	dir := useTempTokenDir(t)
	s := NewFileTokenStore()

	if _, err := s.Get("c1"); !errors.Is(err, ErrArgoTokenNotSet) {
		t.Fatalf("без токена ждали ErrArgoTokenNotSet, получили %v", err)
	}
	if err := s.Put("c1", "tok"); err != nil {
		t.Fatal(err)
	}
	if got, err := s.Get("c1"); err != nil || got != "tok" {
		t.Fatalf("Get = %q, %v", got, err)
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.token"))
	if len(files) != 1 {
		t.Fatalf("ждали один файл токена, есть %v", files)
	}
	// В Windows Unix-прав нет (Perm() всегда 0666): проверяем только там, где они есть.
	if runtime.GOOS != "windows" {
		if st, _ := os.Stat(files[0]); st.Mode().Perm() != 0o600 {
			t.Errorf("права файла %v, ждали 0600", st.Mode().Perm())
		}
	}
	if err := s.Delete("c1"); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete("c1"); err != nil {
		t.Fatalf("повторное удаление не должно быть ошибкой: %v", err)
	}
}

type memTokenStore map[string]string

func (m memTokenStore) Put(id, t string) error { m[id] = t; return nil }
func (m memTokenStore) Get(id string) (string, error) {
	if v, ok := m[id]; ok {
		return v, nil
	}
	return "", ErrArgoTokenNotSet
}
func (m memTokenStore) Delete(id string) error { delete(m, id); return nil }

// Хранилище подменяется целиком; валидация токена остаётся в saveArgoToken.
func TestTokenStoreIsPluggable(t *testing.T) {
	prev := argoTokens
	mem := memTokenStore{}
	SetTokenStore(mem)
	t.Cleanup(func() { argoTokens = prev })

	if err := saveArgoToken("c", "  abc \n"); err != nil {
		t.Fatal(err)
	}
	if mem["c"] != "abc" {
		t.Fatalf("в хранилище %q, ждали нормализованный abc", mem["c"])
	}
	if err := saveArgoToken("c", "bad token"); err == nil {
		t.Fatal("токен с пробелом должен отклоняться до хранилища")
	}
	if err := deleteArgoToken("c"); err != nil || len(mem) != 0 {
		t.Fatalf("удаление: %v, осталось %v", err, mem)
	}
}
