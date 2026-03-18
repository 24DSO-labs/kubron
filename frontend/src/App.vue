<template>
  <div id="app">
    <header class="header">
      <h1>🚀 Kubron</h1>
      <div v-if="activeClusterId" class="connection-info">
        <span class="active-cluster-icon">{{ activeClusterMeta?.icon }}</span>
        <span class="active-cluster-name" :style="{ color: activeClusterMeta?.color }">
          {{ activeClusterMeta?.displayName }}
        </span>
        <button class="btn-disconnect" @click="disconnect">✕</button>
      </div>
    </header>

    <div class="app-body">
      <!-- Cluster Bar -->
      <ClusterBar :clusters="clusters" @select="onSelectCluster" @add="showAddCluster = true" @delete="onDeleteCluster"
        @update="onUpdateCluster" />

      <!-- Нет кластеров -->
      <div v-if="clusters.length === 0 && !showAddCluster" class="welcome-screen">
        <h2>Добро пожаловать в Kubron!</h2>
        <p>Добавьте кластер для начала работы</p>
        <button class="btn" @click="showAddCluster = true">➕ Добавить кластер</button>
      </div>

      <!-- Не подключен но кластеры есть -->
      <div v-else-if="!activeClusterId && clusters.length > 0 && !connecting" class="welcome-screen">
        <h2>Выберите кластер</h2>
        <p>Кликните на иконку кластера слева для подключения</p>
      </div>

      <!-- Основной интерфейс -->
      <template v-else-if="activeClusterId">
        <Sidebar :currentView="currentView" :selectedNamespace="selectedNamespace" :namespaces="namespaces"
          :clusterName="activeClusterMeta?.displayName" @navigate="switchView"
          @nsChange="(ns) => { selectedNamespace = ns; onNamespaceChange() }" />
        <main class="content">
          <div v-if="loading" class="loading">⏳ Загрузка...</div>


          <OverviewView v-if="currentView === 'overview' && !loading" :overview="overview" :pct="pct"
            :cluster-name="activeClusterId" @refresh="refreshData" @navigate="switchView" />
          <PodsView v-if="currentView === 'pods' && !loading" :items="filteredPods" :search="search"
            :statusClass="statusClass" :podMetrics="podMetrics" @refresh="refreshData" @search="(v) => search = v"
            @select="openPodDetail" @yaml="(ns, n) => showYAML('pod', ns, n)" />
          <DeploymentsView v-if="currentView === 'deployments' && !loading" :items="filteredDeployments"
            :search="search" @refresh="refreshData" @search="(v) => search = v" @scale="doScale"
            @yaml="(ns, n) => showYAML('deployment', ns, n)" @restart="restartDeployment"
            @revisions="onShowRevisions" />
          <StatefulSetsView v-if="currentView === 'statefulsets' && !loading" :items="filteredStatefulSets"
            :search="search" @refresh="refreshData" @search="(v) => search = v" />
          <DaemonSetsView v-if="currentView === 'daemonsets' && !loading" :items="filteredDaemonSets" :search="search"
            @refresh="refreshData" @search="(v) => search = v" />
          <JobsView v-if="currentView === 'jobs' && !loading" :items="filteredJobs" :search="search"
            :statusClass="statusClass" @refresh="refreshData" @search="(v) => search = v" />
          <CronJobsView v-if="currentView === 'cronjobs' && !loading" :items="filteredCronJobs" :search="search"
            @refresh="refreshData" @search="(v) => search = v" />
          <ServicesView v-if="currentView === 'services' && !loading" :items="filteredServices" :search="search"
            @refresh="refreshData" @search="(v) => search = v" @select="openServiceDetail"
            @yaml="(ns, n) => showYAML('service', ns, n)" />
          <IngressesView v-if="currentView === 'ingresses' && !loading" :items="filteredIngresses" :search="search"
            @refresh="refreshData" @search="(v) => search = v" />
          <PVView v-if="currentView === 'pv' && !loading" :items="pvs" :statusClass="statusClass"
            @refresh="refreshData" />
          <PVCView v-if="currentView === 'pvc' && !loading" :items="filteredPVCs" :search="search"
            :statusClass="statusClass" @refresh="refreshData" @search="(v) => search = v" />
          <NodesView v-if="currentView === 'nodes' && !loading"
  :items="nodes"
  :metrics="nodeMetrics"
  :statusClass="statusClass"
  :cluster-name="activeClusterId"
  @refresh="refreshData"
  @yaml="(ns, n) => showYAML('node', ns, n)" />
          <EventsView v-if="currentView === 'events' && !loading" :items="events" @refresh="refreshData" />
          <ConfigMapsView v-if="currentView === 'configmaps' && !loading" :items="filteredConfigMaps" :search="search"
            @refresh="refreshData" @search="(v) => search = v" @select="openConfigMapDetail"
            @yaml="(ns, n) => showYAML('configmap', ns, n)" />
          <SecretsView v-if="currentView === 'secrets' && !loading" :items="filteredSecrets" :search="search"
            @refresh="refreshData" @search="(v) => search = v" @yaml="(ns, n) => showYAML('secret', ns, n)" />
          <ClusterSettings v-if="currentView === 'settings' && !loading" :cluster-name="activeClusterId" />
        </main>
      </template>
    </div>

    <!-- Panels -->
    <PodDetailPanel v-if="selectedPod" :pod="selectedPod" :logs="podLogs" v-model:logContainer="logContainer"
  v-model:logLines="logLines" v-model:execCmd="execCmd" :execOutput="execOutput" :statusClass="statusClass"
  v-model:pfLocalPort="pfLocalPort" v-model:pfPodPort="pfPodPort" :portForwards="portForwards"
  :cluster-name="activeClusterId"
  @close="selectedPod = null" @loadLogs="loadLogs" @runExec="runExec" @delete="deletePod" @startPF="startPortForward"
  @stopPF="stopPortForward" @openTerminal="terminalOpen = true" />

    <ServiceDetailPanel v-if="selectedService" :svc="selectedService" :portForwards="portForwards"
      @close="selectedService = null" @forward="forwardServicePort" @stopForward="stopPortForward" />
    <ConfigMapPanel v-if="selectedConfigMap" :cm="selectedConfigMap" @close="selectedConfigMap = null" />
    <RevisionsPanel v-if="revisions.length > 0" :revisions="revisions" @close="revisions = []" @rollback="onRollback" />

    <YamlPanel v-if="yamlContent" :content="yamlContent" :editing="yamlEditing"
      @close="yamlContent = ''; yamlEditing = false" @startEdit="yamlEditing = true" @apply="applyYAML"
      @cancelEdit="yamlEditing = false" @update:content="(v) => yamlContent = v" />

    <TerminalPanel v-if="terminalOpen && selectedPod" :namespace="selectedPod.namespace" :pod="selectedPod.name"
      :containers="selectedPod.containers?.map((c: any) => c.name) || []" :initialContainer="logContainer"
      @close="terminalOpen = false" />

    <!-- Add Cluster Dialog -->
    <AddClusterDialog v-if="showAddCluster" @close="showAddCluster = false" @addFile="onAddClusterFile"
      @addText="onAddClusterText" />

    <div v-if="error" class="error-toast" @click="error = ''">❌ {{ error }}</div>
    <div v-if="connecting" class="overlay">
      <div class="spinner">⏳ Подключение...</div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted } from 'vue'
import { useK8s } from './composables/useK8s'
import { useClusters } from './composables/useClusters'

import ClusterBar from './components/ClusterBar.vue'
import AddClusterDialog from './components/AddClusterDialog.vue'
import Sidebar from './components/Sidebar.vue'
import PodDetailPanel from './components/panels/PodDetailPanel.vue'
import ServiceDetailPanel from './components/panels/ServiceDetailPanel.vue'
import ConfigMapPanel from './components/panels/ConfigMapPanel.vue'
import YamlPanel from './components/panels/YamlPanel.vue'
import RevisionsPanel from './components/panels/RevisionsPanel.vue'
import TerminalPanel from './components/panels/TerminalPanel.vue'
import OverviewView from './components/views/OverviewView.vue'
import PodsView from './components/views/PodsView.vue'
import DeploymentsView from './components/views/DeploymentsView.vue'
import StatefulSetsView from './components/views/StatefulSetsView.vue'
import DaemonSetsView from './components/views/DaemonSetsView.vue'
import JobsView from './components/views/JobsView.vue'
import CronJobsView from './components/views/CronJobsView.vue'
import ServicesView from './components/views/ServicesView.vue'
import IngressesView from './components/views/IngressesView.vue'
import PVView from './components/views/PVView.vue'
import PVCView from './components/views/PVCView.vue'
import NodesView from './components/views/NodesView.vue'
import EventsView from './components/views/EventsView.vue'
import ConfigMapsView from './components/views/ConfigMapsView.vue'
import SecretsView from './components/views/SecretsView.vue'
import ClusterSettings from './components/views/ClusterSettings.vue'

const k8s = useK8s()
const clustersApi = useClusters()

// Destructure k8s
const {
  namespaces, overview, selectedNamespace,
  currentView, search, error, loading,
  filteredPods, filteredDeployments, filteredStatefulSets,
  filteredDaemonSets, filteredJobs, filteredCronJobs,
  filteredServices, filteredIngresses, filteredConfigMaps,
  filteredSecrets, filteredPVCs, pvs, nodes, events,
  selectedPod, podLogs, logContainer, logLines,
  execCmd, execOutput, selectedConfigMap,
  yamlContent, yamlEditing, revisions, portForwards,
  pfLocalPort, pfPodPort,
  selectedService,
  podMetrics, nodeMetrics,
  statusClass, pct,
  switchView, onNamespaceChange, refreshData,
  openPodDetail, loadLogs, deletePod, runExec,
  doScale, restartDeployment, loadRevisions, rollbackDeployment,
  openConfigMapDetail, showYAML, applyYAML,
  startPortForward, stopPortForward,
  openServiceDetail, forwardServicePort,
  loadNamespaces,
} = k8s

const k8sDisconnect = k8s.disconnect

// Destructure clusters
const {
  clusters,
  activeClusterId,
  showAddCluster,
  clusterError,
  loadClusters,
  connectToCluster,
  addClusterFromFile,
  addClusterFromText,
  deleteCluster,
  updateCluster,
  autoConnect,
} = clustersApi

const connecting = ref(false)
const terminalOpen = ref(false)

const activeClusterMeta = computed(() => {
  return clusters.value.find(c => c.id === activeClusterId.value) || null
})

const revisionDepNs = { ns: '', name: '' }

async function onSelectCluster(id: string) {
  if (activeClusterId.value === id) return
  try {
    connecting.value = true
    error.value = ''
    await connectToCluster(id)
    await loadNamespaces()
    currentView.value = 'overview'
    await refreshData()
  } catch (e: any) {
    error.value = 'Ошибка подключения: ' + (e.message || e)
  } finally {
    connecting.value = false
  }
}

async function onAddClusterFile(name: string, icon: string, color: string) {
  const result = await addClusterFromFile(name, icon, color)
  if (result) {
    showAddCluster.value = false
    await onSelectCluster(result.id)
  }
}

async function onAddClusterText(text: string, name: string, icon: string, color: string) {
  const result = await addClusterFromText(text, name, icon, color)
  if (result) {
    showAddCluster.value = false
    await onSelectCluster(result.id)
  }
}

async function onDeleteCluster(id: string) {
  await deleteCluster(id)
}

async function onUpdateCluster(id: string, name: string, icon: string, color: string) {
  await updateCluster(id, name, icon, color)
}

function disconnect() {
  k8sDisconnect()
  activeClusterId.value = ''
  loadClusters()
}

async function onShowRevisions(ns: string, name: string) {
  revisionDepNs.ns = ns
  revisionDepNs.name = name
  await loadRevisions(ns, name)
}

async function onRollback(rev: number) {
  await rollbackDeployment(revisionDepNs.ns, revisionDepNs.name, rev)
}

onMounted(async () => {
  await loadClusters()
  const connected = await autoConnect()
  if (connected) {
    await loadNamespaces()
    currentView.value = 'overview'
    await refreshData()
  }
})
</script>

<style src="./assets/style.css"></style>
