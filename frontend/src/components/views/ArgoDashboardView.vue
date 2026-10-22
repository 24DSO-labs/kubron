<template>
  <div class="ov">
    <header class="ov-header">
      <h1>Argo CD</h1>
      <div class="ov-header-right">
        <span v-if="apps && error" class="ov-stale" :title="error">Update failed, showing last data</span>
        <span v-else-if="updatedAt" class="ov-updated">Updated {{ updatedAt.toLocaleTimeString() }}</span>
        <button v-if="can(Perm.PortForward)" type="button" class="ov-btn ov-btn-auto" @click="openArgoUI">Open Argo
          UI</button>
        <button v-if="apps?.length && can(Perm.Operate)" type="button" class="ov-btn ov-btn-auto"
          @click="requestArgoSync(apps)">Sync…</button>
        <button type="button" class="ov-btn" :disabled="refreshing" @click="refresh">
          {{ refreshing ? 'Refreshing…' : 'Refresh' }}
        </button>
      </div>
    </header>

    <p v-if="loading && !apps" class="ov-state">Loading applications…</p>

    <div v-else-if="!apps" class="ov-state ov-state-error">
      <p>Failed to load Argo CD applications</p>
      <p class="ov-state-detail">{{ error }}</p>
      <button type="button" class="ov-btn ov-btn-primary" @click="refresh">Retry</button>
    </div>

    <p v-else-if="!summary.total" class="ov-state">No applications found in any namespace</p>

    <template v-else>
      <div class="ov-stats">
        <StatCard v-for="s in stats" :key="s.title" :title="s.title" :value="s.value" :sub="s.sub" :tone="s.tone"
          @select="emit('navigate', 'argo-applications')" />
      </div>

      <div class="ov-row ov-row-2">
        <section class="ov-card">
          <h2 class="ov-card-title">Needs attention</h2>
          <p v-if="!summary.problems.length" class="argo-empty">All applications are synced and healthy</p>
          <ul v-else class="argo-list">
            <li v-for="a in summary.problems.slice(0, 10)" :key="key(a)">
              <button type="button" class="argo-row" :class="{ 'argo-row-flash': changedKeys.has(key(a)) }" @click="open(a)">
                <span class="argo-name">{{ a.name }}</span>
                <span class="argo-sub">{{ a.project }} · {{ argoDestination(a) }}</span>
                <span class="argo-tags">
                  <span class="argo-tag" :class="tone(a.syncStatus)">{{ a.syncStatus }}</span>
                  <span class="argo-tag" :class="tone(a.healthStatus)">{{ a.healthStatus }}</span>
                  <span v-if="a.lastSyncPhase === 'Failed' || a.lastSyncPhase === 'Error'" class="argo-tag t-err">Sync
                    {{ a.lastSyncPhase }}</span>
                </span>
              </button>
            </li>
          </ul>
          <button v-if="summary.problems.length > 10" type="button" class="argo-more"
            @click="emit('navigate', 'argo-applications')">
            View all ({{ summary.problems.length }})
          </button>
        </section>

        <section class="ov-card">
          <h2 class="ov-card-title">Recent syncs</h2>
          <p v-if="!summary.recentSyncs.length" class="argo-empty">No syncs yet</p>
          <ul v-else class="argo-list">
            <li v-for="a in summary.recentSyncs" :key="key(a)">
              <button type="button" class="argo-row" :class="{ 'argo-row-flash': changedKeys.has(key(a)) }" @click="open(a)">
                <span class="argo-name">{{ a.name }}</span>
                <span class="argo-sub">{{ formatWhen(a.lastSyncAt) }} · {{ shortRevision(a.revision) || '—' }}</span>
                <span class="argo-tags"><span class="argo-tag" :class="tone(a.lastSyncPhase)">{{ a.lastSyncPhase || '—'
                    }}</span></span>
              </button>
            </li>
          </ul>
        </section>
      </div>

      <div class="ov-row ov-row-2">
        <GroupCard title="By project" :rows="summary.byProject" />
        <GroupCard title="By destination cluster" :rows="summary.byCluster" />
      </div>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed, toRef } from 'vue'
import StatCard from '../overview/StatCard.vue'
import GroupCard from '../overview/ArgoGroupCard.vue'
import { useArgoApplications } from '../../composables/useArgoApplications'
import { argoDestination, shortRevision, summarizeArgo } from '../../resources/argocd'
import { resourceActions } from '../../resources/actions'
import type { ArgoApplicationInfo } from '../../types/argocd'
import '../overview/overview.css'
import { can } from '../../composables/useAuth'
import { Perm } from '../../types/auth'
import { openArgoUI, requestArgoSync } from '../../resources/argoActions'

const props = withDefaults(defineProps<{ clusterName?: string }>(), { clusterName: '' })
const emit = defineEmits<{ (e: 'navigate', view: string): void }>()

const { apps, loading, refreshing, error, updatedAt, changedKeys, refresh } = useArgoApplications(toRef(props, 'clusterName'))

const summary = computed(() => summarizeArgo(apps.value ?? []))

type Tone = 'ok' | 'warn' | 'danger'

const stats = computed(() => {
  const s = summary.value
  const syncTone: Tone = s.outOfSync > 0 ? 'warn' : 'ok'
  const healthTone: Tone = s.degraded + s.missing > 0 ? 'danger' : s.progressing > 0 ? 'warn' : 'ok'
  return [
    { title: 'Applications', value: s.total, sub: s.syncing ? `Syncing: ${s.syncing}` : 'Total', tone: 'ok' as Tone },
    { title: 'Synced', value: s.synced, sub: s.outOfSync ? `Out of sync: ${s.outOfSync}` : 'All in sync', tone: syncTone },
    { title: 'Healthy', value: s.healthy, sub: `Of ${s.total}`, tone: healthTone },
    {
      title: 'Degraded',
      value: s.degraded + s.missing,
      sub: s.missing ? `Missing: ${s.missing}` : s.degraded ? 'Needs attention' : 'None',
      tone: (s.degraded + s.missing > 0 ? 'danger' : 'ok') as Tone,
    },
    {
      title: 'Progressing',
      value: s.progressing,
      sub: s.suspended ? `Suspended: ${s.suspended}` : s.unknown ? `Unknown: ${s.unknown}` : 'In progress',
      tone: (s.progressing > 0 ? 'warn' : 'ok') as Tone,
    },
  ]
})

const key = (a: ArgoApplicationInfo) => `${a.namespace}/${a.name}`

function tone(status: string): string {
  switch (status) {
    case 'Synced': case 'Healthy': case 'Succeeded': return 't-ok'
    case 'Degraded': case 'Missing': case 'Failed': case 'Error': return 't-err'
    case 'OutOfSync': case 'Progressing': case 'Running': case 'Suspended': return 't-warn'
    default: return 't-unk'
  }
}

function formatWhen(iso: string): string {
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString()
}

function open(a: ArgoApplicationInfo) {
  resourceActions.open('application', a.namespace, a.name, a)
}
</script>

<style scoped>
.ov-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
}

.ov-btn-auto { width: auto; }

.ov-header h1 {
  font-size: 20px;
  font-weight: 600;
  margin: 0;
}

.ov-header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.ov-updated {
  font-size: 12px;
  color: var(--ov-muted);
}

.ov-stale {
  font-size: 12px;
  color: var(--ov-warn);
}

.ov-state {
  padding: 60px 0;
  text-align: center;
  font-size: 14px;
  color: var(--ov-muted);
}

.ov-state-error {
  color: var(--ov-text);
}

.ov-state-error p {
  margin-bottom: 8px;
}

.ov-state-detail {
  font-size: 12px;
  color: var(--ov-muted);
}

.ov-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 14px;
  margin-bottom: 14px;
}

.ov-row {
  display: grid;
  gap: 14px;
  margin-bottom: 14px;
}

.ov-row-2 {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

@container (max-width: 760px) {
  .ov-row-2 {
    grid-template-columns: minmax(0, 1fr);
  }
}

.argo-empty {
  margin: 0;
  font-size: 13px;
  color: var(--ov-muted);
}

.argo-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.argo-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  grid-template-areas: 'name tags' 'sub tags';
  gap: 2px 12px;
  align-items: center;
  width: 100%;
  padding: 8px 10px;
  background: transparent;
  border: 0;
  border-radius: 8px;
  font: inherit;
  text-align: left;
  color: var(--ov-text);
  cursor: pointer;
}

.argo-row:hover {
  background: var(--ov-card-hover);
}

/* Приложение только что изменилось (событие argocd:changed): короткая подсветка, без перерисовки списка */
.argo-row-flash {
  animation: argo-flash 3s ease-out;
}

@keyframes argo-flash {
  from { background: color-mix(in srgb, var(--ov-accent) 28%, transparent); }
  to { background: transparent; }
}

@media (prefers-reduced-motion: reduce) {
  .argo-row-flash { animation: none; }
}

.argo-row:focus-visible {
  outline: 2px solid var(--ov-accent);
  outline-offset: -2px;
}

.argo-name {
  grid-area: name;
  font-size: 13px;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.argo-sub {
  grid-area: sub;
  font-size: 12px;
  color: var(--ov-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.argo-tags {
  grid-area: tags;
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.argo-tag {
  padding: 1px 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
}

.t-ok {
  color: var(--ov-ok);
  background: rgba(74, 222, 128, 0.12);
}

.t-warn {
  color: var(--ov-warn);
  background: rgba(251, 191, 36, 0.12);
}

.t-err {
  color: var(--ov-danger);
  background: rgba(248, 113, 113, 0.12);
}

.t-unk {
  color: var(--ov-muted);
  background: rgba(139, 145, 173, 0.12);
}

.argo-more {
  margin-top: 8px;
  padding: 6px 10px;
  background: transparent;
  border: 0;
  font: inherit;
  font-size: 12px;
  color: var(--ov-accent);
  cursor: pointer;
  align-self: flex-start;
}

.argo-more:hover {
  text-decoration: underline;
}
</style>
