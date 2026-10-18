<template>
  <div class="tabbed-panel" :style="panelStyle" @mousedown="startDrag">
    <div class="panel-header" @mousedown.stop="startDrag">
      <div class="tabs-container">
        <div class="tabs-list">
          <div v-for="tab in tabs" :key="tab.id" class="tab" :class="{ active: activeTabId === tab.id }"
            @click="activeTabId = tab.id">
            <span class="tab-icon">{{ tab.icon }}</span>
            <span class="tab-name">{{ tab.title }}</span>
            <button class="tab-close" @click.stop="closeTab(tab.id)">✕</button>
          </div>
        </div>
      </div>
      <button class="btn-close" @click="closeAllTabs">✕</button>
    </div>
    <div class="panel-content">
      <template v-for="tab in tabs" :key="tab.id">
        <div v-if="tab.type === 'logs' && activeTabId === tab.id" class="logs-view">
          <div class="logs-toolbar">
            <div class="toolbar-group">
              <label>Pods:</label>
              <select :value="(tab as LogsTab).selectedPodName" @change="onPodChange($event, tab as LogsTab)"
                class="select-pod">
                <option value="">All Pods</option>
                <option v-for="pod in (tab as LogsTab).pods" :key="pod.name" :value="pod.name">{{ pod.name }}</option>
              </select>
            </div>
            <div class="toolbar-group">
              <label>Containers:</label>
              <select :value="(tab as LogsTab).selectedContainer" @change="onContainerChange($event, tab as LogsTab)"
                class="select-container">
                <option value="">All Containers</option>
                <option v-for="container in getAvailableContainers(tab as LogsTab)" :key="container" :value="container">
                  {{ container }}</option>
              </select>
            </div>
            <div class="toolbar-group">
              <label>Lines:</label>
              <input v-model.number="logLines" type="number" min="10" max="10000" @change="loadLogs(tab as LogsTab)"
                class="input-lines" />
            </div>
            <div class="toolbar-group search-group">
              <input v-model="searchText" type="text" placeholder="Search in logs" class="input-search" />
            </div>
            <div class="toolbar-actions">
              <button @click="showTimestamp = !showTimestamp" class="btn-action" :class="{ active: showTimestamp }"><img
                  :src="timestampIcon" class="icon-svg" /></button>
              <button @click="wrapLines = !wrapLines" class="btn-action" :class="{ active: wrapLines }"><img
                  :src="wrapLinesIcon" class="icon-svg" /></button>
              <button @click="loadLogs(tab as LogsTab)" class="btn-action btn-refresh"><img :src="restartIcon"
                  class="icon-svg" /></button>
              <button @click="downloadLogs(tab as LogsTab)" class="btn-action"><img :src="downloadIcon"
                  class="icon-svg" /></button>
            </div>
          </div>
          <div class="logs-info"><span>Namespace: <strong>{{ tab.namespace }}</strong> | {{ tab.resourceType }}:
              <strong>{{ tab.resourceName }}</strong> | Pods: <strong>{{ (tab as LogsTab).pods.length }}</strong></span>
          </div>
          <div class="logs-content">
            <div v-if="(tab as LogsTab).loading" class="logs-loading">⏳ Loading...</div>
            <pre v-else-if="(tab as LogsTab).rawLogs" class="logs-text" :class="{ 'no-wrap': !wrapLines }"
              v-html="searchText ? getHighlightedLogs(tab as LogsTab) : getDisplayedLogs(tab as LogsTab)"></pre>
            <div v-else class="logs-empty">No logs</div>
          </div>
        </div>
        <div v-else-if="tab.type === 'yaml' && activeTabId === tab.id" class="yaml-view">
          <div class="editor-header">
            <div class="editor-title">
              <span>{{ tab.resourceType }} <strong>{{ tab.resourceName }}</strong></span>
              <span v-if="tab.namespace" class="editor-namespace">in <a href="#" class="ns-link">{{ tab.namespace
                  }}</a></span>
              <span v-if="(tab as YamlTab).content !== (tab as YamlTab).originalContent" class="editor-dirty"
                title="Есть несохранённые изменения">●</span>
            </div>
            <button class="btn btn-sm btn-save"
              :disabled="(tab as YamlTab).saving || (tab as YamlTab).content === (tab as YamlTab).originalContent"
              @click="saveYaml(tab as YamlTab)">{{ (tab as YamlTab).saving ? '⏳ Saving...' : '💾 Save' }}</button>
          </div>
          <div :ref="el => setEditorContainer(tab.id, el as HTMLDivElement)" class="editor-container"></div>
        </div>
        <div v-else-if="tab.type === 'terminal' && activeTabId === tab.id" class="terminal-view">
          <div class="terminal-header">
            <div class="terminal-title"><span>{{ (tab as TerminalTab).mode === 'shell' ? 'Terminal' : 'Attach' }}:
                <strong>{{ (tab as TerminalTab).container }}</strong> @ {{ (tab as TerminalTab).pod }}</span></div>
            <div class="terminal-controls">
              <select :value="(tab as TerminalTab).container"
                @change="onTerminalContainerChange($event, tab as TerminalTab)" class="container-select">
                <option v-for="c in getTerminalContainers(tab as TerminalTab)" :key="c" :value="c">{{ c }}</option>
              </select>
              <span class="status-indicator">{{ terminalStatuses[tab.id] || '🔴' }}</span>
              <button class="btn btn-sm" @click="reconnectTerminal(tab.id)"
                :disabled="terminalConnecting[tab.id] !== false">🔄</button>
            </div>
          </div>
          <div :ref="el => setTerminalContainer(tab.id, el as HTMLDivElement)" class="terminal-container"></div>
        </div>
      </template>
      <div v-if="tabs.length === 0" class="no-tabs">
        <p>No tabs open</p>
      </div>
    </div>
    <div class="resize-handle" @mousedown.stop="startResize"></div>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick, reactive } from 'vue'
import * as monaco from 'monaco-editor'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import 'xterm/css/xterm.css'
import {
  GetPods,
  GetPodLogs,
  StartTerminalSession,
  ReadTerminal,
  WriteTerminal,
  CloseTerminal,
  DetectShell,
  ResizeTerminal,
} from '../../../wailsjs/go/main/App'
import { globalTabbedPanel } from '../../composables/useTabbedPanel'
import type { LogsTab, YamlTab, TerminalTab } from '../../composables/useTabbedPanel'
import restartIcon from '../../assets/icons/restart.svg'
import downloadIcon from '../../assets/icons/download.svg'
import timestampIcon from '../../assets/icons/timestamp.svg'
import wrapLinesIcon from '../../assets/icons/wrap-lines.svg'

const emit = defineEmits<{ (e: 'close'): void; (e: 'apply-yaml', tabId: string): void }>()
const tabs = computed(() => globalTabbedPanel.tabs.value)
const activeTabId = computed({ get: () => globalTabbedPanel.activeTabId.value, set: (val) => { globalTabbedPanel.activeTabId.value = val } })
const sidebarWidth = ref(200)
const panelX = ref(276)
const panelY = ref(window.innerHeight - 400)
const panelWidth = ref(window.innerWidth - sidebarWidth.value - 80)
const panelHeight = ref(400)
const isDragging = ref(false)
const isResizing = ref(false)
const dragStartX = ref(0)
const dragStartY = ref(0)
const dragStartLeft = ref(0)
const dragStartTop = ref(0)
const resizeStartX = ref(0)
const resizeStartY = ref(0)
const resizeStartWidth = ref(0)
const resizeStartHeight = ref(0)
const logLines = ref(100)
const searchText = ref('')
const showTimestamp = ref(false)
const wrapLines = ref(true)
let autoRefreshInterval: ReturnType<typeof setInterval> | null = null
const editors = new Map<string, monaco.editor.IStandaloneCodeEditor>()
const editorContainers = new Map<string, HTMLDivElement>()

// Terminal state
const terminalStatuses = reactive<Record<string, string>>({})
const terminalConnecting = reactive<Record<string, boolean>>({})
const terminalInstances = new Map<string, TerminalInstance>()
const terminalContainers = new Map<string, HTMLDivElement>()
const initializingTabs = new Set<string>()

const DEFAULT_COLS = 80
const DEFAULT_ROWS = 24

interface TerminalInstance {
  terminal: Terminal
  fitAddon: FitAddon
  sessionID: string
  readLoopAborted: boolean
  resizeObserver?: ResizeObserver
  resizeTimer?: ReturnType<typeof setTimeout>
}

const panelStyle = computed(() => ({ position: 'fixed' as const, left: `${panelX.value}px`, top: `${panelY.value}px`, width: `${panelWidth.value}px`, height: `${panelHeight.value}px` }))
const closeTab = (tabId: string) => globalTabbedPanel.closeTab(tabId)
const closeAllTabs = () => globalTabbedPanel.closeAllTabs()

onMounted(() => {
  const sidebar = document.querySelector('.sidebar') as HTMLElement | null
  if (sidebar) sidebarWidth.value = sidebar.offsetWidth
  const style = document.createElement('style')
  style.textContent = `@font-face { font-family: 'codicon'; src: url('${new URL('monaco-editor/esm/vs/base/browser/ui/codicons/codicon/codicon.ttf', import.meta.url).href}') format('truetype'); }`
  document.head.appendChild(style)
  if (tabs.value.length > 0 && activeTabId.value) {
    const activeTab = tabs.value.find((t: any) => t.id === activeTabId.value)
    if (activeTab?.type === 'logs') { loadResourcePods(activeTab as LogsTab); startAutoRefresh() }
    else if (activeTab?.type === 'terminal') initTerminal(activeTab as TerminalTab)
  }
  window.addEventListener('mousemove', handleResize)
  window.addEventListener('mousemove', handleDrag)
  window.addEventListener('mouseup', stopResize)
  window.addEventListener('mouseup', stopDrag)
})

onBeforeUnmount(() => {
  stopAutoRefresh()
  editors.forEach(editor => editor.dispose())
  editors.clear()
  editorContainers.clear()
  cleanupAllTerminals()
  window.removeEventListener('mousemove', handleResize)
  window.removeEventListener('mousemove', handleDrag)
  window.removeEventListener('mouseup', stopResize)
  window.removeEventListener('mouseup', stopDrag)
})

watch(activeTabId, async (newId) => {
  const tab = tabs.value.find((t: any) => t.id === newId)
  if (!tab) return
  if (tab.type === 'logs' && !(tab as LogsTab).rawLogs) { await loadResourcePods(tab as LogsTab); startAutoRefresh() }
  else if (tab.type === 'yaml') { await nextTick(); initializeEditor(tab as YamlTab) }
  else if (tab.type === 'terminal') { await nextTick(); await new Promise(r => setTimeout(r, 50)); await initTerminal(tab as TerminalTab) }
})

// Внешнее изменение content вкладки (не набор текста пользователем) -> обновить редактор
watch(
  () => tabs.value.map((t: any) => (t.type === 'yaml' ? t.content : null)),
  () => {
    for (const t of tabs.value as any[]) {
      if (t.type !== 'yaml') continue
      const ed = editors.get(t.id)
      if (ed && ed.getValue() !== t.content) ed.setValue(t.content)
    }
  },
)

watch(tabs, (newTabs) => {
  const currentTabIds = new Set(newTabs.map((t: any) => t.id))
  for (const [tabId, instance] of terminalInstances) {
    if (!currentTabIds.has(tabId)) { cleanupTerminal(tabId, instance); terminalInstances.delete(tabId); terminalContainers.delete(tabId); initializingTabs.delete(tabId) }
  }
  editors.forEach((editor, id) => { if (!currentTabIds.has(id)) { editor.dispose(); editors.delete(id); editorContainers.delete(id) } })
}, { deep: true })

function getAvailableContainers(tab: LogsTab): string[] {
  if (!tab.selectedPodName) { const containers = new Set<string>(); tab.pods.forEach((pod: any) => { pod.containers?.forEach((c: any) => containers.add(c.name)) }); return Array.from(containers).sort() }
  else { const pod = tab.pods.find(p => p.name === tab.selectedPodName); return pod?.containers?.map((c: any) => c.name) || [] }
}

function getDisplayedLogs(tab: LogsTab): string {
  if (!tab.rawLogs) return ''
  let logs = tab.rawLogs
  if (!showTimestamp.value) { logs = logs.split('\n').map((line: string) => { if (!line.trim()) return line; const m = line.match(/^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z[+-]?\d{0,2}:?\d{0,2})\s+(.*)$/); if (m && m[2]) return m[2]; return line }).join('\n') }
  else { logs = logs.split('\n').map((line: string) => { if (!line.trim()) return line; const m = line.match(/^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z[+-]?\d{0,2}:?\d{0,2})\s+(.*)$/); if (m && m[1]) return `[${m[1]}] ${m[2]}`; return line }).join('\n') }
  if (!searchText.value) return logs
  const search = searchText.value.toLowerCase()
  return logs.split('\n').filter((line: string) => line.toLowerCase().includes(search)).join('\n')
}

function getHighlightedLogs(tab: LogsTab): string {
  if (!tab.rawLogs || !searchText.value) return tab.rawLogs || ''
  const logs = getDisplayedLogs(tab)
  const escaped = searchText.value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  return logs.replace(new RegExp(`(${escaped})`, 'gi'), '<mark class="search-highlight">$1</mark>')
}

async function loadResourcePods(tab: LogsTab) {
  try {
    tab.loading = true
    const allPods = await GetPods(tab.namespace)
    let filteredPods: any[] = []
    const resourceName = tab.resourceName
    switch (tab.resourceType.toLowerCase()) {
      case 'deployment': filteredPods = allPods.filter((pod: any) => { const l = pod.labels || {}; return l['app.kubernetes.io/instance'] === resourceName || l['app'] === resourceName || pod.name.startsWith(resourceName) }); break
      case 'job': filteredPods = allPods.filter((pod: any) => pod.name.startsWith(resourceName + '-')); break
      case 'pod': filteredPods = allPods.filter((pod: any) => pod.name === resourceName); break
      default: filteredPods = []
    }
    tab.pods = filteredPods
    if (tab.pods.length > 0 && !tab.selectedPodName) { tab.selectedPodName = tab.pods[0].name; tab.selectedContainer = tab.pods[0].containers?.[0]?.name || '' }
    if (tab.selectedPodName || tab.pods.length > 0) await loadLogs(tab)
  } catch (e: any) { tab.rawLogs = `Error: ${e}` }
  finally { tab.loading = false }
}

async function onPodChange(event: Event, tab: LogsTab) {
  const target = event.target as HTMLSelectElement
  tab.selectedPodName = target.value
  tab.selectedContainer = ''
  if (tab.selectedPodName) { const pod = tab.pods.find(p => p.name === tab.selectedPodName); if (pod?.containers?.[0]) tab.selectedContainer = pod.containers[0].name }
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
      const container = tab.selectedContainer || tab.pods.find(p => p.name === tab.selectedPodName)?.containers?.[0]?.name || ''
      tab.rawLogs = await GetPodLogs(tab.namespace, tab.selectedPodName, container, logLines.value)
    } else {
      const allLogs: string[] = []
      for (const pod of tab.pods) {
        const container = tab.selectedContainer || pod.containers?.[0]?.name || ''
        try { const logs = await GetPodLogs(tab.namespace, pod.name, container, logLines.value); if (logs) { allLogs.push(`\n=== ${pod.name} ===\n`); allLogs.push(logs) } }
        catch (e: any) { allLogs.push(`\n=== ${pod.name} (error) ===\n${e}`) }
      }
      tab.rawLogs = allLogs.join('')
    }
  } catch (e: any) { tab.rawLogs = `Error: ${e}` }
  finally { tab.loading = false }
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

function startAutoRefresh() { stopAutoRefresh(); autoRefreshInterval = setInterval(() => { const t = tabs.value.find((x: any) => x.id === activeTabId.value); if (t?.type === 'logs') loadLogs(t as LogsTab) }, 45000) }
function stopAutoRefresh() { if (autoRefreshInterval) { clearInterval(autoRefreshInterval); autoRefreshInterval = null } }

// Состояние view (курсор, прокрутка) на случай переключения вкладок
const editorViewStates = new Map<string, monaco.editor.ICodeEditorViewState | null>()

function setEditorContainer(tabId: string, el: HTMLDivElement | null) {
  if (!el) return
  // Функция-ref вызывается при каждой перерисовке: тот же элемент с живым редактором пропускаем
  if (editorContainers.get(tabId) === el && editors.has(tabId)) return
  editorContainers.set(tabId, el)
  const tab = tabs.value.find((t: any) => t.id === tabId)
  if (tab?.type === 'yaml') nextTick(() => initializeEditor(tab as YamlTab))
}

function initializeEditor(tab: YamlTab) {
  const container = editorContainers.get(tab.id)
  if (!container) return

  const existing = editors.get(tab.id)
  if (existing) {
    // Редактор уже создан в этом контейнере: ничего не пересоздаём
    if (container.contains(existing.getDomNode())) return
    // Контейнер новый (вкладку переключали): запоминаем позицию и пересоздаём
    editorViewStates.set(tab.id, existing.saveViewState())
    existing.dispose()
    editors.delete(tab.id)
  }

  const editor = monaco.editor.create(container, {
    value: tab.content,
    language: 'yaml',
    theme: 'vs-dark',
    automaticLayout: true,
    minimap: { enabled: true },
    scrollBeyondLastLine: false,
    wordWrap: 'off',
    fontSize: 12,
    fontFamily: "'Monaco', 'Menlo', monospace",
    lineNumbers: 'on',
    folding: true,
  })

  const state = editorViewStates.get(tab.id)
  if (state) editor.restoreViewState(state)

  editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => saveYaml(tab))
  editor.onDidChangeModelContent(() => { tab.content = editor.getValue() })
  editors.set(tab.id, editor)
}

// Больше не ставим originalContent здесь: App.vue делает это только после успешного apply
function saveYaml(tab: YamlTab) {
  if (tab.saving || tab.content === tab.originalContent) return
  emit('apply-yaml', tab.id)
}

// Terminal functions
function getTerminalContainers(tab: TerminalTab): string[] {
  // Используем новое свойство containers
  if (tab.containers && tab.containers.length > 0) {
    return tab.containers.map(c => c.name)
  }
  // Fallback на podObj
  return tab.podObj?.containers?.map((c: any) => c.name) || []
}

function onTerminalContainerChange(event: Event, tab: TerminalTab) {
  const target = event.target as HTMLSelectElement
  tab.container = target.value
  reconnectTerminal(tab.id)
}

function setTerminalContainer(tabId: string, el: HTMLDivElement | null) {
  if (!el) return
  terminalContainers.set(tabId, el)
  if (terminalInstances.has(tabId) || initializingTabs.has(tabId)) return
  const tab = tabs.value.find((t: any) => t.id === tabId)
  if (!tab || tab.type !== 'terminal') return
  initializingTabs.add(tabId)
  terminalStatuses[tabId] = '🟡'
  terminalConnecting[tabId] = true
  nextTick().then(() => setTimeout(() => initTerminal(tab as TerminalTab), 200))
}

async function initTerminal(tab: TerminalTab) {
  const container = terminalContainers.get(tab.id)
  if (!container) { terminalStatuses[tab.id] = '🔴'; terminalConnecting[tab.id] = true; initializingTabs.delete(tab.id); return }
  terminalStatuses[tab.id] = '🟡'
  terminalConnecting[tab.id] = true
  try {
    let shell = '/bin/sh'
    try { shell = await DetectShell(tab.namespace, tab.pod, tab.container) } catch { shell = '/bin/sh' }

    // Очищаем контейнер перед созданием терминала
    container.innerHTML = ''

    const terminal = new Terminal({
      fontFamily: 'Monaco, Menlo, "Courier New", monospace',
      fontSize: 13,
      lineHeight: 1.2,
      theme: {
        background: '#1e1e1e',
        foreground: '#d4d4d4',
        cursor: '#00ff00',
        cursorAccent: '#00ff00'
      },
      cursorBlink: true,
      disableStdin: false,
      cols: DEFAULT_COLS,
      rows: DEFAULT_ROWS,
      allowProposedApi: true,
      scrollback: 10000,
      tabStopWidth: 4
    })
    const fitAddon = new FitAddon()
    terminal.loadAddon(fitAddon)
    terminal.open(container)

    // Ждем пока терминал откроется
    await nextTick()
    await new Promise(r => setTimeout(r, 150))

    // Fit перед фокусом
    try { fitAddon.fit() } catch { }

    const sessionID = `term_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
    const cols = terminal.cols
    const rows = terminal.rows
    const sessionMode = tab.mode === 'attach' ? '' : shell
    await StartTerminalSession(sessionID, tab.namespace, tab.pod, tab.container, sessionMode, cols, rows)
    const instance: TerminalInstance = { terminal, fitAddon, sessionID, readLoopAborted: false }
    terminalInstances.set(tab.id, instance)
    terminalStatuses[tab.id] = '🟢'
    terminalConnecting[tab.id] = false

    // Обработчики событий
    terminal.onData((data: string) => { if (!instance.sessionID) return; const bytes = Array.from(new TextEncoder().encode(data)); WriteTerminal(instance.sessionID, bytes).catch(() => { }) })
    terminal.onResize(({ cols, rows }) => debouncedResize(tab.id, cols, rows))

    // ResizeObserver для контейнера
    instance.resizeObserver = new ResizeObserver(() => safeFit(tab.id))
    instance.resizeObserver.observe(container)

    // Фокус при клике
    container.addEventListener('mousedown', () => { terminal.focus() })
    container.addEventListener('click', () => { terminal.focus() })

    await new Promise(r => setTimeout(r, 100))

    // Финальный fit и фокус
    try { fitAddon.fit() } catch { }
    readLoop(tab.id, instance)
    terminal.focus()

    // Скролл вниз при подключении
    const viewport = container.querySelector('.xterm-viewport') as HTMLElement
    if (viewport) {
      await nextTick()
      viewport.scrollTop = viewport.scrollHeight
    }
  } catch (e: any) { console.error('Terminal init error:', e); terminalStatuses[tab.id] = '🔴'; terminalConnecting[tab.id] = true }
  initializingTabs.delete(tab.id)
}

function safeFit(tabId: string) { const instance = terminalInstances.get(tabId); if (!instance?.fitAddon) return; try { instance.fitAddon.fit() } catch { } }

function debouncedResize(tabId: string, cols: number, rows: number) {
  const instance = terminalInstances.get(tabId)
  if (!instance?.sessionID) return
  if (instance.resizeTimer) clearTimeout(instance.resizeTimer)
  instance.resizeTimer = setTimeout(async () => { try { await ResizeTerminal(instance.sessionID, cols, rows) } catch { } }, 150)
}

async function readLoop(tabId: string, instance: TerminalInstance) {
  const capturedSessionID = instance.sessionID
  while (!instance.readLoopAborted && capturedSessionID === instance.sessionID && instance.terminal) {
    try {
      const result: any = await ReadTerminal(capturedSessionID)
      if (!result || capturedSessionID !== instance.sessionID) {
        instance.terminal?.write('\r\n[Session ended]\r\n')
        terminalStatuses[tabId] = '🔴'
        terminalConnecting[tabId] = true
        break
      }
      if (typeof result === 'string' && result.length > 0) {
        try {
          const binary = atob(result)
          instance.terminal?.write(binary)
        } catch {
          instance.terminal?.write(result)
        }
      }
    } catch (e: any) {
      if (!instance.readLoopAborted && capturedSessionID === instance.sessionID) {
        instance.terminal?.write(`\r\n[Connection lost]\r\n`)
        terminalStatuses[tabId] = '🔴'
        terminalConnecting[tabId] = true
      }
      break
    }
  }
}

function cleanupTerminal(tabId: string, instance: TerminalInstance) {
  instance.readLoopAborted = true
  if (instance.resizeTimer) clearTimeout(instance.resizeTimer)
  if (instance.sessionID) CloseTerminal(instance.sessionID).catch(() => { })
  if (instance.resizeObserver) instance.resizeObserver.disconnect()
  try { instance.fitAddon = undefined as any } catch { }
  try { instance.terminal.dispose() } catch { }
  terminalStatuses[tabId] = '🔴'
  terminalConnecting[tabId] = true
}

function cleanupAllTerminals() {
  for (const [tabId, instance] of terminalInstances) cleanupTerminal(tabId, instance)
  terminalInstances.clear()
  terminalContainers.clear()
  initializingTabs.clear()
}

async function reconnectTerminal(tabId: string) {
  const instance = terminalInstances.get(tabId)
  if (instance) cleanupTerminal(tabId, instance)
  terminalInstances.delete(tabId)
  initializingTabs.delete(tabId)
  const tab = tabs.value.find((t: any) => t.id === tabId)
  if (tab) { initializingTabs.add(tabId); await initTerminal(tab as TerminalTab) }
}

// Panel resize and drag functions
function startDrag(e: MouseEvent) {
  isDragging.value = true
  dragStartX.value = e.clientX
  dragStartY.value = e.clientY
  dragStartLeft.value = panelX.value
  dragStartTop.value = panelY.value
}

function stopDrag() { isDragging.value = false }

function handleDrag(e: MouseEvent) {
  if (!isDragging.value) return
  const deltaX = e.clientX - dragStartX.value
  const deltaY = e.clientY - dragStartY.value
  panelX.value = dragStartLeft.value + deltaX
  panelY.value = dragStartTop.value + deltaY
}

function startResize(e: MouseEvent) {
  isResizing.value = true
  resizeStartX.value = e.clientX
  resizeStartY.value = e.clientY
  resizeStartWidth.value = panelWidth.value
  resizeStartHeight.value = panelHeight.value
  e.preventDefault()
}

function stopResize() { isResizing.value = false }

function handleResize(e: MouseEvent) {
  if (!isResizing.value) return
  const deltaX = e.clientX - resizeStartX.value
  const deltaY = e.clientY - resizeStartY.value
  panelWidth.value = Math.max(400, resizeStartWidth.value + deltaX)
  panelHeight.value = Math.max(250, resizeStartHeight.value + deltaY)
}
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

/* Logs View */
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
.select-container,
.input-lines,
.input-search {
  background: #1a1b26;
  border: 1px solid #3b3d5c;
  color: #d4d4d4;
  padding: 6px 8px;
  border-radius: 3px;
  font-size: 12px;
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
}

.logs-content {
  flex: 1;
  overflow: auto;
  background: #1a1b26;
}

.logs-text {
  margin: 0;
  padding: 16px;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  line-height: 1.5;
  color: #d4d4d4;
  white-space: pre-wrap;
  word-wrap: break-word;
  text-align: left;
}

.logs-text.no-wrap {
  white-space: pre;
  word-wrap: normal;
  word-break: normal;
  overflow-x: auto;
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

.search-highlight {
  background-color: #7a5e00;
  color: #ffd700;
  font-weight: bold;
}

/* YAML View */
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
}

.editor-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}

.editor-namespace {
  color: #6b6d8a;
  font-size: 12px;
}

.ns-link {
  color: #60a5fa;
  text-decoration: none;
}

.ns-link:hover {
  text-decoration: underline;
}

.btn {
  padding: 6px 12px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
}

.btn-sm {
  padding: 4px 8px;
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

/* Terminal View */
.terminal-view {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.terminal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 16px;
  border-bottom: 1px solid #3b3d5c;
  background: #252526;
}

.terminal-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: #d4d4d4;
}

.terminal-controls {
  display: flex;
  gap: 8px;
  align-items: center;
}

.container-select {
  background: #1a1b26;
  border: 1px solid #3b3d5c;
  color: #d4d4d4;
  padding: 4px 8px;
  border-radius: 3px;
  font-size: 12px;
  min-width: 100px;
  max-width: 200px;
}

.status-indicator {
  font-size: 12px;
}

.terminal-container {
  flex: 1;
  overflow: hidden;
  padding: 0;
  margin: 0;
  position: relative;
  background: #1e1e1e;
  padding-left: 8px !important;
}

:deep(.xterm) {
  width: 100% !important;
  height: 100% !important;
  padding: 0 !important;
  margin: 0 !important;
  position: relative !important;
  left: 0 !important;
}

:deep(.xterm-cursor-layer) {
  z-index: 10 !important;
}

:deep(.xterm-screen) {
  margin: 0 !important;
  padding: 0 !important;
  position: relative !important;
  left: 20px !important;
  text-align: left !important;
}

:deep(.xterm-rows) {
  padding-left: 0 !important;
  margin-left: 0 !important;
}

:deep(.xterm-viewport) {
  overflow-y: auto !important;
  overflow-x: hidden !important;
  position: absolute !important;
  top: 0 !important;
  right: 0 !important;
  bottom: 0 !important;
  width: 12px !important;
  z-index: 100 !important;
}

:deep(.xterm-scroll-area) {
  padding-left: 0 !important;
  margin-left: 0 !important;
}

:deep(.xterm-char-measure-element) {
  position: absolute !important;
  left: -10000px !important;
}

/* Убираем отступы у строк терминала */
:deep(.xterm .xterm-row) {
  padding-left: 0 !important;
  margin-left: 0 !important;
}

/* Скроллбар для терминала */
:deep(.xterm-viewport::-webkit-scrollbar) {
  width: 10px !important;
}

:deep(.xterm-viewport::-webkit-scrollbar-track) {
  background: #1e1e1e !important;
}

:deep(.xterm-viewport::-webkit-scrollbar-thumb) {
  background: #444 !important;
  border-radius: 5px !important;
}

:deep(.xterm-viewport::-webkit-scrollbar-thumb:hover) {
  background: #555 !important;
}

.no-tabs {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #666;
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
}

.resize-handle:hover {
  opacity: 1;
}

:deep(.monaco-editor) {
  background-color: #1a1b26 !important;
}

:deep(.monaco-editor .margin) {
  background-color: #1a1b26 !important;
}

:deep(.monaco-editor .minimap) {
  border-left: 1px solid #3b3d5c;
}

.editor-dirty { color: #facc15; font-size: 14px; }
.btn-save:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
