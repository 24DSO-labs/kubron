<template>
  <div>
    <ResourceListBase
      :items="itemsWithMetrics"
      :search="localSearch"
      resource-title="Pods"
      empty-message="Нет подов"
      resource-type="pods"
      :custom-columns="columns"
      @refresh="refreshData"
      @search="(v) => { localSearch = v; emit('search', v) }"
      @select="onSelect"
      @contextAction="onContextAction"
      @shell:open="onShellOpen"
      @attach:open="onAttachOpen"
      @logs:open="onLogsOpen"
    />
  </div>
</template>

<script lang="ts" setup>
import type { PodInfo, PodMetricsInfo, Column } from '../../../types'
import { useK8s } from '../../../composables/useK8s'
import ResourceListBase from '../ResourceListBase.vue'
import PodContainersCellComponent from './PodContainersCellComponent.vue'
import { computed, ref } from 'vue'
import useResourceView from '../../../composables/useResourceView'

const { filteredPods, refreshData, search } = useK8s()

const props = defineProps<{
  podMetrics: PodMetricsInfo[]
}>()

const emit = defineEmits<{
  (e: 'search', val: string): void
}>()

const localSearch = ref(search.value)

const { onSelect, onContextAction } = useResourceView('pod')

// Merge metrics with pods
const itemsWithMetrics = computed(() => {
  return filteredPods.value.map(pod => {
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

function onShellOpen(namespace: string, podName: string, container: string) {
  const pod = filteredPods.value.find(p => p.name === podName && p.namespace === namespace)
  if (pod) {
    onContextAction(`shell:${container}`, pod)
  }
}

function onAttachOpen(namespace: string, podName: string, container: string) {
  const pod = filteredPods.value.find(p => p.name === podName && p.namespace === namespace)
  if (pod) {
    onContextAction(`attach:${container}`, pod)
  }
}

function onLogsOpen(namespace: string, podName: string, container: string) {
  const pod = filteredPods.value.find(p => p.name === podName && p.namespace === namespace)
  if (pod) {
    onContextAction(`logs:${container}`, pod)
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
