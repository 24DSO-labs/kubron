package k8s

import (
	"context"
	"fmt"
	"sort"
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

	type timedEvent struct {
		info EventInfo
		at   time.Time
	}
	timed := make([]timedEvent, 0, len(eventList.Items))
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

		// Время события: API не гарантирует порядок списка, сортируем сами.
		at := ev.LastTimestamp.Time
		if at.IsZero() {
			at = ev.EventTime.Time
		}
		if at.IsZero() {
			at = ev.CreationTimestamp.Time
		}

		timed = append(timed, timedEvent{at: at, info: EventInfo{
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
		}})
	}

	// Новые сверху
	sort.SliceStable(timed, func(i, j int) bool { return timed[i].at.After(timed[j].at) })

	events := make([]EventInfo, len(timed))
	for i, t := range timed {
		events[i] = t.info
	}
	return events, nil
}
