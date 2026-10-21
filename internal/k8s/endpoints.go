package k8s

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RefInfo — ссылка на объект (targetRef и т.п.).
type RefInfo struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type EndpointAddressInfo struct {
	IP       string   `json:"ip"`
	Hostname string   `json:"hostname"`
	Node     string   `json:"node"`
	Ready    bool     `json:"ready"`
	Target   *RefInfo `json:"target"`
}

type EndpointPortInfo struct {
	Name     string `json:"name"`
	Port     int32  `json:"port"`
	Protocol string `json:"protocol"`
}

type EndpointSubsetInfo struct {
	Addresses []EndpointAddressInfo `json:"addresses"`
	Ports     []EndpointPortInfo    `json:"ports"`
}

type EndpointListInfo struct {
	Name        string               `json:"name"`
	Namespace   string               `json:"namespace"`
	Endpoints   string               `json:"endpoints"` // "ip:port, ip:port +N"
	Addresses   int                  `json:"addresses"`
	Ports       int                  `json:"ports"`
	Age         string               `json:"age"`
	Created     string               `json:"created"`
	Labels      map[string]string    `json:"labels"`
	Annotations map[string]string    `json:"annotations"`
	Subsets     []EndpointSubsetInfo `json:"subsets"`
}

func convertEndpoints(ep corev1.Endpoints) EndpointListInfo {
	info := EndpointListInfo{
		Name:        ep.Name,
		Namespace:   ep.Namespace,
		Age:         formatAge(ep.CreationTimestamp.Time),
		Created:     ep.CreationTimestamp.Format(time.RFC3339),
		Labels:      ep.Labels,
		Annotations: ep.Annotations,
		Subsets:     []EndpointSubsetInfo{},
	}

	var flat []string
	for _, ss := range ep.Subsets {
		sub := EndpointSubsetInfo{Addresses: []EndpointAddressInfo{}, Ports: []EndpointPortInfo{}}
		for _, p := range ss.Ports {
			sub.Ports = append(sub.Ports, EndpointPortInfo{Name: p.Name, Port: p.Port, Protocol: string(p.Protocol)})
		}
		info.Ports += len(ss.Ports)

		add := func(list []corev1.EndpointAddress, ready bool) {
			for _, a := range list {
				addr := EndpointAddressInfo{IP: a.IP, Hostname: a.Hostname, Ready: ready}
				if a.NodeName != nil {
					addr.Node = *a.NodeName
				}
				if r := a.TargetRef; r != nil {
					addr.Target = &RefInfo{Kind: r.Kind, Name: r.Name, Namespace: r.Namespace}
				}
				sub.Addresses = append(sub.Addresses, addr)
				if ready {
					info.Addresses++
					for _, p := range ss.Ports {
						flat = append(flat, hostPort(a.IP, p.Port))
					}
				}
			}
		}
		add(ss.Addresses, true)
		add(ss.NotReadyAddresses, false)
		info.Subsets = append(info.Subsets, sub)
	}

	info.Endpoints = formatEndpointList(flat)
	return info
}

func hostPort(ip string, port int32) string {
	return net.JoinHostPort(ip, strconv.Itoa(int(port)))
}

// formatEndpointList: "ip:port, ip:port, ip:port +N"
func formatEndpointList(flat []string) string {
	switch {
	case len(flat) == 0:
		return "<none>"
	case len(flat) > 3:
		return fmt.Sprintf("%s +%d", strings.Join(flat[:3], ", "), len(flat)-3)
	default:
		return strings.Join(flat, ", ")
	}
}

func (m *Manager) GetEndpoints(namespace string) ([]EndpointListInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	list, err := conn.Client.CoreV1().Endpoints(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	items := make([]EndpointListInfo, 0, len(list.Items))
	for _, ep := range list.Items {
		items = append(items, convertEndpoints(ep))
	}
	return items, nil
}
