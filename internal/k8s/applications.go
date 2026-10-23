package k8s

import (
	"context"
	"fmt"
	"sort"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Стандартные recommended labels Kubernetes: по ним (как в Lens) workload-ы собираются в «приложения».
const (
	labelAppName      = "app.kubernetes.io/name"
	labelAppInstance  = "app.kubernetes.io/instance"
	labelAppVersion   = "app.kubernetes.io/version"
	labelAppManagedBy = "app.kubernetes.io/managed-by"
	labelAppPartOf    = "app.kubernetes.io/part-of"
	labelAppComponent = "app.kubernetes.io/component"
)

// ApplicationWorkload — workload (Deployment/StatefulSet/DaemonSet), входящий в приложение.
type ApplicationWorkload struct {
	Kind     string `json:"kind"`
	Name     string `json:"name"`
	Replicas int32  `json:"replicas"`
	Ready    int32  `json:"ready"`
}

// ApplicationInfo — приложение: группа workload-ов одного namespace с общими
// app.kubernetes.io/instance и app.kubernetes.io/name.
type ApplicationInfo struct {
	// Name — уникальное в namespace имя (ключ строки списка): имя приложения, а при совпадении
	// у разных instance — "instance/имя".
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	Instance    string `json:"instance"`
	Application string `json:"application"`
	Version     string `json:"version"`
	// ManagedBy — значение app.kubernetes.io/managed-by ("Helm", "Kustomize" ...)
	ManagedBy string `json:"managedBy"`
	PartOf    string `json:"partOf"`
	Component string `json:"component"`

	// Status — Running | Progressing | Degraded | Stopped
	Status  string `json:"status"`
	Age     string `json:"age"`
	Created string `json:"created"`

	Workloads []ApplicationWorkload `json:"workloads"`
	Images    []string              `json:"images"`
	// Selector — labels, по которым у пода определяется принадлежность приложению
	Selector     map[string]string `json:"selector"`
	Labels       map[string]string `json:"labels"`
	InternalURLs []string          `json:"internalUrls"`
	ExposedURLs  []string          `json:"exposedUrls"`
}

// GetApplications — приложения namespace (все namespace при пустом значении).
func (m *Manager) GetApplications(namespace string) ([]ApplicationInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	opts := metav1.ListOptions{}
	deps, err := conn.Client.AppsV1().Deployments(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}
	stss, err := conn.Client.AppsV1().StatefulSets(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}
	dss, err := conn.Client.AppsV1().DaemonSets(namespace).List(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Сервисы и ингрессы нужны только для URL: их недоступность (RBAC) не должна ломать список
	var svcs []corev1.Service
	if l, err := conn.Client.CoreV1().Services(namespace).List(ctx, opts); err == nil {
		svcs = l.Items
	}
	var ings []networkingv1.Ingress
	if l, err := conn.Client.NetworkingV1().Ingresses(namespace).List(ctx, opts); err == nil {
		ings = l.Items
	}

	return buildApplications(deps.Items, stss.Items, dss.Items, svcs, ings), nil
}

// appWorkload — единая форма Deployment/StatefulSet/DaemonSet для группировки.
type appWorkload struct {
	kind      string
	name      string
	namespace string
	labels    map[string]string // labels объекта, дополненные labels pod template
	podLabels map[string]string // labels pod template: по ним сервисы выбирают поды
	created   time.Time
	desired   int32
	ready     int32
	images    []string
}

// mergeLabels — labels объекта, недостающие ключи берутся из pod template.
func mergeLabels(obj, tmpl map[string]string) map[string]string {
	out := make(map[string]string, len(obj)+len(tmpl))
	for k, v := range tmpl {
		out[k] = v
	}
	for k, v := range obj {
		out[k] = v
	}
	return out
}

func containerImages(spec corev1.PodSpec) []string {
	images := make([]string, 0, len(spec.Containers))
	for _, c := range spec.Containers {
		images = append(images, c.Image)
	}
	return images
}

func workloadsFrom(deps []appsv1.Deployment, stss []appsv1.StatefulSet, dss []appsv1.DaemonSet) []appWorkload {
	out := make([]appWorkload, 0, len(deps)+len(stss)+len(dss))
	for _, d := range deps {
		var desired int32 = 1 // значение по умолчанию, если spec.replicas не задан
		if d.Spec.Replicas != nil {
			desired = *d.Spec.Replicas
		}
		out = append(out, appWorkload{
			kind: "Deployment", name: d.Name, namespace: d.Namespace,
			labels:    mergeLabels(d.Labels, d.Spec.Template.Labels),
			podLabels: d.Spec.Template.Labels,
			created:   d.CreationTimestamp.Time,
			desired:   desired, ready: d.Status.ReadyReplicas,
			images: containerImages(d.Spec.Template.Spec),
		})
	}
	for _, s := range stss {
		var desired int32 = 1
		if s.Spec.Replicas != nil {
			desired = *s.Spec.Replicas
		}
		out = append(out, appWorkload{
			kind: "StatefulSet", name: s.Name, namespace: s.Namespace,
			labels:    mergeLabels(s.Labels, s.Spec.Template.Labels),
			podLabels: s.Spec.Template.Labels,
			created:   s.CreationTimestamp.Time,
			desired:   desired, ready: s.Status.ReadyReplicas,
			images: containerImages(s.Spec.Template.Spec),
		})
	}
	for _, d := range dss {
		out = append(out, appWorkload{
			kind: "DaemonSet", name: d.Name, namespace: d.Namespace,
			labels:    mergeLabels(d.Labels, d.Spec.Template.Labels),
			podLabels: d.Spec.Template.Labels,
			created:   d.CreationTimestamp.Time,
			desired:   d.Status.DesiredNumberScheduled, ready: d.Status.NumberReady,
			images: containerImages(d.Spec.Template.Spec),
		})
	}
	return out
}

type appKey struct{ namespace, instance, application string }

// buildApplications группирует workload-ы по (namespace, instance, name) и собирает URL
// из сервисов (внутренние) и ингрессов (внешние). Чистая функция: вся работа с API — в GetApplications.
func buildApplications(
	deps []appsv1.Deployment, stss []appsv1.StatefulSet, dss []appsv1.DaemonSet,
	svcs []corev1.Service, ings []networkingv1.Ingress,
) []ApplicationInfo {
	groups := map[appKey][]appWorkload{}
	for _, w := range workloadsFrom(deps, stss, dss) {
		instance, name := w.labels[labelAppInstance], w.labels[labelAppName]
		if instance == "" && name == "" {
			continue // без recommended labels это не «приложение»
		}
		if name == "" {
			name = instance
		}
		if instance == "" {
			instance = name
		}
		k := appKey{w.namespace, instance, name}
		groups[k] = append(groups[k], w)
	}

	keys := make([]appKey, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if a.namespace != b.namespace {
			return a.namespace < b.namespace
		}
		if a.instance != b.instance {
			return a.instance < b.instance
		}
		return a.application < b.application
	})

	// Сколько приложений в namespace носит одно и то же имя: при совпадении имя дополняется instance
	nameCount := map[[2]string]int{}
	for _, k := range keys {
		nameCount[[2]string{k.namespace, k.application}]++
	}

	apps := make([]ApplicationInfo, 0, len(keys))
	for _, k := range keys {
		ws := groups[k]
		sort.Slice(ws, func(i, j int) bool {
			if ws[i].kind != ws[j].kind {
				return ws[i].kind < ws[j].kind
			}
			return ws[i].name < ws[j].name
		})

		app := ApplicationInfo{
			Name:         k.application,
			Namespace:    k.namespace,
			Instance:     k.instance,
			Application:  k.application,
			Workloads:    make([]ApplicationWorkload, 0, len(ws)),
			Images:       []string{},
			InternalURLs: []string{},
			ExposedURLs:  []string{},
			Selector:     map[string]string{},
		}
		if nameCount[[2]string{k.namespace, k.application}] > 1 {
			app.Name = k.instance + "/" + k.application
		}
		if k.instance != "" {
			app.Selector[labelAppInstance] = k.instance
		}
		app.Selector[labelAppName] = k.application

		var desired, ready int32
		var created time.Time
		seenImage := map[string]bool{}
		for i, w := range ws {
			app.Workloads = append(app.Workloads, ApplicationWorkload{Kind: w.kind, Name: w.name, Replicas: w.desired, Ready: w.ready})
			desired += w.desired
			ready += w.ready
			if created.IsZero() || w.created.Before(created) {
				created = w.created
			}
			for _, img := range w.images {
				if !seenImage[img] {
					seenImage[img] = true
					app.Images = append(app.Images, img)
				}
			}
			// Метаданные берём из первого workload, где значение задано
			first := func(dst *string, label string) {
				if *dst == "" {
					*dst = w.labels[label]
				}
			}
			first(&app.Version, labelAppVersion)
			first(&app.ManagedBy, labelAppManagedBy)
			first(&app.PartOf, labelAppPartOf)
			first(&app.Component, labelAppComponent)
			if i == 0 {
				app.Labels = w.labels
			}
		}

		switch {
		case desired == 0:
			app.Status = "Stopped"
		case ready >= desired:
			app.Status = "Running"
		case ready == 0:
			app.Status = "Degraded"
		default:
			app.Status = "Progressing"
		}
		if !created.IsZero() {
			app.Age = formatAge(created)
			app.Created = created.Format(time.RFC3339)
		}

		svcNames := map[string]bool{}
		for _, svc := range svcs {
			if svc.Namespace != k.namespace || !serviceSelectsAny(svc, ws) {
				continue
			}
			svcNames[svc.Name] = true
			app.InternalURLs = append(app.InternalURLs, serviceInternalURLs(svc)...)
		}
		app.ExposedURLs = exposedURLs(k.namespace, svcNames, ings)
		apps = append(apps, app)
	}
	return apps
}

// serviceSelectsAny — выбирает ли сервис поды хотя бы одного из workload-ов (selector ⊆ labels pod template).
func serviceSelectsAny(svc corev1.Service, ws []appWorkload) bool {
	if len(svc.Spec.Selector) == 0 {
		return false
	}
	for _, w := range ws {
		match := true
		for k, v := range svc.Spec.Selector {
			if w.podLabels[k] != v {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// serviceInternalURLs — адреса сервиса внутри кластера: "name.namespace:port".
func serviceInternalURLs(svc corev1.Service) []string {
	urls := make([]string, 0, len(svc.Spec.Ports))
	for _, p := range svc.Spec.Ports {
		urls = append(urls, fmt.Sprintf("%s.%s:%d", svc.Name, svc.Namespace, p.Port))
	}
	return urls
}

// exposedURLs — адреса из ингрессов, чей backend — один из сервисов приложения.
func exposedURLs(namespace string, svcNames map[string]bool, ings []networkingv1.Ingress) []string {
	urls := []string{}
	seen := map[string]bool{}
	for _, ing := range ings {
		if ing.Namespace != namespace {
			continue
		}
		tlsHosts := map[string]bool{}
		for _, t := range ing.Spec.TLS {
			for _, h := range t.Hosts {
				tlsHosts[h] = true
			}
		}
		for _, rule := range ing.Spec.Rules {
			if rule.HTTP == nil || rule.Host == "" {
				continue
			}
			for _, p := range rule.HTTP.Paths {
				if p.Backend.Service == nil || !svcNames[p.Backend.Service.Name] {
					continue
				}
				scheme := "http"
				if tlsHosts[rule.Host] {
					scheme = "https"
				}
				path := p.Path
				if path == "/" {
					path = ""
				}
				u := scheme + "://" + rule.Host + path
				if !seen[u] {
					seen[u] = true
					urls = append(urls, u)
				}
			}
		}
	}
	sort.Strings(urls)
	return urls
}
