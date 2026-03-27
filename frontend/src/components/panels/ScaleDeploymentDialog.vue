<template>
  <div class="scale-dialog-overlay" @click="handleOverlayClick">
    <div class="scale-dialog" @click.stop>
      <div class="dialog-header">
        <h3>Scale Deployment <span class="resource-name">{{ deployment?.namespace }}/{{ deployment?.name }}</span></h3>
      </div>

      <div class="dialog-content">
        <div class="scale-info">
          <div class="info-row">
            <span class="info-label">Current replica scale:</span>
            <span class="info-value">{{ deployment?.replicas }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">Desired number of replicas:</span>
            <span class="info-value">{{ desiredReplicas }}</span>
          </div>
        </div>

        <div class="slider-container">
          <input
            v-model.number="desiredReplicas"
            type="range"
            class="slider"
            :min="0"
            :max="maxReplicas"
          />
          <div class="slider-controls">
            <button class="slider-btn minus-btn" @click="desiredReplicas = Math.max(0, desiredReplicas - 1)" title="Decrease">−</button>
            <button class="slider-btn plus-btn" @click="desiredReplicas = Math.min(maxReplicas, desiredReplicas + 1)" title="Increase">+</button>
          </div>
        </div>
      </div>

      <div class="dialog-footer">
        <button class="btn btn-cancel" @click="handleCancel">Cancel</button>
        <button class="btn btn-scale" @click="handleScale" :disabled="loading">
          {{ loading ? '⏳ Scaling...' : '▶ Scale' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref } from 'vue'
import type { DeploymentInfo } from '../../types'

const props = defineProps<{
  deployment: DeploymentInfo | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'scale', replicas: number): void
}>()

const desiredReplicas = ref(props.deployment?.replicas || 1)
const loading = ref(false)
const maxReplicas = ref(20)

function handleOverlayClick() {
  handleCancel()
}

function handleCancel() {
  emit('close')
}

async function handleScale() {
  if (!props.deployment) return
  
  try {
    loading.value = true
    emit('scale', desiredReplicas.value)
    // Закроем диалог после успеха (сигнал придёт снаружи если нужно)
    setTimeout(() => {
      emit('close')
    }, 500)
  } catch (e) {
    console.error('Scale error:', e)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.scale-dialog-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
}

.scale-dialog {
  background: #1e1f35;
  border: 1px solid #3b3d5c;
  border-radius: 8px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.5);
  width: 90%;
  max-width: 500px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.dialog-header {
  padding: 16px;
  border-bottom: 1px solid #3b3d5c;
  background: #0f1017;
}

.dialog-header h3 {
  margin: 0;
  color: white;
  font-size: 1rem;
  font-weight: 600;
}

.resource-name {
  color: #60a5fa;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
}

.dialog-content {
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.scale-info {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.info-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
}

.info-label {
  color: #9ca0b8;
  font-size: 0.9rem;
  font-weight: 500;
}

.info-value {
  color: #60a5fa;
  font-size: 1rem;
  font-weight: 600;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
}

.slider-container {
  display: flex;
  align-items: center;
  gap: 12px;
}

.slider {
  flex: 1;
  height: 6px;
  border-radius: 3px;
  background: linear-gradient(
    to right,
    #3b82f6 0%,
    #3b82f6 calc(var(--value, 0) * 100%),
    #3b3d5c calc(var(--value, 0) * 100%),
    #3b3d5c 100%
  );
  outline: none;
  -webkit-appearance: none;
  appearance: none;
  cursor: pointer;
}

.slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: #3b82f6;
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.4);
  transition: all 0.15s;
}

.slider::-webkit-slider-thumb:hover {
  transform: scale(1.2);
  box-shadow: 0 2px 12px rgba(59, 130, 246, 0.6);
}

.slider::-moz-range-thumb {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: #3b82f6;
  border: none;
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(59, 130, 246, 0.4);
  transition: all 0.15s;
}

.slider::-moz-range-thumb:hover {
  transform: scale(1.2);
  box-shadow: 0 2px 12px rgba(59, 130, 246, 0.6);
}

.slider-controls {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.slider-btn {
  width: 32px;
  height: 32px;
  border-radius: 4px;
  border: 1px solid #3b3d5c;
  background: #1a1b2e;
  color: #9ca0b8;
  font-size: 1rem;
  cursor: pointer;
  transition: all 0.15s;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
}

.slider-btn:hover {
  background: rgba(59, 130, 246, 0.1);
  border-color: #3b82f6;
  color: #3b82f6;
}

.minus-btn {
  /* Icon − */
}

.plus-btn {
  /* Icon + */
}

.dialog-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 16px;
  border-top: 1px solid #3b3d5c;
  background: #0f1017;
}

.btn {
  padding: 8px 16px;
  border-radius: 4px;
  border: none;
  font-weight: 600;
  font-size: 0.9rem;
  cursor: pointer;
  transition: all 0.15s;
  white-space: nowrap;
}

.btn-cancel {
  background: transparent;
  border: 1px solid #3b3d5c;
  color: #9ca0b8;
}

.btn-cancel:hover {
  background: rgba(255, 255, 255, 0.05);
  border-color: #5a5c7a;
  color: #d4d4d4;
}

.btn-scale {
  background: #3b82f6;
  color: white;
  margin-left: auto;
}

.btn-scale:hover:not(:disabled) {
  background: #2563eb;
}

.btn-scale:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
