<template>
  <div class="detail-panel">
    <div class="detail-header">
      <h3>📦 {{ pod.name }}</h3>
      <button class="btn-close" @click="emit('close')">✕</button>
    </div>
    <div class="detail-meta">
      <div class="meta-row"><span class="meta-label">Namespace:</span> {{ pod.namespace }}</div>
      <div class="meta-row">
        <span class="meta-label">Status:</span>
        <span class="status-badge" :class="statusClass(pod.status)">{{ pod.status }}</span>
      </div>
      <div class="meta-row"><span class="meta-label">Node:</span> {{ pod.node }}</div>
      <div class="meta-row"><span class="meta-label">IP:</span> {{ pod.ip }}</div>
      <div class="meta-row"><span class="meta-label">Age:</span> {{ pod.age }}</div>
    </div>

    <!-- ===== POD METRICS ===== -->
    <div v-if="clusterName" class="detail-section">
      <div class="metrics-toggle" @click="showMetrics = !showMetrics">
        <h4>📈 Metrics</h4>
        <span class="toggle-arrow" :class="{ open: showMetrics }">›</span>
      </div>
      <div v-if="showMetrics" class="pod-metrics-container">
        <PodMetricsPanel
          :cluster-name="clusterName"
          :namespace="pod.namespace"
          :pod-name="pod.name"
        />
      </div>
    </div>

    <div v-if="pod.containers && pod.containers.length > 0" class="detail-section">
      <h4>Containers</h4>
      <div v-for="c in pod.containers" :key="c.name" class="container-item">
        <span class="container-name">{{ c.name }}</span>
        <span class="status-badge" :class="statusClass(c.state)">{{ c.state }}</span>
        <span class="container-image">{{ c.image }}</span>
      </div>
    </div>

    <div v-if="pod.containers && pod.containers.length > 0" class="detail-section">
      <div class="logs-header">
        <h4>Logs</h4>
        <select :value="logContainer" @change="onLogContainerChange" class="log-select">
          <option v-for="c in pod.containers" :key="c.name" :value="c.name">{{ c.name }}</option>
        </select>
        <select :value="logLines" @change="onLogLinesChange" class="log-select">
          <option :value="50">50</option>
          <option :value="100">100</option>
          <option :value="500">500</option>
          <option :value="1000">1000</option>
        </select>
        <button @click="emit('loadLogs')" class="btn btn-sm">🔄</button>
      </div>
      <pre class="log-output">{{ logs || 'Нажмите 🔄 для загрузки' }}</pre>
    </div>

    <div class="detail-section">
      <div class="exec-header"><h4>🖥 Exec</h4></div>
      <div class="exec-input-row">
        <input :value="execCmd" @input="onExecCmdChange" @keyup.enter="emit('runExec')" placeholder="ls -la /tmp" class="exec-input" />
        <button @click="emit('runExec')" class="btn btn-sm">▶ Run</button>
      </div>
      <pre v-if="execOutput" class="exec-output">{{ execOutput }}</pre>
    </div>

    <div class="detail-section">
      <button class="btn btn-terminal" @click="emit('openTerminal')">
        🖥 Open Terminal
      </button>
    </div>

    <div class="detail-section pf-section">
      <h4>🌍 Port Forward</h4>
      <div class="pf-form">
        <label>Local:</label>
        <input type="number" :value="pfLocalPort" @input="onLocalPortChange" class="pf-input" />
        <label>Pod:</label>
        <input type="number" :value="pfPodPort" @input="onPodPortChange" class="pf-input" />
        <button @click="emit('startPF')" class="btn btn-sm">▶ Start</button>
      </div>
      <div v-if="portForwards.length > 0" class="pf-list">
        <div v-for="s in portForwards" :key="s.id" class="pf-item">
          <span class="pf-info">localhost:{{ s.localPort }} → {{ s.pod }}:{{ s.podPort }}</span>
          <span class="status-badge" :class="s.status === 'active' ? 'status-running' : 'status-pending'">{{ s.status }}</span>
          <button class="btn-icon" @click="emit('stopPF', s.id)">⏹</button>
        </div>
      </div>
    </div>

    <div class="detail-actions">
      <button class="btn btn-danger" @click="emit('delete')">🗑 Удалить Pod</button>
    </div>
  </div>
</template>
<script lang="ts" setup>
import { ref } from 'vue'
import type { PortForwardSession } from '../../types'
import PodMetricsPanel from '../metrics/PodMetricsPanel.vue'

const props = defineProps<{
  pod: any
  logs: string
  logContainer: string
  logLines: number
  execCmd: string
  execOutput: string
  statusClass: (s: string) => string
  pfLocalPort: number
  pfPodPort: number
  portForwards: PortForwardSession[]
  clusterName?: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'loadLogs'): void
  (e: 'runExec'): void
  (e: 'delete'): void
  (e: 'update:logContainer', val: string): void
  (e: 'update:logLines', val: number): void
  (e: 'update:execCmd', val: string): void
  (e: 'update:pfLocalPort', val: number): void
  (e: 'update:pfPodPort', val: number): void
  (e: 'startPF'): void
  (e: 'stopPF', id: string): void
  (e: 'openTerminal'): void
}>()

const showMetrics = ref(false)

function onLogContainerChange(e: Event) {
  emit('update:logContainer', (e.target as HTMLSelectElement).value)
}

function onLogLinesChange(e: Event) {
  emit('update:logLines', Number((e.target as HTMLSelectElement).value))
}

function onExecCmdChange(e: Event) {
  emit('update:execCmd', (e.target as HTMLInputElement).value)
}

function onLocalPortChange(e: Event) {
  emit('update:pfLocalPort', Number((e.target as HTMLInputElement).value))
}

function onPodPortChange(e: Event) {
  emit('update:pfPodPort', Number((e.target as HTMLInputElement).value))
}
</script>
