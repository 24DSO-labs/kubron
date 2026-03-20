package k8s

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EndpointListInfo struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Addresses int    `json:"addresses"`
	Ports     int    `json:"ports"`
	Age       string `json:"age"`
}

func (m *Manager) GetEndpoints(namespace string) ([]EndpointListInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	endpointsList, err := conn.Client.CoreV1().Endpoints(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var endpoints []EndpointListInfo
	for _, ep := range endpointsList.Items {
		addressCount := 0
		portCount := 0

		for _, subset := range ep.Subsets {
			addressCount += len(subset.Addresses)
			portCount += len(subset.Ports)
		}

		endpoints = append(endpoints, EndpointListInfo{
			Name:      ep.Name,
			Namespace: ep.Namespace,
			Addresses: addressCount,
			Ports:     portCount,
			Age:       formatAge(ep.CreationTimestamp.Time),
		})
	}

	return endpoints, nil
}

func (m *Manager) GetEndpoint(namespace, name string) (*corev1.Endpoints, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return conn.Client.CoreV1().Endpoints(namespace).Get(ctx, name, metav1.GetOptions{})
}
