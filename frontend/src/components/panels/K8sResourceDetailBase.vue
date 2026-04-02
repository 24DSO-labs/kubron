<template>
  <div class="detail-panel">
    <!-- Header with title and action menu -->
    <div class="detail-header">
      <h3>{{ resourceIcon }} {{ resource.name }}</h3>
      <div class="detail-actions-bar">
        <!-- Custom action menu items (slot or prop-based) -->
        <slot name="actions" :resource="resource"></slot>

        <!-- Close button -->
        <button class="btn-close" @click="emit('close')">✕</button>
      </div>
    </div>

    <!-- Meta info (namespace, age, etc) -->
    <div v-if="showMetaInfo" class="detail-meta">
      <div class="meta-row">
        <span class="meta-label">Name:</span>
        {{ resource.name }}
      </div>
      <div v-if="resource.namespace" class="meta-row">
        <span class="meta-label">Namespace:</span>
        {{ resource.namespace }}
      </div>
      <div v-if="resource.age" class="meta-row">
        <span class="meta-label">Age:</span>
        {{ resource.age }}
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

export interface DetailTab {
  id: string
  label: string
}

const props = withDefaults(
  defineProps<{
    resource: any
    resourceIcon?: string
    showMetaInfo?: boolean
    customTabs?: DetailTab[]
  }>(),
  {
    resourceIcon: '',
    showMetaInfo: true,
    customTabs: () => []
  }
)

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'action', actionId: string, data?: any): void
}>()

defineSlots<{
  actions(props: { resource: any }): any
  properties(props: { resource: any }): any
  events(props: { resource: any }): any
  [key: string]: (props: { resource: any }) => any
}>()

const activeTab = ref('properties')

const tabs = computed<DetailTab[]>(() => {
  const baseTabs: DetailTab[] = [
    { id: 'properties', label: 'Properties' },
    { id: 'events', label: 'Events' }
  ]
  return [...baseTabs, ...props.customTabs]
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

/* ═══════ Header ═══════ */

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
}

.detail-actions-bar {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

/* ═══════ Meta Info ═══════ */

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

/* ═══════ Tabs ═══════ */

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

/* ═══════ Content ═══════ */

.detail-content {
  flex: 1;
  overflow-y: auto;
  padding: 0 16px 16px 16px;
}

.detail-section {
  padding: 8px 0;
}

/* ═══════ Buttons ═══════ */

.btn-close {
  padding: 6px 10px;
  background: transparent;
  border: 1px solid #3b3d5c;
  border-radius: 4px;
  color: #d4d4d4;
  cursor: pointer;
  font-size: 1rem;
  transition: all 0.15s;
  flex-shrink: 0;
}

.btn-close:hover {
  background: rgba(255, 255, 255, 0.1);
  border-color: #5a5c7a;
  color: white;
}

/* ═══════ Scrollbar ═══════ */

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

/* Utility classes */
.flex {
  display: flex;
}

.space-x-4 {
  gap: 1rem;
}

.border-b {
  border-bottom: 1px solid;
}

.border-gray-700 {
  border-bottom-color: #2a2b40;
}

.mb-4 {
  margin-bottom: 1rem;
}

.text-gray-300 {
  color: #d4d4d4;
}
</style>
