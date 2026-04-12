import { ref, type Component, defineAsyncComponent, type AsyncComponentLoader } from 'vue'
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

// Маппинг типов ресурсов к компонентам детальных панелей ( AsyncComponentLoader)
const resourcePanelComponents: Record<string, AsyncComponentLoader> = {
  job: () => import('../components/panels/k8s_resource/JobDetailPanel.vue'),
  pod: () => import('../components/panels/k8s_resource/PodDetailPanel.vue'),
  deployment: () => import('../components/panels/k8s_resource/DeploymentDetailPanel.vue'),
  service: () => import('../components/panels/k8s_resource/ServiceDetailPanel.vue'),
  daemonset: () => import('../components/panels/k8s_resource/DaemonSetDetailPanel.vue'),
  statefulset: () => import('../components/panels/k8s_resource/StatefulSetDetailPanel.vue'),
  configmap: () => import('../components/panels/k8s_resource/ConfigMapPanel.vue'),
  secret: () => import('../components/panels/k8s_resource/SecretDetailPanel.vue'),
  namespace: () => import('../components/panels/k8s_resource/NamespaceDetailPanel.vue'),
  // ... можно добавить другие типы ресурсов
}

// Кэш для асинхронных компонентов
const componentCache: Record<string, Component> = {}

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
   * Получить компонент для детальной панели по типу ресурса
   * @param resourceType - тип ресурса
   */
  function getPanelComponent(resourceType: string): Component | null {
    const type = resourceType.toLowerCase()
    
    // Проверяем кэш
    if (componentCache[type]) {
      return componentCache[type]
    }
    
    const componentLoader = resourcePanelComponents[type]
    if (!componentLoader) {
      console.warn(`No detail panel component found for resource type: ${resourceType}`)
      return null
    }
    
    // Создаём асинхронный компонент и кэшируем
    const asyncComponent = defineAsyncComponent(componentLoader)
    componentCache[type] = asyncComponent
    return asyncComponent
  }

  /**
   * Универсальный обработчик для открытия деталей ресурса
   * @param resourceType - тип ресурса
   * @param resource - объект ресурса
   * @param additionalEvents - дополнительные события
   */
  function openResourceDetail(
    resourceType: string,
    resource: any,
    additionalEvents?: Record<string, (...args: any[]) => void>
  ) {
    const component = getPanelComponent(resourceType)
    if (!component) {
      console.error(`Cannot open detail panel: no component found for ${resourceType}`)
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

    openPanel({
      component,
      resource,
      events,
      props: {}
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
    getPanelComponent,
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
