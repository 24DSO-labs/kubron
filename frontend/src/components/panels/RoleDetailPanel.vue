<template>
  <div class="detail-panel">
    <div class="detail-header">
      <h3>Role: {{ role.metadata?.name }}</h3>
      <div class="detail-actions-bar">
        <!-- Edit YAML -->
        <button class="btn-action" title="Edit YAML" @click="emit('edit', role.metadata?.namespace, role.metadata?.name)">
          ✏️
        </button>

        <!-- Delete -->
        <button class="btn-action btn-danger" title="Delete Role" @click="emit('delete', role.metadata?.namespace, role.metadata?.name)">
          🗑
        </button>

        <!-- Close -->
        <button class="btn-close" @click="emit('close')">✕</button>
      </div>
    </div>

    <div class="tabs flex space-x-4 border-b border-gray-700 mb-4">
      <button v-for="tab in tabs" :key="tab.id" :class="['tab', { active: activeTab === tab.id }]"
        @click="activeTab = tab.id">
        {{ tab.label }}
      </button>
    </div>

    <!-- ═══════════ PROPERTIES ═══════════ -->
    <div v-if="activeTab === 'properties'" class="detail-section text-gray-300">
      <div class="meta-row">
        <span class="meta-label">Created:</span>
        {{ formatAge(role.metadata?.creationTimestamp) }} ago ({{ formatDate(role.metadata?.creationTimestamp) }})
      </div>
      <div class="meta-row">
        <span class="meta-label">Name:</span>
        {{ role.metadata?.name }}
      </div>
      <div class="meta-row">
        <span class="meta-label">Namespace:</span>
        <a href="#" class="namespace-link" @click.prevent="$emit('namespace-click', role.metadata?.namespace)">
          {{ role.metadata?.namespace }}
        </a>
      </div>

      <div class="meta-row cursor-pointer" @click="labelsExpanded = !labelsExpanded">
        <span class="meta-label">
          Labels <span class="expand-icon" :class="{ expanded: labelsExpanded }">›</span>
        </span>
        <div class="meta-content">
          <span v-if="!labelsExpanded">{{ Object.keys(role.metadata?.labels || {}).length }} Labels</span>
          <div v-else class="labels-list">
            <span v-for="[key, value] in Object.entries(role.metadata?.labels || {})" :key="key" class="label-pair">
              <span class="label-key">{{ key }}</span>=<span class="label-value">{{ value }}</span>
            </span>
          </div>
        </div>
      </div>

      <div class="meta-row cursor-pointer" @click="annotationsExpanded = !annotationsExpanded">
        <span class="meta-label">
          Annotations <span class="expand-icon" :class="{ expanded: annotationsExpanded }">›</span>
        </span>
        <div class="meta-content">
          <span v-if="!annotationsExpanded">{{ Object.keys(role.metadata?.annotations || {}).length }} Annotations</span>
          <div v-else class="labels-list">
            <span v-for="[key, value] in Object.entries(role.metadata?.annotations || {})" :key="key" class="label-pair">
              <span class="label-key">{{ key }}</span>=<span class="label-value">{{ value }}</span>
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- ═══════════ RULES ═══════════ -->
    <div v-if="activeTab === 'rules'" class="detail-section text-gray-300">
      <div v-if="!role.rules || role.rules.length === 0" class="rules-empty">
        No rules found
      </div>
      <div v-else>
        <div v-for="(rule, idx) in role.rules" :key="idx" class="rule-item">
          <div v-for="[key, value] in Object.entries(rule)" :key="key" class="rule-field">
            <span class="rule-label">{{ formatRuleFieldName(key) }}:</span>
            <span class="rule-value">{{ formatRuleFieldValue(value) }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- ═══════════ EVENTS ═══════════ -->
    <div v-if="activeTab === 'events'" class="detail-section text-gray-300">
      <div v-if="!events || events.length === 0" class="events-empty">
        No events found
      </div>
      <div v-else>
        <div v-for="event in events" :key="event.metadata?.uid" class="event-item">
          <div class="event-header">
            <span class="event-reason" :class="eventReasonClass(event.reason)">{{ event.reason }}</span>
            <span class="event-age">{{ formatAge(event.metadata?.creationTimestamp) }} ago</span>
          </div>
          <div class="event-message">{{ event.message }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref } from 'vue'

const props = defineProps<{
  role: any
  events?: any[]
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'delete', ns: string, name: string): void
  (e: 'edit', ns: string, name: string): void
  (e: 'namespace-click', ns: string | undefined): void
}>()

// ═══════ Tabs ═══════

const activeTab = ref('properties')
const tabs = [
  { id: 'properties', label: 'Properties' },
  { id: 'rules', label: 'Rules' },
  { id: 'events', label: 'Events' },
]

// ═══════ Properties expand state ═══════

const labelsExpanded = ref(false)
const annotationsExpanded = ref(false)

// ═══════ Helpers ═══════

function formatAge(timestamp?: string): string {
  if (!timestamp) return '—'
  const date = new Date(timestamp)
  const now = new Date()
  const seconds = Math.floor((now.getTime() - date.getTime()) / 1000)
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h`
  return `${Math.floor(seconds / 86400)}d`
}

function formatDate(timestamp?: string): string {
  if (!timestamp) return '—'
  const date = new Date(timestamp)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const seconds = String(date.getSeconds()).padStart(2, '0')
  const tz = 'GMT' + (date.getTimezoneOffset() > 0 ? '-' : '+') + String(Math.abs(Math.floor(date.getTimezoneOffset() / 60))).padStart(2, '0')
  return `${day} ${month}. ${year} г., ${hours}:${minutes}:${seconds} ${tz}`
}

function formatRuleFieldName(key: string): string {
  // Преобразуем camelCase в Title Case с пробелами
  const formatted = key
    .replace(/([A-Z])/g, ' $1')
    .replace(/^./, str => str.toUpperCase())
    .trim()
  return formatted
}

function formatRuleFieldValue(value: any): string {
  if (Array.isArray(value)) {
    return value.length > 0 ? value.join(', ') : '""'
  }
  if (value === null || value === undefined) {
    return '""'
  }
  return String(value)
}

function eventReasonClass(reason?: string): string {
  if (!reason) return 'event-reason-normal'
  if (reason.toLowerCase().includes('warning')) return 'event-reason-warning'
  if (reason.toLowerCase().includes('error')) return 'event-reason-error'
  return 'event-reason-normal'
}
</script>

<style scoped>
/* ═══════ Detail Header Actions ═══════ */

.detail-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid #2a2b40;
  background: #1a1b2e;
  gap: 12px;
}

.detail-header h3 {
  margin: 0;
  flex: 1;
  color: white;
  font-size: 1.1rem;
}

.detail-actions-bar {
  display: flex;
  align-items: center;
  gap: 4px;
}

.btn-action {
  padding: 6px 10px;
  background: transparent;
  border: 1px solid #3b3d5c;
  border-radius: 4px;
  color: white;
  cursor: pointer;
  font-size: 1rem;
  transition: all 0.15s;
  white-space: nowrap;
}

.btn-action:hover {
  background: rgba(255, 255, 255, 0.1);
  border-color: #5a5c7a;
}

.btn-action.btn-danger:hover {
  background: rgba(239, 68, 68, 0.2);
  border-color: #ef4444;
}

.btn-close {
  padding: 6px 10px;
  background: transparent;
  border: 1px solid #3b3d5c;
  border-radius: 4px;
  color: #d4d4d4;
  cursor: pointer;
  font-size: 1rem;
  transition: all 0.15s;
}

.btn-close:hover {
  background: rgba(255, 255, 255, 0.1);
  border-color: #5a5c7a;
  color: white;
}

/* ═══════ Tabs ═══════ */

.tabs {
  overflow-x: auto;
}

.tab {
  padding: 8px 16px;
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  color: gray;
  cursor: pointer;
}

.tab.active {
  border-bottom: 2px solid #3b82f6;
  color: white;
}

/* ═══════ Properties meta rows ═══════ */

.meta-row {
  display: grid;
  grid-template-columns: auto 1fr;
  align-items: baseline;
  margin-bottom: 8px;
  gap: 12px;
  text-align: left;
}

.meta-label {
  font-weight: bold;
  color: #d4d4d4;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.meta-content {
  margin: 0;
  padding: 0;
  text-align: left;
}

.expand-icon {
  display: inline-block;
  transition: transform 0.2s ease;
  margin-left: 4px;
}

.expand-icon.expanded {
  transform: rotate(90deg);
}

.labels-list {
  font-size: 0.875rem;
  margin: 0;
  padding: 0;
}

.label-pair {
  display: block;
  overflow-wrap: break-word;
}

.label-key {
  color: #60a5fa;
  font-weight: 500;
}

.label-value {
  color: #e5e7eb;
}

.namespace-link {
  color: #60a5fa;
  text-decoration: none;
  cursor: pointer;
}

.namespace-link:hover {
  text-decoration: underline;
}

/* ═══════ Rules ═══════ */

.rules-empty {
  color: #6b6d8a;
  padding: 20px;
  text-align: center;
  font-style: italic;
}

.rule-item {
  padding: 12px;
  margin-bottom: 8px;
  border: 1px solid #2a2b40;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.02);
}

.rule-field {
  display: grid;
  grid-template-columns: auto 1fr;
  align-items: baseline;
  margin-bottom: 6px;
  gap: 12px;
}

.rule-label {
  font-weight: 600;
  font-size: 0.875rem;
  color: #60a5fa;
}

.rule-value {
  color: #d4d4d4;
  font-size: 0.875rem;
}

/* ═══════ Events ═══════ */

.events-empty {
  color: #6b6d8a;
  padding: 20px;
  text-align: center;
  font-style: italic;
}

.event-item {
  padding: 12px;
  margin-bottom: 8px;
  border: 1px solid #2a2b40;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.02);
}

.event-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
}

.event-reason {
  font-weight: 600;
  font-size: 0.875rem;
  padding: 2px 8px;
  border-radius: 3px;
}

.event-reason-normal {
  background: rgba(52, 211, 153, 0.2);
  color: #34d399;
}

.event-reason-warning {
  background: rgba(251, 191, 36, 0.2);
  color: #fbbf24;
}

.event-reason-error {
  background: rgba(248, 113, 113, 0.2);
  color: #f87171;
}

.event-age {
  color: #6b6d8a;
  font-size: 0.8rem;
}

.event-message {
  color: #d4d4d4;
  font-size: 0.875rem;
  line-height: 1.4;
}

/* ═══════ Detail Panel ═══════ */

.detail-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #1a1b2e;
  color: #d4d4d4;
  overflow: hidden;
}

.detail-section {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
}

.flex {
  display: flex;
}

.space-x-4 {
  gap: 16px;
}

.border-b {
  border-bottom: 1px solid;
}

.border-gray-700 {
  border-color: #374151;
}

.mb-4 {
  margin-bottom: 16px;
}

.cursor-pointer {
  cursor: pointer;
}

.text-gray-300 {
  color: #d1d5db;
}
</style>
