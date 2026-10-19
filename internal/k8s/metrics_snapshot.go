package k8s

import (
	"encoding/json"
	"sort"
	"strconv"
)

// ClusterMetricsSnapshot — текущая загрузка кластера по данным Prometheus.
// CPU в ядрах, память в байтах; Percent считается от суммарной ёмкости нод, на которых работает node-exporter.
type ClusterMetricsSnapshot struct {
	CPUPercent     float64 `json:"cpuPercent"`
	CPUUsage       float64 `json:"cpuUsage"`
	CPUCapacity    float64 `json:"cpuCapacity"`
	MemoryPercent  float64 `json:"memoryPercent"`
	MemoryUsage    float64 `json:"memoryUsage"`
	MemoryCapacity float64 `json:"memoryCapacity"`
}

func (m *Manager) GetClusterMetricsSnapshot() (*ClusterMetricsSnapshot, error) {
	snapshot := &ClusterMetricsSnapshot{}

	// Агрегаты считает сам Prometheus: суммирование по нодам на стороне клиента
	// давало бы проценты больше 100 на многонодовых кластерах.
	snapshot.CPUUsage = m.promScalar(promQueryClusterCPUUsedCores())
	snapshot.CPUCapacity = m.promScalar(promQueryClusterCPUTotalCores())
	if snapshot.CPUCapacity > 0 {
		snapshot.CPUPercent = snapshot.CPUUsage / snapshot.CPUCapacity * 100
	}

	snapshot.MemoryCapacity = m.promScalar(promQueryClusterMemTotalBytes())
	if snapshot.MemoryCapacity > 0 {
		available := m.promScalar(promQueryClusterMemAvailableBytes())
		snapshot.MemoryUsage = snapshot.MemoryCapacity - available
		snapshot.MemoryPercent = snapshot.MemoryUsage / snapshot.MemoryCapacity * 100
	}

	return snapshot, nil
}

// promScalar выполняет instant-запрос и возвращает число; при любой ошибке — 0
func (m *Manager) promScalar(query string) float64 {
	result, err := m.QueryPrometheus(query)
	if err != nil || result == nil {
		return 0
	}
	return parsePrometheusResult(result)
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
	if !ok {
		return 0
	}

	if resultType != "vector" && resultType != "matrix" {
		return 0
	}

	results, ok := data["result"].([]interface{})
	if !ok || len(results) == 0 {
		return 0
	}

	// Sum all values from all results (for aggregated queries)
	var total float64
	for _, r := range results {
		if resultMap, ok := r.(map[string]interface{}); ok {
			if value, ok := resultMap["value"].([]interface{}); ok && len(value) >= 2 {
				if valueStr, ok := value[1].(string); ok {
					if floatVal, err := strconv.ParseFloat(valueStr, 64); err == nil {

						total += floatVal
					}
				}
			}
		}
	}

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

// GetNodeMetrics — загрузка CPU и памяти по нодам в процентах. Нода определяется по метке
// kubernetes_node (см. promNodeLabel); серии без неё пропускаются.
func (m *Manager) GetNodeMetrics() ([]NodeMetrics, error) {
	byNode := make(map[string]*NodeMetrics)

	// fill разбирает instant-вектор: одно значение на ноду
	fill := func(query string, set func(nm *NodeMetrics, raw string, pct int)) {
		res, err := m.QueryPrometheus(query)
		if err != nil || res == nil || res.Data == nil {
			return
		}
		var data struct {
			Result []struct {
				Metric map[string]string `json:"metric"`
				Value  []interface{}     `json:"value"`
			} `json:"result"`
		}
		if json.Unmarshal(res.Data, &data) != nil {
			return
		}
		for _, r := range data.Result {
			name := r.Metric[promNodeLabel]
			if name == "" || len(r.Value) < 2 {
				continue
			}
			raw, ok := r.Value[1].(string)
			if !ok {
				continue
			}
			f, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				continue
			}
			nm := byNode[name]
			if nm == nil {
				nm = &NodeMetrics{Name: name}
				byNode[name] = nm
			}
			set(nm, raw, int(f+0.5))
		}
	}

	fill(promQueryNodeCPU(), func(nm *NodeMetrics, raw string, pct int) {
		nm.CPUUsage, nm.CPUPct = raw, pct
	})
	fill(promQueryNodeMem(), func(nm *NodeMetrics, raw string, pct int) {
		nm.MemUsage, nm.MemPct = raw, pct
	})

	out := make([]NodeMetrics, 0, len(byNode))
	for _, nm := range byNode {
		out = append(out, *nm)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}
