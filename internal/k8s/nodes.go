package k8s

import (
	"context"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NodeInfo struct {
	Name           string `json:"name"`
	Status         string `json:"status"`
	Roles          string `json:"roles"`
	Version        string `json:"version"`
	OS             string `json:"os"`
	Arch           string `json:"arch"`
	CPUCapacity    string `json:"cpuCapacity"`
	MemoryCapacity string `json:"memoryCapacity"`
	InternalIP     string `json:"internalIP"`
	Age            string `json:"age"`
	PodCIDR        string `json:"podCIDR"`
}

func (m *Manager) GetNodes() ([]NodeInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	nodeList, err := conn.Client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var nodes []NodeInfo
	for _, node := range nodeList.Items {
		nodes = append(nodes, convertNode(node))
	}

	return nodes, nil
}

func convertNode(node corev1.Node) NodeInfo {
	// Статус
	status := "NotReady"
	for _, cond := range node.Status.Conditions {
		if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
			status = "Ready"
			break
		}
	}

	// Роли
	var roles []string
	for label := range node.Labels {
		if strings.HasPrefix(label, "node-role.kubernetes.io/") {
			role := strings.TrimPrefix(label, "node-role.kubernetes.io/")
			if role != "" {
				roles = append(roles, role)
			}
		}
	}
	rolesStr := strings.Join(roles, ", ")
	if rolesStr == "" {
		rolesStr = "<none>"
	}

	// IP
	internalIP := ""
	for _, addr := range node.Status.Addresses {
		if addr.Type == corev1.NodeInternalIP {
			internalIP = addr.Address
			break
		}
	}

	cpu := node.Status.Capacity.Cpu()
	mem := node.Status.Capacity.Memory()

	return NodeInfo{
		Name:           node.Name,
		Status:         status,
		Roles:          rolesStr,
		Version:        node.Status.NodeInfo.KubeletVersion,
		OS:             node.Status.NodeInfo.OSImage,
		Arch:           node.Status.NodeInfo.Architecture,
		CPUCapacity:    cpu.String(),
		MemoryCapacity: formatMemory(mem.Value()),
		InternalIP:     internalIP,
		Age:            formatAge(node.CreationTimestamp.Time),
		PodCIDR:        node.Spec.PodCIDR,
	}
}
