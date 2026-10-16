package k8s

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SecretInfo struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Type      string            `json:"type"`
	DataCount int               `json:"dataCount"`
	Age       string            `json:"age"`
	Labels    map[string]string `json:"labels"`
	Keys      []string          `json:"keys"`
}

func (m *Manager) GetSecrets(namespace string) ([]SecretInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	secretList, err := conn.Client.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var secrets []SecretInfo
	for _, s := range secretList.Items {
		var keys []string
		for k := range s.Data {
			keys = append(keys, k)
		}

		secrets = append(secrets, SecretInfo{
			Name:      s.Name,
			Namespace: s.Namespace,
			Type:      string(s.Type),
			DataCount: len(s.Data),
			Age:       formatAge(s.CreationTimestamp.Time),
			Labels:    s.Labels,
			Keys:      keys,
		})
	}

	return secrets, nil
}

func (m *Manager) DeleteSecret(namespace, name string) error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return conn.Client.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}
