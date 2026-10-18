package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	usersFileName  = "users.json"
	storageVersion = 1
)

// userRecord — запись пользователя в файле. Наружу (в интерфейс) не отдаётся: см. UserInfo.
type userRecord struct {
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

// settingsData — настройки безопасности, общие для всех пользователей.
// Указатель различает «не задано» (берётся значение по умолчанию) и явный 0 («выключено»).
type settingsData struct {
	IdleTimeoutMinutes *int `json:"idleTimeoutMinutes,omitempty"`
}

type fileData struct {
	Version  int          `json:"version"`
	Users    []userRecord `json:"users"`
	Settings settingsData `json:"settings"`
}

// store — файловое хранилище пользователей. Не потокобезопасно: синхронизацию делает Service.
type store struct {
	path     string
	users    []userRecord
	settings settingsData
}

// loadStore читает файл. Отсутствие файла — это пустое хранилище (первый запуск).
// Повреждённый файл — ошибка: молча считать его пустым нельзя, иначе любой смог бы
// пройти первичную настройку и создать администратора.
func loadStore(dir string) (*store, error) {
	s := &store{path: filepath.Join(dir, usersFileName)}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать %s: %w", s.path, err)
	}
	var fd fileData
	if err := json.Unmarshal(data, &fd); err != nil {
		return nil, fmt.Errorf("файл пользователей %s повреждён: %w", s.path, err)
	}
	if fd.Version > storageVersion {
		return nil, fmt.Errorf("файл пользователей %s создан более новой версией Kubron", s.path)
	}
	s.users = fd.Users
	s.settings = fd.Settings
	return s, nil
}

// save атомарно записывает файл: временный файл в той же папке, затем rename.
func (s *store) save() error {
	data, err := json.MarshalIndent(fileData{Version: storageVersion, Users: s.users, Settings: s.settings}, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, usersFileName+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }

	// os.CreateTemp создаёт файл с правами 0600, лишний chmod не нужен.
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		cleanup()
		return err
	}
	return nil
}

func (s *store) findByID(id string) *userRecord {
	for i := range s.users {
		if s.users[i].ID == id {
			return &s.users[i]
		}
	}
	return nil
}

func (s *store) findByName(normalized string) *userRecord {
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
