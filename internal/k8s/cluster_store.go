// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ClusterStore — хранилище кластеров: метаданные и kubeconfig (ADR-0007, этап 3).
// Реализации: файлы в каталоге данных (десктоп, личный режим) и PostgreSQL с шифрованием kubeconfig (сервис).
//
// Идентификаторы проверяет вызывающий (validClusterID). Get и Kubeconfig для отсутствующего кластера
// возвращают ошибку; остальные детали (формат ошибок) — на совести реализации, но сообщения
// файловой реализации сохранены такими же, как до выноса в интерфейс.
type ClusterStore interface {
	// List возвращает кластеры, у которых есть и метаданные, и kubeconfig.
	List() ([]ClusterMeta, error)
	// Get возвращает метаданные кластера.
	Get(id string) (*ClusterMeta, error)
	// Kubeconfig возвращает сохранённый kubeconfig кластера.
	Kubeconfig(id string) ([]byte, error)
	// Save сохраняет метаданные и, если kubeconfig != nil, заменяет kubeconfig кластера. Операция
	// атомарна для вызывающего: если метаданные записать не удалось, прежний kubeconfig восстанавливается
	// (а у нового кластера удаляется).
	Save(meta *ClusterMeta, kubeconfig []byte) error
	// Delete удаляет кластер (метаданные и kubeconfig). Отсутствие кластера не ошибка.
	Delete(id string) error
}

// kubeconfigPather — необязательная возможность хранилища: kubeconfig лежит файлом на диске. Тогда client-go и Helm
// читают его по пути, как раньше (относительные пути внутри kubeconfig резолвятся от его каталога);
// остальные хранилища отдают байты, и файл на диске сервера не создаётся.
type kubeconfigPather interface {
	KubeconfigPath(id string) string
}

// clusterStore — текущее хранилище кластеров. По умолчанию файловое.
var clusterStore ClusterStore = fileClusterStore{}

// NewFileClusterStore возвращает файловое хранилище кластеров (каталог clusters в каталоге данных).
func NewFileClusterStore() ClusterStore { return fileClusterStore{} }

// SetClusterStore подменяет хранилище кластеров (вызывается при запуске приложения).
func SetClusterStore(s ClusterStore) { clusterStore = s }

// fileClusterStore — <данные>/clusters/<id>.json и <id>.kubeconfig (0600).
type fileClusterStore struct{}

func (fileClusterStore) List() ([]ClusterMeta, error) {
	if err := InitKubronDir(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(getClustersDir())
	if err != nil {
		return nil, nil
	}

	var clusters []ClusterMeta
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(getClustersDir(), entry.Name()))
		if err != nil {
			continue
		}

		var meta ClusterMeta
		if err := json.Unmarshal(data, &meta); err != nil {
			continue
		}

		kcPath := filepath.Join(getClustersDir(), meta.KubeconfigFile)
		if _, err := os.Stat(kcPath); os.IsNotExist(err) {
			continue
		}

		clusters = append(clusters, meta)
	}

	return clusters, nil
}

func (fileClusterStore) Get(id string) (*ClusterMeta, error) {
	if !validClusterID(id) {
		return nil, fmt.Errorf("invalid cluster ID %q", id)
	}
	data, err := os.ReadFile(filepath.Join(getClustersDir(), id+".json"))
	if err != nil {
		return nil, err
	}
	var meta ClusterMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// KubeconfigPath — путь к файлу kubeconfig кластера; пусто, если кластера нет.
func (s fileClusterStore) KubeconfigPath(id string) string {
	meta, err := s.Get(id)
	if err != nil {
		return ""
	}
	return filepath.Join(getClustersDir(), meta.KubeconfigFile)
}

func (s fileClusterStore) Kubeconfig(id string) ([]byte, error) {
	path := s.KubeconfigPath(id)
	if path == "" {
		return nil, fmt.Errorf("kubeconfig not found for cluster %s", id)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig: %w", err)
	}
	return data, nil
}

func (fileClusterStore) Save(meta *ClusterMeta, kubeconfig []byte) error {
	if !validClusterID(meta.ID) {
		return fmt.Errorf("invalid cluster ID %q", meta.ID)
	}
	if err := os.MkdirAll(getClustersDir(), 0o700); err != nil {
		return fmt.Errorf("failed to create %s: %w", getClustersDir(), err)
	}
	if meta.KubeconfigFile == "" {
		meta.KubeconfigFile = meta.ID + ".kubeconfig"
	}
	if filepath.Base(meta.KubeconfigFile) != meta.KubeconfigFile {
		return fmt.Errorf("invalid kubeconfig file name %q", meta.KubeconfigFile)
	}

	kcPath := filepath.Join(getClustersDir(), meta.KubeconfigFile)
	var old []byte
	var readErr error
	if kubeconfig != nil {
		old, readErr = os.ReadFile(kcPath)
		if err := writeFileAtomic(kcPath, kubeconfig, 0o600); err != nil {
			return fmt.Errorf("failed to save kubeconfig: %w", err)
		}
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err == nil {
		err = writeFileAtomic(filepath.Join(getClustersDir(), meta.ID+".json"), data, 0o600)
	}
	if err != nil {
		if kubeconfig != nil {
			if readErr == nil {
				_ = writeFileAtomic(kcPath, old, 0o600) // возвращаем прежний kubeconfig
			} else {
				_ = os.Remove(kcPath) // кластер новый: файл без метаданных не нужен
			}
		}
		return err
	}
	return nil
}

func (fileClusterStore) Delete(id string) error {
	dir := getClustersDir()
	for _, name := range []string{id + ".json", id + ".kubeconfig"} {
		if err := os.Remove(filepath.Join(dir, name)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to delete %s: %w", name, err)
		}
	}
	return nil
}
