<template>
  <div class="logs-panel" :style="panelStyle" @mousedown="startDrag">
    <div class="panel-header" @mousedown.stop="startDrag">
      <div class="panel-title">
        <span class="panel-icon">📋</span>
        <span>{{ deployment.name }}</span>
      </div>
      <button class="btn-close" @click="$emit('close')">✕</button>
    </div>

    <div class="logs-toolbar">
      <div class="toolbar-group">
        <label>Pods:</label>
        <select v-model="selectedPodName" @change="onPodChange" class="select-pod">
          <option value="">All Pods</option>
          <option v-for="pod in deploymentPods" :key="pod.name" :value="pod.name">
            {{ pod.name }}
          </option>
        </select>
      </div>

      <div class="toolbar-group">
        <label>Containers:</label>
        <select v-model="selectedContainer" @change="loadLogs" class="select-container">
          <option value="">All Containers</option>
          <option v-for="container in availableContainers" :key="container" :value="container">
            {{ container }}
          </option>
        </select>
      </div>

      <div class="toolbar-group">
        <label>Lines:</label>
        <input v-model.number="logLines" type="number" min="10" max="10000" @change="loadLogs" class="input-lines" />
      </div>

      <div class="toolbar-group search-group">
        <input v-model="searchText" type="text" placeholder="Search in logs" class="input-search" />
      </div>

      <div class="toolbar-actions">
        <button @click="loadLogs" class="btn-action" title="Refresh">🔄</button>
        <button @click="downloadLogs" class="btn-action" title="Download">⬇️</button>
      </div>
    </div>

    <div class="logs-info">
      <span v-if="deploymentPods.length > 0">
        Displaying logs from Namespace: <strong>{{ deployment.namespace }}</strong> | 
        Pods: <strong>{{ deploymentPods.length }}</strong>
      </span>
      <span v-else>No pods found for this deployment</span>
    </div>

    <div class="logs-content">
    <div v-if="loading" class="logs-loading">⏳ Loading logs...</div>
      <pre v-else-if="displayedLogs" class="logs-text">{{ displayedLogs }}</pre>
      <div v-else class="logs-empty">No logs available</div>
    </div>

    <!-- Resize handle -->
    <div class="resize-handle" @mousedown.stop="startResize"></div>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted, watch } from 'vue'
import { GetPods, GetPodLogs } from '../../../wailsjs/go/main/App'

interface Deployment {
  namespace: string
  name: string
}

interface Pod {
  namespace: string
  name: string
  containers?: Array<{ name: string }>
}

const props = defineProps<{
  deployment: Deployment
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const deploymentPods = ref<Pod[]>([])
const selectedPodName = ref('')
const selectedContainer = ref('')
const logLines = ref(100)
const searchText = ref('')
const loading = ref(false)
const rawLogs = ref('')

// Drag and resize state
const panelX = ref(0)
const panelY = ref(0)
const panelWidth = ref(window.innerWidth * 0.6)
const panelHeight = ref(window.innerHeight * 0.6)
const isDragging = ref(false)
const isResizing = ref(false)
const dragStartX = ref(0)
const dragStartY = ref(0)
const resizeStartX = ref(0)
const resizeStartY = ref(0)
const resizeStartWidth = ref(0)
const resizeStartHeight = ref(0)

const panelStyle = computed(() => ({
  position: 'fixed' as const,
  left: `${panelX.value}px`,
  top: `${panelY.value}px`,
  width: `${panelWidth.value}px`,
  height: `${panelHeight.value}px`,
}))

function startDrag(e: MouseEvent) {
  isDragging.value = true
  dragStartX.value = e.clientX - panelX.value
  dragStartY.value = e.clientY - panelY.value
  
  const onMouseMove = (moveEvent: MouseEvent) => {
    if (!isDragging.value) return
    panelX.value = moveEvent.clientX - dragStartX.value
    panelY.value = moveEvent.clientY - dragStartY.value
  }
  
  const onMouseUp = () => {
    isDragging.value = false
    document.removeEventListener('mousemove', onMouseMove)
    document.removeEventListener('mouseup', onMouseUp)
  }
  
  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
}

function startResize(e: MouseEvent) {
  isResizing.value = true
  resizeStartX.value = e.clientX
  resizeStartY.value = e.clientY
  resizeStartWidth.value = panelWidth.value
  resizeStartHeight.value = panelHeight.value
  
  const onMouseMove = (moveEvent: MouseEvent) => {
    if (!isResizing.value) return
    const deltaX = moveEvent.clientX - resizeStartX.value
    const deltaY = moveEvent.clientY - resizeStartY.value
    
    panelWidth.value = Math.max(300, resizeStartWidth.value + deltaX)
    panelHeight.value = Math.max(200, resizeStartHeight.value + deltaY)
  }
  
  const onMouseUp = () => {
    isResizing.value = false
    document.removeEventListener('mousemove', onMouseMove)
    document.removeEventListener('mouseup', onMouseUp)
  }
  
  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
}

const availableContainers = computed(() => {
  if (!selectedPodName.value) {
    // Get all unique containers from all pods
    const containers = new Set<string>()
    deploymentPods.value.forEach(pod => {
      pod.containers?.forEach(c => containers.add(c.name))
    })
    return Array.from(containers).sort()
  } else {
    const pod = deploymentPods.value.find(p => p.name === selectedPodName.value)
    return pod?.containers?.map(c => c.name) || []
  }
})

const displayedLogs = computed(() => {
  if (!rawLogs.value) return ''
  
  if (!searchText.value) return rawLogs.value
  
  const search = searchText.value.toLowerCase()
  return rawLogs.value
    .split('\n')
    .filter(line => line.toLowerCase().includes(search))
    .join('\n')
})

onMounted(async () => {
  await loadDeploymentPods()
})

watch(() => props.deployment.name, async () => {
  selectedPodName.value = ''
  selectedContainer.value = ''
  rawLogs.value = ''
  await loadDeploymentPods()
})

async function loadDeploymentPods() {
  try {
    loading.value = true
    const allPods = await GetPods(props.deployment.namespace)
    
    // Filter pods by deployment labels
    deploymentPods.value = allPods.filter((pod: Pod) => {
      const labels = (pod as any).labels || {}
      return labels['app.kubernetes.io/instance'] === props.deployment.name ||
             labels['app'] === props.deployment.name ||
             pod.name.startsWith(props.deployment.name)
    })

    if (deploymentPods.value.length > 0 && !selectedPodName.value) {
      selectedPodName.value = deploymentPods.value[0].name
      selectedContainer.value = deploymentPods.value[0].containers?.[0]?.name || ''
    }
    
    if (selectedPodName.value || deploymentPods.value.length > 0) {
      await loadLogs()
    }
  } catch (e) {
    console.error('Error loading deployment pods:', e)
  } finally {
    loading.value = false
  }
}

async function onPodChange() {
  selectedContainer.value = ''
  const pod = deploymentPods.value.find(p => p.name === selectedPodName.value)
  if (pod?.containers?.[0]) {
    selectedContainer.value = pod.containers[0].name
  }
  await loadLogs()
}

async function loadLogs() {
  if (!selectedPodName.value && deploymentPods.value.length === 0) return
  
  try {
    loading.value = true
    
    if (selectedPodName.value) {
      // Load from specific pod
      const container = selectedContainer.value || deploymentPods.value.find(p => p.name === selectedPodName.value)?.containers?.[0]?.name || ''
      rawLogs.value = await GetPodLogs(props.deployment.namespace, selectedPodName.value, container, logLines.value)
    } else {
      // Load from all pods
      const allLogs: string[] = []
      for (const pod of deploymentPods.value) {
        const container = selectedContainer.value || pod.containers?.[0]?.name || ''
        try {
          const logs = await GetPodLogs(props.deployment.namespace, pod.name, container, logLines.value)
          if (logs) {
            allLogs.push(`\n========== ${pod.name} ==========\n`)
            allLogs.push(logs)
          }
        } catch (e) {
          allLogs.push(`\n========== ${pod.name} (error) ==========\n`)
          allLogs.push(String(e))
        }
      }
      rawLogs.value = allLogs.join('')
    }
  } catch (e) {
    rawLogs.value = `Error loading logs: ${e}`
  } finally {
    loading.value = false
  }
}

function downloadLogs() {
  const content = displayedLogs.value
  const element = document.createElement('a')
  element.setAttribute('href', 'data:text/plain;charset=utf-8,' + encodeURIComponent(content))
  element.setAttribute('download', `${props.deployment.name}-logs.txt`)
  element.style.display = 'none'
  document.body.appendChild(element)
  element.click()
  document.body.removeChild(element)
}
</script>

<style scoped>
.logs-panel {
  position: fixed;
  bottom: 0;
  right: 0;
  top: 0;
  width: 100%;
  background: #1a1b26;
  border-left: 1px solid #3b3d5c;
  display: flex;
  flex-direction: column;
  z-index: 100;
  overflow: hidden;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid #3b3d5c;
  background: #2a2b3a;
}

.panel-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 500;
  color: #e0e0e0;
}

.panel-icon {
  font-size: 18px;
}

.btn-close {
  background: none;
  border: none;
  color: #999;
  cursor: pointer;
  font-size: 20px;
  padding: 0 8px;
  transition: color 0.2s;
}

.btn-close:hover {
  color: #fff;
}

.logs-toolbar {
  display: flex;
  gap: 12px;
  padding: 12px 16px;
  border-bottom: 1px solid #3b3d5c;
  background: #232430;
  flex-wrap: wrap;
  align-items: center;
}

.toolbar-group {
  display: flex;
  align-items: center;
  gap: 6px;
}

.toolbar-group label {
  color: #999;
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
}

.select-pod,
.select-container,
.input-lines,
.input-search {
  background: #1a1b26;
  border: 1px solid #3b3d5c;
  color: #d4d4d4;
  padding: 6px 8px;
  border-radius: 3px;
  font-size: 12px;
  font-family: inherit;
}

.select-pod,
.select-container {
  min-width: 120px;
}

.input-lines {
  width: 60px;
}

.search-group {
  flex: 1;
  min-width: 200px;
}

.input-search {
  width: 100%;
  padding: 6px 10px;
}

.input-search::placeholder {
  color: #666;
}

.toolbar-actions {
  display: flex;
  gap: 6px;
}

.btn-action {
  background: none;
  border: 1px solid #3b3d5c;
  color: #999;
  cursor: pointer;
  padding: 6px 10px;
  border-radius: 3px;
  font-size: 12px;
  transition: all 0.2s;
}

.btn-action:hover {
  background: rgba(255, 255, 255, 0.1);
  color: #fff;
}

.logs-info {
  padding: 8px 16px;
  background: #232430;
  border-bottom: 1px solid #3b3d5c;
  font-size: 12px;
  color: #999;
}

.logs-content {
  flex: 1;
  overflow: auto;
  background: #0d0e12;
}

.logs-text {
  margin: 0;
  padding: 16px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
  line-height: 1.5;
  color: #a0a0a0;
  white-space: pre-wrap;
  word-wrap: break-word;
}

.logs-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #666;
  font-size: 14px;
}

.logs-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #666;
  font-size: 14px;
}

select:focus,
input:focus {
  outline: none;
  border-color: #4a5568;
  background: rgba(255, 255, 255, 0.05);
}

.panel-header {
  cursor: move;
  user-select: none;
}

.resize-handle {
  position: absolute;
  bottom: 0;
  right: 0;
  width: 20px;
  height: 20px;
  cursor: se-resize;
  background: linear-gradient(135deg, transparent 50%, #3b3d5c 50%);
  opacity: 0.5;
  transition: opacity 0.2s;
}

.resize-handle:hover {
  opacity: 1;
}
</style>
