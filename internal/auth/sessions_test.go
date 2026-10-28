// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"errors"
	"github.com/VadimIl224/kubron/internal/audit"
	"sync/atomic"
	"testing"
	"time"
)

const bobPass = "bobs-own-pass"

// counted подключает к виду счётчик вызовов обработчика очистки.
func counted(v *Service) *int32 {
	var n int32
	v.SetOnLogout(func() { atomic.AddInt32(&n, 1) })
	return &n
}

func loginAs(t *testing.T, v *Service, user, pass string) {
	t.Helper()
	if _, err := v.Login(user, pass); err != nil {
		t.Fatalf("вход %s: %v", user, err)
	}
}

func TestSessionsAreIndependent(t *testing.T) {
	s, _ := setupWithUser(t)
	loginAdmin(t, s)
	v := s.NewSession("10.0.0.1")
	loginAs(t, v, "bob", bobPass)

	if u, _ := s.Current(); u == nil || u.Username != "admin" {
		t.Fatalf("первая сессия: %+v", u)
	}
	if u, _ := v.Current(); u == nil || u.Username != "bob" {
		t.Fatalf("вторая сессия: %+v", u)
	}
	v.Logout()
	if _, err := v.Current(); !errors.Is(err, ErrNotAuthenticated) {
		t.Errorf("после выхода: %v", err)
	}
	if u, err := s.Current(); err != nil || u.Username != "admin" {
		t.Errorf("выход второй сессии затронул первую: %v %+v", err, u)
	}
	// Анонимный вид не видит чужих пользователей
	anon := s.NewSession("10.0.0.2")
	if st := anon.Status(); st.User != nil {
		t.Errorf("аноним видит пользователя: %+v", st.User)
	}
}

func TestPermissionsArePerSession(t *testing.T) {
	s, _ := setupWithUser(t)
	loginAdmin(t, s)
	v := s.NewSession("")
	loginAs(t, v, "bob", bobPass)
	if _, err := s.ListUsers(); err != nil {
		t.Errorf("админ: %v", err)
	}
	if _, err := v.ListUsers(); !errors.Is(err, ErrForbidden) {
		t.Errorf("обычный пользователь: %v", err)
	}
}

func TestLogoutHookIsPerSession(t *testing.T) {
	s, _ := setupWithUser(t)
	loginAdmin(t, s)
	n1 := counted(s)
	v := s.NewSession("")
	n2 := counted(v)
	loginAs(t, v, "bob", bobPass)
	v.Logout()
	if *n1 != 0 || atomic.LoadInt32(n2) != 1 {
		t.Errorf("обработчики: первая=%d вторая=%d", *n1, atomic.LoadInt32(n2))
	}
}

func TestAdminChangesEndTargetSessions(t *testing.T) {
	cases := []struct {
		name string
		act  func(s *Service, bobID string) error
	}{
		{"отключение", func(s *Service, id string) error { _, err := s.SetDisabled(id, true); return err }},
		{"смена роли", func(s *Service, id string) error { _, err := s.UpdateRole(id, RoleOperator); return err }},
		{"новый пароль", func(s *Service, id string) error { _, err := s.ResetPassword(id, "temp-pass-2"); return err }},
		{"удаление", func(s *Service, id string) error { return s.DeleteUser(id) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, bob := setupWithUser(t)
			loginAdmin(t, s)
			v1, v2 := s.NewSession(""), s.NewSession("")
			n1, n2 := counted(v1), counted(v2)
			loginAs(t, v1, "bob", bobPass)
			loginAs(t, v2, "bob", bobPass)
			if err := c.act(s, bob.ID); err != nil {
				t.Fatal(err)
			}
			for i, v := range []*Service{v1, v2} {
				if _, err := v.Current(); !errors.Is(err, ErrNotAuthenticated) {
					t.Errorf("сессия %d жива: %v", i+1, err)
				}
			}
			if atomic.LoadInt32(n1) != 1 || atomic.LoadInt32(n2) != 1 {
				t.Errorf("очистка: %d, %d", *n1, *n2)
			}
			if u, err := s.Current(); err != nil || u.Username != "admin" {
				t.Errorf("сессия администратора пострадала: %v", err)
			}
		})
	}
}

func TestChangePasswordEndsOtherSessionsOnly(t *testing.T) {
	s, _ := setupWithUser(t)
	v1, v2 := s.NewSession(""), s.NewSession("")
	n1, n2 := counted(v1), counted(v2)
	loginAs(t, v1, "bob", bobPass)
	loginAs(t, v2, "bob", bobPass)
	if err := v1.ChangePassword(bobPass, "another-pass-9"); err != nil {
		t.Fatal(err)
	}
	if _, err := v1.Current(); err != nil {
		t.Errorf("текущая сессия должна жить: %v", err)
	}
	if _, err := v2.Current(); !errors.Is(err, ErrNotAuthenticated) {
		t.Errorf("другая сессия должна завершиться: %v", err)
	}
	if atomic.LoadInt32(n1) != 0 || atomic.LoadInt32(n2) != 1 {
		t.Errorf("очистка: %d, %d", *n1, *n2)
	}
}

func TestHooksRunOutsideTheLock(t *testing.T) {
	s, bob := setupWithUser(t)
	loginAdmin(t, s)
	v := s.NewSession("")
	done := make(chan struct{})
	v.SetOnLogout(func() { v.Status(); close(done) }) // обращение к Service из обработчика
	loginAs(t, v, "bob", bobPass)
	finished := make(chan struct{})
	go func() { _, _ = s.SetDisabled(bob.ID, true); close(finished) }()
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("взаимная блокировка: обработчик вызывается под мьютексом")
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("обработчик не вызван")
	}
}

func TestExpireAllIdleHandlesEachSession(t *testing.T) {
	s, _ := setupWithUser(t)
	clock := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return clock }
	loginAdmin(t, s)
	v := s.NewSession("")
	nv := counted(v)
	loginAs(t, v, "bob", bobPass)

	clock = clock.Add(time.Duration(DefaultIdleTimeoutMinutes)*time.Minute - time.Minute)
	if err := s.ReportActivity(); err != nil { // админ работает, bob — нет
		t.Fatal(err)
	}
	clock = clock.Add(2*time.Minute + idleGrace)
	if n := s.ExpireAllIdle(); n != 1 {
		t.Fatalf("завершено %d, ждали 1", n)
	}
	if _, err := v.Current(); !errors.Is(err, ErrNotAuthenticated) {
		t.Errorf("простаивающая сессия жива: %v", err)
	}
	if _, err := s.Current(); err != nil {
		t.Errorf("активная сессия завершена: %v", err)
	}
	time.Sleep(20 * time.Millisecond)
	if atomic.LoadInt32(nv) != 1 {
		t.Errorf("очистка простоя не вызвана: %d", atomic.LoadInt32(nv))
	}
}

func TestReleaseEndsAndUnregistersSession(t *testing.T) {
	s, _ := setupWithUser(t)
	v := s.NewSession("")
	n := counted(v)
	loginAs(t, v, "bob", bobPass)
	before := len(s.sessions)
	v.Release()
	v.Release() // повторный вызов безопасен
	if atomic.LoadInt32(n) != 1 || len(s.sessions) != before-1 {
		t.Errorf("очистка=%d, сессий %d→%d", atomic.LoadInt32(n), before, len(s.sessions))
	}
	anon := s.NewSession("")
	nn := counted(anon)
	anon.Release()
	if atomic.LoadInt32(nn) != 0 {
		t.Error("очистка анонимной сессии не нужна")
	}
}

func TestSourceLockoutAcrossUsernames(t *testing.T) {
	s, _ := setupWithUser(t)
	clock := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return clock }
	attacker := s.NewSession("203.0.113.7")
	for i := 0; i < sourceFreeAttempts; i++ { // разные логины: по логину блокировки нет
		_, _ = attacker.Login("user"+string(rune('a'+i%26))+string(rune('A'+i/26)), "x")
	}
	var locked *LockedError
	if _, err := attacker.Login("bob", bobPass); !errors.As(err, &locked) {
		t.Fatalf("ждали блокировку по адресу даже с верным паролем: %v", err)
	}
	other := s.NewSession("198.51.100.9")
	if _, err := other.Login("bob", bobPass); err != nil {
		t.Errorf("другой адрес не должен страдать: %v", err)
	}
	clock = clock.Add(sourceWindow + time.Minute) // окно прошло — счётчик забыт
	if _, err := attacker.Login("bob", bobPass); err != nil {
		t.Errorf("после окна вход должен работать: %v", err)
	}
}

func TestEmptySourceHasNoAddressLockout(t *testing.T) {
	s, _ := setupWithUser(t)
	v := s.NewSession("")
	for i := 0; i < sourceFreeAttempts+5; i++ {
		_, _ = v.Login("name"+string(rune('a'+i%26))+string(rune('A'+i/26)), "x")
	}
	if _, err := v.Login("bob", bobPass); err != nil {
		t.Errorf("без адреса блокировки по адресу быть не должно: %v", err)
	}
}

// Два пользователя с разными ролями на разных сессиях: один и тот же вызов даёт разные ответы,
// а журнал приписывает действие реальному пользователю и адресу его сессии.
func TestAuditAttributesActionsToSessionUserAndSource(t *testing.T) {
	root, _ := newTestService(t)
	rec := &memRecorder{}
	root.SetAuditor(rec)
	adm := root.NewSession("10.0.0.1")
	if _, err := adm.Setup("admin", goodPass); err != nil {
		t.Fatal(err)
	}
	if _, err := adm.CreateUser("bob", "temp-pass-1", RoleUser); err != nil {
		t.Fatal(err)
	}
	bob := root.NewSession("10.0.0.2")
	if _, err := bob.Login("bob", "temp-pass-1"); err != nil {
		t.Fatal(err)
	}
	if err := bob.ChangePassword("temp-pass-1", "bobs-own-pass"); err != nil {
		t.Fatal(err)
	}

	if a, err := adm.Check(PermWrite); err != nil || a.Username != "admin" || a.Source != "10.0.0.1" {
		t.Fatalf("admin: %+v %v", a, err)
	}
	a, err := bob.Check(PermClusterManage)
	if !errors.Is(err, ErrForbidden) || a.Username != "bob" || a.Source != "10.0.0.2" {
		t.Fatalf("bob должен получить отказ от своего имени: %+v %v", a, err)
	}

	// Отключение bob администратором: событие о сессии bob несёт адрес сессии bob, а не администратора.
	rec.mu.Lock()
	rec.ev = nil
	rec.mu.Unlock()
	users, _ := adm.ListUsers()
	var bobID string
	for _, u := range users {
		if u.Username == "bob" {
			bobID = u.ID
		}
	}
	if _, err := adm.SetDisabled(bobID, true); err != nil {
		t.Fatal(err)
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	var end *audit.Event
	for i := range rec.ev {
		if rec.ev[i].Action == "SessionEnd" {
			end = &rec.ev[i]
		}
	}
	if end == nil || end.Source != "10.0.0.2" {
		t.Errorf("SessionEnd должен нести адрес завершённой сессии: %+v", end)
	}
}
