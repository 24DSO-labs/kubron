import { ref, type Ref } from 'vue'
import {
  GetNamespaces,
  GetClusterOverview, ApplyYAML,
  GetPodMetrics,
  ListPortForwards,
} from '../../wailsjs/go/main/App'
import { useNotifications } from './useNotifications'

import type {
  NamespaceInfo, PodInfo,
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
 *  Ref `error` остаётся для useClusterConnection, который пишет в него сам.
 */

// ─── Уведомления ───

const notifications = useNotifications()
const errText = (e: any): string => e?.message || String(e)
const fail = (prefix: string, e: any) => notifications.notifyError(`${prefix}: ${errText(e)}`)

// ─── State: списки ───

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

const selectedNamespace = ref('') // '' = все namespaces
const currentView = ref('overview')
const search = ref('')
const error = ref('')
const loading = ref(false)

// ─── Helpers ───

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

function disconnect() {
  for (const l of clearableLists) l.value = []
  overview.value = null
  selectedNamespace.value = ''
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

// ─── Навигация ───

function switchView(view: string) {
  search.value = ''
  if (refreshTimeout) { clearTimeout(refreshTimeout); refreshTimeout = null }
  currentView.value = view
  debouncedRefresh()
}

function onNamespaceChange() {
  if (refreshTimeout) { clearTimeout(refreshTimeout); refreshTimeout = null }
  refreshData()
}

// ─── YAML ───

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

// ─── Export ───

export function useK8s() {
  return {
    // State
    namespaces,
    overview,
    podMetrics, portForwards,
    selectedNamespace, currentView, search,
    error, loading,

    // Helpers
    pct, formatCpu, formatMemory, mergePodMetrics,

    // Connection / navigation / data
    disconnect, loadNamespaces,
    switchView, onNamespaceChange, refreshData,

    // YAML
    applyYAML,

    // Metrics / port forward
    loadPodMetrics,
    refreshPortForwards,
  }
}
