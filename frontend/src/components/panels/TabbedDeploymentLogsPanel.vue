<template>
  <div class="tabbed-logs-panel" :style="panelStyle" @mousedown="startDrag">
    <!-- Header with tabs -->
    <div class="panel-header" @mousedown.stop="startDrag">
      <div class="tabs-container">
        <div class="tabs-list">
          <div
            v-for="tab in openTabs"
            :key="tab.id"
            class="tab"
            :class="{ active: activeTabId === tab.id }"
            @click="activeTabId = tab.id"
          >
            <span class="tab-icon">📋</span>
            <span class="tab-name">{{ tab.deployment.name }}</span>
            <button class="tab-close" @click.stop="closeTab(tab.id)">✕</button>
          </div>
        </div>
        <button class="btn-add-tab" @click="$emit('add-tab')" title="Add more deployments">+</button>
      </div>
      <button class="btn-close" @click="$emit('close')">✕</button>
    </div>

    <!-- Toolbar -->
    <div class="logs-toolbar">
      <div class="toolbar-group">
        <label>Pods:</label>
        <select v-model="selectedPodName" @change="onPodChange" class="select-pod">
          <option value="">All Pods</option>
          <option v-for="pod in currentDeploymentPods" :key="pod.name" :value="pod.name">
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

    <!-- Info bar -->
    <div class="logs-info">
      <span v-if="currentTab">
        Displaying logs from Namespace: <strong>{{ currentTab.deployment.namespace }}</strong> for Deployment: <strong>{{ currentTab.deployment.name }}</strong> | 
        Pods: <strong>{{ currentDeploymentPods.length }}</strong>
      </span>
    </div>

    <!-- Logs content -->
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
import { ref, computed, watch } from 'vue'
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

interface Tab {
  id: string
  deployment: Deployment
  pods: Pod[]
  selectedPodName: string
  selectedContainer: string
  rawLogs: string
  loading: boolean
}

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'add-tab'): void
}>()

const openTabs = ref<Tab[]>([])
const activeTabId = ref('')
const logLines = ref(100)
const searchText = ref('')
const pendingDeployment = ref<Deployment | null>(null)
const hasInitialized = ref(false)

// Drag and resize state
const panelX = ref(window.innerWidth * 0.2)
const panelY = ref(window.innerHeight * 0.2)
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

const currentTab = computed(() => openTabs.value.find(t => t.id === activeTabId.value))
const currentDeploymentPods = computed(() => currentTab.value?.pods || [])
const selectedPodName = computed({
  get: () => currentTab.value?.selectedPodName || '',
  set: (val) => {
    if (currentTab.value) currentTab.value.selectedPodName = val
  }
})
const selectedContainer = computed({
  get: () => currentTab.value?.selectedContainer || '',
  set: (val) => {
    if (currentTab.value) currentTab.value.selectedContainer = val
  }
})
const loading = computed(() => currentTab.value?.loading || false)
const rawLogs = computed(() => currentTab.value?.rawLogs || '')

const availableContainers = computed(() => {
  if (!selectedPodName.value) {
    const containers = new Set<string>()
    currentDeploymentPods.value.forEach(pod => {
      pod.containers?.forEach(c => containers.add(c.name))
    })
    return Array.from(containers).sort()
  } else {
    const pod = currentDeploymentPods.value.find(p => p.name === selectedPodName.value)
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

watch(activeTabId, async () => {
  if (currentTab.value && currentTab.value.rawLogs === '') {
    console.log('Watch activeTabId triggered, loading pods for:', currentTab.value.deployment)
    await loadDeploymentPods()
  }
})

// Watch for initialization
watch(
  () => hasInitialized.value,
  async (initialized) => {
    if (initialized && pendingDeployment.value && openTabs.value.length === 0) {
      console.log('Component initialized, adding pending deployment:', pendingDeployment.value)
      await addTab(pendingDeployment.value)
      pendingDeployment.value = null
    }
  }
)

async function addTab(deployment: Deployment) {
  const tabId = `deployment-logs-${deployment.namespace}-${deployment.name}-${Date.now()}`
  const newTab: Tab = {
    id: tabId,
    deployment,
    pods: [],
    selectedPodName: '',
    selectedContainer: '',
    rawLogs: '',
    loading: false,
  }
  openTabs.value.push(newTab)
  activeTabId.value = tabId
  await new Promise(resolve => setTimeout(resolve, 0))
  await loadDeploymentPods()
}

function closeTab(tabId: string) {
  openTabs.value = openTabs.value.filter(t => t.id !== tabId)
  if (activeTabId.value === tabId && openTabs.value.length > 0) {
    activeTabId.value = openTabs.value[0].id
  }
  if (openTabs.value.length === 0) {
    emit('close')
  }
}

async function loadDeploymentPods() {
  if (!currentTab.value) return
  
  try {
    currentTab.value.loading = true
    const allPods = await GetPods(currentTab.value.deployment.namespace)
    
    currentTab.value.pods = allPods.filter((pod: Pod) => {
      const labels = (pod as any).labels || {}
      return labels['app.kubernetes.io/instance'] === currentTab.value!.deployment.name ||
             labels['app'] === currentTab.value!.deployment.name ||
             pod.name.startsWith(currentTab.value!.deployment.name)
    })

    if (currentTab.value.pods.length > 0 && !currentTab.value.selectedPodName) {
      currentTab.value.selectedPodName = currentTab.value.pods[0].name
      currentTab.value.selectedContainer = currentTab.value.pods[0].containers?.[0]?.name || ''
    }
    
    if (currentTab.value.selectedPodName || currentTab.value.pods.length > 0) {
      await loadLogs()
    }
  } catch (e) {
    console.error('Error loading deployment pods:', e)
  } finally {
    if (currentTab.value) currentTab.value.loading = false
  }
}

async function onPodChange() {
  if (!currentTab.value) return
  currentTab.value.selectedContainer = ''
  const pod = currentTab.value.pods.find(p => p.name === currentTab.value!.selectedPodName)
  if (pod?.containers?.[0]) {
    currentTab.value.selectedContainer = pod.containers[0].name
  }
  await loadLogs()
}

async function loadLogs() {
  if (!currentTab.value || (!currentTab.value.selectedPodName && currentTab.value.pods.length === 0)) return
  
  try {
    currentTab.value.loading = true
    
    if (currentTab.value.selectedPodName) {
      const container = currentTab.value.selectedContainer || 
        currentTab.value.pods.find(p => p.name === currentTab.value!.selectedPodName)?.containers?.[0]?.name || ''
      currentTab.value.rawLogs = await GetPodLogs(
        currentTab.value.deployment.namespace,
        currentTab.value.selectedPodName,
        container,
        logLines.value
      )
    } else {
      const allLogs: string[] = []
      for (const pod of currentTab.value.pods) {
        const container = currentTab.value.selectedContainer || pod.containers?.[0]?.name || ''
        try {
          const logs = await GetPodLogs(currentTab.value.deployment.namespace, pod.name, container, logLines.value)
          if (logs) {
            allLogs.push(`\n========== ${pod.name} ==========\n`)
            allLogs.push(logs)
          }
        } catch (e) {
          allLogs.push(`\n========== ${pod.name} (error) ==========\n`)
          allLogs.push(String(e))
        }
      }
      currentTab.value.rawLogs = allLogs.join('')
    }
  } catch (e) {
    if (currentTab.value) currentTab.value.rawLogs = `Error loading logs: ${e}`
  } finally {
    if (currentTab.value) currentTab.value.loading = false
  }
}

function downloadLogs() {
  if (!currentTab.value) return
  const content = displayedLogs.value
  const element = document.createElement('a')
  element.setAttribute('href', 'data:text/plain;charset=utf-8,' + encodeURIComponent(content))
  element.setAttribute('download', `${currentTab.value.deployment.name}-logs.txt`)
  element.style.display = 'none'
  document.body.appendChild(element)
  element.click()
  document.body.removeChild(element)
}

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
    
    panelWidth.value = Math.max(400, resizeStartWidth.value + deltaX)
    panelHeight.value = Math.max(250, resizeStartHeight.value + deltaY)
  }
  
  const onMouseUp = () => {
    isResizing.value = false
    document.removeEventListener('mousemove', onMouseMove)
    document.removeEventListener('mouseup', onMouseUp)
  }
  
  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
}

// Initialize component and mark as ready
hasInitialized.value = true

// Expose addTab and setPendingDeployment for parent component
defineExpose({
  addTab,
  setPendingDeployment: (deployment: Deployment) => {
    console.log('setPendingDeployment called with:', deployment)
    pendingDeployment.value = deployment
    if (hasInitialized.value && openTabs.value.length === 0) {
      addTab(deployment)
      pendingDeployment.value = null
    }
  },
})
</script>

<style scoped>
.tabbed-logs-panel {
  position: fixed;
  background: #1a1b26;
  border: 1px solid #3b3d5c;
  border-radius: 4px;
  display: flex;
  flex-direction: column;
  z-index: 100;
  overflow: hidden;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0;
  border-bottom: 1px solid #3b3d5c;
  background: #2a2b3a;
  min-height: 44px;
  user-select: none;
  cursor: move;
}

.tabs-container {
  display: flex;
  align-items: center;
  gap: 4px;
  flex: 1;
  padding: 0 8px;
  overflow-x: auto;
}

.tabs-list {
  display: flex;
  gap: 4px;
  align-items: center;
}

.tab {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  background: #1e1f2e;
  border: 1px solid #3b3d5c;
  border-radius: 3px 3px 0 0;
  color: #999;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
  min-width: 140px;
}

.tab:hover {
  background: #252a3a;
}

.tab.active {
  background: #2a2b3a;
  border-bottom-color: #2a2b3a;
  color: #e0e0e0;
}

.tab-icon {
  font-size: 14px;
}

.tab-name {
  font-size: 12px;
  font-weight: 500;
}

.tab-close {
  background: none;
  border: none;
  color: #666;
  cursor: pointer;
  padding: 0 4px;
  transition: color 0.2s;
  font-size: 12px;
}

.tab-close:hover {
  color: #999;
}

.btn-add-tab {
  background: none;
  border: none;
  color: #666;
  cursor: pointer;
  padding: 0 8px;
  font-size: 14px;
  transition: color 0.2s;
}

.btn-add-tab:hover {
  color: #999;
}

.btn-close {
  background: none;
  border: none;
  color: #666;
  cursor: pointer;
  padding: 0 12px;
  font-size: 18px;
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