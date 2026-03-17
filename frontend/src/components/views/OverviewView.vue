<template>
  <div>
    <ContentHeader icon="📊" title="Cluster Overview" @refresh="$emit('refresh')" />
    <div v-if="overview" class="overview-grid">
      <div class="overview-card">
        <div class="card-icon">🖥</div>
        <div class="card-info"><div class="card-value">{{overview.readyNodes}}/{{overview.totalNodes}}</div><div class="card-label">Nodes Ready</div></div>
        <div class="card-bar"><div class="bar-fill bar-green" :style="{width:pct(overview.readyNodes,overview.totalNodes)}"></div></div>
      </div>
      <div class="overview-card">
        <div class="card-icon">📦</div>
        <div class="card-info"><div class="card-value">{{overview.runningPods}}/{{overview.totalPods}}</div><div class="card-label">Pods Running</div></div>
        <div class="card-bar"><div class="bar-fill bar-green" :style="{width:pct(overview.runningPods,overview.totalPods)}"></div></div>
      </div>
      <div class="overview-card">
        <div class="card-icon">🚀</div>
        <div class="card-info"><div class="card-value">{{overview.readyDeployments}}/{{overview.totalDeployments}}</div><div class="card-label">Deployments Ready</div></div>
        <div class="card-bar"><div class="bar-fill bar-blue" :style="{width:pct(overview.readyDeployments,overview.totalDeployments)}"></div></div>
      </div>
      <div class="overview-card">
        <div class="card-icon">🌐</div>
        <div class="card-info"><div class="card-value">{{overview.totalServices}}</div><div class="card-label">Services</div></div>
      </div>
      <div class="overview-card">
        <div class="card-icon">📁</div>
        <div class="card-info"><div class="card-value">{{overview.totalNamespaces}}</div><div class="card-label">Namespaces</div></div>
      </div>
      <div class="overview-card" :class="{'card-warning':overview.warningEvents>0}">
        <div class="card-icon">⚡</div>
        <div class="card-info"><div class="card-value">{{overview.warningEvents}}/{{overview.totalEvents}}</div><div class="card-label">Warning Events</div></div>
      </div>
      <div class="overview-card card-alert" v-if="overview.pendingPods>0">
        <div class="card-icon">⏳</div>
        <div class="card-info"><div class="card-value">{{overview.pendingPods}}</div><div class="card-label">Pending Pods</div></div>
      </div>
      <div class="overview-card card-danger" v-if="overview.failedPods>0">
        <div class="card-icon">❌</div>
        <div class="card-info"><div class="card-value">{{overview.failedPods}}</div><div class="card-label">Failed Pods</div></div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { OverviewInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'
defineProps<{ overview: OverviewInfo | null; pct: (a: number, b: number) => string }>()
defineEmits<{ (e: 'refresh'): void }>()
</script>