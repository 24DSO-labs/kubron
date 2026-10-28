// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"errors"
	"github.com/VadimIl224/kubron/internal/audit"
	"github.com/VadimIl224/kubron/internal/auth"
	"testing"
)

// Методы-пробы: authz берёт имя действия из имени вызывающего метода, как в настоящих методах App.
func (a *App) probeWrite(ns, name string) error {
	return a.authz(auth.PermWrite, "namespace", ns, "name", name)
}

func (a *App) probeRead() error { return a.authz(auth.PermRead) }

const testPass = "correct-horse-1"

func newAuditedApp(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	a := &App{}
	a.initAuth(dir) // k8sMgr/pfMgr не нужны, пока не вызывается выход из сессии с хуком
	if a.authErr != nil {
		t.Fatal(a.authErr)
	}
	return a
}

func events(t *testing.T, a *App) []audit.Event {
	t.Helper()
	p, err := a.audit.Query(audit.Filter{Limit: audit.MaxLimit})
	if err != nil {
		t.Fatal(err)
	}
	return p.Events // от новых к старым
}

func TestAuthzWritesAuditTrail(t *testing.T) {
	a := newAuditedApp(t)
	a.auth.SetOnLogout(nil) // в тесте нет менеджеров кластеров

	// Без входа: отказ по аутентификации — это не событие привилегий, журнал не засоряется.
	if err := a.probeWrite("prod", "api"); !errors.Is(err, auth.ErrNotAuthenticated) {
		t.Fatalf("без входа: %v", err)
	}
	if n := len(events(t, a)); n != 0 {
		t.Fatalf("до входа в журнале %d записей, ожидалось 0", n)
	}

	if _, err := a.auth.Setup("admin", testPass); err != nil {
		t.Fatal(err)
	}
	if err := a.probeWrite("prod", "api"); err != nil {
		t.Fatalf("admin: %v", err)
	}
	if err := a.probeRead(); err != nil {
		t.Fatalf("admin read: %v", err)
	}

	ev := events(t, a)
	if len(ev) != 2 || ev[1].Action != "Setup" { // Setup + probeWrite; чтение не журналируется
		t.Fatalf("ожидались Setup и probeWrite, получено %+v", ev)
	}
	got := ev[0]
	if got.Action != "probeWrite" || got.Result != audit.ResultOK || got.User != "admin" || got.Role != "admin" ||
		got.Detail != "namespace=prod name=api" {
		t.Errorf("неверная запись о действии: %+v", got)
	}

	// Обычный пользователь: запись отклоняется и журналируется, чтение — нет.
	if _, err := a.auth.CreateUser("bob", testPass, auth.RoleUser); err != nil {
		t.Fatal(err)
	}
	a.auth.Logout()
	if _, err := a.auth.Login("bob", testPass); err != nil {
		t.Fatal(err)
	}
	if err := a.auth.ChangePassword(testPass, "another-pass-2"); err != nil {
		t.Fatal(err)
	}
	before := len(events(t, a))
	if err := a.probeRead(); err != nil {
		t.Fatalf("user read: %v", err)
	}
	if err := a.probeWrite("prod", "api"); !errors.Is(err, auth.ErrForbidden) {
		t.Fatalf("user write: %v", err)
	}
	ev = events(t, a)
	if len(ev) != before+1 {
		t.Fatalf("ожидалась ровно одна новая запись (отказ), получено %d", len(ev)-before)
	}
	got = ev[0]
	if got.Result != audit.ResultDenied || got.User != "bob" || got.Role != "user" ||
		got.Detail != "namespace=prod name=api need=write" {
		t.Errorf("неверная запись об отказе: %+v", got)
	}
}

func TestUserManagementIsAuditedOnceWithOutcome(t *testing.T) {
	a := newAuditedApp(t)
	a.auth.SetOnLogout(nil)
	if _, err := a.auth.Setup("admin", testPass); err != nil {
		t.Fatal(err)
	}
	if _, err := a.CreateUser("bob", testPass, "user"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.CreateUser("bob", testPass, "user"); err == nil {
		t.Fatal("дубликат логина должен отклоняться")
	}
	var ok, failed int
	for _, e := range events(t, a) {
		if e.Action != "CreateUser" {
			continue
		}
		switch e.Result {
		case audit.ResultOK:
			ok++
		case audit.ResultFailed:
			failed++
		}
	}
	if ok != 1 || failed != 1 {
		t.Errorf("CreateUser: ok=%d failed=%d, ожидалось по одной записи", ok, failed)
	}
}

func TestAuditLogRequiresPermission(t *testing.T) {
	a := newAuditedApp(t)
	a.auth.SetOnLogout(nil)
	if _, err := a.auth.Setup("admin", testPass); err != nil {
		t.Fatal(err)
	}
	if _, err := a.GetAuditLog(audit.Filter{}); err != nil {
		t.Fatalf("admin: %v", err)
	}
	if _, err := a.auth.CreateUser("bob", testPass, auth.RoleUser); err != nil {
		t.Fatal(err)
	}
	a.auth.Logout()
	if _, err := a.auth.Login("bob", testPass); err != nil {
		t.Fatal(err)
	}
	if err := a.auth.ChangePassword(testPass, "another-pass-2"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.GetAuditLog(audit.Filter{}); !errors.Is(err, auth.ErrForbidden) {
		t.Errorf("user не должен читать журнал: %v", err)
	}
}

func TestFailedLoginIsAudited(t *testing.T) {
	a := newAuditedApp(t)
	a.auth.SetOnLogout(nil)
	if _, err := a.auth.Setup("admin", testPass); err != nil {
		t.Fatal(err)
	}
	a.auth.Logout()
	if _, err := a.auth.Login("admin", "wrong-password"); err == nil {
		t.Fatal("неверный пароль должен отклоняться")
	}
	e := events(t, a)[0]
	if e.Action != "Login" || e.Result != audit.ResultFailed || e.User != "admin" {
		t.Errorf("неудачный вход не записан: %+v", e)
	}
}
