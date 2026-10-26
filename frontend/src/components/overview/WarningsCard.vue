<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <section class="ov-card warnings">
    <h2 class="ov-card-title">Warnings, last hour</h2>

    <ul v-if="reasons.length" class="list">
      <li v-for="r in reasons" :key="r.reason" class="row">
        <span class="reason" :title="r.reason">{{ r.reason }}</span>
        <span class="bar" aria-hidden="true"><span class="fill" :style="{ width: pct(r) + '%' }"></span></span>
        <span class="n">{{ r.count }}</span>
      </li>
    </ul>
    <p v-else class="none">No warnings in the last hour</p>

    <button type="button" class="more" @click="emit('viewAll')">View all events →</button>
  </section>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import type { ReasonCount } from '../../types/overview'

const props = defineProps<{ reasons: ReasonCount[] }>()
const emit = defineEmits<{ (e: 'viewAll'): void }>()

const max = computed(() => Math.max(1, ...props.reasons.map(r => r.count)))
const pct = (r: ReasonCount) => Math.round((r.count / max.value) * 100)
</script>

<style scoped>
.list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 12px; flex: 1; }
.row { display: grid; grid-template-columns: minmax(80px, 150px) 1fr auto; align-items: center; gap: 12px; font-size: 13px; }
.reason { font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.bar { height: 6px; background: var(--ov-border-soft); border-radius: 3px; overflow: hidden; }
.fill { display: block; height: 100%; background: var(--ov-warn); border-radius: 3px; }
.n { font-variant-numeric: tabular-nums; color: var(--ov-muted); }
.none { flex: 1; margin: 0; font-size: 13px; color: var(--ov-muted); }
.more { align-self: flex-start; margin-top: 12px; padding: 0; background: none; border: 0; font: inherit; font-size: 12px; color: var(--ov-accent); cursor: pointer; }
.more:hover { text-decoration: underline; }
</style>
