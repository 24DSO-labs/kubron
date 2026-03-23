package k8s

import "fmt"

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

func promQueryPodMemByNs(namespace string) string {
	return fmt.Sprintf(
		`sum by (pod, namespace) (container_memory_working_set_bytes{namespace="%s", container!="", container!="POD"})`,
		namespace,
	)
}

// Для вычисления процентов CPU нужны лимиты/requests
func promQueryPodCPULimitByNs(namespace string) string {
	return fmt.Sprintf(
		`sum by (pod, namespace) (kube_pod_container_resource_limits{namespace="%s", resource="cpu"})`,
		namespace,
	)
}

func promQueryPodMemLimitByNs(namespace string) string {
	return fmt.Sprintf(
		`sum by (pod, namespace) (kube_pod_container_resource_limits{namespace="%s", resource="memory"})`,
		namespace,
	)
}

func promQueryNodeCPUForMetrics() string {
	return promQueryNodeCPU()
}

func promQueryNodeCPUByRole(role string, nodeNames []string) string {
	if len(nodeNames) == 0 {
		return "sum by (instance) (rate(node_cpu_seconds_total{mode!=\"idle\"}[5m])) * 100"
	}
	return "sum by (instance) (rate(node_cpu_seconds_total{mode!=\"idle\",instance=~\"" + joinNodeNames(nodeNames) + "\"}[5m])) * 100"
}

func promQueryNodeMemByRole(role string, nodeNames []string) string {
	if len(nodeNames) == 0 {
		return "avg by (instance) (1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100"
	}
	nodeFilter := "instance=~\"" + joinNodeNames(nodeNames) + "\""
	return "avg by (instance) (1 - (node_memory_MemAvailable_bytes{" + nodeFilter + "} / node_memory_MemTotal_bytes{" + nodeFilter + "})) * 100"
}

func promQueryCPURequests(role string, nodeNames []string) string {
	if len(nodeNames) == 0 {
		return "sum(kube_pod_container_resource_requests_cpu_cores) / 1000"
	}
	return "sum(kube_pod_container_resource_requests_cpu_cores{node=~\"" + joinNodeNames(nodeNames) + "\"})"
}

func promQueryCPULimits(role string, nodeNames []string) string {
	if len(nodeNames) == 0 {
		return "sum(kube_pod_container_resource_limits_cpu_cores) / 1000"
	}
	return "sum(kube_pod_container_resource_limits_cpu_cores{node=~\"" + joinNodeNames(nodeNames) + "\"})"
}

func promQueryMemoryRequests(role string, nodeNames []string) string {
	if len(nodeNames) == 0 {
		return "sum(kube_pod_container_resource_requests_memory_bytes) / 1024 / 1024 / 1024"
	}
	return "sum(kube_pod_container_resource_requests_memory_bytes{node=~\"" + joinNodeNames(nodeNames) + "\"}) / 1024 / 1024 / 1024"
}

func promQueryMemoryLimits(role string, nodeNames []string) string {
	if len(nodeNames) == 0 {
		return "sum(kube_pod_container_resource_limits_memory_bytes) / 1024 / 1024 / 1024"
	}
	return "sum(kube_pod_container_resource_limits_memory_bytes{node=~\"" + joinNodeNames(nodeNames) + "\"}) / 1024 / 1024 / 1024"
}

func joinNodeNames(nodeNames []string) string {
	if len(nodeNames) == 0 {
		return ""
	}
	result := ""
	for i, name := range nodeNames {
		if i > 0 {
			result += "|"
		}
		result += name
	}
	return result
}
