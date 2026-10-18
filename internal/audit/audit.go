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
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
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

// Logger — потокобезопасный журнал.
type Logger struct {
	mu       sync.Mutex
	dir      string
	now      func() time.Time
	maxSize  int64
	writeErr string
}

// Open создаёт журнал в каталоге dir (каталог создаётся при первой записи).
func Open(dir string) *Logger {
	return &Logger{dir: dir, now: time.Now, maxSize: maxFileSize}
}

func (l *Logger) path(i int) string {
	if i == 0 {
		return filepath.Join(l.dir, fileName)
	}
	return filepath.Join(l.dir, fmt.Sprintf("%s.%d", fileName, i))
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

	if err := l.appendLocked(e); err != nil {
		l.writeErr = err.Error()
		return
	}
	l.writeErr = ""
}

func (l *Logger) appendLocked(e Event) error {
	line, err := json.Marshal(e)
	if err != nil {
		return err
	}
	line = append(line, '\n')

	if err := os.MkdirAll(l.dir, 0o700); err != nil {
		return err
	}
	if st, err := os.Stat(l.path(0)); err == nil && st.Size()+int64(len(line)) > l.maxSize {
		if err := l.rotateLocked(); err != nil {
			return err
		}
	}
	f, err := os.OpenFile(l.path(0), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(line); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func (l *Logger) rotateLocked() error {
	if err := os.Remove(l.path(keepFiles - 1)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for i := keepFiles - 2; i >= 0; i-- {
		if err := os.Rename(l.path(i), l.path(i+1)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func parseTime(s, field string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("фильтр %s: ожидается дата в формате RFC3339", field)
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
	for i := 0; i < keepFiles; i++ {
		lines, err := readLines(l.path(i))
		if err != nil {
			return nil, err
		}
		for j := len(lines) - 1; j >= 0; j-- {
			var e Event
			if json.Unmarshal(lines[j], &e) != nil {
				continue // повреждённая строка (например, обрыв записи) не должна скрывать остальные
			}
			if (!since.IsZero() && e.Time.Before(since)) || (!until.IsZero() && !e.Time.Before(until)) {
				continue
			}
			if f.Result != "" && e.Result != f.Result {
				continue
			}
			if (user != "" && !contains(e.User, user)) || (action != "" && !contains(e.Action, action)) || (text != "" && !contains(e.Detail, text)) {
				continue
			}
			if len(page.Events) == limit {
				page.Truncated = true
				return page, nil
			}
			page.Events = append(page.Events, e)
		}
	}
	return page, nil
}

// readLines читает непустые строки файла; отсутствие файла — не ошибка.
func readLines(path string) ([][]byte, error) {
	f, err := os.Open(path)
	// ENOTDIR: на пути лежит файл вместо каталога — журнала там тоже нет.
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать журнал: %w", err)
	}
	defer f.Close()

	var out [][]byte
	r := bufio.NewReaderSize(f, 64<<10)
	for {
		line, err := r.ReadBytes('\n')
		if len(strings.TrimSpace(string(line))) > 0 {
			out = append(out, line)
		}
		if err != nil {
			break
		}
	}
	return out, nil
}
