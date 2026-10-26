<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div class="im-viewer">
    <div class="im-tabs">
      <button
        v-for="t in tabs"
        :key="t.id"
        :class="['im-btn', { active: selected === t.id }]"
        @click="selected = t.id"
      >{{ t.label }}</button>
    </div>
    <div class="im-chart">
      <MultiLineChart
        :key="`${ingress}/${selected}`"
        :title="current.title"
        unit=""
        :fetch-data="current.fetch"
        :format-value="current.format"
        :refresh-interval-ms="30000"
      />
    </div>
    <p class="im-hint">Метрики ingress-nginx: контроллер должен собираться Prometheus-ом Kubron.</p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import MultiLineChart from './MultiLineChart.vue'
import { usePrometheusRange } from '../../composables/useMetrics'

/**
 * Метрики Ingress (ingress-nginx).
 * Каждый запрос обязан вернуть ОДНУ серию: usePrometheusRange берёт result[0], поэтому везде sum().
 *  - Network: скорость отправки байт для ответов 2xx и 5xx;
 *  - Duration: средняя длительность запроса / ответа (sum / count гистограммы).
 */
const props = defineProps<{ clusterName: string; namespace: string; ingress: string }>()
const { queryRange } = usePrometheusRange(props.clusterName)

type Series = { label: string; data: [number, number][] }
interface TabDef {
  id: string
  label: string
  title: string
  format: (v: number) => string
  fetch: (start: number, end: number, step: number) => Promise<Series[]>
}

const esc = (v: string) => v.replace(/\\/g, '\\\\').replace(/"/g, '\\"')
const sel = computed(() => `ingress="${esc(props.ingress)}",namespace="${esc(props.namespace)}"`)
const P = 'nginx_ingress_controller_'

const sumRate = (metric: string, extra = '') => `sum(rate(${P}${metric}{${sel.value}${extra}}[5m]))`
// bytes_sent есть не во всех версиях контроллера: запасной вариант — размер ответа
const bytesSent = (status: string) =>
  `${sumRate('bytes_sent_sum', `,status=~"${status}"`)} or ${sumRate('response_size_sum', `,status=~"${status}"`)}`
// Средняя длительность: sum/count; без трафика результат пуст (пропуск на графике), а не NaN
const avgDuration = (name: string) => `${sumRate(`${name}_sum`)} / ${sumRate(`${name}_count`)} > 0`

const BYTE_UNITS = ['B', 'KiB', 'MiB', 'GiB', 'TiB']
function bytesPerSec(value: number): string {
  if (!isFinite(value) || value <= 0) return '0'
  let v = value
  let i = 0
  while (v >= 1024 && i < BYTE_UNITS.length - 1) { v /= 1024; i++ }
  return `${i ? v.toFixed(1) : Math.round(v)} ${BYTE_UNITS[i]}/s`
}
const seconds = (v: number) => (v < 1 ? `${(v * 1000).toFixed(0)} ms` : `${v.toFixed(2)} s`)

function tab(id: string, label: string, title: string, format: (v: number) => string, lines: [string, string][]): TabDef {
  return {
    id, label, title, format,
    fetch: async (start, end, step) => {
      const data = await Promise.all(lines.map(([, q]) => queryRange(q, start, end, step)))
      return lines.map(([name], i) => ({ label: name, data: data[i] as [number, number][] }))
    },
  }
}

const tabs = computed<TabDef[]>(() => [
  tab('network', 'Network', 'Network', bytesPerSec, [
    ['Bytes sent, status 2xx', bytesSent('2..')],
    ['Bytes sent, status 5xx', bytesSent('5..')],
  ]),
  tab('duration', 'Duration', 'Duration', seconds, [
    ['Request', avgDuration('request_duration_seconds')],
    ['Response', avgDuration('response_duration_seconds')],
  ]),
])

const selected = ref('network')
const current = computed(() => tabs.value.find(t => t.id === selected.value) ?? tabs.value[0])
</script>

<style scoped>
.im-viewer { display: flex; flex-direction: column; gap: 12px; width: 100%; }
.im-tabs { display: flex; gap: 6px; flex-wrap: wrap; }
.im-btn {
  padding: 6px 12px; background: #1e1f35; border: 1px solid #3b3d5c; border-radius: 4px;
  color: #9ca0b8; font-size: 0.85rem; font-weight: 500; cursor: pointer; transition: all 0.15s; white-space: nowrap;
}
.im-btn:hover { background: #262740; border-color: #5a5c7a; color: #c0c2d8; }
.im-btn.active { background: #3b82f6; border-color: #3b82f6; color: white; }
.im-chart { background: #1a1b2e; border: 1px solid #2a2b40; border-radius: 6px; padding: 12px; width: 100%; }
.im-hint { margin: 0; font-size: 0.75rem; color: #6b6d8a; text-align: left; }
</style>