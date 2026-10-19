package audit

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func newLogger(t *testing.T) *Logger {
	t.Helper()
	return Open(t.TempDir())
}

func TestRecordAndQueryNewestFirst(t *testing.T) {
	l := newLogger(t)
	base := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	for i, a := range []string{"Login", "DeletePod", "ScaleDeployment"} {
		l.Record(Event{Time: base.Add(time.Duration(i) * time.Minute), User: "admin", Action: a, Result: ResultOK})
	}
	p, err := l.Query(Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Events) != 3 || p.Events[0].Action != "ScaleDeployment" || p.Events[2].Action != "Login" {
		t.Fatalf("ожидался порядок от новых к старым: %+v", p.Events)
	}
}

func TestFilters(t *testing.T) {
	l := newLogger(t)
	base := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	l.Record(Event{Time: base, User: "Admin", Action: "DeletePod", Result: ResultOK, Detail: "namespace=prod name=api-1"})
	l.Record(Event{Time: base.Add(time.Hour), User: "bob", Action: "DeletePod", Result: ResultDenied, Detail: "namespace=prod name=api-2"})
	l.Record(Event{Time: base.Add(2 * time.Hour), User: "bob", Action: "Login", Result: ResultOK})

	cases := []struct {
		name string
		f    Filter
		want int
	}{
		{"без фильтра", Filter{}, 3},
		{"пользователь без учёта регистра", Filter{User: "ADMIN"}, 1},
		{"действие", Filter{Action: "delete"}, 2},
		{"результат", Filter{Result: ResultDenied}, 1},
		{"текст в detail", Filter{Text: "api-2"}, 1},
		{"с даты", Filter{Since: base.Add(time.Hour).Format(time.RFC3339)}, 2},
		{"до даты (не включая)", Filter{Until: base.Add(time.Hour).Format(time.RFC3339)}, 1},
		{"сочетание", Filter{User: "bob", Action: "delete"}, 1},
	}
	for _, c := range cases {
		p, err := l.Query(c.f)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if len(p.Events) != c.want {
			t.Errorf("%s: получено %d, ожидалось %d", c.name, len(p.Events), c.want)
		}
	}
	if _, err := l.Query(Filter{Since: "вчера"}); err == nil {
		t.Error("некорректная дата должна давать ошибку")
	}
}

func TestLimitAndTruncated(t *testing.T) {
	l := newLogger(t)
	for i := 0; i < 5; i++ {
		l.Record(Event{User: "a", Action: "X", Result: ResultOK})
	}
	p, _ := l.Query(Filter{Limit: 3})
	if len(p.Events) != 3 || !p.Truncated {
		t.Errorf("limit=3: %d записей, truncated=%v", len(p.Events), p.Truncated)
	}
	p, _ = l.Query(Filter{Limit: 5})
	if len(p.Events) != 5 || p.Truncated {
		t.Errorf("limit=5: %d записей, truncated=%v", len(p.Events), p.Truncated)
	}
	p, _ = l.Query(Filter{Limit: 1 << 20})
	if len(p.Events) != 5 {
		t.Errorf("огромный limit должен ограничиваться MaxLimit, получено %d", len(p.Events))
	}
}

func TestFieldsAreSanitized(t *testing.T) {
	l := newLogger(t)
	evil := "bob\n{\"user\":\"admin\",\"action\":\"Fake\"}\r\x00" + strings.Repeat("я", 1000)
	l.Record(Event{User: evil, Action: "Login", Result: ResultFailed})

	data, err := os.ReadFile(filepath.Join(l.dir, fileName))
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(data), "\n"); n != 1 {
		t.Fatalf("запись должна занимать одну строку, переводов строки: %d", n)
	}
	p, _ := l.Query(Filter{})
	if len(p.Events) != 1 || p.Events[0].Action != "Login" {
		t.Fatalf("подделать запись через ввод не должно получаться: %+v", p.Events)
	}
	if got := []rune(p.Events[0].User); len(got) > maxField+1 {
		t.Errorf("поле не обрезано: %d символов", len(got))
	}
}

func TestRotation(t *testing.T) {
	l := newLogger(t)
	l.maxSize = 300
	for i := 0; i < 60; i++ {
		l.Record(Event{User: "u", Action: "Act", Result: ResultOK, Detail: strings.Repeat("x", 50)})
	}
	files, _ := filepath.Glob(filepath.Join(l.dir, fileName+"*"))
	if len(files) < 2 || len(files) > keepFiles {
		t.Fatalf("ожидалась ротация до %d файлов, найдено %d", keepFiles, len(files))
	}
	for _, f := range files {
		if st, _ := os.Stat(f); st.Size() > 400 {
			t.Errorf("%s вырос до %d байт", f, st.Size())
		}
	}
	// старые записи уходят, но свежие читаются сквозь все файлы
	p, err := l.Query(Filter{Limit: MaxLimit})
	if err != nil || len(p.Events) == 0 {
		t.Fatalf("чтение после ротации: %v, %d записей", err, len(p.Events))
	}
}

func TestCorruptLineDoesNotHideOthers(t *testing.T) {
	l := newLogger(t)
	l.Record(Event{User: "a", Action: "First", Result: ResultOK})
	f, _ := os.OpenFile(filepath.Join(l.dir, fileName), os.O_APPEND|os.O_WRONLY, 0o600)
	f.WriteString("{обрыв записи\n")
	f.Close()
	l.Record(Event{User: "a", Action: "Second", Result: ResultOK})

	p, err := l.Query(Filter{})
	if err != nil || len(p.Events) != 2 {
		t.Fatalf("err=%v, записей=%d, ожидалось 2", err, len(p.Events))
	}
}

func TestFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows не поддерживает POSIX-права доступа: os.Stat всегда возвращает 0666")
	}
	l := newLogger(t)
	l.Record(Event{User: "a", Action: "X", Result: ResultOK})
	st, err := os.Stat(filepath.Join(l.dir, fileName))
	if err != nil {
		t.Fatal(err)
	}
	if perm := st.Mode().Perm(); perm&0o077 != 0 {
		t.Errorf("журнал доступен другим пользователям ОС: %v", perm)
	}
}

func TestWriteErrorIsReportedNotFatal(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "file")
	os.WriteFile(blocker, nil, 0o600)
	l := Open(filepath.Join(blocker, "sub")) // каталог нельзя создать: родитель — файл

	l.Record(Event{User: "a", Action: "X", Result: ResultOK}) // не должно паниковать
	p, err := l.Query(Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if p.WriteError == "" {
		t.Error("ошибка записи должна быть видна в Page.WriteError")
	}
}

func TestNilLoggerIsSafe(t *testing.T) {
	var l *Logger
	l.Record(Event{Action: "X"}) // не паникует
}
