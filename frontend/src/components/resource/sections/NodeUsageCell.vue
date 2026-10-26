<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div v-if="usage" class="nu" :title="tip">
    <div class="nu-track">
      <div class="nu-fill" :class="`nu-${col.key}`" :style="{ width: width + '%' }"></div>
    </div>
  </div>
  <span v-else class="nu-none">—</span>
</template>

<script lang="ts" setup>
import { computed } from 'vue'

const props = defineProps<{ item: any; col: any }>()

interface Usage { pct: number; used: number; total: number }

const LABELS: Record<string, string> = { cpu: 'CPU', memory: 'Memory', disk: 'Disk' }

const usage = computed<Usage | null>(() => {
  const u = props.item?.usage?.[props.col.key]
  return u && u.total > 0 ? u : null
})

const width = computed(() => Math.min(Math.max(usage.value?.pct ?? 0, 0), 100))

const KIB = 1024
const UNITS: [number, string][] = [
  [KIB ** 4, 'TiB'],
  [KIB ** 3, 'GiB'],
  [KIB ** 2, 'MiB'],
  [KIB, 'KiB'],
]

/** Байты в читаемый вид: «2.753GiB». trim убирает хвостовые нули (4.500GiB -> 4.5GiB) */
function bytes(v: number, trim = false): string {
  for (const [size, unit] of UNITS) {
    if (v >= size) {
      const s = (v / size).toFixed(3)
      return `${trim ? String(parseFloat(s)) : s}${unit}`
    }
  }
  return `${Math.round(v)}B`
}

/** Ядра: целые без дробной части, иначе до двух знаков */
const cores = (v: number) => (Number.isInteger(v) ? String(v) : String(parseFloat(v.toFixed(2))))

const tip = computed(() => {
  const u = usage.value
  if (!u) return ''
  const label = LABELS[props.col.key] ?? props.col.label
  const pct = `${u.pct.toFixed(2)}%`
  if (props.col.key === 'cpu') return `${label}: ${pct}, ${cores(u.used)} of ${cores(u.total)} cores`
  return `${label}: ${pct}, ${bytes(u.used)} of ${bytes(u.total, true)}`
})
</script>

<style scoped>
.nu { display: flex; align-items: center; height: 100%; cursor: help; }
.nu-track { width: 74px; height: 4px; background: #2f3040; border-radius: 2px; overflow: hidden; }
.nu-fill { height: 100%; border-radius: 2px; transition: width 0.4s ease; min-width: 2px; }
.nu-cpu { background: #3b9edb; }
.nu-memory { background: #c04fc9; }
.nu-disk { background: #f2c94c; }
.nu-none { color: #6b6d8a; }
</style>