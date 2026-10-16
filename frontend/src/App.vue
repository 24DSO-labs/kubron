<template>
  <div id="app">
    <!-- Header -->
    <header class="header">
      <div class="header-brand">
        <img src="./assets/images/logo.svg" alt="Kubron Logo" class="logo" />
      </div>
      <div v-if="activeClusterId" class="connection-info">
        <span class="active-cluster-icon">{{ activeClusterMeta?.icon }}</span>
        <span class="active-cluster-name" :style="{ color: activeClusterMeta?.color }">
          {{ activeClusterMeta?.displayName }}
        </span>
        <button class="btn-disconnect" @click="clusterConn.disconnect()">✕</button>
      </div>
    </header>

    <div class="app-body">
      <!-- Cluster Bar -->
      <ClusterBar :clusters="clusters" @select="clusterConn.selectCluster"
        @add="clusterConn.showAddCluster.value = true" @delete="clusterConn.deleteCluster"
        @update="clusterConn.updateCluster" />

      <!-- Нет кластеров -->
      <div v-if="clusters.length === 0 && !clusterConn.showAddCluster.value" class="welcome-screen">
        <h2>Добро пожаловать в Kubron!</h2>
        <p>Добавьте кластер для начала работы</p>
        <button class="btn" @click="clusterConn.showAddCluster.value = true">➕ Добавить кластер</button>
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
      @addFile="clusterConn.addFromFile" @addText="clusterConn.addFromText" />

    <NotificationToast :notifications="notifications.notifications.value" @dismiss="notifications.dismiss" />

    <!-- Connecting overlay -->
    <div v-if="clusterConn.connecting.value" class="overlay">
      <div class="spinner">⏳ Подключение...</div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, watch, provide } from 'vue'
import { useK8s } from './composables/useK8s'
import { useClusters } from './composables/useClusters'
import { useClusterConnection } from './composables/useClusterConnection'
import { useNotifications } from './composables/useNotifications'
import { viewRegistry } from './composables/useViewRegistry'
import { globalTabbedPanel } from './composables/useTabbedPanel'
import { getResourceByView, getResourceById } from './resources/registry'
import { resourceActions } from './resources/actions'
import { clusterNameKey } from './resources/context'
import { clearResourceStore, reloadActive } from './resources/store'

import ClusterBar from './components/ClusterBar.vue'
import AddClusterDialog from './components/AddClusterDialog.vue'
import Sidebar from './components/Sidebar.vue'
import NotificationToast from './components/NotificationToast.vue'
import ResourceList from './components/resource/ResourceList.vue'
import ResourceHost from './components/resource/ResourceHost.vue'
import TabbedPanel from './components/utils/TabbedPanel.vue'

// ═══════════════════════════════════════════════
// Composables
// ═══════════════════════════════════════════════

const k8s = useK8s()
const clustersApi = useClusters()
const notifications = useNotifications()

const {
  namespaces, overview, selectedNamespace, currentView, error, loading,
  pct,
  switchView, onNamespaceChange, refreshData,
  loadNamespaces,
} = k8s

const {
  clusters, activeClusterId,
  loadClusters, connectToCluster, addClusterFromFile, addClusterFromText,
  deleteCluster, updateCluster, autoConnect,
} = clustersApi

const clusterConn = useClusterConnection({
  clusters,
  activeClusterId,
  error,
  selectedNamespace,
  currentView,
  connectToCluster,
  loadNamespaces,
  refreshData,
  k8sDisconnect: k8s.disconnect,
  loadClusters,
  addClusterFromFile,
  addClusterFromText,
  deleteCluster,
  updateCluster,
  autoConnect,
})
const activeClusterMeta = clusterConn.activeClusterMeta

// ═══════════════════════════════════════════════
// Для вкладок и секций нового каркаса (метрики и т.п.)
// ═══════════════════════════════════════════════

provide(clusterNameKey, activeClusterId)

function closeAllDetailPanels() {
  resourceActions.closeDetail()
}

// Смена кластера: сбросить кэш списков нового каркаса и закрыть панели.
// ResourceList пересоздаётся сам (activeClusterId входит в его :key).
watch(activeClusterId, () => {
  clearResourceStore()
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
    props.overview = overview.value
    props.pct = pct
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
}

const activeViewEvents = computed(() => ({
  refresh: refreshData,
  ...viewHandlers[currentView.value]?.(),
}))

// ═══════════════════════════════════════════════
// YAML из TabbedPanel
// ═══════════════════════════════════════════════

async function applyYAMLTabbed(tabId: string) {
  const tab = globalTabbedPanel.tabs.value.find(t => t.id === tabId)
  if (!tab || tab.type !== 'yaml') return
  try {
    await k8s.applyYAML(tab.content, tab.resourceType, tab.namespace, tab.resourceName)
    notifications.notifySuccess(`${tab.resourceType} ${tab.resourceName} updated`)
    await refreshAll()
  } catch (e: any) {
    notifications.notifyError(`Apply error: ${e.message || e}`)
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

onMounted(() => {
  clusterConn.initialize()
})
</script>

<style src="./assets/style.css"></style>
