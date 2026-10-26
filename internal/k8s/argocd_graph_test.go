// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"fmt"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

func ctrlOwner(apiVersion, kind, name string) []metav1.OwnerReference {
	t := true
	return []metav1.OwnerReference{{APIVersion: apiVersion, Kind: kind, Name: name, Controller: &t}}
}

func meta(ns, name string, lbls map[string]string, owners []metav1.OwnerReference) metav1.ObjectMeta {
	return metav1.ObjectMeta{Name: name, Namespace: ns, Labels: lbls, OwnerReferences: owners}
}

func hasEdge(g *ArgoGraph, from, to, typ string) bool {
	for _, e := range g.Edges {
		if e.From == from && e.To == to && e.Type == typ {
			return true
		}
	}
	return false
}

func findNode(g *ArgoGraph, id string) *ArgoGraphNode {
	for i := range g.Nodes {
		if g.Nodes[i].ID == id {
			return &g.Nodes[i]
		}
	}
	return nil
}

func TestBuildArgoGraphOwnersServiceIngress(t *testing.T) {
	const ns = "prod"
	c := fake.NewSimpleClientset(
		&appsv1.ReplicaSet{ObjectMeta: meta(ns, "web-abc", nil, ctrlOwner("apps/v1", "Deployment", "web"))},
		&appsv1.ReplicaSet{ObjectMeta: meta(ns, "other-xyz", nil, ctrlOwner("apps/v1", "Deployment", "other"))}, // чужой
		&corev1.Pod{ObjectMeta: meta(ns, "web-abc-1", map[string]string{"app": "web"}, ctrlOwner("apps/v1", "ReplicaSet", "web-abc"))},
		&corev1.Pod{ObjectMeta: meta(ns, "web-abc-2", map[string]string{"app": "web"}, ctrlOwner("apps/v1", "ReplicaSet", "web-abc"))},
		&corev1.Pod{ObjectMeta: meta(ns, "stranger", map[string]string{"app": "web"}, nil)}, // совпадает селектор, но не в приложении
		&batchv1.Job{ObjectMeta: meta(ns, "report-1", nil, ctrlOwner("batch/v1", "CronJob", "report"))},
		&corev1.Pod{ObjectMeta: meta(ns, "report-1-p", nil, ctrlOwner("batch/v1", "Job", "report-1"))},
		&corev1.Service{ObjectMeta: meta(ns, "web", nil, nil), Spec: corev1.ServiceSpec{Selector: map[string]string{"app": "web"}}},
		&networkingv1.Ingress{ObjectMeta: meta(ns, "web", nil, nil), Spec: networkingv1.IngressSpec{
			DefaultBackend: &networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: "web"}},
			Rules: []networkingv1.IngressRule{{IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{
				Paths: []networkingv1.HTTPIngressPath{{Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: "shared-auth"}}}},
			}}}},
		}},
	)
	refs := []ArgoResourceRef{
		{Group: "apps", Kind: "Deployment", Namespace: ns, Name: "web", Status: "Synced", Health: "Healthy"},
		{Group: "batch", Kind: "CronJob", Namespace: ns, Name: "report", Status: "OutOfSync"},
		{Kind: "Service", Namespace: ns, Name: "web", Status: "Synced", Health: "Healthy"},
		{Group: "networking.k8s.io", Kind: "Ingress", Namespace: ns, Name: "web", Status: "Synced"},
		{Group: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "reader"}, // cluster-scoped: без детей
	}

	g := buildArgoGraph(context.Background(), c, refs)

	dep := graphID("apps", "Deployment", ns, "web")
	rs := graphID("apps", "ReplicaSet", ns, "web-abc")
	pod1 := graphID("", "Pod", ns, "web-abc-1")
	svc := graphID("", "Service", ns, "web")
	ing := graphID("networking.k8s.io", "Ingress", ns, "web")

	for _, edge := range [][3]string{
		{dep, rs, ArgoEdgeOwns},
		{rs, pod1, ArgoEdgeOwns},
		{graphID("batch", "CronJob", ns, "report"), graphID("batch", "Job", ns, "report-1"), ArgoEdgeOwns},
		{graphID("batch", "Job", ns, "report-1"), graphID("", "Pod", ns, "report-1-p"), ArgoEdgeOwns},
		{svc, pod1, ArgoEdgeSelects},
		{svc, graphID("", "Pod", ns, "web-abc-2"), ArgoEdgeSelects},
		{ing, svc, ArgoEdgeRoutes},
		{ing, graphID("", "Service", ns, "shared-auth"), ArgoEdgeRoutes},
	} {
		if !hasEdge(g, edge[0], edge[1], edge[2]) {
			t.Errorf("нет ребра %s -[%s]-> %s", edge[0], edge[2], edge[1])
		}
	}

	if findNode(g, graphID("apps", "ReplicaSet", ns, "other-xyz")) != nil {
		t.Error("чужой ReplicaSet не должен попасть в граф")
	}
	if findNode(g, graphID("", "Pod", ns, "stranger")) != nil || hasEdge(g, svc, graphID("", "Pod", ns, "stranger"), ArgoEdgeSelects) {
		t.Error("под вне приложения не должен попасть в граф, даже если селектор совпал")
	}
	if n := findNode(g, graphID("", "Service", ns, "shared-auth")); n == nil || n.Managed {
		t.Errorf("внешний Service из Ingress должен быть добавлен с managed=false: %+v", n)
	}
	if n := findNode(g, dep); n == nil || !n.Managed || n.Health != "Healthy" || n.Status != "Synced" {
		t.Errorf("managed-узел потерял статусы: %+v", n)
	}
	if findNode(g, graphID("rbac.authorization.k8s.io", "ClusterRole", "", "reader")) == nil {
		t.Error("cluster-scoped ресурс должен быть узлом")
	}
	if g.Truncated || len(g.Warnings) != 0 {
		t.Errorf("truncated=%v warnings=%v", g.Truncated, g.Warnings)
	}
}

func TestBuildArgoGraphEmpty(t *testing.T) {
	g := buildArgoGraph(context.Background(), fake.NewSimpleClientset(), nil)
	if g.Nodes == nil || g.Edges == nil || g.Warnings == nil || len(g.Nodes) != 0 {
		t.Errorf("пустой граф должен сериализоваться как [] без nil: %+v", g)
	}
}

func TestBuildArgoGraphForbiddenPodsGivesWarningNotError(t *testing.T) {
	c := fake.NewSimpleClientset(
		&appsv1.ReplicaSet{ObjectMeta: meta("prod", "web-abc", nil, ctrlOwner("apps/v1", "Deployment", "web"))},
	)
	c.PrependReactor("list", "pods", func(clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "pods"}, "", nil)
	})
	g := buildArgoGraph(context.Background(), c, []ArgoResourceRef{{Group: "apps", Kind: "Deployment", Namespace: "prod", Name: "web"}})

	if len(g.Warnings) != 1 || !strings.Contains(g.Warnings[0], "pods") {
		t.Fatalf("ожидалось одно предупреждение про pods: %v", g.Warnings)
	}
	if !hasEdge(g, graphID("apps", "Deployment", "prod", "web"), graphID("apps", "ReplicaSet", "prod", "web-abc"), ArgoEdgeOwns) {
		t.Error("частичный граф (до подов) должен сохраниться")
	}
}

func TestBuildArgoGraphNodeLimit(t *testing.T) {
	const ns = "big"
	objs := []runtime.Object{
		&appsv1.ReplicaSet{ObjectMeta: meta(ns, "rs", nil, ctrlOwner("apps/v1", "Deployment", "web"))},
	}
	for i := 0; i < argoGraphMaxNodes+50; i++ {
		objs = append(objs, &corev1.Pod{ObjectMeta: meta(ns, fmt.Sprintf("p-%04d", i), nil, ctrlOwner("apps/v1", "ReplicaSet", "rs"))})
	}
	g := buildArgoGraph(context.Background(), fake.NewSimpleClientset(objs...), []ArgoResourceRef{{Group: "apps", Kind: "Deployment", Namespace: ns, Name: "web"}})

	if !g.Truncated || len(g.Nodes) != argoGraphMaxNodes {
		t.Fatalf("truncated=%v nodes=%d, ожидался предел %d", g.Truncated, len(g.Nodes), argoGraphMaxNodes)
	}
	ids := map[string]bool{}
	for _, n := range g.Nodes {
		ids[n.ID] = true
	}
	for _, e := range g.Edges {
		if !ids[e.From] || !ids[e.To] {
			t.Fatalf("висячее ребро после усечения: %+v", e)
		}
	}
}

func TestBuildArgoGraphOwnerWithoutControllerFlagIgnored(t *testing.T) {
	// ownerReference без controller=true не считается владением графа.
	c := fake.NewSimpleClientset(
		&appsv1.ReplicaSet{ObjectMeta: meta("ns", "rs", nil, []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "Deployment", Name: "web"}})},
	)
	g := buildArgoGraph(context.Background(), c, []ArgoResourceRef{{Group: "apps", Kind: "Deployment", Namespace: "ns", Name: "web"}})
	if len(g.Nodes) != 1 || len(g.Edges) != 0 {
		t.Errorf("nodes=%d edges=%d", len(g.Nodes), len(g.Edges))
	}
}

func TestPodGraphHealth(t *testing.T) {
	waiting := func(reason string) corev1.ContainerStatus {
		return corev1.ContainerStatus{State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: reason}}}
	}
	cases := []struct {
		name string
		pod  corev1.Pod
		want string
	}{
		{"running ready", corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{{Ready: true}}}}, "Healthy"},
		{"running not ready", corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{{Ready: false}}}}, "Progressing"},
		{"pending", corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPending}}, "Progressing"},
		{"failed", corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodFailed}}, "Degraded"},
		{"succeeded", corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodSucceeded}}, "Healthy"},
		{"crashloop", corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{waiting("CrashLoopBackOff")}}}, "Degraded"},
		{"imagepull", corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodPending, ContainerStatuses: []corev1.ContainerStatus{waiting("ImagePullBackOff")}}}, "Degraded"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := podGraphHealth(&tc.pod); got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}
