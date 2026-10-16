package k8s

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NetworkPolicyInfo struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	PodSelector string `json:"podSelector"`
	Age         string `json:"age"`
}

func (m *Manager) GetNetworkPolicies(namespace string) ([]NetworkPolicyInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	networkPoliciesList, err := conn.Client.NetworkingV1().NetworkPolicies(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var networkPolicies []NetworkPolicyInfo
	for _, np := range networkPoliciesList.Items {
		podSelector := "all"
		if np.Spec.PodSelector.MatchLabels != nil && len(np.Spec.PodSelector.MatchLabels) > 0 {
			podSelector = "specific"
		}

		networkPolicies = append(networkPolicies, NetworkPolicyInfo{
			Name:        np.Name,
			Namespace:   np.Namespace,
			PodSelector: podSelector,
			Age:         formatAge(np.CreationTimestamp.Time),
		})
	}

	return networkPolicies, nil
}

func (m *Manager) DeleteNetworkPolicy(namespace, name string) error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	return conn.Client.NetworkingV1().NetworkPolicies(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}
