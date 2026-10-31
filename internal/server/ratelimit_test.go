// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"fmt"
	"testing"
	"time"
)

func limiterAt(rate float64, burst int, clock *time.Time) *RateLimiter {
	l := NewRateLimiter(rate, burst)
	l.now = func() time.Time { return *clock }
	return l
}

func TestRateLimiterBurstThenRefill(t *testing.T) {
	clock := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	l := limiterAt(2, 3, &clock) // 3 сразу, затем 2 в секунду
	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("a"); !ok {
			t.Fatalf("burst request %d refused", i)
		}
	}
	ok, wait := l.Allow("a")
	if ok {
		t.Fatal("the 4th request must be refused")
	}
	if wait <= 0 || wait > time.Second {
		t.Errorf("retry-after: %v", wait)
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Error("another key has its own bucket")
	}
	clock = clock.Add(500 * time.Millisecond) // +1 токен
	if ok, _ := l.Allow("a"); !ok {
		t.Error("a token must have refilled after 500 ms at 2/s")
	}
	if ok, _ := l.Allow("a"); ok {
		t.Error("only one token has refilled")
	}
	clock = clock.Add(time.Hour) // восстановление не выходит за burst
	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("a"); !ok {
			t.Fatalf("after a long pause request %d refused", i)
		}
	}
	if ok, _ := l.Allow("a"); ok {
		t.Error("tokens must be capped by burst")
	}
}

func TestRateLimiterBoundedMemory(t *testing.T) {
	clock := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	l := limiterAt(1, 2, &clock)
	// число бакетов ограничено, даже если адреса не повторяются
	l.Allow("offender")
	l.Allow("offender")
	if ok, _ := l.Allow("offender"); ok {
		t.Fatal("offender must be limited")
	}
	for i := 0; i < 2*maxBuckets; i++ {
		clock = clock.Add(time.Millisecond)
		l.Allow(fmt.Sprintf("10.%d.%d.%d", i>>16&255, i>>8&255, i&255))
	}
	l.mu.Lock()
	n := len(l.buckets)
	l.mu.Unlock()
	if n > maxBuckets {
		t.Errorf("buckets: %d, max %d", n, maxBuckets)
	}
}
