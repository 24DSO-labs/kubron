import { computed } from 'vue'
import type { ComputedRef } from 'vue'

export interface StatefulSetActions {
  restart: (resource: any) => void
  logs: (resource: any) => void
}

export interface K8sActions {
  showYAML: (kind: string, namespace: string, name: string) => void
  refreshData: () => void
  deleteResource: (namespace: string, name: string) => void
}

export function useKubStatefulSetActions(
  selectedResource: ComputedRef<any>,
  resourceActions: StatefulSetActions,
  k8sActions: K8sActions
) {

  const contextMenuItems = computed(() => {
    if (!selectedResource.value) return []
    
    const items = [
      {
        label: 'Show Details',
        action: 'select',
        icon: ''
      },
      {
        label: 'Edit YAML',
        action: 'edit',
        icon: ''
      },
      {
        label: 'Restart',
        action: 'restart',
        icon: ''
      },
      {
        label: 'Logs',
        action: 'logs',
        icon: ''
      },
      { type: 'separator' as const },
      {
        label: 'Delete',
        action: 'delete',
        danger: true,
        icon: ''
      }
    ]

    return items
  })

  const handleContextAction = (action: string, resource: any) => {
    console.log('StatefulSetActions.handleContextAction called:', { action, resource })
    
    switch (action) {
      case 'select':
        // Навигация к деталям ресурса
        console.log('Navigating to StatefulSet details:', resource)
        break
        
      case 'edit':
        k8sActions.showYAML('StatefulSet', resource.namespace, resource.name)
        break
        
      case 'restart':
        resourceActions.restart(resource)
        break
        
      case 'logs':
        resourceActions.logs(resource)
        break
        
      case 'delete':
        k8sActions.deleteResource(resource.namespace, resource.name)
        break
        
      default:
        console.warn('Unknown StatefulSet action:', action)
    }
  }

  return {
    contextMenuItems,
    handleContextAction
  }
}
