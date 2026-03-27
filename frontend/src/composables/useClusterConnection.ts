// composables/useClusterConnection.ts

import { ref, computed, type Ref } from 'vue'

export function useClusterConnection(deps: {
  clusters: Ref<any[]>
  activeClusterId: Ref<string>
  error: Ref<string>
  selectedNamespace: Ref<string>
  currentView: Ref<string>
  connectToCluster: (id: string) => Promise<void>
  loadNamespaces: () => Promise<void>
  refreshData: () => Promise<void>
  k8sDisconnect: () => void
  loadClusters: () => Promise<void>
  addClusterFromFile: (name: string, icon: string, color: string) => Promise<any>
  addClusterFromText: (text: string, name: string, icon: string, color: string) => Promise<any>
  deleteCluster: (id: string) => Promise<void>
  updateCluster: (id: string, name: string, icon: string, color: string) => Promise<void>
  autoConnect: () => Promise<boolean>
}) {
  const {
    clusters, activeClusterId, error,
    selectedNamespace, currentView,
    connectToCluster, loadNamespaces, refreshData,
    k8sDisconnect, loadClusters,
    addClusterFromFile, addClusterFromText,
    deleteCluster, updateCluster, autoConnect,
  } = deps

  const connecting = ref(false)
  const showAddCluster = ref(false)

  const activeClusterMeta = computed(() =>
    (clusters.value || []).find((c: any) => c.id === activeClusterId.value) || null
  )

  async function selectCluster(id: string) {
    if (activeClusterId.value === id) return
    try {
      connecting.value = true
      error.value = ''
      await connectToCluster(id)
      await loadNamespaces()
      selectedNamespace.value = ''
      currentView.value = 'overview'
      await refreshData()
    } catch (e: any) {
      error.value = 'Ошибка подключения: ' + (e.message || e)
    } finally {
      connecting.value = false
    }
  }

  async function addFromFile(name: string, icon: string, color: string) {
    const result = await addClusterFromFile(name, icon, color)
    if (result) {
      showAddCluster.value = false
      await selectCluster(result.id)
    }
  }

  async function addFromText(text: string, name: string, icon: string, color: string) {
    const result = await addClusterFromText(text, name, icon, color)
    if (result) {
      showAddCluster.value = false
      await selectCluster(result.id)
    }
  }

  function disconnect() {
    k8sDisconnect()
    activeClusterId.value = ''
    loadClusters()
  }

  async function initialize() {
    await loadClusters()
    const connected = await autoConnect()
    if (connected) {
      await loadNamespaces()
      currentView.value = 'overview'
      await refreshData()
    }
  }

  return {
    connecting,
    showAddCluster,
    activeClusterMeta,
    selectCluster,
    addFromFile,
    addFromText,
    disconnect,
    deleteCluster,
    updateCluster,
    initialize,
  }
}