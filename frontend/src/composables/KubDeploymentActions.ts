import { computed, type Ref } from 'vue'
import { useKubResourceActions } from './KubResourceActions'

// Интерфейс для специфичных действий Deployment
interface DeploymentActions {
  scale: (dep: any) => void
  restart: (ns: string, name: string) => void
  logs: (ns: string, name: string) => void
}

interface K8sActions {
  showYAML: (kind: string, namespace: string, name: string) => void
  refreshData: () => Promise<void> | void
  deleteResource: (namespace: string, name: string) => Promise<void> | void
}

export function useKubDeploymentActions(
  selectedResource: Ref<any>,
  resourceActions: DeploymentActions,
  k8sActions: K8sActions
) {
  // Передаём тип ресурса в базовый композабл (lowercase)
  const { commonContextActions, handleCommonAction } = useKubResourceActions('deployment', k8sActions)

  // Специфичные действия для контекстного меню
  const specificActions = computed(() => {
    return [
      { label: 'Scale', action: 'scale' },
      { label: 'Restart', action: 'restart' },
      { label: 'Logs', action: 'logs' }
    ]
  })

  // Объединяем общие + специфичные
  const contextMenuItems = computed(() => [
    ...commonContextActions,
    ...specificActions.value
  ])

  // Обработчик всех действий
  const handleContextAction = (action: string, resource: any) => {
    selectedResource.value = resource

    // Сначала общие действия (select, edit, delete)
    if (handleCommonAction(action, resource)) {
      return
    }

    // Специфичные действия
    switch (action) {
      case 'scale':
        resourceActions.scale(resource)
        break
      case 'restart':
        resourceActions.restart(resource.namespace, resource.name)
        break
      case 'logs':
        resourceActions.logs(resource.namespace, resource.name)
        break
      default:
        console.warn(`Unhandled deployment action:`, action)
    }
  }

  return {
    contextMenuItems,
    handleContextAction
  }
}
