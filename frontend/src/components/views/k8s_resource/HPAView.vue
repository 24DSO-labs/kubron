<template>
  <K8sResourceListBaseOld
    :items="items"
    :search="search"
    resource-title="HorizontalPodAutoscalers"
    empty-message="Нет HorizontalPodAutoscalers"
    :columns="columns"
    :context-menu-items="contextMenuItems"
    @refresh="onRefresh"
    @search="onSearch"
    @select="onSelect"
    @context-action="onContextAction"
  />
</template>

<script lang="ts" setup>
import type { HPAInfo } from '../../../types'
import K8sResourceListBase from '../K8sResourceListBase_old.vue'
import type { Column, ContextMenuItem } from '../K8sResourceListBase_old.vue'
import { ref } from 'vue'

defineProps<{ items: HPAInfo[]; search: string }>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'select', item: HPAInfo): void
}>()

const selectedItem = ref<HPAInfo | null>(null)

const columns: Column[] = [
  { key: 'name', label: 'Name', className: 'resource-name' },
  { key: 'namespace', label: 'Namespace', className: 'ns-label' },
  { key: 'reference', label: 'Reference' },
  { key: 'minReplicas', label: 'Min' },
  { key: 'maxReplicas', label: 'Max' },
  { key: 'currentReplicas', label: 'Current' },
  { key: 'age', label: 'Age' }
]

const contextMenuItems: ContextMenuItem[] = [
  { label: 'Show Details', action: 'select' }
]

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
function onSelect(item: HPAInfo) { emit('select', item) }

function onContextAction(action: string, item: HPAInfo) {
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
