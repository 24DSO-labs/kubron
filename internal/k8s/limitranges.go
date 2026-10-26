// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// LimitRangeLimit — одна строка таблицы лимитов: тип объекта + ресурс + границы.
type LimitRangeLimit struct {
	Type                 string `json:"type"` // Container, Pod, PersistentVolumeClaim
	Resource             string `json:"resource"`
	Min                  string `json:"min"`
	Max                  string `json:"max"`
	Default              string `json:"default"`
	DefaultRequest       string `json:"defaultRequest"`
	MaxLimitRequestRatio string `json:"maxLimitRequestRatio"`
}

type LimitRangeInfo struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Age       string `json:"age"`
	Items     int    `json:"items"`

	Created      string            `json:"created"`
	Labels       map[string]string `json:"labels"`
	ControlledBy string            `json:"controlledBy"`
	ManagedBy    string            `json:"managedBy"`
	Annotations  map[string]string `json:"annotations"`
	Finalizers   []string          `json:"finalizers"`
	Limits       []LimitRangeLimit `json:"limits"`
}

func (m *Manager) GetLimitRanges(namespaces []string) ([]LimitRangeInfo, error) {
	return scopedList(m, "GetLimitRanges", "", "limitranges", namespaces, listLimitRanges,
		func(it LimitRangeInfo) string { return it.Namespace })
}

// listLimitRanges читает ресурсы одного namespace ("" — все) для GetLimitRanges.
func listLimitRanges(ctx context.Context, conn *ClusterConnection, namespace string) ([]LimitRangeInfo, error) {
	list, err := conn.Client.CoreV1().LimitRanges(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	limits := make([]LimitRangeInfo, 0, len(list.Items))
	for _, lr := range list.Items {
		limits = append(limits, convertLimitRange(lr))
	}

	return limits, nil
}

func convertLimitRange(lr v1.LimitRange) LimitRangeInfo {
	meta := metaOf(lr.ObjectMeta)

	rows := make([]LimitRangeLimit, 0)
	for _, item := range lr.Spec.Limits {
		// Собираем все ресурсы, упомянутые хотя бы в одном из полей
		names := map[string]struct{}{}
		for _, list := range []v1.ResourceList{item.Min, item.Max, item.Default, item.DefaultRequest, item.MaxLimitRequestRatio} {
			for n := range list {
				names[string(n)] = struct{}{}
			}
		}
		for _, name := range sortedKeys(names) {
			rn := v1.ResourceName(name)
			rows = append(rows, LimitRangeLimit{
				Type:                 string(item.Type),
				Resource:             name,
				Min:                  quantityOf(item.Min, rn),
				Max:                  quantityOf(item.Max, rn),
				Default:              quantityOf(item.Default, rn),
				DefaultRequest:       quantityOf(item.DefaultRequest, rn),
				MaxLimitRequestRatio: quantityOf(item.MaxLimitRequestRatio, rn),
			})
		}
	}

	return LimitRangeInfo{
		Name:         lr.Name,
		Namespace:    lr.Namespace,
		Age:          meta.Age,
		Items:        len(lr.Spec.Limits),
		Created:      meta.Created,
		Labels:       meta.Labels,
		ControlledBy: controlledByOf(lr.OwnerReferences),
		ManagedBy:    managedByOf(lr.ObjectMeta),
		Annotations:  meta.Annotations,
		Finalizers:   meta.Finalizers,
		Limits:       rows,
	}
}

func quantityOf(list v1.ResourceList, name v1.ResourceName) string {
	if q, ok := list[name]; ok {
		return q.String()
	}
	return ""
}
