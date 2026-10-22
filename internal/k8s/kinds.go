package k8s

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

type kindInfo struct {
	gvr        schema.GroupVersionResource
	namespaced bool
}

func gvr(group, version, resource string) schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: group, Version: version, Resource: resource}
}

// Ключи совпадают с ResourceDefinition.id на фронтенде
var kindTable = map[string]kindInfo{
	"pod":                     {gvr("", "v1", "pods"), true},
	"service":                 {gvr("", "v1", "services"), true},
	"endpoint":                {gvr("", "v1", "endpoints"), true},
	"configmap":               {gvr("", "v1", "configmaps"), true},
	"secret":                  {gvr("", "v1", "secrets"), true},
	"serviceaccount":          {gvr("", "v1", "serviceaccounts"), true},
	"resourcequota":           {gvr("", "v1", "resourcequotas"), true},
	"limitrange":              {gvr("", "v1", "limitranges"), true},
	"persistentvolumeclaim":   {gvr("", "v1", "persistentvolumeclaims"), true},
	"event":                   {gvr("", "v1", "events"), true},
	"node":                    {gvr("", "v1", "nodes"), false},
	"namespace":               {gvr("", "v1", "namespaces"), false},
	"persistentvolume":        {gvr("", "v1", "persistentvolumes"), false},
	"deployment":              {gvr("apps", "v1", "deployments"), true},
	"statefulset":             {gvr("apps", "v1", "statefulsets"), true},
	"daemonset":               {gvr("apps", "v1", "daemonsets"), true},
	"replicaset":              {gvr("apps", "v1", "replicasets"), true},
	"job":                     {gvr("batch", "v1", "jobs"), true},
	"cronjob":                 {gvr("batch", "v1", "cronjobs"), true},
	"ingress":                 {gvr("networking.k8s.io", "v1", "ingresses"), true},
	"networkpolicy":           {gvr("networking.k8s.io", "v1", "networkpolicies"), true},
	"ingressclass":            {gvr("networking.k8s.io", "v1", "ingressclasses"), false},
	"storageclass":            {gvr("storage.k8s.io", "v1", "storageclasses"), false},
	"role":                    {gvr("rbac.authorization.k8s.io", "v1", "roles"), true},
	"rolebinding":             {gvr("rbac.authorization.k8s.io", "v1", "rolebindings"), true},
	"clusterrole":             {gvr("rbac.authorization.k8s.io", "v1", "clusterroles"), false},
	"clusterrolebinding":      {gvr("rbac.authorization.k8s.io", "v1", "clusterrolebindings"), false},
	"horizontalpodautoscaler": {gvr("autoscaling", "v2", "horizontalpodautoscalers"), true},
	"poddisruptionbudget":     {gvr("policy", "v1", "poddisruptionbudgets"), true},
	"priorityclass":           {gvr("scheduling.k8s.io", "v1", "priorityclasses"), false},
	"runtimeclass":            {gvr("node.k8s.io", "v1", "runtimeclasses"), false},
	"lease":                   {gvr("coordination.k8s.io", "v1", "leases"), true},
	"endpointslice":           {gvr("discovery.k8s.io", "v1", "endpointslices"), true},

	"mutatingwebhookconfiguration":   {gvr("admissionregistration.k8s.io", "v1", "mutatingwebhookconfigurations"), false},
	"validatingwebhookconfiguration": {gvr("admissionregistration.k8s.io", "v1", "validatingwebhookconfigurations"), false},

	// Argo CD (CRD; читаются через dynamic client, доступны только если ArgoInstalled)
	"application":              {gvr(argoGroup, "v1alpha1", "applications"), true},
	"applicationset":           {gvr(argoGroup, "v1alpha1", "applicationsets"), true},
	"appproject":               {gvr(argoGroup, "v1alpha1", "appprojects"), true},
	"customresourcedefinition": {gvr("apiextensions.k8s.io", "v1", "customresourcedefinitions"), false},
}

const customKindPrefix = "cr:" // "cr:group/version/plural", проверяется через CRD (см. customKindFor)

func init() {
	kindTable["hpa"] = kindTable["horizontalpodautoscaler"]
	kindTable["pdb"] = kindTable["poddisruptionbudget"]
}

// resourceClient возвращает dynamic-клиент нужного ресурса (с учётом namespace)
func resourceClient(conn *ClusterConnection, kind, namespace string) (dynamic.ResourceInterface, kindInfo, error) {
	info, ok := kindTable[kind]
	if !ok {
		if !strings.HasPrefix(kind, customKindPrefix) {
			return nil, kindInfo{}, fmt.Errorf("тип ресурса %q не поддерживается", kind)
		}
		var err error
		if info, err = customKindInfo(conn, kind); err != nil {
			return nil, kindInfo{}, err
		}
	}
	if !info.namespaced {
		return conn.DynamicClient.Resource(info.gvr), info, nil
	}
	if namespace == "" {
		return nil, info, fmt.Errorf("для %s нужен namespace", kind)
	}
	return conn.DynamicClient.Resource(info.gvr).Namespace(namespace), info, nil
}
