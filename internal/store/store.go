// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

// Package store собирает хранилища приложения в одно целое (ADR-0007).
//
// Десктоп использует файловую реализацию (NewFile): users.json, audit.log и файлы токенов
// в каталоге данных. Серверная версия добавит реализацию на PostgreSQL с шифрованием секретов
// (ADR-0008) и подставит её без изменений в ядре.
//
// Хранение kubeconfig и описаний кластеров пока остаётся файловым внутри internal/k8s
// (их путь передаётся в client-go); вынос в Store — задача этапа 3.
package store

import (
	"analog/internal/audit"
	"analog/internal/auth"
	"analog/internal/k8s"
)

// Store — всё, что приложение хранит постоянно.
type Store interface {
	// Users — пользователи и настройки безопасности.
	Users() auth.Backend
	// Audit — журнал аудита.
	Audit() audit.Backend
	// ArgoTokens — токены Argo CD API по кластерам.
	ArgoTokens() k8s.TokenStore
}
