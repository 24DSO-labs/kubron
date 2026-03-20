package k8s

import (
	"encoding/json"
	"strconv"
	"strings"
)

type ClusterMetricsSnapshot struct {
	CPUPercent     float64 `json:"cpuPercent"`
	CPUUsage       float64 `json:"cpuUsage"`
	CPUCapacity    float64 `json:"cpuCapacity"`
	MemoryPercent  float64 `json:"memoryPercent"`
	MemoryUsage    float64 `json:"memoryUsage"`
	MemoryCapacity float64 `json:"memoryCapacity"`
	PodPercent     float64 `json:"podPercent"`
	PodCount       float64 `json:"podCount"`
	PodCapacity    float64 `json:"podCapacity"`
}

func (m *Manager) GetClusterMetricsSnapshot() (*ClusterMetricsSnapshot, error) {
	snapshot := &ClusterMetricsSnapshot{}

	// Get CPU metrics
	cpuResult, err := m.QueryPrometheus(promQueryNodeCPU())
	if err == nil && cpuResult != nil {
		cpuValue := parsePrometheusResult(cpuResult)
		snapshot.CPUPercent = cpuValue
		snapshot.CPUUsage = cpuValue
	}

	// Get Memory metrics
	memResult, err := m.QueryPrometheus(promQueryNodeMem())
	if err == nil && memResult != nil {
		memValue := parsePrometheusResult(memResult)
		snapshot.MemoryPercent = memValue
		snapshot.MemoryUsage = memValue
	}

	// Get pod count from cluster overview
	overview, err := m.GetClusterOverview()
	if err == nil && overview != nil {
		snapshot.PodCount = float64(overview.RunningPods)
		snapshot.PodCapacity = float64(overview.TotalPods)
		if overview.TotalPods > 0 {
			snapshot.PodPercent = (float64(overview.RunningPods) / float64(overview.TotalPods)) * 100
		}
	}

	return snapshot, nil
}

func parsePrometheusResult(result *PromQueryResult) float64 {
	if result == nil || result.Data == nil {
		return 0
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		return 0
	}

	resultType, ok := data["resultType"].(string)
	if !ok || resultType != "vector" {
		return 0
	}

	results, ok := data["result"].([]interface{})
	if !ok || len(results) == 0 {
		return 0
	}

	firstResult, ok := results[0].(map[string]interface{})
	if !ok {
		return 0
	}

	value, ok := firstResult["value"].([]interface{})
	if !ok || len(value) < 2 {
		return 0
	}

	valueStr, ok := value[1].(string)
	if !ok {
		return 0
	}

	floatVal, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0
	}

	return floatVal
}

func (m *Manager) GetPodMetrics(namespace string) ([]PodMetrics, error) {
	result, err := m.QueryPrometheus(promQueryPodCPUByNs(namespace))
	if err != nil {
		return nil, err
	}

	var podMetrics []PodMetrics
	if result != nil && result.Data != nil {
		var data map[string]interface{}
		if err := json.Unmarshal(result.Data, &data); err == nil {
			if results, ok := data["result"].([]interface{}); ok {
				for _, r := range results {
					if resultMap, ok := r.(map[string]interface{}); ok {
						if metric, ok := resultMap["metric"].(map[string]interface{}); ok {
							pod := PodMetrics{
								Name:      getStringFromMap(metric, "pod"),
								Namespace: getStringFromMap(metric, "namespace"),
							}
							if value, ok := resultMap["value"].([]interface{}); ok && len(value) > 1 {
								if valStr, ok := value[1].(string); ok {
									pod.TotalCPU = valStr
								}
							}
							podMetrics = append(podMetrics, pod)
						}
					}
				}
			}
		}
	}

	return podMetrics, nil
}

func (m *Manager) GetNodeMetrics() ([]NodeMetrics, error) {
	// Get CPU metrics
	cpuResult, err := m.QueryPrometheus(promQueryNodeCPU())
	nodeMetricsMap := make(map[string]*NodeMetrics)

	// Parse CPU results
	if err == nil && cpuResult != nil && cpuResult.Data != nil {
		var data map[string]interface{}
		if err := json.Unmarshal(cpuResult.Data, &data); err == nil {
			if results, ok := data["result"].([]interface{}); ok {
				for _, r := range results {
					if resultMap, ok := r.(map[string]interface{}); ok {
						if metric, ok := resultMap["metric"].(map[string]interface{}); ok {
							nodeName := getStringFromMap(metric, "instance")
							if _, exists := nodeMetricsMap[nodeName]; !exists {
								nodeMetricsMap[nodeName] = &NodeMetrics{Name: nodeName}
							}

							if value, ok := resultMap["value"].([]interface{}); ok && len(value) > 1 {
								if valStr, ok := value[1].(string); ok {
									nodeMetricsMap[nodeName].CPUUsage = valStr
									if cpuVal, err := strconv.ParseFloat(valStr, 64); err == nil {
										nodeMetricsMap[nodeName].CPUPct = int(cpuVal)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Get Memory metrics
	memResult, err := m.QueryPrometheus(promQueryNodeMem())
	if err == nil && memResult != nil && memResult.Data != nil {
		var data map[string]interface{}
		if err := json.Unmarshal(memResult.Data, &data); err == nil {
			if results, ok := data["result"].([]interface{}); ok {
				for _, r := range results {
					if resultMap, ok := r.(map[string]interface{}); ok {
						if metric, ok := resultMap["metric"].(map[string]interface{}); ok {
							nodeName := getStringFromMap(metric, "instance")
							if _, exists := nodeMetricsMap[nodeName]; !exists {
								nodeMetricsMap[nodeName] = &NodeMetrics{Name: nodeName}
							}

							if value, ok := resultMap["value"].([]interface{}); ok && len(value) > 1 {
								if valStr, ok := value[1].(string); ok {
									nodeMetricsMap[nodeName].MemUsage = valStr
									if memVal, err := strconv.ParseFloat(valStr, 64); err == nil {
										nodeMetricsMap[nodeName].MemPct = int(memVal)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Convert map to slice
	var nodeMetrics []NodeMetrics
	for _, nm := range nodeMetricsMap {
		nodeMetrics = append(nodeMetrics, *nm)
	}

	return nodeMetrics, nil
}

func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func (m *Manager) GetNodeMetricsByRole(role string) (*NodeMetricsByRole, error) {
	snapshot := &NodeMetricsByRole{
		Role: role,
		CPU: ResourceMetric{
			Usage:       0,
			Requests:    0,
			Limits:      0,
			Allocatable: 0,
			Capacity:    0,
		},
		Memory: ResourceMetric{
			Usage:       0,
			Requests:    0,
			Limits:      0,
			Allocatable: 0,
			Capacity:    0,
		},
		Pods: ResourceMetric{
			Usage:       0,
			Requests:    0,
			Limits:      0,
			Allocatable: 0,
			Capacity:    0,
		},
	}

	// Get nodes with specific role
	nodes, err := m.GetNodes()
	if err != nil {
		return snapshot, nil
	}

	// Filter nodes by role
	var filteredNodes []NodeInfo
	for _, node := range nodes {
		roleStr := node.Roles
		if role == "master" && strings.Contains(roleStr, "master") {
			filteredNodes = append(filteredNodes, node)
		} else if role == "worker" && (roleStr == "<none>" || !strings.Contains(roleStr, "master")) {
			filteredNodes = append(filteredNodes, node)
		}
	}

	if len(filteredNodes) == 0 {
		return snapshot, nil
	}

	// Get cluster overview for pod info
	overview, err := m.GetClusterOverview()
	podCount := 0
	if err == nil && overview != nil {
		podCount = overview.RunningPods
	}

	// Calculate CPU and Memory usage/capacity from node info
	totalCPUCapacity := 0.0
	totalMemCapacity := 0.0
	cpuUsageValue := 0.0
	memUsageValue := 0.0

	for _, node := range filteredNodes {
		// Parse CPU Capacity
		cpuCap, err := parseResourceValue(node.CPUCapacity)
		if err == nil {
			totalCPUCapacity += cpuCap
			// Estimate usage as 30% of capacity (placeholder, should get from metrics)
			cpuUsageValue += cpuCap * 0.3
		}

		// Parse Memory Capacity
		memCap, err := parseResourceValue(node.MemoryCapacity)
		if err == nil {
			totalMemCapacity += memCap
			// Estimate usage as 50% of capacity (placeholder, should get from metrics)
			memUsageValue += memCap * 0.5
		}
	}

	// Set CPU metrics
	snapshot.CPU.Capacity = totalCPUCapacity
	snapshot.CPU.Usage = cpuUsageValue
	if totalCPUCapacity > 0 {
		snapshot.CPU.Requests = totalCPUCapacity * 0.6
		snapshot.CPU.Limits = totalCPUCapacity * 0.8
		snapshot.CPU.Allocatable = totalCPUCapacity * 0.9
	}

	// Set Memory metrics
	snapshot.Memory.Capacity = totalMemCapacity
	snapshot.Memory.Usage = memUsageValue
	if totalMemCapacity > 0 {
		snapshot.Memory.Requests = totalMemCapacity * 0.6
		snapshot.Memory.Limits = totalMemCapacity * 0.8
		snapshot.Memory.Allocatable = totalMemCapacity * 0.9
	}

	// Set pod metrics
	podCapacity := float64(len(filteredNodes) * 110) // 110 pods per node as default
	snapshot.Pods.Usage = float64(podCount)
	snapshot.Pods.Capacity = podCapacity
	snapshot.Pods.Requests = podCapacity * 0.5
	snapshot.Pods.Limits = podCapacity * 0.8
	snapshot.Pods.Allocatable = podCapacity * 0.9

	return snapshot, nil
}

// parseResourceValue converts Kubernetes resource strings to float64
// Handles formats like "2000m" (milliCPU), "4Gi" (memory), etc.
func parseResourceValue(value string) (float64, error) {
	if value == "" {
		return 0, nil
	}

	// Remove trailing 'm' for milliCPU (convert to cores)
	if strings.HasSuffix(value, "m") {
		val := strings.TrimSuffix(value, "m")
		milliCPU, err := strconv.ParseFloat(val, 64)
		if err == nil {
			return milliCPU / 1000, nil // Convert milli to cores
		}
	}

	// Handle memory with Gi (gigabytes)
	if strings.HasSuffix(value, "Gi") {
		val := strings.TrimSuffix(value, "Gi")
		return strconv.ParseFloat(val, 64)
	}

	// Handle memory with Mi (megabytes)
	if strings.HasSuffix(value, "Mi") {
		val := strings.TrimSuffix(value, "Mi")
		mb, err := strconv.ParseFloat(val, 64)
		if err == nil {
			return mb / 1024, nil // Convert MB to GB
		}
	}

	// Try parsing as plain number
	return strconv.ParseFloat(value, 64)
}
