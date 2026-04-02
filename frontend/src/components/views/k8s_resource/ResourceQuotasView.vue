<template>
  <K8sResourceListBase
    :items="items"
    :search="search"
    resource-title="ResourceQuotas"
    empty-message="Нет ResourceQuotas"
    :columns="columns"
    :context-menu-items="contextMenuItems"
    @refresh="onRefresh"
    @search="onSearch"
    @select="onSelect"
    @context-action="onContextAction"
  />
</template>

<script lang="ts" setup>
import type { ResourceQuotaInfo } from '../../../types'
import K8sResourceListBase from '../K8sResourceListBase.vue'
import type { Column, ContextMenuItem } from '../K8sResourceListBase.vue'
import { ref } from 'vue'

defineProps<{ items: ResourceQuotaInfo[]; search: string }>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'select', item: ResourceQuotaInfo): void
}>()

const selectedItem = ref<ResourceQuotaInfo | null>(null)

const columns: Column[] = [
  { key: 'name', label: 'Name', className: 'resource-name' },
  { key: 'namespace', label: 'Namespace', className: 'ns-label' },
  { key: 'cpuLimit', label: 'CPU Limit' },
  { key: 'cpuUsed', label: 'CPU Used' },
  { key: 'memLimit', label: 'Memory Limit' },
  { key: 'memUsed', label: 'Memory Used' },
  { key: 'age', label: 'Age' }
]

const contextMenuItems: ContextMenuItem[] = [
  { label: 'Show Details', action: 'select' }
]

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
function onSelect(item: ResourceQuotaInfo) { emit('select', item) }

function onContextAction(action: string, item: ResourceQuotaInfo) {
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
