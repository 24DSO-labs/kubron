<template>
  <div class="pvc-metrics">
    <MultiLineChart
      title="Volume usage"
      unit=""
      :fetch-data="fetchData"
      :format-value="formatSize"
      :refresh-interval-ms="30000"
    />
  </div>
</template>

<script setup lang="ts">
import MultiLineChart from './MultiLineChart.vue'
import { usePrometheusRange } from '../../composables/useMetrics'

/**
 * Метрики набора подов.
 *  - с ownerKind/ownerRegex: поды одного владельца (Deployment, DaemonSet, ...), выбираются через
 *    kube_pod_owner (kube-state-metrics), а не по имени пода: имя пода обрезается Kubernetes
 *    для длинных имён ресурсов, владелец указан всегда;
 *  - без них: все поды namespace.
 */
const props = defineProps<{
  clusterName: string
  namespace: string
  /** kube_pod_owner.owner_kind: ReplicaSet | DaemonSet | StatefulSet | Job */
  ownerKind?: string
  /** regex для owner_name (в Prometheus он всегда с якорями) */
  ownerRegex?: string
}>()

const { queryRange } = usePrometheusRange(props.clusterName)

const ns = () => `namespace="${props.namespace}"`
const owner = () =>
  `kube_pod_owner{${ns()},owner_kind="${props.ownerKind}",owner_name=~"${props.ownerRegex}",owner_is_controller="true"}`
/** Оставляет только серии подов нужного владельца (значение kube_pod_owner всегда 1); без владельца — как есть */
const own = (expr: string) =>
  props.ownerKind ? `${expr} * on(namespace,pod) group_left() ${owner()}` : expr





const GIB = 1024 ** 3
const esc = (v: string) => v.replace(/\\/g, '\\\\').replace(/"/g, '\\"')
const selector = `namespace="${esc(props.namespace)}",persistentvolumeclaim="${esc(props.claim)}"`

// Метрики отдаёт kubelet и только для смонтированных томов
const q = (metric: string) => `sum(${metric}{${selector}}) / ${GIB}`

async function fetchData(start: number, end: number, step: number) {
  const [used, capacity, available] = await Promise.all([
    queryRange(q('kubelet_volume_stats_used_bytes'), start, end, step),
    queryRange(q('kubelet_volume_stats_capacity_bytes'), start, end, step),
    queryRange(q('kubelet_volume_stats_available_bytes'), start, end, step),
  ])
  return [
    { label: 'Disk Usage', data: used },
    { label: 'Disk Capacity', data: capacity },
    { label: 'Disk Available', data: available },
  ]
}

function formatSize(gib: number): string {
  return gib < 1 ? `${(gib * 1024).toFixed(0)} MiB` : `${gib.toFixed(2)} GiB`
}
</script>

<style scoped>
.pvc-metrics { background: #1a1b2e; border: 1px solid #2a2b40; border-radius: 6px; padding: 12px; }
</style>