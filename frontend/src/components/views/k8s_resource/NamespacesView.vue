<template>
  <K8sResourceListBaseOld
    :items="items"
    :search="search"
    resource-title="Namespaces"
    empty-message="Нет Namespaces"
    :columns="columns"
    :context-menu-items="contextMenuItems"
    @refresh="onRefresh"
    @search="onSearch"
    @select="onSelect"
    @context-action="onContextAction"
  />
</template>

<script lang="ts" setup>
import type { NamespaceInfo } from '../../../types'
import K8sResourceListBase from '../K8sResourceListBase_old.vue'
import type { Column, ContextMenuItem } from '../K8sResourceListBase_old.vue'
import { ref } from 'vue'

defineProps<{ items: NamespaceInfo[]; search: string }>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'select', item: NamespaceInfo): void
}>()

const selectedItem = ref<NamespaceInfo | null>(null)

const columns: Column[] = [
  { key: 'name', label: 'Name', className: 'resource-name' },
  { key: 'status', label: 'Status' },
  { key: 'age', label: 'Age' }
]

const contextMenuItems: ContextMenuItem[] = [
  { label: 'Show Details', action: 'select' }
]

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
function onSelect(item: NamespaceInfo) { emit('select', item) }

function onContextAction(action: string, item: NamespaceInfo) {
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
