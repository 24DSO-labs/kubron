<template>
  <K8sResourceListBase
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
  />
</template>

<script lang="ts" setup>
import type { DaemonSetInfo } from '../../../types'
import K8sResourceListBase from '../K8sResourceListBase.vue'
import type { Column, ContextMenuItem } from '../K8sResourceListBase.vue'
import { ref } from 'vue'

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

const selectedItem = ref<DaemonSetInfo | null>(null)

const columns: Column[] = [
  { key: 'name', label: 'Name', className: 'resource-name' },
  { key: 'namespace', label: 'Namespace', className: 'ns-label' },
  { key: 'desired', label: 'Desired' },
  { key: 'current', label: 'Current' },
  { key: 'ready', label: 'Ready' },
  { key: 'available', label: 'Available' },
  { key: 'age', label: 'Age' }
]

const contextMenuItems: ContextMenuItem[] = [
  { label: 'Show Details', action: 'select' },
  { label: 'Restart', action: 'restart' },
  { label: 'Logs', action: 'logs' },
  { label: 'Edit', action: 'edit' },
  { label: 'Delete', action: 'delete', danger: true }
]

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
function onSelect(item: DaemonSetInfo) { emit('select', item) }

function onContextAction(action: string, item: DaemonSetInfo) {
  selectedItem.value = item
  switch (action) {
    case 'select':
      emit('select', item)
      break
    case 'restart':
      emit('restart', item.namespace, item.name)
      break
    case 'logs':
      emit('logs', item.namespace, item.name)
      break
    case 'edit':
      emit('edit', item.namespace, item.name)
      break
    case 'delete':
      emit('delete', item.namespace, item.name)
      break
  }
}
</script>

<style scoped>
.resource-name { font-weight: 500; }
.ns-label { color: #868d8a; }
</style>
