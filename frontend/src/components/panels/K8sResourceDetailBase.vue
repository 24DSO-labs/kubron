<template>
  <div class="detail-panel">
    <!-- Header with title and action menu -->
    <div class="detail-header">
      <span class="rd-label">{{ resourceType }}</span> 
      <div class="detail-actions-bar">
        <!-- Custom action menu items (slot or prop-based) -->
        <slot name="actions" :resource="resource"></slot>

        <!-- Edit button -->
        <button v-if="showEdit !== false" class="btn-action" @click="emit('edit')" title="Edit">
          <K8sIcon name="edit" alt="Edit" size="lg" />
        </button>

        <!-- Delete button -->
        <button v-if="showDelete !== false" class="btn-action btn-danger" @click="emit('delete')" title="Delete resource">
          <K8sIcon name="delete" alt="Delete" size="lg" danger />
        </button>

        <!-- Close button -->
        <button class="btn-close" @click="emit('close')" title="Close">
          <K8sIcon name="close" alt="Close" size="lg" />
        </button>
      </div>
    </div>

    <!-- Meta info (namespace, age, etc) -->
    <div v-if="showMetaInfo" class="detail-meta">
      <div class="meta-row">
        <span class="meta-label">Name:</span>
        <div class="name-with-copy">
          {{ resource.name }}
          <button class="btn-copy" @click="copyToClipboard(resource.name)" title="Copy name">
            <K8sIcon name="copy" alt="Copy" size="sm" />
          </button>
        </div>
      </div>
      <div v-if="resource.namespace" class="meta-row">
        <span class="meta-label">Namespace:</span>
        <a class="meta-link" @click.prevent="onNamespaceClick">
          {{ resource.namespace }}
        </a>
      </div>
      <div v-if="resource.age" class="meta-row">
        <span class="meta-label">Age:</span>
        <div class="age-display">
          <div class="age-relative">{{ formatAgeRelative(resource.age) }}</div>
          <div class="age-absolute">{{ formatAgeAbsolute(resource.created) }}</div>
        </div>
      </div>
    </div>

    <!-- Tabs -->
    <div v-if="tabs.length > 0" class="tabs flex space-x-4 border-b border-gray-700 mb-4">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        :class="['tab', { active: activeTab === tab.id }]"
        @click="activeTab = tab.id"
      >
        {{ tab.label }}
      </button>
    </div>

    <!-- Tab content -->
    <div class="detail-content">
      <!-- Properties tab -->
      <div v-if="activeTab === 'properties'" class="detail-section text-gray-300">
        <slot name="properties" :resource="resource"></slot>
      </div>

      <!-- Events tab -->
      <div v-if="activeTab === 'events'" class="detail-section text-gray-300">
        <slot name="events" :resource="resource"></slot>
      </div>

      <!-- Additional custom tabs -->
      <template v-for="tab in customTabs" :key="tab.id">
        <div v-if="activeTab === tab.id" class="detail-section text-gray-300">
          <slot :name="`tab-${tab.id}`" :resource="resource"></slot>
        </div>
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from 'vue'
import K8sIcon from '../icons/K8sIcon.vue'

interface DetailTab {
  id: string
  label: string
}

const props = defineProps<{
  resource: any
  resourceIcon?: string
  resourceType?: string
  customTabs?: DetailTab[]
  /** false — скрыть кнопку Edit (для нередактируемых сущностей: helm, port forward и т.п.) */
  showEdit?: boolean
  /** false — скрыть кнопку Delete */
  showDelete?: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'edit'): void
  (e: 'delete'): void
  (e: 'namespaceClick', namespace: string): void
}>()

const activeTab = ref('properties')

const showMetaInfo = computed(() => {
  return props.resource && props.resource.name
})

const tabs = computed<DetailTab[]>(() => {
  const baseTabs: DetailTab[] = [
    { id: 'properties', label: 'Properties' },
    { id: 'events', label: 'Events' }
  ]
  if (props.customTabs) {
    return [...baseTabs, ...props.customTabs]
  }
  return baseTabs
})

function onNamespaceClick() {
  if (props.resource?.namespace) {
    emit('namespaceClick', props.resource.namespace)
  }
}

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text).catch(err => {
    console.error('Failed to copy:', err)
  })
}

function formatAgeRelative(age: string): string {
  return age || '—'
}

function formatAgeAbsolute(created: string): string {
  if (!created) return ''
  try {
    const date = new Date(created)
    return date.toLocaleString()
  } catch {
    return created
  }
}
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

.detail-header h3 {
  margin: 0;
  flex: 1;
  color: white;
  font-size: 1.1rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: flex;
  align-items: center;
  gap: 8px;
}

.resource-type-badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 10px;
  border-radius: 4px;
  background: #2a2b40;
  border: 1px solid #35365a;
  color: #868d8a;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  line-height: 1.4;
  flex-shrink: 0;
}

.detail-actions-bar {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.detail-meta {
  padding: 12px 16px;
  border-bottom: 1px solid #2a2b40;
  background: rgba(255, 255, 255, 0.02);
  flex-shrink: 0;
}

.meta-row {
  display: grid;
  grid-template-columns: auto 1fr;
  align-items: baseline;
  margin-bottom: 6px;
  gap: 12px;
  text-align: left;
  font-size: 0.875rem;
}

.meta-row:last-child {
  margin-bottom: 0;
}

.meta-label {
  font-weight: bold;
  color: #d4d4d4;
  white-space: nowrap;
}

.tabs {
  overflow-x: auto;
  flex-shrink: 0;
  padding: 0 16px;
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

.detail-section {
  padding: 8px 0;
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

.btn-action.btn-danger {
  color: #ef4444;
}

.btn-action.btn-danger:hover {
  background: rgba(239, 68, 68, 0.2);
}

.btn-icon {
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

.btn-action :deep(.k8s-icon) {
  filter: invert(1);
  opacity: 0.85;
  transition: opacity 0.15s;
}

.btn-action:hover :deep(.k8s-icon) {
  opacity: 1;
}

.btn-action.btn-danger :deep(.k8s-icon) {
  filter: invert(31%) sepia(91%) saturate(6000%) hue-rotate(350deg) brightness(95%) contrast(90%);
  opacity: 0.9;
}

.btn-action.btn-danger:hover :deep(.k8s-icon) {
  opacity: 1;
}

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

.name-with-copy {
  display: flex;
  align-items: center;
  gap: 8px;
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

.icon-inline {
  width: 10px;
  height: 10px;
  stroke: currentColor;
  filter: invert(1);
  opacity: 0.8;
}

.btn-copy:hover .icon-inline {
  opacity: 1;
}

.btn-copy:active {
  background: rgba(59, 130, 246, 0.1);
}

.flex { display: flex; }
.space-x-4 { gap: 1rem; }
.border-b { border-bottom: 1px solid; }
.border-gray-700 { border-bottom-color: #2a2b40; }
.mb-4 { margin-bottom: 1rem; }
.text-gray-300 { color: #d4d4d4; }

.age-display {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.age-relative {
  font-weight: 500;
  color: #d4d4d4;
}

.age-absolute {
  font-size: 0.8rem;
  color: #868d8a;
}
</style>