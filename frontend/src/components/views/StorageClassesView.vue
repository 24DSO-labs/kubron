<template>
  <K8sResourceListBase
    :items="items"
    :search="search"
    resource-title="StorageClasses"
    empty-message="Нет StorageClasses"
    :columns="columns"
    :context-menu-items="contextMenuItems"
    @refresh="onRefresh"
    @search="onSearch"
    @select="onSelect"
    @context-action="onContextAction"
  />
</template>

<script lang="ts" setup>
import type { StorageClassInfo } from '../../types'
import K8sResourceListBase from './K8sResourceListBase.vue'
import type { Column, ContextMenuItem } from './K8sResourceListBase.vue'
import { ref } from 'vue'

defineProps<{ items: StorageClassInfo[]; search: string }>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'select', item: StorageClassInfo): void
}>()

const selectedItem = ref<StorageClassInfo | null>(null)

const columns: Column[] = [
  { key: 'name', label: 'Name', className: 'resource-name' },
  { key: 'provisioner', label: 'Provisioner' },
  { key: 'reclaimPolicy', label: 'Reclaim Policy' },
  { key: 'default', label: 'Default' },
  { key: 'age', label: 'Age' }
]

const contextMenuItems: ContextMenuItem[] = [
  { label: 'Show Details', action: 'select' }
]

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
function onSelect(item: StorageClassInfo) { emit('select', item) }

function onContextAction(action: string, item: StorageClassInfo) {
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
