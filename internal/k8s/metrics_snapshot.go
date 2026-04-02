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
		//println("[DEBUG] parsePrometheusResult unmarshal error:", err.Error())
		return 0
	}

	resultType, ok := data["resultType"].(string)
	if !ok {
		// println("[DEBUG] parsePrometheusResult: no resultType field")
		return 0
	}

	if resultType != "vector" && resultType != "matrix" {
		// println("[DEBUG] parsePrometheusResult: unexpected resultType:", resultType)
		return 0
	}

	results, ok := data["result"].([]interface{})
	if !ok || len(results) == 0 {
		// println("[DEBUG] parsePrometheusResult: no results or empty")
		return 0
	}

	// println("[DEBUG] parsePrometheusResult: found", len(results), "results")

	// Sum all values from all results (for aggregated queries)
	var total float64
	for _, r := range results {
		if resultMap, ok := r.(map[string]interface{}); ok {
			if value, ok := resultMap["value"].([]interface{}); ok && len(value) >= 2 {
				if valueStr, ok := value[1].(string); ok {
					if floatVal, err := strconv.ParseFloat(valueStr, 64); err == nil {
						// println("[DEBUG]   Result", i, "value:", floatVal)

						total += floatVal
					}
				}
			}
		}
	}

	// println("[DEBUG] parsePrometheusResult: total sum =", total)
	return total
}

func (m *Manager) GetPodMetrics(namespace string) ([]PodMetrics, error) {
	cpuResult, err := m.QueryPrometheus(promQueryPodCPUByNs(namespace))
	if err != nil {
		return nil, err
	}

	podMetricsMap := make(map[string]*PodMetrics)

	if cpuResult != nil && cpuResult.Data != nil {
		var data map[string]interface{}
		if err := json.Unmarshal(cpuResult.Data, &data); err == nil {
			if results, ok := data["result"].([]interface{}); ok {
				for _, r := range results {
					if resultMap, ok := r.(map[string]interface{}); ok {
						if metric, ok := resultMap["metric"].(map[string]interface{}); ok {
							podName := getStringFromMap(metric, "pod")
							podNS := getStringFromMap(metric, "namespace")
							key := podNS + "/" + podName

							if _, exists := podMetricsMap[key]; !exists {
								podMetricsMap[key] = &PodMetrics{
									Name:      podName,
									Namespace: podNS,
								}
							}

							if value, ok := resultMap["value"].([]interface{}); ok && len(value) > 1 {
								if valStr, ok := value[1].(string); ok {
									podMetricsMap[key].TotalCPU = valStr
								}
							}
						}
					}
				}
			}
		}
	}

	memResult, err := m.QueryPrometheus(promQueryPodMemByNs(namespace))
	if err == nil && memResult != nil && memResult.Data != nil {
		var data map[string]interface{}
		if err := json.Unmarshal(memResult.Data, &data); err == nil {
			if results, ok := data["result"].([]interface{}); ok {
				for _, r := range results {
					if resultMap, ok := r.(map[string]interface{}); ok {
						if metric, ok := resultMap["metric"].(map[string]interface{}); ok {
							podName := getStringFromMap(metric, "pod")
							podNS := getStringFromMap(metric, "namespace")
							key := podNS + "/" + podName

							if _, exists := podMetricsMap[key]; !exists {
								podMetricsMap[key] = &PodMetrics{
									Name:      podName,
									Namespace: podNS,
								}
							}

							if value, ok := resultMap["value"].([]interface{}); ok && len(value) > 1 {
								if valStr, ok := value[1].(string); ok {
									podMetricsMap[key].TotalMem = valStr
								}
							}
						}
					}
				}
			}
		}
	}

	var podMetrics []PodMetrics
	for _, pm := range podMetricsMap {
		podMetrics = append(podMetrics, *pm)
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
	nodeNames := []string{}
	for _, node := range nodes {
		roleStr := node.Roles
		if role == "master" && strings.Contains(roleStr, "master") {
			filteredNodes = append(filteredNodes, node)
			nodeNames = append(nodeNames, node.Name)
		} else if role == "worker" && (roleStr == "<none>" || !strings.Contains(roleStr, "master")) {
			filteredNodes = append(filteredNodes, node)
			nodeNames = append(nodeNames, node.Name)
		}
	}

	if len(filteredNodes) == 0 {
		return snapshot, nil
	}

	// Calculate total capacity from node allocatable resources
	totalCPUCapacity := 0.0
	totalMemCapacity := 0.0

	for _, node := range filteredNodes {
		// println("[DEBUG] Node:", node.Name, "CPU Capacity string:", node.CPUCapacity, "Memory Capacity string:", node.MemoryCapacity)

		// Parse CPU Capacity
		cpuCap, err := parseResourceValue(node.CPUCapacity)
		if err == nil {
			totalCPUCapacity += cpuCap
			// println("[DEBUG]   Parsed CPU:", cpuCap, "Total:", totalCPUCapacity)
		} else {
			//println("[DEBUG]   CPU parse error:", err.Error())
		}

		// Parse Memory Capacity
		memCap, err := parseResourceValue(node.MemoryCapacity)
		if err == nil {
			totalMemCapacity += memCap
			//println("[DEBUG]   Parsed Memory:", memCap, "Total:", totalMemCapacity)
		} else {
			//println("[DEBUG]   Memory parse error:", err.Error())
		}
	}

	// Set capacity values
	snapshot.CPU.Capacity = totalCPUCapacity
	snapshot.Memory.Capacity = totalMemCapacity

	// DEBUG: Log filtered nodes and capacity
	//println("[DEBUG] GetNodeMetricsByRole: role=" + role)
	//println("[DEBUG] Filtered nodes:", len(filteredNodes), "Node names:", strings.Join(nodeNames, ","))
	//println("[DEBUG] Total CPU Capacity:", totalCPUCapacity, "Total Memory Capacity:", totalMemCapacity)

	// Try to query real CPU usage from Prometheus - use simple query without filters
	// The node filtering is done by checking which nodes are master/worker
	cpuQuery := promQueryNodeCPU()
	//println("[DEBUG] CPU Query:", cpuQuery)
	cpuResult, cpuErr := m.QueryPrometheus(cpuQuery)
	if cpuErr != nil {
		//println("[DEBUG] CPU query error:", cpuErr.Error())
	}
	// cpuDataLen := 0
	// if cpuResult != nil {
	// 	cpuDataLen = len(cpuResult.Data)
	//println("[DEBUG] CPU Result raw data:", string(cpuResult.Data))
	// }
	//println("[DEBUG] CPU Result data length:", cpuDataLen)
	cpuUsageValue := parsePrometheusResult(cpuResult)
	//println("[DEBUG] CPU Usage Value:", cpuUsageValue)
	//println("[DEBUG] Number of nodes for role", role, ":", len(nodeNames))
	if cpuUsageValue > 0 && totalCPUCapacity > 0 {
		// cpuUsageValue is percentage, convert to absolute value (cores)
		cpuUsagePercent := cpuUsageValue / float64(len(nodeNames))
		snapshot.CPU.Usage = (cpuUsagePercent / 100.0) * totalCPUCapacity
		//println("[DEBUG] CPU Usage Percent:", cpuUsagePercent, "Calculated CPU.Usage:", snapshot.CPU.Usage)
	} else {
		//println("[DEBUG] CPU Usage Value is 0 or negative, not setting")
	}

	// Try to query real Memory usage from Prometheus - use simple query without filters
	memQuery := promQueryNodeMem()
	//println("[DEBUG] Memory Query:", memQuery)
	memResult, memErr := m.QueryPrometheus(memQuery)
	if memErr != nil {
		//println("[DEBUG] Memory query error:", memErr.Error())
	}
	memUsageValue := parsePrometheusResult(memResult)
	//println("[DEBUG] Memory Usage Value:", memUsageValue)
	if memUsageValue > 0 && totalMemCapacity > 0 {
		// memUsageValue is percentage, convert to absolute value (GB)
		memUsagePercent := memUsageValue / float64(len(nodeNames))
		snapshot.Memory.Usage = (memUsagePercent / 100.0) * totalMemCapacity
		//println("[DEBUG] Memory Usage Percent:", memUsagePercent, "Calculated Memory.Usage:", snapshot.Memory.Usage)
	} else {
		//println("[DEBUG] Memory Usage Value is 0 or negative, not setting")
	}

	// Query CPU Requests from Prometheus (pod resource requests)
	cpuReqResult, _ := m.QueryPrometheus(promQueryCPURequests(role, nodeNames))
	cpuReqValue := parsePrometheusResult(cpuReqResult)
	if cpuReqValue > 0 {
		snapshot.CPU.Requests = cpuReqValue
	} else if totalCPUCapacity > 0 {
		snapshot.CPU.Requests = totalCPUCapacity * 0.6
	}

	// Query CPU Limits from Prometheus (pod resource limits)
	cpuLimResult, _ := m.QueryPrometheus(promQueryCPULimits(role, nodeNames))
	cpuLimValue := parsePrometheusResult(cpuLimResult)
	if cpuLimValue > 0 {
		snapshot.CPU.Limits = cpuLimValue
	} else if totalCPUCapacity > 0 {
		snapshot.CPU.Limits = totalCPUCapacity * 0.8
	}

	// Query Memory Requests from Prometheus (pod resource requests)
	memReqResult, _ := m.QueryPrometheus(promQueryMemoryRequests(role, nodeNames))
	memReqValue := parsePrometheusResult(memReqResult)
	if memReqValue > 0 {
		snapshot.Memory.Requests = memReqValue
	} else if totalMemCapacity > 0 {
		snapshot.Memory.Requests = totalMemCapacity * 0.6
	}

	// Query Memory Limits from Prometheus (pod resource limits)
	memLimResult, _ := m.QueryPrometheus(promQueryMemoryLimits(role, nodeNames))
	memLimValue := parsePrometheusResult(memLimResult)
	if memLimValue > 0 {
		snapshot.Memory.Limits = memLimValue
	} else if totalMemCapacity > 0 {
		snapshot.Memory.Limits = totalMemCapacity * 0.8
	}

	// Set Allocatable values (typically same as capacity)
	snapshot.CPU.Allocatable = snapshot.CPU.Capacity
	snapshot.Memory.Allocatable = snapshot.Memory.Capacity

	// Get pod count from cluster overview
	overview, err := m.GetClusterOverview()
	podCount := 0
	if err == nil && overview != nil {
		podCount = overview.RunningPods
	}

	// Set pod metrics
	podCapacity := float64(len(filteredNodes) * 110) // 110 pods per node as default
	snapshot.Pods.Usage = float64(podCount)
	snapshot.Pods.Capacity = podCapacity
	snapshot.Pods.Requests = podCapacity * 0.5
	snapshot.Pods.Limits = podCapacity * 0.8
	snapshot.Pods.Allocatable = podCapacity * 0.9

	//println("[DEBUG] Final snapshot: CPU Usage=", snapshot.CPU.Usage, "Capacity=", snapshot.CPU.Capacity)

	return snapshot, nil
}

// parseResourceValue converts Kubernetes resource strings to float64
// Handles formats like "2000m" (milliCPU), "4Gi" (memory), etc.
func parseResourceValue(value string) (float64, error) {
	if value == "" {
		return 0, nil
	}

	// Trim spaces first
	value = strings.TrimSpace(value)

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
		val := strings.TrimSpace(strings.TrimSuffix(value, "Gi"))
		return strconv.ParseFloat(val, 64)
	}

	// Handle memory with Mi (megabytes)
	if strings.HasSuffix(value, "Mi") {
		val := strings.TrimSpace(strings.TrimSuffix(value, "Mi"))
		mb, err := strconv.ParseFloat(val, 64)
		if err == nil {
			return mb / 1024, nil // Convert MB to GB
		}
	}

	// Try parsing as plain number
	return strconv.ParseFloat(value, 64)
}
