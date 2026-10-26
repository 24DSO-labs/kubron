// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestPickArgoPort(t *testing.T) {
	ports := []corev1.ServicePort{{Name: "http", Port: 80}, {Name: "https", Port: 443}}
	if p, ok := pickArgoPort(ports, false); !ok || p.Port != 443 {
		t.Errorf("secure: %+v", p)
	}
	if p, ok := pickArgoPort(ports, true); !ok || p.Port != 80 {
		t.Errorf("insecure: %+v", p)
	}
	if p, ok := pickArgoPort([]corev1.ServicePort{{Name: "web", Port: 8080}}, false); !ok || p.Port != 8080 {
		t.Errorf("fallback: %+v", p)
	}
	if _, ok := pickArgoPort(nil, false); ok {
		t.Error("без портов ok должен быть false")
	}
	if argoScheme(false) != "https" || argoScheme(true) != "http" {
		t.Error("схема")
	}
}

func TestFindArgoServerService(t *testing.T) {
	svc := func(ns, name string, labelled bool) *corev1.Service {
		s := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}}
		if labelled {
			s.Labels = map[string]string{"app.kubernetes.io/name": "argocd-server"}
		}
		return s
	}
	ctx := context.Background()

	got, err := findArgoServerService(ctx, fake.NewSimpleClientset(svc("zzz", "argocd-server", true), svc("argocd", "argocd-server", true), svc("x", "other", false)))
	if err != nil || got.Namespace != "argocd" {
		t.Fatalf("ожидался namespace argocd: %+v, %v", got, err)
	}
	if _, err := findArgoServerService(ctx, fake.NewSimpleClientset(svc("x", "other", false))); err == nil {
		t.Error("без сервиса ожидалась ошибка")
	}
}
