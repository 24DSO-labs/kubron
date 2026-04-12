<template>
  <div class="detail-panel">
    <div class="detail-header">
      <div class="header-title">
        <img v-if="resourceIcon" :src="resourceIcon" alt="" class="resource-icon" />
        <h3>{{ title || resource?.name || 'Resource Details' }}</h3>
      </div>
      <div class="detail-actions-bar">
        <!-- Кнопка Edit -->
        <button class="btn-action" title="Edit" @click.stop="emit('edit')">
          <img :src="editIcon" alt="Edit" class="btn-icon" />
        </button>
        
        <!-- Кнопка Delete -->
        <button class="btn-action btn-danger" title="Delete" @click.stop="emit('delete')">
          <img :src="deleteIcon" alt="Delete" class="btn-icon" />
        </button>
        
        <!-- Кастомные кнопки из компонента ресурса -->
        <slot name="actions"></slot>
        
        <button class="btn-close" @click="emit('close')" title="Close">
          <img :src="closeIcon" alt="Close" class="icon-close" />
        </button>
      </div>
    </div>

    <!-- Context Menu Overlay -->
    <div v-if="showContextMenu" class="menu-overlay" @click="hideContextMenu"></div>
    
    <!-- Context Menu -->
    <div v-if="showContextMenu" :style="contextMenuStyle" class="context-menu" @click.stop>
      <template v-for="(item, index) in menuItems" :key="index">
        <!-- Shell submenu -->
        <div
          v-if="item.action === 'shell' && showShellSubmenu"
          :class="['ctx-menu-item', { 'ctx-menu-danger': item.danger }]"
          @mouseenter="cancelShellMenuTimeout"
          @mouseleave="scheduleHideShellMenu"
        >
          <span class="ctx-menu-arrow"><</span>{{ item.label }}
          <div class="ctx-menu-sub">
            <div v-for="container in containers" :key="container.name" class="ctx-menu-sub-item" 
                 @click="handleContextAction({ action: `shell:${container.name}` })">
              {{ container.name }}
            </div>
          </div>
        </div>
        
        <!-- Attach submenu -->
        <div
          v-else-if="item.action === 'attach' && showAttachSubmenu"
          :class="['ctx-menu-item', { 'ctx-menu-danger': item.danger }]"
          @mouseenter="cancelAttachMenuTimeout"
          @mouseleave="scheduleHideAttachMenu"
        >
          <span class="ctx-menu-arrow"><</span>{{ item.label }}
          <div class="ctx-menu-sub">
            <div v-for="container in containers" :key="container.name" class="ctx-menu-sub-item" 
                 @click="handleContextAction({ action: `attach:${container.name}` })">
              {{ container.name }}
            </div>
          </div>
        </div>
        
        <!-- Logs submenu -->
        <div
          v-else-if="item.action === 'logs' && showLogsSubmenu"
          :class="['ctx-menu-item', { 'ctx-menu-danger': item.danger }]"
          @mouseenter="cancelLogsMenuTimeout"
          @mouseleave="scheduleHideLogsMenu"
        >
          <span class="ctx-menu-arrow"><</span>{{ item.label }}
          <div class="ctx-menu-sub">
            <div v-for="container in containers" :key="container.name" class="ctx-menu-sub-item" 
                 @click="handleContextAction({ action: `logs:${container.name}` })">
              {{ container.name }}
            </div>
          </div>
        </div>
        
        <!-- Regular menu items -->
        <div
          v-else-if="!['shell', 'attach', 'logs'].includes(item.action) || 
                     (item.action === 'shell' && !showShellSubmenu) ||
                     (item.action === 'attach' && !showAttachSubmenu) ||
                     (item.action === 'logs' && !showLogsSubmenu)"
          :class="['ctx-menu-item', { 'ctx-menu-danger': item.danger }]"
          @click="handleContextAction({ action: item.action })"
        >
          <span class="ctx-menu-arrow"></span>{{ item.label }}
        </div>
      </template>
    </div>

    <div v-if="showMetaInfo" class="detail-meta">
      <div class="meta-row">
        <span class="meta-label">Name:</span>
        <div class="name-with-copy">
          <span>{{ resource.name }}</span>
          <button class="btn-copy" @click="copyToClipboard(resource.name)" title="Copy name">
            <img :src="copyIcon" alt="Copy" class="icon-inline" />
          </button>
        </div>
      </div>
      <div v-if="resource.namespace" class="meta-row">
        <span class="meta-label">Namespace:</span>
        <a class="meta-link" @click.prevent="onNamespaceClick">{{ resource.namespace }}</a>
      </div>
      <div v-if="resource.creationTimestamp" class="meta-row">
        <span class="meta-label">Age:</span>
        <div class="age-display">
          <div class="age-relative">{{ formatAgeRelative(resource.creationTimestamp) }}</div>
          <div class="age-absolute">{{ formatTimestamp(resource.creationTimestamp) }}</div>
        </div>
      </div>
      <div v-if="resource.uid" class="meta-row">
        <span class="meta-label">UID:</span>
        <div class="uid-with-copy">
          <span class="uid-value">{{ resource.uid }}</span>
          <button class="btn-copy" @click="copyToClipboard(resource.uid)" title="Copy UID">
            <img :src="copyIcon" alt="Copy" class="icon-inline" />
          </button>
        </div>
      </div>
    </div>

    <div class="tabs-container">
      <div class="tabs">
        <button :class="['tab', { active: activeTab === 'properties' }]" @click="activeTab = 'properties'">Properties</button>
        <button :class="['tab', { active: activeTab === 'events' }]" @click="activeTab = 'events'">Events</button>
        <slot name="tabs" :resource="resource" :activeTab="activeTab" :setActiveTab="setActiveTab"></slot>
      </div>
    </div>

    <div class="detail-content">
      <div v-if="activeTab === 'properties'" class="tab-content">
        <slot name="properties" :resource="resource"></slot>
      </div>
      <div v-if="activeTab === 'events'" class="tab-content">
        <slot name="events" :resource="resource"></slot>
      </div>
      <slot name="tab-content" :resource="resource" :activeTab="activeTab"></slot>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, nextTick } from 'vue'
import editIcon from '../../assets/icons/edit.svg'
import deleteIcon from '../../assets/icons/delete.svg'
import closeIcon from '../../assets/icons/close.svg'
import copyIcon from '../../assets/icons/copy.svg'
import restartIcon from '../../assets/icons/restart.svg'

interface ContextMenuItem {
  action: string
  label: string
  danger?: boolean
}

const props = defineProps<{
  resource: any
  resourceIcon?: string
  title?: string
  resourceType?: string
  containers?: any[]
  customMenuItems?: ContextMenuItem[]
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'edit'): void
  (e: 'delete'): void
  (e: 'namespaceClick', namespace: string): void
  (e: 'contextAction', action: string, subAction?: string): void
}>()

const defaultMenuItems: ContextMenuItem[] = [
  { action: 'logs', label: 'Logs' },
  { action: 'yaml', label: 'Edit' },
  { action: 'delete', label: 'Delete', danger: true }
]

const menuItems = computed<ContextMenuItem[]>(() => {
  const items = [...defaultMenuItems]
  const hasContainers = props.containers && props.containers.length > 0
  const isPod = props.resourceType?.toLowerCase() === 'pod' || hasContainers
  
  if (isPod && hasContainers) {
    items.splice(1, 0,
      { action: 'shell', label: 'Shell' },
      { action: 'attach', label: 'Attach' }
    )
  }
  
  if (props.customMenuItems && props.customMenuItems.length > 0) {
    items.push(...props.customMenuItems)
  }
  
  return items
})

const activeTab = ref('properties')
const showContextMenu = ref(false)
const contextMenuStyle = ref<Record<string, string>>({})
const menuButtonRef = ref<HTMLElement | null>(null)
const showShellSubmenu = ref(false)
const showAttachSubmenu = ref(false)
const showLogsSubmenu = ref(false)

let shellMenuTimeout: ReturnType<typeof setTimeout> | null = null
let attachMenuTimeout: ReturnType<typeof setTimeout> | null = null
let logsMenuTimeout: ReturnType<typeof setTimeout> | null = null

const showMetaInfo = computed(() => props.resource && props.resource.name)

function setActiveTab(tabId: string) {
  activeTab.value = tabId
}

function toggleContextMenu() {
  if (showContextMenu.value) {
    hideContextMenu()
  } else {
    showContextMenu.value = true
    nextTick(() => calculateMenuPosition())
  }
}

function hideContextMenu() {
  showContextMenu.value = false
  showShellSubmenu.value = false
  showAttachSubmenu.value = false
  showLogsSubmenu.value = false
}

function calculateMenuPosition() {
  if (!menuButtonRef.value) return
  const btn = menuButtonRef.value
  const rect = btn.getBoundingClientRect()
  const menuW = 180
  const menuH = 130
  let left = rect.right
  let top = rect.bottom
  if (left + menuW > window.innerWidth) left = rect.left - menuW
  if (top + menuH > window.innerHeight) top = rect.top - menuH
  contextMenuStyle.value = {
    position: 'absolute',
    top: top + 'px',
    left: left + 'px',
    zIndex: '10000'
  }
}

function handleContextAction(menuItem: { action: string }) {
  const action = menuItem.action
  
  if (action === 'yaml') {
    emit('edit')
  } else if (action === 'delete') {
    emit('delete')
  } else if (action.startsWith('shell:') || action.startsWith('attach:') || action.startsWith('logs:')) {
    emit('contextAction', action)
  } else if (action === 'logs') {
    emit('contextAction', action)
  } else {
    if (action === 'shell') {
      showShellSubmenu.value = true
      showAttachSubmenu.value = false
      showLogsSubmenu.value = false
    } else if (action === 'attach') {
      showAttachSubmenu.value = true
      showShellSubmenu.value = false
      showLogsSubmenu.value = false
    }
    return
  }
  
  hideContextMenu()
}

function handleAction(action: string) {
  emit('contextAction', action)
}

function onNamespaceClick() {
  if (props.resource?.namespace) {
    emit('namespaceClick', props.resource.namespace)
  }
}

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text).catch(e => console.error('Copy failed:', e))
}

function scheduleHideShellMenu() {
  if (shellMenuTimeout) clearTimeout(shellMenuTimeout)
  shellMenuTimeout = setTimeout(() => { showShellSubmenu.value = false }, 150)
}

function scheduleHideAttachMenu() {
  if (attachMenuTimeout) clearTimeout(attachMenuTimeout)
  attachMenuTimeout = setTimeout(() => { showAttachSubmenu.value = false }, 150)
}

function scheduleHideLogsMenu() {
  if (logsMenuTimeout) clearTimeout(logsMenuTimeout)
  logsMenuTimeout = setTimeout(() => { showLogsSubmenu.value = false }, 150)
}

function cancelShellMenuTimeout() {
  if (shellMenuTimeout) clearTimeout(shellMenuTimeout)
}

function cancelAttachMenuTimeout() {
  if (attachMenuTimeout) clearTimeout(attachMenuTimeout)
}

function cancelLogsMenuTimeout() {
  if (logsMenuTimeout) clearTimeout(logsMenuTimeout)
}

function formatAgeRelative(ts: string | Date): string {
  if (!ts) return '—'
  const date = typeof ts === 'string' ? new Date(ts) : ts
  if (isNaN(date.getTime())) return '—'
  const now = new Date()
  const diff = Math.floor((now.getTime() - date.getTime()) / 1000)
  if (diff < 0) return '0s'
  if (diff < 60) return diff + 's'
  const m = Math.floor(diff / 60)
  if (m < 60) return m + 'm'
  const h = Math.floor(m / 60)
  if (h < 24) return h + 'h'
  const days = Math.floor(h / 24)
  if (days < 7) return days + 'd'
  const w = Math.floor(days / 7)
  if (w < 4) return w + 'w'
  const mo = Math.floor(days / 30)
  if (mo < 12) return mo + 'mo'
  return Math.floor(days / 365) + 'y'
}

function formatTimestamp(ts: string | Date): string {
  if (!ts) return ''
  const date = typeof ts === 'string' ? new Date(ts) : ts
  if (isNaN(date.getTime())) return 'Invalid Date'
  return date.toLocaleString()
}

function formatCpu(cpu: string | number): string {
  if (typeof cpu === 'number') cpu = cpu.toString()
  if (cpu.endsWith('m')) {
    const m = parseInt(cpu.slice(0, -1))
    return m >= 1000 ? (m / 1000).toFixed(1) + ' cores' : m + 'm'
  }
  const c = parseFloat(cpu)
  return isNaN(c) ? cpu : c >= 1 ? c.toFixed(1) + ' cores' : Math.round(c * 1000) + 'm'
}

function formatMem(mem: string | number): string {
  if (typeof mem === 'number') mem = mem.toString()
  const U: Record<string, number> = { Ki: 1024, Mi: 1048576, Gi: 1073741824, Ti: 1099511627776, K: 1000, M: 1000000, G: 1000000000, T: 1000000000000 }
  for (const [u, m] of Object.entries(U)) {
    if (mem.endsWith(u)) return fmtBytes(parseInt(mem.slice(0, -u.length)) * m)
  }
  const b = parseInt(mem)
  return isNaN(b) ? mem : fmtBytes(b)
}

function fmtBytes(b: number): string {
  const U = ['B', 'Ki', 'Mi', 'Gi', 'Ti']
  let i = 0
  let v = b
  while (v >= 1024 && i < U.length - 1) {
    v /= 1024
    i++
  }
  return v.toFixed(1) + ' ' + U[i]
}

function formatConditions(conds: any[]): string {
  if (!conds?.length) return 'No conditions'
  return conds.map(c => (c.type || 'Unknown') + ': ' + (c.status || 'Unknown') + (c.reason ? ' (' + c.reason + ')' : '')).join(' | ')
}

function getStatusClass(s: string): string {
  const m: Record<string, string> = {
    running: 'status-running',
    succeeded: 'status-succeeded',
    completed: 'status-completed',
    pending: 'status-pending',
    failed: 'status-failed',
    crashed: 'status-crashed',
    crashloopbackoff: 'status-crashed',
    imagepullbackoff: 'status-warning',
    evicted: 'status-failed',
    terminating: 'status-terminating',
    unknown: 'status-unknown',
    available: 'status-available',
    unavailable: 'status-unavailable'
  }
  return m[s?.toLowerCase()] || 'status-unknown'
}

defineExpose({
  formatResources_cpu: formatCpu,
  formatResources_memory: formatMem,
  formatConditions,
  getStatusClass,
  formatAgeRelative,
  formatTimestamp,
  copyToClipboard
})
</script>

<style scoped>
.detail-panel {
  background: #1a1b2e;
  color: #d4d4d4;
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.detail-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid #2a2b40;
  background: #1a1b2e;
  gap: 12px;
  flex-shrink: 0;
}

.header-title {
  display: flex;
  align-items: center;
  gap: 12px;
}

.resource-icon {
  width: 24px;
  height: 24px;
}

.header-title h3 {
  margin: 0;
  color: white;
  font-size: 1.1rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.detail-actions-bar {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.btn-action {
  padding: 6px 10px;
  background: transparent;
  border: none;
  border-radius: 4px;
  color: white;
  cursor: pointer;
  font-size: 1rem;
  transition: all 0.15s;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 32px;
  height: 32px;
}

.btn-action:hover {
  background: rgba(255, 255, 255, 0.1);
}

.btn-action .btn-icon {
  width: 24px;
  height: 24px;
  display: inline-block;
  vertical-align: middle;
  filter: invert(1);
  opacity: 0.85;
  transition: opacity 0.15s;
}

.btn-action:hover .btn-icon {
  opacity: 1;
}

.btn-action.btn-danger {
  color: #ef4444;
}

.btn-action.btn-danger:hover {
  background: rgba(239, 68, 68, 0.2);
}

.btn-action.btn-danger .btn-icon {
  filter: invert(1);
  opacity: 0.9;
}

.btn-action.btn-danger:hover .btn-icon {
  opacity: 1;
}

.icon-action {
  width: 24px;
  height: 24px;
  filter: invert(1);
}

.btn-close {
  padding: 6px 10px;
  background: transparent;
  border: none;
  border-radius: 4px;
  color: white;
  cursor: pointer;
  font-size: 1rem;
  transition: all 0.15s;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 32px;
  height: 32px;
}

.btn-close:hover {
  background: rgba(255, 255, 255, 0.05);
}

.icon-close {
  width: 28px;
  height: 28px;
  filter: invert(1);
  opacity: 0.8;
}

.btn-close:hover .icon-close {
  opacity: 1;
}

.menu-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 9998;
  cursor: default;
}

.context-menu {
  background: #1e1f35;
  border: 1px solid #3b3d5c;
  border-radius: 4px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  z-index: 9999;
  min-width: 140px;
  overflow: visible;
  padding: 0;
  position: absolute;
}

.ctx-menu-item {
  padding: 8px 12px;
  color: #d4d4d4;
  cursor: pointer;
  font-size: 0.875rem;
  transition: background 0.15s;
  padding-left: 27px;
  text-align: left;
  position: relative;
}

.ctx-menu-item:hover {
  background: rgba(255, 255, 255, 0.1);
  color: white;
}

.ctx-menu-arrow {
  position: absolute;
  left: 7px;
  top: 6px;
  color: #868d8a;
  font-size: xx-small;
}

.ctx-menu-sub {
  position: absolute;
  top: 0;
  right: 100%;
  margin-right: 4px;
  background: #1e1f35;
  border: 1px solid #3b3d5c;
  border-radius: 4px;
  min-width: 160px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  z-index: 10000;
  white-space: nowrap;
}

.ctx-menu-sub-item {
  padding: 8px 12px;
  color: #d4d4d4;
  cursor: pointer;
  font-size: 0.875rem;
  transition: background 0.15s;
  white-space: nowrap;
}

.ctx-menu-sub-item:hover {
  background: rgba(255, 255, 255, 0.1);
  color: white;
}

.ctx-menu-danger {
  color: #ef4444;
}

.ctx-menu-danger:hover {
  background: rgba(239, 68, 68, 0.2);
}

.detail-meta {
  padding: 12px 16px;
  border-bottom: 1px solid #2a2b40;
  background: rgba(255, 255, 255, 0.02);
  flex-shrink: 0;
}

.meta-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 6px;
  font-size: 14px;
}

.meta-row:last-child {
  margin-bottom: 0;
}

.meta-label {
  font-weight: bold;
  color: #d4d4d4;
  white-space: nowrap;
}

.name-with-copy,
.uid-with-copy {
  display: flex;
  align-items: center;
  gap: 8px;
}

.meta-link {
  color: #60a5fa !important;
  cursor: pointer !important;
  text-decoration: underline;
  text-decoration-color: #60a5fa;
  text-decoration-thickness: 1px;
  text-underline-offset: 2px;
  transition: all 0.15s ease;
  display: inline-block;
  background: none !important;
  border: none !important;
  padding: 0 !important;
  font: inherit !important;
  pointer-events: auto !important;
  user-select: none;
}

.meta-link:hover {
  color: #93c5fd !important;
  text-decoration-color: #93c5fd;
}

.meta-link:active {
  color: #3b82f6 !important;
  text-decoration-color: #3b82f6;
}

.meta-link:focus {
  outline: 1px solid #60a5fa;
  outline-offset: 2px;
}

.age-display {
  display: flex;
  gap: 12px;
  align-items: center;
}

.age-relative {
  font-weight: 500;
  color: #d4d4d4;
}

.age-absolute {
  font-size: 0.8rem;
  color: #868d8a;
}

.uid-value {
  color: #d4d4d4;
  font-family: monospace;
  font-size: 0.875rem;
}

.btn-copy {
  padding: 4px 6px;
  background: transparent;
  border: none;
  border-radius: 3px;
  color: #d4d4d4;
  cursor: pointer;
  font-size: 0.9rem;
  transition: all 0.15s;
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 32px;
  height: 32px;
  flex-shrink: 0;
}

.btn-copy:hover .icon-inline {
  opacity: 1;
}

.btn-copy:active {
  background: rgba(59, 130, 246, 0.1);
}

.icon-inline {
  width: 10px;
  height: 10px;
  stroke: currentColor;
  filter: invert(1);
  opacity: 0.8;
}

.tabs-container {
  overflow-x: auto;
  flex-shrink: 0;
  padding: 0 16px;
}

.tabs {
  display: flex;
  gap: 1rem;
  border-bottom: 1px solid #2a2b40;
}

.tab {
  padding: 8px 16px;
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  color: #868d8a;
  cursor: pointer;
  font-size: 0.875rem;
  transition: color 0.15s, border-color 0.15s;
}

.tab:hover {
  color: #d4d4d4;
}

.tab.active {
  border-bottom-color: #3b82f6;
  color: white;
}

.detail-content {
  flex: 1;
  overflow-y: auto;
  padding: 0 16px 16px 16px;
}

.tab-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* Scrollbar styles */
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: #1a1b2e;
}

::-webkit-scrollbar-thumb {
  background: #35365a;
  border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
  background: #45466a;
}

::-webkit-scrollbar-corner {
  background: #1a1b2e;
}

.detail-content::-webkit-scrollbar {
  width: 8px;
}

.detail-content::-webkit-scrollbar-track {
  background: #1a1b2e;
}

.detail-content::-webkit-scrollbar-thumb {
  background: #35365a;
  border-radius: 4px;
}

.detail-content::-webkit-scrollbar-thumb:hover {
  background: #45466a;
}
</style>
