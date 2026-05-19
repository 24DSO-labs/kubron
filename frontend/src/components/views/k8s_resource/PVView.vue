<template>
  <K8sResourceListBaseOld
    :items="items"
    :search="search"
    resource-title="PersistentVolumes"
    empty-message="Нет PersistentVolumes"
    :columns="columns"
    :context-menu-items="contextMenuItems"
    @refresh="onRefresh"
    @search="onSearch"
    @select="onSelect"
    @context-action="onContextAction"
  />
</template>

<script lang="ts" setup>
import type { PVInfo } from '../../../types'
import K8sResourceListBase from '../K8sResourceListBase_old.vue'
import type { Column, ContextMenuItem } from '../K8sResourceListBase_old.vue'
import { ref } from 'vue'

defineProps<{ items: PVInfo[]; search: string }>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'select', item: PVInfo): void
}>()

const selectedItem = ref<PVInfo | null>(null)

const columns: Column[] = [
  { key: 'name', label: 'Name', className: 'resource-name' },
  { key: 'capacity', label: 'Capacity' },
  { key: 'accessModes', label: 'Access Modes' },
  { key: 'reclaimPolicy', label: 'Reclaim Policy' },
  { key: 'status', label: 'Status' },
  { key: 'claim', label: 'Claim' },
  { key: 'age', label: 'Age' }
]

const contextMenuItems: ContextMenuItem[] = [
  { label: 'Show Details', action: 'select' }
]

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
function onSelect(item: PVInfo) { emit('select', item) }

function onContextAction(action: string, item: PVInfo) {
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
</style>
