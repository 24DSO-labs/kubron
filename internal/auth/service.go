package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Ошибки отдаются во фронтенд как текст (Wails), поэтому сообщения — на языке интерфейса.
var (
	ErrNotAuthenticated       = errors.New("требуется вход в систему")
	ErrForbidden              = errors.New("недостаточно прав для этого действия")
	ErrInvalidCredentials     = errors.New("неверный логин или пароль")
	ErrAccountDisabled        = errors.New("учётная запись отключена")
	ErrSetupDone              = errors.New("первичная настройка уже выполнена")
	ErrSetupRequired          = errors.New("требуется первичная настройка: создайте администратора")
	ErrPasswordChangeRequired = errors.New("требуется сменить пароль")
	ErrUserExists             = errors.New("пользователь с таким логином уже существует")
	ErrUserNotFound           = errors.New("пользователь не найден")
	ErrUnknownRole            = errors.New("неизвестная роль")
	ErrLastAdmin              = errors.New("нельзя лишить доступа последнего администратора")
	ErrSelfAction             = errors.New("это действие нельзя выполнить над собственной учётной записью")
	ErrWrongPassword          = errors.New("текущий пароль указан неверно")
	ErrSamePassword           = errors.New("новый пароль должен отличаться от текущего")
	ErrWeakPassword           = errors.New("пароль не должен совпадать с логином")
)

// LockedError — вход временно заблокирован после серии неудачных попыток.
type LockedError struct{ RetryAfter time.Duration }

func (e *LockedError) Error() string {
	secs := int((e.RetryAfter + time.Second - 1) / time.Second)
	return fmt.Sprintf("слишком много неудачных попыток входа, повторите через %d с", secs)
}

const (
	freeAttempts = 5
	baseLockout  = 15 * time.Second
	maxLockout   = 5 * time.Minute
	maxTracked   = 1024
)

// UserInfo — безопасное представление пользователя для интерфейса (без хеша пароля).
type UserInfo struct {
	ID                 string       `json:"id"`
	Username           string       `json:"username"`
	Role               Role         `json:"role"`
	RoleLabel          string       `json:"roleLabel"`
	Permissions        []Permission `json:"permissions"`
	Disabled           bool         `json:"disabled"`
	MustChangePassword bool         `json:"mustChangePassword"`
	CreatedAt          string       `json:"createdAt"`
	LastLoginAt        string       `json:"lastLoginAt"`
}

// Status — состояние аутентификации при старте интерфейса.
type Status struct {
	NeedsSetup bool      `json:"needsSetup"`
	User       *UserInfo `json:"user"`
}

type failure struct {
	count       int
	lockedUntil time.Time
}

// Service — аутентификация, сессия и управление пользователями.
//
// Приложение настольное: один процесс — один оператор, поэтому сессия одна и живёт в памяти
// до выхода или закрытия приложения. Права текущего пользователя перечитываются из
// хранилища при каждой проверке, так что смена роли действует немедленно.
type Service struct {
	mu        sync.Mutex
	st        *store
	sessionID string
	failures  map[string]*failure
	now       func() time.Time
	onLogout  func()

	dummyOnce sync.Once
	dummyHash string
}

// NewService открывает (или создаёт при первом запуске) хранилище пользователей в каталоге dir.
func NewService(dir string) (*Service, error) {
	st, err := loadStore(dir)
	if err != nil {
		return nil, err
	}
	return &Service{st: st, failures: map[string]*failure{}, now: time.Now}, nil
}

// SetOnLogout задаёт обработчик, вызываемый при завершении сессии
// (закрыть терминалы, остановить port-forward и т.п.).
func (s *Service) SetOnLogout(fn func()) {
	s.mu.Lock()
	s.onLogout = fn
	s.mu.Unlock()
}

func normalizeUsername(u string) string { return strings.ToLower(strings.TrimSpace(u)) }

func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("auth: crypto/rand недоступен: " + err.Error())
	}
	return hex.EncodeToString(b)
}

func toInfo(u *userRecord) *UserInfo {
	info := &UserInfo{
		ID:                 u.ID,
		Username:           u.Username,
		Role:               u.Role,
		RoleLabel:          u.Role.Label(),
		Permissions:        u.Role.Permissions(),
		Disabled:           u.Disabled,
		MustChangePassword: u.MustChangePassword,
		CreatedAt:          u.CreatedAt.UTC().Format(time.RFC3339),
	}
	if u.LastLoginAt != nil {
		info.LastLoginAt = u.LastLoginAt.UTC().Format(time.RFC3339)
	}
	return info
}

// ─── Сессия ────────────────────────────────────────────────

// sessionUserLocked возвращает пользователя текущей сессии. Если его удалили или отключили —
// сессия сбрасывается.
func (s *Service) sessionUserLocked() *userRecord {
	if s.sessionID == "" {
		return nil
	}
	u := s.st.findByID(s.sessionID)
	if u == nil || u.Disabled {
		s.sessionID = ""
		return nil
	}
	return u
}

func (s *Service) authorizeLocked(p Permission) (*userRecord, error) {
	u := s.sessionUserLocked()
	if u == nil {
		return nil, ErrNotAuthenticated
	}
	if u.MustChangePassword {
		return nil, ErrPasswordChangeRequired
	}
	if !u.Role.Has(p) {
		return nil, ErrForbidden
	}
	return u, nil
}

// Authorize проверяет, что есть вошедший пользователь и у него есть право p.
// Это единственная точка проверки прав для всех защищённых методов приложения.
func (s *Service) Authorize(p Permission) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.authorizeLocked(p)
	return err
}

// Status возвращает состояние для стартового экрана.
func (s *Service) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := Status{NeedsSetup: len(s.st.users) == 0}
	if u := s.sessionUserLocked(); u != nil {
		st.User = toInfo(u)
	}
	return st
}

// Current возвращает пользователя сессии (даже если ему нужно сменить пароль).
func (s *Service) Current() (*UserInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u := s.sessionUserLocked()
	if u == nil {
		return nil, ErrNotAuthenticated
	}
	return toInfo(u), nil
}

// ─── Вход / выход ──────────────────────────────────────────

// Setup создаёт первого администратора. Работает только пока в системе нет ни одного пользователя.
func (s *Service) Setup(username, password string) (*UserInfo, error) {
	username = strings.TrimSpace(username)
	if err := validateCredentials(username, password); err != nil {
		return nil, err
	}
	hash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.st.users) > 0 {
		return nil, ErrSetupDone
	}
	now := s.now()
	rec := userRecord{
		ID: newID(), Username: username, Role: RoleAdmin, PasswordHash: hash,
		CreatedAt: now, UpdatedAt: now, LastLoginAt: &now,
	}
	s.st.users = append(s.st.users, rec)
	if err := s.st.save(); err != nil {
		s.st.users = nil
		return nil, fmt.Errorf("не удалось сохранить пользователя: %w", err)
	}
	s.sessionID = rec.ID
	return toInfo(&rec), nil
}

func (s *Service) dummy() string {
	s.dummyOnce.Do(func() { s.dummyHash, _ = hashPassword("kubron-dummy-password") })
	return s.dummyHash
}

// Login проверяет логин и пароль и открывает сессию.
func (s *Service) Login(username, password string) (*UserInfo, error) {
	key := normalizeUsername(username)

	s.mu.Lock()
	if len(s.st.users) == 0 {
		s.mu.Unlock()
		return nil, ErrSetupRequired
	}
	if remaining := s.lockRemainingLocked(key); remaining > 0 {
		s.mu.Unlock()
		return nil, &LockedError{RetryAfter: remaining}
	}
	var snapshot *userRecord
	if rec := s.st.findByName(key); rec != nil {
		c := *rec
		snapshot = &c
	}
	s.mu.Unlock()

	// Хеширование — вне блокировки. Для несуществующего логина считаем фиктивный хеш,
	// чтобы время ответа не выдавало, существует ли пользователь.
	hash := s.dummy()
	if snapshot != nil {
		hash = snapshot.PasswordHash
	}
	ok, rehash := verifyPassword(password, hash)

	var newHash string
	if ok && snapshot != nil && rehash {
		newHash, _ = hashPassword(password)
	}

	s.mu.Lock()
	if !ok || snapshot == nil {
		s.recordFailureLocked(key)
		s.mu.Unlock()
		return nil, ErrInvalidCredentials
	}
	rec := s.st.findByID(snapshot.ID)
	if rec == nil || rec.PasswordHash != snapshot.PasswordHash {
		// Пользователя удалили или сменили пароль, пока шла проверка.
		s.mu.Unlock()
		return nil, ErrInvalidCredentials
	}
	if rec.Disabled {
		s.mu.Unlock()
		return nil, ErrAccountDisabled
	}
	delete(s.failures, key)

	replaced := s.sessionID != "" && s.sessionID != rec.ID
	now := s.now()
	rec.LastLoginAt = &now
	if newHash != "" {
		rec.PasswordHash = newHash
	}
	_ = s.st.save() // не мешаем входу, если диск недоступен для записи
	s.sessionID = rec.ID
	info := toInfo(rec)
	hook := s.onLogout
	s.mu.Unlock()

	if replaced && hook != nil {
		hook()
	}
	return info, nil
}

// Logout завершает сессию и вызывает обработчик очистки.
func (s *Service) Logout() {
	s.mu.Lock()
	had := s.sessionID != ""
	s.sessionID = ""
	hook := s.onLogout
	s.mu.Unlock()
	if had && hook != nil {
		hook()
	}
}

func (s *Service) lockRemainingLocked(key string) time.Duration {
	f := s.failures[key]
	if f == nil {
		return 0
	}
	if d := f.lockedUntil.Sub(s.now()); d > 0 {
		return d
	}
	return 0
}

func (s *Service) recordFailureLocked(key string) {
	if len(s.failures) >= maxTracked {
		now := s.now()
		for k, f := range s.failures {
			if f.lockedUntil.Before(now) {
				delete(s.failures, k)
			}
		}
		if len(s.failures) >= maxTracked { // всё ещё переполнено — сбрасываем всё, это защита от раздувания памяти
			s.failures = map[string]*failure{}
		}
	}
	f := s.failures[key]
	if f == nil {
		f = &failure{}
		s.failures[key] = f
	}
	f.count++
	if f.count >= freeAttempts {
		exp := f.count - freeAttempts
		if exp > 6 {
			exp = 6
		}
		d := baseLockout << uint(exp)
		if d > maxLockout {
			d = maxLockout
		}
		f.lockedUntil = s.now().Add(d)
	}
}

// ChangePassword меняет пароль текущего пользователя.
func (s *Service) ChangePassword(oldPassword, newPassword string) error {
	s.mu.Lock()
	u := s.sessionUserLocked()
	if u == nil {
		s.mu.Unlock()
		return ErrNotAuthenticated
	}
	id, username, oldHash := u.ID, u.Username, u.PasswordHash
	s.mu.Unlock()

	if err := validateCredentialsPassword(username, newPassword); err != nil {
		return err
	}
	if ok, _ := verifyPassword(oldPassword, oldHash); !ok {
		return ErrWrongPassword
	}
	if oldPassword == newPassword {
		return ErrSamePassword
	}
	newHash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	rec := s.st.findByID(id)
	if rec == nil || s.sessionID != id {
		return ErrNotAuthenticated
	}
	prev := *rec
	rec.PasswordHash = newHash
	rec.MustChangePassword = false
	rec.UpdatedAt = s.now()
	if err := s.st.save(); err != nil {
		*rec = prev
		return fmt.Errorf("не удалось сохранить пароль: %w", err)
	}
	return nil
}

// ─── Управление пользователями (право users.manage) ────────

// ListUsers возвращает всех пользователей.
func (s *Service) ListUsers() ([]UserInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.authorizeLocked(PermUsersManage); err != nil {
		return nil, err
	}
	out := make([]UserInfo, 0, len(s.st.users))
	for i := range s.st.users {
		out = append(out, *toInfo(&s.st.users[i]))
	}
	return out, nil
}

// CreateUser создаёт пользователя с временным паролем: при первом входе его придётся сменить.
func (s *Service) CreateUser(username, password string, role Role) (*UserInfo, error) {
	username = strings.TrimSpace(username)
	if !role.Valid() {
		return nil, ErrUnknownRole
	}
	if err := validateCredentials(username, password); err != nil {
		return nil, err
	}
	if err := s.Authorize(PermUsersManage); err != nil {
		return nil, err
	}
	hash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.authorizeLocked(PermUsersManage); err != nil {
		return nil, err
	}
	if s.st.findByName(normalizeUsername(username)) != nil {
		return nil, ErrUserExists
	}
	now := s.now()
	rec := userRecord{
		ID: newID(), Username: username, Role: role, PasswordHash: hash,
		MustChangePassword: true, CreatedAt: now, UpdatedAt: now,
	}
	s.st.users = append(s.st.users, rec)
	if err := s.st.save(); err != nil {
		s.st.users = s.st.users[:len(s.st.users)-1]
		return nil, fmt.Errorf("не удалось сохранить пользователя: %w", err)
	}
	return toInfo(&rec), nil
}

// mutate применяет изменение к пользователю с откатом при ошибке записи.
func (s *Service) mutate(id string, fn func(actor, target *userRecord) error) (*UserInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	actor, err := s.authorizeLocked(PermUsersManage)
	if err != nil {
		return nil, err
	}
	target := s.st.findByID(id)
	if target == nil {
		return nil, ErrUserNotFound
	}
	prev := *target
	if err := fn(actor, target); err != nil {
		*target = prev
		return nil, err
	}
	target.UpdatedAt = s.now()
	if err := s.st.save(); err != nil {
		*target = prev
		return nil, fmt.Errorf("не удалось сохранить изменения: %w", err)
	}
	return toInfo(target), nil
}

// UpdateRole меняет роль пользователя.
func (s *Service) UpdateRole(id string, role Role) (*UserInfo, error) {
	if !role.Valid() {
		return nil, ErrUnknownRole
	}
	return s.mutate(id, func(actor, t *userRecord) error {
		// Свою роль менять нельзя: иначе уже запущенные терминалы и port-forward
		// пережили бы потерю прав. Понизить администратора может только другой администратор.
		if actor.ID == t.ID && role != t.Role {
			return ErrSelfAction
		}
		if t.Role == RoleAdmin && role != RoleAdmin && !t.Disabled && s.st.enabledAdmins() <= 1 {
			return ErrLastAdmin
		}
		t.Role = role
		return nil
	})
}

// SetDisabled включает или отключает учётную запись.
func (s *Service) SetDisabled(id string, disabled bool) (*UserInfo, error) {
	return s.mutate(id, func(actor, t *userRecord) error {
		if disabled {
			if actor.ID == t.ID {
				return ErrSelfAction
			}
			if t.Role == RoleAdmin && !t.Disabled && s.st.enabledAdmins() <= 1 {
				return ErrLastAdmin
			}
		}
		t.Disabled = disabled
		return nil
	})
}

// ResetPassword задаёт пользователю временный пароль; при следующем входе его нужно сменить.
func (s *Service) ResetPassword(id, newPassword string) (*UserInfo, error) {
	if err := ValidatePassword(newPassword); err != nil {
		return nil, err
	}
	if err := s.Authorize(PermUsersManage); err != nil {
		return nil, err
	}
	hash, err := hashPassword(newPassword)
	if err != nil {
		return nil, err
	}
	return s.mutate(id, func(actor, t *userRecord) error {
		if actor.ID == t.ID {
			return ErrSelfAction // свой пароль меняют через ChangePassword
		}
		if strings.EqualFold(newPassword, t.Username) {
			return ErrWeakPassword
		}
		t.PasswordHash = hash
		t.MustChangePassword = true
		return nil
	})
}

// DeleteUser удаляет пользователя.
func (s *Service) DeleteUser(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	actor, err := s.authorizeLocked(PermUsersManage)
	if err != nil {
		return err
	}
	target := s.st.findByID(id)
	if target == nil {
		return ErrUserNotFound
	}
	if actor.ID == target.ID {
		return ErrSelfAction
	}
	if target.Role == RoleAdmin && !target.Disabled && s.st.enabledAdmins() <= 1 {
		return ErrLastAdmin
	}
	prev := append([]userRecord(nil), s.st.users...)
	s.st.remove(id)
	if err := s.st.save(); err != nil {
		s.st.users = prev
		return fmt.Errorf("не удалось сохранить изменения: %w", err)
	}
	return nil
}

// ─── Проверки ввода ────────────────────────────────────────

func validateCredentials(username, password string) error {
	if err := ValidateUsername(username); err != nil {
		return err
	}
	return validateCredentialsPassword(username, password)
}

func validateCredentialsPassword(username, password string) error {
	if err := ValidatePassword(password); err != nil {
		return err
	}
	if strings.EqualFold(password, username) {
		return ErrWeakPassword
	}
	return nil
}
