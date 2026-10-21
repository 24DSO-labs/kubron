package k8s

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func argoObj(kind, ns, name string, extra map[string]interface{}) *unstructured.Unstructured {
	obj := map[string]interface{}{
		"apiVersion": argoGroupVersion,
		"kind":       kind,
		"metadata":   map[string]interface{}{"name": name, "namespace": ns},
	}
	for k, v := range extra {
		obj[k] = v
	}
	return &unstructured.Unstructured{Object: obj}
}

func TestParseArgoApplicationSet(t *testing.T) {
	t.Run("пустой объект", func(t *testing.T) {
		got := parseArgoApplicationSet(argoObj("ApplicationSet", "argocd", "empty", nil))
		if got.AppCount != 0 || len(got.Generators) != 0 || !got.Healthy || got.Applications == nil {
			t.Errorf("defaults: %+v", got)
		}
	})

	t.Run("генераторы, шаблон, приложения, ошибка", func(t *testing.T) {
		got := parseArgoApplicationSet(argoObj("ApplicationSet", "argocd", "set", map[string]interface{}{
			"spec": map[string]interface{}{
				"generators": []interface{}{
					map[string]interface{}{"git": map[string]interface{}{}, "selector": map[string]interface{}{}},
					map[string]interface{}{"list": map[string]interface{}{}},
				},
				"template": map[string]interface{}{"spec": map[string]interface{}{
					"project": "team-a",
					"source":  map[string]interface{}{"repoURL": "https://u:tok@git.example.com/o/r.git"},
				}},
				"syncPolicy": map[string]interface{}{"applicationsSync": "create-only", "preserveResourcesOnDeletion": true},
			},
			"status": map[string]interface{}{
				"resources": []interface{}{
					map[string]interface{}{"name": "b", "status": "OutOfSync", "health": map[string]interface{}{"status": "Degraded"}},
					map[string]interface{}{"name": "a", "status": "Synced", "health": map[string]interface{}{"status": "Healthy"}},
				},
				"conditions": []interface{}{
					map[string]interface{}{"type": "ErrorOccurred", "status": "True", "message": "boom"},
					map[string]interface{}{"type": "ResourcesUpToDate", "status": "False"},
				},
			},
		}))
		if !reflect.DeepEqual(got.Generators, []string{"git", "list"}) {
			t.Errorf("generators: %v", got.Generators)
		}
		if got.TemplateProject != "team-a" || got.RepoURL != "https://git.example.com/o/r.git" {
			t.Errorf("template: %+v", got)
		}
		if got.SyncPolicy != "create-only" || !got.Preserve {
			t.Errorf("policy: %+v", got)
		}
		if got.AppCount != 2 || got.Applications[0].Name != "a" || got.Applications[1].Health != "Degraded" {
			t.Errorf("apps: %+v", got.Applications)
		}
		if got.Healthy || len(got.Conditions) != 2 {
			t.Errorf("conditions: %+v healthy=%v", got.Conditions, got.Healthy)
		}
	})

	t.Run("fallback на applicationStatus", func(t *testing.T) {
		got := parseArgoApplicationSet(argoObj("ApplicationSet", "argocd", "old", map[string]interface{}{
			"status": map[string]interface{}{"applicationStatus": []interface{}{
				map[string]interface{}{"application": "x", "status": "Healthy", "message": "ok"},
			}},
		}))
		if got.AppCount != 1 || got.Applications[0].Name != "x" || got.Applications[0].Status != "Healthy" {
			t.Errorf("apps: %+v", got.Applications)
		}
	})
}

func TestParseArgoProject(t *testing.T) {
	t.Run("пустой объект", func(t *testing.T) {
		got := parseArgoProject(argoObj("AppProject", "argocd", "p", nil))
		if got.SourceRepos == nil || got.Destinations == nil || got.Roles == nil {
			t.Errorf("срезы должны быть не-nil (JSON []): %+v", got)
		}
	})

	t.Run("поля проекта", func(t *testing.T) {
		got := parseArgoProject(argoObj("AppProject", "argocd", "p", map[string]interface{}{
			"spec": map[string]interface{}{
				"description": "team",
				"sourceRepos": []interface{}{"*", "https://u:tok@git.example.com/o/r.git"},
				"destinations": []interface{}{
					map[string]interface{}{"server": "https://kubernetes.default.svc", "namespace": "prod"},
				},
				"clusterResourceWhitelist":   []interface{}{map[string]interface{}{"group": "*", "kind": "*"}},
				"namespaceResourceBlacklist": []interface{}{map[string]interface{}{"group": "", "kind": "ResourceQuota"}},
				"orphanedResources":          map[string]interface{}{"warn": true},
				"roles": []interface{}{map[string]interface{}{
					"name": "ci", "policies": []interface{}{"p, proj:p:ci, applications, sync, p/*, allow"},
					"groups":    []interface{}{"devs"},
					"jwtTokens": []interface{}{map[string]interface{}{"iat": int64(1), "id": "secret-token-id"}},
				}},
			},
		}))
		if got.Description != "team" || len(got.SourceRepos) != 2 || got.SourceRepos[1] != "https://git.example.com/o/r.git" {
			t.Errorf("repos: %+v", got)
		}
		if len(got.Destinations) != 1 || got.Destinations[0].Namespace != "prod" {
			t.Errorf("dest: %+v", got.Destinations)
		}
		if len(got.ClusterResourceWhitelist) != 1 || got.NamespaceResourceBlacklist[0].Kind != "ResourceQuota" || !got.OrphanedMonitoring {
			t.Errorf("lists: %+v", got)
		}
		if len(got.Roles) != 1 || got.Roles[0].Groups[0] != "devs" {
			t.Errorf("roles: %+v", got.Roles)
		}
		// Отрицательный тест: JWT-токены не должны попасть в ответ.
		raw, _ := json.Marshal(got)
		if strings.Contains(string(raw), "secret-token-id") || strings.Contains(string(raw), "u:tok") {
			t.Errorf("чувствительные данные в ответе: %s", raw)
		}
	})
}

func TestListArgoSetsAndProjects(t *testing.T) {
	scheme := runtime.NewScheme()
	gvrToList := map[schema.GroupVersionResource]string{
		kindTable["applicationset"].gvr: "ApplicationSetList",
		kindTable["appproject"].gvr:     "AppProjectList",
	}
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToList,
		argoObj("ApplicationSet", "b-ns", "z", nil),
		argoObj("ApplicationSet", "a-ns", "y", nil),
		argoObj("AppProject", "argocd", "default", nil),
	)
	sets, err := listArgoApplicationSets(context.Background(), dyn, "")
	if err != nil || len(sets) != 2 || sets[0].Namespace != "a-ns" {
		t.Fatalf("sets: %v %+v", err, sets)
	}
	projs, err := listArgoProjects(context.Background(), dyn, "argocd")
	if err != nil || len(projs) != 1 || projs[0].Name != "default" {
		t.Fatalf("projects: %v %+v", err, projs)
	}
}
