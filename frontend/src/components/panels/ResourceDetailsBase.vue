<template>
  <div class="detail-panel">
    <div class="detail-header">
      <div class="header-title">
        <span v-if="resourceIcon" class="resource-icon">{{ resourceIcon }}</span>
        <h3>{{ title || resource?.name || 'Resource Details' }}</h3>
      </div>
      <div class="detail-actions-bar">
        <button ref="menuButtonRef" class="btn-action" @click.stop="toggleContextMenu" title="Actions">
          <span class="menu-dots">⋮</span>
        </button>
        <button class="btn-close" @click="emit('close')" title="Close">
          <img :src="closeIcon" alt="Close" class="icon-close" />
        </button>
      </div>
    </div>
    <div v-if="showContextMenu" class="menu-overlay" @click="hideContextMenu"></div>
    <div v-if="showContextMenu" :style="contextMenuStyle" class="context-menu" @click.stop>
      <div class="ctx-menu-item" @click="handleContextAction('edit')"><img :src="editIcon" alt="Edit"
          class="ctx-menu-icon" /> Edit</div>
      <div class="ctx-menu-item ctx-menu-danger" @click="handleContextAction('delete')"><img :src="deleteIcon"
          alt="Delete" class="ctx-menu-icon" /> Delete</div>
      <div class="ctx-menu-item" @click="handleContextAction('close')"><img :src="closeIcon" alt="Close"
          class="ctx-menu-icon" /> Close</div>
    </div>
    <div v-if="showMetaInfo" class="detail-meta">
      <div class="meta-row"><span class="meta-label">Name:</span>
        <div class="name-with-copy"><span>{{ resource.name }}</span><button class="btn-copy"
            @click="copyToClipboard(resource.name)" title="Copy name"><img :src="copyIcon" alt="Copy"
              class="icon-inline" /></button></div>
      </div>
      <div v-if="resource.namespace" class="meta-row"><span class="meta-label">Namespace:</span><a class="meta-link"
          @click.prevent="onNamespaceClick">{{ resource.namespace }}</a></div>
      <div v-if="resource.creationTimestamp" class="meta-row"><span class="meta-label">Age:</span>
        <div class="age-display">
          <div class="age-relative">{{ formatAgeRelative(resource.creationTimestamp) }}</div>
          <div class="age-absolute">{{ formatTimestamp(resource.creationTimestamp) }}</div>
        </div>
      </div>
      <div v-if="resource.uid" class="meta-row"><span class="meta-label">UID:</span>
        <div class="uid-with-copy"><span class="uid-value">{{ resource.uid }}</span><button class="btn-copy"
            @click="copyToClipboard(resource.uid)" title="Copy UID"><img :src="copyIcon" alt="Copy"
              class="icon-inline" /></button></div>
      </div>
    </div>
    <div class="tabs-container">
      <div class="tabs">
        <button :class="['tab', { active: activeTab === 'properties' }]"
          @click="activeTab = 'properties'">Properties</button>
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
const props = defineProps<{ resource: any; resourceIcon?: string; title?: string }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'edit'): void; (e: 'delete'): void; (e: 'namespaceClick', namespace: string): void }>()
const activeTab = ref('properties'), showContextMenu = ref(false), contextMenuStyle = ref<Record<string, string>>({}), menuButtonRef = ref<HTMLElement | null>(null)
const showMetaInfo = computed(() => props.resource && props.resource.name)
function setActiveTab(tabId: string) { activeTab.value = tabId }
function toggleContextMenu() { if (showContextMenu.value) hideContextMenu(); else { showContextMenu.value = true; nextTick(() => calculateMenuPosition()) } }
function hideContextMenu() { showContextMenu.value = false }
function calculateMenuPosition() { if (!menuButtonRef.value) return; const btn = menuButtonRef.value, rect = btn.getBoundingClientRect(), menuW = 180, menuH = 130; let left = rect.right, top = rect.bottom; if (left + menuW > window.innerWidth) left = rect.left - menuW; if (top + menuH > window.innerHeight) top = rect.top - menuH; contextMenuStyle.value = { position: 'absolute', top: top + 'px', left: left + 'px', zIndex: '10000' } }
function handleContextAction(action: string) { if (action === 'edit') emit('edit'); else if (action === 'delete') emit('delete'); else if (action === 'close') emit('close'); hideContextMenu() }
function onNamespaceClick() { if (props.resource?.namespace) emit('namespaceClick', props.resource.namespace) }
function copyToClipboard(text: string) { navigator.clipboard.writeText(text).catch(e => console.error('Copy failed:', e)) }
function formatAgeRelative(ts: string): string { if (!ts) return '—'; const d = new Date(ts), now = new Date(), diff = Math.floor((now.getTime() - d.getTime()) / 1000); if (diff < 60) return diff + 's'; const m = Math.floor(diff / 60); if (m < 60) return m + 'm'; const h = Math.floor(m / 60); if (h < 24) return h + 'h'; const days = Math.floor(h / 24); if (days < 7) return days + 'd'; const w = Math.floor(days / 7); if (w < 4) return w + 'w'; const mo = Math.floor(days / 30); if (mo < 12) return mo + 'mo'; return Math.floor(days / 365) + 'y' }
function formatTimestamp(ts: string): string { if (!ts) return ''; try { return new Date(ts).toLocaleString() } catch { return ts } }
function formatCpu(cpu: string | number): string { if (typeof cpu === 'number') cpu = cpu.toString(); if (cpu.endsWith('m')) { const m = parseInt(cpu.slice(0, -1)); return m >= 1000 ? (m / 1000).toFixed(1) + ' cores' : m + 'm' } const c = parseFloat(cpu); return isNaN(c) ? cpu : c >= 1 ? c.toFixed(1) + ' cores' : Math.round(c * 1000) + 'm' }
function formatMem(mem: string | number): string { if (typeof mem === 'number') mem = mem.toString(); const U: Record<string, number> = { Ki: 1024, Mi: 1048576, Gi: 1073741824, Ti: 1099511627776, K: 1000, M: 1000000, G: 1000000000, T: 1000000000000 }; for (const [u, m] of Object.entries(U)) if (mem.endsWith(u)) return fmtBytes(parseInt(mem.slice(0, -u.length)) * m); const b = parseInt(mem); return isNaN(b) ? mem : fmtBytes(b) }
function fmtBytes(b: number): string { const U = ['B', 'Ki', 'Mi', 'Gi', 'Ti']; let i = 0, v = b; while (v >= 1024 && i < U.length - 1) { v /= 1024; i++ } return v.toFixed(1) + ' ' + U[i] }
function formatConditions(conds: any[]): string { if (!conds?.length) return 'No conditions'; return conds.map(c => (c.type || 'Unknown') + ': ' + (c.status || 'Unknown') + (c.reason ? ' (' + c.reason + ')' : '')).join(' | ') }
function getStatusClass(s: string): string { const m: Record<string, string> = { running: 'status-running', succeeded: 'status-succeeded', completed: 'status-completed', pending: 'status-pending', failed: 'status-failed', crashed: 'status-crashed', crashloopbackoff: 'status-crashed', imagepullbackoff: 'status-warning', evicted: 'status-failed', terminating: 'status-terminating', unknown: 'status-unknown', available: 'status-available', unavailable: 'status-unavailable' }; return m[s?.toLowerCase()] || 'status-unknown' }
defineExpose({ formatResources_cpu: formatCpu, formatResources_memory: formatMem, formatConditions, getStatusClass, formatAgeRelative, formatTimestamp, copyToClipboard })
</script>
<style scoped>
.detail-panel {
  background: #1a1b2e;
  color: #d4d4d4;
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden
}

.detail-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid #2a2b40;
  background: #1a1b2e;
  gap: 12px;
  flex-shrink: 0
}

.header-title {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
  min-width: 0
}

.resource-icon {
  font-size: 1.25rem;
  flex-shrink: 0
}

.detail-header h3 {
  margin: 0;
  color: white;
  font-size: 1.1rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis
}

.detail-actions-bar {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0
}

.btn-action {
  padding: 6px 10px;
  background: transparent;
  border: none;
  border-radius: 4px;
  color: white;
  cursor: pointer;
  font-size: 1.2rem;
  transition: all .15s;
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 36px;
  height: 36px
}

.btn-action:hover {
  background: rgba(255, 255, 255, .1)
}

.menu-dots {
  font-size: 1.2rem;
  line-height: 1
}

.btn-close {
  padding: 6px 10px;
  background: transparent;
  border: none;
  border-radius: 4px;
  color: white;
  cursor: pointer;
  font-size: 1rem;
  transition: all .15s;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 32px;
  height: 32px
}

.btn-close:hover {
  background: rgba(255, 255, 255, .05)
}

.icon-close {
  width: 24px;
  height: 24px;
  filter: invert(1);
  opacity: .8
}

.btn-close:hover .icon-close {
  opacity: 1
}

.menu-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 9998;
  cursor: default
}

.context-menu {
  position: fixed;
  background: #1e1f35;
  border: 1px solid #3b3d5c;
  border-radius: 4px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, .3);
  z-index: 9999;
  min-width: 180px;
  padding: 4px 0
}

.ctx-menu-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 16px;
  color: #d4d4d4;
  cursor: pointer;
  font-size: .875rem;
  transition: background .15s
}

.ctx-menu-item:hover {
  background: rgba(255, 255, 255, .1);
  color: white
}

.ctx-menu-danger {
  color: #ef4444
}

.ctx-menu-danger:hover {
  background: rgba(239, 68, 68, .2)
}

.ctx-menu-icon {
  width: 16px;
  height: 16px;
  filter: invert(1);
  opacity: .85
}

.detail-meta {
  padding: 12px 16px;
  border-bottom: 1px solid #2a2b40;
  background: rgba(255, 255, 255, .02);
  flex-shrink: 0
}

.meta-row {
  display: grid;
  grid-template-columns: auto 1fr;
  align-items: baseline;
  margin-bottom: 6px;
  gap: 12px;
  font-size: .875rem
}

.meta-row:last-child {
  margin-bottom: 0
}

.meta-label {
  font-weight: bold;
  color: #d4d4d4;
  white-space: nowrap
}

.meta-link {
  color: #60a5fa;
  cursor: pointer;
  text-decoration: underline;
  text-decoration-color: #60a5fa
}

.meta-link:hover {
  color: #93c5fd
}

.name-with-copy,
.uid-with-copy {
  display: flex;
  align-items: center;
  gap: 8px
}

.btn-copy {
  padding: 4px 6px;
  background: transparent;
  border: none;
  border-radius: 3px;
  color: #d4d4d4;
  cursor: pointer;
  font-size: .9rem;
  transition: all .15s;
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 32px;
  height: 32px;
  flex-shrink: 0
}

.btn-copy:hover {
  background: rgba(255, 255, 255, .05)
}

.icon-inline {
  width: 10px;
  height: 10px;
  filter: invert(1);
  opacity: .8
}

.age-display {
  display: flex;
  flex-direction: column;
  gap: 4px
}

.age-relative {
  font-weight: 500;
  color: #d4d4d4
}

.age-absolute {
  font-size: .8rem;
  color: #868d8a
}

.tabs-container {
  overflow-x: auto;
  flex-shrink: 0
}

.tabs {
  display: flex;
  gap: 4px;
  padding: 0 16px;
  border-bottom: 1px solid #2a2b40
}

.tab {
  padding: 10px 16px;
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  color: #868d8a;
  cursor: pointer;
  font-size: .875rem;
  transition: color .15s, border-color .15s
}

.tab:hover {
  color: #d4d4d4
}

.tab.active {
  border-bottom-color: #3b82f6;
  color: white
}

.detail-content {
  flex: 1;
  overflow-y: auto;
  padding: 16px
}

.tab-content {
  color: #d4d4d4
}

::-webkit-scrollbar {
  width: 8px;
  height: 8px
}

::-webkit-scrollbar-track {
  background: #1a1b2e
}

::-webkit-scrollbar-thumb {
  background: #35365a;
  border-radius: 4px
}

::-webkit-scrollbar-thumb:hover {
  background: #45466a
}
</style>
