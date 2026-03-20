<template>
  <div class="flex-1 overflow-auto bg-gray-900">
    <ContentHeader title="Overview" icon="chart-line" />
    <div class="p-6">
       <!-- Overview Cards for Cluster Overview (above charts) -->
        <div class="overview-grid mb-6" v-if="clusterSnapshot">
          <div class="overview-card">
            <div class="card-icon">💾</div>
            <div class="card-info">
              <div class="card-value">{{ clusterSnapshot.cpuUsage?.toFixed(2) || '0' }}/{{ clusterSnapshot.cpuCapacity?.toFixed(2) || '0' }}</div>
              <div class="card-label">CPU Usage</div>
            </div>
          </div>
          <div class="overview-card">
            <div class="card-icon">🧠</div>
            <div class="card-info">
              <div class="card-value">{{ clusterSnapshot.memoryUsage?.toFixed(2) || '0' }}/{{ clusterSnapshot.memoryCapacity?.toFixed(2) || '0' }}</div>
              <div class="card-label">Memory Usage</div>
            </div>
          </div>
          <div class="overview-card">
            <div class="card-icon">📦</div>
            <div class="card-info">
              <div class="card-value">{{ clusterSnapshot.podCount || '0' }}/{{ clusterSnapshot.podCapacity || '0' }}</div>
              <div class="card-label">Pods</div>
            </div>
          </div>
        </div>
      <!-- Dropdown для выбора Master/Worker nodes -->
      <div class="mb-6 flex items-center gap-4 bg-gray-800 rounded-lg shadow p-4 border border-gray-700">
        <div class="flex items-center gap-2">
          <label class="text-sm font-semibold text-gray-200">Select Nodes:</label>
          <select
            v-model="selectedNodeRole"
            @change="loadNodeMetrics"
            class="px-4 py-2 border border-gray-600 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 bg-gray-700 text-white"
          >
            <option value="">Cluster Overview</option>
            <option value="master">Master Nodes</option>
            <option value="worker">Worker Nodes</option>
          </select>
        </div>

        <div class="flex items-center gap-2 ml-auto">
          <span class="text-sm text-gray-300">Time Range:</span>
          <select
            v-model="selectedTimeRange"
            class="px-4 py-2 border border-gray-600 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 bg-gray-700 text-white"
          >
            <option value="1h">1h</option>
            <option value="6h">6h</option>
            <option value="24h">24h</option>
            <option value="7d">7d</option>
          </select>
        </div>
      </div>

      <div v-if="!selectedNodeRole">
       
        <ClusterMetricsPanel :clusterName="clusterName" @update:snapshot="onSnapshotUpdate" />
      </div>

      <div v-else class="space-y-6">
        <div v-if="loading" class="bg-gray-800 rounded-lg shadow p-6 text-center border border-gray-700">
          <p class="text-gray-300">Loading metrics...</p>
        </div>
        <div v-else-if="error" class="bg-gray-800 rounded-lg shadow p-6 text-center border border-red-700">
          <p class="text-red-400">{{ error }}</p>
        </div>
        <template v-else-if="nodeMetrics">
         

          <!-- Donut Cards Row -->
          <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
            <DonutCard title="CPU" :usage="nodeMetrics.CPU?.Usage || 0" :requests="nodeMetrics.CPU?.Requests || 0" :limits="nodeMetrics.CPU?.Limits || 0" :allocatable="nodeMetrics.CPU?.Allocatable || 0" :capacity="nodeMetrics.CPU?.Capacity || 0" unit="cores" />
            <DonutCard title="Memory" :usage="nodeMetrics.Memory?.Usage || 0" :requests="nodeMetrics.Memory?.Requests || 0" :limits="nodeMetrics.Memory?.Limits || 0" :allocatable="nodeMetrics.Memory?.Allocatable || 0" :capacity="nodeMetrics.Memory?.Capacity || 0" unit="GB" />
            <DonutCard title="Pods" :usage="nodeMetrics.Pods?.Usage || 0" :requests="nodeMetrics.Pods?.Requests || 0" :limits="nodeMetrics.Pods?.Limits || 0" :allocatable="nodeMetrics.Pods?.Allocatable || 0" :capacity="nodeMetrics.Pods?.Capacity || 0" unit="count" />
          </div>
        </template>
        <div v-else class="bg-gray-800 rounded-lg shadow p-6 text-center border border-gray-700">
          <p class="text-gray-300">No metrics available</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import ContentHeader from '../ContentHeader.vue'
import ClusterMetricsPanel from '../metrics/ClusterMetricsPanel.vue'
import DonutCard from '../metrics/DonutCard.vue'
import { useMetrics } from '../../composables/useMetrics'

interface ResourceMetric {
  Usage: number
  Requests: number
  Limits: number
  Allocatable: number
  Capacity: number
}

interface NodeMetricsData {
  role?: string
  Role?: string
  CPU: ResourceMetric
  Memory: ResourceMetric
  Pods: ResourceMetric
}

interface Props {
  clusterName?: string
}

withDefaults(defineProps<Props>(), {
  clusterName: '',
})

const selectedNodeRole = ref('')
const selectedTimeRange = ref('1h')
const nodeMetrics = ref<NodeMetricsData | null>(null)
const loading = ref(false)
const error = ref('')

const { getNodeMetricsByRole } = useMetrics()

interface ClusterOverviewMetrics {
  cpuUsage?: number
  memoryUsage?: number
  podCount?: number
}

const clusterSnapshot = ref<any>(null)

const onSnapshotUpdate = (snapshot: any) => {
  clusterSnapshot.value = snapshot
}

// Computed percentages for gauge cards
const cpuPercent = computed(() => {
  if (!nodeMetrics.value?.CPU || nodeMetrics.value.CPU.Capacity === 0) return 0
  return Math.round((nodeMetrics.value.CPU.Usage / nodeMetrics.value.CPU.Capacity) * 100)
})

const memoryPercent = computed(() => {
  if (!nodeMetrics.value?.Memory || nodeMetrics.value.Memory.Capacity === 0) return 0
  return Math.round((nodeMetrics.value.Memory.Usage / nodeMetrics.value.Memory.Capacity) * 100)
})

const podsPercent = computed(() => {
  if (!nodeMetrics.value?.Pods || nodeMetrics.value.Pods.Capacity === 0) return 0
  return Math.round((nodeMetrics.value.Pods.Usage / nodeMetrics.value.Pods.Capacity) * 100)
})

const loadNodeMetrics = async () => {
  if (!selectedNodeRole.value) {
    nodeMetrics.value = null
    error.value = ''
    return
  }
  loading.value = true
  error.value = ''
  try {
    const metrics = await getNodeMetricsByRole(selectedNodeRole.value as 'master' | 'worker')
    console.log('Loaded metrics:', metrics)
    nodeMetrics.value = metrics as unknown as NodeMetricsData
  } catch (err: any) {
    error.value = err.message || 'Failed to load metrics'
    console.error('Error loading metrics:', err)
    nodeMetrics.value = null
  } finally {
    loading.value = false
  }
}
</script>
