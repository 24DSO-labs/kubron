// composables/usePodActions.ts

import { ref, type Ref } from 'vue'
import { EvictPod } from '../../wailsjs/go/main/App'

export interface ActionTab {
  id: string
  type: 'terminal' | 'logs'
  mode?: 'shell' | 'attach'
  namespace: string
  pod: string
  podObj?: any
  container: string
  title: string
}

export function usePodActions(deps: {
  selectedPod: Ref<any>
  logContainer: Ref<string>
  error: Ref<string>
  refreshData: () => Promise<void>
  deletePodByName: (ns: string, name: string) => Promise<void>
  showYAML: (type: string, ns: string, name: string) => void
}) {
  const {
    selectedPod, logContainer, error,
    refreshData, deletePodByName, showYAML,
  } = deps

  // Tabbed Actions Panel
  const tabbedActionsOpen = ref(false)
  const tabbedActionsPanelRef = ref<any | null>(null)
  const openTabs = ref<ActionTab[]>([])
  const activeTabId = ref<string>('')
  
  const terminalOpen = ref(false)
  const terminalMode = ref<'shell' | 'attach'>('shell')
  const terminalPod = ref<any | null>(null)
  const logsOpen = ref(false)
  const logsPod = ref<any | null>(null)
  const yamlEditorOpen = ref(false)
  const yamlEditorMeta = ref({
    resourceType: '',
    resourceName: '',
    namespace: '',
  })

  function resolvePod(podOrNull?: any) {
    return podOrNull || selectedPod.value
  }

  function openAttach(pod?: any) {
    const target = resolvePod(pod)
    if (!target) return
    logContainer.value = target.containers?.[0]?.name || ''
    terminalMode.value = 'attach'
    terminalPod.value = pod ? target : null
    terminalOpen.value = true
  }

  function openShell(pod?: any) {
    const target = resolvePod(pod)
    if (!target) return
    logContainer.value = target.containers?.[0]?.name || ''
    terminalMode.value = 'shell'
    terminalPod.value = pod ? target : null
    terminalOpen.value = true
  }

  function openAttachWithContainer(container: string, pod?: any) {
    const targetPod = pod || selectedPod.value
    if (!targetPod) {
      console.warn('No pod selected for attach')
      return
    }
    
    const tabId = `attach_${targetPod.namespace}_${targetPod.name}_${container}_${Date.now()}`
    const tab: ActionTab = {
      id: tabId,
      type: 'terminal',
      mode: 'attach',
      namespace: targetPod.namespace,
      pod: targetPod.name,
      podObj: targetPod,
      container,
      title: `Attach: ${targetPod.name}/${container}`,
    }
    
    
    openTabs.value.push(tab)
    activeTabId.value = tabId
    tabbedActionsOpen.value = true
    
  }

  function openShellWithContainer(container: string, pod?: any) {
    
    const targetPod = pod || selectedPod.value
    if (!targetPod) {
      
      return
    }
    
    const tabId = `shell_${targetPod.namespace}_${targetPod.name}_${container}_${Date.now()}`
    const tab: ActionTab = {
      id: tabId,
      type: 'terminal',
      mode: 'shell',
      namespace: targetPod.namespace,
      pod: targetPod.name,
      podObj: targetPod,
      container,
      title: `Terminal: ${targetPod.name}/${container}`,
    }
    
    openTabs.value.push(tab)
    activeTabId.value = tabId
    tabbedActionsOpen.value = true
    
  }

  async function evictPod(pod?: any) {
    const target = resolvePod(pod)
    if (!target) return
    try {
      await EvictPod(target.namespace, target.name)
      error.value = ''
      if (!pod) selectedPod.value = null
      await refreshData()
    } catch (e: any) {
      error.value = 'Ошибка Evict: ' + (e.message || e)
    }
  }

  function openLogs(pod?: any, container?: string) {
    const target = resolvePod(pod)
    if (!target) return
    
    const containerName = container || target.containers?.[0]?.name || ''
    const tabId = `logs_${target.namespace}_${target.name}_${containerName}_${Date.now()}`
    const tab: ActionTab = {
      id: tabId,
      type: 'logs',
      namespace: target.namespace,
      pod: target.name,
      podObj: target,
      container: containerName,
      title: `Logs: ${target.name}/${containerName}`,
    }
    
    openTabs.value.push(tab)
    activeTabId.value = tabId
    tabbedActionsOpen.value = true
  }

  function closeTab(tabId: string) {
    openTabs.value = openTabs.value.filter(t => t.id !== tabId)
    if (activeTabId.value === tabId && openTabs.value.length > 0) {
      activeTabId.value = openTabs.value[openTabs.value.length - 1].id
    }
    if (openTabs.value.length === 0) {
      tabbedActionsOpen.value = false
    }
  }

  function editPod(pod?: any) {
    const target = resolvePod(pod)
    if (!target) return
    yamlEditorMeta.value = {
      resourceType: 'Pod',
      resourceName: target.name,
      namespace: target.namespace,
    }
    showYAML('pod', target.namespace, target.name)
    yamlEditorOpen.value = true
  }

  async function removePod(pod?: any) {
    const target = resolvePod(pod)
    if (!target) return
    try {
      await deletePodByName(target.namespace, target.name)
      if (!pod) selectedPod.value = null
      await refreshData()
    } catch (e: any) {
      error.value = 'Ошибка удаления: ' + (e.message || e)
    }
  }

  function closeTerminal() {
    terminalOpen.value = false
    terminalPod.value = null
  }

  function closeLogs() {
    logsOpen.value = false
    logsPod.value = null
  }

  function setTabbedActionsPanelRef(ref: any) {
    tabbedActionsPanelRef.value = ref
  }

  function closeTabbedActions() {
    tabbedActionsOpen.value = false
  }

  return {
    terminalOpen,
    terminalMode,
    terminalPod,
    logsOpen,
    logsPod,
    tabbedActionsOpen,
    tabbedActionsPanelRef,
    openTabs,
    activeTabId,
    yamlEditorOpen,
    yamlEditorMeta,
    openAttach,
    openShell,
    openAttachWithContainer,
    openShellWithContainer,
    evictPod,
    openLogs,
    editPod,
    removePod,
    closeTerminal,
    closeLogs,
    closeTab,
    setTabbedActionsPanelRef,
    closeTabbedActions,
  }
}
