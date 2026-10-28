// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

// Package migrate — тонкий мигратор SQL-схемы (ADR-0007): файлы NNNN_имя.up.sql / .down.sql, журнал
// применённых версий с контрольными суммами, исполнение под блокировкой БД (одна реплика за раз).
//
// Пакет не зависит от драйвера: работает через интерфейс DB. Реализация для PostgreSQL поверх
// database/sql — в postgres.go; логика (порядок, дрейф, откат) тестируется на поддельной БД.
//
// Совместимость при обновлении реплик по очереди: если в БД есть версии новее, чем знает этот
// экземпляр (его уже обновили соседи), Up не падает — старая версия приложения продолжает работать,
// а миграции обязаны оставаться обратно совместимыми (см. Result.Ahead).
package migrate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strconv"
)

// Migration — одна версия схемы.
type Migration struct {
	Version  int
	Name     string
	Up, Down string
	Checksum string // SHA-256 текста Up: правка применённой миграции обнаруживается
}

// Applied — запись журнала применённых миграций.
type Applied struct {
	Version  int
	Name     string
	Checksum string
}

// DB — то, что нужно мигратору от базы данных. Все вызовы идут через одно соединение.
type DB interface {
	// Lock берёт блокировку миграций (pg_advisory_lock) и возвращает функцию её снятия.
	Lock(ctx context.Context) (unlock func(), err error)
	// Ensure создаёт журнал миграций, если его нет.
	Ensure(ctx context.Context) error
	// Applied возвращает применённые миграции по возрастанию версии.
	Applied(ctx context.Context) ([]Applied, error)
	// Apply выполняет миграцию (up или down) и записывает/удаляет строку журнала в одной транзакции.
	Apply(ctx context.Context, m Migration, up bool) error
}

var fileRe = regexp.MustCompile(`^(\d{4})_([a-z0-9_]+)\.(up|down)\.sql$`)

// Load читает миграции из каталога dir файловой системы fsys (обычно embed.FS).
// Версии должны идти подряд с 1; у каждой есть и up, и down (откат проверяется в CI).
func Load(fsys fs.FS, dir string) ([]Migration, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("миграции: %w", err)
	}
	byVersion := map[int]*Migration{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := fileRe.FindStringSubmatch(e.Name())
		if m == nil {
			return nil, fmt.Errorf("миграции: недопустимое имя файла %q (ожидается NNNN_имя.up.sql / .down.sql)", e.Name())
		}
		v, _ := strconv.Atoi(m[1])
		body, err := fs.ReadFile(fsys, path.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("миграции: %w", err)
		}
		cur := byVersion[v]
		if cur == nil {
			cur = &Migration{Version: v, Name: m[2]}
			byVersion[v] = cur
		}
		if cur.Name != m[2] {
			return nil, fmt.Errorf("миграции: версия %04d имеет разные имена: %q и %q", v, cur.Name, m[2])
		}
		if m[3] == "up" {
			cur.Up = string(body)
		} else {
			cur.Down = string(body)
		}
	}

	out := make([]Migration, 0, len(byVersion))
	for _, m := range byVersion {
		out = append(out, *m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	for i := range out {
		if want := i + 1; out[i].Version != want {
			return nil, fmt.Errorf("миграции: ожидалась версия %04d, найдена %04d (нумерация без пропусков)", want, out[i].Version)
		}
		if out[i].Up == "" || out[i].Down == "" {
			return nil, fmt.Errorf("миграции: у версии %04d_%s должны быть и up, и down", out[i].Version, out[i].Name)
		}
		sum := sha256.Sum256([]byte(out[i].Up))
		out[i].Checksum = hex.EncodeToString(sum[:])
	}
	return out, nil
}

// Ошибки мигратора.
var (
	// ErrChecksum — применённая миграция отличается от файла: её правили после применения.
	ErrChecksum = errors.New("миграция изменена после применения")
	// ErrGap — в журнале БД пропущена версия или есть неизвестная миграция среди известных.
	ErrGap = errors.New("журнал миграций не согласован с набором миграций")
)

// Migrator применяет и откатывает миграции.
type Migrator struct {
	DB         DB
	Migrations []Migration
}

// Result — итог Up.
type Result struct {
	// Applied — версии, применённые этим вызовом.
	Applied []int
	// Ahead — версии, которые уже есть в БД, но неизвестны этому экземпляру (БД обновили более новые реплики).
	Ahead []int
}

// state читает журнал и сверяет его с известными миграциями. Вызывать под блокировкой.
func (m *Migrator) state(ctx context.Context) (applied map[int]Applied, ahead []int, err error) {
	if err := m.DB.Ensure(ctx); err != nil {
		return nil, nil, err
	}
	list, err := m.DB.Applied(ctx)
	if err != nil {
		return nil, nil, err
	}
	known := map[int]Migration{}
	for _, mg := range m.Migrations {
		known[mg.Version] = mg
	}
	applied = map[int]Applied{}
	for _, a := range list {
		mg, ok := known[a.Version]
		if !ok {
			ahead = append(ahead, a.Version)
			continue
		}
		if a.Checksum != mg.Checksum {
			return nil, nil, fmt.Errorf("%w: %04d_%s", ErrChecksum, a.Version, a.Name)
		}
		applied[a.Version] = a
	}
	// Среди известных применённые идут префиксом: дыра означает ручное вмешательство в журнал.
	seenGap := false
	for _, mg := range m.Migrations {
		if _, ok := applied[mg.Version]; ok {
			if seenGap {
				return nil, nil, fmt.Errorf("%w: версия %04d применена, а предыдущая нет", ErrGap, mg.Version)
			}
		} else {
			seenGap = true
		}
	}
	sort.Ints(ahead)
	return applied, ahead, nil
}

// Up применяет все ожидающие миграции по порядку под блокировкой БД.
func (m *Migrator) Up(ctx context.Context) (Result, error) {
	unlock, err := m.DB.Lock(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("блокировка миграций: %w", err)
	}
	defer unlock()

	applied, ahead, err := m.state(ctx)
	if err != nil {
		return Result{}, err
	}
	res := Result{Ahead: ahead}
	for _, mg := range m.Migrations {
		if _, ok := applied[mg.Version]; ok {
			continue
		}
		if err := m.DB.Apply(ctx, mg, true); err != nil {
			return res, fmt.Errorf("миграция %04d_%s: %w", mg.Version, mg.Name, err)
		}
		res.Applied = append(res.Applied, mg.Version)
	}
	return res, nil
}

// Down откатывает последние steps применённых миграций (steps <= 0 — все). Версии, неизвестные
// этому экземпляру, не откатываются: откат новых миграций делает версия приложения, которая их знает.
func (m *Migrator) Down(ctx context.Context, steps int) ([]int, error) {
	unlock, err := m.DB.Lock(ctx)
	if err != nil {
		return nil, fmt.Errorf("блокировка миграций: %w", err)
	}
	defer unlock()

	applied, ahead, err := m.state(ctx)
	if err != nil {
		return nil, err
	}
	if len(ahead) > 0 {
		return nil, fmt.Errorf("%w: в БД есть версии новее известных (%v), откат выполняет новая версия приложения", ErrGap, ahead)
	}
	var done []int
	for i := len(m.Migrations) - 1; i >= 0; i-- {
		mg := m.Migrations[i]
		if _, ok := applied[mg.Version]; !ok {
			continue
		}
		if steps > 0 && len(done) >= steps {
			break
		}
		if err := m.DB.Apply(ctx, mg, false); err != nil {
			return done, fmt.Errorf("откат %04d_%s: %w", mg.Version, mg.Name, err)
		}
		done = append(done, mg.Version)
	}
	return done, nil
}

// Status — состояние каждой известной миграции.
type Status struct {
	Version int
	Name    string
	Applied bool
}

// Status возвращает состояние известных миграций (без блокировки: только чтение журнала).
func (m *Migrator) Status(ctx context.Context) ([]Status, error) {
	applied, _, err := m.state(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Status, 0, len(m.Migrations))
	for _, mg := range m.Migrations {
		_, ok := applied[mg.Version]
		out = append(out, Status{Version: mg.Version, Name: mg.Name, Applied: ok})
	}
	return out, nil
}
