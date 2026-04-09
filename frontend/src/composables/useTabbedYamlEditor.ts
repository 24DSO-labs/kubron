import { ref } from 'vue'

export interface YamlTab {
  id: string
  title: string
  icon: string
  resourceType: string
  resourceName: string
  namespace: string
  content: string
  originalContent: string
}

const tabs = ref<YamlTab[]>([])
const activeTabId = ref<string>('')

export function useTabbedYamlEditor() {
  function openTabbedYamlEditor(resourceType: string, resourceName: string, namespace: string, content: string) {
    const existingTab = tabs.value.find(tab => 
      tab.resourceType === resourceType && 
      tab.resourceName === resourceName && 
      tab.namespace === namespace
    )

    if (existingTab) {
      activeTabId.value = existingTab.id
      if (content !== existingTab.content) {
        existingTab.content = content
        existingTab.originalContent = content
      }
    } else {
      const id = `tab-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
      tabs.value.push({
        id,
        title: `${resourceType}/${resourceName}`,
        icon: '📄',
        resourceType,
        resourceName,
        namespace,
        content,
        originalContent: content
      })
      activeTabId.value = id
    }
  }

  function closeTabbedYamlEditor() {
    tabs.value = []
    activeTabId.value = ''
  }

  function isOpen() {
    return tabs.value.length > 0
  }

  return {
    tabs,
    activeTabId,
    openTabbedYamlEditor,
    closeTabbedYamlEditor,
    isOpen
  }
}

export const globalTabbedYamlEditor = useTabbedYamlEditor()
