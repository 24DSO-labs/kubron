// composables/KubDaemonSetActions.ts
import { computed, type Ref } from 'vue'
import { useKubResourceActions } from './KubResourceActions'

// Интерфейс для специфичных действий DaemonSet
interface DaemonSetActions {
  restart: (resource: any) => void
  logs: (resource: any) => void
}

interface K8sActions {
  showYAML: (kind: string, namespace: string, name: string) => void
  refreshData: () => Promise<void> | void
  deleteResource: (namespace: string, name: string) => Promise<void> | void
}

export function useKubDaemonSetActions(
  selectedResource: Ref<any>,
  resourceActions: DaemonSetActions,
  k8sActions: K8sActions
) {
  // Передаём тип ресурса в базовый композабл (lowercase)
  const { commonContextActions, handleCommonAction } = useKubResourceActions('daemonset', k8sActions)

  // Специфичные действия для контекстного меню
  const specificActions = computed(() => {
    return [
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
      case 'restart':
        resourceActions.restart(resource)
        break
      case 'logs':
        resourceActions.logs(resource)
        break
      default:
        console.warn(`Unhandled daemonset action:`, action)
    }
  }

  return {
    contextMenuItems,
    handleContextAction
  }
}
