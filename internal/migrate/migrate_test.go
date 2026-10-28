// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package migrate

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"testing/fstest"
)

func mfs(files map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{}
	for name, body := range files {
		fsys["m/"+name] = &fstest.MapFile{Data: []byte(body)}
	}
	return fsys
}

func TestLoadOrdersAndChecksums(t *testing.T) {
	ms, err := Load(mfs(map[string]string{
		"0002_b.up.sql": "B up", "0002_b.down.sql": "B down",
		"0001_a.up.sql": "A up", "0001_a.down.sql": "A down",
	}), "m")
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 2 || ms[0].Version != 1 || ms[0].Name != "a" || ms[1].Version != 2 || ms[1].Down != "B down" {
		t.Fatalf("%+v", ms)
	}
	if ms[0].Checksum == "" || ms[0].Checksum == ms[1].Checksum {
		t.Error("контрольные суммы должны быть заданы и различаться")
	}
}

func TestLoadRejectsBadSets(t *testing.T) {
	cases := map[string]map[string]string{
		"пропуск версии":   {"0001_a.up.sql": "x", "0001_a.down.sql": "x", "0003_c.up.sql": "x", "0003_c.down.sql": "x"},
		"нет down":         {"0001_a.up.sql": "x"},
		"нет up":           {"0001_a.down.sql": "x"},
		"плохое имя файла": {"001_a.up.sql": "x"},
		"разные имена":     {"0001_a.up.sql": "x", "0001_b.down.sql": "x"},
		"нумерация с 2":    {"0002_a.up.sql": "x", "0002_a.down.sql": "x"},
	}
	for name, files := range cases {
		if _, err := Load(mfs(files), "m"); err == nil {
			t.Errorf("%s: ожидалась ошибка", name)
		}
	}
}

// fakeDB — журнал в памяти; записывает вызовы и умеет падать на заданной версии.
type fakeDB struct {
	applied []Applied
	calls   []string
	failOn  int
	locked  bool
}

func (f *fakeDB) Lock(context.Context) (func(), error) {
	f.locked = true
	f.calls = append(f.calls, "lock")
	return func() { f.locked = false; f.calls = append(f.calls, "unlock") }, nil
}
func (f *fakeDB) Ensure(context.Context) error { return nil }
func (f *fakeDB) Applied(context.Context) ([]Applied, error) {
	return append([]Applied(nil), f.applied...), nil
}
func (f *fakeDB) Apply(_ context.Context, m Migration, up bool) error {
	if !f.locked {
		return errors.New("Apply без блокировки")
	}
	if m.Version == f.failOn {
		return errors.New("boom")
	}
	if up {
		f.calls = append(f.calls, "up")
		f.applied = append(f.applied, Applied{m.Version, m.Name, m.Checksum})
		return nil
	}
	f.calls = append(f.calls, "down")
	for i, a := range f.applied {
		if a.Version == m.Version {
			f.applied = append(f.applied[:i], f.applied[i+1:]...)
			break
		}
	}
	return nil
}

func three() []Migration {
	mk := func(v int, n string) Migration {
		return Migration{Version: v, Name: n, Up: "u" + n, Down: "d" + n, Checksum: "sum-" + n}
	}
	return []Migration{mk(1, "a"), mk(2, "b"), mk(3, "c")}
}

func TestUpAppliesPendingInOrderUnderLock(t *testing.T) {
	db := &fakeDB{}
	m := &Migrator{DB: db, Migrations: three()}
	res, err := m.Up(context.Background())
	if err != nil || !reflect.DeepEqual(res.Applied, []int{1, 2, 3}) {
		t.Fatalf("%+v %v", res, err)
	}
	if db.calls[0] != "lock" || db.calls[len(db.calls)-1] != "unlock" {
		t.Errorf("блокировка должна охватывать все миграции: %v", db.calls)
	}
	// Повторный запуск ничего не делает.
	if res, err := m.Up(context.Background()); err != nil || len(res.Applied) != 0 {
		t.Errorf("повторный Up: %+v %v", res, err)
	}
}

func TestUpResumesAfterFailure(t *testing.T) {
	db := &fakeDB{failOn: 2}
	m := &Migrator{DB: db, Migrations: three()}
	res, err := m.Up(context.Background())
	if err == nil || !reflect.DeepEqual(res.Applied, []int{1}) {
		t.Fatalf("ожидали сбой на 2: %+v %v", res, err)
	}
	if db.locked {
		t.Error("блокировка должна сниматься и при ошибке")
	}
	db.failOn = 0
	if res, err := m.Up(context.Background()); err != nil || !reflect.DeepEqual(res.Applied, []int{2, 3}) {
		t.Fatalf("продолжение: %+v %v", res, err)
	}
}

func TestChecksumDriftIsRejected(t *testing.T) {
	db := &fakeDB{applied: []Applied{{1, "a", "other"}}}
	m := &Migrator{DB: db, Migrations: three()}
	if _, err := m.Up(context.Background()); !errors.Is(err, ErrChecksum) {
		t.Fatalf("ожидали ErrChecksum, получили %v", err)
	}
}

func TestGapInJournalIsRejected(t *testing.T) {
	db := &fakeDB{applied: []Applied{{2, "b", "sum-b"}}}
	m := &Migrator{DB: db, Migrations: three()}
	if _, err := m.Up(context.Background()); !errors.Is(err, ErrGap) {
		t.Fatalf("ожидали ErrGap, получили %v", err)
	}
}

// Реплику со старой версией кода запустили после того, как соседняя уже применила более новую миграцию.
func TestNewerDatabaseDoesNotBlockOlderReplica(t *testing.T) {
	db := &fakeDB{applied: []Applied{{1, "a", "sum-a"}, {2, "b", "sum-b"}, {3, "c", "sum-c"}, {4, "d", "sum-d"}}}
	m := &Migrator{DB: db, Migrations: three()}
	res, err := m.Up(context.Background())
	if err != nil || len(res.Applied) != 0 || !reflect.DeepEqual(res.Ahead, []int{4}) {
		t.Fatalf("%+v %v", res, err)
	}
	if _, err := m.Down(context.Background(), 1); !errors.Is(err, ErrGap) {
		t.Errorf("старая версия не должна откатывать схему, которую не знает: %v", err)
	}
}

func TestDownRollsBackNewestFirst(t *testing.T) {
	db := &fakeDB{}
	m := &Migrator{DB: db, Migrations: three()}
	if _, err := m.Up(context.Background()); err != nil {
		t.Fatal(err)
	}
	done, err := m.Down(context.Background(), 2)
	if err != nil || !reflect.DeepEqual(done, []int{3, 2}) {
		t.Fatalf("%v %v", done, err)
	}
	done, err = m.Down(context.Background(), 0) // все оставшиеся
	if err != nil || !reflect.DeepEqual(done, []int{1}) {
		t.Fatalf("%v %v", done, err)
	}
	st, _ := m.Status(context.Background())
	for _, s := range st {
		if s.Applied {
			t.Errorf("после полного отката применена %+v", s)
		}
	}
}
