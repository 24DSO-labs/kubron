<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <section class="ov-card tp">
    <header class="head">
      <h2 class="ov-card-title">{{ $t('overview.topPods.title') }}</h2>
      <div class="tabs" role="tablist">
        <button v-for="tb in TABS" :key="tb.id" type="button" role="tab" :aria-selected="tab === tb.id"
          :class="{ on: tab === tb.id }" @click="tab = tb.id">{{ tb.label() }}</button>
      </div>
    </header>

    <ul v-if="state === 'ready' && rows.length" class="list">
      <li v-for="p in rows" :key="`${p.namespace}/${p.name}`">
        <button type="button" class="row" @click="emit('open', p)">
          <span class="name" :title="p.name">{{ p.name }}</span>
          <span class="val" :class="tone(p.percent)">{{ Math.round(p.percent) }}%</span>
          <span class="bar" aria-hidden="true"><span class="fill" :class="tone(p.percent)" :style="{ width: Math.min(p.percent, 100) + '%' }"></span></span>
          <span class="sub">{{ p.namespace }} · {{ $t('overview.topPods.usedOfLimit', { used: used(p) }) }}</span>
        </button>
      </li>
    </ul>
    <p v-else-if="state === 'ready'" class="none">{{ tab === 'memory' ? $t('overview.topPods.emptyMemory') : $t('overview.topPods.emptyCpu') }}</p>
    <p v-else-if="state === 'loading'" class="none">{{ $t('common.loading') }}</p>
    <p v-else class="none">{{ $t('overview.topPods.needMetrics') }}</p>
  </section>
</template>

<script lang="ts" setup>
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatBytes, formatCoresLabel } from './format'
import type { HotPod, MetricsState, OverviewHotspots } from '../../types/hotspots'

const props = defineProps<{ data: OverviewHotspots | null; state: MetricsState }>()
const emit = defineEmits<{ (e: 'open', p: HotPod): void }>()

const { t } = useI18n()
// label — функция: текст считается при рендере, чтобы переключение языка его обновляло
const TABS = [
  { id: 'memory', label: () => t('overview.topPods.memory') },
  { id: 'cpu', label: () => 'CPU' },
] as const
const tab = ref<'memory' | 'cpu'>('memory')

const rows = computed(() => (tab.value === 'memory' ? props.data?.topMemory : props.data?.topCpu) ?? [])
const used = (p: HotPod) => (tab.value === 'memory' ? formatBytes(p.used) : formatCoresLabel(p.used))
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
