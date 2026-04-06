// composables/KubPodActions.ts
import { computed, type Ref } from 'vue'
import { useKubResourceActions } from './KubResourceActions'

interface PodActions {
  openAttachWithContainer: (container: string, pod: any) => void
  openShellWithContainer: (container: string, pod: any) => void
  openLogs: (pod: any, container?: string) => void
  evictPod: (pod: any) => void
}

interface K8sActions {
  showYAML: (kind: string, namespace: string, name: string) => void
  refreshData: () => Promise<void> | void
  deleteResource: (namespace: string, name: string) => Promise<void> | void
}

export function useKubPodActions(selectedPod: Ref<any>, podActions: PodActions, k8sActions: K8sActions) {
  const { commonContextActions, handleCommonAction } = useKubResourceActions('pod', k8sActions)
  
  const { 
    openAttachWithContainer, 
    openShellWithContainer, 
    openLogs, 
    evictPod 
  } = podActions

  // Pod-специфичные действия для контекстного меню
  const podSpecificActions = computed(() => {
    if (!selectedPod.value) return []

    const containers = selectedPod.value.containers?.filter((c: any) => !c.isInit) || []
    
    if (containers.length === 0) {
      return [{ label: 'Evict', action: 'evict' }]
    }

    return [
      {
        label: 'Attach Pod',
        submenu: containers.map((c: any) => ({
          label: c.name,
          key: `attach-${c.name}`,
          action: `attachPod:${c.name}`
        }))
      },
      {
        label: 'Shell',
        submenu: containers.map((c: any) => ({
          label: c.name,
          key: `shell-${c.name}`,
          action: `shell:${c.name}`
        }))
      },
      {
        label: 'Logs',
        submenu: containers.map((c: any) => ({
          label: c.name,
          key: `logs-${c.name}`,
          action: `logs:${c.name}`
        }))
      },
      { label: 'Evict', action: 'evict' }
    ]
  })

  // Объединяем общие + pod-специфичные действия
  const contextMenuItems = computed(() => [
    ...commonContextActions,
    ...podSpecificActions.value
  ])

  // Обработчик всех действий контекстного меню
  const handleContextAction = (action: string, pod: any) => {
    selectedPod.value = pod

    // Сначала пробуем обработать общие действия
    if (handleCommonAction(action, pod)) {
      return
    }

    // Затем Pod-специфичные действия
    if (action.startsWith('attachPod:')) {
      const container = action.substring('attachPod:'.length)
      openAttachWithContainer(container, pod)
      return
    }
    if (action.startsWith('shell:')) {
      const container = action.substring('shell:'.length)
      openShellWithContainer(container, pod)
      return
    }
    if (action.startsWith('logs:')) {
      const container = action.substring('logs:'.length)
      openLogs(pod, container)
      return
    }
    if (action === 'evict') {
      evictPod(pod)
      return
    }

    console.warn('Unhandled pod action:', action)
  }

  return {
    contextMenuItems,
    handleContextAction
  }
}