<template>
  <div class="terminal-panel">
    <div class="terminal-header">
      <h3>🖥 Terminal</h3>
      <div class="terminal-info">
        <select :value="container" @change="onContainerChange" class="log-select">
          <option v-for="c in containers" :key="c" :value="c">{{ c }}</option>
        </select>
        <span class="terminal-shell">{{ shell }}</span>
        <button class="btn btn-sm" @click="clearTerminal">🗑 Clear</button>
        <button class="btn-close" @click="$emit('close')">✕</button>
      </div>
    </div>

    <div class="terminal-body" ref="terminalBody" @click="focusInput">
      <div v-for="(entry, i) in history" :key="i" class="terminal-entry">
        <div v-if="entry.type === 'input'" class="terminal-input-line">
          <span class="terminal-prompt">{{ entry.prompt }}</span>
          <span class="terminal-cmd">{{ entry.text }}</span>
        </div>
        <pre v-else-if="entry.type === 'output'" class="terminal-output-text">{{ entry.text }}</pre>
        <pre v-else-if="entry.type === 'error'" class="terminal-error-text">{{ entry.text }}</pre>
        <div v-else-if="entry.type === 'info'" class="terminal-info-text">{{ entry.text }}</div>
      </div>

      <div class="terminal-input-line terminal-current">
        <span class="terminal-prompt">{{ promptStr }}</span>
        <input
          ref="inputRef"
          v-model="currentCmd"
          @keydown="onKeyDown"
          class="terminal-input"
          spellcheck="false"
          autocomplete="off"
        />
        <span v-if="running" class="terminal-spinner">⏳</span>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted, nextTick, watch } from 'vue'

const props = defineProps<{
  namespace: string
  pod: string
  containers: string[]
  initialContainer: string
}>()

const emit = defineEmits<{ (e: 'close'): void }>()

interface HistoryEntry {
  type: 'input' | 'output' | 'error' | 'info'
  text: string
  prompt?: string
}

const container = ref(props.initialContainer)
const shell = ref('/bin/sh')
const cwd = ref('/')
const userHost = ref('')
const history = ref<HistoryEntry[]>([])
const currentCmd = ref('')
const cmdHistory = ref<string[]>([])
const cmdHistoryIndex = ref(-1)
const running = ref(false)

const terminalBody = ref<HTMLElement>()
const inputRef = ref<HTMLInputElement>()

const promptStr = computed(() => {
  if (userHost.value) {
    return `${userHost.value}:${cwd.value}$ `
  }
  return `${cwd.value}$ `
})

onMounted(async () => {
  await initTerminal()
})

watch(container, async () => {
  history.value = []
  await initTerminal()
})

async function execInPod(command: string): Promise<string> {
  const { ExecTerminal } = await import('../../../wailsjs/go/main/App')
  const result = await ExecTerminal(
    props.namespace, props.pod, container.value, shell.value, command
  )
  return result.stdout || ''
}

async function initTerminal() {
  try {
    const { DetectShell } = await import('../../../wailsjs/go/main/App')
    shell.value = await DetectShell(props.namespace, props.pod, container.value)

    // Получаем user и hostname отдельными простыми командами
    let u = 'user'
    let h = props.pod.split('-').slice(0, 2).join('-')

    try {
      const whoami = (await execInPod('whoami')).trim()
      if (whoami) u = whoami
    } catch (e) {}

    try {
      const hn = (await execInPod('hostname')).trim()
      if (hn) h = hn
    } catch (e) {}

    try {
      const pwd = (await execInPod('pwd')).trim()
      if (pwd) cwd.value = pwd
    } catch (e) {}

    userHost.value = `${u}@${h}`

    history.value.push({
      type: 'info',
      text: `Connected to ${container.value} in ${props.pod} (${shell.value})`
    })

    scrollToBottom()
    focusInput()
  } catch (e: any) {
    history.value.push({ type: 'error', text: 'Connection error: ' + (e.message || e) })
  }
}

async function executeCommand() {
  const cmd = currentCmd.value.trim()
  if (!cmd || running.value) return

  history.value.push({ type: 'input', text: cmd, prompt: promptStr.value })
  cmdHistory.value.unshift(cmd)
  cmdHistoryIndex.value = -1
  currentCmd.value = ''

  if (cmd === 'clear') {
    history.value = []
    scrollToBottom()
    return
  }

  if (cmd === 'exit') {
    emit('close')
    return
  }

  running.value = true

  try {
    const { ExecTerminal } = await import('../../../wailsjs/go/main/App')

    // cd в текущую директорию + команда + получаем новый pwd
    const fullCmd = `cd "${cwd.value}" 2>/dev/null; ${cmd} 2>&1; echo ":::NEWCWD:::$(pwd)"`

    const result = await ExecTerminal(
      props.namespace, props.pod, container.value, shell.value, fullCmd
    )

    let output = result.stdout || ''

    // Извлекаем новый cwd
    const marker = ':::NEWCWD:::'
    const idx = output.lastIndexOf(marker)
    if (idx >= 0) {
      const newCwd = output.slice(idx + marker.length).trim()
      if (newCwd) cwd.value = newCwd
      output = output.slice(0, idx)
    }

    if (output.trim()) {
      history.value.push({ type: 'output', text: output.trimEnd() })
    }

    if (result.stderr?.trim()) {
      history.value.push({ type: 'error', text: result.stderr.trimEnd() })
    }
  } catch (e: any) {
    history.value.push({ type: 'error', text: 'Error: ' + (e.message || e) })
  } finally {
    running.value = false
    scrollToBottom()
    focusInput()
  }
}

function onKeyDown(e: KeyboardEvent) {
  if (e.key === 'Enter') {
    e.preventDefault()
    executeCommand()
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    if (cmdHistoryIndex.value < cmdHistory.value.length - 1) {
      cmdHistoryIndex.value++
      currentCmd.value = cmdHistory.value[cmdHistoryIndex.value]
    }
  } else if (e.key === 'ArrowDown') {
    e.preventDefault()
    if (cmdHistoryIndex.value > 0) {
      cmdHistoryIndex.value--
      currentCmd.value = cmdHistory.value[cmdHistoryIndex.value]
    } else {
      cmdHistoryIndex.value = -1
      currentCmd.value = ''
    }
  } else if (e.key === 'Tab') {
    e.preventDefault()
    doTabComplete()
  } else if (e.key === 'c' && e.ctrlKey) {
    e.preventDefault()
    if (running.value) return
    currentCmd.value = ''
    history.value.push({ type: 'input', text: '^C', prompt: promptStr.value })
    scrollToBottom()
  } else if (e.key === 'l' && e.ctrlKey) {
    e.preventDefault()
    clearTerminal()
  }
}

async function doTabComplete() {
  const parts = currentCmd.value.split(/\s+/)
  const partial = parts[parts.length - 1] || ''
  if (!partial) return

  try {
    const { TabComplete } = await import('../../../wailsjs/go/main/App')
    const completions = await TabComplete(
      props.namespace, props.pod, container.value, shell.value, partial
    )
    if (completions && completions.length === 1) {
      parts[parts.length - 1] = completions[0]
      currentCmd.value = parts.join(' ')
    } else if (completions && completions.length > 1) {
      history.value.push({ type: 'output', text: completions.join('  ') })
      scrollToBottom()
    }
  } catch (e) {}
}

function clearTerminal() {
  history.value = []
}

function onContainerChange(e: Event) {
  container.value = (e.target as HTMLSelectElement).value
}

function focusInput() {
  nextTick(() => inputRef.value?.focus())
}

function scrollToBottom() {
  nextTick(() => {
    if (terminalBody.value) {
      terminalBody.value.scrollTop = terminalBody.value.scrollHeight
    }
  })
}
</script>