package services

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (m *MetricsInstaller) GetStatus(ctx context.Context) (*MetricsStatus, error) {
	status := &MetricsStatus{
		Status:     "not_installed",
		Components: make(map[string]string),
	}
	// Check namespace exists
	_, err := m.clientset.CoreV1().Namespaces().Get(ctx, MetricsNamespace, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return status, nil
		}
		return nil, fmt.Errorf("failed to check namespace: %w", err)
	}

	// Check Prometheus StatefulSet
	promSts, err := m.clientset.AppsV1().StatefulSets(MetricsNamespace).Get(ctx, "prometheus", metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			status.Components["prometheus"] = "not_found"
		} else {
			status.Components["prometheus"] = "error"
		}
	} else {
		if promSts.Status.ReadyReplicas > 0 {
			status.Components["prometheus"] = "running"
		} else {
			status.Components["prometheus"] = "pending"
		}
	}

	// Check Node Exporter DaemonSet
	neDaemon, err := m.clientset.AppsV1().DaemonSets(MetricsNamespace).Get(ctx, "node-exporter", metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			status.Components["node-exporter"] = "not_found"
		} else {
			status.Components["node-exporter"] = "error"
		}
	} else {
		if neDaemon.Status.NumberReady > 0 {
			status.Components["node-exporter"] = "running"
		} else {
			status.Components["node-exporter"] = "pending"
		}
	}

	// Check Kube State Metrics Deployment
	ksmDeploy, err := m.clientset.AppsV1().Deployments(MetricsNamespace).Get(ctx, "kube-state-metrics", metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			status.Components["kube-state-metrics"] = "not_found"
		} else {
			status.Components["kube-state-metrics"] = "error"
		}
	} else {
		if ksmDeploy.Status.ReadyReplicas > 0 {
			status.Components["kube-state-metrics"] = "running"
		} else {
			status.Components["kube-state-metrics"] = "pending"
		}
	}

	// Determine overall status
	allRunning := true
	anyFound := false
	for _, cs := range status.Components {
		if cs != "not_found" {
			anyFound = true
		}
		if cs != "running" {
			allRunning = false
		}
	}

	if allRunning && anyFound {
		status.Status = "installed"
		status.PrometheusURL = PrometheusURL
	} else if anyFound {
		status.Status = "degraded"
	}

	return status, nil
}
