<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <svg class="spark" viewBox="0 0 100 36" preserveAspectRatio="none" aria-hidden="true">
    <defs>
      <linearGradient :id="gradientId" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" :stop-color="color" stop-opacity="0.35" />
        <stop offset="100%" :stop-color="color" stop-opacity="0" />
      </linearGradient>
    </defs>
    <template v-if="line">
      <path :d="area" :fill="`url(#${gradientId})`" />
      <path :d="line" fill="none" :stroke="color" stroke-width="1.6" stroke-linejoin="round" vector-effect="non-scaling-stroke" />
    </template>
  </svg>
</template>

<script lang="ts">
// Общий счётчик на модуль: id градиента должен быть уникальным среди всех графиков на странице
let counter = 0
</script>

<script lang="ts" setup>
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  /** Значения ряда по порядку */
  values: number[]
  color?: string
  /** Верх шкалы; по умолчанию максимум ряда (но не меньше 1) */
  max?: number
}>(), { color: '#8b5cf6' })

const gradientId = `spark-grad-${++counter}`
const W = 100
const H = 36
const PAD = 2

const points = computed(() => {
  const v = props.values.filter(n => isFinite(n))
  if (v.length < 2) return []
  const top = props.max ?? Math.max(1, ...v)
  return v.map((n, i) => {
    const x = (i / (v.length - 1)) * W
    const y = H - PAD - (Math.min(Math.max(n, 0), top) / top) * (H - PAD * 2)
    return [x, y] as const
  })
})

const line = computed(() =>
  points.value.length ? points.value.map(([x, y], i) => `${i ? 'L' : 'M'}${x.toFixed(2)} ${y.toFixed(2)}`).join(' ') : '',
)
const area = computed(() => (line.value ? `${line.value} L${W} ${H} L0 ${H} Z` : ''))
</script>

<style scoped>
.spark { display: block; width: 100%; height: 100%; }
</style>
