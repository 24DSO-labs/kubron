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
			t.Fatal("expected permission")
		}
	}
	if calls != 1 {
		t.Errorf("the check must be cached, calls: %d", calls)
	}
	// ключи независимы
	p.can(context.Background(), "apps/deployments", check)
	if calls != 2 {
		t.Errorf("calls=%d", calls)
	}
	clock = clock.Add(listAllAnswerTTL + time.Second)
	p.can(context.Background(), "/pods", check)
	if calls != 3 {
		t.Errorf("the check is repeated after the TTL, calls=%d", calls)
	}
}

func TestListAllProbeErrorIsDeniedWithShortTTL(t *testing.T) {
	clock := time.Unix(1000, 0)
	p := &listAllProbe{now: func() time.Time { return clock }}
	calls := 0
	check := func(context.Context) (bool, error) { calls++; return true, errors.New("boom") }

	if p.can(context.Background(), "k", check) {
		t.Fatal("a failed check must not count as granted")
	}
	p.can(context.Background(), "k", check)
	if calls != 1 {
		t.Errorf("a retry within the short TTL must not hit the API: %d", calls)
	}
	clock = clock.Add(listAllErrorTTL + time.Second)
	p.can(context.Background(), "k", check)
	if calls != 2 {
		t.Errorf("a retry probe after the short TTL: %d", calls)
	}
}

func TestListAllProbeCancelNotCached(t *testing.T) {
	p := &listAllProbe{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if p.can(ctx, "k", func(c context.Context) (bool, error) { return false, c.Err() }) {
		t.Fatal("cancellation means no permission")
	}
	if len(p.entries) != 0 {
		t.Error("context cancellation must not be cached")
	}
}
