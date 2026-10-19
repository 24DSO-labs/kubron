package k8s

import (
	"encoding/json"
	"time"
)

const (
	MetricsNamespace = "kubron-metrics"
	MetricsManifest  = "metrics-stack.yaml"
	PrometheusSvcURL = "http://prometheus.kubron-metrics.svc:80"
)

// MetricsComponentStatus — состояние одного компонента стека метрик.
// Формат совпадает с ComponentStatus во frontend/src/types/metrics.ts.
type MetricsComponentStatus struct {
	Ready             bool  `json:"ready"`
	Replicas          int32 `json:"replicas"`
	AvailableReplicas int32 `json:"availableReplicas"`
}

// MetricsStackStatus — статус стека метрик.
// Status: not_installed | installing | installed | degraded.
type MetricsStackStatus struct {
	Status        string                            `json:"status"`
	Components    map[string]MetricsComponentStatus `json:"components"`
	Message       string                            `json:"message,omitempty"`
	PrometheusURL string                            `json:"prometheusUrl,omitempty"`
}

type PromQueryResult struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
}

type PodMetrics struct {
	Name          string             `json:"name"`
	Namespace     string             `json:"namespace"`
	Containers    []ContainerMetrics `json:"containers"`
	TotalCPU      string             `json:"totalCpu"`
	TotalMem      string             `json:"totalMem"`
	CPUCores      float64            `json:"cpuCores"`
	MemoryBytes   float64            `json:"memoryBytes"`
	CPUPercent    float64            `json:"cpuPercent"`
	MemoryPercent float64            `json:"memoryPercent"`
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

type MetricsSnapshot struct {
	Timestamp time.Time       `json:"timestamp"`
	NodesCPU  json.RawMessage `json:"nodesCpu,omitempty"`
	NodesMem  json.RawMessage `json:"nodesMem,omitempty"`
	PodsCPU   json.RawMessage `json:"podsCpu,omitempty"`
	PodsMem   json.RawMessage `json:"podsMem,omitempty"`
}
