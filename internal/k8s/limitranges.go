package k8s

import (
	"context"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type LimitRangeInfo struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Age       string `json:"age"`
	Items     int    `json:"items"`
}

func (m *Manager) GetLimitRanges(namespace string) ([]LimitRangeInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	list, err := conn.Client.CoreV1().LimitRanges(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var limits []LimitRangeInfo
	for _, lr := range list.Items {
		limits = append(limits, convertLimitRange(lr))
	}

	return limits, nil
}

func convertLimitRange(lr v1.LimitRange) LimitRangeInfo {
	return LimitRangeInfo{
		Name:      lr.Name,
		Namespace: lr.Namespace,
		Age:       formatAge(lr.CreationTimestamp.Time),
		Items:     len(lr.Spec.Limits),
	}
}

func (m *Manager) DeleteLimitRange(namespace, name string) error {
conn, err := m.GetActiveConnection()
if err != nil {
return err
}

ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
defer cancel()

return conn.Client.CoreV1().LimitRanges(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}
