import { useK8s } from './useK8s'
import { useNotifications } from './useNotifications'
import { globalTabbedPanel } from './useTabbedPanel'
import { GetResourceYAML } from '../../wailsjs/go/main/App'

export interface ResourceActionContext {
  resourceType: string
  resource: any
  namespace?: string
  name?: string
}

export interface UseResourceActionsReturn {
  // Основные действия
  deleteResource: (resourceType: string, namespace: string, name: string) => Promise<void>
  editResource: (resourceType: string, namespace: string, name: string) => Promise<void>
  
  // Действия для подов
  openShell: (namespace: string, podName: string, container?: string) => void
  openAttach: (namespace: string, podName: string, container?: string) => void
  openLogs: (namespace: string, podName: string, container?: string) => void
  evictPod: (namespace: string, podName: string) => void
  
  // Действия для deployments
  scaleDeployment: (namespace: string, name: string, replicas: number) => Promise<void>
  restartDeployment: (namespace: string, name: string) => Promise<void>
  
  // Утилиты
  copyToClipboard: (text: string) => void
  formatAge: (timestamp: string) => string
  formatTimestamp: (timestamp: string) => string
}

/**
 * Универсальный composable для действий с ресурсами Kubernetes
 */
export function useResourceActions(): UseResourceActionsReturn {
  const k8s = useK8s()
  const notifications = useNotifications()

  /**
   * Удаление ресурса
   */
  async function deleteResource(resourceType: string, namespace: string, name: string): Promise<void> {
    const actions: Record<string, () => Promise<void>> = {
      pod: () => k8s.deletePodByName(namespace, name),
      deployment: () => k8s.deleteDeploymentByName(namespace, name),
      serviceaccount: () => k8s.deleteServiceAccount(namespace, name),
      role: () => k8s.deleteRole(namespace, name),
      rolebinding: () => k8s.deleteRoleBinding(namespace, name),
      clusterrole: () => k8s.deleteClusterRole(name),
      clusterrolebinding: () => k8s.deleteClusterRoleBinding(name),
      job: () => k8s.deleteJob(namespace, name),
      daemonset: () => k8s.deleteDaemonSet(namespace, name),
    }

    const action = actions[resourceType.toLowerCase()]
    if (!action) {
      console.warn(`Delete action not found for resource type: ${resourceType}`)
      throw new Error(`Delete not implemented for: ${resourceType}`)
    }

    await action()
    notifications.notifySuccess(`${resourceType} "${name}" deleted`)
    await k8s.refreshData()
  }

  /**
   * Редактирование ресурса (открытие YAML)
   */
  async function editResource(resourceType: string, namespace: string, name: string): Promise<void> {
    try {
      const yaml = await GetResourceYAML(namespace, resourceType, name, true)
      globalTabbedPanel.openYamlTab(resourceType, name, namespace, yaml)
    } catch (e: any) {
      notifications.notifyError(`Error loading YAML: ${e.message || e}`)
    }
  }

  /**
   * Открыть shell в поде
   */
  function openShell(namespace: string, podName: string, container?: string) {
    const containerName = container || podName
    globalTabbedPanel.openTerminalTab('pod', podName, namespace, podName, containerName, 'shell')
  }

  /**
   * Открыть attach к поду
   */
  function openAttach(namespace: string, podName: string, container?: string) {
    const containerName = container || podName
    globalTabbedPanel.openTerminalTab('pod', podName, namespace, podName, containerName, 'attach')
  }

  /**
   * Открыть логи пода
   */
  function openLogs(namespace: string, podName: string, container?: string) {
    // Для pod используем первый контейнер если не указан
    const containerName = container || podName
    globalTabbedPanel.openLogsTab('pod', podName, namespace, [{ name: containerName }], containerName)
  }

  /**
   * Evict pod
   */
  function evictPod(namespace: string, podName: string) {
    // TODO: реализовать evict через backend
    console.log('Evict pod:', namespace, podName)
  }

  /**
   * Scale deployment
   */
  async function scaleDeployment(namespace: string, name: string, replicas: number): Promise<void> {
    await k8s.scaleDeployment(namespace, name, replicas)
    notifications.notifySuccess(`Deployment "${name}" scaled to ${replicas} replicas`)
    await k8s.refreshData()
  }

  /**
   * Restart deployment
   */
  async function restartDeployment(namespace: string, name: string): Promise<void> {
    await k8s.restartDeployment(namespace, name)
    notifications.notifySuccess(`Deployment "${name}" restarted`)
    await k8s.refreshData()
  }

  /**
   * Копировать в буфер обмена
   */
  function copyToClipboard(text: string) {
    navigator.clipboard.writeText(text)
      .then(() => notifications.notifySuccess('Copied to clipboard'))
      .catch(() => notifications.notifyError('Failed to copy'))
  }

  /**
   * Форматировать возраст
   */
  function formatAge(timestamp: string): string {
    if (!timestamp) return ''
    const date = new Date(timestamp)
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    const seconds = Math.floor(diff / 1000)
    const minutes = Math.floor(seconds / 60)
    const hours = Math.floor(minutes / 60)
    const days = Math.floor(hours / 24)

    if (days > 0) return `${days}d${hours % 24}h`
    if (hours > 0) return `${hours}h${minutes % 60}m`
    if (minutes > 0) return `${minutes}m${seconds % 60}s`
    return `${seconds}s`
  }

  /**
   * Форматировать timestamp
   */
  function formatTimestamp(timestamp: string): string {
    if (!timestamp) return ''
    return new Date(timestamp).toLocaleString()
  }

  return {
    deleteResource,
    editResource,
    openShell,
    openAttach,
    openLogs,
    evictPod,
    scaleDeployment,
    restartDeployment,
    copyToClipboard,
    formatAge,
    formatTimestamp
  }
}

/**
 * Глобальный экземпляр для использования
 */
export const globalResourceActions = useResourceActions()
