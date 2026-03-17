package k8s

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterOverview struct {
	TotalNodes       int `json:"totalNodes"`
	ReadyNodes       int `json:"readyNodes"`
	TotalPods        int `json:"totalPods"`
	RunningPods      int `json:"runningPods"`
	PendingPods      int `json:"pendingPods"`
	FailedPods       int `json:"failedPods"`
	TotalDeployments int `json:"totalDeployments"`
	ReadyDeployments int `json:"readyDeployments"`
	TotalServices    int `json:"totalServices"`
	TotalNamespaces  int `json:"totalNamespaces"`
	TotalEvents      int `json:"totalEvents"`
	WarningEvents    int `json:"warningEvents"`
}

func (m *Manager) GetClusterOverview() (*ClusterOverview, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	overview := &ClusterOverview{}

	// Nodes
	nodes, err := conn.Client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err == nil {
		overview.TotalNodes = len(nodes.Items)
		for _, node := range nodes.Items {
			for _, cond := range node.Status.Conditions {
				if cond.Type == "Ready" && cond.Status == "True" {
					overview.ReadyNodes++
					break
				}
			}
		}
	}

	// Pods
	pods, err := conn.Client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err == nil {
		overview.TotalPods = len(pods.Items)
		for _, pod := range pods.Items {
			switch pod.Status.Phase {
			case "Running":
				overview.RunningPods++
			case "Pending":
				overview.PendingPods++
			case "Failed":
				overview.FailedPods++
			}
		}
	}

	// Deployments
	deps, err := conn.Client.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err == nil {
		overview.TotalDeployments = len(deps.Items)
		for _, dep := range deps.Items {
			replicas := int32(1)
			if dep.Spec.Replicas != nil {
				replicas = *dep.Spec.Replicas
			}
			if dep.Status.ReadyReplicas == replicas {
				overview.ReadyDeployments++
			}
		}
	}

	// Services
	svcs, err := conn.Client.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err == nil {
		overview.TotalServices = len(svcs.Items)
	}

	// Namespaces
	nsList, err := conn.Client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err == nil {
		overview.TotalNamespaces = len(nsList.Items)
	}

	// Events
	events, err := conn.Client.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	if err == nil {
		overview.TotalEvents = len(events.Items)
		for _, ev := range events.Items {
			if ev.Type == "Warning" {
				overview.WarningEvents++
			}
		}
	}

	return overview, nil
}
