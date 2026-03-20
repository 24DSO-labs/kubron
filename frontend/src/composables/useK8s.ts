import { ref, computed } from 'vue'
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

import type {
  ContextInfo, NamespaceInfo, PodInfo, PodDetailInfo,
  DeploymentInfo, StatefulSetInfo, ReplicaSetInfo, DaemonSetInfo,
  JobInfo, CronJobInfo, ServiceInfo, IngressInfo,
  NodeInfo, EventInfo, ConfigMapInfo, SecretInfo,
  PVInfo, PVCInfo, OverviewInfo,
  PodMetricsInfo, NodeMetricsInfo, RevisionInfo, PortForwardSession,
  ServiceDetail,
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
const pvs = ref<PVInfo[]>([])
const pvcs = ref<PVCInfo[]>([])
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

const selectedPod = ref<PodDetailInfo | null>(null)
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
const selectedService = ref<ServiceDetail | null>(null)

// Port forward form
const pfLocalPort = ref(8080)
const pfPodPort = ref(80)

let autoRefreshTimer: any = null

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
function filterBySearch<T extends { name: string; namespace?: string }>(items: T[]): T[] {
  const s = search.value.toLowerCase()
  if (!s) return items
  return items.filter(x =>
    x.name.toLowerCase().includes(s) ||
    (x.namespace && x.namespace.toLowerCase().includes(s))
  )
}

const filteredPods = computed(() => filterBySearch(pods.value))
const filteredDeployments = computed(() => filterBySearch(deployments.value))
const filteredStatefulSets = computed(() => filterBySearch(statefulsets.value))
const filteredReplicaSets = computed(() => filterBySearch(replicasets.value))
const filteredDaemonSets = computed(() => filterBySearch(daemonsets.value))
const filteredJobs = computed(() => filterBySearch(jobs.value))
const filteredCronJobs = computed(() => filterBySearch(cronjobs.value))
const filteredServices = computed(() => filterBySearch(services.value))
const filteredIngresses = computed(() => filterBySearch(ingresses.value))
const filteredConfigMaps = computed(() => filterBySearch(configmaps.value))
const filteredSecrets = computed(() => filterBySearch(secrets.value))
const filteredPVCs = computed(() => filterBySearch(pvcs.value))

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
    await refreshData()
    startAutoRefresh()
  } catch (e: any) { error.value = 'Ошибка подключения: ' + (e.message || e) }
  finally { connecting.value = false }
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
}

async function loadNamespaces() {
  try { namespaces.value = await GetNamespaces() }
  catch (e: any) { error.value = 'Namespaces error: ' + (e.message || e) }
}

// --- Navigation ---
async function switchView(view: string) {
  currentView.value = view; search.value = ''
  selectedPod.value = null; selectedConfigMap.value = null
  yamlContent.value = ''; execOutput.value = ''; yamlEditing.value = false
  await refreshData()
}

async function onNamespaceChange() { await refreshData() }

// --- Data Loading ---
async function refreshData() {
  loading.value = true
  try {
    const ns = selectedNamespace.value
    switch (currentView.value) {
      case 'overview': overview.value = await GetClusterOverview(); break
      case 'pods':   pods.value = await GetPods(ns); loadPodMetrics(); break
      case 'deployments': deployments.value = await GetDeployments(ns); break
      case 'statefulsets': statefulsets.value = await GetStatefulSets(ns); break
      case 'replicasets': replicasets.value = await GetReplicaSets(ns); break
      case 'daemonsets': daemonsets.value = await GetDaemonSets(ns); break
      case 'jobs': jobs.value = await GetJobs(ns); break
      case 'cronjobs': cronjobs.value = await GetCronJobs(ns); break
      case 'services': services.value = await GetServices(ns); break
      case 'ingresses': ingresses.value = await GetIngresses(ns); break
      case 'nodes':  nodes.value = await GetNodes(); loadNodeMetrics(); break
      case 'events': events.value = await GetEvents(ns); break
      case 'configmaps': configmaps.value = await GetConfigMaps(ns); break
      case 'secrets': secrets.value = await GetSecrets(ns); break
      case 'pv': pvs.value = await GetPersistentVolumes(); break
      case 'pvc': pvcs.value = await GetPersistentVolumeClaims(ns); break
    }
  } catch (e: any) { error.value = 'Ошибка: ' + (e.message || e) }
  finally { loading.value = false }
}

// --- Auto Refresh ---
function startAutoRefresh() {
  stopAutoRefresh()
  autoRefreshTimer = setInterval(() => { refreshData() }, 15000)
}
function stopAutoRefresh() {
  if (autoRefreshTimer) { clearInterval(autoRefreshTimer); autoRefreshTimer = null }
}

// --- Pod Detail ---
async function openPodDetail(pod: PodInfo) {
  try {
    selectedPod.value = await GetPodDetail(pod.namespace, pod.name) as any
    selectedConfigMap.value = null; yamlContent.value = ''
    podLogs.value = ''; execOutput.value = ''; execCmd.value = ''
    yamlEditing.value = false
    if (selectedPod.value?.containers?.length) {
      logContainer.value = selectedPod.value.containers[0].name
    }
  } catch (e: any) { error.value = 'Ошибка: ' + (e.message || e) }
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
    yamlContent.value = await GetResourceYAML(namespace, kind, name)
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
async function startPortForward() {
  if (!selectedPod.value) return
  try {
    await StartPortForward(
      selectedPod.value.namespace,
      selectedPod.value.name,
      pfLocalPort.value,
      pfPodPort.value
    )
    await refreshPortForwards()
  } catch (e: any) { error.value = 'Port forward error: ' + (e.message || e) }
}

async function stopPortForward(id: string) {
  try {
    await StopPortForward(id)
    await refreshPortForwards()
  } catch (e: any) { error.value = 'Stop PF error: ' + (e.message || e) }
}

async function refreshPortForwards() {
  try {
    portForwards.value = await ListPortForwards()
  } catch (e: any) { portForwards.value = [] }
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
    // Пробуем StartServicePortForward, если доступен
    const app = await import('../../wailsjs/go/main/App')
    if (app.StartServicePortForward) {
      await app.StartServicePortForward(namespace, service, port, port)
    } else {
      // Fallback
      await app.StartPortForward(namespace, service, port, port)
    }
    if (selectedService.value) {
      selectedService.value = await GetServiceDetail(namespace, service) as any
    }
    await refreshPortForwards()
  } catch (e: any) { error.value = 'Port forward error: ' + (e.message || e) }
}

// --- Export ---
export function useK8s() {
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
    filteredSecrets, filteredPVCs,

    // Helpers
    statusClass, pct,

    // Actions
    loadContexts, connect, disconnect, loadNamespaces,
    switchView, onNamespaceChange, refreshData,
    openPodDetail, loadLogs, deletePod, runExec,
    doScale, restartDeployment, loadRevisions, rollbackDeployment,
    openConfigMapDetail, showYAML, applyYAML,
    loadPodMetrics, loadNodeMetrics,
    startPortForward, stopPortForward, refreshPortForwards,
    openServiceDetail, forwardServicePort,
  }
}
