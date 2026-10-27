// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type KubronConfig struct {
	LastActiveCluster   string `json:"lastActiveCluster"`
	Theme               string `json:"theme"`
	AutoRefreshInterval int    `json:"autoRefreshInterval"` // секунды, 0 = выключено
}

type ClusterMeta struct {
	ID               string `json:"id"`
	DisplayName      string `json:"displayName"`
	Icon             string `json:"icon"`
	Color            string `json:"color"`
	KubeconfigFile   string `json:"kubeconfigFile"`
	DefaultNamespace string `json:"defaultNamespace"`
	AddedAt          string `json:"addedAt"`
	ContextName      string `json:"contextName"`
	// Environment — метка окружения (dev, test, stage, prod); пусто, если не задана.
	Environment string `json:"environment"`
}

type ClusterStatus struct {
	ClusterMeta
	Connected bool   `json:"connected"`
	Active    bool   `json:"active"`
	Error     string `json:"error"`
}

// GetKubronDir — каталог данных Kubron: KUBRON_DATA_DIR, иначе ~/.kubron.
// Переменная нужна серверу (контейнер, тесты) и не меняет поведение десктопа.
func GetKubronDir() string {
	if dir := os.Getenv("KUBRON_DATA_DIR"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".kubron")
}

func getClustersDir() string {
	return filepath.Join(GetKubronDir(), "clusters")
}

func getConfigPath() string {
	return filepath.Join(GetKubronDir(), "config.json")
}

func InitKubronDir() error {
	dirs := []string{GetKubronDir(), getClustersDir()}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("не удалось создать %s: %w", dir, err)
		}
	}
	return nil
}

func LoadKubronConfig() (*KubronConfig, error) {
	data, err := os.ReadFile(getConfigPath())
	if err != nil {
		return &KubronConfig{Theme: "dark"}, nil
	}
	var config KubronConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return &KubronConfig{Theme: "dark"}, nil
	}
	return &config, nil
}

func SaveKubronConfig(config *KubronConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getConfigPath(), data, 0600)
}

func SaveClusterMeta(meta *ClusterMeta) error {
	if !validClusterID(meta.ID) {
		return fmt.Errorf("недопустимый идентификатор кластера %q", meta.ID)
	}
	path := filepath.Join(getClustersDir(), meta.ID+".json")
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(path, data, 0600)
}

// validClusterID отсекает идентификаторы, которые могли бы выйти за пределы каталога кластеров.
func validClusterID(id string) bool {
	return clusterIDRe.MatchString(id)
}

func LoadClusterMeta(id string) (*ClusterMeta, error) {
	if !validClusterID(id) {
		return nil, fmt.Errorf("недопустимый идентификатор кластера %q", id)
	}
	path := filepath.Join(getClustersDir(), id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var meta ClusterMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func ListClusters() ([]ClusterMeta, error) {
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

// AddClusterFromFile копирует kubeconfig из файла в каталог Kubron и регистрирует кластер.
func AddClusterFromFile(sourcePath string, settings ClusterSettings) (*ClusterMeta, error) {
	if err := InitKubronDir(); err != nil {
		return nil, err
	}

	data, err := readKubeconfigFile(sourcePath)
	if err != nil {
		return nil, err
	}

	return addClusterFromData(data, settings)
}

// AddClusterFromText регистрирует кластер по тексту kubeconfig.
func AddClusterFromText(kubeconfigText string, settings ClusterSettings) (*ClusterMeta, error) {
	if err := InitKubronDir(); err != nil {
		return nil, err
	}
	if len(kubeconfigText) > maxKubeconfigBytes {
		return nil, errKubeconfigTooLarge
	}

	return addClusterFromData([]byte(kubeconfigText), settings)
}

func addClusterFromData(data []byte, settings ClusterSettings) (*ClusterMeta, error) {
	settings, err := settings.normalize("")
	if err != nil {
		return nil, err
	}

	// Сначала проверяем kubeconfig: с битым файлом кластер всё равно не подключится.
	cfg, err := parseKubeconfig(data)
	if err != nil {
		return nil, err
	}
	contextName, err := pickContext(cfg, settings.ContextName)
	if err != nil {
		return nil, err
	}

	id := sanitizeID(settings.DisplayName)
	if id == "" {
		id = fmt.Sprintf("cluster-%d", time.Now().Unix())
	}

	// Уникальность
	if existing, _ := LoadClusterMeta(id); existing != nil {
		id = fmt.Sprintf("%s-%d", id, time.Now().Unix())
	}

	// Сохраняем kubeconfig
	kcFile := id + ".kubeconfig"
	kcPath := filepath.Join(getClustersDir(), kcFile)
	if err := writeFileAtomic(kcPath, data, 0600); err != nil {
		return nil, fmt.Errorf("не удалось сохранить kubeconfig: %w", err)
	}

	meta := &ClusterMeta{
		ID:               id,
		DisplayName:      settings.DisplayName,
		Icon:             settings.Icon,
		Color:            settings.Color,
		Environment:      settings.Environment,
		DefaultNamespace: settings.DefaultNamespace,
		KubeconfigFile:   kcFile,
		AddedAt:          time.Now().Format(time.RFC3339),
		ContextName:      contextName,
	}

	if err := SaveClusterMeta(meta); err != nil {
		os.Remove(kcPath)
		return nil, err
	}

	return meta, nil
}

func DeleteCluster(id string) error {
	if !validClusterID(id) {
		return fmt.Errorf("недопустимый идентификатор кластера %q", id)
	}
	dir := getClustersDir()
	for _, name := range []string{id + ".json", id + ".kubeconfig"} {
		if err := os.Remove(filepath.Join(dir, name)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("не удалось удалить %s: %w", name, err)
		}
	}
	// Токен Argo CD API принадлежит кластеру: после удаления кластера он не должен оставаться на диске.
	if err := deleteArgoToken(id); err != nil {
		return err
	}
	return nil
}

func GetClusterKubeconfigPath(id string) string {
	meta, err := LoadClusterMeta(id)
	if err != nil {
		return ""
	}
	return filepath.Join(getClustersDir(), meta.KubeconfigFile)
}

func sanitizeID(name string) string {
	result := ""
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			result += string(c)
		} else if c == ' ' {
			result += "-"
		}
	}
	if len(result) > 50 {
		result = result[:50]
	}
	return result
}
