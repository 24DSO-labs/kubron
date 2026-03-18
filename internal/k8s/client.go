package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

type ContextInfo struct {
	Name      string `json:"name"`
	Cluster   string `json:"cluster"`
	User      string `json:"user"`
	Namespace string `json:"namespace"`
	IsCurrent bool   `json:"isCurrent"`
}

type ClusterConnection struct {
	Name            string
	Client          *kubernetes.Clientset
	Config          *rest.Config
	DynamicClient   dynamic.Interface
	DiscoveryClient discovery.DiscoveryInterface
}

type Manager struct {
	mu          sync.RWMutex
	connections map[string]*ClusterConnection
	activeCtx   string
	rawConfig   *api.Config
}

func NewManager() *Manager {
	return &Manager{
		connections: make(map[string]*ClusterConnection),
	}
}

func GetKubeconfigPath() string {
	if env := os.Getenv("KUBECONFIG"); env != "" {
		return env
	}
	home, _ := os.UserHomeDir()
	kubeDir := filepath.Join(home, ".kube")
	configPath := filepath.Join(kubeDir, "config")

	info, err := os.Stat(configPath)
	if err == nil {
		if !info.IsDir() {
			return configPath
		}

		entries, err := os.ReadDir(configPath)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					name := entry.Name()
					ext := filepath.Ext(name)
					if ext == ".conf" || ext == ".yaml" || ext == ".yml" {
						return filepath.Join(configPath, name)
					}
				}
			}
		}
	}

	return configPath
}

func (m *Manager) LoadContexts() ([]ContextInfo, error) {
	path := GetKubeconfigPath()
	config, err := clientcmd.LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("не удалось загрузить kubeconfig: %w", err)
	}

	m.mu.Lock()
	m.rawConfig = config
	m.mu.Unlock()

	var contexts []ContextInfo
	for name, ctx := range config.Contexts {
		contexts = append(contexts, ContextInfo{
			Name:      name,
			Cluster:   ctx.Cluster,
			User:      ctx.AuthInfo,
			Namespace: ctx.Namespace,
			IsCurrent: name == config.CurrentContext,
		})
	}

	return contexts, nil
}

func buildConnection(name string, config *rest.Config) (*ClusterConnection, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания клиента: %w", err)
	}
	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания dynamic клиента: %w", err)
	}

	discClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания discovery клиента: %w", err)
	}

	return &ClusterConnection{
		Name:            name,
		Client:          clientset,
		Config:          config,
		DynamicClient:   dynClient,
		DiscoveryClient: discClient,
	}, nil
}

func (m *Manager) ConnectToContext(contextName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.connections[contextName]; exists {
		m.activeCtx = contextName
		return nil
	}

	path := GetKubeconfigPath()

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: path},
		&clientcmd.ConfigOverrides{CurrentContext: contextName},
	).ClientConfig()
	if err != nil {
		return fmt.Errorf("ошибка конфигурации: %w", err)
	}

	config.QPS = 50
	config.Burst = 100

	conn, err := buildConnection(contextName, config)
	if err != nil {
		return err
	}

	m.connections[contextName] = conn
	m.activeCtx = contextName
	return nil
}

func (m *Manager) GetActiveConnection() (*ClusterConnection, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conn, exists := m.connections[m.activeCtx]
	if !exists {
		return nil, fmt.Errorf("нет активного подключения")
	}
	return conn, nil
}

func (m *Manager) GetActiveContext() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeCtx
}

func (m *Manager) TestConnection() error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = conn.Client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{Limit: 1})
	return err
}

func (m *Manager) ConnectToCluster(clusterID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.connections[clusterID]; exists {
		m.activeCtx = clusterID
		return nil
	}

	kcPath := GetClusterKubeconfigPath(clusterID)
	if kcPath == "" {
		return fmt.Errorf("kubeconfig не найден для кластера %s", clusterID)
	}

	meta, err := LoadClusterMeta(clusterID)
	if err != nil {
		return fmt.Errorf("метаданные кластера не найдены: %w", err)
	}

	overrides := &clientcmd.ConfigOverrides{}
	if meta.ContextName != "" {
		overrides.CurrentContext = meta.ContextName
	}

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kcPath},
		overrides,
	).ClientConfig()
	if err != nil {
		return fmt.Errorf("ошибка конфигурации: %w", err)
	}

	config.QPS = 50
	config.Burst = 100

	conn, err := buildConnection(clusterID, config)
	if err != nil {
		return err
	}

	m.connections[clusterID] = conn
	m.activeCtx = clusterID
	return nil
}

func (m *Manager) GetClusterStatuses() []ClusterStatus {
	clusters, err := ListClusters()
	if err != nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	var statuses []ClusterStatus
	for _, cluster := range clusters {
		status := ClusterStatus{
			ClusterMeta: cluster,
			Connected:   false,
			Active:      false,
		}

		if _, ok := m.connections[cluster.ID]; ok {
			status.Connected = true
		}
		if m.activeCtx == cluster.ID {
			status.Active = true
		}

		statuses = append(statuses, status)
	}

	return statuses
}
