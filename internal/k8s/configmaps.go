package k8s

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConfigMapInfo struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	DataCount   int               `json:"dataCount"`
	Age         string            `json:"age"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Data        map[string]string `json:"data"`
}

func (m *Manager) GetConfigMaps(namespace string) ([]ConfigMapInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmList, err := conn.Client.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var configmaps []ConfigMapInfo
	for _, cm := range cmList.Items {
		configmaps = append(configmaps, ConfigMapInfo{
			Name:        cm.Name,
			Namespace:   cm.Namespace,
			DataCount:   len(cm.Data),
			Age:         formatAge(cm.CreationTimestamp.Time),
			Labels:      cm.Labels,
			Annotations: cm.Annotations,
		})
	}

	return configmaps, nil
}

func (m *Manager) GetConfigMapDetail(namespace, name string) (*ConfigMapInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cm, err := conn.Client.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return &ConfigMapInfo{
		Name:        cm.Name,
		Namespace:   cm.Namespace,
		DataCount:   len(cm.Data),
		Age:         formatAge(cm.CreationTimestamp.Time),
		Labels:      cm.Labels,
		Annotations: cm.Annotations,
		Data:        cm.Data,
	}, nil
}

func (m *Manager) UpdateConfigMap(namespace, name string, data map[string]string) error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cm, err := conn.Client.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("не удалось получить ConfigMap: %w", err)
	}

	cm.Data = data

	_, err = conn.Client.CoreV1().ConfigMaps(namespace).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("не удалось обновить ConfigMap: %w", err)
	}

	return nil
}
