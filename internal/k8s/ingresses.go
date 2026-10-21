package k8s

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type IngressBackendInfo struct {
	Service  string `json:"service"`
	Port     string `json:"port"`
	Resource string `json:"resource"` // Kind/name, если backend — не сервис
}

type IngressRuleInfo struct {
	Host     string             `json:"host"`
	Path     string             `json:"path"`
	PathType string             `json:"pathType"`
	Backend  IngressBackendInfo `json:"backend"`
}

type IngressTLSInfo struct {
	Hosts      []string `json:"hosts"`
	SecretName string   `json:"secretName"`
}

type IngressInfo struct {
	Name           string              `json:"name"`
	Namespace      string              `json:"namespace"`
	Class          string              `json:"class"`
	Hosts          []string            `json:"hosts"`
	Paths          string              `json:"paths"`
	Address        string              `json:"address"` // load balancers через запятую
	Age            string              `json:"age"`
	Created        string              `json:"created"`
	TLS            bool                `json:"tls"`
	Labels         map[string]string   `json:"labels"`
	ControlledBy   string              `json:"controlledBy"`
	ManagedBy      string              `json:"managedBy"`
	Annotations    map[string]string   `json:"annotations"`
	Rules          []IngressRuleInfo   `json:"rules"`
	TLSEntries     []IngressTLSInfo    `json:"tlsEntries"`
	DefaultBackend *IngressBackendInfo `json:"defaultBackend"`
	Ports          string              `json:“ports”`
}

// ingressPorts: порты, на которых Ingress принимает трафик (как в Lens): 80 для HTTP-правил, 443 при TLS.
func ingressPorts(ing *networkingv1.Ingress) string {
	var ports []string
	switch {
	case len(ing.Spec.Rules) > 0:
		for _, r := range ing.Spec.Rules {
			if r.HTTP != nil {
				ports = append(ports, "80")
				break
			}
		}
	case ing.Spec.DefaultBackend != nil && ing.Spec.DefaultBackend.Service != nil &&
		ing.Spec.DefaultBackend.Service.Port.Number != 0:
		ports = append(ports, strconv.Itoa(int(ing.Spec.DefaultBackend.Service.Port.Number)))
	}
	if len(ing.Spec.TLS) > 0 {
		ports = append(ports, "443")
	}
	return strings.Join(ports, ", ")
}

func ingressBackend(b networkingv1.IngressBackend) IngressBackendInfo {
	out := IngressBackendInfo{}
	if s := b.Service; s != nil {
		out.Service = s.Name
		if s.Port.Number != 0 {
			out.Port = strconv.Itoa(int(s.Port.Number))
		} else {
			out.Port = s.Port.Name
		}
	}
	if r := b.Resource; r != nil {
		out.Resource = r.Kind + "/" + r.Name
	}
	return out
}

func (b IngressBackendInfo) String() string {
	if b.Service != "" {
		return b.Service + ":" + b.Port
	}
	return b.Resource
}

func convertIngress(ing networkingv1.Ingress) IngressInfo {
	info := IngressInfo{
		Name:         ing.Name,
		Namespace:    ing.Namespace,
		Age:          formatAge(ing.CreationTimestamp.Time),
		Created:      ing.CreationTimestamp.Format(time.RFC3339),
		TLS:          len(ing.Spec.TLS) > 0,
		Labels:       ing.Labels,
		ControlledBy: controlledByOf(ing.OwnerReferences),
		ManagedBy:    managedByOf(ing.ObjectMeta),
		Annotations:  ing.Annotations,
		Hosts:        []string{},
		Rules:        []IngressRuleInfo{},
		TLSEntries:   []IngressTLSInfo{},
	}

	info.Class = ingressClassOf(&ing)

	var paths, lbs []string
	for _, rule := range ing.Spec.Rules {
		host := rule.Host
		if host == "" {
			host = "*"
		}
		info.Hosts = append(info.Hosts, host)
		if rule.HTTP == nil {
			continue
		}
		for _, p := range rule.HTTP.Paths {
			r := IngressRuleInfo{Host: host, Path: p.Path, Backend: ingressBackend(p.Backend)}
			if r.Path == "" {
				r.Path = "/"
			}
			if p.PathType != nil {
				r.PathType = string(*p.PathType)
			}
			info.Rules = append(info.Rules, r)
			paths = append(paths, fmt.Sprintf("%s → %s", r.Path, r.Backend))
		}
	}
	info.Paths = strings.Join(paths, "; ")

	if db := ing.Spec.DefaultBackend; db != nil {
		b := ingressBackend(*db)
		info.DefaultBackend = &b
	}
	for _, t := range ing.Spec.TLS {
		hosts := t.Hosts
		if hosts == nil {
			hosts = []string{}
		}
		info.TLSEntries = append(info.TLSEntries, IngressTLSInfo{Hosts: hosts, SecretName: t.SecretName})
	}
	for _, lb := range ing.Status.LoadBalancer.Ingress {
		if lb.IP != "" {
			lbs = append(lbs, lb.IP)
		} else if lb.Hostname != "" {
			lbs = append(lbs, lb.Hostname)
		}
	}
	info.Address = strings.Join(lbs, ", ")
	info.Ports = ingressPorts(&ing)

	return info
}

func (m *Manager) GetIngresses(namespace string) ([]IngressInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	list, err := conn.Client.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	items := make([]IngressInfo, 0, len(list.Items))
	for _, ing := range list.Items {
		items = append(items, convertIngress(ing))
	}
	return items, nil
}

// ingressClassOf: spec.ingressClassName, иначе устаревшая аннотация.
func ingressClassOf(ing *networkingv1.Ingress) string {
	if ing.Spec.IngressClassName != nil {
		return *ing.Spec.IngressClassName
	}
	return ing.Annotations["kubernetes.io/ingress.class"]
}
