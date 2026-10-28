// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { ref, shallowRef, onMounted, onUnmounted, watch, type Ref } from 'vue'
import { callApp } from '../resources/wails'
import type { ArgoApplicationInfo } from '../types/argocd'
import { errorText } from './errorText'
import { argoActionTick } from '../resources/argoActions'
import { argoLiveTick } from './useArgoLive'

// Основное обновление приходит событием argocd:changed; таймер — подстраховка (informer мог не стартовать)
const REFRESH_MS = 60_000
// Сколько подсвечивать приложение, у которого только что поменялось состояние
const FLASH_MS = 3_000

const appKey = (a: ArgoApplicationInfo) => `${a.namespace}/${a.name}`
const appState = (a: ArgoApplicationInfo) =>
  [a.syncStatus, a.healthStatus, a.revision, a.lastSyncPhase, a.operationOn].join('|')

/**
 * Application из всех namespace для дашборда Argo CD.
 * Тот же контракт, что у useOverview: устаревший ответ отбрасывается, при ошибке остаются последние данные,
 * пока вкладка скрыта, запросы не идут.
 */
export function useArgoApplications(clusterName: Ref<string>) {
  const apps = shallowRef<ArgoApplicationInfo[] | null>(null)
  const loading = ref(true)
  const refreshing = ref(false)
  const error = ref('')
  const updatedAt = ref<Date | null>(null)
  /** Ключи namespace/name приложений, у которых состояние изменилось при последнем обновлении */
  const changedKeys = ref<ReadonlySet<string>>(new Set())

  let seq = 0
  let flashTimer: ReturnType<typeof setTimeout> | null = null

  function markChanged(prev: ArgoApplicationInfo[] | null, next: ArgoApplicationInfo[]) {
    if (!prev) return // первая загрузка: подсвечивать нечего
    const before = new Map(prev.map(a => [appKey(a), appState(a)]))
    const changed = new Set<string>()
    for (const a of next) {
      const old = before.get(appKey(a))
      // новое приложение тоже считается изменением
      if (old === undefined || old !== appState(a)) changed.add(appKey(a))
    }
    if (flashTimer) clearTimeout(flashTimer)
    changedKeys.value = changed
    if (changed.size) {
      flashTimer = setTimeout(() => { changedKeys.value = new Set() }, FLASH_MS)
    }
  }

  let timer: ReturnType<typeof setInterval> | null = null

  async function refresh() {
    const mySeq = ++seq
    refreshing.value = true
    try {
      const res = (await callApp('GetArgoApplications', [])) as ArgoApplicationInfo[]
      if (mySeq !== seq) return
      const next = res ?? []
      markChanged(apps.value, next)
      apps.value = next
      updatedAt.value = new Date()
      error.value = ''
    } catch (e: any) {
      if (mySeq !== seq) return
      error.value = errorText(e)
    } finally {
      if (mySeq === seq) {
        loading.value = false
        refreshing.value = false
      }
    }
  }

  const onVisibility = () => { if (!document.hidden) refresh() }

  onMounted(() => {
    refresh()
    timer = setInterval(() => { if (!document.hidden) refresh() }, REFRESH_MS)
    document.addEventListener('visibilitychange', onVisibility)
  })
  onUnmounted(() => {
    seq++
    if (timer) clearInterval(timer)
    if (flashTimer) clearTimeout(flashTimer)
    document.removeEventListener('visibilitychange', onVisibility)
  })

  watch(clusterName, () => {
    apps.value = null
    changedKeys.value = new Set()
    error.value = ''
    loading.value = true
    refresh()
  })
  watch(argoActionTick, () => refresh())
  watch(argoLiveTick, () => refresh())

  return { apps, loading, refreshing, error, updatedAt, changedKeys, refresh }
}
