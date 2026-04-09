import { ref, computed, defineAsyncComponent } from 'vue'
import useResourceListBase from './useResourceListBase'
import { useDetailPanel } from './useDetailPanel'
import { useK8s } from './useK8s'
import { globalTabbedYamlEditor } from './useTabbedYamlEditor'
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
    console.log('Context action:', action, 'for item:', item)

    if (action.startsWith('logs:')) {
      const container = action.split(':')[1]
      console.log('Opening logs for container', container, 'in', resourceType, item.name)
      // logsPanel.openLogs({ resource: item, selectedContainer: container, containers: item.containers })
    } else if (action === 'details') {
      onSelect(item)
    } else {
      switch (action) {
        case 'logs':
          console.log('Opening logs for', resourceType, '(no specific container):', item.name)
          // logsPanel.openLogs({ resource: item, selectedContainer: 'default' })
          break
        case 'delete':
          console.log('Deleting', resourceType, ':', item.name)
          // k8s.DeleteJob(item.namespace, item.name) // Адаптировать
          break
        case 'yaml':
          GetResourceYAML(item.namespace || '', resourceType, item.name, true)
            .then(yaml => {
              globalTabbedYamlEditor.openTabbedYamlEditor(
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
