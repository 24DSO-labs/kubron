package k8s

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SecretInfo struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Type        string            `json:"type"`
	DataCount   int               `json:"dataCount"`
	Age         string            `json:"age"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Keys        []string          `json:"keys"`
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
			Name:        s.Name,
			Namespace:   s.Namespace,
			Type:        string(s.Type),
			DataCount:   len(s.Data),
			Age:         formatAge(s.CreationTimestamp.Time),
			Labels:      s.Labels,
			Annotations: s.Annotations,
			Keys:        keys,
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

// GetSecretValue возвращает расшифрованное значение одного ключа секрета.
// client-go уже декодирует base64 в sec.Data.
func (m *Manager) GetSecretValue(namespace, name, key string) (string, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sec, err := conn.Client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	val, ok := sec.Data[key]
	if !ok {
		return "", fmt.Errorf("ключ %q не найден в секрете %s", key, name)
	}
	return string(val), nil
}

// GetSecretBase64Value возвращает base64-кодированное значение ключа секрета
// (как оно хранится в Kubernetes).
func (m *Manager) GetSecretBase64Value(namespace, name, key string) (string, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sec, err := conn.Client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	val, ok := sec.Data[key]
	if !ok {
		return "", fmt.Errorf("ключ %q не найден в секрете %s", key, name)
	}
	return base64.StdEncoding.EncodeToString(val), nil
}

// UpdateSecret обновляет данные секрета. Значения передаются в виде строк,
// они конвертируются в []byte (client-go сам кодирует в base64 при отправке).
func (m *Manager) UpdateSecret(namespace, name string, data map[string]string) error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sec, err := conn.Client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("не удалось получить Secret: %w", err)
	}

	sec.Data = make(map[string][]byte, len(data))
	for k, v := range data {
		sec.Data[k] = []byte(v)
	}

	_, err = conn.Client.CoreV1().Secrets(namespace).Update(ctx, sec, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("не удалось обновить Secret: %w", err)
	}
	return nil
}
