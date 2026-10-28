// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

// Package pgstore — хранилище сервиса на PostgreSQL (ADR-0007). Сейчас здесь схема и миграции (задача 3.1a);
// реализация Store поверх pgx появится в задаче 3.1b.
package pgstore

import (
	"embed"

	"github.com/VadimIl224/kubron/internal/migrate"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrations возвращает встроенные миграции схемы.
func Migrations() ([]migrate.Migration, error) {
	return migrate.Load(migrationsFS, "migrations")
}
