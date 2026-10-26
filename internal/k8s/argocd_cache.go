// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"sort"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

const (
	// EventArgoChanged — событие фронтенду: Application создан, изменён или удалён.
	// Отдельное от cluster:changed, чтобы Argo-списки не перезагружались от любых изменений в кластере.
	EventArgoChanged = "argocd:changed"

	// argoChangeDebounce — окно склейки серии изменений (например, массовый sync) в одно уведомление.
	argoChangeDebounce = time.Second

	// argoProbeRetry — как часто повторять пробу прав, если list Application был запрещён или упал.
	argoProbeRetry = 30 * time.Second
)

// argoCache — локальный кеш Application на основе dynamic shared informer.
// После первичной синхронизации список строится из памяти, а об изменениях сообщаем через onChange.
// Пока informer не синхронизирован, вызывающий код читает через API (fallback).
type argoCache struct {
	cancel context.CancelFunc
	inf    cache.SharedIndexInformer

	mu       sync.Mutex
	timer    *time.Timer
	stopped  bool
	debounce time.Duration
	onChange func()
}

func newArgoCache(dyn dynamic.Interface, onChange func(), debounce time.Duration) (*argoCache, error) {
	ctx, cancel := context.WithCancel(context.Background())
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(dyn, 0, metav1.NamespaceAll, nil)
	inf := factory.ForResource(kindTable["application"].gvr).Informer()

	// managedFields тяжёлые и UI не нужны: экономим память при тысячах приложений.
	if err := inf.SetTransform(func(obj interface{}) (interface{}, error) {
		if u, ok := obj.(*unstructured.Unstructured); ok {
			u.SetManagedFields(nil)
		}
		return obj, nil
	}); err != nil {
		cancel()
		return nil, err
	}

	c := &argoCache{cancel: cancel, inf: inf, debounce: debounce, onChange: onChange}
	handler := cache.ResourceEventHandlerFuncs{
		AddFunc:    func(any) { c.changed() },
		UpdateFunc: func(_, _ any) { c.changed() },
		DeleteFunc: func(any) { c.changed() },
	}
	if _, err := inf.AddEventHandler(handler); err != nil {
		cancel()
		return nil, err
	}
	factory.Start(ctx.Done())
	return c, nil
}

// changed помечает кеш изменившимся; серия вызовов склеивается в одно уведомление.
func (c *argoCache) changed() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopped || c.onChange == nil || c.timer != nil {
		return
	}
	c.timer = time.AfterFunc(c.debounce, func() {
		c.mu.Lock()
		c.timer = nil
		stopped := c.stopped
		c.mu.Unlock()
		if !stopped {
			c.onChange()
		}
	})
}

func (c *argoCache) stop() {
	c.mu.Lock()
	c.stopped = true
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	c.mu.Unlock()
	c.cancel()
}

// list возвращает Application из кеша для набора namespace (пустой набор — все);
// false — informer ещё не синхронизирован.
func (c *argoCache) list(scope nsScope) ([]ArgoApplicationInfo, bool) {
	if !c.inf.HasSynced() {
		return nil, false
	}
	objs := c.inf.GetStore().List()
	apps := make([]ArgoApplicationInfo, 0, len(objs))
	for _, o := range objs {
		u, ok := o.(*unstructured.Unstructured)
		if !ok || !scope.has(u.GetNamespace()) {
			continue
		}
		apps = append(apps, parseArgoApplication(u))
	}
	sortArgoApplications(apps)
	return apps, true
}

func sortArgoApplications(apps []ArgoApplicationInfo) {
	sort.SliceStable(apps, func(i, j int) bool {
		if apps[i].Namespace != apps[j].Namespace {
			return apps[i].Namespace < apps[j].Namespace
		}
		return apps[i].Name < apps[j].Name
	})
}

// argoCache лениво запускает informer Application у активного подключения.
// Перед стартом проверяется право list: без него reflector бесконечно повторял бы запрещённые запросы.
// Если права нет (или CRD нет), возвращает nil, а следующая проба состоится не раньше argoProbeRetry.
func (c *ClusterConnection) argoCache(onChange func()) *argoCache {
	c.argoMu.Lock()
	defer c.argoMu.Unlock()
	if c.argo != nil {
		return c.argo
	}
	if time.Now().Before(c.argoRetryAt) {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := c.DynamicClient.Resource(kindTable["application"].gvr).List(ctx, metav1.ListOptions{Limit: 1}); err != nil {
		c.argoRetryAt = time.Now().Add(argoProbeRetry)
		return nil
	}
	ac, err := newArgoCache(c.DynamicClient, onChange, argoChangeDebounce)
	if err != nil {
		c.argoRetryAt = time.Now().Add(argoProbeRetry)
		return nil
	}
	c.argo = ac
	return ac
}

// closeArgoCache останавливает informer Application (вызывается из ClusterConnection.Close).
func (c *ClusterConnection) closeArgoCache() {
	c.argoMu.Lock()
	defer c.argoMu.Unlock()
	if c.argo != nil {
		c.argo.stop()
		c.argo = nil
	}
	c.argoRetryAt = time.Time{}
}
