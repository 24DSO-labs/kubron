import { ref, reactive, computed } from 'vue'
import type {
import { callApp } from '../resources/wails'
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
      status.value = (await callApp('GetMetricsStatus')) as unknown as MetricsStatus
    } catch (e: any) {
      error.value = e.message || String(e)
    }
  }

  async function install() {
    try {
      loading.value = true
      error.value = ''
      await callApp('InstallMetricsStack')
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
      await callApp('UninstallMetricsStack')
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
  const loading = ref(false)

  async function fetch() {
    loading.value = true
    try {
      const data = await callApp('GetClusterMetricsSnapshot')
      if (data) {
        snapshot.value = data as unknown as ClusterMetricsSnapshot
      }
    } catch (e: any) {
      console.error('Failed to fetch cluster metrics:', e)
    } finally {
      loading.value = false
    }
  }

  return { snapshot, loading, fetch }
}

// Node Metrics
export function useNodeMetrics(clusterName: string) {
  const metrics = ref<NodeMetricSnapshot[]>([])

  async function fetch() {
    try {
      const data = await callApp('GetNodeMetrics')
      if (Array.isArray(data)) {
        metrics.value = (data as any[]).map((item: any) => ({
          name: item.name || '',
          cpuUsage: parseFloat(item.cpuUsage) || 0,
          cpuPercent: item.cpuPct || 0,
          cpuCapacity: 100,
          memoryUsage: parseFloat(item.memUsage) || 0,
          memoryPercent: item.memPct || 0,
          memoryCapacity: 100,
          diskPercent: 0,
          diskUsage: 0,
          diskCapacity: 100,
        }))
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
      const data = await callApp('GetPodMetrics', namespace)

      // Backend currently returns only CPU as string (TotalCPU) and uses field names
      // that may not match PodMetricSnapshot exactly. Normalize here to avoid 0/undefined in UI.
      if (Array.isArray(data)) {
        metrics.value = (data as any[]).map((item: any) => ({
          name: item.name ?? item.Name ?? '',
          namespace: item.namespace ?? item.Namespace ?? '',
          cpuUsage: parseFloat(item.cpuUsage ?? item.totalCPU ?? item.TotalCPU ?? '0') || 0,
          // Backend doesn't provide these yet; keep 0 so UI can still render charts (range queries)
          cpuPercent: parseFloat(item.cpuPercent ?? item.CPUPercent ?? '0') || 0,
          memoryUsage: parseFloat(item.memoryUsage ?? item.MemoryUsage ?? '0') || 0,
          memoryPercent: parseFloat(item.memoryPercent ?? item.MemoryPercent ?? '0') || 0,
          memoryTotal: parseFloat(item.memoryTotal ?? item.MemoryTotal ?? '0') || 0,
        })) as unknown as PodMetricSnapshot[]
      } else {
        metrics.value = []
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
      const stepStr = `${step}s`
      const result = await callApp('QueryPrometheusRange', query, startUnix, endUnix, stepStr)

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
