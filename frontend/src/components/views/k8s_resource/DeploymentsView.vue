<template>
  <K8sResourceListBase
    :items="itemsWithStatus"
    :search="search"
    resource-title="Deployments"
    empty-message="Нет Deployments"
    :columns="columns"
    :context-menu-items="contextMenuItems"
    @refresh="onRefresh"
    @search="onSearch"
    @select="onSelect"
    @context-action="onContextAction"
  />
</template>

<script lang="ts" setup>
import type { DeploymentInfo } from '../../../types'
import K8sResourceListBase from '../K8sResourceListBase.vue'
import type { Column, ContextMenuItem } from '../K8sResourceListBase.vue'
import DeploymentsPodsCell from './DeploymentsPodsCell.vue'
import { ref, computed } from 'vue'

const props = defineProps<{ items: DeploymentInfo[]; search: string }>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'select', dep: DeploymentInfo): void
  (e: 'scale', dep: DeploymentInfo, delta: number): void
  (e: 'yaml', ns: string, name: string): void
  (e: 'edit', ns: string, name: string): void
  (e: 'restart', ns: string, name: string): void
  (e: 'logs', ns: string, name: string): void
  (e: 'delete', ns: string, name: string): void
}>()

const selectedDeployment = ref<DeploymentInfo | null>(null)

function getDeploymentStatus(dep: DeploymentInfo): string {
  if (!dep.conditions || dep.conditions.length === 0) {
    return 'Pending'
  }

  const availableCondition = dep.conditions.find(c => c.type === 'Available')
  const progressingCondition = dep.conditions.find(c => c.type === 'Progressing')

  if (availableCondition?.status === 'True' && progressingCondition?.status === 'True') {
    return 'Running'
  }

  if (progressingCondition?.status === 'False') {
    if (progressingCondition?.reason) {
      return progressingCondition.reason
    }
    return 'Failed'
  }

  if (progressingCondition?.reason) {
    return progressingCondition.reason
  }

  return 'Pending'
}

const itemsWithStatus = computed(() => {
  return props.items.map(dep => ({
    ...dep,
    status: getDeploymentStatus(dep)
  }))
})

const columns: Column[] = [
  { key: 'name', label: 'Name', className: 'resource-name' },
  { key: 'namespace', label: 'Namespace', className: 'ns-label' },
  { key: 'pods', label: 'Pods', component: DeploymentsPodsCell },
  { key: 'updatedReplicas', label: 'Replicas' },
  { key: 'age', label: 'Age' },
  { key: 'status', label: 'Status' }
]

const contextMenuItems: ContextMenuItem[] = [
  { label: 'Show Details', action: 'select' },
  { label: 'Scale', action: 'scale' },
  { label: 'Restart', action: 'restart' },
  { label: 'Logs', action: 'logs' },
  { label: 'Edit', action: 'edit' },
  { label: 'Delete', action: 'delete', danger: true }
]

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
function onSelect(dep: DeploymentInfo) {
  console.log('DeploymentsView.onSelect called with:', dep)
  emit('select', dep)
}

function onContextAction(action: string, dep: DeploymentInfo) {
  selectedDeployment.value = dep

  switch (action) {
    case 'select':
      emit('select', dep)
      break
    case 'scale':
      emit('scale', dep, 1)
      break
    case 'restart':
      emit('restart', dep.namespace, dep.name)
      break
    case 'logs':
      emit('logs', dep.namespace, dep.name)
      break
    case 'edit':
      emit('edit', dep.namespace, dep.name)
      break
    case 'delete':
      emit('delete', dep.namespace, dep.name)
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
</style>
