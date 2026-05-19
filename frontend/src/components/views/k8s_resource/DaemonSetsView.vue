<template>
  <K8sResourceListBaseOld
    :items="items"
    :search="search"
    resource-title="DaemonSets"
    empty-message="Нет DaemonSets"
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
import type { DaemonSetInfo } from '../../../types'
import K8sResourceListBase from '../K8sResourceListBase_old.vue'
import type { Column } from '../K8sResourceListBase_old.vue'
import { ref } from 'vue'
import { useKubDaemonSetActions } from '../../../composables/KubDaemonSetActions'

defineProps<{ items: DaemonSetInfo[]; search: string }>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'select', item: DaemonSetInfo): void
  (e: 'restart', ns: string, name: string): void
  (e: 'logs', ns: string, name: string): void
  (e: 'edit', ns: string, name: string): void
  (e: 'delete', ns: string, name: string): void
}>()

const selectedResource = ref<DaemonSetInfo | null>(null)

const columns: Column[] = [
  { key: 'name', label: 'Name', className: 'resource-name' },
  { key: 'namespace', label: 'Namespace', className: 'ns-label' },
  { key: 'desired', label: 'Desired' },
  { key: 'current', label: 'Current' },
  { key: 'ready', label: 'Ready' },
  { key: 'available', label: 'Available' },
  { key: 'age', label: 'Age' }
]

// Определим действия через emit
const resourceActions = {
  restart: (resource: DaemonSetInfo) => {
    emit('restart', resource.namespace, resource.name)
  },
  logs: (resource: DaemonSetInfo) => {
    emit('logs', resource.namespace, resource.name)
  }
}

const k8sActions = {
  showYAML: (kind: string, namespace: string, name: string) => {
    emit('edit', namespace, name)
  },
  refreshData: () => {
    emit('refresh')
  },
  deleteResource: (namespace: string, name: string) => {
    emit('delete', namespace, name)
  }
}

// Используем композабл
const { contextMenuItems, handleContextAction } = useKubDaemonSetActions(
  selectedResource,
  resourceActions,
  k8sActions
)

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }

function onSelect(item: DaemonSetInfo) {
  selectedResource.value = item
  emit('select', item)
}

function onContextOpen(resource: DaemonSetInfo) {
  selectedResource.value = resource
}

function onContextAction(action: string, resource: DaemonSetInfo) {
  selectedResource.value = resource
  handleContextAction(action, resource)
  if (action === 'select') {
    emit('select', resource)
  }
}
</script>

<style scoped>
.resource-name { font-weight: 500; }
.ns-label { color: #868d8a; }
</style>
