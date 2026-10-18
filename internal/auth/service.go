package auth

import (
	"analog/internal/audit"
	"context"
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
	ErrInvalidIdleTimeout     = errors.New("время автовыхода: 0 (выключено) или от 1 до 1440 минут")
)

// LockedError — вход временно заблокирован после серии неудачных попыток.
type LockedError struct{ RetryAfter time.Duration }

func (e *LockedError) Error() string {
	secs := int((e.RetryAfter + time.Second - 1) / time.Second)
	return fmt.Sprintf("слишком много неудачных попыток входа, повторите через %d с", secs)
}

const (
	// DefaultIdleTimeoutMinutes — автовыход по бездействию, пока администратор не задал своё значение.
	DefaultIdleTimeoutMinutes = 15
	// MaxIdleTimeoutMinutes — верхняя граница настройки (сутки).
	MaxIdleTimeoutMinutes = 24 * 60
	// idleGrace — запас бэкенда сверх настроенного времени. Интерфейс сообщает об активности
	// не чаще раза в ~20 с, поэтому его собственный таймер всегда срабатывает раньше и успевает
	// корректно выйти; бэкенд — страховка на случай, если интерфейс завис или недоступен.
	idleGrace = 30 * time.Second

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
	// IdleTimeoutMinutes — через сколько минут без действий пользователя сессия завершается (0 — не завершается).
	IdleTimeoutMinutes int `json:"idleTimeoutMinutes"`
}

type failure struct {
	count       int
	lockedUntil time.Time
}

// Recorder принимает события журнала аудита. Реализация (audit.Logger) не должна
// вызывать методы Service: события пишутся под его блокировкой.
type Recorder interface{ Record(audit.Event) }

// Actor — кто выполняет действие (для журнала аудита).
type Actor struct {
	Username string
	Role     Role
}

// Service — аутентификация, сессия и управление пользователями.
//
// Приложение настольное: один процесс — один оператор, поэтому сессия одна и живёт в памяти
// до выхода или закрытия приложения. Права текущего пользователя перечитываются из
// хранилища при каждой проверке, так что смена роли действует немедленно.
type Service struct {
	mu         sync.Mutex
	st         *store
	sessionID  string
	failures   map[string]*failure
	now        func() time.Time
	onLogout   func()
	rec        Recorder
	lastActive time.Time // последнее действие пользователя в интерфейсе (не вызов API)

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

// SetAuditor подключает журнал аудита.
func (s *Service) SetAuditor(r Recorder) {
	s.mu.Lock()
	s.rec = r
	s.mu.Unlock()
}

// record пишет событие в журнал. Вызывается под s.mu; actor может быть nil.
func (s *Service) record(actor *userRecord, action, result, detail string) {
	if s.rec == nil {
		return
	}
	e := audit.Event{Time: s.now(), Action: action, Result: result, Detail: detail}
	if actor != nil {
		e.User, e.Role = actor.Username, string(actor.Role)
	}
	s.rec.Record(e)
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
	if s.idleExpiredLocked() {
		s.expireLocked(u)
		return nil
	}
	return u
}

// ─── Автовыход по бездействию ──────────────────────────────

func (s *Service) idleMinutesLocked() int {
	if m := s.st.settings.IdleTimeoutMinutes; m != nil {
		return *m
	}
	return DefaultIdleTimeoutMinutes
}

// wall отбрасывает монотонные часы: время сна компьютера должно считаться бездействием.
func (s *Service) wall() time.Time { return s.now().Round(0) }

func (s *Service) touchLocked() { s.lastActive = s.wall() }

func (s *Service) idleExpiredLocked() bool {
	m := s.idleMinutesLocked()
	if m <= 0 {
		return false
	}
	return s.wall().Sub(s.lastActive) > time.Duration(m)*time.Minute+idleGrace
}

// expireLocked завершает сессию из-за бездействия и запускает очистку (терминалы, port-forward).
// Очистка идёт в отдельной горутине: она не должна выполняться под блокировкой Service.
func (s *Service) expireLocked(u *userRecord) {
	s.record(u, "SessionTimeout", audit.ResultOK, fmt.Sprintf("бездействие более %d мин", s.idleMinutesLocked()))
	s.sessionID = ""
	if hook := s.onLogout; hook != nil {
		go hook()
	}
}

// ReportActivity сообщает, что пользователь работает с интерфейсом, и продлевает сессию.
// Обычные вызовы API сессию не продлевают: интерфейс сам опрашивает кластер по таймеру,
// и иначе сессия не истекла бы никогда.
func (s *Service) ReportActivity() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sessionUserLocked() == nil {
		return ErrNotAuthenticated
	}
	s.touchLocked()
	return nil
}

// ExpireIdle завершает сессию, если она простаивала дольше допустимого. Возвращает true, если завершила.
func (s *Service) ExpireIdle() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sessionID == "" {
		return false
	}
	return s.sessionUserLocked() == nil
}

// RunIdleWatcher периодически вызывает ExpireIdle, чтобы терминалы и port-forward закрывались
// вовремя, даже если интерфейс больше ничего не вызывает. Возвращается при отмене ctx.
func (s *Service) RunIdleWatcher(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.ExpireIdle()
		}
	}
}

// SetIdleTimeout задаёт время автовыхода в минутах (0 — выключить) и возвращает сохранённое значение.
func (s *Service) SetIdleTimeout(minutes int) (int, error) {
	if minutes < 0 || minutes > MaxIdleTimeoutMinutes {
		return 0, ErrInvalidIdleTimeout
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	actor, err := s.authorizeLocked(PermUsersManage)
	if err != nil {
		return 0, err
	}
	prev := s.st.settings
	was := s.idleMinutesLocked()
	s.st.settings.IdleTimeoutMinutes = &minutes
	if err := s.st.save(); err != nil {
		s.st.settings = prev
		s.record(actor, "SetIdleTimeout", audit.ResultFailed, err.Error())
		return 0, fmt.Errorf("не удалось сохранить настройку: %w", err)
	}
	s.record(actor, "SetIdleTimeout", audit.ResultOK, fmt.Sprintf("minutes=%d→%d", was, minutes))
	return minutes, nil
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
		return u, ErrForbidden
	}
	return u, nil
}

// Authorize проверяет, что есть вошедший пользователь и у него есть право p.
// Это единственная точка проверки прав для всех защищённых методов приложения.
func (s *Service) Authorize(p Permission) error {
	_, err := s.Check(p)
	return err
}

// Check — то же, что Authorize, но возвращает ещё и действующего пользователя для журнала аудита.
// При ErrForbidden пользователь возвращается тоже: отказ записывается от его имени.
func (s *Service) Check(p Permission) (Actor, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, err := s.authorizeLocked(p)
	if u == nil {
		// authorizeLocked не вернул пользователя: либо нет сессии, либо нужна смена пароля
		if cu := s.sessionUserLocked(); cu != nil {
			return Actor{Username: cu.Username, Role: cu.Role}, err
		}
		return Actor{}, err
	}
	return Actor{Username: u.Username, Role: u.Role}, err
}

// Status возвращает состояние для стартового экрана.
func (s *Service) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := Status{NeedsSetup: len(s.st.users) == 0, IdleTimeoutMinutes: s.idleMinutesLocked()}
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
	s.touchLocked()
	s.record(&rec, "Setup", audit.ResultOK, "создан первый администратор")
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
		s.record(&userRecord{Username: username}, "Login", audit.ResultDenied, "вход временно заблокирован после неудачных попыток")
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
		s.record(&userRecord{Username: username}, "Login", audit.ResultFailed, ErrInvalidCredentials.Error())
		s.mu.Unlock()
		return nil, ErrInvalidCredentials
	}
	rec := s.st.findByID(snapshot.ID)
	if rec == nil || rec.PasswordHash != snapshot.PasswordHash {
		// Пользователя удалили или сменили пароль, пока шла проверка.
		s.record(&userRecord{Username: username}, "Login", audit.ResultFailed, ErrInvalidCredentials.Error())
		s.mu.Unlock()
		return nil, ErrInvalidCredentials
	}
	if rec.Disabled {
		s.record(rec, "Login", audit.ResultDenied, ErrAccountDisabled.Error())
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
	s.touchLocked()
	s.record(rec, "Login", audit.ResultOK, "")
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
	if had {
		s.record(s.st.findByID(s.sessionID), "Logout", audit.ResultOK, "")
	}
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
		s.mu.Lock()
		s.record(s.st.findByID(id), "ChangePassword", audit.ResultFailed, ErrWrongPassword.Error())
		s.mu.Unlock()
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
	s.touchLocked()
	s.record(rec, "ChangePassword", audit.ResultOK, "")
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
	actor := s.sessionUserLocked()
	detail := fmt.Sprintf("user=%s role=%s", username, role)
	if s.st.findByName(normalizeUsername(username)) != nil {
		s.record(actor, "CreateUser", audit.ResultFailed, detail+": "+ErrUserExists.Error())
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
		s.record(actor, "CreateUser", audit.ResultFailed, detail+": "+err.Error())
		return nil, fmt.Errorf("не удалось сохранить пользователя: %w", err)
	}
	s.record(actor, "CreateUser", audit.ResultOK, detail)
	return toInfo(&rec), nil
}

// mutate применяет изменение к пользователю с откатом при ошибке записи.
//
// fn возвращает описание изменения для журнала аудита (например, «admin → user»).
func (s *Service) mutate(id, action string, fn func(actor, target *userRecord) (string, error)) (*UserInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	actor, err := s.authorizeLocked(PermUsersManage)
	if err != nil {
		return nil, err
	}
	target := s.st.findByID(id)
	if target == nil {
		s.record(actor, action, audit.ResultFailed, "id="+id+": "+ErrUserNotFound.Error())
		return nil, ErrUserNotFound
	}
	prev := *target
	who := "user=" + target.Username
	change, err := fn(actor, target)
	if err != nil {
		*target = prev
		s.record(actor, action, audit.ResultFailed, who+": "+err.Error())
		return nil, err
	}
	target.UpdatedAt = s.now()
	if err := s.st.save(); err != nil {
		*target = prev
		s.record(actor, action, audit.ResultFailed, who+": "+err.Error())
		return nil, fmt.Errorf("не удалось сохранить изменения: %w", err)
	}
	if change != "" {
		who += " " + change
	}
	s.record(actor, action, audit.ResultOK, who)
	return toInfo(target), nil
}

// UpdateRole меняет роль пользователя.
func (s *Service) UpdateRole(id string, role Role) (*UserInfo, error) {
	if !role.Valid() {
		return nil, ErrUnknownRole
	}
	return s.mutate(id, "UpdateUserRole", func(actor, t *userRecord) (string, error) {
		// Свою роль менять нельзя: иначе уже запущенные терминалы и port-forward
		// пережили бы потерю прав. Понизить администратора может только другой администратор.
		if actor.ID == t.ID && role != t.Role {
			return "", ErrSelfAction
		}
		if t.Role == RoleAdmin && role != RoleAdmin && !t.Disabled && s.st.enabledAdmins() <= 1 {
			return "", ErrLastAdmin
		}
		change := fmt.Sprintf("role=%s→%s", t.Role, role)
		t.Role = role
		return change, nil
	})
}

// SetDisabled включает или отключает учётную запись.
func (s *Service) SetDisabled(id string, disabled bool) (*UserInfo, error) {
	return s.mutate(id, "SetUserDisabled", func(actor, t *userRecord) (string, error) {
		if disabled {
			if actor.ID == t.ID {
				return "", ErrSelfAction
			}
			if t.Role == RoleAdmin && !t.Disabled && s.st.enabledAdmins() <= 1 {
				return "", ErrLastAdmin
			}
		}
		t.Disabled = disabled
		return fmt.Sprintf("disabled=%t", disabled), nil
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
	return s.mutate(id, "ResetUserPassword", func(actor, t *userRecord) (string, error) {
		if actor.ID == t.ID {
			return "", ErrSelfAction // свой пароль меняют через ChangePassword
		}
		if strings.EqualFold(newPassword, t.Username) {
			return "", ErrWeakPassword
		}
		t.PasswordHash = hash
		t.MustChangePassword = true
		return "", nil
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
		s.record(actor, "DeleteUser", audit.ResultFailed, "id="+id+": "+ErrUserNotFound.Error())
		return ErrUserNotFound
	}
	who := "user=" + target.Username
	if actor.ID == target.ID {
		s.record(actor, "DeleteUser", audit.ResultFailed, who+": "+ErrSelfAction.Error())
		return ErrSelfAction
	}
	if target.Role == RoleAdmin && !target.Disabled && s.st.enabledAdmins() <= 1 {
		s.record(actor, "DeleteUser", audit.ResultFailed, who+": "+ErrLastAdmin.Error())
		return ErrLastAdmin
	}
	prev := append([]userRecord(nil), s.st.users...)
	s.st.remove(id)
	if err := s.st.save(); err != nil {
		s.st.users = prev
		s.record(actor, "DeleteUser", audit.ResultFailed, who+": "+err.Error())
		return fmt.Errorf("не удалось сохранить изменения: %w", err)
	}
	s.record(actor, "DeleteUser", audit.ResultOK, who)
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
