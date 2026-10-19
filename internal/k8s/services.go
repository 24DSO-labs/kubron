package k8s

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ServiceInfo struct {
	Name       string            `json:"name"`
	Namespace  string            `json:"namespace"`
	Type       string            `json:"type"`
	ClusterIP  string            `json:"clusterIP"`
	ExternalIP string            `json:"externalIP"`
	Ports      []string          `json:"ports"`
	Age        string            `json:"age"`
	Labels     map[string]string `json:"labels"`
	Selector   map[string]string `json:"selector"`
	Status     string            `json:"status"`
}

func (m *Manager) GetServices(namespace string) ([]ServiceInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	svcList, err := conn.Client.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var services []ServiceInfo
	for _, svc := range svcList.Items {
		var ports []string
		for _, p := range svc.Spec.Ports {
			port := fmt.Sprintf("%d/%s", p.Port, p.Protocol)
			if p.NodePort > 0 {
				port = fmt.Sprintf("%d:%d/%s", p.Port, p.NodePort, p.Protocol)
			}
			ports = append(ports, port)
		}

		externalIP := ""
		if len(svc.Spec.ExternalIPs) > 0 {
			externalIP = svc.Spec.ExternalIPs[0]
		}
		if svc.Spec.Type == "LoadBalancer" && len(svc.Status.LoadBalancer.Ingress) > 0 {
			ing := svc.Status.LoadBalancer.Ingress[0]
			if ing.IP != "" {
				externalIP = ing.IP
			} else {
				externalIP = ing.Hostname
			}
		}

		status := "Active"
		if svc.Spec.Type == "LoadBalancer" && externalIP == "" {
			status = "Pending"
		}

		services = append(services, ServiceInfo{
			Name:       svc.Name,
			Namespace:  svc.Namespace,
			Type:       string(svc.Spec.Type),
			ClusterIP:  svc.Spec.ClusterIP,
			ExternalIP: externalIP,
			Ports:      ports,
			Age:        formatAge(svc.CreationTimestamp.Time),
			Labels:     svc.Labels,
			Selector:   svc.Spec.Selector,
			Status:     status,
		})
	}

	return services, nil
}
