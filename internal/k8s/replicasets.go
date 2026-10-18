package k8s

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ReplicaSetInfo struct {
	Name         string            `json:"name"`
	Namespace    string            `json:"namespace"`
	Desired      int32             `json:"desired"`
	Current      int32             `json:"current"`
	Ready        int32             `json:"ready"`
	Available    int32             `json:"available"`
	Age          string            `json:"age"`
	Created      string            `json:"created"`
	Images       []string          `json:"images"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	Selector     string            `json:"selector"`
	ControlledBy string            `json:"controlledBy"`
	PodStatuses  map[string]int32  `json:"podStatuses"`
}

func (m *Manager) GetReplicaSets(namespace string) ([]ReplicaSetInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	replicaSetList, err := conn.Client.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Поды неймспейса: считаем статусы подов по фазам для каждого ReplicaSet (по controller ownerRef)
	podList, err := conn.Client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	podStatusesByRS := map[string]map[string]int32{}
	for _, pod := range podList.Items {
		for _, ref := range pod.OwnerReferences {
			if ref.Kind != "ReplicaSet" || ref.Controller == nil || !*ref.Controller {
				continue
			}
			if podStatusesByRS[ref.Name] == nil {
				podStatusesByRS[ref.Name] = map[string]int32{}
			}
			podStatusesByRS[ref.Name][string(pod.Status.Phase)]++
		}
	}

	var replicaSets []ReplicaSetInfo
	for _, rs := range replicaSetList.Items {
		desired := int32(1)
		if rs.Spec.Replicas != nil {
			desired = *rs.Spec.Replicas
		}
		var images []string
		for _, c := range rs.Spec.Template.Spec.Containers {
			images = append(images, c.Image)
		}

		// Selector из MatchLabels: "app=myapp,version=v1"
		selectorStr := ""
		if rs.Spec.Selector != nil && len(rs.Spec.Selector.MatchLabels) > 0 {
			first := true
			for k, v := range rs.Spec.Selector.MatchLabels {
				if !first {
					selectorStr += ", "
				}
				selectorStr += k + "=" + v
				first = false
			}
		}

		// Владелец (обычно Deployment): "Deployment/my-app"
		controlledBy := ""
		if len(rs.OwnerReferences) > 0 {
			ref := rs.OwnerReferences[0]
			controlledBy = fmt.Sprintf("%s/%s", ref.Kind, ref.Name)
		}

		replicaSets = append(replicaSets, ReplicaSetInfo{
			Name:         rs.Name,
			Namespace:    rs.Namespace,
			Desired:      desired,
			Current:      rs.Status.Replicas,
			Ready:        rs.Status.ReadyReplicas,
			Available:    rs.Status.AvailableReplicas,
			Age:          formatAge(rs.CreationTimestamp.Time),
			Created:      rs.CreationTimestamp.Format(time.RFC3339),
			Images:       images,
			Labels:       rs.Labels,
			Annotations:  rs.Annotations,
			Selector:     selectorStr,
			ControlledBy: controlledBy,
			PodStatuses:  podStatusesByRS[rs.Name],
		})
	}

	return replicaSets, nil
}

func (m *Manager) GetReplicaSet(namespace, name string) (*appsv1.ReplicaSet, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return conn.Client.AppsV1().ReplicaSets(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (m *Manager) DeleteReplicaSet(namespace, name string) error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return conn.Client.AppsV1().ReplicaSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}
