<template>
  <K8sResourceListBase
    :items="items"
    :search="search"
    resource-title="Events"
    empty-message="Нет Events"
    :columns="columns"
    :context-menu-items="contextMenuItems"
    @refresh="onRefresh"
    @search="onSearch"
    @select="onSelect"
    @context-action="onContextAction"
  />
</template>

<script lang="ts" setup>
import type { EventInfo } from '../../types'
import K8sResourceListBase from './K8sResourceListBase.vue'
import type { Column, ContextMenuItem } from './K8sResourceListBase.vue'
import { ref } from 'vue'

defineProps<{ items: EventInfo[]; search: string }>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'select', item: EventInfo): void
}>()

const selectedItem = ref<EventInfo | null>(null)

const columns: Column[] = [
  { key: 'type', label: 'Type', className: 'resource-name' },
  { key: 'reason', label: 'Reason' },
  { key: 'object', label: 'Object' },
  { key: 'namespace', label: 'Namespace', className: 'ns-label' },
  { key: 'message', label: 'Message' },
  { key: 'count', label: 'Count' },
  { key: 'age', label: 'Age' }
]

const contextMenuItems: ContextMenuItem[] = [
  { label: 'Show Details', action: 'select' }
]

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
function onSelect(item: EventInfo) { emit('select', item) }

function onContextAction(action: string, item: EventInfo) {
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
