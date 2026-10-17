import { shallowRef, ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useK8s } from '../composables/useK8s'
import { useNotifications } from '../composables/useNotifications'
import { useAppSettings } from '../composables/useAppSettings'
import type { ResourceDefinition } from './types'
import { matchesSearch } from './utils'

// useK8s держит состояние на уровне модуля: namespace и search общие для всего приложения
const k8s = useK8s()
const notifications = useNotifications()

/** Данные по view. shallowRef: списки большие, глубокая реактивность не нужна */
const itemsByView = shallowRef<Record<string, readonly any[]>>({})
const seqByView: Record<string, number> = {}
let activeDef: ResourceDefinition | null = null

/**
 * Загрузка списка. Старые данные остаются на экране, пока идёт запрос,
 * поэтому таблица не мигает и не теряет скролл при обновлении.
 */
export async function loadResource(def: ResourceDefinition): Promise<void> {
  const seq = (seqByView[def.view] = (seqByView[def.view] ?? 0) + 1)
  const ns = def.namespaced ? k8s.selectedNamespace.value : ''
  const publish = (list: readonly any[]) => {
    itemsByView.value = { ...itemsByView.value, [def.view]: list }
  }
  try {
    const data = await def.api.list(ns)
    if (seqByView[def.view] !== seq) return
    const list = Array.isArray(data) ? data : []
    let result: readonly any[] = list

    if (def.onLoaded) {
      // первый раз показываем сразу, дальше ждём метрики и публикуем один раз
      if (!(def.view in itemsByView.value)) publish(list)
      try {
        const enriched = await def.onLoaded(list)
        if (Array.isArray(enriched)) result = enriched
      } catch { /* игнорируем ошибки onLoaded */ }
      if (seqByView[def.view] !== seq) return
    }
    publish(result)
  } catch (e: any) {
    if (seqByView[def.view] !== seq) return
    if (!(def.view in itemsByView.value)) {
      itemsByView.value = { ...itemsByView.value, [def.view]: [] }
    }
    notifications.notifyError(`${def.title}: ${e?.message || e}`)
  }
}
/** Перезагрузить список, который сейчас на экране (после apply YAML, delete и т.п.) */
export async function reloadActive(): Promise<void> {
  if (activeDef) await loadResource(activeDef)
}

/** Найти уже загруженный элемент (для открытия деталей по ссылке) */
export function findLoaded(view: string, ns: string, name: string): any | undefined {
  return itemsByView.value[view]?.find(
    (x: any) => (x.name ?? x.metadata?.name) === name && (x.namespace ?? x.metadata?.namespace ?? '') === ns,
  )
}

/** Сбросить кэш (смена кластера) */
export function clearResourceStore(): void {
  itemsByView.value = {}
}

/** Состояние списка для ResourceList */
export function useResourceList(getDef: () => ResourceDefinition) {
  const items = computed(() => {
    const list = itemsByView.value[getDef().view] ?? []
    return list.filter(x => matchesSearch(x, k8s.search.value))
  })

  // лоадер только пока нет первых данных
  const initialLoading = computed(() => !(getDef().view in itemsByView.value))

  const reload = () => loadResource(getDef())

  // Авто-обновление
  const appSettings = useAppSettings()
  let refreshTimer: ReturnType<typeof setInterval> | null = null

  function startAutoRefresh() {
    stopAutoRefresh()
    const def = getDef()
    if (!def.autoRefresh) return
    const intervalMs = appSettings.getRefreshIntervalMs()
    if (intervalMs <= 0) return
    refreshTimer = setInterval(() => {
      loadResource(def)
    }, intervalMs)
  }

  function stopAutoRefresh() {
    if (refreshTimer) {
      clearInterval(refreshTimer)
      refreshTimer = null
    }
  }

  onMounted(() => {
    activeDef = getDef()
    reload()
    startAutoRefresh()
  })
  onUnmounted(() => {
    if (activeDef === getDef()) activeDef = null
    stopAutoRefresh()
  })

  watch(
    () => k8s.selectedNamespace.value,
    () => {
      if (getDef().namespaced) reload()
    },
  )

  // Внешние изменения данных (def.refreshOn); watch живёт, пока смонтирован список этого ресурса
  watch(() => getDef().refreshOn?.(), () => reload())

  const search = computed({
    get: () => k8s.search.value,
    set: (v: string) => {
      k8s.search.value = v
    },
  })

  return { items, initialLoading, reload, search }
}
