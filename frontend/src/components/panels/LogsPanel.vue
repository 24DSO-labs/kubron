<template>
  <div class="logs-panel">
    <div class="logs-header">
      <h3>📋 Logs</h3>
      <div class="logs-info">
        <select :value="container" @change="onContainerChange" class="log-select">
          <option v-for="c in containers" :key="c" :value="c">{{ c }}</option>
        </select>
        <select :value="lines" @change="onLinesChange" class="log-select">
          <option :value="50">50 lines</option>
          <option :value="100">100 lines</option>
          <option :value="500">500 lines</option>
          <option :value="1000">1000 lines</option>
        </select>
        <button class="btn btn-sm" @click="reload" :disabled="loading">
          🔄 Reload
        </button>
        <button class="btn-close" @click="$emit('close')">✕</button>
      </div>
    </div>
    <div class="logs-container">
      <div v-if="loading" class="logs-loading">Loading logs...</div>
      <pre v-else-if="logs" class="logs-output">{{ logs }}</pre>
      <div v-else class="logs-empty">No logs available</div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import { GetPodLogs } from '../../../wailsjs/go/main/App'

const props = defineProps<{
  namespace: string
  pod: string
  containers: string[]
  initialContainer: string
}>()

defineEmits<{
  (e: 'close'): void
}>()

const container = ref(props.initialContainer)
const lines = ref(100)
const logs = ref('')
const loading = ref(false)

onMounted(() => {
  loadLogs()
})

async function loadLogs() {
  loading.value = true
  try {
    const result = await GetPodLogs(props.namespace, props.pod, container.value, lines.value)
    logs.value = result || 'No logs'
  } catch (e: any) {
    logs.value = `Error: ${e?.message || e}`
  } finally {
    loading.value = false
  }
}

async function reload() {
  await loadLogs()
}

function onContainerChange(e: Event) {
  container.value = (e.target as HTMLSelectElement).value
  loadLogs()
}

function onLinesChange(e: Event) {
  lines.value = Number((e.target as HTMLSelectElement).value)
  loadLogs()
}
</script>

<style scoped>
.logs-panel {
  position: fixed;
  bottom: 0;
  right: 0;
  width: 50%;
  height: 50%;
  display: flex;
  flex-direction: column;
  background: #1e1e1e;
  color: #d4d4d4;
  border-radius: 6px 6px 0 0;
  overflow: hidden;
  border-top: 1px solid #444;
  border-left: 1px solid #444;
  z-index: 100;
  box-shadow: -2px -2px 8px rgba(0, 0, 0, 0.5);
}

.logs-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 16px;
  border-bottom: 1px solid #333;
  background: #252526;
  flex-shrink: 0;
}

.logs-header h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  flex: 1;
}

.logs-info {
  display: flex;
  gap: 8px;
  align-items: center;
}

.log-select {
  padding: 4px 8px;
  font-size: 12px;
  background: #3c3c3c;
  color: #d4d4d4;
  border: 1px solid #555;
  border-radius: 3px;
  cursor: pointer;
}

.btn {
  padding: 4px 12px;
  font-size: 12px;
  background: #0e639c;
  color: white;
  border: none;
  border-radius: 3px;
  cursor: pointer;
}

.btn:hover {
  background: #1177bb;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-sm {
  padding: 4px 8px;
}

.btn-close {
  padding: 4px 8px;
  font-size: 12px;
  background: #444;
  color: #d4d4d4;
  border: none;
  border-radius: 3px;
  cursor: pointer;
}

.btn-close:hover {
  background: #555;
}

.logs-container {
  flex: 1;
  overflow: auto;
  padding: 8px;
}

.logs-output {
  margin: 0;
  padding: 8px;
  background: #1a1a1a;
  border-radius: 3px;
  font-size: 12px;
  line-height: 1.5;
  color: #d4d4d4;
  white-space: pre-wrap;
  word-wrap: break-word;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
}

.logs-loading,
.logs-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #666;
  font-size: 14px;
}
</style>
