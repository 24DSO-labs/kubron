import { onMounted, onUnmounted, ref, shallowRef, watch, type Ref } from 'vue'
import { callApp } from '../resources/wails'
import type { MetricsState, OverviewHotspots } from '../types/hotspots'

const REFRESH_MS = 30_000

/**
 * Топ подов и тома для главного экрана.
 * Бэкенд отдаёт счётчики PVC Pending/Lost из API всегда, а данные Prometheus — только при withMetrics,
 * поэтому запрос идёт и без стека метрик (state != 'ready'), просто без топов.
 * Устаревший ответ (смена кластера) отбрасывается.
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
        const st = (await callApp('GetMetricsStatus')) as unknown as { status: string }
        if (my !== seq) return
        // Go отдаёт 'installed' (not_installed | installing | installed | degraded)
        state.value = st?.status === 'installed' ? 'ready' : ((st?.status as MetricsState) ?? 'not_installed')
      }
      const res = (await callApp('GetOverviewHotspots', state.value === 'ready')) as unknown as OverviewHotspots
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
