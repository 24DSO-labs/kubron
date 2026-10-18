<template>
  <div class="settings-page">
    <h1 class="settings-title">Настройки кластера</h1>

    <!-- App Settings -->
    <div class="settings-card">
      <div class="settings-card-header">
        <div class="settings-card-header-row">
          <div class="settings-card-header-left">
            <div class="settings-card-icon-wrap blue">
              <svg class="settings-card-icon blue" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
            </div>
            <div>
              <h2 class="settings-card-heading">Авто-обновление</h2>
              <p class="settings-card-description">Периодическое обновление списков ресурсов</p>
            </div>
          </div>
        </div>
      </div>

      <div class="settings-card-body">
        <div class="settings-setting-row">
          <div class="settings-setting-field">
            <label class="settings-label">
              Интервал обновления
            </label>
            <div class="settings-setting-row-input">
              <input
                v-model.number="refreshInterval"
                type="number"
                min="0"
                max="300"
                class="settings-input"
                placeholder="10"
              />
              <span class="settings-input-placeholder">секунд</span>
            </div>
            <p class="settings-hint">
              0 — авто-обновление отключено. Списки обновляются только вручную.
            </p>
          </div>
          <button
            @click="saveRefreshSettings"
            :disabled="!settingsLoaded"
            class="settings-save-btn"
          >
            Сохранить
          </button>
        </div>
      </div>
    </div>

    <!-- Metrics Stack -->
    <div class="settings-card">
      <!-- Header -->
      <div class="settings-card-header">
        <div class="settings-card-header-row">
          <div class="settings-card-header-left">
            <div class="settings-card-icon-wrap purple">
              <svg class="settings-card-icon purple" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
              </svg>
            </div>
            <div>
              <h2 class="settings-card-heading">Стек метрик</h2>
              <p class="settings-card-description">Prometheus + Node Exporter + Kube State Metrics</p>
            </div>
          </div>
          <MetricsStatusBadge v-if="status" :status="status.status" />
        </div>
      </div>

      <div class="settings-card-body">
        <!-- NOT INSTALLED -->
        <div v-if="currentState === 'not_installed'">
          <p class="settings-not-installed-text">
            Установите стек метрик Kubron для включения мониторинга кластера с графиками CPU, памяти и сети. Создаёт пространство имён
            <code class="settings-code-inline">kubron-metrics</code>.
          </p>

          <div class="settings-components-list">
            <h4 class="settings-components-title">
              Компоненты
            </h4>
            <div class="settings-components-list-inner">
              <div v-for="c in componentsList" :key="c.name"
                class="settings-component-item">
                <span class="w-2 h-2 rounded-full" :class="c.dotColor"></span>
                <span class="text-gray-300 font-medium">{{ c.name }}</span>
                <span class="text-gray-600 text-xs">— {{ c.desc }}</span>
              </div>
            </div>
          </div>

          <button v-if="canModify" @click="handleInstall" :disabled="loading"
            class="settings-install-btn">
            <svg v-if="loading" class="animate-spin h-3.5 w-3.5" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10"
                stroke="currentColor" stroke-width="3"/>
              <path class="opacity-75" fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"/>
            </svg>
            {{ loading ? 'Установка...' : 'Установить стек метрик' }}
          </button>
          <p v-else class="settings-not-installed-text">Установка стека метрик доступна администратору.</p>
        </div>

        <!-- INSTALLING -->
        <div v-else-if="currentState === 'installing'">
          <div class="settings-installing-status">
            <div class="settings-installing-spinner"></div>
            <p class="settings-installing-text">
              Установка стека метрик... Это может занять несколько минут.
            </p>
          </div>
        </div>

        <!-- INSTALLED -->
        <div v-else-if="currentState === 'installed'">
          <p class="settings-installed-status-text">
            Стек метрик работает. Данные мониторинга собираются.
          </p>

          <!-- Running components -->
          <div v-if="status?.components"
            class="settings-installed-components">
            <h4 class="settings-installed-components-title">
              Работающие компоненты
            </h4>
            <div class="settings-installed-components-inner">
              <div v-for="(comp, name) in status.components" :key="name"
                class="settings-installed-comp-item">
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
          <div v-if="canModify" class="settings-uninstall-section">
            <div class="settings-uninstall-row">
              <div>
                <p class="text-sm text-gray-300">Удалить стек метрик</p>
                <p class="text-xs text-gray-500 mt-0.5">
                  Удаляет пространство имён kubron-metrics и все данные
                </p>
              </div>
              <div v-if="!showConfirm">
                <button @click="showConfirm = true"
                  class="settings-delete-btn">
                  Удалить
                </button>
              </div>
              <div v-else class="settings-confirm-bar">
                <span class="settings-confirm-text">Вы уверены?</span>
                <button @click="showConfirm = false"
                  class="settings-cancel-btn">
                  Отмена
                </button>
                <button @click="handleUninstall" :disabled="loading"
                  class="settings-confirm-btn">
                  Подтвердить удаление
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- DEGRADED -->
        <div v-else-if="currentState === 'degraded'">
          <div class="settings-degraded-warning">
            <div class="settings-degraded-warning-content">
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

          <div v-if="status?.components" class="settings-degraded-components">
            <div class="settings-degraded-components-inner">
              <div v-for="(comp, name) in status.components" :key="name"
                class="settings-degraded-comp-item">
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
              class="settings-refresh-btn">
              Обновить
            </button>
            <button v-if="canModify" @click="handleReinstall" :disabled="loading"
              class="settings-reinstall-btn">
              Переустановить
            </button>
            <button v-if="canModify" @click="showConfirm = true"
              class="settings-delete-btn">
              Удалить
            </button>
          </div>

          <div v-if="showConfirm" class="settings-confirm-bar">
            <span class="settings-confirm-text">Вы уверены?</span>
            <button @click="showConfirm = false"
              class="settings-cancel-btn">
              Отмена
            </button>
            <button @click="handleUninstall" :disabled="loading"
              class="settings-confirm-btn">
              Подтвердить
            </button>
          </div>
        </div>

        <!-- Error -->
        <div v-if="error" class="settings-error-box">
          <p class="settings-error-text">{{ error }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import MetricsStatusBadge from '../metrics/MetricsStatusBadge.vue'
import { useMetricsStatus } from '../../composables/useMetrics'
import { useAppSettings } from '../../composables/useAppSettings'
import { can } from '../../composables/useAuth'
import { Perm } from '../../types/auth'

const props = defineProps<{
  clusterName: string
}>()

// Установка и удаление стека метрик изменяют кластер
const canModify = computed(() => can(Perm.Write))

const {
  status,
  loading,
  error,
  fetchStatus,
  install,
  uninstall,
} = useMetricsStatus(props.clusterName)

const appSettings = useAppSettings()
const refreshInterval = ref(10)
const settingsLoaded = ref(false)

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

async function saveRefreshSettings() {
  await appSettings.saveSettings({
    autoRefreshInterval: refreshInterval.value,
  })
}

onMounted(async () => {
  fetchStatus()
  await appSettings.loadSettings()
  refreshInterval.value = appSettings.settings.value.autoRefreshInterval || 10
  settingsLoaded.value = true
})
</script>
