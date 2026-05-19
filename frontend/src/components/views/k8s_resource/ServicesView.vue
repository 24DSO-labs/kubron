<template>
  <K8sResourceListBaseOld
    :items="items"
    :search="search"
    resource-title="Services"
    empty-message="Нет Services"
    :columns="columns"
    :context-menu-items="contextMenuItems"
    @refresh="onRefresh"
    @search="onSearch"
    @select="onSelect"
    @context-action="onContextAction"
  />
</template>

<script lang="ts" setup>
import type { ServiceInfo } from '../../../types'
import K8sResourceListBase from '../K8sResourceListBase_old.vue'
import type { Column, ContextMenuItem } from '../K8sResourceListBase_old.vue'
import { ref } from 'vue'

defineProps<{ items: ServiceInfo[]; search: string }>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'yaml', ns: string, name: string): void
  (e: 'select', svc: ServiceInfo): void
}>()

const selectedService = ref<ServiceInfo | null>(null)

const columns: Column[] = [
  { key: 'name', label: 'Name', className: 'resource-name' },
  { key: 'namespace', label: 'Namespace', className: 'ns-label' },
  { key: 'type', label: 'Type' },
  { key: 'clusterIP', label: 'Cluster IP' },
  { key: 'externalIP', label: 'External IP' },
  { key: 'ports', label: 'Ports' },
  { key: 'age', label: 'Age' }
]

const contextMenuItems: ContextMenuItem[] = [
  { label: 'Show Details', action: 'select' },
  { label: 'Edit', action: 'edit' }
]

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
function onSelect(svc: ServiceInfo) { emit('select', svc) }

function onContextAction(action: string, svc: ServiceInfo) {
  selectedService.value = svc
  switch (action) {
    case 'select':
      emit('select', svc)
      break
    case 'edit':
      emit('yaml', svc.namespace, svc.name)
      break
  }
}
</script>

<style scoped>
.resource-name {
  font-weight: 500;
}

.ns-label {
  color: #868d8a;
}

.type-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 3px;
  background: rgba(74, 222, 128, 0.2);
  color: #4ade80;
  font-size: 0.8rem;
  font-weight: 500;
}
</style>
