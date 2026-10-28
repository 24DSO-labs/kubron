<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <section class="ov-card cap">
    <h2 class="ov-card-title">{{ $t('overview.capacity.title') }}</h2>

    <template v-if="nodes && nodes.length">
      <div class="best">
        <span class="best-label">{{ $t('overview.capacity.largestSlot') }}</span>
        <span class="best-val">{{ formatCoresLabel(best.cpuFree) }} · {{ formatBytes(best.memoryFree) }}</span>
        <span class="best-node" :title="best.name">{{ $t('overview.capacity.on', { node: best.name }) }}</span>
      </div>

      <ul class="list">
        <li v-for="n in tight" :key="n.name" class="row" :title="n.name">
          <span class="name">{{ n.name }}</span>
          <span class="bars" aria-hidden="true">
            <span class="bar"><span class="fill" :class="tone(n.cpuFree, n.cpuAllocatable)" :style="{ width: usedPct(n.cpuFree, n.cpuAllocatable) + '%' }"></span></span>
            <span class="bar"><span class="fill" :class="tone(n.memoryFree, n.memoryAllocatable)" :style="{ width: usedPct(n.memoryFree, n.memoryAllocatable) + '%' }"></span></span>
          </span>
          <span class="sub">{{ $t('overview.capacity.free', { value: `${formatCoresLabel(n.cpuFree)} · ${formatBytes(n.memoryFree)}` }) }}</span>
        </li>
      </ul>
      <p class="hint">{{ $t('overview.capacity.hint') }}</p>
    </template>
    <p v-else-if="nodes" class="none">{{ $t('overview.capacity.empty') }}</p>
    <p v-else class="none">{{ $t('common.loading') }}</p>
  </section>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { formatBytes, formatCoresLabel } from './format'
import type { NodeHeadroom } from '../../types/insights'

const props = defineProps<{ nodes: NodeHeadroom[] | null }>()

/** «Слот» ограничен тем, что кончится раньше: берём узел с наибольшим min(CPU, память) в долях от самого свободного */
const best = computed(() => {
  const list = props.nodes ?? []
  const maxCpu = Math.max(...list.map(n => n.cpuFree), 1e-9)
  const maxMem = Math.max(...list.map(n => n.memoryFree), 1)
  return list.reduce((a, b) =>
    Math.min(b.cpuFree / maxCpu, b.memoryFree / maxMem) > Math.min(a.cpuFree / maxCpu, a.memoryFree / maxMem) ? b : a)
})

/** Самые заполненные узлы (по худшей из двух долей); на большом кластере — только три */
const tight = computed(() =>
  [...(props.nodes ?? [])]
    .sort((a, b) => Math.max(usedFrac(b.cpuFree, b.cpuAllocatable), usedFrac(b.memoryFree, b.memoryAllocatable))
      - Math.max(usedFrac(a.cpuFree, a.cpuAllocatable), usedFrac(a.memoryFree, a.memoryAllocatable)))
    .slice(0, 3))

const usedFrac = (free: number, total: number) => (total > 0 ? 1 - free / total : 0)
const usedPct = (free: number, total: number) => Math.round(usedFrac(free, total) * 100)
const tone = (free: number, total: number) => {
  const u = usedPct(free, total)
  return u >= 90 ? 'd' : u >= 80 ? 'w' : 'o'
}
</script>

<style scoped>
.best { display: flex; flex-direction: column; gap: 2px; margin-bottom: 14px; }
.best-label { font-size: 12px; color: var(--ov-muted); }
.best-val { font-size: 20px; font-weight: 600; font-variant-numeric: tabular-nums; }
.best-node { font-size: 12px; color: var(--ov-muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 10px; }
.row { display: grid; grid-template-columns: minmax(0, 1fr) 90px; grid-template-areas: 'name bars' 'sub sub'; gap: 3px 10px; align-items: center; }
.name { grid-area: name; font-size: 13px; font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.bars { grid-area: bars; display: flex; flex-direction: column; gap: 3px; }
.bar { height: 4px; background: var(--ov-border-soft); border-radius: 2px; overflow: hidden; }
.fill { display: block; height: 100%; border-radius: 2px; }
.fill.o { background: var(--ov-ok); } .fill.w { background: var(--ov-warn); } .fill.d { background: var(--ov-danger); }
.sub { grid-area: sub; font-size: 11px; color: var(--ov-muted); }
.hint { margin: 12px 0 0; font-size: 11px; color: var(--ov-muted); }
.none { margin: 0; font-size: 13px; color: var(--ov-muted); }
</style>
