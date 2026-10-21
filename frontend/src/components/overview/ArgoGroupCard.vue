<template>
  <section class="ov-card">
    <h2 class="ov-card-title">{{ title }}</h2>
    <ul class="grp">
      <li v-for="r in rows" :key="r.name" class="grp-row">
        <span class="grp-name" :title="r.name">{{ r.name }}</span>
        <span class="grp-bar" aria-hidden="true">
          <span class="grp-fill" :class="{ bad: r.problems > 0 }" :style="{ width: pct(r) + '%' }"></span>
        </span>
        <span class="grp-count">
          {{ r.total }}<span v-if="r.problems" class="grp-bad"> · {{ r.problems }} ⚠</span>
        </span>
      </li>
    </ul>
  </section>
</template>

<script lang="ts" setup>
import { computed } from 'vue'

interface Row { name: string; total: number; problems: number }
const props = defineProps<{ title: string; rows: Row[] }>()

const max = computed(() => Math.max(1, ...props.rows.map(r => r.total)))
const pct = (r: Row) => Math.round((r.total / max.value) * 100)
</script>

<style scoped>
.grp { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 10px; }
.grp-row { display: grid; grid-template-columns: minmax(80px, 160px) 1fr auto; align-items: center; gap: 12px; font-size: 13px; }
.grp-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.grp-bar { height: 6px; background: var(--ov-border-soft); border-radius: 3px; overflow: hidden; }
.grp-fill { display: block; height: 100%; background: var(--ov-accent); border-radius: 3px; }
.grp-fill.bad { background: var(--ov-warn); }
.grp-count { font-variant-numeric: tabular-nums; color: var(--ov-muted); }
.grp-bad { color: var(--ov-warn); }
</style>
