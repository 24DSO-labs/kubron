// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (m *Manager) GetRoles(namespaces []string) (interface{}, error) {
	return scopedList(m, "GetRoles", "rbac.authorization.k8s.io", "roles", namespaces, listRoles,
		func(it rbacv1.Role) string { return it.Namespace })
}

// listRoles читает ресурсы одного namespace ("" — все) для GetRoles.
func listRoles(ctx context.Context, conn *ClusterConnection, namespace string) ([]rbacv1.Role, error) {
	list, err := conn.Client.RbacV1().Roles(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}
