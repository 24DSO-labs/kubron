// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

// Package audit — журнал аудита Kubron: кто, когда и что сделал.
//
// Формат — JSON Lines (одна запись на строку) в файле audit.log каталога данных приложения.
// Записи только добавляются; при достижении maxFileSize файл ротируется
// (audit.log → audit.log.1 → … → audit.log.<keepFiles-1>, самый старый удаляется).
//
// Журнал не должен ломать работу приложения: ошибка записи не прерывает действие,
// а запоминается и показывается в интерфейсе (см. Page.WriteError).
package audit

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Результат действия.
const (
	ResultOK     = "ok"     // действие выполнено (или, для действий кластера, разрешено и запущено)
	ResultFailed = "failed" // действие разрешено, но завершилось ошибкой
	ResultDenied = "denied" // отказано: у пользователя нет права
)

const (
	fileName    = "audit.log"
	maxFileSize = 5 << 20 // 5 МБ
	keepFiles   = 5       // текущий файл + 4 архивных
	maxField    = 256     // символов в одном текстовом поле записи

	DefaultLimit = 200
	MaxLimit     = 1000
)

// Event — запись журнала.
type Event struct {
	Time   time.Time `json:"time"`
	User   string    `json:"user"`             // логин на момент действия; пусто, если вход не выполнен
	Role   string    `json:"role,omitempty"`   // роль на момент действия
	Action string    `json:"action"`           // имя действия: Login, DeletePod, CreateUser, …
	Result string    `json:"result"`           // ok | failed | denied
	Detail string    `json:"detail,omitempty"` // цель и параметры: namespace=default name=nginx
	Source string    `json:"source,omitempty"` // адрес клиента веб-сессии; пусто в десктопе
}

// Filter — условия выборки. Пустое поле не ограничивает выборку.
type Filter struct {
	User   string `json:"user"`   // подстрока логина, без учёта регистра
	Action string `json:"action"` // подстрока имени действия, без учёта регистра
	Text   string `json:"text"`   // подстрока в detail, без учёта регистра
	Result string `json:"result"` // ok | failed | denied
	Since  string `json:"since"`  // RFC3339, включительно
	Until  string `json:"until"`  // RFC3339, не включая
	Limit  int    `json:"limit"`  // по умолчанию DefaultLimit, максимум MaxLimit
}

// Page — результат выборки: записи от новых к старым.
type Page struct {
	Events     []Event `json:"events"`
	Truncated  bool    `json:"truncated"`  // подходящих записей больше, чем Limit
	WriteError string  `json:"writeError"` // последняя ошибка записи в журнал, если была
}

// Logger — потокобезопасный журнал поверх хранилища записей (Backend).
type Logger struct {
	mu       sync.Mutex
	backend  Backend
	now      func() time.Time
	writeErr string
}

// Open создаёт журнал в файлах каталога dir (каталог создаётся при первой записи).
func Open(dir string) *Logger {
	return New(NewFileBackend(dir))
}

// New создаёт журнал поверх произвольного хранилища (файлы, PostgreSQL и т. д.).
func New(b Backend) *Logger {
	return &Logger{backend: b, now: time.Now}
}

// clip обрезает строку по символам (не ломая UTF-8) и убирает управляющие символы:
// запись остаётся однострочной, а значения из чужого ввода (например, логин при неудачном входе)
// не могут подделать вид записи.
func clip(s string) string {
	s = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return ' '
		}
		return r
	}, s)
	if utf8.RuneCountInString(s) <= maxField {
		return s
	}
	r := []rune(s)
	return string(r[:maxField]) + "…"
}

// Record добавляет запись. Nil-логгер ничего не делает, чтобы вызывающему не требовались проверки.
// Ошибка записи не возвращается: её нельзя «починить» на месте, она видна через Query.
func (l *Logger) Record(e Event) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	if e.Time.IsZero() {
		e.Time = l.now()
	}
	e.Time = e.Time.UTC()
	e.User, e.Role, e.Action, e.Result, e.Detail = clip(e.User), clip(e.Role), clip(e.Action), clip(e.Result), clip(e.Detail)

	if err := l.append(e); err != nil {
		l.writeErr = err.Error()
		return
	}
	l.writeErr = ""
}

func (l *Logger) append(e Event) error {
	line, err := json.Marshal(e)
	if err != nil {
		return err
	}
	return l.backend.Append(line)
}

func parseTime(s, field string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("filter %s: expected a date in RFC3339 format", field)
	}
	return t, nil
}

func contains(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), needle)
}

// Query возвращает записи, подходящие под фильтр, от новых к старым.
func (l *Logger) Query(f Filter) (*Page, error) {
	since, err := parseTime(f.Since, "since")
	if err != nil {
		return nil, err
	}
	until, err := parseTime(f.Until, "until")
	if err != nil {
		return nil, err
	}
	limit := f.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	user, action, text := strings.ToLower(strings.TrimSpace(f.User)), strings.ToLower(strings.TrimSpace(f.Action)), strings.ToLower(strings.TrimSpace(f.Text))

	l.mu.Lock()
	defer l.mu.Unlock()

	page := &Page{Events: []Event{}, WriteError: l.writeErr}
	err = l.backend.ScanNewestFirst(func(line []byte) bool {
		var e Event
		if json.Unmarshal(line, &e) != nil {
			return true // повреждённая строка (например, обрыв записи) не должна скрывать остальные
		}
		if (!since.IsZero() && e.Time.Before(since)) || (!until.IsZero() && !e.Time.Before(until)) {
			return true
		}
		if f.Result != "" && e.Result != f.Result {
			return true
		}
		if (user != "" && !contains(e.User, user)) || (action != "" && !contains(e.Action, action)) || (text != "" && !contains(e.Detail, text)) {
			return true
		}
		if len(page.Events) == limit {
			page.Truncated = true
			return false
		}
		page.Events = append(page.Events, e)
		return true
	})
	if err != nil {
		return nil, err
	}
	return page, nil
}
