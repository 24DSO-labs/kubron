<template>
  <K8sResourceListBase
    :items="items"
    :search="search"
    resource-title="ClusterRoleBindings"
    empty-message="Нет ClusterRoleBindings"
    :columns="columns"
    :context-menu-items="contextMenuItems"
    @refresh="onRefresh"
    @search="onSearch"
    @select="onSelect"
    @context-action="onContextAction"
  />
</template>

<script lang="ts" setup>
import type { ClusterRoleBinding } from '../../types'
import K8sResourceListBase from './K8sResourceListBase.vue'
import type { Column, ContextMenuItem } from './K8sResourceListBase.vue'
import { ref } from 'vue'

defineProps<{ items: ClusterRoleBinding[]; search: string }>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'select', item: ClusterRoleBinding): void
  (e: 'edit', ns: string, name: string): void
  (e: 'delete', ns: string, name: string): void
}>()

const selectedItem = ref<ClusterRoleBinding | null>(null)

const columns: Column[] = [
  { key: 'metadata.name', label: 'Name', className: 'resource-name' },
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
function onSelect(item: ClusterRoleBinding) { emit('select', item) }

function onContextAction(action: string, item: ClusterRoleBinding) {
  selectedItem.value = item
  const name = item.metadata?.name || ''
  switch (action) {
    case 'select':
      emit('select', item)
      break
    case 'edit':
      emit('edit', '', name)
      break
    case 'delete':
      emit('delete', '', name)
      break
  }
}
</script>

<style scoped>
.resource-name {
  font-weight: 500;
}
</style>
