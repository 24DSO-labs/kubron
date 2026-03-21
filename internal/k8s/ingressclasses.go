package k8s

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type IngressClassInfo struct {
	Name       string `json:"name"`
	Controller string `json:"controller"`
	Age        string `json:"age"`
}

func (m *Manager) GetIngressClasses() ([]IngressClassInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	list, err := conn.Client.NetworkingV1().IngressClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var items []IngressClassInfo
	for _, ic := range list.Items {
		controller := ""
		if ic.Spec.Controller != "" {
			controller = ic.Spec.Controller
		}

		items = append(items, IngressClassInfo{
			Name:       ic.Name,
			Controller: controller,
			Age:        formatAge(ic.CreationTimestamp.Time),
		})
	}
	return items, nil
}
