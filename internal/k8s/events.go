package k8s

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EventInfo struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Reason       string `json:"reason"`
	Message      string `json:"message"`
	Object       string `json:"object"`
	Count        int32  `json:"count"`
	Age          string `json:"age"`
	Source       string `json:"source"`
	Namespace    string `json:"namespace"`
	InvolvedKind string `json:"involvedKind"`
	InvolvedName string `json:"involvedName"`
}

func (m *Manager) GetEvents(namespace string) ([]EventInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	eventList, err := conn.Client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var events []EventInfo
	for _, ev := range eventList.Items {
		obj := fmt.Sprintf("%s/%s", ev.InvolvedObject.Kind, ev.InvolvedObject.Name)

		age := ""
		if !ev.LastTimestamp.IsZero() {
			age = formatAge(ev.LastTimestamp.Time)
		} else if !ev.EventTime.IsZero() {
			age = formatAge(ev.EventTime.Time)
		}

		source := ""
		if ev.Source.Component != "" {
			source = ev.Source.Component
		}
		if ev.ReportingController != "" {
			source = ev.ReportingController
		}

		events = append(events, EventInfo{
			Name:         ev.Name,
			Type:         ev.Type,
			Reason:       ev.Reason,
			Message:      ev.Message,
			Object:       obj,
			Count:        ev.Count,
			Age:          age,
			Source:       source,
			Namespace:    ev.Namespace,
			InvolvedKind: ev.InvolvedObject.Kind,
			InvolvedName: ev.InvolvedObject.Name,
		})
	}

	// Сортируем: новые сверху
	for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
		events[i], events[j] = events[j], events[i]
	}

	return events, nil
}
