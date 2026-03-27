package k8s

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DeploymentInfo struct {
	Name            string            `json:"name"`
	Namespace       string            `json:"namespace"`
	Replicas        int32             `json:"replicas"`
	ReadyReplicas   int32             `json:"readyReplicas"`
	UpdatedReplicas int32             `json:"updatedReplicas"`
	Age             string            `json:"age"`
	Labels          map[string]string `json:"labels"`
	Strategy        string            `json:"strategy"`
	Images          []string          `json:"images"`
}

func (m *Manager) GetDeployments(namespace string) ([]DeploymentInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	depList, err := conn.Client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var deployments []DeploymentInfo
	for _, dep := range depList.Items {
		var images []string
		for _, c := range dep.Spec.Template.Spec.Containers {
			images = append(images, c.Image)
		}

		var replicas int32 = 0
		if dep.Spec.Replicas != nil {
			replicas = *dep.Spec.Replicas
		}

		deployments = append(deployments, DeploymentInfo{
			Name:            dep.Name,
			Namespace:       dep.Namespace,
			Replicas:        replicas,
			ReadyReplicas:   dep.Status.ReadyReplicas,
			UpdatedReplicas: dep.Status.UpdatedReplicas,
			Age:             formatAge(dep.CreationTimestamp.Time),
			Labels:          dep.Labels,
			Strategy:        string(dep.Spec.Strategy.Type),
			Images:          images,
		})
	}

	return deployments, nil
}

func (m *Manager) ScaleDeployment(namespace, name string, replicas int32) error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	scale, err := conn.Client.AppsV1().Deployments(namespace).GetScale(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("не удалось получить scale: %w", err)
	}

	scale.Spec.Replicas = replicas

	_, err = conn.Client.AppsV1().Deployments(namespace).UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("не удалось масштабировать: %w", err)
	}

	return nil
}

func (m *Manager) DeleteDeployment(namespace, name string) error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deletePolicy := metav1.DeletePropagationForeground
	deleteOptions := metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}

	err = conn.Client.AppsV1().Deployments(namespace).Delete(ctx, name, deleteOptions)
	if err != nil {
		return fmt.Errorf("не удалось удалить deployment: %w", err)
	}

	return nil
}
