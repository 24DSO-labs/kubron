<template>
  <div class="tabbed-panel" :style="panelStyle" @mousedown="startDrag">
    <!-- Header with tabs -->
    <div class="panel-header" @mousedown.stop="startDrag">
      <div class="tabs-container">
        <div class="tabs-list">
          <div
            v-for="tab in tabs"
            :key="tab.id"
            class="tab"
            :class="{ active: activeTabId === tab.id }"
            @click="activeTabId = tab.id"
          >
            <span class="tab-icon">{{ tab.icon }}</span>
            <span class="tab-name">{{ tab.title }}</span>
            <button class="tab-close" @click.stop="closeTab(tab.id)">✕</button>
          </div>
        </div>
      </div>
      <button class="btn-close" @click="closeAllTabs">✕</button>
    </div>

    <!-- Content area -->
    <div class="panel-content">
      <template v-for="tab in tabs" :key="tab.id">
        <!-- Logs View -->
        <div v-if="tab.type === 'logs' && activeTabId === tab.id" class="logs-view">
          <!-- Logs toolbar -->
          <div class="logs-toolbar">
            <div class="toolbar-group">
              <label>Pods:</label>
              <select 
                :value="(tab as LogsTab).selectedPodName" 
                @change="onPodChange($event, tab as LogsTab)" 
                class="select-pod"
              >
                <option value="">All Pods</option>
                <option v-for="pod in (tab as LogsTab).pods" :key="pod.name" :value="pod.name">
                  {{ pod.name }}
                </option>
              </select>
            </div>

            <div class="toolbar-group">
              <label>Containers:</label>
              <select 
                :value="(tab as LogsTab).selectedContainer" 
                @change="onContainerChange($event, tab as LogsTab)" 
                class="select-container"
              >
                <option value="">All Containers</option>
                <option v-for="container in getAvailableContainers(tab as LogsTab)" :key="container" :value="container">
                  {{ container }}
                </option>
              </select>
            </div>

            <div class="toolbar-group">
              <label>Lines:</label>
              <input v-model.number="logLines" type="number" min="10" max="10000" @change="loadLogs(tab as LogsTab)" class="input-lines" />
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
              <button @click="loadLogs(tab as LogsTab)" class="btn-action btn-refresh" title="Refresh">
                <img :src="restartIcon" alt="Refresh" class="icon-svg" />
              </button>
              <button @click="downloadLogs(tab as LogsTab)" class="btn-action" title="Download">
                <img :src="downloadIcon" alt="Download" class="icon-svg" />
              </button>
            </div>
          </div>

          <!-- Logs info bar -->
          <div class="logs-info">
            <span>
              Namespace: <strong>{{ tab.namespace }}</strong> | 
              {{ tab.resourceType }}: <strong>{{ tab.resourceName }}</strong> | 
              Pods: <strong>{{ (tab as LogsTab).pods.length }}</strong>
            </span>
          </div>

          <!-- Logs content -->
          <div class="logs-content">
            <div v-if="(tab as LogsTab).loading" class="logs-loading">⏳ Loading logs...</div>
            <pre 
              v-else-if="(tab as LogsTab).rawLogs" 
              class="logs-text" 
              :class="{ 'no-wrap': !wrapLines }" 
              v-html="searchText ? getHighlightedLogs(tab as LogsTab) : getDisplayedLogs(tab as LogsTab)"
            ></pre>
            <div v-else class="logs-empty">No logs available</div>
          </div>
        </div>

        <!-- YAML Editor View -->
        <div v-else-if="tab.type === 'yaml' && activeTabId === tab.id" class="yaml-view">
          <div class="editor-header">
            <div class="editor-title">
              <span class="editor-icon">✏️</span>
              <span class="editor-resource">{{ tab.resourceType }} <strong>{{ tab.resourceName }}</strong></span>
              <span class="editor-namespace">in namespace <a href="#" class="ns-link">{{ tab.namespace }}</a></span>
            </div>
            <div class="editor-actions">
              <button class="btn btn-sm btn-save" @click="saveYaml(tab as YamlTab)" title="Apply changes">💾 Save</button>
            </div>
          </div>
          <div :ref="el => setEditorContainer(tab.id, el as HTMLDivElement)" class="editor-container"></div>
        </div>
      </template>
    </div>

    <!-- Resize handle -->
    <div class="resize-handle" @mousedown.stop="startResize"></div>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import * as monaco from 'monaco-editor'
import { GetPods, GetPodLogs } from '../../../wailsjs/go/main/App'
import { globalTabbedPanel } from '../../composables/useTabbedPanel'
import type { LogsTab, YamlTab, Tab } from '../../composables/useTabbedPanel'

// Import icons
import restartIcon from '../../assets/icons/restart.svg'
import downloadIcon from '../../assets/icons/download.svg'
import timestampIcon from '../../assets/icons/timestamp.svg'
import wrapLinesIcon from '../../assets/icons/wrap-lines.svg'

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'apply-yaml', tabId: string): void
}>()

// Global state - используем globalTabbedPanel напрямую
const tabs = computed(() => globalTabbedPanel.tabs.value)
const activeTabId = computed({
  get: () => globalTabbedPanel.activeTabId.value,
  set: (val) => { globalTabbedPanel.activeTabId.value = val }
})

// Panel positioning
const sidebarWidth = ref(200)
const panelX = ref(276)
const panelY = ref(window.innerHeight - 400)
const panelWidth = ref(window.innerWidth - sidebarWidth.value - 80)
const panelHeight = ref(400)
const isDragging = ref(false)
const isResizing = ref(false)
const dragStartX = ref(0)
const dragStartY = ref(0)
const resizeStartX = ref(0)
const resizeStartY = ref(0)
const resizeStartWidth = ref(0)
const resizeStartHeight = ref(0)

// Logs specific state
const logLines = ref(100)
const searchText = ref('')
const showTimestamp = ref(false)
const wrapLines = ref(true)
let autoRefreshInterval: ReturnType<typeof setInterval> | null = null

// YAML editors map
const editors = new Map<string, monaco.editor.IStandaloneCodeEditor>()
const editorContainers = new Map<string, HTMLDivElement>()

const panelStyle = computed(() => ({
  position: 'fixed' as const,
  left: `${panelX.value}px`,
  top: `${panelY.value}px`,
  width: `${panelWidth.value}px`,
  height: `${panelHeight.value}px`,
}))

// Используем методы из globalTabbedPanel
const closeTab = (tabId: string) => globalTabbedPanel.closeTab(tabId)
const closeAllTabs = () => globalTabbedPanel.closeAllTabs()

// Lifecycle
onMounted(() => {
  const sidebar = document.querySelector('.sidebar') as HTMLElement | null
  if (sidebar) {
    sidebarWidth.value = sidebar.offsetWidth
  }
  
  // Initialize Monaco editor styles
  const style = document.createElement('style')
  style.textContent = `
    @font-face {
      font-family: 'codicon';
      src: url('${new URL('monaco-editor/esm/vs/base/browser/ui/codicons/codicon/codicon.ttf', import.meta.url).href}') format('truetype');
    }
  `
  document.head.appendChild(style)

  // Initialize first tab if exists
  if (tabs.value.length > 0 && activeTabId.value) {
    const activeTab = tabs.value.find(t => t.id === activeTabId.value)
    if (activeTab?.type === 'logs') {
      loadResourcePods(activeTab as LogsTab)
      startAutoRefresh()
    }
  }
})

onBeforeUnmount(() => {
  stopAutoRefresh()
  editors.forEach(editor => editor.dispose())
  editors.clear()
  editorContainers.clear()
})

// Watchers
watch(activeTabId, async (newId) => {
  const tab = tabs.value.find(t => t.id === newId)
  if (!tab) return
  
  if (tab.type === 'logs' && !(tab as LogsTab).rawLogs) {
    await loadResourcePods(tab as LogsTab)
    startAutoRefresh()
  } else if (tab.type === 'yaml') {
    await nextTick()
    initializeEditor(tab as YamlTab)
  }
})

// Helper functions
function getAvailableContainers(tab: LogsTab): string[] {
  if (!tab.selectedPodName) {
    const containers = new Set<string>()
    tab.pods.forEach(pod => {
      pod.containers?.forEach(c => containers.add(c.name))
    })
    return Array.from(containers).sort()
  } else {
    const pod = tab.pods.find(p => p.name === tab.selectedPodName)
    return pod?.containers?.map(c => c.name) || []
  }
}

function getDisplayedLogs(tab: LogsTab): string {
  if (!tab.rawLogs) return ''
  
  let logs = tab.rawLogs
  
  // Process timestamps
  if (!showTimestamp.value) {
    logs = logs
      .split('\n')
      .map(line => {
        if (!line.trim()) return line
        const timestampRegex = /^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z[+-]?\d{0,2}:?\d{0,2})\s+(.*)$/
        const match = line.match(timestampRegex)
        if (match && match[2]) {
          return match[2]
        }
        return line
      })
      .join('\n')
  } else {
    logs = logs
      .split('\n')
      .map(line => {
        if (!line.trim()) return line
        const timestampRegex = /^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z[+-]?\d{0,2}:?\d{0,2})\s+(.*)$/
        const match = line.match(timestampRegex)
        if (match && match[1]) {
          const fullTimestamp = match[1]
          const message = match[2]
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
}

function getHighlightedLogs(tab: LogsTab): string {
  if (!tab.rawLogs || !searchText.value) return tab.rawLogs || ''
  
  const logs = getDisplayedLogs(tab)
  const search = searchText.value
  const escapedSearch = escapeRegex(search)
  const regex = new RegExp(`(${escapedSearch})`, 'gi')
  return logs.replace(regex, '<mark class="search-highlight">$1</mark>')
}

// Logs functions
async function loadResourcePods(tab: LogsTab) {
  try {
    tab.loading = true
    const allPods = await GetPods(tab.namespace)
    
    let filteredPods: any[] = []
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
      await loadLogs(tab)
    }
  } catch (e) {
    console.error('Error loading resource pods:', e)
    tab.rawLogs = `Error loading pods: ${e}`
  } finally {
    tab.loading = false
  }
}

async function onPodChange(event: Event, tab: LogsTab) {
  const target = event.target as HTMLSelectElement
  tab.selectedPodName = target.value
  tab.selectedContainer = ''
  
  if (tab.selectedPodName) {
    const pod = tab.pods.find(p => p.name === tab.selectedPodName)
    if (pod?.containers?.[0]) {
      tab.selectedContainer = pod.containers[0].name
    }
  }
  
  await loadLogs(tab)
}

async function onContainerChange(event: Event, tab: LogsTab) {
  const target = event.target as HTMLSelectElement
  tab.selectedContainer = target.value
  await loadLogs(tab)
}

async function loadLogs(tab: LogsTab) {
  if (!tab.selectedPodName && tab.pods.length === 0) return
  
  try {
    tab.loading = true
    
    if (tab.selectedPodName) {
      const container = tab.selectedContainer || 
        tab.pods.find(p => p.name === tab.selectedPodName)?.containers?.[0]?.name || ''
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

function downloadLogs(tab: LogsTab) {
  const content = getDisplayedLogs(tab)
  const element = document.createElement('a')
  element.setAttribute('href', 'data:text/plain;charset=utf-8,' + encodeURIComponent(content))
  element.setAttribute('download', `${tab.resourceType}-${tab.resourceName}-logs.txt`)
  element.style.display = 'none'
  document.body.appendChild(element)
  element.click()
  document.body.removeChild(element)
}

function startAutoRefresh() {
  stopAutoRefresh()
  autoRefreshInterval = setInterval(() => {
    const activeTab = tabs.value.find(t => t.id === activeTabId.value)
    if (activeTab?.type === 'logs') {
      loadLogs(activeTab as LogsTab)
    }
  }, 45000)
}

function stopAutoRefresh() {
  if (autoRefreshInterval) {
    clearInterval(autoRefreshInterval)
    autoRefreshInterval = null
  }
}

// YAML Editor functions
function setEditorContainer(tabId: string, el: HTMLDivElement | null) {
  if (el) {
    editorContainers.set(tabId, el)
    const tab = tabs.value.find(t => t.id === tabId)
    if (tab?.type === 'yaml') {
      nextTick(() => initializeEditor(tab as YamlTab))
    }
  }
}

function initializeEditor(tab: YamlTab) {
  const container = editorContainers.get(tab.id)
  if (!container) return
  
  // Dispose existing editor if any
  const existingEditor = editors.get(tab.id)
  if (existingEditor) {
    existingEditor.dispose()
  }
  
  // Create new editor
  const editor = monaco.editor.create(container, {
    value: tab.content,
    language: 'yaml',
    theme: 'vs-dark',
    automaticLayout: true,
    minimap: { enabled: true },
    scrollBeyondLastLine: false,
    wordWrap: 'off',
    fontSize: 12,
    fontFamily: "'Monaco', 'Menlo', 'Ubuntu Mono', monospace",
    lineNumbers: 'on',
    folding: true,
    formatOnPaste: false,
    formatOnType: false,
  })
  
  // Update content on change
  editor.onDidChangeModelContent(() => {
    tab.content = editor.getValue()
  })
  
  editors.set(tab.id, editor)
}

function saveYaml(tab: YamlTab) {
  emit('apply-yaml', tab.id)
  tab.originalContent = tab.content
}

// Panel controls
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

function escapeRegex(str: string): string {
  return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

// Clean up editors when tabs are closed
watch(tabs, (newTabs) => {
  // Clean up editors for closed tabs
  const tabIds = new Set(newTabs.map(t => t.id))
  editors.forEach((editor, id) => {
    if (!tabIds.has(id)) {
      editor.dispose()
      editors.delete(id)
      editorContainers.delete(id)
    }
  })
}, { deep: true })
</script>



<style scoped>
.tabbed-panel {
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
  margin-left: auto;
}

.tab-close:hover {
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

.panel-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* Остальные стили остаются такими же, как в предыдущей версии */
/* Logs View Styles */
.logs-view {
  display: flex;
  flex-direction: column;
  height: 100%;
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

.search-highlight {
  background-color: #7a5e00;
  color: #ffd700;
  font-weight: bold;
}

/* YAML Editor View Styles */
.yaml-view {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.editor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 16px;
  border-bottom: 1px solid #3b3d5c;
  background: #252526;
  flex-shrink: 0;
  gap: 16px;
}

.editor-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  flex: 1;
  min-width: 0;
}

.editor-icon {
  font-size: 16px;
  flex-shrink: 0;
}

.editor-resource {
  color: #d4d4d4;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-weight: 500;
}

.editor-resource strong {
  color: #60a5fa;
}

.editor-namespace {
  color: #6b6d8a;
  font-size: 12px;
  white-space: nowrap;
  flex-shrink: 0;
}

.ns-link {
  color: #60a5fa;
  text-decoration: none;
  cursor: pointer;
}

.ns-link:hover {
  text-decoration: underline;
}

.editor-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.btn {
  padding: 6px 12px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  transition: all 0.15s;
}

.btn-sm {
  padding: 4px 8px;
  font-size: 11px;
}

.btn-save {
  background: #10b981;
  color: white;
}

.btn-save:hover {
  background: #059669;
}

.editor-container {
  flex: 1;
  overflow: hidden;
  position: relative;
}

/* Common input/select focus styles */
select:focus,
input:focus {
  outline: none;
  border-color: #4a5568;
  background: rgba(255, 255, 255, 0.05);
}

/* Resize handle */
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

/* Scrollbar styling */
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: #1a1b26;
}

::-webkit-scrollbar-thumb {
  background: #3b3d5c;
  border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
  background: #4a5568;
}

/* Monaco editor overrides */
:deep(.monaco-editor) {
  background-color: #1a1b26 !important;
}

:deep(.monaco-editor .margin) {
  background-color: #1a1b26 !important;
}

:deep(.monaco-editor .monaco-editor-background) {
  background-color: #1a1b26 !important;
}

:deep(.monaco-editor .minimap) {
  border-left: 1px solid #3b3d5c;
}

:deep(.monaco-editor .line-numbers) {
  color: #666 !important;
}
</style>