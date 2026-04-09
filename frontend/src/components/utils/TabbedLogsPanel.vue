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
            <span class="tab-name">{{ tab.resourceType }}/{{ tab.resourceName }}</span>
            <button class="tab-close" @click.stop="closeTab(tab.id)">✕</button>
          </div>
        </div>
        <button class="btn-add-tab" @click="$emit('add-tab')" title="Add more resources">+</button>
      </div>
      <button class="btn-close" @click="$emit('close')">✕</button>
    </div>

    <!-- Toolbar -->
    <div class="logs-toolbar">
      <div class="toolbar-group">
        <label>Pods:</label>
        <select v-model="selectedPodName" @change="onPodChange" class="select-pod">
          <option value="">All Pods</option>
          <option v-for="pod in currentResourcePods" :key="pod.name" :value="pod.name">
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
        <button 
          @click="showTimestamp = !showTimestamp" 
          class="btn-action" 
          :class="{ active: showTimestamp }"
          title="Show Timestamp"
        >
          <img :src="timestampIcon" alt="Timestamp" class="icon-svg" />
        </button>
        <button 
          @click="wrapLines = !wrapLines" 
          class="btn-action" 
          :class="{ active: wrapLines }"
          title="Wrap Lines"
        >
          <img :src="wrapLinesIcon" alt="Wrap Lines" class="icon-svg" />
        </button>
        <button @click="loadLogs" class="btn-action btn-refresh" title="Refresh">
          <img :src="restartIcon" alt="Refresh" class="icon-svg" />
        </button>
        <button @click="downloadLogs" class="btn-action" title="Download">
          <img :src="downloadIcon" alt="Download" class="icon-svg" />
        </button>
      </div>
    </div>

    <!-- Info bar -->
    <div class="logs-info">
      <span v-if="currentTab">
        Displaying logs from Namespace: <strong>{{ currentTab.namespace }}</strong> for {{ currentTab.resourceType }}: <strong>{{ currentTab.resourceName }}</strong> | 
        Pods: <strong>{{ currentResourcePods.length }}</strong>
      </span>
    </div>

    <!-- Logs content -->
    <div class="logs-content">
      <div v-if="loading" class="logs-loading">⏳ Loading logs...</div>
      <pre v-else-if="displayedLogs" class="logs-text" :class="{ 'no-wrap': !wrapLines }" v-html="searchText ? highlightedLogs : displayedLogs"></pre>
      <div v-else class="logs-empty">No logs available</div>
    </div>

    <!-- Resize handle -->
    <div class="resize-handle" @mousedown.stop="startResize"></div>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, watch, onMounted } from 'vue'
import { GetPods, GetPodLogs } from '../../../wailsjs/go/main/App'
import { globalTabbedLogsPanel } from '../../composables/useTabbedLogsPanel'
import type { PodInfo } from '../../types/index'
import restartIcon from '../../assets/icons/restart.svg'
import downloadIcon from '../../assets/icons/download.svg'
import timestampIcon from '../../assets/icons/timestamp.svg'
import wrapLinesIcon from '../../assets/icons/wrap-lines.svg'

interface MinimalPod {
  name: string
  containers?: Array<{ name: string }>
}

interface Tab {
  id: string
  resourceType: string
  resourceName: string
  namespace: string
  pods: MinimalPod[]
  selectedPodName?: string
  selectedContainer?: string
  rawLogs?: string
  loading: boolean
}

const { tabs: openTabs, activeTabId, closeTabbedLogs } = globalTabbedLogsPanel

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'add-tab'): void
}>()

const logLines = ref(100)
const searchText = ref('')
const currentMatchIndex = ref(0)
const showTimestamp = ref(false)
const wrapLines = ref(true)
const sidebarWidth = ref(200) // Default fallback

onMounted(() => {
  const sidebar = document.querySelector('.sidebar') as HTMLElement | null
  if (sidebar) {
    sidebarWidth.value = sidebar.offsetWidth
  }
})

const hasInitialized = ref(false)

const panelX = ref(276) // Slight adjustment to align with sidebar border
const panelY = ref(window.innerHeight - 300)
const panelWidth = ref(window.innerWidth - sidebarWidth.value - 80)
const panelHeight = ref(300)
const isDragging = ref(false)
const isResizing = ref(false)
const dragStartX = ref(0)
const dragStartY = ref(0)
const resizeStartX = ref(0)
const resizeStartY = ref(0)
const resizeStartWidth = ref(0)
const resizeStartHeight = ref(0)
let autoRefreshInterval: ReturnType<typeof setInterval> | null = null

const panelStyle = computed(() => ({
  position: 'fixed' as const,
  left: `${panelX.value}px`,
  top: `${panelY.value}px`,
  width: `${panelWidth.value}px`,
  height: `${panelHeight.value}px`,
}))

const currentTab = computed(() => openTabs.value.find(t => t.id === activeTabId.value))
const currentResourcePods = computed(() => currentTab.value?.pods || [])
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
    currentResourcePods.value.forEach(pod => {
      pod.containers?.forEach(c => containers.add(c.name))
    })
    return Array.from(containers).sort()
  } else {
    const pod = currentResourcePods.value.find(p => p.name === selectedPodName.value)
    return pod?.containers?.map(c => c.name) || []
  }
})

const matchCount = computed(() => {
  if (!searchText.value || !rawLogs.value) return 0
  const search = searchText.value.toLowerCase()
  const lines = rawLogs.value.split('\n')
  return lines.filter(line => line.toLowerCase().includes(search)).length
})

const displayedLogs = computed(() => {
  if (!rawLogs.value) return ''
  
  let logs = rawLogs.value
  
  // Remove or keep timestamps based on showTimestamp flag
  if (!showTimestamp.value) {
    // Remove timestamps from the beginning of lines (format: YYYY-MM-DDTHH:MM:SS.nnnZ or YYYY-MM-DDTHH:MM:SS.nnnZ+HH:MM)
    logs = logs
      .split('\n')
      .map(line => {
        if (!line.trim()) return line
        // Remove timestamp prefix if present
        const timestampRegex = /^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z[+-]?\d{0,2}:?\d{0,2})\s+(.*)$/
        const match = line.match(timestampRegex)
        if (match && match[2]) {
          return match[2]
        }
        return line
      })
      .join('\n')
  } else {
    // Add formatted timestamps if enabled
    logs = logs
      .split('\n')
      .map(line => {
        if (!line.trim()) return line
        // Check if line already has timestamp at the beginning
        const timestampRegex = /^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z[+-]?\d{0,2}:?\d{0,2})\s+(.*)$/
        const match = line.match(timestampRegex)
        if (match && match[1]) {
          const fullTimestamp = match[1]
          const message = match[2]
          // Keep the full timestamp
          return `[${fullTimestamp}] ${message}`
        }
        return line
      })
      .join('\n')
  }
  
  // Apply search filter
  if (!searchText.value) return logs
  
  const search = searchText.value.toLowerCase()
  return logs
    .split('\n')
    .filter(line => line.toLowerCase().includes(search))
    .join('\n')
})

const highlightedLogs = computed(() => {
  if (!rawLogs.value) return rawLogs.value
  
  if (!searchText.value) return rawLogs.value
  
  const search = searchText.value
  const escapedSearch = escapeRegex(search)
  const regex = new RegExp(`(${escapedSearch})`, 'gi')
  return rawLogs.value.replace(regex, '<mark class="search-highlight">$1</mark>')
})

watch(activeTabId, async () => {
  if (currentTab.value && currentTab.value.rawLogs === '') {
    await loadResourcePods()
  }
  // Reset auto-refresh when switching tabs
  stopAutoRefresh()
  startAutoRefresh()
})

// Watch for initialization
watch(
  () => hasInitialized.value,
  async (initialized) => {
    if (initialized && openTabs.value.length > 0) {
      await loadResourcePods()
      startAutoRefresh()
    }
  }
)

function startAutoRefresh() {
  // Clear existing interval if any
  stopAutoRefresh()
  // Set up auto-refresh every 45 seconds
  autoRefreshInterval = setInterval(() => {
    loadLogs()
  }, 45000) // 45 seconds
}

function stopAutoRefresh() {
  if (autoRefreshInterval) {
    clearInterval(autoRefreshInterval)
    autoRefreshInterval = null
  }
}

// Clean up on component unmount
onMounted(() => {
  const sidebar = document.querySelector('.sidebar') as HTMLElement | null
  if (sidebar) {
    sidebarWidth.value = sidebar.offsetWidth
  }
})

watch(() => openTabs.value.length, (newLength) => {
  if (newLength === 0) {
    stopAutoRefresh()
  }
})

function closeTab(tabId: string) {
  const index = openTabs.value.findIndex((t: Tab) => t.id === tabId)
  if (index !== -1) {
    openTabs.value.splice(index, 1)
  }
  if (activeTabId.value === tabId && openTabs.value.length > 0) {
    activeTabId.value = openTabs.value[0].id
  }
  if (openTabs.value.length === 0) {
    stopAutoRefresh()
    closeTabbedLogs()
    emit('close')
  }
}

async function loadResourcePods() {
  const tab = currentTab.value
  if (!tab) return
  
  try {
    tab.loading = true
    const allPods = await GetPods(tab.namespace)
    
    let filteredPods: MinimalPod[] = []
    const resourceName = tab.resourceName

    switch (tab.resourceType.toLowerCase()) {
      case 'deployment':
        filteredPods = allPods.filter((pod: any) => {
          const podLabels = pod.labels || {}
          return podLabels['app.kubernetes.io/instance'] === resourceName ||
                 podLabels['app'] === resourceName ||
                 pod.name.startsWith(resourceName)
        })
        break
      case 'job':
        filteredPods = allPods.filter((pod: any) => pod.name.startsWith(resourceName + '-'))
        break
      case 'pod':
        filteredPods = allPods.filter((pod: any) => pod.name === resourceName)
        break
      // Add more cases as needed
      default:
        console.warn('Unsupported resource type:', tab.resourceType)
        filteredPods = []
    }

    tab.pods = filteredPods

    if (tab.pods.length > 0 && !tab.selectedPodName) {
      tab.selectedPodName = tab.pods[0].name
      tab.selectedContainer = tab.pods[0].containers?.[0]?.name || ''
    }
    
    if (tab.selectedPodName || tab.pods.length > 0) {
      await loadLogs()
    }
  } catch (e) {
    console.error('Error loading resource pods:', e)
    tab.rawLogs = `Error loading pods: ${e}`
  } finally {
    tab.loading = false
  }
}

async function onPodChange() {
  const tab = currentTab.value
  if (!tab) return
  tab.selectedContainer = ''
  const pod = tab.pods.find((p: MinimalPod) => p.name === tab.selectedPodName)
  if (pod?.containers?.[0]) {
    tab.selectedContainer = pod.containers[0].name
  }
  await loadLogs()
}

async function loadLogs() {
  const tab = currentTab.value
  if (!tab || (!tab.selectedPodName && tab.pods.length === 0)) return
  
  try {
    tab.loading = true
    
    if (tab.selectedPodName) {
      const container = tab.selectedContainer || 
        tab.pods.find((p: MinimalPod) => p.name === tab.selectedPodName)?.containers?.[0]?.name || ''
      tab.rawLogs = await GetPodLogs(
        tab.namespace,
        tab.selectedPodName,
        container,
        logLines.value
      )
    } else {
      const allLogs: string[] = []
      for (const pod of tab.pods) {
        const container = tab.selectedContainer || pod.containers?.[0]?.name || ''
        try {
          const logs = await GetPodLogs(tab.namespace, pod.name, container, logLines.value)
          if (logs) {
            allLogs.push(`\n========== ${pod.name} ==========\n`)
            allLogs.push(logs)
          }
        } catch (e) {
          allLogs.push(`\n========== ${pod.name} (error) ==========\n`)
          allLogs.push(String(e))
        }
      }
      tab.rawLogs = allLogs.join('')
    }
  } catch (e) {
    tab.rawLogs = `Error loading logs: ${e}`
  } finally {
    tab.loading = false
  }
}

function downloadLogs() {
  if (!currentTab.value) return
  const content = displayedLogs.value
  const element = document.createElement('a')
  element.setAttribute('href', 'data:text/plain;charset=utf-8,' + encodeURIComponent(content))
  element.setAttribute('download', `${currentTab.value.resourceType}-${currentTab.value.resourceName}-logs.txt`)
  element.style.display = 'none'
  document.body.appendChild(element)
  element.click()
  document.body.removeChild(element)
}

function escapeRegex(str: string): string {
  return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function goToNextMatch() {
  if (matchCount.value === 0) return
  currentMatchIndex.value = (currentMatchIndex.value + 1) % matchCount.value
}

function goToPrevMatch() {
  if (matchCount.value === 0) return
  currentMatchIndex.value = (currentMatchIndex.value - 1 + matchCount.value) % matchCount.value
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
.select-container {
  min-width: 120px;
  background: #1a1b26;
  color: #d4d4d4;
}

.select-pod option,
.select-container option {
  background: #232430;
  color: #d4d4d4;
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

.btn-action.active {
  background: rgba(79, 99, 255, 0.2);
  border-color: #4f63ff;
  color: #4f63ff;
}

.btn-action.active:hover {
  background: rgba(79, 99, 255, 0.3);
}

.btn-refresh {
  padding: 6px 8px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.icon-svg {
  width: 14px;
  height: 14px;
  
  stroke-width: 2;
  filter: invert(1);
}

.logs-info {
  padding: 8px 16px;
  background: #232430;
  border-bottom: 1px solid #3b3d5c;
  font-size: 12px;
  color: #d4d4d4;
  text-align: left;
}

.search-bar {
  padding: 8px 16px;
  background: #1f2028;
  border-bottom: 1px solid #3b3d5c;
  display: flex;
  align-items: center;
}

.search-input-group {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.input-search-main {
  flex: 1;
  background: #2a2b3a;
  border: 1px solid #3b3d5c;
  color: #d4d4d4;
  padding: 8px 12px;
  border-radius: 4px;
  font-size: 12px;
  font-family: inherit;
}

.input-search-main:focus {
  outline: none;
  border-color: #4a5568;
  box-shadow: 0 0 0 2px rgba(74, 85, 104, 0.2);
}

.input-search-main::placeholder {
  color: #666;
}

.search-controls {
  display: flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
}

.search-nav-btn {
  background: #2a2b3a;
  border: 1px solid #3b3d5c;
  color: #999;
  cursor: pointer;
  padding: 6px 8px;
  border-radius: 3px;
  font-size: 12px;
  transition: all 0.2s;
  min-width: 28px;
  text-align: center;
}

.search-nav-btn:hover {
  background: rgba(255, 255, 255, 0.05);
  color: #d4d4d4;
  border-color: #4a5568;
}

.search-count {
  color: #999;
  font-size: 11px;
  padding: 0 6px;
  min-width: 40px;
  text-align: center;
}

.search-close-btn {
  background: none;
  border: none;
  color: #666;
  cursor: pointer;
  padding: 4px 6px;
  font-size: 14px;
  transition: color 0.2s;
}

.search-close-btn:hover {
  color: #d4d4d4;
}

.search-highlight {
  background-color: #7a5e00;
  color: #ffd700;
  font-weight: bold;
}

.logs-content {
  flex: 1;
  overflow: auto;
  background: #1a1b26;
}

.logs-text {
  margin: 0;
  padding: 16px;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 12px;
  line-height: 1.5;
  color: #d4d4d4;
  white-space: pre-wrap;
  word-wrap: break-word;
  text-align: left;
  word-break: break-word;
}

.logs-text.no-wrap {
  white-space: pre;
  word-wrap: normal;
  word-break: normal;
  overflow-x: auto;
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
