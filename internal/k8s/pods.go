package k8s

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PodInfo struct {
	Name       string            `json:"name"`
	Namespace  string            `json:"namespace"`
	Status     string            `json:"status"`
	Ready      string            `json:"ready"`
	Restarts   int32             `json:"restarts"`
	Age        string            `json:"age"`
	Node       string            `json:"node"`
	IP         string            `json:"ip"`
	Labels     map[string]string `json:"labels"`
	Containers []ContainerInfo   `json:"containers"`
	QOS        string            `json:"qos"`
	Created    string            `json:"created"`
}

type ContainerInfo struct {
	Name         string `json:"name"`
	Image        string `json:"image"`
	Ready        bool   `json:"ready"`
	RestartCount int32  `json:"restartCount"`
	State        string `json:"state"`
	IsInit       bool   `json:"isInit"`
}

func (m *Manager) GetPods(namespace string) ([]PodInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	podList, err := conn.Client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var pods []PodInfo
	for _, pod := range podList.Items {
		pods = append(pods, convertPod(pod))
	}

	return pods, nil
}

func convertPod(pod corev1.Pod) PodInfo {
	var totalRestarts int32
	var readyCount int
	var containers []ContainerInfo

	// Process init containers
	for _, ics := range pod.Status.InitContainerStatuses {
		state := "unknown"
		if ics.State.Running != nil {
			state = "running"
		} else if ics.State.Waiting != nil {
			state = ics.State.Waiting.Reason
		} else if ics.State.Terminated != nil {
			state = ics.State.Terminated.Reason
		}

		containers = append(containers, ContainerInfo{
			Name:         ics.Name,
			Image:        ics.Image,
			Ready:        ics.Ready,
			RestartCount: ics.RestartCount,
			State:        state,
			IsInit:       true,
		})
	}

	// Process regular containers
	for _, cs := range pod.Status.ContainerStatuses {
		totalRestarts += cs.RestartCount
		if cs.Ready {
			readyCount++
		}

		state := "unknown"
		if cs.State.Running != nil {
			state = "running"
		} else if cs.State.Waiting != nil {
			state = cs.State.Waiting.Reason
		} else if cs.State.Terminated != nil {
			state = cs.State.Terminated.Reason
		}

		containers = append(containers, ContainerInfo{
			Name:         cs.Name,
			Image:        cs.Image,
			Ready:        cs.Ready,
			RestartCount: cs.RestartCount,
			State:        state,
			IsInit:       false,
		})
	}

	qos := string(pod.Status.QOSClass)
	created := pod.CreationTimestamp.Format(time.RFC3339)

	status := string(pod.Status.Phase)
	// Более точный статус
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
			status = cs.State.Waiting.Reason
			break
		}
		if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" {
			status = cs.State.Terminated.Reason
			break
		}
	}

	return PodInfo{
		Name:       pod.Name,
		Namespace:  pod.Namespace,
		Status:     status,
		Ready:      fmt.Sprintf("%d/%d", readyCount, len(pod.Spec.Containers)),
		Restarts:   totalRestarts,
		Age:        formatAge(pod.CreationTimestamp.Time),
		Node:       pod.Spec.NodeName,
		IP:         pod.Status.PodIP,
		Labels:     pod.Labels,
		Containers: containers,
		QOS:        qos,
		Created:    created,
	}
}
