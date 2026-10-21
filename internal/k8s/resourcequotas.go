package k8s

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// QuotaEntry — одна строка квоты: ресурс, использовано / лимит и доля в процентах.
type QuotaEntry struct {
	Name    string `json:"name"`
	Hard    string `json:"hard"`
	Used    string `json:"used"`
	Percent int    `json:"percent"` // 0..100+, -1 если посчитать нельзя
}

type ResourceQuotaInfo struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Age       string `json:"age"`
	CPULimit  string `json:"cpuLimit"`
	CPUUsed   string `json:"cpuUsed"`
	MemLimit  string `json:"memLimit"`
	MemUsed   string `json:"memUsed"`

	Created      string            `json:"created"`
	Labels       map[string]string `json:"labels"`
	ControlledBy string            `json:"controlledBy"`
	ManagedBy    string            `json:"managedBy"`
	Annotations  map[string]string `json:"annotations"`
	Finalizers   []string          `json:"finalizers"`
	Quotas       []QuotaEntry      `json:"quotas"`
	Scopes       []string          `json:"scopes"`
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

	quotas := make([]ResourceQuotaInfo, 0, len(list.Items))
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

	// status.hard появляется после первой синхронизации контроллера; до неё берём spec.hard
	hard := q.Status.Hard
	if len(hard) == 0 {
		hard = q.Spec.Hard
	}
	entries := make([]QuotaEntry, 0, len(hard))
	for _, name := range sortedKeys(hard) {
		limit := hard[v1.ResourceName(name)]
		entry := QuotaEntry{Name: name, Hard: limit.String(), Used: "0", Percent: -1}
		if used, ok := q.Status.Used[v1.ResourceName(name)]; ok {
			entry.Used = used.String()
			if limit.Value() > 0 || limit.MilliValue() > 0 {
				entry.Percent = int(used.AsApproximateFloat64() / limit.AsApproximateFloat64() * 100)
			}
		} else if limit.Value() > 0 || limit.MilliValue() > 0 {
			entry.Percent = 0
		}
		entries = append(entries, entry)
	}

	var scopes []string
	for _, s := range q.Spec.Scopes {
		scopes = append(scopes, string(s))
	}
	if q.Spec.ScopeSelector != nil {
		for _, e := range q.Spec.ScopeSelector.MatchExpressions {
			text := fmt.Sprintf("%s %s", e.ScopeName, e.Operator)
			if len(e.Values) > 0 {
				text += fmt.Sprintf(" (%v)", e.Values)
			}
			scopes = append(scopes, text)
		}
	}

	meta := metaOf(q.ObjectMeta)
	return ResourceQuotaInfo{
		Name:         q.Name,
		Namespace:    q.Namespace,
		Age:          meta.Age,
		CPULimit:     cpuLimit,
		CPUUsed:      cpuUsed,
		MemLimit:     memLimit,
		MemUsed:      memUsed,
		Created:      meta.Created,
		Labels:       meta.Labels,
		ControlledBy: controlledByOf(q.OwnerReferences),
		ManagedBy:    managedByOf(q.ObjectMeta),
		Annotations:  meta.Annotations,
		Finalizers:   meta.Finalizers,
		Quotas:       entries,
		Scopes:       scopes,
	}
}
