package k8s

import (
	"fmt"
	"strings"
)

// promLabelValue экранирует значение для подстановки в строковый литерал PromQL.
func promLabelValue(v string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`).Replace(v)
}

func promQueryNodeCPU() string {
	return "sum by (instance) (rate(node_cpu_seconds_total{mode!=" +
		string('"') + "idle" + string('"') + "}[5m])) * 100"
}

func promQueryNodeMem() string {
	return "(1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes) * 100"
}

// Агрегаты по всему кластеру (одно значение): используются на главном экране

func promQueryClusterCPUUsedCores() string {
	return `sum(rate(node_cpu_seconds_total{mode!="idle"}[5m]))`
}

func promQueryClusterCPUTotalCores() string {
	return `count(node_cpu_seconds_total{mode="idle"})`
}

func promQueryClusterMemTotalBytes() string {
	return `sum(node_memory_MemTotal_bytes)`
}

func promQueryClusterMemAvailableBytes() string {
	return `sum(node_memory_MemAvailable_bytes)`
}

func promQueryPodCPUByNs(namespace string) string {
	if namespace == "" {
		return `sum by (pod, namespace) (rate(container_cpu_usage_seconds_total{container!="", container!="POD"}[5m])) * 1000`
	}
	return fmt.Sprintf(
		`sum by (pod, namespace) (rate(container_cpu_usage_seconds_total{container!="", container!="POD", namespace="%s"}[5m])) * 1000`,
		promLabelValue(namespace),
	)
}

func promQueryPodMemByNs(namespace string) string {
	if namespace == "" {
		return `sum by (pod, namespace) (container_memory_working_set_bytes{container!="", container!="POD"})`
	}
	return fmt.Sprintf(
		`sum by (pod, namespace) (container_memory_working_set_bytes{namespace="%s", container!="", container!="POD"})`,
		promLabelValue(namespace),
	)
}

// Для вычисления процентов CPU нужны лимиты/requests
func promQueryPodCPULimitByNs(namespace string) string {
	return fmt.Sprintf(
		`sum by (pod, namespace) (kube_pod_container_resource_limits{namespace="%s", resource="cpu"})`,
		promLabelValue(namespace),
	)
}

func promQueryPodMemLimitByNs(namespace string) string {
	return fmt.Sprintf(
		`sum by (pod, namespace) (kube_pod_container_resource_limits{namespace="%s", resource="memory"})`,
		promLabelValue(namespace),
	)
}
