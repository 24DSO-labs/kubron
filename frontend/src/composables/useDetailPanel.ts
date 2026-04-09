import { ref, type Component, defineAsyncComponent } from 'vue'

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

// Маппинг типов ресурсов к компонентам детальных панелей
const resourcePanelComponents: Record<string, () => Promise<any>> = {
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
    const componentLoader = resourcePanelComponents[resourceType.toLowerCase()]
    if (!componentLoader) {
      console.warn(`No detail panel component found for resource type: ${resourceType}`)
      return null
    }
    return defineAsyncComponent(componentLoader)
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

    const baseEvents: Record<string, (...args: any[]) => void> = {
      close: closePanel,
      edit: () => {
        console.log(`Edit ${resourceType}:`, resource.name)
        // Здесь можно добавить логику редактирования YAML
      },
      delete: () => {
        if (confirm(`Delete ${resourceType} ${resource.name}?`)) {
          console.log(`Delete ${resourceType}:`, resource.name)
          // Здесь можно добавить логику удаления
          closePanel()
        }
      }
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
