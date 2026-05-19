<template>
  <K8sResourceListBaseOld
    :items="items"
    :search="search"
    resource-title="NetworkPolicies"
    empty-message="Нет NetworkPolicies"
    :columns="columns"
    :context-menu-items="contextMenuItems"
    @refresh="onRefresh"
    @search="onSearch"
    @select="onSelect"
    @context-action="onContextAction"
  />
</template>

<script lang="ts" setup>
import type { NetworkPolicyInfo } from '../../../types'
import K8sResourceListBase from '../K8sResourceListBase_old.vue'
import type { Column, ContextMenuItem } from '../K8sResourceListBase_old.vue'
import { ref } from 'vue'

defineProps<{ items: NetworkPolicyInfo[]; search: string }>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'select', item: NetworkPolicyInfo): void
  (e: 'edit', ns: string, name: string): void
  (e: 'delete', ns: string, name: string): void
}>()

const selectedItem = ref<NetworkPolicyInfo | null>(null)

const columns: Column[] = [
  { key: 'name', label: 'Name', className: 'resource-name' },
  { key: 'namespace', label: 'Namespace', className: 'ns-label' },
  { key: 'podSelector', label: 'Pod Selector' },
  { key: 'age', label: 'Age' }
]

const contextMenuItems: ContextMenuItem[] = [
  { label: 'Show Details', action: 'select' },
  { label: 'Edit', action: 'edit' },
  { label: 'Delete', action: 'delete', danger: true }
]

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
function onSelect(item: NetworkPolicyInfo) { emit('select', item) }

function onContextAction(action: string, item: NetworkPolicyInfo) {
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
