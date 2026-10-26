// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"time"
)

// scopeTimeout — единый дедлайн на всю операцию чтения набора namespace
// (как и у прежних одно-namespace методов).
const scopeTimeout = 15 * time.Second

// scopedList — общий каркас методов Get*: нормализует набор namespace, получает активное
// подключение, выбирает стратегию (один общий запрос или параллельные по namespace) и сообщает
// фронтенду о частичных отказах. group/resource нужны только для проверки права на
// cluster-wide list (SelfSubjectAccessReview).
//
// list читает ресурсы ОДНОГО namespace ("" — все) и конвертирует их в тип ответа;
// nsOf возвращает namespace элемента ответа.
func scopedList[T any](
	m *Manager,
	method, group, resource string,
	namespaces []string,
	list func(ctx context.Context, conn *ClusterConnection, ns string) ([]T, error),
	nsOf func(T) string,
) ([]T, error) {
	return scopedListTimeout(m, scopeTimeout, method, group, resource, namespaces, list, nsOf)
}

// scopedListTimeout — scopedList с собственным дедлайном (у Argo и CRD он больше стандартного).
func scopedListTimeout[T any](
	m *Manager,
	timeout time.Duration,
	method, group, resource string,
	namespaces []string,
	list func(ctx context.Context, conn *ClusterConnection, ns string) ([]T, error),
	nsOf func(T) string,
) ([]T, error) {
	scope, err := parseScope(namespaces)
	if err != nil {
		return nil, err
	}
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	canAll := false
	if len(scope.names) > scopeListAllThreshold {
		canAll = conn.listAll.can(ctx, group+"/"+resource, ssarCanListAll(conn.Client, group, resource))
	}

	items, failures, err := listScoped(ctx, scope, canAll,
		func(ctx context.Context, ns string) ([]T, error) { return list(ctx, conn, ns) }, nsOf)
	m.reportPartial(method, failures)
	if err != nil {
		return nil, err
	}
	return items, nil
}
