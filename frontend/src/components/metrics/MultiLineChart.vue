<template>
  <div class="bg-gray-800 rounded-lg border border-gray-700 p-4">
    <div class="flex items-center justify-between mb-3">
      <h3 class="text-sm font-medium text-gray-300">{{ title }}</h3>
      <div class="flex items-center gap-1">
        <button
          v-for="range in timeRanges"
          :key="range.value"
          @click="selectedRange = range.value"
          :class="[
            'px-2 py-0.5 text-xs rounded transition-colors',
            selectedRange === range.value
              ? 'bg-blue-600 text-white'
              : 'bg-gray-700 text-gray-400 hover:bg-gray-600'
          ]"
        >
          {{ range.label }}
        </button>
      </div>
    </div>

    <div v-if="loading && !hasData" class="flex items-center justify-center h-48">
      <div class="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-500"></div>
    </div>

    <div v-else-if="error && !hasData" class="flex items-center justify-center h-48">
      <p class="text-red-400 text-sm">{{ error }}</p>
    </div>

    <div v-else-if="!hasData" class="flex items-center justify-center h-48">
      <p class="text-gray-500 text-sm">No data available</p>
    </div>

    <div v-else class="h-48 relative">
      <div v-if="loading" class="absolute top-0 right-0 z-10">
        <div class="animate-spin rounded-full h-3 w-3 border-b-2 border-blue-500"></div>
      </div>
      <Line :data="chartData" :options="chartOptions" />
    </div>

    <!-- Legend -->
    <div v-if="hasData && series.length > 1" class="mt-3 flex flex-wrap gap-3">
      <div
        v-for="(s, i) in series"
        :key="s.label"
        class="flex items-center gap-1.5 text-xs text-gray-400"
      >
        <span
          class="w-2.5 h-2.5 rounded-full"
          :style="{ backgroundColor: seriesColors[i] }"
        ></span>
        {{ s.label }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { Line } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler,
} from 'chart.js'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend, Filler)

interface SeriesData {
  label: string
  data: [number, number][]
}

const props = withDefaults(defineProps<{
  title: string
  unit?: string
  fetchData: (startUnix: number, endUnix: number, step: number) => Promise<SeriesData[]>
  formatValue?: (value: number) => string
  maxY?: number
  refreshIntervalMs?: number
}>(), {
  unit: '',
  refreshIntervalMs: 30000,
})

const palette = [
  '#3B82F6', '#10B981', '#F59E0B', '#EF4444', '#8B5CF6',
  '#EC4899', '#06B6D4', '#84CC16', '#F97316', '#6366F1',
]

const timeRanges = [
  { label: '1h', value: 3600 },
  { label: '6h', value: 21600 },
  { label: '24h', value: 86400 },
  { label: '7d', value: 604800 },
]

const selectedRange = ref(3600)
const series = ref<SeriesData[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
let refreshTimer: ReturnType<typeof setInterval> | null = null

const hasData = computed(() => series.value.some(s => s.data.length > 0))

const seriesColors = computed(() =>
  series.value.map((_, i) => palette[i % palette.length])
)

const chartData = computed(() => {
  if (!hasData.value) {
    return { labels: [], datasets: [] }
  }

  // Объединяем все timestamps и сортируем
  const allTimestamps = new Set<number>()
  series.value.forEach(s => s.data.forEach(([ts]) => allTimestamps.add(ts)))
  const timestamps = Array.from(allTimestamps).sort((a, b) => a - b)

  const labels = timestamps.map(ts => {
    const d = new Date(ts * 1000)
    if (selectedRange.value <= 86400) {
      return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
    }
    return d.toLocaleDateString([], { month: 'short', day: 'numeric' })
  })

  const datasets = series.value.map((s, i) => {
    const dataMap = new Map(s.data)
    return {
      label: s.label,
      data: timestamps.map(ts => dataMap.get(ts) ?? null),
      borderColor: seriesColors.value[i],
      backgroundColor: 'transparent',
      tension: 0.3,
      pointRadius: 0,
      pointHitRadius: 10,
      borderWidth: 2,
      spanGaps: true,
    }
  })

  return { labels, datasets }
})

const chartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  animation: { duration: 300 },
  interaction: {
    intersect: false,
    mode: 'index' as const,
  },
  plugins: {
    legend: { display: false },
    tooltip: {
      backgroundColor: '#1F2937',
      titleColor: '#F9FAFB',
      bodyColor: '#D1D5DB',
      borderColor: '#374151',
      borderWidth: 1,
      padding: 8,
      callbacks: {
        label: (context: any) => {
          const val = context.parsed.y
          if (val === null) return ''
          const formatted = props.formatValue ? props.formatValue(val) : val.toFixed(2)
          return `${context.dataset.label}: ${formatted} ${props.unit}`.trim()
        },
      },
    },
  },
  scales: {
    x: {
      grid: { color: '#1F2937', drawBorder: false },
      ticks: { color: '#6B7280', maxTicksLimit: 8, font: { size: 10 }, maxRotation: 0 },
    },
    y: {
      min: 0,
      max: props.maxY || undefined,
      grid: { color: '#1F2937', drawBorder: false },
      ticks: {
        color: '#6B7280',
        font: { size: 10 },
        maxTicksLimit: 5,
        callback: (value: number | string) => {
          const num = typeof value === 'string' ? parseFloat(value) : value
          if (props.formatValue) return props.formatValue(num)
          return num.toFixed(2)
        },
      },
    },
  },
}))

async function loadData() {
  try {
    loading.value = true
    error.value = null
    const endUnix = Math.floor(Date.now() / 1000)
    const startUnix = endUnix - selectedRange.value
    const step = Math.max(Math.floor(selectedRange.value / 200), 15)
    series.value = await props.fetchData(startUnix, endUnix, step)
  } catch (e: any) {
    error.value = e.message || String(e)
  } finally {
    loading.value = false
  }
}

watch(selectedRange, () => loadData())

onMounted(() => {
  loadData()
  refreshTimer = setInterval(loadData, props.refreshIntervalMs)
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})
</script>
