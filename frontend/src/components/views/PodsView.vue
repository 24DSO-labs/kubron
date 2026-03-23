<template>
  <div>
    <ContentHeader icon="📦" title="Pods" :count="items.length" :search="search" showSearch @refresh="onRefresh" @search="onSearch" />
    <table v-if="items.length > 0" class="table">
      <thead>
        <tr>
          <th>Name</th><th>Namespace</th><th>Ready</th><th>Status</th>
          <th>CPU</th><th>Memory</th>
          <th>Restarts</th><th>Age</th><th>Node</th><th></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="pod in items" :key="pod.namespace+pod.name" @click="$emit('select', pod)" class="clickable-row">
          <td class="resource-name">{{ pod.name }}</td>
          <td class="ns-label">{{ pod.namespace }}</td>
          <td>{{ pod.ready }}</td>
          <td><span class="status-badge" :class="statusClass(pod.status)">{{ pod.status }}</span></td>
          <td class="mono metric-cell">{{ getPodCPU(pod.name) }}</td>
          <td class="mono metric-cell">{{ getPodMem(pod.name) }}</td>
          <td :class="{ warn: pod.restarts > 5 }">{{ pod.restarts }}</td>
          <td>{{ pod.age }}</td>
          <td>{{ pod.node }}</td>
          <td><button class="btn-icon" @click.stop="$emit('yaml', pod.namespace, pod.name)">📋</button></td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет подов</div>
  </div>
</template>

<script lang="ts" setup>
import type { PodInfo, PodMetricsInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'

const props = defineProps<{
  items: PodInfo[]
  search: string
  statusClass: (s: string) => string
  podMetrics: PodMetricsInfo[]
}>()

function getPodCPU(name: string): string {
  if (!props.podMetrics) return '—'
  const m = props.podMetrics.find(p => p.name === name)
  return m?.totalCpu || '—'
}

function getPodMem(name: string): string {
  if (!props.podMetrics) return '—'
  const m = props.podMetrics.find(p => p.name === name)
  return m?.totalMem || '—'
}

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'select', pod: PodInfo): void
  (e: 'yaml', ns: string, name: string): void
}>()

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
</script>
