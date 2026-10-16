package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type IngressInfo struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Class     string   `json:"class"`
	Hosts     []string `json:"hosts"`
	Paths     string   `json:"paths"`
	Address   string   `json:"address"`
	Age       string   `json:"age"`
	TLS       bool     `json:"tls"`
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

	var items []IngressInfo
	for _, ing := range list.Items {
		var hosts []string
		var paths []string

		for _, rule := range ing.Spec.Rules {
			host := rule.Host
			if host == "" {
				host = "*"
			}
			hosts = append(hosts, host)

			if rule.HTTP != nil {
				for _, p := range rule.HTTP.Paths {
					backend := ""
					if p.Backend.Service != nil {
						backend = fmt.Sprintf("%s:%d", p.Backend.Service.Name, p.Backend.Service.Port.Number)
					}
					paths = append(paths, fmt.Sprintf("%s → %s", p.Path, backend))
				}
			}
		}

		class := ""
		if ing.Spec.IngressClassName != nil {
			class = *ing.Spec.IngressClassName
		}

		address := ""
		if len(ing.Status.LoadBalancer.Ingress) > 0 {
			lb := ing.Status.LoadBalancer.Ingress[0]
			if lb.IP != "" {
				address = lb.IP
			} else {
				address = lb.Hostname
			}
		}

		items = append(items, IngressInfo{
			Name:      ing.Name,
			Namespace: ing.Namespace,
			Class:     class,
			Hosts:     hosts,
			Paths:     strings.Join(paths, "; "),
			Address:   address,
			Age:       formatAge(ing.CreationTimestamp.Time),
			TLS:       len(ing.Spec.TLS) > 0,
		})
	}
	return items, nil
}

func (m *Manager) DeleteIngress(namespace, name string) error {
conn, err := m.GetActiveConnection()
if err != nil {
return err
}

ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
defer cancel()

return conn.Client.NetworkingV1().Ingresses(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}
