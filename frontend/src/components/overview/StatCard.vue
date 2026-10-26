<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <button type="button" class="ov-card stat" :class="`tone-${tone}`" @click="emit('select')">
    <span class="stat-title">{{ title }}</span>
    <span class="stat-value">{{ value }}</span>
    <span class="stat-sub">{{ sub }}</span>
  </button>
</template>

<script lang="ts" setup>
withDefaults(defineProps<{
  title: string
  value: string | number
  sub: string
  /** Цвет подписи: ok (обычный), warn, danger */
  tone?: 'ok' | 'warn' | 'danger'
}>(), { tone: 'ok' })

const emit = defineEmits<{ (e: 'select'): void }>()
</script>

<style scoped>
.stat {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  padding: 16px 18px;
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}
.stat:hover { border-color: var(--ov-accent); background: var(--ov-card-hover); }
.stat:focus-visible { outline: 2px solid var(--ov-accent); outline-offset: 2px; }

.stat-title { font-size: 13px; color: var(--ov-muted); }
.stat-value {
  font-size: 30px;
  font-weight: 600;
  line-height: 1.15;
  color: var(--ov-text);
  font-variant-numeric: tabular-nums;
}
.stat-sub { font-size: 12px; color: var(--ov-muted); }
.tone-warn .stat-sub { color: var(--ov-warn); }
.tone-danger .stat-sub { color: var(--ov-danger); }
</style>
