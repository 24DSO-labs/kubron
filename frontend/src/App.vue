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
      :on-start-port-forward="startPortForward"
      :on-stop-port-forward="stopPortForward"
      @close="selectedPod = null"
      @loadLogs="loadLogs"
      @runExec="runExec"
      @delete="podActions.removePod()"
      @openTerminal="podActions.terminalOpen.value = true"
      @evict="podActions.evictPod()"
      @edit="podActions.editPod()"
      @attach="onPodDetailAttach"
      @terminal="onPodDetailTerminal"
      @logs="onPodDetailLogs"
    />

    <!-- Deployment Detail Panel -->
    <DeploymentDetailPanel
      v-if="selectedDeployment"
      :deployment="selectedDeployment"
      @close="selectedDeployment = null"
      @pod-selected="openPodDetailFromDeployment"
      @scale="onDeploymentScale"
      @restart="onDeploymentRestart"
      @logs="onDeploymentLogs"
      @edit="onDeploymentEdit"
      @delete="onDeploymentDelete"
    />

    <!-- Scale Deployment Dialog -->
    <ScaleDeploymentDialog
      v-if="showScaleDialog && deploymentToScale"
      :deployment="deploymentToScale"
      @close="showScaleDialog = false"
      @scale="onScaleDeployment"
    />

    <!-- Tabbed Deployment Logs Panel -->
    <TabbedDeploymentLogsPanel
      v-if="tabbedDeploymentLogsOpen"
      ref="tabbedDeploymentLogsRef"
      @close="tabbedDeploymentLogsOpen = false"
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
      :namespace="selectedNamespaceDetail"
      @close="selectedNamespaceDetail = null"
    />

    <!-- ServiceAccount Detail Panel -->
    <!-- @ts-ignore -->
    <ServiceAccountDetailPanel
      v-if="selectedServiceAccount"
      :account="selectedServiceAccount"
      @close="selectedServiceAccount = null"
      @edit="onServiceAccountEdit"
      @delete="onServiceAccountDelete"
    />

    <!-- Role Detail Panel -->
    <RoleDetailPanel
      v-if="selectedRole"
      :role="selectedRole"
      @close="selectedRole = null"
      @edit="() => onRoleEdit(selectedRole?.namespace, selectedRole?.name)"
      @delete="() => onRoleDelete(selectedRole?.namespace, selectedRole?.name)"
    />

    <!-- RoleBinding Detail Panel -->
    <RoleBindingDetailPanel
      v-if="selectedRoleBinding"
      :binding="selectedRoleBinding"
      @close="selectedRoleBinding = null"
      @edit="() => onRoleBindingEdit(selectedRoleBinding.namespace, selectedRoleBinding.name)"
      @delete="() => onRoleBindingDelete(selectedRoleBinding.namespace, selectedRoleBinding.name)"
    />

    <!-- ClusterRole Detail Panel -->
    <ClusterRoleDetailPanel
      v-if="selectedClusterRole"
      :role="selectedClusterRole"
      @close="selectedClusterRole = null"
      @edit="() => onClusterRoleEdit('', selectedClusterRole.name)"
      @delete="() => onClusterRoleDelete('', selectedClusterRole.name)"
    />

    <!-- ClusterRoleBinding Detail Panel -->
    <ClusterRoleBindingDetailPanel
      v-if="selectedClusterRoleBinding"
      :binding="selectedClusterRoleBinding"
      @close="selectedClusterRoleBinding = null"
      @edit="() => onClusterRoleBindingEdit('', selectedClusterRoleBinding.name)"
      @delete="() => onClusterRoleBindingDelete('', selectedClusterRoleBinding.name)"
    />

    <!-- Tabbed Actions Panel (Terminal/Logs/Attach) -->
    <TabbedActionsPanel
      v-if="podActions.tabbedActionsOpen.value"
      :pod-actions="podActions"
      @close="podActions.closeTabbedActions()"
      @evict="podActions.evictPod()"
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
const TabbedYamlEditorPanel = defineAsyncComponent(() => import('./components/panels/TabbedYamlEditorPanel.vue'))
const TabbedActionsPanel = defineAsyncComponent(() => import('./components/panels/TabbedPodActionsPanel.vue'))
const RevisionsPanel = defineAsyncComponent(() => import('./components/panels/k8s_resource/RevisionsPanel.vue'))
const NamespaceDetailPanel = defineAsyncComponent(() => import('./components/panels/k8s_resource/NamespaceDetailPanel.vue'))

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

const tabbedActionsPanel = ref<any>(null)

import { watch, nextTick } from 'vue'

watch(tabbedActionsPanel, (newVal) => {
  if (newVal) {
    podActions.setTabbedActionsPanelRef(newVal)
  }
})

// ─── Local State ───

const selectedNamespaceDetail = ref<any | null>(null)
const selectedDeployment = ref<any | null>(null)
const selectedServiceAccount = ref<any | null>(null)
const selectedRole = ref<any | null>(null)
const selectedRoleBinding = ref<any | null>(null)
const selectedClusterRole = ref<any | null>(null)
const selectedClusterRoleBinding = ref<any | null>(null)
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
    events.attachPod = (pod: any, container?: string) => {
      if (container) {
        podActions.openAttachWithContainer(container, pod)
      } else {
        podActions.openAttach(pod)
      }
    }
    events.shell = (pod: any, container?: string) => {
      if (container) {
        podActions.openShellWithContainer(container, pod)
      } else {
        podActions.openShell(pod)
      }
    }
    events.evict = podActions.evictPod
    events.logs = (pod: any, container?: string) => {
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
      if (data.pod) {
        selectedPod.value = data.pod
        podActions.openAttachWithContainer(data.container, data.pod)
      }
    }
    events.terminal = (data: any) => {
      if (data.pod) {
        selectedPod.value = data.pod
        podActions.openShellWithContainer(data.container, data.pod)
      }
    }
  }

  if (view === 'deployments') {
    events.select = (dep: any) => { selectedDeployment.value = dep }
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

function openServiceAccountDetail(sa: any) {
  selectedServiceAccount.value = sa
}

function openPodDetailFromDeployment(pod: PodInfo) {
  console.log('Pod selected from deployment:', pod)
  selectedDeployment.value = null
  openPodDetail(pod)
}

async function onShowRevisions(ns: string, name: string) {
  revisionDepNs.value = { ns, name }
  await loadRevisions(ns, name)
}

async function onRollback(rev: number) {
  await rollbackDeployment(revisionDepNs.value.ns, revisionDepNs.value.name, rev)
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

// ─── Init ───

onMounted(() => clusterConn.initialize())
</script>

<style src="./assets/style.css"></style>
