<template v-if="props.podActions">
  <div class="tabbed-actions-wrapper"
    :style="{ bottom: 0, right: 0, width: '100%', height: '100%', position: 'fixed', zIndex: 50 }">
    <div class="tabbed-actions-panel">
      <div class="resize-handle resize-top" @mousedown="(e) => startResize(e, 'top')"></div>
      <div class="resize-handle resize-bottom" @mousedown="(e) => startResize(e, 'bottom')"></div>
      <div class="resize-handle resize-left" @mousedown="(e) => startResize(e, 'left')"></div>
      <div class="resize-handle resize-right" @mousedown="(e) => startResize(e, 'right')"></div>
      <div class="resize-handle resize-top-left" @mousedown="(e) => startResize(e, 'top-left')"></div>
      <div class="resize-handle resize-top-right" @mousedown="(e) => startResize(e, 'top-right')"></div>
      <div class="resize-handle resize-bottom-left" @mousedown="(e) => startResize(e, 'bottom-left')"></div>
      <div class="resize-handle resize-bottom-right" @mousedown="(e) => startResize(e, 'bottom-right')"></div>

      <div class="tabs-header" @mousedown="startDrag">
        <div class="tabs-container">
          <div v-for="tab in props.podActions.openTabs.value" :key="tab.id" class="tab"
            :class="{ active: props.podActions.activeTabId.value === tab.id }"
            @click="props.podActions.activeTabId.value = tab.id">
            <span class="tab-icon">{{ tab.type === 'terminal' ? '🖥' : '📋' }}</span>
            <span class="tab-title">{{ tab.title }}</span>
            <button class="tab-close" @click.prevent.stop="props.podActions.closeTab(tab.id)"
              title="Close tab">✕</button>
          </div>
        </div>
      </div>

      <div class="actions-content">
        <div v-for="tab in props.podActions.openTabs.value" :key="tab.id" class="action-wrapper"
          :class="{ active: props.podActions.activeTabId.value === tab.id }">
          <!-- Terminal Tab -->
          <template v-if="tab.type === 'terminal'">
            <div class="action-header">
              <div class="action-title">
                <span class="action-icon">🖥</span>
                <span>{{ tab.mode === 'attach' ? 'Attach' : 'Terminal' }}: <strong>{{ tab.container }}</strong> @ {{
                  tab.pod }}</span>
              </div>
              <div class="action-controls">
                <select :value="tab.container" @change="(e) => onTerminalContainerChange(tab.id, e)" class="log-select">
                  <option v-for="c in getContainers(tab)" :key="c" :value="c">{{ c }}</option>
                </select>
                <span class="status-indicator">{{ terminalStatuses[tab.id] || '🔴' }}</span>
                <button class="btn btn-sm" @click="reconnectTerminal(tab.id)"
                  :disabled="terminalConnecting[tab.id] !== false">🔄</button>
              </div>
            </div>
            <div class="terminal-container" :ref="(el) => setTerminalContainer(tab.id, el)"></div>
          </template>

          <!-- Logs Tab -->
          <template v-if="tab.type === 'logs'">
            <div class="action-header">
              <div class="action-title">
                <span class="action-icon">📋</span>
                <span>Logs: <strong>{{ tab.container }}</strong> @ {{ tab.pod }}</span>
              </div>
              <div class="action-controls">
                <select :value="tab.container" @change="(e) => onLogsContainerChange(tab.id, e)" class="log-select">
                  <option v-for="c in getContainers(tab)" :key="c" :value="c">{{ c }}</option>
                </select>
                <select :value="logsLines[tab.id] || 100" @change="(e) => onLogsLinesChange(tab.id, e)"
                  class="log-select">
                  <option :value="50">50</option>
                  <option :value="100">100</option>
                  <option :value="500">500</option>
                  <option :value="1000">1000</option>
                </select>
                <button class="btn btn-sm" @click="reloadLogs(tab.id)" :disabled="logsLoading[tab.id]">🔄</button>
              </div>
            </div>
            <div class="logs-container">
              <div v-if="logsLoading[tab.id]" class="logs-loading">Loading logs...</div>
              <pre v-else-if="logsContent[tab.id]" class="logs-output">{{ logsContent[tab.id] }}</pre>
              <div v-else class="logs-empty">No logs available</div>
            </div>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, reactive, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import 'xterm/css/xterm.css'
import type { ActionTab } from '../../composables/usePodActions'

import {
  StartTerminalSession,
  ReadTerminal,
  WriteTerminal,
  CloseTerminal,
  DetectShell,
  ResizeTerminal,
  GetPodLogs,
} from '../../../wailsjs/go/main/App'

interface Props {
  namespace?: string
  pod?: string
  containers?: string[]
  podActions?: any
}

const props = withDefaults(defineProps<Props>(), {
  namespace: undefined,
  pod: undefined,
  containers: undefined,
  podActions: undefined,
})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'evict'): void
}>()

const panelHeight = ref(400)
const panelWidth = ref(800)
const panelLeft = ref(0)
const panelTop = ref(0)

// ===== Реактивные состояния для UI =====
const terminalStatuses = reactive<Record<string, string>>({})
const terminalConnecting = reactive<Record<string, boolean>>({})
const logsLoading = reactive<Record<string, boolean>>({})
const logsContent = reactive<Record<string, string>>({})
const logsLines = reactive<Record<string, number>>({})

// Счётчик для принудительного обновления UI
const renderKey = ref(0)

interface TerminalInstance {
  terminal: Terminal
  fitAddon: FitAddon
  sessionID: string
  readLoopAborted: boolean
  resizeObserver?: ResizeObserver
  resizeTimer?: ReturnType<typeof setTimeout>
}

const terminalInstances = new Map<string, TerminalInstance>()
const terminalContainers = new Map<string, HTMLDivElement>()
// Множество табов, для которых уже запущена/запускается инициализация
const initializingTabs = new Set<string>()

let isResizing = false
let isDragging = false
let resizeDirection = ''
let dragStartX = 0
let dragStartY = 0
let dragStartLeft = 0
let dragStartTop = 0
let dragStartWidth = 0
let dragStartHeight = 0

const DEFAULT_COLS = 80
const DEFAULT_ROWS = 24

onMounted(() => {
  window.addEventListener('mousemove', handleResize)
  window.addEventListener('mousemove', handleDrag)
  window.addEventListener('mouseup', stopResize)
  window.addEventListener('mouseup', stopDrag)
})

// Watch ТОЛЬКО для логов — терминалы инициализируются из setTerminalContainer
watch(
  () => props.podActions?.openTabs.value?.length,
  async () => {
    const tabs = props.podActions?.openTabs.value
    if (!tabs) return

    for (const tab of tabs as ActionTab[]) {
      if (tab.type === 'logs' && !(tab.id in logsContent) && !initializingTabs.has(tab.id)) {
        initializingTabs.add(tab.id)
        logsLoading[tab.id] = true
        logsLines[tab.id] = 100
        await loadLogs(tab)
      }
    }
  },
  { immediate: true }
)

// Когда переключаемся на таб — перефитим терминал
watch(
  () => props.podActions?.activeTabId.value,
  async (newTabId) => {
    if (!newTabId) return
    await nextTick()
    await new Promise(resolve => setTimeout(resolve, 50))
    const instance = terminalInstances.get(newTabId)
    if (instance?.fitAddon) {
      try {
        instance.fitAddon.fit()
        instance.terminal.focus()
      } catch { }
    }
  }
)

// Следим за закрытием табов — чистим терминалы
watch(
  () => props.podActions?.openTabs.value?.length,
  () => {
    const tabs = props.podActions?.openTabs.value as ActionTab[] | undefined
    if (!tabs) return

    const currentTabIds = new Set(tabs.map(t => t.id))

    // Удаляем терминалы для закрытых табов
    for (const [tabId, instance] of terminalInstances) {
      if (!currentTabIds.has(tabId)) {
        cleanupTerminal(tabId, instance)
        terminalInstances.delete(tabId)
        terminalContainers.delete(tabId)
        initializingTabs.delete(tabId)
      }
    }

    // Удаляем кэш логов для закрытых табов
    for (const tabId of Object.keys(logsContent)) {
      if (!currentTabIds.has(tabId)) {
        delete logsContent[tabId]
        delete logsLoading[tabId]
        delete logsLines[tabId]
        initializingTabs.delete(tabId)
      }
    }
  }
)

onBeforeUnmount(() => {
  window.removeEventListener('mousemove', handleResize)
  window.removeEventListener('mousemove', handleDrag)
  window.removeEventListener('mouseup', stopResize)
  window.removeEventListener('mouseup', stopDrag)
  cleanupAllTerminals()
})

function cleanupAllTerminals() {
  for (const [tabId, instance] of terminalInstances) {
    cleanupTerminal(tabId, instance)
  }
  terminalInstances.clear()
  terminalContainers.clear()
  initializingTabs.clear()
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

// ===== КЛЮЧЕВОЙ МЕТОД — точка входа для инициализации терминала =====
async function setTerminalContainer(tabId: string, el: any) {
  if (!el) {
    // Vue может вызвать с null при unmount
    return
  }

  terminalContainers.set(tabId, el)

  // Если терминал уже инициализирован или в процессе — пропускаем
  if (terminalInstances.has(tabId) || initializingTabs.has(tabId)) {
    return
  }

  // Находим таб
  const tab = props.podActions?.openTabs.value?.find((t: ActionTab) => t.id === tabId)
  if (!tab || tab.type !== 'terminal') return

  // Помечаем как инициализирующийся
  initializingTabs.add(tabId)
  terminalStatuses[tabId] = '🟡'
  terminalConnecting[tabId] = true

  // Даём DOM полностью отрендериться
  await nextTick()
  await new Promise(resolve => setTimeout(resolve, 200))

  // Проверяем что таб всё ещё существует
  const stillExists = props.podActions?.openTabs.value?.find((t: ActionTab) => t.id === tabId)
  if (!stillExists || terminalInstances.has(tabId)) {
    initializingTabs.delete(tabId)
    return
  }

  await initTerminal(tab)
}

function getContainers(tab: ActionTab): string[] {
  return tab.podObj?.containers?.map((c: any) => c.name) || []
}

async function initTerminal(tab: ActionTab) {
  const container = terminalContainers.get(tab.id)
  if (!container) {
    console.warn('initTerminal: container not found for', tab.id)
    terminalStatuses[tab.id] = '🔴'
    terminalConnecting[tab.id] = true
    initializingTabs.delete(tab.id)
    return
  }

  terminalStatuses[tab.id] = '🟡'
  terminalConnecting[tab.id] = true

  try {
    let shell = '/bin/sh'
    try {
      shell = await DetectShell(tab.namespace, tab.pod, tab.container)
    } catch {
      shell = '/bin/sh'
    }

    const terminal = new Terminal({
      fontFamily: 'Monaco, Menlo, "Courier New", monospace',
      fontSize: 13,
      lineHeight: 1.2,
      theme: { background: '#1e1e1e', foreground: '#d4d4d4', cursor: '#00ff00' },
      cursorBlink: true,
      disableStdin: false,
      cols: DEFAULT_COLS,
      rows: DEFAULT_ROWS,
      allowProposedApi: true,
    })

    const fitAddon = new FitAddon()
    terminal.loadAddon(fitAddon)
    terminal.open(container)

    await nextTick()
    await new Promise(resolve => setTimeout(resolve, 100))
    terminal.focus()
    try { fitAddon.fit() } catch { }

    const sessionID = `term_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
    const cols = terminal.cols
    const rows = terminal.rows

    // Для attach режима используем пустую shell (сигнал бэкэнду использовать attach вместо shell)
    const sessionMode = tab.mode === 'attach' ? '' : shell
    
    await StartTerminalSession(sessionID, tab.namespace, tab.pod, tab.container, sessionMode, cols, rows)

    const instance: TerminalInstance = { terminal, fitAddon, sessionID, readLoopAborted: false }
    terminalInstances.set(tab.id, instance)

    // Обновляем реактивные статусы
    terminalStatuses[tab.id] = '🟢'
    terminalConnecting[tab.id] = false

    terminal.onData((data: string) => {
      if (!instance.sessionID) return
      const bytes = Array.from(new TextEncoder().encode(data))
      WriteTerminal(instance.sessionID, bytes).catch(() => { })
    })

    terminal.onResize(({ cols, rows }) => debouncedResize(tab.id, cols, rows))
    instance.resizeObserver = new ResizeObserver(() => safeFit(tab.id))
    instance.resizeObserver.observe(container)
    container.addEventListener('mousedown', () => terminal.focus())
    container.addEventListener('click', () => terminal.focus())

    await new Promise(resolve => setTimeout(resolve, 100))
    readLoop(tab.id, instance)
    terminal.focus()
  } catch (e: any) {
    console.error('Terminal init error:', e)
    terminalStatuses[tab.id] = '🔴'
    terminalConnecting[tab.id] = true
  }
}

function safeFit(tabId: string) {
  const instance = terminalInstances.get(tabId)
  if (!instance?.fitAddon) return
  try {
    instance.fitAddon.fit()
  } catch { }
}

function debouncedResize(tabId: string, cols: number, rows: number) {
  const instance = terminalInstances.get(tabId)
  if (!instance?.sessionID) return
  if (instance.resizeTimer) clearTimeout(instance.resizeTimer)
  instance.resizeTimer = setTimeout(async () => {
    try {
      await ResizeTerminal(instance.sessionID, cols, rows)
    } catch { }
  }, 150)
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

async function loadLogs(tab: ActionTab) {
  const lines = logsLines[tab.id] || 100
  logsLoading[tab.id] = true

  try {
    const result = await GetPodLogs(tab.namespace, tab.pod, tab.container, lines)
    logsContent[tab.id] = result || 'No logs'
  } catch (e: any) {
    logsContent[tab.id] = `Error: ${e?.message || e}`
  } finally {
    logsLoading[tab.id] = false
  }
}

function onTerminalContainerChange(tabId: string, e: Event) {
  const tab = props.podActions.openTabs.value.find((t: ActionTab) => t.id === tabId)
  if (!tab) return
  tab.container = (e.target as HTMLSelectElement).value
  reconnectTerminal(tabId)
}

function onLogsContainerChange(tabId: string, e: Event) {
  const tab = props.podActions.openTabs.value.find((t: ActionTab) => t.id === tabId)
  if (!tab) return
  tab.container = (e.target as HTMLSelectElement).value
  loadLogs(tab)
}

function onLogsLinesChange(tabId: string, e: Event) {
  logsLines[tabId] = Number((e.target as HTMLSelectElement).value)
  const tab = props.podActions.openTabs.value.find((t: ActionTab) => t.id === tabId)
  if (tab) loadLogs(tab)
}

function reloadLogs(tabId: string) {
  const tab = props.podActions.openTabs.value.find((t: ActionTab) => t.id === tabId)
  if (tab) loadLogs(tab)
}

async function reconnectTerminal(tabId: string) {
  const instance = terminalInstances.get(tabId)
  if (instance) cleanupTerminal(tabId, instance)
  terminalInstances.delete(tabId)
  initializingTabs.delete(tabId)

  const tab = props.podActions.openTabs.value.find((t: ActionTab) => t.id === tabId)
  if (tab) {
    initializingTabs.add(tabId)
    await initTerminal(tab)
  }
}

function startResize(e: MouseEvent, direction: string) {
  isResizing = true
  resizeDirection = direction
  dragStartX = e.clientX
  dragStartY = e.clientY
  dragStartLeft = panelLeft.value
  dragStartTop = panelTop.value
  dragStartWidth = panelWidth.value
  dragStartHeight = panelHeight.value
  e.preventDefault()
}

function stopResize() {
  isResizing = false
  resizeDirection = ''
}

function startDrag(e: MouseEvent) {
  isDragging = true
  dragStartX = e.clientX
  dragStartY = e.clientY
  dragStartLeft = panelLeft.value
  dragStartTop = panelTop.value
}

function stopDrag() {
  isDragging = false
}

function handleDrag(e: MouseEvent) {
  if (!isDragging) return
  const deltaX = e.clientX - dragStartX
  const deltaY = e.clientY - dragStartY
  panelLeft.value = dragStartLeft + deltaX
  panelTop.value = dragStartTop + deltaY
}

function handleResize(e: MouseEvent) {
  if (!isResizing) return

  const deltaX = e.clientX - dragStartX
  const deltaY = e.clientY - dragStartY

  if (resizeDirection.includes('top')) {
    panelTop.value = dragStartTop + deltaY
    panelHeight.value = dragStartHeight - deltaY
  }
  if (resizeDirection.includes('bottom')) {
    panelHeight.value = dragStartHeight + deltaY
  }
  if (resizeDirection.includes('left')) {
    panelLeft.value = dragStartLeft + deltaX
    panelWidth.value = dragStartWidth - deltaX
  }
  if (resizeDirection.includes('right')) {
    panelWidth.value = dragStartWidth + deltaX
  }

  if (panelHeight.value < 100) panelHeight.value = 100
  if (panelWidth.value < 300) panelWidth.value = 300
}
</script>


<style scoped>
.tabbed-actions-wrapper {
  position: fixed;
  bottom: 0;
  right: 0;
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  z-index: 50;
  pointer-events: none;
}

.resize-handle {
  position: absolute;
  background: transparent;
  z-index: 100;
  pointer-events: auto;
  user-select: none;
}

.resize-top {
  top: 0;
  left: 0;
  right: 0;
  height: 6px;
  cursor: row-resize;
}

.resize-bottom {
  bottom: 0;
  left: 0;
  right: 0;
  height: 6px;
  cursor: row-resize;
}

.resize-left {
  left: 0;
  top: 0;
  bottom: 0;
  width: 6px;
  cursor: col-resize;
}

.resize-right {
  right: 0;
  top: 0;
  bottom: 0;
  width: 6px;
  cursor: col-resize;
}

.resize-top-left {
  top: 0;
  left: 0;
  width: 10px;
  height: 10px;
  cursor: nwse-resize;
}

.resize-top-right {
  top: 0;
  right: 0;
  width: 10px;
  height: 10px;
  cursor: nesw-resize;
}

.resize-bottom-left {
  bottom: 0;
  left: 0;
  width: 10px;
  height: 10px;
  cursor: nesw-resize;
}

.resize-bottom-right {
  bottom: 0;
  right: 0;
  width: 10px;
  height: 10px;
  cursor: nwse-resize;
}

.resize-handle:hover {
  background: rgba(14, 99, 156, 0.5);
}

.tabbed-actions-panel {
  position: fixed;
  display: flex;
  flex-direction: column;
  background: #1e1e1e;
  color: #d4d4d4;
  border: 2px solid #0e639c;
  box-shadow: 0 -2px 8px rgba(0, 0, 0, 0.5);
  flex-shrink: 0;
  min-height: 100px;
  pointer-events: auto;
  z-index: 1000;
  width: v-bind('panelWidth + "px"');
  height: v-bind('panelHeight + "px"');
  left: v-bind('panelLeft + "px"');
  top: v-bind('panelTop + "px"');
}

.tabs-header {
  display: flex;
  align-items: center;
  background: #252526;
  border-bottom: 1px solid #333;
  flex-shrink: 0;
  gap: 8px;
  padding: 0 8px;
  overflow-x: auto;
  overflow-y: hidden;
  height: 40px;
  cursor: grab;
  scroll-behavior: smooth;
  position: relative;
}

/* Красивый скролл-бар */
.tabs-header::-webkit-scrollbar {
  height: 6px;
}

.tabs-header::-webkit-scrollbar-track {
  background: transparent;
}

.tabs-header::-webkit-scrollbar-thumb {
  background: #555;
  border-radius: 3px;
  transition: background 0.2s;
}

.tabs-header::-webkit-scrollbar-thumb:hover {
  background: #777;
}

/* Firefox */
.tabs-header {
  scrollbar-color: #555 transparent;
  scrollbar-width: thin;
}

.tabs-header:active {
  cursor: grabbing;
}

.tabs-container {
  display: flex;
  gap: 4px;
  flex: 1;
}

.tab {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  background: #1e1e1e;
  border: 1px solid #3e3e42;
  border-bottom: none;
  border-radius: 4px 4px 0 0;
  cursor: pointer;
  color: #858585;
  font-size: 13px;
  white-space: nowrap;
  transition: all 0.15s;
  user-select: none;
}

.tab:hover {
  background: #2d2d30;
  color: #d4d4d4;
}

.tab.active {
  background: #1e1e1e;
  color: #d4d4d4;
  border-color: #0e639c;
  border-bottom-color: #1e1e1e;
}

.tab-icon {
  font-size: 14px;
  flex-shrink: 0;
}

.tab-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tab-close {
  padding: 0 4px;
  background: transparent;
  border: none;
  color: inherit;
  cursor: pointer;
  font-size: 12px;
  flex-shrink: 0;
  opacity: 0.6;
  transition: opacity 0.15s;
}

.tab-close:hover {
  opacity: 1;
  color: #ff6b6b;
}

.actions-content {
  flex: 1;
  overflow: hidden;
  position: relative;
}

.action-wrapper {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: none;
  flex-direction: column;
  opacity: 0;
  transition: opacity 0.2s;
}

.action-wrapper.active {
  display: flex;
  opacity: 1;
}

.action-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 16px;
  border-bottom: 1px solid #333;
  background: #252526;
  flex-shrink: 0;
  gap: 16px;
}

.action-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  flex: 1;
  min-width: 0;
}

.action-icon {
  font-size: 16px;
  flex-shrink: 0;
}

.action-controls {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-shrink: 0;
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

.status-indicator {
  font-size: 12px;
}

.btn {
  padding: 4px 12px;
  font-size: 12px;
  background: #0e639c;
  color: white;
  border: none;
  border-radius: 3px;
  cursor: pointer;
  transition: background 0.15s;
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

.terminal-container {
  flex: 1;
  overflow: hidden;
  padding: 4px;
  display: flex;
  justify-content: flex-start;
  align-items: flex-start;
}

:deep(.xterm-screen) {
  margin: 0 !important;
  margin-left: 0 !important;
  margin-right: auto !important;
}

:deep(.xterm-viewport) {
  width: 100% !important;
}

:deep(.xterm) {
  width: 100% !important;
  height: 100% !important;
  text-align: left;
}


:deep(.xterm textarea) {
  text-align: left;
}

.logs-container {
  flex: 1;
  overflow: auto;
  overflow-y: auto;
  padding: 8px;
  scrollbar-color: #555 transparent;
  scrollbar-width: thin;
}

.logs-container::-webkit-scrollbar {
  width: 6px;
}

.logs-container::-webkit-scrollbar-track {
  background: transparent;
}

.logs-container::-webkit-scrollbar-thumb {
  background: #555;
  border-radius: 3px;
}

.logs-container::-webkit-scrollbar-thumb:hover {
  background: #777;
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
  text-align: left;
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
