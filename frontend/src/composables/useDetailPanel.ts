import { ref, type Component } from 'vue'
import { GetResourceYAML } from '../../wailsjs/go/main/App'
import { globalTabbedPanel } from './useTabbedPanel'

export interface DetailPanelConfig {
  component: Component
  resource: any
  props?: Record<string, any>
  events?: Record<string, (...args: any[]) => void>
}

export interface DetailPanelState {
  isOpen: boolean
  component: Component | null
  resource: any
  props: Record<string, any>
  events: Record<string, (...args: any[]) => void>
}

const detailPanelState = ref<DetailPanelState>({
  isOpen: false,
  component: null,
  resource: null,
  props: {},
  events: {}
})

// Маппинг типов ресурсов к именам компонентов (для динамического resolve)
const resourcePanelNames: Record<string, string> = {
  job: 'JobDetailPanel',
  pod: 'PodDetailPanel',
  // deployment: 'DeploymentDetailPanel',
  service: 'ServiceDetailPanel',
  // daemonset: 'DaemonSetDetailPanel',
  statefulset: 'StatefulSetDetailPanel',
  // configmap: 'ConfigMapPanel',
  secret: 'SecretDetailPanel',
  // namespace: 'NamespaceDetailPanel',
  // ... можно добавить другие типы ресурсов
}

/**
 * Универсальный composable для управления детальными панелями
 */
export function useDetailPanel() {
  /**
   * Открыть детальную панель
   * @param config - конфигурация панели
   */
  function openPanel(config: DetailPanelConfig) {
    detailPanelState.value = {
      isOpen: true,
      component: config.component,
      resource: config.resource,
      props: config.props || {},
      events: config.events || {}
    }
  }

  /**
   * Закрыть детальную панель
   */
  function closePanel() {
    detailPanelState.value = {
      isOpen: false,
      component: null,
      resource: null,
      props: {},
      events: {}
    }
  }

  /**
   * Получить имя компонента для детальной панели по типу ресурса
   * @param resourceType - тип ресурса
   */
  function getPanelComponentName(resourceType: string): string | null {
    const type = resourceType.toLowerCase()
    return resourcePanelNames[type] || null
  }

  /**
   * Универсальный обработчик для открытия деталей ресурса
   * @param resourceType - тип ресурса
   * @param resource - объект ресурса
   * @param additionalEvents - дополнительные события
   */
  async function openResourceDetail(
    resourceType: string,
    resource: any,
    additionalEvents?: Record<string, (...args: any[]) => void>
  ) {
    const componentName = getPanelComponentName(resourceType)
    if (!componentName) {
      console.error(`Cannot open detail panel: no component found for ${resourceType}`)
      return
    }

    // Динамически импортируем компонент по имени
    const component = await import(`../components/panels/k8s_resource/${componentName}.vue`)
      .then(m => m.default)
      .catch(err => {
        console.error(`Failed to load component ${componentName}:`, err)
        return null
      })

    if (!component) {
      console.error(`Cannot open detail panel: failed to load component for ${resourceType}`)
      return
    }

    const handleDelete = () => {
      const ns = resource.namespace || resource.metadata?.namespace || ''
      const name = resource.name || resource.metadata?.name
      // Используем ConfirmationDialog через глобальное событие
      // Эмитим кастомное событие для открытия диалога подтверждения
      const deleteEvent = new CustomEvent('kubron-delete-resource', {
        detail: { resourceType, namespace: ns, name }
      })
      window.dispatchEvent(deleteEvent)
      closePanel()
    }

    const handleContextAction = (action: string) => {
      console.log('Context action from detail panel:', action, resourceType, resource.name)
      if (action === 'delete') {
        handleDelete()
      } else if (action === 'yaml' || action === 'edit') {
        // Edit уже обрабатывается отдельной кнопкой
      } else if (action === 'retry') {
        // Retry для Job - используем универсальное событие
        const actionEvent = new CustomEvent('kubron-context-action', {
          detail: { action: 'retry', resourceType, resource }
        })
        window.dispatchEvent(actionEvent)
      } else {
        // Другие действия (logs, shell, attach, etc.)
        // Эмитим событие для обработки в App.vue
        const actionEvent = new CustomEvent('kubron-context-action', {
          detail: { action, resourceType, resource }
        })
        window.dispatchEvent(actionEvent)
      }
    }

    const baseEvents: Record<string, (...args: any[]) => void> = {
      close: closePanel,
      edit: () => {
        const ns = resource.namespace || resource.metadata?.namespace || ''
        const name = resource.name || resource.metadata?.name
        GetResourceYAML(ns, resourceType, name, true)
          .then(yaml => {
            globalTabbedPanel.openYamlTab(
              resourceType,
              name,
              ns,
              yaml
            )
          })
          .catch(err => {
            console.error('Error loading YAML:', err)
          })
      },
      delete: handleDelete,
      contextAction: handleContextAction
    }

    const events = {
      ...baseEvents,
      ...additionalEvents
    }

    // Подготавливаем специфические props для разных типов компонентов
    let componentProps: Record<string, any> = {}

    // Для PodDetailPanel передаем resource как pod и добавляем statusClass
    if (componentName === 'PodDetailPanel') {
      componentProps = {
        pod: resource,
        statusClass: (s: string) => {
          // Базовая реализация statusClass для подов
          const statusLower = s?.toLowerCase() || ''
          
          if (statusLower === 'running') return 'status-running'
          if (statusLower === 'succeeded') return 'status-succeeded'
          if (statusLower === 'completed') return 'status-completed'
          if (statusLower === 'pending') return 'status-pending'
          if (statusLower === 'failed') return 'status-failed'
          if (statusLower === 'crashed' || statusLower === 'crashloopbackoff') return 'status-crashed'
          if (statusLower === 'imagepullbackoff') return 'status-warning'
          if (statusLower === 'evicted') return 'status-failed'
          if (statusLower === 'terminating') return 'status-terminating'
          if (statusLower === 'unknown') return 'status-unknown'
          if (statusLower === 'progressdeadlineexceeded') return 'status-warning'
          if (statusLower === 'replicafailure') return 'status-failed'
          
          return 'status-unknown'
        }
      }
    }

    openPanel({
      component,
      resource,
      events,
      props: componentProps
    })
  }

  /**
   * Проверить, открыта ли панель
   */
  function isPanelOpen() {
    return detailPanelState.value.isOpen
  }

  /**
   * Получить текущее состояние панели
   */
  function getPanelState() {
    return detailPanelState.value
  }

  return {
    detailPanelState: detailPanelState,
    openPanel,
    closePanel,
    isPanelOpen,
    getPanelState,
    getPanelComponentName,
    openResourceDetail
  }
}

/**
 * Глобальный экземпляр для использования в App.vue
 */
export const globalDetailPanel = useDetailPanel()

/**
 * Composable для автоматического открытия деталей в ResourceListBase
 */
export function useResourceDetailOpener() {
  const { openResourceDetail } = useDetailPanel()

  /**
   * Автоматическое открытие деталей на основе типа ресурса
   * @param resourceType - тип ресурса
   * @param item - объект ресурса
   */
  function openDetails(resourceType: string, item: any) {
    openResourceDetail(resourceType, item)
  }

  return {
    openDetails
  }
}
