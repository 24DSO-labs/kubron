package k8s

import (
	"context"
	"time"

	nodesv1 "k8s.io/api/node/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RuntimeClassInfo struct {
	Name    string `json:"name"`
	Handler string `json:"handler"`
	Age     string `json:"age"`
}

func (m *Manager) GetRuntimeClasses() ([]RuntimeClassInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	list, err := conn.Client.NodeV1().RuntimeClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var rcs []RuntimeClassInfo
	for _, rc := range list.Items {
		rcs = append(rcs, convertRuntimeClass(rc))
	}

	return rcs, nil
}

func convertRuntimeClass(rc nodesv1.RuntimeClass) RuntimeClassInfo {
	return RuntimeClassInfo{
		Name:    rc.Name,
		Handler: rc.Handler,
		Age:     formatAge(rc.CreationTimestamp.Time),
	}
}

func (m *Manager) DeleteRuntimeClass(name string) error {
conn, err := m.GetActiveConnection()
if err != nil {
return err
}

ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
defer cancel()

return conn.Client.NodeV1().RuntimeClasses().Delete(ctx, name, metav1.DeleteOptions{})
}
