// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"reflect"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func int32p(v int32) *int32 { return &v }

func testDeployment(ns, name string, labels map[string]string, replicas, ready int32) appsv1.Deployment {
	return appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name, Labels: labels, CreationTimestamp: metav1.NewTime(time.Now().Add(-48 * time.Hour))},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32p(replicas),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "c", Image: "img:" + name}}},
			},
		},
		Status: appsv1.DeploymentStatus{ReadyReplicas: ready},
	}
}

func TestBuildApplicationsGroupsByInstanceAndName(t *testing.T) {
	es := map[string]string{
		labelAppInstance: "external-secrets", labelAppName: "external-secrets",
		labelAppVersion: "v0.10.7", labelAppManagedBy: "Helm",
	}
	webhook := map[string]string{labelAppInstance: "external-secrets", labelAppName: "external-secrets-webhook"}
	deps := []appsv1.Deployment{
		testDeployment("es", "external-secrets", es, 1, 1),
		testDeployment("es", "external-secrets-webhook", webhook, 2, 1),
		testDeployment("es", "no-labels", nil, 1, 1), // не приложение
	}
	svcs := []corev1.Service{{
		ObjectMeta: metav1.ObjectMeta{Namespace: "es", Name: "external-secrets-webhook"},
		Spec: corev1.ServiceSpec{
			Selector: webhook,
			Ports:    []corev1.ServicePort{{Port: 443}},
		},
	}}
	pt := networkingv1.PathTypePrefix
	ings := []networkingv1.Ingress{{
		ObjectMeta: metav1.ObjectMeta{Namespace: "es", Name: "ing"},
		Spec: networkingv1.IngressSpec{
			TLS: []networkingv1.IngressTLS{{Hosts: []string{"hook.example.com"}}},
			Rules: []networkingv1.IngressRule{{
				Host: "hook.example.com",
				IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{
					Paths: []networkingv1.HTTPIngressPath{{
						Path: "/", PathType: &pt,
						Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: "external-secrets-webhook"}},
					}},
				}},
			}},
		},
	}}

	apps := buildApplications(deps, nil, nil, svcs, ings, nil)
	if len(apps) != 2 {
		t.Fatalf("ожидалось 2 приложения, получено %d: %+v", len(apps), apps)
	}

	main, hook := apps[0], apps[1]
	if main.Name != "external-secrets" || main.Version != "v0.10.7" || main.ManagedBy != "Helm" || main.Status != "Running" {
		t.Errorf("основное приложение: %+v", main)
	}
	if hook.Name != "external-secrets-webhook" || hook.Status != "Progressing" {
		t.Errorf("webhook: %+v", hook)
	}
	if !reflect.DeepEqual(hook.InternalURLs, []string{"external-secrets-webhook.es.svc.cluster.local"}) {
		t.Errorf("internal urls: %v", hook.InternalURLs)
	}
	if !reflect.DeepEqual(hook.ExposedURLs, []string{"https://hook.example.com"}) {
		t.Errorf("exposed urls: %v", hook.ExposedURLs)
	}
	if len(main.InternalURLs) != 0 || len(main.ExposedURLs) != 0 {
		t.Errorf("у основного приложения нет сервисов: %+v", main)
	}
	if want := map[string]string{labelAppInstance: "external-secrets", labelAppName: "external-secrets"}; !reflect.DeepEqual(main.Selector, want) {
		t.Errorf("selector: %v", main.Selector)
	}
}

func TestBuildApplicationsStatusAndNameCollision(t *testing.T) {
	mk := func(instance string, replicas, ready int32) appsv1.Deployment {
		return testDeployment("ns", "dep-"+instance, map[string]string{labelAppInstance: instance, labelAppName: "web"}, replicas, ready)
	}
	apps := buildApplications([]appsv1.Deployment{mk("a", 0, 0), mk("b", 2, 0)}, nil, nil, nil, nil, nil)
	if len(apps) != 2 {
		t.Fatalf("got %d", len(apps))
	}
	if apps[0].Name != "a/web" || apps[0].Status != "Stopped" {
		t.Errorf("a: %+v", apps[0])
	}
	if apps[1].Name != "b/web" || apps[1].Status != "Degraded" {
		t.Errorf("b: %+v", apps[1])
	}
}

// У argocd нет label instance: селектор не должен содержать подставленный instance, иначе поды не найдутся.
func TestBuildApplicationsSelectorOnlyRealLabels(t *testing.T) {
	labels := map[string]string{labelAppName: "argocd-server", labelAppComponent: "server", labelAppPartOf: "argocd"}
	deps := []appsv1.Deployment{testDeployment("argocd", "argocd-server", labels, 1, 1)}
	others := []labeledObject{
		{kind: "ServiceAccount", name: "argocd-server", namespace: "argocd", labels: labels},
		{kind: "Service", name: "other", namespace: "argocd", labels: map[string]string{labelAppName: "x"}},
		{kind: "Service", name: "argocd-server", namespace: "elsewhere", labels: labels},
	}
	apps := buildApplications(deps, nil, nil, nil, nil, others)
	if len(apps) != 1 {
		t.Fatalf("got %d", len(apps))
	}
	if want := map[string]string{labelAppName: "argocd-server"}; !reflect.DeepEqual(apps[0].Selector, want) {
		t.Errorf("selector: %v", apps[0].Selector)
	}
	if want := []ApplicationResource{{Kind: "ServiceAccount", Name: "argocd-server", Component: "server"}}; !reflect.DeepEqual(apps[0].Resources, want) {
		t.Errorf("resources: %+v", apps[0].Resources)
	}
	if apps[0].Workloads[0].Component != "server" {
		t.Errorf("component: %+v", apps[0].Workloads)
	}
}
