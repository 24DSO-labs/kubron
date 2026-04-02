<template>
  <K8sResourceListBase
    :items="items"
    :search="search"
    resource-title="RoleBindings"
    empty-message="Нет RoleBindings"
    :columns="columns"
    :context-menu-items="contextMenuItems"
    @refresh="onRefresh"
    @search="onSearch"
    @select="onSelect"
    @context-action="onContextAction"
  />
</template>

<script lang="ts" setup>
import type { RoleBinding } from '../../../types'
import K8sResourceListBase from '../K8sResourceListBase.vue'
import type { Column, ContextMenuItem } from '../K8sResourceListBase.vue'
import { ref } from 'vue'

defineProps<{ items: RoleBinding[]; search: string }>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'select', item: RoleBinding): void
  (e: 'edit', ns: string, name: string): void
  (e: 'delete', ns: string, name: string): void
}>()

const selectedItem = ref<RoleBinding | null>(null)

const columns: Column[] = [
  { key: 'metadata.name', label: 'Name', className: 'resource-name' },
  { key: 'metadata.namespace', label: 'Namespace', className: 'ns-label' },
  { key: 'roleRef.name', label: 'Role Ref' },
  { key: 'subjects.length', label: 'Bindings' },
  { key: 'age', label: 'Age' }
]

const contextMenuItems: ContextMenuItem[] = [
  { label: 'Show Details', action: 'select' },
  { label: 'Edit', action: 'edit' },
  { label: 'Delete', action: 'delete', danger: true }
]

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
function onSelect(item: RoleBinding) { emit('select', item) }

function onContextAction(action: string, item: RoleBinding) {
  selectedItem.value = item
  const name = item.metadata?.name || ''
  const ns = item.metadata?.namespace || ''
  switch (action) {
    case 'select':
      emit('select', item)
      break
    case 'edit':
      emit('edit', ns, name)
      break
    case 'delete':
      emit('delete', ns, name)
      break
  }
}
</script>

<style scoped>
.resource-name { font-weight: 500; }
.ns-label { color: #868d8a; }
</style>
