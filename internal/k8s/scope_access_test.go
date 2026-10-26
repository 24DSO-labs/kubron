// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestListAllProbeCachesAndExpires(t *testing.T) {
	clock := time.Unix(1000, 0)
	p := &listAllProbe{now: func() time.Time { return clock }}
	calls := 0
	check := func(context.Context) (bool, error) { calls++; return true, nil }

	for i := 0; i < 3; i++ {
		if !p.can(context.Background(), "/pods", check) {
			t.Fatal("ожидалось разрешение")
		}
	}
	if calls != 1 {
		t.Errorf("проверка должна кешироваться, вызовов: %d", calls)
	}
	// ключи независимы
	p.can(context.Background(), "apps/deployments", check)
	if calls != 2 {
		t.Errorf("calls=%d", calls)
	}
	clock = clock.Add(listAllAnswerTTL + time.Second)
	p.can(context.Background(), "/pods", check)
	if calls != 3 {
		t.Errorf("после TTL проверка повторяется, calls=%d", calls)
	}
}

func TestListAllProbeErrorIsDeniedWithShortTTL(t *testing.T) {
	clock := time.Unix(1000, 0)
	p := &listAllProbe{now: func() time.Time { return clock }}
	calls := 0
	check := func(context.Context) (bool, error) { calls++; return true, errors.New("boom") }

	if p.can(context.Background(), "k", check) {
		t.Fatal("при ошибке проверки право не считается выданным")
	}
	p.can(context.Background(), "k", check)
	if calls != 1 {
		t.Errorf("повтор внутри короткого TTL не должен ходить в API: %d", calls)
	}
	clock = clock.Add(listAllErrorTTL + time.Second)
	p.can(context.Background(), "k", check)
	if calls != 2 {
		t.Errorf("после короткого TTL повторная проба: %d", calls)
	}
}

func TestListAllProbeCancelNotCached(t *testing.T) {
	p := &listAllProbe{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if p.can(ctx, "k", func(c context.Context) (bool, error) { return false, c.Err() }) {
		t.Fatal("отмена = нет права")
	}
	if len(p.entries) != 0 {
		t.Error("отмена контекста не должна попадать в кеш")
	}
}
