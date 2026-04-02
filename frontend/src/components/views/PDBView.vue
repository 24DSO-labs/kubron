<template>
  <K8sResourceListBase
    :items="items"
    :search="search"
    resource-title="PodDisruptionBudgets"
    empty-message="Нет PodDisruptionBudgets"
    :columns="columns"
    :context-menu-items="contextMenuItems"
    @refresh="onRefresh"
    @search="onSearch"
    @select="onSelect"
    @context-action="onContextAction"
  />
</template>

<script lang="ts" setup>
import type { PDBInfo } from '../../types'
import K8sResourceListBase from './K8sResourceListBase.vue'
import type { Column, ContextMenuItem } from './K8sResourceListBase.vue'
import { ref } from 'vue'

defineProps<{ items: PDBInfo[]; search: string }>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'select', item: PDBInfo): void
}>()

const selectedItem = ref<PDBInfo | null>(null)

const columns: Column[] = [
  { key: 'name', label: 'Name', className: 'resource-name' },
  { key: 'namespace', label: 'Namespace', className: 'ns-label' },
  { key: 'minAvailable', label: 'Min Available' },
  { key: 'maxUnavailable', label: 'Max Unavailable' },
  { key: 'currentHealthy', label: 'Current Healthy' },
  { key: 'age', label: 'Age' }
]

const contextMenuItems: ContextMenuItem[] = [
  { label: 'Show Details', action: 'select' }
]

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
function onSelect(item: PDBInfo) { emit('select', item) }

function onContextAction(action: string, item: PDBInfo) {
  selectedItem.value = item
  switch (action) {
    case 'select':
      emit('select', item)
      break
  }
}
</script>

<style scoped>
.resource-name { font-weight: 500; }
.ns-label { color: #868d8a; }
</style>
