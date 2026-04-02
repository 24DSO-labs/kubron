<template>
  <K8sResourceDetailBase
    :resource="cm"
    resource-icon="📄"
    @close="$emit('close')"
  >
    <template #actions>
      <button class="btn-action" title="Edit YAML" @click="$emit('edit')">✏️</button>
      <button class="btn-action btn-danger" title="Delete" @click="$emit('delete')">🗑️</button>
    </template>

    <template #properties>
      <div class="cm-properties">
        <div class="meta-row">
          <span class="meta-label">Keys:</span>
          {{ cm.dataCount }}
        </div>

        <div v-if="cm.data" class="cm-data-section">
          <h4>Data</h4>
          <div v-for="(val, key) in cm.data" :key="key" class="cm-data-item">
            <div class="cm-key">{{ key }}</div>
            <pre class="cm-value">{{ val }}</pre>
          </div>
        </div>
      </div>
    </template>

    <template #events>
      <div class="empty-section">No events available</div>
    </template>
  </K8sResourceDetailBase>
</template>

<script lang="ts" setup>
import K8sResourceDetailBase from '../K8sResourceDetailBase.vue'

defineProps<{ cm: any }>()
defineEmits<{
  (e: 'close'): void
  (e: 'edit'): void
  (e: 'delete'): void
}>()
</script>

<style scoped>
.cm-properties {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.meta-row {
  display: grid;
  grid-template-columns: auto 1fr;
  align-items: baseline;
  gap: 12px;
  font-size: 0.875rem;
}

.meta-label {
  font-weight: bold;
  color: #d4d4d4;
  white-space: nowrap;
}

.cm-data-section {
  margin-top: 12px;
}

.cm-data-section h4 {
  margin: 0 0 12px 0;
  color: #e0e2f0;
  font-size: 0.95rem;
  font-weight: 600;
}

.cm-data-item {
  margin-bottom: 12px;
  padding: 8px;
  background: rgba(255, 255, 255, 0.02);
  border-radius: 4px;
  border: 1px solid #2a2b40;
}

.cm-key {
  font-weight: 600;
  color: #60a5fa;
  margin-bottom: 6px;
  font-size: 0.85rem;
}

.cm-value {
  margin: 0;
  padding: 8px;
  background: #1e1f35;
  border-radius: 3px;
  font-size: 0.8rem;
  color: #9ca0b8;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 200px;
  overflow-y: auto;
}

.empty-section {
  padding: 16px;
  text-align: center;
  color: #6b6d8a;
  font-size: 0.875rem;
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
}

.btn-action:hover {
  background: rgba(255, 255, 255, 0.1);
  border-color: #5a5c7a;
}

.btn-action.btn-danger:hover {
  background: rgba(239, 68, 68, 0.2);
  border-color: #ef4444;
}

.btn-danger {
  color: #ef4444;
}
</style>
