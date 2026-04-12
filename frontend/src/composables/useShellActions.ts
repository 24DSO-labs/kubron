// composables/useShellActions.ts
// Composable для управления действиями с терминалом/attach из контекстного меню

import { ref, type Ref } from 'vue'
import { globalTabbedPanel } from './useTabbedPanel'

export interface ShellActionDeps {
  // Опционально: внешнее хранилище табов, если нужно
  openTabs?: Ref<any[]>
  activeTabId?: Ref<string>
}

export interface ShellActionResult {
  /**
   * Открыть терминал (shell) для указанного pod/container
   * @param namespace - namespace ресурса
   * @param pod - имя pod
   * @param container - имя контейнера
   * @param resourceType - тип ресурса (для заголовка)
   * @param resourceName - имя ресурса
   */
  openShell: (
    namespace: string,
    pod: string,
    container: string,
    resourceType?: string,
    resourceName?: string
  ) => Promise<void>

  /**
   * Открыть attach сессию для указанного pod/container
   * @param namespace - namespace ресурса
   * @param pod - имя pod
   * @param container - имя контейнера
   * @param resourceType - тип ресурса (для заголовка)
   * @param resourceName - имя ресурса
   */
  openAttach: (
    namespace: string,
    pod: string,
    container: string,
    resourceType?: string,
    resourceName?: string
  ) => Promise<void>

  /**
   * Обработчик действия из контекстного меню
   * @param action - строка действия вида 'shell:containerName' или 'attach:containerName'
   * @param item - объект ресурса (должен иметь namespace, name, containers)
   */
  handleContextAction: (action: string, item: any) => Promise<void>
}

/**
 * Composable для управления shell/attach действиями
 * Используется в ResourceListBase.vue для обработки действий из контекстного меню
 */
export function useShellActions(deps?: ShellActionDeps): ShellActionResult {
  const { openTabs, activeTabId } = deps || {}

  /**
   * Открыть терминал (shell) для pod
   */
  async function openShell(
    namespace: string,
    pod: string,
    container: string,
    resourceType: string = 'Pod',
    resourceName: string = pod
  ): Promise<void> {
    globalTabbedPanel.openTerminalTab(resourceType, resourceName, namespace, pod, container, 'shell')
  }

  /**
   * Открыть attach сессию для pod
   */
  async function openAttach(
    namespace: string,
    pod: string,
    container: string,
    resourceType: string = 'Pod',
    resourceName: string = pod
  ): Promise<void> {
    globalTabbedPanel.openTerminalTab(resourceType, resourceName, namespace, pod, container, 'attach')
  }

  /**
   * Обработчик действий из контекстного меню
   * Поддерживает форматы:
   * - 'shell:containerName' - открыть терминал
   * - 'attach:containerName' - открыть attach
   * - 'shell' - открыть терминал с первым контейнером
   * - 'attach' - открыть attach с первым контейнером
   */
  async function handleContextAction(action: string, item: any): Promise<void> {
    if (!action) return

    const namespace = item.namespace || 'default'
    const pod = item.name
    const containers = item.containers || []

    // Парсим действие вида 'shell:containerName' или 'attach:containerName'
    const [actionType, containerName] = action.split(':')

    let targetContainer: string

    if (containerName) {
      // Контейнер указан явно
      targetContainer = containerName
    } else {
      // Используем первый контейнер
      targetContainer = containers[0]?.name
      if (!targetContainer) {
        console.warn('No containers available for', pod)
        return
      }
    }

    // Проверяем тип действия
    if (actionType === 'shell') {
      await openShell(namespace, pod, targetContainer, item.kind || 'Pod', pod)
    } else if (actionType === 'attach') {
      await openAttach(namespace, pod, targetContainer, item.kind || 'Pod', pod)
    }
  }

  return {
    openShell,
    openAttach,
    handleContextAction,
  }
}
