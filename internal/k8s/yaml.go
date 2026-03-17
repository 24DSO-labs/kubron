package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

func (m *Manager) GetResourceYAML(namespace, kind, name string) (string, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var data interface{}

	switch kind {
	case "pod":
		data, err = conn.Client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	case "deployment":
		data, err = conn.Client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	case "service":
		data, err = conn.Client.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	case "configmap":
		data, err = conn.Client.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	case "secret":
		data, err = conn.Client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	case "node":
		data, err = conn.Client.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	default:
		// Для неизвестных ресурсов используем dynamic client
		return m.getDynamicYAML(conn, ctx, namespace, kind, name)
	}

	if err != nil {
		return "", fmt.Errorf("не удалось получить %s/%s: %w", kind, name, err)
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("ошибка сериализации: %w", err)
	}

	return string(jsonBytes), nil
}

func (m *Manager) getDynamicYAML(conn *ClusterConnection, ctx context.Context, namespace, kind, name string) (string, error) {
	dynClient, err := dynamic.NewForConfig(conn.Config)
	if err != nil {
		return "", err
	}

	gvr := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: kind + "s",
	}

	var obj interface{}
	if namespace != "" {
		obj, err = dynClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	} else {
		obj, err = dynClient.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
	}

	if err != nil {
		return "", err
	}

	jsonBytes, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}
