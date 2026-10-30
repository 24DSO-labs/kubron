// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package netpolicy

import "sync"

// Scope — класс исходящих соединений, у каждого своя политика (ADR-0015).
type Scope string

const (
	// ScopeHelm — репозитории Helm, индексы, чарты и OCI-реестры. По умолчанию strict.
	ScopeHelm Scope = "helm"
	// ScopeClusterAPI — адрес API-сервера кластера. По умолчанию выключена, при включении — permissive.
	ScopeClusterAPI Scope = "cluster-api"
)

var (
	regMu    sync.RWMutex
	registry = map[Scope]*Policy{}
)

// Set задаёт политику класса; вызывается один раз при старте сервера (cmd/kubron-server), как
// k8s.SetConnectionPolicy. nil снимает политику. Десктоп и режим одного пользователя Set не вызывают.
func Set(scope Scope, p *Policy) {
	regMu.Lock()
	defer regMu.Unlock()
	if p == nil {
		delete(registry, scope)
		return
	}
	registry[scope] = p
}

// For возвращает политику класса или nil, если она не выставлена: методы nil-политики ничего не ограничивают,
// поэтому код ядра вызывает netpolicy.For(...) без проверки режима.
func For(scope Scope) *Policy {
	regMu.RLock()
	defer regMu.RUnlock()
	return registry[scope]
}
