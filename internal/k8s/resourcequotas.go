package k8s

import (
	"context"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ResourceQuotaInfo struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Age       string `json:"age"`
	CPULimit  string `json:"cpuLimit"`
	CPUUsed   string `json:"cpuUsed"`
	MemLimit  string `json:"memLimit"`
	MemUsed   string `json:"memUsed"`
}

func (m *Manager) GetResourceQuotas(namespace string) ([]ResourceQuotaInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	list, err := conn.Client.CoreV1().ResourceQuotas(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var quotas []ResourceQuotaInfo
	for _, quota := range list.Items {
		quotas = append(quotas, convertResourceQuota(quota))
	}

	return quotas, nil
}

func convertResourceQuota(q v1.ResourceQuota) ResourceQuotaInfo {
	var cpuLimit, cpuUsed, memLimit, memUsed string
	if !q.Status.Hard.Cpu().IsZero() {
		cpuLimit = q.Status.Hard.Cpu().String()
	}
	if !q.Status.Used.Cpu().IsZero() {
		cpuUsed = q.Status.Used.Cpu().String()
	}
	if !q.Status.Hard.Memory().IsZero() {
		memLimit = q.Status.Hard.Memory().String()
	}
	if !q.Status.Used.Memory().IsZero() {
		memUsed = q.Status.Used.Memory().String()
	}

	return ResourceQuotaInfo{
		Name:      q.Name,
		Namespace: q.Namespace,
		Age:       formatAge(q.CreationTimestamp.Time),
		CPULimit:  cpuLimit,
		CPUUsed:   cpuUsed,
		MemLimit:  memLimit,
		MemUsed:   memUsed,
	}
}
