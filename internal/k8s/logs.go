package k8s

import (
	"context"
	"fmt"
	"io"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PodDetail struct {
	Name       string            `json:"name"`
	Namespace  string            `json:"namespace"`
	Status     string            `json:"status"`
	Node       string            `json:"node"`
	IP         string            `json:"ip"`
	Labels     map[string]string `json:"labels"`
	Age        string            `json:"age"`
	Containers []ContainerInfo   `json:"containers"`
}

func (m *Manager) GetPodDetail(namespace, name string) (*PodDetail, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pod, err := conn.Client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	p := convertPod(*pod)

	return &PodDetail{
		Name:       p.Name,
		Namespace:  p.Namespace,
		Status:     p.Status,
		Node:       p.Node,
		IP:         p.IP,
		Labels:     p.Labels,
		Age:        p.Age,
		Containers: p.Containers,
	}, nil
}

func (m *Manager) GetPodLogs(namespace, name, container string, lines int64) (string, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	opts := &corev1.PodLogOptions{
		TailLines:  &lines,
		Timestamps: true,
	}

	if container != "" {
		opts.Container = container
	}

	req := conn.Client.CoreV1().Pods(namespace).GetLogs(name, opts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("не удалось получить логи: %w", err)
	}
	defer stream.Close()

	data, err := io.ReadAll(stream)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения логов: %w", err)
	}

	return string(data), nil
}

func (m *Manager) DeletePod(namespace, name string) error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return conn.Client.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (m *Manager) AttachPod(namespace, name, container string) error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pod, err := conn.Client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("не удалось получить информацию о поде: %w", err)
	}

	containerFound := false
	var containerStatus *corev1.ContainerStatus
	for i, c := range pod.Spec.Containers {
		if c.Name == container {
			containerFound = true
			if i < len(pod.Status.ContainerStatuses) {
				containerStatus = &pod.Status.ContainerStatuses[i]
			}
			break
		}
	}

	if !containerFound {
		return fmt.Errorf("контейнер %s не найден в поде %s", container, name)
	}

	if pod.Status.Phase != corev1.PodRunning {
		return fmt.Errorf("под не запущен (статус: %s)", pod.Status.Phase)
	}

	if containerStatus != nil && !containerStatus.Ready {
		return fmt.Errorf("контейнер %s не готов (state: %v)", container, containerStatus.State)
	}

	return nil
}

func (m *Manager) EvictPod(namespace, name string) error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: int64Ptr(30),
	}

	return conn.Client.CoreV1().Pods(namespace).Delete(ctx, name, deleteOptions)
}

func int64Ptr(i int64) *int64 {
	return &i
}
