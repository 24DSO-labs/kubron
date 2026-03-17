<template>
  <aside class="sidebar">
    <div class="nav-section">
      <h3>Cluster</h3>
      <div :class="['nav-item',{active:currentView==='overview'}]" @click="$emit('navigate','overview')">📊 Overview</div>
      <div :class="['nav-item',{active:currentView==='nodes'}]" @click="$emit('navigate','nodes')">🖥 Nodes</div>
      <div :class="['nav-item',{active:currentView==='events'}]" @click="$emit('navigate','events')">⚡ Events</div>
    </div>
    <div class="nav-section">
      <h3>Workloads</h3>
      <div :class="['nav-item',{active:currentView==='pods'}]" @click="$emit('navigate','pods')">📦 Pods</div>
      <div :class="['nav-item',{active:currentView==='deployments'}]" @click="$emit('navigate','deployments')">🚀 Deployments</div>
      <div :class="['nav-item',{active:currentView==='statefulsets'}]" @click="$emit('navigate','statefulsets')">📚 StatefulSets</div>
      <div :class="['nav-item',{active:currentView==='daemonsets'}]" @click="$emit('navigate','daemonsets')">🔄 DaemonSets</div>
      <div :class="['nav-item',{active:currentView==='jobs'}]" @click="$emit('navigate','jobs')">⚙️ Jobs</div>
      <div :class="['nav-item',{active:currentView==='cronjobs'}]" @click="$emit('navigate','cronjobs')">🕐 CronJobs</div>
    </div>
    <div class="nav-section">
      <h3>Network</h3>
      <div :class="['nav-item',{active:currentView==='services'}]" @click="$emit('navigate','services')">🌐 Services</div>
      <div :class="['nav-item',{active:currentView==='ingresses'}]" @click="$emit('navigate','ingresses')">🔀 Ingresses</div>
    </div>
    <div class="nav-section">
      <h3>Storage</h3>
      <div :class="['nav-item',{active:currentView==='pv'}]" @click="$emit('navigate','pv')">💾 PV</div>
      <div :class="['nav-item',{active:currentView==='pvc'}]" @click="$emit('navigate','pvc')">📎 PVC</div>
    </div>
    <div class="nav-section">
      <h3>Config</h3>
      <div :class="['nav-item',{active:currentView==='configmaps'}]" @click="$emit('navigate','configmaps')">📄 ConfigMaps</div>
      <div :class="['nav-item',{active:currentView==='secrets'}]" @click="$emit('navigate','secrets')">🔒 Secrets</div>
    </div>
    <div class="nav-section">
      <h3>Namespace</h3>
      <select :value="selectedNamespace" @change="$emit('nsChange', ($event.target as HTMLSelectElement).value)" class="ns-select">
        <option value="">All namespaces</option>
        <option v-for="ns in namespaces" :key="ns.name" :value="ns.name">{{ ns.name }}</option>
      </select>
    </div>
  </aside>
</template>

<script lang="ts" setup>
import type { NamespaceInfo } from '../types'
defineProps<{ currentView: string; selectedNamespace: string; namespaces: NamespaceInfo[] }>()
defineEmits<{ (e: 'navigate', view: string): void; (e: 'nsChange', ns: string): void }>()
</script>