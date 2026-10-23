/**
 * Потребление CPU / памяти подом: форматирование и слияние метрик со списком подов.
 * Чистая логика без Wails (тесты: podUsage.test.ts).
 *
 * Единицы приходят из Prometheus (internal/k8s/metrics_queries.go): CPU в милликорах, память в байтах.
 */
import type { ColumnDef } from './types'
import type { PodMetricsInfo } from '../types'

export function formatCpu(cpuStr: string | undefined): string {
  if (!cpuStr) return '—'
  const m = parseFloat(cpuStr) // millicores
  if (isNaN(m)) return '—'
  if (m >= 1000) return (m / 1000).toFixed(2) // ядра
  return m.toFixed(1) + 'm'
}

export function formatMemory(memStr: string | undefined): string {
  if (!memStr) return '—'
  const bytes = parseFloat(memStr)
  if (isNaN(bytes)) return '—'
  return (bytes / 1024 / 1024).toFixed(1) + 'MiB'
}

/** Поля пода после mergePodMetrics: готовый текст для ячейки и сырые числа для сортировки */
export interface PodUsage {
  cpu: string
  memory: string
  /** Милликоры; null — метрик по поду нет */
  cpuMillicores: number | null
  /** Байты; null — метрик по поду нет */
  memoryBytes: number | null
}

const toNumber = (v: string | undefined): number | null => {
  if (!v) return null
  const n = parseFloat(v)
  return Number.isNaN(n) ? null : n
}

export function mergePodMetrics<P extends { name: string; namespace: string }>(
  list: P[],
  metrics: PodMetricsInfo[],
): (P & PodUsage)[] {
  const byPod = new Map(metrics.map(m => [`${m.namespace}/${m.name}`, m]))
  return list.map(pod => {
    const metric = byPod.get(`${pod.namespace}/${pod.name}`)
    return {
      ...pod,
      cpu: formatCpu(metric?.totalCpu),
      memory: formatMemory(metric?.totalMem),
      cpuMillicores: toNumber(metric?.totalCpu),
      memoryBytes: toNumber(metric?.totalMem),
    }
  })
}

// p.cpu / p.memory уже отформатированы в mergePodMetrics: повторно их форматировать нельзя
// («1.50» ядра стало бы «1.5m»), а сортировать нужно по сырым числам, не по тексту.
export const podCpuCol: ColumnDef = {
  key: 'cpu', label: 'CPU', className: 'mono metric-cell', animated: true, width: 100,
  formatter: (p: any) => p.cpu ?? '—',
  sortValue: (p: any) => p.cpuMillicores,
}

export const podMemoryCol: ColumnDef = {
  key: 'memory', label: 'MEMORY', className: 'mono metric-cell', animated: true, width: 110,
  formatter: (p: any) => p.memory ?? '—',
  sortValue: (p: any) => p.memoryBytes,
}
