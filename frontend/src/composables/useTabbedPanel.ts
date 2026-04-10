import { ref, computed } from 'vue'

export type TabType = 'logs' | 'yaml'

export interface BaseTab {
  id: string
  type: TabType
  resourceType: string
  resourceName: string
  namespace: string
  icon: string
  title: string
}

export interface LogsTab extends BaseTab {
  type: 'logs'
  selectedPodName?: string
  selectedContainer?: string
  rawLogs?: string
  loading: boolean
  pods: Array<{ name: string; containers?: Array<{ name: string }> }>
}

export interface YamlTab extends BaseTab {
  type: 'yaml'
  content: string
  originalContent: string
}

export type Tab = LogsTab | YamlTab

// Глобальное состояние
const tabs = ref<Tab[]>([])
const activeTabId = ref<string>('')

function openLogsTab(resourceType: string, resourceName: string, namespace: string, containers?: Array<{ name: string }>, selectedContainer?: string) {
  const existingTab = tabs.value.find(tab => 
    tab.type === 'logs' &&
    tab.resourceType === resourceType && 
    tab.resourceName === resourceName && 
    tab.namespace === namespace
  ) as LogsTab | undefined

  if (existingTab) {
    activeTabId.value = existingTab.id
    if (selectedContainer) existingTab.selectedContainer = selectedContainer
  } else {
    const id = `logs-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
    const newTab: LogsTab = {
      id,
      type: 'logs',
      icon: '📋',
      title: `${resourceType}/${resourceName}`,
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

function openYamlTab(resourceType: string, resourceName: string, namespace: string, content: string) {
  const existingTab = tabs.value.find(tab => 
    tab.type === 'yaml' &&
    tab.resourceType === resourceType && 
    tab.resourceName === resourceName && 
    tab.namespace === namespace
  ) as YamlTab | undefined

  if (existingTab) {
    activeTabId.value = existingTab.id
    if (content !== existingTab.content) {
      existingTab.content = content
      existingTab.originalContent = content
    }
  } else {
    const id = `yaml-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
    const newTab: YamlTab = {
      id,
      type: 'yaml',
      icon: '📄',
      title: `${resourceType}/${resourceName}`,
      resourceType,
      resourceName,
      namespace,
      content,
      originalContent: content
    }
    
    tabs.value.push(newTab)
    activeTabId.value = id
  }
}

function closeTab(tabId: string) {
  const index = tabs.value.findIndex(t => t.id === tabId)
  if (index !== -1) {
    tabs.value.splice(index, 1)
  }
  
  if (activeTabId.value === tabId && tabs.value.length > 0) {
    activeTabId.value = tabs.value[Math.max(0, index - 1)].id
  }
  
  if (tabs.value.length === 0) {
    activeTabId.value = ''
  }
}

function closeAllTabs() {
  tabs.value = []
  activeTabId.value = ''
}

function isOpen() {
  return tabs.value.length > 0
}

// Экспортируем глобальный объект
export const globalTabbedPanel = {
  tabs,
  activeTabId,
  openLogsTab,
  openYamlTab,
  closeTab,
  closeAllTabs,
  isOpen
}