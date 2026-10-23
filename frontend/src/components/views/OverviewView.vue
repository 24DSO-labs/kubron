<template>
  <div class="ov">
    <header class="ov-header">
      <h1>Overview</h1>
      <div class="ov-header-right">
        <span v-if="data && error" class="ov-stale" :title="error">Update failed, showing last data</span>
        <ClusterHealthChip v-if="data" :health="data.health" />
        <span v-else-if="updatedAt" class="ov-updated">Updated {{ updatedAt.toLocaleTimeString() }}</span>
        <button type="button" class="ov-btn" :disabled="refreshing" @click="refresh">
          {{ refreshing ? 'Refreshing…' : 'Refresh' }}
        </button>
      </div>
    </header>

    <p v-if="loading && !data" class="ov-state">Loading cluster data…</p>

    <div v-else-if="!data" class="ov-state ov-state-error">
      <p>Failed to load cluster overview</p>
      <p class="ov-state-detail">{{ error }}</p>
      <button type="button" class="ov-btn ov-btn-primary" @click="refresh">Retry</button>
    </div>

    <template v-else>
      <div class="ov-stats">
        <StatCard
          v-for="s in stats"
          :key="s.title"
          :title="s.title"
          :value="s.value"
          :sub="s.sub"
          :tone="s.tone"
          @select="emit('navigate', s.view)"
        />
      </div>

      <div class="ov-row ov-row-3">
  <ResourceUsageCard :cluster-name="clusterName" :requests="data.requests" @open-settings="emit('navigate', 'settings')" />
  <TopPodsCard :data="hot" :state="hotState" @open="openHotPod" />
  <StorageCard :data="hot" :state="hotState" />
</div>

<div class="side">
  <ProblemPodsCard :pods="data.problemPods ?? []" :total="data.problemPodsTotal" @open="openPod" @view-all="emit('navigate', 'pods')" />
  <WarningsCard :reasons="data.warningReasons ?? []" @view-all="emit('navigate', 'events')" />
</div>

      <div class="ov-row ov-row-main">
        <WorkloadsCard :workloads="data.workloads" @open="openWorkload" />
        <ProblemPodsCard
          :pods="data.problemPods ?? []"
          :total="data.problemPodsTotal"
          @open="openPod"
          @view-all="emit('navigate', 'pods')"
        />
      </div>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed, toRef } from 'vue'
import StatCard from '../overview/StatCard.vue'
import ClusterHealthCard from '../overview/ClusterHealthCard.vue'
import ResourceUsageCard from '../overview/ResourceUsageCard.vue'
import WarningsCard from '../overview/WarningsCard.vue'
import WorkloadsCard from '../overview/WorkloadsCard.vue'
import ProblemPodsCard from '../overview/ProblemPodsCard.vue'
import { useOverview } from '../../composables/useOverview'
import { useViewHotkeys } from '../../composables/useViewHotkeys'
import { resourceActions } from '../../resources/actions'
import type { ProblemPod, WorkloadSummary } from '../../types/overview'
import '../overview/overview.css'
import ClusterHealthChip from '../overview/ClusterHealthChip.vue'
import TopPodsCard from '../overview/TopPodsCard.vue'
import StorageCard from '../overview/StorageCard.vue'
import { useHotspots } from '../../composables/useHotspots'
import type { HotPod } from '../../types/hotspots'


const props = withDefaults(defineProps<{ clusterName?: string }>(), { clusterName: '' })
const emit = defineEmits<{ (e: 'navigate', view: string): void }>()

const { data: hot, state: hotState } = useHotspots(toRef(props, 'clusterName'))
const openHotPod = (p: HotPod) => resourceActions.open('pod', p.namespace, p.name)

const { data, loading, refreshing, error, updatedAt, refresh } = useOverview(toRef(props, 'clusterName'))

// Alt+P / D / N / E / S: раньше их регистрировала карточка Shortcuts
useViewHotkeys(view => emit('navigate', view))

type Tone = 'ok' | 'warn' | 'danger'

const stats = computed(() => {
  const o = data.value
  if (!o) return []

  const nodesTone: Tone = o.totalNodes > 0 && o.readyNodes === 0 ? 'danger' : o.readyNodes < o.totalNodes ? 'warn' : 'ok'
  const podsTone: Tone = o.failedPods > 0 ? 'danger' : o.problemPodsTotal > 0 ? 'warn' : 'ok'
  const podsSub = o.problemPodsTotal > 0 ? `Problems: ${o.problemPodsTotal} · Running: ${o.runningPods}` : `Running: ${o.runningPods}`
  const restartsTone: Tone = o.oomKilledLast24h > 0 ? 'danger' : o.restartsLast24h > 0 ? 'warn' : 'ok'
  const topReason = o.warningReasons?.[0]

  return [
    { title: 'Nodes', value: o.totalNodes, sub: `Ready: ${o.readyNodes}`, tone: nodesTone, view: 'nodes' },
    { title: 'Pods', value: o.totalPods, sub: podsSub, tone: podsTone, view: 'pods' },
    {
      title: 'Restarts (24h)',
      value: o.restartsLast24h,
      sub: o.oomKilledLast24h > 0 ? `OOMKilled: ${o.oomKilledLast24h}` : o.restartsLast24h > 0 ? 'Containers restarted' : 'No restarts',
      tone: restartsTone,
      view: 'pods',
    },
    {
      title: 'Warnings (1h)',
      value: o.warningsLastHour,
      sub: topReason ? `${topReason.reason} ×${topReason.count}` : 'No warnings',
      tone: (o.warningsLastHour > 0 ? 'warn' : 'ok') as Tone,
      view: 'events',
    },
  ]
})

function openWorkload(w: WorkloadSummary) {
  resourceActions.open(w.kind.toLowerCase(), w.namespace, w.name)
}

function openPod(p: ProblemPod) {
  resourceActions.open('pod', p.namespace, p.name)
}
</script>

<style scoped>
.ov-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 18px; }
.ov-header h1 { font-size: 20px; font-weight: 600; margin: 0; }
.ov-header-right { display: flex; align-items: center; gap: 12px; }
.ov-updated { font-size: 12px; color: var(--ov-muted); }
.ov-stale { font-size: 12px; color: var(--ov-warn); }

.ov-state { padding: 60px 0; text-align: center; font-size: 14px; color: var(--ov-muted); }
.ov-state-error { color: var(--ov-text); }
.ov-state-error p { margin-bottom: 8px; }
.ov-state-detail { font-size: 12px; color: var(--ov-muted); }
.ov-state-error .ov-btn { margin-top: 8px; }

.ov-stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: 14px; margin-bottom: 14px; }
.ov-row { display: grid; gap: 14px; margin-bottom: 14px; }
.ov-row-3 { grid-template-columns: repeat(3, minmax(0, 1fr)); }
.ov-row-main { grid-template-columns: minmax(0, 1fr) 320px; margin-bottom: 0; }

@container (max-width: 900px) {
  .ov-row-3 { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .ov-row-3 > :last-child { grid-column: 1 / -1; }
}
@container (max-width: 700px) {
  .ov-row-3, .ov-row-main { grid-template-columns: minmax(0, 1fr); }
}

.side { display: flex; flex-direction: column; gap: 14px; min-width: 0; }
</style>
