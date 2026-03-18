import { ref, reactive, computed } from 'vue'
import type {
  MetricsStatus,
  ClusterMetricsSnapshot,
  NodeMetricSnapshot,
  PodMetricSnapshot,
  TimeSeriesDataPoint,
} from '../types/metrics'

// Metrics Status
export function useMetricsStatus(clusterName: string) {
  const status = ref<MetricsStatus | null>(null)
  const loading = ref(false)
  const error = ref('')

  async function fetchStatus() {
    try {
      const { GetMetricsStatus } = await import('../../wailsjs/go/main/App')
      status.value = (await GetMetricsStatus()) as unknown as MetricsStatus
    } catch (e: any) {
      error.value = e.message || String(e)
    }
  }

  async function install() {
    try {
      loading.value = true
      error.value = ''
      const { InstallMetricsStack } = await import('../../wailsjs/go/main/App')
      await InstallMetricsStack()
      await fetchStatus()
    } catch (e: any) {
      error.value = e.message || String(e)
    } finally {
      loading.value = false
    }
  }

  async function uninstall() {
    try {
      loading.value = true
      error.value = ''
      const { UninstallMetricsStack } = await import('../../wailsjs/go/main/App')
      await UninstallMetricsStack()
      await fetchStatus()
    } catch (e: any) {
      error.value = e.message || String(e)
    } finally {
      loading.value = false
    }
  }

  return {
    status,
    loading,
    error,
    fetchStatus,
    install,
    uninstall,
  }
}

// Cluster Metrics
export function useClusterMetrics(clusterName: string) {
  const snapshot = ref<ClusterMetricsSnapshot | null>(null)

  async function fetch() {
    try {
      const { GetClusterMetricsSnapshot } = await import('../../wailsjs/go/main/App')
      const data = await GetClusterMetricsSnapshot()
      if (data) {
        snapshot.value = data as unknown as ClusterMetricsSnapshot
      }
    } catch (e: any) {
      console.error('Failed to fetch cluster metrics:', e)
    }
  }

  return { snapshot, fetch }
}

// Node Metrics
export function useNodeMetrics(clusterName: string) {
  const metrics = ref<NodeMetricSnapshot[]>([])

  async function fetch() {
    try {
      const { GetNodeMetrics } = await import('../../wailsjs/go/main/App')
      const data = await GetNodeMetrics()
      if (Array.isArray(data)) {
        metrics.value = data as unknown as NodeMetricSnapshot[]
      }
    } catch (e: any) {
      console.error('Failed to fetch node metrics:', e)
      metrics.value = []
    }
  }

  return { metrics, fetch }
}

// Pod Metrics
export function usePodMetrics(clusterName: string, namespace: string) {
  const metrics = ref<PodMetricSnapshot[]>([])

  async function fetch() {
    try {
      const { GetPodMetrics } = await import('../../wailsjs/go/main/App')
      const data = await GetPodMetrics(namespace)
      if (Array.isArray(data)) {
        metrics.value = data as unknown as PodMetricSnapshot[]
      }
    } catch (e: any) {
      console.error('Failed to fetch pod metrics:', e)
      metrics.value = []
    }
  }

  return { metrics, fetch }
}

// Prometheus Range Queries
export function usePrometheusRange(clusterName: string) {
  async function queryRange(
    query: string,
    startUnix: number,
    endUnix: number,
    step: number
  ): Promise<TimeSeriesDataPoint[]> {
    try {
      const { QueryPrometheusRange } = await import('../../wailsjs/go/main/App')
      const stepStr = `${step}s`
      const result = await QueryPrometheusRange(query, startUnix, endUnix, stepStr)

      if (!result || result.status !== 'success') {
        return []
      }

      // Parse Prometheus range query result
      const data = result.data as any
      if (!data.result || !Array.isArray(data.result) || data.result.length === 0) {
        return []
      }

      const series = data.result[0]
      if (!series.values || !Array.isArray(series.values)) {
        return []
      }

      // Convert [timestamp_string, value_string] to [timestamp_number, value_number]
      return series.values.map((point: [string, string]) => [
        parseInt(point[0], 10),
        parseFloat(point[1]),
      ])
    } catch (e: any) {
      console.error('Failed to query Prometheus:', e)
      return []
    }
  }

  return { queryRange }
}
