import { onMounted, onUnmounted, ref, shallowRef, watch, type Ref } from 'vue'
import { callApp } from '../resources/wails'
import type { MetricsState, OverviewHotspots } from '../types/hotspots'

const REFRESH_MS = 30_000

/**
 * Топ подов и тома для главного экрана. Нужен стек метрик: без него state != 'ready',
 * карточки показывают подсказку. Устаревший ответ (смена кластера) отбрасывается.
 */
export function useHotspots(clusterName: Ref<string>) {
  const data = shallowRef<OverviewHotspots | null>(null)
  const state = ref<MetricsState>('loading')
  let seq = 0
  let timer: ReturnType<typeof setInterval> | null = null

  async function load() {
    const my = ++seq
    try {
      if (state.value !== 'ready') {
        const st = (await callApp('GetMetricsStatus')) as unknown as { status: MetricsState }
        if (my !== seq) return
        state.value = st?.status ?? 'not_installed'
        if (state.value !== 'ready') return
      }
      const res = (await callApp('GetOverviewHotspots')) as unknown as OverviewHotspots
      if (my !== seq) return
      data.value = res
    } catch {
      if (my === seq && !data.value) state.value = 'not_installed'
    }
  }

  const tick = () => { if (!document.hidden) void load() }
  onMounted(() => { void load(); timer = setInterval(tick, REFRESH_MS) })
  onUnmounted(() => { seq++; if (timer) clearInterval(timer) })
  watch(clusterName, () => { data.value = null; state.value = 'loading'; void load() })

  return { data, state, reload: load }
}
