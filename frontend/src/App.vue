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
      <ClusterBar :clusters="clusters" @select="clusterConn.selectCluster"
        @add="clusterConn.showAddCluster.value = true" @delete="clusterConn.deleteCluster"
        @update="clusterConn.updateCluster" />

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
        <Sidebar :currentView="currentView" :selectedNamespace="selectedNamespace" :namespaces="namespaces"
          :clusterName="activeClusterMeta?.displayName" @navigate="onNavigate" @nsChange="onNsChange" />

        <main class="content">
          <div v-if="loading" class="loading">⏳ Загрузка...</div>

          <component v-if="!loading && activeViewComponent" :is="activeViewComponent" v-bind="activeViewProps"
            v-on="activeViewEvents" />
        </main>
      </template>
    </div>
    <!-- ///////////////////////////////////////////////////// -->
    <!-- Universal Detail Panel -->
    <component v-if="globalDetailPanel.detailPanelState.value.isOpen"
      :is="globalDetailPanel.detailPanelState.value.component"
      :resource="globalDetailPanel.detailPanelState.value.resource"
      v-bind="globalDetailPanel.detailPanelState.value.props" v-on="globalDetailPanel.detailPanelState.value.events" />

    <!-- Unified Tabbed Panel (Logs & YAML Editor) -->
<TabbedPanel 
  v-if="globalTabbedPanel.isOpen()" 
  @close="globalTabbedPanel.closeAllTabs()"
  @apply-yaml="applyYAMLTabbed" 
/>

    <!-- Add Cluster Dialog -->
    <AddClusterDialog v-if="clusterConn.showAddCluster.value" @close="clusterConn.showAddCluster.value = false"
      @addFile="clusterConn.addFromFile" @addText="clusterConn.addFromText" />

    <!-- Notifications -->
    <NotificationToast :notifications="notifications.notifications.value" @dismiss="notifications.dismiss" />
  


    <!-- Pod Detail Panel -->
    <PodDetailPanel v-if="selectedPod" :pod="selectedPod" :logs="podLogs" v-model:logContainer="logContainer"
      v-model:logLines="logLines" v-model:execCmd="execCmd" :execOutput="execOutput" :statusClass="statusClass"
      v-model:pfLocalPort="pfLocalPort" v-model:pfPodPort="pfPodPort" :portForwards="portForwards || []"
      :cluster-name="activeClusterId" :on-start-port-forward="startPortForward" :on-stop-port-forward="stopPortForward"
      @close="selectedPod = null" @loadLogs="loadLogs" @runExec="runExec" @delete="podActions.removePod()"
      @openTerminal="podActions.terminalOpen.value = true" @evict="podActions.evictPod()" @edit="onPodDetailEdit"
      @attach="onPodDetailAttach" @terminal="onPodDetailTerminal" @logs="onPodDetailLogs"
      @open-resource="onOpenResource" />

    <!-- Deployment Detail Panel -->
    <DeploymentDetailPanel v-if="selectedDeployment" :deployment="selectedDeployment" @close="selectedDeployment = null"
      @scale="onDeploymentScale" @restart="onDeploymentRestart" @logs="onDeploymentLogs" @edit="onDeploymentEdit"
      @delete="onDeploymentDelete" @pod-selected="openPodDetailFromDeployment" @open-resource="onOpenResource"
      @show-details="onDeploymentShowReplicaSet" />

    <!-- Scale Deployment Dialog -->
    <ScaleDeploymentDialog v-if="showScaleDialog && deploymentToScale" :deployment="deploymentToScale"
      @close="showScaleDialog = false" @scale="onScaleDeployment" />

    <!-- Tabbed Deployment Logs Panel -->
    <TabbedDeploymentLogsPanel v-if="tabbedDeploymentLogsOpen" ref="tabbedDeploymentLogsRef"
      @close="tabbedDeploymentLogsOpen = false" />

    <!-- Service Detail Panel -->
    <ServiceDetailPanel v-if="selectedService" :svc="selectedService" :portForwards="portForwards"
      @close="selectedService = null" @forward="forwardServicePort" @stopForward="stopPortForward" />

    <!-- ConfigMap Panel -->
    <ConfigMapPanel v-if="selectedConfigMap" :cm="selectedConfigMap" @close="selectedConfigMap = null" />

    <!-- Namespace Detail Panel -->
    <NamespaceDetailPanel v-if="selectedNamespaceDetail" :namespace="selectedNamespaceDetail"
      @close="selectedNamespaceDetail = null" />

    <!-- ServiceAccount Detail Panel -->
    <!-- @ts-ignore -->
    <ServiceAccountDetailPanel v-if="selectedServiceAccount" :account="selectedServiceAccount"
      @close="selectedServiceAccount = null" @edit="onServiceAccountEdit" @delete="onServiceAccountDelete" />

    <!-- ReplicaSet Detail Panel -->
    <ReplicaSetDetailPanel v-if="selectedReplicaSet" :resource="selectedReplicaSet" @close="selectedReplicaSet = null"
      @edit="() => { }" @delete="() => { }" />

    <!-- DaemonSet Detail Panel -->
    <DaemonSetDetailPanel v-if="selectedDaemonSet" :daemonset="selectedDaemonSet" :cluster-name="activeClusterId"
      @close="selectedDaemonSet = null" @restart="onDaemonSetRestart" @logs="onDaemonSetLogs" @edit="onDaemonSetEdit"
      @delete="onDaemonSetDelete" @pod-selected="openPodDetailFromDaemonSet" @open-resource="onOpenResource" />

    <!-- Node Detail View -->
    <NodeDetailView v-if="selectedNode" :node="selectedNode" :status-class="statusClass" :cluster-name="activeClusterId"
      @close="selectedNode = null" @edit="() => { }" />

    <!-- Role Detail Panel -->
    <RoleDetailPanel v-if="selectedRole" :role="selectedRole" @close="selectedRole = null"
      @edit="() => onRoleEdit(selectedRole?.namespace, selectedRole?.name)"
      @delete="() => onRoleDelete(selectedRole?.namespace, selectedRole?.name)" />

    <!-- RoleBinding Detail Panel -->
    <RoleBindingDetailPanel v-if="selectedRoleBinding" :binding="selectedRoleBinding"
      @close="selectedRoleBinding = null"
      @edit="() => onRoleBindingEdit(selectedRoleBinding.namespace, selectedRoleBinding.name)"
      @delete="() => onRoleBindingDelete(selectedRoleBinding.namespace, selectedRoleBinding.name)" />

    <!-- ClusterRole Detail Panel -->
    <ClusterRoleDetailPanel v-if="selectedClusterRole" :role="selectedClusterRole" @close="selectedClusterRole = null"
      @edit="() => onClusterRoleEdit('', selectedClusterRole.name)"
      @delete="() => onClusterRoleDelete('', selectedClusterRole.name)" />

    <!-- ClusterRoleBinding Detail Panel -->
    <ClusterRoleBindingDetailPanel v-if="selectedClusterRoleBinding" :binding="selectedClusterRoleBinding"
      @close="selectedClusterRoleBinding = null"
      @edit="() => onClusterRoleBindingEdit('', selectedClusterRoleBinding.name)"
      @delete="() => onClusterRoleBindingDelete('', selectedClusterRoleBinding.name)" />

    

    <TabbedPodActionsPanel v-if="logsPanel.isOpen.value" :resource="logsPanel.logsData.value?.resource"
      :selectedContainer="logsPanel.logsData.value?.selectedContainer"
      :containers="logsPanel.logsData.value?.containers" @close="logsPanel.closeLogs" />

    
    
    <!-- Connecting overlay -->
    <div v-if="clusterConn.connecting.value" class="overlay">
      <div class="spinner">⏳ Подключение...</div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { useK8s } from './composables/useK8s'
import { useClusters } from './composables/useClusters'
import { usePodActions } from './composables/usePodActions'
import { useClusterConnection } from './composables/useClusterConnection'
import { useNotifications } from './composables/useNotifications'
import { useLogsPanel } from './composables/useLogsPanel'
import { viewRegistry } from './composables/useViewRegistry'
import { globalDetailPanel } from './composables/useDetailPanel'
import { globalTabbedPanel } from './composables/useTabbedPanel'
import type { PodInfo } from './types'

import ClusterBar from './components/ClusterBar.vue'
import AddClusterDialog from './components/AddClusterDialog.vue'
import Sidebar from './components/Sidebar.vue'
import NotificationToast from './components/NotificationToast.vue'

// Ленивая загрузка панелей
import { defineAsyncComponent } from 'vue'
const PodDetailPanel = defineAsyncComponent(() => import('./components/panels/k8s_resource/PodDetailPanel.vue'))
const DeploymentDetailPanel = defineAsyncComponent(() => import('./components/panels/k8s_resource/DeploymentDetailPanel.vue'))
const ScaleDeploymentDialog = defineAsyncComponent(() => import('./components/panels/ScaleDeploymentDialog.vue'))
const TabbedDeploymentLogsPanel = defineAsyncComponent(() => import('./components/panels/TabbedDeploymentLogsPanel.vue'))
const ServiceDetailPanel = defineAsyncComponent(() => import('./components/panels/k8s_resource/ServiceDetailPanel.vue'))
const ServiceAccountDetailPanel = defineAsyncComponent(() => import('./components/panels/k8s_resource/ServiceAccountDetailPanel.vue'))
const RoleDetailPanel = defineAsyncComponent(() => import('./components/panels/k8s_resource/RoleDetailPanel.vue'))
const RoleBindingDetailPanel = defineAsyncComponent(() => import('./components/panels/k8s_resource/RoleBindingDetailPanel.vue'))
const ClusterRoleDetailPanel = defineAsyncComponent(() => import('./components/panels/k8s_resource/ClusterRoleDetailPanel.vue'))
const ClusterRoleBindingDetailPanel = defineAsyncComponent(() => import('./components/panels/k8s_resource/ClusterRoleBindingDetailPanel.vue'))
const ConfigMapPanel = defineAsyncComponent(() => import('./components/panels/k8s_resource/ConfigMapPanel.vue'))
const TabbedPanel = defineAsyncComponent(() => import('./components/utils/TabbedPanel.vue'))

const NamespaceDetailPanel = defineAsyncComponent(() => import('./components/panels/k8s_resource/NamespaceDetailPanel.vue'))
const ReplicaSetDetailPanel = defineAsyncComponent(() => import('./components/panels/k8s_resource/ReplicaSetDetailPanel.vue'))
const DaemonSetDetailPanel = defineAsyncComponent(() => import('./components/panels/k8s_resource/DaemonSetDetailPanel.vue'))
const NodeDetailView = defineAsyncComponent(() => import('./components/views/NodeDetailView.vue'))

// ─── Composables ───

const k8s = useK8s()
const clustersApi = useClusters()
const notifications = useNotifications()
const logsPanel = useLogsPanel()

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
  loadNamespaces, deletePodByName, deleteDeploymentByName,
  getPods, filteredPods, filteredDeployments, filteredStatefulSets,
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
const selectedDeployment = ref<any | null>(null)
const selectedServiceAccount = ref<any | null>(null)
const selectedReplicaSet = ref<any | null>(null)
const selectedRole = ref<any | null>(null)
const selectedRoleBinding = ref<any | null>(null)
const selectedClusterRole = ref<any | null>(null)
const selectedClusterRoleBinding = ref<any | null>(null)
const selectedNode = ref<any | null>(null)
const selectedDaemonSet = ref<any | null>(null)
const showScaleDialog = ref(false)
const deploymentToScale = ref<any | null>(null)
const revisionDepNs = ref({ ns: '', name: '' })
const deploymentYamlEditorOpen = ref(false)
const deploymentYamlEditMeta = ref({ namespace: '', name: '' })
const deploymentLogsOpen = ref(false)
const deploymentForLogs = ref<any | null>(null)
const tabbedDeploymentLogsOpen = ref(false)
const tabbedDeploymentLogsRef = ref<any>(null)

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
    events.attachPod = (pod: any, container?: string) => {
      console.log('[App] Событие attachPod', { pod, container })
      if (container) {
        podActions.openAttachWithContainer(container, pod)
      } else {
        podActions.openAttach(pod)
      }
    }
    events.shell = (pod: any, container?: string) => {
      console.log('[App] Событие shell', { pod, container })
      if (container) {
        podActions.openShellWithContainer(container, pod)
      } else {
        podActions.openShell(pod)
      }
    }
    events.evict = podActions.evictPod
    events.logs = (pod: any, container?: string) => {
      console.log('[App] Событие logs', { pod, container })
      if (container) {
        podActions.openLogs(pod, container)
      } else {
        podActions.openLogs(pod)
      }
    }
    events.edit = podActions.editPod
    events.delete = podActions.removePod
    // События из PodDetailPanel для TabbedPodActionsPanel
    events.attach = (data: any) => {
      console.log('[App] Событие attach', data)
      if (data.pod) {
        selectedPod.value = data.pod
        podActions.openAttachWithContainer(data.container, data.pod)
      }
    }
    events.terminal = (data: any) => {
      console.log('[App] Событие terminal', data)
      if (data.pod) {
        selectedPod.value = data.pod
        podActions.openShellWithContainer(data.container, data.pod)
      }
    }
    events.openLogs = (data) => {
      console.log('[App] Событие openLogs', data)
      logsPanel.openLogs(data)
    }
    // Универсальный обработчик контекстного меню для всех ресурсов
    events.contextAction = createUniversalContextActionHandler('pod')
    events['shell:open'] = (ns: string, podName: string, container: string) => {
      console.log('[App] Событие shell:open', { ns, podName, container })
      handleShellOpen(ns, podName, container)
    }
    events['attach:open'] = (ns: string, podName: string, container: string) => {
      console.log('[App] Событие attach:open', { ns, podName, container })
      handleAttachOpen(ns, podName, container)
    }
    events['logs:open'] = (ns: string, podName: string, container: string) => {
      console.log('[App] Событие logs:open', { ns, podName, container })
      handleLogsOpen(ns, podName, container)
    }
  }

  if (view === 'deployments') {
    events.select = (dep: any) => {
      selectedDeployment.value = dep
    }
    events.scale = (dep: any) => {
      deploymentToScale.value = dep
      showScaleDialog.value = true
    }
    events.yaml = (ns: string, n: string) => showYAML('deployment', ns, n)
    events.restart = (ns: string, n: string) => {
      if (confirm(`Restart deployment ${n}?`)) {
        restartDeployment(ns, n)
      }
    }
    events.logs = (ns: string, n: string) => {
      tabbedDeploymentLogsOpen.value = true
      nextTick(() => {
        if (tabbedDeploymentLogsRef.value) {
          tabbedDeploymentLogsRef.value.setPendingDeployment({ namespace: ns, name: n })
        }
      })
    }
    events.edit = (ns: string, n: string) => {
      podActions.yamlEditorOpen.value = true
      podActions.yamlEditorMeta.value = { resourceType: 'deployment', resourceName: n, namespace: ns }
      showYAML('deployment', ns, n)
    }
    events.delete = (ns: string, n: string) => {
      if (confirm(`Delete deployment ${n}?`)) {
        deleteDeploymentByName(ns, n)
          .then(() => {
            notifications.notifySuccess(`Deployment ${n} deleted`)
            refreshData()
          })
          .catch((e: any) => {
            notifications.notifyError(`Delete error: ${e.message || e}`)
          })
      }
    }
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

  if (view === 'serviceaccounts') {
    events.select = openServiceAccountDetail
    events.yaml = async (ns: string, n: string) => {
      try {
        podActions.yamlEditorMeta.value = { resourceType: 'serviceaccount', resourceName: n, namespace: ns }
        await showYAML('serviceaccount', ns, n)
        podActions.yamlEditorOpen.value = true
      } catch (e: any) {
        notifications.notifyError(`Error loading YAML: ${e.message || e}`)
      }
    }
    events.delete = (ns: string, n: string) => {
      if (confirm(`Delete ServiceAccount ${n}?`)) {
        k8s.deleteServiceAccount(ns, n)
          .then(() => {
            notifications.notifySuccess(`ServiceAccount ${n} deleted`)
            refreshData()
          })
          .catch((e: any) => {
            notifications.notifyError(`Delete error: ${e.message || e}`)
          })
      }
    }
  }

  if (view === 'roles') {
    events.select = (role: any) => { selectedRole.value = role }
    events.edit = (ns: string, n: string) => onRoleEdit(ns, n)
    events.yaml = (ns: string, n: string) => {
      podActions.yamlEditorMeta.value = { resourceType: 'role', resourceName: n, namespace: ns }
      showYAML('role', ns, n)
        .then(() => {
          podActions.yamlEditorOpen.value = true
        })
        .catch((e: any) => {
          notifications.notifyError(`Error loading YAML: ${e.message || e}`)
        })
    }
    events.delete = (ns: string, n: string) => {
      if (confirm(`Delete Role ${n}?`)) {
        k8s.deleteRole(ns, n)
          .then(() => {
            notifications.notifySuccess(`Role ${n} deleted`)
            refreshData()
          })
          .catch((e: any) => {
            notifications.notifyError(`Delete error: ${e.message || e}`)
          })
      }
    }
  }

  if (view === 'rolebindings') {
    events.select = (rb: any) => { selectedRoleBinding.value = rb }
    events.edit = (ns: string, n: string) => onRoleBindingEdit(ns, n)
    events.delete = (ns: string, n: string) => onRoleBindingDelete(ns, n)
  }

  if (view === 'clusterroles') {
    events.select = (cr: any) => { selectedClusterRole.value = cr }
    events.edit = (ns: string, n: string) => onClusterRoleEdit(ns, n)
    events.delete = (ns: string, n: string) => onClusterRoleDelete(ns, n)
  }

  if (view === 'clusterrolebindings') {
    events.select = (crb: any) => { selectedClusterRoleBinding.value = crb }
    events.edit = (ns: string, n: string) => onClusterRoleBindingEdit(ns, n)
    events.delete = (ns: string, n: string) => onClusterRoleBindingDelete('', n)
  }

  if (view === 'replicasets') {
    events.select = (rs: any) => {
      selectedReplicaSet.value = rs
    }
  }

  if (view === 'daemonsets') {
    events.select = (ds: any) => {
      selectedDaemonSet.value = ds
    }
    events.restart = (ns: string, n: string) => {
      if (confirm(`Restart daemonset ${n}?`)) {
        restartDeployment(ns, n)
      }
    }
    events.logs = (ns: string, n: string) => {
      tabbedDeploymentLogsOpen.value = true
      nextTick(() => {
        if (tabbedDeploymentLogsRef.value) {
          tabbedDeploymentLogsRef.value.setPendingDeployment({ namespace: ns, name: n })
        }
      })
    }
    events.edit = (ns: string, n: string) => {
      podActions.yamlEditorOpen.value = true
      podActions.yamlEditorMeta.value = { resourceType: 'daemonset', resourceName: n, namespace: ns }
      showYAML('daemonset', ns, n)
    }
    events.delete = (ns: string, n: string) => {
      if (confirm(`Delete daemonset ${n}?`)) {
        deleteDeploymentByName(ns, n)
          .then(() => {
            notifications.notifySuccess(`DaemonSet ${n} deleted`)
            refreshData()
          })
          .catch((e: any) => {
            notifications.notifyError(`Delete error: ${e.message || e}`)
          })
      }
    }
  }

  if (view === 'portforwarding') {
    events.refresh = refreshPortForwards
    events.stop = stopPortForward
  }

  return events
})

// ─── Handlers ───

// Универсальный обработчик контекстного меню для всех ресурсов
function createUniversalContextActionHandler(resourceType: string) {
  return (action: string, item: any) => {
    // Обработка действий с подменю контейнеров (shell:container, attach:container, logs:container)
    if (action.startsWith('shell:')) {
      const container = action.split(':')[1]
      if (resourceType === 'pod') {
        podActions.openShellWithContainer(container, item)
      }
      return
    }
    if (action.startsWith('attach:')) {
      const container = action.split(':')[1]
      if (resourceType === 'pod') {
        podActions.openAttachWithContainer(container, item)
      }
      return
    }
    if (action.startsWith('logs:')) {
      const container = action.split(':')[1]
      if (resourceType === 'pod') {
        podActions.openLogs(item, container)
      } else {
        openLogsForAnyResource(item, container)
      }
      return
    }
    
    switch (action) {
      case 'details':
        openResourceDetail(resourceType, item)
        break
      case 'shell':
        if (resourceType === 'pod') {
          if (item.containers && item.containers.length > 0) {
            podActions.openShellWithContainer(item.containers[0].name, item)
          }
        }
        break
      case 'attach':
        if (resourceType === 'pod') {
          if (item.containers && item.containers.length > 0) {
            podActions.openAttachWithContainer(item.containers[0].name, item)
          }
        }
        break
      case 'logs':
        if (resourceType === 'pod') {
          if (item.containers && item.containers.length > 0) {
            podActions.openLogs(item, item.containers[0].name)
          }
        } else {
          openLogsForAnyResource(item)
        }
        break
      case 'edit':
        openResourceEdit(resourceType, item)
        break
      case 'delete':
        deleteResource(resourceType, item)
        break
      case 'evict':
        if (resourceType === 'pod') podActions.evictPod(item)
        break
    }
  }
}

function openResourceDetail(resourceType: string, item: any) {
  // Используем глобальную панель деталей
  globalDetailPanel.openResourceDetail(resourceType, item)
}

function openResourceEdit(resourceType: string, item: any) {
  if (resourceType === 'pod') {
    onPodDetailEdit()
  } else {
    // Для других ресурсов открываем YAML редактор
    const ns = item.namespace || ''
    const name = item.name
    podActions.yamlEditorMeta.value = { resourceType, resourceName: name, namespace: ns }
    showYAML(resourceType, ns, name)
      .then(() => {
        podActions.yamlEditorOpen.value = true
      })
      .catch((e: any) => {
        notifications.notifyError(`Error loading YAML: ${e.message || e}`)
      })
  }
}

function deleteResource(resourceType: string, item: any) {
  const ns = item.namespace || ''
  const name = item.name
  if (confirm(`Delete ${resourceType} ${name}?`)) {
    // Используем k8s функции для удаления
    switch (resourceType) {
      case 'pod':
        k8s.deletePodByName(ns, name).then(() => {
          notifications.notifySuccess(`${resourceType} ${name} deleted`)
          refreshData()
        }).catch((e: any) => notifications.notifyError(`Delete error: ${e.message || e}`))
        break
      case 'deployment':
        k8s.deleteDeploymentByName(ns, name).then(() => {
          notifications.notifySuccess(`${resourceType} ${name} deleted`)
          refreshData()
        }).catch((e: any) => notifications.notifyError(`Delete error: ${e.message || e}`))
        break
      case 'serviceaccount':
        k8s.deleteServiceAccount(ns, name).then(() => {
          notifications.notifySuccess(`${resourceType} ${name} deleted`)
          refreshData()
        }).catch((e: any) => notifications.notifyError(`Delete error: ${e.message || e}`))
        break
      case 'role':
        k8s.deleteRole(ns, name).then(() => {
          notifications.notifySuccess(`${resourceType} ${name} deleted`)
          refreshData()
        }).catch((e: any) => notifications.notifyError(`Delete error: ${e.message || e}`))
        break
      case 'rolebinding':
        k8s.deleteRoleBinding(ns, name).then(() => {
          notifications.notifySuccess(`${resourceType} ${name} deleted`)
          refreshData()
        }).catch((e: any) => notifications.notifyError(`Delete error: ${e.message || e}`))
        break
      case 'clusterrole':
        k8s.deleteClusterRole(name).then(() => {
          notifications.notifySuccess(`${resourceType} ${name} deleted`)
          refreshData()
        }).catch((e: any) => notifications.notifyError(`Delete error: ${e.message || e}`))
        break
      case 'clusterrolebinding':
        k8s.deleteClusterRoleBinding(name).then(() => {
          notifications.notifySuccess(`${resourceType} ${name} deleted`)
          refreshData()
        }).catch((e: any) => notifications.notifyError(`Delete error: ${e.message || e}`))
        break
      default:
        console.warn(`Delete not implemented for resource type: ${resourceType}`)
    }
  }
}

function openLogsForAnyResource(item: any, container?: string) {
  // Для всех ресурсов открываем логи через TabbedPanel
  const ns = item.namespace || ''
  const name = item.name
  const containerName = container || name
  handleLogsOpen(ns, name, containerName)
}

// Обработчики для TabbedPanel (shell/attach/logs)
function handleShellOpen(namespace: string, pod: string, container: string) {
  globalTabbedPanel.openTerminalTab('pod', pod, namespace, pod, container, 'shell')
}

function handleAttachOpen(namespace: string, pod: string, container: string) {
  globalTabbedPanel.openTerminalTab('pod', pod, namespace, pod, container, 'attach')
}

function handleLogsOpen(namespace: string, resourceName: string, container: string) {
  // Определяем тип ресурса из контекста (можно расширить)
  // Пока используем 'pod' как default, но в будущем можно передавать тип ресурса
  globalTabbedPanel.openLogsTab('pod', resourceName, namespace, [{ name: container }], container)
}

function closeAllDetailPanels() {
  selectedPod.value = null
  selectedDeployment.value = null
  selectedConfigMap.value = null
  selectedService.value = null
  selectedNamespaceDetail.value = null
  selectedServiceAccount.value = null
  selectedReplicaSet.value = null
  selectedNode.value = null
  selectedRole.value = null
  selectedRoleBinding.value = null
  selectedClusterRole.value = null
  selectedClusterRoleBinding.value = null
  showScaleDialog.value = false
  tabbedDeploymentLogsOpen.value = false
  revisions.value = []
}

async function onPodDetailEdit() {
  if (!selectedPod.value) return
  const ns = selectedPod.value.namespace
  const name = selectedPod.value.name
  podActions.yamlEditorMeta.value = { resourceType: 'pod', resourceName: name, namespace: ns }
  await showYAML('pod', ns, name)
  podActions.yamlEditorOpen.value = true
}

function onNavigate(view: string) {
  closeAllDetailPanels()
  switchView(view)
}

function onNsChange(ns: string) {
  selectedNamespace.value = ns
  onNamespaceChange()
}

function openNamespaceDetail(namespace: any) {
  selectedNamespaceDetail.value = namespace
}

function openServiceAccountDetail(sa: any) {
  selectedServiceAccount.value = sa
}

function openPodDetailFromDeployment(pod: PodInfo) {
  selectedDeployment.value = null
  openPodDetail(pod)
}


async function onShowRevisions(ns: string, name: string) {
  revisionDepNs.value = { ns, name }
  await loadRevisions(ns, name)
}

async function onRollback(rev: string | number) {
  const revNum = typeof rev === 'string' ? parseInt(rev, 10) : rev
  await rollbackDeployment(revisionDepNs.value.ns, revisionDepNs.value.name, revNum)
}

function onPodDetailAttach(data: any) {

  if (data.pod) {
    selectedPod.value = data.pod
    podActions.openAttachWithContainer(data.container, data.pod)
  }
}

function onPodDetailTerminal(data: any) {

  if (data.pod) {
    selectedPod.value = data.pod
    podActions.openShellWithContainer(data.container, data.pod)
  }
}

function onPodDetailLogs(data: any) {

  if (data.pod) {
    selectedPod.value = data.pod
    podActions.openLogs(data.pod, data.container)
  }
}

async function onOpenResource(data: { kind: string; name: string; namespace?: string }) {
  console.log('App.onOpenResource called with:', data)
  let { kind, name, namespace } = data

  // Если name содержит "/" (например "ReplicaSet/my-name"), распарсить
  if (name.includes('/')) {
    const parts = name.split('/')
    if (parts.length === 2) {
      kind = parts[0]
      name = parts[1]
    }
  }


  // Закрываем все открытые панели детей перед открытием нового ресурса
  selectedPod.value = null
  selectedDaemonSet.value = null
  selectedDeployment.value = null
  selectedReplicaSet.value = null

  try {
    switch (kind) {
      case 'Namespace':
        console.log('Opening Namespace:', name)
        const ns = await k8s.getNamespace(name)
        console.log('Namespace loaded:', ns)
        selectedNamespaceDetail.value = ns
        break

      case 'Node':
        const node = await k8s.getNode(name)
        selectedNode.value = node
        break

      case 'ServiceAccount':
        if (namespace) {
          const sa = await k8s.getServiceAccount(namespace, name)
          selectedServiceAccount.value = sa
        }
        break

      case 'ReplicaSet':
        if (namespace) {
          try {
            const rs = await k8s.getReplicaSet(namespace, name)
            selectedReplicaSet.value = rs
          } catch (e: any) {
            console.error('Error loading ReplicaSet:', e)
            throw e
          }
        } else {
          console.warn('No namespace provided for ReplicaSet')
        }
        break

      default:
        notifications.notifyError(`Unknown resource type: ${kind}`)
    }
  } catch (e: any) {
    notifications.notifyError(`Failed to load ${kind}: ${e.message || e}`)
  }
}

async function onScaleDeployment(replicas: number) {
  if (!deploymentToScale.value) return
  try {
    const ns = deploymentToScale.value.namespace
    const name = deploymentToScale.value.name
    const result = await k8s.scaleDeployment(ns, name, replicas)
    notifications.notifySuccess(`Deployment ${name} scaled to ${replicas} replicas`)
    showScaleDialog.value = false
    await refreshData()
  } catch (e: any) {
    notifications.notifyError(`Scale error: ${e.message || e}`)
  }
}

function onDeploymentScale(dep: any) {
  deploymentToScale.value = dep
  showScaleDialog.value = true
}

function onDeploymentRestart(ns: string, name: string) {
  if (confirm(`Restart deployment ${name}?`)) {
    restartDeployment(ns, name)
  }
}

function onDeploymentLogs(ns: string, name: string) {
  tabbedDeploymentLogsOpen.value = true
  nextTick(() => {
    if (tabbedDeploymentLogsRef.value) {
      tabbedDeploymentLogsRef.value.setPendingDeployment({ namespace: ns, name: name })
    }
  })
}

async function onDeploymentEdit() {
  if (!selectedDeployment.value) return
  const ns = selectedDeployment.value.namespace
  const name = selectedDeployment.value.name
  podActions.yamlEditorMeta.value = { resourceType: 'deployment', resourceName: name, namespace: ns }
  await showYAML('deployment', ns, name)
  podActions.yamlEditorOpen.value = true
}

function onDeploymentDelete() {
  if (!selectedDeployment.value) return
  const ns = selectedDeployment.value.namespace
  const name = selectedDeployment.value.name
  if (confirm(`Delete deployment ${name}?`)) {
    deleteDeploymentByName(ns, name)
      .then(() => {
        notifications.notifySuccess(`Deployment ${name} deleted`)
        selectedDeployment.value = null
        refreshData()
      })
      .catch((e: any) => {
        notifications.notifyError(`Delete error: ${e.message || e}`)
      })
  }
}

async function onDeploymentShowReplicaSet(revision: any) {
  if (!selectedDeployment.value) return
  try {
    const ns = selectedDeployment.value.namespace
    const rsName = revision.name
    const rs = await k8s.getReplicaSet(ns, rsName)
    selectedDeployment.value = null
    selectedReplicaSet.value = rs
  } catch (e: any) {
    notifications.notifyError(`Failed to load ReplicaSet: ${e.message || e}`)
  }
}

function openPodDetailFromDaemonSet(pod: PodInfo) {
  selectedDaemonSet.value = null
  openPodDetail(pod)
}

function onDaemonSetRestart(ns: string, name: string) {
  if (confirm(`Restart daemonset ${name}?`)) {
    restartDeployment(ns, name)
  }
}

function onDaemonSetLogs(ns: string, name: string) {
  tabbedDeploymentLogsOpen.value = true
  nextTick(() => {
    if (tabbedDeploymentLogsRef.value) {
      tabbedDeploymentLogsRef.value.setPendingDeployment({ namespace: ns, name: name })
    }
  })
}

async function onDaemonSetEdit() {
  if (!selectedDaemonSet.value) return
  const ns = selectedDaemonSet.value.namespace
  const name = selectedDaemonSet.value.name
  podActions.yamlEditorMeta.value = { resourceType: 'daemonset', resourceName: name, namespace: ns }
  await showYAML('daemonset', ns, name)
  podActions.yamlEditorOpen.value = true
}

function onDaemonSetDelete() {
  if (!selectedDaemonSet.value) return
  const ns = selectedDaemonSet.value.namespace
  const name = selectedDaemonSet.value.name
  if (confirm(`Delete daemonset ${name}?`)) {
    deleteDeploymentByName(ns, name)
      .then(() => {
        notifications.notifySuccess(`DaemonSet ${name} deleted`)
        selectedDaemonSet.value = null
        refreshData()
      })
      .catch((e: any) => {
        notifications.notifyError(`Delete error: ${e.message || e}`)
      })
  }
}

async function onServiceAccountEdit() {
  if (!selectedServiceAccount.value) return
  const ns = selectedServiceAccount.value.namespace
  const name = selectedServiceAccount.value.name

  try {
    podActions.yamlEditorMeta.value = { resourceType: 'serviceaccount', resourceName: name, namespace: ns }
    await showYAML('serviceaccount', ns, name)
    podActions.yamlEditorOpen.value = true
  } catch (e: any) {
    notifications.notifyError(`Error loading YAML: ${e.message || e}`)
  }
}

function onServiceAccountDelete() {
  if (!selectedServiceAccount.value) return
  const ns = selectedServiceAccount.value.namespace
  const name = selectedServiceAccount.value.name

  if (confirm(`Delete ServiceAccount ${name}?`)) {
    k8s.deleteServiceAccount(ns, name)
      .then(() => {
        notifications.notifySuccess(`ServiceAccount ${name} deleted`)
        selectedServiceAccount.value = null
        refreshData()
      })
      .catch((e: any) => {
        notifications.notifyError(`Delete error: ${e.message || e}`)
      })
  }
}

function onServiceAccountNamespaceClick(ns: string | undefined) {
  if (ns) {
    selectedNamespace.value = ns
    onNamespaceChange()
  }
}

async function onRoleEdit(...args: any[]) {
  const [nsOrEvent, nameParam] = args
  const ns = typeof nsOrEvent === 'string' ? nsOrEvent : nsOrEvent?.namespace
  const name = typeof nsOrEvent === 'string' ? nameParam : nsOrEvent?.name

  try {
    podActions.yamlEditorMeta.value = { resourceType: 'role', resourceName: name, namespace: ns }
    await showYAML('role', ns, name)
    podActions.yamlEditorOpen.value = true
  } catch (e: any) {
    notifications.notifyError(`Error loading YAML: ${e.message || e}`)
  }
}

function onRoleDelete(...args: any[]) {
  const [nsOrEvent, nameParam] = args
  const ns = typeof nsOrEvent === 'string' ? nsOrEvent : nsOrEvent?.namespace
  const name = typeof nsOrEvent === 'string' ? nameParam : nsOrEvent?.name

  if (confirm(`Delete Role ${name}?`)) {
    k8s.deleteRole(ns, name)
      .then(() => {
        notifications.notifySuccess(`Role ${name} deleted`)
        selectedRole.value = null
        refreshData()
      })
      .catch((e: any) => {
        notifications.notifyError(`Delete error: ${e.message || e}`)
      })
  }
}

function onRoleNamespaceClick(ns: string | undefined) {
  if (ns) {
    selectedNamespace.value = ns
    onNamespaceChange()
  }
}

async function onRoleBindingEdit(...args: any[]) {
  const [nsOrEvent, nameParam] = args
  const ns = typeof nsOrEvent === 'string' ? nsOrEvent : nsOrEvent?.namespace
  const name = typeof nsOrEvent === 'string' ? nameParam : nsOrEvent?.name

  try {
    podActions.yamlEditorMeta.value = { resourceType: 'rolebinding', resourceName: name, namespace: ns }
    await showYAML('rolebinding', ns, name)
    podActions.yamlEditorOpen.value = true
  } catch (e: any) {
    notifications.notifyError(`Error loading YAML: ${e.message || e}`)
  }
}

function onRoleBindingDelete(...args: any[]) {
  const [nsOrEvent, nameParam] = args
  const ns = typeof nsOrEvent === 'string' ? nsOrEvent : nsOrEvent?.namespace
  const name = typeof nsOrEvent === 'string' ? nameParam : nsOrEvent?.name

  if (confirm(`Delete RoleBinding ${name}?`)) {
    k8s.deleteRoleBinding(ns, name)
      .then(() => {
        notifications.notifySuccess(`RoleBinding ${name} deleted`)
        selectedRoleBinding.value = null
        refreshData()
      })
      .catch((e: any) => {
        notifications.notifyError(`Delete error: ${e.message || e}`)
      })
  }
}

function onRoleBindingNamespaceClick(ns: string | undefined) {
  if (ns) {
    selectedNamespace.value = ns
    onNamespaceChange()
  }
}

async function onClusterRoleEdit(...args: any[]) {
  const [nsOrEvent, nameParam] = args
  const name = typeof nsOrEvent === 'string' ? nameParam : nsOrEvent?.name

  try {
    podActions.yamlEditorMeta.value = { resourceType: 'clusterrole', resourceName: name, namespace: '' }
    await showYAML('clusterrole', '', name)
    podActions.yamlEditorOpen.value = true
  } catch (e: any) {
    notifications.notifyError(`Error loading YAML: ${e.message || e}`)
  }
}

function onClusterRoleDelete(...args: any[]) {
  const name = typeof args[0] === 'string' ? args[1] : args[0]?.name

  if (confirm(`Delete ClusterRole ${name}?`)) {
    k8s.deleteClusterRole(name)
      .then(() => {
        notifications.notifySuccess(`ClusterRole ${name} deleted`)
        selectedClusterRole.value = null
        refreshData()
      })
      .catch((e: any) => {
        notifications.notifyError(`Delete error: ${e.message || e}`)
      })
  }
}

async function onClusterRoleBindingEdit(...args: any[]) {
  const [nsOrEvent, nameParam] = args
  const name = typeof nsOrEvent === 'string' ? nameParam : nsOrEvent?.name

  try {
    podActions.yamlEditorMeta.value = { resourceType: 'clusterrolebinding', resourceName: name, namespace: '' }
    await showYAML('clusterrolebinding', '', name)
    podActions.yamlEditorOpen.value = true
  } catch (e: any) {
    notifications.notifyError(`Error loading YAML: ${e.message || e}`)
  }
}

function onClusterRoleBindingDelete(...args: any[]) {
  const [nsOrEvent, nameParam] = args
  const name = typeof nsOrEvent === 'string' ? nameParam : nsOrEvent?.name

  if (confirm(`Delete ClusterRoleBinding ${name}?`)) {
    k8s.deleteClusterRoleBinding(name)
      .then(() => {
        notifications.notifySuccess(`ClusterRoleBinding ${name} deleted`)
        selectedClusterRoleBinding.value = null
        refreshData()
      })
      .catch((e: any) => {
        notifications.notifyError(`Delete error: ${e.message || e}`)
      })
  }
}

// ─── Watch error → notifications ───

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

async function applyYAMLTabbed(tabId: string) {
  const tab = globalTabbedPanel.tabs.value.find(t => t.id === tabId)
  if (tab && tab.type === 'yaml') {
    try {
      await k8s.applyYAML(tab.content, tab.resourceType, tab.namespace, tab.resourceName)
      notifications.notifySuccess(`${tab.resourceType} ${tab.resourceName} updated`)
      refreshData()
    } catch (e: any) {
      notifications.notifyError(`Apply error: ${e.message || e}`)
    }
  }
}

// ─── Init ───

onMounted(() => clusterConn.initialize())
</script>

<style src="./assets/style.css"></style>
