// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"sync"
	"sync/atomic"
	"testing"
)

// newTestPoolManager создаёт Manager с пулом, в котором уже лежат кластеры ids, и счётчиком
// закрытий (вызовов Close, т.е. остановок informers) по каждому кластеру.
func newTestPoolManager(t *testing.T, ids ...string) (*Manager, map[string]*atomic.Int32) {
	t.Helper()
	m := NewManager()
	closes := map[string]*atomic.Int32{}
	for _, id := range ids {
		closes[id] = &atomic.Int32{}
		m.pool.put(id, &ClusterConnection{Name: id})
	}
	m.pool.closeFn = func(c *ClusterConnection) { closes[c.Name].Add(1) }
	return m, closes
}

func mustActivate(t *testing.T, m *Manager, id string) {
	t.Helper()
	m.viewMu.Lock()
	defer m.viewMu.Unlock()
	if err := m.activateLocked(id); err != nil {
		t.Fatalf("activate %s: %v", id, err)
	}
}

func activeName(t *testing.T, m *Manager) string {
	t.Helper()
	c, err := m.GetActiveConnection()
	if err != nil {
		return ""
	}
	return c.Name
}

func TestViewsHaveIndependentActiveCluster(t *testing.T) {
	root, closes := newTestPoolManager(t, "a", "b")
	v1, v2 := root.ForSession(), root.ForSession()

	mustActivate(t, v1, "a")
	mustActivate(t, v2, "b")

	if got := activeName(t, v1); got != "a" {
		t.Errorf("v1 активен %q, ждали a", got)
	}
	if got := activeName(t, v2); got != "b" {
		t.Errorf("v2 активен %q, ждали b", got)
	}
	if v1.terminalSessions == v2.terminalSessions || v1.terminalSessions == root.terminalSessions {
		t.Error("терминалы должны быть у каждого вида свои")
	}
	for id, n := range closes {
		if n.Load() != 0 {
			t.Errorf("кластер %s закрыт без причины", id)
		}
	}
}

func TestSharedClusterStaysOpenUntilLastViewLeaves(t *testing.T) {
	root, closes := newTestPoolManager(t, "a", "b")
	v1, v2 := root.ForSession(), root.ForSession()

	mustActivate(t, v1, "a")
	mustActivate(t, v2, "a")

	mustActivate(t, v1, "b") // v2 ещё на a: informers a не трогаем
	if n := closes["a"].Load(); n != 0 {
		t.Fatalf("a закрыт при уходе одного из двух видов (closes=%d)", n)
	}
	mustActivate(t, v2, "b") // последний вид ушёл с a
	if n := closes["a"].Load(); n != 1 {
		t.Fatalf("a должен закрыться ровно один раз, closes=%d", n)
	}
	if !root.pool.has("a") {
		t.Error("подключение a должно остаться в пуле для повторного использования")
	}
}

func TestDetachReleasesOnlyOwnReference(t *testing.T) {
	root, closes := newTestPoolManager(t, "a")
	v1, v2 := root.ForSession(), root.ForSession()
	mustActivate(t, v1, "a")
	mustActivate(t, v2, "a")

	v1.Detach()
	if closes["a"].Load() != 0 {
		t.Fatal("Detach одного вида закрыл кластер, который держит другой")
	}
	if activeName(t, v2) != "a" {
		t.Fatal("v2 потерял активный кластер")
	}
	v2.Detach()
	if closes["a"].Load() != 1 {
		t.Fatal("после выхода всех видов кластер должен закрыться")
	}
}

func TestRemoveConnectionAffectsEveryView(t *testing.T) {
	root, _ := newTestPoolManager(t, "a")
	v1, v2 := root.ForSession(), root.ForSession()
	mustActivate(t, v1, "a")
	mustActivate(t, v2, "a")

	if !v1.RemoveConnection("a") {
		t.Error("для вида-инициатора wasActive должно быть true")
	}
	if _, err := v2.GetActiveConnection(); err == nil {
		t.Error("у второго вида подключение должно пропасть (ошибка «нет активного подключения»)")
	}
}

func TestActivateUnknownClusterKeepsPreviousActive(t *testing.T) {
	root, _ := newTestPoolManager(t, "a")
	v := root.ForSession()
	mustActivate(t, v, "a")

	v.viewMu.Lock()
	err := v.activateLocked("missing")
	v.viewMu.Unlock()
	if err == nil {
		t.Fatal("ожидали ошибку для неподключённого кластера")
	}
	if activeName(t, v) != "a" {
		t.Error("неудачная активация не должна сбрасывать прежний кластер")
	}
}

func TestGetOrCreateBuildsOnce(t *testing.T) {
	p := newConnPool()
	var builds atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := p.getOrCreate("a", func() (*ClusterConnection, error) {
				builds.Add(1)
				return &ClusterConnection{Name: "a"}, nil
			}); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if builds.Load() != 1 {
		t.Fatalf("подключение создано %d раз, ждали 1", builds.Load())
	}
}

// Запускать с -race: 50 видов параллельно переключают кластеры; в конце все отсоединяются,
// и каждый кластер закрыт, а счётчики не ушли в минус.
func TestManyViewsConcurrentSwitching(t *testing.T) {
	ids := []string{"a", "b", "c", "d"}
	root, _ := newTestPoolManager(t, ids...)
	root.pool.closeFn = func(*ClusterConnection) {}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			v := root.ForSession()
			for j := 0; j < 200; j++ {
				id := ids[(i+j)%len(ids)]
				v.viewMu.Lock()
				_ = v.activateLocked(id)
				v.viewMu.Unlock()
				if _, err := v.GetActiveConnection(); err != nil {
					t.Errorf("вид %d: %v", i, err)
					return
				}
			}
			v.Detach()
		}(i)
	}
	wg.Wait()

	root.pool.mu.Lock()
	defer root.pool.mu.Unlock()
	for id, e := range root.pool.entries {
		if e.refs != 0 {
			t.Errorf("%s: refs=%d после выхода всех видов", id, e.refs)
		}
	}
}
