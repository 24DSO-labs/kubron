package main

import (
	"analog/internal/audit"
	"analog/internal/auth"
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// initAuth открывает хранилище пользователей. Если файл повреждён, приложение остаётся
// закрытым (authz отказывает всем), а причина показывается на экране входа.
func (a *App) initAuth(dir string) {
	a.audit = audit.Open(dir)
	svc, err := auth.NewService(dir)
	if err != nil {
		a.authErr = err
		return
	}
	svc.SetAuditor(a.audit)
	// При выходе из учётной записи ничего из сессии не должно продолжать работать.
	svc.SetOnLogout(func() {
		a.pfMgr.StopAll()
		a.k8sMgr.CloseAllTerminals()
	})
	a.auth = svc
}

// auditedPermissions — права, использование которых попадает в журнал при разрешении.
// Чтение не журналируется (слишком шумно и безопасно), управление пользователями пишет
// сам auth.Service — вместе с результатом.
var auditedPermissions = map[auth.Permission]bool{
	auth.PermWrite:         true,
	auth.PermExec:          true,
	auth.PermPortForward:   true,
	auth.PermClusterManage: true,
	auth.PermSecretsRead:   true,
}

// quietMethods — вызываются на каждое нажатие клавиши/опрос терминала: в журнал не пишутся
// (начало и конец терминальной сессии записываются методами StartTerminalSession / CloseTerminal).
var quietMethods = map[string]bool{
	"WriteTerminal":  true,
	"ReadTerminal":   true,
	"ResizeTerminal": true,
}

// authz — единственная точка проверки прав в Wails-методах.
// Каждый экспортируемый метод App обязан начинаться с её вызова; это проверяет
// тест internal/auth/policy_test.go.
//
// Необязательные аргументы — пары «ключ, значение» для журнала аудита (цель действия):
// a.authz(auth.PermWrite, "namespace", ns, "name", name). Секреты и содержимое (пароли,
// YAML, ввод терминала) сюда передавать нельзя.
func (a *App) authz(p auth.Permission, kv ...any) error {
	if a.auth == nil {
		return a.authUnavailable()
	}
	actor, err := a.auth.Check(p)
	switch {
	case err == nil:
		if auditedPermissions[p] {
			a.auditCall(actor, audit.ResultOK, p, kv)
		}
	case errors.Is(err, auth.ErrForbidden):
		a.auditCall(actor, audit.ResultDenied, p, kv)
	}
	return err
}

// auditCall записывает вызов метода App. Имя метода берётся из стека, чтобы его не дублировать
// в каждом вызове authz.
func (a *App) auditCall(actor auth.Actor, result string, p auth.Permission, kv []any) {
	if a.audit == nil {
		return
	}
	action := callerMethod(3)
	if quietMethods[action] {
		return
	}
	var parts []string
	for i := 0; i+1 < len(kv); i += 2 {
		parts = append(parts, fmt.Sprintf("%v=%v", kv[i], kv[i+1]))
	}
	if result == audit.ResultDenied {
		parts = append(parts, "need="+string(p))
	}
	a.audit.Record(audit.Event{
		User: actor.Username, Role: string(actor.Role),
		Action: action, Result: result, Detail: strings.Join(parts, " "),
	})
}

// callerMethod возвращает имя метода на skip кадров выше вызывающего callerMethod:
// «main.(*App).DeletePod» → «DeletePod». CallersFrames корректно раскрывает инлайн-вызовы.
func callerMethod(skip int) string {
	pcs := make([]uintptr, 1)
	if runtime.Callers(skip+1, pcs) == 0 {
		return "unknown"
	}
	frame, _ := runtime.CallersFrames(pcs).Next()
	if frame.Function == "" {
		return "unknown"
	}
	return frame.Function[strings.LastIndex(frame.Function, ".")+1:]
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

// ReportActivity вызывается интерфейсом при действиях пользователя (мышь, клавиатура)
// и продлевает сессию. Не требует права: достаточно действующей сессии.
func (a *App) ReportActivity() error {
	if a.auth == nil {
		return a.authUnavailable()
	}
	return a.auth.ReportActivity()
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

// ─── Журнал аудита (audit.read) ────────────────────────────

// GetAuditLog возвращает записи журнала от новых к старым с учётом фильтра.
func (a *App) GetAuditLog(filter audit.Filter) (*audit.Page, error) {
	if err := a.authz(auth.PermAuditRead); err != nil {
		return nil, err
	}
	return a.audit.Query(filter)
}

// SetIdleTimeout задаёт время автовыхода по бездействию в минутах (0 — выключить).
func (a *App) SetIdleTimeout(minutes int) (int, error) {
	if err := a.authz(auth.PermUsersManage); err != nil {
		return 0, err
	}
	return a.auth.SetIdleTimeout(minutes)
}
