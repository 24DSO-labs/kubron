<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
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

const props = defineProps<{ clusterName: string; namespace: string; claim: string }>()
const { queryRange } = usePrometheusRange(props.clusterName)

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