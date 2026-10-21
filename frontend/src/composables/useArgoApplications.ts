import { ref, shallowRef, onMounted, onUnmounted, watch, type Ref } from 'vue'
import { callApp } from '../resources/wails'
import type { ArgoApplicationInfo } from '../types/argocd'

const REFRESH_MS = 30_000

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

  let seq = 0
  let timer: ReturnType<typeof setInterval> | null = null

  async function refresh() {
    const mySeq = ++seq
    refreshing.value = true
    try {
      const res = (await callApp('GetArgoApplications', '')) as ArgoApplicationInfo[]
      if (mySeq !== seq) return
      apps.value = res ?? []
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

  const onVisibility = () => { if (!document.hidden) refresh() }

  onMounted(() => {
    refresh()
    timer = setInterval(() => { if (!document.hidden) refresh() }, REFRESH_MS)
    document.addEventListener('visibilitychange', onVisibility)
  })
  onUnmounted(() => {
    seq++
    if (timer) clearInterval(timer)
    document.removeEventListener('visibilitychange', onVisibility)
  })

  watch(clusterName, () => {
    apps.value = null
    error.value = ''
    loading.value = true
    refresh()
  })

  return { apps, loading, refreshing, error, updatedAt, refresh }
}
