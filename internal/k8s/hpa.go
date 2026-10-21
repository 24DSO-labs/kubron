package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HPAMetric — одна метрика автоскейлера: что измеряется, текущее и целевое значение.
type HPAMetric struct {
	Type    string `json:"type"` // Resource, ContainerResource, Pods, Object, External
	Name    string `json:"name"`
	Current string `json:"current"`
	Target  string `json:"target"`
}

type HPAInfo struct {
	Name            string `json:"name"`
	Namespace       string `json:"namespace"`
	Reference       string `json:"reference"`
	MinReplicas     int32  `json:"minReplicas"`
	MaxReplicas     int32  `json:"maxReplicas"`
	CurrentReplicas int32  `json:"currentReplicas"`
	DesiredReplicas int32  `json:"desiredReplicas"`
	Age             string `json:"age"`

	ReferenceKind  string            `json:"referenceKind"`
	ReferenceName  string            `json:"referenceName"`
	Metrics        []HPAMetric       `json:"metrics"`
	MetricsSummary string            `json:"metricsSummary"` // "cpu: 15% / 50%, memory: 40Mi / 100Mi"
	Status         string            `json:"status"`         // типы условий со статусом True
	Conditions     []ConditionInfo   `json:"conditions"`
	LastScaleTime  string            `json:"lastScaleTime"`
	Created        string            `json:"created"`
	Labels         map[string]string `json:"labels"`
	Annotations    map[string]string `json:"annotations"`
	Finalizers     []string          `json:"finalizers"`
}

func (m *Manager) GetHPA(namespace string) ([]HPAInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	list, err := conn.Client.AutoscalingV2().HorizontalPodAutoscalers(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	hpas := make([]HPAInfo, 0, len(list.Items))
	for _, hpa := range list.Items {
		hpas = append(hpas, convertHPA(hpa))
	}

	return hpas, nil
}

func convertHPA(hpa autoscalingv2.HorizontalPodAutoscaler) HPAInfo {
	ref := ""
	if hpa.Spec.ScaleTargetRef.Kind != "" {
		ref = hpa.Spec.ScaleTargetRef.Kind + "/" + hpa.Spec.ScaleTargetRef.Name
	}

	minReplicas := int32(1)
	if hpa.Spec.MinReplicas != nil {
		minReplicas = *hpa.Spec.MinReplicas
	}

	metrics := make([]HPAMetric, 0, len(hpa.Spec.Metrics))
	summary := make([]string, 0, len(hpa.Spec.Metrics))
	for _, spec := range hpa.Spec.Metrics {
		mt := hpaMetric(spec, hpa.Status.CurrentMetrics)
		metrics = append(metrics, mt)
		summary = append(summary, fmt.Sprintf("%s: %s / %s", mt.Name, mt.Current, mt.Target))
	}

	conditions := make([]ConditionInfo, 0, len(hpa.Status.Conditions))
	var active []string
	for _, c := range hpa.Status.Conditions {
		conditions = append(conditions, ConditionInfo{
			Type:               string(c.Type),
			Status:             string(c.Status),
			LastTransitionTime: formatTime(c.LastTransitionTime),
			Reason:             c.Reason,
			Message:            c.Message,
		})
		if c.Status == "True" {
			active = append(active, string(c.Type))
		}
	}

	lastScale := ""
	if hpa.Status.LastScaleTime != nil {
		lastScale = formatTime(*hpa.Status.LastScaleTime)
	}

	meta := metaOf(hpa.ObjectMeta)
	return HPAInfo{
		Name:            hpa.Name,
		Namespace:       hpa.Namespace,
		Reference:       ref,
		MinReplicas:     minReplicas,
		MaxReplicas:     hpa.Spec.MaxReplicas,
		CurrentReplicas: hpa.Status.CurrentReplicas,
		DesiredReplicas: hpa.Status.DesiredReplicas,
		Age:             meta.Age,
		ReferenceKind:   hpa.Spec.ScaleTargetRef.Kind,
		ReferenceName:   hpa.Spec.ScaleTargetRef.Name,
		Metrics:         metrics,
		MetricsSummary:  strings.Join(summary, ", "),
		Status:          strings.Join(active, ", "),
		Conditions:      conditions,
		LastScaleTime:   lastScale,
		Created:         meta.Created,
		Labels:          meta.Labels,
		Annotations:     meta.Annotations,
		Finalizers:      meta.Finalizers,
	}
}

// hpaMetric собирает название, цель и текущее значение метрики. Текущее значение
// ищется в status.currentMetrics по типу и имени; если контроллер его ещё не посчитал — "unknown".
func hpaMetric(spec autoscalingv2.MetricSpec, current []autoscalingv2.MetricStatus) HPAMetric {
	out := HPAMetric{Type: string(spec.Type), Current: "unknown", Target: "unknown"}

	for _, cur := range current {
		if cur.Type != spec.Type {
			continue
		}
		switch spec.Type {
		case autoscalingv2.ResourceMetricSourceType:
			if spec.Resource == nil || cur.Resource == nil || cur.Resource.Name != spec.Resource.Name {
				continue
			}
			out.Current = hpaCurrent(cur.Resource.Current, spec.Resource.Target)
		case autoscalingv2.ContainerResourceMetricSourceType:
			if spec.ContainerResource == nil || cur.ContainerResource == nil ||
				cur.ContainerResource.Name != spec.ContainerResource.Name ||
				cur.ContainerResource.Container != spec.ContainerResource.Container {
				continue
			}
			out.Current = hpaCurrent(cur.ContainerResource.Current, spec.ContainerResource.Target)
		case autoscalingv2.PodsMetricSourceType:
			if spec.Pods == nil || cur.Pods == nil || cur.Pods.Metric.Name != spec.Pods.Metric.Name {
				continue
			}
			out.Current = hpaCurrent(cur.Pods.Current, spec.Pods.Target)
		case autoscalingv2.ObjectMetricSourceType:
			if spec.Object == nil || cur.Object == nil ||
				cur.Object.Metric.Name != spec.Object.Metric.Name ||
				cur.Object.DescribedObject.Name != spec.Object.DescribedObject.Name {
				continue
			}
			out.Current = hpaCurrent(cur.Object.Current, spec.Object.Target)
		case autoscalingv2.ExternalMetricSourceType:
			if spec.External == nil || cur.External == nil || cur.External.Metric.Name != spec.External.Metric.Name {
				continue
			}
			out.Current = hpaCurrent(cur.External.Current, spec.External.Target)
		}
		break
	}

	switch spec.Type {
	case autoscalingv2.ResourceMetricSourceType:
		if r := spec.Resource; r != nil {
			out.Name, out.Target = string(r.Name), hpaTarget(r.Target)
		}
	case autoscalingv2.ContainerResourceMetricSourceType:
		if r := spec.ContainerResource; r != nil {
			out.Name, out.Target = fmt.Sprintf("%s (container %s)", r.Name, r.Container), hpaTarget(r.Target)
		}
	case autoscalingv2.PodsMetricSourceType:
		if r := spec.Pods; r != nil {
			out.Name, out.Target = r.Metric.Name, hpaTarget(r.Target)
		}
	case autoscalingv2.ObjectMetricSourceType:
		if r := spec.Object; r != nil {
			out.Name = fmt.Sprintf("%s on %s/%s", r.Metric.Name, r.DescribedObject.Kind, r.DescribedObject.Name)
			out.Target = hpaTarget(r.Target)
		}
	case autoscalingv2.ExternalMetricSourceType:
		if r := spec.External; r != nil {
			out.Name, out.Target = r.Metric.Name, hpaTarget(r.Target)
		}
	}
	return out
}

func hpaTarget(t autoscalingv2.MetricTarget) string {
	switch {
	case t.AverageUtilization != nil:
		return fmt.Sprintf("%d%%", *t.AverageUtilization)
	case t.AverageValue != nil:
		return t.AverageValue.String()
	case t.Value != nil:
		return t.Value.String()
	}
	return "unknown"
}

// hpaCurrent показывает значение в той же единице, что и цель (проценты для Utilization).
func hpaCurrent(v autoscalingv2.MetricValueStatus, target autoscalingv2.MetricTarget) string {
	quantity := func(q *resource.Quantity) string {
		if q == nil {
			return ""
		}
		return q.String()
	}
	switch {
	case target.AverageUtilization != nil && v.AverageUtilization != nil:
		return fmt.Sprintf("%d%%", *v.AverageUtilization)
	case target.AverageValue != nil && v.AverageValue != nil:
		return quantity(v.AverageValue)
	case target.Value != nil && v.Value != nil:
		return quantity(v.Value)
	case v.AverageUtilization != nil:
		return fmt.Sprintf("%d%%", *v.AverageUtilization)
	case v.AverageValue != nil:
		return quantity(v.AverageValue)
	case v.Value != nil:
		return quantity(v.Value)
	}
	return "unknown"
}
