// composables/usePodActions.ts

import { ref, type Ref } from 'vue'
import { EvictPod } from '../../wailsjs/go/main/App'

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

  function openAttachWithContainer(container: string) {
    if (!selectedPod.value) return
    logContainer.value = container
    terminalMode.value = 'attach'
    terminalOpen.value = true
  }

  function openShellWithContainer(container: string) {
    if (!selectedPod.value) return
    logContainer.value = container
    terminalMode.value = 'shell'
    terminalOpen.value = true
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
    logContainer.value = container || target.containers?.[0]?.name || ''
    logsPod.value = pod ? target : null
    logsOpen.value = true
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

  return {
    terminalOpen,
    terminalMode,
    terminalPod,
    logsOpen,
    logsPod,
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
  }
}