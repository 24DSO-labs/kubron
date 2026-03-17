package k8s

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

type PodMetrics struct {
	Name       string             `json:"name"`
	Namespace  string             `json:"namespace"`
	Containers []ContainerMetrics `json:"containers"`
	TotalCPU   string             `json:"totalCpu"`
	TotalMem   string             `json:"totalMem"`
}

type ContainerMetrics struct {
	Name   string `json:"name"`
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

type NodeMetrics struct {
	Name     string `json:"name"`
	CPUUsage string `json:"cpuUsage"`
	MemUsage string `json:"memUsage"`
	CPUPct   int    `json:"cpuPct"`
	MemPct   int    `json:"memPct"`
}

func (m *Manager) getMetricsClient() (*metricsv.Clientset, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	client, err := metricsv.NewForConfig(conn.Config)
	if err != nil {
		return nil, fmt.Errorf("metrics-server недоступен: %w", err)
	}
	return client, nil
}

func (m *Manager) GetPodMetrics(namespace string) ([]PodMetrics, error) {
	mc, err := m.getMetricsClient()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	metricsList, err := mc.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("не удалось получить метрики подов: %w", err)
	}

	var result []PodMetrics
	for _, pm := range metricsList.Items {
		var containers []ContainerMetrics
		var totalCPU, totalMem int64

		for _, c := range pm.Containers {
			cpuVal := c.Usage.Cpu().MilliValue()
			memVal := c.Usage.Memory().Value()
			totalCPU += cpuVal
			totalMem += memVal

			containers = append(containers, ContainerMetrics{
				Name:   c.Name,
				CPU:    fmt.Sprintf("%dm", cpuVal),
				Memory: formatMemory(memVal),
			})
		}

		result = append(result, PodMetrics{
			Name:       pm.Name,
			Namespace:  pm.Namespace,
			Containers: containers,
			TotalCPU:   fmt.Sprintf("%dm", totalCPU),
			TotalMem:   formatMemory(totalMem),
		})
	}

	return result, nil
}

func (m *Manager) GetNodeMetrics() ([]NodeMetrics, error) {
	mc, err := m.getMetricsClient()
	if err != nil {
		return nil, err
	}

	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Получаем метрики нод
	metricsList, err := mc.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("не удалось получить метрики нод: %w", err)
	}

	// Получаем capacity нод
	nodeList, err := conn.Client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	capacityMap := make(map[string][2]int64) // name -> [cpuMillis, memBytes]
	for _, node := range nodeList.Items {
		cpuCap := node.Status.Capacity.Cpu().MilliValue()
		memCap := node.Status.Capacity.Memory().Value()
		capacityMap[node.Name] = [2]int64{cpuCap, memCap}
	}

	var result []NodeMetrics
	for _, nm := range metricsList.Items {
		cpuUsage := nm.Usage.Cpu().MilliValue()
		memUsage := nm.Usage.Memory().Value()

		cpuPct := 0
		memPct := 0
		if cap, ok := capacityMap[nm.Name]; ok {
			if cap[0] > 0 {
				cpuPct = int(cpuUsage * 100 / cap[0])
			}
			if cap[1] > 0 {
				memPct = int(memUsage * 100 / cap[1])
			}
		}

		result = append(result, NodeMetrics{
			Name:     nm.Name,
			CPUUsage: fmt.Sprintf("%dm", cpuUsage),
			MemUsage: formatMemory(memUsage),
			CPUPct:   cpuPct,
			MemPct:   memPct,
		})
	}

	return result, nil
}
