import { ref, shallowRef, onMounted, onUnmounted, watch, type Ref } from 'vue'
import { callApp } from '../resources/wails'
import type { ClusterOverview } from '../types/overview'

const REFRESH_MS = 15_000

/**
 * Данные главного экрана с автообновлением.
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

  function onVisibility() {
    if (!document.hidden) refresh()
  }

  onMounted(() => {
    refresh()
    timer = setInterval(() => {
      if (!document.hidden) refresh()
    }, REFRESH_MS)
    document.addEventListener('visibilitychange', onVisibility)
  })

  onUnmounted(() => {
    seq++ // поздний ответ не должен писать в уже размонтированный экран
    if (timer) clearInterval(timer)
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
