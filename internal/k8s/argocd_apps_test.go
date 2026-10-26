// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func argoApp(ns, name string, extra map[string]interface{}) *unstructured.Unstructured {
	obj := map[string]interface{}{
		"apiVersion": argoGroupVersion,
		"kind":       "Application",
		"metadata":   map[string]interface{}{"name": name, "namespace": ns},
	}
	for k, v := range extra {
		obj[k] = v
	}
	return &unstructured.Unstructured{Object: obj}
}

func TestParseArgoApplication(t *testing.T) {
	cases := []struct {
		name  string
		extra map[string]interface{}
		check func(t *testing.T, got ArgoApplicationInfo)
	}{
		{
			name:  "пустой объект: безопасные значения по умолчанию",
			extra: nil,
			check: func(t *testing.T, got ArgoApplicationInfo) {
				if got.Project != "default" || got.SyncStatus != "Unknown" || got.HealthStatus != "Unknown" {
					t.Errorf("defaults: %+v", got)
				}
				if got.AutoSync || got.OperationOn || len(got.Sources) != 0 {
					t.Errorf("неожиданные флаги/источники: %+v", got)
				}
			},
		},
		{
			name: "одиночный spec.source",
			extra: map[string]interface{}{"spec": map[string]interface{}{
				"project": "team-a",
				"source":  map[string]interface{}{"repoURL": "https://github.com/acme/app", "path": "deploy", "targetRevision": "main"},
				"destination": map[string]interface{}{
					"server": "https://kubernetes.default.svc", "namespace": "prod",
				},
			}},
			check: func(t *testing.T, got ArgoApplicationInfo) {
				if got.Project != "team-a" || got.RepoURL != "https://github.com/acme/app" || got.Path != "deploy" || got.TargetRevision != "main" {
					t.Errorf("source: %+v", got)
				}
				if got.DestServer != "https://kubernetes.default.svc" || got.DestNamespace != "prod" || len(got.Sources) != 1 {
					t.Errorf("dest/sources: %+v", got)
				}
			},
		},
		{
			name: "multi-source имеет приоритет, userinfo вырезается из repoURL",
			extra: map[string]interface{}{"spec": map[string]interface{}{
				"source": map[string]interface{}{"repoURL": "https://old/ignored"},
				"sources": []interface{}{
					map[string]interface{}{"repoURL": "https://user:s3cret@git.example.com/org/chart.git", "chart": "web", "targetRevision": "1.2.3"},
					map[string]interface{}{"repoURL": "https://git.example.com/org/values.git", "ref": "values"},
				},
			}},
			check: func(t *testing.T, got ArgoApplicationInfo) {
				if len(got.Sources) != 2 {
					t.Fatalf("sources: %+v", got.Sources)
				}
				if got.RepoURL != "https://git.example.com/org/chart.git" || got.Chart != "web" {
					t.Errorf("первый источник: %+v", got)
				}
				if got.Sources[1].Ref != "values" {
					t.Errorf("ref: %+v", got.Sources[1])
				}
			},
		},
		{
			name: "automated syncPolicy и операция",
			extra: map[string]interface{}{
				"spec": map[string]interface{}{"syncPolicy": map[string]interface{}{
					"automated": map[string]interface{}{"prune": true, "selfHeal": false},
				}},
				"status": map[string]interface{}{"operationState": map[string]interface{}{
					"phase": "Running", "message": "syncing",
				}},
			},
			check: func(t *testing.T, got ArgoApplicationInfo) {
				if !got.AutoSync || !got.Prune || got.SelfHeal {
					t.Errorf("syncPolicy: %+v", got)
				}
				if !got.OperationOn || got.LastSyncPhase != "Running" || got.Message != "syncing" {
					t.Errorf("operation: %+v", got)
				}
			},
		},
		{
			name: "пустой automated {} = автосинк",
			extra: map[string]interface{}{"spec": map[string]interface{}{
				"syncPolicy": map[string]interface{}{"automated": map[string]interface{}{}},
			}},
			check: func(t *testing.T, got ArgoApplicationInfo) {
				if !got.AutoSync {
					t.Errorf("ожидался autoSync: %+v", got)
				}
			},
		},
		{
			name: "status: sync, health, ресурсы, conditions, история",
			extra: map[string]interface{}{"status": map[string]interface{}{
				"sync":   map[string]interface{}{"status": "OutOfSync", "revisions": []interface{}{"abc123"}},
				"health": map[string]interface{}{"status": "Degraded", "message": "pod crashloop"},
				"operationState": map[string]interface{}{
					"phase": "Failed", "finishedAt": "2026-01-02T03:04:05Z",
				},
				"conditions": []interface{}{
					map[string]interface{}{"type": "SyncError", "message": "boom", "lastTransitionTime": "2026-01-02T03:04:05Z"},
					"мусор",
				},
				"resources": []interface{}{
					map[string]interface{}{
						"group": "apps", "version": "v1", "kind": "Deployment", "namespace": "prod", "name": "web",
						"status": "OutOfSync", "health": map[string]interface{}{"status": "Degraded", "message": "x"},
					},
					map[string]interface{}{"kind": "Job", "name": "pre", "hook": true},
				},
				"history": []interface{}{
					map[string]interface{}{"id": int64(1), "revision": "r1", "deployedAt": "2026-01-01T00:00:00Z",
						"source": map[string]interface{}{"repoURL": "https://h/r", "targetRevision": "main", "path": "p"}},
					map[string]interface{}{"id": float64(2), "revision": "r2", "deployedAt": "2026-01-02T00:00:00Z",
						"sources": []interface{}{
							map[string]interface{}{"repoURL": "https://h/r", "chart": "c"},
							map[string]interface{}{"repoURL": "https://h/v"},
						}},
				},
			}},
			check: func(t *testing.T, got ArgoApplicationInfo) {
				if got.SyncStatus != "OutOfSync" || got.HealthStatus != "Degraded" || got.Revision != "abc123" {
					t.Errorf("status: %+v", got)
				}
				if got.LastSyncAt != "2026-01-02T03:04:05Z" || got.LastSyncPhase != "Failed" || got.OperationOn {
					t.Errorf("last sync: %+v", got)
				}
				if len(got.Conditions) != 1 || got.Conditions[0].Type != "SyncError" || got.Conditions[0].Message != "boom" {
					t.Errorf("conditions: %+v", got.Conditions)
				}
				want := []ArgoResourceRef{
					{Group: "apps", Version: "v1", Kind: "Deployment", Namespace: "prod", Name: "web", Status: "OutOfSync", Health: "Degraded", Message: "x"},
					{Kind: "Job", Name: "pre", Hook: true},
				}
				if !reflect.DeepEqual(got.Resources, want) {
					t.Errorf("resources: %+v", got.Resources)
				}
				wantHist := []ArgoHistoryEntry{
					{ID: 2, Revision: "r2", DeployedAt: "2026-01-02T00:00:00Z", Source: "https://h/r (chart c) +1"},
					{ID: 1, Revision: "r1", DeployedAt: "2026-01-01T00:00:00Z", Source: "https://h/r@main (p)"},
				}
				if !reflect.DeepEqual(got.History, wantHist) {
					t.Errorf("history: %+v", got.History)
				}
			},
		},
		{
			name: "неожиданные типы полей не приводят к панике",
			extra: map[string]interface{}{
				"spec":   map[string]interface{}{"source": "строка", "sources": 5, "project": 7},
				"status": map[string]interface{}{"resources": "x", "history": map[string]interface{}{}, "sync": []interface{}{}},
			},
			check: func(t *testing.T, got ArgoApplicationInfo) {
				if got.SyncStatus != "Unknown" || len(got.Sources) != 0 || len(got.Resources) != 0 {
					t.Errorf("%+v", got)
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.check(t, parseArgoApplication(argoApp("argocd", "app", tc.extra)))
		})
	}
}

func TestListArgoApplications(t *testing.T) {
	scheme := runtime.NewScheme()
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme,
		map[schema.GroupVersionResource]string{kindTable["application"].gvr: "ApplicationList"},
		argoApp("argocd", "zeta", nil),
		argoApp("team-b", "alpha", nil),
		argoApp("argocd", "alpha", nil),
	)

	all, err := listArgoApplications(context.Background(), dyn, "")
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, a := range all {
		keys = append(keys, a.Namespace+"/"+a.Name)
	}
	if want := []string{"argocd/alpha", "argocd/zeta", "team-b/alpha"}; !reflect.DeepEqual(keys, want) {
		t.Errorf("все namespace, отсортировано: %v", keys)
	}

	one, err := listArgoApplications(context.Background(), dyn, "team-b")
	if err != nil {
		t.Fatal(err)
	}
	if len(one) != 1 || one[0].Name != "alpha" {
		t.Errorf("фильтр по namespace: %+v", one)
	}
}

func TestStripURLUserinfo(t *testing.T) {
	for in, want := range map[string]string{
		"":                                    "",
		"https://github.com/a/b":              "https://github.com/a/b",
		"https://u:p@github.com/a/b.git":      "https://github.com/a/b.git",
		"git@github.com:a/b.git":              "git@github.com:a/b.git",
		"https://token@host.example/org/repo": "https://host.example/org/repo",
	} {
		if got := stripURLUserinfo(in); got != want {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}
}
