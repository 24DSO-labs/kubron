package k8s

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

func (m *Manager) applyObject(ctx context.Context, conn *ClusterConnection, obj *unstructured.Unstructured) error {
	gvr, err := resolveGVR(conn, obj)
	if err != nil {
		return err
	}
	var client dynamic.ResourceInterface
	if obj.GetNamespace() != "" {
		client = conn.DynamicClient.Resource(gvr).Namespace(obj.GetNamespace())
	} else {
		client = conn.DynamicClient.Resource(gvr)
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

func (m *Manager) deleteObject(ctx context.Context, conn *ClusterConnection, obj *unstructured.Unstructured) error {
	gvr, err := resolveGVR(conn, obj)
	if err != nil {
		return err
	}
	var client dynamic.ResourceInterface
	if obj.GetNamespace() != "" {
		client = conn.DynamicClient.Resource(gvr).Namespace(obj.GetNamespace())
	} else {
		client = conn.DynamicClient.Resource(gvr)
	}

	return client.Delete(ctx, obj.GetName(), metav1.DeleteOptions{})
}

func resolveGVR(conn *ClusterConnection, obj *unstructured.Unstructured) (schema.GroupVersionResource, error) {
	gvk := obj.GroupVersionKind()
	apiResources, err := conn.DiscoveryClient.ServerPreferredResources()
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

	resource := strings.ToLower(gvk.Kind) + "s"
	return schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: resource,
	}, nil
}
