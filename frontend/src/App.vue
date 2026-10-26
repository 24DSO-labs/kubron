<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div id="app">
    <!-- Автовыход по бездействию: следит за вводом и предупреждает перед выходом -->
    <IdleGuard />

    <!-- Экран входа / первичной настройки / смены временного пароля -->
    <AuthScreen v-if="auth.screen.value !== 'app'" :mode="auth.screen.value" />

    <template v-else>
    <!-- Header -->
    <header class="header" :class="{ 'env-prod': activeClusterMeta?.environment === 'prod' }">
      <div class="header-brand">
        <img src="./assets/images/logo.svg" alt="Kubron Logo" class="logo" />
      </div>
      <div v-if="activeClusterId && activeClusterMeta" class="connection-info">
        <ClusterAvatar :name="activeClusterMeta.displayName" :icon="activeClusterMeta.icon"
          :color="activeClusterMeta.color" :size="26" />
        <span class="active-cluster-name">{{ activeClusterMeta.displayName }}</span>
        <span v-if="activeEnvironment" class="env-badge"
          :style="{ color: activeEnvironment.color, borderColor: activeEnvironment.color }">
          {{ activeEnvironment.label }}
        </span>
        <button class="btn-disconnect" title="Отключиться от кластера" aria-label="Отключиться от кластера"
          @click="clusterConn.disconnect()">✕</button>
      </div>
      <UserMenu />
    </header>

    <div class="app-body">
      <!-- Cluster Bar -->
      <ClusterBar :clusters="clusters" @select="clusterConn.selectCluster"
        @add="clusterConn.showAddCluster.value = true" @edit="clusterConn.openEditCluster"
        @delete="clusterConn.deleteCluster" />

      <!-- Нет кластеров -->
      <div v-if="clusters.length === 0 && !clusterConn.showAddCluster.value" class="welcome-screen">
        <h2>Добро пожаловать в Kubron!</h2>
        <template v-if="can(Perm.ClusterManage)">
          <p>Добавьте кластер для начала работы</p>
          <button class="btn" @click="clusterConn.showAddCluster.value = true">Добавить кластер</button>
        </template>
        <p v-else>Кластеры ещё не добавлены. Обратитесь к администратору.</p>
      </div>

      <!-- Не подключен, но кластеры есть -->
      <div v-else-if="!activeClusterId && clusters.length > 0 && !clusterConn.connecting.value" class="welcome-screen">
        <h2>Выберите кластер</h2>
        <p>Кликните на иконку кластера слева для подключения</p>
      </div>

      <!-- Основной интерфейс -->
      <template v-else-if="activeClusterId">
        <Sidebar :currentView="currentView" :selectedNamespace="selectedNamespace" :namespaces="namespaces"
          :clusterName="activeClusterMeta?.displayName" @navigate="onNavigate" @nsChange="onNsChange" />

        <main class="content">
          <!-- Ресурсы нового каркаса (src/resources): список рисует ResourceList -->
          <ResourceList v-if="activeResource" :key="`${activeClusterId}/${activeResource.view}`"
            :def="activeResource" />

          <!-- Остальные view (старый механизм) -->
          <template v-else>
            <div v-if="loading" class="loading">⏳ Загрузка...</div>
            <component v-else-if="activeViewComponent" :is="activeViewComponent" v-bind="activeViewProps"
              v-on="activeViewEvents" />
          </template>
        </main>
      </template>
    </div>

    <!-- ═══ Панели нового каркаса: одна панель деталей и один диалог удаления ═══ -->
    <ResourceHost />

    <!-- Вкладки: Logs / YAML / Terminal -->
    <TabbedPanel v-if="globalTabbedPanel.isOpen()" @close="globalTabbedPanel.closeAllTabs()"
      @apply-yaml="applyYAMLTabbed" />

    <AddClusterDialog v-if="clusterConn.showAddCluster.value" @close="clusterConn.showAddCluster.value = false"
      :pick-file="clusterConn.pickKubeconfigFile" :submit="clusterConn.addCluster" />

    <ClusterEditDialog v-if="clusterConn.editingCluster.value" :key="clusterConn.editingCluster.value.id"
      :cluster="clusterConn.editingCluster.value" :pick-file="clusterConn.pickKubeconfigFile"
      :save="clusterConn.saveCluster" @close="clusterConn.closeEditCluster" @delete="clusterConn.deleteFromEdit" />

    <NotificationToast :notifications="notifications.notifications.value" @dismiss="notifications.dismiss" />

    <!-- Connecting overlay -->
    <div v-if="clusterConn.connecting.value" class="overlay">
      <div class="spinner">⏳ Подключение...</div>
    </div>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, watch, provide } from 'vue'
import { useK8s } from './composables/useK8s'
import { useClusters } from './composables/useClusters'
import { useClusterConnection } from './composables/useClusterConnection'
import { useNotifications } from './composables/useNotifications'
import { useAppSettings } from './composables/useAppSettings'
import { viewRegistry } from './composables/useViewRegistry'
import { globalTabbedPanel } from './composables/useTabbedPanel'
import { getResourceByView, getResourceById } from './resources/registry'
import { resourceActions } from './resources/actions'
import { DEFAULT_NAMESPACE } from './resources/templates'
import { clusterNameKey } from './resources/context'
import { clearResourceStore, reloadActive } from './resources/store'
import { answerConfirm } from './resources/confirm'
import { can, useAuth } from './composables/useAuth'
import { Perm } from './types/auth'

import AuthScreen from './components/auth/AuthScreen.vue'
import UserMenu from './components/auth/UserMenu.vue'
import IdleGuard from './components/auth/IdleGuard.vue'
import ClusterBar from './components/ClusterBar.vue'
import AddClusterDialog from './components/AddClusterDialog.vue'
import ClusterEditDialog from './components/cluster/ClusterEditDialog.vue'
import ClusterAvatar from './components/cluster/ClusterAvatar.vue'
import { findEnvironment } from './components/cluster/clusterStyle'
import Sidebar from './components/Sidebar.vue'
import NotificationToast from './components/NotificationToast.vue'
import ResourceList from './components/resource/ResourceList.vue'
import ResourceHost from './components/resource/ResourceHost.vue'
import TabbedPanel from './components/utils/TabbedPanel.vue'
import { navigationRequest } from './resources/customResources'
import { clearHelmContentCache } from './resources/helmContent'

// ═══════════════════════════════════════════════
// Composables
// ═══════════════════════════════════════════════

const auth = useAuth()
const k8s = useK8s()
const clustersApi = useClusters()
const notifications = useNotifications()

const {
  namespaces, selectedNamespace, currentView, error, loading,
  switchView, onNamespaceChange, refreshData,
  loadNamespaces,
} = k8s

const {
  clusters, activeClusterId,
  loadClusters, connectToCluster, pickKubeconfigFile, addClusterFromFile, addClusterFromText,
  deleteCluster, updateCluster, autoConnect,
} = clustersApi

const clusterConn = useClusterConnection({
  clusters,
  activeClusterId,
  error,
  selectedNamespace,
  namespaces,
  currentView,
  connectToCluster,
  loadNamespaces,
  refreshData,
  k8sDisconnect: k8s.disconnect,
  loadClusters,
  pickKubeconfigFile,
  addClusterFromFile,
  addClusterFromText,
  deleteCluster,
  updateCluster,
  autoConnect,
})
const activeClusterMeta = clusterConn.activeClusterMeta
const activeEnvironment = computed(() => findEnvironment(activeClusterMeta.value?.environment ?? ''))

// ═══════════════════════════════════════════════
// Для вкладок и секций нового каркаса (метрики и т.п.)
// ═══════════════════════════════════════════════

provide(clusterNameKey, activeClusterId)

function closeAllDetailPanels() {
  resourceActions.closeDetail()
}

watch(activeClusterId, () => {
  clearResourceStore()
  clearHelmContentCache()
  answerConfirm(false)
  closeAllDetailPanels()
})

// ═══════════════════════════════════════════════
// Обновление данных
// ═══════════════════════════════════════════════

/** После изменений: обновить и старый стор (если view старый), и новый (если открыт ResourceList) */
const refreshAll = () => Promise.all([refreshData(), reloadActive()])

// ═══════════════════════════════════════════════
// Навигация и открытие ресурсов по ссылке
// ═══════════════════════════════════════════════

function onNavigate(view: string) {
  closeAllDetailPanels()
  switchView(view)
}

function onNsChange(ns: string) {
  selectedNamespace.value = ns
  onNamespaceChange()
}

/** Переход к другому ресурсу из панели (Pod -> ReplicaSet, RoleBinding -> Role, ...) */
function onOpenResource(data: { kind: string; name: string; namespace?: string }) {
  let { kind, name } = data
  const namespace = data.namespace ?? ''

  // Допускаем форму "ReplicaSet/my-name" в поле name
  const parts = name.split('/')
  if (parts.length === 2) [kind, name] = parts

  closeAllDetailPanels()

  // Любой тип из resources/registry.ts открывается без доп. кода
  const def = getResourceById(kind)
  if (def) return resourceActions.open(def.id, namespace, name)

  notifications.notifyError(`Unknown resource type: ${kind}`)
}

// ═══════════════════════════════════════════════
// Динамический view
// ═══════════════════════════════════════════════

const activeResource = computed(() => getResourceByView(currentView.value))

const activeViewConfig = computed(() => viewRegistry[currentView.value] || null)
const activeViewComponent = computed(() => activeViewConfig.value?.component || null)

const activeViewProps = computed(() => {
  const view = currentView.value
  const props: Record<string, any> = {}

  if (view === 'overview') {
    props.clusterName = activeClusterId.value
  }
  if (view === 'argo-dashboard') {
    props.clusterName = activeClusterId.value
  }
  if (view === 'settings') {
    props.clusterName = activeClusterId.value
  }
  return props
})

/** Действия view -> обработчики. Это единственное место, где view связываются с логикой */
const viewHandlers: Record<string, () => Record<string, (...args: any[]) => void>> = {
  overview: () => ({ navigate: onNavigate }),
  'argo-dashboard': () => ({ navigate: onNavigate }),
}

const activeViewEvents = computed(() => ({
  refresh: refreshData,
  ...viewHandlers[currentView.value]?.(),
}))

// ═══════════════════════════════════════════════
// YAML из TabbedPanel
// ═══════════════════════════════════════════════

async function applyYAMLTabbed(tabId: string) {
  if (!can(Perm.Write)) {
    notifications.notifyError('Недостаточно прав для изменения ресурсов')
    return
  }
  const tab = globalTabbedPanel.tabs.value.find(t => t.id === tabId)
  if (!tab || tab.type !== 'yaml' || tab.saving) return
  tab.saving = true
  const applied = tab.content
  try {
    // Кнопка «+»: вкладка с новым манифестом создаёт объект, а не заменяет существующий
    if (tab.mode === 'create') {
      const name = await k8s.createResourceYAML(applied, tab.resourceType, DEFAULT_NAMESPACE)
      notifications.notifySuccess(`${tab.kindLabel ?? tab.resourceType} "${name}" создан`)
      globalTabbedPanel.closeTab(tab.id)
      await refreshAll()
      return
    }
    await k8s.applyYAML(applied, tab.resourceType, tab.namespace, tab.resourceName)
    tab.originalContent = applied
    notifications.notifySuccess(`${tab.resourceType} ${tab.resourceName} updated`)
    await refreshAll()
  } catch (e: any) {
    notifications.notifyError(`${tab.mode === 'create' ? 'Ошибка создания' : 'Apply error'}: ${e?.message || e}`)
  } finally {
    tab.saving = false
  }
}

// ═══════════════════════════════════════════════
// Ошибки из composables, которые пишут в ref `error`
// ═══════════════════════════════════════════════

watch(error, (val) => {
  if (!val) return
  if (val.startsWith('Ошибка') || /error/i.test(val)) notifications.notifyError(val)
  else notifications.notifySuccess(val)
  error.value = ''
})

// ═══════════════════════════════════════════════
// Init
// ═══════════════════════════════════════════════

const appSettings = useAppSettings()

// Данные кластеров загружаем только после входа: до этого бэкенд отвечает «требуется вход»
let appStarted = false
watch(
  () => auth.screen.value,
  screen => {
    if (screen !== 'app' || appStarted) return
    appStarted = true
    appSettings.loadSettings()
    clusterConn.initialize()
  },
  { immediate: true },
)

watch(navigationRequest, view => {
  if (!view) return
  navigationRequest.value = null
  onNavigate(view)
})

onMounted(() => auth.init())
</script>

<style src="./assets/style.css"></style>
