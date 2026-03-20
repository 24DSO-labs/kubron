<template>
  <div class="bg-gradient-to-br from-gray-900 to-gray-950 rounded-lg shadow-lg p-6 border border-gray-700 h-full">
    <h3 class="text-lg font-semibold text-white mb-4">{{ title }}</h3>
    
    <!-- Donut Chart -->
    <div class="flex items-center justify-center mb-6">
      <div class="relative w-32 h-32">
        <canvas ref="chartCanvas" class="w-full h-full"></canvas>
        <div class="absolute inset-0 flex flex-col items-center justify-center">
          <div class="text-2xl font-bold text-white">{{ totalUsagePercent }}%</div>
          <div class="text-xs text-gray-400">used</div>
        </div>
      </div>
    </div>

    <!-- Legend -->
    <div class="space-y-2 text-xs">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <div class="w-2 h-2 rounded-full bg-red-500"></div>
          <span class="text-gray-300">Usage: {{ usage.toFixed(2) }}</span>
        </div>
      </div>
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <div class="w-2 h-2 rounded-full bg-orange-500"></div>
          <span class="text-gray-300">Requests: {{ requests.toFixed(2) }}</span>
        </div>
      </div>
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <div class="w-2 h-2 rounded-full bg-yellow-500"></div>
          <span class="text-gray-300">Limits: {{ limits.toFixed(2) }}</span>
        </div>
      </div>
      <div class="flex items-center justify-between pt-2 border-t border-gray-700">
        <div class="flex items-center gap-2">
          <div class="w-2 h-2 rounded-full bg-green-500"></div>
          <span class="text-gray-300">Allocatable: {{ allocatable.toFixed(2) }}</span>
        </div>
      </div>
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <div class="w-2 h-2 rounded-full bg-blue-500"></div>
          <span class="text-gray-300">Capacity: {{ capacity.toFixed(2) }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { Chart, DoughnutController, ArcElement, Tooltip, Legend, ChartOptions } from 'chart.js'

Chart.register(DoughnutController, ArcElement, Tooltip, Legend)

interface Props {
  title: string
  usage: number
  requests: number
  limits: number
  allocatable: number
  capacity: number
  unit?: string
}

const props = withDefaults(defineProps<Props>(), {
  unit: 'GB',
})

const chartCanvas = ref<HTMLCanvasElement | null>(null)
let chart: Chart | null = null

const usagePercent = computed(() => {
  if (props.capacity === 0) return 0
  return Math.min(100, (props.usage / props.capacity) * 100)
})

const requestsPercent = computed(() => {
  if (props.capacity === 0) return 0
  return Math.min(100 - usagePercent.value, (props.requests / props.capacity) * 100)
})

const limitsPercent = computed(() => {
  if (props.capacity === 0) return 0
  return Math.min(100 - usagePercent.value - requestsPercent.value, (props.limits / props.capacity) * 100)
})

const allocatablePercent = computed(() => {
  if (props.capacity === 0) return 0
  const used = usagePercent.value + requestsPercent.value + limitsPercent.value
  return Math.min(100 - used, (props.allocatable / props.capacity) * 100)
})

const capacityPercent = computed(() => {
  const used = usagePercent.value + requestsPercent.value + limitsPercent.value + allocatablePercent.value
  return Math.max(0, 100 - used)
})

const totalUsagePercent = computed(() => {
  if (props.capacity === 0) return 0
  return Math.round((props.usage / props.capacity) * 100)
})

const updateChart = () => {
  if (!chartCanvas.value) return

  const ctx = chartCanvas.value.getContext('2d')
  if (!ctx) return

  if (chart) {
    chart.destroy()
  }

  const options: ChartOptions<'doughnut'> = {
    responsive: true,
    maintainAspectRatio: true,
    plugins: {
      legend: {
        display: false,
      },
      tooltip: {
        enabled: false,
      },
    },
  }

  chart = new Chart(ctx, {
    type: 'doughnut',
    data: {
      labels: ['Usage', 'Requests', 'Limits', 'Allocatable', 'Free'],
      datasets: [
        {
          data: [
            usagePercent.value,
            requestsPercent.value,
            limitsPercent.value,
            allocatablePercent.value,
            capacityPercent.value,
          ],
          backgroundColor: ['#ef4444', '#f97316', '#eab308', '#22c55e', '#1f2937'],
          borderColor: ['#dc2626', '#ea580c', '#ca8a04', '#16a34a', '#111827'],
          borderWidth: 2,
        },
      ],
    },
    options,
  })
}

onMounted(() => {
  updateChart()
})
</script>

<style scoped>
</style>
