package k8s

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (m *Manager) GetMetricsStatus() (*MetricsStackStatus, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	status := &MetricsStackStatus{
		Status:     "not_installed",
		Components: make(map[string]string),
	}

	// Check namespace
	_, err = conn.Client.CoreV1().Namespaces().Get(ctx, MetricsNamespace, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return status, nil
		}
		return nil, fmt.Errorf("failed to check namespace: %w", err)
	}

	// Check Prometheus StatefulSet
	promSts, err := conn.Client.AppsV1().StatefulSets(MetricsNamespace).Get(ctx, "prometheus", metav1.GetOptions{})
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
	neDaemon, err := conn.Client.AppsV1().DaemonSets(MetricsNamespace).Get(ctx, "node-exporter", metav1.GetOptions{})
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
	ksmDeploy, err := conn.Client.AppsV1().Deployments(MetricsNamespace).Get(ctx, "kube-state-metrics", metav1.GetOptions{})
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
		status.PrometheusURL = PrometheusSvcURL
	} else if anyFound {
		status.Status = "degraded"
	}

	return status, nil
}

func (m *Manager) InstallMetricsStack() error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}
	data, err := loadManifestData()
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	objects, err := parseManifest(data)
	if err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	for _, obj := range objects {
		if err := m.applyObject(ctx, conn, obj); err != nil {
			return fmt.Errorf("failed to apply %s %s: %w", obj.GetKind(), obj.GetName(), err)
		}
	}

	return nil
}

func (m *Manager) UninstallMetricsStack() error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}
	data, err := loadManifestData()
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	objects, err := parseManifest(data)
	if err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Delete in reverse order, skip namespace
	for i := len(objects) - 1; i >= 0; i-- {
		obj := objects[i]
		if obj.GetKind() == "Namespace" {
			continue
		}
		if err := m.deleteObject(ctx, conn, obj); err != nil {
			if !errors.IsNotFound(err) {
				return fmt.Errorf("failed to delete %s %s: %w", obj.GetKind(), obj.GetName(), err)
			}
		}
	}

	// Delete namespace last
	err = conn.Client.CoreV1().Namespaces().Delete(ctx, MetricsNamespace, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete namespace: %w", err)
	}

	return nil
}
