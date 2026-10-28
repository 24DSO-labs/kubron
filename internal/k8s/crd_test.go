// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func widgetCRD() *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apiextensions.k8s.io/v1",
		"kind":       "CustomResourceDefinition",
		"metadata":   map[string]interface{}{"name": "widgets.example.com"},
		"spec": map[string]interface{}{
			"group": "example.com", "scope": "Namespaced",
			"names": map[string]interface{}{"kind": "Widget", "plural": "widgets", "shortNames": []interface{}{"wd"}},
			"versions": []interface{}{
				map[string]interface{}{"name": "v1alpha1", "served": true, "storage": false},
				map[string]interface{}{
					"name": "v1", "served": true, "storage": true,
					"additionalPrinterColumns": []interface{}{
						map[string]interface{}{"name": "Phase", "type": "string", "jsonPath": ".status.phase"},
						map[string]interface{}{"name": "Ready", "type": "string", "jsonPath": `.status.conditions[?(@.type=="Ready")].status`},
						map[string]interface{}{"name": "Age", "type": "date", "jsonPath": ".metadata.creationTimestamp"},
					},
				},
			},
		},
	}}
}

func widget(ns, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "example.com/v1", "kind": "Widget",
		"metadata": map[string]interface{}{"name": name, "namespace": ns},
		"status": map[string]interface{}{
			"phase":      "Running",
			"conditions": []interface{}{map[string]interface{}{"type": "Ready", "status": "True"}},
		},
	}}
}

func newCRDDyn(objs ...runtime.Object) *dynamicfake.FakeDynamicClient {
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{
			kindTable["customresourcedefinition"].gvr: "CustomResourceDefinitionList",
			gvr("example.com", "v1", "widgets"):       "WidgetList",
		}, objs...)
}

func TestParseCRD(t *testing.T) {
	got := parseCRD(widgetCRD())
	if got.Version != "v1" || len(got.Versions) != 2 || got.Scope != "Namespaced" || got.Kind != "Widget" {
		t.Fatalf("%+v", got)
	}
	// Age (creationTimestamp) отфильтрован, остаются Phase и Ready
	if len(got.Columns) != 2 || got.Columns[0].Name != "Phase" {
		t.Errorf("columns: %+v", got.Columns)
	}
}

func TestListCustomResources(t *testing.T) {
	ctx := context.Background()
	dyn := newCRDDyn(widgetCRD(), widget("b", "z"), widget("a", "y"))

	items, err := listCustomResources(ctx, dyn, "example.com", "v1", "widgets", "")
	if err != nil || len(items) != 2 || items[0].Namespace != "a" {
		t.Fatalf("%v %+v", err, items)
	}
	if items[0].Cells["Phase"] != "Running" || items[0].Cells["Ready"] != "True" {
		t.Errorf("cells: %+v", items[0].Cells)
	}
	if len(items[0].Conditions) != 1 {
		t.Errorf("conditions: %+v", items[0].Conditions)
	}
	if one, _ := listCustomResources(ctx, dyn, "example.com", "v1", "widgets", "b"); len(one) != 1 {
		t.Errorf("namespace filter: %+v", one)
	}
}

// Через браузер CRD нельзя добраться до встроенных типов (например, secrets).
func TestCustomKindRejectsNonCRD(t *testing.T) {
	ctx := context.Background()
	dyn := newCRDDyn(widgetCRD())
	for _, c := range [][3]string{
		{"", "v1", "secrets"},
		{"core", "v1", "secrets"},        // CRD secrets.core нет
		{"example.com", "v9", "widgets"}, // версия не обслуживается
		{"example.com", "v1", "../x"},
		{"example.com", "v1", "widgets/x"},
	} {
		if _, _, err := customKindFor(ctx, dyn, c[0], c[1], c[2]); err == nil {
			t.Errorf("%v: expected an error", c)
		}
	}
	if ki, _, err := customKindFor(ctx, dyn, "example.com", "v1", "widgets"); err != nil || !ki.namespaced {
		t.Errorf("%+v %v", ki, err)
	}
}
