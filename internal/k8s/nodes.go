package k8s

import (
	"context"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NodeInfo struct {
	Name           string `json:"name"`
	Status         string `json:"status"`
	Conditions     string `json:"conditions"`
	Roles          string `json:"roles"`
	Version        string `json:"version"`
	OS             string `json:"os"`
	Arch           string `json:"arch"`
	CPUCapacity    string `json:"cpuCapacity"`
	MemoryCapacity string `json:"memoryCapacity"`
	InternalIP     string `json:"internalIP"`
	Age            string `json:"age"`
	PodCIDR        string `json:"podCIDR"`
	Taints         int    `json:"taints"`
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

	// Conditions: Ready/NotReady плюс активные проблемные (MemoryPressure, DiskPressure, ...)
	conds := []string{status}
	for _, cond := range node.Status.Conditions {
		if cond.Type != corev1.NodeReady && cond.Status == corev1.ConditionTrue {
			conds = append(conds, string(cond.Type))
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
	sort.Strings(roles)
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
		Conditions:     strings.Join(conds, ", "),
		Roles:          rolesStr,
		Version:        node.Status.NodeInfo.KubeletVersion,
		OS:             node.Status.NodeInfo.OSImage,
		Arch:           node.Status.NodeInfo.Architecture,
		CPUCapacity:    cpu.String(),
		MemoryCapacity: formatMemory(mem.Value()),
		InternalIP:     internalIP,
		Age:            formatAge(node.CreationTimestamp.Time),
		PodCIDR:        node.Spec.PodCIDR,
		Taints:         len(node.Spec.Taints),
	}
}

// NodeTaintInfo — taint узла
type NodeTaintInfo struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Effect string `json:"effect"`
}

// NodeConditionInfo — condition узла
type NodeConditionInfo struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// NodeDetail — полные данные узла для панели деталей
type NodeDetail struct {
	Name              string              `json:"name"`
	Status            string              `json:"status"`
	Roles             string              `json:"roles"`
	KubeVersion       string              `json:"kubeVersion"`
	ContainerRuntime  string              `json:"containerRuntime"`
	OSImage           string              `json:"osImage"`
	Architecture      string              `json:"architecture"`
	InternalIP        string              `json:"internalIP"`
	ExternalIP        string              `json:"externalIP"`
	PodCIDR           string              `json:"podCIDR"`
	CPUCapacity       string              `json:"cpuCapacity"`
	MemoryCapacity    string              `json:"memoryCapacity"`
	AllocatableCPU    string              `json:"allocatableCPU"`
	AllocatableMemory string              `json:"allocatableMemory"`
	AllocatablePods   string              `json:"allocatablePods"`
	Age               string              `json:"age"`
	Created           string              `json:"created"`
	Labels            map[string]string   `json:"labels"`
	Annotations       map[string]string   `json:"annotations"`
	Taints            []NodeTaintInfo     `json:"taints"`
	Conditions        []NodeConditionInfo `json:"conditions"`
}

func (m *Manager) GetNodeDetail(name string) (*NodeDetail, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	node, err := conn.Client.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	info := convertNode(*node)

	internalIP, externalIP := "", ""
	for _, addr := range node.Status.Addresses {
		switch addr.Type {
		case corev1.NodeInternalIP:
			if internalIP == "" {
				internalIP = addr.Address
			}
		case corev1.NodeExternalIP:
			if externalIP == "" {
				externalIP = addr.Address
			}
		}
	}

	// "containerd://1.7.2" -> "1.7.2"
	runtime := node.Status.NodeInfo.ContainerRuntimeVersion
	if i := strings.Index(runtime, "://"); i >= 0 {
		runtime = runtime[i+3:]
	}

	taints := make([]NodeTaintInfo, 0, len(node.Spec.Taints))
	for _, t := range node.Spec.Taints {
		taints = append(taints, NodeTaintInfo{Key: t.Key, Value: t.Value, Effect: string(t.Effect)})
	}

	conditions := make([]NodeConditionInfo, 0, len(node.Status.Conditions))
	for _, c := range node.Status.Conditions {
		conditions = append(conditions, NodeConditionInfo{
			Type:    string(c.Type),
			Status:  string(c.Status),
			Reason:  c.Reason,
			Message: c.Message,
		})
	}

	return &NodeDetail{
		Name:              info.Name,
		Status:            info.Status,
		Roles:             info.Roles,
		KubeVersion:       info.Version,
		ContainerRuntime:  runtime,
		OSImage:           info.OS,
		Architecture:      info.Arch,
		InternalIP:        internalIP,
		ExternalIP:        externalIP,
		PodCIDR:           info.PodCIDR,
		CPUCapacity:       info.CPUCapacity,
		MemoryCapacity:    info.MemoryCapacity,
		AllocatableCPU:    node.Status.Allocatable.Cpu().String(),
		AllocatableMemory: formatMemory(node.Status.Allocatable.Memory().Value()),
		AllocatablePods:   node.Status.Allocatable.Pods().String(),
		Age:               info.Age,
		Created:           node.CreationTimestamp.UTC().Format(time.RFC3339),
		Labels:            node.Labels,
		Annotations:       node.Annotations,
		Taints:            taints,
		Conditions:        conditions,
	}, nil
}
