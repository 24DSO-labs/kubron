import { ref } from 'vue'
import type {
  ClusterMeta, ClusterSettings, ClusterStatus, ClusterUpdateResult, KubeconfigSource, Outcome,
} from '../types'
import { askConfirm } from '../resources/confirm'
import { errorText } from './errorText'
import { callApp } from '../platform/api'

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

// Добавление и правка возвращают Outcome: диалог показывает ошибку у себя, не теряя введённое
async function addClusterFromFile(path: string, settings: ClusterSettings): Promise<Outcome<ClusterMeta>> {
  try {
    clusterError.value = ''
    const cluster = await callApp('AddClusterFromFile', path, settings)
    await loadClusters()
    return { ok: true, value: cluster }
  } catch (e) {
    clusterError.value = errorText(e)
    return { ok: false, error: clusterError.value }
  }
}

async function addClusterFromText(text: string, settings: ClusterSettings): Promise<Outcome<ClusterMeta>> {
  try {
    clusterError.value = ''
    const cluster = await callApp('AddClusterFromText', text, settings)
    await loadClusters()
    return { ok: true, value: cluster }
  } catch (e) {
    clusterError.value = errorText(e)
    return { ok: false, error: clusterError.value }
  }
}

async function deleteCluster(id: string) {
  // Свой диалог вместо window.confirm: в Wails нативный confirm работает не на всех ОС
  const name = clusters.value.find(c => c.id === id)?.displayName ?? id
  if (!(await askConfirm(`Удалить кластер «${name}»? Его kubeconfig будет удалён из Kubron.`))) return
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

async function updateCluster(
  id: string, settings: ClusterSettings, kubeconfig: KubeconfigSource,
): Promise<Outcome<ClusterUpdateResult>> {
  try {
    clusterError.value = ''
    const result = await callApp('UpdateCluster', id, settings, kubeconfig)
    await loadClusters()
    return { ok: true, value: result }
  } catch (e) {
    clusterError.value = errorText(e)
    return { ok: false, error: clusterError.value }
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