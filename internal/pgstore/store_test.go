// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package pgstore

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/VadimIl224/kubron/internal/secretbox"
	"github.com/VadimIl224/kubron/pkg/ext"
)

func keyString(id string) string {
	b := make([]byte, secretbox.KeySize)
	_, _ = rand.Read(b)
	return id + ":" + base64.StdEncoding.EncodeToString(b)
}

// newTestStore подключается к БД из KUBRON_TEST_DATABASE_URL и пересоздаёт в ней схему public.
// База должна быть одноразовой: её имя обязано содержать «test».
func newTestStore(t *testing.T, keyStr string, prev ...string) *Store {
	t.Helper()
	url := os.Getenv("KUBRON_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("KUBRON_TEST_DATABASE_URL is not set")
	}
	if !strings.Contains(url[strings.LastIndex(url, "/"):], "test") {
		t.Fatal("the test database name must contain \"test\": the schema is dropped")
	}
	keys, err := secretbox.ParseKeyring(keyStr, prev...)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	s, err := Open(ctx, url, keys)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	if _, err := s.pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.Init(ctx); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestInitIsIdempotent(t *testing.T) {
	s := newTestStore(t, keyString("k1"))
	org, team := s.orgID, s.teamID
	if err := s.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	if s.orgID != org || s.teamID != team {
		t.Error("повторный Init создал новые организацию или команду")
	}
	if res, err := s.Migrate(context.Background()); err != nil || len(res.Applied) != 0 {
		t.Errorf("повторная миграция: %+v %v", res, err)
	}
}

func TestOpenRequiresKeys(t *testing.T) {
	if _, err := Open(context.Background(), "postgres://x", nil); !errors.Is(err, secretbox.ErrNoKey) {
		t.Fatal(err)
	}
}

func TestUsersRoundTrip(t *testing.T) {
	s := newTestStore(t, keyString("k1"))
	u := s.Users()
	d, err := u.Load()
	if err != nil || len(d.Users) != 0 || d.Settings.IdleTimeoutMinutes != nil {
		t.Fatalf("пустая БД: %+v %v", d, err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	idle := 0
	in := &ext.UserData{
		Users: []ext.UserRecord{
			{ID: "u1", Username: "Admin", Role: "admin", PasswordHash: "h1", CreatedAt: now, UpdatedAt: now},
			{ID: "u2", Username: "bob", Role: "operator", PasswordHash: "h2", Disabled: true, MustChangePassword: true, CreatedAt: now.Add(time.Second), UpdatedAt: now, LastLoginAt: &now},
		},
		Settings: ext.SecuritySettings{IdleTimeoutMinutes: &idle},
	}
	if err := u.Save(in); err != nil {
		t.Fatal(err)
	}
	got, err := u.Load()
	if err != nil || len(got.Users) != 2 {
		t.Fatalf("%+v %v", got, err)
	}
	if got.Users[0].ID != "u1" || got.Users[1].Username != "bob" || !got.Users[1].Disabled || !got.Users[1].MustChangePassword ||
		got.Users[1].LastLoginAt == nil || !got.Users[1].LastLoginAt.Equal(now) {
		t.Errorf("поля не сохранились: %+v", got.Users)
	}
	if got.Settings.IdleTimeoutMinutes == nil || *got.Settings.IdleTimeoutMinutes != 0 {
		t.Errorf("явный 0 должен отличаться от «не задано»: %+v", got.Settings)
	}

	// Обновление и удаление: u2 пропадает из набора — удаляется; u1 меняется.
	in.Users = []ext.UserRecord{{ID: "u1", Username: "Admin", Role: "admin", PasswordHash: "h1-new", CreatedAt: now, UpdatedAt: now}}
	if err := u.Save(in); err != nil {
		t.Fatal(err)
	}
	got, _ = u.Load()
	if len(got.Users) != 1 || got.Users[0].PasswordHash != "h1-new" {
		t.Errorf("%+v", got.Users)
	}
}

func TestUsersValidation(t *testing.T) {
	s := newTestStore(t, keyString("k1"))
	u := s.Users()
	now := time.Now()
	if err := u.Save(&ext.UserData{Users: []ext.UserRecord{{ID: "u1", Username: "a", Role: "viewer", CreatedAt: now}}}); err == nil {
		t.Error("неизвестная роль должна отклоняться")
	}
	// Имена уникальны без учёта регистра.
	err := u.Save(&ext.UserData{Users: []ext.UserRecord{
		{ID: "u1", Username: "Bob", Role: "user", CreatedAt: now},
		{ID: "u2", Username: "bob", Role: "user", CreatedAt: now},
	}})
	if !errors.Is(err, ErrConflict) {
		t.Errorf("повтор имени: %v", err)
	}
	// Неудачное сохранение не оставляет частичных данных (транзакция).
	if d, _ := u.Load(); len(d.Users) != 0 {
		t.Errorf("частичная запись: %+v", d.Users)
	}
}

func TestAuditAppendAndScan(t *testing.T) {
	s := newTestStore(t, keyString("k1"))
	a := s.Audit()
	if err := a.ScanNewestFirst(func([]byte) bool { t.Fatal("журнал должен быть пуст"); return false }); err != nil {
		t.Fatal(err)
	}
	base := time.Now().UTC().Truncate(time.Second)
	for i, act := range []string{"Login", "DeletePod", "CreateUser"} {
		line, _ := json.Marshal(map[string]any{"time": base.Add(time.Duration(i) * time.Second), "user": "alice", "role": "admin",
			"action": act, "result": "ok", "detail": "x=" + act, "source": "10.0.0.1"})
		if err := a.Append(line); err != nil {
			t.Fatal(err)
		}
	}
	// Неизвестный результат не ломает запись.
	if err := a.Append([]byte(`{"time":"` + base.Add(3*time.Second).Format(time.RFC3339) + `","user":"u","action":"Odd","result":"weird"}`)); err != nil {
		t.Fatal(err)
	}
	if err := a.Append([]byte(`not json`)); err == nil {
		t.Error("некорректная строка должна отклоняться")
	}
	var actions []string
	_ = a.ScanNewestFirst(func(line []byte) bool {
		var e struct{ Action, Result, Source string }
		if err := json.Unmarshal(line, &e); err != nil {
			t.Fatal(err)
		}
		actions = append(actions, e.Action)
		return true
	})
	if strings.Join(actions, ",") != "Odd,CreateUser,DeletePod,Login" {
		t.Errorf("порядок от новых к старым: %v", actions)
	}
	n := 0
	_ = a.ScanNewestFirst(func([]byte) bool { n++; return n < 2 })
	if n != 2 {
		t.Errorf("досрочная остановка: %d", n)
	}
}

func saveCluster(t *testing.T, s *Store, id, name string, kc []byte) {
	t.Helper()
	if err := s.Clusters().Save(&Cluster{ID: id, DisplayName: name, Environment: "dev", ContextName: "ctx"}, kc); err != nil {
		t.Fatal(err)
	}
}

func TestClustersLifecycle(t *testing.T) {
	s := newTestStore(t, keyString("k1"))
	c := s.Clusters()
	kc := []byte("apiVersion: v1\nkind: Config\n")
	saveCluster(t, s, "prod", "Production", kc)

	got, err := c.Get("prod")
	if err != nil || got.DisplayName != "Production" || got.TeamID != s.teamID || got.Mode != "kubeconfig" || got.AddedAt == "" {
		t.Fatalf("%+v %v", got, err)
	}
	if b, err := c.Kubeconfig("prod"); err != nil || !bytes.Equal(b, kc) {
		t.Fatalf("%q %v", b, err)
	}
	// В БД лежит шифротекст, а не kubeconfig.
	var raw []byte
	_ = s.pool.QueryRow(context.Background(), `SELECT kubeconfig_enc FROM clusters WHERE id='prod'`).Scan(&raw)
	if len(raw) == 0 || bytes.Contains(raw, []byte("apiVersion")) {
		t.Fatal("kubeconfig хранится открытым текстом")
	}

	// Правка метаданных без kubeconfig оставляет его прежним.
	if err := c.Save(&Cluster{ID: "prod", DisplayName: "Prod EU", Environment: "prod"}, nil); err != nil {
		t.Fatal(err)
	}
	got, _ = c.Get("prod")
	if got.DisplayName != "Prod EU" || got.Environment != "prod" {
		t.Errorf("%+v", got)
	}
	if b, err := c.Kubeconfig("prod"); err != nil || !bytes.Equal(b, kc) {
		t.Errorf("kubeconfig потерян: %q %v", b, err)
	}
	// Замена kubeconfig.
	if err := c.Save(&Cluster{ID: "prod", DisplayName: "Prod EU"}, []byte("new")); err != nil {
		t.Fatal(err)
	}
	if b, _ := c.Kubeconfig("prod"); string(b) != "new" {
		t.Errorf("%q", b)
	}

	// Новый кластер без kubeconfig недопустим; одноимённый в той же команде — конфликт.
	if err := c.Save(&Cluster{ID: "x", DisplayName: "X"}, nil); err == nil {
		t.Error("кластер без kubeconfig должен отклоняться")
	}
	if err := c.Save(&Cluster{ID: "p2", DisplayName: "prod eu"}, kc); !errors.Is(err, ErrConflict) {
		t.Errorf("повтор имени: %v", err)
	}

	saveCluster(t, s, "dev", "Dev", kc)
	list, err := c.List()
	if err != nil || len(list) != 2 {
		t.Fatalf("%+v %v", list, err)
	}

	if _, err := c.Get("nope"); !errors.Is(err, ErrClusterNotFound) {
		t.Errorf("%v", err)
	}
	if _, err := c.Kubeconfig("nope"); !errors.Is(err, ErrClusterNotFound) {
		t.Errorf("%v", err)
	}
	if err := c.Delete("dev"); err != nil {
		t.Fatal(err)
	}
	if err := c.Delete("dev"); err != nil {
		t.Errorf("повторное удаление не ошибка: %v", err)
	}
	if list, _ = c.List(); len(list) != 1 {
		t.Errorf("%+v", list)
	}
}

func TestInClusterModeNeedsNoKubeconfig(t *testing.T) {
	s := newTestStore(t, keyString("k1"))
	if err := s.Clusters().Save(&Cluster{ID: "self", DisplayName: "Self", Mode: "in-cluster"}, nil); err != nil {
		t.Fatal(err)
	}
	if b, err := s.Clusters().Kubeconfig("self"); err != nil || b != nil {
		t.Errorf("%q %v", b, err)
	}
}

func TestArgoTokens(t *testing.T) {
	s := newTestStore(t, keyString("k1"))
	saveCluster(t, s, "a", "A", []byte("kc"))
	saveCluster(t, s, "b", "B", []byte("kc"))
	ts := s.ArgoTokens()
	if _, err := ts.Get("a"); !errors.Is(err, ext.ErrNotFound) {
		t.Fatalf("%v", err)
	}
	if err := ts.Put("a", "tok-a"); err != nil {
		t.Fatal(err)
	}
	if err := ts.Put("a", "tok-a2"); err != nil { // замена
		t.Fatal(err)
	}
	if v, err := ts.Get("a"); err != nil || v != "tok-a2" {
		t.Fatalf("%q %v", v, err)
	}
	if err := ts.Put("ghost", "x"); !errors.Is(err, ErrClusterNotFound) {
		t.Errorf("токен для несуществующего кластера: %v", err)
	}
	var raw []byte
	_ = s.pool.QueryRow(context.Background(), `SELECT token_enc FROM argo_tokens WHERE cluster_id='a'`).Scan(&raw)
	if bytes.Contains(raw, []byte("tok-a2")) {
		t.Fatal("токен хранится открытым текстом")
	}
	// Шифротекст, перенесённый в строку другого кластера, не расшифровывается (AAD).
	if err := ts.Put("b", "tok-b"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(context.Background(),
		`UPDATE argo_tokens SET token_enc=(SELECT token_enc FROM argo_tokens WHERE cluster_id='a') WHERE cluster_id='b'`); err != nil {
		t.Fatal(err)
	}
	if _, err := ts.Get("b"); !errors.Is(err, secretbox.ErrDecrypt) {
		t.Errorf("подмена между строками: %v", err)
	}
	// Удаление кластера удаляет токен; Delete токена идемпотентен.
	_ = s.Clusters().Delete("a")
	if _, err := ts.Get("a"); !errors.Is(err, ext.ErrNotFound) {
		t.Errorf("%v", err)
	}
	if err := ts.Delete("a"); err != nil {
		t.Error(err)
	}
}

func TestKeyRotationAcrossStores(t *testing.T) {
	oldK, newK := keyString("2026-01"), keyString("2026-07")
	s := newTestStore(t, oldK)
	saveCluster(t, s, "a", "A", []byte("kc"))
	_ = s.ArgoTokens().Put("a", "tok")

	url := os.Getenv("KUBRON_TEST_DATABASE_URL")
	open := func(cur string, prev ...string) *Store {
		keys, err := secretbox.ParseKeyring(cur, prev...)
		if err != nil {
			t.Fatal(err)
		}
		n, err := Open(context.Background(), url, keys)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(n.Close)
		if err := n.Init(context.Background()); err != nil {
			t.Fatal(err)
		}
		return n
	}
	withBoth := open(newK, oldK)
	if b, err := withBoth.Clusters().Kubeconfig("a"); err != nil || string(b) != "kc" {
		t.Fatalf("%q %v", b, err)
	}
	if v, err := withBoth.ArgoTokens().Get("a"); err != nil || v != "tok" {
		t.Fatalf("%q %v", v, err)
	}
	// Новая запись шифруется новым ключом.
	if err := withBoth.Clusters().Save(&Cluster{ID: "a", DisplayName: "A"}, []byte("kc2")); err != nil {
		t.Fatal(err)
	}
	var kid string
	_ = withBoth.pool.QueryRow(context.Background(), `SELECT key_id FROM clusters WHERE id='a'`).Scan(&kid)
	if kid != "2026-07" {
		t.Errorf("key_id=%q", kid)
	}
	// Без прежнего ключа токен, зашифрованный им, не читается.
	if _, err := open(newK).ArgoTokens().Get("a"); !errors.Is(err, secretbox.ErrUnknownKey) {
		t.Errorf("%v", err)
	}
}
