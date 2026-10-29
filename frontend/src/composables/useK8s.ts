// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { ref, type Ref } from 'vue'
import { useNotifications } from './useNotifications'
import { formatCpu, formatMemory, mergePodMetrics } from '../resources/podUsage'

import type {
  NamespaceInfo,
  PodMetricsInfo, PortForwardSession,
} from '../types'
import { callApp } from '../platform/api'
import { resetListFailures } from './useListFailures'
import { normalizeSelection, restoreSelection, sameSelection, saveSelection } from './namespaceSelection'
import { errorText } from './errorText'
import { t } from '../i18n'

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
 *  Ошибки подключения к кластеру useClusterConnection показывает тостом сам.
 */

// ─── Уведомления ───

const notifications = useNotifications()
const fail = (prefix: string, e: unknown) => notifications.notifyError(`${prefix}: ${errorText(e)}`)

// ─── State: списки ───

const namespaces = ref<NamespaceInfo[]>([]) // для выбора namespace в Sidebar

const podMetrics = ref<PodMetricsInfo[]>([])
const portForwards = ref<PortForwardSession[]>([])

/** Всё, что надо очистить при отключении от кластера */
const clearableLists: Ref<any[]>[] = [
  namespaces,
  podMetrics, portForwards,
]

// ─── State: навигация и выбранное ───

// Выбранные namespace: [] = все, [a] = один, [a, b, ...] = набор.
// Массив всегда заменяется целиком (не мутируется): так срабатывают watch и shallowRef.
const selectedNamespaces = ref<string[]>([])
/** Кластер, под которым сохраняется выбор namespace (пусто — до подключения) */
let namespaceScopeId = ''
const currentView = ref('overview')
const search = ref('')
const loading = ref(false)

// ─── Фильтрация по поиску ───
// (списки нового каркаса фильтруются в resources/store.ts)

function disconnect() {
  for (const l of clearableLists) l.value = []
  selectedNamespaces.value = []
  namespaceScopeId = ''
  loadedViews.clear()
}

async function loadNamespaces() {
  try { namespaces.value = await callApp('GetNamespaces') }
  catch (e) { fail(t('app.errors.loadNamespaces'), e) }
}

// ─── Загрузка данных текущего view ───

/**
 * Загрузчик принимает namespace и isCurrent(). Результат применяется, только если
 * за время запроса не стартовал более свежий refresh: так медленный ответ
 * для старого namespace не затирает новый.
 */
type Loader = (ns: string[], isCurrent: () => boolean) => Promise<void>

/**
 * view -> как его грузить. Для view, которых здесь нет (новый каркас, settings),
 * refreshData() ничего не делает.
 */
const loaders: Record<string, Loader> = {}

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
    await loader(selectedNamespaces.value, isCurrent)
    if (isCurrent()) loadedViews.add(view)
  } catch (e) {
    if (isCurrent()) fail(t('app.errors.generic'), e)
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

/**
 * Выбор пользователя: нормализуется по списку namespace кластера, запоминается для кластера
 * и применяется перезагрузкой данных.
 */
function selectNamespaces(next: string[]) {
  const available = namespaces.value.map(n => n.name)
  const normalized = normalizeSelection(next, available.length ? available : undefined)
  if (sameSelection(normalized, selectedNamespaces.value)) return
  selectedNamespaces.value = normalized
  resetListFailures()
  saveSelection(namespaceScopeId, normalized)
  onNamespaceChange()
}

/**
 * Восстановление выбора после loadNamespaces() при подключении: сохранённый для кластера,
 * иначе namespace по умолчанию из настроек кластера, иначе «все».
 */
function restoreNamespaces(clusterId: string, defaultNamespace: string) {
  namespaceScopeId = clusterId
  resetListFailures()
  selectedNamespaces.value = restoreSelection(clusterId, namespaces.value.map(n => n.name), defaultNamespace)
}

function onNamespaceChange() {
  if (refreshTimeout) { clearTimeout(refreshTimeout); refreshTimeout = null }
  refreshData()
}

// ─── YAML ───

/** Бросает исключение; тост и обновление списка делает вызывающий */
async function applyYAML(content: string, kind: string, namespace: string, name: string) {
  await callApp('ApplyYAML', namespace, kind, name, content)
}

/** Создаёт новый объект из манифеста и возвращает его имя */
function createResourceYAML(content: string, kind: string, namespace: string): Promise<string> {
  return callApp('CreateResourceYAML', namespace, kind, content)
}

// ─── Метрики ───

async function loadPodMetrics() {
  try { podMetrics.value = (await callApp('GetPodMetrics', selectedNamespaces.value)) || [] }
  catch { /* оставляем прошлые значения, чтобы таблица не мигала */ }
}

// ─── Port forward ───

async function refreshPortForwards() {
  let next: PortForwardSession[]
  try {
    const data = await callApp('ListPortForwards')
    next = Array.isArray(data) ? data : []
  } catch { next = [] }
  // Не трогаем ref, если ничего не изменилось: иначе refreshOn списка port forward
  // перезагружает список, тот снова вызывает refresh, и так по кругу
  if (JSON.stringify(next) !== JSON.stringify(portForwards.value)) portForwards.value = next
}

// ─── Export ───

export function useK8s() {
  return {
    // State
    namespaces,
    podMetrics, portForwards,
    selectedNamespaces, selectNamespaces, restoreNamespaces, currentView, search,
    loading,

    // Helpers
    formatCpu, formatMemory, mergePodMetrics,

    // Connection / navigation / data
    disconnect, loadNamespaces,
    switchView, onNamespaceChange, refreshData,

    // YAML
    applyYAML,
    createResourceYAML,

    // Metrics / port forward
    loadPodMetrics,
    refreshPortForwards,
  }
}
