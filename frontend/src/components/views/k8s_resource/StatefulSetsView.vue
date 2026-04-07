<template>
  <K8sResourceListBase
    :items="items"
    :search="search"
    resource-title="StatefulSets"
    empty-message="Нет StatefulSets"
    :columns="columns"
    :context-menu-items="contextMenuItems"
    @refresh="onRefresh"
    @search="onSearch"
    @select="onSelect"
    @context-open="onContextOpen"
    @context-action="onContextAction"
  />
</template>

<script lang="ts" setup>
import type { StatefulSetInfo } from '../../../types'
import K8sResourceListBase from '../K8sResourceListBase.vue'
import type { Column, ContextMenuItem } from '../K8sResourceListBase.vue'

import { ref, computed } from 'vue'
import { useKubStatefulSetActions } from '../../../composables/KubStatefulSetActions'

const props = defineProps<{ items: StatefulSetInfo[]; search: string }>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'select', item: StatefulSetInfo): void
  (e: 'edit', ns: string, name: string): void
  (e: 'delete', ns: string, name: string): void
  (e: 'restart', ns: string, name: string): void
  (e: 'logs', ns: string, name: string): void
}>()

const selectedItem = ref<StatefulSetInfo | null>(null)

const columns: Column[] = [
  { key: 'name', label: 'Name', className: 'resource-name' },
  { key: 'namespace', label: 'Namespace', className: 'ns-label' },
  
  { key: 'pods', label: 'Pods', 
    formatter: (item) => `${item.readyReplicas}/${item.replicas}`,
    colorFn: (item) => item.readyReplicas === item.replicas ? '#4ade80' : '#facc15'
  },
  
  { key: 'Replicas', label: 'Replicas', formatter: (item) => `${item.replicas}` },
  { key: 'age', label: 'Age' }
  
]

// Используем композабл для действий
const selectedResource = computed(() => selectedItem.value)

const resourceActions = {
  restart: (resource: any) => {
    emit('restart', resource.namespace, resource.name)
  },
  logs: (resource: any) => {
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

const { handleContextAction } = useKubStatefulSetActions(
  selectedResource,
  resourceActions,
  k8sActions
)

const contextMenuItems: ContextMenuItem[] = [
  { label: 'Show Details', action: 'select' },
  { label: 'Edit', action: 'edit' },
  { label: 'Restart', action: 'restart' },
  { label: 'Logs', action: 'logs' },
  { label: 'Delete', action: 'delete', danger: true }
]

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
function onSelect(item: StatefulSetInfo) { 
  selectedItem.value = item
  emit('select', item) 
}

function onContextOpen(item: StatefulSetInfo) {
  selectedItem.value = item
}

function onContextAction(action: string, item: StatefulSetInfo) {
  selectedItem.value = item
  handleContextAction(action, item)
}
</script>

<style scoped>
.resource-name { font-weight: 500; }
.ns-label { color: #868d8a; }
</style>
