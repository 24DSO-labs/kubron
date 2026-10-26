// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (m *Manager) GetRoleBindings(namespaces []string) (interface{}, error) {
	return scopedList(m, "GetRoleBindings", "rbac.authorization.k8s.io", "rolebindings", namespaces, listRoleBindings,
		func(it rbacv1.RoleBinding) string { return it.Namespace })
}

// listRoleBindings читает ресурсы одного namespace ("" — все) для GetRoleBindings.
func listRoleBindings(ctx context.Context, conn *ClusterConnection, namespace string) ([]rbacv1.RoleBinding, error) {
	list, err := conn.Client.RbacV1().RoleBindings(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}
