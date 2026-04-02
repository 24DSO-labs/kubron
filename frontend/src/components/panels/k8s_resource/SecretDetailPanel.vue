<template>
  <K8sResourceDetailBase :resource="resource" resource-?="?" @close="$emit('close')">
    <template #actions>
      <button class="btn-action" title="Edit YAML" @click="$emit('edit')">??</button>
      <button class="btn-action btn-danger" title="Delete" @click="$emit('delete')">??</button>
    </template>

    <template #properties>
      <div class="resource-properties">
        <p style="color: #6b6d8a; font-size: 0.875rem;">Resource details coming soon...</p>
      </div>
    </template>

    <template #events>
      <div class="empty-section">No events available</div>
    </template>
  </K8sResourceDetailBase>
</template>

<script lang="ts" setup>
import K8sResourceDetailBase from '../K8sResourceDetailBase.vue'

defineProps<{ resource: any }>()
defineEmits<{
  (e: 'close'): void
  (e: 'edit'): void
  (e: 'delete'): void
}>()
</script>

<style scoped>
.resource-properties {
  display: flex;
  flex-direction: column;
  gap: 12px;
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

.btn-action.btn-danger {
  color: #ef4444;
}

.btn-action.btn-danger:hover {
  background: rgba(239, 68, 68, 0.2);
  border-color: #ef4444;
}

.empty-section {
  padding: 16px;
  text-align: center;
  color: #6b6d8a;
  font-size: 0.875rem;
}
</style>
