package k8s

func promQueryNodeCPU() string {
	return "sum by (instance) (rate(node_cpu_seconds_total{mode!=" +
		string('"') + "idle" + string('"') + "}[5m])) * 100"
}

func promQueryNodeMem() string {
	return "(1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes) * 100"
}

func promQueryPodCPUTop() string {
	return "topk(10, sum by (pod, namespace) (rate(container_cpu_usage_seconds_total{container!=" +
		string('"') + string('"') + "}[5m])) * 1000)"
}

func promQueryPodMemTop() string {
	return "topk(10, sum by (pod, namespace) (container_memory_working_set_bytes{container!=" +
		string('"') + string('"') + "}))"
}

func promQueryPodCPUByNs(namespace string) string {
	if namespace == "" {
		return "sum by (pod, namespace, container) (rate(container_cpu_usage_seconds_total{container!=" +
			string('"') + string('"') + "}[5m])) * 1000"
	}
	return "sum by (pod, namespace, container) (rate(container_cpu_usage_seconds_total{container!=" +
		string('"') + string('"') + ",namespace=" +
		string('"') + namespace + string('"') + "}[5m])) * 1000"
}

func promQueryNodeCPUForMetrics() string {
	return promQueryNodeCPU()
}
