// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { ref, shallowRef, onMounted, onUnmounted, watch, type Ref } from 'vue'
import { callApp } from '../resources/wails'
import { onEvent } from '../platform/events'
import type { ClusterOverview } from '../types/overview'

// Основное обновление приходит событием от бэкенда (informers); таймер — подстраховка
const REFRESH_MS = 60_000
// Не чаще одного обновления по событиям за это время
const EVENT_THROTTLE_MS = 5_000
const CLUSTER_CHANGED = 'cluster:changed'

/**
 * Данные главного экрана с автообновлением.
 *  - обновляется по событию cluster:changed от бэкенда, редкий таймер — подстраховка;
 *  - пока вкладка скрыта, запросы не идут;
 *  - ответ устаревшего запроса (смена кластера, быстрые клики по «Refresh») отбрасывается;
 *  - при ошибке на экране остаются последние удачные данные.
 */
export function useOverview(clusterName: Ref<string>) {
  const data = shallowRef<ClusterOverview | null>(null)
  const loading = ref(true) // только пока нет первых данных
  const refreshing = ref(false)
  const error = ref('')
  const updatedAt = ref<Date | null>(null)

  let seq = 0
  let timer: ReturnType<typeof setInterval> | null = null
  let offEvents: (() => void) | null = null
  let lastEventRefresh = 0
  let pendingEvent: ReturnType<typeof setTimeout> | null = null

  async function refresh() {
    const mySeq = ++seq
    refreshing.value = true
    try {
      const res = (await callApp('GetClusterOverview')) as unknown as ClusterOverview
      if (mySeq !== seq) return
      data.value = res
      updatedAt.value = new Date()
      error.value = ''
    } catch (e: any) {
      if (mySeq !== seq) return
      error.value = e?.message || String(e)
    } finally {
      if (mySeq === seq) {
        loading.value = false
        refreshing.value = false
      }
    }
  }

  function onClusterChanged() {
    if (document.hidden) return
    const wait = lastEventRefresh + EVENT_THROTTLE_MS - Date.now()
    if (wait <= 0) {
      lastEventRefresh = Date.now()
      refresh()
    } else if (!pendingEvent) {
      // Укладываем серию событий в одно обновление в конце окна
      pendingEvent = setTimeout(() => {
        pendingEvent = null
        lastEventRefresh = Date.now()
        if (!document.hidden) refresh()
      }, wait)
    }
  }

  function onVisibility() {
    if (!document.hidden) refresh()
  }

  onMounted(() => {
    refresh()
    timer = setInterval(() => {
      if (!document.hidden) refresh()
    }, REFRESH_MS)
    document.addEventListener('visibilitychange', onVisibility)
    offEvents = onEvent(CLUSTER_CHANGED, onClusterChanged)
  })

  onUnmounted(() => {
    seq++ // поздний ответ не должен писать в уже размонтированный экран
    if (timer) clearInterval(timer)
    if (pendingEvent) clearTimeout(pendingEvent)
    offEvents?.()
    document.removeEventListener('visibilitychange', onVisibility)
  })

  // Смена кластера: старые данные показывать нельзя
  watch(clusterName, () => {
    data.value = null
    error.value = ''
    loading.value = true
    refresh()
  })

  return { data, loading, refreshing, error, updatedAt, refresh }
}
