<template>
  <div v-if="modelValue" class="dialog-overlay" @click="handleOverlayClick">
    <div class="dialog" @click.stop>
      <div class="dialog-content">
        <p class="dialog-title">{{ title }}</p>
        <p v-if="resourceName" class="resource-name">{{ resourceName }}</p>
      </div>
      
      <div class="dialog-actions">
        <button class="btn btn-cancel" @click="handleCancel">Отмена</button>
        <button class="btn btn-danger" @click="handleConfirm" :disabled="isDeleting">
          <span v-if="isDeleting" class="spinner"></span>
          Подтвердить
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'

const props = defineProps<{
  modelValue: boolean
  resourceType: string
  resourceName: string
  namespace?: string
  title?: string
  isDeleting?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', val: boolean): void
  (e: 'confirm'): void
  (e: 'cancel'): void
}>()

const title = computed(() => {
  if (props.title) return props.title
  return `Удалить ${props.resourceType}?`
})

function handleCancel() {
  emit('cancel')
  emit('update:modelValue', false)
}

function handleConfirm() {
  emit('confirm')
  if (!props.isDeleting) {
    emit('update:modelValue', false)
  }
}

function handleOverlayClick() {
  emit('cancel')
  emit('update:modelValue', false)
}
</script>

<style scoped>
.dialog-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10000;
}

.dialog {
  background: #1e1f35;
  border: 1px solid #3b3d5c;
  border-radius: 4px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
  min-width: 400px;
  padding: 16px 20px;
}

.dialog-content {
  margin-bottom: 16px;
}

.dialog-title {
  color: #e0e0e0;
  font-size: 14px;
  margin: 0 0 8px 0;
}

.resource-name {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  color: #60a5fa;
  word-break: break-all;
  margin: 0;
}

.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.btn {
  padding: 8px 16px;
  border: none;
  border-radius: 4px;
  font-size: 13px;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-cancel {
  background: #3b3d5c;
  color: #d4d4d4;
}

.btn-cancel:hover {
  background: #4a4d6a;
}

.btn-danger {
  background: #dc2626;
  color: white;
}

.btn-danger:hover {
  background: #b91c1c;
}

.btn-danger:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.spinner {
  width: 12px;
  height: 12px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  margin-right: 6px;
  display: inline-block;
  vertical-align: middle;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
