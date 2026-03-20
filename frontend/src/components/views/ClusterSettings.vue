<template>
  <div class="p-6 max-w-4xl">
    <h1 class="text-xl font-semibold text-white mb-6">Настройки кластера</h1>

    <div class="bg-gray-800 rounded-lg border border-gray-700">
      <!-- Header -->
      <div class="p-5 border-b border-gray-700">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-10 h-10 rounded-lg bg-purple-900/50 flex items-center justify-center">
              <svg class="w-5 h-5 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
              </svg>
            </div>
            <div>
              <h2 class="text-base font-medium text-white">Стек метрик</h2>
              <p class="text-sm text-gray-400">Prometheus + Node Exporter + Kube State Metrics</p>
            </div>
          </div>
          <MetricsStatusBadge v-if="status" :status="status.status" />
        </div>
      </div>

      <div class="p-5">
        <!-- NOT INSTALLED -->
        <div v-if="currentState === 'not_installed'">
          <p class="text-sm text-gray-400 mb-4">
            Установите стек метрик Kubron для включения мониторинга кластера с графиками CPU, памяти и сети. Создаёт пространство имён
            <code class="text-gray-300 bg-gray-700 px-1.5 py-0.5 rounded text-xs">kubron-metrics</code>.
          </p>

          <div class="bg-gray-900/50 rounded-lg p-4 mb-5">
            <h4 class="text-xs font-medium text-gray-500 uppercase tracking-wider mb-3">
              Компоненты
            </h4>
            <div class="space-y-2.5">
              <div v-for="c in componentsList" :key="c.name"
                class="flex items-center gap-2.5 text-sm">
                <span class="w-2 h-2 rounded-full" :class="c.dotColor"></span>
                <span class="text-gray-300 font-medium">{{ c.name }}</span>
                <span class="text-gray-600 text-xs">— {{ c.desc }}</span>
              </div>
            </div>
          </div>

          <button @click="handleInstall" :disabled="loading"
            class="px-6 py-2.5 bg-gradient-to-r from-blue-600 to-blue-700 hover:from-blue-500 hover:to-blue-600 active:from-blue-700 active:to-blue-800 disabled:from-blue-800 disabled:to-blue-800
                   disabled:cursor-not-allowed text-white text-sm font-semibold
                   rounded-lg transition-all duration-200 shadow-md hover:shadow-xl hover:scale-105 flex items-center gap-2">
            <svg v-if="loading" class="animate-spin h-3.5 w-3.5" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10"
                stroke="currentColor" stroke-width="3"/>
              <path class="opacity-75" fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
            </svg>
            {{ loading ? 'Установка...' : 'Установить стек метрик' }}
          </button>
        </div>

        <!-- INSTALLING -->
        <div v-else-if="currentState === 'installing'">
          <div class="flex items-center gap-3">
            <div class="animate-spin rounded-full h-5 w-5 border-b-2 border-blue-500"></div>
            <p class="text-sm text-gray-300">
              Установка стека метрик... Это может занять несколько минут.
            </p>
          </div>
        </div>

        <!-- INSTALLED -->
        <div v-else-if="currentState === 'installed'">
          <p class="text-sm text-gray-400 mb-4">
            Стек метрик работает. Данные мониторинга собираются.
          </p>

          <!-- Running components -->
          <div v-if="status?.components"
            class="bg-gray-900/50 rounded-lg p-4 mb-5">
            <h4 class="text-xs font-medium text-gray-500 uppercase tracking-wider mb-3">
              Работающие компоненты
            </h4>
            <div class="space-y-2.5">
              <div v-for="(comp, name) in status.components" :key="name"
                class="flex items-center justify-between text-sm">
                <div class="flex items-center gap-2.5">
                  <span :class="['w-2 h-2 rounded-full',
                    comp.ready ? 'bg-green-400' : 'bg-yellow-400 animate-pulse']">
                  </span>
                  <span class="text-gray-300">{{ fmtName(name as string) }}</span>
                </div>
                <span class="text-gray-500 text-xs">
                  {{ comp.availableReplicas }}/{{ comp.replicas }} готово
                </span>
              </div>
            </div>
          </div>

          <!-- Uninstall -->
          <div class="pt-4 border-t border-gray-700">
            <div class="flex items-center justify-between">
              <div>
                <p class="text-sm text-gray-300">Удалить стек метрик</p>
                <p class="text-xs text-gray-500 mt-0.5">
                  Удаляет пространство имён kubron-metrics и все данные
                </p>
              </div>
              <div v-if="!showConfirm">
                <button @click="showConfirm = true"
                  class="px-4 py-2 bg-red-500/20 hover:bg-red-500/30 text-red-400 hover:text-red-300
                         text-sm font-medium rounded-lg border border-red-500/50 hover:border-red-500 transition-all duration-200">
                  Удалить
                </button>
              </div>
              <div v-else class="flex items-center gap-2">
                <button @click="showConfirm = false"
                  class="px-4 py-2 bg-gray-700 hover:bg-gray-600 text-gray-300 hover:text-white
                         text-sm font-medium rounded-lg transition-all duration-200">
                  Отмена
                </button>
                <button @click="handleUninstall" :disabled="loading"
                  class="px-4 py-2 bg-gradient-to-r from-red-600 to-red-700 hover:from-red-500 hover:to-red-600 active:from-red-700 active:to-red-800 disabled:from-red-800 disabled:to-red-800
                         text-white text-sm font-semibold rounded-lg transition-all duration-200 shadow-md hover:shadow-lg disabled:cursor-not-allowed">
                  Подтвердить удаление
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- DEGRADED -->
        <div v-else-if="currentState === 'degraded'">
          <div class="bg-yellow-900/20 border border-yellow-700/50 rounded-lg p-4 mb-4">
            <div class="flex items-start gap-2">
              <svg class="w-5 h-5 text-yellow-400 flex-shrink-0 mt-0.5" fill="none"
                stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667
                     1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732
                     0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z"/>
              </svg>
              <div>
                <p class="text-sm text-yellow-300 font-medium">Стек метрик неработоспособен</p>
                <p class="text-xs text-yellow-400/70 mt-1">{{ status?.message }}</p>
              </div>
            </div>
          </div>

          <div v-if="status?.components" class="bg-gray-900/50 rounded-lg p-4 mb-4">
            <div class="space-y-2.5">
              <div v-for="(comp, name) in status.components" :key="name"
                class="flex items-center justify-between text-sm">
                <div class="flex items-center gap-2.5">
                  <span :class="['w-2 h-2 rounded-full',
                    comp.ready ? 'bg-green-400' : 'bg-red-400']"></span>
                  <span class="text-gray-300">{{ fmtName(name as string) }}</span>
                </div>
                <span :class="['text-xs', comp.ready ? 'text-gray-500' : 'text-red-400']">
                  {{ comp.availableReplicas }}/{{ comp.replicas }} готово
                </span>
              </div>
            </div>
          </div>

          <div class="flex items-center gap-2">
            <button @click="fetchStatus"
              class="px-4 py-2 bg-gray-700 hover:bg-gray-600 text-gray-300 hover:text-white
                     text-sm font-medium rounded-lg transition-all duration-200">
              Обновить
            </button>
            <button @click="handleReinstall" :disabled="loading"
              class="px-4 py-2 bg-gradient-to-r from-blue-600 to-blue-700 hover:from-blue-500 hover:to-blue-600 active:from-blue-700 active:to-blue-800 disabled:from-blue-800 disabled:to-blue-800
                     text-white text-sm font-semibold rounded-lg transition-all duration-200 shadow-md hover:shadow-lg disabled:cursor-not-allowed">
              Переустановить
            </button>
            <button @click="showConfirm = true"
              class="px-4 py-2 bg-red-500/20 hover:bg-red-500/30 text-red-400 hover:text-red-300
                     text-sm font-medium rounded-lg border border-red-500/50 hover:border-red-500 transition-all duration-200">
              Удалить
            </button>
          </div>

          <div v-if="showConfirm" class="mt-4 flex items-center gap-2 bg-red-500/10 border border-red-500/30 rounded-lg p-3">
            <span class="text-sm text-gray-300 font-medium">Вы уверены?</span>
            <button @click="showConfirm = false"
              class="ml-auto px-3 py-1.5 bg-gray-700 hover:bg-gray-600 text-gray-300 hover:text-white
                     text-sm font-medium rounded-lg transition-all duration-200">
              Отмена
            </button>
            <button @click="handleUninstall" :disabled="loading"
              class="px-3 py-1.5 bg-gradient-to-r from-red-600 to-red-700 hover:from-red-500 hover:to-red-600 active:from-red-700 active:to-red-800 disabled:from-red-800 disabled:to-red-800
                     text-white text-sm font-semibold rounded-lg transition-all duration-200 shadow-md hover:shadow-lg disabled:cursor-not-allowed">
              Подтвердить
            </button>
          </div>
        </div>

        <!-- Error -->
        <div v-if="error" class="mt-4 bg-red-900/20 border border-red-700/50 rounded-lg p-3">
          <p class="text-sm text-red-400">{{ error }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import MetricsStatusBadge from '../metrics/MetricsStatusBadge.vue'
import { useMetricsStatus } from '../../composables/useMetrics'

const props = defineProps<{
  clusterName: string
}>()

const {
  status,
  loading,
  error,
  fetchStatus,
  install,
  uninstall,
} = useMetricsStatus(props.clusterName)

const showConfirm = ref(false)

const currentState = computed(() => {
  if (!status.value) return 'not_installed'
  return status.value.status
})

const componentsList = [
  { name: 'Prometheus', desc: 'сбор и хранение метрик', dotColor: 'bg-orange-400' },
  { name: 'Node Exporter', desc: 'метрики оборудования и ОС', dotColor: 'bg-green-400' },
  { name: 'Kube State Metrics', desc: 'метрики объектов Kubernetes', dotColor: 'bg-blue-400' },
]

function fmtName(raw: string): string {
  return raw
    .replace(/-/g, ' ')
    .replace(/\b\w/g, c => c.toUpperCase())
}

async function handleInstall() {
  await install()
}

async function handleUninstall() {
  showConfirm.value = false
  await uninstall()
}

async function handleReinstall() {
  await uninstall()
  await install()
}

onMounted(() => {
  fetchStatus()
})
</script>
