// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func init() {
	pbkdf2Iterations = 1000 // тесты не должны ждать полноценного хеширования
}

const goodPass = "correct horse"

func newTestService(t *testing.T) (*Service, string) {
	t.Helper()
	dir := t.TempDir()
	s, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}
	return s, dir
}

// setupWithUser: администратор admin + обычный пользователь bob (пароль уже сменён).
func setupWithUser(t *testing.T) (*Service, *UserInfo) {
	t.Helper()
	s, _ := newTestService(t)
	if _, err := s.Setup("admin", goodPass); err != nil {
		t.Fatal(err)
	}
	bob, err := s.CreateUser("bob", "temp-pass-1", RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	s.Logout()
	if _, err := s.Login("bob", "temp-pass-1"); err != nil {
		t.Fatal(err)
	}
	if err := s.ChangePassword("temp-pass-1", "bobs-own-pass"); err != nil {
		t.Fatal(err)
	}
	s.Logout()
	return s, bob
}

func loginAdmin(t *testing.T, s *Service) {
	t.Helper()
	s.Logout()
	if _, err := s.Login("admin", goodPass); err != nil {
		t.Fatal(err)
	}
}

func TestFirstRunNeedsSetup(t *testing.T) {
	s, _ := newTestService(t)
	st := s.Status()
	if !st.NeedsSetup || st.User != nil {
		t.Fatalf("ожидалась первичная настройка, получено %+v", st)
	}
	if _, err := s.Login("admin", goodPass); !errors.Is(err, ErrSetupRequired) {
		t.Fatalf("вход до настройки: %v", err)
	}
	if err := s.Authorize(PermRead); !errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("без входа доступ должен быть закрыт, получено %v", err)
	}
}

func TestSetupCreatesAdminOnlyOnce(t *testing.T) {
	s, _ := newTestService(t)
	u, err := s.Setup("admin", goodPass)
	if err != nil {
		t.Fatal(err)
	}
	if u.Role != RoleAdmin || u.MustChangePassword {
		t.Fatalf("неверный первый пользователь: %+v", u)
	}
	if err := s.Authorize(PermUsersManage); err != nil {
		t.Fatalf("после настройки админ должен быть вошёл: %v", err)
	}
	if _, err := s.Setup("evil", goodPass); !errors.Is(err, ErrSetupDone) {
		t.Fatalf("повторная настройка должна быть запрещена: %v", err)
	}
}

func TestSetupValidation(t *testing.T) {
	s, _ := newTestService(t)
	cases := []struct{ user, pass string }{
		{"ab", goodPass},                   // короткий логин
		{"bad name", goodPass},             // пробел
		{"admin", "short"},                 // короткий пароль
		{"admin", "ADMIN"},                 // слишком короткий
		{"administrator", "ADMINISTRATOR"}, // совпадает с логином
	}
	for _, c := range cases {
		if _, err := s.Setup(c.user, c.pass); err == nil {
			t.Errorf("Setup(%q, %q) должен был вернуть ошибку", c.user, c.pass)
		}
	}
	if !s.Status().NeedsSetup {
		t.Error("неудачные попытки не должны создавать пользователей")
	}
}

func TestLoginLogout(t *testing.T) {
	s, _ := setupWithUser(t)
	if _, err := s.Login("BOB", "bobs-own-pass"); err != nil { // логин регистронезависим
		t.Fatalf("вход: %v", err)
	}
	if err := s.Authorize(PermRead); err != nil {
		t.Errorf("read: %v", err)
	}
	s.Logout()
	if err := s.Authorize(PermRead); !errors.Is(err, ErrNotAuthenticated) {
		t.Errorf("после выхода доступ должен закрыться, получено %v", err)
	}
}

func TestLoginErrorsDoNotRevealUser(t *testing.T) {
	s, _ := setupWithUser(t)
	_, e1 := s.Login("bob", "wrong-password")
	_, e2 := s.Login("nobody", "wrong-password")
	if !errors.Is(e1, ErrInvalidCredentials) || !errors.Is(e2, ErrInvalidCredentials) {
		t.Fatalf("ожидалась одна и та же ошибка: %v / %v", e1, e2)
	}
}

func TestUserRoleEnforcement(t *testing.T) {
	s, _ := setupWithUser(t)
	if _, err := s.Login("bob", "bobs-own-pass"); err != nil {
		t.Fatal(err)
	}
	for _, p := range []Permission{PermRead, PermSecretsRead} {
		if err := s.Authorize(p); err != nil {
			t.Errorf("user должен иметь %q: %v", p, err)
		}
	}
	for _, p := range []Permission{PermWrite, PermExec, PermPortForward, PermClusterManage, PermUsersManage} {
		if err := s.Authorize(p); !errors.Is(err, ErrForbidden) {
			t.Errorf("user не должен иметь %q, получено %v", p, err)
		}
	}
	if _, err := s.ListUsers(); !errors.Is(err, ErrForbidden) {
		t.Errorf("ListUsers для user: %v", err)
	}
	if _, err := s.CreateUser("eve", goodPass, RoleAdmin); !errors.Is(err, ErrForbidden) {
		t.Errorf("CreateUser для user: %v", err)
	}
}

func TestAdminCanDoEverything(t *testing.T) {
	s, _ := newTestService(t)
	if _, err := s.Setup("admin", goodPass); err != nil {
		t.Fatal(err)
	}
	for _, p := range allPermissions {
		if err := s.Authorize(p); err != nil {
			t.Errorf("admin должен иметь %q: %v", p, err)
		}
	}
}

func TestNewUserMustChangePassword(t *testing.T) {
	s, _ := newTestService(t)
	if _, err := s.Setup("admin", goodPass); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateUser("bob", "temp-pass-1", RoleUser); err != nil {
		t.Fatal(err)
	}
	s.Logout()
	u, err := s.Login("bob", "temp-pass-1")
	if err != nil {
		t.Fatal(err)
	}
	if !u.MustChangePassword {
		t.Fatal("новый пользователь обязан сменить временный пароль")
	}
	if err := s.Authorize(PermRead); !errors.Is(err, ErrPasswordChangeRequired) {
		t.Fatalf("до смены пароля доступ закрыт: %v", err)
	}
	if err := s.ChangePassword("temp-pass-1", "temp-pass-1"); err == nil {
		t.Fatal("новый пароль не должен совпадать со старым")
	}
	if err := s.ChangePassword("nope-nope-1", "fresh-pass-1"); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("неверный текущий пароль: %v", err)
	}
	if err := s.ChangePassword("temp-pass-1", "fresh-pass-1"); err != nil {
		t.Fatal(err)
	}
	if err := s.Authorize(PermRead); err != nil {
		t.Fatalf("после смены пароля доступ открыт: %v", err)
	}
	s.Logout()
	if _, err := s.Login("bob", "temp-pass-1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatal("старый пароль должен перестать работать")
	}
	if _, err := s.Login("bob", "fresh-pass-1"); err != nil {
		t.Fatal(err)
	}
}

func TestLockoutAfterRepeatedFailures(t *testing.T) {
	s, _ := setupWithUser(t)
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return clock }

	for i := 0; i < freeAttempts; i++ {
		if _, err := s.Login("bob", "wrong-password"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("попытка %d: %v", i, err)
		}
	}
	_, err := s.Login("bob", "bobs-own-pass") // даже верный пароль во время блокировки
	var locked *LockedError
	if !errors.As(err, &locked) {
		t.Fatalf("ожидалась блокировка, получено %v", err)
	}
	if !strings.Contains(locked.Error(), "повторите через") {
		t.Errorf("текст ошибки: %q", locked.Error())
	}

	clock = clock.Add(baseLockout + time.Second)
	if _, err := s.Login("bob", "bobs-own-pass"); err != nil {
		t.Fatalf("после окончания блокировки вход должен работать: %v", err)
	}
	// успешный вход сбрасывает счётчик
	s.Logout()
	for i := 0; i < freeAttempts-1; i++ {
		_, _ = s.Login("bob", "wrong-password")
	}
	if _, err := s.Login("bob", "bobs-own-pass"); err != nil {
		t.Fatalf("счётчик должен был сброситься: %v", err)
	}
}

func TestLastAdminProtection(t *testing.T) {
	s, _ := setupWithUser(t)
	loginAdmin(t, s)
	users, err := s.ListUsers()
	if err != nil {
		t.Fatal(err)
	}
	var adminID string
	for _, u := range users {
		if u.Username == "admin" {
			adminID = u.ID
		}
	}
	if _, err := s.UpdateRole(adminID, RoleUser); !errors.Is(err, ErrSelfAction) {
		t.Errorf("смена собственной роли: %v", err)
	}
	if _, err := s.SetDisabled(adminID, true); !errors.Is(err, ErrSelfAction) {
		t.Errorf("отключение самого себя: %v", err)
	}
	if err := s.DeleteUser(adminID); !errors.Is(err, ErrSelfAction) {
		t.Errorf("удаление самого себя: %v", err)
	}
}

func TestPromoteThenDemoteAdmin(t *testing.T) {
	s, bob := setupWithUser(t)
	loginAdmin(t, s)
	if _, err := s.UpdateRole(bob.ID, RoleAdmin); err != nil {
		t.Fatal(err)
	}
	// теперь админов двое: второго можно разжаловать, а последнего (после этого) — нет
	if _, err := s.UpdateRole(bob.ID, RoleUser); err != nil {
		t.Fatalf("разжалование при двух админах: %v", err)
	}
	if _, err := s.UpdateRole(bob.ID, Role("root")); !errors.Is(err, ErrUnknownRole) {
		t.Errorf("неизвестная роль: %v", err)
	}
}

func TestAdminCannotChangeOwnRoleEvenWithOtherAdmin(t *testing.T) {
	s, bob := setupWithUser(t)
	loginAdmin(t, s)
	admin, err := s.Current()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateRole(bob.ID, RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateRole(admin.ID, RoleUser); !errors.Is(err, ErrSelfAction) {
		t.Errorf("самопонижение при двух админах: %v", err)
	}
	// «смена» на ту же роль — не изменение, ошибки быть не должно
	if _, err := s.UpdateRole(admin.ID, RoleAdmin); err != nil {
		t.Errorf("та же роль: %v", err)
	}
}

func TestRoleChangeAppliesImmediately(t *testing.T) {
	s, bob := setupWithUser(t)
	// bob входит, затем админ (под другим процессом/сессией быть не может), поэтому моделируем напрямую
	if _, err := s.Login("bob", "bobs-own-pass"); err != nil {
		t.Fatal(err)
	}
	if err := s.Authorize(PermWrite); !errors.Is(err, ErrForbidden) {
		t.Fatal("bob не должен иметь write")
	}
	s.mu.Lock()
	s.st.findByID(bob.ID).Role = RoleAdmin
	s.mu.Unlock()
	if err := s.Authorize(PermWrite); err != nil {
		t.Errorf("новая роль должна применяться без повторного входа: %v", err)
	}
	s.mu.Lock()
	s.st.findByID(bob.ID).Disabled = true
	s.mu.Unlock()
	if err := s.Authorize(PermRead); !errors.Is(err, ErrNotAuthenticated) {
		t.Errorf("отключённый пользователь теряет сессию: %v", err)
	}
}

func TestDisabledUserCannotLogin(t *testing.T) {
	s, bob := setupWithUser(t)
	loginAdmin(t, s)
	if _, err := s.SetDisabled(bob.ID, true); err != nil {
		t.Fatal(err)
	}
	s.Logout()
	if _, err := s.Login("bob", "bobs-own-pass"); !errors.Is(err, ErrAccountDisabled) {
		t.Fatalf("отключённый вход: %v", err)
	}
	loginAdmin(t, s)
	if _, err := s.SetDisabled(bob.ID, false); err != nil {
		t.Fatal(err)
	}
	s.Logout()
	if _, err := s.Login("bob", "bobs-own-pass"); err != nil {
		t.Fatalf("после включения вход возможен: %v", err)
	}
}

func TestAdminResetPassword(t *testing.T) {
	s, bob := setupWithUser(t)
	loginAdmin(t, s)
	if _, err := s.ResetPassword(bob.ID, "reset-pass-9"); err != nil {
		t.Fatal(err)
	}
	s.Logout()
	if _, err := s.Login("bob", "bobs-own-pass"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatal("старый пароль должен быть недействителен")
	}
	u, err := s.Login("bob", "reset-pass-9")
	if err != nil || !u.MustChangePassword {
		t.Fatalf("после сброса нужна обязательная смена пароля: %v %+v", err, u)
	}
	loginAdmin(t, s)
	users, _ := s.ListUsers()
	for _, u := range users {
		if u.Username == "admin" {
			if _, err := s.ResetPassword(u.ID, "whatever-123"); !errors.Is(err, ErrSelfAction) {
				t.Errorf("сброс собственного пароля: %v", err)
			}
		}
	}
}

func TestDeleteUser(t *testing.T) {
	s, bob := setupWithUser(t)
	loginAdmin(t, s)
	if err := s.DeleteUser(bob.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteUser(bob.ID); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("повторное удаление: %v", err)
	}
	s.Logout()
	if _, err := s.Login("bob", "bobs-own-pass"); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("удалённый пользователь не может войти: %v", err)
	}
}

func TestDuplicateUsernameIsCaseInsensitive(t *testing.T) {
	s, _ := setupWithUser(t)
	loginAdmin(t, s)
	if _, err := s.CreateUser("BOB", goodPass, RoleUser); !errors.Is(err, ErrUserExists) {
		t.Fatalf("дубликат логина: %v", err)
	}
	if _, err := s.CreateUser("eve", goodPass, Role("root")); !errors.Is(err, ErrUnknownRole) {
		t.Fatalf("неизвестная роль: %v", err)
	}
}

func TestPersistenceAndFileSafety(t *testing.T) {
	s, dir := newTestService(t)
	if _, err := s.Setup("admin", goodPass); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, usersFileName)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), goodPass) {
		t.Fatal("пароль не должен храниться в открытом виде")
	}
	if !strings.Contains(string(raw), hashScheme+"$") {
		t.Fatal("в файле должен быть хеш пароля")
	}
	if fi, err := os.Stat(path); err == nil && os.PathSeparator == '/' && fi.Mode().Perm() != 0o600 {
		t.Errorf("права файла %v, ожидалось 0600", fi.Mode().Perm())
	}
	if leftovers, _ := filepath.Glob(filepath.Join(dir, "*.tmp")); len(leftovers) != 0 {
		t.Errorf("остались временные файлы: %v", leftovers)
	}

	// новый экземпляр видит того же пользователя, но сессия не сохраняется
	s2, err := NewService(dir)
	if err != nil {
		t.Fatal(err)
	}
	st := s2.Status()
	if st.NeedsSetup || st.User != nil {
		t.Fatalf("после перезапуска нужен вход: %+v", st)
	}
	if _, err := s2.Login("admin", goodPass); err != nil {
		t.Fatal(err)
	}
}

func TestCorruptFileDoesNotReopenSetup(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, usersFileName), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewService(dir); err == nil {
		t.Fatal("повреждённый файл должен приводить к ошибке, а не к пустому списку пользователей")
	}
}

func TestNewerFileVersionRejected(t *testing.T) {
	dir := t.TempDir()
	data, _ := json.Marshal(fileData{Version: storageVersion + 1})
	if err := os.WriteFile(filepath.Join(dir, usersFileName), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewService(dir); err == nil {
		t.Fatal("файл более новой версии должен отвергаться")
	}
}

func TestUserInfoHasNoSecrets(t *testing.T) {
	s, _ := newTestService(t)
	u, err := s.Setup("admin", goodPass)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(u)
	if strings.Contains(strings.ToLower(string(b)), "hash") || strings.Contains(string(b), goodPass) {
		t.Fatalf("UserInfo не должен содержать хеш или пароль: %s", b)
	}
}

func TestLogoutHook(t *testing.T) {
	s, _ := newTestService(t)
	calls := 0
	s.SetOnLogout(func() { calls++ })
	if _, err := s.Setup("admin", goodPass); err != nil {
		t.Fatal(err)
	}
	s.Logout()
	s.Logout() // повторный выход без сессии не вызывает обработчик
	if calls != 1 {
		t.Fatalf("обработчик вызван %d раз, ожидался 1", calls)
	}
}

func TestPasswordHashing(t *testing.T) {
	h, err := hashPassword(goodPass)
	if err != nil {
		t.Fatal(err)
	}
	h2, _ := hashPassword(goodPass)
	if h == h2 {
		t.Error("соль должна быть случайной")
	}
	if ok, _ := verifyPassword(goodPass, h); !ok {
		t.Error("верный пароль не прошёл проверку")
	}
	if ok, _ := verifyPassword(goodPass+"x", h); ok {
		t.Error("неверный пароль прошёл проверку")
	}
	for _, bad := range []string{"", "x", "a$b$c$d", "pbkdf2-sha256$0$AA$AA", "pbkdf2-sha256$99999999999$AA$AA", "pbkdf2-sha256$1000$!!$!!"} {
		if ok, _ := verifyPassword(goodPass, bad); ok {
			t.Errorf("мусорный хеш %q принят", bad)
		}
	}
}

func TestRehashOnOutdatedParameters(t *testing.T) {
	old := pbkdf2Iterations
	pbkdf2Iterations = 500
	h, _ := hashPassword(goodPass)
	pbkdf2Iterations = old
	ok, rehash := verifyPassword(goodPass, h)
	if !ok || !rehash {
		t.Fatalf("ok=%v rehash=%v: устаревший хеш должен подлежать пересчёту", ok, rehash)
	}
	if _, rehash := verifyPassword(goodPass, mustHash(t, goodPass)); rehash {
		t.Error("свежий хеш не должен пересчитываться")
	}
}

func mustHash(t *testing.T, p string) string {
	t.Helper()
	h, err := hashPassword(p)
	if err != nil {
		t.Fatal(err)
	}
	return h
}
