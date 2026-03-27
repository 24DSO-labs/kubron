<template>
  <div class="bg-gradient-to-br from-gray-800 to-gray-900 rounded-xl border border-gray-700/50 p-5 shadow-lg">
    <div class="flex items-start justify-between mb-3">
      <h3 class="text-sm font-semibold text-gray-300">{{ title }}</h3>
      <span class="text-xs font-medium px-2 py-1 rounded bg-gray-700/50 text-gray-300">
        {{ percent.toFixed(1) }}%
      </span>
    </div>

    <!-- Gauge bar -->
    <div class="mb-3">
      <div class="h-2 bg-gray-700 rounded-full overflow-hidden">
        <div
          class="h-full bg-gradient-to-r from-blue-500 to-cyan-500 transition-all duration-300"
          :style="{ width: Math.min(Math.max(percent, 0), 100) + '%' }"
        ></div>
      </div>
    </div>

    <!-- Values -->
    <div class="flex items-end justify-between">
      <div>
        <p class="text-2xl font-bold text-white">
          {{ formatValue(used) }}
        </p>
        <p class="text-xs text-gray-500 mt-0.5">
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
