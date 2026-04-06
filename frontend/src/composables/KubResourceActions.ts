// composables/KubResourceActions.ts
import { type Ref } from 'vue'

interface K8sActions {
  showYAML: (kind: string, namespace: string, name: string) => void
  refreshData: () => Promise<void> | void
  deleteResource: (namespace: string, name: string) => Promise<void> | void
}

export function useKubResourceActions(resourceKind: string, k8sActions: K8sActions) {
  const { showYAML, refreshData, deleteResource } = k8sActions

  // Общие действия для ВСЕХ k8s ресурсов
  const commonContextActions = [
    { label: 'Show Details', action: 'select' },
    { label: 'Edit', action: 'edit' },
    { label: 'Delete', action: 'delete', danger: true }
  ]

  // Обработчик общих действий
  const handleCommonAction = (action: string, resource: any): boolean => {
    switch (action) {
      case 'select':
        return false
      case 'edit':
        showYAML(resourceKind, resource.namespace || '', resource.name)
        return true
      case 'delete':
        handleDelete(resource)
        return true
      default:
        return false
    }
  }

  const handleDelete = async (resource: any) => {
    if (!confirm(`Delete ${resourceKind} ${resource.name}?`)) return

    try {
      await deleteResource(resource.namespace, resource.name)
      await refreshData()
    } catch (e: any) {
      console.error(`Failed to delete ${resourceKind}:`, e)
    }
  }

  return {
    commonContextActions,
    handleCommonAction
  }
}