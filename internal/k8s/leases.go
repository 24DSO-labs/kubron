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

	LeaseDurationSeconds int32             `json:"leaseDurationSeconds"`
	AcquireTime          string            `json:"acquireTime"`
	RenewTime            string            `json:"renewTime"`
	LeaseTransitions     int32             `json:"leaseTransitions"`
	PreferredHolder      string            `json:"preferredHolder"`
	Strategy             string            `json:"strategy"`
	Created              string            `json:"created"`
	Labels               map[string]string `json:"labels"`
	ControlledBy         string            `json:"controlledBy"`
	ManagedBy            string            `json:"managedBy"`
	Annotations          map[string]string `json:"annotations"`
	Finalizers           []string          `json:"finalizers"`
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

	leases := make([]LeaseInfo, 0, len(list.Items))
	for _, lease := range list.Items {
		leases = append(leases, convertLease(lease))
	}

	return leases, nil
}

func convertLease(lease coordinationv1.Lease) LeaseInfo {
	spec := lease.Spec
	strategy := ""
	if spec.Strategy != nil {
		strategy = string(*spec.Strategy)
	}

	meta := metaOf(lease.ObjectMeta)
	return LeaseInfo{
		Name:                 lease.Name,
		Namespace:            lease.Namespace,
		Holder:               derefStr(spec.HolderIdentity),
		Age:                  meta.Age,
		LeaseDurationSeconds: derefInt32(spec.LeaseDurationSeconds),
		AcquireTime:          formatMicroTime(spec.AcquireTime),
		RenewTime:            formatMicroTime(spec.RenewTime),
		LeaseTransitions:     derefInt32(spec.LeaseTransitions),
		PreferredHolder:      derefStr(spec.PreferredHolder),
		Strategy:             strategy,
		Created:              meta.Created,
		Labels:               meta.Labels,
		ControlledBy:         controlledByOf(lease.OwnerReferences),
		ManagedBy:            managedByOf(lease.ObjectMeta),
		Annotations:          meta.Annotations,
		Finalizers:           meta.Finalizers,
	}
}
