// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	clienttesting "k8s.io/client-go/testing"
)

func newFakeArgoDyn(objs ...runtime.Object) *dynamicfake.FakeDynamicClient {
	gvrToList := map[schema.GroupVersionResource]string{
		kindTable["application"].gvr: "ApplicationList",
	}
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToList, objs...)
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("не дождались: %s", what)
}

func TestArgoCacheListAndNamespaceFilter(t *testing.T) {
	dyn := newFakeArgoDyn(
		argoObj("Application", "ns-b", "z", nil),
		argoObj("Application", "ns-a", "y", nil),
		argoObj("Application", "ns-a", "x", nil),
	)
	ac, err := newArgoCache(dyn, nil, 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer ac.stop()

	waitFor(t, "синхронизация informer", ac.inf.HasSynced)

	all, ok := ac.list("")
	if !ok || len(all) != 3 || all[0].Name != "x" || all[2].Namespace != "ns-b" {
		t.Fatalf("all: ok=%v %+v", ok, all)
	}
	one, ok := ac.list("ns-a")
	if !ok || len(one) != 2 {
		t.Fatalf("ns-a: ok=%v %+v", ok, one)
	}
}

func TestArgoCacheNotifiesOncePerBurst(t *testing.T) {
	dyn := newFakeArgoDyn()
	var calls atomic.Int32
	ac, err := newArgoCache(dyn, func() { calls.Add(1) }, 50*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer ac.stop()
	waitFor(t, "синхронизация informer", ac.inf.HasSynced)

	gvr := kindTable["application"].gvr
	for _, name := range []string{"a", "b", "c"} {
		if _, err := dyn.Resource(gvr).Namespace("argocd").Create(context.Background(), argoObj("Application", "argocd", name, nil), metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	waitFor(t, "уведомление", func() bool { return calls.Load() >= 1 })
	time.Sleep(150 * time.Millisecond)
	if got := calls.Load(); got != 1 {
		t.Fatalf("серия из 3 изменений должна дать 1 уведомление, получили %d", got)
	}
	waitFor(t, "объекты в кеше", func() bool { l, _ := ac.list(""); return len(l) == 3 })

	// Удаление тоже уведомляет.
	if err := dyn.Resource(gvr).Namespace("argocd").Delete(context.Background(), "a", metav1.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "уведомление об удалении", func() bool { return calls.Load() >= 2 })
}

func TestArgoCacheStopSilencesNotifications(t *testing.T) {
	dyn := newFakeArgoDyn()
	var calls atomic.Int32
	ac, err := newArgoCache(dyn, func() { calls.Add(1) }, 30*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	waitFor(t, "синхронизация informer", ac.inf.HasSynced)

	_, _ = dyn.Resource(kindTable["application"].gvr).Namespace("argocd").Create(context.Background(), argoObj("Application", "argocd", "a", nil), metav1.CreateOptions{})
	waitFor(t, "таймер взведён", func() bool { ac.mu.Lock(); defer ac.mu.Unlock(); return ac.timer != nil })
	ac.stop()
	time.Sleep(100 * time.Millisecond)
	if calls.Load() != 0 {
		t.Fatalf("после stop уведомлений быть не должно, получили %d", calls.Load())
	}
}

func TestConnectionArgoCacheLifecycle(t *testing.T) {
	t.Run("есть права: кеш создаётся один раз, Close останавливает", func(t *testing.T) {
		conn := &ClusterConnection{DynamicClient: newFakeArgoDyn(argoObj("Application", "argocd", "a", nil))}
		first := conn.argoCache(nil)
		if first == nil || conn.argoCache(nil) != first {
			t.Fatal("кеш должен создаваться лениво и один раз")
		}
		conn.Close()
		if conn.argo != nil || !first.stopped {
			t.Fatal("Close должен остановить и сбросить кеш")
		}
	})

	t.Run("нет прав на list: nil и пауза до следующей пробы", func(t *testing.T) {
		dyn := newFakeArgoDyn()
		var lists atomic.Int32
		dyn.PrependReactor("list", "applications", func(clienttesting.Action) (bool, runtime.Object, error) {
			lists.Add(1)
			return true, nil, apierrors.NewForbidden(schema.GroupResource{Group: argoGroup, Resource: "applications"}, "", nil)
		})
		conn := &ClusterConnection{DynamicClient: dyn}
		if conn.argoCache(nil) != nil || conn.argoCache(nil) != nil {
			t.Fatal("без прав кеша быть не должно")
		}
		if lists.Load() != 1 {
			t.Fatalf("повторная проба до истечения паузы запрещена, LIST вызван %d раз", lists.Load())
		}
	})
}

func TestGetArgoApplicationsUsesCacheAfterSync(t *testing.T) {
	dyn := newFakeArgoDyn(argoObj("Application", "argocd", "a", nil))
	var lists atomic.Int32
	dyn.PrependReactor("list", "applications", func(clienttesting.Action) (bool, runtime.Object, error) {
		lists.Add(1)
		return false, nil, nil
	})
	conn := &ClusterConnection{Name: "c", DynamicClient: dyn}
	ac := conn.argoCache(nil)
	if ac == nil {
		t.Fatal("кеш не создан")
	}
	defer conn.Close()
	waitFor(t, "синхронизация informer", ac.inf.HasSynced)

	before := lists.Load()
	got, ok := ac.list("")
	if !ok || len(got) != 1 || lists.Load() != before {
		t.Fatalf("чтение из кеша не должно обращаться к API: ok=%v n=%d lists %d→%d", ok, len(got), before, lists.Load())
	}
}
