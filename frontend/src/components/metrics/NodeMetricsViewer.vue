<template>
  <div class="nm-viewer">
    <div class="nm-tabs">
      <button
        v-for="t in tabs"
        :key="t.id"
        :class="['nm-btn', { active: selected === t.id }]"
        @click="selected = t.id"
      >{{ t.label }}</button>
    </div>

    <div class="nm-charts">
      <div v-for="(c, i) in current.charts" :key="`${nodeName}/${selected}/${i}`" class="nm-chart">
        <MultiLineChart
          :title="c.title"
          unit=""
          :fetch-data="c.fetch"
          :format-value="c.format"
          :refresh-interval-ms="30000"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import MultiLineChart from './MultiLineChart.vue'
import { usePrometheusRange } from '../../composables/useMetrics'

/**
 * Метрики одного узла: CPU / Memory / Network / Disk.
 *  - node_*    (node-exporter) — фактическое использование; узел выбирается меткой kubernetes_node
 *                (см. promNodeLabel в Go);
 *  - kube_*    (kube-state-metrics) — requests и ёмкость; там узел лежит в метке node;
 *  - container_* (cAdvisor) — нагрузка подов; к узлу привязываем через kube_pod_info{node=...},
 *                потому что у cAdvisor-серий нет имени Node.
 * Каждый запрос обязан вернуть ОДНУ серию: usePrometheusRange берёт result[0].
 */
const props = defineProps<{ clusterName: string; nodeName: string }>()
const { queryRange } = usePrometheusRange(props.clusterName)

type Series = { label: string; data: [number, number][] }
interface ChartDef {
  title: string
  format: (v: number) => string
  fetch: (start: number, end: number, step: number) => Promise<Series[]>
}
interface TabDef { id: string; label: string; charts: ChartDef[] }

// ─── Форматтеры ───
const BYTE_UNITS = ['B', 'KiB', 'MiB', 'GiB', 'TiB']
function bytes(value: number): string {
  if (!isFinite(value) || value <= 0) return '0'
  let v = value
  let i = 0
  while (v >= 1024 && i < BYTE_UNITS.length - 1) { v /= 1024; i++ }
  return `${i ? v.toFixed(1) : Math.round(v)} ${BYTE_UNITS[i]}`
}
const bytesPerSec = (v: number) => `${bytes(v)}/s`
function cores(v: number): string {
  if (v < 0.001) return '0'
  return v < 1 ? `${(v * 1000).toFixed(0)}m` : v.toFixed(2)
}
const perSec = (v: number) => (v < 0.01 ? '0' : v.toFixed(2))

// ─── Селекторы ───
const esc = (v: string) => v.replace(/\\/g, '\\\\').replace(/"/g, '\\"')
const nodeSel = computed(() => `kubernetes_node="${esc(props.nodeName)}"`)
const ksmSel = computed(() => `node="${esc(props.nodeName)}"`)

// Виртуальные интерфейсы (veth, мосты, оверлей) дублируют трафик
const PHYSICAL_NICS = 'device!~"lo|veth.*|docker.*|cni.*|flannel.*|cali.*|br-.*"'
const ROOT_FS = 'mountpoint="/",fstype!="tmpfs"'

/** Оставляет серии подов, которые стоят на этом узле (kube_pod_info — по одной серии на под) */
const onNode = (expr: string) => `${expr} * on(namespace,pod) group_left() kube_pod_info{${ksmSel.value}}`
/** Requests считаем только у живых подов: завершённые Job'ы ресурсы уже не занимают */
const activePods = (expr: string) =>
  `${expr} * on(namespace,pod) group_left() (kube_pod_status_phase{phase=~"Pending|Running"} == 1)`

function chart(title: string, format: (v: number) => string, lines: [string, string][]): ChartDef {
  return {
    title,
    format,
    fetch: async (start, end, step) => {
      const data = await Promise.all(lines.map(([, q]) => queryRange(q, start, end, step)))
      return lines.map(([label], i) => ({ label, data: data[i] as [number, number][] }))
    },
  }
}

const tabs = computed<TabDef[]>(() => {
  const n = nodeSel.value
  const k = ksmSel.value
  return [
    {
      id: 'cpu',
      label: 'CPU',
      charts: [
        chart('CPU', cores, [
          ['CPU Usage', `sum(rate(node_cpu_seconds_total{mode!="idle",${n}}[5m]))`],
          ['CPU Requests', `sum(${activePods(`kube_pod_container_resource_requests{resource="cpu",${k}}`)})`],
          ['CPU Allocatable Capacity', `sum(kube_node_status_allocatable{resource="cpu",${k}})`],
          ['CPU Capacity', `sum(kube_node_status_capacity{resource="cpu",${k}})`],
        ]),
      ],
    },
    {
      id: 'memory',
      label: 'Memory',
      charts: [
        chart('Memory', bytes, [
          ['Memory Usage', `sum(node_memory_MemTotal_bytes{${n}}) - sum(node_memory_MemAvailable_bytes{${n}})`],
          [
            'Workload Memory Usage',
            `sum(${onNode('container_memory_working_set_bytes{container!="",container!="POD"}')})`,
          ],
          ['Memory Requests', `sum(${activePods(`kube_pod_container_resource_requests{resource="memory",${k}}`)})`],
          ['Memory Allocatable Capacity', `sum(kube_node_status_allocatable{resource="memory",${k}})`],
          ['Memory Capacity', `sum(kube_node_status_capacity{resource="memory",${k}})`],
        ]),
      ],
    },
    {
      id: 'network',
      label: 'Network',
      charts: [
        chart('Network Traffic', bytesPerSec, [
          ['Receive', `sum(rate(node_network_receive_bytes_total{${n},${PHYSICAL_NICS}}[5m]))`],
          ['Transmit', `sum(rate(node_network_transmit_bytes_total{${n},${PHYSICAL_NICS}}[5m]))`],
        ]),
        chart('Packet Errors & Drops (per second)', perSec, [
          ['Receive Errors', `sum(rate(node_network_receive_errs_total{${n},${PHYSICAL_NICS}}[5m]))`],
          ['Transmit Errors', `sum(rate(node_network_transmit_errs_total{${n},${PHYSICAL_NICS}}[5m]))`],
          ['Receive Drops', `sum(rate(node_network_receive_drop_total{${n},${PHYSICAL_NICS}}[5m]))`],
          ['Transmit Drops', `sum(rate(node_network_transmit_drop_total{${n},${PHYSICAL_NICS}}[5m]))`],
        ]),
      ],
    },
    {
      id: 'disk',
      label: 'Disk',
      charts: [
        chart('Disk', bytes, [
          [
            'Disk Usage',
            `max(node_filesystem_size_bytes{${n},${ROOT_FS}}) - max(node_filesystem_avail_bytes{${n},${ROOT_FS}})`,
          ],
          ['Disk Size', `max(node_filesystem_size_bytes{${n},${ROOT_FS}})`],
        ]),
      ],
    },
  ]
})

const selected = ref('cpu')
const current = computed(() => tabs.value.find(t => t.id === selected.value) ?? tabs.value[0])
</script>

<style scoped>
.nm-viewer { display: flex; flex-direction: column; gap: 12px; width: 100%; }
.nm-tabs { display: flex; gap: 6px; flex-wrap: wrap; }
.nm-btn {
  padding: 6px 12px; background: #1e1f35; border: 1px solid #3b3d5c; border-radius: 4px;
  color: #9ca0b8; font-size: 0.85rem; font-weight: 500; cursor: pointer; transition: all 0.15s; white-space: nowrap;
}
.nm-btn:hover { background: #262740; border-color: #5a5c7a; color: #c0c2d8; }
.nm-btn.active { background: #3b82f6; border-color: #3b82f6; color: white; }
.nm-charts { display: flex; flex-direction: column; gap: 12px; }
.nm-chart { background: #1a1b2e; border: 1px solid #2a2b40; border-radius: 6px; padding: 12px; width: 100%; }
</style>
