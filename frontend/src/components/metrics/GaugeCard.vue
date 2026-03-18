<template>
  <div class="bg-gray-800 rounded-lg border border-gray-700 p-4">
    <div class="flex items-center gap-4">
      <!-- Circular gauge -->
      <div class="relative w-16 h-16 flex-shrink-0">
        <svg class="w-16 h-16 -rotate-90" viewBox="0 0 64 64">
          <!-- Background circle -->
          <circle
            cx="32" cy="32" r="28"
            fill="none"
            :stroke="trackColor"
            stroke-width="6"
          />
          <!-- Value arc -->
          <circle
            cx="32" cy="32" r="28"
            fill="none"
            :stroke="gaugeColor"
            stroke-width="6"
            stroke-linecap="round"
            :stroke-dasharray="circumference"
            :stroke-dashoffset="dashOffset"
            class="transition-all duration-700 ease-out"
          />
        </svg>
        <div class="absolute inset-0 flex items-center justify-center">
          <span class="text-sm font-bold text-white">{{ displayPercent }}%</span>
        </div>
      </div>

      <!-- Info -->
      <div class="flex-1 min-w-0">
        <p class="text-sm font-medium text-gray-300 truncate">{{ title }}</p>
        <p class="text-xs text-gray-500 mt-1">
          {{ formatValue(used) }} / {{ formatValue(total) }}
        </p>
        <p v-if="subtitle" class="text-xs text-gray-600 mt-0.5">{{ subtitle }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  title: string
  percent: number
  used: number
  total: number
  subtitle?: string
  unit?: 'bytes' | 'cores' | 'count' | 'percent'
  warningThreshold?: number
  criticalThreshold?: number
}>(), {
  unit: 'count',
  warningThreshold: 70,
  criticalThreshold: 90,
})

const radius = 28
const circumference = 2 * Math.PI * radius

const displayPercent = computed(() => {
  return Math.min(Math.round(props.percent), 100)
})

const dashOffset = computed(() => {
  const pct = Math.min(props.percent, 100) / 100
  return circumference * (1 - pct)
})

const trackColor = '#1F2937'

const gaugeColor = computed(() => {
  if (props.percent >= props.criticalThreshold) return '#EF4444' // red
  if (props.percent >= props.warningThreshold) return '#F59E0B'  // amber
  return '#10B981' // green
})

function formatValue(value: number): string {
  switch (props.unit) {
    case 'bytes':
      return formatBytes(value)
    case 'cores':
      if (value < 1) return (value * 1000).toFixed(0) + 'm'
      return value.toFixed(2)
    case 'percent':
      return value.toFixed(1) + '%'
    default:
      if (value >= 1000000) return (value / 1000000).toFixed(1) + 'M'
      if (value >= 1000) return (value / 1000).toFixed(1) + 'K'
      return value.toString()
  }
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  const val = bytes / Math.pow(1024, i)
  return val.toFixed(1) + ' ' + units[i]
}
</script>