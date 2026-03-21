package k8s

import (
	"context"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type LeaseInfo struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Holder    string `json:"holder"`
	Age       string `json:"age"`
}

func (m *Manager) GetLeases(namespace string) ([]LeaseInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	list, err := conn.Client.CoordinationV1().Leases(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var leases []LeaseInfo
	for _, lease := range list.Items {
		leases = append(leases, convertLease(lease))
	}

	return leases, nil
}

func convertLease(lease coordinationv1.Lease) LeaseInfo {
	holder := ""
	if lease.Spec.HolderIdentity != nil {
		holder = *lease.Spec.HolderIdentity
	}

	return LeaseInfo{
		Name:      lease.Name,
		Namespace: lease.Namespace,
		Holder:    holder,
		Age:       formatAge(lease.CreationTimestamp.Time),
	}
}
