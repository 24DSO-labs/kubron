import { ref, watch, nextTick, type Ref } from 'vue'
import {
  GetContexts, ConnectToContext, GetActiveContext,
  GetNamespaces, GetPods, GetPodLogs, DeletePod,
  ScaleDeployment, RestartDeployment,
  GetClusterOverview, GetResourceYAML, ApplyYAML,
  ExecCommand,
  GetPodMetrics,
  StartPortForward, StopPortForward, ListPortForwards,
} from '../../wailsjs/go/main/App'
import { callApp } from '../resources/wails'
import { useNotifications } from './useNotifications'

import type {
  ContextInfo, NamespaceInfo, PodInfo,
  OverviewInfo,
  PodMetricsInfo, PortForwardSession,
} from '../types'

/*
 * useK8s: состояние и загрузка данных для СТАРЫХ view.
 *
 * Ресурсы, перенесённые на новый каркас (src/resources/registry.ts), здесь больше не хранятся:
 * их данные грузит resources/store.ts. refreshData() для таких view ничего не делает.
 * Сейчас на каркасе: все ресурсы из resources/registry.ts (включая nodes, helm, port forwarding).
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
const overview = ref<OverviewInfo | null>(null)

const podMetrics = ref<PodMetricsInfo[]>([])
const portForwards = ref<PortForwardSession[]>([])

/** Всё, что надо очистить при отключении от кластера */
const clearableLists: Ref<any[]>[] = [
  namespaces,
  podMetrics, portForwards,
]

// ─── State: навигация и выбранное ───

const activeContext = ref('')
const selectedNamespace = ref('') // '' = все namespaces
const currentView = ref('overview')
const search = ref('')
const error = ref('')
const connecting = ref(false)
const loading = ref(false)

const yamlContent = ref('')
const yamlEditing = ref(false)
const yamlEditKind = ref('')
const yamlEditNs = ref('')
const yamlEditName = ref('')

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
// (списки нового каркаса фильтруются в resources/store.ts)

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

/**
 * view -> как его грузить. Для view, которых здесь нет (новый каркас, settings),
 * refreshData() ничего не делает.
 */
const loaders: Record<string, Loader> = {
  overview: async (_ns, isCurrent) => {
    const data = await GetClusterOverview()
    if (isCurrent()) overview.value = data
  },
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
  yamlContent.value = ''
  yamlEditing.value = false
  if (refreshTimeout) { clearTimeout(refreshTimeout); refreshTimeout = null }
  currentView.value = view
  debouncedRefresh()
}

function onNamespaceChange() {
  if (refreshTimeout) { clearTimeout(refreshTimeout); refreshTimeout = null }
  refreshData()
}

// ─── YAML ───

/** Нужен usePodActions. Сам просмотр/редактирование YAML идёт через resourceActions.edit() */
async function showYAML(kind: string, namespace: string, name: string) {
  try {
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

// ─── Port forward ───

async function refreshPortForwards() {
  try {
    const data = await ListPortForwards()
    portForwards.value = Array.isArray(data) ? data : []
  } catch { portForwards.value = [] }
}

async function startPortForward(namespace: string, podName: string, localPort: number, podPort: number) {
  if (!namespace || !podName) return
  try {
    await StartPortForward(namespace, podName, localPort, podPort)
    await new Promise(resolve => setTimeout(resolve, 500))
    await refreshPortForwards()
  } catch (e) {
    fail('Ошибка port forward', e)
    throw e
  }
}

async function stopPortForward(id: string) {
  try {
    await StopPortForward(id)
    await refreshPortForwards()
  } catch (e) { fail('Ошибка остановки port forward', e) }
}

// ─── Export ───

export function useK8s() {
  return {
    // State
    contexts, namespaces,
    overview,
    podMetrics, portForwards,
    activeContext, selectedNamespace, currentView, search,
    error, connecting, loading,
    yamlContent, yamlEditing, yamlEditKind, yamlEditNs, yamlEditName,

    // Helpers
    statusClass, pct, formatCpu, formatMemory, mergePodMetrics,

    // Connection / navigation / data
    loadContexts, connect, disconnect, loadNamespaces,
    switchView, onNamespaceChange, refreshData,

    // YAML
    showYAML, applyYAML,

    // Metrics / port forward
    loadPodMetrics,
    startPortForward, stopPortForward, refreshPortForwards,
  }
}
