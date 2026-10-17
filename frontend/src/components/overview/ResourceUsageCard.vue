<template>
  <section class="ov-card usage">
    <h2 class="ov-card-title">Resource usage</h2>

    <div v-if="state === 'ready'" class="metrics">
      <div v-for="m in metrics" :key="m.label" class="metric">
        <span class="metric-label">{{ m.label }}</span>
        <div class="metric-row">
          <div class="metric-text">
            <span class="metric-percent">{{ m.percent }}</span>
            <span class="metric-detail">{{ m.detail }}</span>
          </div>
          <div class="metric-spark">
            <Sparkline :values="m.series" :color="m.color" :max="100" />
          </div>
        </div>
      </div>
    </div>

    <div v-else class="empty">
      <template v-if="state === 'loading'">
        <p class="empty-text">Loading…</p>
      </template>
      <template v-else-if="state === 'installing'">
        <p class="empty-title">Metrics stack is installing</p>
        <p class="empty-text">Данные появятся, когда компоненты запустятся.</p>
      </template>
      <template v-else-if="state === 'degraded'">
        <p class="empty-title">Metrics stack is not working</p>
        <p class="empty-text">Проверьте компоненты в настройках кластера.</p>
        <button type="button" class="ov-btn" @click="emit('openSettings')">Open settings</button>
      </template>
      <template v-else>
        <p class="empty-title">Metrics are not available</p>
        <p class="empty-text">Установите стек метрик в настройках кластера, чтобы видеть загрузку CPU и памяти.</p>
        <button type="button" class="ov-btn ov-btn-primary" @click="emit('openSettings')">Install metrics</button>
      </template>
    </div>
  </section>
</template>

<script lang="ts" setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import Sparkline from './Sparkline.vue'
import { formatBytes, formatCores } from './format'
import { useClusterMetrics, useMetricsStatus, usePrometheusRange } from '../../composables/useMetrics'

const props = defineProps<{ clusterName: string }>()
const emit = defineEmits<{ (e: 'openSettings'): void }>()

const SNAPSHOT_MS = 15_000
const SERIES_MS = 60_000
const RANGE_SEC = 3600
const STEP_SEC = 120

const CPU_QUERY = '100 * sum(rate(node_cpu_seconds_total{mode!="idle"}[5m])) / count(node_cpu_seconds_total{mode="idle"})'
const MEM_QUERY = '100 * (1 - sum(node_memory_MemAvailable_bytes) / sum(node_memory_MemTotal_bytes))'

const { status, fetchStatus } = useMetricsStatus(props.clusterName)
const { snapshot, fetch: fetchSnapshot } = useClusterMetrics(props.clusterName)
const { queryRange } = usePrometheusRange(props.clusterName)

const checked = ref(false)
const cpuSeries = ref<number[]>([])
const memSeries = ref<number[]>([])

type State = 'loading' | 'ready' | 'not_installed' | 'installing' | 'degraded'
const state = computed<State>(() => {
  if (!checked.value) return 'loading'
  return status.value?.status === 'installed' ? 'ready' : (status.value?.status ?? 'not_installed')
})

const percent = (v?: number) => (v === undefined || !isFinite(v) ? '—' : `${Math.round(v)}%`)

const metrics = computed(() => {
  const s = snapshot.value
  return [
    {
      label: 'CPU',
      percent: percent(s?.cpuPercent),
      detail: s ? `${formatCores(s.cpuUsage)} / ${formatCores(s.cpuCapacity)} cores` : '',
      series: cpuSeries.value,
      color: '#8b5cf6',
    },
    {
      label: 'Memory',
      percent: percent(s?.memoryPercent),
      detail: s ? `${formatBytes(s.memoryUsage)} / ${formatBytes(s.memoryCapacity)}` : '',
      series: memSeries.value,
      color: '#3b82f6',
    },
  ]
})

async function loadSeries() {
  const end = Math.floor(Date.now() / 1000)
  const start = end - RANGE_SEC
  const [cpu, mem] = await Promise.all([
    queryRange(CPU_QUERY, start, end, STEP_SEC),
    queryRange(MEM_QUERY, start, end, STEP_SEC),
  ])
  cpuSeries.value = cpu.map(p => p[1])
  memSeries.value = mem.map(p => p[1])
}

let timers: ReturnType<typeof setInterval>[] = []
let generation = 0 // отбрасывает результаты, пришедшие после смены кластера или размонтирования

function stop() {
  generation++
  timers.forEach(clearInterval)
  timers = []
}

async function start() {
  stop()
  const gen = generation
  checked.value = false
  await fetchStatus()
  if (gen !== generation) return
  checked.value = true
  if (state.value !== 'ready') return

  await Promise.all([fetchSnapshot(), loadSeries()])
  if (gen !== generation) return
  timers.push(
    setInterval(() => { if (!document.hidden) fetchSnapshot() }, SNAPSHOT_MS),
    setInterval(() => { if (!document.hidden) loadSeries() }, SERIES_MS),
  )
}

onMounted(start)
onUnmounted(stop)
watch(() => props.clusterName, start)
</script>

<style scoped>
.metrics { display: flex; flex-direction: column; gap: 18px; flex: 1; justify-content: center; }
.metric-label { font-size: 13px; color: var(--ov-muted); }
.metric-row { display: flex; align-items: flex-end; justify-content: space-between; gap: 16px; margin-top: 2px; }
.metric-text { display: flex; flex-direction: column; min-width: 0; }
.metric-percent { font-size: 28px; font-weight: 600; line-height: 1.15; font-variant-numeric: tabular-nums; }
.metric-detail { font-size: 12px; color: var(--ov-muted); white-space: nowrap; }
.metric-spark { width: 55%; height: 44px; flex-shrink: 0; }

.empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  justify-content: center;
  gap: 6px;
}
.empty-title { font-size: 14px; font-weight: 600; }
.empty-text { font-size: 12px; color: var(--ov-muted); line-height: 1.5; }
.empty .ov-btn { margin-top: 8px; }
</style>
