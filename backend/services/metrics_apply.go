package services

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

func (m *MetricsInstaller) Install(ctx context.Context) error {
	data, err := m.loadManifest()
	if err != nil {
		return err
	}
	objects, err := parseManifest(data)
	if err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	for _, obj := range objects {
		if err := m.applyObject(ctx, obj); err != nil {
			return fmt.Errorf("failed to apply %s %s: %w", obj.GetKind(), obj.GetName(), err)
		}
	}

	return nil
}

func (m *MetricsInstaller) Uninstall(ctx context.Context) error {
	data, err := m.loadManifest()
	if err != nil {
		return err
	}
	objects, err := parseManifest(data)
	if err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Delete in reverse order, skip namespace
	for i := len(objects) - 1; i >= 0; i-- {
		obj := objects[i]
		if obj.GetKind() == "Namespace" {
			continue
		}
		if err := m.deleteObject(ctx, obj); err != nil {
			if !errors.IsNotFound(err) {
				return fmt.Errorf("failed to delete %s %s: %w", obj.GetKind(), obj.GetName(), err)
			}
		}
	}

	// Delete namespace last
	err = m.clientset.CoreV1().Namespaces().Delete(ctx, MetricsNamespace, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete namespace: %w", err)
	}

	return nil
}

func (m *MetricsInstaller) applyObject(ctx context.Context, obj *unstructured.Unstructured) error {
	gvr, err := m.resolveGVR(obj)
	if err != nil {
		return err
	}
	var client dynamic.ResourceInterface
	if obj.GetNamespace() != "" {
		client = m.dynamicClient.Resource(gvr).Namespace(obj.GetNamespace())
	} else {
		client = m.dynamicClient.Resource(gvr)
	}

	existing, err := client.Get(ctx, obj.GetName(), metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, createErr := client.Create(ctx, obj, metav1.CreateOptions{})
			return createErr
		}
		return err
	}

	obj.SetResourceVersion(existing.GetResourceVersion())
	_, err = client.Update(ctx, obj, metav1.UpdateOptions{})
	return err
}

func (m *MetricsInstaller) deleteObject(ctx context.Context, obj *unstructured.Unstructured) error {
	gvr, err := m.resolveGVR(obj)
	if err != nil {
		return err
	}
	var client dynamic.ResourceInterface
	if obj.GetNamespace() != "" {
		client = m.dynamicClient.Resource(gvr).Namespace(obj.GetNamespace())
	} else {
		client = m.dynamicClient.Resource(gvr)
	}

	return client.Delete(ctx, obj.GetName(), metav1.DeleteOptions{})
}

func (m *MetricsInstaller) resolveGVR(obj *unstructured.Unstructured) (schema.GroupVersionResource, error) {
	gvk := obj.GroupVersionKind()
	apiResources, err := m.discoveryClient.ServerPreferredResources()
	if err != nil && len(apiResources) == 0 {
		return schema.GroupVersionResource{}, fmt.Errorf("discovery failed: %w", err)
	}

	for _, list := range apiResources {
		if list == nil {
			continue
		}
		gv, parseErr := schema.ParseGroupVersion(list.GroupVersion)
		if parseErr != nil {
			continue
		}
		if gv.Group != gvk.Group || gv.Version != gvk.Version {
			continue
		}
		for _, r := range list.APIResources {
			if r.Kind == gvk.Kind {
				return schema.GroupVersionResource{
					Group:    gv.Group,
					Version:  gv.Version,
					Resource: r.Name,
				}, nil
			}
		}
	}

	// Fallback
	resource := strings.ToLower(gvk.Kind) + "s"
	return schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: resource,
	}, nil
}
