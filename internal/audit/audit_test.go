// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package audit

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
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
		t.Fatalf("expected newest-first order: %+v", p.Events)
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
		{"no filter", Filter{}, 3},
		{"user, case-insensitive", Filter{User: "ADMIN"}, 1},
		{"action", Filter{Action: "delete"}, 2},
		{"result", Filter{Result: ResultDenied}, 1},
		{"text in detail", Filter{Text: "api-2"}, 1},
		{"since date", Filter{Since: base.Add(time.Hour).Format(time.RFC3339)}, 2},
		{"until date (exclusive)", Filter{Until: base.Add(time.Hour).Format(time.RFC3339)}, 1},
		{"combination", Filter{User: "bob", Action: "delete"}, 1},
	}
	for _, c := range cases {
		p, err := l.Query(c.f)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if len(p.Events) != c.want {
			t.Errorf("%s: got %d, expected %d", c.name, len(p.Events), c.want)
		}
	}
	if _, err := l.Query(Filter{Since: "yesterday"}); err == nil {
		t.Error("an invalid date must give an error")
	}
}

func TestLimitAndTruncated(t *testing.T) {
	l := newLogger(t)
	for i := 0; i < 5; i++ {
		l.Record(Event{User: "a", Action: "X", Result: ResultOK})
	}
	p, _ := l.Query(Filter{Limit: 3})
	if len(p.Events) != 3 || !p.Truncated {
		t.Errorf("limit=3: %d records, truncated=%v", len(p.Events), p.Truncated)
	}
	p, _ = l.Query(Filter{Limit: 5})
	if len(p.Events) != 5 || p.Truncated {
		t.Errorf("limit=5: %d records, truncated=%v", len(p.Events), p.Truncated)
	}
	p, _ = l.Query(Filter{Limit: 1 << 20})
	if len(p.Events) != 5 {
		t.Errorf("a huge limit must be capped by MaxLimit, got %d", len(p.Events))
	}
}

func TestFieldsAreSanitized(t *testing.T) {
	l := newLogger(t)
	evil := "bob\n{\"user\":\"admin\",\"action\":\"Fake\"}\r\x00" + strings.Repeat("я", 1000)
	l.Record(Event{User: evil, Action: "Login", Result: ResultFailed})

	data, err := os.ReadFile(filepath.Join(fb(l).dir, fileName))
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(data), "\n"); n != 1 {
		t.Fatalf("a record must occupy one line, newlines: %d", n)
	}
	p, _ := l.Query(Filter{})
	if len(p.Events) != 1 || p.Events[0].Action != "Login" {
		t.Fatalf("forging a record through input must not be possible: %+v", p.Events)
	}
	if got := []rune(p.Events[0].User); len(got) > maxField+1 {
		t.Errorf("field was not truncated: %d characters", len(got))
	}
}

func TestRotation(t *testing.T) {
	l := newLogger(t)
	fb(l).maxSize = 300
	for i := 0; i < 60; i++ {
		l.Record(Event{User: "u", Action: "Act", Result: ResultOK, Detail: strings.Repeat("x", 50)})
	}
	files, _ := filepath.Glob(filepath.Join(fb(l).dir, fileName+"*"))
	if len(files) < 2 || len(files) > keepFiles {
		t.Fatalf("expected rotation to %d files, found %d", keepFiles, len(files))
	}
	for _, f := range files {
		if st, _ := os.Stat(f); st.Size() > 400 {
			t.Errorf("%s grew to %d bytes", f, st.Size())
		}
	}
	// старые записи уходят, но свежие читаются сквозь все файлы
	p, err := l.Query(Filter{Limit: MaxLimit})
	if err != nil || len(p.Events) == 0 {
		t.Fatalf("reading after rotation: %v, %d records", err, len(p.Events))
	}
}

func TestCorruptLineDoesNotHideOthers(t *testing.T) {
	l := newLogger(t)
	l.Record(Event{User: "a", Action: "First", Result: ResultOK})
	f, _ := os.OpenFile(filepath.Join(fb(l).dir, fileName), os.O_APPEND|os.O_WRONLY, 0o600)
	f.WriteString("{truncated record\n")
	f.Close()
	l.Record(Event{User: "a", Action: "Second", Result: ResultOK})

	p, err := l.Query(Filter{})
	if err != nil || len(p.Events) != 2 {
		t.Fatalf("err=%v, records=%d, expected 2", err, len(p.Events))
	}
}

func TestFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not support POSIX permissions: os.Stat always returns 0666")
	}
	l := newLogger(t)
	l.Record(Event{User: "a", Action: "X", Result: ResultOK})
	st, err := os.Stat(filepath.Join(fb(l).dir, fileName))
	if err != nil {
		t.Fatal(err)
	}
	if perm := st.Mode().Perm(); perm&0o077 != 0 {
		t.Errorf("the journal is accessible to other OS users: %v", perm)
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
		t.Error("a write error must be visible in Page.WriteError")
	}
}

func TestNilLoggerIsSafe(t *testing.T) {
	var l *Logger
	l.Record(Event{Action: "X"}) // не паникует
}

// fb — файловое хранилище журнала, созданное Open.
func fb(l *Logger) *fileBackend { return l.backend.(*fileBackend) }

func TestContextFilterAndScope(t *testing.T) {
	l := New(NewFileBackend(t.TempDir()))
	l.Record(Event{User: "a", Action: "DeletePod", Result: ResultOK, TeamID: "red", ClusterID: "c1"})
	l.Record(Event{User: "b", Action: "DeletePod", Result: ResultOK, TeamID: "blue", ClusterID: "c3"})
	l.Record(Event{User: "c", Action: "Login", Result: ResultOK}) // без команды

	got := func(f Filter) []string {
		p, err := l.Query(f)
		if err != nil {
			t.Fatal(err)
		}
		var users []string
		for _, e := range p.Events {
			users = append(users, e.User)
		}
		return users
	}
	if n := len(got(Filter{})); n != 3 {
		t.Errorf("без фильтра: %d", n)
	}
	if u := got(Filter{Team: "red"}); len(u) != 1 || u[0] != "a" {
		t.Errorf("по команде: %v", u)
	}
	if u := got(Filter{Cluster: "c3"}); len(u) != 1 || u[0] != "b" {
		t.Errorf("по кластеру: %v", u)
	}
	// Ограничение доверенного кода: администратор команды не видит чужие события и события без команды,
	// а запросить чужую команду фильтром не может.
	if u := got(Filter{Scoped: true, Scope: []string{"red"}}); len(u) != 1 || u[0] != "a" {
		t.Errorf("scope: %v", u)
	}
	if u := got(Filter{Scoped: true, Scope: []string{"red"}, Team: "blue"}); len(u) != 0 {
		t.Errorf("чужая команда через фильтр: %v", u)
	}
	if u := got(Filter{Scoped: true}); len(u) != 0 {
		t.Errorf("пустой scope не видит ничего: %v", u)
	}
}

func TestExportFormatsAndFormulaInjection(t *testing.T) {
	l := New(NewFileBackend(t.TempDir()))
	l.Record(Event{User: "=HYPERLINK(\"http://evil\")", Action: "Login", Result: ResultFailed, Detail: "x, \"y\"\nz", TeamID: "red"})
	l.Record(Event{User: "ok", Action: "Logout", Result: ResultOK})

	data, n, trunc, err := l.Export(Filter{}, FormatCSV)
	if err != nil || n != 2 || trunc {
		t.Fatalf("%d %v %v", n, trunc, err)
	}
	r := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(data, []byte("\xEF\xBB\xBF"))))
	rows, err := r.ReadAll()
	if err != nil || len(rows) != 3 || rows[0][0] != "time" {
		t.Fatalf("%v %v", rows, err)
	}
	var evil []string
	for _, row := range rows[1:] {
		if strings.Contains(row[1], "HYPERLINK") {
			evil = row
		}
	}
	if evil == nil || !strings.HasPrefix(evil[1], "'=") || evil[5] != "x, \"y\" z" || evil[7] != "red" {
		t.Errorf("формулы должны экранироваться, остальное — без искажений: %q", evil)
	}

	data, n, _, err = l.Export(Filter{Team: "red"}, FormatJSON)
	var back []Event
	if err != nil || n != 1 || json.Unmarshal(data, &back) != nil || len(back) != 1 || back[0].TeamID != "red" {
		t.Errorf("json: %s %d %v", data, n, err)
	}
	if _, _, _, err := l.Export(Filter{}, "xml"); err == nil {
		t.Error("неизвестный формат")
	}
}
