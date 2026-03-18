import { ref } from 'vue'
import type { ClusterStatus } from '../types'

const clusters = ref<ClusterStatus[]>([])
const activeClusterId = ref('')
const showAddCluster = ref(false)
const clusterError = ref('')

async function loadClusters() {
  try {
    const { GetClusterStatuses } = await import('../../wailsjs/go/main/App')
    clusters.value = await GetClusterStatuses() || []
  } catch (e: any) {
    clusters.value = []
  }
}

async function connectToCluster(id: string) {
  try {
    clusterError.value = ''
    const { ConnectToCluster } = await import('../../wailsjs/go/main/App')
    await ConnectToCluster(id)
    activeClusterId.value = id
    await loadClusters()
  } catch (e: any) {
    clusterError.value = 'Ошибка: ' + (e.message || e)
    throw e
  }
}

async function addClusterFromFile(displayName: string, icon: string, color: string) {
  try {
    clusterError.value = ''
    const { AddClusterFromFile } = await import('../../wailsjs/go/main/App')
    const result = await AddClusterFromFile(displayName, icon, color)
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
    const { AddClusterFromText } = await import('../../wailsjs/go/main/App')
    const result = await AddClusterFromText(text, displayName, icon, color)
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
  if (!confirm('Удалить кластер?')) return
  try {
    const { DeleteCluster } = await import('../../wailsjs/go/main/App')
    await DeleteCluster(id)
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
    const { UpdateCluster } = await import('../../wailsjs/go/main/App')
    await UpdateCluster(id, displayName, icon, color)
    await loadClusters()
  } catch (e: any) {
    clusterError.value = e.message || String(e)
  }
}

async function autoConnect() {
  try {
    const { GetLastActiveCluster } = await import('../../wailsjs/go/main/App')
    const lastId = await GetLastActiveCluster()
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
    addClusterFromFile,
    addClusterFromText,
    deleteCluster,
    updateCluster,
    autoConnect,
  }
}