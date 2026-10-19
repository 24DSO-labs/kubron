import { ref } from 'vue'
import type { ClusterStatus } from '../types'
import { askConfirm } from '../resources/confirm'
import { callApp } from '../resources/wails'

const clusters = ref<ClusterStatus[]>([])
const activeClusterId = ref('')
const showAddCluster = ref(false)
const clusterError = ref('')

async function loadClusters() {
  try {
    clusters.value = await callApp('GetClusterStatuses') || []
  } catch (e: any) {
    clusters.value = []
  }
}

async function connectToCluster(id: string) {
  try {
    clusterError.value = ''
    await callApp('ConnectToCluster', id)
    activeClusterId.value = id
    await loadClusters()
  } catch (e: any) {
    clusterError.value = 'Ошибка: ' + (e.message || e)
    throw e
  }
}

async function pickKubeconfigFile(): Promise<string> {
  try {
    return (await callApp('PickKubeconfigFile')) || ''
  } catch (e: any) {
    clusterError.value = e.message || String(e)
    return ''
  }
}

async function addClusterFromFile(path: string, displayName: string, icon: string, color: string) {
  try {
    clusterError.value = ''
    const result = await callApp('AddClusterFromFile', path, displayName, icon, color)
    if (result) {
      await loadClusters()
    }
    return result
  } catch (e: any) {
    clusterError.value = e.message || String(e)
    return null
  }
}

async function addClusterFromText(text: string, displayName: string, icon: string, color: string) {
  try {
    clusterError.value = ''
    const result = await callApp('AddClusterFromText', text, displayName, icon, color)
    if (result) {
      await loadClusters()
    }
    return result
  } catch (e: any) {
    clusterError.value = e.message || String(e)
    return null
  }
}

async function deleteCluster(id: string) {
  if (!(await askConfirm('Удалить кластер? Подключение будет закрыто, сохранённый kubeconfig удалён.'))) return
  try {
    await callApp('DeleteCluster', id)
    if (activeClusterId.value === id) {
      activeClusterId.value = ''
    }
    await loadClusters()
  } catch (e: any) {
    clusterError.value = e.message || String(e)
  }
}

async function updateCluster(id: string, displayName: string, icon: string, color: string) {
  try {
    await callApp('UpdateCluster', id, displayName, icon, color)
    await loadClusters()
  } catch (e: any) {
    clusterError.value = e.message || String(e)
  }
}

async function autoConnect() {
  try {
    const lastId = await callApp('GetLastActiveCluster')
    if (lastId) {
      const exists = clusters.value.find(c => c.id === lastId)
      if (exists) {
        await connectToCluster(lastId)
        return true
      }
    }
  } catch (e) {}
  return false
}

export function useClusters() {
  return {
    clusters,
    activeClusterId,
    showAddCluster,
    clusterError,
    loadClusters,
    connectToCluster,
    pickKubeconfigFile,
    addClusterFromFile,
    addClusterFromText,
    deleteCluster,
    updateCluster,
    autoConnect,
  }
}