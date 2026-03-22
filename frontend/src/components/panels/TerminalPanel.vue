<template>
  <div class="terminal-panel">
    <div class="terminal-header">
      <h3>🖥 Terminal</h3>
      <div class="terminal-info">
        <select :value="container" @change="onContainerChange" class="log-select">
          <option v-for="c in containers" :key="c" :value="c">{{ c }}</option>
        </select>
        <span class="terminal-shell">{{ shell }}</span>
        <span class="terminal-status">{{ statusIcon }}</span>
        <button class="btn btn-sm" @click="reconnect" :disabled="connecting">
          🔄 Reconnect
        </button>
        <button class="btn-close" @click="$emit('close')">✕</button>
      </div>
    </div>
    <div class="terminal-container" ref="terminalRef"></div>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import 'xterm/css/xterm.css'

import {
  StartTerminalSession,
  ReadTerminal,
  WriteTerminal,
  CloseTerminal,
  DetectShell,
  ResizeTerminal,
} from '../../../wailsjs/go/main/App'

const props = defineProps<{
  namespace: string
  pod: string
  containers: string[]
  initialContainer: string
}>()

defineEmits<{ (e: 'close'): void }>()

const container = ref(props.initialContainer)
const shell = ref('/bin/sh')
const terminalRef = ref<HTMLElement>()
const connecting = ref(false)

type Status = 'connected' | 'disconnected' | 'connecting'
const status = ref<Status>('disconnected')

const statusIcon = computed(() => {
  switch (status.value) {
    case 'connected': return '🟢'
    case 'connecting': return '🟡'
    case 'disconnected': return '🔴'
    default: return '🔴'
  }
})

let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
let sessionID = ''
let readLoopAborted = false
let resizeObserver: ResizeObserver | null = null
let resizeTimer: ReturnType<typeof setTimeout> | null = null

const DEFAULT_COLS = 80
const DEFAULT_ROWS = 24

// ─── Lifecycle ─────────────────────────────────────────

onMounted(async () => {
  await initTerminal()
})

onUnmounted(() => {
  cleanup()
})

watch(container, async () => {
  cleanup()
  await nextTick()
  await initTerminal()
})

// ─── Cleanup ───────────────────────────────────────────

function cleanup() {
  readLoopAborted = true
  status.value = 'disconnected'

  if (resizeTimer) {
    clearTimeout(resizeTimer)
    resizeTimer = null
  }

  if (resizeObserver) {
    resizeObserver.disconnect()
    resizeObserver = null
  }

  if (sessionID) {
    const sid = sessionID
    sessionID = ''
    CloseTerminal(sid).catch(() => {})
  }

  if (terminal) {
    terminal.dispose()
    terminal = null
  }
  fitAddon = null
}

// ─── Init ──────────────────────────────────────────────

async function initTerminal() {
  if (!terminalRef.value || connecting.value) return

  connecting.value = true
  status.value = 'connecting'

  try {
    try {
      shell.value = await DetectShell(props.namespace, props.pod, container.value)
    } catch {
      shell.value = '/bin/sh'
    }

    terminal = new Terminal({
      fontFamily: 'Monaco, Menlo, "Courier New", monospace',
      fontSize: 13,
      lineHeight: 1.2,
      theme: {
        background: '#1e1e1e',
        foreground: '#d4d4d4',
        cursor: '#00ff00',
        selectionBackground: 'rgba(100, 100, 100, 0.5)',
      },
      cursorBlink: true,
      disableStdin: false,
      convertEol: false,
      screenReaderMode: false,
      cols: DEFAULT_COLS,
      rows: DEFAULT_ROWS,
      allowProposedApi: true,
    })

    fitAddon = new FitAddon()
    terminal.loadAddon(fitAddon)
    terminal.open(terminalRef.value)

    await nextTick()
    forceXtermFocus()
    safeFit()

    sessionID = `term_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
    const cols = terminal.cols
    const rows = terminal.rows

    await StartTerminalSession(
      sessionID, props.namespace, props.pod,
      container.value, shell.value, cols, rows
    )

    status.value = 'connected'

    // Input → backend
    terminal.onData((data: string) => {
      if (!sessionID) return
      const bytes = Array.from(new TextEncoder().encode(data))
      WriteTerminal(sessionID, bytes).catch(() => {})
    })

    terminal.onBinary((data: string) => {
      if (!sessionID) return
      const bytes: number[] = []
      for (let i = 0; i < data.length; i++) {
        bytes.push(data.charCodeAt(i))
      }
      WriteTerminal(sessionID, bytes).catch(() => {})
    })

    // Resize
    terminal.onResize(({ cols, rows }) => {
      debouncedResize(cols, rows)
    })

    resizeObserver = new ResizeObserver(() => safeFit())
    resizeObserver.observe(terminalRef.value)

    terminalRef.value.addEventListener('mousedown', () => forceXtermFocus())
    terminalRef.value.addEventListener('click', () => forceXtermFocus())

    readLoop(sessionID)
    forceXtermFocus()

  } catch (e: any) {
    status.value = 'disconnected'
    terminal?.write(`\r\nConnection error: ${e?.message || e}\r\n`)
  } finally {
    connecting.value = false
  }
}

// ─── Focus ─────────────────────────────────────────────

function forceXtermFocus() {
  if (!terminal) return
  terminal.focus()

  const textarea = terminalRef.value?.querySelector('textarea.xterm-helper-textarea') as HTMLTextAreaElement | null
  if (textarea) {
    textarea.focus()
  }
}

// ─── Read loop ─────────────────────────────────────────

async function readLoop(capturedSessionID: string) {
  readLoopAborted = false

  while (!readLoopAborted && capturedSessionID === sessionID && terminal) {
    try {
      const rawResult: any = await ReadTerminal(capturedSessionID)

      if (capturedSessionID !== sessionID || readLoopAborted || !terminal) break

      if (rawResult === null || rawResult === undefined) {
        terminal.write('\r\n[Session ended]\r\n')
        status.value = 'disconnected'
        break
      }

      // Wails returns []byte as base64 string
      if (typeof rawResult === 'string') {
        if (rawResult.length === 0) {
          terminal.write('\r\n[Session ended]\r\n')
          status.value = 'disconnected'
          break
        }

        try {
          const binary = atob(rawResult)
          terminal.write(binary)
        } catch {
          terminal.write(rawResult)
        }
        continue
      }

      if (Array.isArray(rawResult) && rawResult.length > 0) {
        const text = new TextDecoder('utf-8', { fatal: false }).decode(new Uint8Array(rawResult))
        terminal.write(text)
      } else {
        terminal.write('\r\n[Session ended]\r\n')
        status.value = 'disconnected'
        break
      }
    } catch (e: any) {
      if (!readLoopAborted && capturedSessionID === sessionID) {
        terminal?.write(`\r\n[Connection lost: ${e?.message || e}]\r\n`)
        status.value = 'disconnected'
      }
      break
    }
  }
}

// ─── Resize ────────────────────────────────────────────

function debouncedResize(cols: number, rows: number) {
  if (!sessionID) return

  if (resizeTimer) clearTimeout(resizeTimer)
  resizeTimer = setTimeout(async () => {
    try {
      await ResizeTerminal(sessionID, cols, rows)
    } catch {
      // ignore
    }
  }, 150)
}

function safeFit() {
  if (!terminal || !fitAddon) return
  try {
    fitAddon.fit()
  } catch {
    // ignore
  }
}

// ─── Reconnect ─────────────────────────────────────────

async function reconnect() {
  cleanup()
  await nextTick()
  await initTerminal()
}

function onContainerChange(e: Event) {
  container.value = (e.target as HTMLSelectElement).value
}
</script>

<style scoped>
.terminal-panel {
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

.terminal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 16px;
  border-bottom: 1px solid #333;
  background: #252526;
  flex-shrink: 0;
}

.terminal-header h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  flex: 1;
}

.terminal-info {
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

.terminal-shell {
  font-size: 12px;
  color: #999;
  padding: 0 8px;
}

.terminal-status {
  font-size: 10px;
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

.terminal-container {
  flex: 1;
  overflow: hidden;
  padding: 4px;
}

:deep(.xterm) {
  width: 100% !important;
  height: 100% !important;
}

:deep(.xterm-viewport) {
  width: 100% !important;
  height: 100% !important;
  overflow-y: auto !important;
}

:deep(.xterm-screen) {
  width: 100% !important;
  height: 100% !important;
}

:deep(.xterm-helper-textarea) {
  position: absolute !important;
  opacity: 0 !important;
  left: -9999px !important;
  top: 0 !important;
  width: 0 !important;
  height: 0 !important;
  z-index: -10 !important;
}
</style>