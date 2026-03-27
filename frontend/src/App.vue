<template>
  <div id="app">
    <!-- Header -->
    <header class="header">
      <div class="header-brand">
        <img src="./assets/images/logo.svg" alt="Kubron Logo" class="logo" />
        <h1></h1>
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
      <ClusterBar
        :clusters="clusters"
        @select="clusterConn.selectCluster"
        @add="clusterConn.showAddCluster.value = true"
        @delete="clusterConn.deleteCluster"
        @update="clusterConn.updateCluster"
      />

      <!-- Нет кластеров -->
      <div v-if="clusters.length === 0 && !clusterConn.showAddCluster.value" class="welcome-screen">
        <h2>Добро пожаловать в Kubron!</h2>
        <p>Добавьте кластер для начала работы</p>
        <button class="btn" @click="clusterConn.showAddCluster.value = true">➕ Добавить кластер</button>
      </div>

      <!-- Не подключен но кластеры есть -->
      <div v-else-if="!activeClusterId && clusters.length > 0 && !clusterConn.connecting.value" class="welcome-screen">
        <h2>Выберите кластер</h2>
        <p>Кликните на иконку кластера слева для подключения</p>
      </div>

      <!-- Основной интерфейс -->
      <template v-else-if="activeClusterId">
        <Sidebar
          :currentView="currentView"
          :selectedNamespace="selectedNamespace"
          :namespaces="namespaces"
          :clusterName="activeClusterMeta?.displayName"
          @navigate="switchView"
          @nsChange="onNsChange"
        />

        <main class="content">
          <div v-if="loading" class="loading">⏳ Загрузка...</div>

          <component
            v-if="!loading && activeViewComponent"
            :is="activeViewComponent"
            v-bind="activeViewProps"
            v-on="activeViewEvents"
          />
        </main>
      </template>
    </div>

    <!-- Pod Detail Panel -->
    <PodDetailPanel
      v-if="selectedPod"
      :pod="selectedPod"
      :logs="podLogs"
      v-model:logContainer="logContainer"
      v-model:logLines="logLines"
      v-model:execCmd="execCmd"
      :execOutput="execOutput"
      :statusClass="statusClass"
      v-model:pfLocalPort="pfLocalPort"
      v-model:pfPodPort="pfPodPort"
      :portForwards="portForwards || []"
      :cluster-name="activeClusterId"
      @close="selectedPod = null"
      @loadLogs="loadLogs"
      @runExec="runExec"
      @delete="podActions.removePod()"
      @startPF="startPortForward"
      @stopPF="stopPortForward"
      @openTerminal="podActions.terminalOpen.value = true"
      @attach="podActions.openAttachWithContainer"
      @terminal="podActions.openShellWithContainer"
      @evict="podActions.evictPod()"
      @logs="(container: string) => podActions.openLogs(undefined, container)"
      @edit="podActions.editPod()"
    />

    <!-- Service Detail Panel -->
    <ServiceDetailPanel
      v-if="selectedService"
      :svc="selectedService"
      :portForwards="portForwards"
      @close="selectedService = null"
      @forward="forwardServicePort"
      @stopForward="stopPortForward"
    />

    <!-- ConfigMap Panel -->
    <ConfigMapPanel
      v-if="selectedConfigMap"
      :cm="selectedConfigMap"
      @close="selectedConfigMap = null"
    />

    <!-- Revisions Panel -->
    <RevisionsPanel
      v-if="revisions.length > 0"
      :revisions="revisions"
      @close="revisions = []"
      @rollback="onRollback"
    />

    <!-- Namespace Detail Panel -->
    <NamespaceDetailPanel
      v-if="selectedNamespaceDetail"
      :resource="selectedNamespaceDetail"
      @close="selectedNamespaceDetail = null"
    />

    <!-- Terminal Panel -->
    <TerminalPanel
      v-if="podActions.terminalOpen.value && terminalTargetPod"
      :namespace="terminalTargetPod.namespace"
      :pod="terminalTargetPod.name"
      :containers="terminalTargetPod.containers?.map((c: any) => c.name) || []"
      :initialContainer="logContainer"
      :mode="podActions.terminalMode.value"
      @close="podActions.closeTerminal()"
    />

    <!-- Logs Panel -->
    <LogsPanel
      v-if="podActions.logsOpen.value && logsTargetPod"
      :namespace="logsTargetPod.namespace"
      :pod="logsTargetPod.name"
      :containers="logsTargetPod.containers?.map((c: any) => c.name) || []"
      :initialContainer="logContainer"
      @close="podActions.closeLogs()"
    />

    <!-- YAML Editor Panel -->
    <TabbedYamlEditorPanel
      v-if="podActions.yamlEditorOpen.value"
      :content="yamlContent"
      :resourceType="podActions.yamlEditorMeta.value.resourceType"
      :resourceName="podActions.yamlEditorMeta.value.resourceName"
      :namespace="podActions.yamlEditorMeta.value.namespace"
      @update:content="(v: string) => yamlContent = v"
      @apply="applyYAML"
      @cancel="podActions.yamlEditorOpen.value = false"
      @close="podActions.yamlEditorOpen.value = false"
    />

    <!-- Add Cluster Dialog -->
    <AddClusterDialog
      v-if="clusterConn.showAddCluster.value"
      @close="clusterConn.showAddCluster.value = false"
      @addFile="clusterConn.addFromFile"
      @addText="clusterConn.addFromText"
    />

    <!-- Notifications -->
    <NotificationToast
      :notifications="notifications.notifications.value"
      @dismiss="notifications.dismiss"
    />

    <!-- Connecting overlay -->
    <div v-if="clusterConn.connecting.value" class="overlay">
      <div class="spinner">⏳ Подключение...</div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted } from 'vue'
import { useK8s } from './composables/useK8s'
import { useClusters } from './composables/useClusters'
import { usePodActions } from './composables/usePodActions'
import { useClusterConnection } from './composables/useClusterConnection'
import { useNotifications } from './composables/useNotifications'
import { viewRegistry } from './composables/useViewRegistry'

import ClusterBar from './components/ClusterBar.vue'
import AddClusterDialog from './components/AddClusterDialog.vue'
import Sidebar from './components/Sidebar.vue'
import NotificationToast from './components/NotificationToast.vue'

// Ленивая загрузка панелей
import { defineAsyncComponent } from 'vue'
const PodDetailPanel = defineAsyncComponent(() => import('./components/panels/PodDetailPanel.vue'))
const ServiceDetailPanel = defineAsyncComponent(() => import('./components/panels/ServiceDetailPanel.vue'))
const ConfigMapPanel = defineAsyncComponent(() => import('./components/panels/ConfigMapPanel.vue'))
const TabbedYamlEditorPanel = defineAsyncComponent(() => import('./components/panels/TabbedYamlEditorPanel.vue'))
const RevisionsPanel = defineAsyncComponent(() => import('./components/panels/RevisionsPanel.vue'))
const TerminalPanel = defineAsyncComponent(() => import('./components/panels/TerminalPanel.vue'))
const LogsPanel = defineAsyncComponent(() => import('./components/panels/LogsPanel.vue'))
const NamespaceDetailPanel = defineAsyncComponent(() => import('./components/panels/NamespaceDetailPanel.vue'))

// ─── Composables ───

const k8s = useK8s()
const clustersApi = useClusters()
const notifications = useNotifications()

const {
  namespaces, overview, selectedNamespace,
  currentView, search, error, loading,
  selectedPod, podLogs, logContainer, logLines,
  execCmd, execOutput, selectedConfigMap,
  yamlContent, revisions, portForwards,
  pfLocalPort, pfPodPort, selectedService,
  podMetrics, nodeMetrics, pvs, nodes, events,
  statusClass, pct,
  switchView, onNamespaceChange, refreshData,
  openPodDetail, loadLogs, runExec,
  doScale, restartDeployment, loadRevisions, rollbackDeployment,
  openConfigMapDetail, showYAML, applyYAML,
  startPortForward, stopPortForward, refreshPortForwards,
  openServiceDetail, forwardServicePort,
  loadNamespaces, deletePodByName,
  filteredPods, filteredDeployments, filteredStatefulSets,
  filteredDaemonSets, filteredJobs, filteredCronJobs,
  filteredServices, filteredIngresses, filteredConfigMaps,
  filteredSecrets, filteredResourceQuotas, filteredPVCs,
  filteredReplicaSets, filteredLimitRanges, filteredHPA,
  filteredPDB, filteredPriorityClasses, filteredRuntimeClasses,
  filteredLeases, filteredEndpoints, filteredNetworkPolicies,
  filteredIngressClasses, filteredStorageClasses,
  filteredServiceAccounts, filteredRoles, filteredRoleBindings,
  filteredClusterRoles, filteredClusterRoleBindings,
} = k8s

const {
  clusters,
  activeClusterId,
  loadClusters,
  connectToCluster,
  addClusterFromFile,
  addClusterFromText,
  deleteCluster,
  updateCluster,
  autoConnect,
} = clustersApi

// ─── Cluster Connection ───

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

// ─── Pod Actions ───

const podActions = usePodActions({
  selectedPod,
  logContainer,
  error,
  refreshData,
  deletePodByName,
  showYAML,
})

// ─── Local State ───

const selectedNamespaceDetail = ref<any | null>(null)
const revisionDepNs = ref({ ns: '', name: '' })

// ─── Computed: resolve target pods for terminal/logs ───

const terminalTargetPod = computed(() =>
  podActions.terminalPod.value || selectedPod.value
)

const logsTargetPod = computed(() =>
  podActions.logsPod.value || selectedPod.value
)

// ─── Data map for dynamic component ───

const dataMap = computed<Record<string, any>>(() => ({
  filteredPods: filteredPods.value,
  filteredDeployments: filteredDeployments.value,
  filteredStatefulSets: filteredStatefulSets.value,
  filteredDaemonSets: filteredDaemonSets.value,
  filteredJobs: filteredJobs.value,
  filteredCronJobs: filteredCronJobs.value,
  filteredReplicaSets: filteredReplicaSets.value,
  filteredServices: filteredServices.value,
  filteredIngresses: filteredIngresses.value,
  filteredEndpoints: filteredEndpoints.value,
  filteredIngressClasses: filteredIngressClasses.value,
  filteredNetworkPolicies: filteredNetworkPolicies.value,
  filteredPVCs: filteredPVCs.value,
  filteredStorageClasses: filteredStorageClasses.value,
  filteredConfigMaps: filteredConfigMaps.value,
  filteredSecrets: filteredSecrets.value,
  filteredResourceQuotas: filteredResourceQuotas.value,
  filteredLimitRanges: filteredLimitRanges.value,
  filteredHPA: filteredHPA.value,
  filteredPDB: filteredPDB.value,
  filteredPriorityClasses: filteredPriorityClasses.value,
  filteredRuntimeClasses: filteredRuntimeClasses.value,
  filteredLeases: filteredLeases.value,
  filteredServiceAccounts: filteredServiceAccounts.value,
  filteredRoles: filteredRoles.value,
  filteredRoleBindings: filteredRoleBindings.value,
  filteredClusterRoles: filteredClusterRoles.value,
  filteredClusterRoleBindings: filteredClusterRoleBindings.value,
  portForwards: portForwards.value,
  pvs: pvs.value,
  nodes: nodes.value,
  events: events.value,
  namespaces: namespaces.value,
}))

// ─── Dynamic View Component ───

const activeViewConfig = computed(() => viewRegistry[currentView.value] || null)

const activeViewComponent = computed(() => activeViewConfig.value?.component || null)

const activeViewProps = computed(() => {
  const config = activeViewConfig.value
  if (!config) return {}

  const props: Record<string, any> = {}

  // items
  if (config.dataKey) {
    props.items = dataMap.value[config.dataKey] || []
  }

  // search
  if (config.hasSearch) {
    props.search = search.value
  }

  // View-specific props
  const view = currentView.value

  if (view === 'overview') {
  props.overview = overview.value
  props.pct = pct               // ← передаём саму функцию
  props.clusterName = activeClusterId.value
}

  if (view === 'pods') {
    props.statusClass = statusClass
    props.podMetrics = podMetrics.value
  }

  if (view === 'jobs') {
    props.statusClass = statusClass
  }

  if (view === 'pv' || view === 'pvc' || view === 'namespaces') {
    props.statusClass = statusClass
  }

 if (view === 'nodes') {
    props.metrics = nodeMetrics.value
    props.statusClass = statusClass
    props.clusterName = activeClusterId.value
  }

  if (view === 'settings') {
    props.clusterName = activeClusterId.value
  }

  return props
})

const activeViewEvents = computed(() => {
  const config = activeViewConfig.value
  if (!config) return {}

  const events: Record<string, (...args: any[]) => void> = {}
  const view = currentView.value

  // Общие события
  events.refresh = refreshData

  if (config.hasSearch) {
    events.search = (v: string) => { search.value = v }
  }

  // View-specific события
  if (view === 'overview') {
    events.navigate = switchView
  }

  if (view === 'pods') {
    events.select = openPodDetail
    events.attachPod = podActions.openAttach
    events.shell = podActions.openShell
    events.evict = podActions.evictPod
    events.logs = (pod: any) => podActions.openLogs(pod)
    events.edit = podActions.editPod
    events.delete = podActions.removePod
  }

  if (view === 'deployments') {
    events.scale = doScale
    events.yaml = (ns: string, n: string) => showYAML('deployment', ns, n)
    events.restart = restartDeployment
    events.revisions = onShowRevisions
  }

  if (view === 'services') {
    events.select = openServiceDetail
    events.yaml = (ns: string, n: string) => showYAML('service', ns, n)
  }

  if (view === 'configmaps') {
    events.select = openConfigMapDetail
    events.yaml = (ns: string, n: string) => showYAML('configmap', ns, n)
  }

  if (view === 'secrets') {
    events.yaml = (ns: string, n: string) => showYAML('secret', ns, n)
  }

  if (view === 'nodes') {
    events.yaml = (ns: string, n: string) => showYAML('node', ns, n)
  }

  if (view === 'namespaces') {
    events.select = openNamespaceDetail
  }

  if (view === 'portforwarding') {
    events.refresh = refreshPortForwards
    events.stop = stopPortForward
  }

  return events
})

// ─── Handlers ───

function onNsChange(ns: string) {
  selectedNamespace.value = ns
  onNamespaceChange()
}

function openNamespaceDetail(namespace: any) {
  selectedNamespaceDetail.value = namespace
}

async function onShowRevisions(ns: string, name: string) {
  revisionDepNs.value = { ns, name }
  await loadRevisions(ns, name)
}

async function onRollback(rev: number) {
  await rollbackDeployment(revisionDepNs.value.ns, revisionDepNs.value.name, rev)
}

// ─── Watch error → notifications ───

import { watch } from 'vue'

watch(error, (val) => {
  if (val) {
    if (val.startsWith('Ошибка')) {
      notifications.notifyError(val)
    } else {
      notifications.notifySuccess(val)
    }
    // Очищаем error после показа нотификации
    setTimeout(() => { error.value = '' }, 100)
  }
})

// ─── Init ───

onMounted(() => clusterConn.initialize())
</script>

<style src="./assets/style.css"></style>