<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <section class="ov-card bl">
    <h2 class="ov-card-title">Blockers</h2>

    <template v-if="risks">
      <template v-if="risks.quotas.length">
        <h3 class="sub-title">Quotas near limit <span class="n">{{ risks.quotaTotal }}</span></h3>
        <ul class="list">
          <li v-for="q in risks.quotas.slice(0, 4)" :key="`${q.namespace}/${q.name}/${q.resource}`" class="row">
            <span class="name" :title="`${q.namespace}/${q.name}`">{{ q.namespace }} · {{ q.resource }}</span>
            <span class="val" :class="q.percent >= 100 ? 'd' : 'w'">{{ q.percent }}%</span>
            <span class="sub">{{ q.used }} / {{ q.hard }}</span>
          </li>
        </ul>
      </template>

      <template v-if="risks.pdbs.length">
        <h3 class="sub-title">PDB blocks drain <span class="n">{{ risks.pdbTotal }}</span></h3>
        <ul class="list">
          <li v-for="p in risks.pdbs.slice(0, 4)" :key="`${p.namespace}/${p.name}`" class="row">
            <span class="name" :title="`${p.namespace}/${p.name}`">{{ p.namespace }}/{{ p.name }}</span>
            <span class="sub">healthy {{ p.healthy }} of {{ p.expected }}, needs {{ p.desired }}</span>
          </li>
        </ul>
      </template>

      <p v-if="!risks.quotas.length && !risks.pdbs.length" class="none">
        No quota above {{ 80 }}% and no PodDisruptionBudget blocking evictions
      </p>
    </template>
    <p v-else class="none">Loading…</p>
  </section>
</template>

<script lang="ts" setup>
import type { OverviewRisks } from '../../types/insights'

defineProps<{ risks: OverviewRisks | null }>()
</script>

<style scoped>
.sub-title { margin: 0 0 6px; font-size: 12px; font-weight: 500; color: var(--ov-muted); }
.sub-title:not(:first-of-type) { margin-top: 14px; }
.n { color: var(--ov-warn); margin-left: 4px; }
.list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 8px; }
.row { display: grid; grid-template-columns: minmax(0, 1fr) auto; grid-template-areas: 'name val' 'sub sub'; gap: 2px 8px; }
.name { grid-area: name; font-size: 13px; font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.val { grid-area: val; font-size: 13px; font-weight: 600; font-variant-numeric: tabular-nums; }
.sub { grid-area: sub; font-size: 11px; color: var(--ov-muted); }
.w { color: var(--ov-warn); } .d { color: var(--ov-danger); }
.none { margin: 0; font-size: 13px; color: var(--ov-ok); }
</style>
