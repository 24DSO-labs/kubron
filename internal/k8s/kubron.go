package k8s

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"k8s.io/client-go/tools/clientcmd"
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
}

type ClusterStatus struct {
	ClusterMeta
	Connected bool   `json:"connected"`
	Active    bool   `json:"active"`
	Error     string `json:"error"`
}

func GetKubronDir() string {
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
	path := filepath.Join(getClustersDir(), meta.ID+".json")
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func LoadClusterMeta(id string) (*ClusterMeta, error) {
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

func AddClusterFromFile(sourcePath, displayName, icon, color string) (*ClusterMeta, error) {
	if err := InitKubronDir(); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать файл: %w", err)
	}

	return addClusterFromData(data, displayName, icon, color)
}

func AddClusterFromText(kubeconfigText, displayName, icon, color string) (*ClusterMeta, error) {
	if err := InitKubronDir(); err != nil {
		return nil, err
	}

	return addClusterFromData([]byte(kubeconfigText), displayName, icon, color)
}

func addClusterFromData(data []byte, displayName, icon, color string) (*ClusterMeta, error) {
	id := sanitizeID(displayName)
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
	if err := os.WriteFile(kcPath, data, 0600); err != nil {
		return nil, fmt.Errorf("не удалось сохранить kubeconfig: %w", err)
	}

	// Парсим чтобы найти контекст
	contextName := ""
	cfg, err := clientcmd.LoadFromFile(kcPath)
	if err == nil {
		if cfg.CurrentContext != "" {
			contextName = cfg.CurrentContext
		} else {
			for name := range cfg.Contexts {
				contextName = name
				break
			}
		}
	}

	if icon == "" {
		icon = "🔵"
	}
	if color == "" {
		color = "#89b4fa"
	}

	meta := &ClusterMeta{
		ID:             id,
		DisplayName:    displayName,
		Icon:           icon,
		Color:          color,
		KubeconfigFile: kcFile,
		AddedAt:        time.Now().Format(time.RFC3339),
		ContextName:    contextName,
	}

	if err := SaveClusterMeta(meta); err != nil {
		os.Remove(kcPath)
		return nil, err
	}

	return meta, nil
}

func DeleteCluster(id string) error {
	dir := getClustersDir()
	for _, name := range []string{id + ".json", id + ".kubeconfig"} {
		if err := os.Remove(filepath.Join(dir, name)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("не удалось удалить %s: %w", name, err)
		}
	}
	return nil
}

func UpdateClusterMeta(id, displayName, icon, color string) error {
	meta, err := LoadClusterMeta(id)
	if err != nil {
		return err
	}
	if displayName != "" {
		meta.DisplayName = displayName
	}
	if icon != "" {
		meta.Icon = icon
	}
	if color != "" {
		meta.Color = color
	}
	return SaveClusterMeta(meta)
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
