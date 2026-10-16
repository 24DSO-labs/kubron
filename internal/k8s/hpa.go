package k8s

import (
	"context"
	"time"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type HPAInfo struct {
	Name            string `json:"name"`
	Namespace       string `json:"namespace"`
	Reference       string `json:"reference"`
	MinReplicas     int32  `json:"minReplicas"`
	MaxReplicas     int32  `json:"maxReplicas"`
	CurrentReplicas int32  `json:"currentReplicas"`
	DesiredReplicas int32  `json:"desiredReplicas"`
	Age             string `json:"age"`
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

	var hpas []HPAInfo
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

	return HPAInfo{
		Name:            hpa.Name,
		Namespace:       hpa.Namespace,
		Reference:       ref,
		MinReplicas:     minReplicas,
		MaxReplicas:     hpa.Spec.MaxReplicas,
		CurrentReplicas: hpa.Status.CurrentReplicas,
		DesiredReplicas: hpa.Status.DesiredReplicas,
		Age:             formatAge(hpa.CreationTimestamp.Time),
	}
}

func (m *Manager) DeleteHPA(namespace, name string) error {
conn, err := m.GetActiveConnection()
if err != nil {
return err
}

ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
defer cancel()

return conn.Client.AutoscalingV2().HorizontalPodAutoscalers(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}
