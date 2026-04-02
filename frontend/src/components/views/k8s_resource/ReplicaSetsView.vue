<template>
  <K8sResourceListBase
    :items="items"
    :search="search"
    resource-title="ReplicaSets"
    empty-message="Нет ReplicaSets"
    :columns="columns"
    :context-menu-items="contextMenuItems"
    @refresh="onRefresh"
    @search="onSearch"
    @select="onSelect"
    @context-action="onContextAction"
  />

  <!-- ReplicaSet Detail Panel -->
  <ReplicaSetDetailPanel
    v-if="selectedReplicaSet"
    :resource="selectedReplicaSet"
    @close="selectedReplicaSet = null"
    @edit="() => {}"
    @delete="() => {}"
  />
</template>

<script lang="ts" setup>
import type { ReplicaSetInfo } from '../../../types'
import K8sResourceListBase from '../K8sResourceListBase.vue'
import type { Column, ContextMenuItem } from '../K8sResourceListBase.vue'
import { defineAsyncComponent, ref } from 'vue'

const ReplicaSetDetailPanel = defineAsyncComponent(() => import('../../panels/k8s_resource/ReplicaSetDetailPanel.vue'))

defineProps<{ items: ReplicaSetInfo[]; search: string }>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'select', item: ReplicaSetInfo): void
  (e: 'edit', ns: string, name: string): void
  (e: 'delete', ns: string, name: string): void
}>()

const selectedItem = ref<ReplicaSetInfo | null>(null)
const selectedReplicaSet = ref<any | null>(null)

const columns: Column[] = [
  { key: 'name', label: 'Name', className: 'resource-name' },
  { key: 'namespace', label: 'Namespace', className: 'ns-label' },
  { key: 'desired', label: 'Desired' },
  { key: 'ready', label: 'Ready' },
  { key: 'age', label: 'Age' }
]

const contextMenuItems: ContextMenuItem[] = [
  { label: 'Show Details', action: 'select' },
  { label: 'Edit', action: 'edit' },
  { label: 'Delete', action: 'delete', danger: true }
]

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
function onSelect(item: ReplicaSetInfo) {
  selectedReplicaSet.value = item
  emit('select', item)
}

function onContextAction(action: string, item: ReplicaSetInfo) {
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
