<template>
  <K8sResourceListBaseOld
    :items="itemsWithStatus"
    :search="search"
    resource-title="Deployments"
    empty-message="Нет Deployments"
    :columns="columns"
    :context-menu-items="contextMenuItems"
    @refresh="onRefresh"
    @search="onSearch"
    @select="onSelect"
    @context-open="onContextOpen"
    @context-action="onContextAction"
  />
</template>

<script lang="ts" setup>
import type { DeploymentInfo } from '../../../types'
import K8sResourceListBase from '../K8sResourceListBase_old.vue'
import type { Column } from '../K8sResourceListBase_old.vue'
import { ref, computed } from 'vue'
import { useKubDeploymentActions } from '../../../composables/KubDeploymentActions'

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

// Deployment действия — передаем через emit наверх
const resourceActions = {
  scale: (dep: DeploymentInfo) => {
    emit('scale', dep, 1)
  },
  restart: (ns: string, name: string) => {
    emit('restart', ns, name)
  },
  logs: (ns: string, name: string) => {
    emit('logs', ns, name)
  }
}

// Общие k8s действия
const k8sActions = {
  showYAML: (kind: string, namespace: string, name: string) => {
    emit('edit', namespace, name)
  },
  refreshData: () => {
    emit('refresh')
  },
  deleteResource: (namespace: string, name: string) => {
    emit('delete', namespace, name)
  }
}

// Используем новый композабл
const { contextMenuItems, handleContextAction } = useKubDeploymentActions(
  selectedDeployment,
  resourceActions,
  k8sActions
)

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
  { key: 'pods', label: 'Pods', 
    formatter: (item) => `${item.readyReplicas}/${item.replicas}`,
    colorFn: (item) => item.readyReplicas === item.replicas ? '#4ade80' : '#facc15'
  },
  { key: 'updatedReplicas', label: 'Replicas' },
  { key: 'age', label: 'Age' },
  { key: 'status', label: 'Status' }
]

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
function onSelect(dep: DeploymentInfo) {
  emit('select', dep)
}

function onContextOpen(dep: DeploymentInfo) {
  selectedDeployment.value = dep
}

function onContextAction(action: string, dep: DeploymentInfo) {
  selectedDeployment.value = dep
  handleContextAction(action, dep)

  // select обрабатываем здесь
  if (action === 'select') {
    emit('select', dep)
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
