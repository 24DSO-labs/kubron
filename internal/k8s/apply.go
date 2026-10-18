package k8s

import (
	"context"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// ApplyYAML заменяет объект в кластере манифестом из редактора (get -> update).
// Удалённые в редакторе поля реально удаляются.
func (m *Manager) ApplyYAML(namespace, kind, name, yamlData string) error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}
	ri, info, err := resourceClient(conn, kind, namespace)
	if err != nil {
		return err
	}

	// YAML -> JSON -> Unstructured: так числа получаются int64, а не float64
	jsonData, err := yaml.YAMLToJSON([]byte(yamlData))
	if err != nil {
		return fmt.Errorf("ошибка парсинга YAML: %w", err)
	}
	obj := &unstructured.Unstructured{}
	if err := obj.UnmarshalJSON(jsonData); err != nil {
		return fmt.Errorf("некорректный манифест: %w", err)
	}

	// Защита от применения не к тому объекту
	if obj.GetName() != name {
		return fmt.Errorf("metadata.name изменять нельзя (было %q, стало %q)", name, obj.GetName())
	}
	if want := info.gvr.GroupVersion().String(); obj.GetAPIVersion() != want {
		return fmt.Errorf("apiVersion должен быть %q, а не %q", want, obj.GetAPIVersion())
	}
	if info.namespaced {
		if ns := obj.GetNamespace(); ns != "" && ns != namespace {
			return fmt.Errorf("metadata.namespace изменять нельзя (было %q, стало %q)", namespace, ns)
		}
		obj.SetNamespace(namespace)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cur, err := ri.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("не удалось получить текущий объект: %w", err)
	}
	// resourceVersion из редактора вырезан: берём актуальный, иначе Update у части ресурсов отклоняется
	obj.SetResourceVersion(cur.GetResourceVersion())

	if _, err := ri.Update(ctx, obj, metav1.UpdateOptions{}); err != nil {
		if apierrors.IsConflict(err) {
			return fmt.Errorf("объект изменился в кластере, закройте вкладку и откройте YAML заново: %w", err)
		}
		return err
	}
	return nil
}
