package k8s

import (
	"context"
	"time"

	schedulingv1 "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PriorityClassInfo struct {
	Name  string `json:"name"`
	Value int32  `json:"value"`
	Age   string `json:"age"`
}

func (m *Manager) GetPriorityClasses() ([]PriorityClassInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	list, err := conn.Client.SchedulingV1().PriorityClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var pcs []PriorityClassInfo
	for _, pc := range list.Items {
		pcs = append(pcs, convertPriorityClass(pc))
	}

	return pcs, nil
}

func convertPriorityClass(pc schedulingv1.PriorityClass) PriorityClassInfo {
	return PriorityClassInfo{
		Name:  pc.Name,
		Value: pc.Value,
		Age:   formatAge(pc.CreationTimestamp.Time),
	}
}

func (m *Manager) DeletePriorityClass(name string) error {
conn, err := m.GetActiveConnection()
if err != nil {
return err
}

ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
defer cancel()

return conn.Client.SchedulingV1().PriorityClasses().Delete(ctx, name, metav1.DeleteOptions{})
}
