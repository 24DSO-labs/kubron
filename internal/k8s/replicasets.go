package k8s

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ReplicaSetInfo struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Desired   int32    `json:"desired"`
	Ready     int32    `json:"ready"`
	Age       string   `json:"age"`
	Images    []string `json:"images"`
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
		replicaSets = append(replicaSets, ReplicaSetInfo{
			Name:      rs.Name,
			Namespace: rs.Namespace,
			Desired:   desired,
			Ready:     rs.Status.ReadyReplicas,
			Age:       formatAge(rs.CreationTimestamp.Time),
			Images:    images,
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
