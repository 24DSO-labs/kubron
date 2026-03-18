package k8s

import (
	"time"
)

func (m *Manager) GetClusterMetricsSnapshot() (*MetricsSnapshot, error) {
	snapshot := &MetricsSnapshot{
		Timestamp: time.Now(),
	}
	cpuResult, err := m.QueryPrometheus(promQueryNodeCPU())
	if err == nil {
		snapshot.NodesCPU = cpuResult.Data
	}

	memResult, err := m.QueryPrometheus(promQueryNodeMem())
	if err == nil {
		snapshot.NodesMem = memResult.Data
	}

	podCPU, err := m.QueryPrometheus(promQueryPodCPUTop())
	if err == nil {
		snapshot.PodsCPU = podCPU.Data
	}

	podMem, err := m.QueryPrometheus(promQueryPodMemTop())
	if err == nil {
		snapshot.PodsMem = podMem.Data
	}

	return snapshot, nil
}

func (m *Manager) GetPodMetrics(namespace string) ([]PodMetrics, error) {
	_, err := m.QueryPrometheus(promQueryPodCPUByNs(namespace))
	if err != nil {
		return nil, err
	}
	return []PodMetrics{}, nil
}
func (m *Manager) GetNodeMetrics() ([]NodeMetrics, error) {
	_, err := m.QueryPrometheus(promQueryNodeCPU())
	if err != nil {
		return nil, err
	}
	return []NodeMetrics{}, nil
}
