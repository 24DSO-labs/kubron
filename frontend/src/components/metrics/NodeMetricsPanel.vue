<template>
  <div v-if="metricsAvailable">
    <!-- Gauges -->
    <div class="metrics-grid-3">
      <GaugeCard
        title="CPU"
        :percent="nodeData?.cpuPercent ?? 0"
        :used="nodeData?.cpuUsage ?? 0"
        :total="nodeData?.cpuCapacity ?? 0"
        :format="formatCores"
        unit="cores"
      />
      <GaugeCard
        title="Memory"
        :percent="nodeData?.memoryPercent ?? 0"
        :used="nodeData?.memoryUsage ?? 0"
        :total="nodeData?.memoryCapacity ?? 0"
        :format="formatBytes"
      />
      <GaugeCard
        title="Disk"
        :percent="nodeData?.diskPercent ?? 0"
        :used="nodeData?.diskUsage ?? 0"
        :total="nodeData?.diskCapacity ?? 0"
        :format="formatBytes"
      />
    </div>

    <!-- Charts -->
    <div class="metrics-grid-2">
      <MetricsChart
        title="CPU Usage"
        unit="%"
        color="#3B82F6"
        :max-y="100"
        :fetch-data="fetchNodeCpu"
        :format-value="(v: number) => v.toFixed(1)"
      />
      <MetricsChart
        title="Memory Usage"
        unit="%"
        color="#8B5CF6"
        :max-y="100"
        :fetch-data="fetchNodeMemory"
        :format-value="(v: number) => v.toFixed(1)"
      />
      <MetricsChart
        title="Disk Usage"
        unit="%"
        color="#F59E0B"
        :max-y="100"
        :fetch-data="fetchNodeDisk"
        :format-value="(v: number) => v.toFixed(1)"
      />
      <MetricsChart
        title="Network I/O"
        unit="MB/s"
        color="#10B981"
        :fetch-data="fetchNodeNetwork"
        :format-value="(v: number) => v.toFixed(2)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onBeforeUnmount, computed } from 'vue'
import GaugeCard from './GaugeCard.vue'
import MetricsChart from './MetricsChart.vue'
import { formatBytes, formatCores } from '../overview/format'
import {
  useMetricsStatus,
  useNodeMetrics,
  usePrometheusRange,
} from '../../composables/useMetrics'

const props = defineProps<{
  clusterName: string
  nodeName: string
}>()

const { status, fetchStatus } = useMetricsStatus(props.clusterName)
const { metrics: allNodes, fetch: fetchNodes } = useNodeMetrics(props.clusterName)
const { queryRange } = usePrometheusRange(props.clusterName)

let refreshInterval: ReturnType<typeof setInterval> | null = null

const metricsAvailable = computed(() => status.value?.status === 'installed')

const nodeData = computed(() => {
  return allNodes.value.find((n: any) => n.name === props.nodeName) || null
})

/**
 * Селектор ноды. У node-exporter из manifests/metrics-stack.yaml instance — это IP пода,
 * а имя Node лежит в метке kubernetes_node (должна совпадать с promNodeLabel в Go).
 */
const nodeSel = computed(() => {
  const v = props.nodeName.replace(/\\/g, '\\\\').replace(/"/g, '\\"')
  return `kubernetes_node="${v}"`
})

// Физические интерфейсы: виртуальные (veth, мосты, оверлей) дублируют трафик
const NET_DEVICES = 'device!~"lo|veth.*|docker.*|cni.*|flannel.*|cali.*|br-.*"'

async function fetchNodeCpu(
  start: number, end: number, step: number
): Promise<[number, number][]> {
  const q = `100 - (avg(rate(node_cpu_seconds_total{mode="idle",${nodeSel.value}}[5m])) * 100)`
  return await queryRange(q, start, end, step)
}

async function fetchNodeMemory(
  start: number, end: number, step: number
): Promise<[number, number][]> {
  const q = `(1 - (sum(node_memory_MemAvailable_bytes{${nodeSel.value}}) / sum(node_memory_MemTotal_bytes{${nodeSel.value}}))) * 100`
  return await queryRange(q, start, end, step)
}

async function fetchNodeDisk(
  start: number, end: number, step: number
): Promise<[number, number][]> {
  const sel = `${nodeSel.value},mountpoint="/",fstype!="tmpfs"`
  const q = `(1 - (max(node_filesystem_avail_bytes{${sel}}) / max(node_filesystem_size_bytes{${sel}}))) * 100`
  return await queryRange(q, start, end, step)
}

async function fetchNodeNetwork(
  start: number, end: number, step: number
): Promise<[number, number][]> {
  const sel = `${nodeSel.value},${NET_DEVICES}`
  const q = `(sum(rate(node_network_receive_bytes_total{${sel}}[5m])) + sum(rate(node_network_transmit_bytes_total{${sel}}[5m]))) / 1024 / 1024`
  return await queryRange(q, start, end, step)
}

onMounted(async () => {
  await fetchStatus()
  if (metricsAvailable.value) {
    await fetchNodes()
    // Auto-refresh every 10 seconds
    refreshInterval = setInterval(() => {
      fetchNodes()
    }, 10000)
  }
})

onBeforeUnmount(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
})
</script>