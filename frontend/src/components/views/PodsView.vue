<template>
  <div @click="hideContextMenu">
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
          <td class="mono metric-cell">{{ getPodCPU(pod.name) }}</td>
          <td class="mono metric-cell">{{ getPodMem(pod.name) }}</td>
          <td v-html="getContainers(pod)"></td>
          <td><span :class="textClass(pod.status)">{{ pod.status }}</span></td>
          <td :class="{ warn: pod.restarts > 5 }">{{ pod.restarts }}</td>
          <td>{{ pod.age }}</td>
          <td>{{ pod.node }}</td>
          <td>{{ pod.qos }}</td>
          <td>
            <button class="btn-icon" @click.stop="$emit('yaml', pod.namespace, pod.name)">📋</button>
            <button class="btn-icon" @click.stop="openContextMenu(pod, $event)">⋮</button>
          </td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет подов</div>
    <div v-if="showContextMenu" :style="menuStyle" class="cluster-context-menu" @click.stop>
      <div class="ctx-menu-item" @click="handleShowDetails">Show Details</div>
      <div class="ctx-menu-item" @click="handleAttachPod">Attach Pod</div>
      <div class="ctx-menu-item" @click="handleShell">Shell</div>
      <div class="ctx-menu-item" @click="handleEvict">Evict</div>
      <div class="ctx-menu-item" @click="handleCopyLink">Copy Link to Clipboard</div>
      <div class="ctx-menu-item ctx-menu-sub" @click="handleShare">Share ></div>
      <div class="ctx-menu-item" @click="handleLogs">Logs</div>
      <div class="ctx-menu-item" @click="handleEdit">Edit</div>
      <div class="ctx-menu-item ctx-menu-danger" @click="handleDelete">Delete</div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { PodInfo, PodMetricsInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'
import { ref } from 'vue'

const props = defineProps<{
  items: PodInfo[]
  search: string
  statusClass: (s: string) => string
  podMetrics: PodMetricsInfo[]
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'select', pod: PodInfo): void
  (e: 'yaml', ns: string, name: string): void
  (e: 'attachPod', pod: PodInfo): void
  (e: 'shell', pod: PodInfo): void
  (e: 'evict', pod: PodInfo): void
  (e: 'copyLink', pod: PodInfo): void
  (e: 'share', pod: PodInfo): void
  (e: 'logs', pod: PodInfo): void
  (e: 'edit', pod: PodInfo): void
  (e: 'delete', pod: PodInfo): void
}>()

const showContextMenu = ref(false)
const menuStyle = ref<Record<string, string>>({})
const selectedPod = ref<PodInfo | null>(null)

function openContextMenu(pod: PodInfo, event: MouseEvent) {
  selectedPod.value = pod
  const button = event.target as HTMLElement
  const rect = button.getBoundingClientRect()
  const menuWidth = 200
  const menuHeight = 300

  // Horizontal: prefer left (align right to button)
  let left
  const availableLeft = rect.left
  if (availableLeft >= menuWidth) {
    left = rect.left - menuWidth + window.scrollX
  } else {
    left = rect.right + window.scrollX
    if (left + menuWidth > window.innerWidth + window.scrollX) {
      left = window.innerWidth - menuWidth + window.scrollX
    }
  }

  // Vertical: prefer bottom
  let top
  const availableBottom = window.innerHeight - rect.bottom
  if (availableBottom >= menuHeight) {
    top = rect.bottom + window.scrollY
  } else {
    top = rect.top - menuHeight + window.scrollY
    if (top < window.scrollY) {
      top = window.scrollY
    }
  }

  menuStyle.value = {
    position: 'absolute',
    top: `${top}px`,
    left: `${left}px`
  }
  showContextMenu.value = true
}

function hideContextMenu() {
  showContextMenu.value = false
}

function handleShowDetails() {
  if (selectedPod.value) emit('select', selectedPod.value)
  hideContextMenu()
}

function handleAttachPod() {
  if (selectedPod.value) emit('attachPod', selectedPod.value)
  hideContextMenu()
}

function handleShell() {
  if (selectedPod.value) emit('shell', selectedPod.value)
  hideContextMenu()
}

function handleEvict() {
  if (selectedPod.value) emit('evict', selectedPod.value)
  hideContextMenu()
}

function handleCopyLink() {
  if (selectedPod.value) emit('copyLink', selectedPod.value)
  hideContextMenu()
}

function handleShare() {
  if (selectedPod.value) emit('share', selectedPod.value)
  hideContextMenu()
}

function handleLogs() {
  if (selectedPod.value) emit('logs', selectedPod.value)
  hideContextMenu()
}

function handleEdit() {
  if (selectedPod.value) emit('edit', selectedPod.value)
  hideContextMenu()
}

function handleDelete() {
  if (selectedPod.value) emit('delete', selectedPod.value)
  hideContextMenu()
}

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

function onRefresh() { emit('refresh') }

function onSearch(val: string) { emit('search', val) }
</script>
