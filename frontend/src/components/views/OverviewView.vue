<template>
  <div class="ov">
    <header class="ov-header">
      <h1>Overview</h1>
      <div class="ov-header-right">
        <span v-if="data && error" class="ov-stale" :title="error">Update failed, showing last data</span>
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
        <ClusterHealthCard :health="data.health" />
        <ResourceUsageCard :cluster-name="clusterName" @open-settings="emit('navigate', 'settings')" />
        <RecentEventsCard :events="data.recentEvents" @view-all="emit('navigate', 'events')" />
      </div>

      <div class="ov-row ov-row-main">
        <WorkloadsCard :workloads="data.workloads" @open="openWorkload" />
        <ShortcutsCard @navigate="view => emit('navigate', view)" />
      </div>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed, toRef } from 'vue'
import StatCard from '../overview/StatCard.vue'
import ClusterHealthCard from '../overview/ClusterHealthCard.vue'
import ResourceUsageCard from '../overview/ResourceUsageCard.vue'
import RecentEventsCard from '../overview/RecentEventsCard.vue'
import WorkloadsCard from '../overview/WorkloadsCard.vue'
import ShortcutsCard from '../overview/ShortcutsCard.vue'
import { useOverview } from '../../composables/useOverview'
import { resourceActions } from '../../resources/actions'
import type { WorkloadSummary } from '../../types/overview'
import '../overview/overview.css'

const props = withDefaults(defineProps<{ clusterName?: string }>(), { clusterName: '' })
const emit = defineEmits<{ (e: 'navigate', view: string): void }>()

const { data, loading, refreshing, error, updatedAt, refresh } = useOverview(toRef(props, 'clusterName'))

type Tone = 'ok' | 'warn' | 'danger'

const stats = computed(() => {
  const o = data.value
  if (!o) return []

  const podExtra = [
    o.failedPods > 0 ? `Failed: ${o.failedPods}` : '',
    o.pendingPods > 0 ? `Pending: ${o.pendingPods}` : '',
  ].filter(Boolean)

  const nodesTone: Tone = o.totalNodes > 0 && o.readyNodes === 0 ? 'danger' : o.readyNodes < o.totalNodes ? 'warn' : 'ok'
  const podsTone: Tone = o.failedPods > 0 ? 'danger' : o.pendingPods > 0 ? 'warn' : 'ok'

  return [
    { title: 'Nodes', value: o.totalNodes, sub: `Ready: ${o.readyNodes}`, tone: nodesTone, view: 'nodes' },
    {
      title: 'Pods',
      value: o.totalPods,
      sub: [`Running: ${o.runningPods}`, ...podExtra].join(' · '),
      tone: podsTone,
      view: 'pods',
    },
    {
      title: 'Deployments',
      value: o.totalDeployments,
      sub: `Ready: ${o.readyDeployments}`,
      tone: (o.readyDeployments < o.totalDeployments ? 'warn' : 'ok') as Tone,
      view: 'deployments',
    },
    {
      title: 'Namespaces',
      value: o.totalNamespaces,
      sub: o.activeNamespaces === o.totalNamespaces ? 'Active' : `Active: ${o.activeNamespaces}`,
      tone: (o.activeNamespaces < o.totalNamespaces ? 'warn' : 'ok') as Tone,
      view: 'namespaces',
    },
    {
      title: 'Events',
      value: o.warningEvents,
      sub: o.warningEvents > 0 ? 'Warning' : 'No warnings',
      tone: (o.warningEvents > 0 ? 'warn' : 'ok') as Tone,
      view: 'events',
    },
  ]
})

function openWorkload(w: WorkloadSummary) {
  resourceActions.open(w.kind.toLowerCase(), w.namespace, w.name)
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

.ov-stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); gap: 14px; margin-bottom: 14px; }
.ov-row { display: grid; gap: 14px; margin-bottom: 14px; }
.ov-row-3 { grid-template-columns: repeat(3, minmax(0, 1fr)); }
.ov-row-main { grid-template-columns: minmax(0, 1fr) 260px; margin-bottom: 0; }

/* Адаптация по ширине самого экрана, а не окна: боковые панели съедают место */
@container (max-width: 900px) {
  .ov-row-3 { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  /* последняя карточка занимает всю строку, чтобы не оставалось пустой ячейки */
  .ov-row-3 > :last-child { grid-column: 1 / -1; }
}
@container (max-width: 700px) {
  .ov-row-3, .ov-row-main { grid-template-columns: minmax(0, 1fr); }
}
</style>
