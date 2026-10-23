<template>
  <section class="ov-card tp">
    <header class="head">
      <h2 class="ov-card-title">Closest to limit</h2>
      <div class="tabs" role="tablist">
        <button v-for="t in TABS" :key="t.id" type="button" role="tab" :aria-selected="tab === t.id"
          :class="{ on: tab === t.id }" @click="tab = t.id">{{ t.label }}</button>
      </div>
    </header>

    <ul v-if="state === 'ready' && rows.length" class="list">
      <li v-for="p in rows" :key="`${p.namespace}/${p.name}`">
        <button type="button" class="row" @click="emit('open', p)">
          <span class="name" :title="p.name">{{ p.name }}</span>
          <span class="val" :class="tone(p.percent)">{{ Math.round(p.percent) }}%</span>
          <span class="bar" aria-hidden="true"><span class="fill" :class="tone(p.percent)" :style="{ width: Math.min(p.percent, 100) + '%' }"></span></span>
          <span class="sub">{{ p.namespace }} · {{ used(p) }} of limit</span>
        </button>
      </li>
    </ul>
    <p v-else-if="state === 'ready'" class="none">No pods with {{ tab === 'memory' ? 'memory' : 'CPU' }} limits</p>
    <p v-else-if="state === 'loading'" class="none">Loading…</p>
    <p v-else class="none">Install the metrics stack in cluster settings to see this</p>
  </section>
</template>

<script lang="ts" setup>
import { computed, ref } from 'vue'
import { formatBytes, formatCores } from './format'
import type { HotPod, MetricsState, OverviewHotspots } from '../../types/hotspots'

const props = defineProps<{ data: OverviewHotspots | null; state: MetricsState }>()
const emit = defineEmits<{ (e: 'open', p: HotPod): void }>()

const TABS = [{ id: 'memory', label: 'Memory' }, { id: 'cpu', label: 'CPU' }] as const
const tab = ref<'memory' | 'cpu'>('memory')

const rows = computed(() => (tab.value === 'memory' ? props.data?.topMemory : props.data?.topCpu) ?? [])
const used = (p: HotPod) => (tab.value === 'memory' ? formatBytes(p.used) : `${formatCores(p.used)} cores`)
// Память у limit — риск OOMKilled; CPU у limit — троттлинг, он менее опасен, поэтому пороги мягче
const tone = (pct: number) => {
  const [warn, danger] = tab.value === 'memory' ? [75, 90] : [90, 100]
  return pct >= danger ? 'd' : pct >= warn ? 'w' : 'o'
}
</script>

<style scoped>
.head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.head .ov-card-title { margin: 0; }
.tabs { display: flex; gap: 4px; }
.tabs button { padding: 2px 10px; background: transparent; border: 1px solid var(--ov-border); border-radius: 999px; font: inherit; font-size: 11px; color: var(--ov-muted); cursor: pointer; }
.tabs button.on { color: var(--ov-text); border-color: var(--ov-accent); }
.list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 4px; }
.row { display: grid; grid-template-columns: minmax(0, 1fr) auto; grid-template-areas: 'name val' 'bar bar' 'sub sub'; gap: 3px 8px; width: 100%; padding: 6px 8px; background: transparent; border: 0; border-radius: 8px; font: inherit; text-align: left; color: var(--ov-text); cursor: pointer; }
.row:hover, .row:focus-visible { background: var(--ov-card-hover); outline: none; }
.name { grid-area: name; font-size: 13px; font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.val { grid-area: val; font-size: 13px; font-weight: 600; font-variant-numeric: tabular-nums; }
.bar { grid-area: bar; height: 4px; background: var(--ov-border-soft); border-radius: 2px; overflow: hidden; }
.fill { display: block; height: 100%; border-radius: 2px; }
.sub { grid-area: sub; font-size: 11px; color: var(--ov-muted); }
.o { color: var(--ov-ok); } .w { color: var(--ov-warn); } .d { color: var(--ov-danger); }
.fill.o { background: var(--ov-ok); } .fill.w { background: var(--ov-warn); } .fill.d { background: var(--ov-danger); }
.none { margin: 0; font-size: 13px; color: var(--ov-muted); }
</style>
