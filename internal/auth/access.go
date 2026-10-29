// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import "errors"

// ErrClusterNotFound — кластера нет или он принадлежит чужой команде. Намеренно один и тот же ответ:
// пользователь не должен узнавать, что чужие кластеры существуют (этап 3, решение 2).
var ErrClusterNotFound = errors.New("cluster not found")

// Access — модель доступа сервиса «организация → команда → кластер» (этап 3, задача 3.2).
// Реализация (PostgreSQL) должна отвечать быстро: вызывается на каждый вызов API, поэтому кэширует снимок.
// Пока Access не задан (десктоп, личный режим), действует прежняя глобальная роль пользователя.
type Access interface {
	// ClusterTeam возвращает команду-владельца кластера; ok=false — кластера нет.
	ClusterTeam(clusterID string) (teamID string, ok bool)
	// TeamRole возвращает роль пользователя в команде; ok=false — он не её член.
	TeamRole(userID, teamID string) (Role, bool)
	// TeamRoles возвращает роли пользователя во всех его командах (ключ — id команды).
	TeamRoles(userID string) map[string]Role
	// IsOrgAdmin сообщает, что пользователь — администратор организации. Это управление пользователями и командами,
	// но не доступ к данным кластеров: его даёт только членство в команде.
	IsOrgAdmin(userID string) bool
}

// SetAccess включает проверку прав по членству в командах. Вызывается один раз при старте, до первого входа.
func (s *Service) SetAccess(a Access) {
	s.mu.Lock()
	s.access = a
	s.mu.Unlock()
}

// ClusterScoped сообщает, что право действует внутри кластера и определяется ролью в команде-владельце
// (остальные — права организации и управления кластерами).
func ClusterScoped(p Permission) bool {
	switch p {
	case PermRead, PermSecretsRead, PermOperate, PermWrite, PermExec, PermPortForward:
		return true
	}
	return false
}

// accessLocked решает, разрешено ли право p пользователю u. clusterID пуст для методов вне конкретного кластера.
// Возвращает роль, по которой принято решение (для журнала). Вызывать под s.mu; s.access != nil.
//
// Правила:
//   - с кластером: чужой или отсутствующий → ErrClusterNotFound; иначе право определяет роль в его команде
//     (администратор организации без членства доступа к кластеру не получает);
//   - без кластера, права кластера: достаточно иметь это право хотя бы в одной команде (списки, переключение;
//     сами операции всегда идут с активным кластером и проверяются по нему);
//   - clusters.manage без кластера (добавление): администратор хотя бы одной команды (команда назначается в 3.3);
//   - users.manage, audit.read: администратор организации.
func (s *Service) accessLocked(u *UserRecord, p Permission, clusterID string) (Role, error) {
	a := s.access
	if clusterID != "" {
		team, ok := a.ClusterTeam(clusterID)
		if !ok {
			return u.Role, ErrClusterNotFound
		}
		role, ok := a.TeamRole(u.ID, team)
		if !ok {
			return u.Role, ErrClusterNotFound
		}
		if !role.Has(p) {
			return role, ErrForbidden
		}
		return role, nil
	}
	switch {
	case ClusterScoped(p), p == PermClusterManage:
		for _, role := range a.TeamRoles(u.ID) {
			if role.Has(p) {
				return u.Role, nil
			}
		}
		return u.Role, ErrForbidden
	default: // users.manage, audit.read и будущие права организации
		if a.IsOrgAdmin(u.ID) {
			return u.Role, nil
		}
		return u.Role, ErrForbidden
	}
}

// CheckCluster — Check для действия над конкретным кластером. Неизвестный и чужой кластер дают
// ErrClusterNotFound. Без Access ведёт себя как Check (глобальная роль).
func (s *Service) CheckCluster(p Permission, clusterID string) (Actor, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.checkLocked(p, clusterID)
}

// AuthorizeCluster — CheckCluster без возврата пользователя.
func (s *Service) AuthorizeCluster(p Permission, clusterID string) error {
	_, err := s.CheckCluster(p, clusterID)
	return err
}

// CanSeeCluster сообщает, видит ли вошедший пользователь кластер (его команда — команда пользователя).
// Без Access видны все кластеры, если пользователь вошёл. Используется для фильтрации списков.
func (s *Service) CanSeeCluster(clusterID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	u := s.sessionUserLocked()
	if u == nil || u.MustChangePassword {
		return false
	}
	if s.access == nil {
		return true
	}
	team, ok := s.access.ClusterTeam(clusterID)
	if !ok {
		return false
	}
	_, ok = s.access.TeamRole(u.ID, team)
	return ok
}
