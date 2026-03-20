<template>
  <div v-if="metricsAvailable">
    <!-- Gauges row -->
    <div class="grid grid-cols-3 gap-4 mb-6" style="grid-template-columns: repeat(3, minmax(0, 1fr));">
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
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-3">
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
  <div v-else-if="statusChecked" class="bg-gradient-to-r from-gray-800 to-gray-900 rounded-xl border border-gray-700/50 p-5 shadow-lg">
    <div class="flex items-center gap-4">
      <div class="w-14 h-14 rounded-lg bg-gradient-to-br from-purple-500/20 to-blue-500/20 flex items-center justify-center flex-shrink-0">
        <svg class="w-7 h-7 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
        </svg>
      </div>
      <div class="flex-1">
        <p class="text-sm font-semibold text-gray-200">Метрики недоступны</p>
        <p class="text-xs text-gray-400 mt-1">
          Установите стек метрик в Настройках кластера для включения графиков мониторинга.
        </p>
      </div>
      <button @click="$emit('openSettings')"
        class="ml-auto flex-shrink-0 px-4 py-2 bg-gradient-to-r from-blue-600 to-blue-700 hover:from-blue-500 hover:to-blue-600 text-white text-sm font-semibold rounded-lg transition-all duration-200 shadow-md hover:shadow-lg whitespace-nowrap">
        Установить
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import GaugeCard from './GaugeCard.vue'
import MetricsChart from './MetricsChart.vue'
import { useClusterMetrics, usePrometheusRange, useMetricsStatus } from '../../composables/useMetrics'

const props = defineProps<{
  clusterName: string
}>()

const emit = defineEmits<{
  openSettings: []
  'update:snapshot': [snapshot: any]
}>()

const { status, fetchStatus } = useMetricsStatus(props.clusterName)
const { snapshot, fetch: fetchMetrics } = useClusterMetrics(props.clusterName)
const { queryRange } = usePrometheusRange(props.clusterName)

const statusChecked = ref(false)
let refreshInterval: ReturnType<typeof setInterval> | null = null

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

  // Fetch metrics immediately
  if (metricsAvailable.value) {
    await fetchMetrics()
    // Emit snapshot
    if (snapshot.value) {
      emit('update:snapshot', snapshot.value)
    }

    // Set up auto-refresh every 10 seconds
    refreshInterval = setInterval(() => {
      fetchMetrics()
      // Emit updated snapshot
      if (snapshot.value) {
        emit('update:snapshot', snapshot.value)
      }
    }, 10000)
  }
})

onUnmounted(() => {
  if (refreshInterval) {
    clearInterval(refreshInterval)
  }
})
</script>
