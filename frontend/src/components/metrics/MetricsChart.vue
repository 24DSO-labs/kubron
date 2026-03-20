<template>
  <div class="bg-gray-800 rounded-lg border border-gray-700 p-4">
    <div class="flex items-center justify-between mb-3">
      <h3 class="text-sm font-medium text-gray-300">{{ title }}</h3>
      <div class="flex items-center gap-1">
        <button v-for="range in timeRanges" :key="range.value" @click="selectedRange = range.value" :class="[
          'px-2 py-0.5 text-xs rounded transition-colors',
          selectedRange === range.value
            ? 'bg-blue-600 text-white'
            : 'bg-gray-700 text-gray-400 hover:bg-gray-600'
        ]">
          {{ range.label }}
        </button>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading && !dataPoints.length" class="flex items-center justify-center h-48">
      <div class="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-500"></div>
    </div>

    <!-- Error -->
    <div v-else-if="error && !dataPoints.length" class="flex items-center justify-center h-48">
      <div class="text-center">
        <svg class="w-8 h-8 text-red-400 mx-auto mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M12 9v2m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <p class="text-red-400 text-sm">{{ error }}</p>
        <button @click="loadData" class="mt-2 text-xs text-blue-400 hover:text-blue-300">
          Retry
        </button>
      </div>
    </div>

    <!-- No data -->
    <div v-else-if="!dataPoints.length" class="flex items-center justify-center h-48">
      <p class="text-gray-500 text-sm">No data available</p>
    </div>

    <!-- Chart -->
    <div v-else class="h-44 relative">
      <div v-if="loading" class="absolute top-0 right-0 z-10">
        <div class="animate-spin rounded-full h-3 w-3 border-b-2 border-blue-500"></div>
      </div>
      <Line :data="chartData" :options="chartOptions" />
    </div>

    <!-- Current value -->
    <div v-if="currentValue !== null" class="mt-2 flex items-baseline justify-between">
      <span class="text-xs text-gray-500">Current</span>
      <div>
        <span class="text-xl font-bold text-white">{{ formattedCurrentValue }}</span>
        <span v-if="unit" class="text-xs text-gray-400 ml-1">{{ unit }}</span>
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
  Filler,
} from 'chart.js'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Filler)

const props = withDefaults(defineProps<{
  title: string
  unit?: string
  color?: string
  fetchData: (startUnix: number, endUnix: number, step: number) => Promise<[number, number][]>
  formatValue?: (value: number) => string
  maxY?: number
  refreshIntervalMs?: number
}>(), {
  unit: '',
  color: '#3B82F6',
  refreshIntervalMs: 30000,
})

const timeRanges = [
  { label: '1h', value: 3600 },
  { label: '6h', value: 21600 },
  { label: '24h', value: 86400 },
  { label: '7d', value: 604800 },
]

const selectedRange = ref(3600)
const dataPoints = ref<[number, number][]>([])
const loading = ref(false)
const error = ref<string | null>(null)
let refreshTimer: ReturnType<typeof setInterval> | null = null

const currentValue = computed(() => {
  if (!dataPoints.value.length) return null
  return dataPoints.value[dataPoints.value.length - 1][1]
})

const formattedCurrentValue = computed(() => {
  if (currentValue.value === null) return '—'
  if (props.formatValue) return props.formatValue(currentValue.value)
  return currentValue.value.toFixed(2)
})

const chartData = computed(() => {
  const points = dataPoints.value
  return {
    labels: points.map(([ts]) => formatTimestamp(ts)),
    datasets: [
      {
        data: points.map(([, v]) => v),
        borderColor: props.color,
        backgroundColor: hexToRgba(props.color, 0.1),
        fill: true,
        tension: 0.3,
        pointRadius: 0,
        pointHitRadius: 10,
        borderWidth: 2,
      },
    ],
  }
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
      displayColors: false,
      callbacks: {
        title: (items: any[]) => {
          if (!items.length) return ''
          const idx = items[0].dataIndex
          const ts = dataPoints.value[idx]?.[0]
          if (!ts) return ''
          return new Date(ts).toLocaleString()
        },
        label: (context: any) => {
          const val = context.parsed.y
          const formatted = props.formatValue ? props.formatValue(val) : val.toFixed(2)
          return `${formatted} ${props.unit}`.trim()
        },
      },
    },
  },
  scales: {
    x: {
      grid: { color: '#1F2937', drawBorder: false },
      ticks: {
        color: '#6B7280',
        maxTicksLimit: 8,
        font: { size: 10 },
        maxRotation: 0,
      },
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
          return num
        },
      },
    },
  },
}))

function formatTimestamp(ts: number): string {
  const d = new Date(ts)
  if (selectedRange.value <= 86400) {
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }
  return d.toLocaleDateString([], { month: 'short', day: 'numeric' })
}

function hexToRgba(hex: string, alpha: number): string {
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)
  return `rgba(${r}, ${g}, ${b}, ${alpha})`
}

async function loadData() {
  try {
    loading.value = true
    error.value = null
    const endUnix = Math.floor(Date.now() / 1000)
    const startUnix = endUnix - selectedRange.value
    // ~200 точек на график
    const step = Math.max(Math.floor(selectedRange.value / 200), 15)
    dataPoints.value = await props.fetchData(startUnix, endUnix, step)
  } catch (e: any) {
    error.value = e.message || String(e)
  } finally {
    loading.value = false
  }
}

function startRefresh() {
  stopRefresh()
  refreshTimer = setInterval(loadData, props.refreshIntervalMs)
}

function stopRefresh() {
  if (refreshTimer) {
    clearInterval(refreshTimer)
    refreshTimer = null
  }
}

watch(selectedRange, () => {
  loadData()
})

onMounted(() => {
  loadData()
  startRefresh()
})

onUnmounted(() => {
  stopRefresh()
})
</script>
