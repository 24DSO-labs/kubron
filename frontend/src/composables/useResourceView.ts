import { ref, computed, defineAsyncComponent } from 'vue'
import useResourceListBase from './useResourceListBase'
import { useDetailPanel } from './useDetailPanel'
import { useK8s } from './useK8s'
import { globalTabbedPanel } from './useTabbedPanel'
import { GetResourceYAML } from '../../wailsjs/go/main/App'

// Абсолютный путь от src/ — работает ВСЕГДА
const panelModules = import.meta.glob('/src/panels/k8s_resource/*.vue')

export default function useResourceView(resourceType: string, showUniversalDetailFn?: (component: any, resource: any) => void) {
  const { mergedColumns, mergedMenuItems } = useResourceListBase(resourceType)

  const { openResourceDetail } = useDetailPanel()

  const onSelect = (item: any) => {
    openResourceDetail(resourceType, item)
  }

  const onContextAction = (action: string, item: any) => {
    // Обработка действий с подменю контейнеров
    if (action.startsWith('logs:')) {
      const container = action.split(':')[1]
      globalTabbedPanel.openLogsTab(resourceType, item.name, item.namespace || '', item.containers, container)
      return
    }
    if (action.startsWith('shell:')) {
      const container = action.split(':')[1]
      globalTabbedPanel.openTerminalTab(resourceType, 'shell', item.namespace || '', item.name, container, 'shell')
      return
    }
    if (action.startsWith('attach:')) {
      const container = action.split(':')[1]
      globalTabbedPanel.openTerminalTab(resourceType, 'attach', item.namespace || '', item.name, container, 'attach')
      return
    }
    
    if (action === 'details') {
      onSelect(item)
    } else {
      switch (action) {
        case 'logs':
          globalTabbedPanel.openLogsTab(resourceType, item.name, item.namespace || '', item.containers)
          break
        case 'shell':
          // Если нет контейнеров, открываем первый доступный
          if (item.containers && item.containers.length > 0) {
            const container = item.containers[0].name
            globalTabbedPanel.openTerminalTab(resourceType, 'shell', item.namespace || '', item.name, container, 'shell')
          }
          break
        case 'attach':
          if (item.containers && item.containers.length > 0) {
            const container = item.containers[0].name
            globalTabbedPanel.openTerminalTab(resourceType, 'attach', item.namespace || '', item.name, container, 'attach')
          }
          break
        case 'delete':
          console.log('Deleting', resourceType, ':', item.name)
          // k8s.DeleteJob(item.namespace, item.name) // Адаптировать
          break
        case 'yaml':
          GetResourceYAML(item.namespace || '', resourceType, item.name, true)
            .then(yaml => {
              globalTabbedPanel.openYamlTab(
                resourceType,
                item.name,
                item.namespace || '',
                yaml
              )
            })
            .catch(err => {
              console.error('Error loading YAML:', err)
            })
          break
        case 'details':
          onSelect(item)
          break
        default:
          console.log('Unhandled action:', action)
      }
    }
  }

  return {
    mergedColumns,
    mergedMenuItems,
    onContextAction,
    onSelect
  }
}
