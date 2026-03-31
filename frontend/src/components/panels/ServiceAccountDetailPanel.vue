<template>
  <div class="detail-panel">
    <div class="detail-header">
      <h3>ServiceAccount: {{ serviceAccount.metadata.name }}</h3>
      <div class="detail-actions-bar">
        <!-- Edit YAML -->
        <button class="btn-action" title="Edit YAML" @click="emit('edit', serviceAccount.metadata.namespace, serviceAccount.metadata.name)">
          ✏️
        </button>
        <!-- Delete -->
        <button class="btn-action btn-danger" title="Delete ServiceAccount" @click="emit('delete', serviceAccount.metadata.namespace, serviceAccount.metadata.name)">
          🗑
        </button>
        <!-- Close -->
        <button class="btn-close" @click="emit('close')">✕</button>
      </div>
    </div>

    <div class="detail-section text-gray-300">
      <div class="meta-row">
        <span class="meta-label">Created:</span>
        {{ formatAge(serviceAccount.metadata.creationTimestamp) }} ago ({{ serviceAccount.metadata.creationTimestamp }})
      </div>
      <div class="meta-row">
        <span class="meta-label">Name:</span>
        {{ serviceAccount.metadata.name }}
      </div>
      <div class="meta-row">
        <span class="meta-label">Namespace:</span>
        <span class="ns-link" @click="emit('namespace-click', serviceAccount.metadata.namespace)">{{ serviceAccount.metadata.namespace }}</span>
      </div>

      <div class="meta-row cursor-pointer" @click="labelsExpanded = !labelsExpanded">
        <span class="meta-label">
          Labels <span class="expand-icon" :class="{ expanded: labelsExpanded }">›</span>
        </span>
        <div class="meta-content">
          <span v-if="!labelsExpanded">{{ Object.keys(serviceAccount.metadata.labels || {}).length }} Labels</span>
          <div v-else class="labels-list">
            <span v-for="[key, value] in Object.entries(serviceAccount.metadata.labels || {})" :key="key" class="label-pair">
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
          <span v-if="!annotationsExpanded">{{ Object.keys(serviceAccount.metadata.annotations || {}).length }} Annotations</span>
          <div v-else class="labels-list">
            <span v-for="[key, value] in Object.entries(serviceAccount.metadata.annotations || {})" :key="key" class="label-pair">
              <span class="label-key">{{ key }}</span>=<span class="label-value">{{ value }}</span>
            </span>
          </div>
        </div>
      </div>

      <div class="meta-row cursor-pointer" @click="tokensExpanded = !tokensExpanded">
        <span class="meta-label">
          Tokens <span class="expand-icon" :class="{ expanded: tokensExpanded }">›</span>
        </span>
        <div class="meta-content">
          <span v-if="!tokensExpanded">{{ (serviceAccount.secrets || []).length }}</span>
          <div v-else class="labels-list">
            <span v-for="secret in (serviceAccount.secrets || [])" :key="secret.name" class="label-pair">
              {{ secret.name }}
            </span>
            <span v-if="!serviceAccount.secrets || !serviceAccount.secrets.length">-</span>
          </div>
        </div>
      </div>

      <div class="meta-row cursor-pointer" @click="eventsExpanded = !eventsExpanded">
        <span class="meta-label">
          Events <span class="expand-icon" :class="{ expanded: eventsExpanded }">›</span>
        </span>
        <div class="meta-content">
          <span v-if="!eventsExpanded">Events found</span>
          <div v-else class="labels-list">
            <!-- Здесь можно добавить отображение событий, если они есть в данных -->
            <span> - </span> <!-- Placeholder, если событий нет -->
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref } from 'vue'

const props = defineProps<{
  serviceAccount: any
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'delete', ns: string, name: string): void
  (e: 'edit', ns: string, name: string): void
  (e: 'namespace-click', ns: string): void
}>()

const labelsExpanded = ref(false)
const annotationsExpanded = ref(false)
const tokensExpanded = ref(false)
const eventsExpanded = ref(false)

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
</script>

<style scoped>
/* Стили скопированы из PodDetailPanel.vue и адаптированы */

.detail-panel {
  background: #1a1b2e;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  max-height: 80vh;
  overflow: auto;
  color: #d4d4d4;
}

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

.detail-section {
  padding: 12px 16px;
}

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

.status-badge {
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 0.85rem;
  font-weight: 500;
}

/* Цвета статусов, если нужно */
.status-running {
  background: rgba(52, 211, 153, 0.1);
  color: #34d399;
}

.ns-link {
  color: #60a5fa;
  cursor: pointer;
}

.ns-link:hover {
  text-decoration: underline;
}

</style>
