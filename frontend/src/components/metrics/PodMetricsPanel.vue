<template>
  <div v-if="metricsAvailable">
    <!-- Gauges -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
      <GaugeCard
        title="CPU"
        :percent="podData?.cpuPercent ?? 0"
        :used="podData?.cpuUsage ?? 0"
        :total="1"
        unit="cores"
        subtitle="of requested/limit"
      />
      <GaugeCard
        title="Memory"
        :percent="podData?.memoryPercent ?? 0"
        :used="podData?.memoryUsage ?? 0"
        :total="memoryTotal"
        unit="bytes"
        subtitle="of limit"
      />
    </div>

    <!-- Charts -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <MetricsChart
        title="CPU Usage"
        unit="cores"
        color="#3B82F6"
        :fetch-data="fetchPodCpu"
        :format-value="formatCpu"
      />
      <MetricsChart
        title="Memory Usage"
        unit="MB"
        color="#8B5CF6"
        :fetch-data="fetchPodMemory"
        :format-value="formatMemoryMB"
      />
      <MetricsChart
        title="Network Receive"
        unit="KB/s"
        color="#10B981"
        :fetch-data="fetchPodNetRx"
        :format-value="(v: number) => v.toFixed(1)"
      />
      <MetricsChart
        title="Network Transmit"
        unit="KB/s"
        color="#F59E0B"
        :fetch-data="fetchPodNetTx"
        :format-value="(v: number) => v.toFixed(1)"
      />
    </div>
  </div>

  <!-- Not available -->
  <div v-else-if="statusChecked"
    class="bg-gray-800 rounded-lg border border-gray-700 p-5">
    <div class="flex items-center gap-3">
      <svg class="w-5 h-5 text-gray-500" fill="none" stroke="currentColor"
        viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
          d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
      </svg>
      <p class="text-sm text-gray-500">
        Install metrics stack in Cluster Settings to see pod metrics.
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import GaugeCard from './GaugeCard.vue'
import MetricsChart from './MetricsChart.vue'
import {
  useMetricsStatus,
  usePodMetrics,
  usePrometheusRange,
} from '../../composables/useMetrics'

const props = defineProps<{
  clusterName: string
  namespace: string
  podName: string
}>()

const { status, fetchStatus } = useMetricsStatus(props.clusterName)
const { metrics: allPods, fetch: fetchPods } = usePodMetrics(props.clusterName, props.namespace)
const { queryRange } = usePrometheusRange(props.clusterName)

const statusChecked = ref(false)

const metricsAvailable = computed(() => {
  return status.value?.status === 'installed'
})

const podData = computed(() => {
  return allPods.value.find((p: any) => p.name === props.podName) || null
})

const memoryTotal = computed(() => {
  const pd = podData.value
  if (!pd || !pd.memoryPercent || pd.memoryPercent === 0) return 0
  return pd.memoryUsage / (pd.memoryPercent / 100)
})

// --- PromQL queries ---

async function fetchPodCpu(
  start: number, end: number, step: number
): Promise<[number, number][]> {
  const q = `sum(rate(container_cpu_usage_seconds_total{namespace="${props.namespace}",pod="${props.podName}",container!="",container!="POD"}[5m]))`
  return await queryRange(q, start, end, step)
}

async function fetchPodMemory(
  start: number, end: number, step: number
): Promise<[number, number][]> {
  const q = `sum(container_memory_working_set_bytes{namespace="${props.namespace}",pod="${props.podName}",container!="",container!="POD"}) / 1024 / 1024`
  return await queryRange(q, start, end, step)
}

async function fetchPodNetRx(
  start: number, end: number, step: number
): Promise<[number, number][]> {
  const q = `sum(rate(container_network_receive_bytes_total{namespace="${props.namespace}",pod="${props.podName}"}[5m])) / 1024`
  return await queryRange(q, start, end, step)
}

async function fetchPodNetTx(
  start: number, end: number, step: number
): Promise<[number, number][]> {
  const q = `sum(rate(container_network_transmit_bytes_total{namespace="${props.namespace}",pod="${props.podName}"}[5m])) / 1024`
  return await queryRange(q, start, end, step)
}

// --- Formatters ---

function formatCpu(v: number): string {
  if (v < 0.001) return '0'
  if (v < 1) return (v * 1000).toFixed(0) + 'm'
  return v.toFixed(2)
}

function formatMemoryMB(v: number): string {
  if (v < 1) return (v * 1024).toFixed(0) + ' KB'
  if (v >= 1024) return (v / 1024).toFixed(2) + ' GB'
  return v.toFixed(1) + ' MB'
}

onMounted(async () => {
  await fetchStatus()
  statusChecked.value = true
  if (metricsAvailable.value) {
    await fetchPods()
  }
})
</script>
