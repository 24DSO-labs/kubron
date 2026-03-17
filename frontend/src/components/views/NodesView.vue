<template>
  <div>
    <ContentHeader icon="🖥" title="Nodes" :count="items.length" @refresh="onRefresh" />

    <!-- Метрики нод -->
    <div v-if="metrics.length > 0" class="metrics-grid">
      <div v-for="m in metrics" :key="m.name" class="metric-card">
        <div class="metric-name">{{ m.name }}</div>
        <div class="metric-bars">
          <div class="metric-row">
            <span class="metric-label">CPU</span>
            <div class="metric-bar">
              <div class="metric-fill" :class="barColor(m.cpuPct)" :style="{ width: m.cpuPct + '%' }"></div>
            </div>
            <span class="metric-value">{{ m.cpuPct }}%</span>
            <span class="metric-raw">{{ m.cpuUsage }}</span>
          </div>
          <div class="metric-row">
            <span class="metric-label">MEM</span>
            <div class="metric-bar">
              <div class="metric-fill" :class="barColor(m.memPct)" :style="{ width: m.memPct + '%' }"></div>
            </div>
            <span class="metric-value">{{ m.memPct }}%</span>
            <span class="metric-raw">{{ m.memUsage }}</span>
          </div>
        </div>
      </div>
    </div>

    <table v-if="items.length > 0" class="table">
      <thead>
        <tr>
          <th>Name</th><th>Status</th><th>Roles</th><th>Version</th>
          <th>CPU</th><th>Memory</th><th>IP</th><th>OS</th><th>Age</th><th></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="node in items" :key="node.name">
          <td class="resource-name">{{ node.name }}</td>
          <td><span class="status-badge" :class="statusClass(node.status)">{{ node.status }}</span></td>
          <td>{{ node.roles }}</td>
          <td>{{ node.version }}</td>
          <td>{{ node.cpuCapacity }}</td>
          <td>{{ node.memoryCapacity }}</td>
          <td>{{ node.internalIP }}</td>
          <td class="os-cell">{{ node.os }}</td>
          <td>{{ node.age }}</td>
          <td><button class="btn-icon" @click="$emit('yaml', '', node.name)">📋</button></td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script lang="ts" setup>
import type { NodeInfo, NodeMetricsInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'

defineProps<{
  items: NodeInfo[]
  metrics: NodeMetricsInfo[]
  statusClass: (s: string) => string
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'yaml', ns: string, name: string): void
}>()

function onRefresh() { emit('refresh') }

function barColor(pct: number): string {
  if (pct > 85) return 'bar-red'
  if (pct > 60) return 'bar-yellow'
  return 'bar-green'
}
</script>