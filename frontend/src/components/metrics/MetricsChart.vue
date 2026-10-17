<template>
  <div class="chart-card">
    <div class="chart-card-header">
      <h3 class="chart-card-title">{{ title }}</h3>
      <div class="chart-range-buttons">
        <button v-for="range in timeRanges" :key="range.value" @click="selectedRange = range.value" :class="[
          'chart-range-btn',
          selectedRange === range.value ? 'active' : 'inactive'
        ]">
          {{ range.label }}
        </button>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading && !dataPoints.length" class="chart-loading">
      <div class="chart-spinner"></div>
    </div>

    <!-- Error -->
    <div v-else-if="error && !dataPoints.length" class="chart-error">
      <div class="chart-error-content">
        <svg class="chart-error-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M12 9v2m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <p class="chart-error-text">{{ error }}</p>
        <button @click="loadData" class="chart-retry-btn">
          Retry
        </button>
      </div>
    </div>

    <!-- No data -->
    <div v-else-if="!dataPoints.length" class="chart-no-data">
      <p class="chart-no-data-text">No data available</p>
    </div>

    <!-- Chart -->
    <div v-else class="chart-canvas-wrapper">
      <div v-if="loading" class="chart-canvas-loading">
        <div class="chart-spinner small"></div>
      </div>
      <Line :data="chartData" :options="chartOptions" />
    </div>

    <!-- Current value -->
    <div v-if="currentValue !== null" class="chart-current-value-row">
      <span class="chart-current-label">Current</span>
      <div>
        <span class="chart-current-value">{{ formattedCurrentValue }}</span>
        <span v-if="unit" class="chart-current-unit">{{ unit }}</span>
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
          return new Date(ts * 1000).toLocaleString()
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
  const d = new Date(ts * 1000)
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
