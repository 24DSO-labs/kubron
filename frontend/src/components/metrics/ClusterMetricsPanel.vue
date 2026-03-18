<template>
  <div v-if="metricsAvailable">
    <!-- Gauges row -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
      <GaugeCard
        title="CPU Usage"
        :percent="snapshot?.cpuPercent ?? 0"
        :used="snapshot?.cpuUsage ?? 0"
        :total="snapshot?.cpuCapacity ?? 0"
        unit="cores"
      />
      <GaugeCard
        title="Memory Usage"
        :percent="snapshot?.memoryPercent ?? 0"
        :used="snapshot?.memoryUsage ?? 0"
        :total="snapshot?.memoryCapacity ?? 0"
        unit="bytes"
      />
      <GaugeCard
        title="Pods"
        :percent="snapshot?.podPercent ?? 0"
        :used="snapshot?.podCount ?? 0"
        :total="snapshot?.podCapacity ?? 0"
        unit="count"
      />
    </div>

    <!-- Charts row -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <MetricsChart
        title="Cluster CPU Usage"
        unit="%"
        color="#3B82F6"
        :max-y="100"
        :fetch-data="fetchCpuData"
        :format-value="(v: number) => v.toFixed(1)"
      />
      <MetricsChart
        title="Cluster Memory Usage"
        unit="%"
        color="#8B5CF6"
        :max-y="100"
        :fetch-data="fetchMemoryData"
        :format-value="(v: number) => v.toFixed(1)"
      />
    </div>
  </div>

  <!-- Not installed hint -->
  <div v-else-if="statusChecked" class="bg-gray-800 rounded-lg border border-gray-700 p-6">
    <div class="flex items-center gap-4">
      <div class="w-12 h-12 rounded-lg bg-gray-700 flex items-center justify-center">
        <svg class="w-6 h-6 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
        </svg>
      </div>
      <div>
        <p class="text-sm text-gray-300 font-medium">Metrics not available</p>
        <p class="text-xs text-gray-500 mt-1">
          Install the metrics stack in Cluster Settings to enable monitoring graphs.
        </p>
      </div>
      <button @click="$emit('openSettings')"
        class="ml-auto px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white
               text-sm rounded-lg transition-colors whitespace-nowrap">
        Install Metrics
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import GaugeCard from './GaugeCard.vue'
import MetricsChart from './MetricsChart.vue'
import { useClusterMetrics, usePrometheusRange, useMetricsStatus } from '../../composables/useMetrics'

const props = defineProps<{
  clusterName: string
}>()

defineEmits<{
  openSettings: []
}>()

const { status, fetchStatus } = useMetricsStatus(props.clusterName)
const { snapshot } = useClusterMetrics(props.clusterName)
const { queryRange } = usePrometheusRange(props.clusterName)

const statusChecked = ref(false)

const metricsAvailable = computed(() => {
  return status.value?.status === 'installed'
})

async function fetchCpuData(
  startUnix: number, endUnix: number, step: number
): Promise<[number, number][]> {
  return await queryRange(
    '100 - (avg(rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)',
    startUnix, endUnix, step
  )
}

async function fetchMemoryData(
  startUnix: number, endUnix: number, step: number
): Promise<[number, number][]> {
  return await queryRange(
    '(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100',
    startUnix, endUnix, step
  )
}

onMounted(async () => {
  await fetchStatus()
  statusChecked.value = true
})
</script>
