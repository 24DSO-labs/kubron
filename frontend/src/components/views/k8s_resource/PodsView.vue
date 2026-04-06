<template>
  <K8sResourceListBase
    :items="itemsWithMetrics"
    :search="search"
    resource-title="Pods"
    empty-message="Нет подов"
    :columns="columns"
    :context-menu-items="contextMenuItems"
    @refresh="onRefresh"
    @search="onSearch"
    @select="onSelect"
    @context-action="onContextAction"
    @context-open="onContextOpen"
  />
</template>

<script lang="ts" setup>
import type { PodInfo, PodMetricsInfo } from '../../../types'
import K8sResourceListBase from '../K8sResourceListBase.vue'
import type { Column } from '../K8sResourceListBase.vue'
import PodContainersCellComponent from './PodContainersCellComponent.vue'
import { ref, computed } from 'vue'
import { useKubPodActions } from '../../../composables/KubPodActions'

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

// Pod действия — передаем через emit наверх
const podActions = {
  openAttachWithContainer: (container: string, pod: PodInfo) => {
    emit('attachPod', pod, container)
  },
  openShellWithContainer: (container: string, pod: PodInfo) => {
    emit('shell', pod, container)
  },
  openLogs: (pod: PodInfo, container?: string) => {
    emit('logs', pod, container)
  },
  evictPod: (pod: PodInfo) => {
    emit('evict', pod)
  }
}

// Общие k8s действия
const k8sActions = {
  showYAML: (kind: string, namespace: string, name: string) => {
    emit('edit', selectedPod.value!)
  },
  refreshData: () => {
    emit('refresh')
  },
  deleteResource: (namespace: string, name: string) => {
    emit('delete', selectedPod.value!)
  }
}

// Используем новый композабл
const { contextMenuItems, handleContextAction } = useKubPodActions(
  selectedPod,
  podActions,
  k8sActions
)

// Merge metrics with pods
const itemsWithMetrics = computed(() => {
  return props.items.map(pod => {
    const metric = props.podMetrics.find(m => m.name === pod.name && m.namespace === pod.namespace)
    return {
      ...pod,
      cpu: formatCpu(metric?.totalCpu),
      memory: formatMemory(metric?.totalMem)
    }
  })
})

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

function formatCpu(cpuStr: string | undefined): string {
  if (!cpuStr) return '—'
  try {
    const cpu = parseFloat(cpuStr)
    return isNaN(cpu) ? '—' : cpu.toFixed(3)
  } catch {
    return '—'
  }
}

function formatMemory(memStr: string | undefined): string {
  if (!memStr) return '—'
  try {
    const bytes = parseFloat(memStr)
    if (isNaN(bytes)) return '—'
    const mib = bytes / 1024 / 1024
    return mib.toFixed(1) + 'MiB'
  } catch {
    return '—'
  }
}

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }

function onSelect(pod: PodInfo) {
  selectedPod.value = pod
  emit('select', pod)
}

// Вызывается при ОТКРЫТИИ контекстного меню (до выбора действия)
function onContextOpen(pod: PodInfo) {
  selectedPod.value = pod
}

function onContextAction(action: string, pod: PodInfo) {
  selectedPod.value = pod
  handleContextAction(action, pod)

  // select обрабатываем здесь
  if (action === 'select') {
    emit('select', pod)
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
</style>