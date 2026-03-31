package k8s

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (m *Manager) GetRoles(namespace string) (interface{}, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	list, err := conn.Client.RbacV1().Roles(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (m *Manager) DeleteRole(namespace, name string) error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return conn.Client.RbacV1().Roles(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (m *Manager) DeleteRoleBinding(namespace, name string) error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return conn.Client.RbacV1().RoleBindings(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (m *Manager) DeleteClusterRole(name string) error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return conn.Client.RbacV1().ClusterRoles().Delete(ctx, name, metav1.DeleteOptions{})
}
