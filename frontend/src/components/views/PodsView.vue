<template>
  <div>
    <div v-if="showContextMenu" class="menu-overlay" @click="hideContextMenu"></div>
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
          <td @click.stop>
            <button class="btn-icon" @click.stop="openContextMenu(pod, $event)">⋮</button>
          </td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет подов</div>
    <div v-if="showContextMenu" :style="menuStyle" class="cluster-context-menu" @click.stop>
      <div class="ctx-menu-item" @click="handleShowDetails">
        <span class="ctx-menu-arrow"></span>Show Details
      </div>
      <div class="ctx-menu-item-group" @mouseenter="showAttachMenu = true" @mouseleave="showAttachMenu = false">
        <div class="ctx-menu-item ctx-menu-with-arrow">
          <span class="ctx-menu-arrow">‹</span> Attach Pod
        </div>
        <div v-if="showAttachMenu && selectedPod?.containers.length" class="ctx-menu-sub">
          <div v-for="c in selectedPod.containers" :key="c.name" class="ctx-menu-sub-item" @click="handleAttachContainer(c.name)">
            {{ c.name }}
          </div>
        </div>
      </div>
      <div class="ctx-menu-item-group" @mouseenter="showTerminalMenu = true" @mouseleave="showTerminalMenu = false">
        <div class="ctx-menu-item ctx-menu-with-arrow">
          <span class="ctx-menu-arrow">‹</span> Shell
        </div>
        <div v-if="showTerminalMenu && selectedPod?.containers.length" class="ctx-menu-sub">
          <div v-for="c in selectedPod.containers" :key="c.name" class="ctx-menu-sub-item" @click="handleTerminalContainer(c.name)">
            {{ c.name }}
          </div>
        </div>
      </div>
      <div class="ctx-menu-item" @click="handleEvict">
        <span class="ctx-menu-arrow"></span>Evict
      </div>
      <div class="ctx-menu-item-group" @mouseenter="showLogsMenu = true" @mouseleave="showLogsMenu = false">
        <div class="ctx-menu-item ctx-menu-with-arrow">
          <span class="ctx-menu-arrow">‹</span> Logs
        </div>
        <div v-if="showLogsMenu && selectedPod?.containers.length" class="ctx-menu-sub">
          <div v-for="c in selectedPod.containers" :key="c.name" class="ctx-menu-sub-item" @click="handleLogsContainer(c.name)">
            {{ c.name }}
          </div>
        </div>
      </div>
      <div class="ctx-menu-item" @click="handleEdit">
        <span class="ctx-menu-arrow"></span>Edit
      </div>
      <div class="ctx-menu-item ctx-menu-danger" @click="handleDelete">
        <span class="ctx-menu-arrow"></span>Delete
      </div>
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
  (e: 'attachPod', pod: PodInfo): void
  (e: 'shell', pod: PodInfo): void
  (e: 'evict', pod: PodInfo): void
  (e: 'logs', pod: PodInfo): void
  (e: 'edit', pod: PodInfo): void
  (e: 'delete', pod: PodInfo): void
}>()

const showContextMenu = ref(false)
const menuStyle = ref<Record<string, string>>({})
const selectedPod = ref<PodInfo | null>(null)
const showAttachMenu = ref(false)
const showTerminalMenu = ref(false)
const showLogsMenu = ref(false)

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

function toggleAttachMenu() {
  showAttachMenu.value = !showAttachMenu.value
}

function handleAttachContainer(containerName: string) {
  if (selectedPod.value) {
    emit('attachPod', selectedPod.value)
  }
  hideContextMenu()
}

function toggleTerminalMenu() {
  showTerminalMenu.value = !showTerminalMenu.value
}

function handleTerminalContainer(containerName: string) {
  if (selectedPod.value) {
    emit('shell', selectedPod.value)
  }
  hideContextMenu()
}

function toggleLogsMenu() {
  showLogsMenu.value = !showLogsMenu.value
}

function handleLogsContainer(containerName: string) {
  if (selectedPod.value) {
    emit('logs', selectedPod.value)
  }
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

<style scoped>
.menu-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 999;
  cursor: default;
}

.cluster-context-menu {
  background: #1e1f35;
  border: 1px solid #3b3d5c;
  border-radius: 4px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  z-index: 1000;
  min-width: 160px;
}

.ctx-menu-item {
  padding: 6px 12px;
  color: #d4d4d4;
  cursor: pointer;
  font-size: 0.875rem;
  transition: background 0.15s;
  padding-left: 34px;
  text-align: left;
  position: relative;
  margin: -9px;
}

.ctx-menu-item:hover {
  background: rgba(255, 255, 255, 0.1);
  color: white;
}

.ctx-menu-item:first-child {
  border-radius: 3px 3px 0 0;
}

.ctx-menu-item:last-child {
  border-radius: 0 0 3px 3px;
}

.ctx-menu-arrow {
  position: absolute;
  left: 19px;
  color: #868d8a;
  font-size: 0.9rem;
}

.ctx-menu-item-group {
  position: relative;
  display: block;
}

.ctx-menu-sub {
  position: absolute;
  top: 0;
  right: 100%;
  margin-right: 8px;
  background: #1e1f35;
  border: 1px solid #3b3d5c;
  border-radius: 4px;
  min-width: 160px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  z-index: 1001;
  white-space: nowrap;
}

.ctx-menu-sub-item {
  padding: 8px 12px;
  color: #d4d4d4;
  cursor: pointer;
  font-size: 0.875rem;
  transition: background 0.15s;
  white-space: nowrap;
}

.ctx-menu-sub-item:hover {
  background: rgba(255, 255, 255, 0.1);
  color: white;
}

.ctx-menu-sub-item:first-child {
  border-radius: 3px 3px 0 0;
}

.ctx-menu-sub-item:last-child {
  border-radius: 0 0 3px 3px;
}

.ctx-menu-danger {
  color: #ef4444;
}

.ctx-menu-danger:hover {
  background: rgba(239, 68, 68, 0.2);
}
</style>
