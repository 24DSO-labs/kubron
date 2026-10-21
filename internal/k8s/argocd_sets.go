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

// ArgoSetApp — приложение, созданное ApplicationSet (status.resources[] / status.applicationStatus[]).
type ArgoSetApp struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // Synced | OutOfSync | ... либо Healthy | Waiting | Pending | Progressing
	Health  string `json:"health"`
	Message string `json:"message"`
}

// ArgoApplicationSetInfo — элемент списка и панели ApplicationSet.
type ArgoApplicationSetInfo struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Age       string `json:"age"`
	Created   string `json:"created"`

	Generators      []string `json:"generators"`      // типы генераторов: list, clusters, git, matrix ...
	TemplateProject string   `json:"templateProject"` // spec.template.spec.project
	RepoURL         string   `json:"repoURL"`         // из template (без user:token@)
	DestNamespace   string   `json:"destNamespace"`
	SyncPolicy      string   `json:"syncPolicy"` // "" | "create-only" и т.п. (spec.syncPolicy.applicationsSync)
	Preserve        bool     `json:"preserveResourcesOnDeletion"`

	Applications []ArgoSetApp    `json:"applications"`
	AppCount     int             `json:"appCount"`
	Conditions   []ConditionInfo `json:"conditions"`
	Healthy      bool            `json:"healthy"` // нет активных ошибочных conditions

	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

// GetArgoApplicationSets возвращает ApplicationSet из namespace (пустой — из всех).
func (m *Manager) GetArgoApplicationSets(namespace string) ([]ArgoApplicationSetInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return listArgoApplicationSets(ctx, conn.DynamicClient, namespace)
}

func listArgoApplicationSets(ctx context.Context, dyn dynamic.Interface, namespace string) ([]ArgoApplicationSetInfo, error) {
	list, err := dyn.Resource(kindTable["applicationset"].gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list applicationsets: %w", err)
	}
	out := make([]ArgoApplicationSetInfo, 0, len(list.Items))
	for i := range list.Items {
		out = append(out, parseArgoApplicationSet(&list.Items[i]))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Namespace != out[j].Namespace {
			return out[i].Namespace < out[j].Namespace
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// parseArgoApplicationSet защитно разбирает unstructured ApplicationSet.
func parseArgoApplicationSet(u *unstructured.Unstructured) ArgoApplicationSetInfo {
	obj := u.Object
	info := ArgoApplicationSetInfo{
		Name:            u.GetName(),
		Namespace:       u.GetNamespace(),
		Labels:          u.GetLabels(),
		Annotations:     u.GetAnnotations(),
		TemplateProject: strAt(obj, "spec", "template", "spec", "project"),
		RepoURL:         stripURLUserinfo(strAt(obj, "spec", "template", "spec", "source", "repoURL")),
		DestNamespace:   strAt(obj, "spec", "template", "spec", "destination", "namespace"),
		SyncPolicy:      strAt(obj, "spec", "syncPolicy", "applicationsSync"),
		Preserve:        boolAt(obj, "spec", "syncPolicy", "preserveResourcesOnDeletion"),
	}
	if ts := u.GetCreationTimestamp(); !ts.IsZero() {
		info.Age = formatAge(ts.Time)
		info.Created = ts.Format(time.RFC3339)
	}
	if info.RepoURL == "" {
		if srcs, ok := sliceAt(obj, "spec", "template", "spec", "sources"); ok && len(srcs) > 0 {
			if sm, ok := srcs[0].(map[string]interface{}); ok {
				info.RepoURL = stripURLUserinfo(strAt(sm, "repoURL"))
			}
		}
	}

	info.Generators = argoGeneratorTypes(obj)
	info.Applications = argoSetApps(obj)
	info.AppCount = len(info.Applications)
	info.Conditions, info.Healthy = argoSetConditions(obj)
	return info
}

// argoGeneratorTypes: ключи каждого элемента spec.generators (list, git, clusters, matrix ...), по порядку.
func argoGeneratorTypes(obj map[string]interface{}) []string {
	items, _ := sliceAt(obj, "spec", "generators")
	out := make([]string, 0, len(items))
	for _, it := range items {
		gm, ok := it.(map[string]interface{})
		if !ok {
			continue
		}
		keys := make([]string, 0, len(gm))
		for k := range gm {
			// selector/template/values/... — не генераторы, у них нет собственного типа
			if k == "selector" || k == "template" || k == "values" {
				continue
			}
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out = append(out, keys...)
	}
	return out
}

// argoSetApps: status.resources[] (Argo 2.7+) с fallback на status.applicationStatus[].
func argoSetApps(obj map[string]interface{}) []ArgoSetApp {
	var out []ArgoSetApp
	if items, ok := sliceAt(obj, "status", "resources"); ok {
		for _, it := range items {
			rm, ok := it.(map[string]interface{})
			if !ok {
				continue
			}
			out = append(out, ArgoSetApp{
				Name:   strAt(rm, "name"),
				Status: strAt(rm, "status"),
				Health: strAt(rm, "health", "status"),
			})
		}
	}
	if len(out) == 0 {
		items, _ := sliceAt(obj, "status", "applicationStatus")
		for _, it := range items {
			am, ok := it.(map[string]interface{})
			if !ok {
				continue
			}
			out = append(out, ArgoSetApp{
				Name:    strAt(am, "application"),
				Status:  strAt(am, "status"),
				Message: strAt(am, "message"),
			})
		}
	}
	if out == nil {
		out = []ArgoSetApp{}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// argoSetConditions: у ApplicationSet condition есть status (True/False); ErrorOccurred=True — проблема.
func argoSetConditions(obj map[string]interface{}) ([]ConditionInfo, bool) {
	items, _ := sliceAt(obj, "status", "conditions")
	out := make([]ConditionInfo, 0, len(items))
	healthy := true
	for _, it := range items {
		cm, ok := it.(map[string]interface{})
		if !ok {
			continue
		}
		c := ConditionInfo{
			Type:               strAt(cm, "type"),
			Status:             strAt(cm, "status"),
			LastTransitionTime: strAt(cm, "lastTransitionTime"),
			Reason:             strAt(cm, "reason"),
			Message:            strAt(cm, "message"),
		}
		if c.Type == "ErrorOccurred" && c.Status == "True" {
			healthy = false
		}
		out = append(out, c)
	}
	return out, healthy
}
