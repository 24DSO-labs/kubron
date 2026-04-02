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
    @context-action="onContextAction"
  />
</template>

<script lang="ts" setup>
import type { StatefulSetInfo } from '../../../types'
import K8sResourceListBase from './K8sResourceListBase.vue'
import type { Column, ContextMenuItem } from './K8sResourceListBase.vue'
import { ref } from 'vue'

defineProps<{ items: StatefulSetInfo[]; search: string }>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'select', item: StatefulSetInfo): void
  (e: 'edit', ns: string, name: string): void
  (e: 'delete', ns: string, name: string): void
}>()

const selectedItem = ref<StatefulSetInfo | null>(null)

const columns: Column[] = [
  { key: 'name', label: 'Name', className: 'resource-name' },
  { key: 'namespace', label: 'Namespace', className: 'ns-label' },
  { key: 'readyReplicas', label: 'Ready' },
  { key: 'replicas', label: 'Replicas' },
  { key: 'age', label: 'Age' }
]

const contextMenuItems: ContextMenuItem[] = [
  { label: 'Show Details', action: 'select' },
  { label: 'Edit', action: 'edit' },
  { label: 'Delete', action: 'delete', danger: true }
]

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
function onSelect(item: StatefulSetInfo) { emit('select', item) }

function onContextAction(action: string, item: StatefulSetInfo) {
  selectedItem.value = item
  switch (action) {
    case 'select':
      emit('select', item)
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
