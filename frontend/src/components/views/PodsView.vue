<template>
  <div>
    <ContentHeader icon="" title="Pods" :count="items.length" :search="search" showSearch @refresh="onRefresh" @search="onSearch" />
    <table v-if="items.length > 0" class="table">
      <thead>
        <tr>
          <th>Name</th><th>Namespace</th><th>CPU</th><th>Memory</th><th>Containers</th><th>Status</th><th>Restarts</th><th>Age</th><th>Node</th><th>QoS</th><th></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="pod in items" :key="pod.namespace+pod.name" @click="$emit('select', pod)" class="clickable-row">
          <td class="resource-name">{{ pod.name }}</td>
          <td class="ns-label">{{ pod.namespace }}</td>
          <td class="mono metric-cell left-align">{{ getPodCPU(pod.name) }}</td>
          <td class="mono metric-cell left-align">{{ getPodMem(pod.name) }}</td>
          <td class="left-align" v-html="getContainers(pod)"></td>
          <td class="left-align"><span :class="textClass(pod.status)">{{ pod.status }}</span></td>
          <td class="left-align" :class="{ warn: pod.restarts > 5 }">{{ pod.restarts }}</td>
          <td class="left-align">{{ pod.age }}</td>
          <td class="left-align">{{ pod.node }}</td>
          <td class="left-align">{{ pod.qos }}</td>
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
  const val = m?.totalCpu || '—'
  if (val === '—') return val
  return parseFloat(val).toFixed(3)
}

function getPodMem(name: string): string {
  if (!props.podMetrics) return '—'
  const m = props.podMetrics.find(p => p.name === name)
  const val = m?.totalMem || '—'
  if (val === '—') return val
  const match = val.match(/^(\d+\.?\d*)(.*)$/)
  if (match) {
    const num = parseFloat(match[1]).toFixed(1)
    return num + match[2]
  }
  return val
}

function getContainers(pod: PodInfo): string {
  let html = ''
  for (const c of pod.containers) {
    let color = 'red'
    if (c.isInit) {
      color = 'gray'
    } else if (c.ready) {
      color = 'green'
    } else if (c.state.includes('wait') || c.state.includes('Creat') || c.state.includes('Pend') || c.state === 'waiting') {
      color = 'yellow'
    } else if (c.state === 'terminated' || c.state.includes('Complet')) {
      color = 'gray'
    }
    html += `<span class="container-square square-${color}"></span>`
  }
  return html
}


function textClass(s: string): string {
  return `text-${s.toLowerCase()}`
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
