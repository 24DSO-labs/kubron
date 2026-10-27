// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"fmt"
	"strings"
)

// Метка, которую Prometheus-конфиг из manifests/metrics-stack.yaml ставит на метрики
// node-exporter: имя Node (instance там — IP пода, по нему ноду не найти).
const promNodeLabel = "kubernetes_node"

// promLabelValue экранирует значение для подстановки в строковый литерал PromQL.
func promLabelValue(v string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`).Replace(v)
}

// Метрики по нодам: одно значение на ноду (метка kubernetes_node)

func promQueryNodeCPUUsedCores() string {
	return fmt.Sprintf(`sum by (%[1]s) (rate(node_cpu_seconds_total{mode!="idle"}[5m]))`, promNodeLabel)
}

func promQueryNodeCPUCores() string {
	return fmt.Sprintf(`count by (%[1]s) (node_cpu_seconds_total{mode="idle"})`, promNodeLabel)
}

func promQueryNodeMemTotal() string {
	return fmt.Sprintf(`sum by (%[1]s) (node_memory_MemTotal_bytes)`, promNodeLabel)
}

func promQueryNodeMemAvailable() string {
	return fmt.Sprintf(`sum by (%[1]s) (node_memory_MemAvailable_bytes)`, promNodeLabel)
}

// max: корневой раздел может быть виден несколькими устройствами (bind-mount), берём одно
func promQueryNodeDiskSize() string {
	return fmt.Sprintf(`max by (%[1]s) (node_filesystem_size_bytes{mountpoint="/",fstype!="tmpfs"})`, promNodeLabel)
}

func promQueryNodeDiskAvail() string {
	return fmt.Sprintf(`max by (%[1]s) (node_filesystem_avail_bytes{mountpoint="/",fstype!="tmpfs"})`, promNodeLabel)
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

// promQueryPodCPUByNs — CPU подов набора namespace: «все» без фильтра, один — точное совпадение,
// несколько — регулярное выражение (см. promNamespaceMatcher).
func promQueryPodCPUByNs(scope nsScope) string {
	if scope.all() {
		return `sum by (pod, namespace) (rate(container_cpu_usage_seconds_total{container!="", container!="POD"}[5m])) * 1000`
	}
	return fmt.Sprintf(
		`sum by (pod, namespace) (rate(container_cpu_usage_seconds_total{container!="", container!="POD", %s}[5m])) * 1000`,
		promNamespaceMatcher(scope),
	)
}

// promQueryPodMemByNs — память подов набора namespace (семантика набора — как у promQueryPodCPUByNs).
func promQueryPodMemByNs(scope nsScope) string {
	if scope.all() {
		return `sum by (pod, namespace) (container_memory_working_set_bytes{container!="", container!="POD"})`
	}
	return fmt.Sprintf(
		`sum by (pod, namespace) (container_memory_working_set_bytes{%s, container!="", container!="POD"})`,
		promNamespaceMatcher(scope),
	)
}
