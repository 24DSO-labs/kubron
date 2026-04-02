<template>
  <K8sResourceListBase
    :items="items"
    :search="search"
    resource-title="Pods"
    empty-message="Нет подов"
    :columns="columns"
    :context-menu-items="contextMenuItems"
    @refresh="onRefresh"
    @search="onSearch"
    @select="onSelect"
    @context-action="onContextAction"
  />
</template>

<script lang="ts" setup>
import type { PodInfo, PodMetricsInfo } from '../../../types'
import K8sResourceListBase from '../K8sResourceListBase.vue'
import type { Column, ContextMenuItem } from '../K8sResourceListBase.vue'
import PodContainersCellComponent from './PodContainersCellComponent.vue'
import { ref, computed } from 'vue'

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
  (e: 'attachPod', pod: PodInfo, container?: string): void
  (e: 'shell', pod: PodInfo, container?: string): void
  (e: 'evict', pod: PodInfo): void
  (e: 'logs', pod: PodInfo, container?: string): void
  (e: 'edit', pod: PodInfo): void
  (e: 'delete', pod: PodInfo): void
}>()

const selectedPod = ref<PodInfo | null>(null)

const columns: Column[] = [
  { key: 'name', label: 'Name', className: 'resource-name' },
  { key: 'namespace', label: 'Namespace', className: 'ns-label' },
  { key: 'cpu', label: 'CPU', className: 'mono metric-cell' },
  { key: 'memory', label: 'Memory', className: 'mono metric-cell' },
  { key: 'containers', label: 'Containers', component: PodContainersCellComponent },
  { key: 'status', label: 'Status' },
  { key: 'restarts', label: 'Restarts' },
  { key: 'age', label: 'Age' },
  { key: 'node', label: 'Node' },
  { key: 'qos', label: 'QoS' }
]

const contextMenuItems = computed<ContextMenuItem[]>(() => {
  const allContainers = selectedPod.value?.containers || []
  const regularContainers = allContainers.filter(c => !c.isInit)
  return [
    { label: 'Show Details', action: 'select' },
    {
      label: 'Attach Pod',
      submenu: regularContainers.map(c => ({
        label: c.name,
        key: `attach-${c.name}`,
        action: `attachPod:${c.name}`
      })) || []
    },
    {
      label: 'Shell',
      submenu: regularContainers.map(c => ({
        label: c.name,
        key: `shell-${c.name}`,
        action: `shell:${c.name}`
      })) || []
    },
    { label: 'Evict', action: 'evict' },
    {
      label: 'Logs',
      submenu: regularContainers.map(c => ({
        label: c.name,
        key: `logs-${c.name}`,
        action: `logs:${c.name}`
      })) || []
    },
    { label: 'Edit', action: 'edit' },
    { label: 'Delete', action: 'delete', danger: true }
  ]
})

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
function onSelect(pod: PodInfo) { emit('select', pod) }

function onContextAction(action: string, pod: PodInfo) {
  selectedPod.value = pod
  
  // Handle submenu actions with container names
  if (action.startsWith('attachPod:')) {
    const containerName = action.substring('attachPod:'.length)
    emit('attachPod', pod, containerName)
    return
  }
  if (action.startsWith('shell:')) {
    const containerName = action.substring('shell:'.length)
    emit('shell', pod, containerName)
    return
  }
  if (action.startsWith('logs:')) {
    const containerName = action.substring('logs:'.length)
    emit('logs', pod, containerName)
    return
  }
  
  switch (action) {
    case 'select':
      emit('select', pod)
      break
    case 'evict':
      emit('evict', pod)
      break
    case 'edit':
      emit('edit', pod)
      break
    case 'delete':
      emit('delete', pod)
      break
  }
}
</script>

<style scoped>
.resource-name {
  font-weight: 500;
}

.ns-label {
  color: #868d8a;
}

.metric-cell {
  font-family: monospace;
}

.warn {
  color: #facc15;
}

.text-running {
  color: #4ade80;
}

.text-pending {
  color: #facc15;
}

.text-failed {
  color: #ef4444;
}

.text-succeeded {
  color: #4ade80;
}

.text-unknown {
  color: #868d8a;
}

.container-square {
  display: inline-block;
  width: 12px;
  height: 12px;
  margin: 0 2px;
  border-radius: 2px;
}

.square-green {
  background-color: #4ade80;
}

.square-red {
  background-color: #ef4444;
}

.square-yellow {
  background-color: #facc15;
}

.square-gray {
  background-color: #868d8a;
}
</style>
