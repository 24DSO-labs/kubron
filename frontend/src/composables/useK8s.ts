import { ref, computed, watch, nextTick } from 'vue'
import {
  GetContexts, ConnectToContext, GetActiveContext,
  GetNamespaces, GetPods, GetPodDetail, GetPodLogs, DeletePod,
  GetDeployments, ScaleDeployment, RestartDeployment,
  GetDeploymentRevisions, RollbackDeployment,
  GetStatefulSets, GetReplicaSets, GetDaemonSets, GetJobs, GetCronJobs,
  GetServiceDetail, GetServices, GetIngresses,
  GetNodes, GetEvents,
  GetConfigMaps, GetConfigMapDetail, GetSecrets,
  GetPersistentVolumes, GetPersistentVolumeClaims,
  GetClusterOverview, GetResourceYAML, ApplyYAML,
  ExecCommand,
  GetPodMetrics, GetNodeMetrics,
  StartPortForward, StopPortForward, ListPortForwards,
} from '../../wailsjs/go/main/App'

// For lazy-loaded functions (kept for optional runtime feature detection)
const dynamicImportApp = async () => {
  return await import('../../wailsjs/go/main/App')
}

import type {
  ContextInfo, NamespaceInfo, PodInfo,
  DeploymentInfo, StatefulSetInfo, ReplicaSetInfo, DaemonSetInfo,
  JobInfo, CronJobInfo, ServiceInfo, IngressInfo,
  NodeInfo, EventInfo, ConfigMapInfo, SecretInfo, ResourceQuotaInfo,
  PVInfo, PVCInfo, OverviewInfo,
  PodMetricsInfo, NodeMetricsInfo, RevisionInfo, PortForwardSession,
  ServiceDetail, StorageClassInfo,
} from '../types'

// --- State ---
const contexts = ref<ContextInfo[]>([])
const namespaces = ref<NamespaceInfo[]>([])
const pods = ref<PodInfo[]>([])
const deployments = ref<DeploymentInfo[]>([])
const statefulsets = ref<StatefulSetInfo[]>([])
const replicasets = ref<ReplicaSetInfo[]>([])
const daemonsets = ref<DaemonSetInfo[]>([])
const jobs = ref<JobInfo[]>([])
const cronjobs = ref<CronJobInfo[]>([])
const services = ref<ServiceInfo[]>([])
const ingresses = ref<IngressInfo[]>([])
const nodes = ref<NodeInfo[]>([])
const events = ref<EventInfo[]>([])
const configmaps = ref<ConfigMapInfo[]>([])
const secrets = ref<SecretInfo[]>([])
const resourcequotas = ref<ResourceQuotaInfo[]>([])
const limitranges = ref<any[]>([])
const hpa = ref<any[]>([])
const pdb = ref<any[]>([])
const priorityclasses = ref<any[]>([])
const runtimeclasses = ref<any[]>([])
const leases = ref<any[]>([])
const endpoints = ref<any[]>([])
const networkpolicies = ref<any[]>([])
const ingressclasses = ref<any[]>([])
const serviceaccounts = ref<any[]>([])
const roles = ref<any[]>([])
const rolebindings = ref<any[]>([])
const clusterroles = ref<any[]>([])
const clusterrolebindings = ref<any[]>([])
const pvs = ref<PVInfo[]>([])
const pvcs = ref<PVCInfo[]>([])
const storageclasses = ref<StorageClassInfo[]>([])
const overview = ref<OverviewInfo | null>(null)

const podMetrics = ref<PodMetricsInfo[]>([])
const nodeMetrics = ref<NodeMetricsInfo[]>([])
const revisions = ref<RevisionInfo[]>([])
const portForwards = ref<PortForwardSession[]>([])

const activeContext = ref('')
const selectedNamespace = ref('')
const currentView = ref('overview')
const search = ref('')
const error = ref('')
const connecting = ref(false)
const loading = ref(false)

const selectedPod = ref<PodInfo | null>(null)
const podLogs = ref('')
const logContainer = ref('')
const logLines = ref(100)
const execCmd = ref('')
const execOutput = ref('')

const selectedConfigMap = ref<ConfigMapInfo | null>(null)
const yamlContent = ref('')
const yamlEditing = ref(false)
const yamlEditKind = ref('')
const yamlEditNs = ref('')
const yamlEditName = ref('')
const selectedServiceAccount = ref<any | null>(null)

const selectedService = ref<ServiceDetail | null>(null)

// Port forward form
const pfLocalPort = ref(8080)
const pfPodPort = ref(80)

let autoRefreshTimer: any = null
let currentRefreshView: string = ''
let refreshAbortSignal: AbortController | null = null
let isRefreshing = false

let refreshTimeout: any = null
function debouncedRefresh() {
  if (refreshTimeout) clearTimeout(refreshTimeout)
  refreshTimeout = setTimeout(async () => {
    await refreshData()
  }, 100)
}

// --- Helpers ---
function statusClass(status: string): string {
  switch (status.toLowerCase()) {
    case 'running': case 'ready': case 'bound': case 'complete': return 'status-running'
    case 'succeeded': case 'completed': case 'available': return 'status-completed'
    case 'pending': return 'status-pending'
    case 'failed': case 'error': case 'crashloopbackoff': case 'lost': return 'status-error'
    default: return 'status-unknown'
  }
}

function pct(a: number, b: number): string {
  if (b === 0) return '0%'
  return Math.round((a / b) * 100) + '%'
}

// --- Filters ---
function filterBySearch<T extends { name: string; namespace?: string }>(items: T[] | null | undefined): T[] {
  const arr = Array.isArray(items) ? items : []
  const s = search.value.toLowerCase()
  if (!s) return arr
  return arr.filter(x =>
    x.name.toLowerCase().includes(s) ||
    (x.namespace && x.namespace.toLowerCase().includes(s))
  )
}

const filteredPods = computed(() => filterBySearch(pods.value || []))
const filteredDeployments = computed(() => filterBySearch(deployments.value || []))
const filteredStatefulSets = computed(() => filterBySearch(statefulsets.value || []))
const filteredReplicaSets = computed(() => filterBySearch(replicasets.value || []))
const filteredDaemonSets = computed(() => filterBySearch(daemonsets.value || []))
const filteredJobs = computed(() => filterBySearch(jobs.value || []))
const filteredCronJobs = computed(() => filterBySearch(cronjobs.value || []))
const filteredServices = computed(() => filterBySearch(services.value || []))
const filteredIngresses = computed(() => filterBySearch(ingresses.value || []))
const filteredConfigMaps = computed(() => filterBySearch(configmaps.value || []))
const filteredSecrets = computed(() => filterBySearch(secrets.value || []))
const filteredResourceQuotas = computed(() => filterBySearch(resourcequotas.value || []))
const filteredPVCs = computed(() => filterBySearch(pvcs.value || []))
const filteredLimitRanges = computed(() => filterBySearch(limitranges.value || []))
const filteredHPA = computed(() => filterBySearch(hpa.value || []))
const filteredPDB = computed(() => filterBySearch(pdb.value || []))
const filteredPriorityClasses = computed(() => filterBySearch(priorityclasses.value || []))
const filteredRuntimeClasses = computed(() => filterBySearch(runtimeclasses.value || []))
const filteredLeases = computed(() => filterBySearch(leases.value || []))
const filteredEndpoints = computed(() => filterBySearch(endpoints.value || []))
const filteredNetworkPolicies = computed(() => filterBySearch(networkpolicies.value || []))
const filteredIngressClasses = computed(() => filterBySearch(ingressclasses.value || []))
const filteredStorageClasses = computed(() => filterBySearch(storageclasses.value || []))
const filteredServiceAccounts = computed(() => filterBySearch(serviceaccounts.value || []))
const filteredRoles = computed(() => filterBySearch(roles.value || []))
const filteredRoleBindings = computed(() => filterBySearch(rolebindings.value || []))
const filteredClusterRoles = computed(() => filterBySearch(clusterroles.value || []))
const filteredClusterRoleBindings = computed(() => filterBySearch(clusterrolebindings.value || []))

async function openServiceAccountDetail(sa: any) {
  try {
    const app = await dynamicImportApp()
    if ((app as any).GetServiceAccountDetail) {
      selectedServiceAccount.value = await (app as any).GetServiceAccountDetail(sa.metadata.namespace, sa.metadata.name)
    } else {
      selectedServiceAccount.value = sa  // Fallback to list item if detail not available
    }
    selectedPod.value = null
    selectedConfigMap.value = null
    yamlContent.value = ''
  } catch (e: any) {
    error.value = 'Ошибка: ' + (e.message || e)
  }
}


// --- Connection ---
async function loadContexts() {
  try { error.value = ''; contexts.value = await GetContexts() }
  catch (e: any) { error.value = e.message || String(e) }
}

async function connect(name: string) {
  try {
    error.value = ''; connecting.value = true
    await ConnectToContext(name)
    activeContext.value = await GetActiveContext()
    await loadNamespaces()
    currentView.value = 'overview'
    await refreshPortForwards()
    await refreshData()
    startAutoRefresh()
  } catch (e: any) { error.value = 'Ошибка подключения: ' + (e.message || e) }
  finally { 
    await nextTick()
    connecting.value = false 
  }
}

function disconnect() {
  stopAutoRefresh()
  activeContext.value = ''
  pods.value = []; deployments.value = []; statefulsets.value = []; replicasets.value = []
  daemonsets.value = []; jobs.value = []; cronjobs.value = []
  services.value = []; ingresses.value = []; nodes.value = []
  events.value = []; configmaps.value = []; secrets.value = []
  pvs.value = []; pvcs.value = []; namespaces.value = []
  overview.value = null; selectedNamespace.value = ''
  selectedPod.value = null; selectedConfigMap.value = null
  yamlContent.value = ''; podMetrics.value = []; nodeMetrics.value = []
  portForwards.value = []
  nextTick()
}

async function loadNamespaces() {
  try {
    namespaces.value = await GetNamespaces()
    // Set default to 'all' (empty string for all namespaces) if no previous selection
    if (!selectedNamespace.value || selectedNamespace.value === '') {
      selectedNamespace.value = ''  // '' means all namespaces
    }
  }
  catch (e: any) { error.value = 'Namespaces error: ' + (e.message || e) }
}

// --- Navigation ---
function switchView(view: string) {
  search.value = ''
  selectedPod.value = null; selectedConfigMap.value = null
  yamlContent.value = ''; execOutput.value = ''; yamlEditing.value = false
  // Cancel previous request
  if (refreshAbortSignal) {
    refreshAbortSignal.abort()
    refreshAbortSignal = null
  }
  if (refreshTimeout) {
    clearTimeout(refreshTimeout)
    refreshTimeout = null
  }
  currentView.value = view
  debouncedRefresh()
}

function onNamespaceChange() {
  // Refresh immediately on namespace change, no debounce
  if (refreshTimeout) {
    clearTimeout(refreshTimeout)
    refreshTimeout = null
  }
  refreshData()
}

// --- Data Loading ---
async function refreshData() {
  // If already refreshing, abort the previous one
  if (isRefreshing) {
    if (refreshAbortSignal) {
      refreshAbortSignal.abort()
    }
  }
  isRefreshing = true
  loading.value = true
  const viewId = currentView.value
  currentRefreshView = viewId
  // Create a new AbortController for this request
  refreshAbortSignal = new AbortController()

  // Always refresh port forwards (separately to avoid blocking other data)
  refreshPortForwards().catch(() => {})

  try {
    const ns = selectedNamespace.value
    switch (viewId) {
      case 'overview':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          overview.value = await GetClusterOverview()
        }
        break
      case 'pods':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          pods.value = await GetPods(ns)
          loadPodMetrics()
        }
        break
      case 'deployments':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          deployments.value = await GetDeployments(ns)
        }
        break
      case 'statefulsets':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          statefulsets.value = await GetStatefulSets(ns)
        }
        break
      case 'replicasets':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          replicasets.value = await GetReplicaSets(ns)
        }
        break
      case 'daemonsets':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          daemonsets.value = await GetDaemonSets(ns)
        }
        break
      case 'jobs':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          jobs.value = await GetJobs(ns)
        }
        break
      case 'cronjobs':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          cronjobs.value = await GetCronJobs(ns)
        }
        break
      case 'services':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          services.value = await GetServices(ns)
        }
        break
      case 'ingresses':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          ingresses.value = await GetIngresses(ns)
        }
        break
      case 'nodes':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          nodes.value = await GetNodes()
          loadNodeMetrics()
        }
        break
      case 'events':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          events.value = await GetEvents(ns)
        }
        break
      case 'configmaps':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          configmaps.value = await GetConfigMaps(ns)
        }
        break
      case 'secrets':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          secrets.value = await GetSecrets(ns)
        }
        break
      case 'pv':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          pvs.value = await GetPersistentVolumes()
        }
        break
      case 'pvc':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          pvcs.value = await GetPersistentVolumeClaims(ns)
        }
        break
      case 'resourcequotas':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          // TBD when backend binding is added
          resourcequotas.value = []
        }
        break
      case 'limitranges':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          // TBD when backend binding is added
          limitranges.value = []
        }
        break
      case 'hpa':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          // TBD when backend binding is added
          hpa.value = []
        }
        break
      case 'pdb':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          // TBD when backend binding is added
          pdb.value = []
        }
        break
      case 'priorityclasses':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          try {
            const app = await dynamicImportApp()
            if ((app as any).GetPriorityClasses) {
              priorityclasses.value = await (app as any).GetPriorityClasses()
            } else {
              priorityclasses.value = []
            }
          } catch {
            priorityclasses.value = []
          }
        }
        break
      case 'runtimeclasses':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          try {
            const app = await dynamicImportApp()
            if ((app as any).GetRuntimeClasses) {
              runtimeclasses.value = await (app as any).GetRuntimeClasses()
            } else {
              runtimeclasses.value = []
            }
          } catch {
            runtimeclasses.value = []
          }
        }
        break
      case 'leases':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          try {
            const app = await dynamicImportApp()
            if ((app as any).GetLeases) {
              leases.value = await (app as any).GetLeases(ns)
            } else {
              leases.value = []
            }
          } catch {
            leases.value = []
          }
        }
        break
      case 'endpoints':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          try {
            const app = await dynamicImportApp()
            if ((app as any).GetEndpoints) {
              endpoints.value = await (app as any).GetEndpoints(ns)
            } else {
              endpoints.value = []
            }
          } catch {
            endpoints.value = []
          }
        }
        break
      case 'network-policies':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          try {
            const app = await dynamicImportApp()
            if ((app as any).GetNetworkPolicies) {
              networkpolicies.value = await (app as any).GetNetworkPolicies(ns)
            } else {
              networkpolicies.value = []
            }
          } catch {
            networkpolicies.value = []
          }
        }
        break
      case 'ingressclasses':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          try {
            const app = await dynamicImportApp()
            if ((app as any).GetIngressClasses) {
              ingressclasses.value = await (app as any).GetIngressClasses()
            } else {
              ingressclasses.value = []
            }
          } catch {
            ingressclasses.value = []
          }
        }
        break
      case 'storageclasses':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          try {
            const app = await dynamicImportApp()
            if ((app as any).GetStorageClasses) {
              storageclasses.value = await (app as any).GetStorageClasses()
            } else {
              storageclasses.value = []
            }
          } catch {
            storageclasses.value = []
          }
        }
        break
      case 'serviceaccounts':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          try {
            const app = await dynamicImportApp()
            if ((app as any).GetServiceAccounts) {
              const data = await (app as any).GetServiceAccounts(ns)
              serviceaccounts.value = Array.isArray(data) ? data : []
            } else {
              serviceaccounts.value = []
            }
          } catch {
            serviceaccounts.value = []
          }
        }
        break
      case 'roles':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          try {
            const app = await dynamicImportApp()
            if ((app as any).GetRoles) {
              const data = await (app as any).GetRoles(ns)
              roles.value = Array.isArray(data) ? data : []
            } else {
              roles.value = []
            }
          } catch {
            roles.value = []
          }
        }
        break
      case 'rolebindings':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          try {
            const app = await dynamicImportApp()
            if ((app as any).GetRoleBindings) {
              const data = await (app as any).GetRoleBindings(ns)
              rolebindings.value = Array.isArray(data) ? data : []
            } else {
              rolebindings.value = []
            }
          } catch {
            rolebindings.value = []
          }
        }
        break
      case 'clusterroles':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          try {
            const app = await dynamicImportApp()
            if ((app as any).GetClusterRoles) {
              const data = await (app as any).GetClusterRoles()
              clusterroles.value = Array.isArray(data) ? data : []
            } else {
              clusterroles.value = []
            }
          } catch {
            clusterroles.value = []
          }
        }
        break
      case 'clusterrolebindings':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          try {
            const app = await dynamicImportApp()
            if ((app as any).GetClusterRoleBindings) {
              const data = await (app as any).GetClusterRoleBindings()
              clusterrolebindings.value = Array.isArray(data) ? data : []
            } else {
              clusterrolebindings.value = []
            }
          } catch {
            clusterrolebindings.value = []
          }
        }
        break
      case 'portforwarding':
        if (currentRefreshView === viewId && !refreshAbortSignal.signal.aborted) {
          await refreshPortForwards()
        }
        break
    }
  } catch (e: any) {
    if (currentRefreshView === viewId && refreshAbortSignal && !refreshAbortSignal.signal.aborted) {
      error.value = 'Ошибка: ' + (e.message || e)
    }
  } finally {
    if (currentRefreshView === viewId) {
      loading.value = false
      isRefreshing = false
    }
  }
}

// --- Auto Refresh ---
function startAutoRefresh() {
  stopAutoRefresh()
  // Auto-refresh disabled - handled by App.vue instead for better control
  // autoRefreshTimer = setInterval(() => { refreshData() }, 15000)
}
function stopAutoRefresh() {
  if (autoRefreshTimer) { clearInterval(autoRefreshTimer); autoRefreshTimer = null }
}

// --- Pod Detail ---
async function openPodDetail(pod: PodInfo) {
  try {
    // Close other panels and reset selectedPod first
    selectedConfigMap.value = null
    selectedService.value = null
    yamlContent.value = ''
    podLogs.value = ''
    execOutput.value = ''
    execCmd.value = ''
    yamlEditing.value = false
    selectedPod.value = null

    // Wait for previous component to unmount
    await nextTick()

    // Use the pod directly as detail
    const podDetail = { ...pod }
    if (podDetail.containers && podDetail.containers.length > 0) {
      logContainer.value = podDetail.containers[0].name
    }

    // Wait again before setting new pod to reduce patch churn
    await nextTick()
    selectedPod.value = podDetail
  } catch (e: any) {
    error.value = 'Ошибка при открытии pod: ' + (e.message || e)
    selectedPod.value = null
  }
}

async function loadLogs() {
  if (!selectedPod.value) return
  try {
    podLogs.value = '⏳ Загрузка...'
    podLogs.value = await GetPodLogs(
      selectedPod.value.namespace, selectedPod.value.name,
      logContainer.value, logLines.value
    )
  } catch (e: any) { podLogs.value = 'Ошибка: ' + (e.message || e) }
}

async function deletePod() {
  if (!selectedPod.value) return
  if (!confirm('Удалить pod ' + selectedPod.value.name + '?')) return
  try {
    await DeletePod(selectedPod.value.namespace, selectedPod.value.name)
    selectedPod.value = null; await refreshData()
  } catch (e: any) { error.value = 'Delete error: ' + (e.message || e) }
}

async function deletePodByName(ns: string, name: string) {
  try {
    await DeletePod(ns, name)
  } catch (e: any) {
    error.value = 'Ошибка удаления пода: ' + (e.message || e)
    throw e
  }
}

async function runExec() {
  if (!selectedPod.value || !execCmd.value) return
  try {
    execOutput.value = '⏳ Выполнение...'
    execOutput.value = await ExecCommand(
      selectedPod.value.namespace, selectedPod.value.name,
      logContainer.value, execCmd.value
    )
  } catch (e: any) { execOutput.value = 'Ошибка: ' + (e.message || e) }
}

// --- Scale ---
async function doScale(dep: DeploymentInfo, delta: number) {
  const n = dep.replicas + delta
  if (n < 0) return
  try { await ScaleDeployment(dep.namespace, dep.name, n); await refreshData() }
  catch (e: any) { error.value = 'Scale error: ' + (e.message || e) }
}

async function scaleDeployment(namespace: string, name: string, replicas: number) {
  try {
    await ScaleDeployment(namespace, name, replicas)
  } catch (e: any) {
    error.value = 'Scale error: ' + (e.message || e)
    throw e
  }
}

// --- Restart / Rollback ---
async function restartDeployment(namespace: string, name: string) {
  if (!confirm('Restart deployment ' + name + '?')) return
  try {
    await RestartDeployment(namespace, name)
    error.value = ''
    await refreshData()
  } catch (e: any) { error.value = 'Restart error: ' + (e.message || e) }
}

async function loadRevisions(namespace: string, name: string) {
  try {
    revisions.value = await GetDeploymentRevisions(namespace, name)
  } catch (e: any) { error.value = 'Revisions error: ' + (e.message || e) }
}

async function rollbackDeployment(namespace: string, name: string, revision: number) {
  if (!confirm('Rollback ' + name + ' to revision ' + revision + '?')) return
  try {
    await RollbackDeployment(namespace, name, revision)
    revisions.value = []
    await refreshData()
  } catch (e: any) { error.value = 'Rollback error: ' + (e.message || e) }
}

// --- ConfigMap Detail ---
async function openConfigMapDetail(cm: ConfigMapInfo) {
  try {
    selectedConfigMap.value = await GetConfigMapDetail(cm.namespace, cm.name) as any
    selectedPod.value = null; yamlContent.value = ''; yamlEditing.value = false
  } catch (e: any) { error.value = 'Ошибка: ' + (e.message || e) }
}

// --- YAML View / Edit ---
async function showYAML(kind: string, namespace: string, name: string) {
  try {
    selectedPod.value = null; selectedConfigMap.value = null
    yamlContent.value = '⏳ Загрузка...'
    yamlEditing.value = false
    yamlEditKind.value = kind
    yamlEditNs.value = namespace
    yamlEditName.value = name
    yamlContent.value = await GetResourceYAML(namespace, kind, name, true)
  } catch (e: any) { yamlContent.value = 'Ошибка: ' + (e.message || e) }
}

async function applyYAML() {
  try {
    await ApplyYAML(yamlEditNs.value, yamlEditKind.value, yamlEditName.value, yamlContent.value)
    yamlEditing.value = false
    error.value = ''
    await refreshData()
  } catch (e: any) { error.value = 'Apply error: ' + (e.message || e) }
}

// --- Metrics ---
async function loadPodMetrics() {
  try {
    podMetrics.value = await GetPodMetrics(selectedNamespace.value)
  } catch (e: any) { podMetrics.value = [] }
}

async function loadNodeMetrics() {
  try {
    nodeMetrics.value = await GetNodeMetrics()
  } catch (e: any) { nodeMetrics.value = [] }
}

// --- Port Forward ---
async function startPortForward(namespace?: string, podName?: string, localPort?: number, podPort?: number) {
  // Support both old and new calling conventions
  const ns = namespace || selectedPod.value?.namespace
  const pod = podName || selectedPod.value?.name
  const lp = localPort !== undefined ? localPort : pfLocalPort.value
  const pp = podPort !== undefined ? podPort : pfPodPort.value

  if (!ns || !pod) return

  console.log('useK8s.startPortForward called with:', { ns, pod, lp, pp })

  try {
    console.log('Calling StartPortForward backend with params:', ns, pod, lp, pp)
    const result = await StartPortForward(ns, pod, lp, pp)
    console.log('StartPortForward completed, result:', result)
    error.value = ''
    
    // Give the backend a moment to register the forward, then refresh
    await new Promise(resolve => setTimeout(resolve, 500))
    await refreshPortForwards()
  } catch (e: any) { 
    console.error('Port forward error:', e)
    error.value = 'Port forward error: ' + (e.message || e) 
    throw e
  }
}

async function refreshPortForwards() {
  try {
    const data = await ListPortForwards()
    portForwards.value = Array.isArray(data) ? data : []
  } catch (e: any) {
    portForwards.value = []
  }
}

// --- Service Detail ---
async function openServiceDetail(svc: ServiceInfo) {
  try {
    selectedService.value = await GetServiceDetail(svc.namespace, svc.name) as any
    selectedPod.value = null
    selectedConfigMap.value = null
    yamlContent.value = ''
  } catch (e: any) { error.value = 'Ошибка: ' + (e.message || e) }
}

async function forwardServicePort(namespace: string, service: string, port: number) {
  try {
    const app = await dynamicImportApp()
    if ((app as any).StartServicePortForward) {
      await (app as any).StartServicePortForward(namespace, service, port, port)
    } else if ((app as any).StartPortForward) {
      // Fallback (if StartServicePortForward is not available)
      await (app as any).StartPortForward(namespace, service, port, port)
    }
    if (selectedService.value) {
      selectedService.value = await GetServiceDetail(namespace, service) as any
    }
    await refreshPortForwards()
  } catch (e: any) { error.value = 'Port forward error: ' + (e.message || e) }
}

// --- Delete Deployment ---
async function deleteDeploymentByName(ns: string, name: string) {
  try {
    const app = await dynamicImportApp()
    if ((app as any).DeleteDeployment) {
      await (app as any).DeleteDeployment(ns, name)
    } else {
      throw new Error('DeleteDeployment not available')
    }
  } catch (e: any) {
    error.value = 'Ошибка удаления deployment: ' + (e.message || e)
    throw e
  }
}

// --- Delete ServiceAccount ---
async function deleteServiceAccount(ns: string, name: string) {
  try {
    const app = await dynamicImportApp()
    if ((app as any).DeleteServiceAccount) {
      await (app as any).DeleteServiceAccount(ns, name)
    } else {
      throw new Error('DeleteServiceAccount not available')
    }
  } catch (e: any) {
    error.value = 'Ошибка удаления ServiceAccount: ' + (e.message || e)
    throw e
  }
}

// --- Delete Role ---
async function deleteRole(ns: string, name: string) {
  try {
    const app = await dynamicImportApp()
    if ((app as any).DeleteRole) {
      await (app as any).DeleteRole(ns, name)
    } else {
      throw new Error('DeleteRole not available')
    }
  } catch (e: any) {
    error.value = 'Ошибка удаления Role: ' + (e.message || e)
    throw e
  }
}

// --- Delete RoleBinding ---
async function deleteRoleBinding(ns: string, name: string) {
  try {
    const app = await dynamicImportApp()
    if ((app as any).DeleteRoleBinding) {
      await (app as any).DeleteRoleBinding(ns, name)
    } else {
      throw new Error('DeleteRoleBinding not available')
    }
  } catch (e: any) {
    error.value = 'Ошибка удаления RoleBinding: ' + (e.message || e)
    throw e
  }
}

 // --- Delete ClusterRole ---
async function deleteClusterRole(name: string) {
  try {
    const app = await dynamicImportApp()
    if ((app as any).DeleteClusterRole) {
      await (app as any).DeleteClusterRole(name)
    } else {
      throw new Error('DeleteClusterRole not available')
    }
  } catch (e: any) {
    error.value = 'Ошибка удаления ClusterRole: ' + (e.message || e)
    throw e
  }
}

// --- Delete ClusterRoleBinding ---
async function deleteClusterRoleBinding(name: string) {
  try {
    const app = await dynamicImportApp()
    if ((app as any).DeleteClusterRoleBinding) {
      await (app as any).DeleteClusterRoleBinding(name)
    } else {
      throw new Error('DeleteClusterRoleBinding not available')
    }
  } catch (e: any) {
    error.value = 'Ошибка удаления ClusterRoleBinding: ' + (e.message || e)
    throw e
  }
}

// --- Export ---
export function useK8s() {
  // Watch for navigation changes
  watch([currentView, selectedNamespace], () => {
    if (activeContext.value) {
      debouncedRefresh()
    }
  }, { immediate: false })

  return {
    // State
    contexts, namespaces, pods, deployments, statefulsets, replicasets,
    daemonsets, jobs, cronjobs, services, ingresses,
    nodes, events, configmaps, secrets, pvs, pvcs, overview,
    podMetrics, nodeMetrics, revisions, portForwards,
    activeContext, selectedNamespace, currentView, search,
    error, connecting, loading,
    selectedPod, podLogs, logContainer, logLines,
    execCmd, execOutput,
    selectedConfigMap, yamlContent, yamlEditing,
    yamlEditKind, yamlEditNs, yamlEditName,
    pfLocalPort, pfPodPort,
    selectedService,

    // Filtered
    filteredPods, filteredDeployments, filteredStatefulSets, filteredReplicaSets,
    filteredDaemonSets, filteredJobs, filteredCronJobs,
    filteredServices, filteredIngresses, filteredConfigMaps,
    filteredSecrets, filteredResourceQuotas, filteredPVCs,
    filteredLimitRanges, filteredHPA, filteredPDB,
    filteredPriorityClasses, filteredRuntimeClasses, filteredLeases,
    filteredEndpoints, filteredNetworkPolicies, filteredIngressClasses, filteredStorageClasses,
    filteredServiceAccounts, filteredRoles, filteredRoleBindings, filteredClusterRoles, filteredClusterRoleBindings,

    // Helpers
    statusClass, pct,

    // Actions
    loadContexts, connect, disconnect, loadNamespaces,
    switchView, onNamespaceChange, refreshData,
    openPodDetail, loadLogs, deletePod, deletePodByName,
     runExec,
    doScale, scaleDeployment, restartDeployment, loadRevisions, rollbackDeployment, deleteDeploymentByName, deleteServiceAccount, deleteRole, deleteRoleBinding, deleteClusterRole, deleteClusterRoleBinding,
    openConfigMapDetail, showYAML, applyYAML,
    loadPodMetrics, loadNodeMetrics,
    startPortForward, stopPortForward, refreshPortForwards,
    openServiceAccountDetail,
    openServiceDetail, forwardServicePort,
    getPods: GetPods,
    getNamespace, getNode, getServiceAccount, getReplicaSet, deleteReplicaSet,
  }
}

// Add missing stopPortForward function
async function stopPortForward(id: string) {
  try {
    await StopPortForward(id)
    await refreshPortForwards()
  } catch (e: any) { error.value = 'Stop PF error: ' + (e.message || e) }
}

// --- Get Resource Details ---
async function getNamespace(name: string) {
  try {
    const app = await dynamicImportApp()
    if ((app as any).GetNamespaceDetail) {
      const ns = await (app as any).GetNamespaceDetail(name)
      return formatResourceDetail(ns)
    }
    // Fallback: get from list
    const allNs = await GetNamespaces()
    const found = allNs.find(n => n.name === name)
    if (found) {
      return formatResourceDetail(found)
    }
    throw new Error('Namespace not found')
  } catch (e: any) {
    error.value = 'Error loading Namespace: ' + (e.message || e)
    throw e
  }
}

async function getNode(name: string) {
  try {
    const app = await dynamicImportApp()
    if ((app as any).GetNodeDetail) {
      const node = await (app as any).GetNodeDetail(name)
      return formatResourceDetail(node)
    }
    // Fallback: get from list
    const allNodes = await GetNodes()
    const found = allNodes.find(n => n.name === name)
    if (found) {
      return formatResourceDetail(found)
    }
    throw new Error('Node not found')
  } catch (e: any) {
    error.value = 'Error loading Node: ' + (e.message || e)
    throw e
  }
}

function formatResourceDetail(resource: any): any {
  if (!resource) return {}
  
  // Преобразуем объект в нужный формат для DetailPanel
  return {
    name: resource.name || resource.metadata?.name || '',
    namespace: resource.namespace || resource.metadata?.namespace || '',
    age: resource.age || '',
    created: resource.created || resource.createdTime || resource.metadata?.creationTimestamp || '',
    ...resource
  }
}

async function getServiceAccount(namespace: string, name: string) {
  try {
    const app = await dynamicImportApp()
    if ((app as any).GetServiceAccountDetail) {
      const sa = await (app as any).GetServiceAccountDetail(namespace, name)
      return formatResourceDetail(sa)
    }
    throw new Error('GetServiceAccountDetail not available')
  } catch (e: any) {
    error.value = 'Error loading ServiceAccount: ' + (e.message || e)
    throw e
  }
}

async function getReplicaSet(namespace: string, name: string) {
  try {
    // Загружаем YAML для ReplicaSet
    const yaml = await GetResourceYAML(namespace, 'replicaset', name, true)
    // Преобразуем в объект для деталь панели
    const rs: any = {
      name,
      namespace,
      yaml
    }
    return formatResourceDetail(rs)
  } catch (e: any) {
    error.value = 'Error loading ReplicaSet: ' + (e.message || e)
    throw e
  }
}

async function deleteReplicaSet(namespace: string, name: string) {
  try {
    const app = await dynamicImportApp()
    if ((app as any).DeleteReplicaSet) {
      await (app as any).DeleteReplicaSet(namespace, name)
    } else {
      throw new Error('DeleteReplicaSet not available')
    }
  } catch (e: any) {
    error.value = 'Ошибка удаления ReplicaSet: ' + (e.message || e)
    throw e
  }
}
