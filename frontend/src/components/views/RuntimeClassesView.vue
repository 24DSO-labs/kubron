<template>
  <K8sResourceListBase
    :items="items"
    :search="search"
    resource-title="RuntimeClasses"
    empty-message="Нет RuntimeClasses"
    :columns="columns"
    :context-menu-items="contextMenuItems"
    @refresh="onRefresh"
    @search="onSearch"
    @select="onSelect"
    @context-action="onContextAction"
  />
</template>

<script lang="ts" setup>
import type { RuntimeClassInfo } from '../../types'
import K8sResourceListBase from './K8sResourceListBase.vue'
import type { Column, ContextMenuItem } from './K8sResourceListBase.vue'
import { ref } from 'vue'

defineProps<{ items: RuntimeClassInfo[]; search: string }>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'select', item: RuntimeClassInfo): void
}>()

const selectedItem = ref<RuntimeClassInfo | null>(null)

const columns: Column[] = [
  { key: 'name', label: 'Name', className: 'resource-name' },
  { key: 'handler', label: 'Handler' },
  { key: 'age', label: 'Age' }
]

const contextMenuItems: ContextMenuItem[] = [
  { label: 'Show Details', action: 'select' }
]

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
function onSelect(item: RuntimeClassInfo) { emit('select', item) }

function onContextAction(action: string, item: RuntimeClassInfo) {
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
