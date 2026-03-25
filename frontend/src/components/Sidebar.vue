<template>
  <aside class="sidebar">
    <!-- Cluster name -->
    <div class="sidebar-cluster-name" v-if="clusterName">
      <span class="cluster-name-text">{{ clusterName }}</span>
    </div>

    <!-- Navigation Tree -->
    <nav class="nav-tree">
      <!-- Top level items -->
      <div :class="['nav-item', { active: currentView === 'overview' }]" @click="emit('navigate', 'overview')">
        <span class="nav-icon"></span> Overview
      </div>
      <!-- Nodes -->
      <div :class="['nav-item', { active: currentView === 'nodes' }]" @click="emit('navigate', 'nodes')">
        <span class="nav-icon"></span> Nodes
      </div>

      <!-- Events -->
      <div :class="['nav-item', { active: currentView === 'events' }]" @click="emit('navigate', 'events')">
        <span class="nav-icon"></span> Events
      </div>


      <!-- Workloads -->
      <div class="nav-group">
        <div class="nav-group-header" @click="toggle('workloads')">
          <span class="nav-arrow" :class="{ open: expanded.workloads }">›</span>
          <span class="nav-group-icon"></span>
          <span class="nav-group-title">Workloads</span>
        </div>
        <div v-show="expanded.workloads" class="nav-group-items">
          <div :class="['nav-item', { active: currentView === 'pods' }]" @click="emit('navigate', 'pods')">
            <span class="nav-icon"></span> Pods
          </div>
          <div :class="['nav-item', { active: currentView === 'deployments' }]" @click="emit('navigate', 'deployments')">
            <span class="nav-icon"></span> Deployments
          </div>
          <div :class="['nav-item', { active: currentView === 'daemonsets' }]" @click="emit('navigate', 'daemonsets')">
            <span class="nav-icon"></span> Daemon Sets
          </div>
          <div :class="['nav-item', { active: currentView === 'statefulsets' }]" @click="emit('navigate', 'statefulsets')">
            <span class="nav-icon"></span> Stateful Sets
          </div>
          <div :class="['nav-item', { active: currentView === 'replicasets' }]" @click="emit('navigate', 'replicasets')">
            <span class="nav-icon"></span> Replica Sets
          </div>
          <div :class="['nav-item', { active: currentView === 'jobs' }]" @click="emit('navigate', 'jobs')">
            <span class="nav-icon"></span> Jobs
          </div>
          <div :class="['nav-item', { active: currentView === 'cronjobs' }]" @click="emit('navigate', 'cronjobs')">
            <span class="nav-icon"></span> Cron Jobs
          </div>
          
        </div>
      </div>

      <!-- Config -->
      <div class="nav-group">
        <div class="nav-group-header" @click="toggle('config')">
          <span class="nav-arrow" :class="{ open: expanded.config }">›</span>
          <span class="nav-group-icon"></span>
          <span class="nav-group-title">Config</span>
        </div>
        <div v-show="expanded.config" class="nav-group-items">
          <div :class="['nav-item', { active: currentView === 'configmaps' }]" @click="emit('navigate', 'configmaps')">
            <span class="nav-icon"></span> Config Maps
          </div>
          <div :class="['nav-item', { active: currentView === 'secrets' }]" @click="emit('navigate', 'secrets')">
            <span class="nav-icon"></span> Secrets
          </div>
          <div :class="['nav-item', { active: currentView === 'resourcequotas' }]" @click="emit('navigate', 'resourcequotas')">
            <span class="nav-icon"></span> Resource Quotas
          </div>
          <div :class="['nav-item', { active: currentView === 'limitranges' }]" @click="emit('navigate', 'limitranges')">
            <span class="nav-icon"></span> Limit Ranges
          </div>
          <div :class="['nav-item', { active: currentView === 'hpa' }]" @click="emit('navigate', 'hpa')">
            <span class="nav-icon"></span> HPA
          </div>
          <div :class="['nav-item', { active: currentView === 'pdb' }]" @click="emit('navigate', 'pdb')">
            <span class="nav-icon"></span> Pod Disruption Budgets
          </div>
          <div :class="['nav-item', { active: currentView === 'priorityclasses' }]" @click="emit('navigate', 'priorityclasses')">
            <span class="nav-icon"></span> Priority Classes
          </div>
          <div :class="['nav-item', { active: currentView === 'runtimeclasses' }]" @click="emit('navigate', 'runtimeclasses')">
            <span class="nav-icon"></span> Runtime Classes
          </div>
          <div :class="['nav-item', { active: currentView === 'leases' }]" @click="emit('navigate', 'leases')">
            <span class="nav-icon"></span> Leases
          </div>
        </div>
      </div>

      <!-- Network -->
      <div class="nav-group">
        <div class="nav-group-header" @click="toggle('network')">
          <span class="nav-arrow" :class="{ open: expanded.network }">›</span>
          <span class="nav-group-icon"></span>
          <span class="nav-group-title">Network</span>
        </div>
        <div v-show="expanded.network" class="nav-group-items">
          <div :class="['nav-item', { active: currentView === 'services' }]" @click="emit('navigate', 'services')">
            <span class="nav-icon"></span> Services
          </div>
          <div :class="['nav-item', { active: currentView === 'endpoints' }]" @click="emit('navigate', 'endpoints')">
            <span class="nav-icon"></span> Endpoints
          </div>
          <div :class="['nav-item', { active: currentView === 'ingresses' }]" @click="emit('navigate', 'ingresses')">
            <span class="nav-icon"></span> Ingresses
          </div>
          <div :class="['nav-item', { active: currentView === 'ingressclasses' }]" @click="emit('navigate', 'ingressclasses')">
            <span class="nav-icon"></span> Ingress Classes
          </div>
          <div :class="['nav-item', { active: currentView === 'network-policies' }]" @click="emit('navigate', 'network-policies')">
            <span class="nav-icon"></span> Network Policies
          </div>
          <div :class="['nav-item', { active: currentView === 'portforwarding' }]" @click="emit('navigate', 'portforwarding')">
            <span class="nav-icon"></span> Port Forwarding
          </div>
        </div>
      </div>

      <!-- Storage -->
      <div class="nav-group">
        <div class="nav-group-header" @click="toggle('storage')">
          <span class="nav-arrow" :class="{ open: expanded.storage }">›</span>
          <span class="nav-group-icon"></span>
          <span class="nav-group-title">Storage</span>
        </div>
        <div v-show="expanded.storage" class="nav-group-items">
          <div :class="['nav-item', { active: currentView === 'pv' }]" @click="emit('navigate', 'pv')">
            <span class="nav-icon"></span> Persistent Volumes
          </div>
          <div :class="['nav-item', { active: currentView === 'pvc' }]" @click="emit('navigate', 'pvc')">
            <span class="nav-icon"></span> Persistent Volume Claims
          </div>
          <div :class="['nav-item', { active: currentView === 'storageclasses' }]" @click="emit('navigate', 'storageclasses')">
            <span class="nav-icon"></span> Storage Classes
          </div>
        </div>
      </div>

      <!-- Namespaces -->
      <div :class="['nav-item', { active: currentView === 'namespaces' }]" @click="emit('navigate', 'namespaces')">
        <span class="nav-icon"></span> Namespaces
      </div>

      
      <!-- Helm -->
      <div class="nav-group">
        <div class="nav-group-header" @click="toggle('helm')">
          <span class="nav-arrow" :class="{ open: expanded.helm }">›</span>
          <span class="nav-group-icon"></span>
          <span class="nav-group-title">Helm</span>
        </div>
        <div v-show="expanded.helm" class="nav-group-items">
          <div :class="['nav-item', { active: currentView === 'helm-releases' }]" @click="emit('navigate', 'helm-releases')">
            <span class="nav-icon"></span> Releases
          </div>
          <div :class="['nav-item', { active: currentView === 'helm-charts' }]" @click="emit('navigate', 'helm-charts')">
            <span class="nav-icon"></span> Charts
          </div>
        </div>
      </div>

      <!-- Access Control -->
      <div class="nav-group">
        <div class="nav-group-header" @click="toggle('access')">
          <span class="nav-arrow" :class="{ open: expanded.access }">›</span>
          <span class="nav-group-icon"></span>
          <span class="nav-group-title">Access Control</span>
        </div>
        <div v-show="expanded.access" class="nav-group-items">
          <div :class="['nav-item', { active: currentView === 'serviceaccounts' }]" @click="emit('navigate', 'serviceaccounts')">
            <span class="nav-icon"></span> Service Accounts
          </div>
          <div :class="['nav-item', { active: currentView === 'roles' }]" @click="emit('navigate', 'roles')">
            <span class="nav-icon"></span> Roles
          </div>
          <div :class="['nav-item', { active: currentView === 'rolebindings' }]" @click="emit('navigate', 'rolebindings')">
            <span class="nav-icon"></span> Role Bindings
          </div>
          <div :class="['nav-item', { active: currentView === 'clusterroles' }]" @click="emit('navigate', 'clusterroles')">
            <span class="nav-icon"></span> Cluster Roles
          </div>
          <div :class="['nav-item', { active: currentView === 'clusterrolebindings' }]" @click="emit('navigate', 'clusterrolebindings')">
            <span class="nav-icon"></span> Cluster Role Bindings
          </div>
        </div>
      </div>

      <!-- Custom Resources -->
      <div class="nav-group">
        <div class="nav-group-header" @click="toggle('crd')">
          <span class="nav-arrow" :class="{ open: expanded.crd }">›</span>
          <span class="nav-group-icon"></span>
          <span class="nav-group-title">Custom Resources</span>
        </div>
        <div v-show="expanded.crd" class="nav-group-items">
          <div :class="['nav-item', { active: currentView === 'crds' }]" @click="emit('navigate', 'crds')">
            <span class="nav-icon"></span> Definitions
          </div>
        </div>
      </div>

       <!-- Settings -->
      <div class="nav-separator"></div>
      <div :class="['nav-item', { active: currentView === 'settings' }]" @click="emit('navigate', 'settings')">
        <span class="nav-icon">⚙️</span> Settings
      </div>
<button>
  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
      d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
      d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
  </svg>
  Settings
</button>

    </nav>

    <!-- Namespace selector -->
    <div class="ns-selector">
      <label class="ns-label-text">Namespace</label>
      <select
        :key="selectedNamespace"
        :value="selectedNamespace"
        @change="emit('nsChange', ($event.target as HTMLSelectElement).value)"
        class="ns-select"
      >
        <option value="">All namespaces</option>
        <option v-for="ns in namespaces" :key="ns.name" :value="ns.name">{{ ns.name }}</option>
      </select>
    </div>
  </aside>
</template>

<script lang="ts" setup>
import { reactive } from 'vue'
import type { NamespaceInfo } from '../types'

defineProps<{
  currentView: string
  selectedNamespace: string
  namespaces: NamespaceInfo[]
  clusterName?: string
}>()

const emit = defineEmits<{
  (e: 'navigate', view: string): void
  (e: 'nsChange', ns: string): void
}>()

const expanded = reactive({
  workloads: true,
  config: false,
  network: true,
  storage: false,
  helm: false,
  access: false,
  crd: false,
})

function toggle(group: keyof typeof expanded) {
  expanded[group] = !expanded[group]
}
</script>
