// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"fmt"
	"sort"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

// ArgoProjectDestination — разрешённое назначение AppProject.
type ArgoProjectDestination struct {
	Server    string `json:"server"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// ArgoGroupKind — group/kind из whitelist/blacklist ресурсов проекта.
type ArgoGroupKind struct {
	Group string `json:"group"`
	Kind  string `json:"kind"`
}

// ArgoProjectRole — роль проекта; JWT-токены (status/jwtTokens) не отдаются.
type ArgoProjectRole struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Policies    []string `json:"policies"`
	Groups      []string `json:"groups"`
}

// ArgoProjectInfo — элемент списка и панели AppProject.
type ArgoProjectInfo struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	Description string `json:"description"`
	Age         string `json:"age"`
	Created     string `json:"created"`

	SourceRepos                []string                 `json:"sourceRepos"`
	Destinations               []ArgoProjectDestination `json:"destinations"`
	ClusterResourceWhitelist   []ArgoGroupKind          `json:"clusterResourceWhitelist"`
	ClusterResourceBlacklist   []ArgoGroupKind          `json:"clusterResourceBlacklist"`
	NamespaceResourceWhitelist []ArgoGroupKind          `json:"namespaceResourceWhitelist"`
	NamespaceResourceBlacklist []ArgoGroupKind          `json:"namespaceResourceBlacklist"`
	Roles                      []ArgoProjectRole        `json:"roles"`
	OrphanedMonitoring         bool                     `json:"orphanedResourcesMonitoring"`

	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

// GetArgoProjects возвращает AppProject из namespace (пустой — из всех).
func (m *Manager) GetArgoProjects(namespaces []string) ([]ArgoProjectInfo, error) {
	return scopedListTimeout(m, 20*time.Second, "GetArgoProjects", "argoproj.io", "appprojects", namespaces,
		func(ctx context.Context, conn *ClusterConnection, ns string) ([]ArgoProjectInfo, error) {
			return listArgoProjects(ctx, conn.DynamicClient, ns)
		},
		func(it ArgoProjectInfo) string { return it.Namespace })
}

func listArgoProjects(ctx context.Context, dyn dynamic.Interface, namespace string) ([]ArgoProjectInfo, error) {
	list, err := dyn.Resource(kindTable["appproject"].gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list appprojects: %w", err)
	}
	out := make([]ArgoProjectInfo, 0, len(list.Items))
	for i := range list.Items {
		out = append(out, parseArgoProject(&list.Items[i]))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Namespace != out[j].Namespace {
			return out[i].Namespace < out[j].Namespace
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func parseArgoProject(u *unstructured.Unstructured) ArgoProjectInfo {
	obj := u.Object
	info := ArgoProjectInfo{
		Name:        u.GetName(),
		Namespace:   u.GetNamespace(),
		Description: strAt(obj, "spec", "description"),
		Labels:      u.GetLabels(),
		Annotations: u.GetAnnotations(),
	}
	if ts := u.GetCreationTimestamp(); !ts.IsZero() {
		info.Age = formatAge(ts.Time)
		info.Created = ts.Format(time.RFC3339)
	}

	info.SourceRepos = []string{}
	if repos, ok := sliceAt(obj, "spec", "sourceRepos"); ok {
		for _, r := range repos {
			if s, ok := r.(string); ok {
				info.SourceRepos = append(info.SourceRepos, stripURLUserinfo(s))
			}
		}
	}

	info.Destinations = []ArgoProjectDestination{}
	if items, ok := sliceAt(obj, "spec", "destinations"); ok {
		for _, it := range items {
			if dm, ok := it.(map[string]interface{}); ok {
				info.Destinations = append(info.Destinations, ArgoProjectDestination{
					Server: strAt(dm, "server"), Name: strAt(dm, "name"), Namespace: strAt(dm, "namespace"),
				})
			}
		}
	}

	info.ClusterResourceWhitelist = argoGroupKinds(obj, "clusterResourceWhitelist")
	info.ClusterResourceBlacklist = argoGroupKinds(obj, "clusterResourceBlacklist")
	info.NamespaceResourceWhitelist = argoGroupKinds(obj, "namespaceResourceWhitelist")
	info.NamespaceResourceBlacklist = argoGroupKinds(obj, "namespaceResourceBlacklist")
	info.Roles = argoProjectRoles(obj)
	_, info.OrphanedMonitoring = mapAt(obj, "spec", "orphanedResources")
	return info
}

func argoGroupKinds(obj map[string]interface{}, field string) []ArgoGroupKind {
	items, _ := sliceAt(obj, "spec", field)
	out := make([]ArgoGroupKind, 0, len(items))
	for _, it := range items {
		if gm, ok := it.(map[string]interface{}); ok {
			out = append(out, ArgoGroupKind{Group: strAt(gm, "group"), Kind: strAt(gm, "kind")})
		}
	}
	return out
}

func stringsAt(m map[string]interface{}, path ...string) []string {
	items, _ := sliceAt(m, path...)
	out := make([]string, 0, len(items))
	for _, it := range items {
		if s, ok := it.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// argoProjectRoles читает spec.roles; jwtTokens намеренно не копируются (чувствительные данные).
func argoProjectRoles(obj map[string]interface{}) []ArgoProjectRole {
	items, _ := sliceAt(obj, "spec", "roles")
	out := make([]ArgoProjectRole, 0, len(items))
	for _, it := range items {
		rm, ok := it.(map[string]interface{})
		if !ok {
			continue
		}
		out = append(out, ArgoProjectRole{
			Name:        strAt(rm, "name"),
			Description: strAt(rm, "description"),
			Policies:    stringsAt(rm, "policies"),
			Groups:      stringsAt(rm, "groups"),
		})
	}
	return out
}
