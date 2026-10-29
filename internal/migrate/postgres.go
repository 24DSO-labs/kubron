// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package migrate

import (
	"context"
	"database/sql"
	"fmt"
)

// lockKey — ключ рекомендательной блокировки миграций (ASCII «kubr»). Одна и та же у всех реплик.
const lockKey int64 = 0x6B756272

// Запросы вынесены в константы: CI проверяет их на настоящем PostgreSQL (scripts/check-migrations.sh).
const (
	sqlLock    = `SELECT pg_advisory_lock($1)`
	sqlUnlock  = `SELECT pg_advisory_unlock($1)`
	sqlEnsure  = `CREATE TABLE IF NOT EXISTS schema_migrations (version integer PRIMARY KEY, name text NOT NULL, checksum text NOT NULL, applied_at timestamptz NOT NULL DEFAULT now())`
	sqlApplied = `SELECT version, name, checksum FROM schema_migrations ORDER BY version`
	sqlRecord  = `INSERT INTO schema_migrations (version, name, checksum) VALUES ($1, $2, $3)`
	sqlForget  = `DELETE FROM schema_migrations WHERE version = $1`
)

// Postgres — DB поверх выделенного соединения database/sql (драйвер подключает вызывающий, например pgx/stdlib).
// Блокировка рекомендательная и сессионная: она живёт, пока открыто это соединение, поэтому все операции
// идут через один *sql.Conn.
type Postgres struct{ Conn *sql.Conn }

func (p Postgres) Lock(ctx context.Context) (func(), error) {
	if _, err := p.Conn.ExecContext(ctx, sqlLock, lockKey); err != nil {
		return nil, err
	}
	return func() {
		// Контекст вызова мог быть отменён: снимаем блокировку независимо от него.
		_, _ = p.Conn.ExecContext(context.Background(), sqlUnlock, lockKey)
	}, nil
}

func (p Postgres) Ensure(ctx context.Context) error {
	_, err := p.Conn.ExecContext(ctx, sqlEnsure)
	return err
}

func (p Postgres) Applied(ctx context.Context) ([]Applied, error) {
	rows, err := p.Conn.QueryContext(ctx, sqlApplied)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Applied
	for rows.Next() {
		var a Applied
		if err := rows.Scan(&a.Version, &a.Name, &a.Checksum); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// Apply выполняет текст миграции и меняет журнал в одной транзакции (DDL в PostgreSQL транзакционен).
// Текст может содержать несколько операторов: параметров у него нет, драйвер отправляет его целиком.
func (p Postgres) Apply(ctx context.Context, m Migration, up bool) error {
	tx, err := p.Conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }() // после Commit возвращает ErrTxDone, это не ошибка

	body := m.Down
	if up {
		body = m.Up
	}
	if _, err := tx.ExecContext(ctx, body); err != nil {
		return fmt.Errorf("execute: %w", err)
	}
	if up {
		_, err = tx.ExecContext(ctx, sqlRecord, m.Version, m.Name, m.Checksum)
	} else {
		_, err = tx.ExecContext(ctx, sqlForget, m.Version)
	}
	if err != nil {
		return fmt.Errorf("journal: %w", err)
	}
	return tx.Commit()
}
