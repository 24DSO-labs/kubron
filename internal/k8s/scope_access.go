// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"sync"
	"time"

	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// listAllAnswerTTL — как долго помним ответ «право есть/нет» от API.
	listAllAnswerTTL = 5 * time.Minute
	// listAllErrorTTL — после сбоя самой проверки не долбим API: по аналогии с argoRetryAt.
	listAllErrorTTL = 30 * time.Second
)

// listAllProbe кеширует результат проверки права «list на уровне кластера» по ресурсам.
// Нулевое значение готово к работе; встраивается в ClusterConnection (по значению, не копировать).
type listAllProbe struct {
	mu      sync.Mutex
	entries map[string]listAllEntry
	now     func() time.Time // подменяется в тестах
}

type listAllEntry struct {
	allowed bool
	until   time.Time
}

// can возвращает, разрешён ли cluster-wide list для ресурса (ключ — "group/resource").
// Ответ кешируется; ошибка проверки трактуется как «нельзя» (останется per-namespace путь)
// и запоминается на короткий срок.
func (p *listAllProbe) can(ctx context.Context, key string, check func(context.Context) (bool, error)) bool {
	now := time.Now
	p.mu.Lock()
	if p.now != nil {
		now = p.now
	}
	if e, ok := p.entries[key]; ok && now().Before(e.until) {
		p.mu.Unlock()
		return e.allowed
	}
	p.mu.Unlock()

	// сама проверка — вне блокировки; параллельные дубли безвредны (идемпотентный запрос)
	allowed, err := check(ctx)
	ttl := listAllAnswerTTL
	if err != nil {
		if ctx.Err() != nil {
			return false // отмена — не факт об авторизации, не кешируем
		}
		allowed, ttl = false, listAllErrorTTL
	}

	p.mu.Lock()
	if p.entries == nil {
		p.entries = make(map[string]listAllEntry)
	}
	p.entries[key] = listAllEntry{allowed: allowed, until: now().Add(ttl)}
	p.mu.Unlock()
	return allowed
}

// ssarCanListAll — проверка через SelfSubjectAccessReview: можно ли делать list ресурса по всем namespace.
func ssarCanListAll(cs kubernetes.Interface, group, resource string) func(context.Context) (bool, error) {
	return func(ctx context.Context) (bool, error) {
		review := &authv1.SelfSubjectAccessReview{
			Spec: authv1.SelfSubjectAccessReviewSpec{
				ResourceAttributes: &authv1.ResourceAttributes{Verb: "list", Group: group, Resource: resource},
			},
		}
		res, err := cs.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, review, metav1.CreateOptions{})
		if err != nil {
			return false, err
		}
		return res.Status.Allowed, nil
	}
}
