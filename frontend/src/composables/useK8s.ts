import { ref, computed, watch, nextTick, type Ref } from 'vue'
import {
  GetContexts, ConnectToContext, GetActiveContext,
  GetNamespaces, GetPods, GetPodLogs, DeletePod,
  ScaleDeployment, RestartDeployment,
  GetStatefulSets, GetJobs, GetCronJobs,
  GetServiceDetail, GetServices, GetIngresses,
  GetNodes, GetEvents,
  GetSecrets,
  GetPersistentVolumes, GetPersistentVolumeClaims,
  GetClusterOverview, GetResourceYAML, ApplyYAML,
  ExecCommand,
  GetPodMetrics, GetNodeMetrics,
  StartPortForward, StopPortForward, ListPortForwards,
} from '../../wailsjs/go/main/App'
import { callApp } from '../resources/wails'
import { useNotifications } from './useNotifications'

import type {
  ContextInfo, NamespaceInfo, PodInfo,
  StatefulSetInfo,
  JobInfo, CronJobInfo, ServiceInfo, IngressInfo,
  NodeInfo, EventInfo, SecretInfo, ResourceQuotaInfo,
  PVInfo, PVCInfo, OverviewInfo,
  PodMetricsInfo, NodeMetricsInfo, PortForwardSession,
  ServiceDetail, StorageClassInfo,
} from '../types'

/*
 * useK8s: состояние и загрузка данных для СТАРЫХ view.
 *
 * Ресурсы, перенесённые на новый каркас (src/resources/registry.ts), здесь больше не хранятся:
 * их данные грузит resources/store.ts. refreshData() для таких view ничего не делает.
 * Сейчас на каркасе: roles, rolebindings, clusterroles, clusterrolebindings, serviceaccounts,
 * configmaps, namespaces, replicasets, daemonsets, deployments.
 *
 * Правила обработки ошибок:
 *  - функции, у которых есть вызывающий с собственным catch, только бросают исключение;
 *  - функции без такого вызывающего показывают тост сами (fail()).
 *  Ref `error` остаётся только для useClusterConnection / usePodActions, которые пишут в него сами.
 */

// ─── Уведомления ───

const notifications = useNotifications()
const errText = (e: any): string => e?.message || String(e)
const fail = (prefix: string, e: any) => notifications.notifyError(`${prefix}: ${errText(e)}`)

// ─── State: списки ───

const contexts = ref<ContextInfo[]>([])
const namespaces = ref<NamespaceInfo[]>([]) // для выбора namespace в Sidebar
const pods = ref<PodInfo[]>([])
const statefulsets = ref<StatefulSetInfo[]>([])
const jobs = ref<JobInfo[]>([])
const cronjobs = ref<CronJobInfo[]>([])
const services = ref<ServiceInfo[]>([])
const ingresses = ref<IngressInfo[]>([])
const nodes = ref<NodeInfo[]>([])
const events = ref<EventInfo[]>([])
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
const pvs = ref<PVInfo[]>([])
const pvcs = ref<PVCInfo[]>([])
const storageclasses = ref<StorageClassInfo[]>([])
const overview = ref<OverviewInfo | null>(null)

const podMetrics = ref<PodMetricsInfo[]>([])
const nodeMetrics = ref<NodeMetricsInfo[]>([])
const portForwards = ref<PortForwardSession[]>([])

/** Всё, что надо очистить при отключении от кластера */
const clearableLists: Ref<any[]>[] = [
  pods, statefulsets, jobs, cronjobs, services, ingresses, nodes, events, secrets,
  resourcequotas, limitranges, hpa, pdb, priorityclasses, runtimeclasses, leases,
  endpoints, networkpolicies, ingressclasses, pvs, pvcs, storageclasses, namespaces,
  podMetrics, nodeMetrics, portForwards,
]

// ─── State: навигация и выбранное ───

const activeContext = ref('')
const selectedNamespace = ref('') // '' = все namespaces
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

const selectedService = ref<ServiceDetail | null>(null)

const yamlContent = ref('')
const yamlEditing = ref(false)
const yamlEditKind = ref('')
const yamlEditNs = ref('')
const yamlEditName = ref('')

// Форма port-forward
const pfLocalPort = ref(8080)
const pfPodPort = ref(80)

// ─── Helpers ───

function statusClass(status: string): string {
  switch ((status ?? '').toLowerCase()) {
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

function formatCpu(cpuStr: string | undefined): string {
  if (!cpuStr) return '—'
  const cpu = parseFloat(cpuStr)
  return isNaN(cpu) ? '—' : cpu.toFixed(3)
}

function formatMemory(memStr: string | undefined): string {
  if (!memStr) return '—'
  const bytes = parseFloat(memStr)
  if (isNaN(bytes)) return '—'
  return (bytes / 1024 / 1024).toFixed(1) + 'MiB'
}

function mergePodMetrics(list: PodInfo[], metrics: PodMetricsInfo[]) {
  return list.map(pod => {
    const metric = metrics.find(m => m.name === pod.name && m.namespace === pod.namespace)
    return { ...pod, cpu: formatCpu(metric?.totalCpu), memory: formatMemory(metric?.totalMem) }
  })
}

// ─── Фильтрация по поиску ───

function filterBySearch<T extends { name: string; namespace?: string }>(items: T[] | null | undefined): T[] {
  const arr = Array.isArray(items) ? items : []
  const s = search.value.toLowerCase()
  if (!s) return arr
  return arr.filter(x =>
    x.name.toLowerCase().includes(s) ||
    (x.namespace && x.namespace.toLowerCase().includes(s)),
  )
}

const filtered = <T extends { name: string; namespace?: string }>(src: Ref<T[]>) =>
  computed(() => filterBySearch(src.value))

const filteredPods = filtered(pods)
const filteredStatefulSets = filtered(statefulsets)
const filteredJobs = filtered(jobs)
const filteredCronJobs = filtered(cronjobs)
const filteredServices = filtered(services)
const filteredIngresses = filtered(ingresses)
const filteredSecrets = filtered(secrets)
const filteredResourceQuotas = filtered(resourcequotas)
const filteredPVCs = filtered(pvcs)
const filteredLimitRanges = filtered(limitranges)
const filteredHPA = filtered(hpa)
const filteredPDB = filtered(pdb)
const filteredPriorityClasses = filtered(priorityclasses)
const filteredRuntimeClasses = filtered(runtimeclasses)
const filteredLeases = filtered(leases)
const filteredEndpoints = filtered(endpoints)
const filteredNetworkPolicies = filtered(networkpolicies)
const filteredIngressClasses = filtered(ingressclasses)
const filteredStorageClasses = filtered(storageclasses)

// ─── Подключение (legacy; основной путь подключения идёт через useClusters) ───

async function loadContexts() {
  try { contexts.value = await GetContexts() }
  catch (e) { fail('Ошибка загрузки контекстов', e) }
}

async function connect(name: string) {
  try {
    connecting.value = true
    await ConnectToContext(name)
    activeContext.value = await GetActiveContext()
    await loadNamespaces()
    currentView.value = 'overview'
    await refreshPortForwards()
    await refreshData()
  } catch (e) {
    fail('Ошибка подключения', e)
  } finally {
    await nextTick()
    connecting.value = false
  }
}

function disconnect() {
  activeContext.value = ''
  for (const l of clearableLists) l.value = []
  overview.value = null
  selectedNamespace.value = ''
  selectedPod.value = null
  selectedService.value = null
  yamlContent.value = ''
  loadedViews.clear()
}

async function loadNamespaces() {
  try { namespaces.value = await GetNamespaces() }
  catch (e) { fail('Ошибка загрузки namespaces', e) }
}

// ─── Загрузка данных текущего view ───

/**
 * Загрузчик принимает namespace и isCurrent(). Результат применяется, только если
 * за время запроса не стартовал более свежий refresh: так медленный ответ
 * для старого namespace не затирает новый.
 */
type Loader = (ns: string, isCurrent: () => boolean) => Promise<void>

/** Загрузчик "положить массив в ref" (any: типы Wails-моделей и ../types не обязаны совпадать) */
function list(
  target: Ref<any[]>,
  fetch: (ns: string) => Promise<any[] | null | undefined>,
  after?: () => void,
): Loader {
  return async (ns, isCurrent) => {
    const data = await fetch(ns)
    if (!isCurrent()) return
    target.value = Array.isArray(data) ? data : []
    after?.()
  }
}

/** Опциональные биндинги: если функции нет в Go или она упала, получаем пустой список */
async function listOptional(fn: string, ...args: any[]): Promise<any[]> {
  try {
    const data = await callApp(fn, ...args)
    return Array.isArray(data) ? data : []
  } catch {
    return []
  }
}
const opt = (fn: string, namespaced = false) => (ns: string) =>
  listOptional(fn, ...(namespaced ? [ns] : []))

/**
 * view -> как его грузить. Для view, которых здесь нет (новый каркас, settings),
 * refreshData() ничего не делает.
 */
const loaders: Record<string, Loader> = {
  overview: async (_ns, isCurrent) => {
    const data = await GetClusterOverview()
    if (isCurrent()) overview.value = data
  },
  pods: list(pods, ns => GetPods(ns), () => { loadPodMetrics() }),
  statefulsets: list(statefulsets, ns => GetStatefulSets(ns)),
  jobs: list(jobs, ns => GetJobs(ns)),
  cronjobs: list(cronjobs, ns => GetCronJobs(ns)),
  services: list(services, ns => GetServices(ns)),
  ingresses: list(ingresses, ns => GetIngresses(ns)),
  nodes: list(nodes, () => GetNodes(), () => { loadNodeMetrics() }),
  events: list(events, ns => GetEvents(ns)),
  secrets: list(secrets, ns => GetSecrets(ns)),
  pv: list(pvs, () => GetPersistentVolumes()),
  pvc: list(pvcs, ns => GetPersistentVolumeClaims(ns)),

  // TODO: биндингов в Go пока нет
  resourcequotas: list(resourcequotas, async () => []),
  limitranges: list(limitranges, async () => []),
  hpa: list(hpa, async () => []),
  pdb: list(pdb, async () => []),

  // Опциональные биндинги
  priorityclasses: list(priorityclasses, opt('GetPriorityClasses')),
  runtimeclasses: list(runtimeclasses, opt('GetRuntimeClasses')),
  leases: list(leases, opt('GetLeases', true)),
  endpoints: list(endpoints, opt('GetEndpoints', true)),
  'network-policies': list(networkpolicies, opt('GetNetworkPolicies', true)),
  ingressclasses: list(ingressclasses, opt('GetIngressClasses')),
  storageclasses: list(storageclasses, opt('GetStorageClasses')),

  portforwarding: async () => { await refreshPortForwards() },
}

let refreshSeq = 0
/** view, данные которых уже загружались: для них обновление идёт "тихо", без лоадера */
const loadedViews = new Set<string>()

/**
 * `loading` включается только при ПЕРВОЙ загрузке view. Дальше список остаётся на экране,
 * пока приходит новый, поэтому таблица не размонтируется и не теряет скролл.
 */
async function refreshData() {
  const view = currentView.value
  // port forwards обновляем всегда, но отдельно, чтобы не блокировать остальное
  if (view !== 'portforwarding') refreshPortForwards().catch(() => {})

  const loader = loaders[view]
  if (!loader) { loading.value = false; return }

  const seq = ++refreshSeq
  const isCurrent = () => seq === refreshSeq
  loading.value = !loadedViews.has(view)

  try {
    await loader(selectedNamespace.value, isCurrent)
    if (isCurrent()) loadedViews.add(view)
  } catch (e) {
    if (isCurrent()) fail('Ошибка', e)
  } finally {
    if (isCurrent()) loading.value = false
  }
}

let refreshTimeout: ReturnType<typeof setTimeout> | null = null
function debouncedRefresh() {
  if (refreshTimeout) clearTimeout(refreshTimeout)
  refreshTimeout = setTimeout(() => { refreshData() }, 100)
}

// Один раз на модуль (раньше watch создавался при каждом вызове useK8s()).
// Условие activeContext сохранено как было; фактически refresh вызывают switchView и onNamespaceChange.
watch([currentView, selectedNamespace], () => {
  if (activeContext.value) debouncedRefresh()
})

// ─── Навигация ───

function switchView(view: string) {
  search.value = ''
  selectedPod.value = null
  selectedService.value = null
  yamlContent.value = ''
  execOutput.value = ''
  yamlEditing.value = false
  if (refreshTimeout) { clearTimeout(refreshTimeout); refreshTimeout = null }
  currentView.value = view
  debouncedRefresh()
}

function onNamespaceChange() {
  if (refreshTimeout) { clearTimeout(refreshTimeout); refreshTimeout = null }
  refreshData()
}

// ─── Pod ───

async function openPodDetail(pod: PodInfo) {
  try {
    selectedService.value = null
    yamlContent.value = ''
    podLogs.value = ''
    execOutput.value = ''
    execCmd.value = ''
    yamlEditing.value = false
    selectedPod.value = null

    // Даём старой панели размонтироваться, иначе patch пересоздаёт её с мусором (оставлено как было)
    await nextTick()
    if (pod.containers && pod.containers.length > 0) {
      logContainer.value = pod.containers[0].name
    }
    await nextTick()
    selectedPod.value = { ...pod }
  } catch (e) {
    fail('Ошибка при открытии pod', e)
    selectedPod.value = null
  }
}

async function loadLogs() {
  if (!selectedPod.value) return
  try {
    podLogs.value = '⏳ Загрузка...'
    podLogs.value = await GetPodLogs(
      selectedPod.value.namespace, selectedPod.value.name,
      logContainer.value, logLines.value,
    )
  } catch (e) { podLogs.value = 'Ошибка: ' + errText(e) }
}

async function runExec() {
  if (!selectedPod.value || !execCmd.value) return
  try {
    execOutput.value = '⏳ Выполнение...'
    execOutput.value = await ExecCommand(
      selectedPod.value.namespace, selectedPod.value.name,
      logContainer.value, execCmd.value,
    )
  } catch (e) { execOutput.value = 'Ошибка: ' + errText(e) }
}

/** Бросает исключение; тост и подтверждение делает вызывающий */
async function deletePodByName(ns: string, name: string): Promise<void> {
  await DeletePod(ns, name)
}

// ─── Service ───

async function openServiceDetail(svc: ServiceInfo) {
  try {
    selectedService.value = await GetServiceDetail(svc.namespace, svc.name) as any
    selectedPod.value = null
    yamlContent.value = ''
  } catch (e) { fail('Ошибка', e) }
}

async function forwardServicePort(namespace: string, service: string, port: number) {
  try {
    await callApp('StartServicePortForward', namespace, service, port, port)
    if (selectedService.value) {
      selectedService.value = await GetServiceDetail(namespace, service) as any
    }
    await refreshPortForwards()
  } catch (e) { fail('Ошибка port forward', e) }
}

// ─── YAML ───

/** Нужен usePodActions. Сам просмотр/редактирование YAML идёт через resourceActions.edit() */
async function showYAML(kind: string, namespace: string, name: string) {
  try {
    selectedPod.value = null
    yamlContent.value = '⏳ Загрузка...'
    yamlEditing.value = false
    yamlEditKind.value = kind
    yamlEditNs.value = namespace
    yamlEditName.value = name
    yamlContent.value = await GetResourceYAML(namespace, kind, name, true)
  } catch (e) { yamlContent.value = 'Ошибка: ' + errText(e) }
}

/** Бросает исключение; тост и обновление списка делает вызывающий */
async function applyYAML(content: string, kind: string, namespace: string, name: string) {
  await ApplyYAML(namespace, kind, name, content)
}

// ─── Метрики ───

async function loadPodMetrics() {
  try { podMetrics.value = (await GetPodMetrics(selectedNamespace.value)) || [] }
  catch { podMetrics.value = [] }
}

async function loadNodeMetrics() {
  try { nodeMetrics.value = await GetNodeMetrics() }
  catch { nodeMetrics.value = [] }
}

// ─── Port forward ───

async function refreshPortForwards() {
  try {
    const data = await ListPortForwards()
    portForwards.value = Array.isArray(data) ? data : []
  } catch { portForwards.value = [] }
}

async function startPortForward(namespace?: string, podName?: string, localPort?: number, podPort?: number) {
  const ns = namespace || selectedPod.value?.namespace
  const pod = podName || selectedPod.value?.name
  const lp = localPort !== undefined ? localPort : pfLocalPort.value
  const pp = podPort !== undefined ? podPort : pfPodPort.value
  if (!ns || !pod) return

  try {
    await StartPortForward(ns, pod, lp, pp)
    await new Promise(resolve => setTimeout(resolve, 500)) // даём бэкенду зарегистрировать forward
    await refreshPortForwards()
  } catch (e) {
    fail('Ошибка port forward', e)
    throw e // вызывающие панели могут сбросить своё состояние
  }
}

async function stopPortForward(id: string) {
  try {
    await StopPortForward(id)
    await refreshPortForwards()
  } catch (e) { fail('Ошибка остановки port forward', e) }
}

// ─── Операции (бросают исключение; тост делает вызывающий) ───

const call = (fn: string, ...args: any[]) => callApp<void>(fn, ...args)

const deleteJob = (ns: string, name: string) => call('DeleteJob', ns, name)
const retryJob = (ns: string, name: string) => call('RetryJob', ns, name)

// Нужны только старому useResourceActions.ts (удалите его вместе с этими строками).
// Deployment и DaemonSet теперь описаны в resources/definitions.
const scaleDeployment = (ns: string, name: string, replicas: number) => ScaleDeployment(ns, name, replicas)
const restartDeployment = (ns: string, name: string) => RestartDeployment(ns, name)
const deleteDeploymentByName = (ns: string, name: string) => call('DeleteDeployment', ns, name)
const deleteDaemonSet = (ns: string, name: string) => call('DeleteDaemonSet', ns, name)
const deleteServiceAccount = (ns: string, name: string) => call('DeleteServiceAccount', ns, name)
const deleteRole = (ns: string, name: string) => call('DeleteRole', ns, name)
const deleteRoleBinding = (ns: string, name: string) => call('DeleteRoleBinding', ns, name)
const deleteClusterRole = (name: string) => call('DeleteClusterRole', name)
const deleteClusterRoleBinding = (name: string) => call('DeleteClusterRoleBinding', name)

// ─── Node (пока не на каркасе; бросает исключение) ───

function formatResourceDetail(resource: any): any {
  if (!resource) return {}
  return {
    name: resource.name || resource.metadata?.name || '',
    namespace: resource.namespace || resource.metadata?.namespace || '',
    age: resource.age || '',
    created: resource.created || resource.createdTime || resource.metadata?.creationTimestamp || '',
    ...resource,
  }
}

async function getNode(name: string) {
  try {
    return formatResourceDetail(await callApp('GetNodeDetail', name))
  } catch {
    const found = (await GetNodes()).find((n: any) => n.name === name)
    if (!found) throw new Error('Node not found')
    return formatResourceDetail(found)
  }
}

// ─── Export ───

export function useK8s() {
  return {
    // State
    contexts, namespaces, pods, statefulsets,
    jobs, cronjobs, services, ingresses,
    nodes, events, secrets, pvs, pvcs, overview,
    podMetrics, nodeMetrics, portForwards,
    activeContext, selectedNamespace, currentView, search,
    error, connecting, loading,
    selectedPod, podLogs, logContainer, logLines,
    execCmd, execOutput,
    selectedService,
    yamlContent, yamlEditing, yamlEditKind, yamlEditNs, yamlEditName,
    pfLocalPort, pfPodPort,

    // Filtered
    filteredPods, filteredStatefulSets, filteredJobs, filteredCronJobs,
    filteredServices, filteredIngresses, filteredSecrets, filteredResourceQuotas, filteredPVCs,
    filteredLimitRanges, filteredHPA, filteredPDB,
    filteredPriorityClasses, filteredRuntimeClasses, filteredLeases,
    filteredEndpoints, filteredNetworkPolicies, filteredIngressClasses, filteredStorageClasses,

    // Helpers
    statusClass, pct, formatCpu, formatMemory, mergePodMetrics,

    // Connection / navigation / data
    loadContexts, connect, disconnect, loadNamespaces,
    switchView, onNamespaceChange, refreshData,

    // Pod
    openPodDetail, loadLogs, runExec, deletePodByName,
    getPods: GetPods,

    // Job
    deleteJob, retryJob,

    // Service / YAML
    openServiceDetail, forwardServicePort,
    showYAML, applyYAML,

    // Metrics / port forward
    loadPodMetrics, loadNodeMetrics,
    startPortForward, stopPortForward, refreshPortForwards,

    // Node
    getNode,

    // Совместимость со старым useResourceActions.ts
    scaleDeployment, restartDeployment, deleteDeploymentByName, deleteDaemonSet,
    deleteServiceAccount, deleteRole, deleteRoleBinding, deleteClusterRole, deleteClusterRoleBinding,
  }
}
