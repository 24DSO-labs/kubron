<template>
<div class="logs-view">
  <div class="logs-toolbar">
    <div class="toolbar-group">
      <label>Pods:</label>
      <select :value="tab.selectedPodName" @change="onPodChange($event)"
        class="select-pod">
        <option value="">All Pods</option>
        <option v-for="pod in tab.pods" :key="pod.name" :value="pod.name">{{ pod.name }}</option>
      </select>
    </div>
    <div class="toolbar-group">
      <label>Containers:</label>
      <select :value="tab.selectedContainer" @change="onContainerChange($event)"
        class="select-container">
        <option value="">All Containers</option>
        <option v-for="container in availableContainers" :key="container" :value="container">
          {{ container }}</option>
      </select>
    </div>
    <div class="toolbar-group">
      <label>Lines:</label>
      <input v-model.number="logLines" type="number" min="10" max="10000" @change="loadLogs()"
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
      <button @click="loadLogs()" class="btn-action btn-refresh"><img :src="restartIcon"
          class="icon-svg" /></button>
      <button @click="downloadLogs()" class="btn-action"><img :src="downloadIcon"
          class="icon-svg" /></button>
    </div>
  </div>
  <div class="logs-info"><span>Namespace: <strong>{{ tab.namespace }}</strong> | {{ tab.resourceType }}:
      <strong>{{ tab.resourceName }}</strong> | Pods: <strong>{{ tab.pods.length }}</strong></span>
  </div>
  <div class="logs-content">
    <div v-if="tab.loading" class="logs-loading">⏳ Loading...</div>
    <pre v-else-if="tab.rawLogs" class="logs-text" :class="{ 'no-wrap': !wrapLines }"
      v-html="highlightedLogs"></pre>
    <div v-else class="logs-empty">{{ tab.pods.length ? 'No logs' : 'No pods found' }}</div>
  </div>
</div>
</template>

<script lang="ts" setup>
import { ref, computed, watch, onBeforeUnmount } from 'vue'
import { GetPodLogs } from '../../../wailsjs/go/main/App'
import { getWorkloadPods } from '../../resources/workloadPods'
import { containersOf } from '../../resources/utils'
import type { LogsTab } from '../../composables/useTabbedPanel'
import restartIcon from '../../assets/icons/restart.svg'
import downloadIcon from '../../assets/icons/download.svg'
import timestampIcon from '../../assets/icons/timestamp.svg'
import wrapLinesIcon from '../../assets/icons/wrap-lines.svg'

const props = defineProps<{ tab: LogsTab; active: boolean }>()

const AUTO_REFRESH_MS = 45000
const TS_RE = /^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z[+-]?\d{0,2}:?\d{0,2})\s+(.*)$/

// Настройки просмотра относятся к конкретной вкладке
const logLines = ref(100)
const searchText = ref('')
const showTimestamp = ref(false)
const wrapLines = ref(true)

/** Контейнер по умолчанию: обычный (не init), иначе первый из списка */
function defaultContainer(pod: any): string {
  return containersOf(pod)[0]?.name || pod?.containers?.[0]?.name || ''
}

const availableContainers = computed<string[]>(() => {
  const { selectedPodName, pods } = props.tab
  if (!selectedPodName) {
    const names = new Set<string>()
    pods.forEach(pod => pod.containers?.forEach(c => names.add(c.name)))
    return Array.from(names).sort()
  }
  return pods.find(p => p.name === selectedPodName)?.containers?.map(c => c.name) ?? []
})

const displayedLogs = computed(() => {
  const raw = props.tab.rawLogs
  if (!raw) return ''
  let lines = raw.split('\n').map(line => {
    if (!line.trim()) return line
    const m = line.match(TS_RE)
    if (!m) return line
    return showTimestamp.value ? `[${m[1]}] ${m[2]}` : m[2]
  })
  if (searchText.value) {
    const search = searchText.value.toLowerCase()
    lines = lines.filter(line => line.toLowerCase().includes(search))
  }
  return lines.join('\n')
})

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

// HTML для v-html: вывод контейнера недоверенный, поэтому всегда экранируется;
// искомая строка экранируется так же, чтобы совпадать с экранированным текстом.
const highlightedLogs = computed(() => {
  const logs = escapeHtml(displayedLogs.value)
  if (!searchText.value) return logs
  const term = escapeHtml(searchText.value).replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  return logs.replace(new RegExp(`(${term})`, 'gi'), '<mark class="search-highlight">$1</mark>')
})

async function loadResourcePods() {
  const tab = props.tab
  try {
    tab.loading = true
    // Поды ресурса определяются по владельцу (см. resources/workloadPods.ts)
    tab.pods = await getWorkloadPods(tab.resourceType, tab.namespace, tab.resourceName)
    if (tab.pods.length > 0 && !tab.selectedPodName) {
      tab.selectedPodName = tab.pods[0].name
      tab.selectedContainer = defaultContainer(tab.pods[0])
    }
    if (tab.selectedPodName || tab.pods.length > 0) await loadLogs()
  } catch (e: any) { tab.rawLogs = `Error: ${e}` }
  finally { tab.loading = false }
}

async function onPodChange(event: Event) {
  const tab = props.tab
  tab.selectedPodName = (event.target as HTMLSelectElement).value
  tab.selectedContainer = ''
  if (tab.selectedPodName) {
    const pod = tab.pods.find(p => p.name === tab.selectedPodName)
    if (pod) tab.selectedContainer = defaultContainer(pod)
  }
  await loadLogs()
}

async function onContainerChange(event: Event) {
  props.tab.selectedContainer = (event.target as HTMLSelectElement).value
  await loadLogs()
}

async function loadLogs() {
  const tab = props.tab
  if (!tab.selectedPodName && tab.pods.length === 0) return
  try {
    tab.loading = true
    if (tab.selectedPodName) {
      const pod = tab.pods.find(p => p.name === tab.selectedPodName)
      const container = tab.selectedContainer || defaultContainer(pod)
      tab.rawLogs = await GetPodLogs(tab.namespace, tab.selectedPodName, container, logLines.value)
    } else {
      // Все поды сразу: запросы идут параллельно, ошибка одного пода не ломает остальные
      const parts = await Promise.all(tab.pods.map(async pod => {
        const container = tab.selectedContainer || defaultContainer(pod)
        try {
          const logs = await GetPodLogs(tab.namespace, pod.name, container, logLines.value)
          return logs ? `\n=== ${pod.name} ===\n${logs}` : ''
        } catch (e: any) {
          return `\n=== ${pod.name} (error) ===\n${e}`
        }
      }))
      tab.rawLogs = parts.join('')
    }
  } catch (e: any) { tab.rawLogs = `Error: ${e}` }
  finally { tab.loading = false }
}

function downloadLogs() {
  const tab = props.tab
  const element = document.createElement('a')
  element.setAttribute('href', 'data:text/plain;charset=utf-8,' + encodeURIComponent(displayedLogs.value))
  element.setAttribute('download', `${tab.resourceType}-${tab.resourceName}-logs.txt`)
  element.style.display = 'none'
  document.body.appendChild(element)
  element.click()
  document.body.removeChild(element)
}

// Первая загрузка — когда вкладка впервые становится активной
watch(() => props.active, active => {
  if (active && !props.tab.rawLogs && !props.tab.loading) loadResourcePods()
}, { immediate: true })

// Автообновление только для видимой вкладки
const refreshTimer = setInterval(() => {
  if (props.active && !props.tab.loading) loadLogs()
}, AUTO_REFRESH_MS)
onBeforeUnmount(() => clearInterval(refreshTimer))
</script>

<style scoped>
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
</style>
