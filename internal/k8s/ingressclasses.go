package k8s

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const defaultIngressClassAnnotation = "ingressclass.kubernetes.io/is-default-class"

type IngressClassInfo struct {
	Name            string            `json:"name"`
	Controller      string            `json:"controller"`
	Default         bool              `json:"default"`
	ParamsAPIGroup  string            `json:"paramsApiGroup"`
	ParamsKind      string            `json:"paramsKind"`
	ParamsName      string            `json:"paramsName"`
	ParamsScope     string            `json:"paramsScope"`
	ParamsNamespace string            `json:"paramsNamespace"`
	Age             string            `json:"age"`
	Created         string            `json:"created"`
	Labels          map[string]string `json:"labels"`
	Annotations     map[string]string `json:"annotations"`
}

func (m *Manager) GetIngressClasses() ([]IngressClassInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	list, err := conn.Client.NetworkingV1().IngressClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	items := make([]IngressClassInfo, 0, len(list.Items))
	for _, ic := range list.Items {
		info := IngressClassInfo{
			Name:        ic.Name,
			Controller:  ic.Spec.Controller,
			Default:     ic.Annotations[defaultIngressClassAnnotation] == "true",
			Age:         formatAge(ic.CreationTimestamp.Time),
			Created:     ic.CreationTimestamp.Format(time.RFC3339),
			Labels:      ic.Labels,
			Annotations: ic.Annotations,
		}
		if p := ic.Spec.Parameters; p != nil {
			info.ParamsKind, info.ParamsName = p.Kind, p.Name
			if p.APIGroup != nil {
				info.ParamsAPIGroup = *p.APIGroup
			}
			if p.Scope != nil {
				info.ParamsScope = *p.Scope
			}
			if p.Namespace != nil {
				info.ParamsNamespace = *p.Namespace
			}
		}
		items = append(items, info)
	}
	return items, nil
}
