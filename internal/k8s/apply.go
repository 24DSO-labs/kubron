package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

func (m *Manager) ApplyYAML(namespace, kind, name, jsonData string) error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	switch kind {
	case "pod":
		_, err = conn.Client.CoreV1().Pods(namespace).Patch(
			ctx, name, types.MergePatchType, []byte(jsonData), metav1.PatchOptions{},
		)
	case "deployment":
		_, err = conn.Client.AppsV1().Deployments(namespace).Patch(
			ctx, name, types.MergePatchType, []byte(jsonData), metav1.PatchOptions{},
		)
	case "service":
		_, err = conn.Client.CoreV1().Services(namespace).Patch(
			ctx, name, types.MergePatchType, []byte(jsonData), metav1.PatchOptions{},
		)
	case "configmap":
		_, err = conn.Client.CoreV1().ConfigMaps(namespace).Patch(
			ctx, name, types.MergePatchType, []byte(jsonData), metav1.PatchOptions{},
		)
	case "secret":
		_, err = conn.Client.CoreV1().Secrets(namespace).Patch(
			ctx, name, types.MergePatchType, []byte(jsonData), metav1.PatchOptions{},
		)
	default:
		return m.applyDynamic(conn, ctx, namespace, kind, name, jsonData)
	}

	if err != nil {
		return fmt.Errorf("ошибка применения %s/%s: %w", kind, name, err)
	}
	return nil
}

func (m *Manager) applyDynamic(conn *ClusterConnection, ctx context.Context, namespace, kind, name, jsonData string) error {
	dynClient, err := dynamic.NewForConfig(conn.Config)
	if err != nil {
		return err
	}

	var obj unstructured.Unstructured
	if err := json.Unmarshal([]byte(jsonData), &obj.Object); err != nil {
		return fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	gvr := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: kind + "s",
	}

	if namespace != "" {
		_, err = dynClient.Resource(gvr).Namespace(namespace).Update(ctx, &obj, metav1.UpdateOptions{})
	} else {
		_, err = dynClient.Resource(gvr).Update(ctx, &obj, metav1.UpdateOptions{})
	}

	return err
}
