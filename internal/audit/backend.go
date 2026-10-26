// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package audit

// Backend — хранилище записей журнала. Реализации: файлы с ротацией (файловая, десктоп)
// и, позже, PostgreSQL (сервер, ADR-0007). Вызывается под блокировкой Logger.
type Backend interface {
	// Append добавляет запись: одна JSON-строка без перевода строки в конце.
	Append(line []byte) error
	// ScanNewestFirst отдаёт записи (JSON-строки) от новых к старым, пока fn возвращает true.
	// Отсутствие журнала — не ошибка (пустой обход).
	ScanNewestFirst(fn func(line []byte) bool) error
}

// NewFileBackend возвращает файловое хранилище журнала в каталоге dir (с ротацией).
func NewFileBackend(dir string) Backend { return newFileBackend(dir) }
