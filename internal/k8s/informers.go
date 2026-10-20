package k8s

import (
	"context"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// changeDebounce — окно, в течение которого серия изменений склеивается в одно уведомление.
const changeDebounce = 5 * time.Second

// EventClusterChanged — событие, которое бэкенд шлёт фронтенду, когда данные кластера изменились.
const EventClusterChanged = "cluster:changed"

// overviewCache — локальный кеш ресурсов для главного экрана на основе shared informers.
// После первичной синхронизации обзор собирается из памяти и не нагружает API-сервер
// полными LIST-запросами каждые несколько секунд; об изменениях сообщаем через onChange.
// Пока informer не синхронизирован (или ему не хватает прав), вызывающий код
// переключается на прямой запрос к API.
type overviewCache struct {
	cancel context.CancelFunc

	nodes        cache.SharedIndexInformer
	pods         cache.SharedIndexInformer
	namespaces   cache.SharedIndexInformer
	events       cache.SharedIndexInformer
	deployments  cache.SharedIndexInformer
	statefulSets cache.SharedIndexInformer
	daemonSets   cache.SharedIndexInformer

	mu       sync.Mutex
	timer    *time.Timer
	stopped  bool
	onChange func()
}

func newOverviewCache(client kubernetes.Interface, onChange func()) *overviewCache {
	ctx, cancel := context.WithCancel(context.Background())
	factory := informers.NewSharedInformerFactory(client, 0)

	c := &overviewCache{
		cancel:       cancel,
		nodes:        factory.Core().V1().Nodes().Informer(),
		pods:         factory.Core().V1().Pods().Informer(),
		namespaces:   factory.Core().V1().Namespaces().Informer(),
		events:       factory.Core().V1().Events().Informer(),
		deployments:  factory.Apps().V1().Deployments().Informer(),
		statefulSets: factory.Apps().V1().StatefulSets().Informer(),
		daemonSets:   factory.Apps().V1().DaemonSets().Informer(),
		onChange:     onChange,
	}

	handler := cache.ResourceEventHandlerFuncs{
		AddFunc:    func(any) { c.changed() },
		UpdateFunc: func(_, _ any) { c.changed() },
		DeleteFunc: func(any) { c.changed() },
	}
	for _, inf := range []cache.SharedIndexInformer{
		c.nodes, c.pods, c.namespaces, c.events, c.deployments, c.statefulSets, c.daemonSets,
	} {
		_, _ = inf.AddEventHandler(handler)
	}

	factory.Start(ctx.Done())
	return c
}

// changed помечает кеш изменившимся; серия вызовов склеивается в одно уведомление.
func (c *overviewCache) changed() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stopped || c.onChange == nil || c.timer != nil {
		return
	}
	c.timer = time.AfterFunc(changeDebounce, func() {
		c.mu.Lock()
		c.timer = nil
		stopped := c.stopped
		c.mu.Unlock()
		if !stopped {
			c.onChange()
		}
	})
}

func (c *overviewCache) stop() {
	c.mu.Lock()
	c.stopped = true
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	c.mu.Unlock()
	c.cancel()
}

// listCached возвращает копии объектов из кеша; false — informer ещё не готов.
func listCached[T any](inf cache.SharedIndexInformer) ([]T, bool) {
	if !inf.HasSynced() {
		return nil, false
	}
	// Объекты в кеше общие, менять их нельзя: отдаём копии структур верхнего уровня.
	objs := inf.GetStore().List()
	out := make([]T, 0, len(objs))
	for _, o := range objs {
		if v, ok := o.(*T); ok {
			out = append(out, *v)
		}
	}
	return out, true
}

func (c *overviewCache) nodeList() (*corev1.NodeList, bool) {
	items, ok := listCached[corev1.Node](c.nodes)
	return &corev1.NodeList{Items: items}, ok
}

func (c *overviewCache) podList() (*corev1.PodList, bool) {
	items, ok := listCached[corev1.Pod](c.pods)
	return &corev1.PodList{Items: items}, ok
}

func (c *overviewCache) namespaceList() (*corev1.NamespaceList, bool) {
	items, ok := listCached[corev1.Namespace](c.namespaces)
	return &corev1.NamespaceList{Items: items}, ok
}

func (c *overviewCache) eventList() (*corev1.EventList, bool) {
	items, ok := listCached[corev1.Event](c.events)
	return &corev1.EventList{Items: items}, ok
}

func (c *overviewCache) deploymentList() (*appsv1.DeploymentList, bool) {
	items, ok := listCached[appsv1.Deployment](c.deployments)
	return &appsv1.DeploymentList{Items: items}, ok
}

func (c *overviewCache) statefulSetList() (*appsv1.StatefulSetList, bool) {
	items, ok := listCached[appsv1.StatefulSet](c.statefulSets)
	return &appsv1.StatefulSetList{Items: items}, ok
}

func (c *overviewCache) daemonSetList() (*appsv1.DaemonSetList, bool) {
	items, ok := listCached[appsv1.DaemonSet](c.daemonSets)
	return &appsv1.DaemonSetList{Items: items}, ok
}
