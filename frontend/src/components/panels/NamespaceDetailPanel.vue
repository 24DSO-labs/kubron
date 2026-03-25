<template>
  <div class="detail-panel">
    <div class="detail-header">
      <h3>{{ resource.name }}</h3>
      <button class="btn-close" @click="emit('close')">✕</button>
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
        {{ resource.age }} ago ({{ formatTimestamp(resource.creationTimestamp) }})
      </div>
      <div class="meta-row">
        <span class="meta-label">Name:</span>
        {{ resource.name }}
      </div>
      <div class="meta-row">
        <span class="meta-label">Status:</span>
        <span :class="statusClass(resource.status)" class="status-badge">{{ resource.status }}</span>
      </div>

      <div class="meta-row cursor-pointer" @click="labelsExpanded = !labelsExpanded">
        <span class="meta-label">
          Labels <span class="expand-icon" :class="{ expanded: labelsExpanded }">›</span>
        </span>
        <div class="meta-content">
          <span v-if="!labelsExpanded">{{ Object.keys(resource.labels || {}).length }} Labels</span>
          <div v-else class="labels-list">
            <span v-for="[key, value] in Object.entries(resource.labels || {})" :key="key" class="label-pair">
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
          <span v-if="!annotationsExpanded">{{ Object.keys(resource.annotations || {}).length }} Annotations</span>
          <div v-else class="labels-list">
            <span v-for="[key, value] in Object.entries(resource.annotations || {})" :key="key" class="label-pair">
              <span class="label-key">{{ key }}</span>=<span class="label-value">{{ value }}</span>
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- ═══════════ EVENTS ═══════════ -->
    <div v-if="activeTab === 'events'" class="detail-section">
      <div class="text-gray-400 text-center py-8">No events found</div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref } from 'vue'

const props = defineProps<{
  resource: any
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const activeTab = ref('properties')
const tabs = [
  { id: 'properties', label: 'Properties' },
  { id: 'events', label: 'Events' },
]

const labelsExpanded = ref(false)
const annotationsExpanded = ref(false)

function formatTimestamp(timestamp: string): string {
  if (!timestamp) return '—'
  return new Date(timestamp).toLocaleString('en-GB', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    timeZoneName: 'short'
  }).replace(',', ' f.')
}

function statusClass(status: string): string {
  switch (status.toLowerCase()) {
    case 'active': case 'ready': return 'status-running'
    case 'terminating': return 'status-pending'
    default: return 'status-unknown'
  }
}
</script>

<style scoped>
.detail-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #0f1419;
  border-left: 1px solid #1e2128;
  overflow: hidden;
}

.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border-bottom: 1px solid #1e2128;
  flex-shrink: 0;
}

.detail-header h3 {
  margin: 0;
  font-size: 18px;
  color: #fff;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.btn-close {
  background: transparent;
  border: none;
  color: #888;
  cursor: pointer;
  font-size: 20px;
  padding: 0;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
}

.btn-close:hover {
  background: rgba(255, 255, 255, 0.1);
  color: #fff;
}

.tabs {
  padding: 0 16px;
  border-bottom: 1px solid #1e2128;
  flex-shrink: 0;
  overflow-x: auto;
}

.tab {
  padding: 12px 12px;
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  color: #888;
  cursor: pointer;
  font-size: 14px;
  white-space: nowrap;
  transition: all 0.2s;
}

.tab:hover {
  color: #ccc;
}

.tab.active {
  border-bottom-color: #3b82f6;
  color: #fff;
}

.detail-section {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  text-align: left;
}

.meta-row {
  display: grid;
  grid-template-columns: auto 1fr;
  align-items: baseline;
  margin-bottom: 8px;
  gap: 12px;
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

.status-badge {
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 0.875rem;
  font-weight: 500;
}
</style>
