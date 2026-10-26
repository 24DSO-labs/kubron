// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	usersFileName  = "users.json"
	storageVersion = 1
)

type fileData struct {
	Version  int              `json:"version"`
	Users    []UserRecord     `json:"users"`
	Settings SecuritySettings `json:"settings"`
}

// fileBackend — хранилище в файле users.json каталога данных.
type fileBackend struct {
	path string
}

// NewFileBackend возвращает файловое хранилище пользователей в каталоге dir.
func NewFileBackend(dir string) Backend {
	return &fileBackend{path: filepath.Join(dir, usersFileName)}
}

// Load читает файл. Отсутствие файла — пустое хранилище (первый запуск).
func (b *fileBackend) Load() (*Data, error) {
	data, err := os.ReadFile(b.path)
	if errors.Is(err, os.ErrNotExist) {
		return &Data{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать %s: %w", b.path, err)
	}
	var fd fileData
	if err := json.Unmarshal(data, &fd); err != nil {
		return nil, fmt.Errorf("файл пользователей %s повреждён: %w", b.path, err)
	}
	if fd.Version > storageVersion {
		return nil, fmt.Errorf("файл пользователей %s создан более новой версией Kubron", b.path)
	}
	return &Data{Users: fd.Users, Settings: fd.Settings}, nil
}

// Save атомарно записывает файл: временный файл в той же папке, затем rename.
func (b *fileBackend) Save(d *Data) error {
	data, err := json.MarshalIndent(fileData{Version: storageVersion, Users: d.Users, Settings: d.Settings}, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(b.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, usersFileName+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }

	// os.CreateTemp создаёт файл с правами 0600, лишний chmod не нужен.
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(tmpName, b.path); err != nil {
		cleanup()
		return err
	}
	return nil
}
