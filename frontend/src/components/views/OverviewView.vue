<template>
  <div class="flex-1 overflow-auto bg-gray-900">
    <ContentHeader title="Overview" icon="" />
    <div class="p-6">
      <!-- Overview Cards for Cluster Overview (above charts) -->
      <div class="overview-grid mb-6" v-if="clusterOverview">
        <div class="overview-card">
          <div class="card-icon"></div>
          <div class="card-info">
            <div class="card-value">{{ clusterOverview.readyNodes }}/{{ clusterOverview.totalNodes }}</div>
            <div class="card-label">Nodes Ready</div>
          </div>
        </div>
        <div class="overview-card">
          <div class="card-icon"></div>
          <div class="card-info">
            <div class="card-value">{{ clusterOverview.runningPods }}/{{ clusterOverview.totalPods }}</div>
            <div class="card-label">Pods Running</div>
          </div>
        </div>
        <div class="overview-card">
          <div class="card-icon"></div>
          <div class="card-info">
            <div class="card-value">{{ clusterOverview.readyDeployments }}/{{ clusterOverview.totalDeployments }}</div>
            <div class="card-label">Deployments Ready</div>
          </div>
        </div>
        <div class="overview-card">
          <div class="card-icon"></div>
          <div class="card-info">
            <div class="card-value">{{ clusterOverview.totalServices }}</div>
            <div class="card-label">Services</div>
          </div>
        </div>
        <div class="overview-card">
          <div class="card-icon"></div>
          <div class="card-info">
            <div class="card-value">{{ clusterOverview.totalNamespaces }}</div>
            <div class="card-label">Namespaces</div>
          </div>
        </div>
        <div class="overview-card">
          <div class="card-icon"></div>
          <div class="card-info">
            <div class="card-value">{{ clusterOverview.warningEvents }}/{{ clusterOverview.totalEvents }}</div>
            <div class="card-label">Warning Events</div>
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

      <div v-else>
        <div v-if="loading" class="bg-gray-800 rounded-lg shadow p-6 text-center border border-gray-700">
          <p class="text-gray-300">Loading metrics...</p>
        </div>
        <div v-else-if="error" class="bg-gray-800 rounded-lg shadow p-6 text-center border border-red-700">
          <p class="text-red-400">{{ error }}</p>
        </div>
        <template v-else-if="nodeMetrics">
          <!-- Layout: Chart on left, Cards on right in a row -->
          <div class="space-y-6">
            <!-- Chart section -->
            <div class="bg-gray-800 rounded-lg shadow p-6 border border-gray-700">
              <h3 class="text-lg font-semibold text-white mb-4">Memory Usage Over Time</h3>
              <div class="h-80 bg-gray-900 rounded flex items-center justify-center">
                <p class="text-gray-400">Time series chart</p>
              </div>
            </div>
            
            <!-- Cards section in a row -->
            <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
              <DonutCard title="CPU" :usage="nodeMetrics.cpu?.usage || 0" :requests="nodeMetrics.cpu?.requests || 0" :limits="nodeMetrics.cpu?.limits || 0" :allocatable="nodeMetrics.cpu?.allocatable || 0" :capacity="nodeMetrics.cpu?.capacity || 0" unit="cores" />
              <DonutCard title="Memory" :usage="nodeMetrics.memory?.usage || 0" :requests="nodeMetrics.memory?.requests || 0" :limits="nodeMetrics.memory?.limits || 0" :allocatable="nodeMetrics.memory?.allocatable || 0" :capacity="nodeMetrics.memory?.capacity || 0" unit="GB" />
              <DonutCard title="Pods" :usage="nodeMetrics.pods?.usage || 0" :requests="nodeMetrics.pods?.requests || 0" :limits="nodeMetrics.pods?.limits || 0" :allocatable="nodeMetrics.pods?.allocatable || 0" :capacity="nodeMetrics.pods?.capacity || 0" unit="count" />
            </div>
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
import { ref, computed, onMounted } from 'vue'
import ContentHeader from '../ContentHeader.vue'
import ClusterMetricsPanel from '../metrics/ClusterMetricsPanel.vue'
import DonutCard from '../metrics/DonutCard.vue'
import { useMetrics } from '../../composables/useMetrics'
import type { NodeMetricsByRole } from '../../types/metrics'

interface Props {
  clusterName?: string
}

withDefaults(defineProps<Props>(), {
  clusterName: '',
})

const selectedNodeRole = ref('')
const selectedTimeRange = ref('1h')
const nodeMetrics = ref<NodeMetricsByRole | null>(null)
const loading = ref(false)
const error = ref('')

const { getNodeMetricsByRole } = useMetrics()

interface ClusterOverviewMetrics {
  cpuUsage?: number
  memoryUsage?: number
  podCount?: number
}

const clusterSnapshot = ref<any>(null)
const clusterOverview = ref<any>(null)

const onSnapshotUpdate = (snapshot: any) => {
  clusterSnapshot.value = snapshot
}

const loadClusterOverview = async () => {
  try {
    const { GetClusterOverview } = await import('../../../wailsjs/go/main/App')
    const overview = await GetClusterOverview()
    clusterOverview.value = overview
  } catch (err) {
    console.error('Error loading cluster overview:', err)
  }
}

onMounted(() => {
  loadClusterOverview()
  // Refresh every 10 seconds
  setInterval(loadClusterOverview, 10000)
})

// Computed percentages for gauge cards
const cpuPercent = computed(() => {
  if (!nodeMetrics.value?.cpu || nodeMetrics.value.cpu.capacity === 0) return 0
  return Math.round((nodeMetrics.value.cpu.usage / nodeMetrics.value.cpu.capacity) * 100)
})

const memoryPercent = computed(() => {
  if (!nodeMetrics.value?.memory || nodeMetrics.value.memory.capacity === 0) return 0
  return Math.round((nodeMetrics.value.memory.usage / nodeMetrics.value.memory.capacity) * 100)
})

const podsPercent = computed(() => {
  if (!nodeMetrics.value?.pods || nodeMetrics.value.pods.capacity === 0) return 0
  return Math.round((nodeMetrics.value.pods.usage / nodeMetrics.value.pods.capacity) * 100)
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
    console.log('CPU Usage:', metrics?.cpu?.usage)
    console.log('Memory Usage:', metrics?.memory?.usage)
    console.log('Pods Usage:', metrics?.pods?.usage)
    nodeMetrics.value = metrics as unknown as NodeMetricsByRole
    console.log('nodeMetrics.value after setting:', nodeMetrics.value)
  } catch (err: any) {
    error.value = err.message || 'Failed to load metrics'
    console.error('Error loading metrics:', err)
    nodeMetrics.value = null
  } finally {
    loading.value = false
  }
}
</script>
