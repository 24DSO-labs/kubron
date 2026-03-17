package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// === StatefulSets ===

type StatefulSetInfo struct {
	Name          string   `json:"name"`
	Namespace     string   `json:"namespace"`
	Replicas      int32    `json:"replicas"`
	ReadyReplicas int32    `json:"readyReplicas"`
	Age           string   `json:"age"`
	Images        []string `json:"images"`
	ServiceName   string   `json:"serviceName"`
}

func (m *Manager) GetStatefulSets(namespace string) ([]StatefulSetInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	list, err := conn.Client.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var items []StatefulSetInfo
	for _, s := range list.Items {
		var images []string
		for _, c := range s.Spec.Template.Spec.Containers {
			images = append(images, c.Image)
		}
		var replicas int32 = 1
		if s.Spec.Replicas != nil {
			replicas = *s.Spec.Replicas
		}
		items = append(items, StatefulSetInfo{
			Name:          s.Name,
			Namespace:     s.Namespace,
			Replicas:      replicas,
			ReadyReplicas: s.Status.ReadyReplicas,
			Age:           formatAge(s.CreationTimestamp.Time),
			Images:        images,
			ServiceName:   s.Spec.ServiceName,
		})
	}
	return items, nil
}

// === DaemonSets ===

type DaemonSetInfo struct {
	Name         string   `json:"name"`
	Namespace    string   `json:"namespace"`
	Desired      int32    `json:"desired"`
	Current      int32    `json:"current"`
	Ready        int32    `json:"ready"`
	Available    int32    `json:"available"`
	Age          string   `json:"age"`
	Images       []string `json:"images"`
	NodeSelector string   `json:"nodeSelector"`
}

func (m *Manager) GetDaemonSets(namespace string) ([]DaemonSetInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	list, err := conn.Client.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var items []DaemonSetInfo
	for _, d := range list.Items {
		var images []string
		for _, c := range d.Spec.Template.Spec.Containers {
			images = append(images, c.Image)
		}
		ns := ""
		if d.Spec.Template.Spec.NodeSelector != nil {
			var parts []string
			for k, v := range d.Spec.Template.Spec.NodeSelector {
				parts = append(parts, fmt.Sprintf("%s=%s", k, v))
			}
			ns = strings.Join(parts, ", ")
		}
		items = append(items, DaemonSetInfo{
			Name:         d.Name,
			Namespace:    d.Namespace,
			Desired:      d.Status.DesiredNumberScheduled,
			Current:      d.Status.CurrentNumberScheduled,
			Ready:        d.Status.NumberReady,
			Available:    d.Status.NumberAvailable,
			Age:          formatAge(d.CreationTimestamp.Time),
			Images:       images,
			NodeSelector: ns,
		})
	}
	return items, nil
}

// === Jobs ===

type JobInfo struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	Completions string `json:"completions"`
	Succeeded   int32  `json:"succeeded"`
	Failed      int32  `json:"failed"`
	Active      int32  `json:"active"`
	Duration    string `json:"duration"`
	Age         string `json:"age"`
	Status      string `json:"status"`
}

func (m *Manager) GetJobs(namespace string) ([]JobInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	list, err := conn.Client.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var items []JobInfo
	for _, j := range list.Items {
		completions := "1"
		if j.Spec.Completions != nil {
			completions = fmt.Sprintf("%d", *j.Spec.Completions)
		}

		status := "Running"
		if j.Status.Succeeded > 0 {
			status = "Complete"
		} else if j.Status.Failed > 0 {
			status = "Failed"
		}

		duration := ""
		if j.Status.StartTime != nil {
			end := time.Now()
			if j.Status.CompletionTime != nil {
				end = j.Status.CompletionTime.Time
			}
			duration = formatAge(end.Add(-end.Sub(j.Status.StartTime.Time)))
		}

		items = append(items, JobInfo{
			Name:        j.Name,
			Namespace:   j.Namespace,
			Completions: fmt.Sprintf("%d/%s", j.Status.Succeeded, completions),
			Succeeded:   j.Status.Succeeded,
			Failed:      j.Status.Failed,
			Active:      j.Status.Active,
			Duration:    duration,
			Age:         formatAge(j.CreationTimestamp.Time),
			Status:      status,
		})
	}
	return items, nil
}

// === CronJobs ===

type CronJobInfo struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Schedule     string `json:"schedule"`
	Suspend      bool   `json:"suspend"`
	Active       int    `json:"active"`
	LastSchedule string `json:"lastSchedule"`
	Age          string `json:"age"`
}

func (m *Manager) GetCronJobs(namespace string) ([]CronJobInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	list, err := conn.Client.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var items []CronJobInfo
	for _, cj := range list.Items {
		lastSchedule := "—"
		if cj.Status.LastScheduleTime != nil {
			lastSchedule = formatAge(cj.Status.LastScheduleTime.Time)
		}
		suspend := false
		if cj.Spec.Suspend != nil {
			suspend = *cj.Spec.Suspend
		}
		items = append(items, CronJobInfo{
			Name:         cj.Name,
			Namespace:    cj.Namespace,
			Schedule:     cj.Spec.Schedule,
			Suspend:      suspend,
			Active:       len(cj.Status.Active),
			LastSchedule: lastSchedule,
			Age:          formatAge(cj.CreationTimestamp.Time),
		})
	}
	return items, nil
}
