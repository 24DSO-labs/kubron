import { ref } from 'vue'

export interface LogsTab {
  id: string
  resourceType: string
  resourceName: string
  namespace: string
  selectedPodName?: string
  selectedContainer?: string
  rawLogs?: string
  loading: boolean
  pods: Array<{ name: string; containers?: Array<{ name: string }> }>
  containers?: Array<{ name: string }>
}

const tabs = ref<LogsTab[]>([])
const activeTabId = ref<string>('')

export function useTabbedLogsPanel() {
function openTabbedLogs(resourceType: string, resourceName: string, namespace: string, containers?: Array<{ name: string }>, selectedContainer?: string) {
  const existingTab = tabs.value.find(tab => 
    tab.resourceType === resourceType && 
    tab.resourceName === resourceName && 
    tab.namespace === namespace
  )

  if (existingTab) {
    activeTabId.value = existingTab.id
    if (selectedContainer) existingTab.selectedContainer = selectedContainer
  } else {
    const id = `logs-tab-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
    const newTab: LogsTab = {
      id,
      resourceType,
      resourceName,
      namespace,
      selectedContainer,
      loading: false,
      pods: [],
    }

    if (resourceType.toLowerCase() === 'pod' && containers) {
      newTab.pods = [{ name: resourceName, containers }]
      newTab.selectedPodName = resourceName
      if (!selectedContainer && containers.length > 0) {
        newTab.selectedContainer = containers[0].name
      }
    }

    tabs.value.push(newTab)
    activeTabId.value = id
  }
}

  function closeTabbedLogs() {
    tabs.value = []
    activeTabId.value = ''
  }

  function isOpen() {
    return tabs.value.length > 0
  }

  return {
    tabs,
    activeTabId,
    openTabbedLogs,
    closeTabbedLogs,
    isOpen
  }
}

export const globalTabbedLogsPanel = useTabbedLogsPanel()
