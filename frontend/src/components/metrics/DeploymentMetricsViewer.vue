<template>
  <div class="deployment-metrics-viewer">
    <!-- Metric Type Selector -->
    <div class="metrics-selector">
      <button
        v-for="metric in availableMetrics"
        :key="metric.id"
        :class="['metric-btn', { active: selectedMetric === metric.id }]"
        @click="selectedMetric = metric.id"
        :title="metric.title"
      >
        {{ metric.label }}
      </button>
    </div>

    <!-- Chart Display -->
    <div class="metrics-chart-container">
      <MetricsChart
        v-if="currentMetricConfig && !currentMetricConfig.multiLine"
        :key="'single-' + selectedMetric"
        :title="currentMetricConfig.title"
        :unit="currentMetricConfig.unit"
        :color="currentMetricConfig.color || '#3B82F6'"
        :fetch-data="currentMetricConfig.fetchData as any"
        :format-value="currentMetricConfig.formatValue"
        :refresh-interval-ms="30000"
      />
      <MultiLineChart
        v-else-if="currentMetricConfig && currentMetricConfig.multiLine"
        :key="'multi-' + selectedMetric"
        :title="currentMetricConfig.title"
        :unit="currentMetricConfig.unit"
        :fetch-data="currentMetricConfig.fetchData as any"
        :format-value="currentMetricConfig.formatValue"
        :refresh-interval-ms="30000"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import MetricsChart from './MetricsChart.vue'
import MultiLineChart from './MultiLineChart.vue'
import { usePrometheusRange } from '../../composables/useMetrics'

const props = defineProps<{
  clusterName: string
  namespace: string
  deploymentName: string
  selector: string
}>()

const { queryRange } = usePrometheusRange(props.clusterName)

// Parse selector to create label matcher
function getSelectorLabel(): string {
  if (!props.selector) return ''
  // selector format: "app=myapp,version=v1"
  const parts = props.selector.split(',').map(s => s.trim())
  const labels = parts.map(p => {
    const [key, value] = p.split('=').map(s => s.trim())
    return `${key}="${value}"`
  }).join(',')
  return labels
}

// Fetch functions for each metric - query by namespace only, aggregate for all pods
async function fetchCpuUsage(start: number, end: number, step: number) {
  const q = `sum(rate(container_cpu_usage_seconds_total{namespace="${props.namespace}",container!="",container!="POD"}[5m]))`
  return await queryRange(q, start, end, step)
}

async function fetchCpuRequests(start: number, end: number, step: number) {
  const q = `sum(kube_pod_container_resource_requests_cpu_cores{namespace="${props.namespace}"})`
  return await queryRange(q, start, end, step)
}

async function fetchCpuLimits(start: number, end: number, step: number) {
  const q = `sum(kube_pod_container_resource_limits_cpu_cores{namespace="${props.namespace}"})`
  return await queryRange(q, start, end, step)
}

async function fetchCpuCombined(start: number, end: number, step: number) {
  return await fetchCpuUsage(start, end, step)
}

async function fetchMemoryUsage(start: number, end: number, step: number) {
  const q = `sum(container_memory_working_set_bytes{namespace="${props.namespace}",container!="",container!="POD"}) / 1024 / 1024`
  return await queryRange(q, start, end, step)
}

async function fetchMemoryRequests(start: number, end: number, step: number) {
  const q = `sum(kube_pod_container_resource_requests_memory_bytes{namespace="${props.namespace}"}) / 1024 / 1024`
  return await queryRange(q, start, end, step)
}

async function fetchMemoryLimits(start: number, end: number, step: number) {
  const q = `sum(kube_pod_container_resource_limits_memory_bytes{namespace="${props.namespace}"}) / 1024 / 1024`
  return await queryRange(q, start, end, step)
}

async function fetchMemoryCombined(start: number, end: number, step: number) {
  return await fetchMemoryUsage(start, end, step)
}

async function fetchNetworkRx(start: number, end: number, step: number) {
  const q = `sum(rate(container_network_receive_bytes_total{namespace="${props.namespace}"}[5m])) / 1024`
  return await queryRange(q, start, end, step)
}

async function fetchNetworkTx(start: number, end: number, step: number) {
  const q = `sum(rate(container_network_transmit_bytes_total{namespace="${props.namespace}"}[5m])) / 1024`
  return await queryRange(q, start, end, step)
}

async function fetchDiskUsage(start: number, end: number, step: number) {
  const q = `sum(container_fs_usage_bytes{namespace="${props.namespace}"}) / 1024 / 1024`
  return await queryRange(q, start, end, step)
}

async function fetchDiskWrites(start: number, end: number, step: number) {
  const q = `sum(rate(container_fs_writes_bytes_total{namespace="${props.namespace}"}[5m])) / 1024 / 1024`
  return await queryRange(q, start, end, step)
}

async function fetchDiskReads(start: number, end: number, step: number) {
  const q = `sum(rate(container_fs_reads_bytes_total{namespace="${props.namespace}"}[5m])) / 1024 / 1024`
  return await queryRange(q, start, end, step)
}

async function fetchDiskCombined(start: number, end: number, step: number) {
  const usageData = await fetchDiskUsage(start, end, step)
  const writesData = await fetchDiskWrites(start, end, step)
  const readsData = await fetchDiskReads(start, end, step)
  return [
    { label: 'File System Usage', data: usageData },
    { label: 'File System Writes', data: writesData },
    { label: 'File System Reads', data: readsData },
  ]
}

// Formatters
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

function formatNetwork(v: number): string {
  if (v < 0.01) return '0.0'
  if (v < 1) return v.toFixed(2)
  if (v < 1000) return v.toFixed(1)
  if (v < 1000000) return (v / 1000).toFixed(1) + 'M'
  return (v / 1000000).toFixed(1) + 'G'
}

function formatDisk(v: number): string {
  return v.toFixed(2)
}

// Fetch combined network data
async function fetchNetworkCombined(start: number, end: number, step: number) {
  const rxData = await fetchNetworkRx(start, end, step)
  const txData = await fetchNetworkTx(start, end, step)
  return [
    { label: 'Receive', data: rxData },
    { label: 'Transmit', data: txData },
  ]
}

// Available metrics configuration
const availableMetrics = [
  { id: 'cpu', label: 'CPU', title: 'CPU Usage' },
  { id: 'memory', label: 'Memory', title: 'Memory Usage' },
  { id: 'network', label: 'Network', title: 'Network Traffic' },
  { id: 'disk', label: 'Disk I/O', title: 'Disk I/O' },
]

const selectedMetric = ref('cpu')

const metricsConfig: Record<string, any> = {
  cpu: {
    id: 'cpu',
    title: 'CPU Usage',
    unit: 'cores',
    fetchData: fetchCpuCombined,
    formatValue: formatCpu,
    multiLine: false,
  },
  memory: {
    id: 'memory',
    title: 'Memory Usage',
    unit: 'MB',
    fetchData: fetchMemoryCombined,
    formatValue: formatMemoryMB,
    multiLine: false,
  },
  network: {
    id: 'network',
    title: 'Network Traffic',
    unit: 'KB/s',
    fetchData: fetchNetworkCombined,
    formatValue: formatNetwork,
    multiLine: true,
  },
  disk: {
    id: 'disk',
    title: 'Disk I/O',
    unit: 'MB/s',
    fetchData: fetchDiskCombined,
    formatValue: formatDisk,
    multiLine: true,
  },
}

const currentMetricConfig = computed(() => {
  return metricsConfig[selectedMetric.value as keyof typeof metricsConfig]
})
</script>

<style scoped>
.deployment-metrics-viewer {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 8px;
  width: 100%;
}

.metrics-selector {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.metric-btn {
  padding: 6px 12px;
  background: #1e1f35;
  border: 1px solid #3b3d5c;
  border-radius: 4px;
  color: #9ca0b8;
  font-size: 0.85rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
  white-space: nowrap;
}

.metric-btn:hover {
  background: #262740;
  border-color: #5a5c7a;
  color: #c0c2d8;
}

.metric-btn.active {
  background: #3b82f6;
  border-color: #3b82f6;
  color: white;
}

.metrics-chart-container {
  background: #1a1b2e;
  border: 1px solid #2a2b40;
  border-radius: 6px;
  padding: 12px;
  width: 100%;
  min-height: 350px;
}
</style>
