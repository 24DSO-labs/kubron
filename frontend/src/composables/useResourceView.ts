import { ref, computed, defineAsyncComponent } from 'vue'
import useResourceListBase from './useResourceListBase'

// Абсолютный путь от src/ — работает ВСЕГДА
const panelModules = import.meta.glob('/src/panels/k8s_resource/*.vue')

export default function useResourceView(resourceType: string, showUniversalDetailFn?: (component: any, resource: any) => void) {
  const { mergedColumns, mergedMenuItems } = useResourceListBase(resourceType)

  const selectedItem = ref(null)
  const showDetail = ref(false)

  const onSelect = (item: any) => {
    selectedItem.value = item
    
    
  }

  const onContextAction = (action: string, item: any) => {
    console.log('Context action:', action, 'for item:', item)

    if (action.startsWith('logs:')) {
      const container = action.split(':')[1]
      console.log('Opening logs for container', container, 'in', resourceType, item.name)
      // logsPanel.openLogs({ resource: item, selectedContainer: container, containers: item.containers })
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
          console.log('Editing YAML for', resourceType, ':', item.name)
          // yaml.ShowYAML(resourceType, item.namespace, item.name)
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
    onSelect,
    selectedItem,
    showDetail
  }
}
