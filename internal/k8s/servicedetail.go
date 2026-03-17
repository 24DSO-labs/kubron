package k8s

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ServiceDetail struct {
	Name           string            `json:"name"`
	Namespace      string            `json:"namespace"`
	Type           string            `json:"type"`
	ClusterIP      string            `json:"clusterIP"`
	ClusterIPs     []string          `json:"clusterIPs"`
	IPFamilies     string            `json:"ipFamilies"`
	IPFamilyPolicy string            `json:"ipFamilyPolicy"`
	Ports          []ServicePortInfo `json:"ports"`
	Selector       map[string]string `json:"selector"`
	Labels         map[string]string `json:"labels"`
	Annotations    map[string]string `json:"annotations"`
	Age            string            `json:"age"`
	Endpoints      []EndpointInfo    `json:"endpoints"`
}

type ServicePortInfo struct {
	Name       string `json:"name"`
	Port       int32  `json:"port"`
	TargetPort string `json:"targetPort"`
	Protocol   string `json:"protocol"`
	NodePort   int32  `json:"nodePort"`
	Forwarded  bool   `json:"forwarded"`
	LocalPort  int    `json:"localPort"`
}

type EndpointInfo struct {
	Name      string   `json:"name"`
	Addresses []string `json:"addresses"`
}

func (m *Manager) GetServiceDetail(namespace, name string) (*ServiceDetail, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	svc, err := conn.Client.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	var ports []ServicePortInfo
	for _, p := range svc.Spec.Ports {
		ports = append(ports, ServicePortInfo{
			Name:       p.Name,
			Port:       p.Port,
			TargetPort: p.TargetPort.String(),
			Protocol:   string(p.Protocol),
			NodePort:   p.NodePort,
		})
	}

	var clusterIPs []string
	for _, ip := range svc.Spec.ClusterIPs {
		clusterIPs = append(clusterIPs, ip)
	}

	ipFamilies := ""
	for i, f := range svc.Spec.IPFamilies {
		if i > 0 {
			ipFamilies += ", "
		}
		ipFamilies += string(f)
	}

	ipFamilyPolicy := ""
	if svc.Spec.IPFamilyPolicy != nil {
		ipFamilyPolicy = string(*svc.Spec.IPFamilyPolicy)
	}

	// Получаем endpoints
	var endpoints []EndpointInfo
	ep, err := conn.Client.CoreV1().Endpoints(namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		for _, subset := range ep.Subsets {
			var addrs []string
			for _, addr := range subset.Addresses {
				for _, port := range subset.Ports {
					addrs = append(addrs, fmt.Sprintf("%s:%d", addr.IP, port.Port))
				}
			}
			endpoints = append(endpoints, EndpointInfo{
				Name:      name,
				Addresses: addrs,
			})
		}
	}

	return &ServiceDetail{
		Name:           svc.Name,
		Namespace:      svc.Namespace,
		Type:           string(svc.Spec.Type),
		ClusterIP:      svc.Spec.ClusterIP,
		ClusterIPs:     clusterIPs,
		IPFamilies:     ipFamilies,
		IPFamilyPolicy: ipFamilyPolicy,
		Ports:          ports,
		Selector:       svc.Spec.Selector,
		Labels:         svc.Labels,
		Annotations:    svc.Annotations,
		Age:            formatAge(svc.CreationTimestamp.Time),
		Endpoints:      endpoints,
	}, nil
}
