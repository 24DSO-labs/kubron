// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import "time"

// UserRecord — запись пользователя в хранилище. Наружу (в интерфейс) не отдаётся: см. UserInfo.
type UserRecord struct {
	ID                 string     `json:"id"`
	Username           string     `json:"username"`
	Role               Role       `json:"role"`
	PasswordHash       string     `json:"passwordHash"`
	Disabled           bool       `json:"disabled"`
	MustChangePassword bool       `json:"mustChangePassword"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
	LastLoginAt        *time.Time `json:"lastLoginAt,omitempty"`
}

// SecuritySettings — настройки безопасности, общие для всех пользователей.
// Указатель различает «не задано» (берётся значение по умолчанию) и явный 0 («выключено»).
type SecuritySettings struct {
	IdleTimeoutMinutes *int `json:"idleTimeoutMinutes,omitempty"`
}

// Data — всё, что хранится постоянно: пользователи и настройки безопасности.
type Data struct {
	Users    []UserRecord     `json:"users"`
	Settings SecuritySettings `json:"settings"`
}

// Backend — постоянное хранилище пользователей. Реализации: файл users.json (десктоп) и,
// позже, PostgreSQL (сервер, ADR-0007). Вызывается под блокировкой Service.
type Backend interface {
	// Load читает данные. Отсутствие данных — пустой Data (первый запуск). Повреждённые или
	// созданные более новой версией данные — ошибка: молча считать их пустыми нельзя, иначе
	// любой смог бы пройти первичную настройку и создать администратора.
	Load() (*Data, error)
	// Save атомарно сохраняет данные целиком.
	Save(*Data) error
}

// store — пользователи и настройки в памяти поверх Backend. Не потокобезопасно: синхронизацию делает Service.
type store struct {
	backend  Backend
	users    []UserRecord
	settings SecuritySettings
}

func loadStore(b Backend) (*store, error) {
	d, err := b.Load()
	if err != nil {
		return nil, err
	}
	return &store{backend: b, users: d.Users, settings: d.Settings}, nil
}

func (s *store) save() error {
	return s.backend.Save(&Data{Users: s.users, Settings: s.settings})
}

func (s *store) findByID(id string) *UserRecord {
	for i := range s.users {
		if s.users[i].ID == id {
			return &s.users[i]
		}
	}
	return nil
}

func (s *store) findByName(normalized string) *UserRecord {
	for i := range s.users {
		if normalizeUsername(s.users[i].Username) == normalized {
			return &s.users[i]
		}
	}
	return nil
}

func (s *store) remove(id string) {
	for i := range s.users {
		if s.users[i].ID == id {
			s.users = append(s.users[:i], s.users[i+1:]...)
			return
		}
	}
}

// enabledAdmins — число активных администраторов.
func (s *store) enabledAdmins() int {
	n := 0
	for _, u := range s.users {
		if u.Role == RoleAdmin && !u.Disabled {
			n++
		}
	}
	return n
}
