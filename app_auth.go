package main

import (
	"analog/internal/auth"
	"errors"
	"fmt"
	"strings"
)

// initAuth открывает хранилище пользователей. Если файл повреждён, приложение остаётся
// закрытым (authz отказывает всем), а причина показывается на экране входа.
func (a *App) initAuth(dir string) {
	svc, err := auth.NewService(dir)
	if err != nil {
		a.authErr = err
		return
	}
	// При выходе из учётной записи ничего из сессии не должно продолжать работать.
	svc.SetOnLogout(func() {
		a.pfMgr.StopAll()
		a.k8sMgr.CloseAllTerminals()
	})
	a.auth = svc
}

// authz — единственная точка проверки прав в Wails-методах.
// Каждый экспортируемый метод App обязан начинаться с её вызова; это проверяет
// тест internal/auth/policy_test.go.
func (a *App) authz(p auth.Permission) error {
	if a.auth == nil {
		return a.authUnavailable()
	}
	return a.auth.Authorize(p)
}

func (a *App) authUnavailable() error {
	if a.authErr != nil {
		return fmt.Errorf("подсистема авторизации недоступна: %w", a.authErr)
	}
	return errors.New("подсистема авторизации не инициализирована")
}

// ─── Методы, доступные без права доступа ───────────────────
// Это единственные экспортируемые методы, которые не вызывают authz (см. policy_test.go).

// AuthStatus — стартовое состояние: нужна ли первичная настройка и кто вошёл.
func (a *App) AuthStatus() (*auth.Status, error) {
	if a.auth == nil {
		return nil, a.authUnavailable()
	}
	st := a.auth.Status()
	return &st, nil
}

// SetupAdmin создаёт первого администратора (только при пустом списке пользователей).
func (a *App) SetupAdmin(username, password string) (*auth.UserInfo, error) {
	if a.auth == nil {
		return nil, a.authUnavailable()
	}
	return a.auth.Setup(username, password)
}

// Login открывает сессию по логину и паролю.
func (a *App) Login(username, password string) (*auth.UserInfo, error) {
	if a.auth == nil {
		return nil, a.authUnavailable()
	}
	return a.auth.Login(username, password)
}

// Logout закрывает сессию.
func (a *App) Logout() {
	if a.auth != nil {
		a.auth.Logout()
	}
}

// CurrentUser возвращает вошедшего пользователя.
func (a *App) CurrentUser() (*auth.UserInfo, error) {
	if a.auth == nil {
		return nil, a.authUnavailable()
	}
	return a.auth.Current()
}

// ChangePassword меняет пароль вошедшего пользователя (в том числе когда смена обязательна).
func (a *App) ChangePassword(oldPassword, newPassword string) error {
	if a.auth == nil {
		return a.authUnavailable()
	}
	return a.auth.ChangePassword(oldPassword, newPassword)
}

// ─── Управление пользователями (users.manage) ──────────────

// ListRoles возвращает роли и их права.
func (a *App) ListRoles() ([]auth.RoleInfo, error) {
	if err := a.authz(auth.PermUsersManage); err != nil {
		return nil, err
	}
	return auth.Roles(), nil
}

func (a *App) ListUsers() ([]auth.UserInfo, error) {
	if err := a.authz(auth.PermUsersManage); err != nil {
		return nil, err
	}
	return a.auth.ListUsers()
}

func (a *App) CreateUser(username, password, role string) (*auth.UserInfo, error) {
	if err := a.authz(auth.PermUsersManage); err != nil {
		return nil, err
	}
	return a.auth.CreateUser(username, password, auth.Role(strings.TrimSpace(role)))
}

func (a *App) UpdateUserRole(id, role string) (*auth.UserInfo, error) {
	if err := a.authz(auth.PermUsersManage); err != nil {
		return nil, err
	}
	return a.auth.UpdateRole(id, auth.Role(strings.TrimSpace(role)))
}

func (a *App) SetUserDisabled(id string, disabled bool) (*auth.UserInfo, error) {
	if err := a.authz(auth.PermUsersManage); err != nil {
		return nil, err
	}
	return a.auth.SetDisabled(id, disabled)
}

func (a *App) ResetUserPassword(id, newPassword string) (*auth.UserInfo, error) {
	if err := a.authz(auth.PermUsersManage); err != nil {
		return nil, err
	}
	return a.auth.ResetPassword(id, newPassword)
}

func (a *App) DeleteUser(id string) error {
	if err := a.authz(auth.PermUsersManage); err != nil {
		return err
	}
	return a.auth.DeleteUser(id)
}
