<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <section class="ov-card st">
    <h2 class="ov-card-title">{{ $t('overview.storage.title') }}</h2>

    <ul v-if="state === 'ready' && vols.length" class="list">
      <li v-for="v in vols" :key="`${v.namespace}/${v.claim}`" class="row" :title="`${v.namespace}/${v.claim}`">
        <span class="name">{{ v.claim }}</span>
        <span class="val" :class="tone(v.percent)">{{ Math.round(v.percent) }}%</span>
        <span class="bar" aria-hidden="true"><span class="fill" :class="tone(v.percent)" :style="{ width: Math.min(v.percent, 100) + '%' }"></span></span>
        <span class="sub">{{ v.namespace }} · {{ formatBytes(v.used) }} / {{ formatBytes(v.capacity) }}</span>
      </li>
    </ul>
    <p v-else-if="state === 'ready'" class="none">{{ $t('overview.storage.empty') }}</p>
    <p v-else-if="state === 'loading'" class="none">{{ $t('common.loading') }}</p>
    <p v-else class="none">{{ $t('overview.storage.needMetrics') }}</p>

    <p v-if="data && (data.pvcPending || data.pvcLost)" class="alert">
      <span v-if="data.pvcPending">{{ $t('overview.storage.pvcPending', { n: data.pvcPending }) }}</span>
      <span v-if="data.pvcLost">{{ $t('overview.storage.pvcLost', { n: data.pvcLost }) }}</span>
    </p>
  </section>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { formatBytes } from './format'
import type { MetricsState, OverviewHotspots } from '../../types/hotspots'

const props = defineProps<{ data: OverviewHotspots | null; state: MetricsState }>()
const vols = computed(() => props.data?.volumes ?? [])
const tone = (pct: number) => (pct >= 90 ? 'd' : pct >= 80 ? 'w' : 'o')
</script>

<style scoped>
.list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 10px; }
.row { display: grid; grid-template-columns: minmax(0, 1fr) auto; grid-template-areas: 'name val' 'bar bar' 'sub sub'; gap: 3px 8px; }
.name { grid-area: name; font-size: 13px; font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.val { grid-area: val; font-size: 13px; font-weight: 600; font-variant-numeric: tabular-nums; }
.bar { grid-area: bar; height: 4px; background: var(--ov-border-soft); border-radius: 2px; overflow: hidden; }
.fill { display: block; height: 100%; border-radius: 2px; }
.sub { grid-area: sub; font-size: 11px; color: var(--ov-muted); }
.o { color: var(--ov-ok); } .w { color: var(--ov-warn); } .d { color: var(--ov-danger); }
.fill.o { background: var(--ov-ok); } .fill.w { background: var(--ov-warn); } .fill.d { background: var(--ov-danger); }
.none { margin: 0; font-size: 13px; color: var(--ov-muted); }
.alert { display: flex; gap: 12px; margin: 12px 0 0; font-size: 12px; font-weight: 600; color: var(--ov-warn); }
</style>
