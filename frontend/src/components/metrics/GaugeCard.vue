<template>
  <div class="gauge-card">
    <div class="gauge-card-header">
      <h3 class="gauge-card-title">{{ title }}</h3>
      <span class="gauge-card-percent-badge">
        {{ percent.toFixed(1) }}%
      </span>
    </div>

    <!-- Gauge bar -->
    <div class="gauge-card-bar-container">
      <div class="gauge-card-bar-track">
        <div
          class="gauge-card-bar-fill"
          :style="{ width: Math.min(Math.max(percent, 0), 100) + '%' }"
        ></div>
      </div>
    </div>

    <!-- Values -->
    <div class="gauge-card-values">
      <div>
        <p class="gauge-card-value-main">
          {{ formatValue(used) }}
        </p>
        <p class="gauge-card-value-sub">
          of {{ formatValue(total) }} {{ unit }}
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  title: string
  percent: number
  used: number
  total: number
  unit: string
}>()

function formatValue(value: number): string {
  if (value < 0) return '—'
  if (value < 1000) return value.toFixed(0)
  if (value < 1000000) return (value / 1000).toFixed(1) + 'K'
  if (value < 1000000000) return (value / 1000000).toFixed(1) + 'M'
  return (value / 1000000000).toFixed(1) + 'G'
}
</script>
